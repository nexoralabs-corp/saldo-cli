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

func newLoansCommand(state *appState) *cobra.Command {
	cmd := &cobra.Command{Use: "loans", Short: "Manage loans, schedules, and payments"}
	cmd.AddCommand(
		newLoanCreateCommand(state), newLoansListCommand(state), newLoanGetCommand(state),
		newLoanUpdateCommand(state), newLoanArchiveCommand(state), newLoanReactivateCommand(state), newLoanDeleteCommand(state),
		newLoanPaymentCommand(state), newLoanCorrectPaymentCommand(state), newLoanScheduleCommand(state), newLoanAllocationProposalCommand(state),
		newLoanCardInstallmentCommand(state),
	)
	return cmd
}

type loanFlags struct {
	Name, Lender, Currency, StartDate, DueDate, DefaultPaymentAccountID, CreditCardID, CreditCardAccountID, CollectionMode, ExternalReference string
	Outstanding, Principal, Rate, Monthly                                                                                                     float64
	Installments                                                                                                                              int
	CreateAccount                                                                                                                             bool
}

func bindLoanFlags(cmd *cobra.Command, f *loanFlags, creating bool) {
	cmd.Flags().StringVar(&f.Name, "name", "", "loan name")
	cmd.Flags().StringVar(&f.Lender, "lender", "", "lender name")
	cmd.Flags().StringVar(&f.Currency, "currency", "", "ISO currency code")
	cmd.Flags().Float64Var(&f.Outstanding, "outstanding-balance", 0, "current outstanding balance")
	cmd.Flags().Float64Var(&f.Principal, "principal", 0, "initial principal")
	cmd.Flags().Float64Var(&f.Rate, "interest-rate", 0, "annual effective rate as decimal")
	cmd.Flags().IntVar(&f.Installments, "installments", 0, "number of monthly installments")
	cmd.Flags().Float64Var(&f.Monthly, "monthly-payment", 0, "monthly installment amount")
	cmd.Flags().StringVar(&f.StartDate, "start-date", "", "ISO start date")
	cmd.Flags().StringVar(&f.DueDate, "due-date", "", "ISO first installment due date")
	cmd.Flags().StringVar(&f.DefaultPaymentAccountID, "default-payment-account-id", "", "default active source account ID")
	cmd.Flags().StringVar(&f.CreditCardID, "credit-card-id", "", "card contract that collects installments")
	cmd.Flags().StringVar(&f.CreditCardAccountID, "credit-card-account-id", "", "card currency ledger that receives installments")
	cmd.Flags().StringVar(&f.CollectionMode, "collection-mode", "DIRECT", "DIRECT or CREDIT_CARD_STATEMENT")
	cmd.Flags().StringVar(&f.ExternalReference, "external-reference", "", "bank reference, such as ExtraCash operation")
	if creating {
		cmd.Flags().BoolVar(&f.CreateAccount, "create-account", true, "create the matching LOAN liability account")
	}
}

func newLoanCreateCommand(state *appState) *cobra.Command {
	f := loanFlags{}
	cmd := &cobra.Command{Use: "create", Short: "Create a loan and its persisted schedule", RunE: func(cmd *cobra.Command, args []string) error {
		if strings.TrimSpace(f.Name) == "" {
			return fmt.Errorf("--name is required")
		}
		if f.Outstanding <= 0 && f.Principal <= 0 {
			return fmt.Errorf("--outstanding-balance or --principal must be greater than zero")
		}
		if f.Currency == "" {
			f.Currency = "PEN"
		}
		if f.StartDate == "" {
			f.StartDate = time.Now().Format(time.RFC3339)
		}
		if f.Installments <= 0 {
			f.Installments = 1
		}
		input := loanInput(f, cmd, false)
		client, _, _, err := requireSessionClient(state)
		if err != nil {
			return err
		}
		var data struct {
			CreateLoan loan `json:"createLoan"`
		}
		if err = client.Do(context.Background(), createLoanMutation, map[string]any{"input": input}, &data); err != nil {
			return err
		}
		if state.jsonOutput {
			return writeJSON(data.CreateLoan)
		}
		return writeHuman("Created loan %s\n", data.CreateLoan.ID)
	}}
	bindLoanFlags(cmd, &f, true)
	return cmd
}

