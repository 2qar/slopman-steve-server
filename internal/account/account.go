package account

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

var (
	ErrAccountNotFound   = errors.New("account not found")
	ErrInvalidCredential = errors.New("invalid account credentials")
	ErrUsernameTaken     = errors.New("username already taken")
	ErrBanned            = errors.New("account is banned")
)

// Account is the persistent identity used by the game client.
type Account struct {
	ID          int64
	Username    string
	PasswordMD5 string
	Email       string
	SteamID     string
	Identifier  string
	Admin       bool
	Banned      bool
	CreatedAt   int64
	UpdatedAt   int64
	LastSeenAt  int64
}

// Manager owns account persistence and credential checks.
type Manager struct {
	db *sql.DB
}

func NewManager(db *sql.DB) *Manager {
	return &Manager{db: db}
}

// Register creates an account. passwordMD5 is the value the game client sends
// (an MD5 digest of the user's password).
func (m *Manager) Register(ctx context.Context, username, passwordMD5, email, steamID string) (*Account, error) {
	username = strings.TrimSpace(username)
	if !validUsername(username) || passwordMD5 == "" {
		return nil, ErrInvalidCredential
	}

	identifier, err := newIdentifier()
	if err != nil {
		return nil, fmt.Errorf("generate identifier: %w", err)
	}

	now := time.Now().Unix()
	res, err := m.db.ExecContext(ctx, `
		INSERT INTO accounts (username, password_md5, email, steam_id, identifier, created_at, updated_at, last_seen_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, username, passwordMD5, email, steamID, identifier, now, now, now)
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "unique") {
			return nil, ErrUsernameTaken
		}
		return nil, fmt.Errorf("insert account: %w", err)
	}

	id, err := res.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("read account id: %w", err)
	}
	return m.ByID(ctx, id)
}

// Authenticate checks username/passwordMD5 and returns the account.
func (m *Manager) Authenticate(ctx context.Context, username, passwordMD5 string) (*Account, error) {
	acc, err := m.ByUsername(ctx, username)
	if err != nil {
		return nil, ErrInvalidCredential
	}
	if acc.PasswordMD5 != passwordMD5 {
		return nil, ErrInvalidCredential
	}
	if acc.Banned {
		return nil, ErrBanned
	}
	return acc, nil
}

// Validate checks the account_id/identifier pair the game uses as a session.
func (m *Manager) Validate(ctx context.Context, accountID, identifier string) (*Account, error) {
	if accountID == "" || identifier == "" {
		return nil, ErrInvalidCredential
	}
	id, err := strconv.ParseInt(accountID, 10, 64)
	if err != nil {
		return nil, ErrInvalidCredential
	}
	acc, err := m.ByID(ctx, id)
	if err != nil {
		return nil, ErrInvalidCredential
	}
	if acc.Identifier != identifier {
		return nil, ErrInvalidCredential
	}
	if acc.Banned {
		return nil, ErrBanned
	}
	return acc, nil
}

func (m *Manager) ByID(ctx context.Context, id int64) (*Account, error) {
	row := m.db.QueryRowContext(ctx, `
		SELECT id, username, password_md5, email, steam_id, identifier, admin, banned, created_at, updated_at, last_seen_at
		FROM accounts WHERE id = ?`, id)
	return scanAccount(row)
}

func (m *Manager) ByUsername(ctx context.Context, username string) (*Account, error) {
	row := m.db.QueryRowContext(ctx, `
		SELECT id, username, password_md5, email, steam_id, identifier, admin, banned, created_at, updated_at, last_seen_at
		FROM accounts WHERE username = ?`, username)
	return scanAccount(row)
}

// Touch updates the account's last-seen timestamp.
func (m *Manager) Touch(ctx context.Context, id int64) error {
	_, err := m.db.ExecContext(ctx, `UPDATE accounts SET last_seen_at = ? WHERE id = ?`, time.Now().Unix(), id)
	if err != nil {
		return fmt.Errorf("touch account: %w", err)
	}
	return nil
}

func scanAccount(row *sql.Row) (*Account, error) {
	var acc Account
	var admin, banned int64
	err := row.Scan(
		&acc.ID,
		&acc.Username,
		&acc.PasswordMD5,
		&acc.Email,
		&acc.SteamID,
		&acc.Identifier,
		&admin,
		&banned,
		&acc.CreatedAt,
		&acc.UpdatedAt,
		&acc.LastSeenAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrAccountNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("scan account: %w", err)
	}
	acc.Admin = admin != 0
	acc.Banned = banned != 0
	return &acc, nil
}

func newIdentifier() (string, error) {
	raw := make([]byte, 16)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	return hex.EncodeToString(raw), nil
}

func validUsername(username string) bool {
	if len(username) < 1 || len(username) > 24 {
		return false
	}
	for _, r := range username {
		if r < 0x20 || r > 0x7e || strings.ContainsRune("|§€£$@", r) {
			return false
		}
	}
	return true
}
