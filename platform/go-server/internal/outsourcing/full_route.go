package outsourcing

import (
	"errors"
	"fmt"
	"gorm.io/gorm"
	"machining-erp/internal/models"
	"machining-erp/internal/productionflow"
)

func updateOrderDispatchStatus(tx *gorm.DB, o models.OutsourcingOrder) error {
	lines, e := orderLines(tx, o.CorpID, o.ID)
	if e != nil {
		return e
	}
	status := "dispatched"
	anySent := false
	for _, l := range lines {
		if l.DispatchedQuantity > 0 {
			anySent = true
		}
	}
	if !anySent {
		return tx.Model(&o).Update("status", "ordered").Error
	}
	for _, l := range lines {
		if l.DispatchedQuantity < l.Quantity {
			status = "partial_dispatched"
			break
		}
	}
	return tx.Model(&o).Update("status", status).Error
}

// Only returned pieces prove the covered route was performed. No synthetic progress at ordering/dispatch.
// Good, rework and scrap are inspected at the final process; earlier covered stages are summarized on return.
func advanceFullRoute(tx *gorm.DB, item models.OrderItem, d models.OutsourcingDispatch, actor int64, qty int, in ReturnInput) error {
	if d.OutsourcingOrderID == nil {
		return errors.New("整套外协记录缺少订单归属")
	}
	cover := []models.OutsourcingDispatchProcess{}
	if e := tx.Where("corp_id=? AND dispatch_id=?", d.CorpID, d.ID).Order("sequence,id").Find(&cover).Error; e != nil {
		return e
	}
	if len(cover) == 0 || cover[len(cover)-1].OrderProcessID != d.OrderProcessID {
		return errors.New("整套外协工序快照不一致")
	}
	for i, p := range cover {
		if i > 0 {
			if e := tx.Model(&models.OutsourcingOrderLine{}).Where("corp_id=? AND outsourcing_order_id=? AND order_process_id=?", d.CorpID, *d.OutsourcingOrderID, p.OrderProcessID).Update("started_quantity", gorm.Expr("started_quantity+?", qty)).Error; e != nil {
				return e
			}
			if e := productionflow.ReportTx(tx, item, p.OrderProcessID, actor, productionflow.ReportInput{Kind: "start", Quantity: qty, Date: in.Date, Notes: in.Notes, RequestID: reportID(in.RequestID, fmt.Sprintf("full-start-%d", p.OrderProcessID))}, "outsourcing"); e != nil {
				return e
			}
		}
		if i < len(cover)-1 {
			if e := productionflow.ReportTx(tx, item, p.OrderProcessID, actor, productionflow.ReportInput{Kind: "complete", Quantity: qty, Date: in.Date, Notes: in.Notes, RequestID: reportID(in.RequestID, fmt.Sprintf("full-complete-%d", p.OrderProcessID))}, "outsourcing"); e != nil {
				return e
			}
		}
	}
	return nil
}
