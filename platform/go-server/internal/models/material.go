package models

import (
	"time"
)

type Material struct {
	ID       int64   `gorm:"primaryKey;autoIncrement" json:"id"`
	CorpID   int64   `gorm:"column:corp_id;default:0" json:"corp_id"`
	Name     string  `gorm:"column:name" json:"name"`
	Spec     string  `gorm:"column:spec" json:"spec,omitempty"`
	Quantity float64 `gorm:"column:quantity" json:"quantity"`
	Unit     string  `gorm:"column:unit" json:"unit"`
}

func (Material) TableName() string { return "materials" }

type Remnant struct {
	ID           int64     `gorm:"primaryKey;autoIncrement" json:"id"`
	CorpID       int64     `gorm:"column:corp_id;default:0" json:"corp_id"`
	MaterialID   int64     `gorm:"column:material_id;index" json:"material_id"`
	MaterialName string    `gorm:"-" json:"material_name,omitempty"`
	Dimensions   string    `gorm:"column:dimensions" json:"dimensions"`
	PhotoData    string    `gorm:"column:photo_data" json:"photo_data,omitempty"`
	Notes        string    `gorm:"column:notes" json:"notes,omitempty"`
	CreatedAt    time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
}

func (Remnant) TableName() string { return "remnants" }
