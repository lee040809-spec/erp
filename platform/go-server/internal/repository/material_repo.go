package repository

import (
	"gorm.io/gorm"
	"machining-erp/internal/models"
)

type MaterialRepository struct {
	db *gorm.DB
}

func NewMaterialRepository(db *gorm.DB) *MaterialRepository {
	return &MaterialRepository{db: db}
}

func (r *MaterialRepository) GetByCorpID(corpID int64) ([]models.Material, error) {
	var materials []models.Material
	err := r.db.Where("corp_id=?", corpID).Order("id DESC").Find(&materials).Error
	return materials, err
}

func (r *MaterialRepository) Create(m *models.Material) (int64, error) {
	err := r.db.Create(m).Error
	if err != nil {
		return 0, err
	}
	return int64(m.ID), nil
}

type RemnantRepository struct {
	db *gorm.DB
}

func NewRemnantRepository(db *gorm.DB) *RemnantRepository {
	return &RemnantRepository{db: db}
}

func (r *RemnantRepository) GetByCorpID(corpID int64) ([]models.Remnant, error) {
	var remnants []models.Remnant
	err := r.db.Where("corp_id=?", corpID).Order("created_at DESC,id DESC").Find(&remnants).Error
	return remnants, err
}

func (r *RemnantRepository) Create(remnant *models.Remnant) (int64, error) {
	err := r.db.Transaction(func(tx *gorm.DB) error {
		var material models.Material
		if e := tx.Where("id=? AND corp_id=?", remnant.MaterialID, remnant.CorpID).First(&material).Error; e != nil {
			return e
		}
		return tx.Create(remnant).Error
	})
	if err != nil {
		return 0, err
	}
	return int64(remnant.ID), nil
}
