package repository

import (
	"errors"
	"gorm.io/gorm"
	"machining-erp/internal/models"
	"machining-erp/internal/productionflow"
	"strings"
	"unicode/utf8"
)

type RouteChangeOptions struct {
	AllowProcessedChanges bool
	Reason                string
}

// Editing a route never accepts client status, actual timestamps or prices.
func (r *OrderRepository) UpdateRouteForCorp(corp, itemID, actor int64, incoming []models.OrderProcess, plans map[int64]bool, options ...RouteChangeOptions) error {
	var change RouteChangeOptions
	if len(options) > 0 {
		change = options[0]
	}
	change.Reason = strings.TrimSpace(change.Reason)
	if utf8.RuneCountInString(change.Reason) > 500 || change.AllowProcessedChanges && change.Reason == "" {
		return errors.New("调整已加工工序须填写原因，最多500个字")
	}
	return r.db.Transaction(func(tx *gorm.DB) error {
		item, err := productionflow.LockItem(tx, corp, itemID)
		if err != nil {
			return err
		}
		var old []models.OrderProcess
		if err = tx.Where("corp_id=? AND order_item_id=?", corp, itemID).Find(&old).Error; err != nil {
			return err
		}
		byID := map[int64]models.OrderProcess{}
		for _, p := range old {
			byID[p.ID] = p
		}
		for i := range incoming {
			p := &incoming[i]
			if p.ID != 0 {
				previous, ok := byID[p.ID]
				if !ok {
					return ErrOrderScope
				}
				p.Status, p.OutsourcingFee = previous.Status, previous.OutsourcingFee
				p.ActualStartAt, p.ActualEndAt = previous.ActualStartAt, previous.ActualEndAt
				if !plans[p.ID] {
					p.PlannedStartAt, p.PlannedEndAt = previous.PlannedStartAt, previous.PlannedEndAt
				}
			} else {
				p.Status = models.ProcessStatusPending
				p.OutsourcingFee = 0
				p.ActualStartAt = nil
				p.ActualEndAt = nil
			}
		}
		if err = handleProcesses(tx, itemID, corp, incoming, actor, plans, change); err != nil {
			return err
		}
		// Reuse the existing qualified-output and logistics state calculation.
		var current []models.OrderProcess
		if err = tx.Where("corp_id=? AND order_item_id=?", corp, itemID).Find(&current).Error; err != nil {
			return err
		}
		if len(current) > 0 {
			_, _, err = NewOrderRepository(tx).UpdateProcessForCorp(corp, itemID, current[0].ID, nil, nil, nil, actor)
			return err
		}
		return tx.Model(&item).Update("status", models.OrderStatusPending).Error
	})
}

func routeChangeReason(change RouteChangeOptions) string {
	if change.Reason != "" {
		return change.Reason
	}
	return "工艺调整；保留工序编号及已有实际生产记录"
}
