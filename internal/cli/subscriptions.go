package cli

import (
	"context"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

func newSubscriptionsCommand(state *appState) *cobra.Command {
	cmd := &cobra.Command{Use: "subscriptions", Aliases: []string{"services"}, Short: "Manage recurring services and subscriptions"}
	cmd.AddCommand(
		newSubscriptionCreateCommand(state),
		newSubscriptionsListCommand(state, false),
		newSubscriptionsUpcomingCommand(state),
		newSubscriptionGetCommand(state),
		newSubscriptionUpdateCommand(state),
		newSubscriptionArchiveCommand(state),
		newSubscriptionReactivateCommand(state),
		newSubscriptionDeleteCommand(state),
		newSubscriptionChargeCommand(state),
		newSubscriptionCorrectChargeCommand(state),
		newSubscriptionHistoryCommand(state),
	)
	return cmd
}

type subscriptionFlags struct {
	Name, Currency, BillingCycle, NextChargeDate, LegacyNextDate, DueDate string
	CategoryID, AccountID, AmountType, ChargeMode                         string
	Amount, NextChargeAmount                                              float64
	DueDay                                                                int
}

func bindSubscriptionFlags(cmd *cobra.Command, f *subscriptionFlags, update bool) {
	cmd.Flags().StringVar(&f.Name, "name", "", "service name")
	cmd.Flags().Float64Var(&f.Amount, "amount", 0, "base recurring amount")
	cmd.Flags().StringVar(&f.Currency, "currency", "", "ISO currency code")
	cmd.Flags().StringVar(&f.BillingCycle, "billing-cycle", "", "MONTHLY or YEARLY")
	cmd.Flags().StringVar(&f.BillingCycle, "frequency", "", "deprecated alias for --billing-cycle")
	cmd.Flags().StringVar(&f.AmountType, "amount-type", "", "FIXED or VARIABLE")
	cmd.Flags().StringVar(&f.ChargeMode, "charge-mode", "", "AUTOMATIC or MANUAL")
	cmd.Flags().StringVar(&f.NextChargeDate, "next-charge-date", "", "next ISO charge date")
	cmd.Flags().StringVar(&f.LegacyNextDate, "next-date", "", "deprecated alias for --next-charge-date")
	cmd.Flags().StringVar(&f.DueDate, "due-date", "", "ISO due date, separate from the charge date")
	cmd.Flags().Float64Var(&f.NextChargeAmount, "next-charge-amount", 0, "current amount expected for the next charge")
	cmd.Flags().IntVar(&f.DueDay, "due-day", 0, "recurring due day (1-31)")
	cmd.Flags().StringVar(&f.CategoryID, "category-id", "", "expense category ID")
	cmd.Flags().StringVar(&f.AccountID, "account-id", "", "account charged by the service")
	if !update {
		_ = cmd.MarkFlagRequired("name")
	}
}

func subscriptionInput(f subscriptionFlags, cmd *cobra.Command, update bool) (map[string]any, error) {
	input := map[string]any{}
	put := func(flag, key string, value any) {
		if !update || cmd.Flags().Changed(flag) {
			input[key] = value
		}
	}
	if !update || cmd.Flags().Changed("name") {
		if strings.TrimSpace(f.Name) == "" {
			return nil, fmt.Errorf("--name is required")
		}
		input["name"] = strings.TrimSpace(f.Name)
	}
	if !update || cmd.Flags().Changed("amount") {
		if f.Amount <= 0 {
			return nil, fmt.Errorf("--amount must be greater than zero")
		}
		input["amount"] = f.Amount
	}
	if !update && f.Currency == "" {
		f.Currency = "PEN"
	}
	if !update || cmd.Flags().Changed("currency") {
		input["currency"] = strings.ToUpper(strings.TrimSpace(f.Currency))
	}
	if !update && f.BillingCycle == "" {
		f.BillingCycle = "MONTHLY"
	}
	if !update || cmd.Flags().Changed("billing-cycle") || cmd.Flags().Changed("frequency") {
		cycle := strings.ToUpper(strings.TrimSpace(f.BillingCycle))
		if cycle != "MONTHLY" && cycle != "YEARLY" {
			return nil, fmt.Errorf("--billing-cycle must be MONTHLY or YEARLY")
		}
		input["billingCycle"] = cycle
	}
	if !update && f.AmountType == "" {
		f.AmountType = "FIXED"
	}
	if !update || cmd.Flags().Changed("amount-type") {
		amountType := strings.ToUpper(strings.TrimSpace(f.AmountType))
		if amountType != "FIXED" && amountType != "VARIABLE" {
			return nil, fmt.Errorf("--amount-type must be FIXED or VARIABLE")
		}
		input["amountType"] = amountType
	}
	if !update && f.ChargeMode == "" {
		f.ChargeMode = "MANUAL"
	}
	if !update || cmd.Flags().Changed("charge-mode") {
		chargeMode := strings.ToUpper(strings.TrimSpace(f.ChargeMode))
		if chargeMode != "AUTOMATIC" && chargeMode != "MANUAL" {
			return nil, fmt.Errorf("--charge-mode must be AUTOMATIC or MANUAL")
		}
		input["chargeMode"] = chargeMode
	}
	nextChargeDate := f.NextChargeDate
	if nextChargeDate == "" {
		nextChargeDate = f.LegacyNextDate
	}
	if !update || cmd.Flags().Changed("next-charge-date") || cmd.Flags().Changed("next-date") {
		if nextChargeDate != "" {
			input["nextChargeDate"] = nextChargeDate
			// Retain compatibility with backend deployments accepting only the legacy field.
			input["nextRenewalDate"] = nextChargeDate
		}
	}
	put("due-date", "dueDate", emptyToNil(f.DueDate))
	if !update || cmd.Flags().Changed("next-charge-amount") {
		if f.NextChargeAmount < 0 {
			return nil, fmt.Errorf("--next-charge-amount must be greater than zero")
		}
		if f.NextChargeAmount > 0 {
			input["nextChargeAmount"] = f.NextChargeAmount
		}
	}
	if !update || cmd.Flags().Changed("due-day") {
		if f.DueDay < 0 || f.DueDay > 31 {
			return nil, fmt.Errorf("--due-day must be between 1 and 31")
		}
		if f.DueDay > 0 {
			input["dueDay"] = f.DueDay
		}
	}
	put("category-id", "categoryId", emptyToNil(f.CategoryID))
	put("account-id", "accountId", emptyToNil(f.AccountID))
	return input, nil
}

func emptyToNil(value string) any {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return value
}

func newSubscriptionCreateCommand(state *appState) *cobra.Command {
	f := subscriptionFlags{}
	cmd := &cobra.Command{Use: "create", Short: "Create a recurring service", RunE: func(cmd *cobra.Command, args []string) error {
		input, err := subscriptionInput(f, cmd, false)
		if err != nil {
			return err
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
		return writeSubscription(state, data.CreateSubscription, "Created")
	}}
	bindSubscriptionFlags(cmd, &f, false)
	return cmd
}

func newSubscriptionsListCommand(state *appState, upcoming bool) *cobra.Command {
	var status string
	cmd := &cobra.Command{Use: "list", Short: "List services", RunE: func(cmd *cobra.Command, args []string) error {
		status = strings.ToUpper(strings.TrimSpace(status))
		if status == "" {
			status = "ACTIVE"
		}
		if status != "ACTIVE" && status != "ARCHIVED" && status != "ALL" {
			return fmt.Errorf("--status must be active, archived, or all")
		}
		return runSubscriptionsList(state, subscriptionListQuery(status), nil)
	}}
	if !upcoming {
		cmd.Flags().StringVar(&status, "status", "active", "active, archived, or all")
	}
	return cmd
}

func newSubscriptionsUpcomingCommand(state *appState) *cobra.Command {
	var days int
	cmd := &cobra.Command{Use: "upcoming", Short: "List services due soon", RunE: func(cmd *cobra.Command, args []string) error {
		if days < 0 {
			return fmt.Errorf("--days must be zero or greater")
		}
		return runSubscriptionsList(state, upcomingSubscriptionsQuery, map[string]any{"days": days})
	}}
	cmd.Flags().IntVar(&days, "days", 30, "lookahead in days")
	return cmd
}

func newSubscriptionGetCommand(state *appState) *cobra.Command {
	return &cobra.Command{Use: "get <id>", Short: "Get a service", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		client, _, _, err := requireSessionClient(state)
		if err != nil {
			return err
		}
		var data struct {
			Subscription subscription `json:"subscription"`
		}
		if err = client.Do(context.Background(), subscriptionQuery, map[string]any{"id": args[0]}, &data); err != nil {
			return err
		}
		return writeSubscription(state, data.Subscription, "")
	}}
}

func newSubscriptionUpdateCommand(state *appState) *cobra.Command {
	f := subscriptionFlags{}
	cmd := &cobra.Command{Use: "update <id>", Short: "Update a service", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		input, err := subscriptionInput(f, cmd, true)
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
			UpdateSubscription subscription `json:"updateSubscription"`
		}
		if err = client.Do(context.Background(), updateSubscriptionMutation, map[string]any{"id": args[0], "input": input}, &data); err != nil {
			return err
		}
		return writeSubscription(state, data.UpdateSubscription, "Updated")
	}}
	bindSubscriptionFlags(cmd, &f, true)
	return cmd
}

