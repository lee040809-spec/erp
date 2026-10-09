package services

import (
	"fmt"
	"strings"
)

// RuleSQLConverter 将规则公式转换为SQL条件
type RuleSQLConverter struct{}

// NewRuleSQLConverter 创建SQL规则转换器
func NewRuleSQLConverter() *RuleSQLConverter {
	return &RuleSQLConverter{}
}

// ConvertToSQL 将规则公式转换为SQL HAVING条件
// formula: 1/3<=(({当天}-{订单日期})/({订单交期}-{订单日期}))<=1
// 返回: HAVING子句
func (c *RuleSQLConverter) ConvertToSQL(formula string) string {
	formula = strings.TrimSpace(formula)

	// 尝试解析进度公式: 左值<=进度表达式<=右值
	if result := c.parseProgressFormula(formula); result != nil {
		return result.ToSQLCondition()
	}

	return ""
}

// ProgressFormula 进度公式结构
type ProgressFormula struct {
	LeftVal  string
	Op1      string
	Op2      string
	RightVal string
}

// ToSQLCondition 生成SQL HAVING条件
func (p *ProgressFormula) ToSQLCondition() string {
	// 进度 = (今天 - 订单日期) / (交期 - 订单日期)
	// 使用SELECT中的别名
	progressSQL := "progress"

	if p.Op1 == "<=" && p.Op2 == "<=" {
		return fmt.Sprintf("HAVING %s <= %s AND %s <= %s", p.LeftVal, progressSQL, progressSQL, p.RightVal)
	}
	if p.Op1 == ">=" && p.Op2 == ">=" {
		return fmt.Sprintf("HAVING %s >= %s AND %s >= %s", p.LeftVal, progressSQL, progressSQL, p.RightVal)
	}
	return ""
}

// parseProgressFormula 解析进度公式
func (c *RuleSQLConverter) parseProgressFormula(formula string) *ProgressFormula {
	// 查找第一个 <= 或 >=
	idx1 := c.findOperator(formula, 0)
	if idx1 == -1 {
		return nil
	}
	leftVal := formula[:idx1]
	op1 := formula[idx1 : idx1+2]

	// 找到进度表达式的开始（第一个 (）
	exprStart := idx1 + 2
	for exprStart < len(formula) && formula[exprStart] != '(' {
		exprStart++
	}
	if exprStart >= len(formula) {
		return nil
	}

	// 找到进度表达式的结束位置（匹配括号）
	exprEnd := c.findMatchingParen(formula, exprStart)
	if exprEnd == -1 {
		return nil
	}

	// 验证进度表达式格式: (({当天}-{订单日期})/({订单交期}-{订单日期}))
	expr := formula[exprStart : exprEnd+1]
	if !c.isProgressExpression(expr) {
		return nil
	}

	// 找第二个操作符
	idx2 := c.findOperator(formula, exprEnd+1)
	if idx2 == -1 {
		return nil
	}
	op2 := formula[idx2 : idx2+2]
	rightVal := formula[idx2+2:]

	return &ProgressFormula{
		LeftVal:  strings.TrimSpace(leftVal),
		Op1:      op1,
		Op2:      op2,
		RightVal: strings.TrimSpace(rightVal),
	}
}

// isProgressExpression 检查是否为进度表达式
func (c *RuleSQLConverter) isProgressExpression(expr string) bool {
	// 简化检查：包含{当天}、{订单日期}、{订单交期}
	return strings.Contains(expr, "{当天}") &&
		strings.Contains(expr, "{订单日期}") &&
		strings.Contains(expr, "{订单交期}")
}

// findOperator 查找操作符位置
func (c *RuleSQLConverter) findOperator(s string, start int) int {
	for i := start; i < len(s)-1; i++ {
		if (s[i] == '<' || s[i] == '>') && s[i+1] == '=' {
			return i
		}
	}
	return -1
}

// findMatchingParen 找到匹配的右括号
func (c *RuleSQLConverter) findMatchingParen(s string, start int) int {
	depth := 0
	for i := start; i < len(s); i++ {
		if s[i] == '(' {
			depth++
		}
		if s[i] == ')' {
			depth--
		}
		if depth == 0 {
			return i
		}
	}
	return -1
}

// BuildRuleBasedQuery 构建基于规则的订单查询SQL
func (c *RuleSQLConverter) BuildRuleBasedQuery(corpID int, ruleType string, formulas []string) string {
	if len(formulas) == 0 {
		return "SELECT 0 WHERE 1=0"
	}

	// 解析公式获取左右边界值
	var conditions []string
	for _, formula := range formulas {
		if result := c.parseProgressFormula(formula); result != nil {
			// 进度条件: 左值 <= progress <= 右值
			cond := fmt.Sprintf("(%s <= progress AND progress <= %s)", result.LeftVal, result.RightVal)
			conditions = append(conditions, cond)
		}
	}

	if len(conditions) == 0 {
		return "SELECT 0 WHERE 1=0"
	}

	// 构建SQL：使用子查询计算进度
	sql := fmt.Sprintf(`
		SELECT order_id FROM (
			SELECT o.id as order_id,
				DATEDIFF(CURDATE(), COALESCE(o.start_date, o.created_at)) * 1.0 /
				NULLIF(DATEDIFF(
					(SELECT MAX(due_date) FROM order_items WHERE order_id = o.id AND corp_id=o.corp_id AND due_date IS NOT NULL),
					COALESCE(o.start_date, o.created_at)
				), 0) as progress
			FROM orders o
			WHERE o.status NOT IN ('delivered', 'completed')
			AND o.corp_id=%d
			AND EXISTS (SELECT 1 FROM order_items WHERE order_id = o.id AND corp_id=o.corp_id AND due_date IS NOT NULL)
		) t
		WHERE %s
	`, corpID, strings.Join(conditions, " OR "))

	return sql
}
