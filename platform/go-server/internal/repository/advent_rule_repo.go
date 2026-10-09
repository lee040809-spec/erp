package repository

import (
	"gorm.io/gorm"
	"machining-erp/internal/models"
)

type AdventRuleRepository struct {
	db *gorm.DB
}

func NewAdventRuleRepository(db *gorm.DB) *AdventRuleRepository {
	return &AdventRuleRepository{db: db}
}

func (r *AdventRuleRepository) GetByCorpID(corpID int64, name string) ([]models.AdventRule, error) {
	var rules []models.AdventRule
	query := r.db.Where("corp_id=?", corpID).Order("id")
	if name != "" {
		query = query.Where("name LIKE ?", "%"+name+"%")
	}
	err := query.Find(&rules).Error
	return rules, err
}

func (r *AdventRuleRepository) Create(rule *models.AdventRule) (int64, error) {
	err := r.db.Create(rule).Error
	if err != nil {
		return 0, err
	}
	return int64(rule.ID), nil
}

func (r *AdventRuleRepository) Update(rule *models.AdventRule) error {
	var existing models.AdventRule
	if e := r.db.Where("id=? AND corp_id=?", rule.ID, rule.CorpID).First(&existing).Error; e != nil {
		return e
	}
	return r.db.Model(&existing).Where("corp_id=?", rule.CorpID).Updates(map[string]interface{}{
		"name":          rule.Name,
		"description":   rule.Description,
		"formula":       rule.Formula,
		"target_status": rule.TargetStatus,
		"scopeType":     rule.ScopeType,
		"ruleType":      rule.RuleType,
	}).Error
}

func (r *AdventRuleRepository) Delete(corpID, id int64) error {
	result := r.db.Where("id=? AND corp_id=?", id, corpID).Delete(&models.AdventRule{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}
