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
	ID             string         `json:"id"`
	Name           string         `json:"name"`
	AccountType    string         `json:"accountType"`
	Currency       string         `json:"currency"`
	Balance        flexibleFloat  `json:"balance"`
	IsActive       bool           `json:"isActive"`
	BankName       *string        `json:"bankName,omitempty"`
	FamilyWalletID *string        `json:"familyWalletId,omitempty"`
	CreditLimit    *flexibleFloat `json:"creditLimit,omitempty"`
	ClosingDay     *int           `json:"closingDay,omitempty"`
	DueDay         *int           `json:"dueDay,omitempty"`
	Issuer         string         `json:"issuer,omitempty"`
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
	ID                 string        `json:"id"`
	Name               string        `json:"name"`
	Lender             string        `json:"lender,omitempty"`
	Principal          flexibleFloat `json:"principal"`
	RemainingBalance   flexibleFloat `json:"remainingBalance"`
	InterestRate       flexibleFloat `json:"interestRate"`
	Currency           string        `json:"currency"`
	TermMonths         int           `json:"termMonths"`
	MonthlyPayment     flexibleFloat `json:"monthlyPayment"`
	StartDate          string        `json:"startDate"`
	DueDate            *string       `json:"dueDate,omitempty"`
	LiabilityAccountID *string       `json:"liabilityAccountId,omitempty"`
	IsActive           bool          `json:"isActive"`
}

type loanPayment struct {
	ID               string        `json:"id"`
	Amount           flexibleFloat `json:"amount"`
	PrincipalPortion flexibleFloat `json:"principalPortion"`
	InterestPortion  flexibleFloat `json:"interestPortion"`
	Date             string        `json:"date"`
	FromAccountID    *string       `json:"fromAccountId,omitempty"`
	TransactionID    *string       `json:"transactionId,omitempty"`
}

type subscription struct {
	ID              string        `json:"id"`
	Name            string        `json:"name"`
	Amount          flexibleFloat `json:"amount"`
	Currency        string        `json:"currency"`
	BillingCycle    string        `json:"billingCycle"`
	NextRenewalDate string        `json:"nextRenewalDate"`
	DueDay          *int          `json:"dueDay,omitempty"`
	CategoryID      *string       `json:"categoryId,omitempty"`
	IsActive        bool          `json:"isActive"`
}

type budget struct {
	ID           string        `json:"id"`
	Category     category      `json:"category"`
	MonthlyLimit flexibleFloat `json:"monthlyLimit"`
	Currency     string        `json:"currency"`
	IsActive     bool          `json:"isActive"`
}
