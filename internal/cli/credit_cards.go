package cli

import (
	"context"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

func newCreditCardsCommand(state *appState) *cobra.Command {
	cmd := &cobra.Command{Use: "credit-cards", Short: "Manage credit cards"}
	cmd.AddCommand(newCreditCardCreateCommand(state), newCreditCardsListCommand(state), newCreditCardPaymentCommand(state))
	return cmd
}

func newCreditCardCreateCommand(state *appState) *cobra.Command {
	f := accountFlags{Type: "CREDIT_CARD"}
	cmd := &cobra.Command{Use: "create", Short: "Create a credit card", RunE: func(cmd *cobra.Command, args []string) error {
		if strings.TrimSpace(f.Name) == "" {
			return fmt.Errorf("--name is required")
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
		if err := client.Do(context.Background(), createAccountMutation, map[string]any{"input": input}, &data); err != nil {
			return err
		}
		if state.jsonOutput {
			return writeJSON(data.CreateAccount)
		}
		return writeHuman("Created credit card %s\n", data.CreateAccount.ID)
	}}
	bindAccountFlags(cmd, &f, false)
	return cmd
}

func newCreditCardsListCommand(state *appState) *cobra.Command {
	return &cobra.Command{Use: "list", Short: "List credit cards", RunE: func(cmd *cobra.Command, args []string) error {
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
		cards := make([]account, 0)
		for _, item := range data.Accounts {
			if item.AccountType == "CREDIT_CARD" {
				cards = append(cards, item)
			}
		}
		if state.jsonOutput {
			return writeJSON(cards)
		}
		for _, card := range cards {
			if err := writeHuman("%s\t%s\t%.2f %s\t%s\n", card.ID, card.Name, float64(card.Balance), card.Currency, card.Issuer); err != nil {
				return err
			}
		}
		return nil
	}}
}

func newCreditCardPaymentCommand(state *appState) *cobra.Command {
	var cardID, fromID, date, key string
	var amount float64
	cmd := &cobra.Command{Use: "payment", Short: "Record a credit card payment", RunE: func(cmd *cobra.Command, args []string) error {
		if cardID == "" {
			return fmt.Errorf("--card-id is required")
		}
		if fromID == "" {
			return fmt.Errorf("--from-account-id is required")
		}
		if amount <= 0 {
			return fmt.Errorf("--amount must be greater than zero")
		}
		vars := map[string]any{"cardId": cardID, "fromAccountId": fromID, "amount": amount}
		if date != "" {
			vars["date"] = date
		}
		if key != "" {
			vars["idempotencyKey"] = key
		}
		client, _, _, err := requireSessionClient(state)
		if err != nil {
			return err
		}
		var data struct {
			PayCreditCard bool `json:"payCreditCard"`
		}
		if err = client.Do(context.Background(), payCreditCardMutation, vars, &data); err != nil {
			return err
		}
		result := map[string]any{"paid": data.PayCreditCard, "cardId": cardID, "fromAccountId": fromID, "amount": amount}
		if state.jsonOutput {
			return writeJSON(result)
		}
		return writeHuman("Recorded %.2f payment to card %s\n", amount, cardID)
	}}
	cmd.Flags().StringVar(&cardID, "card-id", "", "credit card account ID")
	cmd.Flags().StringVar(&fromID, "from-account-id", "", "source account ID")
	cmd.Flags().Float64Var(&amount, "amount", 0, "payment amount")
	cmd.Flags().StringVar(&date, "date", "", "ISO date or datetime")
	cmd.Flags().StringVar(&key, "idempotency-key", "", "safe retry key")
	return cmd
}

const payCreditCardMutation = `mutation PayCreditCard($cardId: ID!, $fromAccountId: ID!, $amount: Float!, $date: String, $idempotencyKey: String) {
  payCreditCard(cardId: $cardId, fromAccountId: $fromAccountId, amount: $amount, date: $date, idempotencyKey: $idempotencyKey)
}`
