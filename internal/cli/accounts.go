package cli

import (
	"context"

	"github.com/spf13/cobra"
)

func newAccountsCommand(state *appState) *cobra.Command {
	cmd := &cobra.Command{Use: "accounts", Short: "Read Saldo accounts"}
	cmd.AddCommand(&cobra.Command{
		Use:   "list",
		Short: "List accounts",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, _, _, err := requireSessionClient(state)
			if err != nil {
				return err
			}
			var data struct {
				Accounts []account `json:"accounts"`
			}
			if err := client.Do(context.Background(), accountsQuery, nil, &data); err != nil {
				return err
			}
			if state.jsonOutput {
				return writeJSON(data.Accounts)
			}
			for _, account := range data.Accounts {
				if err := writeHuman("%s\t%s\t%.2f %s\n", account.ID, account.Name, float64(account.Balance), account.Currency); err != nil {
					return err
				}
			}
			return nil
		},
	})
	cmd.AddCommand(&cobra.Command{
		Use:   "get <id>",
		Short: "Get an account",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, _, _, err := requireSessionClient(state)
			if err != nil {
				return err
			}
			var data struct {
				Account account `json:"account"`
			}
			if err := client.Do(context.Background(), accountQuery, map[string]any{"id": args[0]}, &data); err != nil {
				return err
			}
			if state.jsonOutput {
				return writeJSON(data.Account)
			}
			return writeHuman("%s\t%s\t%.2f %s\n", data.Account.ID, data.Account.Name, float64(data.Account.Balance), data.Account.Currency)
		},
	})
	return cmd
}

const accountFields = `
id
name
accountType
currency
balance
isActive
bankName
familyWalletId`

const accountsQuery = `
query Accounts {
  accounts { ` + accountFields + ` }
}`

const accountQuery = `
query Account($id: ID!) {
  account(id: $id) { ` + accountFields + ` }
}`
