package handlers

import (
	"machining-erp/internal/middleware"
	"machining-erp/internal/repository"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type OrderItemHandler struct {
	db          *gorm.DB
	itemRepo    *repository.OrderItemRepository
	orderRepo   *repository.OrderRepository
	processRepo *repository.OrderProcessRepository
}

func NewOrderItemHandler(itemRepo *repository.OrderItemRepository, orderRepo *repository.OrderRepository, processRepo *repository.OrderProcessRepository, db *gorm.DB) *OrderItemHandler {
	return &OrderItemHandler{
		db:          db,
		itemRepo:    itemRepo,
		orderRepo:   orderRepo,
		processRepo: processRepo,
	}
}

func (h *OrderItemHandler) Update(c *gin.Context) {
	if !requireModuleAccess(h.db, c, "orders", "production_orders", "production_dashboard") {
		return
	}
	itemID, err := strconv.ParseInt(c.Param("itemId"), 10, 64)
	corpID := middleware.GetCorpID(c)
	if err != nil || itemID <= 0 || !h.itemRepo.BelongsToCorp(itemID, corpID) {
		c.JSON(404, gin.H{"error": "零件不存在"})
		return
	}

	var req struct {
		Status         *string `json:"status"`
		CompletionDate *string `json:"completion_date"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	if req.Status != nil && *req.Status != "pending" && *req.Status != "processing" && *req.Status != "completed" && *req.Status != "delivered" {
		c.JSON(400, gin.H{"error": "零件状态无效"})
		return
	}
	if req.CompletionDate != nil && *req.CompletionDate != "" {
		if _, err := time.Parse("2006-01-02", *req.CompletionDate); err != nil {
			c.JSON(400, gin.H{"error": "完工日期格式无效"})
			return
		}
	}
	if err := h.orderRepo.UpdateItemForCorp(corpID, itemID, req.Status, req.CompletionDate); err != nil {
		orderMutationError(c, err)
		return
	}

	c.JSON(200, gin.H{"success": true})
}

type ProcessHandler struct {
	db          *gorm.DB
	processRepo *repository.OrderProcessRepository
	itemRepo    *repository.OrderItemRepository
	orderRepo   *repository.OrderRepository
}

func NewProcessHandler(processRepo *repository.OrderProcessRepository, itemRepo *repository.OrderItemRepository, orderRepo *repository.OrderRepository, db *gorm.DB) *ProcessHandler {
	return &ProcessHandler{
		db:          db,
		processRepo: processRepo,
		itemRepo:    itemRepo,
		orderRepo:   orderRepo,
	}
}

func (h *ProcessHandler) Update(c *gin.Context) {
	if !requireModuleAccess(h.db, c, "orders", "production_orders", "production_dashboard") {
		return
	}
	itemID, itemErr := strconv.ParseInt(c.Param("itemId"), 10, 64)
	processID, processErr := strconv.ParseInt(c.Param("processId"), 10, 64)
	corpID := middleware.GetCorpID(c)
	if itemErr != nil || processErr != nil || itemID <= 0 || processID <= 0 || !h.itemRepo.BelongsToCorp(itemID, corpID) {
		c.JSON(404, gin.H{"error": "零件或工序不存在"})
		return
	}

	var req struct {
		Status         *string  `json:"status"`
		IsOutsourced   *bool    `json:"is_outsourced"`
		OutsourcingFee *float64 `json:"outsourcing_fee"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	if req.Status != nil && *req.Status != "pending" && *req.Status != "processing" && *req.Status != "completed" {
		c.JSON(400, gin.H{"error": "工序状态无效"})
		return
	}
	if req.OutsourcingFee != nil && !hasModuleAccess(h.db, c, "orders") {
		c.JSON(403, gin.H{"error": "当前账号无订单费用修改权限"})
		return
	}
	if req.OutsourcingFee != nil && *req.OutsourcingFee < 0 {
		c.JSON(400, gin.H{"error": "外协费用不能为负数"})
		return
	}
	itemStatus, orderStatus, err := h.orderRepo.UpdateProcessForCorp(corpID, itemID, processID, req.Status, req.IsOutsourced, req.OutsourcingFee, middleware.GetUserID(c))
	if err != nil {
		orderMutationError(c, err)
		return
	}
	c.JSON(200, gin.H{"success": true, "item_status": itemStatus, "order_status": orderStatus})
}
