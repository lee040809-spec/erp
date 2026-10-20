package handlers

import (
	"encoding/json"
	"errors"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"machining-erp/internal/models"
	"machining-erp/internal/productionflow"
	"math/big"
	"strconv"
	"time"
)

func correctCompletion(tx *gorm.DB, item models.OrderItem, id int64, req recordCorrectionRequest, date time.Time, closed map[string]bool, audit *models.ProductionRecordChange) error {
	rows := []models.ProductionCompletion{}
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("corp_id=? AND order_item_id=? AND status='confirmed'", item.CorpID, item.ID).Order("id").Find(&rows).Error; err != nil {
		return err
	}
	target := -1
	for i := range rows {
		if rows[i].ID == id {
			target = i
		}
	}
	if target < 0 {
		return errors.New("合格记录不存在或已撤销")
	}
	old := rows[target]
	if err := matchesCorrection(req, old.QualifiedQuantity, old.BusinessDate.Format("2006-01-02")); err != nil {
		return err
	}
	audit.BeforeQuantity, audit.BeforeDate = old.QualifiedQuantity, old.BusinessDate.Format("2006-01-02")
	before, _ := json.Marshal(rows)
	audit.BeforeJSON = string(before)
	rows[target].QualifiedQuantity = strconv.Itoa(req.Quantity)
	rows[target].BusinessDate = date
	remaining := new(big.Rat).SetInt64(int64(item.Quantity))
	total := new(big.Rat)
	for i := range rows {
		row := &rows[i]
		q, ok := new(big.Rat).SetString(row.QualifiedQuantity)
		if !ok || q.Sign() <= 0 {
			return errors.New("原合格数量无效，请先核对历史记录")
		}
		total.Add(total, q)
		billable := new(big.Rat)
		if remaining.Sign() > 0 {
			billable.Set(q)
			if billable.Cmp(remaining) > 0 {
				billable.Set(remaining)
			}
			remaining.Sub(remaining, billable)
		}
		nextBillable := billable.FloatString(3)
		if i != target && sameDecimal(row.BillableQuantity, nextBillable) {
			continue
		}
		oldMonth := row.BusinessDate.Format("2006-01")
		if i == target {
			oldMonth = old.BusinessDate.Format("2006-01")
		}
		newMonth := row.BusinessDate.Format("2006-01")
		for _, m := range []string{oldMonth, newMonth} {
			value, locked := closed[m]
			if !locked {
				return errors.New("记录所属月份已变化，请刷新后重试")
			}
			if value {
				return errors.New(m + " 已结账，请先由管理员处理该月份，不能修改已核对金额")
			}
		}
		row.BillableQuantity = nextBillable
		if err := tx.Model(&models.ProductionCompletion{}).Where("id=? AND corp_id=?", row.ID, item.CorpID).Updates(map[string]interface{}{"qualified_quantity": row.QualifiedQuantity, "billable_quantity": nextBillable, "business_date": row.BusinessDate, "output_value": gorm.Expr("ROUND(CAST(? AS DECIMAL(15,3))*CAST(? AS DECIMAL(15,4)),2)", nextBillable, row.UnitPriceSnapshot)}).Error; err != nil {
			return err
		}
	}
	if total.Cmp(new(big.Rat).SetInt64(int64(item.ProductionQuantity-item.ScrapQuantity))) > 0 {
		return errors.New("修改后累计合格数量超过投产数量减去报废数量")
	}
	delivered := 0
	if item.DeliveredQty != nil {
		delivered = *item.DeliveredQty
	}
	if total.Cmp(new(big.Rat).SetInt64(int64(delivered))) < 0 {
		return errors.New("修改后合格数量少于已交货数量，请先在登记交货中更正错误记录")
	}
	if err := productionflow.ValidateCompletionLedger(tx, item); err != nil {
		return err
	}
	if err := validateRecordLedgers(tx, item); err != nil {
		return err
	}
	fulfilled, err := orderFulfilledDate(tx, item.CorpID, item.ID, item.Quantity)
	if err != nil {
		return err
	}
	if err = tx.Table("order_items").Where("id=? AND corp_id=?", item.ID, item.CorpID).Update("completion_date", fulfilled).Error; err != nil {
		return err
	}
	qf, _ := total.Float64()
	if err = syncDeliveryStatus(tx, item.CorpID, item.OrderID, item.ID, delivered, item.Quantity, qf); err != nil {
		return err
	}
	// Read the persisted DECIMAL values for the audit, including any reserve/billable redistribution.
	after := []models.ProductionCompletion{}
	if err = tx.Where("corp_id=? AND order_item_id=? AND status='confirmed'", item.CorpID, item.ID).Order("id").Find(&after).Error; err != nil {
		return err
	}
	data, _ := json.Marshal(after)
	audit.AfterJSON = string(data)
	return nil
}
