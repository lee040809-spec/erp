package repository

import (
	"encoding/json"
	"errors"
	"gorm.io/gorm"
	"machining-erp/internal/models"
	"time"
)

var ErrTimedProcess = errors.New("工序已有实际时间记录，不能直接回退或删除；请在工序时间节点中填写原因更正")

func processNodes(p models.OrderProcess) string {
	b, _ := json.Marshal(struct {
		Name         string               `json:"name"`
		SortOrder    int                  `json:"sort_order"`
		Outsourced   bool                 `json:"is_outsourced"`
		Status       models.ProcessStatus `json:"status"`
		PlannedStart *time.Time           `json:"planned_start_at"`
		PlannedEnd   *time.Time           `json:"planned_end_at"`
		ActualStart  *time.Time           `json:"actual_start_at"`
		ActualEnd    *time.Time           `json:"actual_end_at"`
	}{p.Name, p.SortOrder, p.IsOutsourced, p.Status, p.PlannedStartAt, p.PlannedEndAt, p.ActualStartAt, p.ActualEndAt})
	return string(b)
}

func RecordProcessTimeEvent(tx *gorm.DB, before, after models.OrderProcess, actor int64, source, reason string) error {
	previous, next := processNodes(before), processNodes(after)
	if previous == next {
		return nil
	}
	return tx.Create(&models.ProcessTimeEvent{CorpID: after.CorpID, OrderItemID: after.OrderItemID, ProcessID: after.ID, BeforeNodes: previous, AfterNodes: next, CreatedBy: actor, Source: source, Reason: reason, CreatedAt: time.Now()}).Error
}

// Repeated status requests preserve the original timestamp; pending historical completions keep unknown starts.
func ApplyProcessTimeTransition(tx *gorm.DB, old models.OrderProcess, status models.ProcessStatus, actor int64) (map[string]interface{}, error) {
	fields := map[string]interface{}{}
	if old.Name == "送货" || status == old.Status {
		return fields, nil
	}
	if (old.ActualStartAt != nil || old.ActualEndAt != nil) && (status == models.ProcessStatusPending || old.Status == models.ProcessStatusCompleted) {
		return nil, ErrTimedProcess
	}
	next := old
	next.Status = status
	now := time.Now()
	if status == models.ProcessStatusProcessing && old.ActualStartAt == nil {
		next.ActualStartAt = &now
		fields["actual_start_at"] = now
	}
	if status == models.ProcessStatusCompleted && old.ActualEndAt == nil {
		next.ActualEndAt = &now
		fields["actual_end_at"] = now
	}
	if err := RecordProcessTimeEvent(tx, old, next, actor, "status", "工序状态操作自动记录"); err != nil {
		return nil, err
	}
	return fields, nil
}

func protectTimedProcess(tx *gorm.DB, corp, process int64) error {
	var n int64
	if err := tx.Model(&models.OrderProcess{}).Where("corp_id=? AND id=? AND (actual_start_at IS NOT NULL OR actual_end_at IS NOT NULL)", corp, process).Count(&n).Error; err != nil {
		return err
	}
	if n > 0 {
		return ErrTimedProcess
	}
	return nil
}
