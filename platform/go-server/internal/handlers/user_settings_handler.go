package handlers

import (
	"net/http"

	"machining-erp/internal/repository"
	"machining-erp/internal/services"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

type UserSettingsHandler struct {
	userRepo *repository.UserRepository
	emailSvc *services.EmailService
	authSvc  *services.AuthService
}

func NewUserSettingsHandler(userRepo *repository.UserRepository, emailSvc *services.EmailService, authSvc *services.AuthService) *UserSettingsHandler {
	return &UserSettingsHandler{
		userRepo: userRepo,
		emailSvc: emailSvc,
		authSvc:  authSvc,
	}
}

// ChangePasswordRequest 修改密码请求
type ChangePasswordRequest struct {
	UserID      int64  `json:"user_id" binding:"required"`
	OldPassword string `json:"old_password" binding:"required"`
	NewPassword string `json:"new_password" binding:"required"`
}

// ChangePassword 修改密码
func (h *UserSettingsHandler) ChangePassword(c *gin.Context) {
	var req ChangePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}

	// 从token获取corp_id
	token := c.GetHeader("Authorization")
	if token == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未授权"})
		return
	}
	claims := h.authSvc.GetClaimsFromToken(token)
	if claims == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "token无效"})
		return
	}

	// 查询用户
	user, err := h.userRepo.FindByIDAndCorpID(req.UserID, claims.CorpID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "用户不存在"})
		return
	}

	// 验证旧密码
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.OldPassword)); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "原密码错误"})
		return
	}

	// 加密新密码
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "密码加密失败"})
		return
	}

	// 更新密码
	if err := h.userRepo.UpdatePassword(req.UserID, claims.CorpID, string(hashedPassword)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "密码更新失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "message": "密码修改成功"})
}

// ResetPasswordRequest 密码重置请求
type ResetPasswordRequest struct {
	UserID int64 `json:"user_id" binding:"required"`
}

// ResetPassword 密码重置
func (h *UserSettingsHandler) ResetPassword(c *gin.Context) {
	var req ResetPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}

	// 从token获取corp_id
	token := c.GetHeader("Authorization")
	if token == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未授权"})
		return
	}
	claims := h.authSvc.GetClaimsFromToken(token)
	if claims == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "token无效"})
		return
	}

	// 查询用户
	user, err := h.userRepo.FindByIDAndCorpID(req.UserID, claims.CorpID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "用户不存在"})
		return
	}

	// 生成随机密码
	newPassword := h.emailSvc.GenerateRandomPassword()

	// 加密新密码
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "密码加密失败"})
		return
	}

	// 发送邮件
	emailErr := h.emailSvc.SendResetPasswordEmail(user, newPassword)

	// 更新密码
	if err := h.userRepo.UpdatePassword(req.UserID, claims.CorpID, string(hashedPassword)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "密码更新失败"})
		return
	}

	// 更新邮箱发送状态
	emailSentSuccess := emailErr == nil
	if err := h.userRepo.UpdateEmailSentSuccess(req.UserID, claims.CorpID, emailSentSuccess); err != nil {
		// 日志记录但不影响主要操作
	}

	if emailErr != nil {
		c.JSON(http.StatusOK, gin.H{
			"success":    true,
			"message":    "密码已重置，但邮件发送失败",
			"email_sent": false,
			"error":      emailErr.Error(),
		})
	} else {
		c.JSON(http.StatusOK, gin.H{
			"success":    true,
			"message":    "密码已重置并发送至邮箱",
			"email_sent": true,
		})
	}
}

// ChangeEmailRequest 修改邮箱请求
type ChangeEmailRequest struct {
	UserID   int64  `json:"user_id" binding:"required"`
	NewEmail string `json:"new_email" binding:"required,email"`
}

// ChangeEmail 修改邮箱
func (h *UserSettingsHandler) ChangeEmail(c *gin.Context) {
	var req ChangeEmailRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}

	// 从token获取corp_id
	token := c.GetHeader("Authorization")
	if token == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未授权"})
		return
	}
	claims := h.authSvc.GetClaimsFromToken(token)
	if claims == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "token无效"})
		return
	}

	// 查询用户验证存在
	_, err := h.userRepo.FindByIDAndCorpID(req.UserID, claims.CorpID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "用户不存在"})
		return
	}

	// 更新邮箱（邮箱修改时重置发送状态为false）
	if err := h.userRepo.UpdateEmail(req.UserID, claims.CorpID, req.NewEmail, false); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "邮箱更新失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "message": "邮箱修改成功"})
}

// GetUserInfo 获取当前用户信息
func (h *UserSettingsHandler) GetUserInfo(c *gin.Context) {
	token := c.GetHeader("Authorization")
	if token == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未授权"})
		return
	}
	claims := h.authSvc.GetClaimsFromToken(token)
	if claims == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "token无效"})
		return
	}

	user, err := h.userRepo.FindByID(claims.UserID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "用户不存在"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"id":                 user.ID,
		"username":           user.Username,
		"name":               user.Name,
		"email":              user.Email,
		"email_sent_success": user.EmailSentSuccess,
	})
}
