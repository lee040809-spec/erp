package models

import "time"

// Snapshots contain status and time nodes only, never monetary or drawing fields.
type ProcessTimeEvent struct {
	ID          int64     `gorm:"primaryKey" json:"id"`
	CorpID      int64     `json:"-"`
	OrderItemID int64     `json:"order_item_id"`
	ProcessID   int64     `json:"process_id"`
	BeforeNodes string    `json:"before_nodes"`
	AfterNodes  string    `json:"after_nodes"`
	Reason      string    `json:"reason"`
	Source      string    `json:"source"`
	CreatedBy   int64     `json:"created_by"`
	CreatedAt   time.Time `json:"created_at"`
}

func (ProcessTimeEvent) TableName() string { return "process_time_events" }
