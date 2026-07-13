package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

func newCreditCardsCommand(state *appState) *cobra.Command {
	cmd := &cobra.Command{Use: "credit-cards", Short: "Manage grouped multi-currency credit cards"}
	cmd.AddCommand(
		newCreditCardCreateCommand(state), newCreditCardsListCommand(state), newCreditCardGetCommand(state),
		newCreditCardUpdateCommand(state), newCreditCardLifecycleCommand(state, "archive"),
		newCreditCardLifecycleCommand(state, "reactivate"), newCreditCardDeleteCommand(state),
		newCreditCardCurrenciesCommand(state), newCreditCardPaymentCommand(state),
	)
	return cmd
}

type creditCardCurrencyInput struct {
	Currency                string   `json:"currency"`
	Balance                 *float64 `json:"balance,omitempty"`
	CreditLimit             *float64 `json:"creditLimit,omitempty"`
	MinimumPayment          *float64 `json:"minimumPayment,omitempty"`
	TEA                     *float64 `json:"tea,omitempty"`
	TCEA                    *float64 `json:"tcea,omitempty"`
	CashAdvanceRate         *float64 `json:"cashAdvanceRate,omitempty"`
	AnnualFee               *float64 `json:"annualFee,omitempty"`
	ClosingDay              *int     `json:"closingDay,omitempty"`
	DueDay                  *int     `json:"dueDay,omitempty"`
	NextClosingDate         *string  `json:"nextClosingDate,omitempty"`
	NextDueDate             *string  `json:"nextDueDate,omitempty"`
	DefaultPaymentAccountID *string  `json:"defaultPaymentAccountId,omitempty"`
}

type creditCardFlags struct {
	Name, Issuer, Status, CurrenciesFile string
}

type cardCurrencyFlags struct {
	Currency, DefaultPaymentAccountID, NextClosingDate, NextDueDate             string
	Balance, CreditLimit, MinimumPayment, TEA, TCEA, CashAdvanceRate, AnnualFee float64
	ClosingDay, DueDay                                                          int
}

func bindCardCurrencyFlags(cmd *cobra.Command, f *cardCurrencyFlags, includeBalance bool) {
	cmd.Flags().StringVar(&f.Currency, "currency", "", "currency ledger (PEN, USD, EUR)")
	if includeBalance {
		cmd.Flags().Float64Var(&f.Balance, "balance", 0, "initial balance")
	}
	cmd.Flags().Float64Var(&f.CreditLimit, "credit-limit", 0, "credit limit")
	cmd.Flags().Float64Var(&f.MinimumPayment, "minimum-payment", 0, "minimum payment")
	cmd.Flags().Float64Var(&f.TEA, "tea", 0, "purchase TEA as decimal (0.45 = 45%)")
	cmd.Flags().Float64Var(&f.TCEA, "tcea", 0, "TCEA as decimal (0.55 = 55%)")
	cmd.Flags().Float64Var(&f.CashAdvanceRate, "cash-advance-rate", 0, "cash advance rate as decimal")
	cmd.Flags().Float64Var(&f.AnnualFee, "annual-fee", 0, "annual fee")
	cmd.Flags().IntVar(&f.ClosingDay, "closing-day", 0, "recurring closing day")
	cmd.Flags().IntVar(&f.DueDay, "due-day", 0, "recurring due day")
	cmd.Flags().StringVar(&f.NextClosingDate, "next-closing-date", "", "next closing date (YYYY-MM-DD)")
	cmd.Flags().StringVar(&f.NextDueDate, "next-due-date", "", "next due date (YYYY-MM-DD)")
	cmd.Flags().StringVar(&f.DefaultPaymentAccountID, "default-payment-account-id", "", "default source account ID")
}

