// Package accounting owns ledger changes shared by order editing and finance workflows.
package accounting

import (
	"errors"
	"gorm.io/gorm"
)

// ResolvePending must run within the caller's transaction, after locking the order/item.
// Priced historical snapshots are immutable. Original completion dates are preserved.
func ResolvePending(tx *gorm.DB, corp, item, actor int64) error {
	return ResolvePendingItems(tx, corp, []int64{item}, actor)
}

// One lookup per order, rather than an extra round trip for every part on each save.
func ResolvePendingItems(tx *gorm.DB, corp int64, items []int64, actor int64) error {
	if len(items) == 0 {
		return nil
	}
	var pending []struct {
		ID    int64
		Month string
	}
	if err := tx.Raw("SELECT p.id,DATE_FORMAT(p.business_date,'%Y-%m') AS month FROM production_completions p JOIN order_items i ON i.corp_id=p.corp_id AND i.id=p.order_item_id WHERE p.corp_id=? AND p.order_item_id IN ? AND p.status='confirmed' AND p.pricing_status='pending' AND i.unit_price>0 AND i.currency='CNY' ORDER BY p.business_date,p.id FOR UPDATE", corp, items).Scan(&pending).Error; err != nil {
		return err
	}
	if len(pending) == 0 {
		return nil
	}
	months := map[string]bool{}
	ids := make([]int64, 0, len(pending))
	for _, row := range pending {
		if !months[row.Month] {
			if err := LockMonth(tx, corp, row.Month); err != nil {
				return err
			}
			months[row.Month] = true
		}
		ids = append(ids, row.ID)
	}
	if err := tx.Exec("INSERT INTO completion_price_changes(corp_id,production_completion_id,unit_price,created_by,created_at) SELECT p.corp_id,p.id,i.unit_price,?,NOW(3) FROM production_completions p JOIN order_items i ON i.id=p.order_item_id AND i.corp_id=p.corp_id WHERE p.corp_id=? AND p.id IN ? AND p.pricing_status='pending'", actor, corp, ids).Error; err != nil {
		return err
	}
	return tx.Exec("UPDATE production_completions p JOIN order_items i ON i.id=p.order_item_id AND i.corp_id=p.corp_id SET p.unit_price_snapshot=i.unit_price,p.output_value=ROUND(p.billable_quantity*i.unit_price,2),p.pricing_status='priced' WHERE p.corp_id=? AND p.id IN ? AND p.pricing_status='pending'", corp, ids).Error
}

func LockMonth(tx *gorm.DB, corp int64, month string) error {
	status, err := LockMonthStatus(tx, corp, month)
	if err != nil {
		return err
	}
	if status != "open" {
		return errors.New("相关月份已核对锁定，请先由管理员核对后处理，不能直接补价或记账")
	}
	return nil
}
