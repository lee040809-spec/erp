package repository

import (
	"errors"
	driver "github.com/go-sql-driver/mysql"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"machining-erp/internal/models"
	"machining-erp/internal/workflowtest"
	"os"
	"regexp"
	"strings"
	"testing"
	"time"
)

// Fixtures live only in connection-local temporary tables. Real data is read
// for table definitions and integrity counts, never modified.
func TestOrderIntegrityTemporaryTables(t *testing.T) {
	if os.Getenv("RUN_FINANCE_INTEGRATION") != "1" {
		t.Skip("requires explicit test database environment")
	}
	cfg := driver.NewConfig()
	cfg.User = os.Getenv("MYSQL_USER")
	cfg.Passwd = os.Getenv("MYSQL_PASSWORD")
	cfg.Net = "tcp"
	cfg.Addr = os.Getenv("MYSQL_HOST") + ":3306"
	cfg.DBName = os.Getenv("MYSQL_DATABASE")
	cfg.ParseTime = true
	db, err := gorm.Open(mysql.Open(cfg.FormatDSN()), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal("test database connection failed")
	}
	pool, _ := db.DB()
	pool.SetMaxOpenConns(1)
	pool.SetMaxIdleConns(1)
	defer pool.Close()
	check := func(err error) {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
	}
	// Read existing reference consistency before shadowing tables.
	for _, q := range []string{
		"SELECT COUNT(*) FROM order_items i LEFT JOIN orders o ON o.id=i.order_id WHERE o.id IS NULL OR i.corp_id<>o.corp_id",
		"SELECT COUNT(*) FROM order_processes p LEFT JOIN order_items i ON i.id=p.order_item_id WHERE i.id IS NULL OR p.corp_id<>i.corp_id",
		"SELECT COUNT(*) FROM production_completions p LEFT JOIN order_items i ON i.id=p.order_item_id LEFT JOIN orders o ON o.id=p.order_id WHERE i.id IS NULL OR o.id IS NULL OR p.corp_id<>i.corp_id OR p.order_id<>i.order_id OR p.corp_id<>o.corp_id",
		"SELECT COUNT(*) FROM cost_entries c LEFT JOIN orders o ON o.id=c.order_id LEFT JOIN order_items i ON i.id=c.order_item_id WHERE (c.order_id IS NOT NULL AND (o.id IS NULL OR c.corp_id<>o.corp_id)) OR (c.order_item_id IS NOT NULL AND (i.id IS NULL OR c.corp_id<>i.corp_id)) OR (c.order_id IS NOT NULL AND c.order_item_id IS NOT NULL AND c.order_id<>i.order_id)",
	} {
		var n int64
		check(db.Raw(q).Scan(&n).Error)
		t.Logf("existing reference inconsistencies: %d", n)
		if n != 0 {
			t.Fatal("existing data needs reconciliation")
		}
	}
	for _, name := range []string{"orders", "order_items", "order_processes", "process_time_events", "production_completions", "cost_entries", "production_plans", "order_business_records", "business_entry_policies"} {
		var table, ddl string
		check(pool.QueryRow("SHOW CREATE TABLE `"+name+"`").Scan(&table, &ddl))
		lines := strings.Split(ddl, "\n")
		kept := []string{}
		for _, line := range lines {
			if strings.Contains(line, "FOREIGN KEY") {
				continue
			}
			kept = append(kept, line)
		}
		ddl = strings.Join(kept, "\n")
		ddl = regexp.MustCompile(",\\s*\\)").ReplaceAllString(ddl, "\n)")
		ddl = strings.Replace(ddl, "CREATE TABLE", "CREATE TEMPORARY TABLE", 1)
		ddl = regexp.MustCompile("AUTO_INCREMENT=[0-9]+").ReplaceAllString(ddl, "AUTO_INCREMENT=1")
		check(db.Exec(ddl).Error)
	}
	workflowtest.Tables(t, db, "order_processes", "production_completions", "order_business_records", "cost_entries")
	repo := NewOrderRepository(db)
	fee := 100.0
	order := models.Order{CorpID: 0, OrderNumber: "TEMP-A"}
	items := []models.OrderItem{{PartName: "TEST", Quantity: 10, UnitPrice: 20, SupplyMode: "self_purchased", Currency: "CNY", MaterialCost: &fee, Processes: []models.OrderProcess{{Name: "TEST"}}}}
	_, err = repo.Create(&order, items)
	check(err)
	other := models.Order{CorpID: 1, OrderNumber: "TEMP-B"}
	others := []models.OrderItem{{PartName: "OTHER", Quantity: 1, SupplyMode: "self_purchased", Currency: "CNY", Processes: []models.OrderProcess{{Name: "OTHER"}}}}
	_, err = repo.Create(&other, others)
	check(err)
	for _, corp := range []int64{0, 1} {
		result, err := repo.GetWithFilters(OrderFilters{CorpID: corp, Page: 1, PageSize: 20})
		check(err)
		if result.Total != 1 || len(result.Orders) != 1 || result.Orders[0].CorpID != corp {
			t.Fatal("company isolation failed")
		}
	}
	if err := repo.Delete(other.ID, 0); !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatal("cross-company delete accepted")
	}
	item := items[0]
	item.MaterialCost = new(float64)
	item.SupplyMode = "customer_supplied"
	item.TaxInclusive = true
	check(repo.Update(&order, []models.OrderItem{item}, OrderUpdateOptions{TaxProvided: map[int64]bool{item.ID: true}}))
	var saved models.OrderItem
	check(db.First(&saved, item.ID).Error)
	if saved.MaterialCost == nil || *saved.MaterialCost != 0 || saved.SupplyMode != "customer_supplied" || !saved.TaxInclusive {
		t.Fatal("zero fees/supply/tax not saved")
	}
	item.MaterialCost = nil
	item.SupplyMode = ""
	item.Currency = ""
	item.TaxInclusive = false
	check(repo.Update(&order, []models.OrderItem{item}, OrderUpdateOptions{TaxProvided: map[int64]bool{}}))
	check(db.First(&saved, item.ID).Error)
	if saved.SupplyMode != "customer_supplied" || !saved.TaxInclusive || *saved.MaterialCost != 0 {
		t.Fatal("omitted fields overwritten")
	}
	// Explicit zero prices differ from omitted prices; client totals cannot win.
	item.UnitPrice = 0
	order.TotalAmount = 99999
	check(repo.Update(&order, []models.OrderItem{item}, OrderUpdateOptions{UnitPriceProvided: map[int64]bool{item.ID: true}}))
	check(db.First(&saved, item.ID).Error)
	if saved.UnitPrice != 0 || saved.TotalPrice != 0 || order.TotalAmount != 0 {
		t.Fatal("explicit zero price or derived amount failed")
	}
	item.UnitPrice = 20
	check(repo.Update(&order, []models.OrderItem{item}, OrderUpdateOptions{UnitPriceProvided: map[int64]bool{item.ID: true}}))
	item.UnitPrice = 0
	item.Quantity = 11
	item.ProductionQuantity = 11
	check(repo.Update(&order, []models.OrderItem{item}, OrderUpdateOptions{}))
	check(db.First(&saved, item.ID).Error)
	if saved.UnitPrice != 20 || saved.TotalPrice != 220 || order.TotalAmount != 220 {
		t.Fatal("omitted price not preserved or quantity amount not linked")
	}
	item.Quantity = 10
	item.UnitPrice = 777
	injectedFee := 999.0
	item.MaterialCost = &injectedFee
	item.Processes[0].OutsourcingFee = 888
	check(repo.Update(&order, []models.OrderItem{item}, OrderUpdateOptions{ProtectAmounts: true}))
	check(db.First(&saved, item.ID).Error)
	var savedProcess models.OrderProcess
	check(db.First(&savedProcess, item.Processes[0].ID).Error)
	if saved.UnitPrice != 20 || saved.MaterialCost == nil || *saved.MaterialCost != 0 || savedProcess.OutsourcingFee != 0 || order.TotalAmount != 200 {
		t.Fatal("production edit changed protected prices or fees")
	}
	item = saved
	item.Processes = []models.OrderProcess{savedProcess}
	// A failure at final order aggregation must roll back the changed item.
	injected := errors.New("temporary final save failure")
	check(db.Callback().Update().Before("gorm:update").Register("test:final_failure", func(tx *gorm.DB) {
		fields, ok := tx.Statement.Dest.(map[string]interface{})
		if ok && tx.Statement.Table == "orders" && fields["total_amount"] != nil {
			tx.AddError(injected)
		}
	}))
	item.Quantity = 12
	item.ProductionQuantity = 12
	if !errors.Is(repo.Update(&order, []models.OrderItem{item}), injected) {
		t.Fatal("final save error was lost")
	}
	check(db.Callback().Update().Remove("test:final_failure"))
	check(db.First(&saved, item.ID).Error)
	if saved.Quantity != 10 || saved.TotalPrice != 200 {
		t.Fatal("final save failure did not roll back item")
	}
	item = saved
	item.Processes = []models.OrderProcess{savedProcess}
	completedState := "completed"
	deliveredState := "delivered"
	if !errors.Is(repo.UpdateItemForCorp(0, item.ID, &completedState, nil), ErrCompletionState) || !errors.Is(repo.UpdateItemForCorp(0, item.ID, &deliveredState, nil), ErrDeliveryState) {
		t.Fatal("status endpoint bypassed qualified quantity")
	}
	future := time.Now().AddDate(0, 0, 1).Format("2006-01-02")
	if !errors.Is(repo.UpdateItemForCorp(0, item.ID, nil, &future), ErrCompletionDate) {
		t.Fatal("future actual date accepted")
	}
	// Completing a process must not invent qualified quantity or completion date.
	completed := "completed"
	is, os, err := repo.UpdateProcessForCorp(0, item.ID, item.Processes[0].ID, &completed, nil, nil)
	check(err)
	if is != "processing" || os != "processing" {
		t.Fatal("process bypassed qualified completion")
	}
	check(db.Create(&models.ProductionCompletion{CorpID: 0, OrderID: order.ID, OrderItemID: item.ID, CompletionNo: "TEMP-1", QualifiedQuantity: "5", BillableQuantity: "5", UnitPriceSnapshot: "20", OutputValue: "100", BusinessDate: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), ConfirmedAt: time.Now(), Status: "confirmed", SupplyModeSnapshot: "customer_supplied", Currency: "CNY", TaxInclusive: true, CreatedBy: 1}).Error)
	if !errors.Is(repo.UpdateItemForCorp(0, item.ID, &completedState, nil), ErrCompletionState) {
		t.Fatal("partial quantity marked completed")
	}
	if !errors.Is(repo.Delete(order.ID, 0), ErrPostedOrder) {
		t.Fatal("posted order deleted")
	}
	if !errors.Is(repo.Update(&order, []models.OrderItem{}), ErrPostedOrder) {
		t.Fatal("posted item deleted")
	}
	check(db.First(&item, item.ID).Error)
	item.Quantity = 4
	if !errors.Is(repo.Update(&order, []models.OrderItem{item}), ErrPostedQuantity) {
		t.Fatal("posted quantity reduced")
	}
	item.Quantity = 10
	item.SupplyMode = "self_purchased"
	if !errors.Is(repo.Update(&order, []models.OrderItem{item}), ErrPostedBasis) {
		t.Fatal("posted supply basis changed")
	}
	date := "2026-02-01"
	if !errors.Is(repo.UpdateItemForCorp(0, item.ID, nil, &date), ErrPostedBasis) {
		t.Fatal("posted date changed")
	}
	// A process from another company is rejected, preserving current state.
	if _, _, err = repo.UpdateProcessForCorp(0, item.ID, others[0].Processes[0].ID, &completed, nil, nil); err == nil {
		t.Fatal("foreign process updated")
	}
	item.SupplyMode = "customer_supplied"
	item.Processes = []models.OrderProcess{{ID: others[0].Processes[0].ID, Name: "BAD"}}
	if !errors.Is(repo.Update(&order, []models.OrderItem{item}), ErrOrderScope) {
		t.Fatal("foreign process attached")
	}
	var state models.OrderProcess
	check(db.First(&state, items[0].Processes[0].ID).Error)
	if state.Status != models.ProcessStatusCompleted {
		t.Fatal("failed transaction changed process")
	}
	check(db.Exec("UPDATE production_completions SET qualified_quantity=10,billable_quantity=10,output_value=200").Error)
	is, _, err = repo.UpdateProcessForCorp(0, item.ID, items[0].Processes[0].ID, &completed, nil, nil)
	check(err)
	if is != "completed" {
		t.Fatal("confirmed full quantity not reflected")
	}
	pendingState := "pending"
	if !errors.Is(repo.UpdateItemForCorp(0, item.ID, &pendingState, nil), ErrCompletionState) {
		t.Fatal("full confirmed quantity reopened")
	}
	if !errors.Is(repo.UpdateItemForCorp(0, item.ID, &deliveredState, nil), ErrDeliveryState) {
		t.Fatal("manual delivery status bypassed ledger")
	}
	check(db.Table("order_items").Where("id=?", item.ID).Updates(map[string]interface{}{"status": "delivered", "delivered_quantity": 10}).Error)
	if !errors.Is(repo.UpdateItemForCorp(0, item.ID, &completedState, nil), ErrDeliveryState) {
		t.Fatal("delivered item regressed")
	}
	check(repo.UpdateItemForCorp(0, item.ID, &deliveredState, nil))
	is, _, err = repo.UpdateProcessForCorp(0, item.ID, items[0].Processes[0].ID, &completed, nil, nil)
	check(err)
	if is != "delivered" {
		t.Fatal("delivered state overwritten")
	}
	fresh := models.Order{CorpID: 0, OrderNumber: "ALL-PROCESSES-DONE"}
	newItems := []models.OrderItem{{PartName: "NEW", Quantity: 1, UnitPrice: 2, Processes: []models.OrderProcess{{Name: "TEST", Status: models.ProcessStatusCompleted}}}}
	_, err = repo.Create(&fresh, newItems)
	check(err)
	if newItems[0].Status != models.OrderStatusProcessing {
		t.Fatal("new order bypassed qualified confirmation")
	}
	check(db.Exec("UPDATE order_items SET status='completed' WHERE id=?", newItems[0].ID).Error)
	check(repo.Update(&fresh, newItems))
	saved = models.OrderItem{}
	check(db.First(&saved, newItems[0].ID).Error)
	if saved.Status != models.OrderStatusCompleted {
		t.Fatal("historical status was rewritten without review")
	}

	// Scoped production adjustments leave order amount unchanged and protect existing batches.
	check(repo.UpdateProductionQuantities(0, item.ID, 15, 2))
	var buffered models.OrderItem
	check(db.First(&buffered, item.ID).Error)
	if buffered.Quantity != 10 || buffered.ProductionQuantity != 15 || buffered.ScrapQuantity != 2 || buffered.TotalPrice != 200 {
		t.Fatal("production buffer changed contracted amount")
	}
	if !errors.Is(repo.UpdateProductionQuantities(0, item.ID, 10, 2), ErrPostedQuantity) {
		t.Fatal("production cap below qualified plus scrap accepted")
	}
	if err := repo.UpdateProductionQuantities(1, item.ID, 15, 2); err == nil {
		t.Fatal("another company adjusted production quantities")
	}
	check(db.First(&buffered, item.ID).Error)
	if buffered.ProductionQuantity != 15 {
		t.Fatal("rejected adjustment was not rolled back")
	}

	// Forged delivery process state in generic order editing cannot bypass event records.
	guarded := models.Order{CorpID: 0, OrderNumber: "DELIVERY-GUARD"}
	guardedItems := []models.OrderItem{{PartName: "GUARD", Quantity: 2, Processes: []models.OrderProcess{{Name: "送货", Status: models.ProcessStatusCompleted}}}}
	_, err = repo.Create(&guarded, guardedItems)
	check(err)
	if guardedItems[0].Processes[0].Status != models.ProcessStatusPending {
		t.Fatal("new order fabricated delivery")
	}
	guardedItems[0].Processes[0].Status = models.ProcessStatusCompleted
	check(repo.Update(&guarded, guardedItems))
	var guardedProcess models.OrderProcess
	check(db.First(&guardedProcess, guardedItems[0].Processes[0].ID).Error)
	if guardedProcess.Status != models.ProcessStatusPending {
		t.Fatal("order editing bypassed delivery records")
	}

	// Fuzzy part-name search uses the same item for combined filters and paginates orders.
	yesterday := time.Now().AddDate(0, 0, -1)
	stringPointer := func(value string) *string { return &value }
	drawing := "data:image/png;base64," + strings.Repeat("A", 256<<10)
	searchA := models.Order{CorpID: 0, OrderNumber: "SEARCH-A"}
	searchB := models.Order{CorpID: 0, OrderNumber: "SEARCH-B"}
	searchOther := models.Order{CorpID: 1, OrderNumber: "SEARCH-OTHER"}
	_, err = repo.Create(&searchA, []models.OrderItem{{PartName: "精密单向阀座", DrawingData: &drawing, PartNumber: stringPointer("A-01"), Quantity: 1, DueDate: &yesterday}, {PartName: "套筒", PartNumber: stringPointer("B-02"), Quantity: 1}})
	check(err)
	_, err = repo.Create(&searchB, []models.OrderItem{{PartName: "阀座组件", PartNumber: stringPointer("B-03"), Quantity: 1, DueDate: &yesterday}})
	check(err)
	_, err = repo.Create(&searchOther, []models.OrderItem{{PartName: "阀座", PartNumber: stringPointer("B-04"), Quantity: 1}})
	check(err)
	first, err := repo.GetWithFilters(OrderFilters{CorpID: 0, PartName: "阀座", Page: 1, PageSize: 1})
	check(err)
	second, err := repo.GetWithFilters(OrderFilters{CorpID: 0, PartName: "阀座", Page: 2, PageSize: 1})
	check(err)
	if first.Total != 2 || second.Total != 2 || len(first.Orders) != 1 || len(second.Orders) != 1 || first.Orders[0].ID == second.Orders[0].ID || first.Orders[0].CorpID != 0 || second.Orders[0].CorpID != 0 {
		t.Fatal("fuzzy-name count, pagination or company isolation failed")
	}
	combined, err := repo.GetWithFilters(OrderFilters{CorpID: 0, PartName: "阀座", PartNumber: "B-", Page: 1, PageSize: 50})
	check(err)
	if combined.Total != 1 || len(combined.Orders) != 1 || combined.Orders[0].ID != searchB.ID {
		t.Fatal("part name and number matched different items")
	}
	searchFuture := time.Now().AddDate(0, 0, 10)
	searchMixed := models.Order{CorpID: 0, OrderNumber: "SEARCH-MIXED"}
	_, err = repo.Create(&searchMixed, []models.OrderItem{{PartName: "阀座未来交付", Quantity: 1, DueDate: &searchFuture}, {PartName: "已逾期的套筒", Quantity: 1, DueDate: &yesterday}})
	check(err)
	dated, err := repo.GetWithFilters(OrderFilters{CorpID: 0, PartName: "阀座", DateType: "overdue", Page: 1, PageSize: 50})
	check(err)
	if dated.Total != 2 || len(dated.Orders) != 2 {
		t.Fatal("fuzzy-name overdue filter failed")
	}
	empty, err := repo.GetWithFilters(OrderFilters{CorpID: 0, PartName: "不存在的零件", Page: 1, PageSize: 50})
	check(err)
	if empty.Total != 0 || len(empty.Orders) != 0 {
		t.Fatal("unmatched part name returned orders")
	}
	// A list contains only a flag, and saving it without a drawing preserves the original.
	listed, err := repo.GetWithFilters(OrderFilters{CorpID: 0, OrderNumber: "SEARCH-A", Page: 1, PageSize: 999999})
	check(err)
	var drawingItem models.OrderItem
	for _, row := range listed.Orders[0].Items {
		if row.PartName == "精密单向阀座" {
			drawingItem = row
		}
	}
	if !drawingItem.HasDrawing || drawingItem.DrawingData != nil {
		t.Fatal("list loaded drawing contents")
	}
	check(repo.Update(&listed.Orders[0], listed.Orders[0].Items))
	var storedDrawing string
	check(db.Table("order_items").Where("id=?", drawingItem.ID).Select("drawing_data").Scan(&storedDrawing).Error)
	if storedDrawing != drawing {
		t.Fatal("ordinary edit erased lazy-loaded drawing")
	}
	t.Run("new_order_prices_and_totals", func(t *testing.T) {
		start, _ := time.Parse("2006-01-02", "2026-10-09")
		due, _ := time.Parse("2006-01-02", "2026-11-30")
		priced := models.Order{CorpID: 0, OrderNumber: "PRICE-CHECK", StartDate: &start, DueDate: &due, TotalAmount: 999}
		pricedItems := []models.OrderItem{
			{PartName: "PRICE-A", Quantity: 1000, ProductionQuantity: 1200, UnitPrice: 3.20, TotalPrice: 999, SupplyMode: "self_purchased", Currency: "CNY", StartDate: &start, DueDate: &due, Processes: []models.OrderProcess{{Name: "车削"}, {Name: "送货"}}},
			{PartName: "PRICE-B", Quantity: 3, ProductionQuantity: 3, UnitPrice: 12.50, TotalPrice: 999, SupplyMode: "customer_supplied", Currency: "CNY", StartDate: &start, DueDate: &due, Processes: []models.OrderProcess{{Name: "送货"}}},
		}
		_, e := repo.Create(&priced, pricedItems)
		check(e)
		var storedOrder models.Order
		check(db.First(&storedOrder, priced.ID).Error)
		var storedItems []models.OrderItem
		check(db.Where("order_id=? AND corp_id=0", priced.ID).Order("id").Find(&storedItems).Error)
		if len(storedItems) != 2 || storedItems[0].UnitPrice != 3.2 || storedItems[0].TotalPrice != 3200 || storedItems[1].TotalPrice != 37.5 || storedOrder.TotalAmount != 3237.5 {
			t.Fatal("created price/quantity totals not persisted", storedOrder.TotalAmount, storedItems)
		}
		pricedItems[0].Quantity = 1001
		pricedItems[0].UnitPrice = 3.45
		check(repo.Update(&priced, pricedItems, OrderUpdateOptions{UnitPriceProvided: map[int64]bool{pricedItems[0].ID: true, pricedItems[1].ID: true}}))
		if priced.TotalAmount != 3490.95 {
			t.Fatal("edited quantities/prices did not recalculate", priced.TotalAmount)
		}
		pricedItems[1].UnitPrice = 0
		check(repo.Update(&priced, pricedItems, OrderUpdateOptions{UnitPriceProvided: map[int64]bool{pricedItems[0].ID: true, pricedItems[1].ID: true}}))
		if priced.TotalAmount != 3453.45 {
			t.Fatal("explicit zero price not persisted", priced.TotalAmount)
		}
		t.Log("new order price storage, server derived totals, decimal cents, overproduction exclusion and explicit zero verified")
	})
	t.Log("drawing flags and omitted-drawing edit preservation verified")
	t.Log("fuzzy-name search, combined same-item filters, pagination, overdue and company isolation passed")
	t.Log("production quantity guards and cross-company isolation passed")
	t.Log("zero and omitted prices, protected fees, decimal totals, rollback, explicit completion guards, delivered protection and legacy preservation passed")
}
