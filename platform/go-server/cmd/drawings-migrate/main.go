package main

import (
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"gorm.io/gorm"
	"machining-erp/internal/drawings"
	"machining-erp/internal/models"
	"machining-erp/pkg/database"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Backup and verify all original order-item columns. The migration never rewrites them.
func snapshot(db *sql.DB, out *os.File) (string, int, error) {
	rows, e := db.Query("SELECT * FROM order_items ORDER BY id")
	if e != nil {
		return "", 0, e
	}
	defer rows.Close()
	columns, e := rows.Columns()
	if e != nil {
		return "", 0, e
	}
	quoted := make([]string, len(columns))
	for i, n := range columns {
		quoted[i] = "`" + n + "`"
	}
	hash := sha256.New()
	count := 0
	for rows.Next() {
		data := make([][]byte, len(columns))
		targets := make([]interface{}, len(columns))
		for i := range data {
			targets[i] = &data[i]
		}
		if e = rows.Scan(targets...); e != nil {
			return "", count, e
		}
		encoded, _ := json.Marshal(data)
		hash.Write(encoded)
		hash.Write([]byte("\n"))
		if out != nil {
			literals := make([]string, len(data))
			for i, v := range data {
				if v == nil {
					literals[i] = "NULL"
				} else {
					literals[i] = "X'" + hex.EncodeToString(v) + "'"
				}
			}
			if _, e = fmt.Fprintf(out, "INSERT INTO order_items (%s) VALUES (%s);\n", strings.Join(quoted, ","), strings.Join(literals, ",")); e != nil {
				return "", count, e
			}
		}
		count++
	}
	return hex.EncodeToString(hash.Sum(nil)), count, rows.Err()
}
func run() error {
	output := flag.String("output", "", "backup directory")
	root := flag.String("storage", "data/drawings", "file directory")
	flag.Parse()
	if *output == "" {
		return errors.New("backup directory required")
	}
	if os.Getenv("MYSQL_DATABASE") != "platform_test" && os.Getenv("ALLOW_DRAWING_MIGRATION") != "1" {
		return errors.New("set ALLOW_DRAWING_MIGRATION=1 to migrate another database")
	}
	db, e := database.Connect(&database.Config{Host: os.Getenv("MYSQL_HOST"), User: os.Getenv("MYSQL_USER"), Password: os.Getenv("MYSQL_PASSWORD"), Database: os.Getenv("MYSQL_DATABASE"), Port: 3306})
	if e != nil {
		return errors.New("database connection failed")
	}
	sqlDB, e := db.DB()
	if e != nil {
		return e
	}
	defer sqlDB.Close()
	if e = os.MkdirAll(*output, 0700); e != nil {
		return e
	}
	file, e := os.OpenFile(filepath.Join(*output, "order_items.sql"), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if e != nil {
		return e
	}
	defer file.Close()
	var table, ddl string
	if e = sqlDB.QueryRow("SHOW CREATE TABLE order_items").Scan(&table, &ddl); e != nil {
		return e
	}
	if _, e = fmt.Fprintf(file, "SET NAMES utf8mb4;\n%s;\n", ddl); e != nil {
		return e
	}
	before, count, e := snapshot(sqlDB, file)
	if e != nil {
		return e
	}
	if e = file.Sync(); e != nil {
		return e
	}
	if e = file.Close(); e != nil {
		return e
	}
	schema, e := os.ReadFile("sql/020_part_drawings.sql")
	if e != nil {
		return e
	}
	for _, statement := range strings.Split(string(schema), ";") {
		if strings.TrimSpace(statement) != "" {
			if e = db.Exec(statement).Error; e != nil {
				return e
			}
		}
	}
	var ids []int64
	if e = db.Table("order_items").Where("COALESCE(drawing_data,'')<>''").Order("id").Pluck("id", &ids).Error; e != nil {
		return e
	}
	store := drawings.NewStorage(*root)
	migrated := 0
	skipped := []int64{}
	for _, id := range ids {
		var item models.OrderItem
		if e = db.Select("id,corp_id,drawing_data").Where("id=?", id).Take(&item).Error; e != nil {
			return e
		}
		var existing int64
		if e = db.Model(&models.PartDrawing{}).Where("corp_id=? AND order_item_id=? AND legacy_source=1", item.CorpID, id).Count(&existing).Error; e != nil {
			return e
		}
		if existing > 0 {
			continue
		}
		value := *item.DrawingData
		split := strings.IndexByte(value, ',')
		if split < 0 || split > 100 || !strings.Contains(value[:split], ";base64") {
			skipped = append(skipped, id)
			continue
		}
		header := value[:split]
		ext := ""
		switch header {
		case "data:image/png;base64":
			ext = ".png"
		case "data:image/jpeg;base64":
			ext = ".jpg"
		case "data:application/pdf;base64":
			ext = ".pdf"
		}
		if ext == "" {
			skipped = append(skipped, id)
			continue
		}
		saved, err := store.Save(base64.NewDecoder(base64.StdEncoding, strings.NewReader(value[split+1:])), fmt.Sprintf("原有图纸-%d%s", id, ext))
		if err != nil {
			skipped = append(skipped, id)
			continue
		}
		d := models.PartDrawing{CorpID: item.CorpID, OrderItemID: &id, StorageKey: saved.Key, SHA256: saved.Hash, FileName: saved.Name, MimeType: saved.MIME, Size: saved.Size, Notes: "原有图纸迁移；原数据库字段完整保留", LegacySource: true, IsCurrent: true, CreatedAt: time.Now()}
		if e = db.Transaction(func(tx *gorm.DB) error {
			if e := tx.Create(&d).Error; e != nil {
				return e
			}
			return drawings.Event(tx, d, 0, "legacy_import", "原始字段保留，不改变订单或生产数据")
		}); e != nil {
			path, _ := store.Path(saved.Key)
			_ = os.Remove(path)
			return e
		}
		migrated++
	}
	after, afterCount, e := snapshot(sqlDB, nil)
	if e != nil {
		return e
	}
	result := map[string]interface{}{"rows_before": count, "rows_after": afterCount, "original_sha256": before, "original_data_unchanged": before == after && count == afterCount, "legacy_total": len(ids), "legacy_migrated": migrated, "legacy_compatibility_ids": skipped, "storage_directory": *root, "created_tables": []string{"part_drawings", "drawing_events"}}
	data, _ := json.MarshalIndent(result, "", "  ")
	if e = os.WriteFile(filepath.Join(*output, "verification.json"), data, 0600); e != nil {
		return e
	}
	if before != after || count != afterCount {
		return errors.New("original data verification differs; inspect backup")
	}
	fmt.Printf("Drawing schema ready; %d original rows unchanged, %d legacy drawings migrated, %d kept compatible.\n", count, migrated, len(skipped))
	return nil
}
func main() {
	if e := run(); e != nil {
		fmt.Fprintln(os.Stderr, e)
		os.Exit(1)
	}
}
