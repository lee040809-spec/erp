// Package outsourcing owns dispatch and quality quantities, sharing the part's process ledger.
package outsourcing

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"machining-erp/internal/models"
	"machining-erp/internal/productionflow"
	"strings"
	"time"
)

type DispatchInput struct {
	SupplierID *int64 `json:"supplier_id"`
	OrderID    *int64 `json:"outsourcing_order_id"`
	LineID     *int64 `json:"outsourcing_order_line_id"`
	ItemID     int64  `json:"order_item_id"`
	ProcessID  int64  `json:"order_process_id"`
	BatchID    *int64 `json:"delivery_batch_id"`
	Supplier   string `json:"supplier_name"`
	Quantity   int    `json:"quantity"`
	UnitPrice  string `json:"unit_price"`
	Date       string `json:"dispatch_date"`
	Expected   string `json:"expected_return_date"`
	Notes      string `json:"notes"`
	RequestID  string `json:"request_id"`
}

func SameID(a, b *int64) bool { return a == nil && b == nil || a != nil && b != nil && *a == *b }
func reportID(request, kind string) string {
	sum := sha256.Sum256([]byte(request + ":" + kind))
	return fmt.Sprintf("OX-%x", sum[:16])
}
func Dispatch(db *gorm.DB, corp, actor int64, in DispatchInput) (models.OutsourcingDispatch, error) {
	var out models.OutsourcingDispatch
	date, err := productionflow.Date(in.Date, true)
	if err != nil {
		return out, err
	}
	expected, err := productionflow.Date(in.Expected, false)
	if err != nil || expected.Before(date) {
		return out, errors.New("预计回厂日期不能早于发出日期")
	}
	if !productionflow.RequestID(in.RequestID) || in.Quantity <= 0 || in.Quantity > 2147483647 || !productionflow.Text(in.Notes, 500) || in.OrderID == nil && ((!productionflow.Text(in.Supplier, 100) || in.SupplierID == nil && strings.TrimSpace(in.Supplier) == "") || !decimal(in.UnitPrice, 4)) {
		return out, errors.New("请填写有效供应商、数量、非负单价和备注")
	}
	err = db.Transaction(func(tx *gorm.DB) error {
		item, e := productionflow.LockItem(tx, corp, in.ItemID)
		if e != nil {
			return e
		}
		e = tx.Where("corp_id=? AND request_id=?", corp, in.RequestID).Take(&out).Error
		if e == nil {
			if out.OrderItemID != in.ItemID || !SameID(out.OutsourcingOrderID, in.OrderID) || !SameID(out.OutsourcingOrderLineID, in.LineID) || out.Quantity != in.Quantity || out.DispatchDate.Format("2006-01-02") != in.Date || out.ExpectedReturnDate.Format("2006-01-02") != in.Expected || out.Notes != in.Notes || in.OrderID == nil && (out.OrderProcessID != in.ProcessID || !SameID(out.DeliveryBatchID, in.BatchID) || in.SupplierID == nil && out.SupplierName != in.Supplier || in.SupplierID != nil && !SameID(out.SupplierID, in.SupplierID) || !equalMoney(out.UnitPrice, in.UnitPrice)) {
				return errors.New("请求编号已使用且内容不同")
			}
			return nil
		}
		if e != gorm.ErrRecordNotFound {
			return e
		}
		var covered []models.OutsourcingOrderLine
		scope := "selected"
		startProcess := in.ProcessID
		if in.OrderID != nil {
			var order models.OutsourcingOrder
			if e = tx.Where("corp_id=? AND id=? AND order_item_id=?", corp, *in.OrderID, item.ID).Take(&order).Error; e != nil {
				return errors.New("外协订单不属于当前零件")
			}
			if (order.Status == "cancelled" || order.Status == "deleted") || in.Date < order.OrderDate.Format("2006-01-02") {
				return errors.New("已取消外协订单不能发出，发出日期也不能早于外协下单")
			}
			lines, e := orderLines(tx, corp, order.ID)
			if e != nil {
				return e
			}
			if order.Scope == "full" {
				if in.LineID != nil {
					return errors.New("整套工序请一次发出，无需选择单道工序")
				}
				covered = lines
				scope = "full"
			} else {
				if in.LineID == nil {
					return errors.New("请选择本次实际发出的外协订单工序")
				}
				for _, l := range lines {
					if l.ID == *in.LineID {
						covered = append(covered, l)
					}
				}
			}
			if len(covered) == 0 {
				return errors.New("外协订单工序不存在")
			}
			for index, l := range covered {
				if in.Quantity > l.Quantity-l.DispatchedQuantity {
					return errors.New("本次发出超过外协订单尚未发出数量")
				}
				changes := map[string]interface{}{"dispatched_quantity": gorm.Expr("dispatched_quantity+?", in.Quantity)}
				if index == 0 {
					changes["started_quantity"] = gorm.Expr("started_quantity+?", in.Quantity)
				}
				if e = tx.Model(&models.OutsourcingOrderLine{}).Where("corp_id=? AND id=?", corp, l.ID).Updates(changes).Error; e != nil {
					return e
				}
			}
			startProcess = covered[0].OrderProcessID
			in.ProcessID = covered[len(covered)-1].OrderProcessID
			in.BatchID = order.DeliveryBatchID
			in.SupplierID = &order.SupplierID
			in.UnitPrice = order.UnitPrice
			if scope == "selected" {
				in.UnitPrice = covered[0].UnitPrice
			}
			if e = updateOrderDispatchStatus(tx, order); e != nil {
				return e
			}
		}
		if in.SupplierID != nil {
			supplier, e := activeSupplier(tx, corp, *in.SupplierID)
			if e != nil {
				return e
			}
			in.Supplier = supplier.Name
		}
		var p models.OrderProcess
		if e = tx.Where("id=? AND corp_id=? AND order_item_id=? AND is_outsourced=1 AND name<>'送货'", in.ProcessID, corp, item.ID).Take(&p).Error; e != nil {
			return errors.New("请选择当前零件的外协生产工序")
		}
		// Dispatch may span delivery plans. A selected plan narrows cost allocation, not the shared route.
		if in.BatchID != nil {
			if _, e = productionflow.BatchScope(tx, corp, item.ID, in.BatchID); e != nil {
				return e
			}
		}
		if e = productionflow.ReportTx(tx, item, startProcess, actor, productionflow.ReportInput{Kind: "start", Quantity: in.Quantity, Date: in.Date, Notes: in.Notes, RequestID: reportID(in.RequestID, "start")}, "outsourcing"); e != nil {
			return e
		}
		out = models.OutsourcingDispatch{CorpID: corp, OrderID: item.OrderID, OrderItemID: item.ID, OrderProcessID: p.ID, DeliveryBatchID: in.BatchID, DispatchNo: "WX-" + in.RequestID, SupplierName: in.Supplier, Quantity: in.Quantity, UnitPrice: in.UnitPrice, DispatchDate: date, ExpectedReturnDate: expected, OpenQuantity: in.Quantity, Status: "dispatched", RequestID: in.RequestID, Notes: in.Notes, CreatedBy: actor, CreatedAt: time.Now()}
		out.SupplierID = in.SupplierID
		out.OutsourcingOrderID = in.OrderID
		out.OutsourcingOrderLineID = in.LineID
		out.Scope = scope
		if e = tx.Create(&out).Error; e != nil {
			return e
		}
		if scope == "full" {
			for _, l := range covered {
				if e = tx.Create(&models.OutsourcingDispatchProcess{CorpID: corp, DispatchID: out.ID, OrderProcessID: l.OrderProcessID, Sequence: l.Sequence}).Error; e != nil {
					return e
				}
			}
		}
		return nil
	})
	return out, err
}

