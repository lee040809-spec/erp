package services

import (
	"regexp"
	"strings"
)

// RuleEvaluator 规则公式解析器
type RuleEvaluator struct{}

// NewRuleEvaluator 创建规则解析器
func NewRuleEvaluator() *RuleEvaluator {
	return &RuleEvaluator{}
}

// Evaluate 评估规则公式
func (e *RuleEvaluator) Evaluate(formula string, orderDueDate int64, orderDate int64, today int64) bool {
	// 替换变量
	processed := formula
	processed = strings.ReplaceAll(processed, "{订单交期}", int64ToStr(orderDueDate))
	processed = strings.ReplaceAll(processed, "{订单日期}", int64ToStr(orderDate))
	processed = strings.ReplaceAll(processed, "{当天}", int64ToStr(today))

	return evaluateExpression(processed)
}

func int64ToStr(v int64) string {
	if v == 0 {
		return "0"
	}

	negative := false
	if v < 0 {
		negative = true
		v = -v
	}

	var result []byte
	for v > 0 {
		result = append([]byte{byte('0' + v%10)}, result...)
		v /= 10
	}

	if negative {
		result = append([]byte{'-'}, result...)
	}

	return string(result)
}

// evaluateExpression 计算表达式
func evaluateExpression(expr string) bool {
	expr = strings.TrimSpace(expr)

	// 处理双重比较: a <= x <= b
	if matches := doubleComparisonRegex.FindStringSubmatch(expr); len(matches) == 6 {
		leftVal := evalFloatExpr(matches[1])
		middleExpr := matches[3]
		rightVal := evalFloatExpr(matches[5])

		middleVal := evalFloatExpr(middleExpr)

		// 两个操作符必须相同
		if matches[2] == matches[4] && matches[2] == "<=" {
			return leftVal <= middleVal && middleVal <= rightVal
		}
		if matches[2] == matches[4] && matches[2] == ">=" {
			return leftVal >= middleVal && middleVal >= rightVal
		}
	}

	// 单比较
	if strings.Contains(expr, "<=") {
		parts := strings.SplitN(expr, "<=", 2)
		if len(parts) == 2 {
			return evalFloatExpr(parts[0]) <= evalFloatExpr(parts[1])
		}
	}
	if strings.Contains(expr, ">=") {
		parts := strings.SplitN(expr, ">=", 2)
		if len(parts) == 2 {
			return evalFloatExpr(parts[0]) >= evalFloatExpr(parts[1])
		}
	}
	if strings.Contains(expr, "==") {
		parts := strings.SplitN(expr, "==", 2)
		if len(parts) == 2 {
			return evalFloatExpr(parts[0]) == evalFloatExpr(parts[1])
		}
	}
	if strings.Contains(expr, "<") && !strings.Contains(expr, "<=") {
		parts := strings.SplitN(expr, "<", 2)
		if len(parts) == 2 {
			return evalFloatExpr(parts[0]) < evalFloatExpr(parts[1])
		}
	}
	if strings.Contains(expr, ">") && !strings.Contains(expr, ">=") {
		parts := strings.SplitN(expr, ">", 2)
		if len(parts) == 2 {
			return evalFloatExpr(parts[0]) > evalFloatExpr(parts[1])
		}
	}

	return false
}

// evalFloatExpr 计算浮点数表达式
func evalFloatExpr(expr string) float64 {
	expr = strings.TrimSpace(expr)

	// 去掉外层括号（循环处理多层括号）
	for strings.HasPrefix(expr, "(") && strings.HasSuffix(expr, ")") {
		depth := 0
		match := true
		for i, c := range expr {
			if c == '(' {
				depth++
			} else if c == ')' {
				depth--
			}
			if depth == 0 && i < len(expr)-1 {
				match = false
				break
			}
		}
		if match {
			expr = expr[1 : len(expr)-1]
		} else {
			break
		}
	}

	// 加减法（从右往左找，保持运算顺序）
	depth := 0
	for i := len(expr) - 1; i >= 0; i-- {
		c := expr[i]
		if c == ')' {
			depth++
		} else if c == '(' {
			depth--
		} else if depth == 0 {
			if c == '+' {
				return evalFloatExpr(expr[:i]) + evalFloatExpr(expr[i+1:])
			}
			if c == '-' && i > 0 {
				return evalFloatExpr(expr[:i]) - evalFloatExpr(expr[i+1:])
			}
		}
	}

	// 乘除法（从右往左找）
	depth = 0
	for i := len(expr) - 1; i >= 0; i-- {
		c := expr[i]
		if c == ')' {
			depth++
		} else if c == '(' {
			depth--
		} else if depth == 0 {
			if c == '*' {
				return evalFloatExpr(expr[:i]) * evalFloatExpr(expr[i+1:])
			}
			if c == '/' {
				right := evalFloatExpr(expr[i+1:])
				if right != 0 {
					return evalFloatExpr(expr[:i]) / right
				}
				return 0
			}
		}
	}

	// 纯数字
	return parseFloat(expr)
}

// parseFloat 解析浮点数
func parseFloat(s string) float64 {
	s = strings.TrimSpace(s)
	var result float64
	var decimal float64 = 0.1
	isDecimal := false
	isNegative := false

	if strings.HasPrefix(s, "-") {
		isNegative = true
		s = s[1:]
	}

	for _, c := range s {
		if c >= '0' && c <= '9' {
			if isDecimal {
				result += float64(c-'0') * decimal
				decimal *= 0.1
			} else {
				result = result*10 + float64(c-'0')
			}
		} else if c == '.' {
			isDecimal = true
		}
	}

	if isNegative {
		return -result
	}
	return result
}

// 双重比较正则: 表达式<=表达式<=表达式
// 支持: 1/3<=...<=1 或复杂表达式
var doubleComparisonRegex = regexp.MustCompile(`^(.+?)\s*(<=|>=)\s*(.+?)\s*(<=|>=)\s*(.+)$`)
