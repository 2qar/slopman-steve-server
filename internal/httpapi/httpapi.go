package httpapi

import (
	"context"
	"crypto/md5"
	"database/sql"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"herosiege/internal/account"
	"herosiege/internal/guild"
	"herosiege/internal/leaderboard"
	"herosiege/internal/lobby"
	"herosiege/internal/redeem"
	"herosiege/internal/save"
	"herosiege/internal/session"
)

const (
	adminSecret     = "pQ8QuWbI"
	wormholeSecret  = "AQvf4zqGxY"
	savedRankSecret = "5CQFr5Z8Nk"

	defaultSeason = "1"

	// lobbyMaxAge is how long a lobby is considered active for players_get.
	lobbyMaxAge = 10 * time.Minute
)

// Server holds the HTTP handlers and the domain managers they use.
type Server struct {
	db       *sql.DB
	accounts *account.Manager
	sessions *session.Manager
	saves    *save.Manager
	guilds   *guild.Manager
	boards   *leaderboard.Manager
	lobby    *lobby.Manager
	redeem   *redeem.Manager
}

// Deps is the set of managers an HTTP Server needs.
type Deps struct {
	DB       *sql.DB
	Accounts *account.Manager
	Sessions *session.Manager
	Saves    *save.Manager
	Guilds   *guild.Manager
	Boards   *leaderboard.Manager
	Lobby    *lobby.Manager
	Redeem   *redeem.Manager
}

func New(deps Deps) *Server {
	return &Server{
		db:       deps.DB,
		accounts: deps.Accounts,
		sessions: deps.Sessions,
		saves:    deps.Saves,
		guilds:   deps.Guilds,
		boards:   deps.Boards,
		lobby:    deps.Lobby,
		redeem:   deps.Redeem,
	}
}

