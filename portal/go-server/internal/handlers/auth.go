package handlers

import (
	"portal-erp/internal/services"

	"github.com/gin-gonic/gin"
)

type AuthHandler struct {
	authService *services.AuthService
}

func NewAuthHandler(authService *services.AuthService) *AuthHandler {
	return &AuthHandler{authService: authService}
}

type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type ChangePasswordRequest struct {
	OldPassword string `json:"old_password"`
	NewPassword string `json:"new_password"`
}

// Login 登录
func (h *AuthHandler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": "请求参数错误"})
		return
	}

	token, err := h.authService.Login(req.Username, req.Password)
	if err != nil {
		c.JSON(401, gin.H{"success": false, "error": err.Error()})
		return
	}

	// 获取用户信息
	userID, _ := h.authService.ValidateToken(token)
	user, _ := h.authService.GetUserByID(userID)

	c.JSON(200, gin.H{
		"success": true,
		"token":   token,
		"user": gin.H{
			"id":       user.ID,
			"username": user.Username,
			"email":    user.Email,
			"role_id":  user.RoleID,
			"role_name": user.RoleName,
		},
	})
}

// Logout 登出
func (h *AuthHandler) Logout(c *gin.Context) {
	token := c.GetHeader("Authorization")
	if token != "" {
		h.authService.Logout(token)
	}
	c.JSON(200, gin.H{"success": true})
}

// Status 检查登录状态
func (h *AuthHandler) Status(c *gin.Context) {
	token := c.GetHeader("Authorization")
	if token == "" {
		c.JSON(401, gin.H{"authenticated": false})
		return
	}

	userID, err := h.authService.ValidateToken(token)
	if err != nil {
		c.JSON(401, gin.H{"authenticated": false})
		return
	}

	user, err := h.authService.GetUserByID(userID)
	if err != nil {
		c.JSON(401, gin.H{"authenticated": false})
		return
	}

	c.JSON(200, gin.H{
		"authenticated": true,
		"user": gin.H{
			"id":        user.ID,
			"username":  user.Username,
			"email":     user.Email,
			"role_id":   user.RoleID,
			"role_name": user.RoleName,
		},
	})
}

// ChangePassword 修改密码
func (h *AuthHandler) ChangePassword(c *gin.Context) {
	token := c.GetHeader("Authorization")
	userID, err := h.authService.ValidateToken(token)
	if err != nil {
		c.JSON(401, gin.H{"error": "未授权"})
		return
	}

	var req ChangePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": "请求参数错误"})
		return
	}

	if err := h.authService.ChangePassword(userID, req.OldPassword, req.NewPassword); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	c.JSON(200, gin.H{"success": true, "message": "密码修改成功"})
}