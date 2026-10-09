package handlers

import "testing"

func TestCostAmountPolicy(t *testing.T) {
	cases := []struct {
		gross, tax, cost string
		want             bool
	}{
		{"113.00", "13.00", "100.00", true}, {"113", "13", "113", true},
		{"0.30", "0.10", "0.20", true}, {"113", "13", "110", false},
		{"100", "101", "100", false}, {"0", "0", "0", false},
		{"1.001", "0", "1.001", false}, {"100", "-1", "100", false},
		{"10000000000000000", "0", "10000000000000000", false},
	}
	for _, c := range cases {
		if got := validCostAmounts(c.gross, c.tax, c.cost); got != c.want {
			t.Errorf("%+v got %v", c, got)
		}
	}
}

func TestHistoricalPriceComparison(t *testing.T) {
	if !sameDecimal("20.0000", "20") || sameDecimal("20", "20.01") || sameDecimal("20", "") {
		t.Fatal("price snapshot comparison failed")
	}
	if positiveDecimal("0") || positiveDecimal("-1") || positiveDecimal("") || !positiveDecimal("0.0001") {
		t.Fatal("positive price validation failed")
	}
}

func TestCrossMonthAllocationConservation(t *testing.T) {
	if !allocationFits("100.00", "40.00", "60.00") || allocationFits("100.00", "40.00", "60.01") || allocationFits("100.00", "100.00", "0.01") {
		t.Fatal("allocation must conserve original cost")
	}
	if !allocationFits("0.30", "0.10", "0.20") || allocationFits("1", "-1", "1") {
		t.Fatal("decimal allocation validation failed")
	}
	if decimalSubtract("120.00", "80.01") != "39.99" || decimalAdd("0.10", "0.20") != "0.30" {
		t.Fatal("decimal profit precision failed")
	}
}
