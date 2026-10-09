package middleware

import (
	"portal-erp/internal/services"

	"github.com/gin-gonic/gin"
)

func CORS() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Authorization")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}

func AuthMiddleware(authService *services.AuthService) gin.HandlerFunc {
	return func(c *gin.Context) {
		token := c.GetHeader("Authorization")
		if token == "" {
			c.JSON(401, gin.H{"error": "未授权"})
			c.Abort()
			return
		}

		userID, err := authService.ValidateToken(token)
		if err != nil {
			c.JSON(401, gin.H{"error": "无效的token"})
			c.Abort()
			return
		}

		// 获取用户信息
		user, err := authService.GetUserByID(userID)
		if err != nil {
			c.JSON(401, gin.H{"error": "用户不存在"})
			c.Abort()
			return
		}

		// 将用户信息存入上下文
		c.Set("user_id", userID)
		c.Set("role_id", user.RoleID)
		c.Set("username", user.Username)

		c.Next()
	}
}