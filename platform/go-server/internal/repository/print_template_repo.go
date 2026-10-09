package repository

import (
	"gorm.io/gorm"
	"machining-erp/internal/models"
)

type PrintTemplateRepository struct {
	db *gorm.DB
}

func NewPrintTemplateRepository(db *gorm.DB) *PrintTemplateRepository {
	return &PrintTemplateRepository{db: db}
}

func (r *PrintTemplateRepository) List(corpID int64, name string) ([]models.PrintTemplate, error) {
	var templates []models.PrintTemplate
	query := r.db.Where("corp_id = ?", corpID).Order("id DESC")
	if name != "" {
		query = query.Where("name LIKE ?", "%"+name+"%")
	}
	err := query.Find(&templates).Error
	return templates, err
}

func (r *PrintTemplateRepository) FindByID(corpID int64, id int64) (*models.PrintTemplate, error) {
	var template models.PrintTemplate
	err := r.db.Where("corp_id = ? AND id = ?", corpID, id).First(&template).Error
	if err != nil {
		return nil, err
	}
	return &template, nil
}

func (r *PrintTemplateRepository) FindByName(corpID int64, name string) (*models.PrintTemplate, error) {
	var template models.PrintTemplate
	err := r.db.Where("corp_id = ? AND name = ?", corpID, name).First(&template).Error
	if err != nil {
		return nil, err
	}
	return &template, nil
}

func (r *PrintTemplateRepository) Create(template *models.PrintTemplate) error {
	return r.db.Create(template).Error
}

func (r *PrintTemplateRepository) Update(template *models.PrintTemplate) error {
	updates := map[string]interface{}{
		"name":           template.Name,
		"menu_route":     template.MenuRoute,
		"button_key":     template.ButtonKey,
		"button_name":    template.ButtonName,
		"template":       template.Template,
		"excel_data":     template.ExcelData,
		"excel_filename": template.ExcelFilename,
	}
	return r.db.Model(template).Where("corp_id = ? AND id = ?", template.CorpID, template.ID).Updates(updates).Error
}

// UpdateExcelData 只更新Excel相关字段
func (r *PrintTemplateRepository) UpdateExcelData(corpID int64, id int64, template string, excelData []byte, excelFilename string) error {
	return r.db.Model(&models.PrintTemplate{}).Where("corp_id = ? AND id = ?", corpID, id).Updates(map[string]interface{}{
		"template":       template,
		"excel_data":     excelData,
		"excel_filename": excelFilename,
	}).Error
}

func (r *PrintTemplateRepository) Delete(corpID int64, id int64) error {
	return r.db.Where("corp_id = ? AND id = ?", corpID, id).Delete(&models.PrintTemplate{}).Error
}

// PartialUpdate 部分更新字段
func (r *PrintTemplateRepository) PartialUpdate(corpID int64, id int64, updates map[string]interface{}) error {
	return r.db.Model(&models.PrintTemplate{}).Where("corp_id = ? AND id = ?", corpID, id).Updates(updates).Error
}

// FindByButtonKey 根据按钮标识和菜单路由查询模板
func (r *PrintTemplateRepository) FindByButtonKey(corpID int64, menuRoute string, buttonKey string) (*models.PrintTemplate, error) {
	var template models.PrintTemplate
	err := r.db.Where("corp_id = ? AND menu_route = ? AND button_key = ?", corpID, menuRoute, buttonKey).First(&template).Error
	if err != nil {
		return nil, err
	}
	return &template, nil
}
