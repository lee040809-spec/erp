package models

import (
	"gorm.io/gorm"
	"time"
)

type OrderStatus string

const (
	OrderStatusPending    OrderStatus = "pending"
	OrderStatusProcessing OrderStatus = "processing"
	OrderStatusCompleted  OrderStatus = "completed"
	OrderStatusDelivered  OrderStatus = "delivered"
)

type Priority string

const (
	PriorityLow    Priority = "low"
	PriorityMedium Priority = "medium"
	PriorityHigh   Priority = "high"
)

type Order struct {
	ID                int64       `gorm:"primaryKey;autoIncrement" json:"id"`
	CorpID            int64       `gorm:"column:corp_id;default:0" json:"corp_id"`
	CustomerID        *int64      `gorm:"column:customer_id" json:"customer_id"`
	CustomerName      *string     `gorm:"column:customer_name" json:"customer_name"`
	CustomerShortName *string     `gorm:"column:customer_short_name" json:"customer_short_name"`
	OrderNumber       string      `gorm:"column:order_number" json:"order_number"`
	OrderName         *string     `gorm:"column:order_name" json:"order_name"`
	ContactID         *int64      `gorm:"column:contact_id" json:"contact_id"`
	ContactName       *string     `gorm:"column:contact_name" json:"contact_name"`
	Status            OrderStatus `gorm:"column:status;type:varchar(20);default:pending" json:"status"`
	Priority          Priority    `gorm:"column:priority;type:varchar(20);default:medium" json:"priority"`
	StartDate         *time.Time  `gorm:"column:start_date" json:"start_date"`
	DueDate           *time.Time  `gorm:"column:due_date" json:"due_date"`
	TotalAmount       float64     `gorm:"column:total_amount;default:0" json:"total_amount"`
	Notes             *string     `gorm:"column:notes" json:"notes"`
	CreatedAt         time.Time   `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	Items             []OrderItem `gorm:"foreignKey:OrderID" json:"items"`
}

func (Order) TableName() string { return "orders" }

type OrderItem struct {
	DeliveryPlan        []DeliveryBatchInput `gorm:"-" json:"delivery_plan,omitempty"`
	FlowTracking        bool                 `gorm:"-" json:"flow_tracking"`
	DeliveryBatches     []DeliveryBatch      `gorm:"-" json:"delivery_batches"`
	NextBatchDueDate    string               `gorm:"-" json:"next_batch_due_date"`
	OpenOutsourcing     int                  `gorm:"-" json:"open_outsourcing"`
	ID                  int64                `gorm:"primaryKey;autoIncrement" json:"id"`
	OrderID             int64                `gorm:"column:order_id;index" json:"order_id"`
	CorpID              int64                `gorm:"column:corp_id;default:0" json:"corp_id"`
	PartName            string               `gorm:"column:part_name" json:"part_name"`
	PartNumber          *string              `gorm:"column:part_number" json:"part_number"`
	ProductionQuantity  int                  `gorm:"column:production_quantity;default:0" json:"production_quantity"`
	Quantity            int                  `gorm:"column:quantity" json:"quantity"`
	QualifiedQuantity   float64              `gorm:"-" json:"qualified_quantity"`
	BillableQuantity    float64              `gorm:"-" json:"billable_quantity"`
	ReserveQuantity     float64              `gorm:"-" json:"reserve_quantity"`
	DeliveryGap         float64              `gorm:"-" json:"delivery_gap"`
	DeliveryDate        *time.Time           `gorm:"-" json:"delivery_date,omitempty"`
	ScrapQuantity       int                  `gorm:"column:scrap_quantity;default:0" json:"scrap_quantity"`
	UnitPrice           float64              `gorm:"column:unit_price" json:"unit_price"`
	TotalPrice          float64              `gorm:"column:total_price" json:"total_price"`
	Status              OrderStatus          `gorm:"column:status;type:varchar(20);default:pending" json:"status"`
	HasDrawing          bool                 `gorm:"-" json:"has_drawing"`
	DrawingCount        int                  `gorm:"-" json:"drawing_count"`
	DrawingIDs          []int64              `gorm:"-" json:"drawing_ids,omitempty"`
	DrawingChangeReason string               `gorm:"-" json:"drawing_change_reason,omitempty"`
	DrawingData         *string              `gorm:"column:drawing_data" json:"drawing_data,omitempty"`
	Notes               *string              `gorm:"column:notes" json:"notes,omitempty"`
	CompletionDate      *time.Time           `gorm:"column:completion_date" json:"completion_date,omitempty"`
	StartDate           *time.Time           `gorm:"column:start_date" json:"start_date,omitempty"`
	DueDate             *time.Time           `gorm:"column:due_date" json:"due_date,omitempty"`
	DeliveredQty        *int                 `gorm:"column:delivered_quantity" json:"delivered_quantity,omitempty"`
	ToolCost            *float64             `gorm:"column:tool_cost" json:"tool_cost,omitempty"`
	FixtureCost         *float64             `gorm:"column:fixture_cost" json:"fixture_cost,omitempty"`
	MaterialCost        *float64             `gorm:"column:material_cost" json:"material_cost,omitempty"`
	OtherCost           *float64             `gorm:"column:other_cost" json:"other_cost,omitempty"`
	SupplyMode          string               `gorm:"column:supply_mode;type:varchar(32);default:self_purchased" json:"supply_mode"`
	Currency            string               `gorm:"column:currency;type:char(3);default:CNY" json:"currency"`
	TaxInclusive        bool                 `gorm:"column:tax_inclusive;default:false" json:"tax_inclusive"`
	ItemNotes           *string              `gorm:"column:item_notes" json:"item_notes,omitempty"`
	Processes           []OrderProcess       `gorm:"foreignKey:OrderItemID" json:"processes"`
}

func (OrderItem) TableName() string { return "order_items" }

type ProcessStatus string

const (
	ProcessStatusPending    ProcessStatus = "pending"
	ProcessStatusProcessing ProcessStatus = "processing"
	ProcessStatusCompleted  ProcessStatus = "completed"
)

type OrderProcess struct {
	CuttingSpec     *string        `gorm:"column:cutting_spec;type:varchar(500)" json:"cutting_spec"`
	RetiredBeforeID *int64         `gorm:"-" json:"-"`
	DeletedAt       gorm.DeletedAt `gorm:"column:deleted_at" json:"deleted_at"`
	ID              int64          `gorm:"primaryKey;autoIncrement" json:"id"`
	OrderItemID     int64          `gorm:"column:order_item_id;index" json:"order_item_id"`
	CorpID          int64          `gorm:"column:corp_id;default:0" json:"corp_id"`
	Name            string         `gorm:"column:name" json:"name"`
	IsOutsourced    bool           `gorm:"column:is_outsourced;default:false" json:"is_outsourced"`
	OutsourcingFee  float64        `gorm:"column:outsourcing_fee;default:0" json:"outsourcing_fee"`
	Status          ProcessStatus  `gorm:"column:status;type:varchar(20);default:pending" json:"status"`
	SortOrder       int            `gorm:"column:sort_order;default:0" json:"sort_order"`
	PlannedStartAt  *time.Time     `gorm:"column:planned_start_at" json:"planned_start_at"`
	PlannedEndAt    *time.Time     `gorm:"column:planned_end_at" json:"planned_end_at"`
	ActualStartAt   *time.Time     `gorm:"column:actual_start_at" json:"actual_start_at"`
	ActualEndAt     *time.Time     `gorm:"column:actual_end_at" json:"actual_end_at"`
}

func (OrderProcess) TableName() string { return "order_processes" }
