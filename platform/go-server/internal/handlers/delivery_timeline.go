package handlers

import (
	"gorm.io/gorm"
	"time"
)

func deliveryTimelineFits(tx *gorm.DB, corp, item int64, date time.Time, quantity int) (bool, error) {
	var qualified []struct {
		BusinessDate time.Time
		Quantity     float64
	}
	var deliveries []struct {
		BusinessDate *time.Time
		Quantity     int
	}
	if err := tx.Table("production_completions").Select("business_date,qualified_quantity AS quantity").Where("corp_id=? AND order_item_id=? AND status='confirmed'", corp, item).Find(&qualified).Error; err != nil {
		return false, err
	}
	if err := tx.Table("order_business_records").Select("business_date,quantity").Where("corp_id=? AND order_item_id=? AND kind='delivery' AND status='confirmed'", corp, item).Find(&deliveries).Error; err != nil {
		return false, err
	}
	dates := []time.Time{date}
	for _, row := range deliveries {
		if row.BusinessDate != nil {
			dates = append(dates, *row.BusinessDate)
		}
	}
	for _, at := range dates {
		q := float64(0)
		d := 0
		for _, row := range qualified {
			if !row.BusinessDate.After(at) {
				q += row.Quantity
			}
		}
		for _, row := range deliveries {
			if row.BusinessDate == nil || !row.BusinessDate.After(at) {
				d += row.Quantity
			}
		}
		if !date.After(at) {
			d += quantity
		}
		if float64(d) > q {
			return false, nil
		}
	}
	return true, nil
}
