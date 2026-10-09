package models

import (
	"time"
)

type Company struct {
	ID        int64     `gorm:"primaryKey;autoIncrement" json:"id"`
	Name      string    `gorm:"column:name;not null" json:"name"`
	ShortName *string   `gorm:"column:short_name" json:"short_name,omitempty"`
	Address   *string   `gorm:"column:address" json:"address,omitempty"`
	Phone     *string   `gorm:"column:phone" json:"phone,omitempty"`
	Status    string    `gorm:"column:status;default:active" json:"status"`
	CreatedAt time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt time.Time `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

func (Company) TableName() string { return "companies" }