func newSubscriptionArchiveCommand(state *appState) *cobra.Command {
	return subscriptionLifecycleCommand(state, "archive <id>", "Archive a service", archiveSubscriptionMutation, "ArchiveSubscription", "Archived")
}

func newSubscriptionReactivateCommand(state *appState) *cobra.Command {
	return subscriptionLifecycleCommand(state, "reactivate <id>", "Reactivate a service", reactivateSubscriptionMutation, "ReactivateSubscription", "Reactivated")
}

func newSubscriptionDeleteCommand(state *appState) *cobra.Command {
	return &cobra.Command{Use: "delete <id>", Short: "Permanently delete a service with no charges", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		client, _, _, err := requireSessionClient(state)
		if err != nil {
			return err
		}
		var data struct {
			DeleteSubscription bool `json:"deleteSubscription"`
		}
		if err = client.Do(context.Background(), deleteSubscriptionMutation, map[string]any{"id": args[0]}, &data); err != nil {
			return err
		}
		if state.jsonOutput {
			return writeJSON(map[string]any{"deleted": data.DeleteSubscription, "id": args[0]})
		}
		return writeHuman("Deleted service %s\n", args[0])
	}}
}

func subscriptionLifecycleCommand(state *appState, use, short, mutation, field, verb string) *cobra.Command {
	return &cobra.Command{Use: use, Short: short, Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		client, _, _, err := requireSessionClient(state)
		if err != nil {
			return err
		}
		data := map[string]subscription{}
		if err = client.Do(context.Background(), mutation, map[string]any{"id": args[0]}, &data); err != nil {
			return err
		}
		return writeSubscription(state, data[field], verb)
	}}
}

