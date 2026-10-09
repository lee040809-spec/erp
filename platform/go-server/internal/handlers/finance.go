package handlers

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"machining-erp/internal/middleware"
	"machining-erp/internal/models"
)

type FinanceHandler struct {
	db *gorm.DB
}

func NewFinanceHandler(db *gorm.DB) *FinanceHandler {
	return &FinanceHandler{db: db}
}

func (h *FinanceHandler) GetReconciliation(c *gin.Context) {
	if !requireModuleAccess(h.db, c, "finance", "reconciliation", "orders", "monthly_output") {
		return
	}
	var results []models.Reconciliation
	corpID := middleware.GetCorpID(c)

	err := h.db.Table("orders as o").
		Select(`DATE_FORMAT(o.due_date, '%Y-%m') as month,
			SUM(oi.total_price) as total_amount,
			COUNT(DISTINCT o.id) as order_count,
			SUM(CASE WHEN oi.status = 'delivered' THEN oi.total_price ELSE 0 END) as delivered_amount`).
		Joins("JOIN order_items oi ON o.id = oi.order_id").
		Where("o.corp_id = ? AND oi.corp_id = ?", corpID, corpID).
		Group("month").
		Order("month DESC").
		Scan(&results).Error

	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	c.JSON(200, results)
}
