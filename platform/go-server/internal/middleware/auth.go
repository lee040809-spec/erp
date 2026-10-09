package middleware

import (
	"machining-erp/internal/services"
	"machining-erp/pkg/response"
	"strings"

	"github.com/gin-gonic/gin"
)

func AuthMiddleware(authService *services.AuthService) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 获取 token
		authHeader := c.GetHeader("Authorization")
		token := strings.TrimPrefix(authHeader, "Bearer ")

		if token == "" || !authService.ValidateToken(token) {
			response.Unauthorized(c, response.CodeTokenInvalid, response.CodeTokenInvalid.GetMessage())
			c.Abort()
			return
		}

		// 将用户信息存入context
		claims := authService.GetClaimsFromToken(token)
		if claims != nil {
			c.Set("userID", claims.UserID)
			c.Set("corpID", claims.CorpID)
			c.Set("corpName", claims.CorpName)
			c.Set("username", claims.Username)
			c.Set("roleType", claims.RoleType)
		}

		c.Next()
	}
}

// GetUserID 从context获取用户ID
func GetUserID(c *gin.Context) int64 {
	if v, exists := c.Get("userID"); exists {
		return v.(int64)
	}
	return 0
}

// GetCorpID 从context获取企业ID
func GetCorpID(c *gin.Context) int64 {
	if v, exists := c.Get("corpID"); exists {
		return v.(int64)
	}
	return 0
}

// GetCorpName 从context获取企业名称
func GetCorpName(c *gin.Context) string {
	if v, exists := c.Get("corpName"); exists {
		return v.(string)
	}
	return ""
}

// GetUsername 从context获取用户名
func GetUsername(c *gin.Context) string {
	if v, exists := c.Get("username"); exists {
		return v.(string)
	}
	return ""
}

// GetRoleType 从context获取角色类型
func GetRoleType(c *gin.Context) string {
	if v, exists := c.Get("roleType"); exists {
		return v.(string)
	}
	return ""
}
