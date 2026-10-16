package handlers

import (
	"database/sql"
	"fmt"
	"machining-erp/internal/outsourcing"
	"machining-erp/internal/productionflow"
	"math/big"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"machining-erp/internal/middleware"
	"machining-erp/internal/models"
	"machining-erp/pkg/response"
	"machining-erp/pkg/utils"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var (
	yearPattern  = regexp.MustCompile(`^\d{4}$`)
	moneyPattern = regexp.MustCompile(`^\d+(\.\d{1,2})?$`)
)

var allowedCostTypes = map[string]bool{
	"material": true, "outsourcing": true, "tool": true, "fixture": true,
	"labor": true, "manufacturing": true, "other_direct": true, "period": true,
}

type ProductionFinanceHandler struct {
	db *gorm.DB
}

func NewProductionFinanceHandler(db *gorm.DB) *ProductionFinanceHandler {
	return &ProductionFinanceHandler{db: db}
}

func (h *ProductionFinanceHandler) ListCompletions(c *gin.Context) {
	if !requireModuleAccess(h.db, c, "orders", "production_orders", "production_dashboard", "dashboard", "finance", "monthly_output") {
		return
	}
	itemID, err := strconv.ParseInt(c.Param("itemId"), 10, 64)
	if err != nil || itemID <= 0 {
		response.BadRequest(c, response.CodeInvalidRequest, "零件 ID 无效")
		return
	}
	corpID := middleware.GetCorpID(c)
	var item struct {
		Quantity           int
		ProductionQuantity int
		ScrapQuantity      int
	}
	if err := h.db.Table("order_items").Select("quantity, COALESCE(NULLIF(production_quantity,0),quantity) AS production_quantity, COALESCE(scrap_quantity,0) AS scrap_quantity").Where("id = ? AND corp_id = ?", itemID, corpID).Take(&item).Error; err != nil {
		response.ErrorWithCode(c, http.StatusNotFound, response.CodeNotFound, "零件不存在")
		return
	}
	page, size, offset := boundedPage(c, 50)
	var rows []models.ProductionCompletion
	if err := h.db.Where("corp_id = ? AND order_item_id = ?", corpID, itemID).Order("business_date DESC, id DESC").Limit(size + 1).Offset(offset).Find(&rows).Error; err != nil {
		response.InternalError(c, response.CodeInternal, "读取完工记录失败")
		return
	}
	var confirmed float64
	var billed float64
	if err := h.db.Model(&models.ProductionCompletion{}).
		Where("corp_id = ? AND order_item_id = ? AND status = 'confirmed'", corpID, itemID).
		Select("COALESCE(SUM(qualified_quantity), 0)").Scan(&confirmed).Error; err != nil {
		response.InternalError(c, response.CodeInternal, "读取已完工数量失败")
		return
	}
	if err := h.db.Table("production_completions").Where("corp_id=? AND order_item_id=? AND status='confirmed'", corpID, itemID).Select("COALESCE(SUM(billable_quantity),0)").Scan(&billed).Error; err != nil {
		response.InternalError(c, response.CodeInternal, "读取计产值数量失败")
		return
	}
	remaining := float64(item.ProductionQuantity-item.ScrapQuantity) - confirmed
	if remaining < 0 {
		remaining = 0
	}
	rows, more := trimPage(rows, size)
	var records interface{} = rows
	if !canViewOrderAmounts(h.db, c) {
		records = withoutAmounts(records)
	}
	response.Success(c, gin.H{"page": page, "page_size": size, "has_more": more, "records": records, "ordered_quantity": item.Quantity, "confirmed_quantity": confirmed, "remaining_quantity": remaining, "production_quantity": item.ProductionQuantity, "scrap_quantity": item.ScrapQuantity, "billable_quantity": billed, "reserve_quantity": confirmed - billed, "delivery_gap": max(0, float64(item.Quantity)-confirmed), "billable_remaining": max(0, float64(item.Quantity)-billed)})
}

func (h *ProductionFinanceHandler) CreateCompletion(c *gin.Context) {
	if !requireProductionConfirmation(h.db, c) {
		return
	}
	// Resolve permissions before taking transaction locks, including idempotent replays.
	visibleAmounts := canViewOrderAmounts(h.db, c)
	itemID, err := strconv.ParseInt(c.Param("itemId"), 10, 64)
	if err != nil || itemID <= 0 {
		response.BadRequest(c, response.CodeInvalidRequest, "零件 ID 无效")
		return
	}
	var req struct {
		DeliveryBatchID   *int64  `json:"delivery_batch_id"`
		QualifiedQuantity int     `json:"qualified_quantity"`
		BusinessDate      string  `json:"business_date"`
		Notes             *string `json:"notes"`
		RequestID         string  `json:"request_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.QualifiedQuantity <= 0 {
		response.BadRequest(c, response.CodeInvalidRequest, "合格数量必须是大于 0 的整数")
		return
	}
	if req.RequestID != "" && !regexp.MustCompile(`^[a-zA-Z0-9-]{1,36}$`).MatchString(req.RequestID) {
		response.BadRequest(c, response.CodeInvalidRequest, "请求编号无效")
		return
	}
	notesValue := ""
	if req.Notes != nil {
		notesValue = *req.Notes
	}
	if !checkEvidence(h.db, c, "review_notes", notesValue, 500) {
		return
	}
	businessDate, err := time.Parse("2006-01-02", req.BusinessDate)
	if err != nil || req.BusinessDate > time.Now().Format("2006-01-02") {
		response.BadRequest(c, response.CodeInvalidRequest, "实际完工日期无效或晚于今天")
		return
	}
	corpID := middleware.GetCorpID(c)
	userID := middleware.GetUserID(c)
	month := businessDate.Format("2006-01")

	if h.isMonthClosed(corpID, month) {
		response.BadRequest(c, response.CodeInvalidRequest, "该月份已结账，不能新增完工记录")
		return
	}

	tx := h.db.Begin()
	if tx.Error != nil {
		response.InternalError(c, response.CodeInternal, "启动事务失败")
		return
	}
	defer tx.Rollback()
	if !lockFinanceMonth(tx, c, corpID, month) {
		return
	}

	// Lock the parent before its item, consistent with planning and order editing.
	var parentID int64
	if err := tx.Table("order_items").Where("id=? AND corp_id=?", itemID, corpID).Select("order_id").Scan(&parentID).Error; err != nil || parentID == 0 {
		response.ErrorWithCode(c, http.StatusNotFound, response.CodeNotFound, "零件不存在")
		return
	}
	var parent models.Order
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id=? AND corp_id=?", parentID, corpID).Take(&parent).Error; err != nil {
		response.ErrorWithCode(c, http.StatusNotFound, response.CodeNotFound, "订单不存在")
		return
	}
	var item struct {
		OrderID            int64
		Quantity           int
		ProductionQuantity int
		ScrapQuantity      int
		UnitPrice          sql.NullString
		SupplyMode         string
		Currency           string
		TaxInclusive       bool
	}
	if err := tx.Raw(`SELECT order_id, quantity, COALESCE(NULLIF(production_quantity,0),quantity) AS production_quantity, COALESCE(scrap_quantity,0) AS scrap_quantity, unit_price,
COALESCE(supply_mode, 'self_purchased') AS supply_mode,
COALESCE(currency, 'CNY') AS currency,
COALESCE(tax_inclusive, 0) AS tax_inclusive
FROM order_items WHERE id = ? AND corp_id = ? FOR UPDATE`, itemID, corpID).Scan(&item).Error; err != nil || item.OrderID == 0 {
		response.ErrorWithCode(c, http.StatusNotFound, response.CodeNotFound, "零件不存在")
		return
	}
	completionNo := fmt.Sprintf("PC-%s-%d-%d", businessDate.Format("20060102"), itemID, time.Now().UnixNano())
	if req.RequestID != "" {
		completionNo = fmt.Sprintf("PC-%d-%s", itemID, req.RequestID)
		var existing models.ProductionCompletion
		err := tx.Where("corp_id=? AND completion_no=?", corpID, completionNo).Take(&existing).Error
		if err == nil {
			if !outsourcing.SameID(existing.DeliveryBatchID, req.DeliveryBatchID) || existing.OrderItemID != itemID || existing.BusinessDate.Format("2006-01-02") != req.BusinessDate || !sameDecimal(existing.QualifiedQuantity, strconv.Itoa(req.QualifiedQuantity)) {
				response.BadRequest(c, response.CodeInvalidRequest, "请求编号已使用，提交内容不一致")
				return
			}
			var data interface{} = existing
			if !visibleAmounts {
				data = withoutAmounts(data)
			}
			response.SuccessWithMessage(c, "该完工请求已确认，无需重复记账", data)
			return
		}
		if err != gorm.ErrRecordNotFound {
			response.InternalError(c, response.CodeInternal, "检查重复请求失败")
			return
		}
	}
	if !item.UnitPrice.Valid || strings.TrimSpace(item.UnitPrice.String) == "" || !positiveDecimal(item.UnitPrice.String) || item.Currency != "CNY" {
		response.BadRequest(c, response.CodeInvalidRequest, "零件单价尚未录入或无效，请先在编辑订单中填写大于0的人民币单价，再确认合格完工")
		return
	}
	if e := productionflow.CompletionReady(tx, corpID, itemID, req.QualifiedQuantity, req.DeliveryBatchID, req.BusinessDate); e != nil {
		response.BadRequest(c, response.CodeInvalidRequest, e.Error())
		return
	}
	tracked, e := productionflow.Enabled(tx, corpID, itemID)
	if e != nil {
		response.InternalError(c, response.CodeInternal, "读取数量流转失败")
		return
	}
	var unfinishedProcesses int64
	if err := tx.Table("order_processes").Where("corp_id = ? AND order_item_id = ? AND deleted_at IS NULL AND name <> ? AND status <> 'completed'", corpID, itemID, "送货").Count(&unfinishedProcesses).Error; err != nil {
		response.InternalError(c, response.CodeInternal, "检查工序状态失败")
		return
	}
	if !tracked && unfinishedProcesses > 0 {
		response.BadRequest(c, response.CodeInvalidRequest, "仍有未完成生产工序，暂不能确认最终合格完工")
		return
	}
	var confirmed float64
	var billed float64
	if err := tx.Model(&models.ProductionCompletion{}).
		Where("corp_id = ? AND order_item_id = ? AND status = 'confirmed'", corpID, itemID).
		Select("COALESCE(SUM(qualified_quantity), 0)").Scan(&confirmed).Error; err != nil {
		response.InternalError(c, response.CodeInternal, "检查累计完工数量失败")
		return
	}
	if confirmed+float64(req.QualifiedQuantity) > float64(item.ProductionQuantity-item.ScrapQuantity) {
		response.BadRequest(c, response.CodeInvalidRequest, fmt.Sprintf("本次合格数量超过剩余数量 %.0f", float64(item.ProductionQuantity-item.ScrapQuantity)-confirmed))
		return
	}

	if err := tx.Table("production_completions").Where("corp_id=? AND order_item_id=? AND status='confirmed'", corpID, itemID).Select("COALESCE(SUM(billable_quantity),0)").Scan(&billed).Error; err != nil {
		response.InternalError(c, response.CodeInternal, "读取计产值数量失败")
		return
	}
	billable := max(0, min(float64(req.QualifiedQuantity), float64(item.Quantity)-billed))
	result := tx.Exec(`INSERT INTO production_completions
(delivery_batch_id, corp_id, order_id, order_item_id, completion_no, business_date, qualified_quantity, billable_quantity,
 unit_price_snapshot, output_value, supply_mode_snapshot, currency, tax_inclusive,
 status, notes, created_by, confirmed_at, created_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ROUND(? * CAST(? AS DECIMAL(15,4)), 2), ?, ?, ?, 'confirmed', ?, ?, NOW(3), NOW(3))`,
		req.DeliveryBatchID, corpID, item.OrderID, itemID, completionNo, req.BusinessDate, req.QualifiedQuantity, billable,
		item.UnitPrice.String, billable, item.UnitPrice.String, item.SupplyMode,
		item.Currency, item.TaxInclusive, req.Notes, userID)
	if result.Error != nil {
		response.InternalError(c, response.CodeInternal, "保存完工记录失败")
		return
	}
	var inserted models.ProductionCompletion
	if err := tx.Where("corp_id = ? AND completion_no = ?", corpID, completionNo).Take(&inserted).Error; err != nil {
		response.InternalError(c, response.CodeInternal, "读取完工记录失败")
		return
	}
	newTotal := confirmed + float64(req.QualifiedQuantity)
	if newTotal >= float64(item.Quantity) {
		fulfilled, dateErr := orderFulfilledDate(tx, corpID, itemID, item.Quantity)
		if dateErr != nil {
			response.InternalError(c, response.CodeInternal, "计算完工日期失败")
			return
		}
		if err := tx.Exec("UPDATE order_items SET status = CASE WHEN status='delivered' THEN 'delivered' ELSE 'completed' END, completion_date = ? WHERE id = ? AND corp_id = ?", fulfilled, itemID, corpID).Error; err != nil {
			response.InternalError(c, response.CodeInternal, "更新零件完工状态失败")
			return
		}
	}
	if newTotal < float64(item.Quantity) {
		if err := tx.Exec("UPDATE order_items SET status='processing' WHERE id=? AND corp_id=? AND status<>'delivered'", itemID, corpID).Error; err != nil {
			response.InternalError(c, response.CodeInternal, "更新零件状态失败")
			return
		}
	}
	var statuses []string
	if err := tx.Table("order_items").Where("order_id=? AND corp_id=?", item.OrderID, corpID).Pluck("status", &statuses).Error; err != nil {
		response.InternalError(c, response.CodeInternal, "读取订单状态失败")
		return
	}
	if err := tx.Model(&parent).Update("status", utils.CalculateStatus(statuses)).Error; err != nil {
		response.InternalError(c, response.CodeInternal, "更新订单状态失败")
		return
	}
	if err := tx.Commit().Error; err != nil {
		response.InternalError(c, response.CodeInternal, "提交完工记录失败")
		return
	}
	var data interface{} = inserted
	if !visibleAmounts {
		data = withoutAmounts(data)
	}
	response.SuccessWithMessage(c, "合格完工已确认", data)
}

func (h *ProductionFinanceHandler) CreateCostEntry(c *gin.Context) {
	if !h.requireFinanceWriteAccess(c) {
		return
	}
	var req struct {
		CostType               string  `json:"cost_type"`
		Amount                 string  `json:"amount"`
		TaxAmount              string  `json:"tax_amount"`
		NetAmount              string  `json:"net_amount"`
		BusinessDate           string  `json:"business_date"`
		OrderID                *int64  `json:"order_id"`
		OrderItemID            *int64  `json:"order_item_id"`
		ProductionCompletionID *int64  `json:"production_completion_id"`
		SupplierName           *string `json:"supplier_name"`
		Notes                  *string `json:"notes"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, response.CodeInvalidRequest, "成本数据格式错误")
		return
	}
	if !allowedCostTypes[req.CostType] {
		response.BadRequest(c, response.CodeInvalidRequest, "不支持的成本类型")
		return
	}
	if !validPositiveMoney(req.Amount) {
		response.BadRequest(c, response.CodeInvalidRequest, "金额必须为大于 0、最多两位小数的数字")
		return
	}
	if req.TaxAmount == "" {
		req.TaxAmount = "0.00"
	}
	if !validMoney(req.TaxAmount) {
		response.BadRequest(c, response.CodeInvalidRequest, "税额格式错误")
		return
	}
	if req.NetAmount == "" {
		req.NetAmount = req.Amount
	}
	if !validCostAmounts(req.Amount, req.TaxAmount, req.NetAmount) {
		response.BadRequest(c, response.CodeInvalidRequest, "税额不能超过含税金额；核算金额应等于含税金额（税额计入成本）或含税金额减税额（税额不计入成本）")
		return
	}
	businessDate, err := time.Parse("2006-01-02", req.BusinessDate)
	if err != nil {
		response.BadRequest(c, response.CodeInvalidRequest, "业务日期格式必须为 YYYY-MM-DD")
		return
	}
	month := businessDate.Format("2006-01")
	corpID := middleware.GetCorpID(c)
	tx := h.db.Begin()
	if tx.Error != nil {
		response.InternalError(c, response.CodeInternal, "启动费用事务失败")
		return
	}
	defer tx.Rollback()
	if !lockFinanceMonth(tx, c, corpID, month) {
		return
	}
	if !h.linksBelongToCorp(tx, corpID, req.OrderID, req.OrderItemID, req.ProductionCompletionID) {
		response.BadRequest(c, response.CodeInvalidRequest, "关联的订单、零件或完工批次不存在")
		return
	}
	entry := models.CostEntry{
		CorpID: corpID, CostType: req.CostType, Amount: req.Amount, TaxAmount: req.TaxAmount,
		NetAmount: req.NetAmount, Currency: "CNY", BusinessDate: businessDate,
		AccountingMonth: month, OrderID: req.OrderID, OrderItemID: req.OrderItemID,
		ProductionCompletionID: req.ProductionCompletionID, SupplierName: req.SupplierName,
		Status: "confirmed", Notes: req.Notes, CreatedBy: middleware.GetUserID(c),
	}
	if err := tx.Create(&entry).Error; err != nil {
		response.InternalError(c, response.CodeInternal, "保存成本记录失败")
		return
	}
	if tx.Commit().Error != nil {
		response.InternalError(c, response.CodeInternal, "提交费用事务失败")
		return
	}
	response.SuccessWithMessage(c, "成本费用已保存", entry)
}