func currencyInputFromFlags(f cardCurrencyFlags, cmd *cobra.Command, update bool) (creditCardCurrencyInput, error) {
	input := creditCardCurrencyInput{Currency: strings.ToUpper(strings.TrimSpace(f.Currency))}
	if input.Currency == "" {
		return input, fmt.Errorf("--currency is required")
	}
	putFloat := func(flag string, value float64, target **float64) {
		if !update || cmd.Flags().Changed(flag) {
			v := value
			*target = &v
		}
	}
	putInt := func(flag string, value int, target **int) {
		if !update || cmd.Flags().Changed(flag) {
			v := value
			*target = &v
		}
	}
	putString := func(flag, value string, target **string) {
		if !update || cmd.Flags().Changed(flag) {
			v := strings.TrimSpace(value)
			if v != "" {
				*target = &v
			}
		}
	}
	if !update || cmd.Flags().Changed("balance") {
		v := f.Balance
		input.Balance = &v
	}
	putFloat("credit-limit", f.CreditLimit, &input.CreditLimit)
	putFloat("minimum-payment", f.MinimumPayment, &input.MinimumPayment)
	putFloat("tea", f.TEA, &input.TEA)
	putFloat("tcea", f.TCEA, &input.TCEA)
	putFloat("cash-advance-rate", f.CashAdvanceRate, &input.CashAdvanceRate)
	putFloat("annual-fee", f.AnnualFee, &input.AnnualFee)
	putInt("closing-day", f.ClosingDay, &input.ClosingDay)
	putInt("due-day", f.DueDay, &input.DueDay)
	putString("next-closing-date", f.NextClosingDate, &input.NextClosingDate)
	putString("next-due-date", f.NextDueDate, &input.NextDueDate)
	putString("default-payment-account-id", f.DefaultPaymentAccountID, &input.DefaultPaymentAccountID)
	return input, nil
}

func readCurrencyInputs(path string) ([]creditCardCurrencyInput, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read currencies file: %w", err)
	}
	var inputs []creditCardCurrencyInput
	if err := json.Unmarshal(raw, &inputs); err != nil {
		return nil, fmt.Errorf("parse currencies JSON: %w", err)
	}
	if len(inputs) == 0 {
		return nil, fmt.Errorf("currencies file must contain at least one currency")
	}
	seen := map[string]bool{}
	for i := range inputs {
		inputs[i].Currency = strings.ToUpper(strings.TrimSpace(inputs[i].Currency))
		if inputs[i].Currency == "" || seen[inputs[i].Currency] {
			return nil, fmt.Errorf("currencies file contains an invalid or duplicate currency")
		}
		seen[inputs[i].Currency] = true
	}
	return inputs, nil
}

func newCreditCardCreateCommand(state *appState) *cobra.Command {
	f := creditCardFlags{}
	cmd := &cobra.Command{Use: "create", Short: "Create a grouped credit card", RunE: func(cmd *cobra.Command, args []string) error {
		if strings.TrimSpace(f.Name) == "" {
			return fmt.Errorf("--name is required")
		}
		if strings.TrimSpace(f.CurrenciesFile) == "" {
			return fmt.Errorf("--currencies-file is required (JSON array of currency ledgers)")
		}
		currencies, err := readCurrencyInputs(f.CurrenciesFile)
		if err != nil {
			return err
		}
		client, _, _, err := requireSessionClient(state)
		if err != nil {
			return err
		}
		var data struct {
			CreateCreditCard creditCard `json:"createCreditCard"`
		}
		if err = client.Do(context.Background(), createCreditCardMutation, map[string]any{"input": map[string]any{"name": strings.TrimSpace(f.Name), "issuer": strings.TrimSpace(f.Issuer), "currencies": currencies}}, &data); err != nil {
			return err
		}
		if state.jsonOutput {
			return writeJSON(data.CreateCreditCard)
		}
		return writeHuman("Created credit card %s\n", data.CreateCreditCard.ID)
	}}
	cmd.Flags().StringVar(&f.Name, "name", "", "card name")
	cmd.Flags().StringVar(&f.Issuer, "issuer", "", "card issuer")
	cmd.Flags().StringVar(&f.CurrenciesFile, "currencies-file", "", "JSON array of currency ledgers")
	return cmd
}

