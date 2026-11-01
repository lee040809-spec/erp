package accounting

import (
	"errors"
	"fmt"
	driver "github.com/go-sql-driver/mysql"
	"gorm.io/gorm"
)

// PricingScope is restricted to either an order or a set of company-owned items.
type PricingScope struct {
	OrderID int64
	ItemIDs []int64
}

type PricingMonths map[string]string

var ErrPricingMonthsChanged = errors.New("待补价月份已变化，请重新核对")

// WithPricingMonths always locks months before the caller locks orders/items.
// Discovery is outside the transaction; ResolvePendingItems validates its result
// using current reads after business locks. A changed scope restarts the whole
// transaction, never acquiring another month lock while holding business locks.
// The callback must restore mutable input for each attempt and have no external
// side effects. Committed transactions are never retried.
func WithPricingMonths(db *gorm.DB, corp int64, scope PricingScope, apply func(*gorm.DB, PricingMonths) error) error {
	if scope.OrderID <= 0 && len(scope.ItemIDs) == 0 {
		return errors.New("缺少补价范围")
	}
	for attempt := 0; attempt < 3; attempt++ {
		var months []string
		q := db.Table("production_completions").Where("corp_id=? AND status='confirmed' AND pricing_status='pending'", corp)
		if scope.OrderID > 0 {
			q = q.Where("order_id=?", scope.OrderID)
		} else {
			q = q.Where("order_item_id IN ?", scope.ItemIDs)
		}
		if err := q.Distinct("DATE_FORMAT(business_date,'%Y-%m')").Order("DATE_FORMAT(business_date,'%Y-%m')").Pluck("DATE_FORMAT(business_date,'%Y-%m')", &months).Error; err != nil {
			return err
		}
		err := db.Transaction(func(tx *gorm.DB) error {
			locked := PricingMonths{}
			for _, month := range months {
				status, err := LockMonthStatus(tx, corp, month)
				if err != nil {
					return err
				}
				locked[month] = status
			}
			return apply(tx, locked)
		})
		var mysqlError *driver.MySQLError
		retry := errors.Is(err, ErrPricingMonthsChanged) || errors.As(err, &mysqlError) && (mysqlError.Number == 1213 || mysqlError.Number == 1205)
		if err == nil || !retry {
			return err
		}
		if db.Statement.Context.Err() != nil {
			return db.Statement.Context.Err()
		}
	}
	return fmt.Errorf("业务记录正在被其他操作更新，请刷新后重试")
}
