package models

import (
	"time"
)

type AccountType string

const (
	AccountTypeAdmin AccountType = "admin"
	AccountTypeUser  AccountType = "user"
)

type Role struct {
	ID          int64       `gorm:"primaryKey;autoIncrement" json:"id"`
	CorpID      int64       `gorm:"column:corp_id;default:0" json:"corp_id"`
	Name        string      `gorm:"column:name;not null" json:"name"`
	AccountType AccountType `gorm:"column:account_type;type:varchar(20);not null" json:"account_type"`
	Status      string      `gorm:"column:status;default:active" json:"status"`
	CreatedAt   time.Time   `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt   time.Time   `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

func (Role) TableName() string { return "roles" }
