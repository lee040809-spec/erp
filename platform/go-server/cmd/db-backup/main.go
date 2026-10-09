package main

import (
	"bufio"
	"context"
	"database/sql"
	"encoding/hex"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/go-sql-driver/mysql"
)

type object struct {
	Name string
	Type string
}

func main() {
	var host, port, user, database, output string
	flag.StringVar(&host, "host", "127.0.0.1", "MySQL host")
	flag.StringVar(&port, "port", "3306", "MySQL port")
	flag.StringVar(&user, "user", "root", "MySQL user")
	flag.StringVar(&database, "database", "erp", "database name")
	flag.StringVar(&output, "output", "", "output .sql path")
	flag.Parse()

	if output == "" {
		fatal(errors.New("必须通过 -output 指定备份文件"))
	}
	if err := validateIdentifier(database); err != nil {
		fatal(fmt.Errorf("数据库名无效: %w", err))
	}

	password := os.Getenv("MYSQL_PASSWORD")
	if password == "" {
		fatal(errors.New("MYSQL_PASSWORD environment variable is required"))
	}

	cfg := mysql.NewConfig()
	cfg.User = user
	cfg.Passwd = password
	cfg.Net = "tcp"
	cfg.Addr = host + ":" + port
	cfg.DBName = database
	cfg.Params = map[string]string{"charset": "utf8mb4"}
	cfg.Timeout = 15 * time.Second
	cfg.ReadTimeout = 2 * time.Minute
	cfg.WriteTimeout = 2 * time.Minute

	db, err := sql.Open("mysql", cfg.FormatDSN())
	password = ""
	if err != nil {
		fatal(err)
	}
	defer db.Close()
	db.SetMaxOpenConns(1)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		fatal(fmt.Errorf("数据库连接失败: %w", err))
	}

	if err := os.MkdirAll(filepath.Dir(output), 0750); err != nil {
		fatal(err)
	}
	partial := output + ".partial"
	_ = os.Remove(partial)
	f, err := os.OpenFile(partial, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		fatal(err)
	}
	ok := false
	defer func() {
		_ = f.Close()
		if !ok {
			_ = os.Remove(partial)
		}
	}()
	w := bufio.NewWriterSize(f, 1024*1024)

	if err := dump(ctx, db, w, database, host); err != nil {
		fatal(err)
	}
	if err := w.Flush(); err != nil {
		fatal(err)
	}
	if err := f.Sync(); err != nil {
		fatal(err)
	}
	if err := f.Close(); err != nil {
		fatal(err)
	}
	if err := os.Rename(partial, output); err != nil {
		fatal(err)
	}
	ok = true

	info, err := os.Stat(output)
	if err != nil {
		fatal(err)
	}
	fmt.Printf("备份完成: %s\n文件大小: %d 字节\n", output, info.Size())
}

func dump(ctx context.Context, db *sql.DB, w *bufio.Writer, database, host string) error {
	var version string
	if err := db.QueryRowContext(ctx, "SELECT VERSION()").Scan(&version); err != nil {
		return err
	}

	objects, err := listObjects(ctx, db, database)
	if err != nil {
		return err
	}

	fmt.Fprintf(w, "-- machining-erp database backup\n-- Database: %s\n-- Server: %s\n-- MySQL version: %s\n-- Created at (UTC): %s\n\n", database, host, version, time.Now().UTC().Format(time.RFC3339))
	io.WriteString(w, "SET NAMES utf8mb4;\nSET FOREIGN_KEY_CHECKS=0;\nSET UNIQUE_CHECKS=0;\nSET SQL_MODE='NO_AUTO_VALUE_ON_ZERO';\n\n")

	for _, obj := range objects {
		if obj.Type != "BASE TABLE" {
			continue
		}
		createSQL, err := showCreate(ctx, db, "TABLE", obj.Name)
		if err != nil {
			return fmt.Errorf("读取表 %s 结构失败: %w", obj.Name, err)
		}
		fmt.Fprintf(w, "-- Table: %s\nDROP TABLE IF EXISTS %s;\n%s;\n\n", obj.Name, quoteIdent(obj.Name), createSQL)
	}

	tx, err := db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelRepeatableRead, ReadOnly: true})
	if err != nil {
		return fmt.Errorf("启动只读一致性事务失败: %w", err)
	}
	defer tx.Rollback()

	for _, obj := range objects {
		if obj.Type != "BASE TABLE" {
			continue
		}
		count, err := dumpTableData(ctx, tx, w, obj.Name)
		if err != nil {
			return fmt.Errorf("导出表 %s 数据失败: %w", obj.Name, err)
		}
		fmt.Fprintf(os.Stderr, "已导出 %-32s %d 行\n", obj.Name, count)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("完成一致性事务失败: %w", err)
	}

	for _, obj := range objects {
		if obj.Type != "VIEW" {
			continue
		}
		createSQL, err := showCreate(ctx, db, "VIEW", obj.Name)
		if err != nil {
			return fmt.Errorf("读取视图 %s 结构失败: %w", obj.Name, err)
		}
		fmt.Fprintf(w, "-- View: %s\nDROP VIEW IF EXISTS %s;\n%s;\n\n", obj.Name, quoteIdent(obj.Name), createSQL)
	}

	if err := dumpTriggers(ctx, db, w, database); err != nil {
		return err
	}
	io.WriteString(w, "SET UNIQUE_CHECKS=1;\nSET FOREIGN_KEY_CHECKS=1;\n")
	return nil
}

