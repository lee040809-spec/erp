package handlers

import (
	"net/http"
	"strconv"

	"machining-erp/internal/models"
	"machining-erp/internal/repository"
	"machining-erp/internal/services"

	"github.com/gin-gonic/gin"
)

type RoleHandler struct {
	roleRepo       *repository.RoleRepository
	permissionRepo *repository.PermissionRepository
	resourceRepo   *repository.ResourceRepository
	authSvc        *services.AuthService
}

func NewRoleHandler(roleRepo *repository.RoleRepository, permissionRepo *repository.PermissionRepository, resourceRepo *repository.ResourceRepository, authSvc *services.AuthService) *RoleHandler {
	return &RoleHandler{
		roleRepo:       roleRepo,
		permissionRepo: permissionRepo,
		resourceRepo:   resourceRepo,
		authSvc:        authSvc,
	}
}

// extractToken 从Authorization header中提取token
func extractTokenForRole(c *gin.Context) string {
	token := c.GetHeader("Authorization")
	if len(token) > 7 && token[:7] == "Bearer " {
		token = token[7:]
	}
	return token
}

// List 获取角色列表
func (h *RoleHandler) List(c *gin.Context) {
	token := extractTokenForRole(c)
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

	roles, total, err := h.roleRepo.List(claims.CorpID, page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取角色列表失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":     roles,
		"total":    total,
		"page":     page,
		"pageSize": pageSize,
	})
}

// CreateRequest 创建角色请求
type RoleCreateRequest struct {
	Name        string             `json:"name" binding:"required"`
	AccountType models.AccountType `json:"account_type" binding:"required"`
}

// Create 创建角色
func (h *RoleHandler) Create(c *gin.Context) {
	token := extractTokenForRole(c)
	if token == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未授权"})
		return
	}
	claims := h.authSvc.GetClaimsFromToken(token)
	if claims == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "token无效"})
		return
	}

	var req RoleCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}

	role := &models.Role{
		CorpID:      claims.CorpID,
		Name:        req.Name,
		AccountType: req.AccountType,
		Status:      "active",
	}

	if err := h.roleRepo.Create(role); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建角色失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "data": role})
}

// UpdateRequest 更新角色请求
type RoleUpdateRequest struct {
	Name   string `json:"name"`
	Status string `json:"status"`
}

// Update 更新角色
func (h *RoleHandler) Update(c *gin.Context) {
	token := extractTokenForRole(c)
	if token == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未授权"})
		return
	}
	claims := h.authSvc.GetClaimsFromToken(token)
	if claims == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "token无效"})
		return
	}

	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}

	role, err := h.roleRepo.FindByID(id)
	if err != nil || role.CorpID != claims.CorpID {
		c.JSON(http.StatusNotFound, gin.H{"error": "角色不存在"})
		return
	}

	var req RoleUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}

	if req.Name != "" {
		role.Name = req.Name
	}
	if req.Status != "" {
		role.Status = req.Status
	}

	if err := h.roleRepo.Update(role); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "更新角色失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "data": role})
}

// Delete 删除角色
func (h *RoleHandler) Delete(c *gin.Context) {
	token := extractTokenForRole(c)
	if token == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未授权"})
		return
	}
	claims := h.authSvc.GetClaimsFromToken(token)
	if claims == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "token无效"})
		return
	}

	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}

	// 查询角色
	role, err := h.roleRepo.FindByID(id)
	if err != nil || role.CorpID != claims.CorpID {
		c.JSON(http.StatusNotFound, gin.H{"error": "角色不存在"})
		return
	}

	// 管理员类型的角色不可删除
	if role.AccountType == models.AccountTypeAdmin {
		c.JSON(http.StatusBadRequest, gin.H{"error": "管理员角色不可删除"})
		return
	}

	if err := h.roleRepo.Delete(id, claims.CorpID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "删除角色失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}

// GetPermissions 获取角色的权限列表
func (h *RoleHandler) GetPermissions(c *gin.Context) {
	token := extractTokenForRole(c)
	if token == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未授权"})
		return
	}
	claims := h.authSvc.GetClaimsFromToken(token)
	if claims == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "token无效"})
		return
	}

	roleID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}

	// 验证角色存在且属于当前企业
	role, err := h.roleRepo.FindByID(roleID)
	if err != nil || role.CorpID != claims.CorpID {
		c.JSON(http.StatusNotFound, gin.H{"error": "角色不存在"})
		return
	}

	// 获取角色权限及资源信息
	permissions, err := h.permissionRepo.GetByRoleIDWithResources(roleID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取权限失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": permissions})
}

// GetAllResources 获取所有可用资源（用于绑定）
func (h *RoleHandler) GetAllResources(c *gin.Context) {
	token := extractTokenForRole(c)
	if token == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未授权"})
		return
	}
	claims := h.authSvc.GetClaimsFromToken(token)
	if claims == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "token无效"})
		return
	}

	resources, err := h.resourceRepo.GetAll()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取资源失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": resources})
}

// BindPermissionRequest 绑定权限请求
type BindPermissionRequest struct {
	ResourceID int64  `json:"resource_id" binding:"required"`
	Permission string `json:"permission"` // read, write, admin
}

// BindPermission 绑定资源权限
func (h *RoleHandler) BindPermission(c *gin.Context) {
	token := extractTokenForRole(c)
	if token == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未授权"})
		return
	}
	claims := h.authSvc.GetClaimsFromToken(token)
	if claims == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "token无效"})
		return
	}

	roleID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}

	// 验证角色存在且属于当前企业
	role, err := h.roleRepo.FindByID(roleID)
	if err != nil || role.CorpID != claims.CorpID {
		c.JSON(http.StatusNotFound, gin.H{"error": "角色不存在"})
		return
	}

	var req BindPermissionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}

	// 检查是否已绑定
	existing, _ := h.permissionRepo.FindByRoleAndResource(roleID, req.ResourceID)
	if existing != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "该资源已绑定"})
		return
	}

	// 创建权限
	permission := &models.Permission{
		CorpID:     claims.CorpID,
		RoleID:     roleID,
		ResourceID: req.ResourceID,
		Permission: req.Permission,
	}
	if permission.Permission == "" {
		permission.Permission = "read"
	}

	if err := h.permissionRepo.Create(permission); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "绑定失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "data": permission})
}

// UnbindPermission 解绑资源权限
func (h *RoleHandler) UnbindPermission(c *gin.Context) {
	token := extractTokenForRole(c)
	if token == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未授权"})
		return
	}
	claims := h.authSvc.GetClaimsFromToken(token)
	if claims == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "token无效"})
		return
	}

	roleID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}

	permissionID, err := strconv.ParseInt(c.Param("permission_id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}

	// 验证角色存在且属于当前企业
	role, err := h.roleRepo.FindByID(roleID)
	if err != nil || role.CorpID != claims.CorpID {
		c.JSON(http.StatusNotFound, gin.H{"error": "角色不存在"})
		return
	}

	if err := h.permissionRepo.Delete(permissionID, claims.CorpID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "解绑失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}
