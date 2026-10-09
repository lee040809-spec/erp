package handlers

import (
	"github.com/gin-gonic/gin"
	"log"
	"machining-erp/internal/middleware"
	"machining-erp/internal/models"
	"machining-erp/internal/productionflow"
	"machining-erp/internal/repository"
	"machining-erp/pkg/response"
	"strings"
	"time"
)

func (h *ProductionWorkbenchHandler) List(c *gin.Context) {
	if !h.allowed(c, false) {
		return
	}
	corp := middleware.GetCorpID(c)
	page, size, offset := boundedPage(c, 20)
	today := time.Now().Format("2006-01-02")
	base := ` FROM order_items i JOIN orders o ON o.id=i.order_id AND o.corp_id=i.corp_id
 LEFT JOIN (SELECT b.corp_id,b.order_item_id,MIN(b.due_date) AS next_due FROM order_delivery_batches b LEFT JOIN (SELECT corp_id,delivery_batch_id,SUM(quantity) AS qty FROM order_business_records WHERE kind='delivery' AND status='confirmed' GROUP BY corp_id,delivery_batch_id) d ON d.corp_id=b.corp_id AND d.delivery_batch_id=b.id WHERE COALESCE(d.qty,0)<b.quantity GROUP BY b.corp_id,b.order_item_id) bd ON bd.corp_id=i.corp_id AND bd.order_item_id=i.id
 LEFT JOIN production_plans pp ON pp.corp_id=i.corp_id AND pp.order_item_id=i.id
 LEFT JOIN (SELECT corp_id,order_item_id,SUM(qualified_quantity) AS qualified_quantity, SUM(billable_quantity) AS billable_quantity FROM production_completions WHERE status='confirmed' GROUP BY corp_id,order_item_id) q ON q.corp_id=i.corp_id AND q.order_item_id=i.id
 WHERE i.corp_id=? AND (i.status NOT IN ('completed','delivered') OR (COALESCE(q.qualified_quantity,0)>0 AND COALESCE(NULLIF(i.production_quantity,0),i.quantity)-COALESCE(i.scrap_quantity,0)>q.qualified_quantity))`
	type counts struct {
		View   string `gorm:"column:metric_key" json:"-"`
		Orders int64  `json:"orders"`
		Items  int64  `json:"items"`
		Pieces int64  `json:"pieces"`
	}
	conditions := map[string]string{
		"all":        "",
		"unassigned": " AND COALESCE(pp.assignee,'')=''",
		"today":      " AND COALESCE(pp.planned_end,bd.next_due,i.due_date)=?",
		"overdue":    " AND COALESCE(bd.next_due,i.due_date)<?",
		"blocked":    " AND COALESCE(pp.blocked_reason,'')<>''",
		"outsourced": " AND (EXISTS(SELECT 1 FROM outsourcing_dispatches od WHERE od.corp_id=i.corp_id AND od.order_item_id=i.id AND od.open_quantity+od.rework_waiting_quantity+od.rework_out_quantity>0) OR (NOT EXISTS(SELECT 1 FROM process_quantity_totals pt WHERE pt.corp_id=i.corp_id AND pt.order_item_id=i.id) AND EXISTS(SELECT 1 FROM order_processes op WHERE op.corp_id=i.corp_id AND op.order_item_id=i.id AND op.is_outsourced=1 AND op.name<>'送货' AND op.status<>'completed')))",
	}
	metricSQL := []string{}
	metricArgs := []interface{}{}
	metricKeys := []string{"all", "unassigned", "today", "overdue", "blocked", "outsourced"}
	for _, key := range metricKeys {
		condition := strings.TrimPrefix(conditions[key], " AND ")
		if key == "all" {
			condition = "1=1"
		}
		if key == "outsourced" {
			condition = "COALESCE(wb_out.flag,0)=1"
		}
		metricSQL = append(metricSQL, "COUNT(DISTINCT CASE WHEN "+condition+" THEN o.id END)", "COALESCE(SUM(CASE WHEN "+condition+" THEN 1 ELSE 0 END),0)", "COALESCE(SUM(CASE WHEN "+condition+" THEN i.quantity ELSE 0 END),0)")
		if key == "today" || key == "overdue" {
			metricArgs = append(metricArgs, today, today, today)
		}
	}
	metricBase := strings.Replace(base, " WHERE i.corp_id", " LEFT JOIN (SELECT corp_id,order_item_id,1 AS flag FROM (SELECT corp_id,order_item_id FROM outsourcing_dispatches WHERE open_quantity+rework_waiting_quantity+rework_out_quantity>0 UNION SELECT op.corp_id,op.order_item_id FROM order_processes op LEFT JOIN process_quantity_totals pt ON pt.corp_id=op.corp_id AND pt.order_item_id=op.order_item_id WHERE op.is_outsourced=1 AND op.name<>'送货' AND op.status<>'completed' AND pt.id IS NULL) outs GROUP BY corp_id,order_item_id) wb_out ON wb_out.corp_id=i.corp_id AND wb_out.order_item_id=i.id WHERE i.corp_id", 1)
	metricArgs = append(metricArgs, corp)
	metricResult, metricErr := h.db.Raw("SELECT "+strings.Join(metricSQL, ",")+metricBase, metricArgs...).Rows()
	if metricErr != nil {
		response.InternalError(c, response.CodeInternal, "读取订单和零件数量失败")
		return
	}
	values := make([]int64, len(metricKeys)*3)
	pointers := make([]interface{}, len(values))
	for n := range values {
		pointers[n] = &values[n]
	}
	if !metricResult.Next() {
		metricResult.Close()
		response.InternalError(c, response.CodeInternal, "读取订单和零件数量失败")
		return
	}
	metricErr = metricResult.Scan(pointers...)
	metricResult.Close()
	if metricErr != nil {
		response.InternalError(c, response.CodeInternal, "读取订单和零件数量失败")
		return
	}
	metrics := map[string]counts{}
	for n, key := range metricKeys {
		metrics[key] = counts{Orders: values[n*3], Items: values[n*3+1], Pieces: values[n*3+2]}
	}
	args := []interface{}{corp}
	view := c.Query("view")
	if view == "" {
		view = "all"
	}
	condition, valid := conditions[view]
	if !valid {
		response.BadRequest(c, response.CodeInvalidRequest, "筛选无效")
		return
	}
	base += condition
	if view == "today" || view == "overdue" {
		args = append(args, today)
	}

	if search := strings.TrimSpace(c.Query("search")); search != "" {
		base += " AND (o.order_number LIKE ? OR o.customer_name LIKE ? OR i.part_name LIKE ? OR i.part_number LIKE ?)"
		for n := 0; n < 4; n++ {
			args = append(args, "%"+search+"%")
		}
	}
	var summary counts
	if err := h.db.Raw("SELECT COUNT(DISTINCT o.id) AS orders,COUNT(*) AS items,COALESCE(SUM(i.quantity),0) AS pieces"+base, args...).Scan(&summary).Error; err != nil {
		log.Printf("workbench summary error: %v", err)
		response.InternalError(c, response.CodeInternal, "读取工作台数量失败")
		return
	}
	type row struct {
		Flow               productionflow.Summary     `gorm:"-" json:"flow"`
		ItemID             int64                      `json:"item_id"`
		OrderID            int64                      `json:"order_id"`
		OrderNumber        string                     `json:"order_number"`
		CustomerName       string                     `json:"customer_name"`
		PartName           string                     `json:"part_name"`
		PartNumber         string                     `json:"part_number"`
		Priority           string                     `json:"priority"`
		Status             string                     `json:"status"`
		Quantity           int                        `json:"quantity"`
		ProductionQuantity int                        `json:"production_quantity"`
		BillableQuantity   float64                    `json:"billable_quantity"`
		ReserveQuantity    float64                    `json:"reserve_quantity"`
		DeliveryGap        float64                    `json:"delivery_gap"`
		ScrapQuantity      int                        `json:"scrap_quantity"`
		QualifiedQuantity  float64                    `json:"qualified_quantity"`
		RemainingQuantity  float64                    `json:"remaining_quantity"`
		DueDate            string                     `json:"due_date"`
		StartDate          string                     `json:"start_date"`
		Assignee           string                     `json:"assignee"`
		Equipment          string                     `json:"equipment"`
		PlannedStart       string                     `json:"planned_start"`
		PlannedEnd         string                     `json:"planned_end"`
		EstimatedHours     string                     `json:"estimated_hours"`
		BlockedReason      string                     `json:"blocked_reason"`
		Notes              string                     `json:"notes"`
		HasDrawing         bool                       `json:"has_drawing"`
		Delivery           repository.DeliverySummary `gorm:"-" json:"delivery"`
		Processes          []models.OrderProcess      `gorm:"-" json:"processes"`
	}
	rows := []row{}
	selectSQL := `SELECT i.id AS item_id,o.id AS order_id,o.order_number,COALESCE(o.customer_short_name,o.customer_name,'') AS customer_name,i.part_name,COALESCE(i.part_number,'') AS part_number,o.priority,i.status,i.quantity,COALESCE(NULLIF(i.production_quantity,0),i.quantity) AS production_quantity, COALESCE(q.billable_quantity,0) AS billable_quantity, COALESCE(q.qualified_quantity,0)-COALESCE(q.billable_quantity,0) AS reserve_quantity, GREATEST(i.quantity-COALESCE(q.qualified_quantity,0),0) AS delivery_gap,COALESCE(i.scrap_quantity,0) AS scrap_quantity,COALESCE(q.qualified_quantity,0) AS qualified_quantity,GREATEST(COALESCE(NULLIF(i.production_quantity,0),i.quantity)-COALESCE(i.scrap_quantity,0)-COALESCE(q.qualified_quantity,0),0) AS remaining_quantity,COALESCE(DATE_FORMAT(i.due_date,'%Y-%m-%d'),'') AS due_date,
 COALESCE(DATE_FORMAT(COALESCE(i.start_date,o.start_date),'%Y-%m-%d'),'') AS start_date,COALESCE(pp.assignee,'') AS assignee,COALESCE(pp.equipment,'') AS equipment,COALESCE(DATE_FORMAT(pp.planned_start,'%Y-%m-%d'),'') AS planned_start,COALESCE(DATE_FORMAT(pp.planned_end,'%Y-%m-%d'),'') AS planned_end,COALESCE(CAST(pp.estimated_hours AS CHAR),'0.00') AS estimated_hours,COALESCE(pp.blocked_reason,'') AS blocked_reason,COALESCE(pp.notes,'') AS notes,COALESCE(i.drawing_data,'')<>'' AS has_drawing`
	queryArgs := append(append([]interface{}{}, args...), size, offset)
	err := h.db.Raw(selectSQL+base+` ORDER BY COALESCE(i.start_date,o.start_date) DESC,o.id DESC,i.id DESC LIMIT ? OFFSET ?`, queryArgs...).Scan(&rows).Error
	if err != nil {
		log.Printf("workbench list error: %v", err)
		response.InternalError(c, response.CodeInternal, "读取工作台失败")
		return
	}
	ids := []int64{}
	for _, r := range rows {
		ids = append(ids, r.ItemID)
	}
	processes := []models.OrderProcess{}
	if len(ids) > 0 {
		err = h.db.Select("id,order_item_id,corp_id,name,is_outsourced,status,sort_order,planned_start_at,planned_end_at,actual_start_at,actual_end_at").Where("corp_id=? AND order_item_id IN ?", corp, ids).Order("sort_order,id").Find(&processes).Error
		if err != nil {
			response.InternalError(c, response.CodeInternal, "读取工序失败")
			return
		}
	}
	deliveries, err := repository.LoadDeliverySummaries(h.db, corp, ids)
	if err != nil {
		response.InternalError(c, response.CodeInternal, "读取送货节点失败")
		return
	}
	flows, e := productionflow.Summaries(h.db, corp, ids)
	if e != nil {
		response.InternalError(c, response.CodeInternal, "读取分批生产失败")
		return
	}
	processesByItem := make(map[int64][]models.OrderProcess, len(ids))
	for _, process := range processes {
		processesByItem[process.OrderItemID] = append(processesByItem[process.OrderItemID], process)
	}
	for n := range rows {
		rows[n].Flow = flows[rows[n].ItemID]
		rows[n].Delivery = deliveries[rows[n].ItemID]
		rows[n].Processes = []models.OrderProcess{}
		if itemProcesses := processesByItem[rows[n].ItemID]; itemProcesses != nil {
			rows[n].Processes = itemProcesses
		}
	}
	stats := gin.H{"total": metrics["all"].Items, "unassigned": metrics["unassigned"].Items, "today": metrics["today"].Items, "overdue": metrics["overdue"].Items, "blocked": metrics["blocked"].Items, "outsourced": metrics["outsourced"].Items}
	response.Success(c, gin.H{"items": rows, "total": summary.Items, "summary": summary, "metrics": metrics, "can_manage": h.canManage(c), "page": page, "page_size": size, "stats": stats})
}
