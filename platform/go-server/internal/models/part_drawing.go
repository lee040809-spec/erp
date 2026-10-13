package models

import "time"

// PartDrawing stores metadata only. Files are served through authenticated routes.
type PartDrawing struct {
	ID              int64     `gorm:"primaryKey" json:"id"`
	CorpID          int64     `json:"-"`
	OrderItemID     *int64    `json:"order_item_id"`
	SourceDrawingID *int64    `json:"source_drawing_id,omitempty"`
	ReplacesID      *int64    `json:"replaces_id,omitempty"`
	StorageKey      string    `json:"-"`
	SHA256          string    `json:"sha256"`
	FileName        string    `json:"file_name"`
	MimeType        string    `json:"mime_type"`
	Size            int64     `json:"size"`
	DrawingNumber   string    `json:"drawing_number"`
	Version         string    `json:"version"`
	Notes           string    `json:"notes"`
	LegacySource    bool      `json:"legacy_source"`
	IsCurrent       bool      `json:"is_current"`
	UploadedBy      int64     `json:"uploaded_by"`
	CreatedAt       time.Time `json:"created_at"`
}

func (PartDrawing) TableName() string { return "part_drawings" }

type DrawingEvent struct {
	ID          int64     `gorm:"primaryKey" json:"id"`
	CorpID      int64     `json:"-"`
	OrderItemID int64     `json:"order_item_id"`
	DrawingID   int64     `json:"drawing_id"`
	Action      string    `json:"action"`
	Reason      string    `json:"reason"`
	ActorID     int64     `json:"actor_id"`
	CreatedAt   time.Time `json:"created_at"`
}

func (DrawingEvent) TableName() string { return "drawing_events" }
