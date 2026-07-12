package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

func newTransactionsCommand(state *appState) *cobra.Command {
	cmd := &cobra.Command{Use: "transactions", Short: "Manage transactions"}
	cmd.AddCommand(newTransactionsListCommand(state))
	cmd.AddCommand(newTransactionsCreateCommand(state))
	cmd.AddCommand(newTransactionsDraftCommand(state))
	cmd.AddCommand(newTransactionsTransferCommand(state))
	return cmd
}

func newTransactionsListCommand(state *appState) *cobra.Command {
	var accountID string
	var dateFrom string
	var dateTo string
	var limit int
	var offset int
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List transactions",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, _, _, err := requireSessionClient(state)
			if err != nil {
				return err
			}
			vars := map[string]any{"limit": limit, "offset": offset}
			if accountID != "" {
				vars["accountId"] = accountID
			}
			if dateFrom != "" {
				vars["dateFrom"] = dateFrom
			}
			if dateTo != "" {
				vars["dateTo"] = dateTo
			}
			var data struct {
				Transactions []transaction `json:"transactions"`
			}
			if err := client.Do(context.Background(), transactionsQuery, vars, &data); err != nil {
				return err
			}
			if state.jsonOutput {
				return writeJSON(data.Transactions)
			}
			for _, txn := range data.Transactions {
				if err := writeHuman("%s\t%s\t%.2f\t%s\t%s\n", txn.ID, txn.Kind, float64(txn.Amount), txn.OriginalCurrency, txn.Description); err != nil {
					return err
				}
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&accountID, "account-id", "", "filter by account ID")
	cmd.Flags().StringVar(&dateFrom, "from", "", "inclusive ISO date/datetime lower bound")
	cmd.Flags().StringVar(&dateTo, "to", "", "inclusive ISO date/datetime upper bound")
	cmd.Flags().IntVar(&limit, "limit", 50, "maximum number of transactions")
	cmd.Flags().IntVar(&offset, "offset", 0, "offset for pagination")
	return cmd
}

func newTransactionsCreateCommand(state *appState) *cobra.Command {
	input := transactionCreateFlags{}
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a transaction",
		RunE: func(cmd *cobra.Command, args []string) error {
			if input.AccountID == "" {
				return fmt.Errorf("--account-id is required")
			}
			if input.Amount <= 0 {
				return fmt.Errorf("--amount must be greater than zero")
			}
			input.Kind = strings.ToUpper(strings.TrimSpace(input.Kind))
			if input.Kind != "INCOME" && input.Kind != "EXPENSE" {
				return fmt.Errorf("--kind must be INCOME or EXPENSE")
			}
			if input.Date == "" {
				input.Date = time.Now().Format(time.RFC3339)
			}
			variables := map[string]any{"input": input.toGraphQLInput()}
			client, _, _, err := requireSessionClient(state)
			if err != nil {
				return err
			}
			var data struct {
				CreateTransaction transaction `json:"createTransaction"`
			}
			if err := client.Do(context.Background(), createTransactionMutation, variables, &data); err != nil {
				return err
			}
			if state.jsonOutput {
				return writeJSON(data.CreateTransaction)
			}
			return writeHuman("Created transaction %s\n", data.CreateTransaction.ID)
		},
	}
	bindTransactionCreateFlags(cmd, &input)
	return cmd
}

type transactionCreateFlags struct {
	AccountID      string
	Amount         float64
	Kind           string
	Currency       string
	Date           string
	CategoryID     string
	Description    string
	ExchangeRate   float64
	TagIDs         []string
	TagNames       []string
	IdempotencyKey string
}

func bindTransactionCreateFlags(cmd *cobra.Command, input *transactionCreateFlags) {
	cmd.Flags().StringVar(&input.AccountID, "account-id", "", "account ID")
	cmd.Flags().Float64Var(&input.Amount, "amount", 0, "transaction amount")
	cmd.Flags().StringVar(&input.Kind, "kind", "EXPENSE", "transaction kind: INCOME or EXPENSE")
	cmd.Flags().StringVar(&input.Currency, "currency", "", "original currency; defaults to account currency")
	cmd.Flags().StringVar(&input.Date, "date", "", "ISO datetime; defaults to now")
	cmd.Flags().StringVar(&input.CategoryID, "category-id", "", "category ID")
	cmd.Flags().StringVar(&input.Description, "description", "", "description")
	cmd.Flags().Float64Var(&input.ExchangeRate, "exchange-rate", 0, "exchange rate when currency differs from account")
	cmd.Flags().StringArrayVar(&input.TagIDs, "tag-id", nil, "tag ID; may be repeated")
	cmd.Flags().StringArrayVar(&input.TagNames, "tag", nil, "tag name; may be repeated")
	cmd.Flags().StringVar(&input.IdempotencyKey, "idempotency-key", "", "safe retry key")
}