func loanInput(f loanFlags, cmd *cobra.Command, update bool) map[string]any {
	input := map[string]any{}
	put := func(flag, key string, value any) {
		if !update || cmd.Flags().Changed(flag) {
			input[key] = value
		}
	}
	if !update && f.Currency == "" {
		f.Currency = "PEN"
	}
	put("name", "name", strings.TrimSpace(f.Name))
	put("lender", "lender", strings.TrimSpace(f.Lender))
	put("currency", "currency", strings.ToUpper(strings.TrimSpace(f.Currency)))
	put("principal", "principal", f.Principal)
	put("interest-rate", "interestRate", f.Rate)
	put("start-date", "startDate", f.StartDate)
	put("installments", "termMonths", f.Installments)
	if !update {
		input["createAccount"] = f.CreateAccount
		if f.Principal <= 0 {
			input["principal"] = f.Outstanding
		}
		if f.Outstanding > 0 {
			input["outstandingBalance"] = f.Outstanding
		}
		if f.Monthly > 0 {
			input["monthlyPayment"] = f.Monthly
		}
		if f.DueDate != "" {
			input["dueDate"] = f.DueDate
		}
	} else {
		put("due-date", "dueDate", f.DueDate)
	}
	if update {
		put("default-payment-account-id", "defaultPaymentAccountId", f.DefaultPaymentAccountID)
		put("credit-card-id", "creditCardId", f.CreditCardID)
		put("credit-card-account-id", "creditCardAccountId", f.CreditCardAccountID)
		put("collection-mode", "collectionMode", strings.ToUpper(f.CollectionMode))
		put("external-reference", "externalReference", f.ExternalReference)
	} else if f.DefaultPaymentAccountID != "" {
		input["defaultPaymentAccountId"] = f.DefaultPaymentAccountID
	}
	if !update {
		input["collectionMode"] = strings.ToUpper(f.CollectionMode)
		input["externalReference"] = f.ExternalReference
		if f.CreditCardID != "" {
			input["creditCardId"] = f.CreditCardID
		}
		if f.CreditCardAccountID != "" {
			input["creditCardAccountId"] = f.CreditCardAccountID
		}
	}
	return input
}

func newLoanCardInstallmentCommand(state *appState) *cobra.Command {
	cmd := &cobra.Command{Use: "card-installment", Short: "Post or reverse card-collected loan installments"}
	cmd.AddCommand(
		newLoanCardInstallmentPostCommand(state), newLoanCardInstallmentReverseCommand(state),
	)
	return cmd
}

func newLoanCardInstallmentPostCommand(state *appState) *cobra.Command {
	var key string
	cmd := &cobra.Command{Use: "post <installment-id>", Short: "Move one loan installment to its card ledger", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		if key == "" {
			return fmt.Errorf("--idempotency-key is required")
		}
		client, _, _, err := requireSessionClient(state)
		if err != nil {
			return err
		}
		var data struct {
			PostCreditCardInstallment map[string]any `json:"postCreditCardInstallment"`
		}
		if err = client.Do(context.Background(), postCreditCardInstallmentMutation, map[string]any{"installmentId": args[0], "idempotencyKey": key}, &data); err != nil {
			return err
		}
		if state.jsonOutput {
			return writeJSON(data.PostCreditCardInstallment)
		}
		return writeHuman("Posted installment %s to its card ledger\n", args[0])
	}}
	cmd.Flags().StringVar(&key, "idempotency-key", "", "required safe retry key")
	return cmd
}

func newLoanCardInstallmentReverseCommand(state *appState) *cobra.Command {
	var key string
	cmd := &cobra.Command{Use: "reverse <posting-id>", Short: "Reverse a card installment posting", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		if key == "" {
			return fmt.Errorf("--idempotency-key is required")
		}
		client, _, _, err := requireSessionClient(state)
		if err != nil {
			return err
		}
		var data struct {
			ReverseCreditCardInstallmentPosting map[string]any `json:"reverseCreditCardInstallmentPosting"`
		}
		if err = client.Do(context.Background(), reverseCreditCardInstallmentPostingMutation, map[string]any{"postingId": args[0], "idempotencyKey": key}, &data); err != nil {
			return err
		}
		if state.jsonOutput {
			return writeJSON(data.ReverseCreditCardInstallmentPosting)
		}
		return writeHuman("Reversed card installment posting %s\n", args[0])
	}}
	cmd.Flags().StringVar(&key, "idempotency-key", "", "required safe retry key")
	return cmd
}

