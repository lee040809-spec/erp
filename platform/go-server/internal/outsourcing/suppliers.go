package outsourcing

import (
	"errors"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"machining-erp/internal/models"
	"machining-erp/internal/productionflow"
	"strings"
	"time"
)

type SupplierInput struct {
	Name         string `json:"name"`
	ContactName  string `json:"contact_name"`
	Phone        string `json:"phone"`
	Address      string `json:"address"`
	Capabilities string `json:"capabilities"`
	Notes        string `json:"notes"`
	Active       bool   `json:"active"`
}

func SaveSupplier(db *gorm.DB, corp, id int64, in SupplierInput) (models.OutsourcingSupplier, error) {
	var out models.OutsourcingSupplier
	in.Name = strings.TrimSpace(in.Name)
	if in.Name == "" || !productionflow.Text(in.Name, 100) || !productionflow.Text(in.ContactName, 100) || !productionflow.Text(in.Phone, 60) || !productionflow.Text(in.Address, 300) || !productionflow.Text(in.Capabilities, 300) || !productionflow.Text(in.Notes, 500) {
		return out, errors.New("供应商名称必填，其他资料请勿超过长度限制")
	}
	e := db.Transaction(func(tx *gorm.DB) error {
		if id > 0 {
			if e := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("corp_id=? AND id=?", corp, id).Take(&out).Error; e != nil {
				return errors.New("供应商不存在或不属于本公司")
			}
		}
		var n int64
		if e := tx.Model(&models.OutsourcingSupplier{}).Where("corp_id=? AND name=? AND id<>?", corp, in.Name, id).Count(&n).Error; e != nil {
			return e
		}
		if n > 0 {
			return errors.New("本公司已有同名供应商，请编辑已有资料")
		}
		out.CorpID = corp
		out.Name = in.Name
		out.ContactName = strings.TrimSpace(in.ContactName)
		out.Phone = strings.TrimSpace(in.Phone)
		out.Address = strings.TrimSpace(in.Address)
		out.Capabilities = strings.TrimSpace(in.Capabilities)
		out.Notes = in.Notes
		out.Active = in.Active
		out.UpdatedAt = time.Now()
		if id == 0 {
			out.CreatedAt = time.Now()
			return tx.Create(&out).Error
		}
		return tx.Save(&out).Error
	})
	return out, e
}
func activeSupplier(tx *gorm.DB, corp, id int64) (models.OutsourcingSupplier, error) {
	var s models.OutsourcingSupplier
	if e := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("corp_id=? AND id=?", corp, id).Take(&s).Error; e != nil {
		return s, errors.New("请选择本公司的外协供应商")
	}
	if !s.Active {
		return s, errors.New("该供应商已停用，不能新增订单或发出")
	}
	return s, nil
}
