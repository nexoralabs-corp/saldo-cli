package cli

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoanStatusNormalizesAndRejectsUnknownValues(t *testing.T) {
	for _, value := range []string{"", "active", "ARCHIVED", "all"} {
		if _, err := loanStatus(value); err != nil {
			t.Fatalf("loanStatus(%q): %v", value, err)
		}
	}
	if _, err := loanStatus("paid"); err == nil {
		t.Fatal("expected invalid status error")
	}
}

func TestLoanPaymentValidationRequiresCompleteCrossCurrencyFields(t *testing.T) {
	if err := validatePaymentAmounts(0, 0, 0); err != nil {
		t.Fatalf("same-currency payment should be valid: %v", err)
	}
	if err := validatePaymentAmounts(370, 100, 3.7); err != nil {
		t.Fatalf("cross-currency payment should be valid: %v", err)
	}
	if err := validatePaymentAmounts(100, 100, 0); err != nil {
		t.Fatalf("same-currency explicit amounts should be valid: %v", err)
	}
	if err := validatePaymentAmounts(370, 100, 0); err == nil {
		t.Fatal("expected incomplete cross-currency values to fail")
	}
}

func TestDecodeLoanJSONSupportsArrayAndEnvelope(t *testing.T) {
	dir := t.TempDir()
	arrayFile := filepath.Join(dir, "allocations.json")
	if err := os.WriteFile(arrayFile, []byte(`[{"installmentId":"1","principal":10}]`), 0o600); err != nil {
		t.Fatal(err)
	}
	envelopeFile := filepath.Join(dir, "schedule.json")
	if err := os.WriteFile(envelopeFile, []byte(`{"installments":[{"id":"1","number":1,"dueDate":"2026-08-01","principal":100,"interest":10,"fee":0,"lateFee":0}]}`), 0o600); err != nil {
		t.Fatal(err)
	}
	allocations, err := decodeLoanAllocationsFile(arrayFile)
	if err != nil || len(allocations) != 1 || allocations[0].InstallmentID != "1" {
		t.Fatalf("decode allocations: %#v, %v", allocations, err)
	}
	schedule, err := decodeLoanScheduleFile(envelopeFile)
	if err != nil || len(schedule) != 1 || schedule[0].Number != 1 {
		t.Fatalf("decode schedule: %#v, %v", schedule, err)
	}
}

func TestLoansCommandExposesLifecycleAndScheduleCommands(t *testing.T) {
	cmd := newLoansCommand(&appState{})
	for _, path := range [][]string{{"get"}, {"update"}, {"archive"}, {"reactivate"}, {"delete"}, {"schedule", "get"}, {"schedule", "update"}, {"propose-allocation"}, {"correct-payment"}} {
		found, _, err := cmd.Find(path)
		if err != nil || found == cmd {
			t.Fatalf("missing loans command %v: %v", path, err)
		}
	}
}

func TestLoanUpdateInputIncludesOutstandingAndMonthlyPayment(t *testing.T) {
	f := loanFlags{Outstanding: 761.80, Monthly: 95.23}
	cmd := newLoanUpdateCommand(&appState{})
	if err := cmd.Flags().Set("outstanding-balance", "761.80"); err != nil {
		t.Fatal(err)
	}
	if err := cmd.Flags().Set("monthly-payment", "95.23"); err != nil {
		t.Fatal(err)
	}
	input := loanInput(f, cmd, true)
	if input["outstandingBalance"] != 761.80 || input["monthlyPayment"] != 95.23 {
		t.Fatalf("expected correction fields in update input, got %#v", input)
	}
}
