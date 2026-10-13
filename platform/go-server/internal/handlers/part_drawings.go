package handlers

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"machining-erp/internal/drawings"
	"machining-erp/internal/middleware"
	"machining-erp/internal/models"
	"mime"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

type PartDrawingHandler struct {
	db      *gorm.DB
	store   *drawings.Storage
	uploads chan struct{}
}

func NewPartDrawingHandler(db *gorm.DB, root string) *PartDrawingHandler {
	return &PartDrawingHandler{db: db, store: drawings.NewStorage(root), uploads: make(chan struct{}, 2)}
}
func (h *PartDrawingHandler) access(c *gin.Context, write bool) bool {
	if write {
		return requireModuleAccess(h.db, c, "orders", "production_orders", "production_dashboard")
	}
	return requireModuleAccess(h.db, c, "orders", "production_orders", "dashboard", "production_dashboard", "overdue", "warning_orders", "imminent_orders", "production_overdue", "production_warning", "production_imminent", "production_delivery", "reconciliation")
}
func drawingID(c *gin.Context, key string) (int64, bool) {
	id, e := strconv.ParseInt(c.Param(key), 10, 64)
	if e != nil || id <= 0 {
		c.JSON(400, gin.H{"error": "无效的零件或图纸编号"})
		return 0, false
	}
	return id, true
}
func drawingError(c *gin.Context, e error) {
	if e != nil {
		c.JSON(400, gin.H{"error": e.Error()})
	}
}
func (h *PartDrawingHandler) List(c *gin.Context) {
	if !h.access(c, false) {
		return
	}
	id, ok := drawingID(c, "itemId")
	if !ok {
		return
	}
	db := h.db.WithContext(c.Request.Context())
	corp := middleware.GetCorpID(c)
	var item struct {
		ID         int64
		Legacy     bool
		CustomerID *int64
	}
	if e := db.Table("order_items i").Select("i.id,COALESCE(i.drawing_data,'')<>'' AS legacy,o.customer_id").Joins("JOIN orders o ON o.id=i.order_id AND o.corp_id=i.corp_id").Where("i.id=? AND i.corp_id=?", id, corp).Take(&item).Error; e != nil {
		c.JSON(404, gin.H{"error": "零件不存在"})
		return
	}
	rows := []models.PartDrawing{}
	if e := db.Where("corp_id=? AND order_item_id=?", corp, id).Order("is_current DESC,id DESC").Limit(200).Find(&rows).Error; e != nil {
		drawingError(c, e)
		return
	}
	events := []models.DrawingEvent{}
	if e := db.Where("corp_id=? AND order_item_id=?", corp, id).Order("id DESC").Limit(50).Find(&events).Error; e != nil {
		drawingError(c, e)
		return
	}
	for _, row := range rows {
		if row.LegacySource {
			item.Legacy = false
			break
		}
	}
	c.JSON(200, gin.H{"data": gin.H{"customer_id": item.CustomerID, "drawings": rows, "legacy": item.Legacy, "events": events, "can_manage": hasModuleAccess(h.db, c, "orders", "production_orders", "production_dashboard")}})
}
func (h *PartDrawingHandler) Upload(c *gin.Context) {
	if !h.access(c, true) {
		return
	}
	select {
	case h.uploads <- struct{}{}:
		defer func() { <-h.uploads }()
	default:
		c.JSON(503, gin.H{"error": "正在处理图纸，请稍后上传"})
		return
	}
	file, e := c.FormFile("file")
	if e != nil {
		c.JSON(400, gin.H{"error": "请选择图纸文件（每份不超过20MB）"})
		return
	}
	if file.Size > drawings.MaxFileSize {
		c.JSON(413, gin.H{"error": "单份图纸不能超过20MB"})
		return
	}
	version, number, notes := strings.TrimSpace(c.PostForm("version")), strings.TrimSpace(c.PostForm("drawing_number")), strings.TrimSpace(c.PostForm("notes"))
	if len([]rune(version)) > 100 || len([]rune(number)) > 100 || len([]rune(notes)) > 500 {
		c.JSON(400, gin.H{"error": "图号/版本最多100字，备注最多500字"})
		return
	}
	itemID, _ := strconv.ParseInt(c.PostForm("item_id"), 10, 64)
	replaceID, _ := strconv.ParseInt(c.PostForm("replaces_id"), 10, 64)
	if itemID < 0 || replaceID < 0 {
		c.JSON(400, gin.H{"error": "无效的零件或图纸编号"})
		return
	}
	corp, actor := middleware.GetCorpID(c), middleware.GetUserID(c)
	db := h.db.WithContext(c.Request.Context())
	var pending int64
	if e = db.Model(&models.PartDrawing{}).Where("corp_id=? AND uploaded_by=? AND order_item_id IS NULL", corp, actor).Count(&pending).Error; e != nil {
		drawingError(c, e)
		return
	}
	if pending >= 100 {
		c.JSON(400, gin.H{"error": "待关联图纸较多，请先保存订单关联已上传图纸"})
		return
	}
	reader, e := file.Open()
	if e != nil {
		drawingError(c, e)
		return
	}
	defer reader.Close()
	saved, e := h.store.Save(reader, file.Filename)
	if e != nil {
		drawingError(c, e)
		return
	}
	d := models.PartDrawing{CorpID: corp, StorageKey: saved.Key, SHA256: saved.Hash, FileName: saved.Name, MimeType: saved.MIME, Size: saved.Size, Version: version, DrawingNumber: number, Notes: notes, UploadedBy: actor, CreatedAt: time.Now()}
	if replaceID > 0 {
		d.ReplacesID = &replaceID
	}
	e = db.Transaction(func(tx *gorm.DB) error {
		if e := tx.Create(&d).Error; e != nil {
			return e
		}
		if itemID > 0 {
			item, e := drawings.Item(tx, corp, itemID)
			if e != nil {
				return fmt.Errorf("零件不存在")
			}
			return drawings.Bind(tx, item, []int64{d.ID}, actor, c.PostForm("reason"))
		}
		return nil
	})
	if e != nil {
		path, _ := h.store.Path(saved.Key)
		_ = os.Remove(path)
		drawingError(c, e)
		return
	}
	if itemID > 0 {
		d.OrderItemID = &itemID
		d.IsCurrent = true
	}
	c.JSON(200, gin.H{"data": d})
}
func (h *PartDrawingHandler) State(c *gin.Context) {
	if !h.access(c, true) {
		return
	}
	id, ok := drawingID(c, "drawingId")
	if !ok {
		return
	}
	var req struct {
		IsCurrent *bool  `json:"is_current"`
		Reason    string `json:"reason"`
	}
	if c.ShouldBindJSON(&req) != nil || req.IsCurrent == nil {
		c.JSON(400, gin.H{"error": "请选择当前生产图纸或历史图纸"})
		return
	}
	corp, actor := middleware.GetCorpID(c), middleware.GetUserID(c)
	e := h.db.WithContext(c.Request.Context()).Transaction(func(tx *gorm.DB) error {
		// Serialize every modification on the item before locking any drawing row.
		var initial models.PartDrawing
		if e := tx.Where("id=? AND corp_id=? AND order_item_id IS NOT NULL", id, corp).Take(&initial).Error; e != nil {
			return fmt.Errorf("图纸不存在")
		}
		item, e := drawings.Item(tx, corp, *initial.OrderItemID)
		if e != nil {
			return fmt.Errorf("零件不存在")
		}
		var d models.PartDrawing
		if e = tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id=? AND corp_id=?", id, corp).Take(&d).Error; e != nil {
			return e
		}
		if d.IsCurrent == *req.IsCurrent {
			return nil
		}
		if e = drawings.ChangeReason(tx, item, req.Reason); e != nil {
			return e
		}
		if e = tx.Model(&d).Update("is_current", *req.IsCurrent).Error; e != nil {
			return e
		}
		action := "archived"
		if *req.IsCurrent {
			action = "current"
		}
		return drawings.Event(tx, d, actor, action, req.Reason)
	})
	if e != nil {
		drawingError(c, e)
		return
	}
	c.JSON(200, gin.H{"data": gin.H{"saved": true}})
}
func (h *PartDrawingHandler) Search(c *gin.Context) {
	if !h.access(c, false) {
		return
	}
	q := strings.TrimSpace(c.Query("q"))
	customer, _ := strconv.ParseInt(c.Query("customer_id"), 10, 64)
	if len([]rune(q)) < 2 || len([]rune(q)) > 100 || customer <= 0 {
		c.JSON(400, gin.H{"error": "请选择客户，并输入至少两个字的零件名称、图号或文件名"})
		return
	}
	type found struct {
		models.PartDrawing
		PartName     string `json:"part_name"`
		PartNumber   string `json:"part_number"`
		OrderNumber  string `json:"order_number"`
		CustomerName string `json:"customer_name"`
	}
	rows := []found{}
	q = "%" + strings.NewReplacer("\\", "\\\\", "%", "\\%", "_", "\\_").Replace(q) + "%"
	e := h.db.WithContext(c.Request.Context()).Table("part_drawings d").Select("d.*,i.part_name,COALESCE(i.part_number,'') part_number,o.order_number,COALESCE(NULLIF(o.customer_short_name,''),o.customer_name,'') customer_name").Joins("JOIN order_items i ON i.id=d.order_item_id AND i.corp_id=d.corp_id").Joins("JOIN orders o ON o.id=i.order_id AND o.corp_id=i.corp_id").Where("d.corp_id=? AND o.customer_id=? AND (i.part_name LIKE ? OR i.part_number LIKE ? OR d.file_name LIKE ? OR d.drawing_number LIKE ?)", middleware.GetCorpID(c), customer, q, q, q, q).Order("d.id DESC").Limit(30).Scan(&rows).Error
	if e != nil {
		drawingError(c, e)
		return
	}
	c.JSON(200, gin.H{"data": rows})
}
func (h *PartDrawingHandler) Reference(c *gin.Context) {
	if !h.access(c, true) {
		return
	}
	var req struct {
		SourceID int64  `json:"source_id"`
		ItemID   int64  `json:"item_id"`
		Reason   string `json:"reason"`
	}
	if c.ShouldBindJSON(&req) != nil || req.SourceID <= 0 || req.ItemID < 0 {
		c.JSON(400, gin.H{"error": "请选择历史图纸"})
		return
	}
	corp, actor := middleware.GetCorpID(c), middleware.GetUserID(c)
	var d models.PartDrawing
	e := h.db.WithContext(c.Request.Context()).Transaction(func(tx *gorm.DB) error {
		if e := tx.Where("id=? AND corp_id=? AND order_item_id IS NOT NULL", req.SourceID, corp).Take(&d).Error; e != nil {
			return fmt.Errorf("历史图纸不存在")
		}
		d.LegacySource = false
		d.SourceDrawingID = &req.SourceID
		d.ID = 0
		d.OrderItemID = nil
		d.ReplacesID = nil
		d.IsCurrent = false
		d.UploadedBy = actor
		d.CreatedAt = time.Now()
		if e := tx.Create(&d).Error; e != nil {
			return e
		}
		if req.ItemID > 0 {
			item, e := drawings.Item(tx, corp, req.ItemID)
			if e != nil {
				return fmt.Errorf("零件不存在")
			}
			return drawings.Bind(tx, item, []int64{d.ID}, actor, req.Reason)
		}
		return nil
	})
	if e != nil {
		drawingError(c, e)
		return
	}
	if req.ItemID > 0 {
		d.OrderItemID = &req.ItemID
		d.IsCurrent = true
	}
	c.JSON(200, gin.H{"data": d})
}
func (h *PartDrawingHandler) File(c *gin.Context) {
	if !h.access(c, false) {
		return
	}
	id, ok := drawingID(c, "drawingId")
	if !ok {
		return
	}
	var d models.PartDrawing
	db := h.db.WithContext(c.Request.Context())
	corp := middleware.GetCorpID(c)
	if e := db.Where("id=? AND corp_id=?", id, corp).Take(&d).Error; e != nil {
		c.JSON(404, gin.H{"error": "图纸不存在"})
		return
	}
	if d.OrderItemID == nil {
		if d.UploadedBy != middleware.GetUserID(c) {
			c.JSON(404, gin.H{"error": "图纸不存在"})
			return
		}
	} else {
		var n int64
		if e := db.Table("order_items").Where("id=? AND corp_id=?", *d.OrderItemID, corp).Count(&n).Error; e != nil || n != 1 {
			c.JSON(404, gin.H{"error": "零件不存在"})
			return
		}
	}
	path, e := h.store.Path(d.StorageKey)
	if e != nil {
		drawingError(c, e)
		return
	}
	file, e := os.Open(path)
	if e != nil {
		c.JSON(404, gin.H{"error": "图纸文件缺失，请检查文件目录或备份"})
		return
	}
	defer file.Close()
	stat, e := file.Stat()
	if e != nil {
		drawingError(c, e)
		return
	}
	c.Header("Cache-Control", "private, no-store")
	c.Header("X-Content-Type-Options", "nosniff")
	c.Header("Content-Type", d.MimeType)
	mode := "inline"
	if c.Query("download") == "1" || d.MimeType == "application/octet-stream" {
		mode = "attachment"
	}
	c.Header("Content-Disposition", mime.FormatMediaType(mode, map[string]string{"filename": d.FileName}))
	http.ServeContent(c.Writer, c.Request, d.FileName, stat.ModTime(), file)
}
