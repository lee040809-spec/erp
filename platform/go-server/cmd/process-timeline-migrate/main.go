// Adds nullable nodes without manufacturing historical time data, and compares every original column/row.
package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	driver "github.com/go-sql-driver/mysql"
	"machining-erp/internal/workflowmigration"
	"os"
	"strings"
	"time"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
func run() error {
	apply := flag.Bool("apply", false, "apply additive schema migration")
	out := flag.String("output", "", "comparison report path")
	flag.Parse()
	if os.Getenv("MYSQL_DATABASE") != "platform_test" {
		return fmt.Errorf("this migration helper is restricted to platform_test; use reviewed SQL for production")
	}
	cfg := driver.NewConfig()
	cfg.User = os.Getenv("MYSQL_USER")
	cfg.Passwd = os.Getenv("MYSQL_PASSWORD")
	cfg.Net = "tcp"
	cfg.Addr = os.Getenv("MYSQL_HOST") + ":3306"
	cfg.DBName = os.Getenv("MYSQL_DATABASE")
	cfg.Timeout = 10 * time.Second
	cfg.ReadTimeout = time.Minute
	cfg.WriteTimeout = time.Minute
	db, err := sql.Open("mysql", cfg.FormatDSN())
	if err != nil {
		return fmt.Errorf("connect failed")
	}
	defer db.Close()
	db.SetMaxOpenConns(1)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	if err = db.PingContext(ctx); err != nil {
		return fmt.Errorf("database unavailable")
	}
	before, err := workflowmigration.Snapshot(ctx, db, nil)
	if err != nil {
		return err
	}
	if *apply {
		adds := []string{}
		for _, column := range []string{"planned_start_at", "planned_end_at", "actual_start_at", "actual_end_at"} {
			var n int
			if err = db.QueryRowContext(ctx, "SELECT COUNT(*) FROM information_schema.columns WHERE table_schema=DATABASE() AND table_name='order_processes' AND column_name=?", column).Scan(&n); err != nil {
				return err
			}
			if n == 0 {
				adds = append(adds, "ADD COLUMN `"+column+"` DATETIME(3) NULL")
			}
		}
		if len(adds) > 0 {
			if _, err = db.ExecContext(ctx, "ALTER TABLE order_processes "+strings.Join(adds, ",")); err != nil {
				return err
			}
		}
		if _, err = db.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS process_time_events(id BIGINT NOT NULL AUTO_INCREMENT PRIMARY KEY,corp_id BIGINT NOT NULL,order_item_id BIGINT NOT NULL,process_id BIGINT NOT NULL,before_nodes TEXT NOT NULL,after_nodes TEXT NOT NULL,reason VARCHAR(500) NOT NULL DEFAULT '',source VARCHAR(32) NOT NULL,created_by BIGINT NOT NULL,created_at DATETIME(3) NOT NULL,KEY ix_process_time_item(corp_id,order_item_id,id),KEY ix_process_time_process(corp_id,process_id,id)) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`); err != nil {
			return err
		}
	}
	after, err := workflowmigration.Snapshot(ctx, db, before)
	if err != nil {
		return err
	}
	for i, t := range before {
		if t.Rows != after[i].Rows || t.Hash != after[i].Hash {
			return fmt.Errorf("original data mismatch in %s", t.Name)
		}
	}
	var timed int64
	if err = db.QueryRowContext(ctx, "SELECT COUNT(*) FROM order_processes WHERE planned_start_at IS NOT NULL OR planned_end_at IS NOT NULL OR actual_start_at IS NOT NULL OR actual_end_at IS NOT NULL").Scan(&timed); err != nil {
		return err
	}
	report := struct {
		Database              string `json:"database"`
		Applied               bool   `json:"applied"`
		OriginalDataUnchanged bool   `json:"original_data_unchanged"`
		Before, After         []workflowmigration.TableCheck
		TimedRows             int64 `json:"timed_rows"`
	}{cfg.DBName, *apply, true, before, after, timed}
	b, _ := json.MarshalIndent(report, "", "  ")
	if *out != "" {
		if err = os.WriteFile(*out, b, 0600); err != nil {
			return err
		}
	}
	fmt.Printf("Schema verified; original rows and columns unchanged in %d tables; timed rows=%d\n", len(before), timed)
	return nil
}
