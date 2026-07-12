package session

import (
	"encoding/json"
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
	if _, err := Save("", want); err != nil {
		t.Fatal(err)
	}
	got, _, err := Load("")
	if err != nil {
		t.Fatal(err)
	}
	if got.AccessToken != "access" || got.RefreshToken != "refresh" || got.Email != "a@test" || got.APIURL != "https://saldo.test/graphql/" {
		t.Fatalf("loaded session mismatch: %#v", got)
	}
}

func TestLoadSelectsProfiles(t *testing.T) {
	t.Setenv("SALDO_SESSION", filepath.Join(t.TempDir(), "session.json"))
	if _, err := Save("", &Session{AccessToken: "first", Email: "a@test"}); err != nil {
		t.Fatal(err)
	}
	if _, err := Save("", &Session{AccessToken: "second", Email: "b@test"}); err != nil {
		t.Fatal(err)
	}
	gotDefault, _, err := Load("")
	if err != nil {
		t.Fatal(err)
	}
	if gotDefault.Email != "a@test" || gotDefault.AccessToken != "first" {
		t.Fatalf("expected first profile as default, got %#v", gotDefault)
	}
	gotSelected, _, err := Load("b@test")
	if err != nil {
		t.Fatal(err)
	}
	if gotSelected.Email != "b@test" || gotSelected.AccessToken != "second" {
		t.Fatalf("expected selected profile, got %#v", gotSelected)
	}
}

func TestSaveConfigOnlyDoesNotCreateProfile(t *testing.T) {
	t.Setenv("SALDO_SESSION", filepath.Join(t.TempDir(), "session.json"))
	if _, err := Save("missing@test", &Session{APIURL: "https://saldo.test/graphql/", Email: "missing@test"}); err != nil {
		t.Fatal(err)
	}
	store, _, err := LoadStore()
	if err != nil {
		t.Fatal(err)
	}
	if store.APIURL != "https://saldo.test/graphql/" {
		t.Fatalf("expected API URL to be saved, got %#v", store)
	}
	if len(store.Profiles) != 0 {
		t.Fatalf("expected no profiles, got %#v", store.Profiles)
	}
}

func TestLoadMigratesLegacySession(t *testing.T) {
	path := filepath.Join(t.TempDir(), "session.json")
	t.Setenv("SALDO_SESSION", path)
	raw, err := json.Marshal(&Session{
		APIURL:       "https://saldo.test/graphql/",
		AccessToken:  "access",
		RefreshToken: "refresh",
		UserID:       "7",
		Email:        "legacy@test",
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, raw, 0o600); err != nil {
		t.Fatal(err)
	}
	got, _, err := Load("")
	if err != nil {
		t.Fatal(err)
	}
	if got.Email != "legacy@test" || got.AccessToken != "access" || got.APIURL != "https://saldo.test/graphql/" {
		t.Fatalf("expected legacy session to load as default profile, got %#v", got)
	}
}

func TestClearRemovesSession(t *testing.T) {
	path := filepath.Join(t.TempDir(), "session.json")
	t.Setenv("SALDO_SESSION", path)
	if _, err := Save("", &Session{AccessToken: "access"}); err != nil {
		t.Fatal(err)
	}
	if err := Clear(""); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("expected session to be removed, stat err=%v", err)
	}
}

func TestClearSelectedProfileKeepsOthers(t *testing.T) {
	t.Setenv("SALDO_SESSION", filepath.Join(t.TempDir(), "session.json"))
	if _, err := Save("", &Session{AccessToken: "first", Email: "a@test"}); err != nil {
		t.Fatal(err)
	}
	if _, err := Save("", &Session{AccessToken: "second", Email: "b@test"}); err != nil {
		t.Fatal(err)
	}
	if err := Clear("a@test"); err != nil {
		t.Fatal(err)
	}
	got, _, err := Load("")
	if err != nil {
		t.Fatal(err)
	}
	if got.Email != "b@test" || got.AccessToken != "second" {
		t.Fatalf("expected remaining profile as default, got %#v", got)
	}
}
