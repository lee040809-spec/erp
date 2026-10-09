package handlers

import (
	"crypto/sha256"
	"fmt"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"machining-erp/internal/productionflow"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/gin-gonic/gin"
	"machining-erp/internal/middleware"
	"machining-erp/internal/models"
	"machining-erp/pkg/response"
	"machining-erp/pkg/utils"
)

// Historical status is a candidate for review, never evidence of qualified quantity.
func (h *ProductionFinanceHandler) PreviewHistory(c *gin.Context) {
	if !h.requireFinanceAccess(c) {
		return
	}
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	if page < 1 {
		page = 1
	}
	year := c.Query("year")
	if year != "" && !yearPattern.MatchString(year) {
		response.BadRequest(c, response.CodeInvalidRequest, "年份格式错误")
		return
	}
	query := h.db.Table("order_items i").Joins("JOIN orders o ON o.id=i.order_id AND o.corp_id=i.corp_id").
		Where("i.corp_id=? AND (i.status IN ('completed','delivered') OR COALESCE(hq.history_count,0)>0)", middleware.GetCorpID(c)).Joins(`LEFT JOIN (SELECT corp_id,order_item_id,
 SUM(CASE WHEN status='confirmed' THEN qualified_quantity ELSE 0 END) AS confirmed_quantity,
 SUM(CASE WHEN completion_no NOT LIKE 'HISTORY-%' THEN 1 ELSE 0 END) AS ordinary_count,
 SUM(CASE WHEN completion_no LIKE 'HISTORY-%' THEN 1 ELSE 0 END) AS history_count
 FROM production_completions GROUP BY corp_id,order_item_id) hq ON hq.corp_id=i.corp_id AND hq.order_item_id=i.id`).
		Where("COALESCE(hq.ordinary_count,0)=0 AND COALESCE(NULLIF(i.production_quantity,0),i.quantity)-COALESCE(i.scrap_quantity,0)>COALESCE(hq.confirmed_quantity,0)")
	// Missing dates remain visible so they can be reviewed and corrected.
	if year != "" {
		query = query.Where("YEAR(i.completion_date)=? OR i.completion_date IS NULL", year)
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		response.InternalError(c, response.CodeInternal, "读取历史候选数量失败")
		return
	}
	rows := make([]map[string]interface{}, 0)
	err := query.Select(`i.id AS order_item_id, i.order_id, o.order_number, COALESCE(o.customer_name,'') AS customer_name,
 i.part_name, i.quantity, COALESCE(NULLIF(i.production_quantity,0),i.quantity) AS production_quantity, COALESCE(i.scrap_quantity,0) AS scrap_quantity,
 DATE_FORMAT(i.completion_date,'%Y-%m-%d') AS business_date, CAST(i.unit_price AS CHAR) AS unit_price,
 COALESCE(i.supply_mode,'self_purchased') AS supply_mode, COALESCE(i.currency,'CNY') AS currency,
 i.tax_inclusive,CAST(COALESCE(hq.confirmed_quantity,0) AS UNSIGNED) AS confirmed_quantity,CAST(COALESCE(NULLIF(i.production_quantity,0),i.quantity)-COALESCE(i.scrap_quantity,0)-COALESCE(hq.confirmed_quantity,0) AS UNSIGNED) AS remaining_quantity`).Order("i.completion_date DESC,i.id DESC").Offset((page - 1) * 50).Limit(50).Find(&rows).Error
	if err != nil {
		response.InternalError(c, response.CodeInternal, "读取历史候选失败")
		return
	}
	response.Success(c, gin.H{"items": rows, "total": total, "page": page, "page_size": 50})
}

