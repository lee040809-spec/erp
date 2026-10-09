// Package productionflow owns quantity conservation and delivery plans; it has no HTTP dependency.
package productionflow

import (
	"errors"
	"fmt"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"machining-erp/internal/models"
	"machining-erp/pkg/utils"
	"regexp"
	"time"
	"unicode/utf8"
)

var Zone = time.FixedZone("Asia/Shanghai", 8*3600)

func Today() string { return time.Now().In(Zone).Format("2006-01-02") }
func Date(value string, actual bool) (time.Time, error) {
	d, err := time.Parse("2006-01-02", value)
	if err != nil || actual && value > Today() {
		return d, errors.New("日期须为有效日期，实际发生日期不能晚于今天")
	}
	return d, nil
}
func RequestID(id string) bool        { return regexp.MustCompile(`^[a-zA-Z0-9-]{1,36}$`).MatchString(id) }
func Text(value string, max int) bool { return utf8.RuneCountInString(value) <= max }
func LockItem(tx *gorm.DB, corp, itemID int64) (models.OrderItem, error) {
	var item models.OrderItem
	if err := tx.Where("id=? AND corp_id=?", itemID, corp).Take(&item).Error; err != nil {
		return item, errors.New("零件不存在或不属于当前公司")
	}
	if err := tx.Table("orders").Clauses(clause.Locking{Strength: "UPDATE"}).Where("id=? AND corp_id=?", item.OrderID, corp).Take(&models.Order{}).Error; err != nil {
		return item, err
	}
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id=? AND corp_id=?", itemID, corp).Take(&item).Error; err != nil {
		return item, err
	}
	if item.ProductionQuantity == 0 {
		item.ProductionQuantity = item.Quantity
	}
	return item, nil
}
func Enabled(tx *gorm.DB, corp, item int64) (bool, error) {
	var n int64
	err := tx.Model(&models.ProcessQuantity{}).Where("corp_id=? AND order_item_id=?", corp, item).Count(&n).Error
	return n > 0, err
}
func RefreshStatus(tx *gorm.DB, item models.OrderItem) error {
	var qualified int
	if err := tx.Table("production_completions").Where("corp_id=? AND order_item_id=? AND status='confirmed'", item.CorpID, item.ID).Select("CAST(COALESCE(SUM(qualified_quantity),0) AS SIGNED)").Scan(&qualified).Error; err != nil {
		return err
	}
	status := "processing"
	if qualified >= item.Quantity {
		status = "completed"
	}
	if item.DeliveredQty != nil && *item.DeliveredQty >= item.Quantity {
		status = "delivered"
	}
	if err := tx.Table("order_items").Where("id=? AND corp_id=?", item.ID, item.CorpID).Update("status", status).Error; err != nil {
		return err
	}
	var statuses []string
	if err := tx.Table("order_items").Where("order_id=? AND corp_id=?", item.OrderID, item.CorpID).Pluck("status", &statuses).Error; err != nil {
		return err
	}
	return tx.Table("orders").Where("id=? AND corp_id=?", item.OrderID, item.CorpID).Update("status", utils.CalculateStatus(statuses)).Error
}
func route(tx *gorm.DB, corp, item int64) ([]models.OrderProcess, error) {
	var ps []models.OrderProcess
	err := tx.Where("corp_id=? AND order_item_id=? AND name<>'送货'", corp, item).Order("sort_order,id").Find(&ps).Error
	return ps, err
}
func SyncProcesses(tx *gorm.DB, item models.OrderItem) error {
	var totals []models.ProcessQuantity
	if err := tx.Where("corp_id=? AND order_item_id=?", item.CorpID, item.ID).Find(&totals).Error; err != nil {
		return err
	}
	for _, t := range totals {
		status := "pending"
		if t.StartedQuantity > 0 {
			status = "processing"
		}
		if t.StartedQuantity > 0 && t.CompletedQuantity >= item.ProductionQuantity-item.ScrapQuantity {
			status = "completed"
		}
		if err := tx.Table("order_processes").Where("id=? AND corp_id=?", t.OrderProcessID, item.CorpID).Update("status", status).Error; err != nil {
			return err
		}
	}
	return RefreshStatus(tx, item)
}
func BatchScope(tx *gorm.DB, corp, item int64, batch *int64) (*models.DeliveryBatch, error) {
	var count int64
	if err := tx.Model(&models.DeliveryBatch{}).Where("corp_id=? AND order_item_id=?", corp, item).Count(&count).Error; err != nil {
		return nil, err
	}
	if batch == nil {
		if count > 0 {
			return nil, errors.New("该零件已设置分批交货，请选择具体交货批次")
		}
		return nil, nil
	}
	var b models.DeliveryBatch
	if err := tx.Where("id=? AND corp_id=? AND order_item_id=?", *batch, corp, item).Take(&b).Error; err != nil {
		return nil, errors.New("交货批次不属于当前零件")
	}
	return &b, nil
}
func ProtectItem(tx *gorm.DB, corp, item int64) error {
	for _, table := range []string{"order_delivery_batches", "process_quantity_totals", "outsourcing_dispatches", "outsourcing_orders"} {
		var n int64
		if err := tx.Table(table).Where("corp_id=? AND order_item_id=?", corp, item).Count(&n).Error; err != nil {
			return err
		}
		if n > 0 {
			return fmt.Errorf("零件已有分批交货、工序数量或外协台账，不能直接删除")
		}
	}
	return nil
}
