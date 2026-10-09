package utils

import (
	"strings"
	"testing"
)

func TestRuleSQLConverterBounds(t *testing.T) {
	expression := "(({当天}-{订单日期})/({订单交期}-{订单日期}))"
	if got := NewRuleSQLConverter().ConvertToSQL("0<=(({当天}-{订单日期})/({交货日期}-{订单日期}))<=1"); got != "HAVING 0 <= progress AND progress <= 1" {
		t.Fatal("delivery date alias failed", got)
	}
	c := NewRuleSQLConverter()
	for _, tt := range []struct{ formula, want string }{
		{"1/3<=" + expression + "<=1", "HAVING 1/3 <= progress AND progress <= 1"},
		{".75>=" + expression + ">=.25", "HAVING .75 >= progress AND progress >= .25"},
		{"-1 <= " + expression + " <= 0.5", "HAVING -1 <= progress AND progress <= 0.5"},
		{"0 OR 1=1<=" + expression + "<=1", ""},
		{"0<=" + expression + "<=1 UNION SELECT 1", ""},
		{"1/0<=" + expression + "<=1", ""},
		{"0<=" + expression + ">=1", ""},
		{"0<=evil" + expression + "<=1", ""},
		{"0<=" + expression + "evil<=1", ""},
		{"0<=(({当天}+{订单日期})/({订单交期}-{订单日期}))<=1", ""},
		{strings.Repeat("9", 400) + "<=" + expression + "<=1", ""},
	} {
		t.Run(tt.formula, func(t *testing.T) {
			if got := c.ConvertToSQL(tt.formula); got != tt.want {
				t.Fatalf("got %q want %q", got, tt.want)
			}
		})
	}
	sql := c.BuildRuleBasedQuery(37, "warning", []string{".75>=" + expression + ">=.25"})
	if !strings.Contains(sql, "o.corp_id=37") || !strings.Contains(sql, ".75 >= progress") {
		t.Fatal("company or descending bounds missing")
	}
}