func (h *ProductionFinanceHandler) ImportHistory(c *gin.Context) {
	if !h.requireHistoryWriteAccess(c) {
		return
	}
	var req struct {
		ItemID         int64  `json:"order_item_id"`
		Quantity       int    `json:"qualified_quantity"`
		Date           string `json:"business_date"`
		ExpectedPrice  string `json:"expected_price"`
		Notes          string `json:"notes"`
		Reviewed       bool   `json:"reviewed"`
		SourceRef      string `json:"source_ref"`
		RequestID      string `json:"request_id"`
		ConfirmedPrice string `json:"confirmed_price"`
	}
	if c.ShouldBindJSON(&req) != nil || !req.Reviewed || req.ItemID <= 0 || req.Quantity <= 0 || utf8.RuneCountInString(req.Notes) > 500 {
		response.BadRequest(c, response.CodeInvalidRequest, "请核对合格数量、日期、单价；备注最多500字")
		return
	}
	req.SourceRef = strings.ToUpper(strings.TrimSpace(req.SourceRef))
	req.Notes = strings.TrimSpace(req.Notes)
	if utf8.RuneCountInString(req.SourceRef) > 80 || (req.SourceRef != "" && utf8.RuneCountInString(req.Notes) > 400) {
		response.BadRequest(c, response.CodeInvalidRequest, "请填写有效的原始批次依据编号（最多80字）和审核说明（分批导入最多400字）")
		return
	}
	if !checkEvidence(h.db, c, "source_ref", req.SourceRef, 80) || !checkEvidence(h.db, c, "review_notes", req.Notes, 400) {
		return
	}
	if req.RequestID != "" && !regexp.MustCompile(`^[A-Za-z0-9-]{1,36}$`).MatchString(req.RequestID) {
		response.BadRequest(c, response.CodeInvalidRequest, "请求编号无效")
		return
	}
	date, err := time.Parse("2006-01-02", req.Date)
	if err != nil || req.Date > time.Now().Format("2006-01-02") {
		response.BadRequest(c, response.CodeInvalidRequest, "完工日期无效或晚于今天")
		return
	}
	corpID := middleware.GetCorpID(c)
	month := date.Format("2006-01")
	tx := h.db.Begin()
	if tx.Error != nil {
		response.InternalError(c, response.CodeInternal, "启动导入事务失败")
		return
	}
	defer tx.Rollback()
	// Serialize import and month closing using the same month row.
	if err = tx.Exec("INSERT IGNORE INTO monthly_closings (corp_id,accounting_month,status,rule_version) VALUES (?,?,'open','v1')", corpID, month).Error; err != nil {
		response.InternalError(c, response.CodeInternal, "检查月结失败")
		return
	}
	var closing struct{ Status string }
	if err = tx.Raw("SELECT status FROM monthly_closings WHERE corp_id=? AND accounting_month=? FOR UPDATE", corpID, month).Scan(&closing).Error; err != nil {
		response.InternalError(c, response.CodeInternal, "检查月结失败")
		return
	}
	if closing.Status == "closed" {
		response.BadRequest(c, response.CodeInvalidRequest, "该月份已结账，不能导入")
		return
	}
	var parentID int64
	if err = tx.Table("order_items").Where("id=? AND corp_id=?", req.ItemID, corpID).Select("order_id").Scan(&parentID).Error; err != nil || parentID == 0 {
		response.ErrorWithCode(c, http.StatusNotFound, response.CodeNotFound, "零件不存在")
		return
	}
	var parent models.Order
	if err = tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id=? AND corp_id=?", parentID, corpID).Take(&parent).Error; err != nil {
		response.ErrorWithCode(c, http.StatusNotFound, response.CodeNotFound, "订单不存在")
		return
	}
	var item struct {
		OrderID            int64
		Quantity           int
		ProductionQuantity int
		ScrapQuantity      int
		Status             string
		UnitPrice          string
		SupplyMode         string
		Currency           string
		TaxInclusive       bool
	}
	err = tx.Raw("SELECT order_id,quantity,COALESCE(NULLIF(production_quantity,0),quantity) AS production_quantity,COALESCE(scrap_quantity,0) AS scrap_quantity,status,CAST(unit_price AS CHAR) AS unit_price,COALESCE(supply_mode,'self_purchased') AS supply_mode,COALESCE(currency,'CNY') AS currency,tax_inclusive FROM order_items WHERE id=? AND corp_id=? FOR UPDATE", req.ItemID, corpID).Scan(&item).Error
	if err != nil || item.OrderID == 0 {
		response.ErrorWithCode(c, http.StatusNotFound, response.CodeNotFound, "零件不存在")
		return
	}
	var prior struct {
		Count         int64
		Quantity      float64
		Billable      float64
		OrdinaryCount int64
	}
	if err = tx.Table("production_completions").Where("corp_id=? AND order_item_id=?", corpID, req.ItemID).Select("COUNT(*) AS count,COALESCE(SUM(CASE WHEN status='confirmed' THEN qualified_quantity ELSE 0 END),0) AS quantity,COALESCE(SUM(CASE WHEN status='confirmed' THEN billable_quantity ELSE 0 END),0) AS billable,COALESCE(SUM(completion_no NOT LIKE 'HISTORY-%'),0) AS ordinary_count").Scan(&prior).Error; err != nil {
		response.InternalError(c, response.CodeInternal, "读取已审核数量失败")
		return
	}
	if item.Status != "completed" && item.Status != "delivered" && prior.Count == 0 {
		response.BadRequest(c, response.CodeInvalidRequest, "仅可导入历史已完工或已交货零件")
		return
	}
	if !positiveDecimal(item.UnitPrice) || !sameDecimal(item.UnitPrice, req.ExpectedPrice) || item.Currency != "CNY" {
		response.BadRequest(c, response.CodeInvalidRequest, "单价缺失、已变化或币种不是人民币，请刷新并核对订单")
		return
	}
	price := item.UnitPrice
	if req.ConfirmedPrice != "" {
		price = req.ConfirmedPrice
	}
	if !regexp.MustCompile(`^\d{1,11}(\.\d{1,4})?$`).MatchString(price) || !positiveDecimal(price) {
		response.BadRequest(c, response.CodeInvalidRequest, "审核单价必须大于0，最多11位整数和4位小数")
		return
	}
	number := fmt.Sprintf("HISTORY-%d", req.ItemID)
	notes := "历史审核导入：" + req.Notes
	if req.SourceRef != "" || req.RequestID != "" {
		if req.SourceRef == "" {
			number = fmt.Sprintf("HISTORY-%d-AUTO-%s", req.ItemID, req.RequestID)
		} else {
			digest := sha256.Sum256([]byte(req.SourceRef))
			number = fmt.Sprintf("HISTORY-%d-%x", req.ItemID, digest[:16])
			notes = "历史审核导入[" + req.SourceRef + "]：" + req.Notes
		}
		var existing models.ProductionCompletion
		err = tx.Where("corp_id=? AND completion_no=?", corpID, number).Take(&existing).Error
		if err == nil {
			if existing.Status != "confirmed" || existing.BusinessDate.Format("2006-01-02") != req.Date || !sameDecimal(existing.QualifiedQuantity, strconv.Itoa(req.Quantity)) || !sameDecimal(existing.UnitPriceSnapshot, price) || existing.Notes == nil || *existing.Notes != notes {
				response.BadRequest(c, response.CodeInvalidRequest, "原始批次依据编号已使用，内容不一致；请核对原始记录")
				return
			}
			response.SuccessWithMessage(c, "该历史批次已导入，无需重复记账", gin.H{"completion_no": number})
			return
		}
		if err != gorm.ErrRecordNotFound {
			response.InternalError(c, response.CodeInternal, "检查原始批次编号失败")
			return
		}
	}
	tracked, e := productionflow.Enabled(tx, corpID, req.ItemID)
	if e != nil || tracked {
		response.BadRequest(c, response.CodeInvalidRequest, "已启用工序数量跟踪，请通过分批生产确认合格完工，不使用历史补录")
		return
	}
	if prior.OrdinaryCount > 0 || (req.SourceRef == "" && req.RequestID == "" && prior.Count > 0) {
		response.BadRequest(c, response.CodeInvalidRequest, "该零件已有完工台账；历史分批请刷新页面取得系统批次编号，普通报工零件不能转为历史导入")
		return
	}
	remaining := float64(item.ProductionQuantity-item.ScrapQuantity) - prior.Quantity
	if float64(req.Quantity) > remaining {
		response.BadRequest(c, response.CodeInvalidRequest, fmt.Sprintf("本次合格数量超过待审核剩余数量 %.0f", remaining))
		return
	}
	billable := max(0, min(float64(req.Quantity), float64(item.Quantity)-prior.Billable))
	err = tx.Exec(`INSERT INTO production_completions (corp_id,order_id,order_item_id,completion_no,business_date,qualified_quantity,billable_quantity,unit_price_snapshot,output_value,supply_mode_snapshot,currency,tax_inclusive,status,notes,created_by,confirmed_at,created_at)
 VALUES (?,?,?,?,?,?,?,?,ROUND(?*CAST(? AS DECIMAL(15,4)),2),?,?,?,'confirmed',?,?,NOW(3),NOW(3))`, corpID, item.OrderID, req.ItemID, number, req.Date, req.Quantity, billable, price, billable, price, item.SupplyMode, item.Currency, item.TaxInclusive, notes, middleware.GetUserID(c)).Error
	if err != nil {
		response.InternalError(c, response.CodeInternal, "保存历史完工失败")
		return
	}
	if prior.Quantity+float64(req.Quantity) >= float64(item.Quantity) {
		finalDate, dateErr := orderFulfilledDate(tx, corpID, req.ItemID, item.Quantity)
		if dateErr != nil {
			response.InternalError(c, response.CodeInternal, "读取完工日期失败")
			return
		}
		if err = tx.Exec("UPDATE order_items SET status=CASE WHEN status='delivered' THEN 'delivered' ELSE 'completed' END,completion_date=? WHERE id=? AND corp_id=?", finalDate, req.ItemID, corpID).Error; err != nil {
			response.InternalError(c, response.CodeInternal, "同步历史完工状态失败")
			return
		}
		var statuses []string
		if err = tx.Table("order_items").Where("order_id=? AND corp_id=?", item.OrderID, corpID).Pluck("status", &statuses).Error; err != nil {
			response.InternalError(c, response.CodeInternal, "读取历史订单状态失败")
			return
		}
		if err = tx.Model(&parent).Update("status", utils.CalculateStatus(statuses)).Error; err != nil {
			response.InternalError(c, response.CodeInternal, "同步历史订单状态失败")
			return
		}
	}
	if tx.Commit().Error != nil {
		response.InternalError(c, response.CodeInternal, "提交历史导入失败")
		return
	}
	response.SuccessWithMessage(c, "历史完工已审核计入产值", gin.H{"completion_no": number})
}

