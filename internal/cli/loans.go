package cli

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

func newLoansCommand(state *appState) *cobra.Command {
	cmd := &cobra.Command{Use: "loans", Short: "Manage loans and installments"}
	cmd.AddCommand(newLoanCreateCommand(state), newLoansListCommand(state), newLoanPaymentCommand(state))
	return cmd
}

func newLoanCreateCommand(state *appState) *cobra.Command {
	var name, lender, currency, startDate, dueDate string
	var outstanding, principal, rate, monthly float64
	var installments int
	var createAccount bool
	cmd := &cobra.Command{Use: "create", Short: "Create a loan", RunE: func(cmd *cobra.Command, args []string) error {
		if strings.TrimSpace(name) == "" {
			return fmt.Errorf("--name is required")
		}
		if outstanding <= 0 && principal <= 0 {
			return fmt.Errorf("--outstanding-balance or --principal must be greater than zero")
		}
		if currency == "" {
			currency = "PEN"
		}
		if startDate == "" {
			startDate = time.Now().Format(time.RFC3339)
		}
		if installments <= 0 {
			installments = 1
		}
		input := map[string]any{"name": name, "lender": lender, "currency": strings.ToUpper(currency), "interestRate": rate, "startDate": startDate, "termMonths": installments, "createAccount": createAccount}
		if principal > 0 {
			input["principal"] = principal
		} else {
			input["principal"] = outstanding
		}
		if outstanding > 0 {
			input["outstandingBalance"] = outstanding
		}
		if monthly > 0 {
			input["monthlyPayment"] = monthly
		}
		if dueDate != "" {
			input["dueDate"] = dueDate
		}
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
	cmd.Flags().StringVar(&name, "name", "", "loan name")
	cmd.Flags().StringVar(&lender, "lender", "", "lender name")
	cmd.Flags().StringVar(&currency, "currency", "PEN", "ISO currency code")
	cmd.Flags().Float64Var(&outstanding, "outstanding-balance", 0, "current outstanding balance")
	cmd.Flags().Float64Var(&principal, "principal", 0, "initial principal")
	cmd.Flags().Float64Var(&rate, "interest-rate", 0, "annual effective rate as decimal")
	cmd.Flags().IntVar(&installments, "installments", 1, "number of monthly installments")
	cmd.Flags().Float64Var(&monthly, "monthly-payment", 0, "monthly installment amount")
	cmd.Flags().StringVar(&startDate, "start-date", "", "ISO start date")
	cmd.Flags().StringVar(&dueDate, "due-date", "", "ISO final due date")
	cmd.Flags().BoolVar(&createAccount, "create-account", true, "create the matching LOAN liability account")
	return cmd
}

func newLoansListCommand(state *appState) *cobra.Command {
	return &cobra.Command{Use: "list", Short: "List active loans", RunE: func(cmd *cobra.Command, args []string) error {
		client, _, _, err := requireSessionClient(state)
		if err != nil {
			return err
		}
		var data struct {
			Loans []loan `json:"loans"`
		}
		if err = client.Do(context.Background(), loansQuery, nil, &data); err != nil {
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
}

func newLoanPaymentCommand(state *appState) *cobra.Command {
	var loanID, fromID, date, key string
	var amount float64
	cmd := &cobra.Command{Use: "payment", Short: "Record a loan payment", RunE: func(cmd *cobra.Command, args []string) error {
		if loanID == "" {
			return fmt.Errorf("--loan-id is required")
		}
		if fromID == "" {
			return fmt.Errorf("--from-account-id is required")
		}
		if amount <= 0 {
			return fmt.Errorf("--amount must be greater than zero")
		}
		if date == "" {
			date = time.Now().Format("2006-01-02")
		}
		vars := map[string]any{"loanId": loanID, "fromAccountId": fromID, "amount": amount, "date": date}
		if key != "" {
			vars["idempotencyKey"] = key
		}
		client, _, _, err := requireSessionClient(state)
		if err != nil {
			return err
		}
		var data struct {
			RecordLoanPayment loanPayment `json:"recordLoanPayment"`
		}
		if err = client.Do(context.Background(), loanPaymentMutation, vars, &data); err != nil {
			return err
		}
		if state.jsonOutput {
			return writeJSON(data.RecordLoanPayment)
		}
		return writeHuman("Recorded loan payment %s (principal %.2f, interest %.2f)\n", data.RecordLoanPayment.ID, float64(data.RecordLoanPayment.PrincipalPortion), float64(data.RecordLoanPayment.InterestPortion))
	}}
	cmd.Flags().StringVar(&loanID, "loan-id", "", "loan ID")
	cmd.Flags().StringVar(&fromID, "from-account-id", "", "source account ID")
	cmd.Flags().Float64Var(&amount, "amount", 0, "payment amount")
	cmd.Flags().StringVar(&date, "date", "", "ISO date")
	cmd.Flags().StringVar(&key, "idempotency-key", "", "safe retry key")
	return cmd
}

const loanFields = `id name lender principal remainingBalance interestRate currency startDate termMonths monthlyPayment dueDate liabilityAccountId isActive`
const loansQuery = `query Loans{loans{` + loanFields + `}}`
const createLoanMutation = `mutation CreateLoan($input:CreateLoanInput!){createLoan(input:$input){` + loanFields + `}}`
const loanPaymentMutation = `mutation LoanPayment($loanId:ID!,$fromAccountId:ID,$amount:Float!,$date:String,$idempotencyKey:String){recordLoanPayment(loanId:$loanId,fromAccountId:$fromAccountId,amount:$amount,date:$date,idempotencyKey:$idempotencyKey){id amount principalPortion interestPortion date fromAccountId transactionId}}`
