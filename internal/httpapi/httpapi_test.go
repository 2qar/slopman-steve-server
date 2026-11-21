package httpapi_test

import (
	"database/sql"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"strings"
	"testing"

	"herosiege/internal/account"
	"herosiege/internal/guild"
	"herosiege/internal/httpapi"
	"herosiege/internal/leaderboard"
	"herosiege/internal/lobby"
	"herosiege/internal/redeem"
	"herosiege/internal/save"
	"herosiege/internal/session"
	"herosiege/internal/store"
)

type testApp struct {
	db     *sql.DB
	server *httptest.Server
}

func newTestApp(t *testing.T) *testApp {
	t.Helper()

	db, err := store.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	accounts := account.NewManager(db)
	sessions := session.NewManager(accounts)
	saves := save.NewManager(db)
	guilds := guild.NewManager(db, saves)
	boards := leaderboard.NewManager(db)
	lobbyMgr := lobby.NewManager(db)
	redeemMgr := redeem.NewManager(db)

	api := httpapi.New(httpapi.Deps{
		DB:       db,
		Accounts: accounts,
		Sessions: sessions,
		Saves:    saves,
		Guilds:   guilds,
		Boards:   boards,
		Lobby:    lobbyMgr,
		Redeem:   redeemMgr,
	})

	server := httptest.NewServer(api.Handler())
	t.Cleanup(server.Close)

	return &testApp{db: db, server: server}
}

func (a *testApp) get(t *testing.T, path string) string {
	t.Helper()
	return a.do(t, http.MethodGet, path, nil)
}

func (a *testApp) post(t *testing.T, path string, form url.Values) string {
	t.Helper()
	return a.do(t, http.MethodPost, path, form)
}

