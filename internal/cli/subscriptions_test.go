package cli

import (
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestSubscriptionInputCreatesServiceChargeFields(t *testing.T) {
	f := subscriptionFlags{}
	cmd := &cobra.Command{}
	bindSubscriptionFlags(cmd, &f, false)
	if err := cmd.ParseFlags([]string{
		"--name", "Movistar", "--amount", "110", "--currency", "pen",
		"--billing-cycle", "monthly", "--amount-type", "variable", "--charge-mode", "automatic",
		"--next-charge-date", "2026-08-15T00:00:00Z", "--due-date", "2026-08-20T00:00:00Z",
		"--next-charge-amount", "120", "--due-day", "20", "--account-id", "4", "--category-id", "8",
	}); err != nil {
		t.Fatalf("parse flags: %v", err)
	}

	input, err := subscriptionInput(f, cmd, false)
	if err != nil {
		t.Fatalf("subscription input: %v", err)
	}
	for key, want := range map[string]any{
		"name":             "Movistar",
		"currency":         "PEN",
		"billingCycle":     "MONTHLY",
		"amountType":       "VARIABLE",
		"chargeMode":       "AUTOMATIC",
		"nextChargeDate":   "2026-08-15T00:00:00Z",
		"nextRenewalDate":  "2026-08-15T00:00:00Z",
		"dueDate":          "2026-08-20T00:00:00Z",
		"nextChargeAmount": float64(120),
		"dueDay":           20,
		"accountId":        "4",
		"categoryId":       "8",
	} {
		if got := input[key]; got != want {
			t.Errorf("%s = %#v, want %#v", key, got, want)
		}
	}
}

func TestSubscriptionUpdateOnlyIncludesChangedFields(t *testing.T) {
	f := subscriptionFlags{}
	cmd := &cobra.Command{}
	bindSubscriptionFlags(cmd, &f, true)
	if err := cmd.ParseFlags([]string{"--charge-mode", "automatic", "--next-charge-amount", "15.5"}); err != nil {
		t.Fatalf("parse flags: %v", err)
	}

	input, err := subscriptionInput(f, cmd, true)
	if err != nil {
		t.Fatalf("subscription input: %v", err)
	}
	if len(input) != 2 {
		t.Fatalf("input = %#v, want only two fields", input)
	}
	if input["chargeMode"] != "AUTOMATIC" || input["nextChargeAmount"] != float64(15.5) {
		t.Fatalf("unexpected update input: %#v", input)
	}
}

func TestSubscriptionChargeRequiresIdempotencyKey(t *testing.T) {
	cmd := newSubscriptionsCommand(&appState{})
	cmd.SetArgs([]string{"charge", "12", "--actual-amount", "20"})
	err := cmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "--idempotency-key is required") {
		t.Fatalf("expected idempotency key error, got %v", err)
	}
}

func TestSubscriptionCommandParity(t *testing.T) {
	cmd := newSubscriptionsCommand(&appState{})
	for _, name := range []string{"create", "list", "upcoming", "get", "update", "archive", "reactivate", "delete", "charge", "correct-charge", "history"} {
		child, _, err := cmd.Find([]string{name})
		if err != nil || child == cmd {
			t.Fatalf("missing subscriptions %s command: %v", name, err)
		}
	}
}

func TestSubscriptionStatusQueryUsesEnumLiteral(t *testing.T) {
	query := subscriptionListQuery("ARCHIVED")
	if !strings.Contains(query, "subscriptions(status:ARCHIVED)") {
		t.Fatalf("query should use GraphQL enum literal: %s", query)
	}
}
