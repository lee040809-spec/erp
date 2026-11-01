// Additive upgrade with a hash comparison of every original table/column.
package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	driver "github.com/go-sql-driver/mysql"
	"machining-erp/internal/accounting"
	"machining-erp/internal/config"
	"machining-erp/internal/workflowmigration"
	"os"
	"time"
)

func main() {
	if e := run(); e != nil {
		fmt.Fprintln(os.Stderr, e)
		os.Exit(1)
	}
}
func run() error {
	if _, err := config.Initialize(); err != nil {
		return err
	}
	apply := flag.Bool("apply", false, "apply reviewed additive changes")
	out := flag.String("output", "", "comparison report")
	flag.Parse()
	if os.Getenv("MYSQL_DATABASE") != "platform_test" {
		return fmt.Errorf("restricted to platform_test")
	}
	cfg := driver.NewConfig()
	cfg.User = os.Getenv("MYSQL_USER")
	cfg.Passwd = os.Getenv("MYSQL_PASSWORD")
	cfg.Net = "tcp"
	cfg.Addr = fmt.Sprintf("%s:%d", config.Load().MySQLHost, config.Load().MySQLPort)
	cfg.DBName = os.Getenv("MYSQL_DATABASE")
	cfg.Timeout = 10 * time.Second
	cfg.ReadTimeout = time.Minute
	cfg.WriteTimeout = time.Minute
	db, e := sql.Open("mysql", cfg.FormatDSN())
	if e != nil {
		return e
	}
	defer db.Close()
	db.SetMaxOpenConns(1)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	before, e := workflowmigration.Snapshot(ctx, db, nil)
	if e != nil {
		return e
	}
	if *apply {
		var n int
		if e = db.QueryRowContext(ctx, "SELECT COUNT(*) FROM information_schema.columns WHERE table_schema=DATABASE() AND table_name='production_completions' AND column_name='pricing_status'").Scan(&n); e != nil {
			return e
		}
		if n == 0 {
			if _, e = db.ExecContext(ctx, accounting.PricingColumn); e != nil {
				return e
			}
		}
		for _, ddl := range accounting.Tables {
			if _, e = db.ExecContext(ctx, ddl); e != nil {
				return e
			}
		}
	}
	after, e := workflowmigration.Snapshot(ctx, db, before)
	if e != nil {
		return e
	}
	for i, t := range before {
		if t.Rows != after[i].Rows || t.Hash != after[i].Hash {
			return fmt.Errorf("original data mismatch: %s", t.Name)
		}
	}
	result := struct {
		OriginalDataUnchanged bool `json:"original_data_unchanged"`
		Before, After         []workflowmigration.TableCheck
	}{true, before, after}
	if *out != "" {
		b, _ := json.MarshalIndent(result, "", "  ")
		if e = os.WriteFile(*out, b, 0600); e != nil {
			return e
		}
	}
	fmt.Printf("Verified %d original tables: all original rows and column hashes unchanged\n", len(before))
	return nil
}
