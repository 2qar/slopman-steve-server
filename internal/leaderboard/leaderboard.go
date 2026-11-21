package leaderboard

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

type WormholeEntry struct {
	AccountIDs string
	PlayerData string
	Wormhole   int64
}

type SiegeEntry struct {
	AccountIDs string
	PlayerData string
	Wave       int64
}

type HeroLevelEntry struct {
	Name      string
	Class     int64
	HeroLevel int64
}

// Manager owns leaderboard persistence and ranking.
type Manager struct {
	db *sql.DB
}

func NewManager(db *sql.DB) *Manager {
	return &Manager{db: db}
}

func (m *Manager) SubmitWormhole(ctx context.Context, accountID int64, accountIDs, slots, playerData string, players, wormhole, hardcore, beta int64) error {
	now := time.Now().Unix()
	var current int64
	err := m.db.QueryRowContext(ctx, `
		SELECT wormhole FROM wormhole_scores
		WHERE account_ids = ? AND players = ? AND hardcore = ? AND beta = ?
	`, accountIDs, players, hardcore, beta).Scan(&current)
	if errors.Is(err, sql.ErrNoRows) {
		_, err := m.db.ExecContext(ctx, `
			INSERT INTO wormhole_scores (account_id, account_ids, slots, players, player_data, wormhole, hardcore, beta, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		`, accountID, accountIDs, slots, players, playerData, wormhole, hardcore, beta, now)
		if err != nil {
			return fmt.Errorf("insert wormhole score: %w", err)
		}
		return nil
	}
	if err != nil {
		return fmt.Errorf("query wormhole score: %w", err)
	}

	if wormhole <= current {
		return nil
	}
	_, err = m.db.ExecContext(ctx, `
		UPDATE wormhole_scores
		SET account_id = ?, slots = ?, player_data = ?, wormhole = ?, updated_at = ?
		WHERE account_ids = ? AND players = ? AND hardcore = ? AND beta = ?
	`, accountID, slots, playerData, wormhole, now, accountIDs, players, hardcore, beta)
	if err != nil {
		return fmt.Errorf("update wormhole score: %w", err)
	}
	return nil
}

func (m *Manager) BestWormhole(ctx context.Context, accountIDs, slots string, players, hardcore, beta int64) (int64, error) {
	var wormhole int64
	err := m.db.QueryRowContext(ctx, `
		SELECT wormhole FROM wormhole_scores
		WHERE account_ids = ? AND players = ? AND hardcore = ? AND beta = ?
	`, accountIDs, players, hardcore, beta).Scan(&wormhole)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, nil
	}
	if err != nil {
		return 0, fmt.Errorf("query best wormhole: %w", err)
	}
	return wormhole, nil
}

func (m *Manager) TopWormhole(ctx context.Context, players, hardcore, beta int64, limit int) ([]WormholeEntry, error) {
	rows, err := m.db.QueryContext(ctx, `
		SELECT account_ids, player_data, wormhole
		FROM wormhole_scores
		WHERE players = ? AND hardcore = ? AND beta = ?
		ORDER BY wormhole DESC
		LIMIT ?
	`, players, hardcore, beta, limit)
	if err != nil {
		return nil, fmt.Errorf("query wormhole top: %w", err)
	}
	defer rows.Close()

	var entries []WormholeEntry
	for rows.Next() {
		var e WormholeEntry
		if err := rows.Scan(&e.AccountIDs, &e.PlayerData, &e.Wormhole); err != nil {
			return nil, fmt.Errorf("scan wormhole top: %w", err)
		}
		entries = append(entries, e)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate wormhole top: %w", err)
	}
	return entries, nil
}

func (m *Manager) SubmitSiege(ctx context.Context, accountID int64, accountIDs, playerData string, players, wave, difficulty, hardcore int64) error {
	now := time.Now().Unix()
	var current int64
	err := m.db.QueryRowContext(ctx, `
		SELECT wave FROM siege_scores
		WHERE account_ids = ? AND players = ? AND difficulty = ? AND hardcore = ?
	`, accountIDs, players, difficulty, hardcore).Scan(&current)
	if errors.Is(err, sql.ErrNoRows) {
		_, err := m.db.ExecContext(ctx, `
			INSERT INTO siege_scores (account_id, account_ids, players, player_data, wave, difficulty, hardcore, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		`, accountID, accountIDs, players, playerData, wave, difficulty, hardcore, now)
		if err != nil {
			return fmt.Errorf("insert siege score: %w", err)
		}
		return nil
	}
	if err != nil {
		return fmt.Errorf("query siege score: %w", err)
	}
	if wave <= current {
		return nil
	}

	_, err = m.db.ExecContext(ctx, `
		UPDATE siege_scores
		SET account_id = ?, player_data = ?, wave = ?, updated_at = ?
		WHERE account_ids = ? AND players = ? AND difficulty = ? AND hardcore = ?
	`, accountID, playerData, wave, now, accountIDs, players, difficulty, hardcore)
	if err != nil {
		return fmt.Errorf("update siege score: %w", err)
	}
	return nil
}

func (m *Manager) BestSiege(ctx context.Context, accountIDs string, players, difficulty, hardcore int64) (int64, error) {
	var wave int64
	err := m.db.QueryRowContext(ctx, `
		SELECT wave FROM siege_scores
		WHERE account_ids = ? AND players = ? AND difficulty = ? AND hardcore = ?
	`, accountIDs, players, difficulty, hardcore).Scan(&wave)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, nil
	}
	if err != nil {
		return 0, fmt.Errorf("query best siege: %w", err)
	}
	return wave, nil
}

