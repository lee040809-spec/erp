package services

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/smtp"
	"strings"

	"machining-erp/internal/config"
	"machining-erp/internal/models"
	"machining-erp/internal/repository"
)

type EmailService struct {
	smtpHost     string
	smtpPort     string
	smtpUsername string
	smtpPassword string
	fromEmail    string
	userRepo     *repository.UserRepository
}

func NewEmailService(cfg *config.Config, userRepo *repository.UserRepository) *EmailService {
	return &EmailService{
		smtpHost:     cfg.SMTPHost,
		smtpPort:     cfg.SMTPPort,
		smtpUsername: cfg.SMTPUsername,
		smtpPassword: cfg.SMTPPassword,
		fromEmail:    cfg.FromEmail,
		userRepo:     userRepo,
	}
}

// GenerateRandomPassword 生成8位随机密码
func (s *EmailService) GenerateRandomPassword() string {
	bytes := make([]byte, 4)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

// SendEmail 发送邮件
func (s *EmailService) SendEmail(to string, subject string, body string) error {
	if s.smtpHost == "" || s.smtpUsername == "" {
		return fmt.Errorf("SMTP配置不完整")
	}

	auth := smtp.PlainAuth("", s.smtpUsername, s.smtpPassword, s.smtpHost)

	msg := fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\n\r\n%s",
		s.fromEmail, to, subject, body)

	addr := fmt.Sprintf("%s:%s", s.smtpHost, s.smtpPort)
	return smtp.SendMail(addr, auth, s.fromEmail, []string{to}, []byte(msg))
}

// SendResetPasswordEmail 发送密码重置邮件
func (s *EmailService) SendResetPasswordEmail(user *models.User, newPassword string) error {
	if user.Email == "" || !strings.Contains(user.Email, "@") {
		return fmt.Errorf("用户邮箱格式不正确")
	}

	subject := "ERP系统密码重置通知"
	body := fmt.Sprintf(`
您好 %s，

您的ERP系统密码已被重置，新密码为：%s

请尽快登录系统修改密码。

此邮件由系统自动发送，请勿回复。
`, user.Name, newPassword)

	return s.SendEmail(user.Email, subject, body)
}
