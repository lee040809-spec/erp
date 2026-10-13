package handlers

import (
	"errors"
	"fmt"
	"machining-erp/internal/middleware"
	"machining-erp/internal/models"
	"machining-erp/internal/repository"
	"machining-erp/internal/services"
	"machining-erp/pkg/utils"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type OrderHandler struct {
	db          *gorm.DB
	repo        *repository.OrderRepository
	itemRepo    *repository.OrderItemRepository
	processRepo *repository.OrderProcessRepository
	orderNumSvc *services.OrderNumberService
}

func NewOrderHandler(repo *repository.OrderRepository, itemRepo *repository.OrderItemRepository, processRepo *repository.OrderProcessRepository, orderNumSvc *services.OrderNumberService, db *gorm.DB) *OrderHandler {
	return &OrderHandler{
		db:          db,
		repo:        repo,
		itemRepo:    itemRepo,
		processRepo: processRepo,
		orderNumSvc: orderNumSvc,
	}
}

func (h *OrderHandler) List(c *gin.Context) {
	if !requireModuleAccess(h.db, c, "orders", "production_orders", "dashboard", "production_dashboard", "overdue", "warning_orders", "imminent_orders", "production_overdue", "production_warning", "production_imminent", "production_delivery", "reconciliation") {
		return
	}
	startTime := time.Now()
	page, pageSize, _ := boundedPage(c, 20)

	filters := repository.OrderFilters{
		CorpID:       middleware.GetCorpID(c),
		DueDateStart: c.Query("dueDateStart"),
		DueDateEnd:   c.Query("dueDateEnd"),
		OrderNumber:  c.Query("orderNumber"),
		PartNumber:   c.Query("partNumber"),
		PartName:     strings.TrimSpace(c.Query("partName")),
		CustomerName: c.Query("customerName"),
		Priority:     c.Query("priority"),
		Status:       c.Query("status"),
		DateType:     c.Query("dateType"), // overdue, warning, near_due
		Page:         page,
		PageSize:     pageSize,
	}

	result, err := h.repo.GetWithFilters(filters)
	elapsed := time.Since(startTime)
	fmt.Printf("[Order List] Query took %v ms\n", elapsed.Milliseconds())
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	var data interface{} = result.Orders
	if !canViewOrderAmounts(h.db, c) {
		data = withoutAmounts(data)
	}
	c.JSON(200, gin.H{
		"data":     data,
		"total":    result.Total,
		"page":     page,
		"pageSize": pageSize,
	})
}

type CreateOrderRequest struct {
	CustomerID        *int64                   `json:"customer_id"`
	CustomerName      *string                  `json:"customer_name"`
	CustomerShortName *string                  `json:"customer_short_name"`
	OrderNumber       string                   `json:"order_number"`
	OrderName         *string                  `json:"order_name"`
	ContactName       *string                  `json:"contact_name"`
	Priority          models.Priority          `json:"priority"`
	StartDate         string                   `json:"start_date"`
	DueDate           string                   `json:"due_date"`
	TotalAmount       float64                  `json:"total_amount"`
	Notes             *string                  `json:"notes"`
	Items             []CreateOrderItemRequest `json:"items"`
}

// CreateOrderItemRequest 创建订单项请求结构体（日期用string避免解析问题）
type CreateOrderItemRequest struct {
	DeliveryPlan        []models.DeliveryBatchInput `json:"delivery_plan"`
	ID                  *int64                      `json:"id"`
	OrderID             *int64                      `json:"order_id"`
	PartName            string                      `json:"part_name"`
	PartNumber          *string                     `json:"part_number"`
	ProductionQuantity  int                         `json:"production_quantity"`
	Quantity            int                         `json:"quantity"`
	ScrapQuantity       int                         `json:"scrap_quantity"`
	UnitPrice           *float64                    `json:"unit_price"`
	TotalPrice          float64                     `json:"total_price"`
	Status              models.OrderStatus          `json:"status"`
	DrawingData         *string                     `json:"drawing_data"`
	DrawingIDs          []int64                     `json:"drawing_ids"`
	DrawingChangeReason string                      `json:"drawing_change_reason"`
	Notes               *string                     `json:"notes"`
	StartDate           string                      `json:"start_date"`
	DueDate             string                      `json:"due_date"`
	DeliveredQty        int                         `json:"delivered_quantity"`
	ToolCost            *float64                    `json:"tool_cost"`
	FixtureCost         *float64                    `json:"fixture_cost"`
	MaterialCost        *float64                    `json:"material_cost"`
	OtherCost           *float64                    `json:"other_cost"`
	SupplyMode          string                      `json:"supply_mode"`
	Currency            string                      `json:"currency"`
	TaxInclusive        *bool                       `json:"tax_inclusive"`
	ItemNotes           *string                     `json:"item_notes"`
	CompletionDate      string                      `json:"completion_date"`
	Processes           []ProcessRequest            `json:"processes"`
}

// ProcessRequest 工序请求结构体
type ProcessRequest struct {
	CuttingSpec    *string              `json:"cutting_spec"`
	ID             *int64               `json:"id"`
	OrderItemID    *int64               `json:"order_item_id"`
	Name           string               `json:"name"`
	IsOutsourced   bool                 `json:"is_outsourced"`
	OutsourcingFee float64              `json:"outsourcing_fee"`
	Status         models.ProcessStatus `json:"status"`
	SortOrder      int                  `json:"sort_order"`
	PlannedStart   *string              `json:"planned_start_at"`
	PlannedEnd     *string              `json:"planned_end_at"`
}

func (p ProcessRequest) planDates() (*time.Time, *time.Time, error) {
	start, err := parseProcessNode(p.PlannedStart, nil)
	if err != nil {
		return nil, nil, err
	}
	end, err := parseProcessNode(p.PlannedEnd, nil)
	if err != nil {
		return nil, nil, err
	}
	if p.Name == "送货" && (start != nil || end != nil) {
		return nil, nil, errors.New("送货日期请在交货记录中登记")
	}
	return start, end, utils.ValidateProcessPlan(start, end)
}

func parseDate(s string) *time.Time {
	if s == "" {
		return nil
	}
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		return nil
	}
	return &t
}

