// Copyright (c) 2026 Michael D Henderson.

package domain_test

import (
	"errors"
	"testing"

	"github.com/maloquacious/gl/internal/domain"
	"github.com/maloquacious/gl/money"
)

func TestNewAccountInfo(t *testing.T) {
	tests := []struct {
		name        string
		accountID   string
		description string
		want        error
	}{
		{name: "valid", accountID: "1000", description: "Cash"},
		{name: "bad id", accountID: "", description: "Cash", want: domain.ErrBadAccountID},
		{name: "bad description", accountID: "1000", description: "", want: domain.ErrBadAccountName},
		{name: "id checked before description", accountID: "", description: "", want: domain.ErrBadAccountID},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := domain.NewAccountInfo(tt.accountID, tt.description)
			if tt.want != nil {
				if !errors.Is(err, tt.want) {
					t.Fatalf("error = %v; want %v", err, tt.want)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if got.AccountID != domain.AccountID(tt.accountID) || got.Description != tt.description {
				t.Fatalf("AccountInfo = %+v", got)
			}
		})
	}
}

func TestAccountHasBalance(t *testing.T) {
	account := domain.Account{
		Info:    domain.AccountInfo{AccountID: "1000", Description: "Cash"},
		Balance: money.MustNewMinor(0, money.USD),
	}
	if account.AccountID() != "1000" {
		t.Fatalf("AccountID() = %q; want 1000", account.AccountID())
	}
	// An account is created with a zero balance, which is one of the three things
	// removeAccount checks before deleting it.
	if account.HasBalance() {
		t.Fatal("HasBalance() = true for a zero balance")
	}

	account.Balance = money.MustNewMinor(-1, money.USD)
	if !account.HasBalance() {
		t.Fatal("HasBalance() = false for a non-zero balance")
	}
}
