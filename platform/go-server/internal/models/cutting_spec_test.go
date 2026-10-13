package models

import (
	"strings"
	"testing"
)

func TestCuttingSpecification(t *testing.T) {
	value := "  Φ30 × 80 mm  "
	p := OrderProcess{CuttingSpec: &value}
	if NormalizeCuttingSpec(&p) != nil || *p.CuttingSpec != "Φ30 × 80 mm" {
		t.Fatal("trim specification but retain units")
	}
	blank := "   "
	p.CuttingSpec = &blank
	if NormalizeCuttingSpec(&p) != nil || p.CuttingSpec == nil || *p.CuttingSpec != "" {
		t.Fatal("explicit clearing must differ from omission")
	}
	p.CuttingSpec = nil
	if NormalizeCuttingSpec(&p) != nil || p.CuttingSpec != nil {
		t.Fatal("older client omission must remain nil")
	}
	long := strings.Repeat("料", 501)
	p.CuttingSpec = &long
	if NormalizeCuttingSpec(&p) == nil {
		t.Fatal("overlong specification must be rejected")
	}
}
