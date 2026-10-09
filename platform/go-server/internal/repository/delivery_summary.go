package repository

import "gorm.io/gorm"

// DeliverySummary uses recorded deliveries, never a process status as a shipping date.
type DeliverySummary struct {
	LastDate        string `json:"last_date"`
	Delivered       int    `json:"delivered_quantity"`
	DatedQuantity   int    `json:"dated_quantity"`
	UndatedQuantity int    `json:"undated_quantity"`
	Batches         int    `json:"batches"`
}

func LoadDeliverySummaries(db *gorm.DB, corp int64, itemIDs []int64) (map[int64]DeliverySummary, error) {
	result := map[int64]DeliverySummary{}
	if len(itemIDs) == 0 {
		return result, nil
	}
	var rows []struct {
		ItemID int64
		DeliverySummary
	}
	err := db.Raw(`SELECT i.id AS item_id,COALESCE(i.delivered_quantity,0) AS delivered,
 COALESCE(DATE_FORMAT(MAX(r.business_date),'%Y-%m-%d'),'') AS last_date,
 COALESCE(SUM(CASE WHEN r.business_date IS NOT NULL THEN r.quantity ELSE 0 END),0) AS dated_quantity,
 COUNT(r.id) AS batches
 FROM order_items i LEFT JOIN order_business_records r ON r.corp_id=i.corp_id AND r.order_item_id=i.id AND r.kind='delivery' AND r.status='confirmed'
 WHERE i.corp_id=? AND i.id IN ? GROUP BY i.id,i.delivered_quantity`, corp, itemIDs).Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	for _, row := range rows {
		row.UndatedQuantity = max(0, row.Delivered-row.DatedQuantity)
		result[row.ItemID] = row.DeliverySummary
	}
	return result, nil
}
