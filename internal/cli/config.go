package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"saldo-cli/internal/session"
)

func newConfigCommand(state *appState) *cobra.Command {
	cmd := &cobra.Command{Use: "config", Short: "Manage CLI configuration"}
	cmd.AddCommand(newConfigSetCommand(state))
	cmd.AddCommand(newConfigGetCommand(state))
	return cmd
}

func newConfigSetCommand(state *appState) *cobra.Command {
	cmd := &cobra.Command{Use: "set", Short: "Set a config value"}
	cmd.AddCommand(&cobra.Command{
		Use:   "api-url <url>",
		Short: "Set the GraphQL API URL",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			apiURL := strings.TrimSpace(args[0])
			if apiURL == "" {
				return fmt.Errorf("api-url cannot be empty")
			}
			s, _, err := session.Load()
			if err != nil {
				return err
			}
			s.APIURL = apiURL
			path, err := session.Save(s)
			if err != nil {
				return err
			}
			if state.jsonOutput {
				return writeJSON(session.View(s, path))
			}
			return writeHuman("API URL set to %s\n", apiURL)
		},
	})
	return cmd
}

func newConfigGetCommand(state *appState) *cobra.Command {
	return &cobra.Command{
		Use:   "get",
		Short: "Show current CLI configuration without exposing tokens",
		RunE: func(cmd *cobra.Command, args []string) error {
			s, path, err := session.Load()
			if err != nil {
				return err
			}
			resolved := session.ResolveAPIURL(state.apiURL, s)
			view := session.View(s, path)
			if resolved != "" {
				view.APIURL = resolved
			}
			if state.jsonOutput {
				return writeJSON(view)
			}
			if view.APIURL == "" {
				return writeHuman("API URL: not set\nSession: %s\n", view.Path)
			}
			return writeHuman("API URL: %s\nSession: %s\n", view.APIURL, view.Path)
		},
	}
}

