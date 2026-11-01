package handlers

import (
	"errors"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"machining-erp/internal/accounting"
	"machining-erp/internal/middleware"
	"machining-erp/internal/productionflow"
	"machining-erp/pkg/response"
	"regexp"
	"strconv"
)

func (h *ProductionFinanceHandler) PriceCompletion(c *gin.Context) {
	if !requireModuleAccess(h.db, c, "orders") {
		return
	}
	itemID, _ := strconv.ParseInt(c.Param("itemId"), 10, 64)
	var req struct {
		Price string `json:"unit_price"`
	}
	if c.ShouldBindJSON(&req) != nil || itemID <= 0 || !validPositivePrice(req.Price) {
		response.BadRequest(c, response.CodeInvalidRequest, "请填写大于0、不超过99999999.99、最多2位小数的人民币单价")
		return
	}
	corp, actor := middleware.GetCorpID(c), middleware.GetUserID(c)
	err := accounting.WithPricingMonths(h.db.WithContext(c.Request.Context()), corp, accounting.PricingScope{ItemIDs: []int64{itemID}}, func(tx *gorm.DB, locked accounting.PricingMonths) error {
		item, err := productionflow.LockItem(tx, corp, itemID)
		if err != nil {
			return err
		}
		if item.Currency != "CNY" {
			return errors.New("请先在订单资料中核对人民币结算币种")
		}
		// This action fills missing prices only; price renegotiations belong in order editing.
		var old string
		if err = tx.Raw("SELECT CAST(COALESCE(unit_price,0) AS CHAR) FROM order_items WHERE corp_id=? AND id=? FOR UPDATE", corp, itemID).Scan(&old).Error; err != nil {
			return err
		}
		if positiveDecimal(old) && !sameDecimal(old, req.Price) {
			return errors.New("该零件已有单价，请刷新；价格变更请在订单资料中核对")
		}
		if err = accounting.ValidateOrderAmounts(tx, corp, item.OrderID, map[int64]string{itemID: req.Price}); err != nil {
			return err
		}
		if !positiveDecimal(old) {
			result := tx.Exec("UPDATE order_items SET unit_price=?,currency='CNY',total_price=ROUND(quantity*CAST(? AS DECIMAL(15,4)),2) WHERE corp_id=? AND id=? AND (unit_price IS NULL OR unit_price=0)", req.Price, req.Price, corp, itemID)
			if result.Error != nil {
				return result.Error
			}
			if result.RowsAffected != 1 {
				return errors.New("单价已变化，请刷新后重试")
			}
		}
		if err = accounting.ResolvePending(tx, corp, itemID, actor, locked); err != nil {
			return err
		}
		return tx.Exec("UPDATE orders SET total_amount=(SELECT COALESCE(SUM(total_price),0) FROM order_items WHERE corp_id=? AND order_id=?) WHERE corp_id=? AND id=?", corp, item.OrderID, corp, item.OrderID).Error
	})
	if err != nil {
		response.BadRequest(c, response.CodeInvalidRequest, "补价未保存："+err.Error())
		return
	}
	response.SuccessWithMessage(c, "单价已补齐，待补价合格数量已按原完工月份计入产值", nil)
}

func validPositivePrice(value string) bool {
	return orderUnitPricePattern.MatchString(value) && positiveDecimal(value)
}

var orderUnitPricePattern = regexp.MustCompile(`^\d{1,8}(\.\d{1,2})?$`)
var unitPricePattern = regexp.MustCompile(`^\d{1,11}(\.\d{1,4})?$`)

func validNonnegativePrice(value string) bool { return unitPricePattern.MatchString(value) }
