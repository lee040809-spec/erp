package repository

import (
	"errors"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"machining-erp/internal/models"
	"machining-erp/internal/productionflow"
	"machining-erp/pkg/utils"
	"time"
)

var (
	ErrOrderScope      = errors.New("订单、零件或工序不属于当前公司和订单")
	ErrPostedOrder     = errors.New("订单已有完工或费用台账，不能直接删除；请通过核算调整流程处理")
	ErrPostedQuantity  = errors.New("投产数量须不少于订单数量和已确认合格数加报废数，订单数量不能小于已计产值数量")
	ErrCompletionState = errors.New("合格完工数量未确认或状态与已确认数量不一致，请通过合格完工报工处理")
	ErrDeliveryState   = errors.New("已交付零件不能直接回退状态，请通过交付调整流程处理")
	ErrCompletionDate  = errors.New("实际完工日期无效或晚于今天")
	ErrPostedBasis     = errors.New("已有完工台账的零件不能直接修改供料方式、币种或含税口径")
)

func protectPostedItem(tx *gorm.DB, corpID, itemID int64) error {
	if e := productionflow.ProtectItem(tx, corpID, itemID); e != nil {
		return e
	}
	var count int64
	if err := tx.Table("order_business_records").Where("corp_id=? AND order_item_id=?", corpID, itemID).Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return ErrPostedOrder
	}
	if err := tx.Table("production_completions").Where("corp_id=? AND order_item_id=?", corpID, itemID).Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return ErrPostedOrder
	}
	if err := tx.Table("cost_entries").Where("corp_id=? AND order_item_id=?", corpID, itemID).Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return ErrPostedOrder
	}
	if err := tx.Model(&models.OrderProcess{}).Where("corp_id=? AND order_item_id=? AND (actual_start_at IS NOT NULL OR actual_end_at IS NOT NULL)", corpID, itemID).Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return ErrTimedProcess
	}
	return nil
}

func (r *OrderRepository) UpdateProcessForCorp(corpID, itemID, processID int64, status *string, outsourced *bool, fee *float64, actors ...int64) (string, string, error) {
	var itemStatus, orderStatus string
	err := r.db.Transaction(func(tx *gorm.DB) error {
		var item models.OrderItem
		if err := tx.Where("id=? AND corp_id=?", itemID, corpID).Take(&item).Error; err != nil {
			return err
		}
		var order models.Order
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id=? AND corp_id=?", item.OrderID, corpID).Take(&order).Error; err != nil {
			return err
		}
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id=? AND corp_id=? AND order_id=?", itemID, corpID, order.ID).Take(&item).Error; err != nil {
			return err
		}
		var process models.OrderProcess
		if err := tx.Where("id=? AND order_item_id=? AND corp_id=?", processID, itemID, corpID).Take(&process).Error; err != nil {
			return err
		}
		on, e := productionflow.Enabled(tx, corpID, itemID)
		if e != nil {
			return e
		}
		reserved, e := productionflow.OutsourcingReserved(tx, corpID, itemID, processID)
		if e != nil {
			return e
		}
		if reserved > 0 && ((status != nil && *status != string(process.Status)) || (outsourced != nil && *outsourced != process.IsOutsourced)) {
			return errors.New("已有外协下单预留，请通过工序数量和外协流转登记，不可直接修改工序状态或外协标记")
		}
		if on && ((status != nil && *status != string(process.Status)) || (outsourced != nil && *outsourced != process.IsOutsourced)) {
			return errors.New("已启用工序数量跟踪，请通过分批生产窗口报工，状态由数量自动更新")
		}
		updates := map[string]interface{}{}
		if status != nil && process.Name == "送货" {
			return ErrDeliveryState
		}
		if status != nil {
			var actor int64
			if len(actors) > 0 {
				actor = actors[0]
			}
			timing, err := ApplyProcessTimeTransition(tx, process, models.ProcessStatus(*status), actor)
			if err != nil {
				return err
			}
			for key, value := range timing {
				updates[key] = value
			}
			updates["status"] = *status
		}
		if outsourced != nil {
			updates["is_outsourced"] = *outsourced
		}
		if fee != nil {
			updates["outsourcing_fee"] = *fee
		}
		if len(updates) > 0 {
			if err := tx.Model(&process).Updates(updates).Error; err != nil {
				return err
			}
		}
		var statuses []string
		if err := tx.Model(&models.OrderProcess{}).Where("order_item_id=? AND corp_id=?", itemID, corpID).Pluck("status", &statuses).Error; err != nil {
			return err
		}
		itemStatus = utils.CalculateStatus(statuses)
		var confirmed float64
		if err := tx.Table("production_completions").Where("corp_id=? AND order_item_id=? AND status='confirmed'", corpID, itemID).Select("COALESCE(SUM(qualified_quantity),0)").Scan(&confirmed).Error; err != nil {
			return err
		}
		if confirmed > 0 && confirmed >= float64(item.Quantity) {
			itemStatus = "completed"
		} else if itemStatus == "completed" {
			itemStatus = "processing"
		}
		// Logistics delivery remains independent from reopening a production process.
		if item.Status == models.OrderStatusDelivered {
			itemStatus = "delivered"
		}
		if err := tx.Model(&item).Update("status", itemStatus).Error; err != nil {
			return err
		}
		statuses = nil
		if err := tx.Model(&models.OrderItem{}).Where("order_id=? AND corp_id=?", order.ID, corpID).Pluck("status", &statuses).Error; err != nil {
			return err
		}
		orderStatus = utils.CalculateStatus(statuses)
		return tx.Model(&order).Update("status", orderStatus).Error
	})
	return itemStatus, orderStatus, err
}