func floatValue(v *float64) float64 {
	if v == nil {
		return 0
	}
	return *v
}

func floatPtr(v float64) *float64 {
	return &v
}

func intPtr(v int) *int {
	return &v
}

func (h *OrderHandler) Create(c *gin.Context) {
	if !requireModuleAccess(h.db, c, "orders", "production_orders") {
		return
	}
	var req CreateOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	if req.CustomerID == nil || req.StartDate == "" || req.DueDate == "" {
		c.JSON(400, gin.H{"error": "Missing customer_id, start_date or due_date"})
		return
	}

	if len(req.Items) == 0 {
		c.JSON(400, gin.H{"error": "订单必须包含至少一个零件"})
		return
	}

	for i, item := range req.Items {
		if item.PartName == "" {
			c.JSON(400, gin.H{"error": fmt.Sprintf("零件名称不能为空 (第 %d 行)", i+1)})
			return
		}
		if item.SupplyMode != "" && item.SupplyMode != "self_purchased" && item.SupplyMode != "customer_supplied" {
			c.JSON(400, gin.H{"error": fmt.Sprintf("材料来源无效 (第 %d 行)", i+1)})
			return
		}
	}

	orderNumber := req.OrderNumber
	if orderNumber == "" {
		var err error
		orderNumber, err = h.orderNumSvc.Generate()
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
	}

	order := &models.Order{
		CorpID:            middleware.GetCorpID(c),
		CustomerID:        req.CustomerID,
		CustomerName:      req.CustomerName,
		CustomerShortName: req.CustomerShortName,
		OrderNumber:       orderNumber,
		OrderName:         req.OrderName,
		ContactName:       req.ContactName,
		Priority:          req.Priority,
		StartDate:         parseDate(req.StartDate),
		DueDate:           parseDate(req.DueDate),
		Notes:             req.Notes,
		TotalAmount:       req.TotalAmount,
	}

	// 转换请求结构体为models
	items := make([]models.OrderItem, len(req.Items))
	for i, reqItem := range req.Items {
		if reqItem.SupplyMode == "" {
			reqItem.SupplyMode = "self_purchased"
		}
		if reqItem.Currency == "" {
			reqItem.Currency = "CNY"
		}
		// 转换工序
		processes := make([]models.OrderProcess, len(reqItem.Processes))
		for j, p := range reqItem.Processes {
			var processID, orderItemID int64
			if p.ID != nil {
				processID = *p.ID
			}
			if p.OrderItemID != nil {
				orderItemID = *p.OrderItemID
			}
			plannedStart, plannedEnd, planErr := p.planDates()
			if planErr != nil {
				c.JSON(400, gin.H{"error": fmt.Sprintf("第%d项零件的%s：%s", i+1, p.Name, planErr.Error())})
				return
			}
			processes[j] = models.OrderProcess{
				ID:             processID,
				OrderItemID:    orderItemID,
				Name:           p.Name,
				IsOutsourced:   p.IsOutsourced,
				OutsourcingFee: p.OutsourcingFee,
				CuttingSpec:    p.CuttingSpec,
				Status:         p.Status,
				PlannedStartAt: plannedStart,
				PlannedEndAt:   plannedEnd,
			}
		}
		var itemID, orderID int64
		if reqItem.ID != nil {
			itemID = *reqItem.ID
		}
		if reqItem.OrderID != nil {
			orderID = *reqItem.OrderID
		}
		items[i] = models.OrderItem{
			ID:                  itemID,
			OrderID:             orderID,
			PartName:            reqItem.PartName,
			PartNumber:          reqItem.PartNumber,
			ProductionQuantity:  reqItem.ProductionQuantity,
			Quantity:            reqItem.Quantity,
			ScrapQuantity:       reqItem.ScrapQuantity,
			UnitPrice:           floatValue(reqItem.UnitPrice),
			TotalPrice:          reqItem.TotalPrice,
			Status:              reqItem.Status,
			DrawingData:         reqItem.DrawingData,
			DrawingIDs:          reqItem.DrawingIDs,
			DrawingChangeReason: reqItem.DrawingChangeReason,
			Notes:               reqItem.Notes,
			StartDate:           parseDate(reqItem.StartDate),
			DueDate:             parseDate(reqItem.DueDate),
			DeliveredQty:        intPtr(reqItem.DeliveredQty),
			ToolCost:            reqItem.ToolCost,
			FixtureCost:         reqItem.FixtureCost,
			MaterialCost:        reqItem.MaterialCost,
			OtherCost:           reqItem.OtherCost,
			SupplyMode:          reqItem.SupplyMode,
			Currency:            reqItem.Currency,
			TaxInclusive:        reqItem.TaxInclusive != nil && *reqItem.TaxInclusive,
			ItemNotes:           reqItem.ItemNotes,
			CompletionDate:      parseDate(reqItem.CompletionDate),
			Processes:           processes,
			DeliveryPlan:        reqItem.DeliveryPlan,
		}
	}

	if !hasModuleAccess(h.db, c, "orders") {
		for i := range items {
			items[i].UnitPrice = 0
			items[i].ToolCost, items[i].FixtureCost = nil, nil
			items[i].MaterialCost, items[i].OtherCost = nil, nil
			for j := range items[i].Processes {
				items[i].Processes[j].OutsourcingFee = 0
			}
		}
	}
	id, err := h.repo.Create(order, items, middleware.GetUserID(c))
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, gin.H{"id": id})
}

