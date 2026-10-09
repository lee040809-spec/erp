package services

import (
	"errors"
	"strings"
	"sync"
	"time"

	"machining-erp/internal/models"
	"machining-erp/internal/repository"
	"machining-erp/pkg/jwt"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type LoginState struct {
	LastSeen    time.Time
	Attempts    int
	LockedUntil *time.Time
}

type AuthService struct {
	nextCleanup  time.Time
	mu           sync.RWMutex
	loginStates  map[int64]*LoginState // 按用户ID记录登录状态
	maxAttempts  int
	lockDuration time.Duration
	jwt          *jwt.JWT

	userRepo       *repository.UserRepository
	companyRepo    *repository.CompanyRepository
	resourceRepo   *repository.ResourceRepository
	permissionRepo *repository.PermissionRepository
	roleRepo       *repository.RoleRepository
}

func NewAuthService(
	userRepo *repository.UserRepository,
	companyRepo *repository.CompanyRepository,
	resourceRepo *repository.ResourceRepository,
	permissionRepo *repository.PermissionRepository,
	roleRepo *repository.RoleRepository,
	jwtConfig jwt.JWTConfig,
	maxAttempts int,
	lockDuration time.Duration,
) *AuthService {
	return &AuthService{
		loginStates:    make(map[int64]*LoginState),
		maxAttempts:    maxAttempts,
		lockDuration:   lockDuration,
		jwt:            jwt.NewJWT(jwtConfig),
		userRepo:       userRepo,
		companyRepo:    companyRepo,
		resourceRepo:   resourceRepo,
		permissionRepo: permissionRepo,
		roleRepo:       roleRepo,
	}
}

type LoginResult struct {
	Success           bool
	Token             string
	User              *UserInfo
	Error             string
	Locked            bool
	RemainingMinutes  int
	RemainingAttempts int
}

type UserInfo struct {
	ID        int64             `json:"id"`
	CorpID    int64             `json:"corp_id"`
	CorpName  string            `json:"corp_name"`
	Username  string            `json:"username"`
	Name      string            `json:"name"`
	Email     string            `json:"email"`
	RoleType  string            `json:"role_type"`
	RoleName  string            `json:"role_name"`
	ExpiredAt *time.Time        `json:"expired_at,omitempty"`
	Resources []models.Resource `json:"resources,omitempty"`
}

// CheckUser 检查用户是否存在，返回公司信息
func (s *AuthService) CheckUser(username string) (*UserInfo, string, error) {
	username = strings.TrimSpace(username)
	if username == "" {
		return nil, "请输入账号", nil
	}
	user, err := s.userRepo.FindByUsername(username)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, "账号不存在", nil
		}
		return nil, "系统错误", err
	}

	// 检查公司是否存在
	exists, err := s.companyRepo.ExistsByID(user.CorpID)
	if err != nil {
		return nil, "系统错误", err
	}
	if !exists {
		return nil, "所属公司不存在", nil
	}

	company, err := s.companyRepo.FindByID(user.CorpID)
	if err != nil {
		company = &models.Company{Name: ""}
	}

	roleType := ""
	if user.RoleType != nil {
		roleType = string(*user.RoleType)
	}
	return &UserInfo{
		ID:       user.ID,
		CorpID:   user.CorpID,
		CorpName: company.Name,
		Username: user.Username,
		Name:     user.Name,
		RoleType: roleType,
	}, "", nil
}

