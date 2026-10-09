package handlers

import (
	"net/http"
	"strconv"

	"machining-erp/internal/models"
	"machining-erp/internal/repository"
	"machining-erp/internal/services"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

type UserHandler struct {
	userRepo *repository.UserRepository
	roleRepo *repository.RoleRepository
	authSvc  *services.AuthService
}

func NewUserHandler(userRepo *repository.UserRepository, roleRepo *repository.RoleRepository, authSvc *services.AuthService) *UserHandler {
	return &UserHandler{
		userRepo: userRepo,
		roleRepo: roleRepo,
		authSvc:  authSvc,
	}
}

// extractToken 从Authorization header中提取token
func extractToken(c *gin.Context) string {
	token := c.GetHeader("Authorization")
	if len(token) > 7 && token[:7] == "Bearer " {
		token = token[7:]
	}
	return token
}

// List 获取用户列表
func (h *UserHandler) List(c *gin.Context) {
	token := extractToken(c)
	if token == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未授权"})
		return
	}
	claims := h.authSvc.GetClaimsFromToken(token)
	if claims == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "token无效"})
		return
	}

	// 分页参数
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))

	// 筛选参数
	filters := make(map[string]string)
	if username := c.Query("username"); username != "" {
		filters["username"] = username
	}
	if name := c.Query("name"); name != "" {
		filters["name"] = name
	}
	if status := c.Query("status"); status != "" {
		filters["status"] = status
	}
	if roleType := c.Query("role_type"); roleType != "" {
		filters["role_type"] = roleType
	}

	users, total, err := h.userRepo.List(claims.CorpID, page, pageSize, filters)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取用户列表失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":     users,
		"total":    total,
		"page":     page,
		"pageSize": pageSize,
	})
}

// CreateRequest 创建用户请求
type UserCreateRequest struct {
	Username string               `json:"username" binding:"required"`
	Password string               `json:"password" binding:"required"`
	Name     string               `json:"name" binding:"required"`
	Email    string               `json:"email"`
	Phone    string               `json:"phone"`
	RoleType *models.UserRoleType `json:"role_type"`
	RoleID   *int64               `json:"role_id"`
	Status   string               `json:"status"`
}

// Create 创建用户
func (h *UserHandler) Create(c *gin.Context) {
	token := extractToken(c)
	if token == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未授权"})
		return
	}
	claims := h.authSvc.GetClaimsFromToken(token)
	if claims == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "token无效"})
		return
	}

	var req UserCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}

	// 检查用户名是否已存在
	existUser, _ := h.userRepo.FindByUsernameAndCorpID(req.Username, claims.CorpID)
	if existUser != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "用户名已存在"})
		return
	}

	// 加密密码
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "密码加密失败"})
		return
	}

	// 创建用户
	user := &models.User{
		CorpID:   claims.CorpID,
		Username: req.Username,
		Password: string(hashedPassword),
		Name:     req.Name,
		Email:    req.Email,
		Phone:    req.Phone,
		RoleType: req.RoleType,
		RoleID:   req.RoleID,
		Status:   req.Status,
	}
	if user.Status == "" {
		user.Status = "active"
	}

	if err := h.userRepo.Create(user); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建用户失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    user,
	})
}

// UpdateRequest 更新用户请求
type UserUpdateRequest struct {
	Name     string               `json:"name"`
	Email    string               `json:"email"`
	Phone    string               `json:"phone"`
	RoleType *models.UserRoleType `json:"role_type"`
	RoleID   *int64               `json:"role_id"`
	Status   string               `json:"status"`
}

