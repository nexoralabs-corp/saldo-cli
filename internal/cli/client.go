package cli

import (
	"fmt"

	"saldo-cli/internal/graphql"
	"saldo-cli/internal/session"
)

func loadSessionClient(state *appState) (*graphql.Client, *session.Session, string, error) {
	s, path, err := session.Load()
	if err != nil {
		return nil, nil, "", err
	}
	apiURL := session.ResolveAPIURL(state.apiURL, s)
	if apiURL == "" {
		return nil, nil, path, fmt.Errorf("missing API URL; set SALDO_API_URL or run `saldo config set api-url <url>`")
	}
	s.APIURL = apiURL
	save := func(updated *session.Session) error {
		_, err := session.Save(updated)
		return err
	}
	client := graphql.NewClient(apiURL, graphql.WithSession(s, save))
	return client, s, path, nil
}

func requireSessionClient(state *appState) (*graphql.Client, *session.Session, string, error) {
	client, s, path, err := loadSessionClient(state)
	if err != nil {
		return nil, nil, "", err
	}
	if s.AccessToken == "" && s.RefreshToken == "" {
		return nil, nil, path, fmt.Errorf("not logged in; run `saldo auth login --email <email>`")
	}
	return client, s, path, nil
}

