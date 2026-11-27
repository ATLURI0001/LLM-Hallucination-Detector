package engine

import "testing"

func TestHallucinationRateContract(t *testing.T) {
	c := NewHallucinationRateContract(0.1)
	if err := c.Validate(0.05); err != nil {
		t.Fatal(err)
	}
	if err := c.Validate(0.2); err == nil {
		t.Fatal("expected error for rate exceeding max")
	}
}
