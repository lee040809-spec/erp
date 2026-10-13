package handlers

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"machining-erp/internal/middleware"
	"machining-erp/internal/models"
	"machining-erp/internal/repository"
	"machining-erp/pkg/response"
	"strconv"
	"strings"
	"time"
)

type ProductionWorkbenchHandler struct{ db *gorm.DB }

func NewProductionWorkbenchHandler(db *gorm.DB) *ProductionWorkbenchHandler {
	return &ProductionWorkbenchHandler{db: db}
}

type workbenchAccess struct {
	Manage    int64
	Dashboard int64
}

func (h *ProductionWorkbenchHandler) readAccess(c *gin.Context) (workbenchAccess, error) {
	if middleware.GetRoleType(c) == "admin" {
		return workbenchAccess{1, 1}, nil
	}
	if value, ok := c.Get("workbenchReadAccess"); ok {
		return value.(workbenchAccess), nil
	}
	var access workbenchAccess
	err := h.db.WithContext(c.Request.Context()).Table("users u").Joins("JOIN permissions p ON p.role_id=u.role_id AND p.corp_id=u.corp_id").Joins("JOIN resources r ON r.id=p.resource_id").
		Select("COALESCE(MAX(CASE WHEN r.resource_key IN ('production_dashboard','production_orders') THEN 1 ELSE 0 END),0) AS manage,COALESCE(MAX(CASE WHEN r.resource_key='dashboard' THEN 1 ELSE 0 END),0) AS dashboard").
		Where("u.id=? AND u.corp_id=?", middleware.GetUserID(c), middleware.GetCorpID(c)).Scan(&access).Error
	if err == nil {
		c.Set("workbenchReadAccess", access)
	}
	return access, err
}
func (h *ProductionWorkbenchHandler) canManage(c *gin.Context) bool {
	access, err := h.readAccess(c)
	return err == nil && access.Manage > 0
}
func (h *ProductionWorkbenchHandler) allowed(c *gin.Context, write bool) bool {
	access, err := h.readAccess(c)
	if err == nil && (access.Manage > 0 || !write && access.Dashboard > 0) {
		return true
	}
	response.Forbidden(c, response.CodeForbidden, "无生产工作台权限")
	return false
}

type planRequest struct {
	ItemIDs        []int64 `json:"item_ids"`
	Assignee       *string `json:"assignee"`
	Equipment      *string `json:"equipment"`
	PlannedStart   *string `json:"planned_start"`
	PlannedEnd     *string `json:"planned_end"`
	EstimatedHours *string `json:"estimated_hours"`
	BlockedReason  *string `json:"blocked_reason"`
	Notes          *string `json:"notes"`
}

func (h *ProductionWorkbenchHandler) Save(c *gin.Context) {
	if !h.allowed(c, true) {
		return
	}
	var req planRequest
	if c.ShouldBindJSON(&req) != nil || len(req.ItemIDs) == 0 || len(req.ItemIDs) > 100 {
		response.BadRequest(c, response.CodeInvalidRequest, "请选择1至100个零件")
		return
	}
	for _, v := range []*string{req.Assignee, req.Equipment} {
		if v != nil && len([]rune(*v)) > 100 {
			response.BadRequest(c, response.CodeInvalidRequest, "负责人或设备名称过长")
			return
		}
	}
	for _, v := range []*string{req.BlockedReason, req.Notes} {
		if v != nil && len([]rune(*v)) > 500 {
			response.BadRequest(c, response.CodeInvalidRequest, "原因或备注最多500字")
			return
		}
	}
	for _, v := range []*string{req.PlannedStart, req.PlannedEnd} {
		if v != nil && *v != "" {
			if _, err := time.Parse("2006-01-02", *v); err != nil {
				response.BadRequest(c, response.CodeInvalidRequest, "计划日期无效")
				return
			}
		}
	}
	if req.EstimatedHours != nil {
		n, err := strconv.ParseFloat(*req.EstimatedHours, 64)
		if err != nil || n < 0 || n > 100000 || !moneyPattern.MatchString(*req.EstimatedHours) {
			response.BadRequest(c, response.CodeInvalidRequest, "预计工时应为0至100000且最多两位小数")
			return
		}
	}
	corp := middleware.GetCorpID(c)
	ids := []int64{}
	seen := map[int64]bool{}
	for _, id := range req.ItemIDs {
		if id <= 0 || seen[id] {
			response.BadRequest(c, response.CodeInvalidRequest, "零件编号无效或重复")
			return
		}
		seen[id] = true
		ids = append(ids, id)
	}
	// Parent order then item locks, matching the order editor/deletion lock order.
	err := h.db.Transaction(func(tx *gorm.DB) error {
		var orderIDs []int64
		if err := tx.Table("order_items").Where("corp_id=? AND id IN ?", corp, ids).Distinct().Pluck("order_id", &orderIDs).Error; err != nil {
			return err
		}
		var orders []models.Order
		if err := tx.Raw("SELECT * FROM orders WHERE corp_id=? AND id IN ? ORDER BY id FOR UPDATE", corp, orderIDs).Scan(&orders).Error; err != nil {
			return err
		}
		var items []models.OrderItem
		if err := tx.Raw("SELECT * FROM order_items WHERE corp_id=? AND id IN ? ORDER BY id FOR UPDATE", corp, ids).Scan(&items).Error; err != nil {
			return err
		}
		if len(items) != len(ids) {
			return fmt.Errorf("零件不存在或不属于当前公司")
		}
		parents := map[int64]bool{}
		for _, o := range orders {
			parents[o.ID] = true
		}
		for _, item := range items {
			if !parents[item.OrderID] || item.Status == models.OrderStatusCompleted || item.Status == models.OrderStatusDelivered {
				return fmt.Errorf("已完成、已交货或无有效订单的零件不能安排")
			}
			if err := tx.Exec("INSERT IGNORE INTO production_plans(corp_id,order_item_id,updated_by) VALUES (?,?,?)", corp, item.ID, middleware.GetUserID(c)).Error; err != nil {
				return err
			}
			changes := map[string]interface{}{"updated_by": middleware.GetUserID(c), "updated_at": time.Now()}
			for k, v := range map[string]*string{"assignee": req.Assignee, "equipment": req.Equipment, "blocked_reason": req.BlockedReason, "notes": req.Notes, "estimated_hours": req.EstimatedHours} {
				if v != nil {
					changes[k] = strings.TrimSpace(*v)
				}
			}
			for k, v := range map[string]*string{"planned_start": req.PlannedStart, "planned_end": req.PlannedEnd} {
				if v != nil {
					if *v == "" {
						changes[k] = nil
					} else {
						changes[k] = *v
					}
				}
			}
			if err := tx.Table("production_plans").Where("corp_id=? AND order_item_id=?", corp, item.ID).Updates(changes).Error; err != nil {
				return err
			}
			var dates struct {
				Start string
				End   string
			}
			if err := tx.Raw("SELECT COALESCE(DATE_FORMAT(planned_start,'%Y-%m-%d'),'') AS start,COALESCE(DATE_FORMAT(planned_end,'%Y-%m-%d'),'') AS end FROM production_plans WHERE corp_id=? AND order_item_id=?", corp, item.ID).Scan(&dates).Error; err != nil {
				return err
			}
			if dates.Start != "" && dates.End != "" && dates.Start > dates.End {
				return fmt.Errorf("计划开始不能晚于计划完工")
			}
		}
		return nil
	})
	if err != nil {
		response.BadRequest(c, response.CodeInvalidRequest, err.Error())
		return
	}
	response.SuccessWithMessage(c, "生产安排已保存", gin.H{"count": len(ids)})
}
func (h *ProductionWorkbenchHandler) Detail(c *gin.Context) {
	if !h.allowed(c, false) {
		return
	}
	id, err := strconv.ParseInt(c.Param("itemId"), 10, 64)
	if err != nil || id <= 0 {
		response.BadRequest(c, response.CodeInvalidRequest, "零件编号无效")
		return
	}
	var item models.OrderItem
	if h.db.Where("id=? AND corp_id=?", id, middleware.GetCorpID(c)).Take(&item).Error != nil {
		response.BadRequest(c, response.CodeInvalidRequest, "零件不存在")
		return
	}
	response.Success(c, gin.H{"drawing_data": item.DrawingData})
}

