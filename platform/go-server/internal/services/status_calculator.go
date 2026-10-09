package services

import (
	"machining-erp/internal/models"
	"time"
)

// CalculateStatus 根据子项状态计算父项状态
// 规则:
//   - 所有子项都是 completed/delivered -> 返回 completed
//   - 所有子项都是 pending -> 返回 pending
//   - 否则 -> 返回 processing
func CalculateStatus(statuses []string) string {
	if len(statuses) == 0 {
		return "pending"
	}

	allCompleted := true
	allPending := true

	for _, status := range statuses {
		if status != "completed" && status != "delivered" {
			allCompleted = false
		}
		if status != "pending" {
			allPending = false
		}
	}

	if allCompleted {
		return "completed"
	}
	if allPending {
		return "pending"
	}
	return "processing"
}

// CalculateOrderItemStatus 计算订单项状态(基于工序)
func CalculateOrderItemStatus(processes []models.OrderProcess) string {
	statuses := make([]string, len(processes))
	for i, p := range processes {
		statuses[i] = string(p.Status)
	}
	return CalculateStatus(statuses)
}

// CalculateOrderStatus 计算订单状态(基于订单项)
func CalculateOrderStatus(items []models.OrderItem) string {
	statuses := make([]string, len(items))
	for i, item := range items {
		statuses[i] = string(item.Status)
	}
	return CalculateStatus(statuses)
}

// FormatDateNow 返回当前日期的 YYYY-MM-DD 格式字符串
func FormatDateNow() string {
	return time.Now().Format("2006-01-02")
}
