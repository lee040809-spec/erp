package services

import (
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"portal-erp/internal/models"
	"portal-erp/internal/repository"
	"sync"
	"time"
)

type AuthService struct {
	userRepo       *repository.UserRepository
	adminUser      string
	adminPassword  string
	maxAttempts    int
	lockDuration   time.Duration
	tokens         map[string]int // token -> userID
	tokenMutex     sync.RWMutex
}

func NewAuthService(userRepo *repository.UserRepository, adminUser, adminPassword string, maxAttempts int, lockDuration time.Duration) *AuthService {
	return &AuthService{
		userRepo:      userRepo,
		adminUser:     adminUser,
		adminPassword: adminPassword,
		maxAttempts:   maxAttempts,
		lockDuration:  lockDuration,
		tokens:        make(map[string]int),
	}
}

// Login 验证登录，返回token或错误
func (s *AuthService) Login(username, password string) (string, error) {
	user, err := s.userRepo.GetByUsername(username)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", errors.New("用户名或密码错误")
		}
		return "", err
	}

	// 检查是否被锁定
	if user.LockUntil != nil && user.LockUntil.After(time.Now()) {
		return "", errors.New("账号已被锁定，请稍后再试")
	}

	// 验证密码
	if !s.verifyPassword(password, user.PasswordHash) {
		// 增加失败次数
		attempts := user.LoginAttempts + 1
		if attempts >= s.maxAttempts {
			// 锁定账号
			lockUntil := time.Now().Add(s.lockDuration)
			s.userRepo.LockUser(user.ID, lockUntil)
			return "", errors.New("登录失败次数过多，账号已被锁定30分钟")
		}
		s.userRepo.UpdateLoginAttempts(user.ID, attempts)
		return "", errors.New("用户名或密码错误")
	}

	// 登录成功，重置失败次数
	s.userRepo.UpdateLoginAttempts(user.ID, 0)
	s.userRepo.UnlockUser(user.ID)

	// 生成token
	token := s.generateToken()
	s.tokenMutex.Lock()
	s.tokens[token] = user.ID
	s.tokenMutex.Unlock()

	return token, nil
}

// Logout 登出
func (s *AuthService) Logout(token string) {
	s.tokenMutex.Lock()
	delete(s.tokens, token)
	s.tokenMutex.Unlock()
}

// ValidateToken 验证token，返回用户ID
func (s *AuthService) ValidateToken(token string) (int, error) {
	s.tokenMutex.RLock()
	userID, exists := s.tokens[token]
	s.tokenMutex.RUnlock()

	if !exists {
		return 0, errors.New("无效的token")
	}
	return userID, nil
}

// GetUserByID 根据ID获取用户
func (s *AuthService) GetUserByID(id int) (*models.User, error) {
	return s.userRepo.GetByID(id)
}

// ChangePassword 修改密码
func (s *AuthService) ChangePassword(userID int, oldPassword, newPassword string) error {
	user, err := s.userRepo.GetByID(userID)
	if err != nil {
		return err
	}

	if !s.verifyPassword(oldPassword, user.PasswordHash) {
		return errors.New("原密码错误")
	}

	newHash := s.HashPassword(newPassword)
	return s.userRepo.UpdatePassword(userID, newHash)
}

// ResetPassword 重置密码（管理员操作）
func (s *AuthService) ResetPassword(userID int) (string, error) {
	newPassword := s.generateRandomPassword()
	newHash := s.HashPassword(newPassword)
	err := s.userRepo.UpdatePassword(userID, newHash)
	if err != nil {
		return "", err
	}
	return newPassword, nil
}

// hashPassword 密码哈希
func (s *AuthService) HashPassword(password string) string {
	hash := sha256.Sum256([]byte(password + "portal_salt_2026"))
	return hex.EncodeToString(hash[:])
}

// verifyPassword 验证密码
func (s *AuthService) verifyPassword(password, hash string) bool {
	return s.HashPassword(password) == hash
}

// generateToken 生成token
func (s *AuthService) generateToken() string {
	b := make([]byte, 32)
	rand.Read(b)
	return hex.EncodeToString(b)
}

// generateRandomPassword 生成随机密码
func (s *AuthService) generateRandomPassword() string {
	const chars = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789!@#$%"
	b := make([]byte, 12)
	rand.Read(b)
	for i := range b {
		b[i] = chars[int(b[i])%len(chars)]
	}
	return string(b)
}

// InitAdmin 初始化管理员账号
func (s *AuthService) InitAdmin() error {
	_, err := s.userRepo.GetByUsername(s.adminUser)
	if err == sql.ErrNoRows {
		admin := &models.User{
			Username:     s.adminUser,
			PasswordHash: s.HashPassword(s.adminPassword),
			Email:        "admin@portal.local",
			RoleID:       1, // 管理员角色
		}
		return s.userRepo.Create(admin)
	}
	return nil
}