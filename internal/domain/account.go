// Copyright (c) 2026 Michael D Henderson.

package domain

import "github.com/maloquacious/gl/money"

// AccountInfo is an account reference and its description.
type AccountInfo struct {
	AccountID   AccountID
	Description string
}

// NewAccountInfo validates an account reference and description.
func NewAccountInfo(accountID, description string) (AccountInfo, error) {
	id, err := NewAccountID(accountID)
	if err != nil {
		return AccountInfo{}, err
	}
	if err := ValidateAccountName(description); err != nil {
		return AccountInfo{}, err
	}
	return AccountInfo{AccountID: id, Description: description}, nil
}

// Account describes an account and its balance.
//
// The balance is stored on the account and updated inside the posting
// transaction, not derived on read: get_all_accounts returns a balance for every
// account, and deriving each one would scan every entry in the ledger. The
// integrity check that reconciles stored balances against posted entries is what
// keeps the stored value honest.
type Account struct {
	Info             AccountInfo
	IsControlAccount bool
	Balance          money.Money
}

// AccountID returns the account reference.
func (a Account) AccountID() AccountID { return a.Info.AccountID }

// HasBalance reports whether the account carries a non-zero balance, which is one
// of the three conditions that make an account too much in use to remove. The
// other two - entries and transactions referencing it - are counted by the store.
func (a Account) HasBalance() bool { return !a.Balance.IsZero() }
