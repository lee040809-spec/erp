package handlers

import (
	"errors"
	"github.com/gin-gonic/gin"
	"machining-erp/internal/middleware"
	"machining-erp/internal/outsourcing"
)

func (h *OutsourcingHandler) UpdateDispatch(c *gin.Context) {
	if !flowWrite(h.db, c) {
		return
	}
	var in outsourcing.DispatchEditInput
	if c.ShouldBindJSON(&in) != nil || flowID(c, "dispatchId") <= 0 {
		flowResult(c, errors.New("修改信息格式错误"), nil)
		return
	}
	if !checkEvidence(h.db, c, "operation_notes", in.Notes, 500) {
		return
	}
	out, e := outsourcing.UpdateDispatch(h.db, middleware.GetCorpID(c), flowID(c, "dispatchId"), middleware.GetUserID(c), in)
	var data interface{} = out
	if !canViewOrderAmounts(h.db, c) {
		data = withoutAmounts(data)
	}
	flowResult(c, e, data)
}
func (h *OutsourcingHandler) DeleteDispatch(c *gin.Context) {
	if !flowWrite(h.db, c) {
		return
	}
	var in outsourcing.RemoveInput
	if c.ShouldBindJSON(&in) != nil || flowID(c, "dispatchId") <= 0 {
		flowResult(c, errors.New("撤销信息格式错误"), nil)
		return
	}
	flowResult(c, outsourcing.CancelDispatch(h.db, middleware.GetCorpID(c), flowID(c, "dispatchId"), middleware.GetUserID(c), in), nil)
}
