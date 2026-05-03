package cli

import "testing"

func TestSnapshotSectionsMapsNames(t *testing.T) {
	got := snapshotSections([]string{"net-worth", "credit_cards", "transactions"})
	if got["netWorth"] != true {
		t.Fatalf("expected netWorth enabled: %#v", got)
	}
	if got["creditCards"] != true {
		t.Fatalf("expected creditCards enabled: %#v", got)
	}
	if got["transactions"] != true {
		t.Fatalf("expected transactions enabled: %#v", got)
	}
	if got["loans"] != false {
		t.Fatalf("expected loans disabled: %#v", got)
	}
}

