package cli

import (
	"encoding/json"
	"testing"
)

func TestFlexibleFloatAcceptsNumber(t *testing.T) {
	var value flexibleFloat
	if err := json.Unmarshal([]byte(`12.34`), &value); err != nil {
		t.Fatal(err)
	}
	if float64(value) != 12.34 {
		t.Fatalf("expected 12.34, got %v", value)
	}
}

func TestFlexibleFloatAcceptsString(t *testing.T) {
	var value flexibleFloat
	if err := json.Unmarshal([]byte(`"12.34"`), &value); err != nil {
		t.Fatal(err)
	}
	if float64(value) != 12.34 {
		t.Fatalf("expected 12.34, got %v", value)
	}
}

