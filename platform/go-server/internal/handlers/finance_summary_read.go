package handlers

import (
	"gorm.io/gorm"
	"machining-erp/pkg/database"
	"strconv"
)

type financeProduced struct {
	Month, Quantity, Billable, Reserve, Output string
	Count                                      int64
}
type financeSpent struct{ Month, Material, Outsourcing, Other, Total string }
type financePeriod struct{ Month, Amount string }
type financeClosing struct{ AccountingMonth, Status string }
type financePending struct {
	Amount string
	Count  int64
}
type financeSummaryReads struct {
	Outputs  []financeProduced
	Costs    []financeSpent
	Periods  []financePeriod
	Orders   []orderMonthAmounts
	Closings []financeClosing
	Pending  financePending
}

// All reads are independent and share the existing bounded (three-reader) helper.
// Transaction/temporary-table connections remain serial in ReadParallel.
func readFinanceSummary(db *gorm.DB, corp int64, year string) (financeSummaryReads, error) {
	var s financeSummaryReads
	start, end := year+"-01-01", strconv.Itoa(mustYear(year)+1)+"-01-01"
	err := database.ReadParallel(db,
		func(db *gorm.DB) error {
			var err error
			s.Orders, err = loadOrderMonthAmounts(db, corp, start, end)
			return err
		},
		func(db *gorm.DB) error {
			return db.Raw(`SELECT DATE_FORMAT(business_date,'%Y-%m') AS month,CAST(SUM(qualified_quantity) AS CHAR) AS quantity,CAST(SUM(billable_quantity) AS CHAR) AS billable,CAST(SUM(qualified_quantity-billable_quantity) AS CHAR) AS reserve,COUNT(*) AS count,CAST(SUM(output_value) AS CHAR) AS output FROM production_completions WHERE corp_id=? AND status='confirmed' AND business_date>=? AND business_date<? GROUP BY month`, corp, start, end).Scan(&s.Outputs).Error
		},
		func(db *gorm.DB) error {
			return db.Raw(`SELECT DATE_FORMAT(p.business_date,'%Y-%m') AS month,
 CAST(SUM(CASE WHEN e.cost_type='material' THEN a.allocated_amount ELSE 0 END) AS CHAR) AS material,
 CAST(SUM(CASE WHEN e.cost_type='outsourcing' THEN a.allocated_amount ELSE 0 END) AS CHAR) AS outsourcing,
 CAST(SUM(CASE WHEN e.cost_type NOT IN ('material','outsourcing') THEN a.allocated_amount ELSE 0 END) AS CHAR) AS other,CAST(SUM(a.allocated_amount) AS CHAR) AS total
 FROM cost_allocations a JOIN production_completions p ON p.id=a.production_completion_id AND p.corp_id=a.corp_id AND p.status='confirmed'
 JOIN cost_entries e ON e.id=a.cost_entry_id AND e.corp_id=a.corp_id AND e.status='confirmed' AND e.cost_type<>'period'
 WHERE a.corp_id=? AND p.business_date>=? AND p.business_date<? GROUP BY month`, corp, start, end).Scan(&s.Costs).Error
		},
		func(db *gorm.DB) error {
			return db.Raw("SELECT accounting_month AS month,CAST(SUM(net_amount) AS CHAR) AS amount FROM cost_entries WHERE corp_id=? AND status='confirmed' AND cost_type='period' AND accounting_month>=? AND accounting_month<? GROUP BY accounting_month", corp, year+"-01", strconv.Itoa(mustYear(year)+1)+"-01").Scan(&s.Periods).Error
		},
		func(db *gorm.DB) error {
			return db.Table("monthly_closings").Where("corp_id=? AND accounting_month>=? AND accounting_month<?", corp, year+"-01", strconv.Itoa(mustYear(year)+1)+"-01").Find(&s.Closings).Error
		},
		func(db *gorm.DB) error {
			return db.Raw(`SELECT CAST(COALESCE(SUM(e.net_amount-COALESCE(a.total,0)),0) AS CHAR) AS amount,COUNT(*) AS count FROM cost_entries e
 LEFT JOIN (SELECT corp_id,cost_entry_id,SUM(allocated_amount) AS total FROM cost_allocations WHERE corp_id=? GROUP BY corp_id,cost_entry_id) a ON a.corp_id=e.corp_id AND a.cost_entry_id=e.id
 WHERE e.corp_id=? AND e.status='confirmed' AND e.cost_type<>'period' AND e.net_amount>COALESCE(a.total,0)`, corp, corp).Scan(&s.Pending).Error
		})
	return s, err
}
