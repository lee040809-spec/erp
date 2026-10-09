package repository

import (
	"machining-erp/internal/models"
	"reflect"
	"testing"
	"time"
)

func TestSortOrderItemsByBusinessDate(t *testing.T) {
	date := func(s string) *time.Time {
		v, e := time.Parse("2006-01-02", s)
		if e != nil {
			t.Fatal(e)
		}
		return &v
	}
	for _, sample := range []struct {
		name   string
		parent *time.Time
		want   []int64
	}{
		{"parent fallback", date("2026-10-01"), []int64{3, 2, 4, 1}},
		{"missing dates last", nil, []int64{3, 2, 1, 4}},
	} {
		t.Run(sample.name, func(t *testing.T) {
			items := []models.OrderItem{{ID: 1, StartDate: date("2026-09-30")}, {ID: 4}, {ID: 2, StartDate: date("2026-10-02")}, {ID: 3, StartDate: date("2026-10-02")}}
			sortOrderItemsByDate(items, sample.parent)
			ids := []int64{}
			for _, item := range items {
				ids = append(ids, item.ID)
			}
			if !reflect.DeepEqual(ids, sample.want) {
				t.Fatalf("got %v want %v", ids, sample.want)
			}
		})
	}
}