func (h *ProductionFinanceHandler) FinanceDetails(c *gin.Context) {
	if !h.requireFinanceAccess(c) {
		return
	}
	month := c.Query("month")
	if _, err := time.Parse("2006-01", month); err != nil {
		response.BadRequest(c, response.CodeInvalidRequest, "请选择有效月份")
		return
	}
	page, size, offset := boundedPage(c, 50)
	rows := make([]map[string]interface{}, 0)
	err := h.db.Raw(`SELECT p.id,p.completion_no,p.order_item_id,o.order_number,COALESCE(o.customer_name,'') AS customer_name,i.part_name,
 DATE_FORMAT(p.business_date,'%Y-%m-%d') AS business_date,CAST(p.qualified_quantity AS CHAR) AS qualified_quantity, CAST(p.billable_quantity AS CHAR) AS billable_quantity, CAST(p.qualified_quantity-p.billable_quantity AS CHAR) AS reserve_quantity,
 CAST(p.unit_price_snapshot AS CHAR) AS unit_price_snapshot,CAST(p.output_value AS CHAR) AS output_value,p.supply_mode_snapshot,p.notes
 FROM production_completions p LEFT JOIN orders o ON o.id=p.order_id AND o.corp_id=p.corp_id
 LEFT JOIN order_items i ON i.id=p.order_item_id AND i.corp_id=p.corp_id
 WHERE p.corp_id=? AND p.status='confirmed' AND p.business_date>=? AND p.business_date<DATE_ADD(?,INTERVAL 1 MONTH)
 ORDER BY p.business_date DESC,p.id DESC LIMIT ? OFFSET ?`, middleware.GetCorpID(c), month+"-01", month+"-01", size+1, offset).Scan(&rows).Error
	if err != nil {
		response.InternalError(c, response.CodeInternal, "读取产值明细失败")
		return
	}
	costs := make([]models.CostEntry, 0)
	if err = h.db.Where("corp_id=? AND accounting_month=? AND status='confirmed'", middleware.GetCorpID(c), month).Order("business_date DESC,id DESC").Limit(size + 1).Offset(offset).Find(&costs).Error; err != nil {
		response.InternalError(c, response.CodeInternal, "读取费用明细失败")
		return
	}
	allocations := make([]map[string]interface{}, 0)
	err = h.db.Raw(`SELECT a.id,a.cost_entry_id,e.cost_type,DATE_FORMAT(e.business_date,'%Y-%m-%d') AS cost_date,
a.production_completion_id,p.completion_no,CAST(a.allocated_amount AS CHAR) AS allocated_amount,a.allocation_basis,a.created_by
FROM cost_allocations a JOIN cost_entries e ON e.id=a.cost_entry_id AND e.corp_id=a.corp_id AND e.status='confirmed'
JOIN production_completions p ON p.id=a.production_completion_id AND p.corp_id=a.corp_id AND p.status='confirmed'
WHERE a.corp_id=? AND p.business_date>=? AND p.business_date<DATE_ADD(?,INTERVAL 1 MONTH)
ORDER BY a.id DESC LIMIT ? OFFSET ?`, middleware.GetCorpID(c), month+"-01", month+"-01", size+1, offset).Scan(&allocations).Error
	if err != nil {
		response.InternalError(c, response.CodeInternal, "读取分摊明细失败")
		return
	}
	var batchCount int64
	if err := h.db.Model(&models.ProductionCompletion{}).Where("corp_id=? AND status='confirmed' AND business_date>=? AND business_date<DATE_ADD(?,INTERVAL 1 MONTH)", middleware.GetCorpID(c), month+"-01", month+"-01").Count(&batchCount).Error; err != nil {
		response.InternalError(c, response.CodeInternal, "读取批次总数失败")
		return
	}
	rows, batchesMore := trimPage(rows, size)
	costs, costsMore := trimPage(costs, size)
	allocations, allocationsMore := trimPage(allocations, size)
	response.Success(c, gin.H{"page": page, "has_completions": batchCount > 0, "page_size": size, "has_more": batchesMore || costsMore || allocationsMore, "completions": rows, "costs": costs, "allocations": allocations})
}
