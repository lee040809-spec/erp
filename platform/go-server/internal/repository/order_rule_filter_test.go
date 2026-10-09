package repository

import (
	driver "github.com/go-sql-driver/mysql"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"os"
	"reflect"
	"sort"
	"strings"
	"testing"
)

func TestRuleScopeTemporaryTables(t *testing.T) {
	if os.Getenv("RUN_FINANCE_INTEGRATION") != "1" {
		t.Skip("requires test DB")
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
	exec("CREATE TEMPORARY TABLE advent_rules(id BIGINT PRIMARY KEY,corp_id BIGINT,name VARCHAR(100),description VARCHAR(100),formula VARCHAR(1000),target_status VARCHAR(20),scopeType VARCHAR(20),ruleType VARCHAR(20),created_at DATETIME)")
	exec("CREATE TEMPORARY TABLE orders(id BIGINT PRIMARY KEY,corp_id BIGINT,order_number VARCHAR(100),customer_name VARCHAR(100),customer_short_name VARCHAR(100),start_date DATE,created_at DATETIME,status VARCHAR(30))")
	exec("CREATE TEMPORARY TABLE order_items(id BIGINT PRIMARY KEY,order_id BIGINT,corp_id BIGINT,due_date DATE,status VARCHAR(30))")
	for i := 1; i <= 4; i++ {
		corp := 0
		if i == 4 {
			corp = 1
		}
		exec("INSERT INTO orders VALUES(?,?,?, ?,?,DATE_SUB(CURDATE(),INTERVAL 5 DAY),NOW(),'pending')", i, corp, "ORDER-"+strings.Repeat("X", i), "CUSTOMER-"+strings.Repeat("X", i), "SHORT-"+strings.Repeat("X", i))
		status := "pending"
		if i == 3 {
			status = "processing"
		}
		exec("INSERT INTO order_items VALUES(?,?,?,DATE_ADD(CURDATE(),INTERVAL 5 DAY),?)", i, i, corp, status)
	}
	expression := "(({当天}-{订单日期})/({订单交期}-{订单日期}))"
	repo := NewOrderRepository(db)
	check := func(want []int64) {
		t.Helper()
		ids, e := repo.getOrderIDsByRuleType(0, "warning")
		if e != nil {
			t.Fatal(e)
		}
		sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
		if !reflect.DeepEqual(ids, want) {
			t.Fatalf("got %v want %v", ids, want)
		}
	}
	exec("INSERT INTO advent_rules VALUES(1,0,'', '',?, 'pending','general','warning',NOW())", ".75>="+expression+">=.25")
	check([]int64{1, 2})
	exec("UPDATE advent_rules SET scopeType='specific',name='ORDER-X'")
	check([]int64{})
	exec("UPDATE advent_rules SET name='CUSTOMER-XX'")
	check([]int64{})
	exec("UPDATE advent_rules SET scopeType='general',target_status='processing'")
	check([]int64{3})
	exec("UPDATE advent_rules SET ruleType='imminent'")
	check([]int64{})
	exec("UPDATE advent_rules SET ruleType='warning',formula=?", "0 OR 1=1<="+expression+"<=1")
	check([]int64{})
}
