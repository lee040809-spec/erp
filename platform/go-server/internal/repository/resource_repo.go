package repository

import (
	"gorm.io/gorm"
	"machining-erp/internal/models"
)

type ResourceRepository struct {
	db *gorm.DB
}

func NewResourceRepository(db *gorm.DB) *ResourceRepository {
	return &ResourceRepository{db: db}
}

// GetAll 获取所有资源
func (r *ResourceRepository) GetAll() ([]models.Resource, error) {
	var resources []models.Resource
	err := r.db.Where("status = ?", "active").Order("sort_order ASC").Find(&resources).Error
	return resources, err
}

// GetByIDs 根据ID列表获取资源
func (r *ResourceRepository) GetByIDs(ids []int64) ([]models.Resource, error) {
	var resources []models.Resource
	err := r.db.Where("id IN ? AND status = ?", ids, "active").Order("sort_order ASC").Find(&resources).Error
	return resources, err
}

// GetByRolePermissions 根据角色权限获取资源
func (r *ResourceRepository) GetByRolePermissions(roleID int64) ([]models.Resource, error) {
	var resources []models.Resource
	err := r.db.Table("resources r").
		Select("r.*").
		Joins("JOIN permissions p ON p.resource_id = r.id").
		Where("p.role_id = ? AND r.status = ?", roleID, "active").
		Order("r.sort_order ASC").
		Find(&resources).Error
	return resources, err
}

// GetByPlatformType 根据平台类型获取菜单资源
func (r *ResourceRepository) GetByPlatformType(platformType models.PlatformType) ([]models.Resource, error) {
	var resources []models.Resource
	err := r.db.Where("status = ? AND platform_type = ? AND resource_type = ?", "active", platformType, "menu").
		Order("sort_order ASC").
		Find(&resources).Error
	return resources, err
}

// List 获取资源列表（带分页和筛选）
func (r *ResourceRepository) List(page, pageSize int, filters map[string]string) ([]models.Resource, int64, error) {
	var resources []models.Resource
	var total int64

	query := r.db.Model(&models.Resource{})

	// 应用筛选
	if resourceType, ok := filters["resource_type"]; ok && resourceType != "" {
		query = query.Where("resource_type = ?", resourceType)
	}
	if platformType, ok := filters["platform_type"]; ok && platformType != "" {
		query = query.Where("platform_type = ?", platformType)
	}
	if status, ok := filters["status"]; ok && status != "" {
		query = query.Where("status = ?", status)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	if err := query.Offset(offset).Limit(pageSize).Order("sort_order ASC").Find(&resources).Error; err != nil {
		return nil, 0, err
	}

	return resources, total, nil
}

// GetWithPageResources 获取page_resources不为空的资源
func (r *ResourceRepository) GetWithPageResources() ([]models.Resource, error) {
	var resources []models.Resource
	err := r.db.Where("status = ? AND page_resources IS NOT NULL AND page_resources != '' AND page_resources != '[]'", "active").
		Order("sort_order ASC").
		Find(&resources).Error
	return resources, err
}