func listObjects(ctx context.Context, db *sql.DB, database string) ([]object, error) {
	rows, err := db.QueryContext(ctx, `SELECT TABLE_NAME, TABLE_TYPE FROM information_schema.TABLES WHERE TABLE_SCHEMA = ? ORDER BY CASE TABLE_TYPE WHEN 'BASE TABLE' THEN 0 ELSE 1 END, TABLE_NAME`, database)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []object
	for rows.Next() {
		var o object
		if err := rows.Scan(&o.Name, &o.Type); err != nil {
			return nil, err
		}
		out = append(out, o)
	}
	return out, rows.Err()
}

func showCreate(ctx context.Context, db *sql.DB, kind, name string) (string, error) {
	rows, err := db.QueryContext(ctx, "SHOW CREATE "+kind+" "+quoteIdent(name))
	if err != nil {
		return "", err
	}
	defer rows.Close()
	cols, err := rows.Columns()
	if err != nil {
		return "", err
	}
	if !rows.Next() {
		return "", sql.ErrNoRows
	}
	values := make([]sql.RawBytes, len(cols))
	dest := make([]any, len(cols))
	for i := range values {
		dest[i] = &values[i]
	}
	if err := rows.Scan(dest...); err != nil {
		return "", err
	}
	if len(values) < 2 {
		return "", errors.New("SHOW CREATE 返回列不足")
	}
	return string(values[1]), nil
}

func dumpTableData(ctx context.Context, tx *sql.Tx, w *bufio.Writer, table string) (int64, error) {
	rows, err := tx.QueryContext(ctx, "SELECT * FROM "+quoteIdent(table))
	if err != nil {
		return 0, err
	}
	defer rows.Close()
	cols, err := rows.Columns()
	if err != nil {
		return 0, err
	}
	types, err := rows.ColumnTypes()
	if err != nil {
		return 0, err
	}
	quotedCols := make([]string, len(cols))
	for i, col := range cols {
		quotedCols[i] = quoteIdent(col)
	}
	prefix := "INSERT INTO " + quoteIdent(table) + " (" + strings.Join(quotedCols, ",") + ") VALUES\n"
	values := make([]sql.RawBytes, len(cols))
	dest := make([]any, len(cols))
	for i := range values {
		dest[i] = &values[i]
	}
	var count int64
	batchRows := 0
	for rows.Next() {
		if err := rows.Scan(dest...); err != nil {
			return count, err
		}
		if batchRows == 0 {
			io.WriteString(w, prefix)
		} else {
			io.WriteString(w, ",\n")
		}
		io.WriteString(w, "(")
		for i, raw := range values {
			if i > 0 {
				io.WriteString(w, ",")
			}
			io.WriteString(w, sqlLiteral(raw, types[i].DatabaseTypeName()))
		}
		io.WriteString(w, ")")
		batchRows++
		count++
		if batchRows >= 250 {
			io.WriteString(w, ";\n")
			batchRows = 0
		}
	}
	if batchRows > 0 {
		io.WriteString(w, ";\n")
	}
	if count > 0 {
		io.WriteString(w, "\n")
	}
	return count, rows.Err()
}

func dumpTriggers(ctx context.Context, db *sql.DB, w *bufio.Writer, database string) error {
	rows, err := db.QueryContext(ctx, `SELECT TRIGGER_NAME FROM information_schema.TRIGGERS WHERE TRIGGER_SCHEMA = ? ORDER BY TRIGGER_NAME`, database)
	if err != nil {
		return err
	}
	var names []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			rows.Close()
			return err
		}
		names = append(names, name)
	}
	if err := rows.Close(); err != nil {
		return err
	}
	sort.Strings(names)
	for _, name := range names {
		createSQL, err := showCreate(ctx, db, "TRIGGER", name)
		if err != nil {
			return fmt.Errorf("读取触发器 %s 失败: %w", name, err)
		}
		fmt.Fprintf(w, "-- Trigger: %s\nDROP TRIGGER IF EXISTS %s;\nDELIMITER ;;\n%s;;\nDELIMITER ;\n\n", name, quoteIdent(name), createSQL)
	}
	return nil
}

func sqlLiteral(raw sql.RawBytes, dbType string) string {
	if raw == nil {
		return "NULL"
	}
	t := strings.ToUpper(dbType)
	switch t {
	case "TINYINT", "SMALLINT", "MEDIUMINT", "INT", "INTEGER", "BIGINT", "DECIMAL", "NUMERIC", "FLOAT", "DOUBLE", "REAL", "YEAR":
		if _, err := strconv.ParseFloat(string(raw), 64); err == nil {
			return string(raw)
		}
	case "BIT", "BINARY", "VARBINARY", "TINYBLOB", "BLOB", "MEDIUMBLOB", "LONGBLOB", "GEOMETRY":
		return "0x" + strings.ToUpper(hex.EncodeToString(raw))
	}
	return "'" + escapeMySQL(raw) + "'"
}

func escapeMySQL(v []byte) string {
	var b strings.Builder
	for _, c := range v {
		switch c {
		case 0:
			b.WriteString(`\0`)
		case '\n':
			b.WriteString(`\n`)
		case '\r':
			b.WriteString(`\r`)
		case '\\':
			b.WriteString(`\\`)
		case '\'':
			b.WriteString(`\'`)
		case '"':
			b.WriteString(`\"`)
		case 0x1a:
			b.WriteString(`\Z`)
		default:
			b.WriteByte(c)
		}
	}
	return b.String()
}

func quoteIdent(s string) string {
	return "`" + strings.ReplaceAll(s, "`", "``") + "`"
}

func validateIdentifier(s string) error {
	if s == "" {
		return errors.New("不能为空")
	}
	for _, r := range s {
		if !(r == '_' || r >= '0' && r <= '9' || r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z') {
			return fmt.Errorf("包含不支持的字符 %q", r)
		}
	}
	return nil
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, "错误:", err)
	os.Exit(1)
}
