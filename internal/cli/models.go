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
	ID             string        `json:"id"`
	Name           string        `json:"name"`
	AccountType    string        `json:"accountType"`
	Currency       string        `json:"currency"`
	Balance        flexibleFloat `json:"balance"`
	IsActive       bool          `json:"isActive"`
	BankName       *string       `json:"bankName,omitempty"`
	FamilyWalletID *string       `json:"familyWalletId,omitempty"`
}

type category struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Type  string `json:"type"`
	Color string `json:"color"`
}

type tag struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Color string `json:"color"`
}

type transaction struct {
	ID               string        `json:"id"`
	AccountID        string        `json:"accountId"`
	Amount           flexibleFloat `json:"amount"`
	Kind             string        `json:"kind"`
	OriginalCurrency string        `json:"originalCurrency"`
	ExchangeRate     *flexibleFloat `json:"exchangeRate,omitempty"`
	Description      string         `json:"description"`
	Date             string         `json:"date"`
	Category         *category      `json:"category,omitempty"`
	Account          *account       `json:"account,omitempty"`
	Tags             []tag          `json:"tags,omitempty"`
}
