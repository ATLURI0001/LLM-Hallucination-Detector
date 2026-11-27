package engine

import "testing"

func TestCitationFormatContract(t *testing.T) {
	c := NewCitationFormatContract()
	if err := c.Validate("[1]"); err != nil {
		t.Fatal(err)
	}
	if err := c.Validate("1"); err == nil {
		t.Fatal("expected error for unbracketed citation")
	}
}
