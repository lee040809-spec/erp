package handlers

import (
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

func TestOutsourcingOrdersIntegrationTemporaryTables(t *testing.T) {
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
	fail := func(e error) {
		t.Helper()
		if e == nil {
			t.Fatal("expected operation to be rejected")
		}
	}
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
		check(db.Exec(ddl).Error)
	}
	workflowtest.Tables(t, db, "production_completions", "order_business_records", "cost_entries")
	today := productionflow.Today()
	date, _ := time.Parse("2006-01-02", today)
	start := date.AddDate(0, 0, -2)
	due := date.AddDate(0, 1, 0)
	a, e := outsourcing.SaveSupplier(db, 0, 0, outsourcing.SupplierInput{Name: "供应商甲", Active: true, Capabilities: "整套加工"})
	check(e)
	b, e := outsourcing.SaveSupplier(db, 0, 0, outsourcing.SupplierInput{Name: "供应商乙", Active: true})
	check(e)
	_, e = outsourcing.SaveSupplier(db, 0, 0, outsourcing.SupplierInput{Name: a.Name, Active: true})
	fail(e)
	_, e = outsourcing.SaveSupplier(db, 99, a.ID, outsourcing.SupplierInput{Name: a.Name, Active: true})
	fail(e)
	repo := repository.NewOrderRepository(db)
	newItem := func(name string) (models.OrderItem, []models.OrderProcess) {
		t.Helper()
		o := models.Order{CorpID: 0, OrderNumber: name, StartDate: &start, DueDate: &due}
		p := models.OrderItem{PartName: name, Quantity: 1000, ProductionQuantity: 1000, UnitPrice: 2, StartDate: &start, DueDate: &due, Currency: "CNY", SupplyMode: "self_purchased", Processes: []models.OrderProcess{{Name: "下料", Status: "pending"}, {Name: "铣削", Status: "pending"}, {Name: "检验", Status: "pending"}}}
		_, e := repo.Create(&o, []models.OrderItem{p}, 1)
		check(e)
		var item models.OrderItem
		check(db.Where("order_id=?", o.ID).Take(&item).Error)
		ps := []models.OrderProcess{}
		check(db.Where("order_item_id=?", item.ID).Order("sort_order,id").Find(&ps).Error)
		return item, ps
	}
	create := func(item models.OrderItem, scope, request string, supplier int64, qty int, lines []outsourcing.OrderLineInput) (models.OutsourcingOrder, error) {
		return outsourcing.CreateOrder(db, 0, 1, outsourcing.OrderInput{ItemID: item.ID, SupplierID: supplier, Scope: scope, Quantity: qty, UnitPrice: "2", Date: today, Expected: due.Format("2006-01-02"), RequestID: request, Lines: lines})
	}
	item, ps := newItem("PREORDER-SPLIT")
	oa, e := create(item, "selected", "book-a", a.ID, 0, []outsourcing.OrderLineInput{{ProcessID: ps[1].ID, Quantity: 600, UnitPrice: "1"}})
	check(e)
	ob, e := create(item, "selected", "book-b", b.ID, 0, []outsourcing.OrderLineInput{{ProcessID: ps[1].ID, Quantity: 400, UnitPrice: "1.2"}})
	check(e)
	replay, e := create(item, "selected", "book-a", a.ID, 0, []outsourcing.OrderLineInput{{ProcessID: ps[1].ID, Quantity: 600, UnitPrice: "1"}})
	check(e)
	if replay.ID != oa.ID {
		t.Fatal("duplicate purchase order")
	}
	_, e = create(item, "selected", "book-a", a.ID, 0, []outsourcing.OrderLineInput{{ProcessID: ps[1].ID, Quantity: 601, UnitPrice: "1"}})
	fail(e)
	_, e = create(item, "selected", "overbook", a.ID, 0, []outsourcing.OrderLineInput{{ProcessID: ps[1].ID, Quantity: 1, UnitPrice: "1"}})
	fail(e)
	var n int64
	check(db.Model(&models.ProcessQuantityEvent{}).Where("order_item_id=?", item.ID).Count(&n).Error)
	if n != 0 {
		t.Fatal("ordering created false production progress")
	}
	check(db.Transaction(func(tx *gorm.DB) error {
		locked, e := productionflow.LockItem(tx, 0, item.ID)
		if e != nil {
			return e
		}
		return productionflow.Activate(tx, locked, nil, 1)
	}))
	send := func(o models.OutsourcingOrder, line *int64, qty int, request string) (models.OutsourcingDispatch, error) {
		return outsourcing.DispatchOrder(db, 0, o.ID, 1, outsourcing.OrderDispatchInput{LineID: line, Quantity: qty, Date: today, Expected: due.Format("2006-01-02"), RequestID: request})
	}
	_, e = send(oa, &oa.Lines[0].ID, 600, "before-upstream")
	fail(e)
	report := func(p int64, kind string, qty int, req string) {
		t.Helper()
		check(productionflow.Report(db, 0, item.ID, p, 1, productionflow.ReportInput{Kind: kind, Quantity: qty, Date: today, RequestID: req}))
	}
	report(ps[0].ID, "start", 1000, "cut-start")
	report(ps[0].ID, "complete", 1000, "cut-complete")
	_, e = outsourcing.Dispatch(db, 0, 1, outsourcing.DispatchInput{ItemID: item.ID, ProcessID: ps[1].ID, SupplierID: &a.ID, Quantity: 1, UnitPrice: "1", Date: today, Expected: due.Format("2006-01-02"), RequestID: "steal-reservation"})
	fail(e)
	da, e := send(oa, &oa.Lines[0].ID, 600, "send-a")
	check(e)
	dbb, e := send(ob, &ob.Lines[0].ID, 400, "send-b")
	check(e)
	_, e = send(oa, &oa.Lines[0].ID, 1, "excess-send")
	fail(e)
	dReplay, e := send(oa, &oa.Lines[0].ID, 600, "send-a")
	check(e)
	if dReplay.ID != da.ID {
		t.Fatal("duplicate actual shipment")
	}
	fail(outsourcing.CancelOrder(db, 0, oa.ID))
	check(outsourcing.Receive(db, 0, da.ID, 1, outsourcing.ReturnInput{Kind: "return", Good: 600, Date: today, RequestID: "return-a"}))
	check(outsourcing.Receive(db, 0, dbb.ID, 1, outsourcing.ReturnInput{Kind: "return", Good: 400, Date: today, RequestID: "return-b"}))
	var total models.ProcessQuantity
	check(db.Where("order_process_id=?", ps[1].ID).Take(&total).Error)
	if total.StartedQuantity != 1000 || total.CompletedQuantity != 1000 {
		t.Fatal("multiple supplier quantities did not combine")
	}
	_, e = outsourcing.DispatchOrder(db, 99, oa.ID, 1, outsourcing.OrderDispatchInput{})
	fail(e)

	whole, route := newItem("FULL-ROUTE")
	fo, e := create(whole, "full", "book-full", a.ID, 1000, nil)
	check(e)
	check(db.Transaction(func(tx *gorm.DB) error {
		locked, e := productionflow.LockItem(tx, 0, whole.ID)
		if e != nil {
			return e
		}
		return productionflow.Activate(tx, locked, nil, 1)
	}))
	fd, e := send(fo, nil, 1000, "full-send")
	check(e)
	if fd.Scope != "full" || fd.OrderProcessID != route[2].ID {
		t.Fatal("full route was not one physical dispatch")
	}
	_, e = create(whole, "selected", "steal-full-downstream", b.ID, 0, []outsourcing.OrderLineInput{{ProcessID: route[1].ID, Quantity: 1, UnitPrice: "1"}})
	fail(e)
	first := outsourcing.ReturnInput{Kind: "return", Good: 450, Rework: 40, Scrap: 10, Date: today, Notes: "末道质检", RequestID: "full-first-return"}
	check(outsourcing.Receive(db, 0, fd.ID, 1, first))
	check(outsourcing.Receive(db, 0, fd.ID, 1, first))
	flow, e := productionflow.Read(db, 0, whole.ID)
	check(e)
	if flow.Processes[0].Started != 1000 || flow.Processes[0].Completed != 500 || flow.Processes[1].Completed != 500 || flow.Processes[2].Started != 500 || flow.Processes[2].Completed != 450 || flow.Item.ScrapQuantity != 10 {
		t.Fatalf("wrong full-route shared progress: %+v", flow.Processes)
	}
	_, e = outsourcing.Settle(db, 0, fd.ID, 1, outsourcing.SettlementInput{Amount: "1900", Tax: "0", Date: today})
	fail(e)
	check(outsourcing.SendRework(db, 0, fd.ID, 1, outsourcing.ReworkInput{Quantity: 40, Date: today, RequestID: "full-rework-send"}))
	check(outsourcing.Receive(db, 0, fd.ID, 1, outsourcing.ReturnInput{Kind: "rework_return", Good: 30, Scrap: 10, Date: today, Notes: "返工后检验", RequestID: "full-rework-return"}))
	check(outsourcing.Receive(db, 0, fd.ID, 1, outsourcing.ReturnInput{Kind: "return", Good: 500, Date: today, RequestID: "full-final-return"}))
	flow, e = productionflow.Read(db, 0, whole.ID)
	check(e)
	if flow.Processes[0].Completed != 1000 || flow.Processes[1].Completed != 1000 || flow.Processes[2].Started != 1000 || flow.Processes[2].Completed != 980 || flow.Item.ScrapQuantity != 20 {
		t.Fatal("rework repeated earlier full-route progress")
	}
	st, e := outsourcing.Settle(db, 0, fd.ID, 1, outsourcing.SettlementInput{Amount: "1960", Tax: "0", Date: today})
	check(e)
	st2, e := outsourcing.Settle(db, 0, fd.ID, 1, outsourcing.SettlementInput{Amount: "1960", Tax: "0", Date: today})
	check(e)
	if st.ID != st2.ID {
		t.Fatal("full-route cost posted more than once")
	}
	check(db.Model(&models.CostEntry{}).Where("order_item_id=?", whole.ID).Count(&n).Error)
	if n != 1 {
		t.Fatal("one route repeated cost per process")
	}
	spare, sps := newItem("CANCEL-RESERVATION")
	co, e := create(spare, "selected", "cancel-book", a.ID, 0, []outsourcing.OrderLineInput{{ProcessID: sps[1].ID, Quantity: 1000, UnitPrice: "1"}})
	check(e)
	check(outsourcing.CancelOrder(db, 0, co.ID))
	_, e = create(spare, "selected", "after-cancel", b.ID, 0, []outsourcing.OrderLineInput{{ProcessID: sps[1].ID, Quantity: 1000, UnitPrice: "1"}})
	check(e)
	t.Run("edit_delete_and_reverse_without_business_writes", func(t *testing.T) {
		check := func(e error) {
			t.Helper()
			if e != nil {
				t.Fatal(e)
			}
		}
		fail := func(e error) {
			t.Helper()
			if e == nil {
				t.Fatal("expected operation to be rejected")
			}
		}
		readOrder := func(id int64) models.OutsourcingOrder {
			t.Helper()
			var o models.OutsourcingOrder
			check(db.Where("corp_id=0 AND id=?", id).Take(&o).Error)
			check(db.Where("corp_id=0 AND outsourcing_order_id=?", id).Order("sequence,id").Find(&o.Lines).Error)
			return o
		}
		readDispatch := func(id int64) models.OutsourcingDispatch {
			t.Helper()
			var d models.OutsourcingDispatch
			check(db.Where("corp_id=0 AND id=?", id).Take(&d).Error)
			return d
		}
		editable, eps := newItem("EDIT-DELETE")
		book, e := create(editable, "selected", "edit-book", a.ID, 0, []outsourcing.OrderLineInput{{ProcessID: eps[0].ID, Quantity: 300, UnitPrice: "1"}, {ProcessID: eps[1].ID, Quantity: 300, UnitPrice: "2"}})
		check(e)
		other, e := create(editable, "selected", "other-book", b.ID, 0, []outsourcing.OrderLineInput{{ProcessID: eps[0].ID, Quantity: 400, UnitPrice: "3"}})
		check(e)
		original := readOrder(book.ID)
		originalToken := outsourcing.OrderToken(original)
		in := outsourcing.OrderInput{ItemID: editable.ID, SupplierID: b.ID, Scope: "selected", Date: today, Expected: due.Format("2006-01-02"), RequestID: "edit-request", Notes: "调整供应商及数量", Lines: []outsourcing.OrderLineInput{{ProcessID: eps[0].ID, Quantity: 700, UnitPrice: "1.5"}}}
		_, e = outsourcing.UpdateOrder(db, 0, 1, book.ID, in, originalToken)
		fail(e)
		if outsourcing.OrderToken(readOrder(book.ID)) != originalToken {
			t.Fatal("failed edit did not roll back")
		}
		in.Lines[0].Quantity = 600
		_, e = outsourcing.UpdateOrder(db, 99, 1, book.ID, in, originalToken)
		fail(e)
		updated, e := outsourcing.UpdateOrder(db, 0, 1, book.ID, in, originalToken)
		check(e)
		if updated.ID != book.ID || len(updated.Lines) != 1 || updated.SupplierID != b.ID {
			t.Fatal("edit changed identity or did not remove process")
		}
		_, e = outsourcing.UpdateOrder(db, 0, 1, book.ID, in, originalToken)
		check(e) // lost response retry
		reserved, e := productionflow.OutsourcingReserved(db, 0, editable.ID, eps[0].ID)
		check(e)
		if reserved != 1000 {
			t.Fatal("edit did not exclude its own reservation")
		}
		reserved, e = productionflow.OutsourcingReserved(db, 0, editable.ID, eps[1].ID)
		check(e)
		if reserved != 0 {
			t.Fatal("removed process reservation remained")
		}
		in.Notes = "stale overwrite"
		_, e = outsourcing.UpdateOrder(db, 0, 1, book.ID, in, originalToken)
		fail(e)
		fail(outsourcing.DeleteOrder(db, 0, book.ID, 1, outsourcing.RemoveInput{EditToken: originalToken, Reason: "stale"}))
		check(outsourcing.DeleteOrder(db, 0, book.ID, 1, outsourcing.RemoveInput{EditToken: outsourcing.OrderToken(readOrder(book.ID)), Reason: "取消误下单"}))
		check(outsourcing.DeleteOrder(db, 0, book.ID, 1, outsourcing.RemoveInput{Reason: "重试"}))
		reserved, e = productionflow.OutsourcingReserved(db, 0, editable.ID, eps[0].ID)
		check(e)
		if reserved != 400 {
			t.Fatal("delete did not release reservation")
		}
		check(outsourcing.DeleteOrder(db, 0, other.ID, 1, outsourcing.RemoveInput{EditToken: outsourcing.OrderToken(readOrder(other.ID)), Reason: "清理未发出的测试订单"}))
		_, e = send(readOrder(book.ID), nil, 1, "send-deleted")
		fail(e)
		check(db.Model(&models.ProcessQuantityEvent{}).Where("order_item_id=?", editable.ID).Count(&n).Error)
		if n != 0 {
			t.Fatal("edit/delete generated false production")
		}

		whole, route := newItem("REVERSE-FULL")
		full, e := create(whole, "full", "reverse-book", a.ID, 1000, nil)
		check(e)
		check(db.Transaction(func(tx *gorm.DB) error {
			locked, e := productionflow.LockItem(tx, 0, whole.ID)
			if e != nil {
				return e
			}
			return productionflow.Activate(tx, locked, nil, 1)
		}))
		actual, e := send(full, nil, 600, "reverse-dispatch")
		check(e)
		actual = readDispatch(actual.ID)
		fail(outsourcing.CancelDispatch(db, 99, actual.ID, 1, outsourcing.RemoveInput{EditToken: outsourcing.DispatchToken(actual), Reason: "wrong corp"}))
		_, e = outsourcing.UpdateDispatch(db, 0, actual.ID, 1, outsourcing.DispatchEditInput{EditToken: "stale", Expected: due.Format("2006-01-02"), Notes: "wrong token"})
		fail(e)
		_, e = outsourcing.UpdateDispatch(db, 0, actual.ID, 1, outsourcing.DispatchEditInput{EditToken: outsourcing.DispatchToken(actual), Expected: due.AddDate(0, 0, 2).Format("2006-01-02"), Notes: "调整回厂安排"})
		check(e)
		fail(outsourcing.CancelDispatch(db, 0, actual.ID, 1, outsourcing.RemoveInput{EditToken: outsourcing.DispatchToken(actual), Reason: "stale after edit"}))
		actual = readDispatch(actual.ID)
		check(outsourcing.CancelDispatch(db, 0, actual.ID, 1, outsourcing.RemoveInput{EditToken: outsourcing.DispatchToken(actual), Reason: "发出数量误登记"}))
		check(outsourcing.CancelDispatch(db, 0, actual.ID, 1, outsourcing.RemoveInput{Reason: "retry"}))
		for _, l := range readOrder(full.ID).Lines {
			if l.DispatchedQuantity != 0 || l.StartedQuantity != 0 {
				t.Fatal("full route counters not restored")
			}
		}
		var first models.OrderProcess
		check(db.First(&first, route[0].ID).Error)
		if first.ActualStartAt != nil || first.Status != "pending" {
			t.Fatal("reversal retained false start date or status")
		}
		flow, e := productionflow.Read(db, 0, whole.ID)
		check(e)
		if flow.Processes[0].Started != 0 || flow.Processes[0].OutsourcingReserved != 1000 {
			t.Fatal("full route reversal did not restore ledger/reservation")
		}
		fail(outsourcing.Receive(db, 0, actual.ID, 1, outsourcing.ReturnInput{Kind: "return", Good: 1, Date: today, RequestID: "revived-cancelled"}))
		full = readOrder(full.ID)
		actual2, e := send(full, nil, 1000, "corrected-dispatch")
		check(e)
		check(outsourcing.Receive(db, 0, actual2.ID, 1, outsourcing.ReturnInput{Kind: "return", Good: 1000, Date: today, RequestID: "corrected-return"}))
		fail(outsourcing.CancelDispatch(db, 0, actual2.ID, 1, outsourcing.RemoveInput{EditToken: outsourcing.DispatchToken(readDispatch(actual2.ID)), Reason: "cannot erase returned"}))
		// A standalone dispatch must restore only its own opening, retaining another supplier's progress.
		direct, dps := newItem("REVERSE-DIRECT")
		check(db.Model(&dps[0]).Update("is_outsourced", true).Error)
		check(db.Transaction(func(tx *gorm.DB) error {
			locked, e := productionflow.LockItem(tx, 0, direct.ID)
			if e != nil {
				return e
			}
			return productionflow.Activate(tx, locked, nil, 1)
		}))
		directInput := outsourcing.DispatchInput{ItemID: direct.ID, ProcessID: dps[0].ID, SupplierID: &a.ID, Quantity: 300, UnitPrice: "2", Date: today, Expected: due.Format("2006-01-02"), RequestID: "direct-reverse-a"}
		d1, e := outsourcing.Dispatch(db, 0, 1, directInput)
		check(e)
		directInput.RequestID = "direct-keep-b"
		directInput.SupplierID = &b.ID
		directInput.Quantity = 200
		_, e = outsourcing.Dispatch(db, 0, 1, directInput)
		check(e)
		check(outsourcing.CancelDispatch(db, 0, d1.ID, 1, outsourcing.RemoveInput{EditToken: outsourcing.DispatchToken(readDispatch(d1.ID)), Reason: "撤销其中一个供应商的误发出"}))
		flow, e = productionflow.Read(db, 0, direct.ID)
		check(e)
		if flow.Processes[0].Started != 200 {
			t.Fatal("direct reversal changed another supplier's progress")
		}
		first = models.OrderProcess{}
		check(db.First(&first, dps[0].ID).Error)
		if first.ActualStartAt == nil {
			t.Fatal("reversal erased another active dispatch date")
		}
		check(db.Model(&models.OutsourcingChange{}).Count(&n).Error)
		if n != 6 {
			t.Fatalf("expected 6 audit entries, got %d", n)
		}
		check(db.Model(&models.ProcessQuantityEvent{}).Where("order_item_id=? AND source_type='outsourcing_reversal'", whole.ID).Count(&n).Error)
		if n != 1 {
			t.Fatal("retry duplicated reversal")
		}
	})
	_, e = outsourcing.SaveSupplier(db, 0, b.ID, outsourcing.SupplierInput{Name: b.Name, Active: false})
	check(e)
	_, e = create(spare, "selected", "inactive-book", b.ID, 0, []outsourcing.OrderLineInput{{ProcessID: sps[0].ID, Quantity: 100, UnitPrice: "1"}})
	fail(e)
	// HTTP authorization remains the original production/order authority; no silent role changes.
	gin.SetMode(gin.TestMode)
	handler := NewOutsourcingHandler(db)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest("POST", "/", strings.NewReader(`{}`))
	c.Set("corpID", int64(0))
	c.Set("roleType", "user")
	c.Set("userID", int64(-999))
	handler.CreateOrder(c)
	if rec.Code != 403 {
		t.Fatalf("missing order authority status=%d", rec.Code)
	}
	t.Log(fmt.Sprintf("verified pre-order, two suppliers, quantity reservations, full-route return, rework and one cost: %s", today))
}
