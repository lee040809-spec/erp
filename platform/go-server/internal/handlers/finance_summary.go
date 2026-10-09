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
	start := year + "-01-01"
	end := strconv.Itoa(mustYear(year)+1) + "-01-01"
	type produced struct {
		Month    string
		Quantity string
		Billable string
		Reserve  string
		Count    int64
		Output   string
	}
	type spent struct {
		Month       string
		Material    string
		Outsourcing string
		Other       string
		Total       string
	}
	type period struct {
		Month  string
		Amount string
	}
	var outputs []produced
	var costs []spent
	var periods []period
	err := h.db.Raw(`SELECT DATE_FORMAT(business_date,'%Y-%m') AS month,CAST(SUM(qualified_quantity) AS CHAR) AS quantity,CAST(SUM(billable_quantity) AS CHAR) AS billable,CAST(SUM(qualified_quantity-billable_quantity) AS CHAR) AS reserve,COUNT(*) AS count,CAST(SUM(output_value) AS CHAR) AS output
 FROM production_completions WHERE corp_id=? AND status='confirmed' AND business_date>=? AND business_date<? GROUP BY month`, corpID, start, end).Scan(&outputs).Error
	if err == nil {
		err = h.db.Raw(`SELECT DATE_FORMAT(p.business_date,'%Y-%m') AS month,
 CAST(SUM(CASE WHEN e.cost_type='material' THEN a.allocated_amount ELSE 0 END) AS CHAR) AS material,
 CAST(SUM(CASE WHEN e.cost_type='outsourcing' THEN a.allocated_amount ELSE 0 END) AS CHAR) AS outsourcing,
 CAST(SUM(CASE WHEN e.cost_type NOT IN ('material','outsourcing') THEN a.allocated_amount ELSE 0 END) AS CHAR) AS other,
 CAST(SUM(a.allocated_amount) AS CHAR) AS total
 FROM cost_allocations a JOIN production_completions p ON p.id=a.production_completion_id AND p.corp_id=a.corp_id AND p.status='confirmed'
 JOIN cost_entries e ON e.id=a.cost_entry_id AND e.corp_id=a.corp_id AND e.status='confirmed' AND e.cost_type<>'period'
 WHERE a.corp_id=? AND p.business_date>=? AND p.business_date<? GROUP BY month`, corpID, start, end).Scan(&costs).Error
	}
	if err == nil {
		err = h.db.Raw("SELECT accounting_month AS month,CAST(SUM(net_amount) AS CHAR) AS amount FROM cost_entries WHERE corp_id=? AND status='confirmed' AND cost_type='period' AND accounting_month>=? AND accounting_month<? GROUP BY accounting_month", corpID, year+"-01", strconv.Itoa(mustYear(year)+1)+"-01").Scan(&periods).Error
	}
	if err != nil {
		response.InternalError(c, response.CodeInternal, "读取经营汇总失败")
		return
	}
	byMonth := map[string]*models.MonthlyFinanceSummary{}
	get := func(month string) *models.MonthlyFinanceSummary {
		if byMonth[month] == nil {
			byMonth[month] = &models.MonthlyFinanceSummary{Month: month, QualifiedQuantity: "0", BillableQuantity: "0", ReserveQuantity: "0", OutputValue: "0.00", MaterialCost: "0.00", OutsourcingCost: "0.00", OtherProductionCost: "0.00", ProductionCost: "0.00", PeriodExpense: "0.00", ClosingStatus: "open"}
		}
		return byMonth[month]
	}
	for _, p := range outputs {
		m := get(p.Month)
		m.QualifiedQuantity = p.Quantity
		m.BillableQuantity = p.Billable
		m.ReserveQuantity = p.Reserve
		m.CompletionCount = p.Count
		m.OutputValue = p.Output
	}
	for _, s := range costs {
		m := get(s.Month)
		m.MaterialCost = s.Material
		m.OutsourcingCost = s.Outsourcing
		m.OtherProductionCost = s.Other
		m.ProductionCost = s.Total
	}
	for _, p := range periods {
		get(p.Month).PeriodExpense = p.Amount
	}
	var closings []struct {
		AccountingMonth string
		Status          string
	}
	if h.db.Table("monthly_closings").Where("corp_id=? AND accounting_month>=? AND accounting_month<?", corpID, year+"-01", strconv.Itoa(mustYear(year)+1)+"-01").Find(&closings).Error != nil {
		response.InternalError(c, response.CodeInternal, "读取月结失败")
		return
	}
	for _, cl := range closings {
		if m := byMonth[cl.AccountingMonth]; m != nil {
			m.ClosingStatus = cl.Status
		}
	}
	annual := models.AnnualFinanceSummary{Year: year, OutputValue: "0.00", ProductionCost: "0.00", PeriodExpense: "0.00"}
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
			annual.ProductionCost = decimalAdd(annual.ProductionCost, m.ProductionCost)
			annual.PeriodExpense = decimalAdd(annual.PeriodExpense, m.PeriodExpense)
		}
	}
	annual.ProductionProfit = decimalSubtract(annual.OutputValue, annual.ProductionCost)
	annual.OperatingProfit = decimalSubtract(annual.ProductionProfit, annual.PeriodExpense)
	var pending struct {
		Amount string
		Count  int64
	}
	if h.db.Raw(`SELECT CAST(COALESCE(SUM(e.net_amount-COALESCE(a.total,0)),0) AS CHAR) AS amount,COUNT(*) AS count FROM cost_entries e
 LEFT JOIN (SELECT corp_id,cost_entry_id,SUM(allocated_amount) AS total FROM cost_allocations GROUP BY corp_id,cost_entry_id) a ON a.corp_id=e.corp_id AND a.cost_entry_id=e.id
 WHERE e.corp_id=? AND e.status='confirmed' AND e.cost_type<>'period' AND e.net_amount>COALESCE(a.total,0)`, corpID).Scan(&pending).Error != nil {
		response.InternalError(c, response.CodeInternal, "读取未分摊成本失败")
		return
	}
	response.Success(c, gin.H{"year": year, "months": months, "annual": annual, "pending_cost_amount": pending.Amount, "pending_cost_count": pending.Count})
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
