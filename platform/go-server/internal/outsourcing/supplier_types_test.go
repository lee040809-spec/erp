package outsourcing

import "testing"

func TestSupplierTypes(t *testing.T) {
	for _, value := range []string{"", "material,", "invalid", "outsourcing,unknown"} {
		if _, err := NormalizeSupplierTypes(value); err == nil {
			t.Fatalf("accepted invalid types %q", value)
		}
	}
	value, err := NormalizeSupplierTypes(" material,outsourcing, material,other ")
	if err != nil || value != "outsourcing,material,other" {
		t.Fatalf("unexpected canonical set %q: %v", value, err)
	}
	if !HasSupplierType(value, "material") || HasSupplierType("material", "outsourcing") || HasSupplierType(value, "out") {
		t.Fatal("type membership must match whole type")
	}
}
