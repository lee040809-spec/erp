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
	"net/http/httptest"
	"os"
	"regexp"
	"strings"
	"testing"
)

func TestReferenceCompanyIsolationTemporaryTables(t *testing.T) {
	if os.Getenv("RUN_FINANCE_INTEGRATION") != "1" {
		t.Skip("requires test DB")
	}
	gin.SetMode(gin.TestMode)
	cfg := driver.NewConfig()
	cfg.User = os.Getenv("MYSQL_USER")
	cfg.Passwd = os.Getenv("MYSQL_PASSWORD")
	cfg.Net = "tcp"
	cfg.Addr = os.Getenv("MYSQL_HOST") + ":3306"
	cfg.DBName = os.Getenv("MYSQL_DATABASE")
	cfg.ParseTime = true
	db, e := gorm.Open(mysql.Open(cfg.FormatDSN()), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if e != nil {
		t.Fatal("connection failed")
	}
	pool, _ := db.DB()
	pool.SetMaxOpenConns(1)
	pool.SetMaxIdleConns(1)
	defer pool.Close()
	exec := func(q string, args ...interface{}) {
		t.Helper()
		if e := db.Exec(q, args...).Error; e != nil {
			t.Fatal(e)
		}
	}
	// Read-only checks run before temporary fixtures shadow persistent tables.
	for _, q := range []string{"SELECT COUNT(*) FROM contacts c LEFT JOIN customers p ON p.id=c.customer_id WHERE p.id IS NULL OR p.corp_id<>c.corp_id", "SELECT COUNT(*) FROM remnants r LEFT JOIN materials m ON m.id=r.material_id WHERE m.id IS NULL OR m.corp_id<>r.corp_id"} {
		var n int64
		if e := db.Raw(q).Scan(&n).Error; e != nil {
			t.Fatal(e)
		}
		t.Logf("existing reference mismatch count: %d", n)
		if n != 0 {
			t.Fatal("real reference ownership needs reconciliation")
		}
	}
	for _, q := range []string{
		"SELECT COUNT(*) FROM order_delivery_batches b LEFT JOIN order_items i ON i.id=b.order_item_id WHERE i.id IS NULL OR b.order_id<>i.order_id OR b.corp_id<>i.corp_id",
		"SELECT COUNT(*) FROM outsourcing_dispatches d LEFT JOIN order_items i ON i.id=d.order_item_id WHERE i.id IS NULL OR d.order_id<>i.order_id OR d.corp_id<>i.corp_id",
		"SELECT COUNT(*) FROM outsourcing_settlements s LEFT JOIN outsourcing_dispatches d ON d.id=s.dispatch_id LEFT JOIN cost_entries e ON e.id=s.cost_entry_id WHERE d.id IS NULL OR s.corp_id<>d.corp_id OR (s.cost_entry_id IS NOT NULL AND (e.id IS NULL OR e.corp_id<>s.corp_id))",
		"SELECT COUNT(*) FROM cost_allocations a LEFT JOIN cost_entries e ON e.id=a.cost_entry_id LEFT JOIN production_completions p ON p.id=a.production_completion_id WHERE e.id IS NULL OR p.id IS NULL OR a.corp_id<>e.corp_id OR a.corp_id<>p.corp_id",
	} {
		var n int64
		if e := db.Raw(q).Scan(&n).Error; e != nil {
			t.Fatal(e)
		}
		t.Logf("existing batch/outsource/cost reference mismatch count: %d", n)
		if n != 0 {
			t.Fatal("real business references need reconciliation")
		}
	}
	var summary struct {
		Count int64
		Total string
	}
	if e := db.Raw("SELECT COUNT(*) count,CAST(COALESCE(SUM(output_value),0) AS CHAR) total FROM production_completions WHERE status='confirmed'").Scan(&summary).Error; e != nil {
		t.Fatal(e)
	}
	t.Logf("real confirmed output batches=%d value=%s", summary.Count, summary.Total)
	summary.Count = 0
	summary.Total = ""
	if e := db.Raw("SELECT COUNT(*) count,CAST(COALESCE(SUM(amount),0) AS CHAR) total FROM cost_entries").Scan(&summary).Error; e != nil {
		t.Fatal(e)
	}
	t.Logf("real cost rows=%d amount=%s; completeness is not implied by zero cost", summary.Count, summary.Total)
	for _, name := range []string{"customers", "contacts", "materials", "remnants", "advent_rules"} {
		var table, ddl string
		if e := pool.QueryRow("SHOW CREATE TABLE "+name).Scan(&table, &ddl); e != nil {
			t.Fatal(e)
		}
		lines := []string{}
		for _, line := range strings.Split(ddl, "\n") {
			if !strings.Contains(line, "FOREIGN KEY") {
				lines = append(lines, line)
			}
		}
		ddl = strings.Join(lines, "\n")
		ddl = regexp.MustCompile(",\\s*\\)").ReplaceAllString(ddl, "\n)")
		ddl = strings.Replace(ddl, "CREATE TABLE", "CREATE TEMPORARY TABLE", 1)
		ddl = regexp.MustCompile("AUTO_INCREMENT=[0-9]+").ReplaceAllString(ddl, "AUTO_INCREMENT=1")
		exec(ddl)
	}
	exec("CREATE TEMPORARY TABLE users(id BIGINT,corp_id BIGINT,role_id BIGINT)")
	exec("CREATE TEMPORARY TABLE permissions(role_id BIGINT,corp_id BIGINT,resource_id BIGINT)")
	exec("CREATE TEMPORARY TABLE resources(id BIGINT,resource_key VARCHAR(100))")
	exec("INSERT INTO users VALUES(1,0,1)")
	exec("INSERT INTO resources VALUES(1,'orders'),(2,'customers'),(3,'inventory'),(4,'advent_rules')")
	exec("INSERT INTO customers(id,corp_id,name,short_name) VALUES(1,0,'OWN','O'),(2,1,'FOREIGN','F')")
	exec("INSERT INTO contacts(id,corp_id,customer_id,name,contact) VALUES(11,0,1,'OWN','LOCAL'),(22,1,2,'FOREIGN','OTHER')")
	exec("INSERT INTO materials(id,corp_id,name,quantity,unit) VALUES(1,0,'OWN',10,'kg'),(2,1,'FOREIGN',10,'kg')")
	exec("INSERT INTO remnants(id,corp_id,material_id,dimensions) VALUES(1,0,1,'OWN'),(2,1,2,'FOREIGN')")
	formula := "0<=(({当天}-{订单日期})/({订单交期}-{订单日期}))<=1"
	exec("INSERT INTO advent_rules(id,corp_id,name,formula,scopeType,ruleType,target_status) VALUES(1,0,'OWN',?,'general','warning','pending'),(2,1,'FOREIGN',?,'general','warning','pending')", formula, formula)
	customer := NewCustomerHandler(repository.NewCustomerRepository(db), db)
	material := NewMaterialHandler(repository.NewMaterialRepository(db), db)
	remnant := NewRemnantHandler(repository.NewRemnantRepository(db), db)
	rule := NewAdventRuleHandler(repository.NewAdventRuleRepository(db), db)
	call := func(fn gin.HandlerFunc, method, id, body, role string, want int) *httptest.ResponseRecorder {
		t.Helper()
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Set("corpID", int64(0))
		c.Set("userID", int64(1))
		c.Set("roleType", role)
		c.Params = gin.Params{{Key: "id", Value: id}}
		c.Request = httptest.NewRequest(method, "/", strings.NewReader(body))
		c.Request.Header.Set("Content-Type", "application/json")
		fn(c)
		if w.Code != want {
			t.Fatalf("%s id=%s got %d want %d: %s", method, id, w.Code, want, w.Body.String())
		}
		return w
	}
	list := func(fn gin.HandlerFunc) {
		t.Helper()
		w := call(fn, "GET", "", "", "admin", 200)
		var rows []map[string]interface{}
		if e := json.Unmarshal(w.Body.Bytes(), &rows); e != nil {
			t.Fatal(e)
		}
		if len(rows) != 1 || rows[0]["corp_id"] != float64(0) {
			t.Fatal("list leaked company")
		}
	}
	list(customer.List)
	list(material.List)
	list(remnant.List)
	list(rule.List)
	call(material.List, "GET", "", "", "user", 403)
	call(customer.Create, "POST", "", `{"name":"NEW","short_name":"N"}`, "user", 403)
	exec("INSERT INTO permissions VALUES(1,0,1)")
	call(customer.List, "GET", "", "", "user", 200)
	call(customer.Create, "POST", "", `{"name":"NEW","short_name":"N"}`, "user", 403)
	body := `{ "name":"EDIT", "short_name":"E", "contacts":[] }`
	call(customer.Update, "PUT", "2", body, "admin", 404)
	call(customer.Delete, "DELETE", "2", "", "admin", 404)
	malicious := `{ "name":"EDIT", "short_name":"E", "contacts":[{"id":22,"name":"ATTACK","contact":"X"}] }`
	call(customer.Update, "PUT", "1", malicious, "admin", 404)
	var saved models.Customer
	db.First(&saved, 1)
	if saved.Name != "OWN" {
		t.Fatal("failed contact update did not roll back")
	}
	own := `{ "name":"EDIT", "short_name":"E", "contacts":[{"id":11,"name":"KEPT","contact":"OK","corp_id":1}] }`
	call(customer.Update, "PUT", "1", own, "admin", 200)
	var ct models.Contact
	db.First(&ct, 11)
	if ct.CustomerID != 1 || ct.CorpID != 0 || ct.Name != "KEPT" {
		t.Fatal("owned contact identity changed")
	}
	call(customer.Create, "POST", "", `{"name":"FOREIGN","short_name":"F","contacts":[{"id":22,"corp_id":1,"customer_id":2,"name":"NEW","contact":"LOCAL"}]}`, "admin", 200)
	ct = models.Contact{}
	if e := db.First(&ct, 22).Error; e != nil {
		t.Fatal(e)
	}
	if ct.CorpID != 1 || ct.Name != "FOREIGN" {
		t.Fatal("foreign contact changed")
	}
	call(material.Create, "POST", "", `{"id":2,"corp_id":1,"name":"NEW","unit":"kg","quantity":10}`, "admin", 200)
	call(material.Create, "POST", "", `{"name":"BAD","unit":"kg","quantity":-1}`, "admin", 400)
	call(remnant.Create, "POST", "", `{"material_id":2,"dimensions":"BAD"}`, "admin", 404)
	call(remnant.Create, "POST", "", `{"id":2,"corp_id":1,"material_id":1,"dimensions":"NEW"}`, "admin", 200)
	ruleBody := fmt.Sprintf(`{"name":"EDIT","formula":%q,"scopeType":"general","ruleType":"warning","target_status":"pending"}`, formula)
	call(rule.Update, "PUT", "2", ruleBody, "admin", 404)
	call(rule.Delete, "DELETE", "2", "", "admin", 404)
	call(rule.Update, "PUT", "1", ruleBody, "admin", 200)
	call(rule.Update, "PUT", "1", ruleBody, "admin", 200)
	call(rule.Create, "POST", "", `{"formula":"invalid"}`, "admin", 400)
	call(rule.Create, "POST", "", fmt.Sprintf(`{"id":2,"corp_id":1,"formula":%q,"name":"NEW","scopeType":"general","ruleType":"warning"}`, formula), "admin", 200)
	call(customer.Delete, "DELETE", "invalid", "", "admin", 400)
	var count int64
	db.Model(&models.Material{}).Where("id=2 AND corp_id=1 AND name='FOREIGN'").Count(&count)
	if count != 1 {
		t.Fatal("foreign material modified")
	}
	db.Model(&models.Remnant{}).Where("id=2 AND corp_id=1 AND dimensions='FOREIGN'").Count(&count)
	if count != 1 {
		t.Fatal("foreign remnant modified")
	}
	db.Model(&models.AdventRule{}).Where("id=2 AND corp_id=1 AND name='FOREIGN'").Count(&count)
	if count != 1 {
		t.Fatal("foreign rule modified")
	}
}