func (f transactionCreateFlags) toGraphQLInput() map[string]any {
	input := map[string]any{
		"accountId":        f.AccountID,
		"amount":           f.Amount,
		"kind":             f.Kind,
		"date":             f.Date,
		"originalCurrency": f.Currency,
		"description":      f.Description,
	}
	if f.CategoryID != "" {
		input["categoryId"] = f.CategoryID
	}
	if f.ExchangeRate > 0 {
		input["exchangeRate"] = f.ExchangeRate
	}
	if len(f.TagIDs) > 0 {
		input["tagIds"] = f.TagIDs
	}
	if len(f.TagNames) > 0 {
		input["tagNames"] = f.TagNames
	}
	if f.IdempotencyKey != "" {
		input["idempotencyKey"] = f.IdempotencyKey
	}
	return input
}

func newTransactionsTransferCommand(state *appState) *cobra.Command {
	var fromID, toID, note, key string
	var amount float64
	cmd := &cobra.Command{Use: "transfer", Short: "Transfer money between accounts", RunE: func(cmd *cobra.Command, args []string) error {
		if fromID == "" {
			return fmt.Errorf("--from-account-id is required")
		}
		if toID == "" {
			return fmt.Errorf("--to-account-id is required")
		}
		if fromID == toID {
			return fmt.Errorf("source and destination accounts must differ")
		}
		if amount <= 0 {
			return fmt.Errorf("--amount must be greater than zero")
		}
		input := map[string]any{"fromAccountId": fromID, "toAccountId": toID, "amount": amount, "note": note}
		if key != "" {
			input["idempotencyKey"] = key
		}
		client, _, _, err := requireSessionClient(state)
		if err != nil {
			return err
		}
		var data struct {
			CreateTransfer bool `json:"createTransfer"`
		}
		if err = client.Do(context.Background(), createTransferMutation, map[string]any{"input": input}, &data); err != nil {
			return err
		}
		result := map[string]any{"transferred": data.CreateTransfer, "fromAccountId": fromID, "toAccountId": toID, "amount": amount}
		if state.jsonOutput {
			return writeJSON(result)
		}
		return writeHuman("Transferred %.2f from %s to %s\n", amount, fromID, toID)
	}}
	cmd.Flags().StringVar(&fromID, "from-account-id", "", "source account ID")
	cmd.Flags().StringVar(&toID, "to-account-id", "", "destination account ID")
	cmd.Flags().Float64Var(&amount, "amount", 0, "transfer amount")
	cmd.Flags().StringVar(&note, "note", "", "transfer note")
	cmd.Flags().StringVar(&key, "idempotency-key", "", "safe retry key")
	return cmd
}

type transactionDraftInput struct {
	AccountID    string      `json:"accountId,omitempty"`
	AccountName  string      `json:"account,omitempty"`
	Merchant     string      `json:"merchant,omitempty"`
	Date         string      `json:"date,omitempty"`
	Items        []draftItem `json:"items,omitempty"`
	Total        float64     `json:"total"`
	Currency     string      `json:"currency,omitempty"`
	CategoryID   string      `json:"categoryId,omitempty"`
	CategoryName string      `json:"category,omitempty"`
	Description  string      `json:"description,omitempty"`
	Tags         []string    `json:"tags,omitempty"`
}

type draftItem struct {
	Name   string  `json:"name"`
	Amount float64 `json:"amount"`
}

type transactionDraftPreview struct {
	Status        string      `json:"status"`
	Account       *account    `json:"account,omitempty"`
	AccountInput  string      `json:"accountInput,omitempty"`
	Amount        float64     `json:"amount"`
	Kind          string      `json:"kind"`
	Currency      string      `json:"currency,omitempty"`
	Category      *category   `json:"category,omitempty"`
	CategoryInput string      `json:"categoryInput,omitempty"`
	Description   string      `json:"description"`
	Items         []draftItem `json:"items,omitempty"`
	Tags          []tag       `json:"tags,omitempty"`
	TagInputs     []string    `json:"tagInputs,omitempty"`
	ReadyToCommit bool        `json:"readyToCommit"`
	Warnings      []string    `json:"warnings,omitempty"`
}

func newTransactionsDraftCommand(state *appState) *cobra.Command {
	var file string
	cmd := &cobra.Command{
		Use:   "draft",
		Short: "Normalize a transaction draft from agent-extracted JSON",
		RunE: func(cmd *cobra.Command, args []string) error {
			var raw []byte
			var err error
			if file == "" || file == "-" {
				text, readErr := readAll(os.Stdin)
				err = readErr
				raw = []byte(text)
			} else {
				raw, err = os.ReadFile(file)
			}
			if err != nil {
				return fmt.Errorf("read draft: %w", err)
			}
			var draft transactionDraftInput
			if err := json.Unmarshal(raw, &draft); err != nil {
				return fmt.Errorf("parse draft JSON: %w", err)
			}
			preview, err := buildDraftPreview(state, draft)
			if err != nil {
				return err
			}
			return writeJSON(preview)
		},
	}
	cmd.Flags().StringVar(&file, "file", "-", "draft JSON file, or - for stdin")
	return cmd
}

