package repository

import (
	"gorm.io/gorm"
	"machining-erp/internal/models"
)

type CompanyRepository struct {
	db *gorm.DB
}

func NewCompanyRepository(db *gorm.DB) *CompanyRepository {
	return &CompanyRepository{db: db}
}

// FindByID 根据ID查询公司
func (r *CompanyRepository) FindByID(id int64) (*models.Company, error) {
	var company models.Company
	err := r.db.First(&company, id).Error
	if err != nil {
		return nil, err
	}
	return &company, nil
}

// ExistsByID 检查公司是否存在
func (r *CompanyRepository) ExistsByID(id int64) (bool, error) {
	var count int64
	err := r.db.Model(&models.Company{}).Where("id = ? AND status = ?", id, "active").Count(&count).Error
	return count > 0, err
}
