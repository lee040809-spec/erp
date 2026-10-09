package handlers

import (
	"github.com/gin-gonic/gin"
	"machining-erp/internal/middleware"
	"machining-erp/internal/outsourcing"
	"machining-erp/pkg/response"
	"math/big"
	"time"
)

func (h *ProductionFinanceHandler) PendingCosts(c *gin.Context) {
	if !h.requireFinanceAccess(c) {
		return
	}
	page, size, offset := boundedPage(c, 50)
	rows := make([]map[string]interface{}, 0)
	err := h.db.Raw(`SELECT e.id,e.cost_type,DATE_FORMAT(e.business_date,'%Y-%m-%d') AS business_date,e.order_id,e.order_item_id,e.supplier_name,e.notes,
 CAST(e.net_amount AS CHAR) AS net_amount,CAST(e.net_amount-COALESCE(a.total,0) AS CHAR) AS remaining_amount
 FROM cost_entries e LEFT JOIN (SELECT cost_entry_id,corp_id,SUM(allocated_amount) AS total FROM cost_allocations GROUP BY corp_id,cost_entry_id) a
 ON a.cost_entry_id=e.id AND a.corp_id=e.corp_id
 WHERE e.corp_id=? AND e.status='confirmed' AND e.cost_type<>'period' AND e.net_amount>COALESCE(a.total,0)
 ORDER BY e.business_date,e.id LIMIT ? OFFSET ?`, middleware.GetCorpID(c), size+1, offset).Scan(&rows).Error
	if err != nil {
		response.InternalError(c, response.CodeInternal, "读取待分摊成本失败")
		return
	}
	rows, more := trimPage(rows, size)
	response.Success(c, gin.H{"items": rows, "page": page, "has_more": more})
}

