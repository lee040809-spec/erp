package models

import (
	"time"
)

type Contact struct {
	ID         int64  `gorm:"primaryKey;autoIncrement" json:"id,omitempty"`
	CorpID     int64  `gorm:"column:corp_id;default:0" json:"corp_id,omitempty"`
	CustomerID int64  `gorm:"column:customer_id;index" json:"customer_id,omitempty"`
	Name       string `gorm:"column:name" json:"name"`
	Contact    string `gorm:"column:contact" json:"contact"`
}

func (Contact) TableName() string { return "contacts" }

type Customer struct {
	ID        int64     `gorm:"primaryKey;autoIncrement" json:"id"`
	CorpID    int64     `gorm:"column:corp_id;default:0" json:"corp_id"`
	Name      string    `gorm:"column:name;size:100" json:"name"`
	ShortName string    `gorm:"column:short_name;size:50" json:"short_name,omitempty"`
	Contacts  []Contact `gorm:"foreignKey:CustomerID" json:"contacts,omitempty"`
	CreatedAt time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
}

func (Customer) TableName() string { return "customers" }
