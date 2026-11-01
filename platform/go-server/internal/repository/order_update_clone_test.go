package repository

import (
	"machining-erp/internal/models"
	"testing"
)

func TestCloneUpdateItemsForTransactionRetry(t *testing.T) {
	original := []models.OrderItem{{PartName: "new", Processes: []models.OrderProcess{{Name: "加工"}}, DrawingIDs: []int64{7}, DeliveryPlan: []models.DeliveryBatchInput{{Quantity: 10}}}}
	attempt := cloneUpdateItems(original)
	attempt[0].ID = 100
	attempt[0].Processes[0].ID = 200
	attempt[0].DrawingIDs[0] = 8
	attempt[0].DeliveryPlan[0].Quantity = 5
	if original[0].ID != 0 || original[0].Processes[0].ID != 0 || original[0].DrawingIDs[0] != 7 || original[0].DeliveryPlan[0].Quantity != 10 {
		t.Fatal("a rolled-back attempt mutated the next attempt's input")
	}
	if cloneUpdateItems(nil) != nil || cloneUpdateItems([]models.OrderItem{}) == nil {
		t.Fatal("nil and empty update scopes must remain distinct")
	}
}
