package repository

import (
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"machining-erp/internal/models"
	"machining-erp/internal/productionflow"
)

func validateProductionQuantities(ordered, production, scrap int, qualified, billable float64) error {
	if ordered <= 0 || production < ordered || production > 2147483647 || scrap < 0 || scrap > production || float64(production-scrap) < qualified || float64(ordered) < billable {
		return ErrPostedQuantity
	}
	return nil
}

// A production adjustment never changes the contracted quantity or amount.
func (r *OrderRepository) UpdateProductionQuantities(corpID, itemID int64, production, scrap int) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		var item models.OrderItem
		if err := tx.Where("id=? AND corp_id=?", itemID, corpID).Take(&item).Error; err != nil {
			return err
		}
		var order models.Order
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id=? AND corp_id=?", item.OrderID, corpID).Take(&order).Error; err != nil {
			return err
		}
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id=? AND corp_id=?", itemID, corpID).Take(&item).Error; err != nil {
			return err
		}
		var posted struct {
			Qualified float64
			Billable  float64
		}
		if err := tx.Table("production_completions").Where("corp_id=? AND order_item_id=? AND status='confirmed'", corpID, itemID).Select("COALESCE(SUM(qualified_quantity),0) AS qualified,COALESCE(SUM(billable_quantity),0) AS billable").Scan(&posted).Error; err != nil {
			return err
		}
		if e := productionflow.GuardQuantities(tx, corpID, itemID, item.Quantity, production); e != nil {
			return e
		}
		if err := validateProductionQuantities(item.Quantity, production, scrap, posted.Qualified, posted.Billable); err != nil {
			return err
		}
		return tx.Model(&item).Updates(map[string]interface{}{"production_quantity": production, "scrap_quantity": scrap}).Error
	})
}
