package cli

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/cobra"
)

func TestCardStatusFilter(t *testing.T) {
	for _, tc := range []struct{ in, want string }{{"active", "ACTIVE"}, {"ARCHIVED", "ARCHIVED"}, {"all", "ALL"}} {
		got, err := cardStatusFilter(tc.in)
		if err != nil || got != tc.want {
			t.Fatalf("cardStatusFilter(%q) = %q, %v", tc.in, got, err)
		}
	}
	if _, err := cardStatusFilter("paid"); err == nil {
		t.Fatal("expected invalid status error")
	}
}

func TestCurrencyInputForUpdateOnlyIncludesChangedFlags(t *testing.T) {
	f := cardCurrencyFlags{Currency: "pen", CreditLimit: 2500}
	cmd := &cobra.Command{}
	bindCardCurrencyFlags(cmd, &f, false)
	if err := cmd.Flags().Set("currency", "pen"); err != nil {
		t.Fatal(err)
	}
	if err := cmd.Flags().Set("credit-limit", "2500"); err != nil {
		t.Fatal(err)
	}
	input, err := currencyInputFromFlags(f, cmd, true)
	if err != nil {
		t.Fatal(err)
	}
	if input.Currency != "PEN" || input.CreditLimit == nil || *input.CreditLimit != 2500 {
		t.Fatalf("unexpected input: %#v", input)
	}
	if input.Balance != nil || input.TCEA != nil || input.DefaultPaymentAccountID != nil {
		t.Fatalf("unchanged fields must be omitted: %#v", input)
	}
}

func TestReadCurrencyInputsRejectsDuplicates(t *testing.T) {
	path := filepath.Join(t.TempDir(), "currencies.json")
	if err := os.WriteFile(path, []byte(`[{"currency":"pen"},{"currency":"PEN"}]`), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := readCurrencyInputs(path); err == nil {
		t.Fatal("expected duplicate currency error")
	}
}
