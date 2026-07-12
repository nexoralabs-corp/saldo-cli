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

type Store struct {
	APIURL   string    `json:"apiUrl,omitempty"`
	Profiles []Session `json:"profiles,omitempty"`
}

type PublicView struct {
	Path      string    `json:"path"`
	Profile   string    `json:"profile,omitempty"`
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

func Load(profile string) (*Session, string, error) {
	store, path, err := LoadStore()
	if err != nil {
		return nil, "", err
	}
	s := store.Selected(profile)
	if s == nil {
		return &Session{APIURL: store.APIURL, Email: strings.TrimSpace(profile)}, path, nil
	}
	if s.APIURL == "" {
		s.APIURL = store.APIURL
	}
	return s, path, nil
}

func LoadStore() (*Store, string, error) {
	path, err := Path()
	if err != nil {
		return nil, "", err
	}
	raw, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return &Store{}, path, nil
	}
	if err != nil {
		return nil, path, fmt.Errorf("read session: %w", err)
	}
	var shape map[string]json.RawMessage
	if err := json.Unmarshal(raw, &shape); err != nil {
		return nil, path, fmt.Errorf("parse session: %w", err)
	}
	if _, ok := shape["profiles"]; ok {
		var store Store
		if err := json.Unmarshal(raw, &store); err != nil {
			return nil, path, fmt.Errorf("parse session: %w", err)
		}
		return &store, path, nil
	}

	var store Store
	var s Session
	if err := json.Unmarshal(raw, &s); err != nil {
		return nil, path, fmt.Errorf("parse session: %w", err)
	}
	store.APIURL = s.APIURL
	if s.AccessToken != "" || s.RefreshToken != "" || s.Email != "" || s.UserID != "" {
		s.APIURL = ""
		store.Profiles = append(store.Profiles, s)
	}
	return &store, path, nil
}

func (store *Store) Selected(profile string) *Session {
	if len(store.Profiles) == 0 {
		return nil
	}
	profile = strings.TrimSpace(profile)
	if profile == "" {
		s := store.Profiles[0]
		return &s
	}
	for _, candidate := range store.Profiles {
		if strings.EqualFold(candidate.Email, profile) {
			s := candidate
			return &s
		}
	}
	return nil
}

func Save(profile string, s *Session) (string, error) {
	store, path, err := LoadStore()
	if err != nil {
		return "", err
	}
	saved := *s
	saved.UpdatedAt = time.Now().UTC()
	if saved.APIURL != "" {
		store.APIURL = saved.APIURL
	}
	saved.APIURL = ""
	if saved.AccessToken != "" || saved.RefreshToken != "" || saved.UserID != "" {
		store.Upsert(profile, &saved)
	}
	s.UpdatedAt = saved.UpdatedAt
	return SaveStore(path, store)
}

func (store *Store) Upsert(profile string, s *Session) {
	profile = strings.TrimSpace(profile)
	for i := range store.Profiles {
		if profile != "" && strings.EqualFold(store.Profiles[i].Email, profile) {
			store.Profiles[i] = *s
			return
		}
		if s.Email != "" && strings.EqualFold(store.Profiles[i].Email, s.Email) {
			store.Profiles[i] = *s
			return
		}
	}
	store.Profiles = append(store.Profiles, *s)
}

func SaveStore(path string, store *Store) (string, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return path, fmt.Errorf("create session directory: %w", err)
	}
	raw, err := json.MarshalIndent(store, "", "  ")
	if err != nil {
		return path, fmt.Errorf("encode session: %w", err)
	}
	if err := os.WriteFile(path, raw, 0o600); err != nil {
		return path, fmt.Errorf("write session: %w", err)
	}
	return path, nil
}

func Clear(profile string) error {
	store, path, err := LoadStore()
	if err != nil {
		return err
	}
	if len(store.Profiles) > 0 {
		store.Remove(profile)
	}
	if store.APIURL == "" && len(store.Profiles) == 0 {
		if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("remove session: %w", err)
		}
		return nil
	}
	_, err = SaveStore(path, store)
	return err
}

func (store *Store) Remove(profile string) {
	profile = strings.TrimSpace(profile)
	if profile == "" {
		store.Profiles = store.Profiles[1:]
		return
	}
	for i, candidate := range store.Profiles {
		if strings.EqualFold(candidate.Email, profile) {
			store.Profiles = append(store.Profiles[:i], store.Profiles[i+1:]...)
			return
		}
	}
}

func ClearAll() error {
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
		Profile:   s.Email,
		APIURL:    s.APIURL,
		UserID:    s.UserID,
		Email:     s.Email,
		LoggedIn:  s.AccessToken != "" || s.RefreshToken != "",
		UpdatedAt: s.UpdatedAt,
	}
}
