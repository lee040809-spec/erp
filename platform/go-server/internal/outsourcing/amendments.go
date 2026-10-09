package outsourcing

import (
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"gorm.io/gorm"
	"machining-erp/internal/models"
	"machining-erp/internal/productionflow"
	"strings"
	"time"
)

func token(v interface{}) string                        { b, _ := json.Marshal(v); return fmt.Sprintf("%x", sha256.Sum256(b)) }
func OrderToken(o models.OutsourcingOrder) string       { o.EditToken = ""; return token(o) }
func DispatchToken(d models.OutsourcingDispatch) string { d.EditToken = ""; return token(d) }

func sameOrderInput(o models.OutsourcingOrder, in OrderInput) bool {
	if o.OrderItemID != in.ItemID || o.SupplierID != in.SupplierID || o.Scope != in.Scope || !SameID(o.DeliveryBatchID, in.BatchID) || o.OrderDate.Format("2006-01-02") != in.Date || o.ExpectedReturnDate.Format("2006-01-02") != in.Expected || o.Notes != in.Notes {
		return false
	}
	if in.Scope == "full" {
		return len(o.Lines) > 0 && o.Lines[0].Quantity == in.Quantity && equalMoney(o.UnitPrice, in.UnitPrice)
	}
	if len(in.Lines) != len(o.Lines) {
		return false
	}
	m := map[int64]OrderLineInput{}
	for _, l := range in.Lines {
		if _, ok := m[l.ProcessID]; ok {
			return false
		}
		m[l.ProcessID] = l
	}
	for _, l := range o.Lines {
		v, ok := m[l.OrderProcessID]
		if !ok || v.Quantity != l.Quantity || !equalMoney(v.UnitPrice, l.UnitPrice) {
			return false
		}
	}
	return true
}
func auditChange(tx *gorm.DB, corp, actor int64, entity string, id int64, action, reason string, before, after interface{}) error {
	b, e := json.Marshal(before)
	if e != nil {
		return e
	}
	a, e := json.Marshal(after)
	if e != nil {
		return e
	}
	return tx.Create(&models.OutsourcingChange{CorpID: corp, EntityType: entity, EntityID: id, Action: action, Reason: reason, BeforeJSON: string(b), AfterJSON: string(a), CreatedBy: actor, CreatedAt: time.Now()}).Error
}
func editableOrder(tx *gorm.DB, o models.OutsourcingOrder) error {
	if o.Status == "cancelled" || o.Status == "deleted" {
		return errors.New("已取消或删除的外协订单不能修改")
	}
	for _, l := range o.Lines {
		if l.DispatchedQuantity > 0 || l.StartedQuantity > 0 {
			return errors.New("该外协订单已有实物发出，不能直接修改工序、数量或报价；请先处理发出记录")
		}
	}
	var n int64
	if e := tx.Model(&models.OutsourcingDispatch{}).Where("corp_id=? AND outsourcing_order_id=? AND status<>'cancelled'", o.CorpID, o.ID).Count(&n).Error; e != nil {
		return e
	}
	if n > 0 {
		return errors.New("该外协订单已有有效发出记录，不能修改或删除")
	}
	return nil
}

type RemoveInput struct {
	EditToken string `json:"edit_token"`
	Reason    string `json:"reason"`
}

func DeleteOrder(db *gorm.DB, corp, id, actor int64, in RemoveInput) error {
	if strings.TrimSpace(in.Reason) == "" || !productionflow.Text(in.Reason, 500) {
		return errors.New("请填写删除原因（最多500字）")
	}
	return db.Transaction(func(tx *gorm.DB) error {
		var o models.OutsourcingOrder
		if e := tx.Where("corp_id=? AND id=?", corp, id).Take(&o).Error; e != nil {
			return errors.New("外协订单不存在")
		}
		if _, e := productionflow.LockItem(tx, corp, o.OrderItemID); e != nil {
			return e
		}
		if e := tx.Where("corp_id=? AND id=?", corp, id).Take(&o).Error; e != nil {
			return e
		}
		var e error
		o.Lines, e = orderLines(tx, corp, id)
		if e != nil {
			return e
		}
		if o.Status == "deleted" {
			return nil
		}
		if in.EditToken == "" || in.EditToken != OrderToken(o) {
			return errors.New("外协订单已变化，请刷新后重新删除")
		}
		if o.Status != "cancelled" {
			if e = editableOrder(tx, o); e != nil {
				return e
			}
		}
		before := o
		o.Status = "deleted"
		if e = tx.Model(&o).Update("status", o.Status).Error; e != nil {
			return e
		}
		return auditChange(tx, corp, actor, "order", id, "delete", in.Reason, before, o)
	})
}

type DispatchEditInput struct {
	EditToken string `json:"edit_token"`
	Expected  string `json:"expected_return_date"`
	Notes     string `json:"notes"`
}

