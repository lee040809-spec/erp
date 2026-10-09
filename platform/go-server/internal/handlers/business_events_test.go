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
	"machining-erp/internal/repository"
	"machining-erp/internal/workflowtest"
	"net/http/httptest"
	"os"
	"regexp"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestBusinessEventsTemporaryTables(t *testing.T) {
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
	exec := func(query string) { t.Helper(); check(db.Exec(query).Error) }

	var actualAccount struct {
		ID       int64
		CorpID   int64
		RoleType string
	}
	check(db.Table("users").Select("id,corp_id,role_type").Where("username=?", "lilin").Take(&actualAccount).Error)
	actualContext, _ := gin.CreateTestContext(httptest.NewRecorder())
	actualContext.Set("userID", actualAccount.ID)
	actualContext.Set("corpID", actualAccount.CorpID)
	actualContext.Set("roleType", actualAccount.RoleType)
	actualFinance := NewProductionFinanceHandler(db)
	if !actualFinance.canImportHistory(actualContext) {
		t.Fatal("affected lilin account still cannot import historical completions")
	}
	t.Logf("actual lilin permissions (read-only check): history import=%v, cost administration=%v", actualFinance.canImportHistory(actualContext), actualFinance.canManageFinanceCosts(actualContext))
	for _, name := range []string{"orders", "order_items", "order_processes", "process_time_events", "production_completions", "cost_entries", "cost_allocations", "monthly_closings", "production_plans", "order_business_records", "business_entry_policies", "users", "roles", "permissions", "resources"} {
		var table, ddl string
		check(pool.QueryRow("SHOW CREATE TABLE `"+name+"`").Scan(&table, &ddl))
		kept := []string{}
		for _, line := range strings.Split(ddl, "\n") {
			if !strings.Contains(line, "FOREIGN KEY") {
				kept = append(kept, line)
			}
		}
		ddl = strings.Join(kept, "\n")
		ddl = regexp.MustCompile(",\\s*\\)").ReplaceAllString(ddl, "\n)")
		ddl = strings.Replace(ddl, "CREATE TABLE", "CREATE TEMPORARY TABLE", 1)
		ddl = regexp.MustCompile("AUTO_INCREMENT=[0-9]+").ReplaceAllString(ddl, "AUTO_INCREMENT=1")
		exec(ddl)
	}
	workflowtest.Tables(t, db, "production_completions", "order_business_records", "cost_entries")
	exec("INSERT INTO orders(id,corp_id,order_number,status,due_date) VALUES(1,0,'EVENT-TEST','processing','2026-07-10'),(2,1,'OTHER','processing','2026-07-10')")
	exec("INSERT INTO order_items(id,corp_id,order_id,part_name,quantity,production_quantity,unit_price,status,supply_mode,currency) VALUES(1,0,1,'BUFFER',100,110,10,'processing','self_purchased','CNY'),(2,1,2,'OTHER',10,10,10,'processing','self_purchased','CNY'),(3,0,1,'HISTORY',10,12,5,'completed','self_purchased','CNY')")
	exec("INSERT INTO order_processes(id,corp_id,order_item_id,name,status) VALUES(1,0,1,'加工','completed'),(2,0,1,'送货','pending')")
	h := NewBusinessEventHandler(db)
	f := NewProductionFinanceHandler(db)
	p := NewBusinessPolicyHandler(db)
	corp := int64(0)
	role := "admin"
	call := func(fn gin.HandlerFunc, payload interface{}, params gin.Params, url string, want int) map[string]interface{} {
		t.Helper()
		body, _ := json.Marshal(payload)
		ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
		rec := httptest.NewRecorder()
		ctx, _ = gin.CreateTestContext(rec)
		ctx.Request = httptest.NewRequest("POST", url, strings.NewReader(string(body)))
		ctx.Request.Header.Set("Content-Type", "application/json")
		ctx.Params = params
		ctx.Set("corpID", corp)
		ctx.Set("userID", int64(1))
		ctx.Set("roleType", role)
		ctx.Set("corpName", "测试")
		fn(ctx)
		if rec.Code != want {
			t.Fatalf("%d want %d: %s", rec.Code, want, rec.Body.String())
		}
		result := map[string]interface{}{}
		check(json.Unmarshal(rec.Body.Bytes(), &result))
		return result
	}
	orderParams := gin.Params{{Key: "id", Value: "1"}}
	itemParams := gin.Params{{Key: "itemId", Value: "1"}}
	event := func(kind string, quantity int, date, request string) map[string]interface{} {
		return map[string]interface{}{"kind": kind, "order_item_id": 1, "quantity": quantity, "business_date": date, "request_id": request, "severity": "normal"}
	}
	scrap := event("scrap", 5, "2026-06-01", "scrap-1")
	call(h.Create, scrap, orderParams, "/", 200)
	call(h.Create, scrap, orderParams, "/", 200)
	var sum int
	check(db.Table("order_items").Where("id=1").Select("scrap_quantity").Scan(&sum).Error)
	if sum != 5 {
		t.Fatal("duplicate scrap counted twice")
	}
	changed := event("scrap", 6, "2026-06-01", "scrap-1")
	call(h.Create, changed, orderParams, "/", 400)
	complete := func(q int, date, request string) {
		call(f.CreateCompletion, map[string]interface{}{"qualified_quantity": q, "business_date": date, "request_id": request}, itemParams, "/", 200)
	}
	complete(80, "2026-06-01", "q-first")
	// A partially produced order is available to deliver before its total contract is complete.
	result, err := repository.NewOrderRepository(db).GetWithFilters(repository.OrderFilters{CorpID: 0, Status: "delivery_ready", Page: 1, PageSize: 20})
	check(err)
	if result.Total != 1 {
		t.Fatal("partial qualified production omitted from delivery")
	}
	delivery1 := call(h.Create, event("delivery", 60, "2026-06-02", "delivery-1"), orderParams, "/", 200)["data"].(map[string]interface{})
	call(h.Create, event("delivery", 21, "2026-06-02", "delivery-too-many"), orderParams, "/", 400)
	complete(25, "2026-07-01", "q-final")
	call(h.Create, event("delivery", 40, "2026-06-01", "backdated-delivery"), orderParams, "/", 400)
	delivery2 := call(h.Create, event("delivery", 40, "2026-07-03", "delivery-2"), orderParams, "/", 200)["data"].(map[string]interface{})
	call(h.Create, event("delivery", 40, "2026-07-03", "delivery-2"), orderParams, "/", 200)
	call(h.Create, event("delivery", 1, "2026-07-03", "delivery-spare"), orderParams, "/", 400)
	call(h.Create, event("scrap", 1, "2026-07-03", "scrap-too-many"), orderParams, "/", 400)
	var item models.OrderItem
	check(db.Where("id=1").Take(&item).Error)
	if item.ScrapQuantity != 5 || item.DeliveredQty == nil || *item.DeliveredQty != 100 || item.Status != "delivered" || item.CompletionDate == nil || item.CompletionDate.Format("2006-01-02") != "2026-07-01" {
		t.Fatalf("item aggregate incorrect %+v", item)
	}
	printed := call(h.Print, nil, gin.Params{{Key: "recordId", Value: strconvID(delivery2["id"])}}, "/", 200)["data"].(map[string]interface{})
	printOrder := printed["order"].(map[string]interface{})
	printItems := printOrder["items"].([]interface{})
	if len(printItems) != 1 || printItems[0].(map[string]interface{})["quantity"] != float64(40) || printOrder["total_amount"] != float64(400) || !strings.HasPrefix(printOrder["due_date"].(string), "2026-07-10") {
		t.Fatalf("specific delivery print wrong: %+v", printed)
	}
	var oldDate string
	check(db.Raw("SELECT DATE_FORMAT(completion_date,'%Y-%m-%d') FROM order_items WHERE id=1").Scan(&oldDate).Error)
	exec("UPDATE order_items SET production_quantity=112 WHERE id=1")
	complete(2, "2026-08-01", "q-spares")
	var newDate string
	check(db.Raw("SELECT DATE_FORMAT(completion_date,'%Y-%m-%d') FROM order_items WHERE id=1").Scan(&newDate).Error)
	if oldDate != newDate {
		t.Fatal("spare completion overwrote contractual date")
	}
	exception := map[string]interface{}{"kind": "exception", "business_date": "2026-07-04", "request_id": "exception-1", "category": "质量异常", "severity": "urgent", "description": "客户反馈尺寸问题"}
	row := call(h.Create, exception, orderParams, "/", 200)["data"].(map[string]interface{})
	params := gin.Params{{Key: "recordId", Value: strconvID(row["id"])}}
	call(h.Resolve, map[string]interface{}{"resolution": "复检完成，安排补件"}, params, "/", 200)
	call(h.Resolve, map[string]interface{}{"resolution": "重复处理"}, params, "/", 400)
	call(h.Reverse, map[string]interface{}{"reason": "登记数量错误"}, gin.Params{{Key: "recordId", Value: strconvID(delivery2["id"])}}, "/", 200)
	check(db.Where("id=1").Take(&item).Error)
	if *item.DeliveredQty != 60 || item.Status != "completed" {
		t.Fatal("reversed delivery did not update status and quantity")
	}
	call(h.Create, event("delivery", 40, "2026-07-05", "delivery-corrected"), orderParams, "/", 200)
	corp = 1
	call(h.List, nil, orderParams, "/", 404)
	call(h.Print, nil, gin.Params{{Key: "recordId", Value: strconvID(delivery1["id"])}}, "/", 404)
	corp = 0
	check(db.Exec("INSERT INTO roles(id,corp_id,name,account_type) VALUES(1,0,'业务','user')").Error)
	check(db.Exec("INSERT INTO users(id,corp_id,username,password,role_id,role_type) VALUES(1,0,'temp','temp',1,'user')").Error)
	exec("INSERT INTO resources(id,resource_type,resource_key,name) VALUES(100,'page','orders','订单')")
	exec("INSERT INTO permissions(corp_id,role_id,resource_id) VALUES(0,1,100)")
	call(p.Save, map[string]interface{}{"defaults": EvidencePolicy{true, true, true, true, true}, "role_modes": map[string]string{}}, nil, "/", 200)
	history := map[string]interface{}{"order_item_id": 3, "qualified_quantity": 4, "business_date": "2026-04-01", "expected_price": "5", "request_id": "optional-1", "reviewed": true}
	call(f.ImportHistory, history, nil, "/", 400)
	role = "user"
	call(p.Save, map[string]interface{}{"defaults": EvidencePolicy{}}, nil, "/", 403)
	role = "admin"
	call(p.Save, map[string]interface{}{"defaults": EvidencePolicy{true, true, true, true, true}, "role_modes": map[string]string{"1": "basic"}}, nil, "/", 200)
	role = "user"
	policy, err := effectiveEvidencePolicy(db, contextForPolicy())
	check(err)
	if policy.SourceRef {
		t.Fatal("role override failed")
	}
	// Finance write permission is kept separate from field requirements.
	call(f.ImportHistory, history, nil, "/", 403)

	// monthly_output grants historical production entry, without cost administration.
	exec("INSERT INTO resources(id,resource_type,resource_key,name) VALUES(101,'menu','monthly_output','产值统计'),(102,'menu','finance','财务报表')")
	exec("INSERT INTO permissions(corp_id,role_id,resource_id) VALUES(0,1,101)")
	access := call(f.GetFinanceAccess, nil, nil, "/", 200)["data"].(map[string]interface{})
	if access["can_import_history"] != true || access["can_manage_costs"] != false {
		t.Fatal("history permission coupled to cost administration")
	}
	call(f.ImportHistory, history, nil, "/", 200)
	call(f.ImportHistory, history, nil, "/", 200)
	call(f.CreateCostEntry, map[string]interface{}{"cost_type": "material", "amount": "1.00", "business_date": "2026-04-01"}, nil, "/", 403)
	call(f.AllocateCost, map[string]interface{}{"cost_entry_id": 1, "production_completion_id": 1, "allocated_amount": "1.00"}, nil, "/", 403)
	call(f.CloseMonth, map[string]interface{}{"reviewed_cost_complete": true}, gin.Params{{Key: "month", Value: "2026-04"}}, "/", 403)
	// Revocation is checked before idempotent replay; finance read access is not import access.
	exec("DELETE FROM permissions WHERE resource_id=101")
	call(f.ImportHistory, history, nil, "/", 403)
	exec("INSERT INTO permissions(corp_id,role_id,resource_id) VALUES(0,1,102)")
	access = call(f.GetFinanceAccess, nil, nil, "/", 200)["data"].(map[string]interface{})
	if access["can_import_history"] != false || access["can_manage_costs"] != false {
		t.Fatal("read-only finance user can write")
	}
	call(f.PreviewHistory, nil, nil, "/?year=2026", 200)
	call(f.ImportHistory, history, nil, "/", 403)
	t.Log("output-statistics user imports optional-evidence history successfully; duplicates, revocation, finance read-only access, costs, allocations and closing remain protected")
	role = "admin"
	policyData := call(p.Get, nil, nil, "/", 200)["data"].(map[string]interface{})
	if policyData["effective"].(map[string]interface{})["source_ref"] != false {
		t.Fatal("role-specific basic policy not applied")
	}
	call(f.ImportHistory, history, nil, "/", 200)
	call(f.ImportHistory, history, nil, "/", 200)
	history["qualified_quantity"] = 8
	history["business_date"] = "2026-05-01"
	history["request_id"] = "optional-2"
	call(f.ImportHistory, history, nil, "/", 200)
	var total float64
	check(db.Table("production_completions").Where("order_item_id=3").Select("SUM(output_value)").Scan(&total).Error)
	if total != 50 {
		t.Fatal("optional evidence split imports changed billable cap")
	}
	// Repeated history must remain accessible without loading all rows in one response.
	for i := 0; i < 52; i++ {
		check(db.Create(&BusinessRecord{CorpID: 0, OrderID: 1, Kind: "exception", Category: "分页测试", RequestID: fmt.Sprintf("paging-%d", i), Status: "open", CreatedBy: 1}).Error)
	}
	pageOne := call(h.List, nil, orderParams, "/?page=1&pageSize=50", 200)["data"].(map[string]interface{})
	pageTwo := call(h.List, nil, orderParams, "/?page=2&pageSize=50", 200)["data"].(map[string]interface{})
	if len(pageOne["records"].([]interface{})) != 50 || pageOne["has_more"] != true || len(pageTwo["records"].([]interface{})) == 0 {
		t.Fatal("business paging truncated records")
	}
	// Lazy drawings use order-view permissions and enforce company boundaries.
	exec("UPDATE order_items SET drawing_data='data:image/png;base64,TEST' WHERE id=1")
	drawingHandler := &OrderHandler{db: db}
	call(drawingHandler.Drawing, nil, itemParams, "/", 200)
	corp = 1
	call(drawingHandler.Drawing, nil, itemParams, "/", 404)
	corp = 0
	role = "user"
	call(RuntimeMetrics(db), nil, nil, "/", 403)
	role = "admin"
	metrics := call(RuntimeMetrics(db), nil, nil, "/", 200)
	if metrics["heap_alloc_bytes"] == nil || metrics["db_open"] == nil {
		t.Fatal("runtime metrics missing")
	}
	if os.Getenv("RUN_MEMORY_SOAK") == "1" {
		runtime.GC()
		var before, after runtime.MemStats
		runtime.ReadMemStats(&before)
		until := time.Now().Add(time.Minute)
		requests := 0
		for time.Now().Before(until) {
			call(h.List, nil, orderParams, "/?page=1&pageSize=50", 200)
			requests++
		}
		runtime.GC()
		runtime.ReadMemStats(&after)
		t.Logf("60-second read-only soak: requests=%d heap_before=%d heap_after=%d goroutines=%d", requests, before.HeapAlloc, after.HeapAlloc, runtime.NumGoroutine())
		if after.HeapAlloc > before.HeapAlloc+(32<<20) {
			t.Fatal("read-only soak retained excessive heap")
		}
	}
	t.Log("paged business history, scoped drawings and admin-only runtime metrics verified")
	t.Log("scrap/delivery aggregates, partial delivery, current-batch printing, chronological completion, spare cap, duplicate prevention, reversal, company isolation, exception resolution, admin settings and optional evidence verified")
}
func strconvID(value interface{}) string { return fmt.Sprintf("%.0f", value.(float64)) }
func contextForPolicy() *gin.Context {
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	ctx.Set("corpID", int64(0))
	ctx.Set("userID", int64(1))
	ctx.Set("roleType", "user")
	return ctx
}
