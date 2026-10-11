package productionflow

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"gorm.io/gorm"
	"machining-erp/internal/models"
	"time"
)

type ReportInput struct {
	Kind      string `json:"kind"`
	Quantity  int    `json:"quantity"`
	Date      string `json:"business_date"`
	Notes     string `json:"notes"`
	RequestID string `json:"request_id"`
}

func Report(db *gorm.DB, corp, itemID, processID, actor int64, in ReportInput) error {
	if !RequestID(in.RequestID) || !Text(in.Notes, 500) {
		return errors.New("请求编号或备注无效")
	}
	return db.Transaction(func(tx *gorm.DB) error {
		item, err := LockItem(tx, corp, itemID)
		if err != nil {
			return err
		}
		return ReportTx(tx, item, processID, actor, in, "manual")
	})
}

// ReportTx requires the caller to hold the parent order and item locks, in that order.
func ReportTx(tx *gorm.DB, item models.OrderItem, processID, actor int64, in ReportInput, source string) error {
	if in.Quantity <= 0 || in.Quantity > 2147483647 || (in.Kind != "start" && in.Kind != "complete" && in.Kind != "scrap") {
		return errors.New("工序数量须为正整数，操作只能是开工、完工或报废")
	}
	date, err := Date(in.Date, true)
	if err != nil {
		return err
	}
	if item.StartDate != nil && date.Before(*item.StartDate) {
		return errors.New("实际工序日期不能早于下单日期")
	}
	var existing models.ProcessQuantityEvent
	err = tx.Where("corp_id=? AND request_id=?", item.CorpID, in.RequestID).Take(&existing).Error
	if err == nil {
		if existing.OrderItemID != item.ID || existing.OrderProcessID != processID || existing.Kind != in.Kind || existing.Quantity != in.Quantity || existing.BusinessDate == nil || existing.BusinessDate.Format("2006-01-02") != in.Date || existing.Notes != in.Notes || existing.SourceType != source {
			return errors.New("请求编号已使用，内容不一致")
		}
		return nil
	}
	if err != gorm.ErrRecordNotFound {
		return err
	}
	ps, err := route(tx, item.CorpID, item.ID)
	if err != nil {
		return err
	}
	var p *models.OrderProcess
	for i := range ps {
		if ps[i].ID == processID {
			p = &ps[i]
		}
	}
	if p == nil {
		return errors.New("工序不属于当前零件")
	}
	if p.IsOutsourced && source == "manual" {
		return errors.New("外协工序请通过外协发出和回厂质检登记，不能直接报工")
	}
	var total models.ProcessQuantity
	if err = tx.Where("corp_id=? AND order_process_id=? AND order_item_id=?", item.CorpID, processID, item.ID).Take(&total).Error; err != nil {
		return errors.New("请先启用工序数量跟踪")
	}
	if in.Kind == "scrap" {
		var qualified int
		if err = tx.Table("production_completions").Where("corp_id=? AND order_item_id=? AND status='confirmed'", item.CorpID, item.ID).Select("CAST(COALESCE(SUM(qualified_quantity),0) AS SIGNED)").Scan(&qualified).Error; err != nil {
			return err
		}
		if item.ScrapQuantity+in.Quantity+qualified > item.ProductionQuantity {
			return errors.New("报废数量超过未确认合格的投产余量")
		}
	}
	if in.Kind == "start" {
		if err = GuardReservedCapacity(tx, item, processID, in.Quantity); err != nil {
			return err
		}
	}
	event := models.ProcessQuantityEvent{CorpID: item.CorpID, OrderItemID: item.ID, OrderProcessID: processID, Kind: in.Kind, Quantity: in.Quantity, BusinessDate: &date, Notes: in.Notes, SourceType: source, RequestID: in.RequestID, CreatedBy: actor, CreatedAt: time.Now()}
	if err = tx.Create(&event).Error; err != nil {
		return err
	}
	field := map[string]string{"start": "started_quantity", "complete": "completed_quantity", "scrap": "scrap_quantity"}[in.Kind]
	if err = tx.Model(&total).Update(field, gorm.Expr(field+" + ?", in.Quantity)).Error; err != nil {
		return err
	}
	if err = validateChronology(tx, item, ps); err != nil {
		return err
	}
	if in.Kind == "scrap" {
		item.ScrapQuantity += in.Quantity
		if err = tx.Table("order_items").Where("id=? AND corp_id=?", item.ID, item.CorpID).Update("scrap_quantity", item.ScrapQuantity).Error; err != nil {
			return err
		}
		sum := sha256.Sum256([]byte(in.RequestID))
		requestID := fmt.Sprintf("%x", sum[:16])
		if err = tx.Table("order_business_records").Create(map[string]interface{}{"corp_id": item.CorpID, "order_id": item.OrderID, "order_item_id": item.ID, "kind": "scrap", "quantity": in.Quantity, "business_date": date, "process_name": p.Name, "description": in.Notes, "notes": in.Notes, "source_type": "process_quantity", "request_id": requestID, "status": "confirmed", "created_by": actor, "created_at": time.Now()}).Error; err != nil {
			return err
		}
	}
	changes := map[string]interface{}{}
	if in.Kind == "start" && (p.ActualStartAt == nil || in.Date < p.ActualStartAt.In(Zone).Format("2006-01-02")) {
		changes["actual_start_at"] = time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, Zone)
	}
	if in.Kind == "complete" {
		var after models.ProcessQuantity
		if err = tx.First(&after, total.ID).Error; err != nil {
			return err
		}
		if after.CompletedQuantity >= item.ProductionQuantity-item.ScrapQuantity {
			changes["actual_end_at"] = time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, Zone)
		}
	}
	if len(changes) > 0 {
		if err = tx.Model(p).Updates(changes).Error; err != nil {
			return err
		}
	}
	return SyncProcesses(tx, item)
}

