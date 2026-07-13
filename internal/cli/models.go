package cli

import (
	"encoding/json"
	"fmt"
	"strconv"
)

type flexibleFloat float64

func (f *flexibleFloat) UnmarshalJSON(raw []byte) error {
	var number float64
	if err := json.Unmarshal(raw, &number); err == nil {
		*f = flexibleFloat(number)
		return nil
	}
	var text string
	if err := json.Unmarshal(raw, &text); err != nil {
		return err
	}
	parsed, err := strconv.ParseFloat(text, 64)
	if err != nil {
		return fmt.Errorf("parse decimal %q: %w", text, err)
	}
	*f = flexibleFloat(parsed)
	return nil
}

func (f flexibleFloat) MarshalJSON() ([]byte, error) {
	return json.Marshal(float64(f))
}

type user struct {
	ID        string `json:"id"`
	Email     string `json:"email"`
	FirstName string `json:"firstName,omitempty"`
	LastName  string `json:"lastName,omitempty"`
	IsActive  bool   `json:"isActive"`
}

type account struct {
	ID                      string         `json:"id"`
	Name                    string         `json:"name"`
	AccountType             string         `json:"accountType"`
	Currency                string         `json:"currency"`
	Balance                 flexibleFloat  `json:"balance"`
	IsActive                bool           `json:"isActive"`
	BankName                *string        `json:"bankName,omitempty"`
	FamilyWalletID          *string        `json:"familyWalletId,omitempty"`
	CreditLimit             *flexibleFloat `json:"creditLimit,omitempty"`
	ClosingDay              *int           `json:"closingDay,omitempty"`
	DueDay                  *int           `json:"dueDay,omitempty"`
	Issuer                  string         `json:"issuer,omitempty"`
	CreditCardID            *string        `json:"creditCardId,omitempty"`
	MinimumPayment          *flexibleFloat `json:"minimumPayment,omitempty"`
	TEA                     *flexibleFloat `json:"tea,omitempty"`
	TCEA                    *flexibleFloat `json:"tcea,omitempty"`
	CashAdvanceRate         *flexibleFloat `json:"cashAdvanceRate,omitempty"`
	AnnualFee               *flexibleFloat `json:"annualFee,omitempty"`
	NextClosingDate         *string        `json:"nextClosingDate,omitempty"`
	NextDueDate             *string        `json:"nextDueDate,omitempty"`
	DefaultPaymentAccountID *string        `json:"defaultPaymentAccountId,omitempty"`
}

type creditCard struct {
	ID             string    `json:"id"`
	Name           string    `json:"name"`
	Issuer         string    `json:"issuer,omitempty"`
	ContractStatus string    `json:"contractStatus"`
	ArchivedAt     *string   `json:"archivedAt,omitempty"`
	CreatedAt      string    `json:"createdAt,omitempty"`
	UpdatedAt      string    `json:"updatedAt,omitempty"`
	Currencies     []account `json:"currencies"`
}

type creditCardPayment struct {
	ID             string         `json:"id"`
	DebitedAmount  flexibleFloat  `json:"debitedAmount"`
	AppliedAmount  flexibleFloat  `json:"appliedAmount"`
	ExchangeRate   *flexibleFloat `json:"exchangeRate,omitempty"`
	IdempotencyKey string         `json:"idempotencyKey"`
	CreatedAt      string         `json:"createdAt,omitempty"`
}

type category struct {
	ID       string  `json:"id"`
	Name     string  `json:"name"`
	Type     string  `json:"type"`
	Color    string  `json:"color"`
	ParentID *string `json:"parentId,omitempty"`
}

type tag struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Color string `json:"color"`
}

type transaction struct {
	ID               string         `json:"id"`
	AccountID        string         `json:"accountId"`
	Amount           flexibleFloat  `json:"amount"`
	Kind             string         `json:"kind"`
	OriginalCurrency string         `json:"originalCurrency"`
	ExchangeRate     *flexibleFloat `json:"exchangeRate,omitempty"`
	Description      string         `json:"description"`
	Date             string         `json:"date"`
	Category         *category      `json:"category,omitempty"`
	Account          *account       `json:"account,omitempty"`
	Tags             []tag          `json:"tags,omitempty"`
	TransactionType  string         `json:"transactionType,omitempty"`
	IdempotencyKey   *string        `json:"idempotencyKey,omitempty"`
}

