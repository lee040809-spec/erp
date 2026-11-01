package accounting

import "testing"

func TestOutputAmountFits(t *testing.T) {
	for _, row := range []struct {
		qty   int64
		price string
		want  bool
	}{
		{100, "2.50", true}, {0, "99999999999.9999", true},
		{100001, "99999999999.9999", false}, {99999, "99999999999.9999", true},
		{1, "-1", false}, {1, "invalid", false},
	} {
		if got := OutputAmountFits(row.qty, row.price); got != row.want {
			t.Errorf("%d * %s fits=%v want %v", row.qty, row.price, got, row.want)
		}
	}
}
