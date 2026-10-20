package handlers

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"machining-erp/internal/middleware"
	"machining-erp/internal/models"
	"machining-erp/pkg/response"
	"strconv"
)

type ProductionRecordHandler struct{ db *gorm.DB }

func NewProductionRecordHandler(db *gorm.DB) *ProductionRecordHandler {
	return &ProductionRecordHandler{db}
}
func (h *ProductionRecordHandler) canEdit(c *gin.Context, kind string) bool {
	if kind == "completion" {
		return canConfirmProduction(h.db, c)
	}
	return kind == "delivery" && hasModuleAccess(h.db, c, "orders", "production_delivery")
}

type correctableRecord struct {
	ID           int64  `json:"id"`
	Quantity     string `json:"quantity"`
	BusinessDate string `json:"business_date"`
	Status       string `json:"status"`
	SourceType   string `json:"source_type"`
	BatchNo      string `json:"batch_no"`
}

func (h *ProductionRecordHandler) List(c *gin.Context) {
	if !requireModuleAccess(h.db, c, "orders", "production_orders", "production_dashboard", "production_delivery", "dashboard", "monthly_output") {
		return
	}
	kind := c.Query("kind")
	if kind != "completion" && kind != "delivery" {
		response.BadRequest(c, response.CodeInvalidRequest, "记录类型无效")
		return
	}
	itemID, _ := strconv.ParseInt(c.Param("itemId"), 10, 64)
	corp := middleware.GetCorpID(c)
	var item models.OrderItem
	if itemID <= 0 || h.db.Where("id=? AND corp_id=?", itemID, corp).Take(&item).Error != nil {
		response.ErrorWithCode(c, 404, response.CodeNotFound, "零件不存在")
		return
	}
	page, size, offset := boundedPage(c, 20)
	table, quantity, filter, source := "production_completions", "qualified_quantity", "", "'completion'"
	if kind == "delivery" {
		table, quantity, filter, source = "order_business_records", "quantity", " AND r.kind='delivery'", "r.source_type"
	}
	rows := []correctableRecord{}
	err := h.db.Table(table+" r").Select("r.id,CAST(r."+quantity+" AS CHAR) AS quantity,COALESCE(DATE_FORMAT(r.business_date,'%Y-%m-%d'),'') AS business_date,r.status,"+source+" AS source_type,COALESCE(b.batch_no,'') AS batch_no").Joins("LEFT JOIN order_delivery_batches b ON b.id=r.delivery_batch_id AND b.corp_id=r.corp_id").Where("r.corp_id=? AND r.order_item_id=?"+filter, corp, itemID).Order("r.business_date DESC,r.id DESC").Limit(size + 1).Offset(offset).Scan(&rows).Error
	if err != nil {
		response.InternalError(c, response.CodeInternal, "读取已登记记录失败")
		return
	}
	rows, more := trimPage(rows, size)
	ids := []int64{}
	for _, row := range rows {
		ids = append(ids, row.ID)
	}
	changes := []models.ProductionRecordChange{}
	var revision int64
	if h.db.Model(&models.ProductionRecordChange{}).Where("corp_id=? AND order_item_id=?", corp, itemID).Select("COALESCE(MAX(id),0)").Scan(&revision).Error != nil {
		response.InternalError(c, response.CodeInternal, "读取修改版本失败")
		return
	}
	if len(ids) > 0 {
		if h.db.Where("corp_id=? AND kind=? AND record_id IN ?", corp, kind, ids).Order("id DESC").Limit(100).Find(&changes).Error != nil {
			response.InternalError(c, response.CodeInternal, "读取修改历史失败")
			return
		}
	}
	response.Success(c, gin.H{"records": rows, "changes": changes, "revision": revision, "can_edit": h.canEdit(c, kind), "page": page, "has_more": more})
}
