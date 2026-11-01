package accounting

import (
	"errors"
	"gorm.io/gorm"
	"math/big"
)

// These limits match the existing DECIMAL columns; validation must not depend on
// a server's strict SQL mode silently clamping an overflowing amount.
func ValidateOrderAmounts(tx *gorm.DB, corp, order int64, overrides map[int64]string) error {
	var rows []struct {
		ID       int64
		Quantity int64
		Price    string
	}
	if err := tx.Raw("SELECT id,quantity,CAST(COALESCE(unit_price,0) AS CHAR) AS price FROM order_items WHERE corp_id=? AND order_id=? ORDER BY id FOR UPDATE", corp, order).Scan(&rows).Error; err != nil {
		return err
	}
	total := new(big.Rat)
	for _, row := range rows {
		price := row.Price
		if value, exists := overrides[row.ID]; exists {
			price = value
		}
		p, ok := new(big.Rat).SetString(price)
		if !ok || p.Sign() < 0 || row.Quantity <= 0 {
			return errors.New("零件数量或单价格式无效")
		}
		amount := new(big.Rat).Mul(p, new(big.Rat).SetInt64(row.Quantity))
		rounded, _ := new(big.Rat).SetString(amount.FloatString(2))
		limit, _ := new(big.Rat).SetString("99999999.99")
		if rounded.Cmp(limit) > 0 {
			return errors.New("零件总金额超过系统可保存上限99999999.99，请核对数量与单价")
		}
		total.Add(total, rounded)
	}
	limit, _ := new(big.Rat).SetString("9999999999.99")
	if total.Cmp(limit) > 0 {
		return errors.New("订单总金额超过系统可保存上限9999999999.99")
	}
	return nil
}

func OutputAmountFits(quantity int64, price string) bool {
	p, ok := new(big.Rat).SetString(price)
	if !ok || p.Sign() < 0 || quantity < 0 {
		return false
	}
	amount := new(big.Rat).Mul(p, new(big.Rat).SetInt64(quantity))
	rounded, _ := new(big.Rat).SetString(amount.FloatString(2))
	limit, _ := new(big.Rat).SetString("9999999999999999.99")
	return rounded.Cmp(limit) <= 0
}
