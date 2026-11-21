package httpapi

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"herosiege/internal/account"
)

const leaderboardLimit = 100

func (s *Server) handleWormholeSend(w http.ResponseWriter, r *http.Request) {
	acc, err := s.requireAccount(r.Context(), r)
	if err != nil {
		writeText(w, http.StatusOK, "-1")
		return
	}

	err = s.boards.SubmitWormhole(
		r.Context(),
		acc.ID,
		formValue(r, "account_ids"),
		formValue(r, "slots"),
		formValue(r, "player_data"),
		parseInt64(formValue(r, "players")),
		parseInt64(formValue(r, "wormhole")),
		parseInt64(formValue(r, "hardcore")),
		parseInt64(formValue(r, "beta")),
	)
	if err != nil {
		writeText(w, http.StatusOK, "-1")
		return
	}
	writeText(w, http.StatusOK, "1")
}

func (s *Server) handleWormholeGet(w http.ResponseWriter, r *http.Request) {
	acc, err := s.requireAccount(r.Context(), r)
	if err != nil {
		writeText(w, http.StatusOK, "-1")
		return
	}

	accountIDs := formValue(r, "account_ids")
	slots := formValue(r, "slots")
	players := parseInt64(formValue(r, "players"))
	hardcore := parseInt64(formValue(r, "hardcore"))
	beta := parseInt64(formValue(r, "beta"))

	level, err := s.boards.BestWormhole(r.Context(), accountIDs, slots, players, hardcore, beta)
	if err != nil {
		writeText(w, http.StatusOK, "-1")
		return
	}

	checksum := wormholeLevelChecksum(acc, accountIDs, slots, players, hardcore, beta, level)
	writeText(w, http.StatusOK, fmt.Sprintf("%d|%s", level, checksum))
}

func (s *Server) handleWormholeTop(w http.ResponseWriter, r *http.Request) {
	_, err := s.requireAccount(r.Context(), r)
	if err != nil {
		writeText(w, http.StatusOK, "-1")
		return
	}

	players := parseInt64(formValue(r, "players"))
	hardcore := parseInt64(formValue(r, "hardcore"))
	beta := parseInt64(formValue(r, "beta"))

	entries, err := s.boards.TopWormhole(r.Context(), players, hardcore, beta, leaderboardLimit)
	if err != nil {
		writeText(w, http.StatusOK, "-1")
		return
	}

	fields := []string{strconv.Itoa(len(entries))}
	for _, entry := range entries {
		fields = append(fields, wormholeEntry(entry.PlayerData, entry.Wormhole))
	}
	writeText(w, http.StatusOK, strings.Join(fields, "£"))
}

func (s *Server) handleWormholeRank(w http.ResponseWriter, r *http.Request) {
	acc, err := s.requireAccount(r.Context(), r)
	if err != nil {
		writeText(w, http.StatusOK, "0")
		return
	}

	rank, err := s.boards.RankWormhole(r.Context(), acc.ID, parseInt64(formValue(r, "hardcore")))
	if err != nil {
		writeText(w, http.StatusOK, "0")
		return
	}
	writeText(w, http.StatusOK, strconv.FormatInt(rank, 10))
}

func (s *Server) handleWormholeSavedRank(w http.ResponseWriter, r *http.Request) {
	acc, err := s.requireAccount(r.Context(), r)
	if err != nil {
		writeText(w, http.StatusOK, "0")
		return
	}

	rank, err := s.boards.SavedRank(r.Context(), acc.ID, parseInt64(formValue(r, "hardcore")))
	if err != nil {
		writeText(w, http.StatusOK, "0")
		return
	}

	checksum := md5hex(accountIDString(acc), acc.Identifier, strconv.FormatInt(rank, 10), formValue(r, "save_id"), "0", savedRankSecret)
	writeText(w, http.StatusOK, fmt.Sprintf("%d|%s", rank, checksum))
}

func (s *Server) handleSiegeSend(w http.ResponseWriter, r *http.Request) {
	acc, err := s.requireAccount(r.Context(), r)
	if err != nil {
		writeText(w, http.StatusOK, "-1")
		return
	}

	err = s.boards.SubmitSiege(
		r.Context(),
		acc.ID,
		formValue(r, "account_ids"),
		formValue(r, "player_data"),
		parseInt64(formValue(r, "players")),
		parseInt64(formValue(r, "wave")),
		parseInt64(formValue(r, "difficulty")),
		parseInt64(formValue(r, "hardcore")),
	)
	if err != nil {
		writeText(w, http.StatusOK, "-1")
		return
	}
	writeText(w, http.StatusOK, "1")
}

func (s *Server) handleSiegeGet(w http.ResponseWriter, r *http.Request) {
	_, err := s.requireAccount(r.Context(), r)
	if err != nil {
		writeText(w, http.StatusOK, "-1")
		return
	}

	wave, err := s.boards.BestSiege(
		r.Context(),
		formValue(r, "account_ids"),
		parseInt64(formValue(r, "players")),
		parseInt64(formValue(r, "difficulty")),
		parseInt64(formValue(r, "hardcore")),
	)
	if err != nil {
		writeText(w, http.StatusOK, "-1")
		return
	}

	writeText(w, http.StatusOK, fmt.Sprintf("%d|%d", wave, wave))
}

