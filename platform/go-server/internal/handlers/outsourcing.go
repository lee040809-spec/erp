package handlers

import (
	"errors"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"machining-erp/internal/middleware"
	"machining-erp/internal/models"
	"machining-erp/internal/outsourcing"
	"machining-erp/pkg/response"
	"strconv"
)

type OutsourcingHandler struct{ db *gorm.DB }

func NewOutsourcingHandler(db *gorm.DB) *OutsourcingHandler { return &OutsourcingHandler{db} }
func (h *OutsourcingHandler) List(c *gin.Context) {
	if !flowRead(h.db, c) {
		return
	}
	item, _ := strconv.ParseInt(c.Query("order_item_id"), 10, 64)
	if item <= 0 {
		flowResult(c, errors.New("请选择零件"), nil)
		return
	}
	corp := middleware.GetCorpID(c)
	var n int64
	if e := h.db.Model(&models.OrderItem{}).Where("corp_id=? AND id=?", corp, item).Count(&n).Error; e != nil || n == 0 {
		flowResult(c, errors.New("零件不存在"), nil)
		return
	}
	page, size, offset := boundedPage(c, 30)
	rows := []models.OutsourcingDispatch{}
	if e := h.db.Where("corp_id=? AND order_item_id=?", corp, item).Order("id DESC").Limit(size + 1).Offset(offset).Find(&rows).Error; e != nil {
		flowResult(c, e, nil)
		return
	}
	rows, more := trimPage(rows, size)
	for i := range rows {
		rows[i].EditToken = outsourcing.DispatchToken(rows[i])
	}
	ids := []int64{}
	for _, r := range rows {
		ids = append(ids, r.ID)
	}
	settlements := []models.OutsourcingSettlement{}
	events := []models.OutsourcingEvent{}
	if len(ids) > 0 {
		if e := h.db.Where("corp_id=? AND dispatch_id IN ?", corp, ids).Find(&settlements).Error; e != nil {
			flowResult(c, e, nil)
			return
		}
		if e := h.db.Where("corp_id=? AND dispatch_id IN ?", corp, ids).Order("id DESC").Limit(200).Find(&events).Error; e != nil {
			flowResult(c, e, nil)
			return
		}
	}
	var data interface{} = gin.H{"items": rows, "settlements": settlements, "events": events, "page": page, "has_more": more}
	if !canViewOrderAmounts(h.db, c) {
		data = withoutAmounts(data)
	}
	response.Success(c, data)
}
func (h *OutsourcingHandler) Dispatch(c *gin.Context) {
	if !flowWrite(h.db, c) {
		return
	}
	var in outsourcing.DispatchInput
	if c.ShouldBindJSON(&in) != nil {
		flowResult(c, errors.New("外协发出格式错误"), nil)
		return
	}
	if !checkEvidence(h.db, c, "operation_notes", in.Notes, 500) {
		return
	}
	out, err := outsourcing.Dispatch(h.db, middleware.GetCorpID(c), middleware.GetUserID(c), in)
	var data interface{} = out
	if !canViewOrderAmounts(h.db, c) {
		data = withoutAmounts(data)
	}
	flowResult(c, err, data)
}
func (h *OutsourcingHandler) Receive(c *gin.Context) {
	if !flowWrite(h.db, c) {
		return
	}
	var in outsourcing.ReturnInput
	if c.ShouldBindJSON(&in) != nil {
		flowResult(c, errors.New("回厂质检格式错误"), nil)
		return
	}
	if !checkEvidence(h.db, c, "operation_notes", in.Notes, 500) {
		return
	}
	flowResult(c, outsourcing.Receive(h.db, middleware.GetCorpID(c), flowID(c, "dispatchId"), middleware.GetUserID(c), in), nil)
}
func (h *OutsourcingHandler) Rework(c *gin.Context) {
	if !flowWrite(h.db, c) {
		return
	}
	var in outsourcing.ReworkInput
	if c.ShouldBindJSON(&in) != nil {
		flowResult(c, errors.New("返工格式错误"), nil)
		return
	}
	if !checkEvidence(h.db, c, "operation_notes", in.Notes, 500) {
		return
	}
	flowResult(c, outsourcing.SendRework(h.db, middleware.GetCorpID(c), flowID(c, "dispatchId"), middleware.GetUserID(c), in), nil)
}
func (h *OutsourcingHandler) Settle(c *gin.Context) {
	if !NewProductionFinanceHandler(h.db).requireFinanceWriteAccess(c) {
		return
	}
	var in outsourcing.SettlementInput
	if c.ShouldBindJSON(&in) != nil {
		flowResult(c, errors.New("对账格式错误"), nil)
		return
	}
	if !checkEvidence(h.db, c, "review_notes", in.Notes, 500) {
		return
	}
	out, err := outsourcing.Settle(h.db, middleware.GetCorpID(c), flowID(c, "dispatchId"), middleware.GetUserID(c), in)
	flowResult(c, err, out)
}

func (h *OutsourcingHandler) CostStatus(c *gin.Context) {
	if !NewProductionFinanceHandler(h.db).requireFinanceAccess(c) {
		return
	}
	var result struct {
		ID        int64  `json:"cost_entry_id"`
		Remaining string `json:"remaining_amount"`
	}
	e := h.db.Raw(`SELECT e.id,CAST(e.net_amount-COALESCE(a.amount,0) AS CHAR) AS remaining FROM outsourcing_settlements s JOIN cost_entries e ON e.id=s.cost_entry_id AND e.corp_id=s.corp_id LEFT JOIN (SELECT corp_id,cost_entry_id,SUM(allocated_amount) AS amount FROM cost_allocations GROUP BY corp_id,cost_entry_id) a ON a.corp_id=e.corp_id AND a.cost_entry_id=e.id WHERE s.corp_id=? AND s.dispatch_id=? AND e.status='confirmed'`, middleware.GetCorpID(c), flowID(c, "dispatchId")).Scan(&result).Error
	if e != nil || result.ID == 0 {
		flowResult(c, errors.New("该外协单没有可分摊费用"), nil)
		return
	}
	flowResult(c, nil, result)
}