type loan struct {
	ID                      string        `json:"id"`
	Name                    string        `json:"name"`
	Lender                  string        `json:"lender,omitempty"`
	Principal               flexibleFloat `json:"principal"`
	RemainingBalance        flexibleFloat `json:"remainingBalance"`
	InterestRate            flexibleFloat `json:"interestRate"`
	Currency                string        `json:"currency"`
	TermMonths              int           `json:"termMonths"`
	MonthlyPayment          flexibleFloat `json:"monthlyPayment"`
	StartDate               string        `json:"startDate"`
	DueDate                 *string       `json:"dueDate,omitempty"`
	LiabilityAccountID      *string       `json:"liabilityAccountId,omitempty"`
	DefaultPaymentAccountID *string       `json:"defaultPaymentAccountId,omitempty"`
	IsActive                bool          `json:"isActive"`
	ArchivedAt              *string       `json:"archivedAt,omitempty"`
}

type loanPayment struct {
	ID               string                  `json:"id"`
	Amount           flexibleFloat           `json:"amount"`
	PrincipalPortion flexibleFloat           `json:"principalPortion"`
	InterestPortion  flexibleFloat           `json:"interestPortion"`
	Date             string                  `json:"date"`
	FromAccountID    *string                 `json:"fromAccountId,omitempty"`
	TransactionID    *string                 `json:"transactionId,omitempty"`
	SourceAmount     *flexibleFloat          `json:"sourceAmount,omitempty"`
	AppliedAmount    *flexibleFloat          `json:"appliedAmount,omitempty"`
	ExchangeRate     *flexibleFloat          `json:"exchangeRate,omitempty"`
	Allocations      []loanPaymentAllocation `json:"allocations,omitempty"`
}

type loanInstallment struct {
	ID            string        `json:"id"`
	Number        int           `json:"number"`
	DueDate       string        `json:"dueDate"`
	Principal     flexibleFloat `json:"principal"`
	Interest      flexibleFloat `json:"interest"`
	Fee           flexibleFloat `json:"fee"`
	LateFee       flexibleFloat `json:"lateFee"`
	PaidPrincipal flexibleFloat `json:"paidPrincipal"`
	PaidInterest  flexibleFloat `json:"paidInterest"`
	PaidFee       flexibleFloat `json:"paidFee"`
	PaidLateFee   flexibleFloat `json:"paidLateFee"`
	Status        string        `json:"status"`
	Total         flexibleFloat `json:"total"`
	PaidTotal     flexibleFloat `json:"paidTotal"`
}

type loanPaymentAllocation struct {
	InstallmentID string        `json:"installmentId"`
	Principal     flexibleFloat `json:"principal"`
	Interest      flexibleFloat `json:"interest"`
	Fee           flexibleFloat `json:"fee"`
	LateFee       flexibleFloat `json:"lateFee"`
}

type subscription struct {
	ID               string         `json:"id"`
	Name             string         `json:"name"`
	Amount           flexibleFloat  `json:"amount"`
	Currency         string         `json:"currency"`
	BillingCycle     string         `json:"billingCycle"`
	AmountType       string         `json:"amountType"`
	ChargeMode       string         `json:"chargeMode"`
	NextRenewalDate  string         `json:"nextRenewalDate"`
	NextChargeDate   string         `json:"nextChargeDate"`
	DueDate          *string        `json:"dueDate,omitempty"`
	NextChargeAmount *flexibleFloat `json:"nextChargeAmount,omitempty"`
	DueDay           *int           `json:"dueDay,omitempty"`
	CategoryID       *string        `json:"categoryId,omitempty"`
	ArchivedAt       *string        `json:"archivedAt,omitempty"`
	IsActive         bool           `json:"isActive"`
}

type subscriptionCharge struct {
	ID             string        `json:"id"`
	PlannedAmount  flexibleFloat `json:"plannedAmount"`
	ActualAmount   flexibleFloat `json:"actualAmount"`
	ScheduledFor   string        `json:"scheduledFor"`
	ChargedAt      string        `json:"chargedAt"`
	Status         string        `json:"status"`
	IdempotencyKey *string       `json:"idempotencyKey,omitempty"`
	TransactionID  *string       `json:"transactionId,omitempty"`
}

type budget struct {
	ID           string        `json:"id"`
	Category     category      `json:"category"`
	MonthlyLimit flexibleFloat `json:"monthlyLimit"`
	Currency     string        `json:"currency"`
	IsActive     bool          `json:"isActive"`
}
