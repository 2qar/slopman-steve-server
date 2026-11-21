package session

import (
	"context"
	"fmt"

	"herosiege/internal/account"
)

// Manager is a thin session layer over accounts. Hero Siege does not use a
// bearer token; a session is the (account_id, identifier) pair returned by
// login and sent with every subsequent request.
type Manager struct {
	accounts *account.Manager
}

func NewManager(accounts *account.Manager) *Manager {
	return &Manager{accounts: accounts}
}

// Login authenticates a user and marks the account as active.
func (m *Manager) Login(ctx context.Context, username, passwordMD5 string) (*account.Account, error) {
	acc, err := m.accounts.Authenticate(ctx, username, passwordMD5)
	if err != nil {
		return nil, err
	}
	if err := m.accounts.Touch(ctx, acc.ID); err != nil {
		return nil, fmt.Errorf("touch account: %w", err)
	}
	return acc, nil
}

// Validate authenticates a request using the game's persistent session pair.
func (m *Manager) Validate(ctx context.Context, accountID, identifier string) (*account.Account, error) {
	acc, err := m.accounts.Validate(ctx, accountID, identifier)
	if err != nil {
		return nil, err
	}
	if err := m.accounts.Touch(ctx, acc.ID); err != nil {
		return nil, fmt.Errorf("touch account: %w", err)
	}
	return acc, nil
}
