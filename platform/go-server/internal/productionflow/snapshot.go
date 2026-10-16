package productionflow

import (
	"errors"
	"gorm.io/gorm"
	"machining-erp/internal/models"
	"machining-erp/pkg/database"
)

func fmtDelivery() error {
	return errors.New("交货数量或日期超过本批已确认的合格数量/要求数量，不能提前占用未来完工")
}

type ProcessProgress struct {
	models.OrderProcess
	Tracking            bool `json:"tracking"`
	Started             int  `json:"started_quantity"`
	Completed           int  `json:"completed_quantity"`
	Scrap               int  `json:"scrap_quantity"`
	AvailableStart      int  `json:"available_start"`
	InProgress          int  `json:"in_progress"`
	OutsourcingReserved int  `json:"outsourcing_reserved"`
	AvailableOrder      int  `json:"available_order"`
}
type Snapshot struct {
	HistoricalProcessNames map[int64]string              `json:"historical_process_names"`
	Item                   models.OrderItem              `json:"item"`
	Tracking               bool                          `json:"tracking"`
	Processes              []ProcessProgress             `json:"processes"`
	Batches                []models.DeliveryBatch        `json:"batches"`
	Events                 []models.ProcessQuantityEvent `json:"events"`
	AvailableCompletion    int                           `json:"available_completion"`
}

func Read(db *gorm.DB, corp, itemID int64) (Snapshot, error) {
	s := Snapshot{Processes: []ProcessProgress{}, Batches: []models.DeliveryBatch{}, Events: []models.ProcessQuantityEvent{}}
	if err := db.Select("id,corp_id,order_id,part_name,part_number,quantity,production_quantity,scrap_quantity,delivered_quantity,start_date,due_date,status").Where("id=? AND corp_id=?", itemID, corp).Take(&s.Item).Error; err != nil {
		return s, err
	}
	if s.Item.ProductionQuantity == 0 {
		s.Item.ProductionQuantity = s.Item.Quantity
	}
	var ps []models.OrderProcess
	var totals []models.ProcessQuantity
	var reservations []struct {
		OrderProcessID int64
		Quantity       int
	}
	var losses []archivedLoss
	var confirmed int
	type quantity struct {
		LastDate        string
		DeliveryBatchID int64
		Quantity        int
	}
	var qs, ds []quantity
	err := database.ReadParallel(db,
		func(db *gorm.DB) error { var e error; ps, e = route(db, corp, itemID); return e },
		func(db *gorm.DB) error {
			return db.Where("corp_id=? AND order_item_id=?", corp, itemID).Find(&totals).Error
		},
		func(db *gorm.DB) error {
			return db.Table("outsourcing_order_lines l").Joins("JOIN outsourcing_orders o ON o.id=l.outsourcing_order_id AND o.corp_id=l.corp_id").Where("l.corp_id=? AND l.order_item_id=? AND o.status NOT IN ('cancelled','deleted')", corp, itemID).Select("l.order_process_id,CAST(SUM(l.quantity-l.started_quantity) AS SIGNED) AS quantity").Group("l.order_process_id").Scan(&reservations).Error
		},
		func(db *gorm.DB) error { var e error; losses, e = archivedLosses(db, corp, itemID); return e },
		func(db *gorm.DB) error {
			var e error
			s.HistoricalProcessNames, e = archivedProcessNames(db, corp, itemID)
			return e
		},
		func(db *gorm.DB) error {
			return db.Table("production_completions").Where("corp_id=? AND order_item_id=? AND status='confirmed'", corp, itemID).Select("CAST(COALESCE(SUM(qualified_quantity),0) AS SIGNED)").Scan(&confirmed).Error
		},
		func(db *gorm.DB) error {
			return db.Where("corp_id=? AND order_item_id=?", corp, itemID).Order("due_date,id").Find(&s.Batches).Error
		},
		func(db *gorm.DB) error {
			return db.Table("production_completions").Select("delivery_batch_id,CAST(SUM(qualified_quantity) AS SIGNED) AS quantity").Where("corp_id=? AND order_item_id=? AND status='confirmed' AND delivery_batch_id IS NOT NULL", corp, itemID).Group("delivery_batch_id").Find(&qs).Error
		},
		func(db *gorm.DB) error {
			return db.Table("order_business_records").Select("delivery_batch_id,SUM(quantity) AS quantity,COALESCE(DATE_FORMAT(MAX(business_date),'%Y-%m-%d'),'') AS last_date").Where("corp_id=? AND order_item_id=? AND status='confirmed' AND kind='delivery' AND delivery_batch_id IS NOT NULL", corp, itemID).Group("delivery_batch_id").Find(&ds).Error
		},
		func(db *gorm.DB) error {
			return db.Where("corp_id=? AND order_item_id=?", corp, itemID).Order("id DESC").Limit(50).Find(&s.Events).Error
		})
	if err != nil {
		return s, err
	}
	byID := map[int64]models.ProcessQuantity{}
	for _, t := range totals {
		byID[t.OrderProcessID] = t
	}
	s.Tracking = len(totals) > 0
	reserved := map[int64]int{}
	for _, r := range reservations {
		reserved[r.OrderProcessID] = r.Quantity
	}
	available := s.Item.ProductionQuantity
	for _, p := range ps {
		available -= lossBefore(losses, p.ID)
		t := byID[p.ID]
		s.Processes = append(s.Processes, ProcessProgress{OrderProcess: p, Tracking: t.ID > 0, Started: t.StartedQuantity, Completed: t.CompletedQuantity, Scrap: t.ScrapQuantity, AvailableStart: max(0, available-t.StartedQuantity), InProgress: max(0, t.StartedQuantity-t.CompletedQuantity-t.ScrapQuantity), OutsourcingReserved: reserved[p.ID], AvailableOrder: max(0, s.Item.ProductionQuantity-t.StartedQuantity-reserved[p.ID])})
		available = t.CompletedQuantity
	}
	if len(ps) > 0 {
		available -= terminalLoss(losses)
	}
	s.AvailableCompletion = max(0, available-confirmed)
	completed := map[int64]int{}
	for _, q := range qs {
		completed[q.DeliveryBatchID] = q.Quantity
	}
	delivered := map[int64]quantity{}
	for _, q := range ds {
		delivered[q.DeliveryBatchID] = q
	}
	for i := range s.Batches {
		s.Batches[i].QualifiedQuantity = completed[s.Batches[i].ID]
		q := delivered[s.Batches[i].ID]
		s.Batches[i].DeliveredQuantity = q.Quantity
		s.Batches[i].LastDeliveryDate = q.LastDate
	}
	return s, nil
}

