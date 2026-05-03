package graphql

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"saldo-cli/internal/session"
)

func TestDoSendsBearerToken(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer access-token" {
			t.Fatalf("Authorization header = %q", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":{"me":{"id":"1","email":"a@test"}}}`))
	}))
	defer server.Close()

	s := &session.Session{AccessToken: "access-token"}
	client := NewClient(server.URL, WithSession(s, nil), WithHTTPClient(server.Client()))
	var out struct {
		Me struct {
			ID    string `json:"id"`
			Email string `json:"email"`
		} `json:"me"`
	}
	if err := client.Do(context.Background(), `query { me { id email } }`, nil, &out); err != nil {
		t.Fatal(err)
	}
	if out.Me.Email != "a@test" {
		t.Fatalf("unexpected response: %#v", out)
	}
}

func TestDoRefreshesAndRetriesAuthErrors(t *testing.T) {
	requestCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		w.Header().Set("Content-Type", "application/json")
		switch requestCount {
		case 1:
			if got := r.Header.Get("Authorization"); got != "Bearer old-access" {
				t.Fatalf("first Authorization header = %q", got)
			}
			_, _ = w.Write([]byte(`{"errors":[{"message":"Token is invalid or expired"}]}`))
		case 2:
			cookie, err := r.Cookie("refresh_token")
			if err != nil || cookie.Value != "refresh-token" {
				t.Fatalf("refresh cookie mismatch: cookie=%#v err=%v", cookie, err)
			}
			_, _ = w.Write([]byte(`{"data":{"refreshToken":"new-access"}}`))
		case 3:
			if got := r.Header.Get("Authorization"); got != "Bearer new-access" {
				t.Fatalf("retry Authorization header = %q", got)
			}
			_, _ = w.Write([]byte(`{"data":{"me":{"id":"1","email":"a@test"}}}`))
		default:
			t.Fatalf("unexpected request %d", requestCount)
		}
	}))
	defer server.Close()

	s := &session.Session{AccessToken: "old-access", RefreshToken: "refresh-token"}
	client := NewClient(server.URL, WithSession(s, nil), WithHTTPClient(server.Client()))
	var out struct {
		Me struct {
			ID    string `json:"id"`
			Email string `json:"email"`
		} `json:"me"`
	}
	if err := client.Do(context.Background(), `query { me { id email } }`, nil, &out); err != nil {
		t.Fatal(err)
	}
	if out.Me.Email != "a@test" {
		t.Fatalf("unexpected response: %#v", out)
	}
	if s.AccessToken != "new-access" {
		t.Fatalf("expected refreshed token, got %q", s.AccessToken)
	}
}

func TestRawReturnsGraphQLErrors(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"errors":[{"message":"Invalid credentials"}]}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, WithHTTPClient(server.Client()))
	err := client.Raw(context.Background(), `mutation { login }`, nil, nil)
	if err == nil || !strings.Contains(err.Error(), "Invalid credentials") {
		t.Fatalf("expected GraphQL error, got %v", err)
	}
}
