package repository

import (
	"errors"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"machining-erp/internal/models"
)

type CustomerRepository struct{ db *gorm.DB }

func NewCustomerRepository(db *gorm.DB) *CustomerRepository { return &CustomerRepository{db: db} }
func (r *CustomerRepository) GetByCorpID(corpID int64) ([]models.Customer, error) {
	customers := []models.Customer{}
	err := r.db.Preload("Contacts", "corp_id = ?", corpID).Where("corp_id = ?", corpID).Order("name").Find(&customers).Error
	return customers, err
}
func (r *CustomerRepository) Create(corpID int64, name, shortName string, contacts []models.Contact) (int64, error) {
	for i := range contacts {
		contacts[i].ID = 0
		contacts[i].CustomerID = 0
		contacts[i].CorpID = corpID
	}
	customer := models.Customer{CorpID: corpID, Name: name, ShortName: shortName, Contacts: contacts}
	err := r.db.Create(&customer).Error
	return customer.ID, err
}
func (r *CustomerRepository) Update(corpID, id int64, name, shortName string, contacts []models.Contact) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		var customer models.Customer
		if e := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id=? AND corp_id=?", id, corpID).First(&customer).Error; e != nil {
			return e
		}
		if e := tx.Model(&customer).Where("corp_id=?", corpID).Updates(map[string]interface{}{"name": name, "short_name": shortName}).Error; e != nil {
			return e
		}
		var existing []models.Contact
		if e := tx.Where("customer_id=? AND corp_id=?", id, corpID).Find(&existing).Error; e != nil {
			return e
		}
		owned := map[int64]bool{}
		for _, ct := range existing {
			owned[ct.ID] = true
		}
		seen := map[int64]bool{}
		kept := []int64{}
		for _, ct := range contacts {
			ct.CorpID = corpID
			ct.CustomerID = id
			if ct.ID > 0 {
				if !owned[ct.ID] {
					return gorm.ErrRecordNotFound
				}
				if seen[ct.ID] {
					return errors.New("联系人重复")
				}
				seen[ct.ID] = true
				kept = append(kept, ct.ID)
				if e := tx.Model(&models.Contact{}).Where("id=? AND customer_id=? AND corp_id=?", ct.ID, id, corpID).Updates(map[string]interface{}{"name": ct.Name, "contact": ct.Contact}).Error; e != nil {
					return e
				}
			} else {
				if e := tx.Create(&ct).Error; e != nil {
					return e
				}
				kept = append(kept, ct.ID)
			}
		}
		removal := tx.Where("customer_id=? AND corp_id=?", id, corpID)
		if len(kept) > 0 {
			removal = removal.Where("id NOT IN ?", kept)
		}
		return removal.Delete(&models.Contact{}).Error
	})
}
func (r *CustomerRepository) Delete(corpID, id int64) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		var customer models.Customer
		if e := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id=? AND corp_id=?", id, corpID).First(&customer).Error; e != nil {
			return e
		}
		if e := tx.Where("customer_id=? AND corp_id=?", id, corpID).Delete(&models.Contact{}).Error; e != nil {
			return e
		}
		return tx.Where("id=? AND corp_id=?", id, corpID).Delete(&models.Customer{}).Error
	})
}
func (r *CustomerRepository) NameExists(corpID int64, name string, excludeID int64) (bool, error) {
	var n int64
	e := r.db.Model(&models.Customer{}).Where("corp_id=? AND name=? AND id<>?", corpID, name, excludeID).Count(&n).Error
	return n > 0, e
}