// Check the whole dated ledger, so backdated reports cannot consume future upstream output.
func validateChronology(tx *gorm.DB, item models.OrderItem, ps []models.OrderProcess) error {
	type dayRow struct {
		Day            string
		OrderProcessID int64
		Kind           string
		Quantity       int
		SourceType     string
	}
	var rows []dayRow
	losses, err := archivedLosses(tx, item.CorpID, item.ID)
	if err != nil {
		return err
	}
	anchors := map[int64]int64{}
	for _, l := range losses {
		anchors[l.ProcessID] = l.BeforeID
	}
	archivedScrap := map[int64]int{}
	if err := tx.Table("process_quantity_events").Select("COALESCE(DATE_FORMAT(business_date,'%Y-%m-%d'),'') AS day,order_process_id,kind,SUM(quantity) AS quantity,source_type").Where("corp_id=? AND order_item_id=?", item.CorpID, item.ID).Group("business_date,order_process_id,kind,source_type").Order("business_date,order_process_id,kind").Find(&rows).Error; err != nil {
		return err
	}
	totals := map[int64]*models.ProcessQuantity{}
	for _, p := range ps {
		totals[p.ID] = &models.ProcessQuantity{}
	}
	check := func() error {
		available := item.ProductionQuantity
		for _, p := range ps {
			available -= archivedScrap[p.ID]
			t := totals[p.ID]
			if t.StartedQuantity > available || t.CompletedQuantity+t.ScrapQuantity > t.StartedQuantity {
				return fmt.Errorf("%s数量/日期不守恒：后道开工不得超过截至当天前道已完工数量，完工加报废不得超过已开工数量", p.Name)
			}
			available = t.CompletedQuantity
		}
		return nil
	}
	for i, r := range rows {
		t := totals[r.OrderProcessID]
		if t == nil {
			if anchor, retired := anchors[r.OrderProcessID]; retired {
				if r.Kind == "scrap" {
					archivedScrap[anchor] += r.Quantity
				}
				if i == len(rows)-1 || rows[i+1].Day != r.Day {
					if err := check(); err != nil {
						return err
					}
				}
				continue
			}
			return errors.New("工序数量台账与当前流程不一致")
		}
		switch r.Kind {
		case "start":
			t.StartedQuantity += r.Quantity
		case "complete":
			t.CompletedQuantity += r.Quantity
		case "scrap":
			t.ScrapQuantity += r.Quantity
		}
		if i == len(rows)-1 || rows[i+1].Day != r.Day {
			if err := check(); err != nil {
				return err
			}
		}
	}
	var baselineCount int64
	if err := tx.Model(&models.ProcessQuantityEvent{}).Where("corp_id=? AND order_item_id=? AND source_type='baseline'", item.CorpID, item.ID).Count(&baselineCount).Error; err != nil {
		return err
	}
	if baselineCount > 0 {
		var activated time.Time
		if err := tx.Model(&models.ProcessQuantity{}).Where("corp_id=? AND order_item_id=?", item.CorpID, item.ID).Select("MIN(activated_at)").Scan(&activated).Error; err != nil {
			return err
		}
		for _, r := range rows {
			if r.Day != "" && r.Day < activated.In(Zone).Format("2006-01-02") {
				return errors.New("历史基线日期未知，新报工不能早于数量跟踪启用日期")
			}
		}
	}
	return nil
}