func (h *ProductionFinanceHandler) ListCostEntries(c *gin.Context) {
	if !h.requireFinanceAccess(c) {
		return
	}
	month := c.Query("month")
	query := h.db.Where("corp_id = ?", middleware.GetCorpID(c))
	if month != "" {
		if _, err := time.Parse("2006-01", month); err != nil {
			response.BadRequest(c, response.CodeInvalidRequest, "月份格式必须为 YYYY-MM")
			return
		}
		query = query.Where("accounting_month = ?", month)
	}
	var entries []models.CostEntry
	page, size, offset := boundedPage(c, 50)
	if err := query.Order("business_date DESC, id DESC").Limit(size + 1).Offset(offset).Find(&entries).Error; err != nil {
		response.InternalError(c, response.CodeInternal, "读取成本记录失败")
		return
	}
	entries, more := trimPage(entries, size)
	response.Success(c, gin.H{"items": entries, "page": page, "has_more": more})
}

func (h *ProductionFinanceHandler) CloseMonth(c *gin.Context) {
	if !h.requireFinanceWriteAccess(c) {
		return
	}
	month := c.Param("month")
	if !regexp.MustCompile(`^\d{4}-(0[1-9]|1[0-2])$`).MatchString(month) {
		response.BadRequest(c, response.CodeInvalidRequest, "月份格式必须为 YYYY-MM")
		return
	}
	corpID := middleware.GetCorpID(c)
	userID := middleware.GetUserID(c)
	var req struct {
		Reviewed bool   `json:"reviewed_cost_complete"`
		Notes    string `json:"notes"`
	}
	if c.ShouldBindJSON(&req) != nil || !req.Reviewed || len(req.Notes) > 500 {
		response.BadRequest(c, response.CodeInvalidRequest, "请确认成本已完整核对并填写月结依据")
		return
	}
	if !checkEvidence(h.db, c, "closing_notes", req.Notes, 500) {
		return
	}
	if month >= time.Now().Format("2006-01") {
		response.BadRequest(c, response.CodeInvalidRequest, "只能结算已结束月份")
		return
	}
	tx := h.db.Begin()
	if tx.Error != nil {
		response.InternalError(c, response.CodeInternal, "启动月结失败")
		return
	}
	defer tx.Rollback()
	if !lockFinanceMonth(tx, c, corpID, month) {
		return
	}
	var pending int64
	err := tx.Raw(`SELECT COUNT(*) FROM cost_entries e LEFT JOIN
(SELECT cost_entry_id,corp_id,SUM(allocated_amount) AS total FROM cost_allocations GROUP BY corp_id,cost_entry_id) a ON a.cost_entry_id=e.id AND a.corp_id=e.corp_id
WHERE e.corp_id=? AND e.status='confirmed' AND e.cost_type<>'period' AND e.accounting_month<=? AND e.net_amount>COALESCE(a.total,0)`, corpID, month).Scan(&pending).Error
	if err != nil {
		response.InternalError(c, response.CodeInternal, "检查成本完整性失败")
		return
	}
	if pending > 0 {
		response.BadRequest(c, response.CodeInvalidRequest, "本月或更早月份仍有未分摊生产费用，请完成分摊后月结")
		return
	}
	err = tx.Exec("UPDATE monthly_closings SET status='closed',rule_version='manual-v1',closed_by=?,closed_at=NOW(3),notes=? WHERE corp_id=? AND accounting_month=?", userID, req.Notes, corpID, month).Error
	if err != nil || tx.Commit().Error != nil {
		response.InternalError(c, response.CodeInternal, "月结失败")
		return
	}
	response.SuccessWithMessage(c, "月份已结账", gin.H{"month": month, "status": "closed"})
}

