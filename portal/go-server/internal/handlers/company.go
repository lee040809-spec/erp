package handlers

import (
	"portal-erp/internal/models"
	"portal-erp/internal/repository"
	"strconv"

	"github.com/gin-gonic/gin"
)

type CompanyHandler struct {
	companyRepo *repository.CompanyRepository
}

func NewCompanyHandler(companyRepo *repository.CompanyRepository) *CompanyHandler {
	return &CompanyHandler{companyRepo: companyRepo}
}

// List 获取公司列表
func (h *CompanyHandler) List(c *gin.Context) {
	companies, err := h.companyRepo.List()
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, companies)
}

// Get 获取单个公司
func (h *CompanyHandler) Get(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	company, err := h.companyRepo.GetByID(id)
	if err != nil {
		c.JSON(404, gin.H{"error": "公司不存在"})
		return
	}
	c.JSON(200, company)
}

// Create 创建公司
func (h *CompanyHandler) Create(c *gin.Context) {
	var company models.Company
	if err := c.ShouldBindJSON(&company); err != nil {
		c.JSON(400, gin.H{"error": "请求参数错误"})
		return
	}

	if company.Status == 0 {
		company.Status = 1
	}

	if err := h.companyRepo.Create(&company); err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	c.JSON(200, gin.H{"success": true, "id": company.ID})
}

// Update 更新公司
func (h *CompanyHandler) Update(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))

	var company models.Company
	if err := c.ShouldBindJSON(&company); err != nil {
		c.JSON(400, gin.H{"error": "请求参数错误"})
		return
	}

	company.ID = id
	if err := h.companyRepo.Update(&company); err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	c.JSON(200, gin.H{"success": true})
}

// Delete 删除公司
func (h *CompanyHandler) Delete(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	if err := h.companyRepo.Delete(id); err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, gin.H{"success": true})
}