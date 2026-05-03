package graphql

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"saldo-cli/internal/session"
)

type Client struct {
	apiURL     string
	httpClient *http.Client
	session    *session.Session
	save       func(*session.Session) error
}

type Option func(*Client)

func WithHTTPClient(httpClient *http.Client) Option {
	return func(c *Client) {
		c.httpClient = httpClient
	}
}

func WithSession(s *session.Session, save func(*session.Session) error) Option {
	return func(c *Client) {
		c.session = s
		c.save = save
	}
}

func NewClient(apiURL string, opts ...Option) *Client {
	c := &Client{
		apiURL:     strings.TrimSpace(apiURL),
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

type requestBody struct {
	Query     string         `json:"query"`
	Variables map[string]any `json:"variables,omitempty"`
}

type responseBody struct {
	Data   json.RawMessage `json:"data"`
	Errors []gqlError      `json:"errors,omitempty"`
}

type gqlError struct {
	Message string `json:"message"`
}

func (c *Client) Do(ctx context.Context, query string, variables map[string]any, out any) error {
	if c.apiURL == "" {
		return errors.New("missing API URL; set SALDO_API_URL or run `saldo config set api-url <url>`")
	}
	if err := c.refreshIfNeeded(ctx); err != nil {
		return err
	}
	err := c.do(ctx, query, variables, out)
	if err == nil || !looksAuthRelated(err) || c.session == nil || c.session.RefreshToken == "" {
		return err
	}
	if refreshErr := c.refresh(ctx); refreshErr != nil {
		return err
	}
	return c.do(ctx, query, variables, out)
}

func (c *Client) Raw(ctx context.Context, query string, variables map[string]any, out any) error {
	if c.apiURL == "" {
		return errors.New("missing API URL; set SALDO_API_URL or run `saldo config set api-url <url>`")
	}
	return c.do(ctx, query, variables, out)
}

func (c *Client) do(ctx context.Context, query string, variables map[string]any, out any) error {
	payload, err := json.Marshal(requestBody{Query: query, Variables: variables})
	if err != nil {
		return fmt.Errorf("encode GraphQL request: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.apiURL, bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("create GraphQL request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	if c.session != nil && c.session.AccessToken != "" {
		req.Header.Set("Authorization", "Bearer "+c.session.AccessToken)
	}
	if strings.Contains(query, "refreshToken") && c.session != nil && c.session.RefreshToken != "" {
		req.AddCookie(&http.Cookie{Name: "refresh_token", Value: c.session.RefreshToken, Path: "/graphql/"})
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("send GraphQL request: %w", err)
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read GraphQL response: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("GraphQL HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(raw)))
	}
	var decoded responseBody
	if err := json.Unmarshal(raw, &decoded); err != nil {
		return fmt.Errorf("parse GraphQL response: %w", err)
	}
	if len(decoded.Errors) > 0 {
		messages := make([]string, 0, len(decoded.Errors))
		for _, gqlErr := range decoded.Errors {
			messages = append(messages, gqlErr.Message)
		}
		return errors.New(strings.Join(messages, "; "))
	}
	if out == nil {
		return nil
	}
	if len(decoded.Data) == 0 || string(decoded.Data) == "null" {
		return errors.New("GraphQL response did not include data")
	}
	if err := json.Unmarshal(decoded.Data, out); err != nil {
		return fmt.Errorf("decode GraphQL data: %w", err)
	}
	return nil
}

func (c *Client) refreshIfNeeded(ctx context.Context) error {
	if c.session == nil || c.session.AccessToken == "" || c.session.RefreshToken == "" {
		return nil
	}
	if !jwtExpiresSoon(c.session.AccessToken, 30*time.Second) {
		return nil
	}
	return c.refresh(ctx)
}

func (c *Client) refresh(ctx context.Context) error {
	var data struct {
		RefreshToken string `json:"refreshToken"`
	}
	if err := c.do(ctx, `mutation RefreshToken { refreshToken }`, nil, &data); err != nil {
		return fmt.Errorf("refresh token: %w", err)
	}
	c.session.AccessToken = data.RefreshToken
	if c.save != nil {
		if err := c.save(c.session); err != nil {
			return err
		}
	}
	return nil
}

func jwtExpiresSoon(token string, window time.Duration) bool {
	parts := strings.Split(token, ".")
	if len(parts) < 2 {
		return false
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return false
	}
	var claims struct {
		Exp int64 `json:"exp"`
	}
	if err := json.Unmarshal(payload, &claims); err != nil || claims.Exp == 0 {
		return false
	}
	return time.Unix(claims.Exp, 0).Before(time.Now().Add(window))
}

func looksAuthRelated(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "auth") ||
		strings.Contains(msg, "credential") ||
		strings.Contains(msg, "token") ||
		strings.Contains(msg, "permission") ||
		strings.Contains(msg, "not authenticated")
}