func newLoansListCommand(state *appState) *cobra.Command {
	var status string
	cmd := &cobra.Command{Use: "list", Short: "List loans by archival status", RunE: func(cmd *cobra.Command, args []string) error {
		status, err := loanStatus(status)
		if err != nil {
			return err
		}
		client, _, _, err := requireSessionClient(state)
		if err != nil {
			return err
		}
		var data struct {
			Loans []loan `json:"loans"`
		}
		if err = client.Do(context.Background(), loansQuery, map[string]any{"status": status}, &data); err != nil {
			return err
		}
		if state.jsonOutput {
			return writeJSON(data.Loans)
		}
		for _, item := range data.Loans {
			if err := writeHuman("%s\t%s\t%.2f %s\t%s\n", item.ID, item.Name, float64(item.RemainingBalance), item.Currency, item.Lender); err != nil {
				return err
			}
		}
		return nil
	}}
	cmd.Flags().StringVar(&status, "status", "active", "active, archived, or all")
	return cmd
}

func loanStatus(value string) (string, error) {
	switch strings.ToUpper(strings.TrimSpace(value)) {
	case "", "ACTIVE":
		return "ACTIVE", nil
	case "ARCHIVED":
		return "ARCHIVED", nil
	case "ALL":
		return "ALL", nil
	default:
		return "", fmt.Errorf("--status must be active, archived, or all")
	}
}

func newLoanGetCommand(state *appState) *cobra.Command {
	return loanGetLikeCommand(state, "get <id>", "Get a loan with its schedules and payments", loanQuery)
}

func loanGetLikeCommand(state *appState, use, short, query string) *cobra.Command {
	return &cobra.Command{Use: use, Short: short, Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		client, _, _, err := requireSessionClient(state)
		if err != nil {
			return err
		}
		var data struct {
			Loan loan `json:"loan"`
		}
		if err = client.Do(context.Background(), query, map[string]any{"id": args[0]}, &data); err != nil {
			return err
		}
		if state.jsonOutput {
			return writeJSON(data.Loan)
		}
		return writeHuman("%s\t%s\t%.2f %s\n", data.Loan.ID, data.Loan.Name, float64(data.Loan.RemainingBalance), data.Loan.Currency)
	}}
}

func newLoanUpdateCommand(state *appState) *cobra.Command {
	f := loanFlags{}
	var clearDefault bool
	cmd := &cobra.Command{Use: "update <id>", Short: "Update a loan or its default payment account", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		input := loanInput(f, cmd, true)
		if clearDefault {
			input["defaultPaymentAccountId"] = nil
		}
		if len(input) == 0 {
			return fmt.Errorf("at least one field must be provided")
		}
		client, _, _, err := requireSessionClient(state)
		if err != nil {
			return err
		}
		var data struct {
			UpdateLoan loan `json:"updateLoan"`
		}
		if err = client.Do(context.Background(), updateLoanMutation, map[string]any{"id": args[0], "input": input}, &data); err != nil {
			return err
		}
		if state.jsonOutput {
			return writeJSON(data.UpdateLoan)
		}
		return writeHuman("Updated loan %s\n", data.UpdateLoan.ID)
	}}
	bindLoanFlags(cmd, &f, false)
	cmd.Flags().BoolVar(&clearDefault, "clear-default-payment-account", false, "remove the configured default payment account")
	return cmd
}

func newLoanArchiveCommand(state *appState) *cobra.Command {
	return newLoanLifecycleCommand(state, "archive <id>", "Archive a loan", "archiveLoan")
}
func newLoanReactivateCommand(state *appState) *cobra.Command {
	return newLoanLifecycleCommand(state, "reactivate <id>", "Reactivate a loan without changing paid status", "reactivateLoan")
}
func newLoanDeleteCommand(state *appState) *cobra.Command {
	return newLoanLifecycleCommand(state, "delete <id>", "Permanently delete a loan without financial history", "deleteLoan")
}