func newCreditCardsListCommand(state *appState) *cobra.Command {
	status := "active"
	cmd := &cobra.Command{Use: "list", Short: "List grouped credit cards", RunE: func(cmd *cobra.Command, args []string) error {
		filter, err := cardStatusFilter(status)
		if err != nil {
			return err
		}
		client, _, _, err := requireSessionClient(state)
		if err != nil {
			return err
		}
		var data struct {
			CreditCards []creditCard `json:"creditCards"`
		}
		if err = client.Do(context.Background(), creditCardsQuery, map[string]any{"status": filter}, &data); err != nil {
			return err
		}
		if state.jsonOutput {
			return writeJSON(data.CreditCards)
		}
		for _, card := range data.CreditCards {
			if err := writeHuman("%s\t%s\t%s\t%d currencies\n", card.ID, card.Name, card.ContractStatus, len(card.Currencies)); err != nil {
				return err
			}
		}
		return nil
	}}
	cmd.Flags().StringVar(&status, "status", "active", "active, archived, or all")
	return cmd
}

func cardStatusFilter(status string) (string, error) {
	switch strings.ToUpper(strings.TrimSpace(status)) {
	case "ACTIVE":
		return "ACTIVE", nil
	case "ARCHIVED":
		return "ARCHIVED", nil
	case "ALL":
		return "ALL", nil
	}
	return "", fmt.Errorf("--status must be active, archived, or all")
}

func newCreditCardGetCommand(state *appState) *cobra.Command {
	return &cobra.Command{Use: "get <id>", Short: "Get a grouped credit card", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		client, _, _, err := requireSessionClient(state)
		if err != nil {
			return err
		}
		var data struct {
			CreditCard creditCard `json:"creditCard"`
		}
		if err = client.Do(context.Background(), creditCardQuery, map[string]any{"id": args[0]}, &data); err != nil {
			return err
		}
		if state.jsonOutput {
			return writeJSON(data.CreditCard)
		}
		return writeHuman("%s\t%s\t%s\n", data.CreditCard.ID, data.CreditCard.Name, data.CreditCard.ContractStatus)
	}}
}

func newCreditCardUpdateCommand(state *appState) *cobra.Command {
	f := creditCardFlags{}
	cmd := &cobra.Command{Use: "update <id>", Short: "Update card details or contract state", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		input := map[string]any{}
		if cmd.Flags().Changed("name") {
			input["name"] = strings.TrimSpace(f.Name)
		}
		if cmd.Flags().Changed("issuer") {
			input["issuer"] = strings.TrimSpace(f.Issuer)
		}
		if cmd.Flags().Changed("status") {
			status := strings.ToUpper(strings.TrimSpace(f.Status))
			if status != "ACTIVE" && status != "CANCELLED" {
				return fmt.Errorf("--status must be active or cancelled")
			}
			input["contractStatus"] = status
		}
		if len(input) == 0 {
			return fmt.Errorf("provide --name, --issuer, or --status")
		}
		client, _, _, err := requireSessionClient(state)
		if err != nil {
			return err
		}
		var data struct {
			UpdateCreditCard creditCard `json:"updateCreditCard"`
		}
		if err = client.Do(context.Background(), updateCreditCardMutation, map[string]any{"id": args[0], "input": input}, &data); err != nil {
			return err
		}
		if state.jsonOutput {
			return writeJSON(data.UpdateCreditCard)
		}
		return writeHuman("Updated credit card %s\n", data.UpdateCreditCard.ID)
	}}
	cmd.Flags().StringVar(&f.Name, "name", "", "card name")
	cmd.Flags().StringVar(&f.Issuer, "issuer", "", "card issuer")
	cmd.Flags().StringVar(&f.Status, "status", "", "contract status: active or cancelled")
	return cmd
}

func newCreditCardLifecycleCommand(state *appState, action string) *cobra.Command {
	mutation := archiveCreditCardMutation
	field := "archiveCreditCard"
	if action == "reactivate" {
		mutation, field = reactivateCreditCardMutation, "reactivateCreditCard"
	}
	return &cobra.Command{Use: action + " <id>", Short: strings.Title(action) + " a credit card", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		client, _, _, err := requireSessionClient(state)
		if err != nil {
			return err
		}
		var data map[string]creditCard
		if err = client.Do(context.Background(), mutation, map[string]any{"id": args[0]}, &data); err != nil {
			return err
		}
		if state.jsonOutput {
			return writeJSON(data[field])
		}
		return writeHuman("%sd credit card %s\n", strings.Title(action), args[0])
	}}
}

