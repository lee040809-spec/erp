package productionflow

import (
	"errors"
	"gorm.io/gorm"
	"machining-erp/internal/models"
)

// Downstream full-route quantities stay reserved until their shared ledger actually starts.
func OutsourcingReserved(tx *gorm.DB, corp, item, process int64) (int, error) {
	var n int
	e := tx.Table("outsourcing_order_lines l").Joins("JOIN outsourcing_orders o ON o.id=l.outsourcing_order_id AND o.corp_id=l.corp_id").Where("l.corp_id=? AND l.order_item_id=? AND l.order_process_id=? AND o.status NOT IN ('cancelled','deleted')", corp, item, process).Select("CAST(COALESCE(SUM(l.quantity-l.started_quantity),0) AS SIGNED)").Scan(&n).Error
	return n, e
}
func GuardReservedCapacity(tx *gorm.DB, item models.OrderItem, process int64, adding int) error {
	reserved, e := OutsourcingReserved(tx, item.CorpID, item.ID, process)
	if e != nil {
		return e
	}
	var started int
	if e = tx.Model(&models.ProcessQuantity{}).Where("corp_id=? AND order_item_id=? AND order_process_id=?", item.CorpID, item.ID, process).Select("COALESCE(MAX(started_quantity),0)").Scan(&started).Error; e != nil {
		return e
	}
	if int64(started)+int64(reserved)+int64(adding) > int64(item.ProductionQuantity) {
		return errors.New("该工序已开工及外协订单预留数量超过投产数量，请减少本次数量或先取消未发出的外协订单")
	}
	return nil
}
