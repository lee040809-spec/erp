package outsourcing

import (
	"errors"
	"strings"
)

var supplierTypeOrder = []string{"outsourcing", "material", "tooling", "fixture", "other"}

func ValidSupplierType(value string) bool {
	for _, item := range supplierTypeOrder {
		if item == value {
			return true
		}
	}
	return false
}

// A canonical bounded set allows one supplier to serve several purposes.
func NormalizeSupplierTypes(value string) (string, error) {
	selected := map[string]bool{}
	for _, item := range strings.Split(value, ",") {
		item = strings.TrimSpace(item)
		if !ValidSupplierType(item) {
			return "", errors.New("请至少选择一种有效的供应商类型")
		}
		selected[item] = true
	}
	result := []string{}
	for _, item := range supplierTypeOrder {
		if selected[item] {
			result = append(result, item)
		}
	}
	return strings.Join(result, ","), nil
}

func HasSupplierType(value, kind string) bool {
	for _, item := range strings.Split(value, ",") {
		if item == kind {
			return true
		}
	}
	return false
}