func (a *testApp) do(t *testing.T, method, path string, form url.Values) string {
	t.Helper()

	var body io.Reader
	if form != nil {
		body = strings.NewReader(form.Encode())
	}

	req, err := http.NewRequest(method, a.server.URL+path, body)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	if form != nil {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("%s %s: %v", method, path, err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read response: %v", err)
	}
	return string(data)
}

func register(t *testing.T, a *testApp, username, password string) (accountID, identifier string) {
	t.Helper()

	body := a.post(t, "/login_check.php?reason=register&ref=game", url.Values{
		"username":  {username},
		"password1": {password},
		"email":     {username + "@example.com"},
		"steam_id":  {"non_steam"},
		"flood":     {"1"},
	})

	parts := strings.Split(body, "|")
	if len(parts) < 3 || parts[0] != "1" {
		t.Fatalf("register %q: got %q", username, body)
	}
	return parts[1], parts[2]
}

func login(t *testing.T, a *testApp, username, password string) string {
	t.Helper()
	return a.post(t, "/login_check.php?reason=login&ref=game", url.Values{
		"username": {username},
		"password": {password},
		"steam_id": {"non_steam"},
		"version":  {"1"},
		"platform": {"0"},
	})
}

func TestRegisterLogin(t *testing.T) {
	a := newTestApp(t)

	accountID, identifier := register(t, a, "hero", "password")
	if accountID == "0" || identifier == "" {
		t.Fatalf("bad register response: %q %q", accountID, identifier)
	}

	body := login(t, a, "hero", "password")
	parts := strings.Split(body, "|")
	if len(parts) < 8 {
		t.Fatalf("login response missing fields: %q", body)
	}
	if parts[0] != "1" || parts[1] != accountID || parts[2] != identifier {
		t.Fatalf("login did not return account session: %q", body)
	}

	bad := login(t, a, "hero", "wrong")
	if !strings.HasPrefix(bad, "0|0|0|") {
		t.Fatalf("bad password should fail: %q", bad)
	}
}

func TestSaveAndLoadRoundTrip(t *testing.T) {
	a := newTestApp(t)
	accountID, identifier := register(t, a, "savetest", "password")

	slotData := "10|20|30|40|Hero|1|5|100|200|3|1|1|1|1|1|1|1|1|1|0|0|0|0|0|0"
	body := a.post(t, "/save.php", url.Values{
		"account_id": {accountID},
		"identifier": {identifier},
		"slot":       {"1"},
		"slot_data":  {slotData},
		"checksum":   {"ignored"},
	})
	if body != "1" {
		t.Fatalf("save failed: %q", body)
	}

	body = a.get(t, "/load.php?account_id="+accountID+"&identifier="+identifier)
	sections := strings.Split(body, "§")
	if len(sections) != 18 {
		t.Fatalf("load returned %d sections: %q", len(sections), body)
	}
	if sections[0] != slotData {
		t.Fatalf("slot 0 = %q, want %q", sections[0], slotData)
	}

	// Inventory/stash round trip.
	a.post(t, "/save.php", url.Values{
		"account_id":     {accountID},
		"identifier":     {identifier},
		"slot":           {"1"},
		"slot_data":      {slotData},
		"inventory_data": {"inv-data"},
		"inventory_list": {"inv-list"},
		"relic_list":     {"relic-list"},
	})
	a.post(t, "/save_stash.php", url.Values{
		"account_id":    {accountID},
		"identifier":    {identifier},
		"hardcore":      {"0"},
		"account_stash": {"stash-data"},
	})

	body = a.get(t, "/load_inventory_all.php?account_id="+accountID+"&identifier="+identifier)
	sections = strings.Split(body, "§")
	if len(sections) != 55 {
		t.Fatalf("load_inventory_all returned %d sections: %q", len(sections), body)
	}
	if sections[0] != "inv-data" || sections[18] != "inv-list" || sections[36] != "relic-list" || sections[54] != "stash-data" {
		t.Fatalf("inventory round trip mismatch: %q", body)
	}

	// Per-slot inventory/stash load (load_inventory_stash.php).
	body = a.post(t, "/load_inventory_stash.php", url.Values{
		"account_id": {accountID},
		"identifier": {identifier},
		"slot":       {"1"},
		"hardcore":   {"0"},
	})
	if want := "inv-data§relic-list§inv-list§stash-data"; body != want {
		t.Fatalf("load_inventory_stash = %q, want %q", body, want)
	}
}

func TestAccountDataRoundTrip(t *testing.T) {
	a := newTestApp(t)
	accountID, identifier := register(t, a, "shop", "password")

	body := a.get(t, "/load_account_data.php?account_id="+accountID+"&identifier="+identifier)
	if body != "0" {
		t.Fatalf("initial account data = %q, want 0", body)
	}

	accountData := "0|0|1|0|250|0|0|0|xyz|1|0|250|0|0|0|0|0|0|0|0|0|0§0|0|0|0|0|0|0§0|0|0|0|0|0|0§0|0|0|0|0|0|0|0|0|0|0|0|0|0|0|0|0|0|0|0§0|0§0|0|0|0§0|6|0|0|0|0§0|0|0|0|0|0|0§0|0§0|0|0|0|0|0|0|0|0|0|0|0|0|0|0|0§0|0|0|0|0|0|0|0|0|0|0|0"
	body = a.post(t, "/save_account_data.php", url.Values{
		"account_id":   {accountID},
		"identifier":   {identifier},
		"account_data": {accountData},
		"checksum":     {"ignored"},
	})
	if body != "1" {
		t.Fatalf("save_account_data failed: %q", body)
	}

	body = a.get(t, "/load_account_data.php?account_id="+accountID+"&identifier="+identifier)
	if body != accountData {
		t.Fatalf("account data round trip mismatch: got %q", body)
	}
}

func TestGuildFlow(t *testing.T) {
	a := newTestApp(t)
	ownerID, ownerIdent := register(t, a, "owner", "password")
	memberID, memberIdent := register(t, a, "member", "password")

	body := a.post(t, "/guild_add.php", url.Values{
		"name":        {"Knights"},
		"tag":         {"KNT"},
		"description": {"A guild"},
		"color":       {"12345"},
		"account_id":  {ownerID},
		"identifier":  {ownerIdent},
		"slot":        {"1"},
	})
	guildID := strings.TrimSuffix(strings.Split(body, "|")[0], "")
	if !strings.HasPrefix(body, "1|") && !strings.HasPrefix(body, guildID+"|") {
		t.Fatalf("guild_add failed: %q", body)
	}

	body = a.post(t, "/guild_player_get.php", url.Values{
		"account_id": {ownerID},
		"identifier": {ownerIdent},
		"slot":       {"1"},
	})
	if !strings.HasPrefix(body, guildID+"|") {
		t.Fatalf("owner guild get = %q, want guild %s", body, guildID)
	}

	a.post(t, "/guild_player_add.php", url.Values{
		"account_id":        {ownerID},
		"identifier":        {ownerIdent},
		"slot":              {"1"},
		"guild_id":          {guildID},
		"status":            {"0"},
		"target_account_id": {memberID},
		"target_slot":       {"1"},
	})

	_ = memberIdent
	body = a.post(t, "/guild_get_all_players.php", url.Values{
		"guild_id":   {guildID},
		"account_id": {ownerID},
		"identifier": {ownerIdent},
		"page":       {"0"},
	})
	if !strings.HasPrefix(body, "2§") {
		t.Fatalf("guild member list = %q, want 2 members", body)
	}
}

func TestLeaderboardRoundTrip(t *testing.T) {
	a := newTestApp(t)
	accountID, identifier := register(t, a, "board", "password")

	body := a.post(t, "/wormhole_send_highscore.php", url.Values{
		"account_id":  {accountID},
		"identifier":  {identifier},
		"account_ids": {accountID},
		"slots":       {"1"},
		"players":     {"1"},
		"player_data": {"Hero|1"},
		"wormhole":    {"42"},
		"hardcore":    {"0"},
		"beta":        {"0"},
	})
	if body != "1" {
		t.Fatalf("wormhole send failed: %q", body)
	}

	body = a.post(t, "/wormhole_get_highscore.php", url.Values{
		"account_id":  {accountID},
		"identifier":  {identifier},
		"account_ids": {accountID},
		"slots":       {"1"},
		"players":     {"1"},
		"hardcore":    {"0"},
		"beta":        {"0"},
	})
	if !strings.HasPrefix(body, "42|") {
		t.Fatalf("wormhole get = %q, want 42", body)
	}

	a.post(t, "/hero_level_send_highscore.php", url.Values{
		"account_id": {accountID},
		"identifier": {identifier},
		"name":       {"Hero"},
		"class":      {"1"},
		"hero_level": {"99"},
		"hardcore":   {"0"},
		"beta":       {"0"},
	})

	body = a.post(t, "/hero_level_get_top_scores.php", url.Values{
		"account_id": {accountID},
		"identifier": {identifier},
		"hardcore":   {"0"},
		"beta":       {"0"},
	})
	if !strings.Contains(body, "Hero|1|99") {
		t.Fatalf("hero top missing entry: %q", body)
	}

	// Endgame score and rank endpoints.
	a.post(t, "/endgame_send_highscore.php", url.Values{
		"account_id":     {accountID},
		"identifier":     {identifier},
		"character_name": {"Hero"},
		"class":          {"1"},
		"wormhole_level": {"77"},
		"hero_level":     {"99"},
		"hardcore":       {"0"},
	})

	body = a.post(t, "/wormhole_get_highscore_rank.php", url.Values{
		"account_id": {accountID},
		"identifier": {identifier},
		"hardcore":   {"0"},
		"class":      {"1"},
	})
	if body != "1" {
		t.Fatalf("wormhole rank = %q, want 1", body)
	}

	body = a.post(t, "/wormhole_get_saved_rank.php", url.Values{
		"account_id": {accountID},
		"identifier": {identifier},
		"hardcore":   {"0"},
		"save_id":    {"1"},
	})
	if !strings.HasPrefix(body, "1|") {
		t.Fatalf("wormhole saved rank = %q, want rank 1", body)
	}
}

func TestServerListAndMatchmaking(t *testing.T) {
	a := newTestApp(t)
	hostID, hostIdent := register(t, a, "host", "password")
	clientID, clientIdent := register(t, a, "client", "password")

	body := a.post(t, "/server_add.php", url.Values{
		"account_id":      {hostID},
		"identifier":      {hostIdent},
		"name":            {"Host"},
		"password":        {""},
		"current_players": {"1"},
		"max_players":     {"4"},
		"siege_mode":      {"0"},
		"hardcore":        {"0"},
		"dedicated":       {"0"},
		"region":          {"0"},
		"season":          {"1"},
		"level":           {"1"},
		"item_level":      {"0"},
		"wormhole":        {"0"},
		"public":          {"1"},
	})
	if body != "1" {
		t.Fatalf("server_add failed: %q", body)
	}

	body = a.post(t, "/serverlist_get.php", url.Values{
		"siege_mode": {"0"},
		"hardcore":   {"0"},
		"region":     {"0"},
	})
	if body == "0" || !strings.Contains(body, "Host") {
		t.Fatalf("serverlist missing host: %q", body)
	}

	body = a.post(t, "/server_get_matchmaking.php", url.Values{
		"account_id": {clientID},
		"identifier": {clientIdent},
		"hardcore":   {"0"},
		"region":     {"0"},
		"season":     {"1"},
		"level":      {"1"},
		"item_level": {"0"},
		"wormhole":   {"0"},
	})
	if !strings.HasPrefix(body, "1|") {
		t.Fatalf("matchmaking failed: %q", body)
	}

	// Heartbeat and removal.
	body = a.post(t, "/server_update.php", url.Values{
		"current_players": {"2"},
		"account_id":      {hostID},
		"identifier":      {hostIdent},
	})
	if body != "1" {
		t.Fatalf("server_update failed: %q", body)
	}

	body = a.post(t, "/server_delete.php", url.Values{
		"account_id": {hostID},
		"identifier": {hostIdent},
	})
	if body != "1" {
		t.Fatalf("server_delete failed: %q", body)
	}

	body = a.post(t, "/serverlist_get.php", url.Values{
		"siege_mode": {"0"},
		"hardcore":   {"0"},
		"region":     {"0"},
	})
	if body != "0" {
		t.Fatalf("serverlist after delete = %q, want 0", body)
	}
}

func TestRedeemCode(t *testing.T) {
	a := newTestApp(t)

	if _, err := a.db.Exec(`INSERT INTO redeem_codes (code, kind, amount) VALUES ('TESTCODE', 1, 100)`); err != nil {
		t.Fatalf("insert code: %v", err)
	}

	body := a.get(t, "/code_redeem.php?code=TESTCODE")
	if body != "1/100" {
		t.Fatalf("redeem = %q, want 1/100", body)
	}

	body = a.get(t, "/code_redeem.php?code=TESTCODE")
	if body != "0" {
		t.Fatalf("second redeem = %q, want 0", body)
	}
}

func TestPlayerUpdate(t *testing.T) {
	a := newTestApp(t)
	accountID, identifier := register(t, a, "update", "password")

	body := a.get(t, "/player_update.php?account_id="+accountID+"&identifier="+identifier)
	if body != "1|OK" {
		t.Fatalf("player_update = %q", body)
	}

	// The client calls player_delete.php on logout; it should acknowledge.
	body = a.get(t, "/player_delete.php?account_id="+accountID+"&identifier="+identifier)
	if body != "1" {
		t.Fatalf("player_delete = %q", body)
	}
}

func TestMetaEndpoints(t *testing.T) {
	a := newTestApp(t)

	if body := a.get(t, "/get_season_event.php"); body != "0" {
		t.Fatalf("get_season_event = %q, want 0", body)
	}
	if body := a.get(t, "/get_server_text.php"); body != "0" {
		t.Fatalf("get_server_text = %q, want 0", body)
	}
	if body := a.get(t, "/herssi_get_all.php"); body != "0000" {
		t.Fatalf("herssi_get_all = %q, want 0000", body)
	}
	if body := a.get(t, "/players_get.php"); body != "0" {
		t.Fatalf("players_get = %q, want 0", body)
	}
}

func Example_app() {
	fmt.Println("hero siege backend")
	// Output:
	// hero siege backend
}
