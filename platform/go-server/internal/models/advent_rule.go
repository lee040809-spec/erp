package models

import (
	"time"
)

type AdventRule struct {
	ID           int64     `gorm:"primaryKey;autoIncrement" json:"id"`
	CorpID       int64     `gorm:"column:corp_id;default:0" json:"corp_id"`
	Name         string    `gorm:"column:name" json:"name"`
	Description  string    `gorm:"column:description" json:"description,omitempty"`
	Formula      string    `gorm:"column:formula" json:"formula"`
	TargetStatus string    `gorm:"column:target_status" json:"target_status"`
	ScopeType    string    `gorm:"column:scopeType" json:"scopeType"`
	RuleType     string    `gorm:"column:ruleType" json:"ruleType"`
	CreatedAt    time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
}

func (AdventRule) TableName() string { return "advent_rules" }

type Reconciliation struct {
	Month           string  `gorm:"-" json:"month"`
	TotalAmount     float64 `gorm:"-" json:"total_amount"`
	OrderCount      int     `gorm:"-" json:"order_count"`
	DeliveredAmount float64 `gorm:"-" json:"delivered_amount"`
}
