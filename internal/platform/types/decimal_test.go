package types

import (
	"testing"
)

func TestDecimal(t *testing.T) {
	d, err := NewDecimalFromString("123.45")
	if err != nil {
		t.Fatalf("unexpected error parsing decimal: %v", err)
	}

	if d.StringFixed(2) != "123.45" {
		t.Errorf("expected 123.45, got %s", d.StringFixed(2))
	}
}