func lockFinanceMonth(tx *gorm.DB, c *gin.Context, corpID int64, month string) bool {
	if tx.Exec("INSERT IGNORE INTO monthly_closings (corp_id,accounting_month,status,rule_version) VALUES (?,?,'open','v1')", corpID, month).Error != nil {
		response.InternalError(c, response.CodeInternal, "创建月份锁失败")
		return false
	}
	var row struct{ Status string }
	if tx.Raw("SELECT status FROM monthly_closings WHERE corp_id=? AND accounting_month=? FOR UPDATE", corpID, month).Scan(&row).Error != nil {
		response.InternalError(c, response.CodeInternal, "检查月份锁失败")
		return false
	}
	if row.Status == "closed" {
		response.BadRequest(c, response.CodeInvalidRequest, "该月份已结账，不能新增或更改核算记录")
		return false
	}
	return true
}

func (h *ProductionFinanceHandler) isMonthClosed(corpID int64, month string) bool {
	var count int64
	h.db.Table("monthly_closings").Where("corp_id = ? AND accounting_month = ? AND status = 'closed'", corpID, month).Count(&count)
	return count > 0
}

func (h *ProductionFinanceHandler) linksBelongToCorp(db *gorm.DB, corpID int64, orderID, itemID, completionID *int64) bool {
	// Lock the parent order before its item, matching order editing/deletion.
	var linkedItem models.OrderItem
	if itemID != nil {
		if db.Where("id=? AND corp_id=?", *itemID, corpID).Take(&linkedItem).Error != nil {
			return false
		}
	}
	var linkedCompletion models.ProductionCompletion
	if completionID != nil {
		if db.Where("id=? AND corp_id=? AND status='confirmed'", *completionID, corpID).Take(&linkedCompletion).Error != nil {
			return false
		}
	}
	parentID := orderID
	if parentID == nil && itemID != nil {
		parentID = &linkedItem.OrderID
	}
	if parentID == nil && completionID != nil {
		parentID = &linkedCompletion.OrderID
	}
	if parentID != nil {
		var parent models.Order
		if db.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id=? AND corp_id=?", *parentID, corpID).Take(&parent).Error != nil {
			return false
		}
	}
	if itemID != nil {
		if db.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id=? AND corp_id=?", *itemID, corpID).Take(&linkedItem).Error != nil {
			return false
		}
	}
	checks := []struct {
		table string
		id    *int64
	}{{"orders", orderID}, {"order_items", itemID}, {"production_completions", completionID}}
	for _, check := range checks {
		if check.id == nil {
			continue
		}
		var count int64
		if err := db.Table(check.table).Where("corp_id = ? AND id = ?", corpID, *check.id).Count(&count).Error; err != nil || count != 1 {
			return false
		}
	}
	if orderID != nil && itemID != nil {
		var count int64
		if err := db.Table("order_items").Where("corp_id = ? AND id = ? AND order_id = ?", corpID, *itemID, *orderID).Count(&count).Error; err != nil || count != 1 {
			return false
		}
	}
	if completionID != nil && itemID != nil {
		var count int64
		if err := db.Table("production_completions").Where("corp_id = ? AND id = ? AND order_item_id = ?", corpID, *completionID, *itemID).Count(&count).Error; err != nil || count != 1 {
			return false
		}
	}
	if completionID != nil && orderID != nil {
		var count int64
		if err := db.Table("production_completions").Where("corp_id=? AND id=? AND order_id=? AND status='confirmed'", corpID, *completionID, *orderID).Count(&count).Error; err != nil || count != 1 {
			return false
		}
	}
	return true
}

