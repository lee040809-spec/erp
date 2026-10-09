package outsourcing

import (
	"errors"
	"gorm.io/gorm"
	"machining-erp/internal/models"
	"machining-erp/internal/productionflow"
	"time"
)

type OrderLineInput struct {
	ProcessID int64  `json:"order_process_id"`
	Quantity  int    `json:"quantity"`
	UnitPrice string `json:"unit_price"`
}
type OrderInput struct {
	ItemID       int64            `json:"order_item_id"`
	SupplierID   int64            `json:"supplier_id"`
	Scope        string           `json:"scope"`
	BatchID      *int64           `json:"delivery_batch_id"`
	Quantity     int              `json:"quantity"`
	UnitPrice    string           `json:"unit_price"`
	Date         string           `json:"order_date"`
	Expected     string           `json:"expected_return_date"`
	Notes        string           `json:"notes"`
	ChangeReason string           `json:"change_reason"`
	RequestID    string           `json:"request_id"`
	Lines        []OrderLineInput `json:"lines"`
}

func orderLines(tx *gorm.DB, corp, id int64) ([]models.OutsourcingOrderLine, error) {
	rows := []models.OutsourcingOrderLine{}
	e := tx.Where("corp_id=? AND outsourcing_order_id=?", corp, id).Order("sequence,id").Find(&rows).Error
	return rows, e
}
func CreateOrder(db *gorm.DB, corp, actor int64, in OrderInput) (models.OutsourcingOrder, error) {
	return saveOrder(db, corp, actor, 0, in, "")
}
func UpdateOrder(db *gorm.DB, corp, actor, id int64, in OrderInput, token string) (models.OutsourcingOrder, error) {
	return saveOrder(db, corp, actor, id, in, token)
}
func saveOrder(db *gorm.DB, corp, actor, id int64, in OrderInput, token string) (models.OutsourcingOrder, error) {
	var out models.OutsourcingOrder
	date, e := productionflow.Date(in.Date, true)
	if e != nil {
		return out, e
	}
	expected, e := productionflow.Date(in.Expected, false)
	if e != nil || expected.Before(date) {
		return out, errors.New("预计回厂不能早于外协下单日期")
	}
	if !productionflow.RequestID(in.RequestID) || !productionflow.Text(in.Notes, 500) || !productionflow.Text(in.ChangeReason, 500) || (in.Scope != "full" && in.Scope != "selected") || len(in.Lines) > 50 {
		return out, errors.New("外协订单信息无效")
	}
	if in.Scope == "full" && (in.Quantity <= 0 || in.Quantity > 2147483647 || !decimal(in.UnitPrice, 4)) {
		return out, errors.New("请填写整套工序外协数量与非负单件总价")
	}
	e = db.Transaction(func(tx *gorm.DB) error {
		item, e := productionflow.LockItem(tx, corp, in.ItemID)
		if e != nil {
			return e
		}
		var before models.OutsourcingOrder
		if id > 0 {
			if e = tx.Where("corp_id=? AND id=? AND order_item_id=?", corp, id, item.ID).Take(&before).Error; e != nil {
				return errors.New("外协订单不存在")
			}
			before.Lines, e = orderLines(tx, corp, id)
			if e != nil {
				return e
			}
			if e = editableOrder(tx, before); e != nil {
				return e
			}
			if token != "" && sameOrderInput(before, in) {
				out = before
				out.EditToken = OrderToken(out)
				return nil
			}
			if token == "" || token != OrderToken(before) {
				return errors.New("外协订单已变化，请刷新后重新修改")
			}
			if e = tx.Model(&before).Update("status", "cancelled").Error; e != nil {
				return e
			}
		} else {
			e = tx.Where("corp_id=? AND request_id=?", corp, in.RequestID).Take(&out).Error
			if e == nil {
				if out.OrderItemID != in.ItemID || out.SupplierID != in.SupplierID || out.Scope != in.Scope || !SameID(out.DeliveryBatchID, in.BatchID) || out.OrderDate.Format("2006-01-02") != in.Date || out.ExpectedReturnDate.Format("2006-01-02") != in.Expected || out.Notes != in.Notes {
					return errors.New("请求编号已使用且内容不同")
				}
				out.Lines, e = orderLines(tx, corp, out.ID)
				if e != nil {
					return e
				}
				if in.Scope == "full" {
					if len(out.Lines) == 0 || out.Lines[0].Quantity != in.Quantity || !equalMoney(out.UnitPrice, in.UnitPrice) {
						return errors.New("请求编号已使用且数量或单价不同")
					}
				} else {
					if len(in.Lines) != len(out.Lines) {
						return errors.New("请求编号已使用且工序不同")
					}
					m := map[int64]OrderLineInput{}
					for _, l := range in.Lines {
						if _, ok := m[l.ProcessID]; ok {
							return errors.New("工序重复")
						}
						m[l.ProcessID] = l
					}
					for _, l := range out.Lines {
						v, ok := m[l.OrderProcessID]
						if !ok || v.Quantity != l.Quantity || !equalMoney(v.UnitPrice, l.UnitPrice) {
							return errors.New("请求编号已使用且工序数量或单价不同")
						}
					}
				}
				return nil
			}
			if e != gorm.ErrRecordNotFound {
				return e
			}
		}
		if item.StartDate != nil && in.Date < item.StartDate.Format("2006-01-02") {
			return errors.New("外协下单日期不能早于零件下单日期")
		}
		supplier, e := activeSupplier(tx, corp, in.SupplierID)
		if e != nil {
			return e
		}
		if in.BatchID != nil {
			if _, e = productionflow.BatchScope(tx, corp, item.ID, in.BatchID); e != nil {
				return e
			}
		}
		ps := []models.OrderProcess{}
		if e = tx.Where("corp_id=? AND order_item_id=? AND name<>'送货'", corp, item.ID).Order("sort_order,id").Find(&ps).Error; e != nil {
			return e
		}
		if len(ps) == 0 || len(ps) > 50 {
			return errors.New("请先录入零件生产工序（最多50道）")
		}
		selected := map[int64]OrderLineInput{}
		if in.Scope == "full" {
			for _, p := range ps {
				selected[p.ID] = OrderLineInput{p.ID, in.Quantity, "0"}
			}
		} else {
			for _, l := range in.Lines {
				if l.Quantity <= 0 || l.Quantity > 2147483647 || !decimal(l.UnitPrice, 4) {
					return errors.New("各工序数量与单价无效")
				}
				if _, ok := selected[l.ProcessID]; ok {
					return errors.New("同一张外协订单不能重复选择工序")
				}
				selected[l.ProcessID] = l
			}
			if len(selected) == 0 {
				return errors.New("请选择需外协的工序")
			}
		}
		tracking, e := productionflow.Enabled(tx, corp, item.ID)
		if e != nil {
			return e
		}
		lines := []models.OutsourcingOrderLine{}
		for sequence, p := range ps {
			l, ok := selected[p.ID]
			if !ok {
				continue
			}
			var started int
			if e = tx.Model(&models.ProcessQuantity{}).Where("corp_id=? AND order_process_id=?", corp, p.ID).Select("COALESCE(MAX(started_quantity),0)").Scan(&started).Error; e != nil {
				return e
			}
			if !tracking && (p.Status != "pending" && p.Status != "" || p.ActualStartAt != nil || p.ActualEndAt != nil) {
				return errors.New("选中工序已有历史进度，请先核对并启用数量跟踪")
			}
			if !p.IsOutsourced && started > 0 {
				return errors.New("已按自制开工的工序不能改为外协，请先处理原报工")
			}
			if e = productionflow.GuardReservedCapacity(tx, item, p.ID, l.Quantity); e != nil {
				return e
			}
			if e = tx.Model(&models.OrderProcess{}).Where("corp_id=? AND id=?", corp, p.ID).Update("is_outsourced", true).Error; e != nil {
				return e
			}
			lines = append(lines, models.OutsourcingOrderLine{CorpID: corp, OrderItemID: item.ID, OrderProcessID: p.ID, ProcessName: p.Name, Sequence: sequence, Quantity: l.Quantity, UnitPrice: l.UnitPrice})
		}
		if len(lines) != len(selected) {
			return errors.New("所选工序不属于当前零件")
		}
		price := "0"
		if in.Scope == "full" {
			price = in.UnitPrice
		}
		out = models.OutsourcingOrder{CorpID: corp, OrderID: item.OrderID, OrderItemID: item.ID, SupplierID: supplier.ID, SupplierName: supplier.Name, OrderNo: "WXD-" + in.RequestID, Scope: in.Scope, DeliveryBatchID: in.BatchID, UnitPrice: price, OrderDate: date, ExpectedReturnDate: expected, Status: "ordered", Notes: in.Notes, RequestID: in.RequestID, CreatedBy: actor, CreatedAt: time.Now()}
		if id > 0 {
			out.ID = before.ID
			out.OrderNo = before.OrderNo
			out.RequestID = before.RequestID
			out.CreatedBy = before.CreatedBy
			out.CreatedAt = before.CreatedAt
			if e = tx.Save(&out).Error; e != nil {
				return e
			}
			// Only unused lines can be replaced; snapshots of the prior order are kept in the audit log.
			if e = tx.Where("corp_id=? AND outsourcing_order_id=?", corp, id).Delete(&models.OutsourcingOrderLine{}).Error; e != nil {
				return e
			}
		} else if e = tx.Create(&out).Error; e != nil {
			return e
		}
		for i := range lines {
			lines[i].OutsourcingOrderID = out.ID
			for _, old := range before.Lines {
				if old.OrderProcessID == lines[i].OrderProcessID {
					lines[i].ID = old.ID
				}
			}
		}
		if e = tx.Create(&lines).Error; e != nil {
			return e
		}
		out.Lines = lines
		if id > 0 {
			reason := in.ChangeReason
			if reason == "" {
				reason = in.Notes
			}
			if e = auditChange(tx, corp, actor, "order", id, "update", reason, before, out); e != nil {
				return e
			}
		}
		out.EditToken = OrderToken(out)
		return nil
	})
	return out, e
}
func CancelOrder(db *gorm.DB, corp, id int64) error {
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
		lines, e := orderLines(tx, corp, id)
		if e != nil {
			return e
		}
		for _, l := range lines {
			if l.DispatchedQuantity > 0 {
				return errors.New("外协订单已有实物发出，不能取消；请继续登记回厂和对账")
			}
		}
		if o.Status == "deleted" {
			return errors.New("已删除的外协订单不能取消")
		}
		return tx.Model(&o).Update("status", "cancelled").Error
	})
}

type OrderDispatchInput struct {
	LineID    *int64 `json:"order_line_id"`
	Quantity  int    `json:"quantity"`
	Date      string `json:"dispatch_date"`
	Expected  string `json:"expected_return_date"`
	Notes     string `json:"notes"`
	RequestID string `json:"request_id"`
}

func DispatchOrder(db *gorm.DB, corp, id, actor int64, in OrderDispatchInput) (models.OutsourcingDispatch, error) {
	var o models.OutsourcingOrder
	if e := db.Where("corp_id=? AND id=?", corp, id).Take(&o).Error; e != nil {
		return models.OutsourcingDispatch{}, errors.New("外协订单不存在")
	}
	return Dispatch(db, corp, actor, DispatchInput{ItemID: o.OrderItemID, OrderID: &id, LineID: in.LineID, Quantity: in.Quantity, Date: in.Date, Expected: in.Expected, Notes: in.Notes, RequestID: in.RequestID})
}
