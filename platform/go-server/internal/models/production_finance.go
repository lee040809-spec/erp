package models

import "time"

type ProductionCompletion struct {
	DeliveryBatchID    *int64     `json:"delivery_batch_id"`
	ID                 int64      `gorm:"primaryKey;autoIncrement" json:"id"`
	CorpID             int64      `gorm:"column:corp_id" json:"corp_id"`
	OrderID            int64      `gorm:"column:order_id" json:"order_id"`
	OrderItemID        int64      `gorm:"column:order_item_id" json:"order_item_id"`
	CompletionNo       string     `gorm:"column:completion_no" json:"completion_no"`
	BusinessDate       time.Time  `gorm:"column:business_date" json:"business_date"`
	BillableQuantity   string     `gorm:"column:billable_quantity" json:"billable_quantity"`
	QualifiedQuantity  string     `gorm:"column:qualified_quantity" json:"qualified_quantity"`
	UnitPriceSnapshot  string     `gorm:"column:unit_price_snapshot" json:"unit_price_snapshot"`
	OutputValue        string     `gorm:"column:output_value" json:"output_value"`
	SupplyModeSnapshot string     `gorm:"column:supply_mode_snapshot" json:"supply_mode_snapshot"`
	Currency           string     `gorm:"column:currency" json:"currency"`
	TaxInclusive       bool       `gorm:"column:tax_inclusive" json:"tax_inclusive"`
	Status             string     `gorm:"column:status" json:"status"`
	Notes              *string    `gorm:"column:notes" json:"notes,omitempty"`
	CreatedBy          int64      `gorm:"column:created_by" json:"created_by"`
	ConfirmedAt        time.Time  `gorm:"column:confirmed_at" json:"confirmed_at"`
	ReversedBy         *int64     `gorm:"column:reversed_by" json:"reversed_by,omitempty"`
	ReversedAt         *time.Time `gorm:"column:reversed_at" json:"reversed_at,omitempty"`
	ReversalReason     *string    `gorm:"column:reversal_reason" json:"reversal_reason,omitempty"`
	CreatedAt          time.Time  `gorm:"column:created_at" json:"created_at"`
}

func (ProductionCompletion) TableName() string { return "production_completions" }

type CostEntry struct {
	DeliveryBatchID        *int64    `json:"delivery_batch_id"`
	ID                     int64     `gorm:"primaryKey;autoIncrement" json:"id"`
	CorpID                 int64     `gorm:"column:corp_id" json:"corp_id"`
	CostType               string    `gorm:"column:cost_type" json:"cost_type"`
	Amount                 string    `gorm:"column:amount" json:"amount"`
	TaxAmount              string    `gorm:"column:tax_amount" json:"tax_amount"`
	NetAmount              string    `gorm:"column:net_amount" json:"net_amount"`
	Currency               string    `gorm:"column:currency" json:"currency"`
	BusinessDate           time.Time `gorm:"column:business_date" json:"business_date"`
	AccountingMonth        string    `gorm:"column:accounting_month" json:"accounting_month"`
	OrderID                *int64    `gorm:"column:order_id" json:"order_id,omitempty"`
	OrderItemID            *int64    `gorm:"column:order_item_id" json:"order_item_id,omitempty"`
	ProductionCompletionID *int64    `gorm:"column:production_completion_id" json:"production_completion_id,omitempty"`
	SupplierName           *string   `gorm:"column:supplier_name" json:"supplier_name,omitempty"`
	SourceType             *string   `gorm:"column:source_type" json:"source_type,omitempty"`
	SourceID               *string   `gorm:"column:source_id" json:"source_id,omitempty"`
	Status                 string    `gorm:"column:status" json:"status"`
	Notes                  *string   `gorm:"column:notes" json:"notes,omitempty"`
	CreatedBy              int64     `gorm:"column:created_by" json:"created_by"`
	CreatedAt              time.Time `gorm:"column:created_at" json:"created_at"`
}

func (CostEntry) TableName() string { return "cost_entries" }

type MonthlyFinanceSummary struct {
	OrderAmount         string          `json:"order_amount"`
	OrderCosts          OrderCostTotals `json:"order_costs"`
	Month               string          `json:"month"`
	QualifiedQuantity   string          `json:"qualified_quantity"`
	BillableQuantity    string          `json:"billable_quantity"`
	ReserveQuantity     string          `json:"reserve_quantity"`
	CompletionCount     int64           `json:"completion_count"`
	OutputValue         string          `json:"output_value"`
	MaterialCost        string          `json:"material_cost"`
	OutsourcingCost     string          `json:"outsourcing_cost"`
	OtherProductionCost string          `json:"other_production_cost"`
	ProductionCost      string          `json:"production_cost"`
	ProductionProfit    string          `json:"production_profit"`
	PeriodExpense       string          `json:"period_expense"`
	OperatingProfit     string          `json:"operating_profit"`
	ClosingStatus       string          `json:"closing_status"`
}

type AnnualFinanceSummary struct {
	OrderAmount      string `json:"order_amount"`
	Year             string `json:"year"`
	OutputValue      string `json:"output_value"`
	ProductionCost   string `json:"production_cost"`
	ProductionProfit string `json:"production_profit"`
	PeriodExpense    string `json:"period_expense"`
	OperatingProfit  string `json:"operating_profit"`
}

// Display reference from order entry; never added to actual cost allocations.
type OrderCostTotals struct {
	Material    string `json:"material"`
	Outsourcing string `json:"outsourcing"`
	Tool        string `json:"tool"`
	Fixture     string `json:"fixture"`
	Other       string `json:"other"`
}
