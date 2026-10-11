package productionflow

import (
	"errors"
	"gorm.io/gorm"
	"machining-erp/internal/models"
)

// GuardRoute checks each affected process rather than locking the whole part.
func GuardRoute(tx *gorm.DB, corp, item int64, incoming []models.OrderProcess, allow ...bool) error {
	archive := len(allow) > 0 && allow[0]
	return guardRouteChange(tx, corp, item, incoming, archive)
}
func GuardQuantities(tx *gorm.DB, corp, item int64, ordered, production int) error {
	var processIDs []int64
	if e := tx.Model(&models.OutsourcingOrderLine{}).Where("corp_id=? AND order_item_id=?", corp, item).Distinct("order_process_id").Pluck("order_process_id", &processIDs).Error; e != nil {
		return e
	}
	for _, id := range processIDs {
		if e := GuardReservedCapacity(tx, models.OrderItem{ID: item, CorpID: corp, ProductionQuantity: production}, id, 0); e != nil {
			return e
		}
	}
	var plans []models.DeliveryBatch
	if err := tx.Where("corp_id=? AND order_item_id=?", corp, item).Find(&plans).Error; err != nil {
		return err
	}
	if len(plans) > 0 {
		q, p := 0, 0
		for _, b := range plans {
			q += b.Quantity
			p += b.ProductionQuantity
		}
		if q != ordered || p != production {
			return errors.New("已有分批交货计划，订单/投产数量须与批次数量合计一致，请先调整批次计划")
		}
	}
	var started int
	if err := tx.Model(&models.ProcessQuantity{}).Where("corp_id=? AND order_item_id=?", corp, item).Select("COALESCE(MAX(started_quantity),0)").Scan(&started).Error; err != nil {
		return err
	}
	if production < started {
		return errors.New("投产数量不能低于工序已开工数量")
	}
	return nil
}

func GuardDates(tx *gorm.DB, item models.OrderItem) error {
	var plans []models.DeliveryBatch
	if e := tx.Where("corp_id=? AND order_item_id=?", item.CorpID, item.ID).Find(&plans).Error; e != nil {
		return e
	}
	for _, b := range plans {
		day := b.DueDate.Format("2006-01-02")
		if item.DueDate == nil || day > item.DueDate.Format("2006-01-02") || item.StartDate != nil && day < item.StartDate.Format("2006-01-02") {
			return errors.New("零件下单日期和总交期须包含已有交货批次的日期")
		}
	}
	return nil
}
