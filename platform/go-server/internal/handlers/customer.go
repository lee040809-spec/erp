package handlers

import (
	"gorm.io/gorm"
	"machining-erp/internal/middleware"
	"machining-erp/internal/models"
	"machining-erp/internal/repository"

	"github.com/gin-gonic/gin"
)

type CustomerHandler struct {
	repo *repository.CustomerRepository
	db   *gorm.DB
}

func NewCustomerHandler(repo *repository.CustomerRepository, db *gorm.DB) *CustomerHandler {
	return &CustomerHandler{repo: repo, db: db}
}

func (h *CustomerHandler) List(c *gin.Context) {
	if !requireModuleAccess(h.db, c, "customers", "orders", "production_orders", "dashboard", "production_dashboard", "production_delivery", "reconciliation", "finance", "monthly_output", "overdue", "warning_orders", "imminent_orders", "production_overdue", "production_warning", "production_imminent") {
		return
	}
	corpID := middleware.GetCorpID(c)
	customers, err := h.repo.GetByCorpID(corpID)
	if err != nil {
		referenceError(c, err)
		return
	}
	c.JSON(200, customers)
}

func (h *CustomerHandler) Create(c *gin.Context) {
	if !requireModuleAccess(h.db, c, "customers") {
		return
	}
	corpID := middleware.GetCorpID(c)

	var req struct {
		Name      string           `json:"name"`
		ShortName string           `json:"short_name"`
		Contacts  []models.Contact `json:"contacts"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	if req.Name == "" {
		c.JSON(400, gin.H{"error": "客户名称不能为空"})
		return
	}
	if req.ShortName == "" {
		c.JSON(400, gin.H{"error": "客户简称不能为空"})
		return
	}

	// 检查客户名称是否已存在
	exists, err := h.repo.NameExists(middleware.GetCorpID(c), req.Name, 0)
	if err != nil {
		referenceError(c, err)
		return
	}
	if exists {
		c.JSON(400, gin.H{"error": "客户名称已存在"})
		return
	}

	id, err := h.repo.Create(corpID, req.Name, req.ShortName, req.Contacts)
	if err != nil {
		referenceError(c, err)
		return
	}
	c.JSON(200, gin.H{"id": id})
}

func (h *CustomerHandler) Update(c *gin.Context) {
	if !requireModuleAccess(h.db, c, "customers") {
		return
	}
	id, valid := referenceID(c)
	if !valid {
		return
	}

	var req struct {
		Name      string           `json:"name"`
		ShortName string           `json:"short_name"`
		Contacts  []models.Contact `json:"contacts"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	if req.Name == "" {
		c.JSON(400, gin.H{"error": "客户名称不能为空"})
		return
	}
	if req.ShortName == "" {
		c.JSON(400, gin.H{"error": "客户简称不能为空"})
		return
	}

	// 检查客户名称是否已存在（排除当前客户）
	exists, err := h.repo.NameExists(middleware.GetCorpID(c), req.Name, id)
	if err != nil {
		referenceError(c, err)
		return
	}
	if exists {
		c.JSON(400, gin.H{"error": "客户名称已存在"})
		return
	}

	if err := h.repo.Update(middleware.GetCorpID(c), id, req.Name, req.ShortName, req.Contacts); err != nil {
		referenceError(c, err)
		return
	}
	c.JSON(200, gin.H{"success": true})
}

func (h *CustomerHandler) Delete(c *gin.Context) {
	if !requireModuleAccess(h.db, c, "customers") {
		return
	}
	id, valid := referenceID(c)
	if !valid {
		return
	}

	if err := h.repo.Delete(middleware.GetCorpID(c), id); err != nil {
		referenceError(c, err)
		return
	}
	c.JSON(200, gin.H{"success": true})
}
