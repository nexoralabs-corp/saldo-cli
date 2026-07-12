package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

type registrationImport struct {
	AccountID      string   `json:"accountId"`
	Amount         float64  `json:"amount"`
	Kind           string   `json:"kind,omitempty"`
	Currency       string   `json:"currency,omitempty"`
	Date           string   `json:"date"`
	CategoryID     string   `json:"categoryId,omitempty"`
	Description    string   `json:"description,omitempty"`
	ExchangeRate   float64  `json:"exchangeRate,omitempty"`
	TagIDs         []string `json:"tagIds,omitempty"`
	Tags           []string `json:"tags,omitempty"`
	IdempotencyKey string   `json:"idempotencyKey"`
}

type importItemResult struct {
	Index          int          `json:"index"`
	IdempotencyKey string       `json:"idempotencyKey,omitempty"`
	Status         string       `json:"status"`
	Transaction    *transaction `json:"transaction,omitempty"`
	Errors         []string     `json:"errors,omitempty"`
}
type importPreview struct {
	DryRun bool               `json:"dryRun"`
	Valid  bool               `json:"valid"`
	Total  int                `json:"total"`
	Ready  int                `json:"ready"`
	Items  []importItemResult `json:"items"`
}

func newImportCommand(state *appState) *cobra.Command {
	cmd := &cobra.Command{Use: "import", Short: "Safely import data in bulk"}
	cmd.AddCommand(newImportRegistrationsCommand(state))
	return cmd
}

func newImportRegistrationsCommand(state *appState) *cobra.Command {
	var file string
	var dryRun bool
	cmd := &cobra.Command{Use: "registrations", Short: "Import transaction registrations from JSON", RunE: func(cmd *cobra.Command, args []string) error {
		raw, err := os.ReadFile(file)
		if err != nil {
			return fmt.Errorf("read import file: %w", err)
		}
		items, err := decodeRegistrations(raw)
		if err != nil {
			return err
		}
		preview := validateRegistrations(items, dryRun)
		if !preview.Valid || dryRun {
			return writeJSON(preview)
		}
		client, _, _, err := requireSessionClient(state)
		if err != nil {
			return err
		}
		for i, item := range items {
			input := registrationGraphQLInput(item)
			var data struct {
				CreateTransaction transaction `json:"createTransaction"`
			}
			if err := client.Do(context.Background(), createTransactionMutation, map[string]any{"input": input}, &data); err != nil {
				preview.Valid = false
				preview.Items[i].Status = "failed"
				preview.Items[i].Errors = []string{err.Error()}
				continue
			}
			preview.Items[i].Status = "imported"
			preview.Items[i].Transaction = &data.CreateTransaction
		}
		if state.jsonOutput {
			return writeJSON(preview)
		}
		failed := 0
		for _, item := range preview.Items {
			if item.Status == "failed" {
				failed++
			}
		}
		if failed > 0 {
			return fmt.Errorf("import completed with %d failed registration(s); rerun safely with the same idempotency keys", failed)
		}
		return writeHuman("Imported %d registrations\n", len(items))
	}}
	cmd.Flags().StringVar(&file, "file", "", "JSON file")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "validate and preview without writing")
	_ = cmd.MarkFlagRequired("file")
	return cmd
}

func decodeRegistrations(raw []byte) ([]registrationImport, error) {
	var items []registrationImport
	if len(bytes.TrimSpace(raw)) == 0 {
		return nil, fmt.Errorf("import file is empty")
	}
	if bytes.HasPrefix(bytes.TrimSpace(raw), []byte("[")) {
		if err := json.Unmarshal(raw, &items); err != nil {
			return nil, fmt.Errorf("parse registrations: %w", err)
		}
		return items, nil
	}
	var envelope struct {
		Registrations []registrationImport `json:"registrations"`
	}
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return nil, fmt.Errorf("parse registrations: %w", err)
	}
	return envelope.Registrations, nil
}

func validateRegistrations(items []registrationImport, dryRun bool) importPreview {
	result := importPreview{DryRun: dryRun, Valid: true, Total: len(items), Items: make([]importItemResult, len(items))}
	keys := map[string]int{}
	fingerprints := map[string]int{}
	if len(items) == 0 {
		result.Valid = false
	}
	for i, item := range items {
		entry := importItemResult{Index: i, IdempotencyKey: item.IdempotencyKey, Status: "ready"}
		kind := strings.ToUpper(strings.TrimSpace(item.Kind))
		if kind == "" {
			kind = "EXPENSE"
		}
		if item.AccountID == "" {
			entry.Errors = append(entry.Errors, "accountId is required")
		}
		if item.Amount <= 0 {
			entry.Errors = append(entry.Errors, "amount must be greater than zero")
		}
		if kind != "INCOME" && kind != "EXPENSE" {
			entry.Errors = append(entry.Errors, "kind must be INCOME or EXPENSE")
		}
		if item.Date == "" {
			entry.Errors = append(entry.Errors, "date is required")
		}
		key := strings.TrimSpace(item.IdempotencyKey)
		if key == "" {
			entry.Errors = append(entry.Errors, "idempotencyKey is required")
		} else if first, ok := keys[key]; ok {
			entry.Errors = append(entry.Errors, fmt.Sprintf("duplicate idempotencyKey; first used at index %d", first))
		} else {
			keys[key] = i
		}
		fingerprint := fmt.Sprintf("%s|%.2f|%s|%s|%s", item.AccountID, item.Amount, kind, item.Date, strings.ToLower(strings.TrimSpace(item.Description)))
		if first, ok := fingerprints[fingerprint]; ok {
			entry.Errors = append(entry.Errors, fmt.Sprintf("duplicate registration content; first seen at index %d", first))
		} else {
			fingerprints[fingerprint] = i
		}
		if len(entry.Errors) > 0 {
			entry.Status = "invalid"
			result.Valid = false
		} else {
			result.Ready++
		}
		result.Items[i] = entry
	}
	return result
}

func registrationGraphQLInput(item registrationImport) map[string]any {
	kind := strings.ToUpper(strings.TrimSpace(item.Kind))
	if kind == "" {
		kind = "EXPENSE"
	}
	input := map[string]any{"accountId": item.AccountID, "amount": item.Amount, "kind": kind, "date": item.Date, "originalCurrency": strings.ToUpper(item.Currency), "description": item.Description, "idempotencyKey": strings.TrimSpace(item.IdempotencyKey)}
	if item.CategoryID != "" {
		input["categoryId"] = item.CategoryID
	}
	if item.ExchangeRate > 0 {
		input["exchangeRate"] = item.ExchangeRate
	}
	if len(item.TagIDs) > 0 {
		input["tagIds"] = item.TagIDs
	}
	if len(item.Tags) > 0 {
		input["tagNames"] = item.Tags
	}
	return input
}