func (h *ProductionFinanceHandler) AllocateCost(c *gin.Context) {
	if !h.requireFinanceWriteAccess(c) {
		return
	}
	var req struct {
		CostID       int64  `json:"cost_entry_id"`
		CompletionID int64  `json:"production_completion_id"`
		Amount       string `json:"allocated_amount"`
		Basis        string `json:"allocation_basis"`
	}
	if c.ShouldBindJSON(&req) != nil || req.CostID <= 0 || req.CompletionID <= 0 || !validPositiveMoney(req.Amount) || len(req.Basis) > 100 {
		response.BadRequest(c, response.CodeInvalidRequest, "请选择费用、完工批次，填写正数金额和分摊依据（最多100字）")
		return
	}
	if !checkEvidence(h.db, c, "allocation_basis", req.Basis, 100) {
		return
	}
	corpID := middleware.GetCorpID(c)
	tx := h.db.Begin()
	if tx.Error != nil {
		response.InternalError(c, response.CodeInternal, "启动成本分摊失败")
		return
	}
	defer tx.Rollback()
	var batch struct {
		DeliveryBatchID    *int64
		Month              string
		OrderID            int64
		OrderItemID        int64
		SupplyModeSnapshot string
		Currency           string
	}
	err := tx.Raw("SELECT DATE_FORMAT(business_date,'%Y-%m') AS month,delivery_batch_id,order_id,order_item_id,supply_mode_snapshot,currency FROM production_completions WHERE id=? AND corp_id=? AND status='confirmed'", req.CompletionID, corpID).Scan(&batch).Error
	if err != nil || batch.OrderID == 0 || batch.Currency != "CNY" {
		response.BadRequest(c, response.CodeInvalidRequest, "完工批次不存在或币种不匹配")
		return
	}
	if err = tx.Exec("INSERT IGNORE INTO monthly_closings (corp_id,accounting_month,status,rule_version) VALUES (?,?,'open','v1')", corpID, batch.Month).Error; err != nil {
		response.InternalError(c, response.CodeInternal, "检查月份失败")
		return
	}
	var closing struct{ Status string }
	if err = tx.Raw("SELECT status FROM monthly_closings WHERE corp_id=? AND accounting_month=? FOR UPDATE", corpID, batch.Month).Scan(&closing).Error; err != nil {
		response.InternalError(c, response.CodeInternal, "检查月份失败")
		return
	}
	if closing.Status == "closed" {
		response.BadRequest(c, response.CodeInvalidRequest, "完工月份已结账，不能分摊成本")
		return
	}
	var cost struct {
		DeliveryBatchID        *int64
		ID                     int64
		NetAmount              string
		CostType               string
		OrderID                *int64
		OrderItemID            *int64
		ProductionCompletionID *int64
		Currency               string
		AccountingMonth        string
	}
	err = tx.Raw("SELECT id,CAST(net_amount AS CHAR) AS net_amount,delivery_batch_id,cost_type,order_id,order_item_id,production_completion_id,currency,accounting_month FROM cost_entries WHERE id=? AND corp_id=? AND status='confirmed' FOR UPDATE", req.CostID, corpID).Scan(&cost).Error
	if err != nil || cost.ID == 0 || cost.CostType == "period" || cost.Currency != "CNY" {
		response.BadRequest(c, response.CodeInvalidRequest, "费用不存在、币种不匹配或属于期间费用")
		return
	}
	if (cost.DeliveryBatchID != nil && !outsourcing.SameID(cost.DeliveryBatchID, batch.DeliveryBatchID)) || (cost.OrderID != nil && *cost.OrderID != batch.OrderID) || (cost.OrderItemID != nil && *cost.OrderItemID != batch.OrderItemID) || (cost.ProductionCompletionID != nil && *cost.ProductionCompletionID != req.CompletionID) {
		response.BadRequest(c, response.CodeInvalidRequest, "费用关联订单、零件或指定批次与目标完工批次不一致")
		return
	}
	if cost.CostType == "material" && batch.SupplyModeSnapshot == "customer_supplied" {
		response.BadRequest(c, response.CodeInvalidRequest, "客户供料批次不能分摊自购原材料成本")
		return
	}
	var allocated struct {
		Total      string
		Duplicates int64
	}
	if err = tx.Raw("SELECT CAST(COALESCE(SUM(allocated_amount),0) AS CHAR) AS total,COALESCE(SUM(production_completion_id=?),0) AS duplicates FROM cost_allocations WHERE corp_id=? AND cost_entry_id=?", req.CompletionID, corpID, req.CostID).Scan(&allocated).Error; err != nil {
		response.InternalError(c, response.CodeInternal, "检查已分摊金额失败")
		return
	}
	if allocated.Duplicates > 0 {
		response.BadRequest(c, response.CodeInvalidRequest, "该费用已经分摊到此批次，不能重复提交")
		return
	}
	if !allocationFits(cost.NetAmount, allocated.Total, req.Amount) {
		response.BadRequest(c, response.CodeInvalidRequest, "分摊金额超过该费用尚未分摊的余额")
		return
	}
	err = tx.Exec("INSERT INTO cost_allocations (corp_id,cost_entry_id,production_completion_id,allocated_amount,allocation_basis,rule_version,created_by,created_at) VALUES (?,?,?,?,?,'manual-v1',?,?)", corpID, req.CostID, req.CompletionID, req.Amount, req.Basis, middleware.GetUserID(c), time.Now()).Error
	if err != nil {
		response.InternalError(c, response.CodeInternal, "保存分摊失败")
		return
	}
	if tx.Commit().Error != nil {
		response.InternalError(c, response.CodeInternal, "提交分摊失败")
		return
	}
	response.SuccessWithMessage(c, "成本已计入完工月份", nil)
}

func allocationFits(total, allocated, addition string) bool {
	t, ok := new(big.Rat).SetString(total)
	if !ok {
		return false
	}
	a, ok := new(big.Rat).SetString(allocated)
	if !ok {
		return false
	}
	n, ok := new(big.Rat).SetString(addition)
	return ok && a.Sign() >= 0 && n.Sign() > 0 && new(big.Rat).Add(a, n).Cmp(t) <= 0
}
