package drawings

import (
	"errors"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"machining-erp/internal/models"
	"strings"
	"time"
)

func Event(tx *gorm.DB, d models.PartDrawing, actor int64, action, reason string) error {
	if d.OrderItemID == nil {
		return nil
	}
	return tx.Create(&models.DrawingEvent{CorpID: d.CorpID, OrderItemID: *d.OrderItemID, DrawingID: d.ID, Action: action, Reason: reason, ActorID: actor, CreatedAt: time.Now()}).Error
}
func Item(tx *gorm.DB, corp, id int64) (models.OrderItem, error) {
	var item models.OrderItem
	e := tx.Select("id,corp_id,order_id,part_name,part_number,status").Clauses(clause.Locking{Strength: "UPDATE"}).Where("id=? AND corp_id=?", id, corp).Take(&item).Error
	return item, e
}
func ChangeReason(tx *gorm.DB, item models.OrderItem, reason string) error {
	if len([]rune(reason)) > 500 {
		return errors.New("变更说明最多500字")
	}
	var count int64
	if e := tx.Model(&models.OrderProcess{}).Where("corp_id=? AND order_item_id=? AND (planned_start_at IS NOT NULL OR planned_end_at IS NOT NULL OR actual_start_at IS NOT NULL OR actual_end_at IS NOT NULL OR status<>?)", item.CorpID, item.ID, "pending").Count(&count).Error; e != nil {
		return e
	}
	if (item.Status != "pending" || count > 0) && strings.TrimSpace(reason) == "" {
		return errors.New("零件已安排或投入生产，请填写图纸变更说明，并核对工艺")
	}
	return nil
}

// Bind runs inside the order transaction, so an invalid drawing also rolls back the order.
// Existing links are idempotent; omitted drawing_ids never removes a drawing.
func Bind(tx *gorm.DB, item models.OrderItem, ids []int64, actor int64, reason string) error {
	if len(ids) == 0 {
		return nil
	}
	if len(ids) > 20 {
		return errors.New("每次最多关联20份图纸")
	}
	locked, e := Item(tx, item.CorpID, item.ID)
	if e != nil {
		return e
	}
	seen := map[int64]bool{}
	for _, id := range ids {
		if seen[id] {
			continue
		}
		seen[id] = true
		var d models.PartDrawing
		if e = tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id=? AND corp_id=?", id, item.CorpID).Take(&d).Error; e != nil {
			return errors.New("图纸不存在或不属于当前公司")
		}
		if d.OrderItemID != nil {
			if *d.OrderItemID == item.ID {
				continue
			}
			return errors.New("图纸已关联其他零件，请使用历史图纸引用")
		}
		if d.UploadedBy != actor {
			return errors.New("不能使用其他人员未保存的图纸")
		}
		if e = ChangeReason(tx, locked, reason); e != nil {
			return e
		}
		if d.SourceDrawingID != nil {
			var source models.PartDrawing
			if e = tx.Where("id=? AND corp_id=?", *d.SourceDrawingID, item.CorpID).Take(&source).Error; e != nil {
				return e
			}
			if source.OrderItemID == nil {
				return errors.New("历史图纸未关联零件")
			}
			var original models.OrderItem
			if e = tx.Select("id,order_id,part_number").Where("id=? AND corp_id=?", *source.OrderItemID, item.CorpID).Take(&original).Error; e != nil {
				return e
			}
			var srcOrder, dstOrder models.Order
			if e = tx.Select("id,customer_id").Where("id=? AND corp_id=?", original.OrderID, item.CorpID).Take(&srcOrder).Error; e != nil {
				return e
			}
			if e = tx.Select("id,customer_id").Where("id=? AND corp_id=?", locked.OrderID, item.CorpID).Take(&dstOrder).Error; e != nil {
				return e
			}
			if srcOrder.CustomerID == nil || dstOrder.CustomerID == nil || *srcOrder.CustomerID != *dstOrder.CustomerID {
				return errors.New("引用历史图纸必须属于相同客户，请核对客户")
			}
			if original.PartNumber != nil && locked.PartNumber != nil && strings.TrimSpace(*original.PartNumber) != "" && strings.TrimSpace(*locked.PartNumber) != "" && *original.PartNumber != *locked.PartNumber {
				return errors.New("历史图纸的零件图号与当前零件不一致")
			}
		}
		if d.ReplacesID != nil {
			var old models.PartDrawing
			if e = tx.Where("id=? AND corp_id=? AND order_item_id=?", *d.ReplacesID, item.CorpID, item.ID).Take(&old).Error; e != nil {
				return errors.New("被替换的图纸不属于此零件")
			}
			if e = tx.Model(&old).Update("is_current", false).Error; e != nil {
				return e
			}
			if e = Event(tx, old, actor, "replaced", reason); e != nil {
				return e
			}
		}
		var total int64
		if e = tx.Model(&models.PartDrawing{}).Where("corp_id=? AND order_item_id=?", item.CorpID, item.ID).Count(&total).Error; e != nil {
			return e
		}
		if total >= 200 {
			return errors.New("该零件图纸版本过多，请联系管理员归档")
		}
		if e = tx.Model(&d).Updates(map[string]interface{}{"order_item_id": item.ID, "is_current": true}).Error; e != nil {
			return e
		}
		d.OrderItemID = &item.ID
		if e = Event(tx, d, actor, "linked", reason); e != nil {
			return e
		}
	}
	return nil
}
