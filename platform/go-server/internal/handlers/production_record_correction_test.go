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
	"machining-erp/internal/workflowtest"
	"net/http/httptest"
	"os"
	"regexp"
	"strings"
	"testing"
	"time"
)

func TestProductionRecordCorrectionsTemporaryTables(t *testing.T) {
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
	db, err := gorm.Open(mysql.Open(cfg.FormatDSN()), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
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
	// Every write is to a connection-local temporary table; persistent ERP rows are never touched.
	for _, name := range []string{"orders", "order_items", "order_processes", "process_time_events", "production_completions", "order_business_records", "cost_entries", "cost_allocations", "monthly_closings"} {
		var table, ddl string
		check(pool.QueryRow("SHOW CREATE TABLE `"+name+"`").Scan(&table, &ddl))
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
	exec("CREATE TEMPORARY TABLE users(id BIGINT,corp_id BIGINT,role_id BIGINT)")
	exec("CREATE TEMPORARY TABLE permissions(corp_id BIGINT,role_id BIGINT,resource_id BIGINT)")
	exec("CREATE TEMPORARY TABLE resources(id BIGINT,resource_key VARCHAR(100))")
	exec("INSERT INTO users VALUES(1,0,1)")
	exec("INSERT INTO resources VALUES(1,'orders'),(2,'monthly_output')")
	exec("INSERT INTO permissions VALUES(0,1,1),(0,1,2)")
	exec("INSERT INTO orders(id,corp_id,order_number,status) VALUES(1,0,'CORRECTION-1','completed'),(2,1,'OTHER','completed')")
	exec("INSERT INTO order_items(id,corp_id,order_id,part_name,quantity,production_quantity,unit_price,currency,status,delivered_quantity) VALUES(1,0,1,'PART',100,110,10,'CNY','completed',0),(2,1,2,'OTHER',100,110,10,'CNY','completed',0)")
	exec("INSERT INTO order_processes(id,corp_id,order_item_id,name,status) VALUES(1,0,1,'加工','completed'),(2,0,1,'送货','pending')")
	exec("INSERT INTO production_completions(id,corp_id,order_id,order_item_id,completion_no,business_date,qualified_quantity,billable_quantity,unit_price_snapshot,output_value,supply_mode_snapshot,currency,status,created_by,confirmed_at,created_at) VALUES(1,0,1,1,'TEST-1','2026-09-01',80,80,10,800,'self_purchased','CNY','confirmed',1,NOW(),NOW()),(2,0,1,1,'TEST-2','2026-09-10',30,20,20,400,'self_purchased','CNY','confirmed',1,NOW(),NOW())")
	exec("INSERT INTO cost_entries(id,corp_id,cost_type,amount,tax_amount,net_amount,currency,business_date,accounting_month,production_completion_id,status,created_by,created_at) VALUES(1,0,'material',50,0,50,'CNY','2026-09-01','2026-09',1,'confirmed',1,NOW())")
	exec("INSERT INTO cost_allocations(corp_id,cost_entry_id,production_completion_id,allocated_amount,allocation_basis,rule_version,created_by,created_at) VALUES(0,1,1,50,'original','manual-v1',1,NOW())")
	h := NewProductionRecordHandler(db)
	role := "user"
	counter := 0
	call := func(fn gin.HandlerFunc, payload interface{}, params gin.Params, url string, want int) map[string]interface{} {
		t.Helper()
		body, _ := json.Marshal(payload)
		rec := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(rec)
		c.Request = httptest.NewRequest("POST", url, strings.NewReader(string(body)))
		c.Request.Header.Set("Content-Type", "application/json")
		c.Params = params
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
	params := func(kind, id string) gin.Params {
		return gin.Params{{Key: "itemId", Value: "1"}, {Key: "kind", Value: kind}, {Key: "recordId", Value: id}}
	}
	edit := func(kind, id string, q int, date, reason string) recordCorrectionRequest {
		t.Helper()
		counter++
		var revision int64
		check(db.Model(&models.ProductionRecordChange{}).Where("corp_id=0 AND order_item_id=1").Select("COALESCE(MAX(id),0)").Scan(&revision).Error)
		var old struct {
			Quantity string
			Day      string
		}
		table, field := "production_completions", "qualified_quantity"
		if kind == "delivery" {
			table, field = "order_business_records", "quantity"
		}
		check(db.Table(table).Select("CAST("+field+" AS CHAR) AS quantity,DATE_FORMAT(business_date,'%Y-%m-%d') AS day").Where("id=?", id).Take(&old).Error)
		return recordCorrectionRequest{Quantity: q, Date: date, Reason: reason, RequestID: fmt.Sprintf("correct-%d", counter), ExpectedQuantity: old.Quantity, ExpectedDate: old.Day, ExpectedRevision: revision}
	}
	blank := edit("completion", "1", 70, "2026-09-01", "  ")
	call(h.Correct, blank, params("completion", "1"), "/", 400)
	req := edit("completion", "1", 70, "2026-10-01", "数量多录10件，按原检验记录更正")
	call(h.Correct, req, params("completion", "1"), "/", 200)
	call(h.Correct, req, params("completion", "1"), "/", 200) // network retry creates one audit
	var count int64
	check(db.Model(&models.ProductionRecordChange{}).Count(&count).Error)
	if count != 1 {
		t.Fatal("retry duplicated audit")
	}
	var other models.ProductionCompletion
	check(db.First(&other, 2).Error)
	if !sameDecimal(other.BillableQuantity, "30") || !sameDecimal(other.OutputValue, "600") {
		t.Fatal("spare split or snapshot price not recomputed")
	}
	var item models.OrderItem
	check(db.First(&item, 1).Error)
	if item.CompletionDate == nil || item.CompletionDate.Format("2006-01-02") != "2026-10-01" {
		t.Fatal("fulfillment date not refreshed")
	}
	var linked struct {
		Month  string
		Amount string
	}
	check(db.Raw("SELECT DATE_FORMAT(p.business_date,'%Y-%m') AS month,CAST(a.allocated_amount AS CHAR) AS amount FROM cost_allocations a JOIN production_completions p ON p.id=a.production_completion_id AND p.corp_id=a.corp_id WHERE a.corp_id=0 AND a.cost_entry_id=1").Scan(&linked).Error)
	if linked.Month != "2026-10" || !sameDecimal(linked.Amount, "50") {
		t.Fatal("correction lost the existing cost link or month attribution")
	}
	changedRequest := req
	changedRequest.Reason = "different"
	call(h.Correct, changedRequest, params("completion", "1"), "/", 400)
	exec("INSERT INTO monthly_closings(corp_id,accounting_month,status,rule_version) VALUES(0,'2026-09','closed','v1') ON DUPLICATE KEY UPDATE status='closed'")
	allowed := edit("completion", "1", 70, "2026-10-02", "同月日期填写有误")
	call(h.Correct, allowed, params("completion", "1"), "/", 200)
	blocked := edit("completion", "1", 80, "2026-10-02", "数量更正影响已结账备用分配")
	call(h.Correct, blocked, params("completion", "1"), "/", 400)
	exec("UPDATE monthly_closings SET status='open'")
	exec("INSERT INTO order_business_records(id,corp_id,order_id,order_item_id,kind,quantity,business_date,source_type,request_id,status,created_by,created_at) VALUES(1,0,1,1,'delivery',50,'2026-10-02','business','delivery-1','confirmed',1,NOW())")
	exec("UPDATE order_items SET delivered_quantity=50 WHERE id=1")
	delivery := edit("delivery", "1", 60, "2026-10-03", "按签收单更正")
	call(h.Correct, delivery, params("delivery", "1"), "/", 200)
	call(h.Correct, delivery, params("delivery", "1"), "/", 200)
	item = models.OrderItem{}
	check(db.First(&item, 1).Error)
	if item.DeliveredQty == nil || *item.DeliveredQty != 60 {
		t.Fatal("delivery aggregate not refreshed")
	}
	tooFew := edit("completion", "1", 20, "2026-10-02", "数量小于已交货")
	call(h.Correct, tooFew, params("completion", "1"), "/", 400)
	tooLate := edit("completion", "1", 70, "2026-10-04", "完工日期晚于已交货")
	call(h.Correct, tooLate, params("completion", "1"), "/", 400)
	future := edit("delivery", "1", 60, time.Now().AddDate(0, 0, 1).Format("2006-01-02"), "未来日期")
	call(h.Correct, future, params("delivery", "1"), "/", 400)
	oversized := edit("delivery", "1", 101, "2026-10-03", "超出订单")
	call(h.Correct, oversized, params("delivery", "1"), "/", 400)
	stale := edit("delivery", "1", 59, "2026-10-03", "旧版本")
	stale.ExpectedRevision = 0
	call(h.Correct, stale, params("delivery", "1"), "/", 400)
	foreign := params("delivery", "1")
	foreign[0].Value = "2"
	call(h.Correct, delivery, foreign, "/", 400)
	exec("DELETE FROM permissions")
	call(h.Correct, delivery, params("delivery", "1"), "/", 403)
	exec("INSERT INTO permissions VALUES(0,1,1),(0,1,2)")
	data := call(h.List, nil, gin.Params{{Key: "itemId", Value: "1"}}, "/?kind=completion", 200)["data"].(map[string]interface{})
	if !data["can_edit"].(bool) || len(data["changes"].([]interface{})) != 2 {
		t.Fatal("business user cannot see correction history")
	}
	for _, v := range data["changes"].([]interface{}) {
		audit := v.(map[string]interface{})
		if _, leak := audit["before_json"]; leak {
			t.Fatal("audit leaked financial snapshots")
		}
		if strings.TrimSpace(audit["reason"].(string)) == "" {
			t.Fatal("missing required reason")
		}
	}
	var final models.ProductionCompletion
	check(db.First(&final, 1).Error)
	if !sameDecimal(final.QualifiedQuantity, "70") || final.BusinessDate.Format("2006-01-02") != "2026-10-02" {
		t.Fatal("rejected correction was not rolled back")
	}
	// Tracked parts cannot consume terminal output from a future date.
	exec("INSERT INTO process_quantity_totals(corp_id,order_item_id,order_process_id,started_quantity,completed_quantity,scrap_quantity,activated_at) VALUES(0,1,1,110,110,0,'2026-09-01')")
	exec("INSERT INTO process_quantity_events(corp_id,order_item_id,order_process_id,kind,quantity,business_date,source_type,request_id,created_by,created_at) VALUES(0,1,1,'complete',110,'2026-09-05','internal','last-output',1,NOW())")
	tracked := edit("completion", "1", 70, "2026-09-02", "早于末道实际完工")
	call(h.Correct, tracked, params("completion", "1"), "/", 400)
	// A valid edit remains possible on tracked parts; batch caps and batch dates still apply.
	exec("INSERT INTO order_delivery_batches(id,corp_id,order_id,order_item_id,batch_no,quantity,production_quantity,due_date,created_by,created_at) VALUES(1,0,1,1,'首批',70,75,'2026-10-30',1,NOW()),(2,0,1,1,'尾批',30,35,'2026-11-30',1,NOW())")
	exec("UPDATE production_completions SET delivery_batch_id=id WHERE corp_id=0")
	exec("UPDATE order_business_records SET delivery_batch_id=1,quantity=20 WHERE id=1")
	exec("UPDATE order_items SET delivered_quantity=20 WHERE id=1")
	batchDate := edit("completion", "1", 70, "2026-10-04", "本批晚于交货，其他批次不能替代")
	call(h.Correct, batchDate, params("completion", "1"), "/", 400)
	batchCap := edit("completion", "1", 76, "2026-10-02", "超过首批投产上限")
	call(h.Correct, batchCap, params("completion", "1"), "/", 400)
	batchOK := edit("completion", "1", 71, "2026-10-02", "按分批检验单多确认1件")
	call(h.Correct, batchOK, params("completion", "1"), "/", 200)
	exec("UPDATE order_business_records SET source_type='legacy',business_date=NULL WHERE id=1")
	legacy := edit("delivery", "1", 21, "2026-10-03", "按原交货单补正历史日期和数量")
	call(h.Correct, legacy, params("delivery", "1"), "/", 200)
	t.Log("correction audit, required reasons, ordinary permissions, stale/retry guards, quantity/date conservation, month locks, snapshot pricing and rollback verified in temporary tables")
}