func newCreditCardDeleteCommand(state *appState) *cobra.Command {
	return &cobra.Command{Use: "delete <id>", Short: "Safely delete an empty credit card", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		client, _, _, err := requireSessionClient(state)
		if err != nil {
			return err
		}
		var data struct {
			DeleteCreditCard bool `json:"deleteCreditCard"`
		}
		if err = client.Do(context.Background(), deleteCreditCardMutation, map[string]any{"id": args[0]}, &data); err != nil {
			return err
		}
		if state.jsonOutput {
			return writeJSON(map[string]any{"deleted": data.DeleteCreditCard, "id": args[0]})
		}
		return writeHuman("Deleted credit card %s\n", args[0])
	}}
}

func newCreditCardCurrenciesCommand(state *appState) *cobra.Command {
	cmd := &cobra.Command{Use: "currencies", Short: "Manage card currency ledgers"}
	cmd.AddCommand(newCreditCardCurrencyAddCommand(state), newCreditCardCurrencyUpdateCommand(state), newCreditCardCurrencyRemoveCommand(state), newCreditCardCurrencyDefaultCommand(state))
	return cmd
}

func newCreditCardCurrencyAddCommand(state *appState) *cobra.Command {
	f := cardCurrencyFlags{}
	cmd := &cobra.Command{Use: "add <card-id>", Short: "Add a currency ledger", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		input, err := currencyInputFromFlags(f, cmd, false)
		if err != nil {
			return err
		}
		client, _, _, err := requireSessionClient(state)
		if err != nil {
			return err
		}
		var data struct {
			AddCreditCardCurrency account `json:"addCreditCardCurrency"`
		}
		if err = client.Do(context.Background(), addCreditCardCurrencyMutation, map[string]any{"cardId": args[0], "input": input}, &data); err != nil {
			return err
		}
		if state.jsonOutput {
			return writeJSON(data.AddCreditCardCurrency)
		}
		return writeHuman("Added %s ledger to card %s\n", input.Currency, args[0])
	}}
	bindCardCurrencyFlags(cmd, &f, true)
	return cmd
}

func newCreditCardCurrencyUpdateCommand(state *appState) *cobra.Command {
	f := cardCurrencyFlags{}
	cmd := &cobra.Command{Use: "update <card-id>", Short: "Update one currency ledger", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		input, err := currencyInputFromFlags(f, cmd, true)
		if err != nil {
			return err
		}
		raw, _ := json.Marshal(input)
		var fields map[string]any
		_ = json.Unmarshal(raw, &fields)
		delete(fields, "currency")
		delete(fields, "balance")
		if len(fields) == 0 {
			return fmt.Errorf("provide a currency field to update")
		}
		client, _, _, err := requireSessionClient(state)
		if err != nil {
			return err
		}
		var data struct {
			UpdateCreditCardCurrency account `json:"updateCreditCardCurrency"`
		}
		if err = client.Do(context.Background(), updateCreditCardCurrencyMutation, map[string]any{"cardId": args[0], "currency": input.Currency, "input": fields}, &data); err != nil {
			return err
		}
		if state.jsonOutput {
			return writeJSON(data.UpdateCreditCardCurrency)
		}
		return writeHuman("Updated %s ledger on card %s\n", input.Currency, args[0])
	}}
	bindCardCurrencyFlags(cmd, &f, false)
	return cmd
}

