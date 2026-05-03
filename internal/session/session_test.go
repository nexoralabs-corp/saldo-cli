package session

import (
	"os"
	"path/filepath"
	"testing"
)

func TestPathUsesSaldoSession(t *testing.T) {
	t.Setenv("SALDO_SESSION", filepath.Join(t.TempDir(), "agent-session.json"))
	path, err := Path()
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Base(path) != "agent-session.json" {
		t.Fatalf("expected explicit session path, got %s", path)
	}
}

func TestSaveLoadRoundTrip(t *testing.T) {
	t.Setenv("SALDO_SESSION", filepath.Join(t.TempDir(), "session.json"))
	want := &Session{APIURL: "https://saldo.test/graphql/", AccessToken: "access", RefreshToken: "refresh", UserID: "7", Email: "a@test"}
	if _, err := Save(want); err != nil {
		t.Fatal(err)
	}
	got, _, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if got.AccessToken != "access" || got.RefreshToken != "refresh" || got.Email != "a@test" {
		t.Fatalf("loaded session mismatch: %#v", got)
	}
}

func TestClearRemovesSession(t *testing.T) {
	path := filepath.Join(t.TempDir(), "session.json")
	t.Setenv("SALDO_SESSION", path)
	if _, err := Save(&Session{AccessToken: "access"}); err != nil {
		t.Fatal(err)
	}
	if err := Clear(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("expected session to be removed, stat err=%v", err)
	}
}

