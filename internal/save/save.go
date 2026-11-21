package save

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"
)

// SlotCount is the number of character slots the game client displays.
const SlotCount = 18

type Slot struct {
	Slot          int
	SlotData      string
	InventoryData string
	InventoryList string
	RelicList     string
}

// Manager owns character saves, stashes and account data.
type Manager struct {
	db *sql.DB
}

func NewManager(db *sql.DB) *Manager {
	return &Manager{db: db}
}

// SaveSlot upserts a single character save.
func (m *Manager) SaveSlot(ctx context.Context, accountID int64, slot int, slotData, inventoryData, inventoryList, relicList string) error {
	if slot < 1 || slot > SlotCount {
		return fmt.Errorf("slot out of range: %d", slot)
	}
	_, err := m.db.ExecContext(ctx, `
		INSERT INTO saves (account_id, slot, slot_data, inventory_data, inventory_list, relic_list, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(account_id, slot) DO UPDATE SET
			slot_data = excluded.slot_data,
			inventory_data = excluded.inventory_data,
			inventory_list = excluded.inventory_list,
			relic_list = excluded.relic_list,
			updated_at = excluded.updated_at
	`, accountID, slot, slotData, inventoryData, inventoryList, relicList, time.Now().Unix())
	if err != nil {
		return fmt.Errorf("save slot: %w", err)
	}
	return nil
}

// DeleteSlot removes a character save.
func (m *Manager) DeleteSlot(ctx context.Context, accountID int64, slot int) error {
	_, err := m.db.ExecContext(ctx, `DELETE FROM saves WHERE account_id = ? AND slot = ?`, accountID, slot)
	if err != nil {
		return fmt.Errorf("delete slot: %w", err)
	}
	return nil
}

// LoadSlots returns all 18 slots. Missing slots have SlotData "0".
func (m *Manager) LoadSlots(ctx context.Context, accountID int64) ([]Slot, error) {
	slots := make([]Slot, SlotCount)
	for i := range slots {
		slots[i].Slot = i + 1
		slots[i].SlotData = "0"
	}

	rows, err := m.db.QueryContext(ctx, `
		SELECT slot, slot_data, inventory_data, inventory_list, relic_list
		FROM saves WHERE account_id = ? ORDER BY slot`, accountID)
	if err != nil {
		return nil, fmt.Errorf("query slots: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var s Slot
		if err := rows.Scan(&s.Slot, &s.SlotData, &s.InventoryData, &s.InventoryList, &s.RelicList); err != nil {
			return nil, fmt.Errorf("scan slot: %w", err)
		}
		if s.Slot >= 1 && s.Slot <= SlotCount {
			slots[s.Slot-1] = s
		}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate slots: %w", err)
	}
	return slots, nil
}

// LoadSlot returns one slot. Missing slots are reported with SlotData "0".
func (m *Manager) LoadSlot(ctx context.Context, accountID int64, slot int) (*Slot, error) {
	if slot < 1 || slot > SlotCount {
		return nil, fmt.Errorf("slot out of range: %d", slot)
	}
	row := m.db.QueryRowContext(ctx, `
		SELECT slot, slot_data, inventory_data, inventory_list, relic_list
		FROM saves WHERE account_id = ? AND slot = ?`, accountID, slot)

	var s Slot
	err := row.Scan(&s.Slot, &s.SlotData, &s.InventoryData, &s.InventoryList, &s.RelicList)
	if errors.Is(err, sql.ErrNoRows) {
		return &Slot{Slot: slot, SlotData: "0"}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("query slot: %w", err)
	}
	return &s, nil
}

// SaveStash stores the shared stash for a specific hardcore flag.
func (m *Manager) SaveStash(ctx context.Context, accountID int64, hardcore int, data string) error {
	_, err := m.db.ExecContext(ctx, `
		INSERT INTO stash (account_id, hardcore, data, updated_at)
		VALUES (?, ?, ?, ?)
		ON CONFLICT(account_id, hardcore) DO UPDATE SET
			data = excluded.data,
			updated_at = excluded.updated_at
	`, accountID, hardcore, data, time.Now().Unix())
	if err != nil {
		return fmt.Errorf("save stash: %w", err)
	}
	return nil
}

// LoadStash returns the stash for a hardcore flag, or "".
func (m *Manager) LoadStash(ctx context.Context, accountID int64, hardcore int) (string, error) {
	var data string
	err := m.db.QueryRowContext(ctx, `SELECT data FROM stash WHERE account_id = ? AND hardcore = ?`, accountID, hardcore).Scan(&data)
	if errors.Is(err, sql.ErrNoRows) {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("query stash: %w", err)
	}
	return data, nil
}

// SaveAccountData stores the opaque shop/account data blob.
func (m *Manager) SaveAccountData(ctx context.Context, accountID int64, data string) error {
	_, err := m.db.ExecContext(ctx, `
		INSERT INTO account_data (account_id, data, updated_at)
		VALUES (?, ?, ?)
		ON CONFLICT(account_id) DO UPDATE SET
			data = excluded.data,
			updated_at = excluded.updated_at
	`, accountID, data, time.Now().Unix())
	if err != nil {
		return fmt.Errorf("save account data: %w", err)
	}
	return nil
}

// LoadAccountData returns the shop/account data blob, or "0".
func (m *Manager) LoadAccountData(ctx context.Context, accountID int64) (string, error) {
	var data string
	err := m.db.QueryRowContext(ctx, `SELECT data FROM account_data WHERE account_id = ?`, accountID).Scan(&data)
	if errors.Is(err, sql.ErrNoRows) {
		return "0", nil
	}
	if err != nil {
		return "", fmt.Errorf("query account data: %w", err)
	}
	return data, nil
}

// LoadInventoryAll returns the 55-field inventory response used by
// load_inventory_all.php: 18 inventory blobs, 18 inventory lists, 18 relic
// lists and the softcore stash.
func (m *Manager) LoadInventoryAll(ctx context.Context, accountID int64) (string, error) {
	slots, err := m.LoadSlots(ctx, accountID)
	if err != nil {
		return "", err
	}
	stash, err := m.LoadStash(ctx, accountID, 0)
	if err != nil {
		return "", err
	}

	fields := make([]string, 0, 55)
	for _, s := range slots {
		fields = append(fields, s.InventoryData)
	}
	for _, s := range slots {
		fields = append(fields, s.InventoryList)
	}
	for _, s := range slots {
		fields = append(fields, s.RelicList)
	}
	fields = append(fields, stash)
	return strings.Join(fields, "§"), nil
}