func newLoanLifecycleCommand(state *appState, use, short, field string) *cobra.Command {
	return &cobra.Command{Use: use, Short: short, Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		client, _, _, err := requireSessionClient(state)
		if err != nil {
			return err
		}
		if field == "deleteLoan" {
			var data struct {
				DeleteLoan bool `json:"deleteLoan"`
			}
			if err = client.Do(context.Background(), deleteLoanMutation, map[string]any{"id": args[0]}, &data); err != nil {
				return err
			}
			if state.jsonOutput {
				return writeJSON(map[string]any{"id": args[0], "deleted": data.DeleteLoan})
			}
			return writeHuman("Deleted loan %s\n", args[0])
		}
		var data struct {
			Loan loan `json:"loan"`
		}
		mutation := archiveLoanMutation
		if field == "reactivateLoan" {
			mutation = reactivateLoanMutation
		}
		if err = client.Do(context.Background(), mutation, map[string]any{"id": args[0]}, &data); err != nil {
			return err
		}
		if state.jsonOutput {
			return writeJSON(data.Loan)
		}
		return writeHuman("%s loan %s\n", strings.Title(strings.TrimSuffix(field, "Loan")), data.Loan.ID)
	}}
}

func newLoanPaymentCommand(state *appState) *cobra.Command {
	var loanID, fromID, date, key, allocationsFile string
	var amount, sourceAmount, appliedAmount, exchangeRate float64
	cmd := &cobra.Command{Use: "payment", Short: "Record an idempotent loan payment", RunE: func(cmd *cobra.Command, args []string) error {
		if loanID == "" || fromID == "" {
			return fmt.Errorf("--loan-id and --from-account-id are required")
		}
		if key == "" {
			return fmt.Errorf("--idempotency-key is required")
		}
		if amount <= 0 {
			return fmt.Errorf("--amount must be greater than zero")
		}
		if err := validatePaymentAmounts(sourceAmount, appliedAmount, exchangeRate); err != nil {
			return err
		}
		if date == "" {
			date = time.Now().Format("2006-01-02")
		}
		vars := map[string]any{"loanId": loanID, "fromAccountId": fromID, "amount": amount, "date": date, "idempotencyKey": key}
		if sourceAmount > 0 {
			vars["sourceAmount"] = sourceAmount
			vars["appliedAmount"] = appliedAmount
			if exchangeRate > 0 {
				vars["exchangeRate"] = exchangeRate
			}
		}
		if allocationsFile != "" {
			rows, err := decodeLoanAllocationsFile(allocationsFile)
			if err != nil {
				return err
			}
			vars["allocations"] = rows
		}
		return executeLoanPayment(state, loanPaymentMutation, vars, "Recorded")
	}}
	cmd.Flags().StringVar(&loanID, "loan-id", "", "loan ID")
	cmd.Flags().StringVar(&fromID, "from-account-id", "", "source account ID")
	cmd.Flags().Float64Var(&amount, "amount", 0, "debt amount applied (legacy alias when values match)")
	cmd.Flags().Float64Var(&sourceAmount, "source-amount", 0, "amount actually debited from the source account")
	cmd.Flags().Float64Var(&appliedAmount, "applied-amount", 0, "amount applied to the loan currency")
	cmd.Flags().Float64Var(&exchangeRate, "exchange-rate", 0, "bank FX rate for cross-currency payments")
	cmd.Flags().StringVar(&allocationsFile, "allocations-file", "", "JSON array or {allocations:[...]} overriding proposed allocation")
	cmd.Flags().StringVar(&date, "date", "", "ISO date")
	cmd.Flags().StringVar(&key, "idempotency-key", "", "required safe retry key")
	return cmd
}

func newLoanCorrectPaymentCommand(state *appState) *cobra.Command {
	var paymentID, fromID, allocationsFile string
	var sourceAmount, appliedAmount, exchangeRate float64
	cmd := &cobra.Command{Use: "correct-payment", Short: "Correct a payment's source amounts or allocations", RunE: func(cmd *cobra.Command, args []string) error {
		if paymentID == "" {
			return fmt.Errorf("--payment-id is required")
		}
		if err := validatePaymentAmounts(sourceAmount, appliedAmount, exchangeRate); err != nil {
			return err
		}
		vars := map[string]any{"paymentId": paymentID}
		if fromID != "" {
			vars["fromAccountId"] = fromID
		}
		if sourceAmount > 0 {
			vars["sourceAmount"] = sourceAmount
			vars["appliedAmount"] = appliedAmount
			if exchangeRate > 0 {
				vars["exchangeRate"] = exchangeRate
			}
		}
		if allocationsFile != "" {
			rows, err := decodeLoanAllocationsFile(allocationsFile)
			if err != nil {
				return err
			}
			vars["allocations"] = rows
		}
		if len(vars) == 1 {
			return fmt.Errorf("provide payment fields or --allocations-file to correct")
		}
		return executeLoanPayment(state, correctLoanPaymentMutation, vars, "Corrected")
	}}
	cmd.Flags().StringVar(&paymentID, "payment-id", "", "payment ID")
	cmd.Flags().StringVar(&fromID, "from-account-id", "", "replacement source account ID")
	cmd.Flags().Float64Var(&sourceAmount, "source-amount", 0, "replacement debited amount")
	cmd.Flags().Float64Var(&appliedAmount, "applied-amount", 0, "replacement applied amount")
	cmd.Flags().Float64Var(&exchangeRate, "exchange-rate", 0, "replacement bank FX rate")
	cmd.Flags().StringVar(&allocationsFile, "allocations-file", "", "JSON array or {allocations:[...]} replacing allocations")
	return cmd
}

