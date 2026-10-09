package repository

import (
	"fmt"
	"gorm.io/gorm"
	"machining-erp/internal/models"
	"machining-erp/internal/productionflow"
	"sort"
	"time"
)

// OrderFilters 订单筛选条件
type OrderFilters struct {
	CorpID       int64
	DueDateStart string
	DueDateEnd   string
	OrderNumber  string
	PartName     string
	PartNumber   string
	CustomerName string
	Priority     string
	Status       string
	Page         int
	PageSize     int
	DateType     string // "overdue" 逾期, "warning" 告警, "near_due" 临期
}

// OrderListResult 订单列表返回结果
type OrderListResult struct {
	Orders []models.Order
	Total  int64
}

func (r *OrderRepository) GetWithFilters(filters OrderFilters) (OrderListResult, error) {
	startTime := time.Now()
	if filters.PageSize <= 0 {
		filters.PageSize = 20
	}
	if filters.PageSize > 100 {
		filters.PageSize = 100
	}
	if filters.Page < 1 {
		filters.Page = 1
	}
	if filters.Page > 1000000 {
		filters.Page = 1000000
	}

	// 1. 构建基础查询
	baseQuery := r.db.Model(&models.Order{})

	// 按 corpID 过滤
	baseQuery = baseQuery.Where("corp_id = ?", filters.CorpID)

	// 应用订单级别筛选
	if filters.Status == "delivery_ready" {
		baseQuery = baseQuery.Where(deliveryReadyFilter)
	} else if filters.Status != "" {
		baseQuery = baseQuery.Where("status = ?", filters.Status)
	}
	if filters.OrderNumber != "" {
		baseQuery = baseQuery.Where("order_number LIKE ?", "%"+filters.OrderNumber+"%")
	}
	if filters.CustomerName != "" {
		baseQuery = baseQuery.Where("customer_name LIKE ? OR customer_short_name LIKE ?",
			"%"+filters.CustomerName+"%", "%"+filters.CustomerName+"%")
	}
	if filters.Priority != "" {
		baseQuery = baseQuery.Where("priority = ?", filters.Priority)
	}

	// 根据期限类型筛选
	today := time.Now().Format("2006-01-02")

	if filters.DateType == "overdue" {
		baseQuery = baseQuery.Where("status NOT IN ?", []string{"delivered", "completed"})
	}
	if filters.DateType != "" && filters.DateType != "overdue" {
		baseQuery = baseQuery.Where("status NOT IN ?", []string{"delivered", "completed"})
		ruleType := map[string]string{"near_due": "imminent", "warning": "warning"}[filters.DateType]
		if ruleType == "" {
			baseQuery = baseQuery.Where("1 = 0")
		} else {
			ids, err := r.getOrderIDsByRuleType(int(filters.CorpID), ruleType)
			if err != nil {
				return OrderListResult{}, err
			}
			if len(ids) > 0 {
				baseQuery = baseQuery.Where("id IN ?", ids)
			} else {
				baseQuery = baseQuery.Where("1=0")
			}
		}
	}

	// 如果有零件级别筛选，需要子查询
	if filters.DueDateStart != "" || filters.DueDateEnd != "" || filters.PartNumber != "" || filters.PartName != "" || filters.DateType == "overdue" {
		subQuery := r.db.Model(&models.OrderItem{}).Select("DISTINCT order_id").Where("corp_id = ?", filters.CorpID)
		if filters.DateType == "overdue" {
			subQuery = subQuery.Where("due_date IS NOT NULL AND due_date < ?", today)
		}
		if filters.DueDateStart != "" {
			subQuery = subQuery.Where("due_date >= ?", filters.DueDateStart)
		}
		if filters.DueDateEnd != "" {
			subQuery = subQuery.Where("due_date <= ?", filters.DueDateEnd)
		}
		if filters.PartName != "" {
			subQuery = subQuery.Where("part_name LIKE ?", "%"+filters.PartName+"%")
		}
		if filters.PartNumber != "" {
			subQuery = subQuery.Where("part_number LIKE ?", "%"+filters.PartNumber+"%")
		}
		baseQuery = baseQuery.Where("id IN (?)", subQuery)
	}

	// 2. 先计算总数
	var total int64
	t1 := time.Now()
	if err := baseQuery.Session(&gorm.Session{}).Count(&total).Error; err != nil {
		return OrderListResult{}, err
	}
	fmt.Printf("[OrderRepo] Count query took %v ms, total=%d\n", time.Since(t1).Milliseconds(), total)

	if total == 0 {
		return OrderListResult{Orders: []models.Order{}, Total: 0}, nil
	}

	// 3. 查询订单列表（分页+排序）
	var orders []models.Order
	orderQuery := baseQuery.Session(&gorm.Session{})

	// 排序和分页
	orderQuery = orderQuery.Order("start_date DESC, id DESC")
	if filters.PageSize > 0 {
		offset := 0
		if filters.Page > 1 {
			offset = (filters.Page - 1) * filters.PageSize
		}
		orderQuery = orderQuery.Limit(filters.PageSize).Offset(offset)
	}

	t2 := time.Now()
	if err := orderQuery.Find(&orders).Error; err != nil {
		return OrderListResult{}, err
	}
	fmt.Printf("[OrderRepo] Orders query took %v ms, count=%d\n", time.Since(t2).Milliseconds(), len(orders))

	if len(orders) == 0 {
		return OrderListResult{Orders: []models.Order{}, Total: total}, nil
	}

	// 收集订单ID
	orderIDs := make([]int64, len(orders))
	for i, o := range orders {
		orderIDs[i] = o.ID
	}

	// 4. 查询零件信息
	var items []models.OrderItem
	if err := r.db.Omit("drawing_data").Where("order_id IN ? AND corp_id=?", orderIDs, filters.CorpID).Order("id DESC").Find(&items).Error; err != nil {
		return OrderListResult{}, err
	}

	type drawingFlag struct {
		ID         int64
		HasDrawing bool
	}
	var flags []drawingFlag
	if err := r.db.Table("order_items").Select("id,COALESCE(drawing_data,'')<>'' AS has_drawing").Where("order_id IN ? AND corp_id=?", orderIDs, filters.CorpID).Scan(&flags).Error; err != nil {
		return OrderListResult{}, err
	}
	drawings := map[int64]bool{}
	for _, f := range flags {
		drawings[f.ID] = f.HasDrawing
	}
	for i := range items {
		items[i].HasDrawing = drawings[items[i].ID]
	}
	// 收集零件ID
	itemIDs := make([]int64, len(items))
	for i, it := range items {
		itemIDs[i] = it.ID
	}

	// 5. 查询工序信息
	var processes []models.OrderProcess
	if len(itemIDs) > 0 {
		if err := r.db.Where("order_item_id IN ? AND corp_id=?", itemIDs, filters.CorpID).Order("sort_order ASC").Find(&processes).Error; err != nil {
			return OrderListResult{}, err
		}
	}

	// Confirmed quantities are physical counts; billable counts remain contracted.
	type quantities struct {
		ItemID    int64
		Qualified float64
		Billable  float64
	}
	var totals []quantities
	if len(itemIDs) > 0 {
		if err := r.db.Table("production_completions").Where("corp_id=? AND order_item_id IN ? AND status='confirmed'", filters.CorpID, itemIDs).Select("order_item_id AS item_id,SUM(qualified_quantity) AS qualified,SUM(billable_quantity) AS billable").Group("order_item_id").Scan(&totals).Error; err != nil {
			return OrderListResult{}, err
		}
	}
	quantitiesByID := map[int64]quantities{}
	for _, q := range totals {
		quantitiesByID[q.ItemID] = q
	}
	for n := range items {
		q := quantitiesByID[items[n].ID]
		items[n].QualifiedQuantity = q.Qualified
		items[n].BillableQuantity = q.Billable
		items[n].ReserveQuantity = q.Qualified - q.Billable
		items[n].DeliveryGap = max(0, float64(items[n].Quantity)-q.Qualified)
		if items[n].ProductionQuantity == 0 {
			items[n].ProductionQuantity = items[n].Quantity
		}
	}
	var dates []struct {
		ItemID int64
		Date   *time.Time
	}
	if len(itemIDs) > 0 {
		if err := r.db.Table("order_business_records").Select("order_item_id AS item_id,MAX(business_date) AS date").Where("corp_id=? AND order_item_id IN ? AND kind='delivery' AND status='confirmed'", filters.CorpID, itemIDs).Group("order_item_id").Scan(&dates).Error; err != nil {
			return OrderListResult{}, err
		}
	}
	datesByID := map[int64]*time.Time{}
	for _, row := range dates {
		datesByID[row.ItemID] = row.Date
	}
	for n := range items {
		items[n].DeliveryDate = datesByID[items[n].ID]
	}
	summaries, e := productionflow.Summaries(r.db, filters.CorpID, itemIDs)
	if e != nil {
		return OrderListResult{}, e
	}
	for n := range items {
		s := summaries[items[n].ID]
		items[n].FlowTracking = s.Tracking
		items[n].DeliveryBatches = s.Batches
		items[n].NextBatchDueDate = s.NextDueDate
		items[n].OpenOutsourcing = s.OpenOutsourcing
	}
	// 6. 组装数据
	// 构建工序映射
	processMap := make(map[int64][]models.OrderProcess)
	for _, p := range processes {
		processMap[p.OrderItemID] = append(processMap[p.OrderItemID], p)
	}

	// 构建零件映射
	itemMap := make(map[int64][]models.OrderItem)
	for _, it := range items {
		it.Processes = processMap[it.ID]
		itemMap[it.OrderID] = append(itemMap[it.OrderID], it)
	}

	// 组装订单
	for i := range orders {
		orders[i].Items = itemMap[orders[i].ID]
		sortOrderItemsByDate(orders[i].Items, orders[i].StartDate)
	}

	fmt.Printf("[OrderRepo] GetWithFilters total took %v ms\n", time.Since(startTime).Milliseconds())
	return OrderListResult{Orders: orders, Total: total}, nil
}

func (r *OrderRepository) GetAllWithItems() ([]models.Order, error) {
	result, err := r.GetWithFilters(OrderFilters{})
	if err != nil {
		return nil, err
	}
	return result.Orders, nil
}

// Parts use their own business date, falling back only to their parent order.
// Undated parts stay last; IDs make equal dates stable across refreshes.
func sortOrderItemsByDate(items []models.OrderItem, orderDate *time.Time) {
	date := func(item models.OrderItem) *time.Time {
		if item.StartDate != nil {
			return item.StartDate
		}
		return orderDate
	}
	sort.SliceStable(items, func(i, j int) bool {
		a, b := date(items[i]), date(items[j])
		if a == nil && b != nil {
			return false
		}
		if a != nil && b == nil {
			return true
		}
		if a != nil && b != nil && !a.Equal(*b) {
			return a.After(*b)
		}
		return items[i].ID > items[j].ID
	})
}