func buildDraftPreview(state *appState, draft transactionDraftInput) (transactionDraftPreview, error) {
	client, _, _, err := requireSessionClient(state)
	if err != nil {
		return transactionDraftPreview{}, err
	}
	preview := transactionDraftPreview{
		Status:        "draft",
		Amount:        draft.Total,
		Kind:          "EXPENSE",
		Currency:      draft.Currency,
		Items:         draft.Items,
		TagInputs:     draft.Tags,
		AccountInput:  draft.AccountName,
		CategoryInput: draft.CategoryName,
	}
	if draft.Description != "" {
		preview.Description = draft.Description
	} else {
		preview.Description = buildDraftDescription(draft)
	}
	itemTotal := 0.0
	for _, item := range draft.Items {
		itemTotal += item.Amount
	}
	if len(draft.Items) > 0 && abs(itemTotal-draft.Total) > 0.01 {
		preview.Warnings = append(preview.Warnings, fmt.Sprintf("item total %.2f does not match draft total %.2f", itemTotal, draft.Total))
	}
	if draft.Total <= 0 {
		preview.Warnings = append(preview.Warnings, "total must be greater than zero")
	}

	accounts, err := fetchAccounts(client)
	if err != nil {
		return preview, err
	}
	preview.Account = matchAccount(accounts, draft.AccountID, draft.AccountName)
	if preview.Account == nil {
		preview.Warnings = append(preview.Warnings, "account could not be resolved")
	}

	categories, err := fetchCategories(client, draft.CategoryName)
	if err != nil {
		return preview, err
	}
	preview.Category = matchCategory(categories, draft.CategoryID, draft.CategoryName)
	if preview.Category == nil && draft.CategoryName != "" {
		preview.Warnings = append(preview.Warnings, "category could not be resolved")
	}

	for _, tagName := range draft.Tags {
		tags, err := fetchTags(client, tagName)
		if err != nil {
			return preview, err
		}
		if len(tags) > 0 {
			preview.Tags = append(preview.Tags, tags[0])
		}
	}
	preview.ReadyToCommit = preview.Account != nil && draft.Total > 0
	return preview, nil
}

func buildDraftDescription(draft transactionDraftInput) string {
	names := make([]string, 0, len(draft.Items))
	for _, item := range draft.Items {
		if item.Name != "" {
			names = append(names, item.Name)
		}
	}
	if draft.Merchant != "" && len(names) > 0 {
		return draft.Merchant + ": " + strings.Join(names, ", ")
	}
	if draft.Merchant != "" {
		return draft.Merchant
	}
	return strings.Join(names, ", ")
}

func fetchAccounts(client interface {
	Do(context.Context, string, map[string]any, any) error
}) ([]account, error) {
	var data struct {
		Accounts []account `json:"accounts"`
	}
	err := client.Do(context.Background(), accountsQuery, nil, &data)
	return data.Accounts, err
}

func fetchCategories(client interface {
	Do(context.Context, string, map[string]any, any) error
}, query string) ([]category, error) {
	var data struct {
		SearchCategories []category `json:"searchCategories"`
	}
	err := client.Do(context.Background(), categoriesQuery, map[string]any{"query": query}, &data)
	return data.SearchCategories, err
}

func fetchTags(client interface {
	Do(context.Context, string, map[string]any, any) error
}, query string) ([]tag, error) {
	var data struct {
		MyTags []tag `json:"myTags"`
	}
	err := client.Do(context.Background(), tagsQuery, map[string]any{"query": query}, &data)
	return data.MyTags, err
}

func matchAccount(accounts []account, id string, name string) *account {
	for _, candidate := range accounts {
		if id != "" && candidate.ID == id {
			value := candidate
			return &value
		}
	}
	for _, candidate := range accounts {
		if strings.EqualFold(candidate.Name, name) {
			value := candidate
			return &value
		}
	}
	return nil
}

func matchCategory(categories []category, id string, name string) *category {
	for _, candidate := range categories {
		if id != "" && candidate.ID == id {
			value := candidate
			return &value
		}
	}
	for _, candidate := range categories {
		if strings.EqualFold(candidate.Name, name) {
			value := candidate
			return &value
		}
	}
	if len(categories) > 0 && name != "" {
		return &categories[0]
	}
	return nil
}

func abs(value float64) float64 {
	if value < 0 {
		return -value
	}
	return value
}

const transactionFields = `
id
accountId
amount
kind
originalCurrency
exchangeRate
description
date
transactionType
idempotencyKey
account { ` + accountFields + ` }
category { ` + categoryFields + ` }
tags { ` + tagFields + ` }`

const transactionsQuery = `
query Transactions($accountId: ID, $dateFrom: DateTime, $dateTo: DateTime, $limit: Int!, $offset: Int!) {
  transactions(accountId: $accountId, dateFrom: $dateFrom, dateTo: $dateTo, limit: $limit, offset: $offset) {
    ` + transactionFields + `
  }
}`

const createTransactionMutation = `
mutation CreateTransaction($input: CreateTransactionInput!) {
  createTransaction(input: $input) {
    ` + transactionFields + `
  }
}`

const createTransferMutation = `mutation Transfer($input:CreateTransferInput!){createTransfer(input:$input)}`