type UpdateOrderRequest struct {
	CustomerID        *int64                   `json:"customer_id"`
	CustomerName      *string                  `json:"customer_name"`
	CustomerShortName *string                  `json:"customer_short_name"`
	OrderNumber       string                   `json:"order_number"`
	OrderName         *string                  `json:"order_name"`
	ContactName       *string                  `json:"contact_name"`
	Priority          models.Priority          `json:"priority"`
	StartDate         string                   `json:"start_date"`
	DueDate           string                   `json:"due_date"`
	Notes             *string                  `json:"notes"`
	TotalAmount       float64                  `json:"total_amount"`
	Status            models.OrderStatus       `json:"status"`
	Items             []CreateOrderItemRequest `json:"items"`
}

func (h *OrderHandler) Update(c *gin.Context) {
	if !requireModuleAccess(h.db, c, "orders", "production_orders") {
		return
	}
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	corpID := middleware.GetCorpID(c)
	if _, err := h.repo.GetByIDForCorp(id, corpID); err != nil {
		c.JSON(404, gin.H{"error": "订单不存在"})
		return
	}

	var req UpdateOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	for i, item := range req.Items {
		if item.SupplyMode != "" && item.SupplyMode != "self_purchased" && item.SupplyMode != "customer_supplied" {
			c.JSON(400, gin.H{"error": fmt.Sprintf("材料来源无效 (第 %d 行)", i+1)})
			return
		}
	}

	fmt.Printf("[Order Update] Received order_number: %s\n", req.OrderNumber)

	if req.StartDate == "" || req.DueDate == "" {
		c.JSON(400, gin.H{"error": "start_date and due_date cannot be empty"})
		return
	}

	order := &models.Order{
		ID:                id,
		CorpID:            corpID,
		CustomerID:        req.CustomerID,
		CustomerName:      req.CustomerName,
		CustomerShortName: req.CustomerShortName,
		OrderNumber:       req.OrderNumber,
		OrderName:         req.OrderName,
		ContactName:       req.ContactName,
		Priority:          req.Priority,
		StartDate:         parseDate(req.StartDate),
		DueDate:           parseDate(req.DueDate),
		Notes:             req.Notes,
		TotalAmount:       req.TotalAmount,
	}

	// 转换请求结构体为models
	items := make([]models.OrderItem, len(req.Items))
	for i, reqItem := range req.Items {
		// 转换工序
		processes := make([]models.OrderProcess, len(reqItem.Processes))
		for j, p := range reqItem.Processes {
			var processID, orderItemID int64
			if p.ID != nil {
				processID = *p.ID
			}
			if p.OrderItemID != nil {
				orderItemID = *p.OrderItemID
			}
			plannedStart, plannedEnd, planErr := p.planDates()
			if planErr != nil {
				c.JSON(400, gin.H{"error": fmt.Sprintf("第%d项零件的%s：%s", i+1, p.Name, planErr.Error())})
				return
			}
			processes[j] = models.OrderProcess{
				ID:             processID,
				OrderItemID:    orderItemID,
				Name:           p.Name,
				IsOutsourced:   p.IsOutsourced,
				OutsourcingFee: p.OutsourcingFee,
				CuttingSpec:    p.CuttingSpec,
				Status:         p.Status,
				PlannedStartAt: plannedStart,
				PlannedEndAt:   plannedEnd,
			}
		}
		var itemID, orderID int64
		if reqItem.ID != nil {
			itemID = *reqItem.ID
		}
		if reqItem.OrderID != nil {
			orderID = *reqItem.OrderID
		}
		items[i] = models.OrderItem{
			ID:                  itemID,
			OrderID:             orderID,
			PartName:            reqItem.PartName,
			PartNumber:          reqItem.PartNumber,
			ProductionQuantity:  reqItem.ProductionQuantity,
			Quantity:            reqItem.Quantity,
			ScrapQuantity:       reqItem.ScrapQuantity,
			UnitPrice:           floatValue(reqItem.UnitPrice),
			TotalPrice:          reqItem.TotalPrice,
			Status:              reqItem.Status,
			DrawingData:         reqItem.DrawingData,
			DrawingIDs:          reqItem.DrawingIDs,
			DrawingChangeReason: reqItem.DrawingChangeReason,
			Notes:               reqItem.Notes,
			StartDate:           parseDate(reqItem.StartDate),
			DueDate:             parseDate(reqItem.DueDate),
			DeliveredQty:        intPtr(reqItem.DeliveredQty),
			ToolCost:            reqItem.ToolCost,
			FixtureCost:         reqItem.FixtureCost,
			MaterialCost:        reqItem.MaterialCost,
			OtherCost:           reqItem.OtherCost,
			SupplyMode:          reqItem.SupplyMode,
			Currency:            reqItem.Currency,
			TaxInclusive:        reqItem.TaxInclusive != nil && *reqItem.TaxInclusive,
			ItemNotes:           reqItem.ItemNotes,
			CompletionDate:      parseDate(reqItem.CompletionDate),
			Processes:           processes,
			DeliveryPlan:        reqItem.DeliveryPlan,
		}
	}

	taxFlags := map[int64]bool{}
	priceFlags := map[int64]bool{}
	planFlags := map[int64]bool{}
	for i, item := range req.Items {
		for _, process := range item.Processes {
			if process.ID != nil {
				planFlags[*process.ID] = process.PlannedStart != nil || process.PlannedEnd != nil
			}
		}
		taxFlags[items[i].ID] = item.TaxInclusive != nil
		priceFlags[items[i].ID] = item.UnitPrice != nil && hasModuleAccess(h.db, c, "orders")
	}
	if err := h.repo.Update(order, items, repository.OrderUpdateOptions{ActorID: middleware.GetUserID(c), ProcessPlansProvided: planFlags, TaxProvided: taxFlags, UnitPriceProvided: priceFlags, ProtectAmounts: !hasModuleAccess(h.db, c, "orders")}); err != nil {
		orderMutationError(c, err)
		return
	}
	c.JSON(200, gin.H{"success": true})
}

