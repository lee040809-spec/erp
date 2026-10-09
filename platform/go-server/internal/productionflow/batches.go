package productionflow

import (
	"errors"
	"gorm.io/gorm"
	"machining-erp/internal/models"
	"strings"
	"time"
)

func ConfigureBatches(tx *gorm.DB, item models.OrderItem, inputs []models.DeliveryBatchInput, baseline []models.ProcessBaseline, actor int64) error {
	if len(inputs) == 0 || len(inputs) > 50 {
		return errors.New("交货批次须为1至50批")
	}
	var old []models.DeliveryBatch
	if err := tx.Where("corp_id=? AND order_item_id=?", item.CorpID, item.ID).Find(&old).Error; err != nil {
		return err
	}
	// Never silently assign already posted quantities to an invented historical delivery requirement.
	if len(old) == 0 {
		for _, table := range []string{"production_completions", "order_business_records"} {
			var n int64
			q := tx.Table(table).Where("corp_id=? AND order_item_id=?", item.CorpID, item.ID)
			if table == "order_business_records" {
				q = q.Where("kind='delivery'")
			}
			if err := q.Count(&n).Error; err != nil {
				return err
			}
			if n > 0 {
				return errors.New("该零件已有历史完工/交货台账，不能自动拆分归属；原流程保留，请在未报工零件或新订单设置交货批次")
			}
		}
	}
	ordered, production := int64(0), int64(0)
	names := map[string]bool{}
	oldIDs := map[int64]bool{}
	for _, b := range old {
		oldIDs[b.ID] = true
	}
	kept := map[int64]bool{}
	rows := []models.DeliveryBatch{}
	for _, in := range inputs {
		in.BatchNo = strings.TrimSpace(in.BatchNo)
		if in.BatchNo == "" || !Text(in.BatchNo, 40) || names[in.BatchNo] || !Text(in.Notes, 500) {
			return errors.New("批次名称不能为空、重复或超过40字，备注最多500字")
		}
		names[in.BatchNo] = true
		if in.ProductionQuantity == 0 {
			in.ProductionQuantity = in.Quantity
		}
		if in.Quantity <= 0 || in.ProductionQuantity < in.Quantity || in.ProductionQuantity > 2147483647 {
			return errors.New("批次数量须为正整数，投产数量不能小于要求交货数量")
		}
		d, err := Date(in.DueDate, false)
		if err != nil {
			return err
		}
		if item.DueDate == nil || in.DueDate > item.DueDate.Format("2006-01-02") || item.StartDate != nil && in.DueDate < item.StartDate.Format("2006-01-02") {
			return errors.New("批次交期须在下单日期与零件总交期之间")
		}
		if in.ID != 0 && !oldIDs[in.ID] {
			return errors.New("批次不属于当前零件")
		}
		if kept[in.ID] && in.ID != 0 {
			return errors.New("批次ID重复")
		}
		kept[in.ID] = true
		ordered += int64(in.Quantity)
		production += int64(in.ProductionQuantity)
		rows = append(rows, models.DeliveryBatch{ID: in.ID, CorpID: item.CorpID, OrderID: item.OrderID, OrderItemID: item.ID, BatchNo: in.BatchNo, Quantity: in.Quantity, ProductionQuantity: in.ProductionQuantity, DueDate: d, Notes: in.Notes, CreatedBy: actor, CreatedAt: time.Now()})
	}
	if ordered != int64(item.Quantity) || production != int64(item.ProductionQuantity) {
		return errors.New("各批要求交货数量合计须等于订单数量，各批投产数量合计须等于零件投产数量")
	}
	// Editing the plan after posting keeps identities, quantities and posted batch dates intact.
	for _, b := range old {
		var n int64
		for _, table := range []string{"production_completions", "order_business_records", "outsourcing_dispatches"} {
			if err := tx.Table(table).Where("corp_id=? AND delivery_batch_id=?", item.CorpID, b.ID).Count(&n).Error; err != nil {
				return err
			}
			if n > 0 {
				var next *models.DeliveryBatch
				for i := range rows {
					if rows[i].ID == b.ID {
						next = &rows[i]
					}
				}
				if next == nil || next.Quantity != b.Quantity || next.ProductionQuantity != b.ProductionQuantity || next.DueDate.Format("2006-01-02") != b.DueDate.Format("2006-01-02") {
					return errors.New("已有完工/交货/外协记录的批次不能删除或改变数量和交期，请保留原记录")
				}
			}
		}
	}
	if err := Activate(tx, item, baseline, actor); err != nil {
		return err
	}
	for _, b := range old {
		if !kept[b.ID] {
			if err := tx.Delete(&b).Error; err != nil {
				return err
			}
		}
	}
	for i := range rows {
		b := &rows[i]
		if b.ID == 0 {
			if err := tx.Create(b).Error; err != nil {
				return err
			}
		} else {
			if err := tx.Model(b).Updates(map[string]interface{}{"batch_no": b.BatchNo, "quantity": b.Quantity, "production_quantity": b.ProductionQuantity, "due_date": b.DueDate, "notes": b.Notes}).Error; err != nil {
				return err
			}
		}
	}
	return nil
}
