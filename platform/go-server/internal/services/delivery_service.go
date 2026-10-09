package services

import (
	"fmt"
	"strings"
	"time"

	"github.com/mozillazg/go-pinyin"
	"gorm.io/gorm"
)

// DeliveryService 送货单预览服务
type DeliveryService struct {
	db *gorm.DB
}

func NewDeliveryService(db *gorm.DB) *DeliveryService {
	return &DeliveryService{db: db}
}

// DeliveryOther 送货单其他计算字段
type DeliveryOther struct {
	SmallTotal     string `json:"small_total"`     // 小写总计
	BigTotal       string `json:"big_total"`       // 大写总计
	TotalQuantity  int    `json:"total_quantity"`  // 总数量
	DeliveryNumber string `json:"delivery_number"` // 送货单号
}

// DeliveryOrder 送货单订单数据（用于预览）
type DeliveryOrder struct {
	ID                int64          `json:"id"`
	CorpID            int64          `json:"corp_id"`
	CustomerID        *int64         `json:"customer_id"`
	CustomerName      *string        `json:"customer_name"`
	CustomerShortName *string        `json:"customer_short_name"`
	OrderNumber       string         `json:"order_number"`
	OrderName         *string        `json:"order_name"`
	ContactID         *int64         `json:"contact_id"`
	ContactName       *string        `json:"contact_name"`
	Status            string         `json:"status"`
	Priority          string         `json:"priority"`
	StartDate         *string        `json:"start_date"`
	DueDate           *string        `json:"due_date"`
	TotalAmount       float64        `json:"total_amount"`
	Notes             *string        `json:"notes"`
	ContactInfo       *string        `json:"contact_info"` // 联系方式
	Items             []DeliveryItem `json:"items"`
	Other             DeliveryOther  `json:"other"` // 其他计算字段
}

// DeliveryItem 送货单零件数据
type DeliveryItem struct {
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

// GetOrderByID 根据订单ID获取送货单预览数据
func (s *DeliveryService) GetOrderByID(orderID int64, corpName string) (*DeliveryOrder, error) {
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
		TotalPrice     float64
		Status         string
		CompletionDate *string
		DeliveredQty   *int
		ScrapQuantity  int
		Notes          *string
	}

	err = s.db.Table("order_items").Where("order_id = ?", orderID).Order("id ASC").Find(&items).Error
	if err != nil {
		return nil, err
	}

	// 组装数据，计算总金额和总数量
	deliveryItems := make([]DeliveryItem, len(items))
	var smallTotal float64 = 0
	var totalQuantity int = 0

	for i, it := range items {
		// 计算 total_price
		totalPrice := it.TotalPrice
		if totalPrice == 0 && it.UnitPrice > 0 && it.Quantity > 0 {
			totalPrice = it.UnitPrice * float64(it.Quantity)
		}
		deliveryItems[i] = DeliveryItem{
			RowIndex:       i + 1, // 序号（1-based）
			ID:             it.ID,
			PartName:       it.PartName,
			PartNumber:     it.PartNumber,
			Quantity:       it.Quantity,
			UnitPrice:      it.UnitPrice,
			TotalPrice:     totalPrice,
			Status:         it.Status,
			CompletionDate: it.CompletionDate,
			DeliveredQty:   it.DeliveredQty,
			ScrapQuantity:  it.ScrapQuantity,
			Notes:          it.Notes,
		}
		smallTotal += totalPrice
		totalQuantity += it.Quantity
	}

	// 生成送货单号：公司首字母拼音-S当前日期-订单后三位
	deliveryNumber := generateDeliveryNumber(corpName, order.OrderNumber)

	// 组装其他字段
	other := DeliveryOther{
		SmallTotal:     fmt.Sprintf("%.2f", smallTotal),
		BigTotal:       numberToChinese(smallTotal),
		TotalQuantity:  totalQuantity,
		DeliveryNumber: deliveryNumber,
	}

	result := &DeliveryOrder{
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
		Items:             deliveryItems,
		Other:             other,
	}

	return result, nil
}

// generateDeliveryNumber 生成送货单号
// 格式：公司首字母拼音-S当前日期-订单后三位
// 例如：YHS-S20260127-001
func generateDeliveryNumber(corpName, orderNumber string) string {
	// 获取公司名称首字母拼音
	corpInitials := getChineseInitials(corpName)

	// 获取当前日期
	dateStr := time.Now().Format("20060102")

	// 获取订单号后三位
	orderSuffix := "001"
	if len(orderNumber) >= 3 {
		orderSuffix = orderNumber[len(orderNumber)-3:]
	} else if len(orderNumber) > 0 {
		orderSuffix = fmt.Sprintf("%03s", orderNumber)
	}

	return fmt.Sprintf("%s-S%s-%s", corpInitials, dateStr, orderSuffix)
}

// getChineseInitials 获取中文字符串的首字母拼音（使用go-pinyin库）
func getChineseInitials(s string) string {
	if s == "" {
		return "X"
	}

	// 使用 go-pinyin 库转换
	a := pinyin.NewArgs()
	a.Style = pinyin.FirstLetter // 只取首字母

	result := pinyin.Pinyin(s, a)
	var initials []string
	for _, py := range result {
		if len(py) > 0 {
			initials = append(initials, strings.ToUpper(py[0]))
		}
		// 只取前3个字符的首字母
		if len(initials) >= 3 {
			break
		}
	}

	if len(initials) == 0 {
		return "X"
	}
	return strings.Join(initials, "")
}

// numberToChinese 数字转大写中文金额
func numberToChinese(num float64) string {
	n := int(num * 100) // 转换为分，处理小数
	if n == 0 {
		return "零元整"
	}

	digits := []string{"零", "壹", "贰", "叁", "肆", "伍", "陆", "柒", "捌", "玖"}
	units := []string{"分", "角", "元", "拾", "佰", "仟", "万", "拾", "佰", "仟", "亿"}

	var result []string
	zeroFlag := false

	for i := 0; n > 0 && i < len(units); i++ {
		digit := n % 10
		n = n / 10

		if digit == 0 {
			zeroFlag = true
			if i == 2 { // 元位
				result = append([]string{"元"}, result...)
			}
		} else {
			if zeroFlag {
				result = append([]string{"零"}, result...)
				zeroFlag = false
			}
			result = append([]string{digits[digit] + units[i]}, result...)
		}
	}

	// 处理末尾的"分"
	if len(result) > 0 && strings.HasSuffix(result[len(result)-1], "分") {
		// 有分，不需要"整"
	} else if len(result) > 0 {
		result = append(result, "整")
	}

	return strings.Join(result, "")
}

// Expose the same amount-in-words formatter for a specific delivery record.
func DeliveryChineseAmount(value float64) string { return numberToChinese(value) }
