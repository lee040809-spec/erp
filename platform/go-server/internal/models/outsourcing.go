package models

import "time"

type OutsourcingDispatch struct {
	EditToken              string    `json:"edit_token" gorm:"-"`
	SupplierID             *int64    `json:"supplier_id"`
	OutsourcingOrderID     *int64    `json:"outsourcing_order_id"`
	OutsourcingOrderLineID *int64    `json:"outsourcing_order_line_id"`
	Scope                  string    `json:"scope"`
	ID                     int64     `json:"id" gorm:"primaryKey"`
	CorpID                 int64     `json:"corp_id"`
	OrderID                int64     `json:"order_id"`
	OrderItemID            int64     `json:"order_item_id"`
	OrderProcessID         int64     `json:"order_process_id"`
	DeliveryBatchID        *int64    `json:"delivery_batch_id"`
	DispatchNo             string    `json:"dispatch_no"`
	SupplierName           string    `json:"supplier_name"`
	Quantity               int       `json:"quantity"`
	UnitPrice              string    `json:"unit_price"`
	DispatchDate           time.Time `json:"dispatch_date"`
	ExpectedReturnDate     time.Time `json:"expected_return_date"`
	OpenQuantity           int       `json:"open_quantity"`
	ReworkWaitingQuantity  int       `json:"rework_waiting_quantity"`
	ReworkOutQuantity      int       `json:"rework_out_quantity"`
	AcceptedQuantity       int       `json:"accepted_quantity"`
	ConcessionQuantity     int       `json:"concession_quantity"`
	ScrapQuantity          int       `json:"scrap_quantity"`
	Status                 string    `json:"status"`
	RequestID              string    `json:"request_id"`
	Notes                  string    `json:"notes"`
	CreatedBy              int64     `json:"created_by"`
	CreatedAt              time.Time `json:"created_at"`
}

func (OutsourcingDispatch) TableName() string { return "outsourcing_dispatches" }

type OutsourcingEvent struct {
	ID                 int64     `json:"id" gorm:"primaryKey"`
	CorpID             int64     `json:"corp_id"`
	DispatchID         int64     `json:"dispatch_id"`
	Kind               string    `json:"kind"`
	Quantity           int       `json:"quantity"`
	GoodQuantity       int       `json:"good_quantity"`
	ConcessionQuantity int       `json:"concession_quantity"`
	ReworkQuantity     int       `json:"rework_quantity"`
	ScrapQuantity      int       `json:"scrap_quantity"`
	BusinessDate       time.Time `json:"business_date"`
	Notes              string    `json:"notes"`
	RequestID          string    `json:"request_id"`
	CreatedBy          int64     `json:"created_by"`
	CreatedAt          time.Time `json:"created_at"`
}

func (OutsourcingEvent) TableName() string { return "outsourcing_events" }

type OutsourcingSettlement struct {
	ID           int64     `json:"id" gorm:"primaryKey"`
	CorpID       int64     `json:"corp_id"`
	DispatchID   int64     `json:"dispatch_id"`
	Quantity     int       `json:"quantity"`
	Amount       string    `json:"amount"`
	TaxAmount    string    `json:"tax_amount"`
	NetAmount    string    `json:"net_amount"`
	BusinessDate time.Time `json:"business_date"`
	ReferenceNo  string    `json:"reference_no"`
	Notes        string    `json:"notes"`
	CostEntryID  *int64    `json:"cost_entry_id"`
	CreatedBy    int64     `json:"created_by"`
	CreatedAt    time.Time `json:"created_at"`
}

func (OutsourcingSettlement) TableName() string { return "outsourcing_settlements" }