func (h *ProductionFinanceHandler) requireFinanceAccess(c *gin.Context) bool {
	return h.requireFinancePermission(c, false)
}

func (h *ProductionFinanceHandler) requireFinanceWriteAccess(c *gin.Context) bool {
	return h.requireFinancePermission(c, true)
}

func (h *ProductionFinanceHandler) requireFinancePermission(c *gin.Context, write bool) bool {
	if middleware.GetRoleType(c) == "admin" {
		return true
	}
	resourceKeys := []string{"finance", "monthly_output", "btn-view-finance-summary"}
	if write {
		resourceKeys = []string{"btn-manage-cost-entry"}
	}
	allowed, err := checkModuleAccess(h.db, c, resourceKeys...)
	if err != nil || !allowed {
		message := "无经营报表查看权限"
		if write {
			message = "无成本费用管理权限，请联系管理员"
		}
		response.Forbidden(c, response.CodeForbidden, message)
		return false
	}
	return true
}

func validMoney(value string) bool {
	if len(value) > 19 || !moneyPattern.MatchString(value) {
		return false
	}
	_, ok := new(big.Rat).SetString(value)
	return ok
}

func positiveDecimal(value string) bool {
	v, ok := new(big.Rat).SetString(value)
	return ok && v.Sign() > 0
}

func sameDecimal(a, b string) bool {
	x, ok := new(big.Rat).SetString(a)
	if !ok {
		return false
	}
	y, ok := new(big.Rat).SetString(b)
	return ok && x.Cmp(y) == 0
}

func validCostAmounts(gross, tax, cost string) bool {
	if !validPositiveMoney(gross) || !validMoney(tax) || !validPositiveMoney(cost) {
		return false
	}
	g, _ := new(big.Rat).SetString(gross)
	t, _ := new(big.Rat).SetString(tax)
	n, _ := new(big.Rat).SetString(cost)
	max, _ := new(big.Rat).SetString("9999999999999999.99")
	if g.Cmp(max) > 0 || t.Cmp(g) > 0 {
		return false
	}
	return n.Cmp(g) == 0 || n.Cmp(new(big.Rat).Sub(g, t)) == 0
}

func validPositiveMoney(value string) bool {
	if !validMoney(value) {
		return false
	}
	v, _ := new(big.Rat).SetString(value)
	return v.Sign() > 0
}
