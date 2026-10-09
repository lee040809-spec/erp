package services

import (
	"fmt"
	"strings"

	"github.com/mozillazg/go-pinyin"
	"gorm.io/gorm"
)

// ReconciliationService 对账单预览服务
type ReconciliationService struct {
	db *gorm.DB
}

func NewReconciliationService(db *gorm.DB) *ReconciliationService {
	return &ReconciliationService{db: db}
}

// ReconciliationOther 对账单其他计算字段
type ReconciliationOther struct {
	SmallTotal           string `json:"small_total"`           // 小写总计
	BigTotal             string `json:"big_total"`             // 大写总计
	TotalQuantity        int    `json:"total_quantity"`        // 总数量
	ReconciliationNumber string `json:"reconciliation_number"` // 对账单号
}

// ReconciliationOrder 对账单订单数据（用于预览）
type ReconciliationOrder struct {
	ID                int64                `json:"id"`
	CorpID            int64                `json:"corp_id"`
	CustomerID        *int64               `json:"customer_id"`
	CustomerName      *string              `json:"customer_name"`
	CustomerShortName *string              `json:"customer_short_name"`
	OrderNumber       string               `json:"order_number"`
	OrderName         *string              `json:"order_name"`
	ContactID         *int64               `json:"contact_id"`
	ContactName       *string              `json:"contact_name"`
	Status            string               `json:"status"`
	Priority          string               `json:"priority"`
	StartDate         *string              `json:"start_date"`
	DueDate           *string              `json:"due_date"`
	TotalAmount       float64              `json:"total_amount"`
	Notes             *string              `json:"notes"`
	ContactInfo       *string              `json:"contact_info"`
	Items             []ReconciliationItem `json:"items"`
	Other             ReconciliationOther  `json:"other"`
}

// ReconciliationItem 对账单零件数据
type ReconciliationItem struct {
	RowIndex       int     `json:"row_index"` // 序号（1-based）
	ID             int64   `json:"id"`
	PartName       string  `json:"part_name"`
	PartNumber     *string `json:"part_number"`
	Quantity       int     `json:"quantity"`
	UnitPrice      float64 `json:"unit_price"`
	TotalPrice     float64 `json:"total_price"`
	Status         string  `json:"status"`
	CompletionDate *string `json:"completion_date"`
	DeliveredQty   *int    `json:"delivered_quantity"`
	ScrapQuantity  int     `json:"scrap_quantity"`
	Notes          *string `json:"notes"`
}

// GetOrderByID 根据订单ID获取对账单预览数据
func (s *ReconciliationService) GetOrderByID(orderID int64, corpName string) (*ReconciliationOrder, error) {
	// 查询订单基本信息
	var order struct {
		ID                int64
		CorpID            int64
		CustomerID        *int64
		CustomerName      *string
		CustomerShortName *string
		OrderNumber       string
		OrderName         *string
		ContactID         *int64
		ContactName       *string
		Status            string
		Priority          string
		StartDate         *string
		DueDate           *string
		TotalAmount       float64
		Notes             *string
	}

	err := s.db.Table("orders").Where("id = ?", orderID).First(&order).Error
	if err != nil {
		return nil, err
	}

	// 查询联系方式
	var contactInfo *string
	if order.ContactID != nil {
		var contact struct {
			Phone string
		}
		s.db.Table("contacts").Where("id = ?", *order.ContactID).Select("phone").First(&contact)
		if contact.Phone != "" {
			contactInfo = &contact.Phone
		}
	}

	// 查询零件信息
	var items []struct {
		ID             int64
		PartName       string
		PartNumber     *string
		Quantity       int
		UnitPrice      float64
		Status         string
		CompletionDate *string
		DeliveredQty   *int
		ScrapQuantity  int
		Notes          *string
	}
	s.db.Table("order_items").Where("order_id = ?", orderID).Find(&items)

	// 转换为对账单零件数据
	reconciliationItems := make([]ReconciliationItem, len(items))
	for i, item := range items {
		totalPrice := float64(item.Quantity) * item.UnitPrice
		reconciliationItems[i] = ReconciliationItem{
			RowIndex:       i + 1,
			ID:             item.ID,
			PartName:       item.PartName,
			PartNumber:     item.PartNumber,
			Quantity:       item.Quantity,
			UnitPrice:      item.UnitPrice,
			TotalPrice:     totalPrice,
			Status:         item.Status,
			CompletionDate: item.CompletionDate,
			DeliveredQty:   item.DeliveredQty,
			ScrapQuantity:  item.ScrapQuantity,
			Notes:          item.Notes,
		}
	}

	// 生成对账单号
	reconciliationNumber := generateReconciliationNumber(corpName, order.OrderNumber)

	// 计算汇总数据
	var totalQuantity int
	var smallTotal float64
	for _, item := range reconciliationItems {
		totalQuantity += item.Quantity
		smallTotal += item.TotalPrice
	}

	result := &ReconciliationOrder{
		ID:                order.ID,
		CorpID:            order.CorpID,
		CustomerID:        order.CustomerID,
		CustomerName:      order.CustomerName,
		CustomerShortName: order.CustomerShortName,
		OrderNumber:       order.OrderNumber,
		OrderName:         order.OrderName,
		ContactID:         order.ContactID,
		ContactName:       order.ContactName,
		Status:            order.Status,
		Priority:          order.Priority,
		StartDate:         order.StartDate,
		DueDate:           order.DueDate,
		TotalAmount:       order.TotalAmount,
		Notes:             order.Notes,
		ContactInfo:       contactInfo,
		Items:             reconciliationItems,
		Other: ReconciliationOther{
			SmallTotal:           fmt.Sprintf("%.2f", smallTotal),
			BigTotal:             numberToChinese(smallTotal),
			TotalQuantity:        totalQuantity,
			ReconciliationNumber: reconciliationNumber,
		},
	}

	return result, nil
}

