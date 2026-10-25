package accounting

import (
	_ "embed"
	"strings"
)

// schema.sql is the sole source used by migration tools and temporary-table tests.
//
//go:embed schema.sql
var schema string
var PricingColumn string
var Tables []string

func init() {
	for _, s := range strings.Split(schema, ";") {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		if PricingColumn == "" {
			PricingColumn = s
		} else {
			Tables = append(Tables, s)
		}
	}
}