func executeLoanPayment(state *appState, mutation string, vars map[string]any, verb string) error {
	client, _, _, err := requireSessionClient(state)
	if err != nil {
		return err
	}
	var data struct {
		RecordLoanPayment  loanPayment `json:"recordLoanPayment"`
		CorrectLoanPayment loanPayment `json:"correctLoanPayment"`
	}
	if err = client.Do(context.Background(), mutation, vars, &data); err != nil {
		return err
	}
	payment := data.RecordLoanPayment
	if payment.ID == "" {
		payment = data.CorrectLoanPayment
	}
	if state.jsonOutput {
		return writeJSON(payment)
	}
	return writeHuman("%s loan payment %s (principal %.2f, interest %.2f)\n", verb, payment.ID, float64(payment.PrincipalPortion), float64(payment.InterestPortion))
}

func validatePaymentAmounts(sourceAmount, appliedAmount, exchangeRate float64) error {
	if sourceAmount == 0 && appliedAmount == 0 && exchangeRate == 0 {
		return nil
	}
	if sourceAmount <= 0 || appliedAmount <= 0 {
		return fmt.Errorf("--source-amount and --applied-amount must both be greater than zero when either is provided")
	}
	if exchangeRate < 0 {
		return fmt.Errorf("--exchange-rate cannot be negative")
	}
	if exchangeRate == 0 && sourceAmount != appliedAmount {
		return fmt.Errorf("--exchange-rate is required when debited and applied amounts differ")
	}
	return nil
}

func newLoanScheduleCommand(state *appState) *cobra.Command {
	cmd := &cobra.Command{Use: "schedule", Short: "View or replace the persisted installment schedule"}
	cmd.AddCommand(
		(&CobraScheduleGet{state: state}).Command(),
		newLoanScheduleUpdateCommand(state),
	)
	return cmd
}

// CobraScheduleGet keeps the schedule command small while using the same output contract as other get commands.
type CobraScheduleGet struct{ state *appState }

func (c *CobraScheduleGet) Command() *cobra.Command {
	return &cobra.Command{Use: "get <loan-id>", Short: "Get persisted installments", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		client, _, _, err := requireSessionClient(c.state)
		if err != nil {
			return err
		}
		var data struct {
			LoanInstallments []loanInstallment `json:"loanInstallments"`
		}
		if err = client.Do(context.Background(), loanInstallmentsQuery, map[string]any{"loanId": args[0]}, &data); err != nil {
			return err
		}
		if c.state.jsonOutput {
			return writeJSON(data.LoanInstallments)
		}
		for _, row := range data.LoanInstallments {
			if err := writeHuman("%d\t%s\t%.2f\t%.2f\t%s\n", row.Number, row.DueDate, float64(row.Principal), float64(row.Interest), row.Status); err != nil {
				return err
			}
		}
		return nil
	}}
}

func newLoanScheduleUpdateCommand(state *appState) *cobra.Command {
	var file string
	cmd := &cobra.Command{Use: "update <loan-id>", Short: "Replace an editable schedule from JSON", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		rows, err := decodeLoanScheduleFile(file)
		if err != nil {
			return err
		}
		client, _, _, err := requireSessionClient(state)
		if err != nil {
			return err
		}
		var data struct {
			UpdateLoanInstallments []loanInstallment `json:"updateLoanInstallments"`
		}
		if err = client.Do(context.Background(), updateLoanInstallmentsMutation, map[string]any{"loanId": args[0], "installments": rows}, &data); err != nil {
			return err
		}
		if state.jsonOutput {
			return writeJSON(data.UpdateLoanInstallments)
		}
		return writeHuman("Updated %d installments for loan %s\n", len(data.UpdateLoanInstallments), args[0])
	}}
	cmd.Flags().StringVar(&file, "file", "", "JSON array or {installments:[...]} payload")
	_ = cmd.MarkFlagRequired("file")
	return cmd
}

