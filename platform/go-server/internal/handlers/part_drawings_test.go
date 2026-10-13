package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"image"
	"image/png"
	"machining-erp/internal/drawings"
	"machining-erp/internal/models"
	"machining-erp/pkg/database"
	"mime/multipart"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"
)

func TestPartDrawingsTemporaryTables(t *testing.T) {
	if os.Getenv("RUN_FINANCE_INTEGRATION") != "1" {
		t.Skip("explicit test database required")
	}
	db, e := database.Connect(&database.Config{Host: os.Getenv("MYSQL_HOST"), User: os.Getenv("MYSQL_USER"), Password: os.Getenv("MYSQL_PASSWORD"), Database: os.Getenv("MYSQL_DATABASE"), Port: 3306})
	if e != nil {
		t.Fatal("test database connection failed")
	}
	pool, _ := db.DB()
	pool.SetMaxOpenConns(1)
	pool.SetMaxIdleConns(1)
	defer pool.Close()
	exec := func(q string) {
		t.Helper()
		if e := db.Exec(q).Error; e != nil {
			t.Fatal(e)
		}
	}
	schema, e := os.ReadFile("../../sql/020_part_drawings.sql")
	if e != nil {
		t.Fatal(e)
	}
	for _, q := range strings.Split(string(schema), ";") {
		if strings.TrimSpace(q) != "" {
			exec(strings.Replace(q, "CREATE TABLE IF NOT EXISTS", "CREATE TEMPORARY TABLE", 1))
		}
	}
	exec("CREATE TEMPORARY TABLE orders (id BIGINT PRIMARY KEY,corp_id BIGINT,customer_id BIGINT,order_number VARCHAR(100),customer_name VARCHAR(100),customer_short_name VARCHAR(100))")
	exec("CREATE TEMPORARY TABLE order_items (id BIGINT PRIMARY KEY,corp_id BIGINT,order_id BIGINT,part_name VARCHAR(100),part_number VARCHAR(100),status VARCHAR(30),drawing_data LONGTEXT)")
	exec("CREATE TEMPORARY TABLE order_processes (id BIGINT PRIMARY KEY,corp_id BIGINT,order_item_id BIGINT,status VARCHAR(30),planned_start_at DATETIME NULL,planned_end_at DATETIME NULL,actual_start_at DATETIME NULL,actual_end_at DATETIME NULL,deleted_at DATETIME NULL)")
	exec("CREATE TEMPORARY TABLE users (id BIGINT PRIMARY KEY,corp_id BIGINT,role_id BIGINT)")
	exec("CREATE TEMPORARY TABLE permissions (corp_id BIGINT,role_id BIGINT,resource_id BIGINT)")
	exec("CREATE TEMPORARY TABLE resources (id BIGINT PRIMARY KEY,resource_key VARCHAR(100))")
	exec("INSERT INTO users VALUES (9,0,9)")
	exec("INSERT INTO resources VALUES (1,'dashboard')")
	exec("INSERT INTO permissions VALUES (0,9,1)")
	exec("INSERT INTO orders VALUES (1,0,10,'DRAW-001','测试客户','测试'),(2,0,20,'DRAW-002','另一客户','另'),(3,1,10,'DRAW-003','隔离客户','隔离')")
	exec("INSERT INTO order_items VALUES (1,0,1,'零件A','A-1','pending',NULL),(2,0,1,'零件A','A-1','processing',NULL),(3,0,2,'零件A','A-1','pending',NULL),(4,1,3,'隔离零件','A-1','pending',NULL),(5,0,1,'另一图号','B-1','pending',NULL)")
	root := t.TempDir()
	h := NewPartDrawingHandler(db, root)
	var imageData bytes.Buffer
	_ = png.Encode(&imageData, image.NewRGBA(image.Rect(0, 0, 8, 8)))
	call := func(handler gin.HandlerFunc, path, body string, corp, actor int64) *httptest.ResponseRecorder {
		t.Helper()
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Set("corpID", corp)
		c.Set("userID", actor)
		c.Set("roleType", "admin")
		c.Request = httptest.NewRequest("GET", path, strings.NewReader(body))
		c.Request.Header.Set("Content-Type", "application/json")
		if strings.Contains(path, "order-items/4") {
			c.Params = gin.Params{{Key: "itemId", Value: "4"}}
		} else {
			c.Params = gin.Params{{Key: "itemId", Value: "1"}, {Key: "drawingId", Value: "1"}}
		}
		handler(c)
		return w
	}
	upload := func(item string, reason string) *httptest.ResponseRecorder {
		t.Helper()
		var b bytes.Buffer
		writer := multipart.NewWriter(&b)
		_ = writer.WriteField("item_id", item)
		_ = writer.WriteField("reason", reason)
		_ = writer.WriteField("version", "A")
		file, _ := writer.CreateFormFile("file", "测试图纸.png")
		_, _ = file.Write(imageData.Bytes())
		_ = writer.Close()
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Set("corpID", int64(0))
		c.Set("userID", int64(7))
		c.Set("roleType", "admin")
		c.Request = httptest.NewRequest("POST", "/drawings", &b)
		c.Request.Header.Set("Content-Type", writer.FormDataContentType())
		h.Upload(c)
		return w
	}
	response := upload("1", "")
	if response.Code != 200 {
		t.Fatal("upload", response.Code, response.Body.String())
	}
	var result struct{ Data models.PartDrawing }
	if e = json.Unmarshal(response.Body.Bytes(), &result); e != nil {
		t.Fatal(e)
	}
	if result.Data.ID != 1 || !result.Data.IsCurrent {
		t.Fatal("drawing not linked", response.Body.String())
	}
	// A read-only dashboard grant must not become drawing management access.
	for index, handler := range []gin.HandlerFunc{h.List, h.Upload, h.State, h.Reference} {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Set("corpID", int64(0))
		c.Set("userID", int64(9))
		c.Set("roleType", "user")
		c.Params = gin.Params{{Key: "itemId", Value: "1"}, {Key: "drawingId", Value: "1"}}
		c.Request = httptest.NewRequest("GET", "/drawings", nil)
		handler(c)
		if index == 0 && w.Code == 200 {
			var list struct {
				Data struct {
					CanManage bool `json:"can_manage"`
				}
			}
			if err := json.Unmarshal(w.Body.Bytes(), &list); err != nil || list.Data.CanManage {
				t.Fatal("read-only dashboard can manage drawings", w.Body.String())
			}
		} else if index == 0 || w.Code != 403 {
			t.Fatal("read-only drawing permission", w.Code, w.Body.String())
		}
	}
	if w := call(h.File, "/drawings/1/file", "", 0, 7); w.Code != 200 || !bytes.Equal(w.Body.Bytes(), imageData.Bytes()) {
		t.Fatal("original download differs", w.Code)
	}
	if w := call(h.File, "/drawings/1/file", "", 1, 7); w.Code != 404 {
		t.Fatal("cross-company file leaked", w.Code)
	}
	if w := call(h.List, "/order-items/4/drawings", "", 0, 7); w.Code != 404 {
		t.Fatal("cross-company item leaked", w.Code)
	}
	if w := upload("2", ""); w.Code != 400 || !strings.Contains(w.Body.String(), "变更说明") {
		t.Fatal("production protection missing", w.Code, w.Body.String())
	}
	if w := upload("2", "补充工艺图，已核对"); w.Code != 200 {
		t.Fatal("explained production change failed", w.Body.String())
	}
	if w := call(h.State, "/drawings/1", `{"is_current":false,"reason":"停用"}`, 0, 7); w.Code != 200 {
		t.Fatal(w.Body.String())
	}
	var current models.PartDrawing
	_ = db.First(&current, 1).Error
	if current.IsCurrent {
		t.Fatal("archive did not persist")
	}
	if w := call(h.File, "/drawings/1/file", "", 0, 7); w.Code != 200 {
		t.Fatal("archive erased history")
	}
	if w := call(h.State, "/drawings/1", `{"is_current":true}`, 0, 7); w.Code != 200 {
		t.Fatal(w.Body.String())
	}
	if w := call(h.Reference, "/drawings/reference", `{"source_id":1,"item_id":3}`, 0, 7); w.Code != 400 || !strings.Contains(w.Body.String(), "相同客户") {
		t.Fatal("cross-customer reference allowed", w.Code, w.Body.String())
	}
	if w := call(h.Reference, "/drawings/reference", `{"source_id":1,"item_id":5}`, 0, 7); w.Code != 400 || !strings.Contains(w.Body.String(), "图号") {
		t.Fatal("wrong part number reference allowed", w.Code, w.Body.String())
	}
	if w := call(h.Reference, "/drawings/reference", `{"source_id":1,"item_id":2,"reason":"核对版本A"}`, 0, 7); w.Code != 200 {
		t.Fatal("valid reference failed", w.Body.String())
	}
	if w := call(h.Search, "/drawings/search?customer_id=10&q=零件", "", 0, 7); w.Code != 200 {
		t.Fatal("history search", w.Body.String())
	}
	// Drafts belong to their uploader; transactional attachment rolls back on any invalid ID.
	draft := models.PartDrawing{CorpID: 0, StorageKey: current.StorageKey, SHA256: current.SHA256, FileName: "draft.png", MimeType: "image/png", Size: 1, UploadedBy: 8, CreatedAt: time.Now()}
	if e = db.Create(&draft).Error; e != nil {
		t.Fatal(e)
	}
	var item models.OrderItem
	_ = db.First(&item, 1).Error
	if e = db.Transaction(func(tx *gorm.DB) error { return drawings.Bind(tx, item, []int64{draft.ID}, 7, "") }); e == nil {
		t.Fatal("another user's draft accepted")
	}
	if e = db.Transaction(func(tx *gorm.DB) error { return drawings.Bind(tx, item, []int64{draft.ID, 999999}, 8, "") }); e == nil {
		t.Fatal("invalid batch accepted")
	}
	_ = db.First(&draft, draft.ID).Error
	if draft.OrderItemID != nil {
		t.Fatal("partial draft attachment committed")
	}
	if e = db.Transaction(func(tx *gorm.DB) error { return drawings.Bind(tx, item, []int64{draft.ID}, 8, "") }); e != nil {
		t.Fatal(e)
	}
	var count int
	query := "SELECT " + drawings.CountSQL("i") + " FROM order_items i WHERE i.id=1"
	if e = db.Raw(query).Scan(&count).Error; e != nil {
		t.Fatal(e)
	}
	if count != 2 {
		t.Fatal("metadata count mismatch", count)
	}
	var events int64
	_ = db.Model(&models.DrawingEvent{}).Count(&events).Error
	if events < 6 {
		t.Fatal("audit missing", events)
	}
	t.Log(fmt.Sprintf("Uploads, exact download, company/owner/customer/part isolation, processing protection, archive/restore, references, atomic binding, metadata and audit verified (%d events).", events))
}
