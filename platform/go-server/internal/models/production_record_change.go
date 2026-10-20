package models

import "time"

// Append-only audit; corrections retain the original completion/delivery IDs and their links.
type ProductionRecordChange struct {
	ID             int64     `gorm:"primaryKey" json:"id"`
	CorpID         int64     `json:"-"`
	OrderItemID    int64     `json:"-"`
	Kind           string    `json:"kind"`
	RecordID       int64     `json:"record_id"`
	RequestID      string    `json:"-"`
	RequestHash    string    `json:"-"`
	Reason         string    `json:"reason"`
	BeforeJSON     string    `gorm:"column:before_json" json:"-"`
	AfterJSON      string    `gorm:"column:after_json" json:"-"`
	BeforeQuantity string    `json:"before_quantity"`
	AfterQuantity  string    `json:"after_quantity"`
	BeforeDate     string    `json:"before_date"`
	AfterDate      string    `json:"after_date"`
	CreatedBy      int64     `json:"created_by"`
	CreatedAt      time.Time `json:"created_at"`
}