func (r *OrderRepository) GetByIDForCorp(id, corpID int64) (*models.Order, error) {
	var order models.Order
	err := r.db.Where("id=? AND corp_id=?", id, corpID).Take(&order).Error
	return &order, err
}

// UpdateItemForCorp keeps logistics edits atomic and protects finance dates.
func (r *OrderRepository) UpdateItemForCorp(corpID, itemID int64, status *string, date *string) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		var item models.OrderItem
		if err := tx.Where("id=? AND corp_id=?", itemID, corpID).Take(&item).Error; err != nil {
			return err
		}
		var order models.Order
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id=? AND corp_id=?", item.OrderID, corpID).Take(&order).Error; err != nil {
			return err
		}
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id=? AND corp_id=?", itemID, corpID).Take(&item).Error; err != nil {
			return err
		}
		updates := map[string]interface{}{}
		if date != nil && *date != "" {
			if _, err := time.Parse("2006-01-02", *date); err != nil || *date > time.Now().Format("2006-01-02") {
				return ErrCompletionDate
			}
		}
		if date != nil {
			var count int64
			if err := tx.Table("production_completions").Where("corp_id=? AND order_item_id=?", corpID, itemID).Count(&count).Error; err != nil {
				return err
			}
			if count > 0 {
				return ErrPostedBasis
			}
			if *date == "" {
				updates["completion_date"] = nil
			} else {
				updates["completion_date"] = *date
			}
		}
		if status != nil && *status == "delivered" && string(item.Status) != "delivered" {
			return ErrDeliveryState
		}
		if status != nil && *status != string(item.Status) {
			if item.Status == models.OrderStatusDelivered {
				return ErrDeliveryState
			}
			var confirmed float64
			if err := tx.Table("production_completions").Where("corp_id=? AND order_item_id=? AND status='confirmed'", corpID, itemID).Select("COALESCE(SUM(qualified_quantity),0)").Scan(&confirmed).Error; err != nil {
				return err
			}
			full := confirmed > 0 && confirmed >= float64(item.Quantity)
			if ((*status == "completed" || *status == "delivered") && !full) || (full && (*status == "pending" || *status == "processing")) {
				return ErrCompletionState
			}
			updates["status"] = *status
		}
		if len(updates) > 0 {
			if err := tx.Model(&item).Updates(updates).Error; err != nil {
				return err
			}
		}
		var statuses []string
		if err := tx.Model(&models.OrderItem{}).Where("order_id=? AND corp_id=?", order.ID, corpID).Pluck("status", &statuses).Error; err != nil {
			return err
		}
		return tx.Model(&order).Update("status", utils.CalculateStatus(statuses)).Error
	})
}
