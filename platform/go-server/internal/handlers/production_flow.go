package handlers

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"machining-erp/internal/middleware"
	"machining-erp/internal/models"
	"machining-erp/internal/productionflow"
	"machining-erp/pkg/response"
	"strconv"
)

type ProductionFlowHandler struct{ db *gorm.DB }

func NewProductionFlowHandler(db *gorm.DB) *ProductionFlowHandler { return &ProductionFlowHandler{db} }
func flowID(c *gin.Context, key string) int64 {
	id, _ := strconv.ParseInt(c.Param(key), 10, 64)
	return id
}
func flowRead(db *gorm.DB, c *gin.Context) bool {
	return requireModuleAccess(db, c, "orders", "production_orders", "production_dashboard", "dashboard", "production_delivery", "monthly_output")
}
func flowWrite(db *gorm.DB, c *gin.Context) bool {
	return requireModuleAccess(db, c, "orders", "production_orders", "production_dashboard")
}
func flowResult(c *gin.Context, err error, data interface{}) {
	if err != nil {
		response.BadRequest(c, response.CodeInvalidRequest, err.Error())
		return
	}
	response.Success(c, data)
}
func (h *ProductionFlowHandler) Read(c *gin.Context) {
	if !flowRead(h.db, c) {
		return
	}
	s, err := productionflow.Read(h.db, middleware.GetCorpID(c), flowID(c, "itemId"))
	if err != nil {
		flowResult(c, err, nil)
		return
	}
	var data interface{} = s
	if !canViewOrderAmounts(h.db, c) {
		data = withoutAmounts(data)
	}
	response.Success(c, gin.H{"flow": data, "can_manage": hasModuleAccess(h.db, c, "orders", "production_orders", "production_dashboard"), "can_complete": canConfirmProduction(h.db, c), "can_settle": NewProductionFinanceHandler(h.db).canManageFinanceCosts(c)})
}
func (h *ProductionFlowHandler) Activate(c *gin.Context) {
	if !flowWrite(h.db, c) {
		return
	}
	var in struct {
		Baseline []models.ProcessBaseline `json:"baseline"`
	}
	if c.ShouldBindJSON(&in) != nil {
		response.BadRequest(c, response.CodeInvalidRequest, "数量基线格式错误")
		return
	}
	err := h.db.Transaction(func(tx *gorm.DB) error {
		item, e := productionflow.LockItem(tx, middleware.GetCorpID(c), flowID(c, "itemId"))
		if e != nil {
			return e
		}
		return productionflow.Activate(tx, item, in.Baseline, middleware.GetUserID(c))
	})
	flowResult(c, err, nil)
}
func (h *ProductionFlowHandler) Batches(c *gin.Context) {
	if !flowWrite(h.db, c) {
		return
	}
	var in struct {
		Batches  []models.DeliveryBatchInput `json:"batches"`
		Baseline []models.ProcessBaseline    `json:"baseline"`
	}
	if c.ShouldBindJSON(&in) != nil {
		response.BadRequest(c, response.CodeInvalidRequest, "分批交货格式错误")
		return
	}
	err := h.db.Transaction(func(tx *gorm.DB) error {
		item, e := productionflow.LockItem(tx, middleware.GetCorpID(c), flowID(c, "itemId"))
		if e != nil {
			return e
		}
		return productionflow.ConfigureBatches(tx, item, in.Batches, in.Baseline, middleware.GetUserID(c))
	})
	flowResult(c, err, nil)
}
func (h *ProductionFlowHandler) Report(c *gin.Context) {
	if !flowWrite(h.db, c) {
		return
	}
	var in productionflow.ReportInput
	if c.ShouldBindJSON(&in) != nil {
		response.BadRequest(c, response.CodeInvalidRequest, "报工格式错误")
		return
	}
	if !checkEvidence(h.db, c, "operation_notes", in.Notes, 500) {
		return
	}
	flowResult(c, productionflow.Report(h.db, middleware.GetCorpID(c), flowID(c, "itemId"), flowID(c, "processId"), middleware.GetUserID(c), in), nil)
}