func UpdateDispatch(db *gorm.DB, corp, id, actor int64, in DispatchEditInput) (models.OutsourcingDispatch, error) {
	var out models.OutsourcingDispatch
	expected, e := productionflow.Date(in.Expected, false)
	if e != nil {
		return out, e
	}
	if !productionflow.Text(in.Notes, 500) {
		return out, errors.New("说明不能超过500字")
	}
	e = db.Transaction(func(tx *gorm.DB) error {
		_, d, e := lock(tx, corp, id)
		if e != nil {
			return e
		}
		if d.Status == "cancelled" || d.Status == "settled" {
			return errors.New("已撤销或入账的外协记录不能修改")
		}
		if expected.Before(d.DispatchDate) {
			return errors.New("预计回厂不能早于实际发出日期")
		}
		// A successful retry is harmless even when the original response was lost.
		if d.ExpectedReturnDate.Format("2006-01-02") == in.Expected && d.Notes == in.Notes {
			out = d
			return nil
		}
		if in.EditToken == "" || in.EditToken != DispatchToken(d) {
			return errors.New("外协记录已变化，请刷新后重新修改")
		}
		before := d
		d.ExpectedReturnDate = expected
		d.Notes = in.Notes
		if e = tx.Model(&d).Updates(map[string]interface{}{"expected_return_date": expected, "notes": in.Notes}).Error; e != nil {
			return e
		}
		if e = auditChange(tx, corp, actor, "dispatch", id, "update", in.Notes, before, d); e != nil {
			return e
		}
		out = d
		return nil
	})
	out.EditToken = DispatchToken(out)
	return out, e
}

// CancelDispatch reverses only an untouched shipment. Quality and accounting history is never erased.
func CancelDispatch(db *gorm.DB, corp, id, actor int64, in RemoveInput) error {
	if strings.TrimSpace(in.Reason) == "" || !productionflow.Text(in.Reason, 500) {
		return errors.New("请填写撤销原因（最多500字）")
	}
	return db.Transaction(func(tx *gorm.DB) error {
		item, d, e := lock(tx, corp, id)
		if e != nil {
			return e
		}
		if d.Status == "cancelled" {
			return nil
		}
		if in.EditToken == "" || in.EditToken != DispatchToken(d) {
			return errors.New("外协记录已变化，请刷新后重新撤销")
		}
		if d.Status != "dispatched" || d.OpenQuantity != d.Quantity || d.AcceptedQuantity+d.ScrapQuantity+d.ReworkOutQuantity+d.ReworkWaitingQuantity != 0 {
			return errors.New("已有回厂、返工或对账记录，不能删除发出记录")
		}
		for _, m := range []interface{}{&models.OutsourcingEvent{}, &models.OutsourcingSettlement{}} {
			var n int64
			if e = tx.Model(m).Where("corp_id=? AND dispatch_id=?", corp, id).Count(&n).Error; e != nil {
				return e
			}
			if n > 0 {
				return errors.New("已有回厂或费用记录，不能删除发出记录")
			}
		}
		startProcess := d.OrderProcessID
		if d.OutsourcingOrderID != nil {
			var o models.OutsourcingOrder
			if e = tx.Where("corp_id=? AND id=?", corp, *d.OutsourcingOrderID).Take(&o).Error; e != nil {
				return e
			}
			lines, e := orderLines(tx, corp, o.ID)
			if e != nil {
				return e
			}
			covered := []models.OutsourcingOrderLine{}
			for _, l := range lines {
				if d.Scope == "full" || d.OutsourcingOrderLineID != nil && l.ID == *d.OutsourcingOrderLineID {
					covered = append(covered, l)
				}
			}
			if len(covered) == 0 {
				return errors.New("发出记录的工序归属异常，请核对")
			}
			startProcess = covered[0].OrderProcessID
			for i, l := range covered {
				if l.DispatchedQuantity < d.Quantity || i == 0 && l.StartedQuantity < d.Quantity {
					return errors.New("外协工序累计数量异常，无法撤销")
				}
				values := map[string]interface{}{"dispatched_quantity": gorm.Expr("dispatched_quantity-?", d.Quantity)}
				if i == 0 {
					values["started_quantity"] = gorm.Expr("started_quantity-?", d.Quantity)
				}
				if e = tx.Model(&l).Updates(values).Error; e != nil {
					return e
				}
			}
			if e = updateOrderDispatchStatus(tx, o); e != nil {
				return e
			}
		}
		if e = productionflow.ReverseOutsourcingStart(tx, item, startProcess, actor, d.Quantity, reportID(d.RequestID, "start"), in.Reason); e != nil {
			return e
		}
		before := d
		d.Status = "cancelled"
		d.OpenQuantity = 0
		if e = tx.Model(&d).Updates(map[string]interface{}{"status": "cancelled", "open_quantity": 0}).Error; e != nil {
			return e
		}
		return auditChange(tx, corp, actor, "dispatch", id, "delete", in.Reason, before, d)
	})
}