// Update 更新用户
func (h *UserHandler) Update(c *gin.Context) {
	token := extractToken(c)
	if token == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未授权"})
		return
	}
	claims := h.authSvc.GetClaimsFromToken(token)
	if claims == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "token无效"})
		return
	}

	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}

	// 查询用户
	user, err := h.userRepo.FindByIDAndCorpID(id, claims.CorpID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "用户不存在"})
		return
	}

	var req UserUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}

	// 更新字段
	if req.Name != "" {
		user.Name = req.Name
	}
	if req.Email != "" {
		user.Email = req.Email
	}
	if req.Phone != "" {
		user.Phone = req.Phone
	}
	if req.RoleType != nil {
		user.RoleType = req.RoleType
	}
	if req.RoleID != nil {
		user.RoleID = req.RoleID
	}
	if req.Status != "" {
		user.Status = req.Status
	}

	if err := h.userRepo.Update(user); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "更新用户失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    user,
	})
}

// Delete 删除用户
func (h *UserHandler) Delete(c *gin.Context) {
	token := extractToken(c)
	if token == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未授权"})
		return
	}
	claims := h.authSvc.GetClaimsFromToken(token)
	if claims == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "token无效"})
		return
	}

	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}

	// 不能删除自己
	if id == claims.UserID {
		c.JSON(http.StatusBadRequest, gin.H{"error": "不能删除自己"})
		return
	}

	// 查询用户验证存在及角色类型
	user, err := h.userRepo.FindByIDAndCorpID(id, claims.CorpID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "用户不存在"})
		return
	}

	// 管理员用户不可删除
	if user.RoleType != nil && *user.RoleType == models.UserRoleTypeAdmin {
		c.JSON(http.StatusBadRequest, gin.H{"error": "管理员用户不可删除"})
		return
	}

	if err := h.userRepo.Delete(id, claims.CorpID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "删除用户失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}

// BindRole 绑定角色
func (h *UserHandler) BindRole(c *gin.Context) {
	token := extractToken(c)
	if token == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未授权"})
		return
	}
	claims := h.authSvc.GetClaimsFromToken(token)
	if claims == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "token无效"})
		return
	}

	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}

	// 查询用户验证存在
	user, err := h.userRepo.FindByIDAndCorpID(id, claims.CorpID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "用户不存在"})
		return
	}

	// 管理员用户不可绑定角色
	if user.RoleType != nil && *user.RoleType == models.UserRoleTypeAdmin {
		c.JSON(http.StatusBadRequest, gin.H{"error": "管理员用户不可绑定角色"})
		return
	}

	var req struct {
		RoleID *int64 `json:"role_id"` // 可选，null表示解绑
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}

	// 解绑角色（role_id 为 null）
	if req.RoleID == nil {
		user.RoleID = nil
		if err := h.userRepo.Update(user); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "解绑角色失败"})
			return
		}
		// 重新查询用户以获取关联的角色信息
		updatedUser, err := h.userRepo.FindByIDAndCorpID(id, claims.CorpID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "获取用户信息失败"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"success": true, "data": updatedUser})
		return
	}

	// 验证角色存在且属于当前企业
	role, err := h.roleRepo.FindByID(*req.RoleID)
	if err != nil || role.CorpID != claims.CorpID {
		c.JSON(http.StatusNotFound, gin.H{"error": "角色不存在"})
		return
	}

	// 更新用户的角色ID
	user.RoleID = req.RoleID
	if err := h.userRepo.Update(user); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "绑定角色失败"})
		return
	}

	// 重新查询用户以获取关联的角色信息
	updatedUser, err := h.userRepo.FindByIDAndCorpID(id, claims.CorpID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取用户信息失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "data": updatedUser})
}

// ResetPassword 重置用户密码
func (h *UserHandler) ResetPassword(c *gin.Context) {
	token := extractToken(c)
	if token == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未授权"})
		return
	}
	claims := h.authSvc.GetClaimsFromToken(token)
	if claims == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "token无效"})
		return
	}

	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}

	var req struct {
		NewPassword string `json:"new_password" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}

	// 查询用户验证存在
	_, err = h.userRepo.FindByIDAndCorpID(id, claims.CorpID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "用户不存在"})
		return
	}

	// 加密新密码
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "密码加密失败"})
		return
	}

	// 更新密码
	if err := h.userRepo.UpdatePassword(id, claims.CorpID, string(hashedPassword)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "密码更新失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "message": "密码已重置"})
}
