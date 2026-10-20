package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	mysqlDriver "github.com/go-sql-driver/mysql"
	"gorm.io/gorm"
	"io"
	"os"
	"testing"
	"time"
)

func TestStartupRetryTransientNetwork(t *testing.T) {
	calls := 0
	logs := 0
	want := &gorm.DB{}
	got, err := retryStartup(context.Background(), func() (*gorm.DB, error) {
		calls++
		if calls < 3 {
			return nil, fmt.Errorf("initial read: %w", io.EOF)
		}
		return want, nil
	}, 0, func(int) { logs++ })
	if err != nil || got != want || calls != 3 || logs != 2 {
		t.Fatalf("network recovery failed calls=%d logs=%d err=%v", calls, logs, err)
	}
	calls = 0
	_, err = retryStartup(context.Background(), func() (*gorm.DB, error) { calls++; return nil, io.EOF }, 0, nil)
	if err == nil || calls != 3 {
		t.Fatal("persistent failure must have a bounded retry count")
	}
}
func TestStartupDoesNotRetryCredentialOrSchemaErrors(t *testing.T) {
	for _, code := range []uint16{1045, 1049, 1050, 1054, 1146} {
		calls := 0
		_, err := retryStartup(context.Background(), func() (*gorm.DB, error) {
			calls++
			return nil, fmt.Errorf("read table: %w", &mysqlDriver.MySQLError{Number: code})
		}, 0, nil)
		if err == nil || calls != 1 {
			t.Fatalf("SQL error %d incorrectly retried", code)
		}
	}
}
func TestStartupRetryCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	calls := 0
	_, err := retryStartup(ctx, func() (*gorm.DB, error) { calls++; cancel(); return nil, io.EOF }, time.Hour, nil)
	if !errors.Is(err, context.Canceled) || calls != 1 {
		t.Fatal("cancelled startup kept retrying")
	}
}
func TestStartupCoreTablesReadOnly(t *testing.T) {
	if os.Getenv("RUN_DATABASE_READONLY") != "1" {
		t.Skip("explicit read-only database test required")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	db, err := ConnectForStartup(ctx, &Config{Host: os.Getenv("MYSQL_HOST"), Port: 3306, User: os.Getenv("MYSQL_USER"), Password: os.Getenv("MYSQL_PASSWORD"), Database: os.Getenv("MYSQL_DATABASE"), ReadTimeout: 20 * time.Second}, func(db *gorm.DB) error {
		// READ ONLY prevents accidentally reintroducing CREATE/ALTER at ordinary startup.
		return db.Transaction(func(tx *gorm.DB) error { return CheckCoreTables(tx) }, &sql.TxOptions{ReadOnly: true})
	}, func(attempt int) { t.Logf("transient connection retry %d", attempt) })
	if err != nil {
		t.Fatal(err)
	}
	pool, _ := db.DB()
	defer pool.Close()
	cancel()
	requestContext, requestCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer requestCancel()
	if err := db.WithContext(requestContext).Exec("SELECT 1").Error; err != nil {
		t.Fatalf("normal requests inherited cancelled startup context: %v", err)
	}
	t.Log("all core tables verified inside a read-only transaction; no schema metadata or DDL required")
}
