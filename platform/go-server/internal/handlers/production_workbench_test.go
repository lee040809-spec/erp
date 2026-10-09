package handlers

import (
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	driver "github.com/go-sql-driver/mysql"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"machining-erp/internal/repository"
	"machining-erp/internal/services"
	"machining-erp/internal/workflowtest"
	"machining-erp/pkg/jwt"
	"net/http/httptest"
	"os"
	"reflect"
	"regexp"
	"strings"
	"testing"
	"time"
)

func TestWorkbenchTemporaryTables(t *testing.T) {
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
	exec := func(sql string) { t.Helper(); check(db.Exec(sql).Error) }
	for _, name := range []string{"orders", "order_items", "order_processes", "process_time_events", "production_completions", "production_plans", "order_business_records", "business_entry_policies"} {
		var n int64
		check(db.Table(name).Count(&n).Error)
		t.Logf("real table %s rows: %d", name, n)
		var table, ddl string
		check(pool.QueryRow("SHOW CREATE TABLE `"+name+"`").Scan(&table, &ddl))
		lines := strings.Split(ddl, "\n")
		kept := []string{}
		for _, line := range lines {
			if !strings.Contains(line, "FOREIGN KEY") {
				kept = append(kept, line)
			}
		}
		ddl = strings.Join(kept, "\n")
		ddl = regexp.MustCompile(",\\s*\\)").ReplaceAllString(ddl, "\n)")
		exec(strings.Replace(ddl, "CREATE TABLE", "CREATE TEMPORARY TABLE", 1))
	}
	workflowtest.Tables(t, db, "production_completions", "order_business_records")
	exec("CREATE TEMPORARY TABLE users(id BIGINT,corp_id BIGINT,role_id BIGINT)")
	exec("CREATE TEMPORARY TABLE permissions(role_id BIGINT,corp_id BIGINT,resource_id BIGINT)")
	exec("CREATE TEMPORARY TABLE resources(id BIGINT,resource_key VARCHAR(100),status VARCHAR(20) DEFAULT 'active',sort_order INT DEFAULT 0)")
	exec("INSERT INTO users VALUES(1,0,1)")
	exec("INSERT INTO resources(id,resource_key) VALUES(1,'production_dashboard'),(2,'dashboard')")
	exec("INSERT INTO orders(id,corp_id,order_number,priority) VALUES(1,0,'WB-1','high'),(2,1,'FOREIGN','medium')")
	exec("UPDATE orders SET start_date='2026-09-20' WHERE id=1")
	exec("INSERT INTO order_items(id,corp_id,order_id,part_name,quantity,status,due_date) VALUES(1,0,1,'TEST',10,'processing',CURDATE()),(2,0,1,'LATE',20,'pending',DATE_SUB(CURDATE(),INTERVAL 1 DAY)),(3,1,2,'FOREIGN',10,'pending',CURDATE()),(4,0,1,'DONE',10,'completed',CURDATE())")
	exec("UPDATE order_items SET start_date='2026-09-25' WHERE id=1")
	exec("INSERT INTO order_processes(id,corp_id,order_item_id,name,status,is_outsourced) VALUES(1,0,1,'加工','processing',1)")
	h := NewProductionWorkbenchHandler(db)
	gin.SetMode(gin.TestMode)
	call := func(fn gin.HandlerFunc, body, url, role string, want int) map[string]interface{} {
		t.Helper()
		rec := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(rec)
		c.Request = httptest.NewRequest("PATCH", url, strings.NewReader(body))
		c.Request.Header.Set("Content-Type", "application/json")
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
	result := call(h.List, "", "/?pageSize=1", "admin", 200)["data"].(map[string]interface{})
	if result["total"] != float64(2) || len(result["items"].([]interface{})) != 1 {
		t.Fatal("pagination or company scope failed")
	}
	if result["items"].([]interface{})[0].(map[string]interface{})["start_date"] != "2026-09-25" {
		t.Fatal("newest part order date must precede an older overdue task")
	}
	partDate := call(h.List, "", "/?search=TEST", "admin", 200)["data"].(map[string]interface{})["items"].([]interface{})[0].(map[string]interface{})
	if partDate["start_date"] != "2026-09-25" {
		t.Fatal("part order date must take precedence")
	}
	exec("UPDATE orders SET start_date=NULL WHERE id=1")
	missingDate := call(h.List, "", "/?search=LATE", "admin", 200)["data"].(map[string]interface{})["items"].([]interface{})[0].(map[string]interface{})
	if missingDate["start_date"] != "" {
		t.Fatal("missing business date must not use creation timestamp")
	}
	exec("UPDATE orders SET start_date='2026-09-20' WHERE id=1")
	stats := result["stats"].(map[string]interface{})
	if stats["unassigned"] != float64(2) || stats["outsourced"] != float64(1) {
		t.Fatal("stats incorrect")
	}
	outsourced := call(h.List, "", "/?view=outsourced", "admin", 200)["data"].(map[string]interface{})
	if outsourced["total"] != float64(1) {
		t.Fatal("outsourced category must return its matching item")
	}
	for _, category := range []string{"all", "unassigned", "today", "overdue", "blocked", "outsourced"} {
		call(h.List, "", "/?view="+category, "admin", 200)
	}
	for _, sample := range []struct {
		search string
		want   float64
	}{{"TEST", 1}, {"LATE", 0}} {
		data := call(h.List, "", "/?view=outsourced&search="+sample.search, "admin", 200)["data"].(map[string]interface{})
		if data["total"] != sample.want {
			t.Fatal("outsourced search does not match its category")
		}
	}
	// Pending external work belongs to the current category; completed and delivery steps do not.
	exec("UPDATE order_processes SET status='pending' WHERE id=1")
	if call(h.List, "", "/?view=outsourced", "admin", 200)["data"].(map[string]interface{})["total"] != float64(1) {
		t.Fatal("pending external work must be included")
	}
	exec("UPDATE order_processes SET status='completed' WHERE id=1")
	if call(h.List, "", "/?view=outsourced", "admin", 200)["data"].(map[string]interface{})["total"] != float64(0) {
		t.Fatal("completed external work must be excluded")
	}
	exec("UPDATE order_processes SET name='送货',status='processing' WHERE id=1")
	if call(h.List, "", "/?view=outsourced", "admin", 200)["data"].(map[string]interface{})["total"] != float64(0) {
		t.Fatal("delivery must not count as external manufacturing")
	}
	exec("UPDATE order_processes SET name='加工' WHERE id=1")
	summary := result["summary"].(map[string]interface{})
	if summary["orders"] != float64(1) || summary["items"] != float64(2) || summary["pieces"] != float64(30) {
		t.Fatal("order count, item rows and piece quantity must be distinct and independent of pagination")
	}
	filtered := call(h.List, "", "/?search=LATE", "admin", 200)["data"].(map[string]interface{})["summary"].(map[string]interface{})
	if filtered["orders"] != float64(1) || filtered["items"] != float64(1) || filtered["pieces"] != float64(20) {
		t.Fatal("filtered counts incorrect")
	}
	call(h.Save, `{"item_ids":[1,2],"assignee":"张师傅","equipment":"加工中心","planned_start":"2026-10-05","planned_end":"2026-10-06","estimated_hours":"4.50"}`, "/", "admin", 200)
	var count int64
	check(db.Table("production_plans").Where("assignee='张师傅'").Count(&count).Error)
	if count != 2 {
		t.Fatal("batch save missing")
	}
	call(h.Save, `{"item_ids":[1,3],"assignee":"BAD"}`, "/", "admin", 400)
	call(h.Save, `{"item_ids":[1,4],"assignee":"BAD"}`, "/", "admin", 400)
	call(h.Save, `{"item_ids":[1,2],"planned_start":"2026-10-07"}`, "/", "admin", 400)
	check(db.Table("production_plans").Where("assignee='张师傅' AND planned_start='2026-10-05'").Count(&count).Error)
	if count != 2 {
		t.Fatal("failed batch did not roll back")
	}
	call(h.Save, `{"item_ids":[1],"blocked_reason":"缺料"}`, "/", "admin", 200)
	result = call(h.List, "", "/?view=blocked", "admin", 200)["data"].(map[string]interface{})
	if result["total"] != float64(1) {
		t.Fatal("blocked filter not linked")
	}
	call(h.Save, `{"item_ids":[1],"blocked_reason":""}`, "/", "admin", 200)
	call(h.List, "", "/", "user", 403)
	exec("INSERT INTO permissions VALUES(1,0,2)")
	legacy := call(h.List, "", "/", "user", 200)["data"].(map[string]interface{})
	if legacy["can_manage"] != false {
		t.Fatal("legacy dashboard must not grant new production writes")
	}
	call(h.Save, `{"item_ids":[1],"assignee":"BAD"}`, "/", "user", 403)
	exec("INSERT INTO permissions VALUES(1,0,1)")
	call(h.List, "", "/", "user", 200)
	// A user's ID must not be used as their role ID when refreshing menus.
	exec("UPDATE users SET role_id=9 WHERE id=1")
	exec("INSERT INTO permissions VALUES(9,0,2)")
	auth := services.NewAuthService(repository.NewUserRepository(db), nil, repository.NewResourceRepository(db), nil, nil, jwt.JWTConfig{}, 5, time.Minute)
	menus, menuErr := auth.GetUserResources(1, "user")
	check(menuErr)
	if len(menus) != 1 || menus[0].ResourceKey != "dashboard" {
		t.Fatal("menu refresh used user ID instead of actual role ID")
	}
	orderRepo := repository.NewOrderRepository(db)
	itemRepo := repository.NewOrderItemRepository(db)
	processRepo := repository.NewOrderProcessRepository(db)
	orders := NewOrderHandler(orderRepo, itemRepo, processRepo, nil, db)
	exec("UPDATE order_items SET unit_price=20,total_price=200,tool_cost=5,material_cost=7 WHERE id=1")
	exec("UPDATE order_processes SET outsourcing_fee=77 WHERE id=1")
	maskedOrder := call(orders.List, "", "/?pageSize=10", "user", 200)["data"].([]interface{})[0].(map[string]interface{})
	if maskedOrder["total_amount"] != nil {
		t.Fatal("production order response leaked total")
	}
	maskedItem := maskedOrder["items"].([]interface{})[0].(map[string]interface{})
	if maskedItem["unit_price"] != nil || maskedItem["material_cost"] != nil {
		t.Fatal("production order response leaked price or cost")
	}
	call(orders.Delete, "", "/", "user", 403)
	exec("INSERT INTO resources(id,resource_key) VALUES(3,'production_orders'),(4,'orders')")
	exec("INSERT INTO permissions VALUES(9,0,3)")
	request := `{"start_date":"2026-10-05","due_date":"2026-10-06","total_amount":99999,"items":[{"id":1,"part_name":"TEST","quantity":10,"unit_price":999,"tool_cost":999,"material_cost":999,"processes":[{"id":1,"name":"加工","status":"processing","is_outsourced":true,"outsourcing_fee":999}]},{"id":2,"part_name":"LATE","quantity":20},{"id":4,"part_name":"DONE","quantity":10}]}`
	patchOrder := func(fn gin.HandlerFunc, body string, want int) {
		t.Helper()
		rec := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(rec)
		c.Request = httptest.NewRequest("PATCH", "/", strings.NewReader(body))
		c.Request.Header.Set("Content-Type", "application/json")
		c.Params = gin.Params{{Key: "id", Value: "1"}}
		c.Set("corpID", int64(0))
		c.Set("userID", int64(1))
		c.Set("roleType", "user")
		fn(c)
		if rec.Code != want {
			t.Fatalf("order save status %d want %d: %s", rec.Code, want, rec.Body.String())
		}
	}
	patchOrder(orders.Update, request, 200)
	var actual struct{ UnitPrice, ToolCost, MaterialCost, TotalPrice float64 }
	check(db.Table("order_items").Where("id=1").Take(&actual).Error)
	var actualFee float64
	check(db.Table("order_processes").Where("id=1").Select("outsourcing_fee").Scan(&actualFee).Error)
	if actual.UnitPrice != 20 || actual.ToolCost != 5 || actual.MaterialCost != 7 || actual.TotalPrice != 200 || actualFee != 77 {
		t.Fatal("production order patch changed hidden financial data")
	}
	exec("INSERT INTO permissions VALUES(9,0,4)")
	request = strings.Replace(request, `"unit_price":999`, `"unit_price":0`, 1)
	patchOrder(orders.Update, request, 200)
	var actualTotal float64
	check(db.Table("orders").Where("id=1").Select("total_amount").Scan(&actualTotal).Error)
	if actualTotal != 0 {
		t.Fatal("explicit zero price did not recalculate total")
	}
	// A fulfilled order may still have physical buffer pieces to report.
	exec("UPDATE order_items SET production_quantity=12,status='completed' WHERE id=1")
	exec("INSERT INTO production_completions(corp_id,order_id,order_item_id,completion_no,business_date,qualified_quantity,billable_quantity,unit_price_snapshot,output_value,supply_mode_snapshot,currency,tax_inclusive,status,created_by,confirmed_at) VALUES(0,1,1,'WB-BUFFER',CURDATE(),10,10,20,200,'self_purchased','CNY',0,'confirmed',1,NOW())")
	buffered := call(h.List, "", "/?search=TEST", "admin", 200)["data"].(map[string]interface{})
	if buffered["total"] != float64(1) {
		t.Fatal("buffer reporting task disappeared after order fulfillment")
	}
	row := buffered["items"].([]interface{})[0].(map[string]interface{})
	if row["production_quantity"] != float64(12) || row["remaining_quantity"] != float64(2) || row["delivery_gap"] != float64(0) || row["status"] != "completed" {
		t.Fatal("buffer workbench counts incorrect", row)
	}
	exec("UPDATE production_completions SET qualified_quantity=12 WHERE completion_no='WB-BUFFER'")
	finished := call(h.List, "", "/?search=TEST", "admin", 200)["data"].(map[string]interface{})
	if finished["total"] != float64(0) {
		t.Fatal("fully reported buffer task remained pending")
	}
	// Fixtures deliberately disagree on status, due date, creation timestamp and business date.
	exec("UPDATE orders SET start_date='2026-09-20',status='pending' WHERE id=1")
	exec("UPDATE order_items SET start_date=NULL WHERE id=2")
	exec("INSERT INTO orders(id,corp_id,order_number,priority,status,start_date,created_at) VALUES(3,0,'WB-SORT-NEW','low','completed','2026-09-30','2026-09-01'),(4,0,'WB-SORT-UNDATED','high','pending',NULL,'2026-10-08')")
	exec("INSERT INTO order_items(id,corp_id,order_id,part_name,quantity,status,start_date,due_date) VALUES(5,0,3,'SORT-OLD',10,'pending','2026-09-28','2030-12-31'),(6,0,3,'SORT-FALLBACK',10,'pending',NULL,'2030-12-31'),(7,0,4,'SORT-OWN',10,'pending','2026-10-01','2030-12-31'),(8,0,4,'SORT-UNDATED',10,'pending',NULL,'2030-12-31'),(9,0,3,'SORT-TIE-A',10,'pending','2026-10-02','2030-12-31'),(10,0,3,'SORT-TIE-B',10,'pending','2026-10-02','2030-12-31')")
	wantIDs := []float64{10, 9, 7, 6, 5, 2, 8}
	gotIDs := []float64{}
	for page := 1; page <= 4; page++ {
		paged := call(h.List, "", fmt.Sprintf("/?pageSize=2&page=%d", page), "admin", 200)["data"].(map[string]interface{})
		if paged["total"] != float64(len(wantIDs)) {
			t.Fatal("sort pagination total mismatch")
		}
		for _, value := range paged["items"].([]interface{}) {
			gotIDs = append(gotIDs, value.(map[string]interface{})["item_id"].(float64))
		}
	}
	if !reflect.DeepEqual(gotIDs, wantIDs) {
		t.Fatalf("workbench order across pages: got %v want %v", gotIDs, wantIDs)
	}
	pageOne, e := orderRepo.GetWithFilters(repository.OrderFilters{CorpID: 0, Page: 1, PageSize: 2})
	check(e)
	pageTwo, e := orderRepo.GetWithFilters(repository.OrderFilters{CorpID: 0, Page: 2, PageSize: 2})
	check(e)
	if pageOne.Total != 3 || len(pageOne.Orders) != 2 || pageOne.Orders[0].ID != 3 || pageOne.Orders[1].ID != 1 || len(pageTwo.Orders) != 1 || pageTwo.Orders[0].ID != 4 {
		t.Fatal("order date sorting, null placement or tenant pagination failed")
	}
	partIDs := []int64{}
	for _, item := range pageOne.Orders[0].Items {
		partIDs = append(partIDs, item.ID)
	}
	if !reflect.DeepEqual(partIDs, []int64{10, 9, 6, 5}) {
		t.Fatalf("part dates must use own date before parent fallback: %v", partIDs)
	}
	filteredSort, e := orderRepo.GetWithFilters(repository.OrderFilters{CorpID: 0, PartName: "SORT-OLD", Page: 1, PageSize: 1})
	check(e)
	if filteredSort.Total != 1 || len(filteredSort.Orders) != 1 || filteredSort.Orders[0].ID != 3 {
		t.Fatal("shared count/list part filter drift")
	}
	filteredSecond, e := orderRepo.GetWithFilters(repository.OrderFilters{CorpID: 0, PartName: "SORT-OLD", Page: 2, PageSize: 1})
	check(e)
	if filteredSecond.Total != 1 || len(filteredSecond.Orders) != 0 {
		t.Fatal("count query contaminated by list pagination")
	}
	hugePage := call(h.List, "", "/?page=999999999999999999&pageSize=999999", "admin", 200)["data"].(map[string]interface{})
	if hugePage["page"] != float64(1000000) || hugePage["page_size"] != float64(100) || len(hugePage["items"].([]interface{})) != 0 {
		t.Fatal("unbounded pagination or offset overflow", hugePage)
	}
	t.Log("business-date descending sort, null fallback, ID ties, cross-page stability and shared count/filter checks passed")
	t.Log("fulfilled orders retain buffer reporting tasks until all planned qualified pieces are reported")
	t.Log("order API amount omission, production edit preservation, direct permission checks and explicit zero price passed")
	t.Log("company isolation, server pagination, stats, batch rollback, date validation, blocked filter, explicit clear and permission checks passed")
}