// GetOrdersByIDs 根据多个订单ID获取汇总的对账单数据
func (s *ReconciliationService) GetOrdersByIDs(orderIDs []int64, corpName string) (*ReconciliationOrder, error) {
	if len(orderIDs) == 0 {
		return nil, fmt.Errorf("no order IDs provided")
	}

	var allItems []ReconciliationItem
	var firstOrder *ReconciliationOrder
	var totalAmount float64
	var totalQuantity int
	rowIndex := 0

	for _, orderID := range orderIDs {
		order, err := s.GetOrderByID(orderID, corpName)
		if err != nil {
			continue
		}

		if firstOrder == nil {
			firstOrder = order
		}

		// 添加零件，重新编号
		for _, item := range order.Items {
			item.RowIndex = rowIndex + 1
			allItems = append(allItems, item)
			rowIndex++
			totalQuantity += item.Quantity
			totalAmount += item.TotalPrice
		}
	}

	if firstOrder == nil {
		return nil, fmt.Errorf("no valid orders found")
	}

	// 生成汇总对账单号
	reconciliationNumber := generateReconciliationNumber(corpName, "汇总")

	// 汇总数据
	mergedOrder := *firstOrder
	mergedOrder.Items = allItems
	mergedOrder.TotalAmount = totalAmount
	mergedOrder.Other.TotalQuantity = totalQuantity
	mergedOrder.Other.SmallTotal = fmt.Sprintf("%.2f", totalAmount)
	mergedOrder.Other.BigTotal = numberToChinese(totalAmount)
	mergedOrder.Other.ReconciliationNumber = reconciliationNumber

	return &mergedOrder, nil
}

// generateReconciliationNumber 生成对账单号
func generateReconciliationNumber(corpName, orderNumber string) string {
	// 格式: 公司简称拼音-日期-序号
	pyArgs := pinyin.NewArgs()
	pyArgs.Style = pinyin.FirstLetter
	pinyinResult := pinyin.Pinyin(corpName, pyArgs)

	var shortPy string
	for _, py := range pinyinResult {
		if len(py) > 0 {
			shortPy += py[0]
		}
	}

	if shortPy == "" {
		shortPy = "DZ"
	}

	dateStr := strings.Split(strings.Split(orderNumber, "-")[0], "_")[0]
	if len(dateStr) < 6 {
		dateStr = strings.ReplaceAll(orderNumber, "-", "")[:8]
	}

	return fmt.Sprintf("%s-%s-DZ", strings.ToUpper(shortPy), dateStr)
}