func (s *Server) handleSiegeTop(w http.ResponseWriter, r *http.Request) {
	_, err := s.requireAccount(r.Context(), r)
	if err != nil {
		writeText(w, http.StatusOK, "-1")
		return
	}

	players := parseInt64(formValue(r, "players"))
	difficulty := parseInt64(formValue(r, "difficulty"))
	hardcore := parseInt64(formValue(r, "hardcore"))

	entries, err := s.boards.TopSiege(r.Context(), players, difficulty, hardcore, leaderboardLimit)
	if err != nil {
		writeText(w, http.StatusOK, "-1")
		return
	}

	fields := []string{strconv.Itoa(len(entries))}
	for _, entry := range entries {
		fields = append(fields, fmt.Sprintf("%s€%d", normalizePlayerInfo(entry.PlayerData), entry.Wave))
	}
	writeText(w, http.StatusOK, strings.Join(fields, "£"))
}

func (s *Server) handleHeroLevelSend(w http.ResponseWriter, r *http.Request) {
	acc, err := s.requireAccount(r.Context(), r)
	if err != nil {
		writeText(w, http.StatusOK, "-1")
		return
	}

	err = s.boards.SubmitHeroLevel(
		r.Context(),
		acc.ID,
		formValue(r, "name"),
		parseInt64(formValue(r, "class")),
		parseInt64(formValue(r, "hero_level")),
		parseInt64(formValue(r, "hardcore")),
		parseInt64(formValue(r, "beta")),
	)
	if err != nil {
		writeText(w, http.StatusOK, "-1")
		return
	}
	writeText(w, http.StatusOK, "1")
}

func (s *Server) handleHeroLevelTop(w http.ResponseWriter, r *http.Request) {
	_, err := s.requireAccount(r.Context(), r)
	if err != nil {
		writeText(w, http.StatusOK, "-1")
		return
	}

	entries, err := s.boards.TopHeroLevel(r.Context(), parseInt64(formValue(r, "hardcore")), parseInt64(formValue(r, "beta")), leaderboardLimit)
	if err != nil {
		writeText(w, http.StatusOK, "-1")
		return
	}

	fields := []string{strconv.Itoa(len(entries))}
	for _, entry := range entries {
		fields = append(fields, fmt.Sprintf("%s|%d|%d", entry.Name, entry.Class, entry.HeroLevel))
	}
	writeText(w, http.StatusOK, strings.Join(fields, "§"))
}

func (s *Server) handleEndgameSend(w http.ResponseWriter, r *http.Request) {
	acc, err := s.requireAccount(r.Context(), r)
	if err != nil {
		writeText(w, http.StatusOK, "-1")
		return
	}

	name := formValue(r, "character_name")
	class := parseInt64(formValue(r, "class"))
	wormhole := parseInt64(formValue(r, "wormhole_level"))
	heroLevel := parseInt64(formValue(r, "hero_level"))
	hardcore := parseInt64(formValue(r, "hardcore"))

	if err := s.boards.SubmitEndgame(r.Context(), acc.ID, name, class, wormhole, heroLevel, hardcore); err != nil {
		writeText(w, http.StatusOK, "-1")
		return
	}

	// Keep the individual wormhole ladder fed as well so wormhole_get_top_scores
	// returns endgame submissions.
	if err := s.boards.SubmitWormhole(
		r.Context(),
		acc.ID,
		accountIDString(acc),
		"",
		name+"|"+strconv.FormatInt(class, 10),
		1,
		wormhole,
		hardcore,
		0,
	); err != nil {
		writeText(w, http.StatusOK, "-1")
		return
	}

	writeText(w, http.StatusOK, "1")
}

func wormholeLevelChecksum(acc *account.Account, accountIDs, slots string, players, hardcore, beta, level int64) string {
	return md5hex(
		accountIDString(acc),
		acc.Identifier,
		accountIDs,
		slots,
		strconv.FormatInt(players, 10),
		strconv.FormatInt(beta, 10),
		strconv.FormatInt(hardcore, 10),
		strconv.FormatInt(level, 10),
		wormholeSecret,
	)
}

// wormholeEntry renders one wormhole leaderboard entry.
func wormholeEntry(playerData string, wormhole int64) string {
	return fmt.Sprintf("%s€%s€%d", normalizePlayerInfo(playerData), normalizeGuildInfo(), wormhole)
}

func normalizePlayerInfo(playerData string) string {
	sections := strings.Split(playerData, "§")
	out := make([]string, 4)
	for i := 0; i < 4; i++ {
		if i < len(sections) {
			out[i] = sections[i]
		}
	}
	return strings.Join(out, "§")
}

func normalizeGuildInfo() string {
	return "|0§|0§|0§|0"
}