func (h *ProductionWorkbenchHandler) SaveQuantities(c *gin.Context) {
	if !h.allowed(c, true) {
		return
	}
	id, err := strconv.ParseInt(c.Param("itemId"), 10, 64)
	var req struct {
		ProductionQuantity int `json:"production_quantity"`
		ScrapQuantity      int `json:"scrap_quantity"`
	}
	if err != nil || id <= 0 || c.ShouldBindJSON(&req) != nil {
		response.BadRequest(c, response.CodeInvalidRequest, "数量格式无效")
		return
	}
	var old struct {
		Scrap int `gorm:"column:scrap_quantity"`
	}
	if h.db.Table("order_items").Where("id=? AND corp_id=?", id, middleware.GetCorpID(c)).Select("scrap_quantity").Take(&old).Error != nil {
		response.BadRequest(c, response.CodeInvalidRequest, "零件不存在")
		return
	}
	if req.ScrapQuantity != old.Scrap {
		response.BadRequest(c, response.CodeInvalidRequest, "请通过登记报废按次记录，不能直接改累计报废数")
		return
	}
	if err := repository.NewOrderRepository(h.db).UpdateProductionQuantities(middleware.GetCorpID(c), id, req.ProductionQuantity, req.ScrapQuantity); err != nil {
		orderMutationError(c, err)
		return
	}
	response.SuccessWithMessage(c, "投产与报废数量已保存", nil)
}

// Plan reads a part's arrangement; writes retain the workbench permissions.
func (h *ProductionWorkbenchHandler) Plan(c *gin.Context) {
	if !flowRead(h.db, c) {
		return
	}
	id := flowID(c, "itemId")
	var count int64
	if id <= 0 || h.db.Table("order_items").Where("id=? AND corp_id=?", id, middleware.GetCorpID(c)).Count(&count).Error != nil || count != 1 {
		c.JSON(404, gin.H{"code": response.CodeNotFound, "message": "零件不存在"})
		return
	}
	var plan struct {
		Assignee       string  `json:"assignee"`
		Equipment      string  `json:"equipment"`
		PlannedStart   *string `json:"planned_start"`
		PlannedEnd     *string `json:"planned_end"`
		EstimatedHours string  `json:"estimated_hours"`
		BlockedReason  string  `json:"blocked_reason"`
		Notes          string  `json:"notes"`
	}
	if err := h.db.Table("production_plans").Where("order_item_id=? AND corp_id=?", id, middleware.GetCorpID(c)).Select("assignee,equipment,planned_start,planned_end,CAST(estimated_hours AS CHAR) AS estimated_hours,blocked_reason,notes").Limit(1).Scan(&plan).Error; err != nil {
		response.InternalError(c, response.CodeInternal, "读取生产安排失败")
		return
	}
	response.Success(c, gin.H{"plan": plan, "can_manage": h.canManage(c)})
}