func newSubscriptionChargeCommand(state *appState) *cobra.Command {
	var actualAmount float64
	var key string
	cmd := &cobra.Command{Use: "charge <subscription-id>", Short: "Record a manual service charge", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		if actualAmount <= 0 {
			return fmt.Errorf("--actual-amount must be greater than zero")
		}
		if strings.TrimSpace(key) == "" {
			return fmt.Errorf("--idempotency-key is required for charges")
		}
		client, _, _, err := requireSessionClient(state)
		if err != nil {
			return err
		}
		var data struct {
			ChargeSubscription subscriptionCharge `json:"chargeSubscription"`
		}
		vars := map[string]any{"id": args[0], "actualAmount": actualAmount, "idempotencyKey": strings.TrimSpace(key)}
		if err = client.Do(context.Background(), chargeSubscriptionMutation, vars, &data); err != nil {
			return err
		}
		return writeSubscriptionCharge(state, data.ChargeSubscription, "Recorded charge")
	}}
	cmd.Flags().Float64Var(&actualAmount, "actual-amount", 0, "actual amount charged")
	cmd.Flags().StringVar(&key, "idempotency-key", "", "required stable safe retry key")
	return cmd
}

func newSubscriptionCorrectChargeCommand(state *appState) *cobra.Command {
	var actualAmount float64
	cmd := &cobra.Command{Use: "correct-charge <charge-id>", Short: "Correct a service charge amount by delta", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		if actualAmount <= 0 {
			return fmt.Errorf("--actual-amount must be greater than zero")
		}
		client, _, _, err := requireSessionClient(state)
		if err != nil {
			return err
		}
		var data struct {
			UpdateSubscriptionCharge subscriptionCharge `json:"updateSubscriptionCharge"`
		}
		if err = client.Do(context.Background(), updateSubscriptionChargeMutation, map[string]any{"id": args[0], "actualAmount": actualAmount}, &data); err != nil {
			return err
		}
		return writeSubscriptionCharge(state, data.UpdateSubscriptionCharge, "Corrected charge")
	}}
	cmd.Flags().Float64Var(&actualAmount, "actual-amount", 0, "corrected actual amount")
	return cmd
}

