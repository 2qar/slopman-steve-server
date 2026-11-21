package guild

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"herosiege/internal/save"
)

var ErrNotMember = errors.New("account is not a guild member")
var ErrNotOwner = errors.New("account is not the guild owner")

type Guild struct {
	ID          int64
	Name        string
	Tag         string
	Description string
	Color       int64
	Status      int64
}

type Member struct {
	Name      string
	Class     int64
	AccountID int64
	Slot      int64
	Status    int64
}

type AccountGuild struct {
	GuildID int64
	Slot    int64
	Name    string
}

// Manager owns guilds and memberships.
type Manager struct {
	db   *sql.DB
	save *save.Manager
}

func NewManager(db *sql.DB, saves *save.Manager) *Manager {
	return &Manager{db: db, save: saves}
}

// CreateGuild creates a guild and makes (accountID, slot) its owner.
func (m *Manager) CreateGuild(ctx context.Context, name, tag, description string, color int64, accountID, slot int64) (int64, error) {
	now := time.Now().Unix()
	res, err := m.db.ExecContext(ctx, `
		INSERT INTO guilds (name, tag, description, color, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`, name, tag, description, color, now, now)
	if err != nil {
		return 0, fmt.Errorf("insert guild: %w", err)
	}
	guildID, err := res.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("read guild id: %w", err)
	}

	if _, err := m.db.ExecContext(ctx, `
		INSERT OR REPLACE INTO guild_members (guild_id, account_id, slot, status, joined_at)
		VALUES (?, ?, ?, 1, ?)
	`, guildID, accountID, slot, now); err != nil {
		return 0, fmt.Errorf("insert owner: %w", err)
	}
	return guildID, nil
}

// GetForPlayer returns the guild membership for an account and character slot.
// A missing membership is returned as Guild{ID: 0}.
func (m *Manager) GetForPlayer(ctx context.Context, accountID, slot int64) (*Guild, error) {
	row := m.db.QueryRowContext(ctx, `
		SELECT g.id, g.name, g.tag, g.description, g.color, gm.status
		FROM guild_members gm
		JOIN guilds g ON g.id = gm.guild_id
		WHERE gm.account_id = ? AND gm.slot = ?
	`, accountID, slot)

	var g Guild
	err := row.Scan(&g.ID, &g.Name, &g.Tag, &g.Description, &g.Color, &g.Status)
	if errors.Is(err, sql.ErrNoRows) {
		return &Guild{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("query guild for player: %w", err)
	}
	return &g, nil
}

// ListForAccount returns every guild membership for an account.
func (m *Manager) ListForAccount(ctx context.Context, accountID int64) ([]AccountGuild, error) {
	rows, err := m.db.QueryContext(ctx, `
		SELECT g.id, gm.slot, g.name
		FROM guild_members gm
		JOIN guilds g ON g.id = gm.guild_id
		WHERE gm.account_id = ?
		ORDER BY gm.slot
	`, accountID)
	if err != nil {
		return nil, fmt.Errorf("query account guilds: %w", err)
	}
	defer rows.Close()

	var guilds []AccountGuild
	for rows.Next() {
		var g AccountGuild
		if err := rows.Scan(&g.GuildID, &g.Slot, &g.Name); err != nil {
			return nil, fmt.Errorf("scan account guild: %w", err)
		}
		guilds = append(guilds, g)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate account guilds: %w", err)
	}
	return guilds, nil
}

// Members returns a page of up to 100 guild members.
func (m *Manager) Members(ctx context.Context, guildID, page int64) ([]Member, error) {
	offset := page * 100
	rows, err := m.db.QueryContext(ctx, `
		SELECT a.username, gm.account_id, gm.slot, gm.status
		FROM guild_members gm
		JOIN accounts a ON a.id = gm.account_id
		WHERE gm.guild_id = ?
		ORDER BY gm.status DESC, gm.joined_at ASC
		LIMIT 100 OFFSET ?
	`, guildID, offset)
	if err != nil {
		return nil, fmt.Errorf("query guild members: %w", err)
	}
	defer rows.Close()

	var members []Member
	for rows.Next() {
		var mem Member
		if err := rows.Scan(&mem.Name, &mem.AccountID, &mem.Slot, &mem.Status); err != nil {
			return nil, fmt.Errorf("scan guild member: %w", err)
		}
		members = append(members, mem)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate guild members: %w", err)
	}

	// Resolve classes after the rows are closed: the database is configured
	// with a single connection, and querying inside the loop would deadlock.
	for i := range members {
		members[i].Class = m.classForSlot(ctx, members[i].AccountID, members[i].Slot)
	}
	return members, nil
}

// SetMember inserts or updates a membership.
func (m *Manager) SetMember(ctx context.Context, guildID, accountID, slot, status int64) error {
	_, err := m.db.ExecContext(ctx, `
		INSERT OR REPLACE INTO guild_members (guild_id, account_id, slot, status, joined_at)
		VALUES (?, ?, ?, ?, ?)
	`, guildID, accountID, slot, status, time.Now().Unix())
	if err != nil {
		return fmt.Errorf("set guild member: %w", err)
	}
	return nil
}

// RemoveMember removes a membership.
func (m *Manager) RemoveMember(ctx context.Context, guildID, accountID, slot int64) error {
	_, err := m.db.ExecContext(ctx, `
		DELETE FROM guild_members WHERE guild_id = ? AND account_id = ? AND slot = ?
	`, guildID, accountID, slot)
	if err != nil {
		return fmt.Errorf("remove guild member: %w", err)
	}
	return nil
}

// DeleteGuild deletes a guild and its memberships if accountID owns it.
func (m *Manager) DeleteGuild(ctx context.Context, guildID, accountID int64) error {
	var status int64
	err := m.db.QueryRowContext(ctx, `
		SELECT status FROM guild_members WHERE guild_id = ? AND account_id = ?
	`, guildID, accountID).Scan(&status)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrNotMember
	}
	if err != nil {
		return fmt.Errorf("query owner: %w", err)
	}
	if status != 1 {
		return ErrNotOwner
	}

	if _, err := m.db.ExecContext(ctx, `DELETE FROM guild_members WHERE guild_id = ?`, guildID); err != nil {
		return fmt.Errorf("delete guild members: %w", err)
	}
	if _, err := m.db.ExecContext(ctx, `DELETE FROM guilds WHERE id = ?`, guildID); err != nil {
		return fmt.Errorf("delete guild: %w", err)
	}
	return nil
}

// EditDescription updates a guild description.
func (m *Manager) EditDescription(ctx context.Context, guildID, accountID int64, description string) error {
	var status int64
	err := m.db.QueryRowContext(ctx, `
		SELECT status FROM guild_members WHERE guild_id = ? AND account_id = ?
	`, guildID, accountID).Scan(&status)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrNotMember
	}
	if err != nil {
		return fmt.Errorf("query member: %w", err)
	}
	if status != 1 {
		return ErrNotOwner
	}

	_, err = m.db.ExecContext(ctx, `UPDATE guilds SET description = ?, updated_at = ? WHERE id = ?`, description, time.Now().Unix(), guildID)
	if err != nil {
		return fmt.Errorf("update guild: %w", err)
	}
	return nil
}

func (m *Manager) classForSlot(ctx context.Context, accountID, slot int64) int64 {
	s, err := m.save.LoadSlot(ctx, accountID, int(slot))
	if err != nil || s.SlotData == "0" {
		return 0
	}
	fields := strings.Split(s.SlotData, "|")
	if len(fields) < 6 {
		return 0
	}
	var class int64
	fmt.Sscanf(fields[5], "%d", &class)
	return class
}