// Login 登录验证
func (s *AuthService) Login(username, password string) *LoginResult {
	// 1. 查询用户
	user, err := s.userRepo.FindByUsername(username)
	if err != nil {
		return &LoginResult{
			Success: false,
			Error:   "账号不存在",
		}
	}

	// 2. 检查公司是否存在（在验证密码之前）
	exists, err := s.companyRepo.ExistsByID(user.CorpID)
	if err != nil || !exists {
		return &LoginResult{
			Success: false,
			Error:   "所属公司不存在",
		}
	}

	// 3. 检查账号有效期
	now := time.Now()
	if user.ExpiredAt != nil && now.After(*user.ExpiredAt) {
		return &LoginResult{
			Success: false,
			Error:   "账号已过期",
		}
	}

	// 4. 检查锁定状态
	s.mu.Lock()
	defer s.mu.Unlock()

	s.cleanupLoginStates(now)
	state, exists := s.loginStates[user.ID]
	if !exists {
		state = &LoginState{}
		s.loginStates[user.ID] = state
	}

	state.LastSeen = now

	if state.LockedUntil != nil && now.Before(*state.LockedUntil) {
		remaining := int(state.LockedUntil.Sub(now).Minutes())
		return &LoginResult{
			Success:          false,
			Locked:           true,
			Error:            "账户已锁定",
			RemainingMinutes: remaining,
		}
	}

	// 4. 检查锁定是否过期
	if state.LockedUntil != nil && now.After(*state.LockedUntil) {
		state.Attempts = 0
		state.LockedUntil = nil
	}

	// 5. 验证密码
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)); err != nil {
		state.Attempts++
		remaining := s.maxAttempts - state.Attempts

		if state.Attempts >= s.maxAttempts {
			lockedUntil := now.Add(s.lockDuration)
			state.LockedUntil = &lockedUntil
			return &LoginResult{
				Success:          false,
				Locked:           true,
				Error:            "登录失败次数过多，账户已锁定",
				RemainingMinutes: int(s.lockDuration.Minutes()),
			}
		}

		return &LoginResult{
			Success:           false,
			Error:             "密码错误",
			RemainingAttempts: remaining,
		}
	}

	// 6. 登录成功，重置锁定状态
	state.Attempts = 0
	state.LockedUntil = nil

	company, _ := s.companyRepo.FindByID(user.CorpID)
	corpName := ""
	if company != nil {
		corpName = company.Name
	}

	// 7. 获取资源
	var resources []models.Resource
	if user.RoleType != nil && *user.RoleType == models.UserRoleTypeAdmin {
		// 管理员直接获取业务平台菜单
		resources, _ = s.resourceRepo.GetByPlatformType(models.PlatformTypeBusiness)
	} else if user.RoleID != nil && *user.RoleID > 0 {
		// 普通用户通过角色权限获取菜单
		resources, _ = s.resourceRepo.GetByRolePermissions(*user.RoleID)
	}

	// 8. 生成token
	roleType := ""
	if user.RoleType != nil {
		roleType = string(*user.RoleType)
	}

	// 9. 获取角色名称
	roleName := ""
	if roleType == string(models.UserRoleTypeAdmin) {
		// 管理员使用用户姓名
		roleName = user.Name
	} else if user.RoleID != nil && *user.RoleID > 0 {
		// 非管理员通过role_id查询角色名称
		role, err := s.roleRepo.FindByID(*user.RoleID)
		if err == nil && role != nil {
			roleName = role.Name
		}
	}

	token, err := s.jwt.GenerateToken(user.ID, user.CorpID, corpName, user.Username, roleType)
	if err != nil {
		return &LoginResult{
			Success: false,
			Error:   "生成token失败",
		}
	}

	return &LoginResult{
		Success: true,
		Token:   token,
		User: &UserInfo{
			ID:        user.ID,
			CorpID:    user.CorpID,
			CorpName:  corpName,
			Username:  user.Username,
			Name:      user.Name,
			Email:     user.Email,
			RoleType:  roleType,
			RoleName:  roleName,
			ExpiredAt: user.ExpiredAt,
			Resources: resources,
		},
	}
}

// ValidateToken 验证token
func (s *AuthService) ValidateToken(tokenString string) bool {
	_, err := s.jwt.ParseToken(tokenString)
	return err == nil
}

// GetClaimsFromToken 从token获取声明
func (s *AuthService) GetClaimsFromToken(tokenString string) *jwt.Claims {
	claims, err := s.jwt.ParseToken(tokenString)
	if err != nil {
		return nil
	}
	return claims
}

// Logout 登出（JWT无状态，这里可以做token黑名单处理）
func (s *AuthService) Logout(token string) {
	// JWT无状态，登出时可以在redis中维护黑名单
	// 暂时不做处理
}

// IsLocked 检查全局锁定状态（保留兼容）
func (s *AuthService) IsLocked() (bool, int) {
	return false, 0
}

// GetUser 获取用户信息（从token解析）
func (s *AuthService) GetUser(token string) *UserInfo {
	claims := s.GetClaimsFromToken(token)
	if claims == nil {
		return nil
	}

	return &UserInfo{
		ID:       claims.UserID,
		CorpID:   claims.CorpID,
		CorpName: claims.CorpName,
		Username: claims.Username,
		RoleType: claims.RoleType,
	}
}

// GetUserResources 获取用户资源（菜单）
func (s *AuthService) GetUserResources(userID int64, roleType string) ([]models.Resource, error) {
	if roleType == "admin" {
		// 管理员直接获取业务平台菜单
		return s.resourceRepo.GetByPlatformType(models.PlatformTypeBusiness)
	}
	// The caller supplies a user ID, which is not interchangeable with a role ID.
	user, err := s.userRepo.FindByID(userID)
	if err != nil {
		return nil, err
	}
	if user.RoleID == nil || *user.RoleID <= 0 {
		return []models.Resource{}, nil
	}
	return s.resourceRepo.GetByRolePermissions(*user.RoleID)
}

// Called with mu held. Preserve active locks and outstanding failed-attempt counters.
func (s *AuthService) cleanupLoginStates(now time.Time) {
	if now.Before(s.nextCleanup) {
		return
	}
	s.nextCleanup = now.Add(time.Hour)
	for id, state := range s.loginStates {
		if now.Sub(state.LastSeen) < 24*time.Hour {
			continue
		}
		if state.LockedUntil != nil && now.Before(*state.LockedUntil) {
			continue
		}
		if state.Attempts > 0 && state.LockedUntil == nil {
			continue
		}
		delete(s.loginStates, id)
	}
}