func newLoanAllocationProposalCommand(state *appState) *cobra.Command {
	var loanID string
	var appliedAmount float64
	cmd := &cobra.Command{Use: "propose-allocation", Short: "Propose oldest-first payment allocations", RunE: func(cmd *cobra.Command, args []string) error {
		if loanID == "" || appliedAmount <= 0 {
			return fmt.Errorf("--loan-id and --applied-amount greater than zero are required")
		}
		client, _, _, err := requireSessionClient(state)
		if err != nil {
			return err
		}
		var data struct {
			ProposedLoanPaymentAllocations []loanPaymentAllocation `json:"proposedLoanPaymentAllocations"`
		}
		if err = client.Do(context.Background(), proposedLoanPaymentAllocationsQuery, map[string]any{"loanId": loanID, "appliedAmount": appliedAmount}, &data); err != nil {
			return err
		}
		if state.jsonOutput {
			return writeJSON(data.ProposedLoanPaymentAllocations)
		}
		for _, row := range data.ProposedLoanPaymentAllocations {
			if err := writeHuman("%s\tcapital %.2f\tinterest %.2f\tfee %.2f\tinsurance %.2f\tlate %.2f\n", row.InstallmentID, float64(row.Principal), float64(row.Interest), float64(row.Fee), float64(row.Insurance), float64(row.LateFee)); err != nil {
				return err
			}
		}
		return nil
	}}
	cmd.Flags().StringVar(&loanID, "loan-id", "", "loan ID")
	cmd.Flags().Float64Var(&appliedAmount, "applied-amount", 0, "amount in loan currency to allocate")
	return cmd
}

type loanScheduleInput struct {
	ID        string  `json:"id,omitempty"`
	Number    int     `json:"number"`
	DueDate   string  `json:"dueDate"`
	Principal float64 `json:"principal"`
	Interest  float64 `json:"interest"`
	Fee       float64 `json:"fee"`
	Insurance float64 `json:"insurance"`
	LateFee   float64 `json:"lateFee"`
}
type loanAllocationInput struct {
	InstallmentID string  `json:"installmentId"`
	Principal     float64 `json:"principal"`
	Interest      float64 `json:"interest"`
	Fee           float64 `json:"fee"`
	Insurance     float64 `json:"insurance"`
	LateFee       float64 `json:"lateFee"`
}

func decodeLoanScheduleFile(file string) ([]loanScheduleInput, error) {
	var rows []loanScheduleInput
	if err := decodeLoanJSONFile(file, "installments", &rows); err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, fmt.Errorf("schedule file must contain at least one installment")
	}
	for _, row := range rows {
		if row.Number <= 0 || row.DueDate == "" || row.Principal < 0 || row.Interest < 0 || row.Fee < 0 || row.Insurance < 0 || row.LateFee < 0 {
			return nil, fmt.Errorf("schedule file contains an invalid installment")
		}
	}
	return rows, nil
}

func decodeLoanAllocationsFile(file string) ([]loanAllocationInput, error) {
	var rows []loanAllocationInput
	if err := decodeLoanJSONFile(file, "allocations", &rows); err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, fmt.Errorf("allocations file must contain at least one allocation")
	}
	for _, row := range rows {
		if row.InstallmentID == "" || row.Principal < 0 || row.Interest < 0 || row.Fee < 0 || row.Insurance < 0 || row.LateFee < 0 {
			return nil, fmt.Errorf("allocations file contains an invalid allocation")
		}
	}
	return rows, nil
}

func decodeLoanJSONFile[T any](file, envelopeKey string, target *[]T) error {
	raw, err := os.ReadFile(file)
	if err != nil {
		return fmt.Errorf("read %s: %w", file, err)
	}
	if len(strings.TrimSpace(string(raw))) == 0 {
		return fmt.Errorf("%s is empty", file)
	}
	if err := json.Unmarshal(raw, target); err == nil {
		return nil
	}
	var envelope map[string]json.RawMessage
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return fmt.Errorf("parse %s: %w", file, err)
	}
	payload, ok := envelope[envelopeKey]
	if !ok {
		return fmt.Errorf("%s must be an array or contain %q", file, envelopeKey)
	}
	if err := json.Unmarshal(payload, target); err != nil {
		return fmt.Errorf("parse %s %q: %w", file, envelopeKey, err)
	}
	return nil
}

