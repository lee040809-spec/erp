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
	seedLegacySuppliers := true
	for _, table := range before {
		if table.Name == "outsourcing_suppliers" {
			seedLegacySuppliers = false
			break
		}
	}
	if *apply {
		for _, q := range workflowmigration.Tables {
			if _, err = db.ExecContext(ctx, q); err != nil {
				return err
			}
		}
		for _, c := range workflowmigration.Columns {
			var n int
			if err = db.QueryRowContext(ctx, "SELECT COUNT(*) FROM information_schema.columns WHERE table_schema=DATABASE() AND table_name=? AND column_name=?", c.Table, c.Name).Scan(&n); err != nil {
				return err
			}
			if n == 0 {
				if _, err = db.ExecContext(ctx, "ALTER TABLE "+c.Table+" ADD COLUMN "+c.Name+" "+c.Definition); err != nil {
					return err
				}
			}
		}
		for _, q := range workflowmigration.Tables {
			if _, err = db.ExecContext(ctx, q); err != nil {
				return err
			}
		}
		// Seed only on first installation. A renamed supplier's historical transaction snapshot
		// must never be mistaken for a second supplier on subsequent additive upgrades.
		if seedLegacySuppliers {
			seed, seedErr := db.BeginTx(ctx, nil)
			if seedErr != nil {
				return seedErr
			}
			if _, err = seed.ExecContext(ctx, `INSERT INTO outsourcing_suppliers(corp_id,name,created_at,updated_at) SELECT DISTINCT d.corp_id,TRIM(d.supplier_name),NOW(3),NOW(3) FROM outsourcing_dispatches d WHERE TRIM(d.supplier_name)<>'' AND NOT EXISTS (SELECT 1 FROM outsourcing_suppliers s WHERE s.corp_id=d.corp_id AND s.name=TRIM(d.supplier_name))`); err != nil {
				seed.Rollback()
				return err
			}
			if _, err = seed.ExecContext(ctx, `UPDATE outsourcing_dispatches d JOIN outsourcing_suppliers s ON s.corp_id=d.corp_id AND s.name=TRIM(d.supplier_name) SET d.supplier_id=s.id WHERE d.supplier_id IS NULL`); err != nil {
				seed.Rollback()
				return err
			}
			if err = seed.Commit(); err != nil {
				return err
			}
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
