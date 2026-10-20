package handlers

import (
	"crypto/sha256"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"machining-erp/internal/middleware"
	"machining-erp/internal/models"
	"machining-erp/internal/productionflow"
	"machining-erp/pkg/response"
	"sort"
	"strconv"
	"strings"
	"time"
)

type recordCorrectionRequest struct {
	Quantity         int    `json:"quantity"`
	Date             string `json:"business_date"`
	Reason           string `json:"reason"`
	RequestID        string `json:"request_id"`
	ExpectedQuantity string `json:"expected_quantity"`
	ExpectedDate     string `json:"expected_date"`
	ExpectedRevision int64  `json:"expected_revision"`
}

func (h *ProductionRecordHandler) Correct(c *gin.Context) {
	kind := c.Param("kind")
	if !h.canEdit(c, kind) {
		response.Forbidden(c, response.CodeForbidden, "没有此类记录的修改权限")
		return
	}
	itemID, _ := strconv.ParseInt(c.Param("itemId"), 10, 64)
	recordID, _ := strconv.ParseInt(c.Param("recordId"), 10, 64)
	var req recordCorrectionRequest
	if c.ShouldBindJSON(&req) != nil || itemID <= 0 || recordID <= 0 || req.Quantity <= 0 || req.Quantity > 2147483647 || !productionflow.RequestID(req.RequestID) {
		response.BadRequest(c, response.CodeInvalidRequest, "请填写有效数量和修改请求编号")
		return
	}
	req.Reason = strings.TrimSpace(req.Reason)
	if req.Reason == "" || !productionflow.Text(req.Reason, 500) {
		response.BadRequest(c, response.CodeInvalidRequest, "修改原因必填，最多500字")
		return
	}
	date, err := productionflow.Date(req.Date, true)
	if err != nil {
		response.BadRequest(c, response.CodeInvalidRequest, err.Error())
		return
	}
	corp, actor := middleware.GetCorpID(c), middleware.GetUserID(c)
	raw, _ := json.Marshal([]interface{}{itemID, kind, recordID, req})
	hash := fmt.Sprintf("%x", sha256.Sum256(raw))
	err = h.db.Transaction(func(tx *gorm.DB) error {
		// Finance months precede order/item locks, matching completion creation and cost posting.
		// Lock all currently represented months; only changed months are required to be open.
		months, err := correctionMonths(tx, corp, itemID, kind, req.Date)
		if err != nil {
			return err
		}
		closed, err := lockCorrectionMonths(tx, corp, months)
		if err != nil {
			return err
		}
		item, err := productionflow.LockItem(tx, corp, itemID)
		if err != nil {
			return err
		}
		var replay models.ProductionRecordChange
		err = tx.Where("corp_id=? AND request_id=?", corp, req.RequestID).Take(&replay).Error
		if err == nil {
			if replay.RequestHash != hash {
				return errors.New("请求编号已使用，修改内容不一致")
			}
			return nil
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		var revision int64
		if err = tx.Model(&models.ProductionRecordChange{}).Where("corp_id=? AND order_item_id=?", corp, itemID).Select("COALESCE(MAX(id),0)").Scan(&revision).Error; err != nil {
			return err
		}
		if revision != req.ExpectedRevision {
			return errors.New("记录已被其他操作修改，请刷新记录后重新填写")
		}
		audit := models.ProductionRecordChange{CorpID: corp, OrderItemID: itemID, Kind: kind, RecordID: recordID, RequestID: req.RequestID, RequestHash: hash, Reason: req.Reason, AfterQuantity: strconv.Itoa(req.Quantity), AfterDate: req.Date, CreatedBy: actor, CreatedAt: time.Now()}
		if kind == "completion" {
			err = correctCompletion(tx, item, recordID, req, date, closed, &audit)
		} else {
			err = correctDelivery(tx, item, recordID, req, date, &audit)
		}
		if err != nil {
			return err
		}
		return tx.Create(&audit).Error
	}, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		response.BadRequest(c, response.CodeInvalidRequest, "修改未保存："+err.Error())
		return
	}
	response.SuccessWithMessage(c, "已修改，数量、日期和相关统计已同步，修改原因已保留", nil)
}
func correctionMonths(tx *gorm.DB, corp, item int64, kind, date string) ([]string, error) {
	months := []string{}
	if kind == "completion" {
		if err := tx.Table("production_completions").Where("corp_id=? AND order_item_id=? AND status='confirmed'", corp, item).Distinct("DATE_FORMAT(business_date,'%Y-%m')").Pluck("DATE_FORMAT(business_date,'%Y-%m')", &months).Error; err != nil {
			return nil, err
		}
		months = append(months, date[:7])
	}
	set := map[string]bool{}
	for _, m := range months {
		set[m] = true
	}
	months = nil
	for m := range set {
		months = append(months, m)
	}
	sort.Strings(months)
	return months, nil
}
func lockCorrectionMonths(tx *gorm.DB, corp int64, months []string) (map[string]bool, error) {
	closed := map[string]bool{}
	for _, m := range months {
		if err := tx.Exec("INSERT IGNORE INTO monthly_closings (corp_id,accounting_month,status,rule_version) VALUES (?,?,'open','v1')", corp, m).Error; err != nil {
			return nil, err
		}
		var row struct{ Status string }
		if err := tx.Raw("SELECT status FROM monthly_closings WHERE corp_id=? AND accounting_month=? FOR UPDATE", corp, m).Scan(&row).Error; err != nil {
			return nil, err
		}
		closed[m] = row.Status == "closed"
	}
	return closed, nil
}
func matchesCorrection(req recordCorrectionRequest, quantity, date string) error {
	if !sameDecimal(req.ExpectedQuantity, quantity) || req.ExpectedDate != date {
		return errors.New("原数量或日期已变化，请刷新后重试")
	}
	if sameDecimal(strconv.Itoa(req.Quantity), quantity) && req.Date == date {
		return errors.New("数量和日期没有变化")
	}
	return nil
}
func correctDelivery(tx *gorm.DB, item models.OrderItem, id int64, req recordCorrectionRequest, date time.Time, audit *models.ProductionRecordChange) error {
	var row BusinessRecord
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id=? AND corp_id=? AND order_item_id=? AND kind='delivery' AND status='confirmed'", id, item.CorpID, item.ID).Take(&row).Error; err != nil {
		return errors.New("交货记录不存在或已撤销")
	}
	// A legacy aggregate may have an unknown date. Only an explicit, audited correction
	// can date it, and the existing aggregate must still match its full source ledger.
	oldDate := ""
	if row.BusinessDate != nil {
		oldDate = row.BusinessDate.Format("2006-01-02")
	}
	if err := matchesCorrection(req, strconv.Itoa(row.Quantity), oldDate); err != nil {
		return err
	}
	var sum int
	if err := tx.Model(&BusinessRecord{}).Where("corp_id=? AND order_item_id=? AND kind='delivery' AND status='confirmed'", item.CorpID, item.ID).Select("COALESCE(SUM(quantity),0)").Scan(&sum).Error; err != nil {
		return err
	}
	old := 0
	if item.DeliveredQty != nil {
		old = *item.DeliveredQty
	}
	if old != sum {
		return errors.New("已交货汇总与原记录不一致，请先核对历史数据")
	}
	audit.BeforeQuantity, audit.BeforeDate = strconv.Itoa(row.Quantity), oldDate
	before, _ := json.Marshal(row)
	audit.BeforeJSON = string(before)
	if err := tx.Model(&BusinessRecord{}).Where("id=? AND corp_id=?", id, item.CorpID).Updates(map[string]interface{}{"quantity": req.Quantity, "business_date": date}).Error; err != nil {
		return err
	}
	row.Quantity, row.BusinessDate = req.Quantity, &date
	after, _ := json.Marshal(row)
	audit.AfterJSON = string(after)
	total := old - atoiQuantity(audit.BeforeQuantity) + req.Quantity
	var qualified float64
	if err := tx.Table("production_completions").Where("corp_id=? AND order_item_id=? AND status='confirmed'", item.CorpID, item.ID).Select("COALESCE(SUM(qualified_quantity),0)").Scan(&qualified).Error; err != nil {
		return err
	}
	if total > item.Quantity || float64(total) > qualified {
		return errors.New("修改后累计交货超过订单数量或已确认合格数量")
	}
	item.DeliveredQty = &total
	if err := validateRecordLedgers(tx, item); err != nil {
		return err
	}
	if err := tx.Table("order_items").Where("id=? AND corp_id=?", item.ID, item.CorpID).Update("delivered_quantity", total).Error; err != nil {
		return err
	}
	return syncDeliveryStatus(tx, item.CorpID, item.OrderID, item.ID, total, item.Quantity, qualified)
}
func atoiQuantity(s string) int { v, _ := strconv.Atoi(s); return v }

