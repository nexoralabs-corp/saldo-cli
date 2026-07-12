package cli

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"golang.org/x/term"

	"saldo-cli/internal/graphql"
	"saldo-cli/internal/session"
)

func newAuthCommand(state *appState) *cobra.Command {
	cmd := &cobra.Command{Use: "auth", Short: "Authenticate with Saldo"}
	cmd.AddCommand(newAuthLoginCommand(state))
	cmd.AddCommand(newAuthWhoamiCommand(state))
	cmd.AddCommand(newAuthProfilesCommand(state))
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
				line, err := readPassword()
				if err != nil {
					return err
				}
				password = line
			}

			s, _, err := session.Load(state.profile)
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
			path, err := session.Save(state.profile, s)
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

func readPassword() (string, error) {
	fmt.Fprint(os.Stderr, "Password: ")
	if term.IsTerminal(int(os.Stdin.Fd())) {
		return readMaskedPassword()
	}

	line, err := bufio.NewReader(os.Stdin).ReadString('\n')
	if err != nil {
		return "", fmt.Errorf("read password: %w", err)
	}
	return strings.TrimRight(line, "\r\n"), nil
}

func readMaskedPassword() (string, error) {
	fd := int(os.Stdin.Fd())
	oldState, err := term.MakeRaw(fd)
	if err != nil {
		return "", fmt.Errorf("read password: %w", err)
	}
	restored := false
	restore := func() {
		if !restored {
			term.Restore(fd, oldState)
			restored = true
		}
	}
	defer restore()

	var password []byte
	var buf [1]byte
	for {
		n, err := os.Stdin.Read(buf[:])
		if err != nil {
			restore()
			fmt.Fprintln(os.Stderr)
			return "", fmt.Errorf("read password: %w", err)
		}
		if n == 0 {
			continue
		}

		switch b := buf[0]; b {
		case '\r', '\n':
			restore()
			fmt.Fprintln(os.Stderr)
			return string(password), nil
		case 3, 4:
			restore()
			fmt.Fprintln(os.Stderr)
			return "", fmt.Errorf("password input canceled")
		case 8, 127:
			if len(password) > 0 {
				password = password[:len(password)-1]
				fmt.Fprint(os.Stderr, "\b \b")
			}
		default:
			if b >= 32 {
				password = append(password, b)
				fmt.Fprint(os.Stderr, "*")
			}
		}
	}
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
			if _, err := session.Save(state.profile, s); err != nil {
				return err
			}
			if state.jsonOutput {
				return writeJSON(map[string]any{"user": data.Me, "session": session.View(s, path)})
			}
			return writeHuman("%s (%s)\n", data.Me.Email, data.Me.ID)
		},
	}
}

func newAuthProfilesCommand(state *appState) *cobra.Command {
	return &cobra.Command{
		Use:     "profiles",
		Aliases: []string{"sessions"},
		Short:   "List saved login profiles",
		RunE: func(cmd *cobra.Command, args []string) error {
			store, path, err := session.LoadStore()
			if err != nil {
				return err
			}
			profiles := make([]session.PublicView, 0, len(store.Profiles))
			for _, profile := range store.Profiles {
				if profile.APIURL == "" {
					profile.APIURL = store.APIURL
				}
				profiles = append(profiles, session.View(&profile, path))
			}
			if state.jsonOutput {
				return writeJSON(map[string]any{"profiles": profiles})
			}
			if len(profiles) == 0 {
				return writeHuman("No saved profiles\n")
			}
			for i, profile := range profiles {
				prefix := " "
				if i == 0 {
					prefix = "*"
				}
				if profile.UserID == "" {
					if err := writeHuman("%s %s\n", prefix, profile.Email); err != nil {
						return err
					}
					continue
				}
				if err := writeHuman("%s %s (%s)\n", prefix, profile.Email, profile.UserID); err != nil {
					return err
				}
			}
			return nil
		},
	}
}

func newAuthLogoutCommand(state *appState) *cobra.Command {
	var all bool
	cmd := &cobra.Command{
		Use:   "logout",
		Short: "Clear the private CLI session",
		RunE: func(cmd *cobra.Command, args []string) error {
			if all {
				if err := session.ClearAll(); err != nil {
					return err
				}
			} else if err := session.Clear(state.profile); err != nil {
				return err
			}
			if state.jsonOutput {
				return writeJSON(map[string]any{"loggedIn": false, "profile": state.profile, "all": all})
			}
			return writeHuman("Logged out\n")
		},
	}
	cmd.Flags().BoolVar(&all, "all", false, "clear all saved login profiles")
	return cmd
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
