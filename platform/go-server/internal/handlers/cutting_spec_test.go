package handlers

import (
	"encoding/json"
	"github.com/gin-gonic/gin"
	driver "github.com/go-sql-driver/mysql"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"machining-erp/internal/models"
	"machining-erp/internal/repository"
	"machining-erp/internal/workflowtest"
	"net/http/httptest"
	"os"
	"regexp"
	"strings"
	"testing"
)

func TestCuttingSpecTemporaryTables(t *testing.T) {
	if os.Getenv("RUN_FINANCE_INTEGRATION") != "1" {
		t.Skip("explicit test database environment required")
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
	exec := func(q string) { t.Helper(); check(db.Exec(q).Error) }
	for _, name := range []string{"orders", "order_items", "order_processes", "process_time_events", "production_completions", "order_business_records", "cost_entries", "production_plans", "business_entry_policies"} {
		var table, ddl string
		check(pool.QueryRow("SHOW CREATE TABLE `"+name+"`").Scan(&table, &ddl))
		lines := []string{}
		for _, line := range strings.Split(ddl, "\n") {
			if !strings.Contains(line, "FOREIGN KEY") {
				lines = append(lines, line)
			}
		}
		ddl = regexp.MustCompile(",\\s*\\)").ReplaceAllString(strings.Join(lines, "\n"), "\n)")
		exec(strings.Replace(ddl, "CREATE TABLE", "CREATE TEMPORARY TABLE", 1))
	}
	workflowtest.Tables(t, db, "order_processes", "production_completions", "order_business_records")
	exec("CREATE TEMPORARY TABLE users(id BIGINT,corp_id BIGINT,role_id BIGINT)")
	exec("CREATE TEMPORARY TABLE permissions(role_id BIGINT,corp_id BIGINT,resource_id BIGINT)")
	exec("CREATE TEMPORARY TABLE resources(id BIGINT,resource_key VARCHAR(100))")
	exec("INSERT INTO users VALUES(1,0,1)")
	exec("INSERT INTO resources VALUES(1,'dashboard'),(2,'orders')")
	exec("INSERT INTO orders(id,corp_id,order_number) VALUES(1,0,'TIME-1'),(2,1,'FOREIGN')")
	exec("UPDATE orders SET start_date='2026-09-20' WHERE id=1")
	exec("INSERT INTO order_items(id,corp_id,order_id,part_name,quantity,production_quantity,unit_price,total_price,status) VALUES(1,0,1,'TIMED',10,10,3,30,'pending'),(2,1,2,'FOREIGN',10,10,9,90,'pending')")
	exec("INSERT INTO order_processes(id,corp_id,order_item_id,name,status,is_outsourced,outsourcing_fee) VALUES(1,0,1,'铣削','pending',1,100),(2,1,2,'FOREIGN','pending',0,0),(3,0,1,'送货','pending',0,0),(4,0,1,'历史加工','completed',0,0)")
	h := NewProcessTimelineHandler(db)
	gin.SetMode(gin.TestMode)
	call := func(fn gin.HandlerFunc, body string, item, process, role string, want int) map[string]interface{} {
		t.Helper()
		rec := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(rec)
		c.Request = httptest.NewRequest("PATCH", "/", strings.NewReader(body))
		c.Request.Header.Set("Content-Type", "application/json")
		c.Params = gin.Params{{Key: "itemId", Value: item}, {Key: "processId", Value: process}, {Key: "id", Value: item}}
		c.Set("corpID", int64(0))
		c.Set("userID", int64(1))
		c.Set("roleType", role)
		fn(c)
		if rec.Code != want {
			t.Fatalf("status %d want %d: %s", rec.Code, want, rec.Body.String())
		}
		var out map[string]interface{}
		check(json.Unmarshal(rec.Body.Bytes(), &out))
		return out
	}
	orders := NewOrderHandler(repository.NewOrderRepository(db), nil, nil, nil, db)
	payload := func(spec string) string {
		return `{"order_number":"SPEC","start_date":"2026-09-20","due_date":"2026-10-20","items":[{"id":1,"part_name":"TIMED","quantity":10,"production_quantity":10,"unit_price":3,"processes":[{"id":1,"name":"下料","is_outsourced":true,"outsourcing_fee":100,"status":"pending"` + spec + `},{"id":3,"name":"送货","status":"pending"},{"id":4,"name":"历史加工","status":"completed"}]}]}`
	}
	call(orders.Update, payload(`,"cutting_spec":"  Φ30 × 80 mm  "`), "1", "", "admin", 200)
	call(orders.Update, payload(""), "1", "", "admin", 200)
	var p models.OrderProcess
	check(db.First(&p, 1).Error)
	if p.CuttingSpec == nil || *p.CuttingSpec != "Φ30 × 80 mm" {
		t.Fatal("order request conversion or omitted field preservation failed")
	}
	route := func(spec string) string {
		return `{"processes":[{"id":1,"name":"下料","is_outsourced":true` + spec + `},{"id":3,"name":"送货"},{"id":4,"name":"历史加工"}]}`
	}
	call(h.SaveRoute, route(`,"cutting_spec":"板料 120 × 60 × 8 mm"`), "1", "", "admin", 200)
	call(h.SaveRoute, route(""), "1", "", "admin", 200)
	data := call(h.List, "", "1", "", "admin", 200)["data"].(map[string]interface{})
	if data["processes"].([]interface{})[0].(map[string]interface{})["cutting_spec"] != "板料 120 × 60 × 8 mm" {
		t.Fatal("route update/read or omitted spec preservation failed")
	}
	call(h.SaveRoute, route(`,"cutting_spec":""`), "1", "", "admin", 200)
	p = models.OrderProcess{}
	check(db.First(&p, 1).Error)
	if p.CuttingSpec == nil || *p.CuttingSpec != "" {
		t.Fatal("explicit clear failed")
	}
	if p.OutsourcingFee != 100 || p.Status != "pending" || p.ActualStartAt != nil {
		t.Fatal("specification edit changed unrelated production fields")
	}
	foreign := models.OrderProcess{}
	check(db.First(&foreign, 2).Error)
	if foreign.CuttingSpec != nil {
		t.Fatal("foreign company modified")
	}
	t.Log("order/route updates, re-query, omitted field preservation, explicit clearing and company isolation passed on temporary tables")
}