func newCreditCardCurrencyRemoveCommand(state *appState) *cobra.Command {
	var currency string
	cmd := &cobra.Command{Use: "remove <card-id>", Short: "Remove an empty currency ledger", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		if strings.TrimSpace(currency) == "" {
			return fmt.Errorf("--currency is required")
		}
		client, _, _, err := requireSessionClient(state)
		if err != nil {
			return err
		}
		var data struct {
			RemoveCreditCardCurrency bool `json:"removeCreditCardCurrency"`
		}
		if err = client.Do(context.Background(), removeCreditCardCurrencyMutation, map[string]any{"cardId": args[0], "currency": strings.ToUpper(currency)}, &data); err != nil {
			return err
		}
		if state.jsonOutput {
			return writeJSON(map[string]any{"removed": data.RemoveCreditCardCurrency, "currency": strings.ToUpper(currency)})
		}
		return writeHuman("Removed %s ledger from card %s\n", strings.ToUpper(currency), args[0])
	}}
	cmd.Flags().StringVar(&currency, "currency", "", "currency ledger")
	return cmd
}

func newCreditCardCurrencyDefaultCommand(state *appState) *cobra.Command {
	var currency, accountID string
	cmd := &cobra.Command{Use: "set-default <card-id>", Short: "Set a currency ledger's default payment account", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		if currency == "" || accountID == "" {
			return fmt.Errorf("--currency and --account-id are required")
		}
		client, _, _, err := requireSessionClient(state)
		if err != nil {
			return err
		}
		var data struct {
			UpdateCreditCardCurrency account `json:"updateCreditCardCurrency"`
		}
		if err = client.Do(context.Background(), updateCreditCardCurrencyMutation, map[string]any{"cardId": args[0], "currency": strings.ToUpper(currency), "input": map[string]any{"defaultPaymentAccountId": accountID}}, &data); err != nil {
			return err
		}
		if state.jsonOutput {
			return writeJSON(data.UpdateCreditCardCurrency)
		}
		return writeHuman("Set default payment account for %s on card %s\n", strings.ToUpper(currency), args[0])
	}}
	cmd.Flags().StringVar(&currency, "currency", "", "currency ledger")
	cmd.Flags().StringVar(&accountID, "account-id", "", "source account ID")
	return cmd
}

func newCreditCardPaymentCommand(state *appState) *cobra.Command {
	var cardID, fromID, currency, date, key string
	var amount, debited, applied, rate float64
	cmd := &cobra.Command{Use: "payment", Short: "Record an idempotent credit card payment", RunE: func(cmd *cobra.Command, args []string) error {
		if cardID == "" || fromID == "" {
			return fmt.Errorf("--card-id and --from-account-id are required")
		}
		client, _, _, err := requireSessionClient(state)
		if err != nil {
			return err
		}
		if currency == "" {
			if amount <= 0 {
				return fmt.Errorf("--amount is required for the legacy payment form")
			}
			vars := map[string]any{"cardId": cardID, "fromAccountId": fromID, "amount": amount}
			if date != "" {
				vars["date"] = date
			}
			if key != "" {
				vars["idempotencyKey"] = key
			}
			var data struct {
				PayCreditCard bool `json:"payCreditCard"`
			}
			if err = client.Do(context.Background(), legacyPayCreditCardMutation, vars, &data); err != nil {
				return err
			}
			if state.jsonOutput {
				return writeJSON(map[string]any{"paid": data.PayCreditCard, "cardId": cardID, "amount": amount})
			}
			return writeHuman("Recorded %.2f payment to card %s\n", amount, cardID)
		}
		if debited <= 0 || applied <= 0 {
			return fmt.Errorf("--debit-amount and --applied-amount must be greater than zero")
		}
		if strings.TrimSpace(key) == "" {
			return fmt.Errorf("--idempotency-key is required for currency payments")
		}
		if debited != applied && !cmd.Flags().Changed("exchange-rate") {
			return fmt.Errorf("--exchange-rate is required when debit and applied amounts differ")
		}
		input := map[string]any{"cardId": cardID, "currency": strings.ToUpper(currency), "fromAccountId": fromID, "debitedAmount": debited, "appliedAmount": applied, "idempotencyKey": key}
		if date != "" {
			input["date"] = date
		}
		if cmd.Flags().Changed("exchange-rate") {
			if rate <= 0 {
				return fmt.Errorf("--exchange-rate must be greater than zero")
			}
			input["exchangeRate"] = rate
		}
		var data struct {
			PayCreditCardPayment creditCardPayment `json:"payCreditCardPayment"`
		}
		if err = client.Do(context.Background(), payCreditCardMutation, map[string]any{"input": input}, &data); err != nil {
			return err
		}
		if state.jsonOutput {
			return writeJSON(data.PayCreditCardPayment)
		}
		return writeHuman("Recorded %.2f applied to %s card balance\n", applied, strings.ToUpper(currency))
	}}
	cmd.Flags().StringVar(&cardID, "card-id", "", "credit card ID")
	cmd.Flags().StringVar(&fromID, "from-account-id", "", "source account ID")
	cmd.Flags().StringVar(&currency, "currency", "", "debt currency (uses new payment contract)")
	cmd.Flags().Float64Var(&amount, "amount", 0, "legacy same-currency payment amount")
	cmd.Flags().Float64Var(&debited, "debit-amount", 0, "amount debited from source account")
	cmd.Flags().Float64Var(&applied, "applied-amount", 0, "amount applied to card debt")
	cmd.Flags().Float64Var(&rate, "exchange-rate", 0, "bank FX rate from source to debt currency")
	cmd.Flags().StringVar(&date, "date", "", "ISO date or datetime")
	cmd.Flags().StringVar(&key, "idempotency-key", "", "required safe retry key for currency payments")
	return cmd
}

