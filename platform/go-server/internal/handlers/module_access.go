package handlers

import (
	"encoding/json"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"machining-erp/internal/middleware"
	"machining-erp/pkg/response"
)

func hasModuleAccess(db *gorm.DB, c *gin.Context, keys ...string) bool {
	allowed, _ := checkModuleAccess(db, c, keys...)
	return allowed
}
func checkModuleAccess(db *gorm.DB, c *gin.Context, keys ...string) (bool, error) {
	if middleware.GetRoleType(c) == "admin" {
		return true, nil
	}
	// Reuse one successful permission read within this request only. A new
	// request always rechecks the database, so role edits take effect immediately.
	value, loaded := c.Get("moduleAccessKeys")
	var granted map[string]bool
	if loaded {
		granted = value.(map[string]bool)
	} else {
		var resourceKeys []string
		err := db.Table("users u").Joins("JOIN permissions p ON p.role_id=u.role_id AND p.corp_id=u.corp_id").Joins("JOIN resources r ON r.id=p.resource_id").Where("u.id=? AND u.corp_id=?", middleware.GetUserID(c), middleware.GetCorpID(c)).Distinct().Pluck("r.resource_key", &resourceKeys).Error
		if err != nil {
			return false, err
		}
		granted = map[string]bool{}
		for _, key := range resourceKeys {
			granted[key] = true
		}
		c.Set("moduleAccessKeys", granted)
	}
	for _, key := range keys {
		if granted[key] {
			return true, nil
		}
	}
	return false, nil
}
func requireModuleAccess(db *gorm.DB, c *gin.Context, keys ...string) bool {
	allowed, err := checkModuleAccess(db, c, keys...)
	if err != nil {
		response.InternalError(c, response.CodeInternal, "数据库连接暂时异常，无法核验功能权限，请刷新后重试")
		return false
	}
	if allowed {
		return true
	}
	response.Forbidden(c, response.CodeForbidden, "当前账号没有此功能权限")
	return false
}
func canViewOrderAmounts(db *gorm.DB, c *gin.Context) bool {
	return hasModuleAccess(db, c, "orders", "finance", "monthly_output", "btn-view-finance-summary")
}

// Omit sensitive fields rather than returning plausible zero monetary values.
func withoutAmounts(value interface{}) interface{} {
	data, err := json.Marshal(value)
	if err != nil {
		return nil
	}
	var out interface{}
	if json.Unmarshal(data, &out) != nil {
		return nil
	}
	var scrub func(interface{})
	scrub = func(v interface{}) {
		switch row := v.(type) {
		case map[string]interface{}:
			for _, key := range []string{"total_amount", "unit_price", "total_price", "tool_cost", "fixture_cost", "material_cost", "other_cost", "outsourcing_fee", "unit_price_snapshot", "output_value", "amount", "tax_amount", "net_amount"} {
				delete(row, key)
			}
			for _, child := range row {
				scrub(child)
			}
		case []interface{}:
			for _, child := range row {
				scrub(child)
			}
		}
	}
	scrub(out)
	return out
}
