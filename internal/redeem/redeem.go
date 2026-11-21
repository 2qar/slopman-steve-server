package redeem

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

var ErrCodeNotFound = errors.New("redeem code not found")
var ErrCodeUsed = errors.New("redeem code already used")

type Code struct {
	Code     string
	Kind     int64
	Amount   int64
	Slot     int64
	Name     string
	Class    int64
	Level    int64
	Hardcore int64
}

// Manager owns redeem codes.
type Manager struct {
	db *sql.DB
}

func NewManager(db *sql.DB) *Manager {
	return &Manager{db: db}
}

// Redeem marks a code used and returns it.
func (m *Manager) Redeem(ctx context.Context, code string, accountID int64) (*Code, error) {
	now := time.Now().Unix()
	res, err := m.db.ExecContext(ctx, `
		UPDATE redeem_codes
		SET used_by = ?, used_at = ?
		WHERE code = ? AND used_by IS NULL
	`, accountID, now, code)
	if err != nil {
		return nil, fmt.Errorf("redeem code: %w", err)
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return nil, fmt.Errorf("read redeem rows affected: %w", err)
	}
	if affected == 0 {
		var usedBy sql.NullInt64
		err := m.db.QueryRowContext(ctx, `SELECT used_by FROM redeem_codes WHERE code = ?`, code).Scan(&usedBy)
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrCodeNotFound
		}
		if err != nil {
			return nil, fmt.Errorf("query redeem code: %w", err)
		}
		return nil, ErrCodeUsed
	}

	row := m.db.QueryRowContext(ctx, `
		SELECT code, kind, amount, slot, name, class, level, hardcore
		FROM redeem_codes WHERE code = ?
	`, code)

	var c Code
	if err := row.Scan(&c.Code, &c.Kind, &c.Amount, &c.Slot, &c.Name, &c.Class, &c.Level, &c.Hardcore); err != nil {
		return nil, fmt.Errorf("scan redeem code: %w", err)
	}
	return &c, nil
}