const creditCardFields = `id name issuer contractStatus archivedAt createdAt updatedAt currencies { id currency balance isActive creditLimit minimumPayment tea tcea cashAdvanceRate annualFee closingDay dueDay nextClosingDate nextDueDate defaultPaymentAccountId }`
const creditCardsQuery = `query CreditCards($status: String!) { creditCards(status: $status) { ` + creditCardFields + ` } }`
const creditCardQuery = `query CreditCard($id: ID!) { creditCard(id: $id) { ` + creditCardFields + ` } }`
const createCreditCardMutation = `mutation CreateCreditCard($input: CreateCreditCardInput!) { createCreditCard(input: $input) { ` + creditCardFields + ` } }`
const updateCreditCardMutation = `mutation UpdateCreditCard($id: ID!, $input: UpdateCreditCardInput!) { updateCreditCard(id: $id, input: $input) { ` + creditCardFields + ` } }`
const addCreditCardCurrencyMutation = `mutation AddCreditCardCurrency($cardId: ID!, $input: CreditCardCurrencyInput!) { addCreditCardCurrency(cardId: $cardId, input: $input) { ` + accountFields + ` } }`
const updateCreditCardCurrencyMutation = `mutation UpdateCreditCardCurrency($cardId: ID!, $currency: String!, $input: UpdateCreditCardCurrencyInput!) { updateCreditCardCurrency(cardId: $cardId, currency: $currency, input: $input) { ` + accountFields + ` } }`
const removeCreditCardCurrencyMutation = `mutation RemoveCreditCardCurrency($cardId: ID!, $currency: String!) { removeCreditCardCurrency(cardId: $cardId, currency: $currency) }`
const archiveCreditCardMutation = `mutation ArchiveCreditCard($id: ID!) { archiveCreditCard(id: $id) { ` + creditCardFields + ` } }`
const reactivateCreditCardMutation = `mutation ReactivateCreditCard($id: ID!) { reactivateCreditCard(id: $id) { ` + creditCardFields + ` } }`
const deleteCreditCardMutation = `mutation DeleteCreditCard($id: ID!) { deleteCreditCard(id: $id) }`
const payCreditCardMutation = `mutation PayCreditCard($input: PayCreditCardInput!) { payCreditCardPayment(input: $input) { id debitedAmount appliedAmount exchangeRate idempotencyKey createdAt } }`
const legacyPayCreditCardMutation = `mutation PayCreditCard($cardId: ID!, $fromAccountId: ID!, $amount: Float!, $date: String, $idempotencyKey: String) { payCreditCard(cardId: $cardId, fromAccountId: $fromAccountId, amount: $amount, date: $date, idempotencyKey: $idempotencyKey) }`
