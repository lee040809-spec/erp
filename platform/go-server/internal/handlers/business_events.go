package handlers

import (
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"machining-erp/internal/middleware"
	"machining-erp/internal/models"
	"machining-erp/internal/outsourcing"
	"machining-erp/internal/productionflow"
	"machining-erp/internal/services"
	"machining-erp/pkg/response"
	"machining-erp/pkg/utils"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

type BusinessRecord struct {
	DeliveryBatchID *int64     `json:"delivery_batch_id"`
	ID              int64      `json:"id"`
	CorpID          int64      `json:"corp_id"`
	OrderID         int64      `json:"order_id"`
	OrderItemID     *int64     `json:"order_item_id"`
	Kind            string     `json:"kind"`
	Quantity        int        `json:"quantity"`
	BusinessDate    *time.Time `json:"business_date"`
	ProcessName     string     `json:"process_name"`
	Category        string     `json:"category"`
	Severity        string     `json:"severity"`
	Description     string     `json:"description"`
	Notes           string     `json:"notes"`
	SourceType      string     `json:"source_type"`
	RequestID       string     `json:"request_id"`
	Status          string     `json:"status"`
	CreatedBy       int64      `json:"created_by"`
	ResolvedBy      *int64     `json:"resolved_by"`
	ResolvedAt      *time.Time `json:"resolved_at"`
	Resolution      string     `json:"resolution"`
	CreatedAt       time.Time  `json:"created_at"`
}

func (BusinessRecord) TableName() string { return "order_business_records" }

type BusinessEventHandler struct{ db *gorm.DB }

func NewBusinessEventHandler(db *gorm.DB) *BusinessEventHandler { return &BusinessEventHandler{db} }
func (h *BusinessEventHandler) can(c *gin.Context, kind string) bool {
	switch kind {
	case "scrap":
		return hasModuleAccess(h.db, c, "orders", "production_orders", "production_dashboard")
	case "delivery":
		return hasModuleAccess(h.db, c, "orders", "production_delivery")
	case "exception":
		return hasModuleAccess(h.db, c, "orders", "production_orders", "production_dashboard", "production_delivery")
	}
	return false
}
func (h *BusinessEventHandler) List(c *gin.Context) {
	if !requireModuleAccess(h.db, c, "orders", "production_orders", "production_dashboard", "production_delivery", "dashboard") {
		return
	}
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	corp := middleware.GetCorpID(c)
	var order models.Order
	if h.db.Where("id=? AND corp_id=?", id, corp).Take(&order).Error != nil {
		response.ErrorWithCode(c, http.StatusNotFound, response.CodeNotFound, "订单不存在")
		return
	}
	rows := []map[string]interface{}{}
	err := h.db.Raw(`SELECT i.id,i.part_name,i.part_number,i.quantity,COALESCE(NULLIF(i.production_quantity,0),i.quantity) AS production_quantity,COALESCE(i.scrap_quantity,0) AS scrap_quantity,COALESCE(i.delivered_quantity,0) AS delivered_quantity,
 COALESCE(q.total,0) AS qualified_quantity,DATE_FORMAT(i.completion_date,'%Y-%m-%d') AS completion_date,
 DATE_FORMAT(d.delivery_date,'%Y-%m-%d') AS delivery_date,GREATEST(LEAST(i.quantity,COALESCE(q.total,0))-COALESCE(i.delivered_quantity,0),0) AS available_delivery
 FROM order_items i LEFT JOIN (SELECT corp_id,order_item_id,SUM(qualified_quantity) AS total FROM production_completions WHERE status='confirmed' GROUP BY corp_id,order_item_id) q ON q.corp_id=i.corp_id AND q.order_item_id=i.id
 LEFT JOIN (SELECT corp_id,order_item_id,MAX(business_date) AS delivery_date FROM order_business_records WHERE kind='delivery' AND status='confirmed' GROUP BY corp_id,order_item_id) d ON d.corp_id=i.corp_id AND d.order_item_id=i.id
 WHERE i.corp_id=? AND i.order_id=? ORDER BY i.id`, corp, id).Scan(&rows).Error
	if err != nil {
		response.InternalError(c, response.CodeInternal, "读取数量失败")
		return
	}
	page, size, offset := boundedPage(c, 50)
	records := []BusinessRecord{}
	if err = h.db.Where("corp_id=? AND order_id=?", corp, id).Order("id DESC").Limit(size + 1).Offset(offset).Find(&records).Error; err != nil {
		response.InternalError(c, response.CodeInternal, "读取业务记录失败")
		return
	}
	policy, err := effectiveEvidencePolicy(h.db, c)
	if err != nil {
		response.InternalError(c, response.CodeInternal, "读取填写要求失败")
		return
	}
	completions := []map[string]interface{}{}
	if err = h.db.Table("production_completions").Select("id,order_item_id,business_date,qualified_quantity,billable_quantity,status").Where("corp_id=? AND order_id=?", corp, id).Order("business_date DESC,id DESC").Limit(size + 1).Offset(offset).Find(&completions).Error; err != nil {
		response.InternalError(c, response.CodeInternal, "读取完工发生记录失败")
		return
	}
	records, recordsMore := trimPage(records, size)
	completions, completionsMore := trimPage(completions, size)
	response.Success(c, gin.H{"page": page, "page_size": size, "has_more": recordsMore || completionsMore, "completions": completions, "items": rows, "records": records, "can_complete": hasModuleAccess(h.db, c, "production_orders", "production_dashboard"), "can_scrap": h.can(c, "scrap"), "can_delivery": h.can(c, "delivery"), "can_exception": h.can(c, "exception"), "can_reverse": middleware.GetRoleType(c) == "admin", "evidence": policy})
}

type businessRequest struct {
	DeliveryBatchID *int64 `json:"delivery_batch_id"`
	ItemID          *int64 `json:"order_item_id"`
	Kind            string `json:"kind"`
	Quantity        int    `json:"quantity"`
	Date            string `json:"business_date"`
	Process         string `json:"process_name"`
	Category        string `json:"category"`
	Severity        string `json:"severity"`
	Description     string `json:"description"`
	Notes           string `json:"notes"`
	RequestID       string `json:"request_id"`
}

func (h *BusinessEventHandler) Create(c *gin.Context) {
	var req businessRequest
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	if c.ShouldBindJSON(&req) != nil || id <= 0 || !regexp.MustCompile(`^[A-Za-z0-9-]{1,36}$`).MatchString(req.RequestID) {
		response.BadRequest(c, response.CodeInvalidRequest, "业务记录格式或请求编号无效")
		return
	}
	if !h.can(c, req.Kind) {
		response.Forbidden(c, response.CodeForbidden, "没有此业务登记权限")
		return
	}
	if req.Severity == "" {
		req.Severity = "normal"
	}
	req.Notes = strings.TrimSpace(req.Notes)
	req.Description = strings.TrimSpace(req.Description)
	date, err := time.Parse("2006-01-02", req.Date)
	if err != nil || req.Date > time.Now().Format("2006-01-02") {
		response.BadRequest(c, response.CodeInvalidRequest, "发生日期无效或晚于今天")
		return
	}
	if utf8.RuneCountInString(req.Description) > 500 || utf8.RuneCountInString(req.Process) > 80 || utf8.RuneCountInString(req.Category) > 40 || (req.Severity != "normal" && req.Severity != "urgent") {
		response.BadRequest(c, response.CodeInvalidRequest, "说明长度或紧急程度无效")
		return
	}
	if !checkEvidence(h.db, c, "operation_notes", req.Notes, 500) {
		return
	}
	if req.Kind == "exception" {
		if req.Description == "" || req.Category == "" {
			response.BadRequest(c, response.CodeInvalidRequest, "请填写异常类型和异常情况")
			return
		}
		req.Quantity = 0
	} else if req.ItemID == nil || *req.ItemID <= 0 || req.Quantity <= 0 || req.Quantity > 2147483647 {
		response.BadRequest(c, response.CodeInvalidRequest, "请选择零件并填写本次数量")
		return
	}
	corp := middleware.GetCorpID(c)
	tx := h.db.Begin()
	defer tx.Rollback()
	if tx.Error != nil {
		response.InternalError(c, response.CodeInternal, "启动业务登记失败")
		return
	}
	var order models.Order
	if tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id=? AND corp_id=?", id, corp).Take(&order).Error != nil {
		response.ErrorWithCode(c, http.StatusNotFound, response.CodeNotFound, "订单不存在")
		return
	}
	var item models.OrderItem
	if req.ItemID != nil {
		if tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id=? AND order_id=? AND corp_id=?", *req.ItemID, id, corp).Take(&item).Error != nil {
			response.BadRequest(c, response.CodeInvalidRequest, "零件不属于当前订单")
			return
		}
	}
	var existing BusinessRecord
	err = tx.Where("corp_id=? AND request_id=?", corp, req.RequestID).Take(&existing).Error
	if err == nil {
		sameItem := (existing.OrderItemID == nil && req.ItemID == nil) || (existing.OrderItemID != nil && req.ItemID != nil && *existing.OrderItemID == *req.ItemID)
		if !outsourcing.SameID(existing.DeliveryBatchID, req.DeliveryBatchID) || existing.OrderID != id || !sameItem || existing.Kind != req.Kind || existing.Quantity != req.Quantity || existing.BusinessDate == nil || existing.BusinessDate.Format("2006-01-02") != req.Date || existing.Notes != req.Notes || existing.Description != req.Description || existing.ProcessName != req.Process || existing.Category != req.Category || existing.Severity != req.Severity || existing.Status == "reversed" {
			response.BadRequest(c, response.CodeInvalidRequest, "请求编号已使用，内容不一致")
			return
		}
		response.SuccessWithMessage(c, "记录已保存，无需重复登记", existing)
		return
	}
	if err != gorm.ErrRecordNotFound {
		response.InternalError(c, response.CodeInternal, "检查重复登记失败")
		return
	}
	if req.Kind == "scrap" {
		on, e := productionflow.Enabled(tx, corp, item.ID)
		if e != nil || on {
			response.BadRequest(c, response.CodeInvalidRequest, "该零件已启用工序数量跟踪，请在具体工序或外协回厂中登记报废")
			return
		}
	}
	if req.Kind == "delivery" {
		if e := productionflow.ValidateDelivery(tx, corp, item.ID, req.DeliveryBatchID, req.Quantity, req.Date); e != nil {
			response.BadRequest(c, response.CodeInvalidRequest, e.Error())
			return
		}
	}
	if req.Kind != "exception" {
		var sum int64
		if err = tx.Table("order_business_records").Where("corp_id=? AND order_item_id=? AND kind=? AND status='confirmed'", corp, item.ID, req.Kind).Select("COALESCE(SUM(quantity),0)").Scan(&sum).Error; err != nil {
			response.InternalError(c, response.CodeInternal, "读取累计数量失败")
			return
		}
		old := item.ScrapQuantity
		if req.Kind == "delivery" {
			old = 0
			if item.DeliveredQty != nil {
				old = *item.DeliveredQty
			}
		}
		if int64(old) != sum {
			response.BadRequest(c, response.CodeInvalidRequest, "历史存量与业务记录不一致，请管理员核对，不能覆盖原数据")
			return
		}
		var qualified float64
		if err = tx.Table("production_completions").Where("corp_id=? AND order_item_id=? AND status='confirmed'", corp, item.ID).Select("COALESCE(SUM(qualified_quantity),0)").Scan(&qualified).Error; err != nil {
			response.InternalError(c, response.CodeInternal, "读取合格完工数量失败")
			return
		}
		production := item.ProductionQuantity
		if production == 0 {
			production = item.Quantity
		}
		if req.Kind == "scrap" && float64(old+req.Quantity) > float64(production)-qualified {
			response.BadRequest(c, response.CodeInvalidRequest, "报废数量超过未确认合格的投产余量；已确认合格件的质量问题请先登记异常")
			return
		}
		if req.Kind == "delivery" && (float64(old+req.Quantity) > qualified || old+req.Quantity > item.Quantity) {
			response.BadRequest(c, response.CodeInvalidRequest, "本次交货超过未交的已确认合格数量或订单数量；备用件不自动交货")
			return
		}
		if req.Kind == "delivery" {
			valid, timelineErr := deliveryTimelineFits(tx, corp, item.ID, date, req.Quantity)
			if timelineErr != nil {
				response.InternalError(c, response.CodeInternal, "核对交货日期失败")
				return
			}
			if !valid {
				response.BadRequest(c, response.CodeInvalidRequest, "交货日期与已确认合格批次不一致：不能在合格完工前交货或挤占已有交货记录")
				return
			}
		}
		field := "scrap_quantity"
		if req.Kind == "delivery" {
			field = "delivered_quantity"
		}
		if err = tx.Model(&item).Update(field, old+req.Quantity).Error; err != nil {
			response.InternalError(c, response.CodeInternal, "更新累计数量失败")
			return
		}
		if req.Kind == "delivery" {
			if err = syncDeliveryStatus(tx, corp, id, item.ID, old+req.Quantity, item.Quantity, qualified); err != nil {
				response.InternalError(c, response.CodeInternal, "更新交货状态失败")
				return
			}
		}
	}
	row := BusinessRecord{DeliveryBatchID: req.DeliveryBatchID, CorpID: corp, OrderID: id, OrderItemID: req.ItemID, Kind: req.Kind, Quantity: req.Quantity, BusinessDate: &date, ProcessName: req.Process, Category: req.Category, Severity: req.Severity, Description: req.Description, Notes: req.Notes, RequestID: req.RequestID, Status: "confirmed", SourceType: "business", CreatedBy: middleware.GetUserID(c)}
	if req.Kind == "exception" {
		row.Status = "open"
	}
	if tx.Create(&row).Error != nil || tx.Commit().Error != nil {
		response.InternalError(c, response.CodeInternal, "保存业务记录失败")
		return
	}
	response.SuccessWithMessage(c, "登记成功，订单汇总已更新", row)
}
func syncDeliveryStatus(tx *gorm.DB, corp, order, item int64, delivered, ordered int, qualified float64) error {
	status := "processing"
	if qualified >= float64(ordered) {
		status = "completed"
	}
	if delivered >= ordered {
		status = "delivered"
	}
	if err := tx.Table("order_items").Where("corp_id=? AND id=?", corp, item).Update("status", status).Error; err != nil {
		return err
	}
	if delivered >= ordered {
		if err := tx.Table("order_processes").Where("corp_id=? AND order_item_id=? AND name='送货'", corp, item).Update("status", "completed").Error; err != nil {
			return err
		}
	} else {
		if err := tx.Table("order_processes").Where("corp_id=? AND order_item_id=? AND name='送货'", corp, item).Update("status", "pending").Error; err != nil {
			return err
		}
	}
	var statuses []string
	if err := tx.Table("order_items").Where("order_id=? AND corp_id=?", order, corp).Pluck("status", &statuses).Error; err != nil {
		return err
	}
	return tx.Table("orders").Where("id=? AND corp_id=?", order, corp).Update("status", utils.CalculateStatus(statuses)).Error
}
func (h *BusinessEventHandler) Resolve(c *gin.Context) {
	if !h.can(c, "exception") {
		response.Forbidden(c, response.CodeForbidden, "没有异常处理权限")
		return
	}
	var req struct {
		Resolution string `json:"resolution"`
	}
	if c.ShouldBindJSON(&req) != nil || strings.TrimSpace(req.Resolution) == "" || utf8.RuneCountInString(req.Resolution) > 500 {
		response.BadRequest(c, response.CodeInvalidRequest, "请填写处理结果（最多500字）")
		return
	}
	id, _ := strconv.ParseInt(c.Param("recordId"), 10, 64)
	result := h.db.Model(&BusinessRecord{}).Where("id=? AND corp_id=? AND kind='exception' AND status='open'", id, middleware.GetCorpID(c)).Updates(map[string]interface{}{"status": "resolved", "resolution": strings.TrimSpace(req.Resolution), "resolved_by": middleware.GetUserID(c), "resolved_at": time.Now()})
	if result.Error != nil {
		response.InternalError(c, response.CodeInternal, "保存异常处理失败")
		return
	}
	if result.RowsAffected != 1 {
		response.BadRequest(c, response.CodeInvalidRequest, "异常不存在或已处理")
		return
	}
	response.SuccessWithMessage(c, "异常已处理，原始记录保留", nil)
}
func (h *BusinessEventHandler) Reverse(c *gin.Context) {
	if middleware.GetRoleType(c) != "admin" {
		response.Forbidden(c, response.CodeForbidden, "仅管理员可冲销数量记录")
		return
	}
	var req struct {
		Reason string `json:"reason"`
	}
	if c.ShouldBindJSON(&req) != nil || strings.TrimSpace(req.Reason) == "" || utf8.RuneCountInString(req.Reason) > 500 {
		response.BadRequest(c, response.CodeInvalidRequest, "请填写冲销原因")
		return
	}
	id, _ := strconv.ParseInt(c.Param("recordId"), 10, 64)
	corp := middleware.GetCorpID(c)
	var row BusinessRecord
	if h.db.Where("id=? AND corp_id=?", id, corp).Take(&row).Error != nil {
		response.ErrorWithCode(c, http.StatusNotFound, response.CodeNotFound, "记录不存在")
		return
	}
	if row.SourceType == "process_quantity" || row.SourceType == "legacy" || row.Kind == "exception" || row.OrderItemID == nil {
		response.BadRequest(c, response.CodeInvalidRequest, "历史存量不能直接冲销；异常请使用处理入口")
		return
	}
	err := h.db.Transaction(func(tx *gorm.DB) error {
		var order models.Order
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id=? AND corp_id=?", row.OrderID, corp).Take(&order).Error; err != nil {
			return err
		}
		var item models.OrderItem
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id=? AND corp_id=?", *row.OrderItemID, corp).Take(&item).Error; err != nil {
			return err
		}
		if err := tx.Where("id=? AND corp_id=?", id, corp).Take(&row).Error; err != nil {
			return err
		}
		if row.Status != "confirmed" {
			return errors.New("记录已经冲销")
		}
		field := "scrap_quantity"
		old := item.ScrapQuantity
		if row.Kind == "delivery" {
			field = "delivered_quantity"
			old = 0
			if item.DeliveredQty != nil {
				old = *item.DeliveredQty
			}
		}
		var sum int64
		if err := tx.Table("order_business_records").Where("corp_id=? AND order_item_id=? AND kind=? AND status='confirmed'", corp, item.ID, row.Kind).Select("COALESCE(SUM(quantity),0)").Scan(&sum).Error; err != nil {
			return err
		}
		if int64(old) != sum || old < row.Quantity {
			return errors.New("累计数量不一致，请核对历史记录")
		}
		if err := tx.Model(&item).Update(field, old-row.Quantity).Error; err != nil {
			return err
		}
		if row.Kind == "delivery" {
			var q float64
			if err := tx.Table("production_completions").Where("corp_id=? AND order_item_id=? AND status='confirmed'", corp, item.ID).Select("COALESCE(SUM(qualified_quantity),0)").Scan(&q).Error; err != nil {
				return err
			}
			if err := syncDeliveryStatus(tx, corp, row.OrderID, item.ID, old-row.Quantity, item.Quantity, q); err != nil {
				return err
			}
		}
		return tx.Model(&row).Updates(map[string]interface{}{"status": "reversed", "resolution": strings.TrimSpace(req.Reason), "resolved_by": middleware.GetUserID(c), "resolved_at": time.Now()}).Error
	})
	if err != nil {
		response.BadRequest(c, response.CodeInvalidRequest, "冲销失败："+err.Error())
		return
	}
	response.SuccessWithMessage(c, "记录已冲销，原记录和原因保留", nil)
}
func (h *BusinessEventHandler) Print(c *gin.Context) {
	if !requireModuleAccess(h.db, c, "orders", "production_delivery") {
		return
	}
	id, _ := strconv.ParseInt(c.Param("recordId"), 10, 64)
	corp := middleware.GetCorpID(c)
	var row BusinessRecord
	if h.db.Where("id=? AND corp_id=? AND kind='delivery' AND status='confirmed'", id, corp).Take(&row).Error != nil || row.OrderItemID == nil {
		response.ErrorWithCode(c, http.StatusNotFound, response.CodeNotFound, "交货记录不存在或已冲销")
		return
	}
	order, err := services.NewDeliveryService(h.db).GetOrderByID(row.OrderID, middleware.GetCorpName(c))
	if err != nil || order.CorpID != corp {
		response.ErrorWithCode(c, http.StatusNotFound, response.CodeNotFound, "订单不存在")
		return
	}
	selected := []services.DeliveryItem{}
	for _, it := range order.Items {
		if it.ID == *row.OrderItemID {
			it.RowIndex = 1
			it.Quantity = row.Quantity
			it.TotalPrice = float64(row.Quantity) * it.UnitPrice
			quantity := row.Quantity
			it.DeliveredQty = &quantity
			selected = append(selected, it)
		}
	}
	if len(selected) != 1 {
		response.BadRequest(c, response.CodeInvalidRequest, "交货零件数据缺失")
		return
	}
	order.Items = selected
	order.Other.TotalQuantity = row.Quantity
	order.TotalAmount = selected[0].TotalPrice
	order.Other.SmallTotal = fmt.Sprintf("%.2f", order.TotalAmount)
	order.Other.BigTotal = services.DeliveryChineseAmount(order.TotalAmount)
	order.Other.DeliveryNumber = fmt.Sprintf("SH-%d", row.ID)
	var result interface{} = order
	if !canViewOrderAmounts(h.db, c) {
		order.Other.SmallTotal = ""
		order.Other.BigTotal = ""
		result = withoutAmounts(order)
	}
	response.Success(c, gin.H{"order": result, "delivery_date": row.BusinessDate, "record_id": row.ID, "source_type": row.SourceType})
}
