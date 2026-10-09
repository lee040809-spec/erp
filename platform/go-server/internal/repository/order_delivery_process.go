package repository

import (
	"gorm.io/gorm"
	"machining-erp/internal/models"
)

// Delivery states follow recorded quantities; generic order editing cannot change them.
func protectDeliveryProcesses(tx *gorm.DB, item *models.OrderItem) error {
	status := models.ProcessStatusPending
	if item.ID != 0 {
		var stored models.OrderItem
		if err := tx.Select("id,quantity,delivered_quantity").Where("id=? AND corp_id=?", item.ID, item.CorpID).Take(&stored).Error; err != nil {
			return err
		}
		if stored.DeliveredQty != nil {
			if *stored.DeliveredQty >= stored.Quantity {
				status = models.ProcessStatusCompleted
			}
		} else {
			// Preserve legacy states where an actual quantity was never captured.
			var old []models.OrderProcess
			if err := tx.Where("order_item_id=? AND corp_id=? AND name='送货'", item.ID, item.CorpID).Order("id").Find(&old).Error; err != nil {
				return err
			}
			if len(old) > 0 {
				status = old[0].Status
			}
		}
	}
	for i := range item.Processes {
		if item.Processes[i].Name == "送货" {
			item.Processes[i].Status = status
		}
	}
	return nil
}
