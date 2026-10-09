package handlers

import (
	"machining-erp/internal/services"

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

// CheckUser 检查用户是否存在
func (h *AuthHandler) CheckUser(c *gin.Context) {
	var req struct {
		Username string `json:"username"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": "Invalid request"})
		return
	}

	userInfo, errMsg, err := h.authService.CheckUser(req.Username)
	if err != nil {
		c.JSON(500, gin.H{"error": "系统错误"})
		return
	}
	if errMsg != "" {
		c.JSON(400, gin.H{"error": errMsg})
		return
	}

	c.JSON(200, gin.H{
		"success": true,
		"user":    userInfo,
	})
}

// Login 登录
func (h *AuthHandler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": "Invalid request"})
		return
	}

	result := h.authService.Login(req.Username, req.Password)

	if result.Success {
		c.JSON(200, gin.H{
			"success": true,
			"token":   result.Token,
			"user":    result.User,
		})
	} else if result.Locked {
		c.JSON(403, gin.H{
			"success":          false,
			"error":            result.Error,
			"locked":           true,
			"remainingMinutes": result.RemainingMinutes,
		})
	} else {
		c.JSON(401, gin.H{
			"success":           false,
			"error":             result.Error,
			"remainingAttempts": result.RemainingAttempts,
		})
	}
}

// Status 获取认证状态
func (h *AuthHandler) Status(c *gin.Context) {
	token := c.GetHeader("Authorization")
	if len(token) > 7 && token[:7] == "Bearer " {
		token = token[7:]
	}

	if token != "" && h.authService.ValidateToken(token) {
		c.JSON(200, gin.H{
			"authenticated": true,
			"user":          h.authService.GetUser(token),
		})
	} else {
		c.JSON(200, gin.H{"authenticated": false})
	}
}

// Logout 登出
func (h *AuthHandler) Logout(c *gin.Context) {
	token := c.GetHeader("Authorization")
	if len(token) > 7 && token[:7] == "Bearer " {
		token = token[7:]
	}
	h.authService.Logout(token)
	c.JSON(200, gin.H{"success": true})
}

// GetCurrentUser 获取当前用户信息（包含资源）
func (h *AuthHandler) GetCurrentUser(c *gin.Context) {
	token := c.GetHeader("Authorization")
	if len(token) > 7 && token[:7] == "Bearer " {
		token = token[7:]
	}

	claims := h.authService.GetClaimsFromToken(token)
	if claims == nil {
		c.JSON(401, gin.H{"error": "无效的token"})
		return
	}

	resources, err := h.authService.GetUserResources(claims.UserID, claims.RoleType)
	if err != nil {
		c.JSON(500, gin.H{"error": "获取资源失败"})
		return
	}

	c.JSON(200, gin.H{
		"id":        claims.UserID,
		"corp_id":   claims.CorpID,
		"corp_name": claims.CorpName,
		"username":  claims.Username,
		"role_type": claims.RoleType,
		"resources": resources,
	})
}