// Replay both total and per-batch ledgers after the proposed edit, before committing anything.
func validateRecordLedgers(tx *gorm.DB, item models.OrderItem) error {
	var delivered int
	if err := tx.Table("order_business_records").Where("corp_id=? AND order_item_id=? AND kind='delivery' AND status='confirmed'", item.CorpID, item.ID).Select("COALESCE(SUM(quantity),0)").Scan(&delivered).Error; err != nil {
		return err
	}
	summary := 0
	if item.DeliveredQty != nil {
		summary = *item.DeliveredQty
	}
	if summary != delivered {
		return errors.New("已交货汇总与原记录不一致，请先核对历史数据")
	}
	valid, err := deliveryTimelineFits(tx, item.CorpID, item.ID, time.Now(), 0)
	if err != nil {
		return err
	}
	if !valid {
		return errors.New("修改会导致某次交货早于合格完工或交货数量不足，请先更正对应的交货记录")
	}
	var batches []models.DeliveryBatch
	if err = tx.Where("corp_id=? AND order_item_id=?", item.CorpID, item.ID).Find(&batches).Error; err != nil {
		return err
	}
	for _, b := range batches {
		var q float64
		if err = tx.Table("production_completions").Where("corp_id=? AND delivery_batch_id=? AND status='confirmed'", item.CorpID, b.ID).Select("COALESCE(SUM(qualified_quantity),0)").Scan(&q).Error; err != nil {
			return err
		}
		if q > float64(b.ProductionQuantity) {
			return errors.New("修改后合格数量超过该交货批次投产数量")
		}
		if err = productionflow.ValidateDelivery(tx, item.CorpID, item.ID, &b.ID, 0, productionflow.Today()); err != nil {
			return err
		}
	}
	return nil
}
