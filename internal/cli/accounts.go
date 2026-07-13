package cli

import (
	"context"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

func newAccountsCommand(state *appState) *cobra.Command {
	cmd := &cobra.Command{Use: "accounts", Short: "Manage Saldo accounts"}
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
	cmd.AddCommand(newAccountCreateCommand(state))
	cmd.AddCommand(newAccountUpdateCommand(state))
	cmd.AddCommand(newAccountDeleteCommand(state))
	return cmd
}

type accountFlags struct {
	Name, Type, Currency, BankName, Issuer string
	Balance, CreditLimit                   float64
	ClosingDay, DueDay                     int
}

func bindAccountFlags(cmd *cobra.Command, f *accountFlags, includeType bool) {
	cmd.Flags().StringVar(&f.Name, "name", "", "account name")
	if includeType {
		cmd.Flags().StringVar(&f.Type, "type", "", "CASH, BANK, CREDIT_CARD, or LOAN")
	}
	cmd.Flags().StringVar(&f.Currency, "currency", "", "ISO currency code")
	cmd.Flags().Float64Var(&f.Balance, "balance", 0, "current or initial balance")
	cmd.Flags().StringVar(&f.BankName, "bank-name", "", "bank name")
	cmd.Flags().StringVar(&f.Issuer, "issuer", "", "card issuer")
	cmd.Flags().Float64Var(&f.CreditLimit, "credit-limit", 0, "credit limit")
	cmd.Flags().IntVar(&f.ClosingDay, "closing-day", 0, "statement closing day")
	cmd.Flags().IntVar(&f.DueDay, "due-day", 0, "payment due day")
}

func validateAccountType(value string) (string, error) {
	value = strings.ToUpper(strings.TrimSpace(value))
	switch value {
	case "CASH", "BANK", "CREDIT_CARD", "LOAN":
		return value, nil
	}
	return "", fmt.Errorf("--type must be CASH, BANK, CREDIT_CARD, or LOAN")
}

func accountInput(f accountFlags, cmd *cobra.Command, update bool) (map[string]any, error) {
	input := map[string]any{}
	put := func(flag, key string, value any) {
		if !update || cmd.Flags().Changed(flag) {
			input[key] = value
		}
	}
	put("name", "name", strings.TrimSpace(f.Name))
	if !update || cmd.Flags().Changed("type") {
		t, err := validateAccountType(f.Type)
		if err != nil {
			return nil, err
		}
		input["accountType"] = t
	}
	if !update && f.Currency == "" {
		f.Currency = "PEN"
	}
	put("currency", "currency", strings.ToUpper(f.Currency))
	put("balance", "balance", f.Balance)
	put("bank-name", "bankName", f.BankName)
	put("issuer", "issuer", f.Issuer)
	put("credit-limit", "creditLimit", f.CreditLimit)
	put("closing-day", "closingDay", f.ClosingDay)
	put("due-day", "dueDay", f.DueDay)
	return input, nil
}

func newAccountCreateCommand(state *appState) *cobra.Command {
	f := accountFlags{}
	cmd := &cobra.Command{Use: "create", Short: "Create an account", RunE: func(cmd *cobra.Command, args []string) error {
		if strings.TrimSpace(f.Name) == "" {
			return fmt.Errorf("--name is required")
		}
		if f.Type == "" {
			return fmt.Errorf("--type is required")
		}
		input, err := accountInput(f, cmd, false)
		if err != nil {
			return err
		}
		client, _, _, err := requireSessionClient(state)
		if err != nil {
			return err
		}
		var data struct {
			CreateAccount account `json:"createAccount"`
		}
		if err = client.Do(context.Background(), createAccountMutation, map[string]any{"input": input}, &data); err != nil {
			return err
		}
		if state.jsonOutput {
			return writeJSON(data.CreateAccount)
		}
		return writeHuman("Created account %s\n", data.CreateAccount.ID)
	}}
	bindAccountFlags(cmd, &f, true)
	return cmd
}

func newAccountUpdateCommand(state *appState) *cobra.Command {
	f := accountFlags{}
	cmd := &cobra.Command{Use: "update <id>", Short: "Update an account", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		input, err := accountInput(f, cmd, true)
		if err != nil {
			return err
		}
		if len(input) == 0 {
			return fmt.Errorf("at least one field must be provided")
		}
		client, _, _, err := requireSessionClient(state)
		if err != nil {
			return err
		}
		var data struct {
			UpdateAccount account `json:"updateAccount"`
		}
		if err = client.Do(context.Background(), updateAccountMutation, map[string]any{"id": args[0], "input": input}, &data); err != nil {
			return err
		}
		if state.jsonOutput {
			return writeJSON(data.UpdateAccount)
		}
		return writeHuman("Updated account %s\n", data.UpdateAccount.ID)
	}}
	bindAccountFlags(cmd, &f, true)
	return cmd
}

func newAccountDeleteCommand(state *appState) *cobra.Command {
	return &cobra.Command{Use: "delete <id>", Short: "Delete an account", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		client, _, _, err := requireSessionClient(state)
		if err != nil {
			return err
		}
		var data struct {
			DeleteAccount bool `json:"deleteAccount"`
		}
		if err = client.Do(context.Background(), deleteAccountMutation, map[string]any{"id": args[0]}, &data); err != nil {
			return err
		}
		if state.jsonOutput {
			return writeJSON(map[string]any{"deleted": data.DeleteAccount, "id": args[0]})
		}
		return writeHuman("Deleted account %s\n", args[0])
	}}
}

const accountFields = `
id
name
accountType
currency
balance
isActive
bankName
familyWalletId
creditLimit
closingDay
dueDay
nextClosingDate
nextDueDate
tea
tcea
cashAdvanceRate
annualFee
minimumPayment
creditCardId
defaultPaymentAccountId
issuer`

const accountsQuery = `
query Accounts {
  accounts { ` + accountFields + ` }
}`

const accountQuery = `
query Account($id: ID!) {
  account(id: $id) { ` + accountFields + ` }
}`

const createAccountMutation = `mutation CreateAccount($input: CreateAccountInput!) {
  createAccount(input: $input) { ` + accountFields + ` }
}`

const updateAccountMutation = `mutation UpdateAccount($id: ID!, $input: UpdateAccountInput!) {
  updateAccount(id: $id, input: $input) { ` + accountFields + ` }
}`

const deleteAccountMutation = `mutation DeleteAccount($id: ID!) { deleteAccount(id: $id) }`
