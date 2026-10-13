package models

import (
	"errors"
	"strings"
	"unicode/utf8"
)

// Nil means omitted by an older client; an explicit empty string clears the specification.
func NormalizeCuttingSpec(p *OrderProcess) error {
	if p.CuttingSpec == nil {
		return nil
	}
	value := strings.TrimSpace(*p.CuttingSpec)
	if utf8.RuneCountInString(value) > 500 {
		return errors.New("下料尺寸规格最多500个字")
	}
	p.CuttingSpec = &value
	return nil
}
func SameCuttingSpec(a, b *string) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return *a == *b
}
