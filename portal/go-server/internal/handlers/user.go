package handlers

import (
	"portal-erp/internal/models"
	"portal-erp/internal/repository"
	"portal-erp/internal/services"
	"strconv"

	"github.com/gin-gonic/gin"
)

type UserHandler struct {
	userRepo    *repository.UserRepository
	roleRepo    *repository.RoleRepository
	authService *services.AuthService
}

func NewUserHandler(userRepo *repository.UserRepository, roleRepo *repository.RoleRepository, authService *services.AuthService) *UserHandler {
	return &UserHandler{
		userRepo:    userRepo,
		roleRepo:    roleRepo,
		authService: authService,
	}
}

// List 获取用户列表
func (h *UserHandler) List(c *gin.Context) {
	users, err := h.userRepo.List()
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, users)
}

// Get 获取单个用户
func (h *UserHandler) Get(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	user, err := h.userRepo.GetByID(id)
	if err != nil {
		c.JSON(404, gin.H{"error": "用户不存在"})
		return
	}
	c.JSON(200, user)
}

// Create 创建用户
func (h *UserHandler) Create(c *gin.Context) {
	var req struct {
		Username  string `json:"username"`
		Password  string `json:"password"`
		Email     string `json:"email"`
		CompanyID int    `json:"company_id"`
		RoleID    int    `json:"role_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": "请求参数错误"})
		return
	}

	// 检查用户名是否已存在
	_, err := h.userRepo.GetByUsername(req.Username)
	if err == nil {
		c.JSON(400, gin.H{"error": "用户名已存在"})
		return
	}

	user := &models.User{
		Username:     req.Username,
		PasswordHash: h.authService.HashPassword(req.Password),
		Email:        req.Email,
		CompanyID:    req.CompanyID,
		RoleID:       req.RoleID,
	}

	if err := h.userRepo.Create(user); err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	c.JSON(200, gin.H{"success": true, "id": user.ID})
}

// Update 更新用户
func (h *UserHandler) Update(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))

	var req struct {
		Email     string `json:"email"`
		CompanyID int    `json:"company_id"`
		RoleID    int    `json:"role_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": "请求参数错误"})
		return
	}

	user := &models.User{
		ID:        id,
		Email:     req.Email,
		CompanyID: req.CompanyID,
		RoleID:    req.RoleID,
	}

	if err := h.userRepo.Update(user); err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	c.JSON(200, gin.H{"success": true})
}

// Delete 删除用户
func (h *UserHandler) Delete(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	if err := h.userRepo.Delete(id); err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, gin.H{"success": true})
}

// ResetPassword 重置密码（管理员操作）
func (h *UserHandler) ResetPassword(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	newPassword, err := h.authService.ResetPassword(id)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, gin.H{
		"success":      true,
		"new_password": newPassword,
		"message":      "密码已重置，请通过邮件通知用户",
	})
}

// Unlock 解锁用户
func (h *UserHandler) Unlock(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	if err := h.userRepo.UnlockUser(id); err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, gin.H{"success": true, "message": "用户已解锁"})
}