const loanFields = `id name lender principal remainingBalance interestRate currency startDate termMonths monthlyPayment dueDate liabilityAccountId defaultPaymentAccountId isActive archivedAt creditCardId creditCardAccountId collectionMode externalReference`
const installmentFields = `id number dueDate principal interest fee insurance lateFee paidPrincipal paidInterest paidFee paidInsurance paidLateFee status total paidTotal`
const allocationFields = `installmentId principal interest fee insurance lateFee`
const paymentFields = `id amount sourceAmount appliedAmount exchangeRate principalPortion interestPortion date fromAccountId transactionId allocations{` + allocationFields + `}`
const loansQuery = `query Loans($status:LoanListStatus!){loans(status:$status){` + loanFields + `}}`
const loanQuery = `query Loan($id:ID!){loan(id:$id){` + loanFields + ` installments{` + installmentFields + `} payments{` + paymentFields + `}}}`
const loanInstallmentsQuery = `query LoanInstallments($loanId:ID!){loanInstallments(loanId:$loanId){` + installmentFields + `}}`
const proposedLoanPaymentAllocationsQuery = `query ProposedLoanPaymentAllocations($loanId:ID!,$appliedAmount:Float!){proposedLoanPaymentAllocations(loanId:$loanId,appliedAmount:$appliedAmount){` + allocationFields + `}}`
const createLoanMutation = `mutation CreateLoan($input:CreateLoanInput!){createLoan(input:$input){` + loanFields + `}}`
const updateLoanMutation = `mutation UpdateLoan($id:ID!,$input:UpdateLoanInput!){updateLoan(id:$id,input:$input){` + loanFields + `}}`
const archiveLoanMutation = `mutation ArchiveLoan($id:ID!){archiveLoan(id:$id){` + loanFields + `}}`
const reactivateLoanMutation = `mutation ReactivateLoan($id:ID!){reactivateLoan(id:$id){` + loanFields + `}}`
const deleteLoanMutation = `mutation DeleteLoan($id:ID!){deleteLoan(id:$id)}`
const loanPaymentMutation = `mutation LoanPayment($loanId:ID!,$fromAccountId:ID,$amount:Float!,$date:String,$idempotencyKey:String,$sourceAmount:Float,$appliedAmount:Float,$exchangeRate:Float,$allocations:[LoanPaymentAllocationInput!]){recordLoanPayment(loanId:$loanId,fromAccountId:$fromAccountId,amount:$amount,date:$date,idempotencyKey:$idempotencyKey,sourceAmount:$sourceAmount,appliedAmount:$appliedAmount,exchangeRate:$exchangeRate,allocations:$allocations){` + paymentFields + `}}`
const correctLoanPaymentMutation = `mutation CorrectLoanPayment($paymentId:ID!,$fromAccountId:ID,$sourceAmount:Float,$appliedAmount:Float,$exchangeRate:Float,$allocations:[LoanPaymentAllocationInput!]){correctLoanPayment(paymentId:$paymentId,fromAccountId:$fromAccountId,sourceAmount:$sourceAmount,appliedAmount:$appliedAmount,exchangeRate:$exchangeRate,allocations:$allocations){` + paymentFields + `}}`
const updateLoanInstallmentsMutation = `mutation UpdateLoanInstallments($loanId:ID!,$installments:[LoanInstallmentInput!]!){updateLoanInstallments(loanId:$loanId,installments:$installments){` + installmentFields + `}}`
const postCreditCardInstallmentMutation = `mutation PostCreditCardInstallment($installmentId:ID!,$idempotencyKey:String!){postCreditCardInstallment(installmentId:$installmentId,idempotencyKey:$idempotencyKey){id installmentId creditCardId cardAccountId transactionId principal interest fee insurance status idempotencyKey}}`
const reverseCreditCardInstallmentPostingMutation = `mutation ReverseCreditCardInstallmentPosting($postingId:ID!,$idempotencyKey:String!){reverseCreditCardInstallmentPosting(postingId:$postingId,idempotencyKey:$idempotencyKey){id installmentId creditCardId cardAccountId transactionId principal interest fee insurance status idempotencyKey}}`
