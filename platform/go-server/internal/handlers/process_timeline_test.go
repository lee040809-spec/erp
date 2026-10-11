package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
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
	"time"
)

func TestProcessNodeValidation(t *testing.T) {
	v := "2026-09-01T09:30"
	node, err := parseProcessNode(&v, nil)
	if err != nil || node.UTC().Format("15:04") != "01:30" {
		t.Fatal("Beijing time parsing failed")
	}
	dayValue := "2026-09-01"
	day, err := parseProcessNode(&dayValue, nil)
	if err != nil || day.In(timelineZone).Format("2006-01-02 15:04") != "2026-09-01 00:00" {
		t.Fatal("date-only Beijing parsing failed")
	}
	a := time.Date(2026, 9, 1, 1, 30, 0, 0, time.UTC)
	b := a.Add(time.Hour)
	previous := a.AddDate(0, 0, -1)
	now := b.Add(time.Hour)
	future := now.AddDate(0, 0, 1)
	cases := []struct {
		p  models.OrderProcess
		ok bool
	}{{models.OrderProcess{PlannedStartAt: &a}, false}, {models.OrderProcess{PlannedStartAt: &b, PlannedEndAt: &previous}, false}, {models.OrderProcess{PlannedStartAt: &a, PlannedEndAt: &a}, true}, {models.OrderProcess{PlannedStartAt: &a, PlannedEndAt: &b}, true}, {models.OrderProcess{ActualStartAt: &b, ActualEndAt: &previous}, false}, {models.OrderProcess{ActualStartAt: &b, ActualEndAt: &a}, true}, {models.OrderProcess{ActualEndAt: &a}, true}, {models.OrderProcess{ActualStartAt: &future}, false}, {models.OrderProcess{ActualStartAt: &now}, true}}
	for i, c := range cases {
		err := validateProcessNodes(c.p, now.Add(-time.Minute))
		if (err == nil) != c.ok {
			t.Fatalf("validation case %d: %v", i, err)
		}
	}
}

