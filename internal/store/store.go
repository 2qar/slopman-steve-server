package store

import (
	"database/sql"
	"fmt"

	_ "modernc.org/sqlite"
)

// migrations is executed in order on a fresh database. Each statement is
// idempotent so opening an existing database is safe.
var migrations = []string{
	`CREATE TABLE IF NOT EXISTS accounts (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		username TEXT NOT NULL UNIQUE,
		password_md5 TEXT NOT NULL,
		email TEXT NOT NULL DEFAULT '',
		steam_id TEXT NOT NULL DEFAULT '',
		identifier TEXT NOT NULL,
		admin INTEGER NOT NULL DEFAULT 0,
		banned INTEGER NOT NULL DEFAULT 0,
		created_at INTEGER NOT NULL,
		updated_at INTEGER NOT NULL,
		last_seen_at INTEGER NOT NULL DEFAULT 0
	)`,
	`CREATE TABLE IF NOT EXISTS saves (
		account_id INTEGER NOT NULL,
		slot INTEGER NOT NULL,
		slot_data TEXT NOT NULL,
		inventory_data TEXT NOT NULL DEFAULT '',
		inventory_list TEXT NOT NULL DEFAULT '',
		relic_list TEXT NOT NULL DEFAULT '',
		updated_at INTEGER NOT NULL,
		PRIMARY KEY (account_id, slot)
	)`,
	`CREATE TABLE IF NOT EXISTS stash (
		account_id INTEGER NOT NULL,
		hardcore INTEGER NOT NULL DEFAULT 0,
		data TEXT NOT NULL DEFAULT '',
		updated_at INTEGER NOT NULL,
		PRIMARY KEY (account_id, hardcore)
	)`,
	`CREATE TABLE IF NOT EXISTS account_data (
		account_id INTEGER PRIMARY KEY,
		data TEXT NOT NULL,
		updated_at INTEGER NOT NULL
	)`,
	`CREATE TABLE IF NOT EXISTS guilds (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL,
		tag TEXT NOT NULL,
		description TEXT NOT NULL DEFAULT '',
		color INTEGER NOT NULL DEFAULT 16777215,
		created_at INTEGER NOT NULL,
		updated_at INTEGER NOT NULL
	)`,
	`CREATE TABLE IF NOT EXISTS guild_members (
		guild_id INTEGER NOT NULL,
		account_id INTEGER NOT NULL,
		slot INTEGER NOT NULL,
		status INTEGER NOT NULL DEFAULT 0,
		joined_at INTEGER NOT NULL,
		PRIMARY KEY (guild_id, account_id, slot),
		UNIQUE (account_id, slot)
	)`,
	`CREATE TABLE IF NOT EXISTS game_servers (
		account_id INTEGER PRIMARY KEY,
		ip TEXT NOT NULL,
		name TEXT NOT NULL,
		password TEXT NOT NULL DEFAULT '',
		current_players INTEGER NOT NULL DEFAULT 1,
		max_players INTEGER NOT NULL DEFAULT 4,
		siege_mode INTEGER NOT NULL DEFAULT 0,
		hardcore INTEGER NOT NULL DEFAULT 0,
		dedicated INTEGER NOT NULL DEFAULT 0,
		region INTEGER NOT NULL DEFAULT 0,
		season INTEGER NOT NULL DEFAULT 1,
		level INTEGER NOT NULL DEFAULT 0,
		item_level INTEGER NOT NULL DEFAULT 0,
		wormhole INTEGER NOT NULL DEFAULT 0,
		public INTEGER NOT NULL DEFAULT 0,
		updated_at INTEGER NOT NULL,
		last_seen_at INTEGER NOT NULL
	)`,
	`CREATE TABLE IF NOT EXISTS wormhole_scores (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		account_id INTEGER NOT NULL,
		account_ids TEXT NOT NULL,
		slots TEXT NOT NULL DEFAULT '',
		players INTEGER NOT NULL,
		player_data TEXT NOT NULL,
		wormhole INTEGER NOT NULL,
		hardcore INTEGER NOT NULL DEFAULT 0,
		beta INTEGER NOT NULL DEFAULT 0,
		updated_at INTEGER NOT NULL,
		UNIQUE (account_ids, players, hardcore, beta)
	)`,
	`CREATE TABLE IF NOT EXISTS siege_scores (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		account_id INTEGER NOT NULL,
		account_ids TEXT NOT NULL,
		players INTEGER NOT NULL,
		player_data TEXT NOT NULL,
		wave INTEGER NOT NULL,
		difficulty INTEGER NOT NULL DEFAULT 0,
		hardcore INTEGER NOT NULL DEFAULT 0,
		updated_at INTEGER NOT NULL,
		UNIQUE (account_ids, players, difficulty, hardcore)
	)`,
	`CREATE TABLE IF NOT EXISTS hero_level_scores (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		account_id INTEGER NOT NULL,
		name TEXT NOT NULL,
		class INTEGER NOT NULL DEFAULT 0,
		hero_level INTEGER NOT NULL,
		hardcore INTEGER NOT NULL DEFAULT 0,
		beta INTEGER NOT NULL DEFAULT 0,
		updated_at INTEGER NOT NULL,
		UNIQUE (account_id, hardcore, beta)
	)`,
	`CREATE TABLE IF NOT EXISTS endgame_scores (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		account_id INTEGER NOT NULL,
		name TEXT NOT NULL,
		class INTEGER NOT NULL DEFAULT 0,
		wormhole_level INTEGER NOT NULL,
		hero_level INTEGER NOT NULL DEFAULT 0,
		hardcore INTEGER NOT NULL DEFAULT 0,
		updated_at INTEGER NOT NULL,
		UNIQUE (account_id, hardcore)
	)`,
	`CREATE TABLE IF NOT EXISTS redeem_codes (
		code TEXT PRIMARY KEY,
		kind INTEGER NOT NULL,
		amount INTEGER NOT NULL DEFAULT 0,
		slot INTEGER NOT NULL DEFAULT 0,
		name TEXT NOT NULL DEFAULT '',
		class INTEGER NOT NULL DEFAULT 0,
		level INTEGER NOT NULL DEFAULT 0,
		hardcore INTEGER NOT NULL DEFAULT 0,
		used_by INTEGER,
		used_at INTEGER
	)`,
	`CREATE INDEX IF NOT EXISTS idx_saves_account ON saves(account_id)`,
	`CREATE INDEX IF NOT EXISTS idx_guild_members_account ON guild_members(account_id)`,
	`CREATE INDEX IF NOT EXISTS idx_guild_members_guild ON guild_members(guild_id)`,
	`CREATE INDEX IF NOT EXISTS idx_wormhole_scores_lookup ON wormhole_scores(players, hardcore, beta, wormhole DESC)`,
	`CREATE INDEX IF NOT EXISTS idx_siege_scores_lookup ON siege_scores(players, difficulty, hardcore, wave DESC)`,
	`CREATE INDEX IF NOT EXISTS idx_hero_level_scores_lookup ON hero_level_scores(hardcore, beta, hero_level DESC)`,
}

// Open opens (and if necessary creates) a SQLite database at path.
func Open(path string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}

	// WAL mode allows concurrent readers while a single writer commits.
	if _, err := db.Exec(`PRAGMA journal_mode = WAL`); err != nil {
		db.Close()
		return nil, fmt.Errorf("enable wal: %w", err)
	}
	if _, err := db.Exec(`PRAGMA foreign_keys = ON`); err != nil {
		db.Close()
		return nil, fmt.Errorf("enable foreign keys: %w", err)
	}
	if _, err := db.Exec(`PRAGMA busy_timeout = 5000`); err != nil {
		db.Close()
		return nil, fmt.Errorf("set busy timeout: %w", err)
	}

	// A small pool keeps concurrent HTTP handlers moving while SQLite
	// serializes writes internally.
	db.SetMaxOpenConns(8)

	for _, statement := range migrations {
		if _, err := db.Exec(statement); err != nil {
			db.Close()
			return nil, fmt.Errorf("run migration: %w", err)
		}
	}

	return db, nil
}
