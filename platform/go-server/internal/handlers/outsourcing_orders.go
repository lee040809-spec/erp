package handlers

import (
	"errors"
	"github.com/gin-gonic/gin"
	"machining-erp/internal/middleware"
	"machining-erp/internal/models"
	"machining-erp/internal/outsourcing"
	"strconv"
	"strings"
)

func (h *OutsourcingHandler) Suppliers(c *gin.Context) {
	if !flowRead(h.db, c) {
		return
	}
	corp := middleware.GetCorpID(c)
	page, size, offset := boundedPage(c, 30)
	q := h.db.Where("corp_id=?", corp)
	if c.Query("active") == "1" {
		q = q.Where("active=1")
	}
	if s := strings.TrimSpace(c.Query("search")); s != "" {
		pattern := "%" + s + "%"
		q = q.Where("name LIKE ? OR contact_name LIKE ? OR phone LIKE ? OR capabilities LIKE ?", pattern, pattern, pattern, pattern)
	}
	rows := []models.OutsourcingSupplier{}
	if e := q.Order("active DESC,id DESC").Limit(size + 1).Offset(offset).Find(&rows).Error; e != nil {
		flowResult(c, e, nil)
		return
	}
	rows, more := trimPage(rows, size)
	flowResult(c, nil, gin.H{"items": rows, "page": page, "has_more": more, "can_manage": hasModuleAccess(h.db, c, "orders", "production_orders", "production_dashboard")})
}
func (h *OutsourcingHandler) SaveSupplier(c *gin.Context) {
	if !flowWrite(h.db, c) {
		return
	}
	if c.Request.Method == "PATCH" && flowID(c, "supplierId") <= 0 {
		flowResult(c, errors.New("供应商编号无效"), nil)
		return
	}
	var in outsourcing.SupplierInput
	if c.ShouldBindJSON(&in) != nil {
		flowResult(c, errors.New("供应商资料格式错误"), nil)
		return
	}
	out, e := outsourcing.SaveSupplier(h.db, middleware.GetCorpID(c), flowID(c, "supplierId"), in)
	flowResult(c, e, out)
}
func (h *OutsourcingHandler) Orders(c *gin.Context) {
	if !flowRead(h.db, c) {
		return
	}
	corp := middleware.GetCorpID(c)
	page, size, offset := boundedPage(c, 20)
	item, _ := strconv.ParseInt(c.Query("order_item_id"), 10, 64)
	if item <= 0 {
		flowResult(c, errors.New("请选择零件"), nil)
		return
	}
	var n int64
	if e := h.db.Model(&models.OrderItem{}).Where("corp_id=? AND id=?", corp, item).Count(&n).Error; e != nil || n == 0 {
		flowResult(c, errors.New("零件不存在"), nil)
		return
	}
	rows := []models.OutsourcingOrder{}
	if e := h.db.Where("corp_id=? AND order_item_id=?", corp, item).Order("id DESC").Limit(size + 1).Offset(offset).Find(&rows).Error; e != nil {
		flowResult(c, e, nil)
		return
	}
	rows, more := trimPage(rows, size)
	ids := []int64{}
	for _, o := range rows {
		ids = append(ids, o.ID)
	}
	lines := []models.OutsourcingOrderLine{}
	if len(ids) > 0 {
		if e := h.db.Where("corp_id=? AND outsourcing_order_id IN ?", corp, ids).Order("sequence,id").Find(&lines).Error; e != nil {
			flowResult(c, e, nil)
			return
		}
	}
	for i := range rows {
		rows[i].Lines = []models.OutsourcingOrderLine{}
		for _, l := range lines {
			if l.OutsourcingOrderID == rows[i].ID {
				rows[i].Lines = append(rows[i].Lines, l)
			}
		}
		rows[i].EditToken = outsourcing.OrderToken(rows[i])
	}
	var data interface{} = gin.H{"items": rows, "page": page, "has_more": more}
	if !canViewOrderAmounts(h.db, c) {
		data = withoutAmounts(data)
	}
	flowResult(c, nil, data)
}
func (h *OutsourcingHandler) UpdateOrder(c *gin.Context) {
	if !flowWrite(h.db, c) {
		return
	}
	var in struct {
		outsourcing.OrderInput
		EditToken string `json:"edit_token"`
	}
	if c.ShouldBindJSON(&in) != nil || flowID(c, "outsourceOrderId") <= 0 {
		flowResult(c, errors.New("修改信息格式错误"), nil)
		return
	}
	if !checkEvidence(h.db, c, "operation_notes", in.Notes, 500) {
		return
	}
	// Prices hidden from the current role are preserved, rather than overwritten with zero.
	if !canViewOrderAmounts(h.db, c) {
		var old models.OutsourcingOrder
		if e := h.db.Where("corp_id=? AND id=?", middleware.GetCorpID(c), flowID(c, "outsourceOrderId")).Take(&old).Error; e != nil {
			flowResult(c, errors.New("外协订单不存在"), nil)
			return
		}
		in.UnitPrice = old.UnitPrice
		lines := []models.OutsourcingOrderLine{}
		if e := h.db.Where("corp_id=? AND outsourcing_order_id=?", middleware.GetCorpID(c), old.ID).Find(&lines).Error; e != nil {
			flowResult(c, e, nil)
			return
		}
		prices := map[int64]string{}
		for _, l := range lines {
			prices[l.OrderProcessID] = l.UnitPrice
		}
		for i := range in.Lines {
			in.Lines[i].UnitPrice = "0"
			if p, ok := prices[in.Lines[i].ProcessID]; ok {
				in.Lines[i].UnitPrice = p
			}
		}
	}
	out, e := outsourcing.UpdateOrder(h.db, middleware.GetCorpID(c), middleware.GetUserID(c), flowID(c, "outsourceOrderId"), in.OrderInput, in.EditToken)
	var data interface{} = out
	if !canViewOrderAmounts(h.db, c) {
		data = withoutAmounts(data)
	}
	flowResult(c, e, data)
}
func (h *OutsourcingHandler) DeleteOrder(c *gin.Context) {
	if !flowWrite(h.db, c) {
		return
	}
	var in outsourcing.RemoveInput
	if c.ShouldBindJSON(&in) != nil || flowID(c, "outsourceOrderId") <= 0 {
		flowResult(c, errors.New("删除信息格式错误"), nil)
		return
	}
	flowResult(c, outsourcing.DeleteOrder(h.db, middleware.GetCorpID(c), flowID(c, "outsourceOrderId"), middleware.GetUserID(c), in), nil)
}
func (h *OutsourcingHandler) CreateOrder(c *gin.Context) {
	if !flowWrite(h.db, c) {
		return
	}
	var in outsourcing.OrderInput
	if c.ShouldBindJSON(&in) != nil {
		flowResult(c, errors.New("外协下单格式错误"), nil)
		return
	}
	if !checkEvidence(h.db, c, "operation_notes", in.Notes, 500) {
		return
	}
	out, e := outsourcing.CreateOrder(h.db, middleware.GetCorpID(c), middleware.GetUserID(c), in)
	var data interface{} = out
	if !canViewOrderAmounts(h.db, c) {
		data = withoutAmounts(data)
	}
	flowResult(c, e, data)
}
func (h *OutsourcingHandler) CancelOrder(c *gin.Context) {
	if !flowWrite(h.db, c) {
		return
	}
	flowResult(c, outsourcing.CancelOrder(h.db, middleware.GetCorpID(c), flowID(c, "outsourceOrderId")), nil)
}
func (h *OutsourcingHandler) DispatchOrder(c *gin.Context) {
	if !flowWrite(h.db, c) {
		return
	}
	var in outsourcing.OrderDispatchInput
	if c.ShouldBindJSON(&in) != nil {
		flowResult(c, errors.New("实物发出格式错误"), nil)
		return
	}
	if !checkEvidence(h.db, c, "operation_notes", in.Notes, 500) {
		return
	}
	out, e := outsourcing.DispatchOrder(h.db, middleware.GetCorpID(c), flowID(c, "outsourceOrderId"), middleware.GetUserID(c), in)
	var data interface{} = out
	if !canViewOrderAmounts(h.db, c) {
		data = withoutAmounts(data)
	}
	flowResult(c, e, data)
}
