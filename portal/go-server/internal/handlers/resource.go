package handlers

import (
	"portal-erp/internal/models"
	"portal-erp/internal/repository"
	"strconv"

	"github.com/gin-gonic/gin"
)

type ResourceHandler struct {
	resourceRepo *repository.ResourceRepository
	roleRepo     *repository.RoleRepository
}

func NewResourceHandler(resourceRepo *repository.ResourceRepository, roleRepo *repository.RoleRepository) *ResourceHandler {
	return &ResourceHandler{
		resourceRepo: resourceRepo,
		roleRepo:     roleRepo,
	}
}

// List 获取资源列表
func (h *ResourceHandler) List(c *gin.Context) {
	resources, err := h.resourceRepo.List()
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, resources)
}

// Get 获取单个资源
func (h *ResourceHandler) Get(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	res, err := h.resourceRepo.GetByID(id)
	if err != nil {
		c.JSON(404, gin.H{"error": "资源不存在"})
		return
	}
	c.JSON(200, res)
}

// Create 创建资源
func (h *ResourceHandler) Create(c *gin.Context) {
	var res models.Resource
	if err := c.ShouldBindJSON(&res); err != nil {
		c.JSON(400, gin.H{"error": "请求参数错误"})
		return
	}

	if err := h.resourceRepo.Create(&res); err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	c.JSON(200, gin.H{"success": true, "id": res.ID})
}

// Update 更新资源
func (h *ResourceHandler) Update(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))

	var res models.Resource
	if err := c.ShouldBindJSON(&res); err != nil {
		c.JSON(400, gin.H{"error": "请求参数错误"})
		return
	}

	res.ID = id
	if err := h.resourceRepo.Update(&res); err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	c.JSON(200, gin.H{"success": true})
}

// Delete 删除资源
func (h *ResourceHandler) Delete(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	if err := h.resourceRepo.Delete(id); err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, gin.H{"success": true})
}

// GetMenus 获取当前用户的菜单
func (h *ResourceHandler) GetMenus(c *gin.Context) {
	roleID, exists := c.Get("role_id")
	if !exists {
		c.JSON(401, gin.H{"error": "未授权"})
		return
	}

	menus, err := h.resourceRepo.GetMenusByRoleID(roleID.(int))
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	c.JSON(200, menus)
}