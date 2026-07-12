package cli

import (
	"context"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

func newSubscriptionsCommand(state *appState) *cobra.Command {
	cmd := &cobra.Command{Use: "subscriptions", Short: "Manage recurring services and subscriptions"}
	cmd.AddCommand(newSubscriptionCreateCommand(state), newSubscriptionsListCommand(state, false), newSubscriptionsUpcomingCommand(state))
	return cmd
}

func newSubscriptionCreateCommand(state *appState) *cobra.Command {
	var name, currency, frequency, categoryID, accountID, nextDate string
	var amount float64
	var dueDay int
	cmd := &cobra.Command{Use: "create", Short: "Create a recurring subscription", RunE: func(cmd *cobra.Command, args []string) error {
		if strings.TrimSpace(name) == "" {
			return fmt.Errorf("--name is required")
		}
		if amount <= 0 {
			return fmt.Errorf("--amount must be greater than zero")
		}
		frequency = strings.ToUpper(frequency)
		if frequency != "MONTHLY" && frequency != "YEARLY" {
			return fmt.Errorf("--frequency must be MONTHLY or YEARLY")
		}
		if dueDay < 0 || dueDay > 31 {
			return fmt.Errorf("--due-day must be between 1 and 31")
		}
		input := map[string]any{"name": name, "amount": amount, "currency": strings.ToUpper(currency), "frequency": frequency}
		if dueDay > 0 {
			input["dueDay"] = dueDay
		}
		if categoryID != "" {
			input["categoryId"] = categoryID
		}
		if accountID != "" {
			input["accountId"] = accountID
		}
		if nextDate != "" {
			input["nextRenewalDate"] = nextDate
		}
		client, _, _, err := requireSessionClient(state)
		if err != nil {
			return err
		}
		var data struct {
			CreateSubscription subscription `json:"createSubscription"`
		}
		if err = client.Do(context.Background(), createSubscriptionMutation, map[string]any{"input": input}, &data); err != nil {
			return err
		}
		if state.jsonOutput {
			return writeJSON(data.CreateSubscription)
		}
		return writeHuman("Created subscription %s\n", data.CreateSubscription.ID)
	}}
	cmd.Flags().StringVar(&name, "name", "", "subscription name")
	cmd.Flags().Float64Var(&amount, "amount", 0, "recurring amount")
	cmd.Flags().StringVar(&currency, "currency", "PEN", "ISO currency code")
	cmd.Flags().StringVar(&frequency, "frequency", "MONTHLY", "MONTHLY or YEARLY")
	cmd.Flags().IntVar(&dueDay, "due-day", 0, "day of month")
	cmd.Flags().StringVar(&categoryID, "category-id", "", "category ID")
	cmd.Flags().StringVar(&accountID, "account-id", "", "account charged when renewed")
	cmd.Flags().StringVar(&nextDate, "next-date", "", "next ISO renewal date")
	return cmd
}

func newSubscriptionsListCommand(state *appState, upcoming bool) *cobra.Command {
	return &cobra.Command{Use: "list", Short: "List subscriptions", RunE: func(cmd *cobra.Command, args []string) error {
		return runSubscriptionsList(state, subscriptionsQuery, nil)
	}}
}
func newSubscriptionsUpcomingCommand(state *appState) *cobra.Command {
	var days int
	cmd := &cobra.Command{Use: "upcoming", Short: "List subscriptions due soon", RunE: func(cmd *cobra.Command, args []string) error {
		return runSubscriptionsList(state, upcomingSubscriptionsQuery, map[string]any{"days": days})
	}}
	cmd.Flags().IntVar(&days, "days", 30, "lookahead in days")
	return cmd
}
func runSubscriptionsList(state *appState, query string, vars map[string]any) error {
	client, _, _, err := requireSessionClient(state)
	if err != nil {
		return err
	}
	var data struct {
		Subscriptions         []subscription `json:"subscriptions"`
		UpcomingSubscriptions []subscription `json:"upcomingSubscriptions"`
	}
	if err = client.Do(context.Background(), query, vars, &data); err != nil {
		return err
	}
	items := data.Subscriptions
	if items == nil {
		items = data.UpcomingSubscriptions
	}
	if state.jsonOutput {
		return writeJSON(items)
	}
	for _, item := range items {
		if err := writeHuman("%s\t%s\t%.2f %s\t%s\n", item.ID, item.Name, float64(item.Amount), item.Currency, item.NextRenewalDate); err != nil {
			return err
		}
	}
	return nil
}

const subscriptionFields = `id name amount currency billingCycle nextRenewalDate dueDay categoryId isActive`
const subscriptionsQuery = `query Subscriptions{subscriptions{` + subscriptionFields + `}}`
const upcomingSubscriptionsQuery = `query UpcomingSubscriptions($days:Int!){upcomingSubscriptions(days:$days){` + subscriptionFields + `}}`
const createSubscriptionMutation = `mutation CreateSubscription($input:CreateSubscriptionInput!){createSubscription(input:$input){` + subscriptionFields + `}}`