type ReturnInput struct {
	Kind       string `json:"kind"` // return or rework_return
	Good       int    `json:"good_quantity"`
	Concession int    `json:"concession_quantity"`
	Rework     int    `json:"rework_quantity"`
	Scrap      int    `json:"scrap_quantity"`
	Date       string `json:"business_date"`
	Notes      string `json:"notes"`
	RequestID  string `json:"request_id"`
}
type ReworkInput struct {
	Quantity  int    `json:"quantity"`
	Date      string `json:"business_date"`
	Notes     string `json:"notes"`
	RequestID string `json:"request_id"`
}

func lock(tx *gorm.DB, corp, id int64) (models.OrderItem, models.OutsourcingDispatch, error) {
	var d models.OutsourcingDispatch
	if err := tx.Where("corp_id=? AND id=?", corp, id).Take(&d).Error; err != nil {
		return models.OrderItem{}, d, errors.New("外协单不存在")
	}
	item, err := productionflow.LockItem(tx, corp, d.OrderItemID)
	if err != nil {
		return item, d, err
	}
	err = tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id=? AND corp_id=?", id, corp).Take(&d).Error
	return item, d, err
}
func eventDate(tx *gorm.DB, d models.OutsourcingDispatch, date string) error {
	if date < d.DispatchDate.Format("2006-01-02") {
		return errors.New("回厂/返工日期不能早于外协发出")
	}
	var latest string
	if err := tx.Model(&models.OutsourcingEvent{}).Where("corp_id=? AND dispatch_id=?", d.CorpID, d.ID).Select("COALESCE(DATE_FORMAT(MAX(business_date),'%Y-%m-%d'),'')").Scan(&latest).Error; err != nil {
		return err
	}
	if date < latest {
		return errors.New("外协流转日期不能早于该单最近一次流转，请核对")
	}
	return nil
}
func persist(tx *gorm.DB, d *models.OutsourcingDispatch) error {
	if d.Quantity != d.OpenQuantity+d.ReworkWaitingQuantity+d.ReworkOutQuantity+d.AcceptedQuantity+d.ScrapQuantity {
		return errors.New("外协数量不守恒")
	}
	d.Status = "dispatched"
	if d.OpenQuantity == 0 && d.ReworkOutQuantity == 0 && d.ReworkWaitingQuantity == 0 {
		d.Status = "returned"
	} else if d.ReworkWaitingQuantity > 0 {
		d.Status = "quality_pending"
	} else if d.ReworkOutQuantity > 0 {
		d.Status = "reworking"
	}
	return tx.Save(d).Error
}
func Receive(db *gorm.DB, corp, id, actor int64, in ReturnInput) error {
	date, err := productionflow.Date(in.Date, true)
	if err != nil {
		return err
	}
	total := int64(in.Good) + int64(in.Concession) + int64(in.Rework) + int64(in.Scrap)
	if (in.Kind != "return" && in.Kind != "rework_return") || in.Good < 0 || in.Concession < 0 || in.Rework < 0 || in.Scrap < 0 || total <= 0 || total > 2147483647 || !productionflow.RequestID(in.RequestID) || !productionflow.Text(in.Notes, 500) {
		return errors.New("回厂数量和请求信息无效")
	}
	if (in.Concession > 0 || in.Rework > 0 || in.Scrap > 0) && strings.TrimSpace(in.Notes) == "" {
		return errors.New("让步接收、返工或报废须填写处理说明")
	}
	return db.Transaction(func(tx *gorm.DB) error {
		item, d, e := lock(tx, corp, id)
		if e != nil {
			return e
		}
		var previous models.OutsourcingEvent
		e = tx.Where("corp_id=? AND request_id=?", corp, in.RequestID).Take(&previous).Error
		if e == nil {
			if previous.DispatchID != id || previous.Kind != in.Kind || previous.GoodQuantity != in.Good || previous.ConcessionQuantity != in.Concession || previous.ReworkQuantity != in.Rework || previous.ScrapQuantity != in.Scrap || previous.BusinessDate.Format("2006-01-02") != in.Date || previous.Notes != in.Notes {
				return errors.New("请求编号已使用且内容不同")
			}
			return nil
		}
		if e != gorm.ErrRecordNotFound {
			return e
		}
		if d.Status == "settled" || d.Status == "cancelled" {
			return errors.New("已对账入账的外协单不能再登记回厂")
		}
		if e = eventDate(tx, d, in.Date); e != nil {
			return e
		}
		if in.Kind == "return" {
			if int(total) > d.OpenQuantity {
				return errors.New("本次回厂超过尚未回厂数量")
			}
			d.OpenQuantity -= int(total)
		} else {
			if int(total) > d.ReworkOutQuantity {
				return errors.New("本次返工回厂超过已送返工数量")
			}
			d.ReworkOutQuantity -= int(total)
		}
		accepted := in.Good + in.Concession
		if d.Scope == "full" && in.Kind == "return" {
			if e = advanceFullRoute(tx, item, d, actor, int(total), in); e != nil {
				return e
			}
		}
		if accepted > 0 {
			if e = productionflow.ReportTx(tx, item, d.OrderProcessID, actor, productionflow.ReportInput{Kind: "complete", Quantity: accepted, Date: in.Date, Notes: in.Notes, RequestID: reportID(in.RequestID, "complete")}, "outsourcing"); e != nil {
				return e
			}
		}
		if in.Scrap > 0 {
			if e = productionflow.ReportTx(tx, item, d.OrderProcessID, actor, productionflow.ReportInput{Kind: "scrap", Quantity: in.Scrap, Date: in.Date, Notes: in.Notes, RequestID: reportID(in.RequestID, "scrap")}, "outsourcing"); e != nil {
				return e
			}
		}
		d.AcceptedQuantity += accepted
		d.ConcessionQuantity += in.Concession
		d.ScrapQuantity += in.Scrap
		d.ReworkWaitingQuantity += in.Rework
		if e = persist(tx, &d); e != nil {
			return e
		}
		return tx.Create(&models.OutsourcingEvent{CorpID: corp, DispatchID: id, Kind: in.Kind, Quantity: int(total), GoodQuantity: in.Good, ConcessionQuantity: in.Concession, ReworkQuantity: in.Rework, ScrapQuantity: in.Scrap, BusinessDate: date, Notes: in.Notes, RequestID: in.RequestID, CreatedBy: actor, CreatedAt: time.Now()}).Error
	})
}
func SendRework(db *gorm.DB, corp, id, actor int64, in ReworkInput) error {
	date, err := productionflow.Date(in.Date, true)
	if err != nil {
		return err
	}
	if in.Quantity <= 0 || !productionflow.RequestID(in.RequestID) || !productionflow.Text(in.Notes, 500) {
		return errors.New("返工发出信息无效")
	}
	return db.Transaction(func(tx *gorm.DB) error {
		_, d, e := lock(tx, corp, id)
		if e != nil {
			return e
		}
		var prior models.OutsourcingEvent
		e = tx.Where("corp_id=? AND request_id=?", corp, in.RequestID).Take(&prior).Error
		if e == nil {
			if prior.DispatchID != id || prior.Kind != "rework_send" || prior.Quantity != in.Quantity || prior.BusinessDate.Format("2006-01-02") != in.Date || prior.Notes != in.Notes {
				return errors.New("请求编号已使用且内容不同")
			}
			return nil
		}
		if e != gorm.ErrRecordNotFound {
			return e
		}
		if in.Quantity > d.ReworkWaitingQuantity || d.Status == "settled" {
			return errors.New("返工发出数量超过待返工数量")
		}
		if e = eventDate(tx, d, in.Date); e != nil {
			return e
		}
		d.ReworkWaitingQuantity -= in.Quantity
		d.ReworkOutQuantity += in.Quantity
		if e = persist(tx, &d); e != nil {
			return e
		}
		return tx.Create(&models.OutsourcingEvent{CorpID: corp, DispatchID: id, Kind: "rework_send", Quantity: in.Quantity, BusinessDate: date, Notes: in.Notes, RequestID: in.RequestID, CreatedBy: actor, CreatedAt: time.Now()}).Error
	})
}
