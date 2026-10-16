package handlers

import (
	"encoding/json"
	"github.com/gin-gonic/gin"
	driver "github.com/go-sql-driver/mysql"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"machining-erp/internal/workflowtest"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

// Opt-in integration test. All fixtures shadow real tables with connection-local
// temporary tables; neither real rows nor the real schema are changed.
func TestFinanceIntegrationTemporaryTables(t *testing.T) {
	if os.Getenv("RUN_FINANCE_INTEGRATION") != "1" {
		t.Skip("requires explicit database environment")
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
	exec := func(sql string) {
		t.Helper()
		if err := db.Exec(sql).Error; err != nil {
			t.Fatal(err)
		}
	}

	exec("CREATE TEMPORARY TABLE order_business_records (id BIGINT PRIMARY KEY AUTO_INCREMENT,corp_id BIGINT,order_id BIGINT,order_item_id BIGINT,kind VARCHAR(20),quantity INT,business_date DATE,status VARCHAR(20))")
	exec("CREATE TEMPORARY TABLE business_entry_policies (corp_id BIGINT PRIMARY KEY,policy TEXT,role_modes TEXT)")
	schema, err := os.ReadFile("../../sql/012_add_production_finance.sql")
	if err != nil {
		t.Fatal(err)
	}
	for _, table := range []string{"production_completions", "cost_entries", "cost_allocations", "monthly_closings"} {
		marker := "CREATE TABLE IF NOT EXISTS `" + table + "`"
		index := strings.Index(string(schema), marker)
		if index < 0 {
			t.Fatal("missing table schema")
		}
		statement := strings.SplitN(string(schema)[index:], ";", 2)[0]
		exec(strings.Replace(statement, "CREATE TABLE IF NOT EXISTS", "CREATE TEMPORARY TABLE", 1))
	}
	exec("ALTER TABLE production_completions ADD COLUMN billable_quantity DECIMAL(12,3) NOT NULL DEFAULT 0")
	exec("CREATE TEMPORARY TABLE orders (id BIGINT PRIMARY KEY,corp_id BIGINT,order_number VARCHAR(50),customer_name VARCHAR(50),status VARCHAR(20))")
	exec("CREATE TEMPORARY TABLE order_items (id BIGINT PRIMARY KEY,corp_id BIGINT,order_id BIGINT,part_name VARCHAR(50),quantity INT,scrap_quantity INT,status VARCHAR(20),completion_date DATE NULL,unit_price DECIMAL(15,4),supply_mode VARCHAR(32),currency CHAR(3),tax_inclusive TINYINT,production_quantity INT DEFAULT 0)")
	exec("CREATE TEMPORARY TABLE order_processes (id BIGINT,corp_id BIGINT,order_item_id BIGINT,name VARCHAR(50),status VARCHAR(20))")
	workflowtest.Tables(t, db, "order_processes", "production_completions", "order_business_records", "cost_entries")
	exec("INSERT INTO orders(id,corp_id,order_number,customer_name,status) VALUES (1,0,'TEST','TEST','processing')")
	exec("INSERT INTO order_items(id,corp_id,order_id,part_name,quantity,scrap_quantity,status,completion_date,unit_price,supply_mode,currency,tax_inclusive,production_quantity) VALUES (1,0,1,'TEST',100,0,'processing',NULL,20,'self_purchased','CNY',0,0),(2,0,1,'HISTORY',10,2,'delivered','2026-03-01',2,'customer_supplied','CNY',0,0)")
	exec("ALTER TABLE orders ADD COLUMN start_date DATE NULL")
	exec("ALTER TABLE order_items ADD COLUMN start_date DATE NULL,ADD COLUMN material_cost DECIMAL(15,2),ADD COLUMN tool_cost DECIMAL(15,2),ADD COLUMN fixture_cost DECIMAL(15,2),ADD COLUMN other_cost DECIMAL(15,2)")
	exec("ALTER TABLE order_processes ADD COLUMN outsourcing_fee DECIMAL(15,2) DEFAULT 0,ADD COLUMN is_outsourced TINYINT DEFAULT 0")
	exec("UPDATE orders SET start_date='2026-01-10'")
	exec("UPDATE order_items SET start_date='2026-02-01',material_cost=123.45,tool_cost=2,fixture_cost=3,other_cost=4 WHERE id=1")
	h := NewProductionFinanceHandler(db)
	gin.SetMode(gin.TestMode)
	roleType := "admin"
	userID := int64(1)
	call := func(fn gin.HandlerFunc, body string, params gin.Params, url string, want int) map[string]interface{} {
		t.Helper()
		rec := httptest.NewRecorder()
		ctx, _ := gin.CreateTestContext(rec)
		ctx.Request = httptest.NewRequest("POST", url, strings.NewReader(body))
		ctx.Request.Header.Set("Content-Type", "application/json")
		ctx.Params = params
		ctx.Set("corpID", int64(0))
		ctx.Set("userID", userID)
		ctx.Set("roleType", roleType)
		fn(ctx)
		if rec.Code != want {
			t.Fatalf("status %d want %d: %s", rec.Code, want, rec.Body.String())
		}
		var result map[string]interface{}
		if json.Unmarshal(rec.Body.Bytes(), &result) != nil {
			t.Fatal("invalid response")
		}
		return result
	}
	item := gin.Params{{Key: "itemId", Value: "1"}}
	first := call(h.CreateCompletion, `{"qualified_quantity":40,"business_date":"2026-01-01","request_id":"test-first"}`, item, "/", 200)["data"].(map[string]interface{})
	call(h.CreateCompletion, `{"qualified_quantity":40,"business_date":"2026-01-01","request_id":"test-first"}`, item, "/", 200)
	call(h.CreateCompletion, `{"qualified_quantity":41,"business_date":"2026-01-01","request_id":"test-first"}`, item, "/", 400)
	exec("UPDATE order_items SET unit_price=30 WHERE id=1")
	second := call(h.CreateCompletion, `{"qualified_quantity":60,"business_date":"2026-02-01"}`, item, "/", 200)["data"].(map[string]interface{})
	var linkedStatus string
	if db.Table("orders").Where("id=1").Select("status").Scan(&linkedStatus).Error != nil || linkedStatus != "completed" {
		t.Fatal("completion did not update order status")
	}
	if first["output_value"] != "800.00" || second["output_value"] != "1800.00" {
		t.Fatal("price snapshots or output incorrect")
	}
	call(h.CreateCompletion, `{"qualified_quantity":1,"business_date":"2026-02-01"}`, item, "/", 400)
	cost := call(h.CreateCostEntry, `{"cost_type":"material","amount":"1000.00","tax_amount":"0","net_amount":"1000.00","business_date":"2025-12-01","order_id":1,"order_item_id":1}`, nil, "/", 200)["data"].(map[string]interface{})
	if cost["id"] != float64(1) {
		t.Fatal("unexpected temporary cost id")
	}
	call(h.AllocateCost, `{"cost_entry_id":1,"production_completion_id":1,"allocated_amount":"400","allocation_basis":"40%"}`, nil, "/", 200)
	call(h.AllocateCost, `{"cost_entry_id":1,"production_completion_id":1,"allocated_amount":"400","allocation_basis":"repeat"}`, nil, "/", 400)
	closeParams := gin.Params{{Key: "month", Value: "2026-01"}}
	closeBody := `{"reviewed_cost_complete":true,"notes":"test review"}`
	call(h.CloseMonth, closeBody, closeParams, "/", 400)
	call(h.AllocateCost, `{"cost_entry_id":1,"production_completion_id":2,"allocated_amount":"600.01","allocation_basis":"over"}`, nil, "/", 400)
	call(h.AllocateCost, `{"cost_entry_id":1,"production_completion_id":2,"allocated_amount":"600","allocation_basis":"60%"}`, nil, "/", 200)
	call(h.CreateCostEntry, `{"cost_type":"period","amount":"50","tax_amount":"0","net_amount":"50","business_date":"2026-02-02"}`, nil, "/", 200)
	exec("INSERT INTO order_processes(id,corp_id,order_item_id,name,status,is_outsourced,outsourcing_fee) VALUES(99,0,1,'外协测试','completed',1,80),(100,0,1,'内部加工','completed',0,900),(101,0,1,'送货','completed',1,900)")
	data := call(h.GetSummary, "", nil, "/?year=2026", 200)["data"].(map[string]interface{})
	annual := data["annual"].(map[string]interface{})
	if annual["order_amount"] != "3020.00" {
		t.Fatalf("order amount must use ordered quantity and current price, not completion quantities: %+v", annual)
	}
	for _, raw := range data["months"].([]interface{}) {
		m := raw.(map[string]interface{})
		if m["month"] == "2026-02" {
			cost := m["order_costs"].(map[string]interface{})
			if m["order_amount"] != "3000.00" || !sameDecimal(cost["material"].(string), "123.45") || !sameDecimal(cost["outsourcing"].(string), "80") {
				t.Fatalf("part-date priority or entered cost grouping incorrect: %+v", m)
			}
		}
	}
	if annual["output_value"] != "2600.00" || annual["production_cost"] != "1000.00" || annual["operating_profit"] != "1550.00" || data["pending_cost_count"] != float64(0) {
		t.Fatalf("cross-month summary incorrect: %+v", data)
	}
	call(h.CloseMonth, closeBody, closeParams, "/", 200)
	call(h.CreateCostEntry, `{"cost_type":"period","amount":"1","business_date":"2026-01-02"}`, nil, "/", 400)
	call(h.CreateCompletion, `{"qualified_quantity":1,"business_date":"2026-01-02"}`, item, "/", 400)
	call(h.AllocateCost, `{"cost_entry_id":1,"production_completion_id":1,"allocated_amount":"1","allocation_basis":"closed"}`, nil, "/", 400)
	call(h.ImportHistory, `{"order_item_id":2,"qualified_quantity":9,"business_date":"2026-03-01","expected_price":"2","notes":"review","reviewed":true}`, nil, "/", 400)
	call(h.ImportHistory, `{"order_item_id":2,"qualified_quantity":8,"business_date":"2026-03-01","expected_price":"2","notes":"review","reviewed":true}`, nil, "/", 200)
	call(h.ImportHistory, `{"order_item_id":2,"qualified_quantity":8,"business_date":"2026-03-01","expected_price":"2","notes":"repeat","reviewed":true}`, nil, "/", 400)
	call(h.CreateCostEntry, `{"cost_type":"material","amount":"1","business_date":"2026-03-01"}`, nil, "/", 200)
	call(h.AllocateCost, `{"cost_entry_id":3,"production_completion_id":3,"allocated_amount":"1","allocation_basis":"customer material"}`, nil, "/", 400)
	call(h.FinanceDetails, "", nil, "/?month=2026-01", 200)
	call(h.PreviewHistory, "", nil, "/?year=2026", 200)
	exec("INSERT INTO order_items(id,corp_id,order_id,part_name,quantity,scrap_quantity,status,completion_date,unit_price,supply_mode,currency,tax_inclusive,production_quantity) VALUES(4,0,1,'SPLIT-HISTORY',10,0,'delivered','2026-04-01',5,'customer_supplied','CNY',0,0)")
	firstHistory := `{"order_item_id":4,"qualified_quantity":4,"business_date":"2026-04-01","expected_price":"5","confirmed_price":"3.3333","source_ref":"QC-4-A","notes":"review first batch","reviewed":true}`
	call(h.ImportHistory, firstHistory, nil, "/", 200)
	call(h.ImportHistory, firstHistory, nil, "/", 200)
	changedHistory := strings.Replace(firstHistory, `"qualified_quantity":4`, `"qualified_quantity":5`, 1)
	call(h.ImportHistory, changedHistory, nil, "/", 400)
	exec("UPDATE order_items SET status='processing' WHERE id=4")
	historyRows := call(h.PreviewHistory, "", nil, "/?year=2026", 200)["data"].(map[string]interface{})["items"].([]interface{})
	if len(historyRows) != 1 || historyRows[0].(map[string]interface{})["remaining_quantity"] != float64(6) {
		t.Fatalf("partially audited history not visible with correct remainder: %+v", historyRows)
	}
	tooMany := `{"order_item_id":4,"qualified_quantity":7,"business_date":"2026-05-01","expected_price":"5","confirmed_price":"7.0001","source_ref":"QC-4-B","notes":"review second batch","reviewed":true}`
	call(h.ImportHistory, tooMany, nil, "/", 400)
	secondHistory := strings.Replace(tooMany, `"qualified_quantity":7`, `"qualified_quantity":6`, 1)
	call(h.ImportHistory, secondHistory, nil, "/", 200)
	historySummary := call(h.GetSummary, "", nil, "/?year=2026", 200)["data"].(map[string]interface{})["months"].([]interface{})
	foundMonths := map[string]string{}
	for _, row := range historySummary {
		m := row.(map[string]interface{})
		foundMonths[m["month"].(string)] = m["output_value"].(string)
	}
	if foundMonths["2026-04"] != "13.33" || foundMonths["2026-05"] != "42.00" {
		t.Fatalf("historical price snapshots or cross-month rounding incorrect: %+v", foundMonths)
	}
	afterHistory := call(h.PreviewHistory, "", nil, "/?year=2026", 200)["data"].(map[string]interface{})["items"].([]interface{})
	if len(afterHistory) != 0 {
		t.Fatal("fully audited history remained as candidate")
	}
	var completedHistory struct {
		Status         string
		CompletionDate string
	}
	if err := db.Table("order_items").Where("id=4").Select("status,DATE_FORMAT(completion_date,'%Y-%m-%d') AS completion_date").Take(&completedHistory).Error; err != nil || completedHistory.Status != "completed" || completedHistory.CompletionDate != "2026-05-01" {
		t.Fatal("historical final status or date not linked")
	}
	var currentPrice string
	if err := db.Table("order_items").Where("id=4").Select("CAST(unit_price AS CHAR)").Scan(&currentPrice).Error; err != nil || !sameDecimal(currentPrice, "5") {
		t.Fatal("historical import changed current order price")
	}
	t.Log("historical split months, remaining quantity, per-batch audited price, source reference replay and full-audit removal passed")

	// Production buffers absorb scrap, but never create extra contracted revenue.
	exec("INSERT INTO order_items(id,corp_id,order_id,part_name,quantity,scrap_quantity,status,completion_date,unit_price,supply_mode,currency,tax_inclusive,production_quantity) VALUES(5,0,1,'BUFFER',100,5,'processing',NULL,10,'self_purchased','CNY',0,110)")
	buffer := gin.Params{{Key: "itemId", Value: "5"}}
	bufferFirst := call(h.CreateCompletion, `{"qualified_quantity":80,"business_date":"2026-06-01","request_id":"buffer-first"}`, buffer, "/", 200)["data"].(map[string]interface{})
	if bufferFirst["billable_quantity"] != "80.000" || bufferFirst["output_value"] != "800.00" {
		t.Fatal("first buffered output incorrect", bufferFirst)
	}
	bufferSecond := call(h.CreateCompletion, `{"qualified_quantity":25,"business_date":"2026-07-01","request_id":"buffer-second"}`, buffer, "/", 200)["data"].(map[string]interface{})
	if bufferSecond["billable_quantity"] != "20.000" || bufferSecond["output_value"] != "200.00" {
		t.Fatal("reserve incorrectly billed", bufferSecond)
	}
	call(h.CreateCompletion, `{"qualified_quantity":25,"business_date":"2026-07-01","request_id":"buffer-second"}`, buffer, "/", 200)
	info := call(h.ListCompletions, "", buffer, "/", 200)["data"].(map[string]interface{})
	if info["ordered_quantity"] != float64(100) || info["production_quantity"] != float64(110) || info["confirmed_quantity"] != float64(105) || info["billable_quantity"] != float64(100) || info["reserve_quantity"] != float64(5) || info["delivery_gap"] != float64(0) || info["remaining_quantity"] != float64(0) {
		t.Fatal("buffer physical and financial quantities incorrect", info)
	}
	call(h.CreateCompletion, `{"qualified_quantity":1,"business_date":"2026-07-02"}`, buffer, "/", 400)
	exec("UPDATE order_items SET production_quantity=112 WHERE id=5")
	third := call(h.CreateCompletion, `{"qualified_quantity":2,"business_date":"2026-08-01","request_id":"buffer-reserve"}`, buffer, "/", 200)["data"].(map[string]interface{})
	if third["billable_quantity"] != "0.000" || third["output_value"] != "0.00" {
		t.Fatal("reserve-only batch created income")
	}
	// Scrap does not reduce a customer's delivery target.
	exec("INSERT INTO order_items(id,corp_id,order_id,part_name,quantity,scrap_quantity,status,completion_date,unit_price,supply_mode,currency,tax_inclusive,production_quantity) VALUES(6,0,1,'SCRAP-SHORTAGE',10,2,'processing',NULL,10,'customer_supplied','CNY',0,10)")
	shortage := gin.Params{{Key: "itemId", Value: "6"}}
	call(h.CreateCompletion, `{"qualified_quantity":8,"business_date":"2026-09-01"}`, shortage, "/", 200)
	var shortageStatus string
	if err := db.Table("order_items").Where("id=6").Select("status").Scan(&shortageStatus).Error; err != nil || shortageStatus != "processing" {
		t.Fatal("scrap reduced the contracted delivery target")
	}
	exec("UPDATE order_items SET production_quantity=12 WHERE id=6")
	call(h.CreateCompletion, `{"qualified_quantity":2,"business_date":"2026-09-02"}`, shortage, "/", 200)
	// The history route uses the same cap and spare split.
	exec("INSERT INTO order_items(id,corp_id,order_id,part_name,quantity,scrap_quantity,status,completion_date,unit_price,supply_mode,currency,tax_inclusive,production_quantity) VALUES(7,0,1,'BUFFER-HISTORY',10,0,'delivered','2026-09-01',2,'customer_supplied','CNY',0,12)")
	call(h.ImportHistory, `{"order_item_id":7,"qualified_quantity":12,"business_date":"2026-09-01","expected_price":"2","reviewed":true,"notes":"verified count including two spare parts","source_ref":"BUFFER-HISTORY-1"}`, nil, "/", 200)
	var historical struct {
		Billable string
		Output   string
	}
	if err := db.Table("production_completions").Where("order_item_id=7").Select("CAST(billable_quantity AS CHAR) AS billable,CAST(output_value AS CHAR) AS output").Take(&historical).Error; err != nil || historical.Billable != "10.000" || historical.Output != "20.00" {
		t.Fatal("history buffer billed spare parts", historical)
	}
	bufferSummary := call(h.GetSummary, "", nil, "/?year=2026", 200)["data"].(map[string]interface{})
	foundReserve := false
	for _, raw := range bufferSummary["months"].([]interface{}) {
		m := raw.(map[string]interface{})
		if m["month"] == "2026-07" {
			foundReserve = true
			if m["qualified_quantity"] != "25.000" || m["billable_quantity"] != "20.000" || m["reserve_quantity"] != "5.000" || m["output_value"] != "200.00" {
				t.Fatal("monthly spare split incorrect", m)
			}
		}
	}
	if !foundReserve {
		t.Fatal("monthly buffered output missing")
	}
	t.Log("BUFFER PASS: order100 production110 scrap5 qualified105 billed100 spare5; split months, replay, reserve-only, caps, delivery shortage and historical import")
	exec("CREATE TEMPORARY TABLE users(id BIGINT,corp_id BIGINT,role_id BIGINT)")
	exec("CREATE TEMPORARY TABLE permissions(role_id BIGINT,corp_id BIGINT,resource_id BIGINT)")
	exec("CREATE TEMPORARY TABLE resources(id BIGINT,resource_key VARCHAR(100))")
	exec("INSERT INTO users VALUES(1,0,7),(2,0,8),(3,1,7)")
	exec("INSERT INTO resources VALUES(1,'production_dashboard'),(2,'orders')")
	exec("INSERT INTO permissions VALUES(7,0,1),(8,0,2)")
	roleType = "user"
	userID = 3
	call(h.ListCompletions, "", item, "/", 403)
	call(h.CreateCompletion, `{"qualified_quantity":1,"business_date":"2026-03-02"}`, item, "/", 403)
	roleType = "user"
	userID = 1
	exec("INSERT INTO order_items(id,corp_id,order_id,part_name,quantity,scrap_quantity,status,completion_date,unit_price,supply_mode,currency,tax_inclusive,production_quantity) VALUES(3,0,1,'PERMISSION',2,0,'processing',NULL,12.3456,'self_purchased','CNY',0,0)")
	permissionItem := gin.Params{{Key: "itemId", Value: "3"}}
	body := `{"qualified_quantity":1,"business_date":"2026-03-02","request_id":"production-permission"}`
	masked := call(h.CreateCompletion, body, permissionItem, "/", 200)["data"].(map[string]interface{})
	if _, ok := masked["unit_price_snapshot"]; ok {
		t.Fatal("production response leaked price")
	}
	if _, ok := masked["output_value"]; ok {
		t.Fatal("production response leaked value")
	}
	call(h.CreateCompletion, body, permissionItem, "/", 200)
	rows := call(h.ListCompletions, "", permissionItem, "/", 200)["data"].(map[string]interface{})["records"].([]interface{})
	if len(rows) != 1 || rows[0].(map[string]interface{})["output_value"] != nil {
		t.Fatal("production list leaked prices or duplicate entry")
	}
	userID = 2
	call(h.CreateCompletion, body, permissionItem, "/", 403)
	priced := call(h.ListCompletions, "", permissionItem, "/", 200)["data"].(map[string]interface{})["records"].([]interface{})[0].(map[string]interface{})
	if priced["output_value"] != "12.35" {
		t.Fatal("authorized amount or decimal rounding incorrect")
	}
	t.Log("completion module access, cross-company role isolation, production amount omission and duplicate replay passed")
	t.Log("temporary-table integration: split months, price snapshots, cost conservation, period costs, duplicate rejection, historical review and month locks passed")
}
