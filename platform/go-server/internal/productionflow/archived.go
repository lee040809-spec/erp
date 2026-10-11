package productionflow

import (
	"encoding/json"
	"fmt"
	"gorm.io/gorm"
	"machining-erp/internal/models"
)

type archivedLoss struct {
	ProcessID int64
	BeforeID  int64
	Quantity  int
}

// Stable process IDs anchor retired scrap to the next surviving operation; a zero anchor is terminal.
// Follow archived anchors when that successor is later retired, without rewriting audit history.
func archivedLosses(tx *gorm.DB, corp, item int64) ([]archivedLoss, error) {
	var ps []models.OrderProcess
	if err := tx.Unscoped().Select("id").Where("corp_id=? AND order_item_id=? AND deleted_at IS NOT NULL", corp, item).Find(&ps).Error; err != nil {
		return nil, err
	}
	if len(ps) == 0 {
		return nil, nil
	}
	ids := []int64{}
	archived := map[int64]bool{}
	for _, p := range ps {
		ids = append(ids, p.ID)
		archived[p.ID] = true
	}
	var auditIDs []int64
	if err := tx.Model(&models.ProcessTimeEvent{}).Where("corp_id=? AND order_item_id=? AND process_id IN ? AND source='route'", corp, item, ids).Group("process_id").Pluck("MAX(id)", &auditIDs).Error; err != nil {
		return nil, err
	}
	var events []models.ProcessTimeEvent
	if len(auditIDs) > 0 {
		if err := tx.Where("corp_id=? AND id IN ?", corp, auditIDs).Find(&events).Error; err != nil {
			return nil, err
		}
	}
	anchors := map[int64]int64{}
	for _, e := range events {
		var node struct {
			BeforeID *int64 `json:"retired_before_id"`
		}
		if err := json.Unmarshal([]byte(e.AfterNodes), &node); err != nil {
			return nil, fmt.Errorf("历史工序调整记录无效")
		}
		if node.BeforeID != nil {
			anchors[e.ProcessID] = *node.BeforeID
		}
	}
	var quantities []models.ProcessQuantity
	if err := tx.Where("corp_id=? AND order_item_id=? AND order_process_id IN ?", corp, item, ids).Find(&quantities).Error; err != nil {
		return nil, err
	}
	scrap := map[int64]int{}
	for _, q := range quantities {
		scrap[q.OrderProcessID] = q.ScrapQuantity
	}
	losses := []archivedLoss{}
	for _, p := range ps {
		anchor := anchors[p.ID]
		visited := map[int64]bool{p.ID: true}
		for archived[anchor] {
			if visited[anchor] {
				return nil, fmt.Errorf("历史工序衔接形成循环")
			}
			visited[anchor] = true
			anchor = anchors[anchor]
		}
		losses = append(losses, archivedLoss{ProcessID: p.ID, BeforeID: anchor, Quantity: scrap[p.ID]})
	}
	return losses, nil
}
func lossBefore(losses []archivedLoss, process int64) int {
	n := 0
	for _, l := range losses {
		if l.BeforeID == process {
			n += l.Quantity
		}
	}
	return n
}
func terminalLoss(losses []archivedLoss) int { return lossBefore(losses, 0) }
func archivedProcessNames(tx *gorm.DB, corp, item int64) (map[int64]string, error) {
	var ps []models.OrderProcess
	err := tx.Unscoped().Select("id,name").Where("corp_id=? AND order_item_id=? AND deleted_at IS NOT NULL", corp, item).Order("deleted_at DESC,id DESC").Limit(100).Find(&ps).Error
	names := map[int64]string{}
	for _, p := range ps {
		names[p.ID] = p.Name
	}
	return names, err
}
