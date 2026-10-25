package handlers

import (
	"github.com/gin-gonic/gin"
	"machining-erp/internal/accounting"
	"machining-erp/internal/middleware"
	"machining-erp/pkg/response"
	"strconv"
	"strings"
	"time"
)

func (h *ProductionFinanceHandler) DeliveryReconciliationList(c *gin.Context) {
	if !requireModuleAccess(h.db, c, "reconciliation") {
		return
	}
	corp := middleware.GetCorpID(c)
	page, size, offset := boundedPage(c, 50)
	if c.Query("mode") == "statements" {
		rows := []map[string]interface{}{}
		err := h.db.Raw("SELECT d.id,d.statement_no,d.customer_id,d.customer_name,DATE_FORMAT(d.business_date,'%Y-%m-%d') AS business_date,CAST(d.amount AS CHAR) AS amount,d.status,d.notes,d.reversal_reason FROM delivery_reconciliations d WHERE d.corp_id=? ORDER BY d.business_date DESC,d.id DESC LIMIT ? OFFSET ?", corp, size+1, offset).Scan(&rows).Error
		if err != nil {
			response.InternalError(c, response.CodeInternal, "读取对账单失败")
			return
		}
		rows, more := trimPage(rows, size)
		response.Success(c, gin.H{"items": rows, "has_more": more, "page": page})
		return
	}
	start, end := c.Query("start"), c.Query("end")
	if _, e := time.Parse("2006-01-02", start); e != nil {
		response.BadRequest(c, response.CodeInvalidRequest, "请选择送货开始日期")
		return
	}
	if _, e := time.Parse("2006-01-02", end); e != nil || end < start {
		response.BadRequest(c, response.CodeInvalidRequest, "送货结束日期无效")
		return
	}
	q := h.db.Raw(accounting.DeliverySelect+" WHERE r.corp_id=? AND r.kind='delivery' AND r.status='confirmed' AND r.business_date>=? AND r.business_date<=? AND (?='' OR o.customer_name LIKE ? OR o.customer_short_name LIKE ?) ORDER BY r.business_date DESC,r.id DESC LIMIT ? OFFSET ?", corp, start, end, c.Query("customer"), "%"+c.Query("customer")+"%", "%"+c.Query("customer")+"%", size+1, offset)
	rows := []accounting.DeliveryLine{}
	if err := q.Scan(&rows).Error; err != nil {
		response.InternalError(c, response.CodeInternal, "读取送货对账记录失败")
		return
	}
	rows, more := trimPage(rows, size)
	response.Success(c, gin.H{"items": rows, "has_more": more, "page": page})
}
func (h *ProductionFinanceHandler) CreateDeliveryReconciliation(c *gin.Context) {
	if !requireModuleAccess(h.db, c, "reconciliation") {
		return
	}
	var in accounting.ReconcileInput
	if c.ShouldBindJSON(&in) != nil {
		response.BadRequest(c, response.CodeInvalidRequest, "对账资料格式不正确")
		return
	}
	out, e := accounting.Reconcile(h.db.WithContext(c.Request.Context()), middleware.GetCorpID(c), middleware.GetUserID(c), in)
	if e != nil {
		response.BadRequest(c, response.CodeInvalidRequest, "对账未保存："+e.Error())
		return
	}
	response.Success(c, out)
}
func (h *ProductionFinanceHandler) DeliveryReconciliationDetail(c *gin.Context) {
	if !requireModuleAccess(h.db, c, "reconciliation") {
		return
	}
	id, _ := strconv.ParseInt(c.Param("statementId"), 10, 64)
	corp := middleware.GetCorpID(c)
	var header accounting.Statement
	if h.db.Where("corp_id=? AND id=?", corp, id).Take(&header).Error != nil {
		response.BadRequest(c, response.CodeInvalidRequest, "对账单不存在")
		return
	}
	rows := []map[string]interface{}{}
	if e := h.db.Table("delivery_reconciliation_lines").Where("corp_id=? AND reconciliation_id=?", corp, id).Order("id").Find(&rows).Error; e != nil {
		response.InternalError(c, response.CodeInternal, "读取对账明细失败")
		return
	}
	response.Success(c, gin.H{"header": header, "lines": rows})
}
func (h *ProductionFinanceHandler) ReverseDeliveryReconciliation(c *gin.Context) {
	if !requireModuleAccess(h.db, c, "reconciliation") {
		return
	}
	var in struct {
		Reason string `json:"reason"`
	}
	id, _ := strconv.ParseInt(c.Param("statementId"), 10, 64)
	if c.ShouldBindJSON(&in) != nil {
		response.BadRequest(c, response.CodeInvalidRequest, "请填写撤销原因")
		return
	}
	if e := accounting.ReverseStatement(h.db.WithContext(c.Request.Context()), middleware.GetCorpID(c), middleware.GetUserID(c), id, strings.TrimSpace(in.Reason)); e != nil {
		response.BadRequest(c, response.CodeInvalidRequest, e.Error())
		return
	}
	response.SuccessWithMessage(c, "已撤销对账，送货记录可重新选择；原对账明细仍保留", nil)
}
