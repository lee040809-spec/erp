package handlers

import (
	"portal-erp/internal/models"
	"portal-erp/internal/repository"
	"portal-erp/internal/services"
	"strconv"

	"github.com/gin-gonic/gin"
)

type SuperUserHandler struct {
	superUserRepo *repository.SuperUserRepository
	roleRepo      *repository.RoleRepository
	authService   *services.AuthService
}

func NewSuperUserHandler(superUserRepo *repository.SuperUserRepository, roleRepo *repository.RoleRepository, authService *services.AuthService) *SuperUserHandler {
	return &SuperUserHandler{
		superUserRepo: superUserRepo,
		roleRepo:      roleRepo,
		authService:   authService,
	}
}

// List 获取超级用户列表
func (h *SuperUserHandler) List(c *gin.Context) {
	users, err := h.superUserRepo.List()
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, users)
}

// Get 获取单个超级用户
func (h *SuperUserHandler) Get(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	user, err := h.superUserRepo.GetByID(id)
	if err != nil {
		c.JSON(404, gin.H{"error": "用户不存在"})
		return
	}
	c.JSON(200, user)
}

// Create 创建超级用户
func (h *SuperUserHandler) Create(c *gin.Context) {
	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
		Email    string `json:"email"`
		RoleID   int    `json:"role_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": "请求参数错误"})
		return
	}

	// 检查用户名是否已存在
	_, err := h.superUserRepo.GetByUsername(req.Username)
	if err == nil {
		c.JSON(400, gin.H{"error": "用户名已存在"})
		return
	}

	user := &models.SuperUser{
		Username:     req.Username,
		PasswordHash: h.authService.HashPassword(req.Password),
		Email:        req.Email,
		RoleID:       req.RoleID,
	}

	if err := h.superUserRepo.Create(user); err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	c.JSON(200, gin.H{"success": true, "id": user.ID})
}

// Update 更新超级用户
func (h *SuperUserHandler) Update(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))

	var req struct {
		Email  string `json:"email"`
		RoleID int    `json:"role_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": "请求参数错误"})
		return
	}

	user := &models.SuperUser{
		ID:     id,
		Email:  req.Email,
		RoleID: req.RoleID,
	}

	if err := h.superUserRepo.Update(user); err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	c.JSON(200, gin.H{"success": true})
}

// Delete 删除超级用户
func (h *SuperUserHandler) Delete(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	if err := h.superUserRepo.Delete(id); err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, gin.H{"success": true})
}

// ResetPassword 重置密码
func (h *SuperUserHandler) ResetPassword(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	newPassword, err := h.authService.ResetPassword(id)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, gin.H{
		"success":      true,
		"new_password": newPassword,
		"message":      "密码已重置",
	})
}

// Unlock 解锁超级用户
func (h *SuperUserHandler) Unlock(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	if err := h.superUserRepo.UnlockUser(id); err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, gin.H{"success": true, "message": "用户已解锁"})
}