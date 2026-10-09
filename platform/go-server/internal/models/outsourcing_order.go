package models

import "time"

type OutsourcingSupplier struct {
	ID           int64     `json:"id" gorm:"primaryKey"`
	CorpID       int64     `json:"corp_id"`
	Name         string    `json:"name"`
	ContactName  string    `json:"contact_name"`
	Phone        string    `json:"phone"`
	Address      string    `json:"address"`
	Capabilities string    `json:"capabilities"`
	Notes        string    `json:"notes"`
	Active       bool      `json:"active"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

func (OutsourcingSupplier) TableName() string { return "outsourcing_suppliers" }

// A purchase order reserves capacity; it does not report physical dispatch or production.
type OutsourcingOrder struct {
	EditToken          string                 `json:"edit_token" gorm:"-"`
	ID                 int64                  `json:"id" gorm:"primaryKey"`
	CorpID             int64                  `json:"corp_id"`
	OrderID            int64                  `json:"order_id"`
	OrderItemID        int64                  `json:"order_item_id"`
	SupplierID         int64                  `json:"supplier_id"`
	SupplierName       string                 `json:"supplier_name"`
	OrderNo            string                 `json:"order_no"`
	Scope              string                 `json:"scope"` // full (one shipment) or selected (dispatch each process)
	DeliveryBatchID    *int64                 `json:"delivery_batch_id"`
	UnitPrice          string                 `json:"unit_price"` // full route price per part, never repeated per process
	OrderDate          time.Time              `json:"order_date"`
	ExpectedReturnDate time.Time              `json:"expected_return_date"`
	Status             string                 `json:"status"`
	Notes              string                 `json:"notes"`
	RequestID          string                 `json:"request_id"`
	CreatedBy          int64                  `json:"created_by"`
	CreatedAt          time.Time              `json:"created_at"`
	Lines              []OutsourcingOrderLine `json:"lines" gorm:"-"`
}

func (OutsourcingOrder) TableName() string { return "outsourcing_orders" }

type OutsourcingOrderLine struct {
	ID                 int64  `json:"id" gorm:"primaryKey"`
	CorpID             int64  `json:"corp_id"`
	OutsourcingOrderID int64  `json:"outsourcing_order_id"`
	OrderItemID        int64  `json:"order_item_id"`
	OrderProcessID     int64  `json:"order_process_id"`
	ProcessName        string `json:"process_name"`
	Sequence           int    `json:"sequence"`
	Quantity           int    `json:"quantity"`
	DispatchedQuantity int    `json:"dispatched_quantity"`
	StartedQuantity    int    `json:"started_quantity"`
	UnitPrice          string `json:"unit_price"`
}

func (OutsourcingOrderLine) TableName() string { return "outsourcing_order_lines" }

// Snapshot of processes actually covered by one physical full-route dispatch.
type OutsourcingDispatchProcess struct {
	ID             int64 `json:"id" gorm:"primaryKey"`
	CorpID         int64 `json:"corp_id"`
	DispatchID     int64 `json:"dispatch_id"`
	OrderProcessID int64 `json:"order_process_id"`
	Sequence       int   `json:"sequence"`
}

func (OutsourcingDispatchProcess) TableName() string { return "outsourcing_dispatch_processes" }

// Changes are retained even when an unused order or a mistaken dispatch is removed from the active list.
type OutsourcingChange struct {
	ID         int64     `json:"id" gorm:"primaryKey"`
	CorpID     int64     `json:"corp_id"`
	EntityType string    `json:"entity_type"`
	EntityID   int64     `json:"entity_id"`
	Action     string    `json:"action"`
	Reason     string    `json:"reason"`
	BeforeJSON string    `json:"before_json"`
	AfterJSON  string    `json:"after_json"`
	CreatedBy  int64     `json:"created_by"`
	CreatedAt  time.Time `json:"created_at"`
}

func (OutsourcingChange) TableName() string { return "outsourcing_changes" }
