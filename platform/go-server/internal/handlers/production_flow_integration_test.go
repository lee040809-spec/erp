package handlers

import (
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	driver "github.com/go-sql-driver/mysql"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"machining-erp/internal/models"
	"machining-erp/internal/outsourcing"
	"machining-erp/internal/productionflow"
	"machining-erp/internal/repository"
	"machining-erp/internal/workflowtest"
	"net/http/httptest"
	"os"
	"regexp"
	"strings"
	"testing"
	"time"
)

func TestProductionFlowIntegrationTemporaryTables(t *testing.T) {
	if os.Getenv("RUN_FINANCE_INTEGRATION") != "1" {
		t.Skip("explicit test database required")
	}
	cfg := driver.NewConfig()
	cfg.User = os.Getenv("MYSQL_USER")
	cfg.Passwd = os.Getenv("MYSQL_PASSWORD")
	cfg.Net = "tcp"
	cfg.Addr = os.Getenv("MYSQL_HOST") + ":3306"
	cfg.DBName = os.Getenv("MYSQL_DATABASE")
	cfg.ParseTime = true
	db, e := gorm.Open(mysql.Open(cfg.FormatDSN()), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if e != nil {
		t.Fatal("database unavailable")
	}
	pool, _ := db.DB()
	pool.SetMaxOpenConns(1)
	pool.SetMaxIdleConns(1)
	defer pool.Close()
	check := func(e error) {
		t.Helper()
		if e != nil {
			t.Fatal(e)
		}
	}
	exec := func(q string) { t.Helper(); check(db.Exec(q).Error) }
	// Definitions are read; every write below targets a connection-local TEMPORARY table.
	for _, name := range []string{"orders", "order_items", "order_processes", "process_time_events", "production_completions", "order_business_records", "cost_entries", "cost_allocations", "monthly_closings", "business_entry_policies", "production_plans"} {
		var table, ddl string
		check(pool.QueryRow("SHOW CREATE TABLE `"+name+"`").Scan(&table, &ddl))
		kept := []string{}
		for _, line := range strings.Split(ddl, "\n") {
			if !strings.Contains(line, "FOREIGN KEY") {
				kept = append(kept, line)
			}
		}
		ddl = regexp.MustCompile(",\\s*\\)").ReplaceAllString(strings.Join(kept, "\n"), "\n)")
		ddl = strings.Replace(ddl, "CREATE TABLE", "CREATE TEMPORARY TABLE", 1)
		ddl = regexp.MustCompile("AUTO_INCREMENT=[0-9]+").ReplaceAllString(ddl, "AUTO_INCREMENT=1")
		exec(ddl)
	}
	workflowtest.Tables(t, db, "order_processes", "production_completions", "order_business_records", "cost_entries")
	date := productionflow.Today()
	start, _ := time.Parse("2006-01-02", date)
	start = start.AddDate(0, 0, -3)
	due := start.AddDate(0, 1, 0)
	repo := repository.NewOrderRepository(db)
	order := models.Order{CorpID: 0, OrderNumber: "FLOW-TEMP", StartDate: &start, DueDate: &due}
	part := models.OrderItem{PartName: "PARTIAL", Quantity: 1000, ProductionQuantity: 1100, UnitPrice: 2, StartDate: &start, DueDate: &due, Currency: "CNY", SupplyMode: "self_purchased", DeliveryPlan: []models.DeliveryBatchInput{{BatchNo: "首批", Quantity: 500, ProductionQuantity: 500, DueDate: date}, {BatchNo: "余量", Quantity: 500, ProductionQuantity: 600, DueDate: due.Format("2006-01-02")}}, Processes: []models.OrderProcess{{Name: "下料", Status: "pending"}, {Name: "铣削", Status: "pending"}, {Name: "检验", Status: "pending"}}}
	_, e = repo.Create(&order, []models.OrderItem{part}, 1)
	check(e)
	var item models.OrderItem
	check(db.Where("order_id=?", order.ID).Take(&item).Error)
	s, e := productionflow.Read(db, 0, item.ID)
	check(e)
	if !s.Tracking || len(s.Batches) != 2 {
		t.Fatal("new-order batch plan not saved atomically")
	}
	counter := 0
	report := func(index int, kind string, qty int) {
		t.Helper()
		counter++
		check(productionflow.Report(db, 0, item.ID, s.Processes[index].ID, 1, productionflow.ReportInput{Kind: kind, Quantity: qty, Date: date, RequestID: fmt.Sprintf("test-%d", counter)}))
	}
	report(0, "start", 1100)
	t.Run("started_route_edit_preserves_history", func(t *testing.T) {
		var current []models.OrderProcess
		check(db.Where("corp_id=0 AND order_item_id=?", item.ID).Order("sort_order,id").Find(&current).Error)
		original := current[0]
		current[0].Name = "下料（工艺修订）"
		current[1], current[2] = current[2], current[1]
		current = append(current, models.OrderProcess{Name: "新增检验", Status: "pending"})
		check(repo.UpdateRouteForCorp(0, item.ID, 1, current, nil))
		revised, e := productionflow.Read(db, 0, item.ID)
		check(e)
		if len(revised.Processes) != 4 || revised.Processes[0].ID != original.ID || revised.Processes[0].Started != 1100 || revised.Processes[3].Tracking != true || revised.Processes[3].Started != 0 {
			t.Fatal("route edit reset quantities, IDs or missed new-step tracking")
		}
		if revised.Processes[0].ActualStartAt == nil || !revised.Processes[0].ActualStartAt.Equal(*original.ActualStartAt) {
			t.Fatal("route edit changed actual start")
		}
		// Unused new steps can be deleted; restore the old sequence for the remaining workflow.
		current = []models.OrderProcess{revised.Processes[0].OrderProcess, revised.Processes[2].OrderProcess, revised.Processes[1].OrderProcess}
		check(repo.UpdateRouteForCorp(0, item.ID, 1, current, nil))
		if e := repo.UpdateRouteForCorp(0, item.ID, 1, current[1:], nil); e == nil {
			t.Fatal("started step deleted")
		}
		var events int64
		check(db.Model(&models.ProcessTimeEvent{}).Where("corp_id=0 AND order_item_id=? AND source='route'", item.ID).Count(&events).Error)
		if events == 0 {
			t.Fatal("route changes were not audited")
		}
	})

	report(0, "complete", 500)
	if e = productionflow.Report(db, 0, item.ID, s.Processes[1].ID, 1, productionflow.ReportInput{Kind: "start", Quantity: 1, Date: start.AddDate(0, 0, 1).Format("2006-01-02"), RequestID: "consume-future"}); e == nil {
		t.Fatal("backdated downstream consumed future output")
	}
	report(1, "start", 500)
	t.Run("started_downstream_cannot_lose_supply", func(t *testing.T) {
		var current []models.OrderProcess
		check(db.Where("corp_id=0 AND order_item_id=?", item.ID).Order("sort_order,id").Find(&current).Error)
		invalid := append([]models.OrderProcess{current[0], {Name: "误插前道", Status: "pending"}}, current[1:]...)
		if e := repo.UpdateRouteForCorp(0, item.ID, 1, invalid, nil); e == nil {
			t.Fatal("new empty upstream permitted before an already started step")
		}
		snapshot, e := productionflow.Read(db, 0, item.ID)
		check(e)
		if len(snapshot.Processes) != 3 || snapshot.Processes[1].Started != 500 {
			t.Fatal("failed route change was not rolled back")
		}
	})

	if e = productionflow.Report(db, 0, item.ID, s.Processes[1].ID, 1, productionflow.ReportInput{Kind: "start", Quantity: 1, Date: date, RequestID: "too-much"}); e == nil {
		t.Fatal("downstream consumed unfinished upstream")
	}
	report(1, "complete", 500)
	report(2, "start", 500)
	report(2, "complete", 500)
	finance := NewProductionFinanceHandler(db)
	business := NewBusinessEventHandler(db)
	gin.SetMode(gin.TestMode)
	call := func(fn gin.HandlerFunc, body string, params gin.Params, want int) {
		t.Helper()
		rec := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(rec)
		c.Request = httptest.NewRequest("POST", "/", strings.NewReader(body))
		c.Request.Header.Set("Content-Type", "application/json")
		c.Params = params
		c.Set("userID", int64(1))
		c.Set("corpID", int64(0))
		c.Set("roleType", "admin")
		fn(c)
		if rec.Code != want {
			t.Fatalf("status=%d wanted=%d: %s", rec.Code, want, rec.Body.String())
		}
	}
	complete := func(batch int64, qty int, request string, want int) {
		t.Helper()
		call(finance.CreateCompletion, fmt.Sprintf(`{"delivery_batch_id":%d,"qualified_quantity":%d,"business_date":%q,"request_id":%q}`, batch, qty, date, request), gin.Params{{Key: "itemId", Value: fmt.Sprint(item.ID)}}, want)
	}
	b1, b2 := s.Batches[0].ID, s.Batches[1].ID
	call(finance.CreateCompletion, fmt.Sprintf(`{"delivery_batch_id":%d,"qualified_quantity":500,"business_date":%q,"request_id":"backdated-complete"}`, b1, start.AddDate(0, 0, 1).Format("2006-01-02")), gin.Params{{Key: "itemId", Value: fmt.Sprint(item.ID)}}, 400)
	complete(b1, 500, "partial-first", 200)
	complete(b1, 500, "partial-first", 200)
	complete(b2, 1, "before-last", 400)
	call(business.Create, fmt.Sprintf(`{"kind":"delivery","delivery_batch_id":%d,"order_item_id":%d,"quantity":500,"business_date":%q,"request_id":"ship-first"}`, b1, item.ID, date), gin.Params{{Key: "id", Value: fmt.Sprint(order.ID)}}, 200)
	s, e = productionflow.Read(db, 0, item.ID)
	check(e)
	if s.Batches[0].DeliveredQuantity != 500 || s.Processes[0].Completed != 500 || s.Item.Status == "delivered" {
		t.Fatal("first delivery incorrectly closed whole production")
	}
	report(0, "complete", 600)
	report(1, "start", 600)
	report(1, "complete", 600)
	report(2, "start", 600)
	report(2, "complete", 600)
	complete(b2, 600, "partial-rest", 200)
	var totals struct {
		Qualified int
		Billable  int
		Amount    string
	}
	check(db.Raw("SELECT CAST(SUM(qualified_quantity) AS SIGNED) AS qualified,CAST(SUM(billable_quantity) AS SIGNED) AS billable,CAST(SUM(output_value) AS CHAR) AS amount FROM production_completions WHERE order_item_id=?", item.ID).Scan(&totals).Error)
	if totals.Qualified != 1100 || totals.Billable != 1000 || !sameDecimal(totals.Amount, "2000") {
		t.Fatalf("reserve billed or double completion: %+v", totals)
	}
	call(business.Create, fmt.Sprintf(`{"kind":"delivery","delivery_batch_id":%d,"order_item_id":%d,"quantity":500,"business_date":%q,"request_id":"ship-rest"}`, b2, item.ID, date), gin.Params{{Key: "id", Value: fmt.Sprint(order.ID)}}, 200)
	complete(b2, 1, "over-cap", 400)
	if e = repo.Delete(order.ID, 0); e == nil {
		t.Fatal("posted batch order deleted")
	}
	t.Log("first 500 can complete and ship while remaining 600 share the same route; billable cap=1000, reserve=100")

	outOrder := models.Order{CorpID: 0, OrderNumber: "OUTSOURCE-TEMP", StartDate: &start, DueDate: &due}
	_, e = repo.Create(&outOrder, []models.OrderItem{{PartName: "OUTSOURCE", Quantity: 500, ProductionQuantity: 500, UnitPrice: 3, StartDate: &start, DueDate: &due, Currency: "CNY", SupplyMode: "self_purchased", Processes: []models.OrderProcess{{Name: "下料", Status: "pending"}, {Name: "表面处理", Status: "pending", IsOutsourced: true}}}}, 1)
	check(e)
	var oi models.OrderItem
	check(db.Where("order_id=?", outOrder.ID).Take(&oi).Error)
	check(db.Transaction(func(tx *gorm.DB) error {
		i, e := productionflow.LockItem(tx, 0, oi.ID)
		if e != nil {
			return e
		}
		return productionflow.Activate(tx, i, nil, 1)
	}))
	osnap, e := productionflow.Read(db, 0, oi.ID)
	check(e)
	for _, kind := range []string{"start", "complete"} {
		check(productionflow.Report(db, 0, oi.ID, osnap.Processes[0].ID, 1, productionflow.ReportInput{Kind: kind, Quantity: 500, Date: date, RequestID: "cut-" + kind}))
	}
	input := outsourcing.DispatchInput{ItemID: oi.ID, ProcessID: osnap.Processes[1].ID, Supplier: "TEST-SUPPLIER", Quantity: 500, UnitPrice: "2", Date: date, Expected: date, RequestID: "send-500"}
	d, e := outsourcing.Dispatch(db, 0, 1, input)
	check(e)
	dReplay, e := outsourcing.Dispatch(db, 0, 1, input)
	check(e)
	if d.ID != dReplay.ID {
		t.Fatal("dispatch duplicated")
	}

	t.Run("outsourcing_record_does_not_lock_unrelated_steps", func(t *testing.T) {
		var current []models.OrderProcess
		check(db.Where("corp_id=0 AND order_item_id=?", oi.ID).Order("sort_order,id").Find(&current).Error)
		current[1].Name = "表面处理（工艺修订）"
		expanded := append(append([]models.OrderProcess{}, current...), models.OrderProcess{Name: "后续检验", Status: "pending"})
		check(repo.UpdateRouteForCorp(0, oi.ID, 1, expanded, nil))
		snapshot, e := productionflow.Read(db, 0, oi.ID)
		check(e)
		if len(snapshot.Processes) != 3 || snapshot.Processes[1].ID != osnap.Processes[1].ID || snapshot.Processes[1].Started != 500 {
			t.Fatal("outsourcing-linked route change lost production quantity")
		}
		unchanged := []models.OrderProcess{snapshot.Processes[0].OrderProcess, snapshot.Processes[1].OrderProcess}
		check(repo.UpdateRouteForCorp(0, oi.ID, 1, unchanged, nil))
		unchanged[1].IsOutsourced = false
		if e := repo.UpdateRouteForCorp(0, oi.ID, 1, unchanged, nil); e == nil {
			t.Fatal("linked outsource step changed to internal production")
		}
	})

	t.Run("outstanding_external_work_cannot_be_retired", func(t *testing.T) {
		var ps []models.OrderProcess
		check(db.Where("corp_id=0 AND order_item_id=?", oi.ID).Order("sort_order,id").Find(&ps).Error)
		if e := repo.UpdateRouteForCorp(0, oi.ID, 1, ps[:1], nil, repository.RouteChangeOptions{AllowProcessedChanges: true, Reason: "仍有待回厂"}); e == nil {
			t.Fatal("outstanding external transaction detached")
		}
	})
	input.Quantity = 501
	if _, e = outsourcing.Dispatch(db, 0, 1, input); e == nil {
		t.Fatal("same request changed silently")
	}
	input.Quantity = 500
	if _, e = outsourcing.Dispatch(db, 1, 1, input); e == nil {
		t.Fatal("foreign company accessed part")
	}
	receipt := outsourcing.ReturnInput{Kind: "return", Good: 445, Concession: 5, Rework: 30, Scrap: 20, Date: date, Notes: "质量核对", RequestID: "receive-500"}
	check(outsourcing.Receive(db, 0, d.ID, 1, receipt))
	check(outsourcing.Receive(db, 0, d.ID, 1, receipt))
	if _, e = outsourcing.Settle(db, 0, d.ID, 1, outsourcing.SettlementInput{Amount: "900", Date: date}); e == nil {
		t.Fatal("unresolved rework settled")
	}
	check(outsourcing.SendRework(db, 0, d.ID, 1, outsourcing.ReworkInput{Quantity: 30, Date: date, RequestID: "rework-send"}))
	check(outsourcing.Receive(db, 0, d.ID, 1, outsourcing.ReturnInput{Kind: "rework_return", Good: 20, Scrap: 10, Date: date, Notes: "返工后二次报废", RequestID: "rework-receive"}))
	osnap, e = productionflow.Read(db, 0, oi.ID)
	check(e)
	if osnap.Item.ScrapQuantity != 30 || osnap.Processes[1].Completed != 470 || osnap.AvailableCompletion != 470 {
		t.Fatalf("quality quantities wrong: %+v", osnap)
	}
	call(finance.CreateCompletion, fmt.Sprintf(`{"qualified_quantity":470,"business_date":%q,"request_id":"qualified-outsource"}`, date), gin.Params{{Key: "itemId", Value: fmt.Sprint(oi.ID)}}, 200)
	settlement := outsourcing.SettlementInput{Amount: "940", Tax: "40", Net: "900", Date: date, Reference: "TEST-RECON"}
	settled, e := outsourcing.Settle(db, 0, d.ID, 1, settlement)
	check(e)
	again, e := outsourcing.Settle(db, 0, d.ID, 1, settlement)
	check(e)
	if settled.CostEntryID == nil || again.ID != settled.ID {
		t.Fatal("settlement cost not atomic/idempotent")
	}
	var pc models.ProductionCompletion
	check(db.Where("order_item_id=?", oi.ID).Take(&pc).Error)
	var otherPC models.ProductionCompletion
	check(db.Where("order_item_id=?", item.ID).Take(&otherPC).Error)
	call(finance.AllocateCost, fmt.Sprintf(`{"cost_entry_id":%d,"production_completion_id":%d,"allocated_amount":"10"}`, *settled.CostEntryID, otherPC.ID), nil, 400)
	call(finance.AllocateCost, fmt.Sprintf(`{"cost_entry_id":%d,"production_completion_id":%d,"allocated_amount":"900","allocation_basis":"TEST"}`, *settled.CostEntryID, pc.ID), nil, 200)
	call(finance.AllocateCost, fmt.Sprintf(`{"cost_entry_id":%d,"production_completion_id":%d,"allocated_amount":"1"}`, *settled.CostEntryID, pc.ID), nil, 400)
	var costs int64
	check(db.Table("cost_entries").Count(&costs).Error)
	if costs != 1 {
		t.Fatal("outsourcing cost repeated")
	}
	t.Log("outsourcing 500 → accepted470 / scrap30; rework not double-started; exact net cost900 posted and allocated once")

	t.Run("settled_external_history_survives_retirement", func(t *testing.T) {
		var ps []models.OrderProcess
		check(db.Where("corp_id=0 AND order_item_id=?", oi.ID).Order("sort_order,id").Find(&ps).Error)
		check(repo.UpdateRouteForCorp(0, oi.ID, 1, ps[:1], nil, repository.RouteChangeOptions{AllowProcessedChanges: true, Reason: "取消已结束外协工序"}))
		view, e := productionflow.Read(db, 0, oi.ID)
		check(e)
		if view.AvailableCompletion != 0 || view.Item.ScrapQuantity != 30 {
			t.Fatal("retired external scrap duplicated good output")
		}
		var archived models.OrderProcess
		check(db.Unscoped().First(&archived, ps[1].ID).Error)
		if !archived.DeletedAt.Valid {
			t.Fatal("external history missing")
		}
		var count int64
		check(db.Model(&models.OutsourcingSettlement{}).Where("dispatch_id=?", d.ID).Count(&count).Error)
		if count != 1 {
			t.Fatal("retirement erased settlement")
		}
		if e := productionflow.CompletionReady(db, 0, oi.ID, 1, nil, date); e == nil {
			t.Fatal("scrapped external units were counted again")
		}
	})
	var scrapRecord BusinessRecord
	check(db.Where("order_item_id=? AND source_type='process_quantity'", oi.ID).Take(&scrapRecord).Error)
	call(business.Reverse, `{"reason":"cannot desync automated scrap"}`, gin.Params{{Key: "recordId", Value: fmt.Sprint(scrapRecord.ID)}}, 400)
	check(db.Exec("UPDATE monthly_closings SET status='closed' WHERE corp_id=0 AND accounting_month=?", date[:7]).Error)
	call(finance.CreateCompletion, fmt.Sprintf(`{"qualified_quantity":1,"business_date":%q,"request_id":"closed-flow"}`, date), gin.Params{{Key: "itemId", Value: fmt.Sprint(oi.ID)}}, 400)
	if _, e = outsourcing.Settle(db, 0, d.ID, 1, settlement); e == nil {
		t.Fatal("closed cost month not protected")
	}
	// Production user without finance cost authority cannot settle.
	rec := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(rec)
	ctx.Set("corpID", int64(0))
	ctx.Set("userID", int64(99))
	ctx.Set("roleType", "business")
	exec("CREATE TEMPORARY TABLE users(id BIGINT,corp_id BIGINT,role_id BIGINT)")
	exec("CREATE TEMPORARY TABLE permissions(role_id BIGINT,corp_id BIGINT,resource_id BIGINT)")
	exec("CREATE TEMPORARY TABLE resources(id BIGINT,resource_key VARCHAR(100))")
	NewOutsourcingHandler(db).Settle(ctx)
	if rec.Code != 403 {
		b, _ := json.Marshal(rec.Body.String())
		t.Fatalf("cost authority bypassed: %s", b)
	}
}

func TestProcessedRouteIntegrationTemporaryTables(t *testing.T) {
	if os.Getenv("RUN_FINANCE_INTEGRATION") != "1" {
		t.Skip("explicit test database required")
	}
	cfg := driver.NewConfig()
	cfg.User = os.Getenv("MYSQL_USER")
	cfg.Passwd = os.Getenv("MYSQL_PASSWORD")
	cfg.Net = "tcp"
	cfg.Addr = os.Getenv("MYSQL_HOST") + ":3306"
	cfg.DBName = os.Getenv("MYSQL_DATABASE")
	cfg.ParseTime = true
	db, e := gorm.Open(mysql.Open(cfg.FormatDSN()), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if e != nil {
		t.Fatal("database unavailable")
	}
	pool, _ := db.DB()
	pool.SetMaxOpenConns(1)
	pool.SetMaxIdleConns(1)
	defer pool.Close()
	check := func(e error) {
		t.Helper()
		if e != nil {
			t.Fatal(e)
		}
	}
	exec := func(q string) { t.Helper(); check(db.Exec(q).Error) }
	// Definitions are read; every write below targets a connection-local TEMPORARY table.
	for _, name := range []string{"orders", "order_items", "order_processes", "process_time_events", "production_completions", "order_business_records", "cost_entries", "cost_allocations", "monthly_closings", "business_entry_policies", "production_plans"} {
		var table, ddl string
		check(pool.QueryRow("SHOW CREATE TABLE `"+name+"`").Scan(&table, &ddl))
		kept := []string{}
		for _, line := range strings.Split(ddl, "\n") {
			if !strings.Contains(line, "FOREIGN KEY") {
				kept = append(kept, line)
			}
		}
		ddl = regexp.MustCompile(",\\s*\\)").ReplaceAllString(strings.Join(kept, "\n"), "\n)")
		ddl = strings.Replace(ddl, "CREATE TABLE", "CREATE TEMPORARY TABLE", 1)
		ddl = regexp.MustCompile("AUTO_INCREMENT=[0-9]+").ReplaceAllString(ddl, "AUTO_INCREMENT=1")
		exec(ddl)
	}
	workflowtest.Tables(t, db, "order_processes", "production_completions", "order_business_records", "cost_entries")
	date := productionflow.Today()
	start, _ := time.Parse("2006-01-02", date)
	start = start.AddDate(0, 0, -3)
	due := start.AddDate(0, 1, 0)

	repo := repository.NewOrderRepository(db)
	exec("CREATE TEMPORARY TABLE users(id BIGINT,corp_id BIGINT,role_id BIGINT)")
	exec("CREATE TEMPORARY TABLE permissions(role_id BIGINT,corp_id BIGINT,resource_id BIGINT)")
	exec("CREATE TEMPORARY TABLE resources(id BIGINT,resource_key VARCHAR(100))")

	order := models.Order{CorpID: 0, OrderNumber: "RETIRED-TEST", StartDate: &start, DueDate: &due}
	part := models.OrderItem{PartName: "RETIRED-TEST", Quantity: 100, ProductionQuantity: 100, UnitPrice: 2, SupplyMode: "self_purchased", Currency: "CNY", Processes: []models.OrderProcess{{Name: "下料", Status: "pending"}, {Name: "铣削", Status: "pending"}, {Name: "检验", Status: "pending"}, {Name: "送货", Status: "pending"}}}
	_, e = repo.Create(&order, []models.OrderItem{part}, 1)
	check(e)
	var item models.OrderItem
	check(db.Where("order_id=?", order.ID).Take(&item).Error)
	check(db.Transaction(func(tx *gorm.DB) error {
		i, e := productionflow.LockItem(tx, 0, item.ID)
		if e != nil {
			return e
		}
		return productionflow.Activate(tx, i, nil, 1)
	}))
	original, e := productionflow.Read(db, 0, item.ID)
	check(e)
	count := 0
	report := func(process int64, kind string, q int, wantError bool) {
		t.Helper()
		count++
		e := productionflow.Report(db, 0, item.ID, process, 1, productionflow.ReportInput{Kind: kind, Quantity: q, Date: date, RequestID: fmt.Sprintf("retire-%d", count)})
		if wantError {
			if e == nil {
				t.Fatal("invalid quantity accepted")
			}
		} else {
			check(e)
		}
	}
	cut, mill, inspection := original.Processes[0].ID, original.Processes[1].ID, original.Processes[2].ID
	report(cut, "start", 100, false)
	report(cut, "complete", 100, false)
	report(mill, "start", 60, false)
	report(mill, "complete", 40, false)
	report(mill, "scrap", 5, false)
	h := NewProcessTimelineHandler(db)
	call := func(fn gin.HandlerFunc, itemID int64, body string, role string, want int) map[string]interface{} {
		t.Helper()
		rec := httptest.NewRecorder()
		ctx, _ := gin.CreateTestContext(rec)
		ctx.Set("corpID", int64(0))
		ctx.Set("userID", int64(1))
		ctx.Set("roleType", role)
		ctx.Params = gin.Params{{Key: "itemId", Value: fmt.Sprint(itemID)}}
		ctx.Request = httptest.NewRequest("PATCH", "/", strings.NewReader(body))
		ctx.Request.Header.Set("Content-Type", "application/json")
		fn(ctx)
		if rec.Code != want {
			t.Fatalf("HTTP want %d got %d: %s", want, rec.Code, rec.Body.String())
		}
		var out struct{ Data map[string]interface{} }
		json.Unmarshal(rec.Body.Bytes(), &out)
		return out.Data
	}
	load := func() []models.OrderProcess {
		t.Helper()
		var ps []models.OrderProcess
		check(db.Where("corp_id=0 AND order_item_id=?", item.ID).Order("sort_order,id").Find(&ps).Error)
		return ps
	}
	payload := func(ps []models.OrderProcess, allow bool, reason string) string {
		data, e := json.Marshal(map[string]interface{}{"processes": ps, "allow_processed_changes": allow, "reason": reason})
		check(e)
		return string(data)
	}
	current := load()
	current[1].Name = "铣削（已开工工艺修订）"
	current[1].Status = "pending"
	call(h.SaveRoute, item.ID, payload(current, true, "更正工序名称"), "admin", 200)
	var retained models.OrderProcess
	check(db.First(&retained, mill).Error)
	if retained.Status != "processing" || retained.ActualStartAt == nil {
		t.Fatal("amendment reset actual production")
	}
	current = load()
	current[0].IsOutsourced = true
	call(h.SaveRoute, item.ID, payload(current, true, "更正已完成工序的加工方式"), "admin", 200)
	current = load()
	current[0].IsOutsourced = false
	call(h.SaveRoute, item.ID, payload(current, true, "恢复加工方式"), "admin", 200)
	current = load()
	next := []models.OrderProcess{current[0], current[2], current[3]}
	call(h.SaveRoute, item.ID, payload(next, true, ""), "admin", 400)
	call(h.SaveRoute, item.ID, payload(next, false, "未确认删除已加工工序"), "admin", 400)
	call(h.SaveRoute, item.ID, payload(next, true, "工艺变更，取消铣削并保留报废记录"), "business", 403)
	var beforeEvents int64
	check(db.Model(&models.ProcessQuantityEvent{}).Where("order_item_id=?", item.ID).Count(&beforeEvents).Error)
	call(h.SaveRoute, item.ID, payload(next, true, "工艺变更，取消铣削并保留报废记录"), "admin", 200)
	archived := models.OrderProcess{}
	check(db.Unscoped().First(&archived, mill).Error)
	if !archived.DeletedAt.Valid || archived.Status != "processing" || archived.ActualStartAt == nil {
		t.Fatal("removed step lost history")
	}
	if db.First(&models.OrderProcess{}, mill).Error != gorm.ErrRecordNotFound {
		t.Fatal("removed step remains active")
	}
	var totals models.ProcessQuantity
	check(db.Where("order_process_id=?", mill).Take(&totals).Error)
	if totals.StartedQuantity != 60 || totals.CompletedQuantity != 40 || totals.ScrapQuantity != 5 {
		t.Fatal("retired totals changed")
	}
	var events int64
	check(db.Model(&models.ProcessQuantityEvent{}).Where("order_item_id=?", item.ID).Count(&events).Error)
	if beforeEvents != events {
		t.Fatal("route retirement fabricated or erased reports")
	}
	revised, e := productionflow.Read(db, 0, item.ID)
	check(e)
	if len(revised.Processes) != 2 || revised.Processes[1].AvailableStart != 95 || revised.Item.ScrapQuantity != 5 || revised.HistoricalProcessNames[mill] != archived.Name {
		t.Fatalf("retired scrap/history mismatch: %+v", revised)
	}
	report(inspection, "start", 96, true)
	report(inspection, "start", 95, false)
	report(inspection, "complete", 95, false)
	finance := NewProductionFinanceHandler(db)
	call(finance.CreateCompletion, item.ID, fmt.Sprintf("{\"qualified_quantity\":95,\"business_date\":%q,\"request_id\":\"retired-qualified\"}", date), "admin", 200)
	current = load()
	call(h.SaveRoute, item.ID, payload([]models.OrderProcess{current[0], current[2]}, true, "取消末道检验，保留原合格确认"), "admin", 200)
	revised, e = productionflow.Read(db, 0, item.ID)
	check(e)
	if len(revised.Processes) != 1 || revised.AvailableCompletion != 0 {
		t.Fatalf("retired successor lost scrap anchor or duplicated output: %+v", revised)
	}
	if e := productionflow.CompletionReady(db, 0, item.ID, 1, nil, date); e == nil {
		t.Fatal("retired terminal duplicated confirmed output")
	}
	data := call(h.List, item.ID, "", "admin", 200)
	if len(data["archived_processes"].([]interface{})) != 2 {
		t.Fatal("timeline omitted archived history")
	}
	current = load()
	call(h.SaveRoute, item.ID, payload([]models.OrderProcess{current[1]}, true, "不能删除全部生产工序"), "admin", 400)
	var pc models.ProductionCompletion
	check(db.Where("order_item_id=?", item.ID).Take(&pc).Error)
	if pc.QualifiedQuantity != "95.000" && pc.QualifiedQuantity != "95" {
		t.Fatalf("posted quantity changed: %s", pc.QualifiedQuantity)
	}
	t.Log("in-progress and completed retirement preserves reports, scrap, qualified output, successor anchors, audit and permissions")
}
