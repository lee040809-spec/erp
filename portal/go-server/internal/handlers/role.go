package handlers

import (
	"portal-erp/internal/models"
	"portal-erp/internal/repository"
	"strconv"

	"github.com/gin-gonic/gin"
)

type RoleHandler struct {
	roleRepo     *repository.RoleRepository
	resourceRepo *repository.ResourceRepository
}

func NewRoleHandler(roleRepo *repository.RoleRepository, resourceRepo *repository.ResourceRepository) *RoleHandler {
	return &RoleHandler{
		roleRepo:     roleRepo,
		resourceRepo: resourceRepo,
	}
}

// List 获取角色列表
func (h *RoleHandler) List(c *gin.Context) {
	roles, err := h.roleRepo.List()
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	// 为每个角色获取资源
	type RoleWithResources struct {
		models.Role
		Resources []models.Resource `json:"resources"`
	}

	var result []RoleWithResources
	for _, role := range roles {
		resources, _ := h.roleRepo.GetResourcesByRoleID(role.ID)
		result = append(result, RoleWithResources{
			Role:      role,
			Resources: resources,
		})
	}

	c.JSON(200, result)
}

// Get 获取单个角色
func (h *RoleHandler) Get(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	role, err := h.roleRepo.GetByID(id)
	if err != nil {
		c.JSON(404, gin.H{"error": "角色不存在"})
		return
	}

	resources, _ := h.roleRepo.GetResourcesByRoleID(id)

	c.JSON(200, gin.H{
		"id":           role.ID,
		"name":         role.Name,
		"company_id":   role.CompanyID,
		"company_name": role.CompanyName,
		"description":  role.Description,
		"created_at":   role.CreatedAt,
		"resources":    resources,
	})
}

// Create 创建角色
func (h *RoleHandler) Create(c *gin.Context) {
	var req struct {
		Name        string `json:"name"`
		CompanyID   int    `json:"company_id"`
		Description string `json:"description"`
		Resources   []int  `json:"resources"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": "请求参数错误"})
		return
	}

	role := &models.Role{
		Name:        req.Name,
		CompanyID:   req.CompanyID,
		Description: req.Description,
	}

	if err := h.roleRepo.Create(role); err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	// 关联资源
	if len(req.Resources) > 0 {
		h.roleRepo.UpdateRoleResources(role.ID, req.Resources)
	}

	c.JSON(200, gin.H{"success": true, "id": role.ID})
}

// Update 更新角色
func (h *RoleHandler) Update(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))

	var req struct {
		Name        string `json:"name"`
		CompanyID   int    `json:"company_id"`
		Description string `json:"description"`
		Resources   []int  `json:"resources"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": "请求参数错误"})
		return
	}

	role := &models.Role{
		ID:          id,
		Name:        req.Name,
		CompanyID:   req.CompanyID,
		Description: req.Description,
	}

	if err := h.roleRepo.Update(role); err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	// 更新资源关联
	if req.Resources != nil {
		h.roleRepo.UpdateRoleResources(id, req.Resources)
	}

	c.JSON(200, gin.H{"success": true})
}

// Delete 删除角色
func (h *RoleHandler) Delete(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	if err := h.roleRepo.Delete(id); err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, gin.H{"success": true})
}