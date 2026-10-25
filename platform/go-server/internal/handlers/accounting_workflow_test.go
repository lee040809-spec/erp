package handlers

import (
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	driver "github.com/go-sql-driver/mysql"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"machining-erp/internal/accounting"
	"machining-erp/internal/workflowtest"
	"net/http/httptest"
	"os"
	"regexp"
	"strings"
	"testing"
	"time"
)

func TestAccountingWorkflowTemporaryTables(t *testing.T) {
	if os.Getenv("RUN_FINANCE_INTEGRATION") != "1" {
		t.Skip("explicit database test required")
	}
	cfg := driver.NewConfig()
	cfg.User = os.Getenv("MYSQL_USER")
	cfg.Passwd = os.Getenv("MYSQL_PASSWORD")
	cfg.Net = "tcp"
	cfg.Addr = os.Getenv("MYSQL_HOST") + ":3306"
	cfg.DBName = os.Getenv("MYSQL_DATABASE")
	cfg.ParseTime = true
	cfg.Timeout = 10 * time.Second
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
	for _, name := range []string{"orders", "order_items", "order_processes", "production_completions", "order_business_records", "cost_entries", "cost_allocations", "monthly_closings", "business_entry_policies"} {
		var nameOut, ddl string
		check(pool.QueryRow("SHOW CREATE TABLE `"+name+"`").Scan(&nameOut, &ddl))
		kept := []string{}
		for _, line := range strings.Split(ddl, "\n") {
			if !strings.Contains(line, "FOREIGN KEY") {
				kept = append(kept, line)
			}
		}
		ddl = regexp.MustCompile(",\\s*\\)").ReplaceAllString(strings.Join(kept, "\n"), "\n)")
		exec(strings.Replace(ddl, "CREATE TABLE", "CREATE TEMPORARY TABLE", 1))
	}
	workflowtest.Tables(t, db, "order_processes", "production_completions", "order_business_records", "cost_entries")
	exec("ALTER TABLE production_completions AUTO_INCREMENT=1")
	exec("INSERT INTO orders(id,corp_id,customer_id,order_number,status) VALUES(1,0,1,'PENDING-PRICE','processing'),(2,1,2,'OTHER','processing')")
	exec("INSERT INTO order_items(id,corp_id,order_id,part_name,part_number,quantity,production_quantity,unit_price,currency,supply_mode,status,delivered_quantity) VALUES(1,0,1,'零件A','A',100,110,0,'CNY','self_purchased','processing',0),(2,1,2,'OTHER','B',1,1,0,'CNY','self_purchased','processing',0)")
	exec("INSERT INTO order_processes(id,corp_id,order_item_id,name,status,actual_end_at) VALUES(1,0,1,'加工','completed','2026-01-01'),(2,0,1,'送货','pending',NULL)")
	h := NewProductionFinanceHandler(db)
	call := func(fn gin.HandlerFunc, body string, params gin.Params, want int) map[string]interface{} {
		t.Helper()
		rec := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(rec)
		c.Request = httptest.NewRequest("POST", "/", strings.NewReader(body))
		c.Request.Header.Set("Content-Type", "application/json")
		c.Params = params
		c.Set("corpID", int64(0))
		c.Set("userID", int64(1))
		c.Set("roleType", "admin")
		fn(c)
		if rec.Code != want {
			t.Fatalf("status %d want %d: %s", rec.Code, want, rec.Body.String())
		}
		var result map[string]interface{}
		check(json.Unmarshal(rec.Body.Bytes(), &result))
		return result
	}
	item := gin.Params{{Key: "itemId", Value: "1"}}
	first := `{"qualified_quantity":20,"business_date":"2026-01-02","request_id":"11111111-1111-4111-8111-111111111111"}`
	call(h.CreateCompletion, first, item, 200)
	call(h.CreateCompletion, first, item, 200)
	call(h.CreateCompletion, `{"qualified_quantity":90,"business_date":"2026-02-02"}`, item, 200)
	var pending, rows int64
	check(db.Table("production_completions").Where("pricing_status='pending'").Count(&pending).Error)
	if pending != 2 {
		t.Fatal("pending completions were not preserved", pending)
	}
	call(h.CloseMonth, `{"reviewed_cost_complete":true}`, gin.Params{{Key: "month", Value: "2026-01"}}, 400)
	exec("UPDATE monthly_closings SET status='closed' WHERE accounting_month='2026-01'")
	call(h.PriceCompletion, `{"unit_price":"2.50"}`, item, 400)
	var price string
	check(db.Raw("SELECT CAST(unit_price AS CHAR) FROM order_items WHERE id=1").Scan(&price).Error)
	if positiveDecimal(price) {
		t.Fatal("closed month pricing failed to roll back item price")
	}
	exec("UPDATE monthly_closings SET status='open'")
	call(h.PriceCompletion, `{"unit_price":"2.50"}`, item, 200)
	call(h.PriceCompletion, `{"unit_price":"2.5001"}`, item, 400)
	call(h.PriceCompletion, `{"unit_price":"2.5"}`, item, 200)
	var totals struct {
		Output   string
		Billable string
		Reserve  string
	}
	check(db.Raw("SELECT CAST(SUM(output_value) AS CHAR) AS output,CAST(SUM(billable_quantity) AS CHAR) AS billable,CAST(SUM(qualified_quantity-billable_quantity) AS CHAR) AS reserve FROM production_completions").Scan(&totals).Error)
	if !sameDecimal(totals.Output, "250") || !sameDecimal(totals.Billable, "100") || !sameDecimal(totals.Reserve, "10") {
		t.Fatal("price cap or spare quantity incorrect", totals)
	}
	var monthValues []struct {
		Month  string
		Output string
	}
	check(db.Raw("SELECT DATE_FORMAT(business_date,'%Y-%m') AS month,CAST(output_value AS CHAR) AS output FROM production_completions ORDER BY business_date").Scan(&monthValues).Error)
	if len(monthValues) != 2 || monthValues[0].Month != "2026-01" || !sameDecimal(monthValues[0].Output, "50") || !sameDecimal(monthValues[1].Output, "200") {
		t.Fatal("pricing moved the original completion month", monthValues)
	}
	cost := `{"cost_type":"material","amount":"10","business_date":"2026-03-01","order_id":1,"order_item_id":1,"production_completion_id":1,"auto_allocate":true,"request_id":"22222222-2222-4222-8222-222222222222"}`
	call(h.CreateCostEntry, cost, nil, 200)
	call(h.CreateCostEntry, cost, nil, 200)
	check(db.Table("cost_entries").Count(&rows).Error)
	if rows != 1 {
		t.Fatal("retry duplicated cost", rows)
	}
	check(db.Table("cost_allocations").Count(&rows).Error)
	if rows != 1 {
		t.Fatal("one-step allocation duplicated", rows)
	}
	call(h.CreateCostEntry, strings.Replace(cost, `"amount":"10"`, `"amount":"11"`, 1), nil, 400)
	// A rejected allocation must leave neither an expense nor an idempotency reservation.
	bad := strings.ReplaceAll(cost, "22222222-2222-4222-8222-222222222222", "33333333-3333-4333-8333-333333333333")
	bad = strings.Replace(bad, `"production_completion_id":1`, `"production_completion_id":999`, 1)
	call(h.CreateCostEntry, bad, nil, 400)
	check(db.Table("cost_entries").Count(&rows).Error)
	if rows != 1 {
		t.Fatal("failed one-step allocation leaked expense")
	}
	exec("INSERT INTO order_business_records(id,corp_id,order_id,order_item_id,kind,quantity,business_date,status,source_type,request_id,created_by) VALUES(1,0,1,1,'delivery',10,'2026-01-10','confirmed','business','DELIVERY-1',1),(2,0,1,1,'delivery',40,'2026-02-10','confirmed','business','DELIVERY-2',1)")
	exec("UPDATE order_items SET delivered_quantity=50 WHERE id=1")
	in := accounting.ReconcileInput{CustomerID: 1, Date: "2026-03-01", DeliveryIDs: []int64{1, 2}, RequestID: "44444444-4444-4444-8444-444444444444"}
	statement, e := accounting.Reconcile(db, 0, 1, in)
	check(e)
	if !sameDecimal(statement.Amount, "125") {
		t.Fatal("wrong delivered amount", statement.Amount)
	}
	replay, e := accounting.Reconcile(db, 0, 1, in)
	check(e)
	if replay.ID != statement.ID {
		t.Fatal("replay duplicated statement")
	}
	in.RequestID = "55555555-5555-4555-8555-555555555555"
	if _, e = accounting.Reconcile(db, 0, 1, in); e == nil {
		t.Fatal("delivery reconciled twice")
	}
	if e = accounting.RequireUnreconciled(db, 0, 1); e == nil {
		t.Fatal("reconciled delivery remained editable")
	}
	if e = accounting.RequireUnreconciled(db, 1, 1); e != nil {
		t.Fatal("company scope leaked")
	}
	exec("UPDATE order_items SET unit_price=4 WHERE id=1")
	check(db.Table("delivery_reconciliations").Where("id=?", statement.ID).Select("CAST(amount AS CHAR)").Scan(&price).Error)
	if !sameDecimal(price, "125") {
		t.Fatal("statement snapshot changed")
	}
	if e = accounting.ReverseStatement(db, 0, 1, statement.ID, ""); e == nil {
		t.Fatal("reason was optional")
	}
	check(accounting.ReverseStatement(db, 0, 1, statement.ID, "日期核对后重新出单"))
	check(accounting.RequireUnreconciled(db, 0, 1))
	check(db.Table("delivery_reconciliation_lines").Count(&rows).Error)
	if rows != 2 {
		t.Fatal("reversal removed history")
	}
	in.RequestID = "66666666-6666-4666-8666-666666666666"
	next, e := accounting.Reconcile(db, 0, 1, in)
	check(e)
	if !sameDecimal(next.Amount, "200") {
		t.Fatal("new reconciliation did not use current price")
	}
	check(db.Table("completion_price_changes").Count(&rows).Error)
	if rows != 2 {
		t.Fatal("pricing audit missing or duplicated", rows)
	}
	exec("INSERT INTO order_items(id,corp_id,order_id,part_name,quantity,production_quantity,unit_price,currency,supply_mode,status) VALUES(3,0,1,'历史待补价',10,10,0,'CNY','self_purchased','completed')")
	call(h.ImportHistory, `{"order_item_id":3,"qualified_quantity":5,"business_date":"2026-04-01","expected_price":"0","reviewed":true,"request_id":"77777777-7777-4777-8777-777777777777"}`, nil, 200)
	check(db.Table("production_completions").Where("order_item_id=3 AND pricing_status='pending'").Count(&rows).Error)
	if rows != 1 {
		t.Fatal("history missing-price batch was treated as final zero output")
	}
	call(h.PriceCompletion, `{"unit_price":"3"}`, gin.Params{{Key: "itemId", Value: "3"}}, 200)
	check(db.Table("production_completions").Where("order_item_id=3").Select("CAST(output_value AS CHAR)").Scan(&price).Error)
	if !sameDecimal(price, "15") {
		t.Fatal("history pending pricing did not settle")
	}
	t.Log(fmt.Sprintf("verified pending pricing, original months, spare cap, closed-month rollback, idempotent cost+allocation and saved delivery reconciliation with immutable history; statement=%d", next.ID))
}
