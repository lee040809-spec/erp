package outsourcing

import (
	"errors"
	"gorm.io/gorm"
	"machining-erp/internal/models"
	"machining-erp/internal/productionflow"
	"math/big"
	"regexp"
	"strconv"
	"strings"
	"time"
)

func decimal(v string, scale int) bool {
	return regexp.MustCompile(`^\d{1,11}(\.\d{1,` + strconv.Itoa(scale) + `})?$`).MatchString(v)
}
func equalMoney(a, b string) bool {
	x, ok := new(big.Rat).SetString(a)
	y, yes := new(big.Rat).SetString(b)
	return ok && yes && x.Cmp(y) == 0
}

type SettlementInput struct {
	Amount    string `json:"amount"`
	Tax       string `json:"tax_amount"`
	Net       string `json:"net_amount"`
	Date      string `json:"business_date"`
	Reference string `json:"reference_no"`
	Notes     string `json:"notes"`
}

func Settle(db *gorm.DB, corp, id, actor int64, in SettlementInput) (models.OutsourcingSettlement, error) {
	var out models.OutsourcingSettlement
	date, err := productionflow.Date(in.Date, true)
	if err != nil {
		return out, err
	}
	if in.Tax == "" {
		in.Tax = "0"
	}
	if in.Net == "" {
		if a, ok := new(big.Rat).SetString(in.Amount); ok {
			if tax, yes := new(big.Rat).SetString(in.Tax); yes {
				in.Net = new(big.Rat).Sub(a, tax).FloatString(2)
			}
		}
	}
	if !decimal(in.Amount, 2) || !decimal(in.Tax, 2) || !decimal(in.Net, 2) || !productionflow.Text(in.Reference, 100) || !productionflow.Text(in.Notes, 500) {
		return out, errors.New("对账金额/税额须为非负金额，最多两位小数")
	}
	amount, _ := new(big.Rat).SetString(in.Amount)
	tax, _ := new(big.Rat).SetString(in.Tax)
	net, _ := new(big.Rat).SetString(in.Net)
	if new(big.Rat).Add(tax, net).Cmp(amount) != 0 {
		return out, errors.New("含税金额须等于税额加不含税金额")
	}
	err = db.Transaction(func(tx *gorm.DB) error {
		// Finance month before order/item locks, matching completion and month-closing writers.
		month := date.Format("2006-01")
		if e := tx.Exec("INSERT IGNORE INTO monthly_closings(corp_id,accounting_month,status,rule_version) VALUES (?,?,'open','v1')", corp, month).Error; e != nil {
			return e
		}
		var closing string
		if e := tx.Raw("SELECT status FROM monthly_closings WHERE corp_id=? AND accounting_month=? FOR UPDATE", corp, month).Scan(&closing).Error; e != nil {
			return e
		}
		if closing == "closed" {
			return errors.New("费用月份已结账，不能对账入账")
		}
		_, d, e := lock(tx, corp, id)
		if e != nil {
			return e
		}
		e = tx.Where("corp_id=? AND dispatch_id=?", corp, id).Take(&out).Error
		if e == nil {
			if !equalMoney(out.Amount, in.Amount) || !equalMoney(out.TaxAmount, in.Tax) || !equalMoney(out.NetAmount, in.Net) || out.BusinessDate.Format("2006-01-02") != in.Date || out.ReferenceNo != in.Reference || out.Notes != in.Notes {
				return errors.New("该外协单已对账，不允许重复入账或覆盖")
			}
			return nil
		}
		if e != gorm.ErrRecordNotFound {
			return e
		}
		if d.OpenQuantity+d.ReworkOutQuantity+d.ReworkWaitingQuantity > 0 {
			return errors.New("尚有未回厂或待处理不合格品，完成处置后再对账")
		}
		if e = eventDate(tx, d, in.Date); e != nil {
			return e
		}
		price, _ := new(big.Rat).SetString(d.UnitPrice)
		expected := new(big.Rat).Mul(price, new(big.Rat).SetInt64(int64(d.AcceptedQuantity))).FloatString(2)
		if !equalMoney(expected, in.Amount) && strings.TrimSpace(in.Notes) == "" {
			return errors.New("对账金额与接收数量×约定单价不同，请填写调整说明")
		}
		out = models.OutsourcingSettlement{CorpID: corp, DispatchID: id, Quantity: d.AcceptedQuantity, Amount: in.Amount, TaxAmount: in.Tax, NetAmount: in.Net, BusinessDate: date, ReferenceNo: in.Reference, Notes: in.Notes, CreatedBy: actor, CreatedAt: time.Now()}
		if amount.Sign() > 0 {
			source := "outsourcing_dispatch"
			sourceID := strconv.FormatInt(id, 10)
			cost := models.CostEntry{CorpID: corp, CostType: "outsourcing", Amount: in.Amount, TaxAmount: in.Tax, NetAmount: in.Net, Currency: "CNY", BusinessDate: date, AccountingMonth: month, OrderID: &d.OrderID, OrderItemID: &d.OrderItemID, DeliveryBatchID: d.DeliveryBatchID, SupplierName: &d.SupplierName, SourceType: &source, SourceID: &sourceID, Status: "confirmed", Notes: &in.Notes, CreatedBy: actor, CreatedAt: time.Now()}
			if e = tx.Create(&cost).Error; e != nil {
				return e
			}
			out.CostEntryID = &cost.ID
		}
		if e = tx.Create(&out).Error; e != nil {
			return e
		}
		return tx.Model(&d).Update("status", "settled").Error
	})
	return out, err
}
