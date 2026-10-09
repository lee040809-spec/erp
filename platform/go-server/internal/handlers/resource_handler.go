package handlers

import (
	"net/http"
	"strconv"

	"machining-erp/internal/repository"
	"machining-erp/internal/services"

	"github.com/gin-gonic/gin"
)

type ResourceHandler struct {
	resourceRepo *repository.ResourceRepository
	authSvc      *services.AuthService
}

func NewResourceHandler(resourceRepo *repository.ResourceRepository, authSvc *services.AuthService) *ResourceHandler {
	return &ResourceHandler{
		resourceRepo: resourceRepo,
		authSvc:      authSvc,
	}
}

// extractToken 从Authorization header中提取token
func extractTokenForResource(c *gin.Context) string {
	token := c.GetHeader("Authorization")
	if len(token) > 7 && token[:7] == "Bearer " {
		token = token[7:]
	}
	return token
}

// List 获取资源列表（只读）
func (h *ResourceHandler) List(c *gin.Context) {
	token := extractTokenForResource(c)
	if token == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未授权"})
		return
	}
	claims := h.authSvc.GetClaimsFromToken(token)
	if claims == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "token无效"})
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))

	filters := make(map[string]string)
	if resourceType := c.Query("resource_type"); resourceType != "" {
		filters["resource_type"] = resourceType
	}
	if platformType := c.Query("platform_type"); platformType != "" {
		filters["platform_type"] = platformType
	}
	if status := c.Query("status"); status != "" {
		filters["status"] = status
	}

	resources, total, err := h.resourceRepo.List(page, pageSize, filters)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取资源列表失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":     resources,
		"total":    total,
		"page":     page,
		"pageSize": pageSize,
	})
}

// GetWithPageResources 获取有页面资源的资源列表（用于打印模板绑定）
func (h *ResourceHandler) GetWithPageResources(c *gin.Context) {
	resources, err := h.resourceRepo.GetWithPageResources()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取资源列表失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": resources,
	})
}
