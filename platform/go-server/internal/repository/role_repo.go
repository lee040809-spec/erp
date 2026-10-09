package repository

import (
	"gorm.io/gorm"
	"machining-erp/internal/models"
)

type RoleRepository struct {
	db *gorm.DB
}

func NewRoleRepository(db *gorm.DB) *RoleRepository {
	return &RoleRepository{db: db}
}

// FindByID 根据ID查询角色
func (r *RoleRepository) FindByID(id int64) (*models.Role, error) {
	var role models.Role
	err := r.db.First(&role, id).Error
	if err != nil {
		return nil, err
	}
	return &role, nil
}

// GetByCorpID 根据企业ID获取角色列表
func (r *RoleRepository) GetByCorpID(corpID int64) ([]models.Role, error) {
	var roles []models.Role
	err := r.db.Where("corp_id = ? AND status = ?", corpID, "active").Find(&roles).Error
	return roles, err
}

// List 获取角色列表（带分页）
func (r *RoleRepository) List(corpID int64, page, pageSize int) ([]models.Role, int64, error) {
	var roles []models.Role
	var total int64

	query := r.db.Model(&models.Role{}).Where("corp_id = ?", corpID)

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	if err := query.Offset(offset).Limit(pageSize).Order("created_at DESC").Find(&roles).Error; err != nil {
		return nil, 0, err
	}

	return roles, total, nil
}

// Create 创建角色
func (r *RoleRepository) Create(role *models.Role) error {
	return r.db.Create(role).Error
}

// Update 更新角色
func (r *RoleRepository) Update(role *models.Role) error {
	return r.db.Save(role).Error
}

// Delete 删除角色
func (r *RoleRepository) Delete(id int64, corpID int64) error {
	return r.db.Where("id = ? AND corp_id = ?", id, corpID).Delete(&models.Role{}).Error
}
