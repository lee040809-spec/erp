package main

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	driver "github.com/go-sql-driver/mysql"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// An additive migration; never rewrites existing process values.
func run() error {
	apply := flag.Bool("apply", false, "add cutting_spec after verified SQL backup")
	output := flag.String("output", "", "backup SQL path")
	flag.Parse()
	cfg := driver.NewConfig()
	cfg.User = os.Getenv("MYSQL_USER")
	cfg.Passwd = os.Getenv("MYSQL_PASSWORD")
	cfg.Net = "tcp"
	cfg.Addr = os.Getenv("MYSQL_HOST") + ":3306"
	cfg.DBName = os.Getenv("MYSQL_DATABASE")
	cfg.Timeout = 10 * time.Second
	cfg.ReadTimeout = 60 * time.Second
	cfg.WriteTimeout = 60 * time.Second
	db, e := sql.Open("mysql", cfg.FormatDSN())
	if e != nil {
		return fmt.Errorf("database setup failed")
	}
	defer db.Close()
	db.SetMaxOpenConns(1)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	if e = db.PingContext(ctx); e != nil {
		return fmt.Errorf("database connection failed")
	}
	var table, ddl string
	if e = db.QueryRowContext(ctx, "SHOW CREATE TABLE order_processes").Scan(&table, &ddl); e != nil {
		return e
	}
	colsRows, e := db.QueryContext(ctx, "SHOW COLUMNS FROM order_processes")
	if e != nil {
		return e
	}
	var columns []string
	has := false
	for colsRows.Next() {
		var n, typ, nullable, key, extra string
		var def sql.NullString
		if e = colsRows.Scan(&n, &typ, &nullable, &key, &def, &extra); e != nil {
			colsRows.Close()
			return e
		}
		if n == "cutting_spec" {
			has = true
			continue
		}
		columns = append(columns, "`"+n+"`")
	}
	if e = colsRows.Err(); e != nil {
		colsRows.Close()
		return e
	}
	colsRows.Close()
	snapshot := func(file *os.File) (string, int, error) {
		rows, e := db.QueryContext(ctx, "SELECT "+strings.Join(columns, ",")+" FROM order_processes ORDER BY id")
		if e != nil {
			return "", 0, e
		}
		defer rows.Close()
		hash := sha256.New()
		count := 0
		for rows.Next() {
			values := make([][]byte, len(columns))
			targets := make([]interface{}, len(columns))
			for i := range values {
				targets[i] = &values[i]
			}
			if e = rows.Scan(targets...); e != nil {
				return "", count, e
			}
			encoded, _ := json.Marshal(values)
			hash.Write(encoded)
			hash.Write([]byte("\n"))
			if file != nil {
				literal := make([]string, len(values))
				for i, v := range values {
					if v == nil {
						literal[i] = "NULL"
					} else {
						literal[i] = "X'" + hex.EncodeToString(v) + "'"
					}
				}
				if _, e = fmt.Fprintf(file, "INSERT INTO order_processes (%s) VALUES (%s);\n", strings.Join(columns, ","), strings.Join(literal, ",")); e != nil {
					return "", count, e
				}
			}
			count++
		}
		return hex.EncodeToString(hash.Sum(nil)), count, rows.Err()
	}
	if *apply && has {
		fmt.Println("cutting_spec already exists; no migration needed")
		return nil
	}
	if !*apply {
		_, count, e := snapshot(nil)
		if e != nil {
			return e
		}
		fmt.Printf("processes=%d cutting_spec_exists=%t\n", count, has)
		return nil
	}
	if *output == "" {
		return fmt.Errorf("backup output path is required")
	}
	if e = os.MkdirAll(filepath.Dir(*output), 0700); e != nil {
		return e
	}
	file, e := os.OpenFile(*output, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if e != nil {
		return e
	}
	if _, e = fmt.Fprintf(file, "-- Process data backup before additive cutting specification migration\nSET NAMES utf8mb4;\n%s;\n", ddl); e != nil {
		file.Close()
		return e
	}
	before, count, e := snapshot(file)
	if e != nil {
		file.Close()
		return e
	}
	if e = file.Sync(); e != nil {
		file.Close()
		return e
	}
	if e = file.Close(); e != nil {
		return e
	}
	if !has {
		if _, e = db.ExecContext(ctx, "ALTER TABLE order_processes ADD COLUMN cutting_spec VARCHAR(500) NULL COMMENT '下料尺寸规格'"); e != nil {
			return e
		}
	}
	after, afterCount, e := snapshot(nil)
	if e != nil {
		return e
	}
	if before != after || count != afterCount {
		return fmt.Errorf("existing data verification differs; keep backup and inspect")
	}
	var present int
	if e = db.QueryRowContext(ctx, "SELECT COUNT(*) FROM information_schema.columns WHERE table_schema=DATABASE() AND table_name='order_processes' AND column_name='cutting_spec'").Scan(&present); e != nil {
		return e
	}
	if present != 1 {
		return fmt.Errorf("new column verification failed")
	}
	result := map[string]interface{}{"rows_before": count, "rows_after": afterCount, "existing_columns_sha256": before, "existing_data_unchanged": true, "column": "cutting_spec", "added": !has, "sql_backup": *output}
	data, _ := json.MarshalIndent(result, "", "  ")
	if e = os.WriteFile(*output+".verification.json", data, 0600); e != nil {
		return e
	}
	fmt.Printf("cutting_spec ready: %d rows verified unchanged; SQL backup saved\n", count)
	return nil
}
func main() {
	if e := run(); e != nil {
		fmt.Fprintln(os.Stderr, e)
		os.Exit(1)
	}
}
