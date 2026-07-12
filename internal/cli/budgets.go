package cli

import (
	"context"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

func newBudgetsCommand(state *appState) *cobra.Command {
	cmd := &cobra.Command{Use: "budgets", Short: "Manage monthly category budgets"}
	cmd.AddCommand(newBudgetsListCommand(state), newBudgetCreateCommand(state), newBudgetUpdateCommand(state), newBudgetDeleteCommand(state))
	return cmd
}

func newBudgetsListCommand(state *appState) *cobra.Command {
	return &cobra.Command{Use: "list", Short: "List monthly budgets", RunE: func(cmd *cobra.Command, args []string) error {
		client, _, _, err := requireSessionClient(state)
		if err != nil {
			return err
		}
		var data struct {
			BudgetRules []budget `json:"budgetRules"`
		}
		if err = client.Do(context.Background(), budgetsQuery, nil, &data); err != nil {
			return err
		}
		if state.jsonOutput {
			return writeJSON(data.BudgetRules)
		}
		for _, item := range data.BudgetRules {
			if err := writeHuman("%s\t%s\t%.2f %s\n", item.ID, item.Category.Name, float64(item.MonthlyLimit), item.Currency); err != nil {
				return err
			}
		}
		return nil
	}}
}

func newBudgetCreateCommand(state *appState) *cobra.Command {
	var categoryID, currency string
	var limit float64
	cmd := &cobra.Command{Use: "create", Short: "Create or replace a category budget", RunE: func(cmd *cobra.Command, args []string) error {
		if categoryID == "" {
			return fmt.Errorf("--category-id is required")
		}
		if limit <= 0 {
			return fmt.Errorf("--monthly-limit must be greater than zero")
		}
		input := map[string]any{"categoryId": categoryID, "monthlyLimit": limit}
		if currency != "" {
			input["currency"] = strings.ToUpper(currency)
		}
		client, _, _, err := requireSessionClient(state)
		if err != nil {
			return err
		}
		var data struct {
			UpsertBudgetRule budget `json:"upsertBudgetRule"`
		}
		if err = client.Do(context.Background(), upsertBudgetMutation, map[string]any{"input": input}, &data); err != nil {
			return err
		}
		if state.jsonOutput {
			return writeJSON(data.UpsertBudgetRule)
		}
		return writeHuman("Saved budget %s\n", data.UpsertBudgetRule.ID)
	}}
	cmd.Flags().StringVar(&categoryID, "category-id", "", "category ID")
	cmd.Flags().Float64Var(&limit, "monthly-limit", 0, "monthly spending limit")
	cmd.Flags().StringVar(&currency, "currency", "PEN", "ISO currency code")
	return cmd
}

func newBudgetUpdateCommand(state *appState) *cobra.Command {
	var categoryID, currency string
	var limit float64
	cmd := &cobra.Command{Use: "update <id>", Short: "Update a monthly budget", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		input := map[string]any{}
		if cmd.Flags().Changed("category-id") {
			input["categoryId"] = categoryID
		}
		if cmd.Flags().Changed("monthly-limit") {
			if limit <= 0 {
				return fmt.Errorf("--monthly-limit must be greater than zero")
			}
			input["monthlyLimit"] = limit
		}
		if cmd.Flags().Changed("currency") {
			input["currency"] = strings.ToUpper(currency)
		}
		if len(input) == 0 {
			return fmt.Errorf("at least one field must be provided")
		}
		client, _, _, err := requireSessionClient(state)
		if err != nil {
			return err
		}
		var data struct {
			UpdateBudget budget `json:"updateBudget"`
		}
		if err = client.Do(context.Background(), updateBudgetMutation, map[string]any{"id": args[0], "input": input}, &data); err != nil {
			return err
		}
		if state.jsonOutput {
			return writeJSON(data.UpdateBudget)
		}
		return writeHuman("Updated budget %s\n", data.UpdateBudget.ID)
	}}
	cmd.Flags().StringVar(&categoryID, "category-id", "", "category ID")
	cmd.Flags().Float64Var(&limit, "monthly-limit", 0, "monthly spending limit")
	cmd.Flags().StringVar(&currency, "currency", "", "ISO currency code")
	return cmd
}

func newBudgetDeleteCommand(state *appState) *cobra.Command {
	return &cobra.Command{Use: "delete <id>", Short: "Delete a monthly budget", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		client, _, _, err := requireSessionClient(state)
		if err != nil {
			return err
		}
		var data struct {
			DeleteBudgetRule bool `json:"deleteBudgetRule"`
		}
		if err = client.Do(context.Background(), deleteBudgetMutation, map[string]any{"id": args[0]}, &data); err != nil {
			return err
		}
		if state.jsonOutput {
			return writeJSON(map[string]any{"deleted": data.DeleteBudgetRule, "id": args[0]})
		}
		return writeHuman("Deleted budget %s\n", args[0])
	}}
}

const budgetFields = `id monthlyLimit currency isActive category{` + categoryFields + `}`
const budgetsQuery = `query Budgets{budgetRules{` + budgetFields + `}}`
const upsertBudgetMutation = `mutation UpsertBudget($input:UpsertBudgetRuleInput!){upsertBudgetRule(input:$input){` + budgetFields + `}}`
const updateBudgetMutation = `mutation UpdateBudget($id:ID!,$input:UpdateBudgetInput!){updateBudget(id:$id,input:$input){` + budgetFields + `}}`
const deleteBudgetMutation = `mutation DeleteBudget($id:ID!){deleteBudgetRule(id:$id)}`