func newSubscriptionHistoryCommand(state *appState) *cobra.Command {
	return &cobra.Command{Use: "history <subscription-id>", Short: "List service charge history", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		client, _, _, err := requireSessionClient(state)
		if err != nil {
			return err
		}
		var data struct {
			SubscriptionCharges []subscriptionCharge `json:"subscriptionCharges"`
		}
		if err = client.Do(context.Background(), subscriptionChargesQuery, map[string]any{"subscriptionId": args[0]}, &data); err != nil {
			return err
		}
		if state.jsonOutput {
			return writeJSON(data.SubscriptionCharges)
		}
		for _, charge := range data.SubscriptionCharges {
			if err := writeHuman("%s\t%.2f\t%.2f\t%s\t%s\n", charge.ID, float64(charge.PlannedAmount), float64(charge.ActualAmount), charge.ChargedAt, charge.Status); err != nil {
				return err
			}
		}
		return nil
	}}
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
		if err := writeHuman("%s\t%s\t%.2f %s\t%s\t%s\n", item.ID, item.Name, float64(item.Amount), item.Currency, item.NextChargeDate, item.ChargeMode); err != nil {
			return err
		}
	}
	return nil
}

func writeSubscription(state *appState, item subscription, verb string) error {
	if state.jsonOutput {
		return writeJSON(item)
	}
	if verb == "" {
		return writeHuman("%s\t%s\t%.2f %s\t%s\t%s\n", item.ID, item.Name, float64(item.Amount), item.Currency, item.NextChargeDate, item.ChargeMode)
	}
	return writeHuman("%s service %s\n", verb, item.ID)
}

func writeSubscriptionCharge(state *appState, charge subscriptionCharge, verb string) error {
	if state.jsonOutput {
		return writeJSON(charge)
	}
	return writeHuman("%s %s (%.2f)\n", verb, charge.ID, float64(charge.ActualAmount))
}

const subscriptionFields = `
id name amount currency billingCycle amountType chargeMode nextRenewalDate nextChargeDate dueDate
nextChargeAmount dueDay categoryId archivedAt isActive`
const subscriptionChargeFields = `id plannedAmount actualAmount scheduledFor chargedAt status idempotencyKey transaction { id }`

func subscriptionListQuery(status string) string {
	return `query Subscriptions{subscriptions(status:` + status + `){` + subscriptionFields + `}}`
}

const upcomingSubscriptionsQuery = `query UpcomingSubscriptions($days:Int!){upcomingSubscriptions(days:$days){` + subscriptionFields + `}}`
const subscriptionQuery = `query Subscription($id:ID!){subscription(id:$id){` + subscriptionFields + `}}`
const subscriptionChargesQuery = `query SubscriptionCharges($subscriptionId:ID!){subscriptionCharges(subscriptionId:$subscriptionId){` + subscriptionChargeFields + `}}`
const createSubscriptionMutation = `mutation CreateSubscription($input:CreateSubscriptionInput!){createSubscription(input:$input){` + subscriptionFields + `}}`
const updateSubscriptionMutation = `mutation UpdateSubscription($id:ID!,$input:UpdateSubscriptionInput!){updateSubscription(id:$id,input:$input){` + subscriptionFields + `}}`
const archiveSubscriptionMutation = `mutation ArchiveSubscription($id:ID!){archiveSubscription(id:$id){` + subscriptionFields + `}}`
const reactivateSubscriptionMutation = `mutation ReactivateSubscription($id:ID!){reactivateSubscription(id:$id){` + subscriptionFields + `}}`
const deleteSubscriptionMutation = `mutation DeleteSubscription($id:ID!){deleteSubscription(id:$id)}`
const chargeSubscriptionMutation = `mutation ChargeSubscription($id:ID!,$actualAmount:Float,$idempotencyKey:String!){chargeSubscription(id:$id,actualAmount:$actualAmount,idempotencyKey:$idempotencyKey){` + subscriptionChargeFields + `}}`
const updateSubscriptionChargeMutation = `mutation UpdateSubscriptionCharge($id:ID!,$actualAmount:Float!){updateSubscriptionCharge(id:$id,actualAmount:$actualAmount){` + subscriptionChargeFields + `}}`
