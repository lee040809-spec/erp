package handlers

import (
	"gorm.io/gorm"
	"machining-erp/internal/middleware"
	"machining-erp/internal/models"
	"machining-erp/internal/repository"
	"machining-erp/pkg/utils"

	"github.com/gin-gonic/gin"
)

type AdventRuleHandler struct {
	repo *repository.AdventRuleRepository
	db   *gorm.DB
}

func NewAdventRuleHandler(repo *repository.AdventRuleRepository, db *gorm.DB) *AdventRuleHandler {
	return &AdventRuleHandler{repo: repo, db: db}
}

func (h *AdventRuleHandler) List(c *gin.Context) {
	if !requireModuleAccess(h.db, c, "advent_rules") {
		return
	}
	name := c.Query("name")
	rules, err := h.repo.GetByCorpID(middleware.GetCorpID(c), name)
	if err != nil {
		referenceError(c, err)
		return
	}
	c.JSON(200, rules)
}

func (h *AdventRuleHandler) Create(c *gin.Context) {
	if !requireModuleAccess(h.db, c, "advent_rules") {
		return
	}
	var req models.AdventRule
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	if utils.NewRuleSQLConverter().ConvertToSQL(req.Formula) == "" {
		c.JSON(400, gin.H{"error": "规则公式无效，请使用有效的进度范围比较"})
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

func (h *AdventRuleHandler) Update(c *gin.Context) {
	if !requireModuleAccess(h.db, c, "advent_rules") {
		return
	}
	id, valid := referenceID(c)
	if !valid {
		return
	}

	var req models.AdventRule
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	if utils.NewRuleSQLConverter().ConvertToSQL(req.Formula) == "" {
		c.JSON(400, gin.H{"error": "规则公式无效，请使用有效的进度范围比较"})
		return
	}
	req.ID = id
	req.CorpID = middleware.GetCorpID(c)
	if err := h.repo.Update(&req); err != nil {
		referenceError(c, err)
		return
	}
	c.JSON(200, gin.H{"success": true})
}

func (h *AdventRuleHandler) Delete(c *gin.Context) {
	if !requireModuleAccess(h.db, c, "advent_rules") {
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
