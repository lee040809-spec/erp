package handlers

import (
	"gorm.io/gorm"
	"time"
)

// Use chronological batches, so later spare production never changes contractual completion.
func orderFulfilledDate(db *gorm.DB, corp, item int64, ordered int) (*time.Time, error) {
	var rows []struct {
		BusinessDate      time.Time
		QualifiedQuantity float64
	}
	if err := db.Table("production_completions").Select("business_date,qualified_quantity").Where("corp_id=? AND order_item_id=? AND status='confirmed'", corp, item).Order("business_date,id").Find(&rows).Error; err != nil {
		return nil, err
	}
	total := float64(0)
	for _, row := range rows {
		total += row.QualifiedQuantity
		if total >= float64(ordered) {
			date := row.BusinessDate
			return &date, nil
		}
	}
	return nil, nil
}
