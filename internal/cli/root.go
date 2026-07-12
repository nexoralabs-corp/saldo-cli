package cli

import (
	"github.com/spf13/cobra"
)

type appState struct {
	jsonOutput bool
	apiURL     string
	profile    string
}

func NewRootCommand() *cobra.Command {
	state := &appState{}
	cmd := &cobra.Command{
		Use:           "saldo",
		Short:         "Agent-friendly CLI for Saldo",
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	cmd.PersistentFlags().BoolVar(&state.jsonOutput, "json", false, "emit machine-readable JSON")
	cmd.PersistentFlags().StringVar(&state.apiURL, "api-url", "", "GraphQL API URL; overrides SALDO_API_URL and saved config")
	cmd.PersistentFlags().StringVar(&state.profile, "profile", "", "session profile email; defaults to the first saved profile")
	cmd.PersistentFlags().StringVar(&state.profile, "account", "", "alias for --profile")

	cmd.AddCommand(newAuthCommand(state))
	cmd.AddCommand(newConfigCommand(state))
	cmd.AddCommand(newAccountsCommand(state))
	cmd.AddCommand(newTransactionsCommand(state))
	cmd.AddCommand(newCategoriesCommand(state))
	cmd.AddCommand(newTagsCommand(state))
	cmd.AddCommand(newSnapshotCommand(state))

	return cmd
}
