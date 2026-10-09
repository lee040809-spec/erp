package repository

import (
	driver "github.com/go-sql-driver/mysql"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"os"
	"testing"
)

func TestDeliverySummariesTemporaryTables(t *testing.T) {
	if os.Getenv("RUN_FINANCE_INTEGRATION") != "1" || os.Getenv("MYSQL_DATABASE") != "platform_test" {
		t.Skip("explicit test database required")
	}
	cfg := driver.NewConfig()
	cfg.User, cfg.Passwd = os.Getenv("MYSQL_USER"), os.Getenv("MYSQL_PASSWORD")
	cfg.Net, cfg.Addr, cfg.DBName, cfg.ParseTime = "tcp", os.Getenv("MYSQL_HOST")+":3306", os.Getenv("MYSQL_DATABASE"), true
	db, err := gorm.Open(mysql.Open(cfg.FormatDSN()), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal("test database connection failed")
	}
	pool, _ := db.DB()
	pool.SetMaxOpenConns(1)
	pool.SetMaxIdleConns(1)
	defer pool.Close()
	exec := func(query string) {
		t.Helper()
		if err := db.Exec(query).Error; err != nil {
			t.Fatal(err)
		}
	}
	// Connection-local tables shadow production names and vanish when the connection closes.
	exec("CREATE TEMPORARY TABLE order_items(id BIGINT,corp_id BIGINT,delivered_quantity INT)")
	exec("CREATE TEMPORARY TABLE order_business_records(id BIGINT,corp_id BIGINT,order_item_id BIGINT,kind VARCHAR(20),status VARCHAR(20),quantity INT,business_date DATE)")
	exec("INSERT INTO order_items VALUES(1,0,8),(2,0,3),(3,0,0),(4,1,99)")
	exec("INSERT INTO order_business_records VALUES(1,0,1,'delivery','confirmed',2,'2026-10-01'),(2,0,1,'delivery','confirmed',3,'2026-10-06'),(3,0,1,'delivery','confirmed',3,NULL),(4,0,1,'delivery','reversed',100,'2026-10-07'),(5,1,1,'delivery','confirmed',99,'2026-12-31'),(6,0,1,'scrap','confirmed',100,'2026-12-31')")
	rows, err := LoadDeliverySummaries(db, 0, []int64{1, 2, 3, 4})
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 3 {
		t.Fatal("company isolation failed")
	}
	if rows[1] != (DeliverySummary{LastDate: "2026-10-06", Delivered: 8, DatedQuantity: 5, UndatedQuantity: 3, Batches: 3}) {
		t.Fatalf("confirmed delivery summary incorrect: %+v", rows[1])
	}
	if rows[2].LastDate != "" || rows[2].UndatedQuantity != 3 {
		t.Fatal("legacy quantity invented a date")
	}
	if rows[3] != (DeliverySummary{}) {
		t.Fatal("unshipped item invented a delivery")
	}
	if empty, err := LoadDeliverySummaries(db, 0, nil); err != nil || len(empty) != 0 {
		t.Fatal("empty page failed")
	}
}
