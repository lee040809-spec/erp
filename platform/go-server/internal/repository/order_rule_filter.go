package repository

import (
	"machining-erp/internal/models"
	"machining-erp/pkg/utils"
	"strings"
)

func (r *OrderRepository) getOrderIDsByRuleType(corpID int, ruleType string) ([]int64, error) {
	var rules []models.AdventRule
	if e := r.db.Where("corp_id=? AND ruleType=?", corpID, ruleType).Find(&rules).Error; e != nil {
		return nil, e
	}
	converter := utils.NewRuleSQLConverter()
	conditions := []string{}
	statusFlags := map[string]string{"pending": "has_pending", "processing": "has_processing", "completed": "has_completed", "delivered": "has_delivered"}
	for _, rule := range rules {
		// A specific rule is a draft until explicit order bindings are implemented.
		// The rule name is a label, never an order/customer selector.
		if rule.ScopeType != "general" {
			continue
		}
		condition := strings.TrimPrefix(converter.ConvertToSQL(rule.Formula), "HAVING ")
		if condition == "" {
			continue
		}
		if rule.TargetStatus != "" {
			flag, ok := statusFlags[rule.TargetStatus]
			if !ok {
				continue
			}
			condition += " AND t." + flag + "=1"
		}
		conditions = append(conditions, "("+condition+")")
	}
	if len(conditions) == 0 {
		return []int64{}, nil
	}
	// Aggregate item dates and statuses once; works with connection-local temporary tables too.
	query :=
		"SELECT t.id FROM(SELECT o.id,p.has_pending,p.has_processing,p.has_completed,p.has_delivered," +
			"DATEDIFF(CURDATE(),COALESCE(o.start_date,o.created_at))*1.0/NULLIF(DATEDIFF(p.due_date,COALESCE(o.start_date,o.created_at)),0) AS progress " +
			"FROM orders o JOIN(SELECT order_id,corp_id,MAX(due_date) AS due_date,MAX(status='pending') AS has_pending,MAX(status='processing') AS has_processing,MAX(status='completed') AS has_completed,MAX(status='delivered') AS has_delivered FROM order_items WHERE corp_id=? GROUP BY order_id,corp_id) p ON p.order_id=o.id AND p.corp_id=o.corp_id " +
			"WHERE o.corp_id=? AND o.status NOT IN('completed','delivered')) t WHERE " + strings.Join(conditions, " OR ")
	ids := []int64{}
	e := r.db.Raw(query, corpID, corpID).Scan(&ids).Error
	return ids, e
}
