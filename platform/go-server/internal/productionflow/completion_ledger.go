package productionflow

import (
	"errors"
	"gorm.io/gorm"
	"machining-erp/internal/models"
	"time"
)

// Recheck confirmed output against the dated terminal-process ledger after a correction.
func ValidateCompletionLedger(tx *gorm.DB, item models.OrderItem) error {
	on, err := Enabled(tx, item.CorpID, item.ID)
	if err != nil || !on {
		return err
	}
	ps, err := route(tx, item.CorpID, item.ID)
	if err != nil {
		return err
	}
	if len(ps) == 0 {
		return errors.New("缺少末道生产工序，不能修改合格记录")
	}
	losses, err := archivedLosses(tx, item.CorpID, item.ID)
	if err != nil {
		return err
	}
	tailIDs := []int64{}
	for _, l := range losses {
		if l.BeforeID == 0 {
			tailIDs = append(tailIDs, l.ProcessID)
		}
	}
	type datedQuantity struct {
		Day      string
		Quantity float64
	}
	completions := []datedQuantity{}
	if err = tx.Table("production_completions").Select("DATE_FORMAT(business_date,'%Y-%m-%d') AS day,SUM(qualified_quantity) AS quantity").Where("corp_id=? AND order_item_id=? AND status='confirmed'", item.CorpID, item.ID).Group("business_date").Order("business_date").Scan(&completions).Error; err != nil {
		return err
	}
	var baseline int64
	if err = tx.Model(&models.ProcessQuantityEvent{}).Where("corp_id=? AND order_item_id=? AND source_type='baseline'", item.CorpID, item.ID).Count(&baseline).Error; err != nil {
		return err
	}
	var activated time.Time
	if baseline > 0 {
		if err = tx.Model(&models.ProcessQuantity{}).Where("corp_id=? AND order_item_id=?", item.CorpID, item.ID).Select("MIN(activated_at)").Scan(&activated).Error; err != nil {
			return err
		}
	}
	total := float64(0)
	for _, r := range completions {
		if baseline > 0 && r.Day < activated.In(Zone).Format("2006-01-02") {
			return errors.New("合格日期不能早于历史工序数量基线启用日期")
		}
		total += r.Quantity
		var available float64
		if err = tx.Model(&models.ProcessQuantityEvent{}).Where("corp_id=? AND order_process_id=? AND kind='complete' AND (business_date<=? OR business_date IS NULL)", item.CorpID, ps[len(ps)-1].ID, r.Day).Select("COALESCE(SUM(quantity),0)").Scan(&available).Error; err != nil {
			return err
		}
		if len(tailIDs) > 0 {
			var scrap float64
			if err = tx.Model(&models.ProcessQuantityEvent{}).Where("corp_id=? AND order_item_id=? AND order_process_id IN ? AND kind='scrap' AND (business_date<=? OR business_date IS NULL)", item.CorpID, item.ID, tailIDs, r.Day).Select("COALESCE(SUM(quantity),0)").Scan(&scrap).Error; err != nil {
				return err
			}
			available -= scrap
		}
		if total > available {
			return errors.New("修改后合格数量或日期超过截至当天末道已完工数量，请先核对工序报工")
		}
	}
	return nil
}
