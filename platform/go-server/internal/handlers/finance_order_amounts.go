package handlers

import (
	"gorm.io/gorm"
	"machining-erp/internal/models"
)

// Order-entry amounts are a reference, separate from posted costs and profit.
type orderMonthAmounts struct {
	Month       string
	Amount      string
	Material    string
	Outsourcing string
	Tool        string
	Fixture     string
	Other       string
}

func loadOrderMonthAmounts(db *gorm.DB, corp int64, start, end string) ([]orderMonthAmounts, error) {
	rows := []orderMonthAmounts{}
	err := db.Raw(`SELECT DATE_FORMAT(COALESCE(i.start_date,o.start_date),'%Y-%m') AS month,
 CAST(SUM(ROUND(i.quantity*COALESCE(i.unit_price,0),2)) AS CHAR) AS amount,
 CAST(SUM(COALESCE(i.material_cost,0)) AS CHAR) AS material,
 CAST(SUM(COALESCE(p.amount,0)) AS CHAR) AS outsourcing,
 CAST(SUM(COALESCE(i.tool_cost,0)) AS CHAR) AS tool,
 CAST(SUM(COALESCE(i.fixture_cost,0)) AS CHAR) AS fixture,
 CAST(SUM(COALESCE(i.other_cost,0)) AS CHAR) AS other
 FROM order_items i JOIN orders o ON o.id=i.order_id AND o.corp_id=i.corp_id
 LEFT JOIN (SELECT order_item_id,corp_id,SUM(outsourcing_fee) AS amount FROM order_processes WHERE corp_id=? AND deleted_at IS NULL AND is_outsourced=1 AND name<>'送货' GROUP BY order_item_id,corp_id) p ON p.order_item_id=i.id AND p.corp_id=i.corp_id
 WHERE i.corp_id=? AND COALESCE(i.start_date,o.start_date)>=? AND COALESCE(i.start_date,o.start_date)<? GROUP BY month`, corp, corp, start, end).Scan(&rows).Error
	return rows, err
}

func enteredOrderCosts(row orderMonthAmounts) models.OrderCostTotals {
	return models.OrderCostTotals{Material: row.Material, Outsourcing: row.Outsourcing, Tool: row.Tool, Fixture: row.Fixture, Other: row.Other}
}

// Only added to an authorized, bounded workbench page query. The
// process fee total is calculated from the existing paged process read.
const workbenchOrderCostColumns = `,
 CAST(COALESCE(i.material_cost,0) AS CHAR) AS entered_material,
 CAST(COALESCE(i.tool_cost,0) AS CHAR) AS entered_tool,
 CAST(COALESCE(i.fixture_cost,0) AS CHAR) AS entered_fixture,
 CAST(COALESCE(i.other_cost,0) AS CHAR) AS entered_other`