// Handler returns the fully routed HTTP handler.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()

	// Auth / account.
	mux.HandleFunc("POST /login_check.php", s.handleLoginCheck)
	mux.HandleFunc("GET /login_check.php", s.handleResendEmail)

	// Saves and account data.
	mux.HandleFunc("GET /load.php", s.handleLoad)
	mux.HandleFunc("POST /save.php", s.handleSave)
	mux.HandleFunc("GET /load_inventory_all.php", s.handleLoadInventoryAll)
	mux.HandleFunc("POST /load_inventory_stash.php", s.handleLoadInventoryStash)
	mux.HandleFunc("POST /save_stash.php", s.handleSaveStash)
	mux.HandleFunc("GET /load_account_data.php", s.handleLoadAccountData)
	mux.HandleFunc("POST /save_account_data.php", s.handleSaveAccountData)
	mux.HandleFunc("GET /load_profile_inventory.php", s.handleLoadProfileInventory)

	// Player/session management.
	mux.HandleFunc("GET /player_update.php", s.handlePlayerUpdate)
	mux.HandleFunc("GET /player_delete.php", s.handlePlayerDelete)
	mux.HandleFunc("GET /players_get.php", s.handlePlayersGet)
	mux.HandleFunc("POST /daily_reward_check.php", s.handleDailyRewardCheck)
	mux.HandleFunc("POST /steam_ban_check.php", s.handleSteamBanCheck)

	// Meta/status.
	mux.HandleFunc("GET /ping.php", s.handlePing)
	mux.HandleFunc("GET /get_season_event.php", s.handleGetSeasonEvent)
	mux.HandleFunc("GET /get_server_text.php", s.handleGetServerText)
	mux.HandleFunc("GET /herssi_get_all.php", s.handleHerssiGetAll)

	// Lobby.
	mux.HandleFunc("POST /server_add.php", s.handleServerAdd)
	mux.HandleFunc("POST /server_update.php", s.handleServerUpdate)
	mux.HandleFunc("POST /server_delete.php", s.handleServerDelete)
	mux.HandleFunc("POST /serverlist_get.php", s.handleServerList)
	mux.HandleFunc("POST /server_get_matchmaking.php", s.handleServerMatchmaking)

	// Guilds.
	mux.HandleFunc("POST /account_guilds_get.php", s.handleAccountGuildsGet)
	mux.HandleFunc("POST /guild_add.php", s.handleGuildAdd)
	mux.HandleFunc("POST /guild_delete.php", s.handleGuildDelete)
	mux.HandleFunc("POST /guild_edit.php", s.handleGuildEdit)
	mux.HandleFunc("POST /guild_get_all_players.php", s.handleGuildGetAllPlayers)
	mux.HandleFunc("POST /guild_player_add.php", s.handleGuildPlayerAdd)
	mux.HandleFunc("POST /guild_player_delete.php", s.handleGuildPlayerDelete)
	mux.HandleFunc("POST /guild_player_get.php", s.handleGuildPlayerGet)

	// Leaderboards.
	mux.HandleFunc("POST /wormhole_send_highscore.php", s.handleWormholeSend)
	mux.HandleFunc("POST /wormhole_get_highscore.php", s.handleWormholeGet)
	mux.HandleFunc("POST /wormhole_get_top_scores.php", s.handleWormholeTop)
	mux.HandleFunc("POST /wormhole_get_highscore_rank.php", s.handleWormholeRank)
	mux.HandleFunc("POST /wormhole_get_saved_rank.php", s.handleWormholeSavedRank)
	mux.HandleFunc("POST /siege_mode_send_highscore.php", s.handleSiegeSend)
	mux.HandleFunc("POST /siege_mode_get_highscore.php", s.handleSiegeGet)
	mux.HandleFunc("GET /siege_mode_get_top_scores.php", s.handleSiegeTop)
	mux.HandleFunc("POST /hero_level_send_highscore.php", s.handleHeroLevelSend)
	mux.HandleFunc("POST /hero_level_get_top_scores.php", s.handleHeroLevelTop)
	mux.HandleFunc("POST /endgame_send_highscore.php", s.handleEndgameSend)

	// Reporting/moderation.
	mux.HandleFunc("POST /submit_report.php", s.handleSubmitReport)
	mux.HandleFunc("POST /ban.php", s.handleBan)
	mux.HandleFunc("POST /admin_ban.php", s.handleAdminBan)

	// Redeem codes.
	mux.HandleFunc("GET /code_redeem.php", s.handleCodeRedeem)
	mux.HandleFunc("GET /ingame/herosiege/code_redeem.php", s.handleCodeRedeem)

	return mux
}

func writeText(w http.ResponseWriter, status int, body string) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(status)
	_, _ = io.WriteString(w, body)
}

func md5hex(parts ...string) string {
	h := md5.New()
	for _, part := range parts {
		_, _ = io.WriteString(h, part)
	}
	return hex.EncodeToString(h.Sum(nil))
}

func parseInt64(s string) int64 {
	n, _ := strconv.ParseInt(s, 10, 64)
	return n
}

func parseForm(r *http.Request) error {
	return r.ParseForm()
}

func formValue(r *http.Request, key string) string {
	if err := parseForm(r); err != nil {
		return ""
	}
	return r.FormValue(key)
}

func remoteIP(r *http.Request) string {
	host := r.RemoteAddr
	if idx := strings.LastIndex(host, ":"); idx != -1 {
		host = host[:idx]
	}
	return host
}

// requireAccount validates the game's account_id/identifier session pair.
func (s *Server) requireAccount(ctx context.Context, r *http.Request) (*account.Account, error) {
	return s.sessions.Validate(ctx, formValue(r, "account_id"), formValue(r, "identifier"))
}

func accountIDString(acc *account.Account) string {
	return strconv.FormatInt(acc.ID, 10)
}

func loginSuccess(acc *account.Account) string {
	adminStatus := 0
	if acc.Admin {
		adminStatus = 1
	}
	adminCheckSum := md5hex(accountIDString(acc), acc.Identifier, adminSecret, strconv.Itoa(adminStatus))
	return fmt.Sprintf("1|%s|%s|Login successful|1|%d|%s|%s",
		accountIDString(acc), acc.Identifier, adminStatus, adminCheckSum, defaultSeason)
}
