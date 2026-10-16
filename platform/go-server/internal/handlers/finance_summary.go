package handlers

import (
	"github.com/gin-gonic/gin"
	"machining-erp/internal/middleware"
	"machining-erp/internal/models"
	"machining-erp/pkg/response"
	"math/big"
	"strconv"
	"time"
)

func (h *ProductionFinanceHandler) GetSummary(c *gin.Context) {
	if !h.requireFinanceAccess(c) {
		return
	}
	year := c.DefaultQuery("year", strconv.Itoa(time.Now().Year()))
	if !yearPattern.MatchString(year) {
		response.BadRequest(c, response.CodeInvalidRequest, "年份格式必须为 YYYY")
		return
	}
	corpID := middleware.GetCorpID(c)
	s, err := readFinanceSummary(h.db.WithContext(c.Request.Context()), corpID, year)
	if err != nil {
		response.InternalError(c, response.CodeInternal, "读取经营汇总失败")
		return
	}
	byMonth := map[string]*models.MonthlyFinanceSummary{}
	get := func(month string) *models.MonthlyFinanceSummary {
		if byMonth[month] == nil {
			byMonth[month] = &models.MonthlyFinanceSummary{Month: month, QualifiedQuantity: "0", BillableQuantity: "0", ReserveQuantity: "0", OutputValue: "0.00", MaterialCost: "0.00", OutsourcingCost: "0.00", OtherProductionCost: "0.00", ProductionCost: "0.00", PeriodExpense: "0.00", ClosingStatus: "open"}
			byMonth[month].OrderAmount = "0.00"
			byMonth[month].OrderCosts = models.OrderCostTotals{Material: "0.00", Outsourcing: "0.00", Tool: "0.00", Fixture: "0.00", Other: "0.00"}
		}
		return byMonth[month]
	}
	for _, o := range s.Orders {
		m := get(o.Month)
		m.OrderAmount = o.Amount
		m.OrderCosts = enteredOrderCosts(o)
	}
	for _, p := range s.Outputs {
		m := get(p.Month)
		m.QualifiedQuantity = p.Quantity
		m.BillableQuantity = p.Billable
		m.ReserveQuantity = p.Reserve
		m.CompletionCount = p.Count
		m.OutputValue = p.Output
	}
	for _, s := range s.Costs {
		m := get(s.Month)
		m.MaterialCost = s.Material
		m.OutsourcingCost = s.Outsourcing
		m.OtherProductionCost = s.Other
		m.ProductionCost = s.Total
	}
	for _, p := range s.Periods {
		get(p.Month).PeriodExpense = p.Amount
	}
	for _, cl := range s.Closings {
		if m := byMonth[cl.AccountingMonth]; m != nil {
			m.ClosingStatus = cl.Status
		}
	}
	annual := models.AnnualFinanceSummary{Year: year, OrderAmount: "0.00", OutputValue: "0.00", ProductionCost: "0.00", PeriodExpense: "0.00"}
	months := make([]models.MonthlyFinanceSummary, 0)
	for i := 12; i >= 1; i-- {
		key := year + "-"
		if i < 10 {
			key += "0"
		}
		key += strconv.Itoa(i)
		if m := byMonth[key]; m != nil {
			m.ProductionProfit = decimalSubtract(m.OutputValue, m.ProductionCost)
			m.OperatingProfit = decimalSubtract(m.ProductionProfit, m.PeriodExpense)
			months = append(months, *m)
			annual.OutputValue = decimalAdd(annual.OutputValue, m.OutputValue)
			annual.OrderAmount = decimalAdd(annual.OrderAmount, m.OrderAmount)
			annual.ProductionCost = decimalAdd(annual.ProductionCost, m.ProductionCost)
			annual.PeriodExpense = decimalAdd(annual.PeriodExpense, m.PeriodExpense)
		}
	}
	annual.ProductionProfit = decimalSubtract(annual.OutputValue, annual.ProductionCost)
	annual.OperatingProfit = decimalSubtract(annual.ProductionProfit, annual.PeriodExpense)
	response.Success(c, gin.H{"year": year, "months": months, "annual": annual, "pending_cost_amount": s.Pending.Amount, "pending_cost_count": s.Pending.Count})
}

func mustYear(year string) int { n, _ := strconv.Atoi(year); return n }
func decimalAdd(a, b string) string {
	x, _ := new(big.Rat).SetString(a)
	y, _ := new(big.Rat).SetString(b)
	return new(big.Rat).Add(x, y).FloatString(2)
}
func decimalSubtract(a, b string) string {
	x, _ := new(big.Rat).SetString(a)
	y, _ := new(big.Rat).SetString(b)
	return new(big.Rat).Sub(x, y).FloatString(2)
}
