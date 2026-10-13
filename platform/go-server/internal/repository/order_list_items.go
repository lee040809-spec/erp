package repository

import (
	"gorm.io/gorm"
	"machining-erp/internal/drawings"
	"machining-erp/internal/models"
	"strings"
	"time"
)

func loadOrderListItems(db *gorm.DB, corp int64, orders []int64) ([]models.OrderItem, error) {
	if len(orders) == 0 {
		return nil, nil
	}
	statement := &gorm.Statement{DB: db}
	if err := statement.Parse(&models.OrderItem{}); err != nil {
		return nil, err
	}
	columns := make([]string, 0, len(statement.Schema.DBNames)+4)
	for _, name := range statement.Schema.DBNames {
		if name != "drawing_data" {
			columns = append(columns, "i.`"+name+"`")
		}
	}
	columns = append(columns, drawings.CountSQL("i")+" AS list_drawing_count", "COALESCE(q.qualified,0) AS list_qualified", "COALESCE(q.billable,0) AS list_billable", "d.last_date AS list_delivery_date")
	var rows []struct {
		models.OrderItem
		ListDrawingCount int
		ListQualified    float64
		ListBillable     float64
		ListDeliveryDate *time.Time
	}
	err := db.Table("order_items AS i").Select(strings.Join(columns, ",")).
		Joins("LEFT JOIN (SELECT order_item_id,SUM(qualified_quantity) AS qualified,SUM(billable_quantity) AS billable FROM production_completions WHERE corp_id=? AND status='confirmed' GROUP BY order_item_id) q ON q.order_item_id=i.id", corp).
		Joins("LEFT JOIN (SELECT order_item_id,MAX(business_date) AS last_date FROM order_business_records WHERE corp_id=? AND kind='delivery' AND status='confirmed' GROUP BY order_item_id) d ON d.order_item_id=i.id", corp).
		Where("i.order_id IN ? AND i.corp_id=?", orders, corp).Order("i.id DESC").Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	items := make([]models.OrderItem, 0, len(rows))
	for _, row := range rows {
		item := row.OrderItem
		item.DrawingCount = row.ListDrawingCount
		item.HasDrawing = row.ListDrawingCount > 0
		item.QualifiedQuantity = row.ListQualified
		item.BillableQuantity = row.ListBillable
		item.ReserveQuantity = row.ListQualified - row.ListBillable
		item.DeliveryGap = max(0, float64(item.Quantity)-row.ListQualified)
		item.DeliveryDate = row.ListDeliveryDate
		if item.ProductionQuantity == 0 {
			item.ProductionQuantity = item.Quantity
		}
		items = append(items, item)
	}
	return items, nil
}