func CompletionReady(tx *gorm.DB, corp, item int64, quantity int, batch *int64, date string) error {
	on, err := Enabled(tx, corp, item)
	if err != nil {
		return err
	}
	if on {
		ps, err := route(tx, corp, item)
		if err != nil {
			return err
		}
		if len(ps) == 0 {
			return errors.New("缺少生产工序")
		}
		var last models.ProcessQuantity
		if err = tx.Where("corp_id=? AND order_process_id=?", corp, ps[len(ps)-1].ID).Take(&last).Error; err != nil {
			return err
		}
		var confirmed int
		if err = tx.Table("production_completions").Where("corp_id=? AND order_item_id=? AND status='confirmed'", corp, item).Select("CAST(COALESCE(SUM(qualified_quantity),0) AS SIGNED)").Scan(&confirmed).Error; err != nil {
			return err
		}
		losses, e := archivedLosses(tx, corp, item)
		if e != nil {
			return e
		}
		if confirmed+quantity > last.CompletedQuantity-terminalLoss(losses) {
			return errors.New("本次合格数量超过末道工序已完工、尚未确认的数量；请先报末道完工或外协回厂质检")
		}
	}
	if on {
		ps, e := route(tx, corp, item)
		if e != nil {
			return e
		}
		var available, confirmed int
		if e = tx.Model(&models.ProcessQuantityEvent{}).Where("corp_id=? AND order_process_id=? AND kind='complete' AND (business_date<=? OR business_date IS NULL)", corp, ps[len(ps)-1].ID, date).Select("COALESCE(SUM(quantity),0)").Scan(&available).Error; e != nil {
			return e
		}
		if e = tx.Table("production_completions").Where("corp_id=? AND order_item_id=? AND status='confirmed'", corp, item).Select("CAST(COALESCE(SUM(qualified_quantity),0) AS SIGNED)").Scan(&confirmed).Error; e != nil {
			return e
		}
		losses, lossErr := archivedLosses(tx, corp, item)
		if lossErr != nil {
			return lossErr
		}
		tailIDs := []int64{}
		for _, l := range losses {
			if l.BeforeID == 0 {
				tailIDs = append(tailIDs, l.ProcessID)
			}
		}
		var retiredScrap int
		if len(tailIDs) > 0 {
			if e = tx.Model(&models.ProcessQuantityEvent{}).Where("corp_id=? AND order_item_id=? AND order_process_id IN ? AND kind='scrap' AND (business_date<=? OR business_date IS NULL)", corp, item, tailIDs, date).Select("COALESCE(SUM(quantity),0)").Scan(&retiredScrap).Error; e != nil {
				return e
			}
		}
		available -= retiredScrap
		if confirmed+quantity > available {
			return errors.New("合格完工日期早于末道工序数量完工日期")
		}
		var n int64
		if e = tx.Model(&models.ProcessQuantityEvent{}).Where("corp_id=? AND order_item_id=? AND source_type='baseline'", corp, item).Count(&n).Error; e != nil {
			return e
		}
		if n > 0 {
			var activated time.Time
			if e = tx.Model(&models.ProcessQuantity{}).Where("corp_id=? AND order_item_id=?", corp, item).Select("MIN(activated_at)").Scan(&activated).Error; e != nil {
				return e
			}
			if date < activated.In(Zone).Format("2006-01-02") {
				return errors.New("新完工确认不能早于历史数量基线启用日期")
			}
		}
	}
	b, err := BatchScope(tx, corp, item, batch)
	if err != nil {
		return err
	}
	if b != nil {
		var q int
		if err = tx.Table("production_completions").Where("corp_id=? AND delivery_batch_id=? AND status='confirmed'", corp, b.ID).Select("CAST(COALESCE(SUM(qualified_quantity),0) AS SIGNED)").Scan(&q).Error; err != nil {
			return err
		}
		if q+quantity > b.ProductionQuantity {
			return errors.New("合格数量超过本交货批次的投产数量")
		}
	}
	return nil
}