func TestProcessTimelineTemporaryTables(t *testing.T) {
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
	repo := repository.NewOrderRepository(db)
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
	call(h.List, "", "1", "", "user", 403)
	call(h.SaveRoute, `{ "processes":[{"id":1,"name":"铣削"}] }`, "1", "", "user", 403)
	exec("INSERT INTO permissions VALUES(1,0,1)")
	data := call(h.List, "", "1", "", "user", 200)["data"].(map[string]interface{})
	if date, ok := data["start_date"].(string); !ok || !strings.HasPrefix(date, "2026-09-20") {
		t.Fatal("timeline order date fallback missing")
	}
	exec("UPDATE order_items SET start_date='2026-09-25' WHERE id=1")
	partDate := call(h.List, "", "1", "", "user", 200)["data"].(map[string]interface{})
	if date, ok := partDate["start_date"].(string); !ok || !strings.HasPrefix(date, "2026-09-25") {
		t.Fatal("timeline must prefer part order date")
	}
	exec("UPDATE orders SET start_date=NULL WHERE id=1")
	exec("UPDATE order_items SET start_date=NULL WHERE id=1")
	if call(h.List, "", "1", "", "user", 200)["data"].(map[string]interface{})["start_date"] != nil {
		t.Fatal("timeline invented missing order date")
	}
	if data["can_manage"] != false {
		t.Fatal("read-only dashboard gained write permission")
	}
	if data["quantity"] != float64(10) || data["delivery"].(map[string]interface{})["last_date"] != "" {
		t.Fatal("timeline shipping metadata lost quantity or invented date")
	}
	if _, leak := data["processes"].([]interface{})[0].(map[string]interface{})["outsourcing_fee"]; leak {
		t.Fatal("timeline leaked amounts")
	}
	body := `{"planned_start_at":"2026-09-01T08:00","planned_end_at":"2026-09-01T10:00"}`
	call(h.Save, body, "1", "1", "user", 403)
	exec("INSERT INTO permissions VALUES(1,0,2)")
	// Legacy aggregate statuses must not change while scheduling dates only.
	exec("UPDATE order_items SET status='completed' WHERE id=1")
	exec("UPDATE orders SET status='completed' WHERE id=1")
	call(h.Save, body, "1", "1", "user", 200)
	var scheduledItem models.OrderItem
	check(db.First(&scheduledItem, 1).Error)
	if scheduledItem.Status != models.OrderStatusCompleted {
		t.Fatal("planning time changed legacy item status")
	}
	var scheduledOrder models.Order
	check(db.First(&scheduledOrder, 1).Error)
	if scheduledOrder.Status != models.OrderStatusCompleted {
		t.Fatal("planning time changed legacy order status")
	}
	exec("UPDATE order_items SET status='pending' WHERE id=1")
	exec("UPDATE orders SET status='pending' WHERE id=1")
	call(h.List, "", "2", "", "admin", 404)
	call(h.Save, body, "1", "2", "admin", 404)
	call(h.Save, body, "2", "2", "admin", 404)
	call(h.Save, body, "1", "3", "admin", 400)
	call(h.Save, `{"planned_end_at":"2026-08-30T10:00"}`, "1", "1", "admin", 400)
	call(h.Save, `{"actual_start_at":"2026-09-01T08:00"}`, "1", "1", "admin", 400)
	call(h.Save, `{"actual_start_at":"2099-09-01T08:00","reason":"invalid"}`, "1", "1", "admin", 400)
	start := "processing"
	_, _, err = repo.UpdateProcessForCorp(0, 1, 1, &start, nil, nil, 1)
	check(err)
	var p models.OrderProcess
	check(db.First(&p, 1).Error)
	if p.ActualStartAt == nil || p.ActualEndAt != nil {
		t.Fatal("start did not create actual node")
	}
	stamp := p.ActualStartAt.UnixMilli()
	_, _, err = repo.UpdateProcessForCorp(0, 1, 1, &start, nil, nil, 1)
	check(err)
	check(db.First(&p, 1).Error)
	if p.ActualStartAt.UnixMilli() != stamp {
		t.Fatal("retry changed start timestamp")
	}
	completed := "completed"
	_, _, err = repo.UpdateProcessForCorp(0, 1, 1, &completed, nil, nil, 1)
	check(err)
	check(db.First(&p, 1).Error)
	if p.ActualEndAt == nil || p.ActualEndAt.Before(*p.ActualStartAt) {
		t.Fatal("completion timestamps invalid")
	}
	pending := "pending"
	_, _, err = repo.UpdateProcessForCorp(0, 1, 1, &pending, nil, nil, 1)
	if !errors.Is(err, repository.ErrTimedProcess) {
		t.Fatal("completed timing silently discarded")
	}
	call(h.Save, `{"actual_start_at":"2026-09-01T10:00","actual_end_at":"2026-08-31T08:00","reason":"invalid"}`, "1", "1", "admin", 400)
	call(h.Save, `{"actual_start_at":"2026-09-01T08:00","actual_end_at":"2026-09-01T10:00","reason":"根据车间记录补录"}`, "1", "1", "admin", 200)
	check(db.First(&p, 1).Error)
	if p.Status != models.ProcessStatusCompleted || p.PlannedStartAt == nil || p.OutsourcingFee != 100 {
		t.Fatal("manual correction lost plan/status/cost")
	}
	call(h.Save, `{"planned_start_at":"","planned_end_at":""}`, "1", "1", "admin", 200)
	// Scan NULLs into a fresh object; GORM leaves existing pointer values on reused destinations.
	p = models.OrderProcess{}
	check(db.First(&p, 1).Error)
	if p.PlannedStartAt != nil || p.ActualStartAt == nil {
		t.Fatal("clearing plan affected actual")
	}
	var n int64
	check(db.Model(&models.ProcessTimeEvent{}).Count(&n).Error)
	beforeCount := n
	call(h.Save, `{"actual_start_at":"2026-09-01T08:00","actual_end_at":"2026-09-01T10:00","reason":"根据车间记录补录"}`, "1", "1", "admin", 200)
	check(db.Model(&models.ProcessTimeEvent{}).Count(&n).Error)
	if n != beforeCount {
		t.Fatal("same time correction duplicated audit")
	}
	if !errors.Is(repo.Delete(1, 0), repository.ErrTimedProcess) {
		t.Fatal("timed order deleted")
	}
	// Editing ordinary fields preserves time nodes; deleting a timed process rolls the whole order back.
	var item models.OrderItem
	check(db.First(&item, 1).Error)
	check(db.Preload("Processes").First(&item, 1).Error)
	var order models.Order
	check(db.First(&order, 1).Error)
	check(repo.Update(&order, []models.OrderItem{item}, repository.OrderUpdateOptions{ActorID: 1}))
	check(db.First(&p, 1).Error)
	if p.ActualStartAt == nil || p.ActualEndAt == nil {
		t.Fatal("order edit erased timestamps")
	}

	routeBody := `{"processes":[{"id":1,"name":"铣削（精加工）","is_outsourced":true,"status":"pending","outsourcing_fee":0,"actual_start_at":""},{"id":3,"name":"送货"},{"id":4,"name":"历史加工"},{"name":"新增检验"}]}`
	call(h.SaveRoute, routeBody, "1", "", "user", 200)
	saved := models.OrderProcess{}
	check(db.First(&saved, 1).Error)
	if saved.Name != "铣削（精加工）" || saved.Status != "completed" || saved.OutsourcingFee != 100 || saved.ActualStartAt == nil || saved.ActualEndAt == nil {
		t.Fatal("route API changed status, price or actual nodes")
	}
	call(h.SaveRoute, `{"processes":[{"id":2,"name":"外国工序"}]}`, "1", "", "admin", 400)
	call(h.SaveRoute, `{"processes":[{"id":1,"name":"错误计划","is_outsourced":true,"planned_start_at":"2026-10-08","planned_end_at":"2026-10-07"}]}`, "1", "", "admin", 400)
	item.Processes = []models.OrderProcess{}
	if !errors.Is(repo.Update(&order, []models.OrderItem{item}), repository.ErrTimedProcess) {
		t.Fatal("timed process removed")
	}
	check(db.Table("production_completions").Count(&n).Error)
	if n != 0 {
		t.Fatal("process time generated output value")
	}
	check(db.First(&item, 1).Error)
	if item.Quantity != 10 || item.UnitPrice != 3 || item.TotalPrice != 30 {
		t.Fatal("timeline changed quantity/price")
	}
	var legacy models.OrderProcess
	check(db.First(&legacy, 4).Error)
	if legacy.ActualStartAt != nil || legacy.ActualEndAt != nil {
		t.Fatal("legacy dates invented")
	}
	// The actual order entry API persists optional daily plans in the same order transaction.
	orders := NewOrderHandler(repo, repository.NewOrderItemRepository(db), repository.NewOrderProcessRepository(db), nil, db)
	createBody := `{"customer_id":1,"order_number":"DAY-ENTRY","start_date":"2026-10-06","due_date":"2026-10-08","items":[{"part_name":"DAY-ENTRY","quantity":10,"unit_price":3,"processes":[{"name":"铣削","status":"pending","planned_start_at":"2026-10-06","planned_end_at":"2026-10-06"}]}]}`
	created := call(orders.Create, createBody, "", "", "user", 200)
	createdID := fmt.Sprintf("%.0f", created["id"].(float64))
	var entryItem models.OrderItem
	check(db.Where("order_id=?", createdID).Take(&entryItem).Error)
	var entryProcess models.OrderProcess
	check(db.Where("order_item_id=?", entryItem.ID).Take(&entryProcess).Error)
	if entryProcess.PlannedStartAt == nil || entryProcess.PlannedEndAt == nil || entryProcess.PlannedStartAt.In(timelineZone).Format("2006-01-02") != "2026-10-06" || entryProcess.ActualStartAt != nil || entryProcess.ActualEndAt != nil {
		t.Fatal("order entry did not persist date-only plan or invented actual dates")
	}
	orderPayload := func(nodes string) string {
		return fmt.Sprintf(`{"customer_id":1,"order_number":"DAY-ENTRY","start_date":"2026-10-06","due_date":"2026-10-08","items":[{"id":%d,"part_name":"DAY-ENTRY","quantity":10,"unit_price":3,"processes":[{"id":%d,"name":"铣削","status":"pending"%s}]}]}`, entryItem.ID, entryProcess.ID, nodes)
	}
	call(orders.Update, orderPayload(`,"planned_start_at":"2026-10-07","planned_end_at":"2026-10-07"`), createdID, "", "user", 200)
	call(orders.Update, orderPayload(""), createdID, "", "user", 200)
	entryProcess = models.OrderProcess{}
	check(db.Where("order_item_id=?", entryItem.ID).Take(&entryProcess).Error)
	if entryProcess.PlannedStartAt == nil || entryProcess.PlannedStartAt.In(timelineZone).Format("2006-01-02") != "2026-10-07" {
		t.Fatal("order plan update failed or omitted dates erased the plan")
	}
	call(orders.Update, orderPayload(`,"planned_start_at":"2026-10-08","planned_end_at":"2026-10-07"`), createdID, "", "user", 400)
	call(orders.Update, orderPayload(`,"planned_start_at":"","planned_end_at":""`), createdID, "", "user", 200)
	entryProcess = models.OrderProcess{}
	check(db.Where("order_item_id=?", entryItem.ID).Take(&entryProcess).Error)
	if entryProcess.PlannedStartAt != nil || entryProcess.PlannedEndAt != nil || entryProcess.ActualStartAt != nil || entryProcess.ActualEndAt != nil {
		t.Fatal("explicit plan clear failed or affected actual dates")
	}
	check(db.Model(&models.ProcessTimeEvent{}).Where("process_id=? AND source='order'", entryProcess.ID).Count(&n).Error)
	if n != 2 {
		t.Fatal("order plan changes not audited")
	}
	// An explicitly approved legacy retirement keeps actual dates, fee snapshots and order-delete protection.
	call(h.SaveRoute, `{"allow_processed_changes":true,"reason":"取消历史铣削，保留原加工记录","processes":[{"id":3,"name":"送货"},{"id":4,"name":"历史加工"}]}`, "1", "", "admin", 200)
	retired := models.OrderProcess{}
	check(db.Unscoped().First(&retired, 1).Error)
	if !retired.DeletedAt.Valid || retired.ActualStartAt == nil || retired.ActualEndAt == nil || retired.OutsourcingFee != 100 || retired.Status != "completed" {
		t.Fatal("legacy retirement erased actual dates, fee or status")
	}
	foreign := models.OrderProcess{}
	check(db.First(&foreign, 2).Error)
	if foreign.Name != "FOREIGN" || foreign.DeletedAt.Valid {
		t.Fatal("foreign company route changed")
	}
	if !errors.Is(repo.Delete(1, 0), repository.ErrTimedProcess) {
		t.Fatal("archived actual production allowed whole-order deletion")
	}

}