func (m *Manager) TopSiege(ctx context.Context, players, difficulty, hardcore int64, limit int) ([]SiegeEntry, error) {
	rows, err := m.db.QueryContext(ctx, `
		SELECT account_ids, player_data, wave
		FROM siege_scores
		WHERE players = ? AND difficulty = ? AND hardcore = ?
		ORDER BY wave DESC
		LIMIT ?
	`, players, difficulty, hardcore, limit)
	if err != nil {
		return nil, fmt.Errorf("query siege top: %w", err)
	}
	defer rows.Close()

	var entries []SiegeEntry
	for rows.Next() {
		var e SiegeEntry
		if err := rows.Scan(&e.AccountIDs, &e.PlayerData, &e.Wave); err != nil {
			return nil, fmt.Errorf("scan siege top: %w", err)
		}
		entries = append(entries, e)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate siege top: %w", err)
	}
	return entries, nil
}

func (m *Manager) SubmitHeroLevel(ctx context.Context, accountID int64, name string, class, heroLevel, hardcore, beta int64) error {
	now := time.Now().Unix()
	var current int64
	err := m.db.QueryRowContext(ctx, `
		SELECT hero_level FROM hero_level_scores
		WHERE account_id = ? AND hardcore = ? AND beta = ?
	`, accountID, hardcore, beta).Scan(&current)
	if errors.Is(err, sql.ErrNoRows) {
		_, err := m.db.ExecContext(ctx, `
			INSERT INTO hero_level_scores (account_id, name, class, hero_level, hardcore, beta, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, ?)
		`, accountID, name, class, heroLevel, hardcore, beta, now)
		if err != nil {
			return fmt.Errorf("insert hero level score: %w", err)
		}
		return nil
	}
	if err != nil {
		return fmt.Errorf("query hero level score: %w", err)
	}
	if heroLevel <= current {
		return nil
	}

	_, err = m.db.ExecContext(ctx, `
		UPDATE hero_level_scores
		SET name = ?, class = ?, hero_level = ?, updated_at = ?
		WHERE account_id = ? AND hardcore = ? AND beta = ?
	`, name, class, heroLevel, now, accountID, hardcore, beta)
	if err != nil {
		return fmt.Errorf("update hero level score: %w", err)
	}
	return nil
}

func (m *Manager) TopHeroLevel(ctx context.Context, hardcore, beta int64, limit int) ([]HeroLevelEntry, error) {
	rows, err := m.db.QueryContext(ctx, `
		SELECT name, class, hero_level
		FROM hero_level_scores
		WHERE hardcore = ? AND beta = ?
		ORDER BY hero_level DESC
		LIMIT ?
	`, hardcore, beta, limit)
	if err != nil {
		return nil, fmt.Errorf("query hero level top: %w", err)
	}
	defer rows.Close()

	var entries []HeroLevelEntry
	for rows.Next() {
		var e HeroLevelEntry
		if err := rows.Scan(&e.Name, &e.Class, &e.HeroLevel); err != nil {
			return nil, fmt.Errorf("scan hero level top: %w", err)
		}
		entries = append(entries, e)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate hero level top: %w", err)
	}
	return entries, nil
}

func (m *Manager) SubmitEndgame(ctx context.Context, accountID int64, name string, class, wormholeLevel, heroLevel, hardcore int64) error {
	now := time.Now().Unix()
	var current int64
	err := m.db.QueryRowContext(ctx, `
		SELECT wormhole_level FROM endgame_scores
		WHERE account_id = ? AND hardcore = ?
	`, accountID, hardcore).Scan(&current)
	if errors.Is(err, sql.ErrNoRows) {
		_, err := m.db.ExecContext(ctx, `
			INSERT INTO endgame_scores (account_id, name, class, wormhole_level, hero_level, hardcore, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, ?)
		`, accountID, name, class, wormholeLevel, heroLevel, hardcore, now)
		if err != nil {
			return fmt.Errorf("insert endgame score: %w", err)
		}
	} else if err != nil {
		return fmt.Errorf("query endgame score: %w", err)
	} else if wormholeLevel > current {
		_, err = m.db.ExecContext(ctx, `
			UPDATE endgame_scores
			SET name = ?, class = ?, wormhole_level = ?, hero_level = ?, updated_at = ?
			WHERE account_id = ? AND hardcore = ?
		`, name, class, wormholeLevel, heroLevel, now, accountID, hardcore)
		if err != nil {
			return fmt.Errorf("update endgame score: %w", err)
		}
	}
	return nil
}

// RankWormhole returns a 1-based rank for an account in the endgame ladder.
func (m *Manager) RankWormhole(ctx context.Context, accountID, hardcore int64) (int64, error) {
	var wormhole int64
	err := m.db.QueryRowContext(ctx, `
		SELECT wormhole_level FROM endgame_scores WHERE account_id = ? AND hardcore = ?
	`, accountID, hardcore).Scan(&wormhole)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, nil
	}
	if err != nil {
		return 0, fmt.Errorf("query account wormhole: %w", err)
	}

	var better int64
	err = m.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM endgame_scores
		WHERE hardcore = ? AND wormhole_level > ?
	`, hardcore, wormhole).Scan(&better)
	if err != nil {
		return 0, fmt.Errorf("count better wormhole scores: %w", err)
	}
	return better + 1, nil
}

// SavedRank returns a 1-based rank in the saved endgame ladder. The original
// game passes a save_id but the backend kept a single seasonal ladder.
func (m *Manager) SavedRank(ctx context.Context, accountID, hardcore int64) (int64, error) {
	rank, err := m.RankWormhole(ctx, accountID, hardcore)
	if err != nil {
		return 0, err
	}
	if rank == 0 {
		return 100, nil
	}
	return rank, nil
}
