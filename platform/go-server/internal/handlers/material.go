package handlers

import (
	"gorm.io/gorm"
	"machining-erp/internal/middleware"
	"machining-erp/internal/models"
	"machining-erp/internal/repository"
	"math"
	"strings"

	"github.com/gin-gonic/gin"
)

type MaterialHandler struct {
	repo *repository.MaterialRepository
	db   *gorm.DB
}

func NewMaterialHandler(repo *repository.MaterialRepository, db *gorm.DB) *MaterialHandler {
	return &MaterialHandler{repo: repo, db: db}
}

func (h *MaterialHandler) List(c *gin.Context) {
	if !requireModuleAccess(h.db, c, "inventory") {
		return
	}
	materials, err := h.repo.GetByCorpID(middleware.GetCorpID(c))
	if err != nil {
		referenceError(c, err)
		return
	}
	c.JSON(200, materials)
}

func (h *MaterialHandler) Create(c *gin.Context) {
	if !requireModuleAccess(h.db, c, "inventory") {
		return
	}
	var req models.Material
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	if strings.TrimSpace(req.Name) == "" || strings.TrimSpace(req.Unit) == "" || req.Quantity < 0 || math.IsNaN(req.Quantity) || math.IsInf(req.Quantity, 0) {
		c.JSON(400, gin.H{"error": "材料名称、单位及非负数量必须有效"})
		return
	}
	req.ID = 0
	req.CorpID = middleware.GetCorpID(c)
	id, err := h.repo.Create(&req)
	if err != nil {
		referenceError(c, err)
		return
	}
	c.JSON(200, gin.H{"id": id})
}

type RemnantHandler struct {
	repo *repository.RemnantRepository
	db   *gorm.DB
}

func NewRemnantHandler(repo *repository.RemnantRepository, db *gorm.DB) *RemnantHandler {
	return &RemnantHandler{repo: repo, db: db}
}

func (h *RemnantHandler) List(c *gin.Context) {
	if !requireModuleAccess(h.db, c, "inventory") {
		return
	}
	remnants, err := h.repo.GetByCorpID(middleware.GetCorpID(c))
	if err != nil {
		referenceError(c, err)
		return
	}
	c.JSON(200, remnants)
}

func (h *RemnantHandler) Create(c *gin.Context) {
	if !requireModuleAccess(h.db, c, "inventory") {
		return
	}
	var req models.Remnant
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	req.ID = 0
	req.CorpID = middleware.GetCorpID(c)
	id, err := h.repo.Create(&req)
	if err != nil {
		referenceError(c, err)
		return
	}
	c.JSON(200, gin.H{"id": id})
}
