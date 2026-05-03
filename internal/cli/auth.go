package cli

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"saldo-cli/internal/graphql"
	"saldo-cli/internal/session"
)

func newAuthCommand(state *appState) *cobra.Command {
	cmd := &cobra.Command{Use: "auth", Short: "Authenticate with Saldo"}
	cmd.AddCommand(newAuthLoginCommand(state))
	cmd.AddCommand(newAuthWhoamiCommand(state))
	cmd.AddCommand(newAuthLogoutCommand(state))
	return cmd
}

func newAuthLoginCommand(state *appState) *cobra.Command {
	var email string
	var password string
	cmd := &cobra.Command{
		Use:   "login",
		Short: "Log in and store a private CLI session",
		RunE: func(cmd *cobra.Command, args []string) error {
			email = strings.TrimSpace(email)
			if email == "" {
				return fmt.Errorf("--email is required")
			}
			if password == "" {
				password = os.Getenv("SALDO_PASSWORD")
			}
			if password == "" {
				fmt.Fprint(os.Stderr, "Password: ")
				line, err := bufio.NewReader(os.Stdin).ReadString('\n')
				if err != nil {
					return fmt.Errorf("read password: %w", err)
				}
				password = strings.TrimRight(line, "\r\n")
			}

			s, _, err := session.Load()
			if err != nil {
				return err
			}
			apiURL := session.ResolveAPIURL(state.apiURL, s)
			if apiURL == "" {
				return fmt.Errorf("missing API URL; set SALDO_API_URL or run `saldo config set api-url <url>`")
			}
			client := graphql.NewClient(apiURL)
			var data struct {
				Login struct {
					AccessToken  string `json:"accessToken"`
					RefreshToken string `json:"refreshToken"`
					User         user   `json:"user"`
				} `json:"login"`
			}
			err = client.Raw(context.Background(), loginMutation, map[string]any{
				"input": map[string]any{"email": email, "password": password},
			}, &data)
			if err != nil {
				return err
			}
			s.APIURL = apiURL
			s.AccessToken = data.Login.AccessToken
			s.RefreshToken = data.Login.RefreshToken
			s.UserID = data.Login.User.ID
			s.Email = data.Login.User.Email
			path, err := session.Save(s)
			if err != nil {
				return err
			}
			if state.jsonOutput {
				return writeJSON(map[string]any{
					"loggedIn": true,
					"user":     data.Login.User,
					"session":  session.View(s, path),
				})
			}
			return writeHuman("Logged in as %s\n", data.Login.User.Email)
		},
	}
	cmd.Flags().StringVar(&email, "email", "", "Saldo user email")
	cmd.Flags().StringVar(&password, "password", "", "Saldo user password; may also use SALDO_PASSWORD")
	return cmd
}

func newAuthWhoamiCommand(state *appState) *cobra.Command {
	return &cobra.Command{
		Use:   "whoami",
		Short: "Show the authenticated user",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, s, path, err := requireSessionClient(state)
			if err != nil {
				return err
			}
			var data struct {
				Me user `json:"me"`
			}
			if err := client.Do(context.Background(), whoamiQuery, nil, &data); err != nil {
				return err
			}
			s.UserID = data.Me.ID
			s.Email = data.Me.Email
			if _, err := session.Save(s); err != nil {
				return err
			}
			if state.jsonOutput {
				return writeJSON(map[string]any{"user": data.Me, "session": session.View(s, path)})
			}
			return writeHuman("%s (%s)\n", data.Me.Email, data.Me.ID)
		},
	}
}

func newAuthLogoutCommand(state *appState) *cobra.Command {
	return &cobra.Command{
		Use:   "logout",
		Short: "Clear the private CLI session",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := session.Clear(); err != nil {
				return err
			}
			if state.jsonOutput {
				return writeJSON(map[string]any{"loggedIn": false})
			}
			return writeHuman("Logged out\n")
		},
	}
}

const loginMutation = `
mutation Login($input: LoginInput!) {
  login(input: $input) {
    accessToken
    refreshToken
    user { id email firstName lastName isActive }
  }
}`

const whoamiQuery = `
query Whoami {
  me { id email firstName lastName isActive }
}`
