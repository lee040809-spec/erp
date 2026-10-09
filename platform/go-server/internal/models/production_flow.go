package models

import "time"

// Delivery batches share the part's process route. They do not duplicate inventory or output.
type DeliveryBatch struct {
	ID                 int64     `json:"id" gorm:"primaryKey"`
	CorpID             int64     `json:"corp_id"`
	OrderID            int64     `json:"order_id"`
	OrderItemID        int64     `json:"order_item_id"`
	BatchNo            string    `json:"batch_no"`
	Quantity           int       `json:"quantity"`
	ProductionQuantity int       `json:"production_quantity"`
	DueDate            time.Time `json:"due_date"`
	Notes              string    `json:"notes"`
	CreatedBy          int64     `json:"created_by"`
	CreatedAt          time.Time `json:"created_at"`
	QualifiedQuantity  int       `json:"qualified_quantity" gorm:"-"`
	DeliveredQuantity  int       `json:"delivered_quantity" gorm:"-"`
	LastDeliveryDate   string    `json:"last_delivery_date" gorm:"-"`
}

func (DeliveryBatch) TableName() string { return "order_delivery_batches" }

type DeliveryBatchInput struct {
	ID                 int64  `json:"id"`
	BatchNo            string `json:"batch_no"`
	Quantity           int    `json:"quantity"`
	ProductionQuantity int    `json:"production_quantity"`
	DueDate            string `json:"due_date"`
	Notes              string `json:"notes"`
}

type ProcessQuantity struct {
	ID                int64     `json:"id" gorm:"primaryKey"`
	CorpID            int64     `json:"corp_id"`
	OrderItemID       int64     `json:"order_item_id"`
	OrderProcessID    int64     `json:"order_process_id"`
	StartedQuantity   int       `json:"started_quantity"`
	CompletedQuantity int       `json:"completed_quantity"`
	ScrapQuantity     int       `json:"scrap_quantity"`
	ActivatedAt       time.Time `json:"activated_at"`
}

func (ProcessQuantity) TableName() string { return "process_quantity_totals" }

type ProcessQuantityEvent struct {
	ID             int64      `json:"id" gorm:"primaryKey"`
	CorpID         int64      `json:"corp_id"`
	OrderItemID    int64      `json:"order_item_id"`
	OrderProcessID int64      `json:"order_process_id"`
	Kind           string     `json:"kind"`
	Quantity       int        `json:"quantity"`
	BusinessDate   *time.Time `json:"business_date"`
	Notes          string     `json:"notes"`
	SourceType     string     `json:"source_type"`
	RequestID      string     `json:"request_id"`
	CreatedBy      int64      `json:"created_by"`
	CreatedAt      time.Time  `json:"created_at"`
}

func (ProcessQuantityEvent) TableName() string { return "process_quantity_events" }

type ProcessBaseline struct {
	ProcessID int64 `json:"process_id"`
	Started   int   `json:"started_quantity"`
	Completed int   `json:"completed_quantity"`
	Scrap     int   `json:"scrap_quantity"`
}