func (h *OrderHandler) Delete(c *gin.Context) {
	if !requireModuleAccess(h.db, c, "orders", "production_orders") {
		return
	}
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)

	if err := h.repo.Delete(id, middleware.GetCorpID(c)); err != nil {
		orderMutationError(c, err)
		return
	}
	c.JSON(200, gin.H{"success": true})
}

func orderMutationError(c *gin.Context, err error) {
	if errors.Is(err, gorm.ErrRecordNotFound) || errors.Is(err, repository.ErrOrderScope) {
		c.JSON(404, gin.H{"error": "订单、零件或工序不存在"})
		return
	}
	if errors.Is(err, repository.ErrTimedProcess) || errors.Is(err, repository.ErrCompletionState) || errors.Is(err, repository.ErrDeliveryState) || errors.Is(err, repository.ErrCompletionDate) || errors.Is(err, repository.ErrPostedOrder) || errors.Is(err, repository.ErrPostedQuantity) || errors.Is(err, repository.ErrPostedBasis) {
		c.JSON(409, gin.H{"error": err.Error()})
		return
	}
	c.JSON(500, gin.H{"error": "订单保存失败，请重试"})
}

// GetDashboardItems 获取工作看板零件数据
func (h *OrderHandler) GetDashboardItems(c *gin.Context) {
	if !requireModuleAccess(h.db, c, "orders", "production_orders", "dashboard", "production_dashboard", "overdue", "warning_orders", "imminent_orders", "production_overdue", "production_warning", "production_imminent", "production_delivery", "reconciliation") {
		return
	}
	page, _ := strconv.Atoi(c.Query("page"))
	pageSize, _ := strconv.Atoi(c.Query("pageSize"))

	result, err := h.repo.GetDashboardItems(middleware.GetCorpID(c), page, pageSize)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	var data interface{} = result
	if !canViewOrderAmounts(h.db, c) {
		data = withoutAmounts(data)
	}
	c.JSON(200, gin.H{
		"code": 0,
		"data": data,
	})
}

