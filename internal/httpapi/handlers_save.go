package httpapi

import (
	"net/http"
	"strconv"
	"strings"
)

func (s *Server) handleLoad(w http.ResponseWriter, r *http.Request) {
	acc, err := s.requireAccount(r.Context(), r)
	if err != nil {
		writeText(w, http.StatusOK, "")
		return
	}

	slots, err := s.saves.LoadSlots(r.Context(), acc.ID)
	if err != nil {
		writeText(w, http.StatusOK, "")
		return
	}

	fields := make([]string, len(slots))
	for i, slot := range slots {
		fields[i] = slot.SlotData
	}
	writeText(w, http.StatusOK, strings.Join(fields, "§"))
}

func (s *Server) handleSave(w http.ResponseWriter, r *http.Request) {
	acc, err := s.requireAccount(r.Context(), r)
	if err != nil {
		writeText(w, http.StatusOK, "-1")
		return
	}

	slot := int(parseInt64(formValue(r, "slot")))
	slotData := formValue(r, "slot_data")
	inventoryData := formValue(r, "inventory_data")
	inventoryList := formValue(r, "inventory_list")
	relicList := formValue(r, "relic_list")

	// The game sends slot_data=-1 to delete a slot.
	if slotData == "-1" || inventoryData == "-1" {
		if err := s.saves.DeleteSlot(r.Context(), acc.ID, slot); err != nil {
			writeText(w, http.StatusOK, "-1")
			return
		}
		writeText(w, http.StatusOK, "1")
		return
	}

	if err := s.saves.SaveSlot(r.Context(), acc.ID, slot, slotData, inventoryData, inventoryList, relicList); err != nil {
		writeText(w, http.StatusOK, "-1")
		return
	}
	writeText(w, http.StatusOK, "1")
}

func (s *Server) handleLoadInventoryAll(w http.ResponseWriter, r *http.Request) {
	acc, err := s.requireAccount(r.Context(), r)
	if err != nil {
		writeText(w, http.StatusOK, "")
		return
	}

	body, err := s.saves.LoadInventoryAll(r.Context(), acc.ID)
	if err != nil {
		writeText(w, http.StatusOK, "")
		return
	}
	writeText(w, http.StatusOK, body)
}

func (s *Server) handleLoadInventoryStash(w http.ResponseWriter, r *http.Request) {
	acc, err := s.requireAccount(r.Context(), r)
	if err != nil {
		writeText(w, http.StatusOK, "")
		return
	}

	slot := int(parseInt64(formValue(r, "slot")))
	hardcore := int(parseInt64(formValue(r, "hardcore")))

	slotData, err := s.saves.LoadSlot(r.Context(), acc.ID, slot)
	if err != nil {
		writeText(w, http.StatusOK, "")
		return
	}
	stash, err := s.saves.LoadStash(r.Context(), acc.ID, hardcore)
	if err != nil {
		writeText(w, http.StatusOK, "")
		return
	}

	fields := []string{slotData.InventoryData, slotData.RelicList, slotData.InventoryList, stash}
	writeText(w, http.StatusOK, strings.Join(fields, "§"))
}

func (s *Server) handleSaveStash(w http.ResponseWriter, r *http.Request) {
	acc, err := s.requireAccount(r.Context(), r)
	if err != nil {
		writeText(w, http.StatusOK, "-1")
		return
	}

	hardcore := int(parseInt64(formValue(r, "hardcore")))
	stash := formValue(r, "account_stash")

	if err := s.saves.SaveStash(r.Context(), acc.ID, hardcore, stash); err != nil {
		writeText(w, http.StatusOK, "-1")
		return
	}
	writeText(w, http.StatusOK, "1")
}

func (s *Server) handleLoadAccountData(w http.ResponseWriter, r *http.Request) {
	acc, err := s.requireAccount(r.Context(), r)
	if err != nil {
		writeText(w, http.StatusOK, "0")
		return
	}

	data, err := s.saves.LoadAccountData(r.Context(), acc.ID)
	if err != nil {
		writeText(w, http.StatusOK, "0")
		return
	}
	writeText(w, http.StatusOK, data)
}

func (s *Server) handleSaveAccountData(w http.ResponseWriter, r *http.Request) {
	acc, err := s.requireAccount(r.Context(), r)
	if err != nil {
		writeText(w, http.StatusOK, "-1")
		return
	}

	if err := s.saves.SaveAccountData(r.Context(), acc.ID, formValue(r, "account_data")); err != nil {
		writeText(w, http.StatusOK, "-1")
		return
	}
	writeText(w, http.StatusOK, "1")
}

func (s *Server) handleLoadProfileInventory(w http.ResponseWriter, r *http.Request) {
	accountID := parseInt64(formValue(r, "account_id"))
	slot := int(parseInt64(formValue(r, "slot")))

	slotData, err := s.saves.LoadSlot(r.Context(), accountID, slot)
	if err != nil || slotData.SlotData == "0" {
		writeText(w, http.StatusOK, "0")
		return
	}
	if slotData.InventoryData == "" {
		writeText(w, http.StatusOK, "0")
		return
	}
	writeText(w, http.StatusOK, slotData.InventoryData)
}

func (s *Server) handlePlayerUpdate(w http.ResponseWriter, r *http.Request) {
	acc, err := s.requireAccount(r.Context(), r)
	if err != nil {
		writeText(w, http.StatusOK, "-1|Invalid session")
		return
	}
	if acc.Banned {
		writeText(w, http.StatusOK, "2|Account banned")
		return
	}
	writeText(w, http.StatusOK, "1|OK")
}

func (s *Server) handlePlayerDelete(w http.ResponseWriter, r *http.Request) {
	// Despite the endpoint name, the client calls this on logout and only
	// clears its local session. Validate the session and acknowledge it.
	_, err := s.requireAccount(r.Context(), r)
	if err != nil {
		writeText(w, http.StatusOK, "0")
		return
	}
	writeText(w, http.StatusOK, "1")
}

func (s *Server) handleDailyRewardCheck(w http.ResponseWriter, r *http.Request) {
	_, _ = s.requireAccount(r.Context(), r)
	writeText(w, http.StatusOK, "1")
}

func (s *Server) handlePlayersGet(w http.ResponseWriter, r *http.Request) {
	// Players online is derived from active lobbies.
	count, err := s.lobby.OnlinePlayers(r.Context(), lobbyMaxAge)
	if err != nil {
		writeText(w, http.StatusOK, "0")
		return
	}
	writeText(w, http.StatusOK, strconv.FormatInt(count, 10))
}
