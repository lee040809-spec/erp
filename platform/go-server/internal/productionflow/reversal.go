package productionflow

import (
	"errors"
	"gorm.io/gorm"
	"machining-erp/internal/models"
	"time"
)

// Called only after the outsourcing service checked no physical return or financial posting exists.
// The original event is retained; a signed correction restores the dated ledger in the same transaction.
func ReverseOutsourcingStart(tx *gorm.DB, item models.OrderItem, process, actor int64, quantity int, request, reason string) error {
	var original models.ProcessQuantityEvent
	if e := tx.Where("corp_id=? AND order_item_id=? AND order_process_id=? AND request_id=? AND kind='start' AND source_type='outsourcing'", item.CorpID, item.ID, process, request).Take(&original).Error; e != nil {
		return errors.New("缺少原始外协开工记录，无法安全撤销")
	}
	if original.Quantity != quantity || original.Quantity <= 0 {
		return errors.New("原始外协开工数量与发出记录不一致，无法撤销")
	}
	correction := original
	correction.ID = 0
	correction.Quantity = -original.Quantity
	correction.SourceType = "outsourcing_reversal"
	correction.RequestID = request + "-R"
	correction.Notes = reason
	correction.CreatedBy = actor
	correction.CreatedAt = time.Now()
	if e := tx.Create(&correction).Error; e != nil {
		return e
	}
	result := tx.Model(&models.ProcessQuantity{}).Where("corp_id=? AND order_item_id=? AND order_process_id=? AND started_quantity>=?", item.CorpID, item.ID, process, original.Quantity).Update("started_quantity", gorm.Expr("started_quantity-?", original.Quantity))
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected != 1 {
		return errors.New("工序开工累计异常，无法撤销")
	}
	ps, e := route(tx, item.CorpID, item.ID)
	if e != nil {
		return e
	}
	if e = validateChronology(tx, item, ps); e != nil {
		return errors.New("撤销会影响已有工序完工或后续进度，请先核对其他生产记录")
	}
	var earliest struct{ Day *time.Time }
	if e = tx.Raw(`SELECT MIN(business_date) AS day FROM (SELECT business_date FROM process_quantity_events WHERE corp_id=? AND order_process_id=? AND kind='start' AND business_date IS NOT NULL GROUP BY business_date HAVING SUM(quantity)>0) active_dates`, item.CorpID, process).Scan(&earliest).Error; e != nil {
		return e
	}
	if e = tx.Model(&models.OrderProcess{}).Where("corp_id=? AND id=?", item.CorpID, process).Update("actual_start_at", earliest.Day).Error; e != nil {
		return e
	}
	return SyncProcesses(tx, item)
}