// GetDashboardStats 获取看板卡片统计数据
func (h *OrderHandler) GetDashboardStats(c *gin.Context) {
	if !requireModuleAccess(h.db, c, "orders", "production_orders", "dashboard", "production_dashboard", "overdue", "warning_orders", "imminent_orders", "production_overdue", "production_warning", "production_imminent", "production_delivery", "reconciliation") {
		return
	}
	stats, err := h.repo.GetDashboardStats(middleware.GetCorpID(c))
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, gin.H{
		"code": 0,
		"data": stats,
	})
}

// Drawing is fetched separately from lists and remains scoped to the caller's company.
func (h *OrderHandler) Drawing(c *gin.Context) {
	if !requireModuleAccess(h.db, c, "orders", "production_orders", "dashboard", "production_dashboard", "overdue", "warning_orders", "imminent_orders", "production_overdue", "production_warning", "production_imminent", "production_delivery", "reconciliation") {
		return
	}
	id, err := strconv.ParseInt(c.Param("itemId"), 10, 64)
	if err != nil || id <= 0 {
		c.JSON(400, gin.H{"error": "零件编号无效"})
		return
	}
	var item models.OrderItem
	if err = h.db.Select("id,drawing_data").Where("id=? AND corp_id=?", id, middleware.GetCorpID(c)).Take(&item).Error; err != nil {
		c.JSON(404, gin.H{"error": "零件不存在"})
		return
	}
	c.JSON(200, gin.H{"data": gin.H{"drawing_data": item.DrawingData}})
}
