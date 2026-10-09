package repository

import (
	"gorm.io/gorm"
	"machining-erp/internal/models"
)

type UserRepository struct {
	db *gorm.DB
}

func NewUserRepository(db *gorm.DB) *UserRepository {
	return &UserRepository{db: db}
}

// FindByUsername 根据用户名查询用户
func (r *UserRepository) FindByUsername(username string) (*models.User, error) {
	var user models.User
	err := r.db.Where("username = ?", username).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// FindByID 根据ID查询用户
func (r *UserRepository) FindByID(id int64) (*models.User, error) {
	var user models.User
	err := r.db.First(&user, id).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// Create 创建用户
func (r *UserRepository) Create(user *models.User) error {
	return r.db.Create(user).Error
}

// Update 更新用户
func (r *UserRepository) Update(user *models.User) error {
	return r.db.Save(user).Error
}

// UpdatePassword 更新用户密码
func (r *UserRepository) UpdatePassword(userID int64, corpID int64, newPassword string) error {
	return r.db.Model(&models.User{}).Where("id = ? AND corp_id = ?", userID, corpID).Update("password", newPassword).Error
}

// UpdateEmail 更新用户邮箱
func (r *UserRepository) UpdateEmail(userID int64, corpID int64, newEmail string, emailSentSuccess bool) error {
	return r.db.Model(&models.User{}).Where("id = ? AND corp_id = ?", userID, corpID).Updates(map[string]interface{}{
		"email":              newEmail,
		"email_sent_success": emailSentSuccess,
	}).Error
}

// UpdateEmailSentSuccess 更新邮箱发送成功标识
func (r *UserRepository) UpdateEmailSentSuccess(userID int64, corpID int64, success bool) error {
	return r.db.Model(&models.User{}).Where("id = ? AND corp_id = ?", userID, corpID).Update("email_sent_success", success).Error
}

// FindByIDAndCorpID 根据ID和CorpID查询用户
func (r *UserRepository) FindByIDAndCorpID(userID int64, corpID int64) (*models.User, error) {
	var user models.User
	err := r.db.Preload("Role").Where("id = ? AND corp_id = ?", userID, corpID).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// List 获取用户列表（带分页和筛选）
func (r *UserRepository) List(corpID int64, page, pageSize int, filters map[string]string) ([]models.User, int64, error) {
	var users []models.User
	var total int64

	query := r.db.Model(&models.User{}).Where("corp_id = ?", corpID)

	// 应用筛选
	if username, ok := filters["username"]; ok && username != "" {
		query = query.Where("username LIKE ?", "%"+username+"%")
	}
	if name, ok := filters["name"]; ok && name != "" {
		query = query.Where("name LIKE ?", "%"+name+"%")
	}
	if status, ok := filters["status"]; ok && status != "" {
		query = query.Where("status = ?", status)
	}
	if roleType, ok := filters["role_type"]; ok && roleType != "" {
		query = query.Where("role_type = ?", roleType)
	}

	// 获取总数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 分页查询
	offset := (page - 1) * pageSize
	if err := query.Preload("Role").Offset(offset).Limit(pageSize).Order("created_at DESC").Find(&users).Error; err != nil {
		return nil, 0, err
	}

	return users, total, nil
}

// Delete 删除用户
func (r *UserRepository) Delete(id int64, corpID int64) error {
	return r.db.Where("id = ? AND corp_id = ?", id, corpID).Delete(&models.User{}).Error
}

// FindByUsernameAndCorpID 根据用户名和企业ID查询用户
func (r *UserRepository) FindByUsernameAndCorpID(username string, corpID int64) (*models.User, error) {
	var user models.User
	err := r.db.Where("username = ? AND corp_id = ?", username, corpID).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}