// Enrich paged lists with one query per table rather than requests per part.
type Summary struct {
	Tracking        bool                   `json:"tracking"`
	Batches         []models.DeliveryBatch `json:"batches"`
	OpenOutsourcing int                    `json:"open_outsourcing"`
	NextDueDate     string                 `json:"next_due_date"`
}

func Summaries(db *gorm.DB, corp int64, ids []int64) (map[int64]Summary, error) {
	result := map[int64]Summary{}
	if len(ids) == 0 {
		return result, nil
	}
	for _, id := range ids {
		result[id] = Summary{Batches: []models.DeliveryBatch{}}
	}
	var rows []struct {
		models.DeliveryBatch
		SummaryItemID          int64
		SummaryTracking        bool
		SummaryOpenOutsourcing int
		Qualified              int    `gorm:"column:qualified_quantity"`
		Delivered              int    `gorm:"column:delivered_quantity"`
		LastDate               string `gorm:"column:last_delivery_date"`
	}
	if err := db.Raw(`SELECT b.*,i.id AS summary_item_id,
 EXISTS(SELECT 1 FROM process_quantity_totals pt WHERE pt.corp_id=i.corp_id AND pt.order_item_id=i.id) AS summary_tracking,
 COALESCE(outs.quantity,0) AS summary_open_outsourcing,
 COALESCE(q.quantity,0) AS qualified_quantity,COALESCE(d.quantity,0) AS delivered_quantity,COALESCE(d.last_date,'') AS last_delivery_date
 FROM order_items i
 LEFT JOIN order_delivery_batches b ON b.corp_id=i.corp_id AND b.order_item_id=i.id
 LEFT JOIN (SELECT delivery_batch_id,CAST(SUM(qualified_quantity) AS SIGNED) AS quantity FROM production_completions WHERE corp_id=? AND status='confirmed' GROUP BY delivery_batch_id) q ON q.delivery_batch_id=b.id
 LEFT JOIN (SELECT delivery_batch_id,SUM(quantity) AS quantity,DATE_FORMAT(MAX(business_date),'%Y-%m-%d') AS last_date FROM order_business_records WHERE corp_id=? AND status='confirmed' AND kind='delivery' GROUP BY delivery_batch_id) d ON d.delivery_batch_id=b.id
 LEFT JOIN (SELECT order_item_id,SUM(open_quantity+rework_waiting_quantity+rework_out_quantity) AS quantity FROM outsourcing_dispatches WHERE corp_id=? GROUP BY order_item_id) outs ON outs.order_item_id=i.id
 WHERE i.corp_id=? AND i.id IN ? ORDER BY b.due_date,b.id`, corp, corp, corp, corp, ids).Scan(&rows).Error; err != nil {
		return nil, err
	}
	for _, r := range rows {
		s := result[r.SummaryItemID]
		s.Tracking = r.SummaryTracking
		s.OpenOutsourcing = r.SummaryOpenOutsourcing
		if r.ID != 0 {
			b := r.DeliveryBatch
			b.QualifiedQuantity = r.Qualified
			b.DeliveredQuantity = r.Delivered
			b.LastDeliveryDate = r.LastDate
			s.Batches = append(s.Batches, b)
			if b.DeliveredQuantity < b.Quantity && (s.NextDueDate == "" || b.DueDate.Format("2006-01-02") < s.NextDueDate) {
				s.NextDueDate = b.DueDate.Format("2006-01-02")
			}
		}
		result[r.SummaryItemID] = s
	}
	return result, nil
}
func ValidateDelivery(tx *gorm.DB, corp, item int64, batch *int64, quantity int, date string) error {
	b, err := BatchScope(tx, corp, item, batch)
	if err != nil {
		return err
	}
	if b == nil {
		return nil
	}
	var delivered, qualified int
	if err = tx.Table("order_business_records").Where("corp_id=? AND delivery_batch_id=? AND kind='delivery' AND status='confirmed'", corp, b.ID).Select("COALESCE(SUM(quantity),0)").Scan(&delivered).Error; err != nil {
		return err
	}
	if err = tx.Table("production_completions").Where("corp_id=? AND delivery_batch_id=? AND status='confirmed'", corp, b.ID).Select("CAST(COALESCE(SUM(qualified_quantity),0) AS SIGNED)").Scan(&qualified).Error; err != nil {
		return err
	}
	if delivered+quantity > b.Quantity || delivered+quantity > qualified {
		return fmtDelivery()
	}
	// Replay every batch date, including the proposed record. Future production cannot justify earlier shipping.
	type event struct {
		Day       string
		Qualified int
		Delivered int
	}
	var events []event
	if err = tx.Raw(`SELECT DATE_FORMAT(business_date,'%Y-%m-%d') AS day,CAST(SUM(q) AS SIGNED) AS qualified,SUM(d) AS delivered FROM (
 SELECT business_date,qualified_quantity AS q,0 AS d FROM production_completions WHERE corp_id=? AND delivery_batch_id=? AND status='confirmed'
 UNION ALL SELECT business_date,0,quantity FROM order_business_records WHERE corp_id=? AND delivery_batch_id=? AND kind='delivery' AND status='confirmed'
 UNION ALL SELECT ?,0,?) x GROUP BY business_date ORDER BY business_date`, corp, b.ID, corp, b.ID, date, quantity).Scan(&events).Error; err != nil {
		return err
	}
	q, d := 0, 0
	for _, e := range events {
		q += e.Qualified
		d += e.Delivered
		if d > q {
			return fmtDelivery()
		}
	}
	return nil
}
