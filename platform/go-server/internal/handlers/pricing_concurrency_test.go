package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	driver "github.com/go-sql-driver/mysql"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"machining-erp/internal/accounting"
	"machining-erp/internal/models"
	"machining-erp/internal/repository"
)

// This suite requires a disposable schema with real shared tables. Connection-
// local TEMPORARY tables cannot test transactions on two independent connections.
func TestPricingConcurrencyIsolatedDatabase(t *testing.T) {
	if os.Getenv("RUN_PRICING_CONCURRENCY") != "1" {
		t.Skip("explicit isolated concurrency environment required")
	}
	if !strings.HasPrefix(os.Getenv("MYSQL_DATABASE"), "erp_audit_test_") {
		t.Fatal("refusing to write outside a disposable audit database")
	}
	open := func() *gorm.DB {
		cfg := driver.NewConfig()
		cfg.User, cfg.Passwd, cfg.DBName = os.Getenv("MYSQL_USER"), os.Getenv("MYSQL_PASSWORD"), os.Getenv("MYSQL_DATABASE")
		cfg.Net, cfg.Addr, cfg.ParseTime = "tcp", os.Getenv("MYSQL_HOST")+":3306", true
		cfg.Timeout, cfg.ReadTimeout, cfg.WriteTimeout = 10*time.Second, 20*time.Second, 20*time.Second
		cfg.Params = map[string]string{"transaction_isolation": "'REPEATABLE-READ'"}
		db, err := gorm.Open(mysql.Open(cfg.FormatDSN()), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
		if err != nil {
			t.Fatal("isolated database unavailable:", err)
		}
		pool, _ := db.DB()
		pool.SetMaxOpenConns(1)
		pool.SetMaxIdleConns(1)
		t.Cleanup(func() { pool.Close() })
		return db
	}
	db, a, b := open(), open(), open()
	check := func(t *testing.T, err error) {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
	}
	var isolation string
	check(t, a.Raw("SELECT @@transaction_isolation").Scan(&isolation).Error)
	if isolation != "REPEATABLE-READ" {
		t.Fatal("test must exercise REPEATABLE READ", isolation)
	}
	var next int64 = time.Now().UnixMilli()%100000000 + 100000
	fixture := func(t *testing.T, price string, pending bool, qty int) int64 {
		next += 10
		id := next
		check(t, db.Exec("INSERT INTO orders(id,corp_id,customer_id,order_number,status) VALUES(?,0,1,?,'processing')", id, fmt.Sprint("AUDIT-", id)).Error)
		check(t, db.Exec("INSERT INTO order_items(id,corp_id,order_id,part_name,part_number,quantity,production_quantity,unit_price,currency,supply_mode,status,delivered_quantity) VALUES(?,0,?,'AUDIT','A',?,?,?,'CNY','self_purchased','completed',0)", id, id, qty, qty, price).Error)
		check(t, db.Exec("INSERT INTO order_processes(id,corp_id,order_item_id,name,status) VALUES(?,0,?,'加工','completed'),(?,0,?,'送货','pending')", id, id, id+1, id).Error)
		if pending {
			check(t, db.Exec("INSERT INTO production_completions(corp_id,order_id,order_item_id,completion_no,business_date,qualified_quantity,billable_quantity,unit_price_snapshot,output_value,supply_mode_snapshot,currency,tax_inclusive,status,pricing_status,created_by,confirmed_at,created_at) VALUES(0,?,?,?,'2026-01-02',10,10,0,0,'self_purchased','CNY',0,'confirmed','pending',1,NOW(),NOW())", id, id, fmt.Sprint("PENDING-", id)).Error)
		}
		return id
	}
	type reply struct {
		code int
		body string
	}
	requestUser := next + 7
	call := func(conn *gorm.DB, id int64, body string, fn func(*ProductionFinanceHandler) gin.HandlerFunc, corp int64, admin bool) reply {
		ctx, cancel := context.WithTimeout(context.Background(), 18*time.Second)
		defer cancel()
		rec := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(rec)
		c.Request = httptest.NewRequest("POST", "/", strings.NewReader(body)).WithContext(ctx)
		c.Request.Header.Set("Content-Type", "application/json")
		c.Params = gin.Params{{Key: "itemId", Value: fmt.Sprint(id)}, {Key: "statementId", Value: fmt.Sprint(id)}}
		c.Set("corpID", corp)
		c.Set("userID", requestUser)
		if admin {
			c.Set("roleType", "admin")
		} else {
			c.Set("roleType", "business")
		}
		fn(NewProductionFinanceHandler(conn.WithContext(ctx)))(c)
		return reply{rec.Code, rec.Body.String()}
	}
	priceFn := func(h *ProductionFinanceHandler) gin.HandlerFunc { return h.PriceCompletion }
	completionFn := func(h *ProductionFinanceHandler) gin.HandlerFunc { return h.CreateCompletion }
	concurrent := func(t *testing.T, first, second func() reply) []reply {
		start := make(chan struct{})
		out := make(chan reply, 2)
		for _, fn := range []func() reply{first, second} {
			go func(work func() reply) { <-start; out <- work() }(fn)
		}
		close(start)
		result := []reply{}
		for i := 0; i < 2; i++ {
			select {
			case r := <-out:
				result = append(result, r)
			case <-time.After(25 * time.Second):
				t.Fatal("concurrent transactions did not complete")
			}
		}
		return result
	}

	t.Run("stale_snapshot_cannot_overwrite_price", func(t *testing.T) {
		id := fixture(t, "0", false, 10)
		blocker := db.Begin()
		check(t, blocker.Error)
		defer blocker.Rollback()
		var locked int64
		check(t, blocker.Raw("SELECT id FROM orders WHERE id=? FOR UPDATE", id).Scan(&locked).Error)
		arrived := make(chan struct{}, 2)
		for _, conn := range []*gorm.DB{a, b} {
			var once sync.Once
			check(t, conn.Callback().Query().After("gorm:query").Register("audit:read_barrier", func(tx *gorm.DB) {
				_, locking := tx.Statement.Clauses["FOR"]
				if tx.Statement.Table == "order_items" && !locking && tx.RowsAffected > 0 {
					once.Do(func() { arrived <- struct{}{} })
				}
			}))
		}
		defer a.Callback().Query().Remove("audit:read_barrier")
		defer b.Callback().Query().Remove("audit:read_barrier")
		out := make(chan reply, 2)
		go func() { out <- call(a, id, `{"unit_price":"2.50"}`, priceFn, 0, true) }()
		go func() { out <- call(b, id, `{"unit_price":"3.50"}`, priceFn, 0, true) }()
		for i := 0; i < 2; i++ {
			select {
			case <-arrived:
			case <-time.After(10 * time.Second):
				t.Fatal("both ordinary reads must precede releasing order lock")
			}
		}
		check(t, blocker.Commit().Error)
		x, y := <-out, <-out
		if !(x.code == 200 && y.code == 400 || x.code == 400 && y.code == 200) {
			t.Fatalf("expected one accepted price and one conflict: %+v %+v", x, y)
		}
	})
	t.Run("pending_snapshot_matches_winner_and_replays_once", func(t *testing.T) {
		id := fixture(t, "0", true, 100)
		out := concurrent(t, func() reply { return call(a, id, `{"unit_price":"2.50"}`, priceFn, 0, true) }, func() reply { return call(b, id, `{"unit_price":"3.50"}`, priceFn, 0, true) })
		if !(out[0].code == 200 && out[1].code == 400 || out[0].code == 400 && out[1].code == 200) {
			t.Fatalf("%+v", out)
		}
		var current, snapshot string
		check(t, db.Raw("SELECT CAST(unit_price AS CHAR) FROM order_items WHERE id=?", id).Scan(&current).Error)
		check(t, db.Raw("SELECT CAST(unit_price_snapshot AS CHAR) FROM production_completions WHERE order_item_id=?", id).Scan(&snapshot).Error)
		if !sameDecimal(current, snapshot) {
			t.Fatal("snapshot differs from winning order price", current, snapshot)
		}
		replays := concurrent(t, func() reply { return call(a, id, fmt.Sprintf(`{"unit_price":%q}`, current), priceFn, 0, true) }, func() reply { return call(b, id, fmt.Sprintf(`{"unit_price":%q}`, current), priceFn, 0, true) })
		for _, r := range replays {
			if r.code != 200 {
				t.Fatalf("same-price replay failed: %+v", r)
			}
		}
		var n int64
		check(t, db.Table("completion_price_changes").Where("production_completion_id IN (SELECT id FROM production_completions WHERE order_item_id=?)", id).Count(&n).Error)
		if n != 1 {
			t.Fatal("duplicate pricing audit", n)
		}
	})
	t.Run("completion_and_both_pricing_entries_share_lock_order", func(t *testing.T) {
		for _, viaOrder := range []bool{false, true} {
			id := fixture(t, "0", true, 100)
			var order models.Order
			var items []models.OrderItem
			check(t, db.Where("id=?", id).Take(&order).Error)
			check(t, db.Where("id=?", id).Preload("Processes").Find(&items).Error)
			items[0].UnitPrice = 2.5
			out := concurrent(t, func() reply {
				if !viaOrder {
					return call(a, id, `{"unit_price":"2.50"}`, priceFn, 0, true)
				}
				err := repository.NewOrderRepository(a).Update(&order, items, repository.OrderUpdateOptions{ActorID: 1, UnitPriceProvided: map[int64]bool{id: true}})
				if err != nil {
					return reply{400, err.Error()}
				}
				return reply{200, "order saved"}
			}, func() reply {
				return call(b, id, fmt.Sprintf(`{"qualified_quantity":20,"business_date":"2026-01-02","request_id":"complete-%d"}`, id), completionFn, 0, true)
			})
			for _, r := range out {
				if r.code != 200 {
					t.Fatalf("viaOrder=%v: %+v", viaOrder, r)
				}
			}
			var output string
			check(t, db.Raw("SELECT CAST(SUM(output_value) AS CHAR) FROM production_completions WHERE order_item_id=?", id).Scan(&output).Error)
			if !sameDecimal(output, "75") {
				t.Fatal("concurrent output conservation", output)
			}
		}
	})
	t.Run("null_prices_and_stale_history_candidates", func(t *testing.T) {
		id := fixture(t, "0", false, 10)
		check(t, db.Exec("UPDATE order_items SET unit_price=NULL WHERE id=?", id).Error)
		for _, expected := range []string{"null", `""`, `"0"`} {
			r := call(a, id, fmt.Sprintf(`{"order_item_id":%d,"qualified_quantity":1,"business_date":"2026-01-02","expected_price":%s,"reviewed":true,"request_id":"null-%d-%d"}`, id, expected, id, len(expected)), func(h *ProductionFinanceHandler) gin.HandlerFunc { return h.ImportHistory }, 0, true)
			if r.code != 200 || !strings.Contains(r.body, `"pricing_status":"pending"`) || !strings.Contains(r.body, "待补价") {
				t.Fatalf("NULL compatibility: %+v", r)
			}
		}
		check(t, db.Exec("UPDATE order_items SET unit_price=3 WHERE id=?", id).Error)
		r := call(a, id, fmt.Sprintf(`{"order_item_id":%d,"qualified_quantity":1,"business_date":"2026-01-02","expected_price":null,"reviewed":true}`, id), func(h *ProductionFinanceHandler) gin.HandlerFunc { return h.ImportHistory }, 0, true)
		if r.code != 400 {
			t.Fatal("stale missing-price candidate accepted", r)
		}
	})
	t.Run("month_scope_change_restarts_without_late_month_lock", func(t *testing.T) {
		id := fixture(t, "0", false, 10)
		attempts := 0
		check(t, accounting.WithPricingMonths(a, 0, accounting.PricingScope{ItemIDs: []int64{id}}, func(tx *gorm.DB, months accounting.PricingMonths) error {
			attempts++
			if attempts == 1 {
				check(t, db.Exec("INSERT INTO production_completions(corp_id,order_id,order_item_id,completion_no,business_date,qualified_quantity,billable_quantity,unit_price_snapshot,output_value,supply_mode_snapshot,currency,status,pricing_status,created_by,confirmed_at,created_at) VALUES(0,?,?,?,'2026-02-02',1,1,0,0,'self_purchased','CNY','confirmed','pending',1,NOW(),NOW())", id, id, fmt.Sprint("scope-", id)).Error)
			}
			if err := tx.Exec("UPDATE order_items SET unit_price=2 WHERE id=?", id).Error; err != nil {
				return err
			}
			return accounting.ResolvePending(tx, 0, id, 1, months)
		}))
		if attempts != 2 {
			t.Fatal("scope change did not restart transaction", attempts)
		}
	})
	t.Run("permissions_tenant_and_decimal_boundaries", func(t *testing.T) {
		id := fixture(t, "0", false, 2)
		r := call(a, id, `{"unit_price":"2"}`, priceFn, 0, false)
		if r.code != 403 {
			t.Fatal("unprivileged pricing allowed", r)
		}
		r = call(a, id, `{"unit_price":"2"}`, priceFn, 77, true)
		if r.code != 400 {
			t.Fatal("cross-company pricing allowed", r)
		}
		r = call(a, id, `{"unit_price":"99999999.99"}`, priceFn, 0, true)
		if r.code != 400 {
			t.Fatal("DECIMAL total overflow accepted", r)
		}
		var p string
		check(t, db.Raw("SELECT CAST(unit_price AS CHAR) FROM order_items WHERE id=?", id).Scan(&p).Error)
		if positiveDecimal(p) {
			t.Fatal("overflow partially saved price")
		}
		one := fixture(t, "0", false, 1)
		r = call(a, one, `{"unit_price":"99999999.99"}`, priceFn, 0, true)
		if r.code != 200 {
			t.Fatal("valid DECIMAL boundary rejected", r)
		}
		// Real isolated role/resource joins, rather than an injected permission map.
		check(t, db.Exec("INSERT INTO roles(id,corp_id,name,account_type) VALUES(?,0,'审核测试角色','user')", requestUser).Error)
		check(t, db.Exec("INSERT INTO users(id,corp_id,username,password,role_id,role_type) VALUES(?,0,?,'not-a-login',?,'user')", requestUser, fmt.Sprint("audit-", requestUser), requestUser).Error)
		grant := func(key string) {
			check(t, db.Exec("INSERT INTO resources(resource_type,resource_key,name) VALUES('page',?,'审核测试权限') ON DUPLICATE KEY UPDATE resource_key=VALUES(resource_key)", key).Error)
			var resource int64
			check(t, db.Raw("SELECT id FROM resources WHERE resource_key=?", key).Scan(&resource).Error)
			check(t, db.Exec("DELETE FROM permissions WHERE corp_id=0 AND role_id=?", requestUser).Error)
			check(t, db.Exec("INSERT INTO permissions(corp_id,role_id,resource_id) VALUES(0,?,?)", requestUser, resource).Error)
		}
		grant("orders")
		r = call(a, id, `{"unit_price":"2"}`, priceFn, 0, false)
		if r.code != 200 {
			t.Fatal("authorized ordinary user cannot fill price", r)
		}
		grant("monthly_output")
		history := fixture(t, "0", false, 10)
		r = call(a, history, `{"unit_price":"2"}`, priceFn, 0, false)
		if r.code != 403 {
			t.Fatal("output-only role can overwrite order price", r)
		}
		body := fmt.Sprintf(`{"order_item_id":%d,"qualified_quantity":1,"business_date":"2026-01-02","expected_price":null,"reviewed":true,"request_id":"ordinary-%d"}`, history, history)
		r = call(a, history, body, func(h *ProductionFinanceHandler) gin.HandlerFunc { return h.ImportHistory }, 0, false)
		if r.code != 200 || !strings.Contains(r.body, `"pricing_status":"pending"`) {
			t.Fatal("authorized ordinary user cannot record pending history", r)
		}
		r = call(a, history, body, func(h *ProductionFinanceHandler) gin.HandlerFunc { return h.ImportHistory }, 77, false)
		if r.code != 403 {
			t.Fatal("other-company role inherited permissions", r)
		}
		grant("reconciliation")
		check(t, db.Exec("INSERT INTO order_business_records(id,corp_id,order_id,order_item_id,kind,quantity,business_date,status,request_id,source_type,created_by,created_at) VALUES(?,0,?,?,'delivery',1,'2026-01-02','confirmed',?,'audit-test',?,NOW())", id, id, id, fmt.Sprint("delivery-", id), requestUser).Error)
		validBody := fmt.Sprintf(`{"customer_id":1,"business_date":"2026-01-02","delivery_ids":[%d],"request_id":"rec-%d"}`, id, id)
		r = call(a, id, validBody, func(h *ProductionFinanceHandler) gin.HandlerFunc { return h.CreateDeliveryReconciliation }, 0, false)
		if r.code != 200 {
			t.Fatal("authorized ordinary user cannot reconcile", r)
		}
		var saved struct{ Data struct{ ID int64 } }
		check(t, json.Unmarshal([]byte(r.body), &saved))
		detail := call(a, saved.Data.ID, `{}`, func(h *ProductionFinanceHandler) gin.HandlerFunc { return h.DeliveryReconciliationDetail }, 0, false)
		if detail.code != 200 {
			t.Fatal("ordinary statement detail failed", detail)
		}
		foreignDetail := call(a, saved.Data.ID, `{}`, func(h *ProductionFinanceHandler) gin.HandlerFunc { return h.DeliveryReconciliationDetail }, 77, true)
		if foreignDetail.code != 400 {
			t.Fatal("cross-company statement detail leaked", foreignDetail)
		}
		badReverse := call(a, saved.Data.ID, `{"reason":""}`, func(h *ProductionFinanceHandler) gin.HandlerFunc { return h.ReverseDeliveryReconciliation }, 0, false)
		if badReverse.code != 400 {
			t.Fatal("empty reversal reason accepted", badReverse)
		}
		goodReverse := call(a, saved.Data.ID, `{"reason":"审核测试更正"}`, func(h *ProductionFinanceHandler) gin.HandlerFunc { return h.ReverseDeliveryReconciliation }, 0, false)
		if goodReverse.code != 200 {
			t.Fatal("ordinary reversal failed", goodReverse)
		}
		denied := call(a, id, `{}`, func(h *ProductionFinanceHandler) gin.HandlerFunc { return h.CreateDeliveryReconciliation }, 77, false)
		if denied.code != 403 {
			t.Fatal("cross-company reconciliation permission leak", denied)
		}
		// The authorized role reaches business validation; a foreign-company
		// delivery ID must still be rejected rather than becoming a statement.
		foreign := fixture(t, "2", false, 10)
		check(t, db.Exec("UPDATE orders SET corp_id=77 WHERE id=?", foreign).Error)
		check(t, db.Exec("UPDATE order_items SET corp_id=77 WHERE id=?", foreign).Error)
		check(t, db.Exec("INSERT INTO order_business_records(id,corp_id,order_id,order_item_id,kind,quantity,business_date,status,request_id,source_type,created_by,created_at) VALUES(?,77,?,?,'delivery',1,'2026-01-02','confirmed',?,'audit-test',?,NOW())", foreign, foreign, foreign, fmt.Sprint("foreign-delivery-", foreign), requestUser).Error)
		foreignBody := fmt.Sprintf(`{"customer_id":1,"business_date":"2026-01-02","delivery_ids":[%d],"request_id":"foreign-%d"}`, foreign, foreign)
		r = call(a, id, foreignBody, func(h *ProductionFinanceHandler) gin.HandlerFunc { return h.CreateDeliveryReconciliation }, 0, false)
		if r.code != 400 {
			t.Fatal("invalid delivery created a reconciliation", r)
		}
		var data map[string]interface{}
		check(t, json.Unmarshal([]byte(r.body), &data))
	})
}
