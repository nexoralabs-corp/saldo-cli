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
	CreditLineRateToBase    *flexibleFloat `json:"creditLineRateToBase,omitempty"`
}

type creditCard struct {
	ID                  string                  `json:"id"`
	Name                string                  `json:"name"`
	Issuer              string                  `json:"issuer,omitempty"`
	ContractStatus      string                  `json:"contractStatus"`
	ArchivedAt          *string                 `json:"archivedAt,omitempty"`
	CreatedAt           string                  `json:"createdAt,omitempty"`
	UpdatedAt           string                  `json:"updatedAt,omitempty"`
	Currencies          []account               `json:"currencies"`
	CreditLimitMode     string                  `json:"creditLimitMode,omitempty"`
	SharedCreditLimit   *flexibleFloat          `json:"sharedCreditLimit,omitempty"`
	SharedLimitCurrency *string                 `json:"sharedLimitCurrency,omitempty"`
	LimitSummary        *creditCardLimitSummary `json:"limitSummary,omitempty"`
}

type creditCardLimitCurrencySummary struct {
	Currency    string         `json:"currency"`
	Debt        flexibleFloat  `json:"debt"`
	RateToBase  *flexibleFloat `json:"rateToBase,omitempty"`
	UsedInBase  *flexibleFloat `json:"usedInBase,omitempty"`
	CreditLimit *flexibleFloat `json:"creditLimit,omitempty"`
}

type creditCardLimitSummary struct {
	Mode        string                           `json:"mode"`
	Currency    *string                          `json:"currency,omitempty"`
	Limit       *flexibleFloat                   `json:"limit,omitempty"`
	Used        *flexibleFloat                   `json:"used,omitempty"`
	Available   *flexibleFloat                   `json:"available,omitempty"`
	Utilization *flexibleFloat                   `json:"utilization,omitempty"`
	MissingRate bool                             `json:"missingRate"`
	Currencies  []creditCardLimitCurrencySummary `json:"currencies"`
}

type creditCardBalanceAdjustment struct {
	ID              string        `json:"id"`
	CardAccountID   string        `json:"cardAccountId"`
	PreviousBalance flexibleFloat `json:"previousBalance"`
	TargetAmount    flexibleFloat `json:"targetAmount"`
	BalanceSide     string        `json:"balanceSide"`
	Delta           flexibleFloat `json:"delta"`
	EffectiveDate   string        `json:"effectiveDate"`
	Reason          string        `json:"reason"`
	Note            string        `json:"note,omitempty"`
	Source          string        `json:"source"`
	TransactionID   *string       `json:"transactionId,omitempty"`
	IdempotencyKey  string        `json:"idempotencyKey"`
}

type creditCardStatementEntry struct {
	ID            string        `json:"id"`
	OperationDate *string       `json:"operationDate,omitempty"`
	Description   string        `json:"description"`
	Amount        flexibleFloat `json:"amount"`
	Currency      string        `json:"currency"`
	IsCredit      bool          `json:"isCredit"`
	EntryType     string        `json:"entryType"`
	Status        string        `json:"status"`
	TransactionID *string       `json:"transactionId,omitempty"`
}

type creditCardStatement struct {
	ID               string                     `json:"id"`
	CardAccountID    string                     `json:"cardAccountId"`
	ClosingDate      string                     `json:"closingDate"`
	DueDate          *string                    `json:"dueDate,omitempty"`
	OpeningBalance   flexibleFloat              `json:"openingBalance"`
	StatementBalance flexibleFloat              `json:"statementBalance"`
	MinimumPayment   *flexibleFloat             `json:"minimumPayment,omitempty"`
	TotalPayment     *flexibleFloat             `json:"totalPayment,omitempty"`
	Status           string                     `json:"status"`
	Source           string                     `json:"source"`
	Entries          []creditCardStatementEntry `json:"entries,omitempty"`
}

type creditCardStatementImport struct {
	ID         string                `json:"id"`
	Filename   string                `json:"filename"`
	MimeType   string                `json:"mimeType"`
	ParserName string                `json:"parserName"`
	Status     string                `json:"status"`
	Summary    map[string]any        `json:"summary"`
	Statements []creditCardStatement `json:"statements"`
}

type creditCardChargeRule struct {
	ID             string         `json:"id"`
	CardAccountID  string         `json:"cardAccountId"`
	Name           string         `json:"name"`
	ChargeType     string         `json:"chargeType"`
	NextChargeDate string         `json:"nextChargeDate"`
	Calculation    string         `json:"calculation"`
	FixedAmount    *flexibleFloat `json:"fixedAmount,omitempty"`
	Percentage     *flexibleFloat `json:"percentage,omitempty"`
	WaiverPolicy   string         `json:"waiverPolicy"`
	IsActive       bool           `json:"isActive"`
}

type creditCardChargeOccurrence struct {
	ID               string        `json:"id"`
	RuleID           string        `json:"ruleId"`
	ScheduledFor     string        `json:"scheduledFor"`
	BaseBalance      flexibleFloat `json:"baseBalance"`
	CalculatedAmount flexibleFloat `json:"calculatedAmount"`
	WaivedAmount     flexibleFloat `json:"waivedAmount"`
	FinalAmount      flexibleFloat `json:"finalAmount"`
	Status           string        `json:"status"`
	TransactionID    *string       `json:"transactionId,omitempty"`
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
	CreditCardID            *string       `json:"creditCardId,omitempty"`
	CreditCardAccountID     *string       `json:"creditCardAccountId,omitempty"`
	CollectionMode          string        `json:"collectionMode,omitempty"`
	ExternalReference       string        `json:"externalReference,omitempty"`
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
	Insurance     flexibleFloat `json:"insurance"`
	LateFee       flexibleFloat `json:"lateFee"`
	PaidPrincipal flexibleFloat `json:"paidPrincipal"`
	PaidInterest  flexibleFloat `json:"paidInterest"`
	PaidFee       flexibleFloat `json:"paidFee"`
	PaidInsurance flexibleFloat `json:"paidInsurance"`
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
	Insurance     flexibleFloat `json:"insurance"`
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
