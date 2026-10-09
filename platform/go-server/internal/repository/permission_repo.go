package repository

import (
	"gorm.io/gorm"
	"machining-erp/internal/models"
)

type PermissionRepository struct {
	db *gorm.DB
}

func NewPermissionRepository(db *gorm.DB) *PermissionRepository {
	return &PermissionRepository{db: db}
}

// GetByUserID 根据用户ID获取权限列表
func (r *PermissionRepository) GetByUserID(userID int64) ([]models.Permission, error) {
	var permissions []models.Permission
	err := r.db.Where("user_id = ?", userID).Find(&permissions).Error
	return permissions, err
}

// Create 创建权限
func (r *PermissionRepository) Create(permission *models.Permission) error {
	return r.db.Create(permission).Error
}

// DeleteByUserID 删除用户的所有权限
func (r *PermissionRepository) DeleteByUserID(userID int64) error {
	return r.db.Where("user_id = ?", userID).Delete(&models.Permission{}).Error
}

// GetByRoleID 获取角色的所有权限
func (r *PermissionRepository) GetByRoleID(roleID int64) ([]models.Permission, error) {
	var permissions []models.Permission
	err := r.db.Where("role_id = ?", roleID).Find(&permissions).Error
	return permissions, err
}

// GetByRoleIDWithResources 获取角色的权限及资源信息
func (r *PermissionRepository) GetByRoleIDWithResources(roleID int64) ([]map[string]interface{}, error) {
	var results []map[string]interface{}
	err := r.db.Table("permissions p").
		Select("p.id, p.role_id, p.resource_id, p.permission, r.resource_key, r.name, r.resource_type, r.icon, r.path, r.sort_order").
		Joins("LEFT JOIN resources r ON r.id = p.resource_id").
		Where("p.role_id = ?", roleID).
		Order("r.sort_order ASC").
		Find(&results).Error
	return results, err
}

// Delete 删除权限
func (r *PermissionRepository) Delete(id int64, corpID int64) error {
	return r.db.Where("id = ? AND corp_id = ?", id, corpID).Delete(&models.Permission{}).Error
}

// DeleteByRoleID 删除角色的所有权限
func (r *PermissionRepository) DeleteByRoleID(roleID int64, corpID int64) error {
	return r.db.Where("role_id = ? AND corp_id = ?", roleID, corpID).Delete(&models.Permission{}).Error
}

// FindByRoleAndResource 查询角色是否已有某资源权限
func (r *PermissionRepository) FindByRoleAndResource(roleID int64, resourceID int64) (*models.Permission, error) {
	var permission models.Permission
	err := r.db.Where("role_id = ? AND resource_id = ?", roleID, resourceID).First(&permission).Error
	if err != nil {
		return nil, err
	}
	return &permission, nil
}

// BatchCreate 批量创建权限
func (r *PermissionRepository) BatchCreate(permissions []models.Permission) error {
	return r.db.Create(&permissions).Error
}
