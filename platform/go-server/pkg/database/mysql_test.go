package database

import (
	"context"
	driver "github.com/go-sql-driver/mysql"
	"os"
	"testing"
	"time"
)

func TestDriverConfig(t *testing.T) {
	cfg := &Config{Host: "::1", Port: 3306, User: "appuser", Password: "test@pass:/?&", Database: "test"}
	c, e := driver.ParseDSN(driverConfig(cfg).FormatDSN())
	if e != nil {
		t.Fatal(e)
	}
	if c.Passwd != cfg.Password || c.Addr != "[::1]:3306" || !c.InterpolateParams || !c.CheckConnLiveness || !c.ParseTime || c.Loc != time.Local || c.Params["charset"] != "utf8mb4" {
		t.Fatal("connection configuration not preserved")
	}
	if c.MultiStatements {
		t.Fatal("multiple statements must remain disabled")
	}
}
func TestDatabaseReadOnlyParameters(t *testing.T) {
	if os.Getenv("RUN_DATABASE_READONLY") != "1" {
		t.Skip("read-only database verification requires explicit opt-in")
	}
	cfg := &Config{Host: os.Getenv("MYSQL_HOST"), Port: 3306, User: os.Getenv("MYSQL_USER"), Password: os.Getenv("MYSQL_PASSWORD"), Database: os.Getenv("MYSQL_DATABASE")}
	db, e := Connect(cfg)
	if e != nil {
		t.Fatal("database connection failed")
	}
	sqlDB, _ := db.DB()
	defer sqlDB.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	db = db.WithContext(ctx)
	for _, value := range []string{"下料 Φ30×45", "O'Brien \\ ? ; DROP TABLE ignored", "line\n中文🙂", ""} {
		var got string
		if e = db.Raw("SELECT ?", value).Scan(&got).Error; e != nil {
			t.Fatal(e)
		}
		if got != value {
			t.Fatal("parameter round trip mismatch")
		}
	}
	var row struct {
		Amount  string
		Missing int
		Day     time.Time
	}
	if e = db.Raw("SELECT CAST(? AS DECIMAL(12,2)) AS amount, (? IS NULL) AS missing, DATE(?) AS day", "12345.67", nil, "2026-10-11").Scan(&row).Error; e != nil {
		t.Fatal(e)
	}
	if row.Amount != "12345.67" || row.Missing != 1 || row.Day.Format("2006-01-02") != "2026-10-11" {
		t.Fatalf("numeric/date/null mismatch: %+v", row)
	}
}

func TestWarmReadPoolAfterIdle(t *testing.T) {
	if os.Getenv("RUN_DATABASE_READONLY") != "1" {
		t.Skip("explicit read-only verification required")
	}
	db, err := Connect(&Config{Host: os.Getenv("MYSQL_HOST"), Port: 3306, User: os.Getenv("MYSQL_USER"), Password: os.Getenv("MYSQL_PASSWORD"), Database: os.Getenv("MYSQL_DATABASE")})
	if err != nil {
		t.Fatal("connection failed")
	}
	pool, _ := db.DB()
	defer pool.Close()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan struct{})
	go func() { defer close(done); KeepReadConnectionsWarm(ctx, pool, 3) }()
	deadline := time.Now().Add(10 * time.Second)
	for pool.Stats().Idle < 3 && time.Now().Before(deadline) {
		time.Sleep(100 * time.Millisecond)
	}
	if pool.Stats().Idle < 3 {
		t.Fatal("read connections did not warm")
	}
	closed := pool.Stats().MaxIdleTimeClosed
	time.Sleep(45 * time.Second)
	stats := pool.Stats()
	if stats.Idle < 3 || stats.MaxIdleTimeClosed != closed {
		t.Fatalf("idle pool was lost: %+v", stats)
	}
	start := time.Now()
	if err = pool.PingContext(ctx); err != nil {
		t.Fatal("idle connection not usable")
	}
	t.Logf("after 45 seconds idle: ready=%d ping=%v", stats.Idle, time.Since(start))
	cancel()
	select {
	case <-done:
	case <-time.After(10 * time.Second):
		t.Fatal("pool monitor did not stop")
	}
}
