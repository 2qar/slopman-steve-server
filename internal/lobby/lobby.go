package lobby

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

type Server struct {
	AccountID      int64
	IP             string
	Name           string
	Password       string
	CurrentPlayers int64
	MaxPlayers     int64
	SiegeMode      int64
	Hardcore       int64
	Dedicated      int64
	Region         int64
	Season         int64
	Level          int64
	ItemLevel      int64
	Wormhole       int64
	Public         int64
}

// Manager owns the lobby/game-server registry.
type Manager struct {
	db *sql.DB
}

func NewManager(db *sql.DB) *Manager {
	return &Manager{db: db}
}

// Upsert registers a game server for the calling account. The caller's public
// IP is supplied by the HTTP layer from the request remote address.
func (m *Manager) Upsert(ctx context.Context, s Server, ip string) error {
	now := time.Now().Unix()
	_, err := m.db.ExecContext(ctx, `
		INSERT INTO game_servers (
			account_id, ip, name, password, current_players, max_players,
			siege_mode, hardcore, dedicated, region, season, level, item_level,
			wormhole, public, updated_at, last_seen_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(account_id) DO UPDATE SET
			ip = excluded.ip,
			name = excluded.name,
			password = excluded.password,
			current_players = excluded.current_players,
			max_players = excluded.max_players,
			siege_mode = excluded.siege_mode,
			hardcore = excluded.hardcore,
			dedicated = excluded.dedicated,
			region = excluded.region,
			season = excluded.season,
			level = excluded.level,
			item_level = excluded.item_level,
			wormhole = excluded.wormhole,
			public = excluded.public,
			updated_at = excluded.updated_at,
			last_seen_at = excluded.last_seen_at
	`, s.AccountID, ip, s.Name, s.Password, s.CurrentPlayers, s.MaxPlayers,
		s.SiegeMode, s.Hardcore, s.Dedicated, s.Region, s.Season, s.Level, s.ItemLevel,
		s.Wormhole, s.Public, now, now)
	if err != nil {
		return fmt.Errorf("upsert game server: %w", err)
	}
	return nil
}

// Touch refreshes a server's player count and last-seen time.
func (m *Manager) Touch(ctx context.Context, accountID, currentPlayers int64) error {
	now := time.Now().Unix()
	_, err := m.db.ExecContext(ctx, `
		UPDATE game_servers SET current_players = ?, last_seen_at = ?, updated_at = ?
		WHERE account_id = ?
	`, currentPlayers, now, now, accountID)
	if err != nil {
		return fmt.Errorf("touch game server: %w", err)
	}
	return nil
}

// Delete removes the caller's server.
func (m *Manager) Delete(ctx context.Context, accountID int64) error {
	_, err := m.db.ExecContext(ctx, `DELETE FROM game_servers WHERE account_id = ?`, accountID)
	if err != nil {
		return fmt.Errorf("delete game server: %w", err)
	}
	return nil
}

// List returns servers matching the client's filters.
func (m *Manager) List(ctx context.Context, siegeMode, hardcore, region int64) ([]Server, error) {
	rows, err := m.db.QueryContext(ctx, `
		SELECT account_id, ip, name, password, current_players, max_players,
		       siege_mode, hardcore, dedicated, region, season, level, item_level, wormhole, public
		FROM game_servers
		WHERE siege_mode = ? AND hardcore = ? AND region = ?
		ORDER BY public DESC, updated_at DESC
	`, siegeMode, hardcore, region)
	if err != nil {
		return nil, fmt.Errorf("query game servers: %w", err)
	}
	defer rows.Close()

	var servers []Server
	for rows.Next() {
		var s Server
		if err := rows.Scan(
			&s.AccountID, &s.IP, &s.Name, &s.Password, &s.CurrentPlayers, &s.MaxPlayers,
			&s.SiegeMode, &s.Hardcore, &s.Dedicated, &s.Region, &s.Season, &s.Level,
			&s.ItemLevel, &s.Wormhole, &s.Public,
		); err != nil {
			return nil, fmt.Errorf("scan game server: %w", err)
		}
		servers = append(servers, s)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate game servers: %w", err)
	}
	return servers, nil
}

// Matchmaking finds a public server compatible with the request.
func (m *Manager) Matchmaking(ctx context.Context, accountID, hardcore, region, season int64) (*Server, error) {
	row := m.db.QueryRowContext(ctx, `
		SELECT account_id, ip, name, password, current_players, max_players,
		       siege_mode, hardcore, dedicated, region, season, level, item_level, wormhole, public
		FROM game_servers
		WHERE hardcore = ? AND region = ? AND season = ? AND public = 1
		  AND account_id != ? AND current_players < max_players
		ORDER BY updated_at DESC
		LIMIT 1
	`, hardcore, region, season, accountID)

	var s Server
	err := row.Scan(
		&s.AccountID, &s.IP, &s.Name, &s.Password, &s.CurrentPlayers, &s.MaxPlayers,
		&s.SiegeMode, &s.Hardcore, &s.Dedicated, &s.Region, &s.Season, &s.Level,
		&s.ItemLevel, &s.Wormhole, &s.Public,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("query matchmaking: %w", err)
	}
	return &s, nil
}

// OnlinePlayers returns the number of players currently in active lobbies.
func (m *Manager) OnlinePlayers(ctx context.Context, maxAge time.Duration) (int64, error) {
	since := time.Now().Add(-maxAge).Unix()
	var total int64
	err := m.db.QueryRowContext(ctx, `
		SELECT COALESCE(SUM(current_players), 0) FROM game_servers WHERE last_seen_at >= ?
	`, since).Scan(&total)
	if err != nil {
		return 0, fmt.Errorf("query online players: %w", err)
	}
	return total, nil
}
