package cli

import "testing"

func TestValidateRegistrationsRejectsDuplicateKeysAndContent(t *testing.T) {
	items := []registrationImport{
		{AccountID: "1", Amount: 10, Kind: "EXPENSE", Date: "2026-07-12", Description: "Internet", IdempotencyKey: "july-1"},
		{AccountID: "1", Amount: 10, Kind: "EXPENSE", Date: "2026-07-12", Description: "Internet", IdempotencyKey: "july-1"},
	}
	preview := validateRegistrations(items, true)
	if preview.Valid {
		t.Fatal("expected duplicate registrations to be invalid")
	}
	if len(preview.Items[1].Errors) != 2 {
		t.Fatalf("expected key and content duplicate errors, got %#v", preview.Items[1].Errors)
	}
}

func TestValidateRegistrationsAcceptsRetrySafeBatch(t *testing.T) {
	items := []registrationImport{
		{AccountID: "1", Amount: 110, Date: "2026-07-12", IdempotencyKey: "movistar-2026-07"},
		{AccountID: "1", Amount: 25, Kind: "INCOME", Date: "2026-07-13", IdempotencyKey: "refund-2026-07"},
	}
	preview := validateRegistrations(items, true)
	if !preview.Valid || preview.Ready != 2 {
		t.Fatalf("unexpected preview: %#v", preview)
	}
}

func TestDecodeRegistrationsSupportsEnvelopeAndArray(t *testing.T) {
	for _, raw := range []string{
		`[{"accountId":"1","amount":10,"date":"2026-07-12","idempotencyKey":"a"}]`,
		`{"registrations":[{"accountId":"1","amount":10,"date":"2026-07-12","idempotencyKey":"a"}]}`,
	} {
		items, err := decodeRegistrations([]byte(raw))
		if err != nil || len(items) != 1 {
			t.Fatalf("decode %s: items=%#v err=%v", raw, items, err)
		}
	}
}
