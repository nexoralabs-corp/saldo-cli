package session

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	envSession = "SALDO_SESSION"
	envAPIURL  = "SALDO_API_URL"
)

type Session struct {
	APIURL       string    `json:"apiUrl,omitempty"`
	AccessToken  string    `json:"accessToken,omitempty"`
	RefreshToken string    `json:"refreshToken,omitempty"`
	UserID       string    `json:"userId,omitempty"`
	Email        string    `json:"email,omitempty"`
	UpdatedAt    time.Time `json:"updatedAt,omitempty"`
}

type PublicView struct {
	Path      string    `json:"path"`
	APIURL    string    `json:"apiUrl,omitempty"`
	UserID    string    `json:"userId,omitempty"`
	Email     string    `json:"email,omitempty"`
	LoggedIn  bool      `json:"loggedIn"`
	UpdatedAt time.Time `json:"updatedAt,omitempty"`
}

func Path() (string, error) {
	if explicit := strings.TrimSpace(os.Getenv(envSession)); explicit != "" {
		return filepath.Abs(explicit)
	}
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("resolve config directory: %w", err)
	}
	return filepath.Join(configDir, "saldo", "session.json"), nil
}

func Load() (*Session, string, error) {
	path, err := Path()
	if err != nil {
		return nil, "", err
	}
	raw, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return &Session{}, path, nil
	}
	if err != nil {
		return nil, path, fmt.Errorf("read session: %w", err)
	}
	var s Session
	if err := json.Unmarshal(raw, &s); err != nil {
		return nil, path, fmt.Errorf("parse session: %w", err)
	}
	return &s, path, nil
}

func Save(s *Session) (string, error) {
	path, err := Path()
	if err != nil {
		return "", err
	}
	s.UpdatedAt = time.Now().UTC()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return path, fmt.Errorf("create session directory: %w", err)
	}
	raw, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return path, fmt.Errorf("encode session: %w", err)
	}
	if err := os.WriteFile(path, raw, 0o600); err != nil {
		return path, fmt.Errorf("write session: %w", err)
	}
	return path, nil
}

func Clear() error {
	path, err := Path()
	if err != nil {
		return err
	}
	if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("remove session: %w", err)
	}
	return nil
}

func ResolveAPIURL(flagValue string, s *Session) string {
	if flagValue = strings.TrimSpace(flagValue); flagValue != "" {
		return flagValue
	}
	if envValue := strings.TrimSpace(os.Getenv(envAPIURL)); envValue != "" {
		return envValue
	}
	if s != nil {
		return strings.TrimSpace(s.APIURL)
	}
	return ""
}

func View(s *Session, path string) PublicView {
	return PublicView{
		Path:      path,
		APIURL:    s.APIURL,
		UserID:    s.UserID,
		Email:     s.Email,
		LoggedIn:  s.AccessToken != "" || s.RefreshToken != "",
		UpdatedAt: s.UpdatedAt,
	}
}

