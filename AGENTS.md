# AGENTS.md

## What this is

Go replacement backend for Hero Siege's legacy GameMaker `data.win` HTTP API.

Expected request/response formats are encoded in `internal/httpapi/handlers_*.go`.
Keep this document in sync if the repo structure changes.

## Data and tooling

Reverse engineering starts from a GameMaker `data.win` dump.

Run `scripts/dump-datawin.sh` to produce the dump. Dumped code goes into
`.dump/` (gitignored).

To repoint the game client at this backend, patch the hardcoded URL in
`data.win` with `scripts/umt_patch_url.csx` (UndertaleModTool GUI or CLI).
`scripts/patch-datawin-url.sh` wraps the CLI invocation for convenience.

Do not hardcode local paths here or in code.

## Reverse engineering

When `.dump/CodeEntries` exists, grep it for:

- `http_get(`
- `http_post_string(`
- `global.networkaddress`

Request creation is usually in `*_Create_0.gml` / `*_Alarm_0.gml`; response
parsing is usually in the matching `*_Other_62.gml`.

## Repo layout

```text
cmd/herosiege-server        entrypoint
internal/httpapi            one handler per endpoint, routing, e2e tests
internal/store              SQLite schema + migrations
internal/account            account persistence/credentials
internal/session            account_id + identifier validation
internal/save               character slots, inventory, stash, account data
internal/guild              guilds and memberships
internal/leaderboard        wormhole/siege/hero/endgame scores
internal/lobby              game-server registry and matchmaking
internal/redeem             redeem codes
```

## Build and test

```sh
go build ./...
go test ./...
make check
```

Coverage:

```sh
go test -coverpkg=./... -coverprofile=coverage.out ./internal/httpapi
go tool cover -func=coverage.out
```

## Session warm-up

`modernc.org/sqlite` is slow to compile on first use. Run `go build ./...`
once at the start of a session.

## Gotchas

- SQLite runs in WAL mode with a small connection pool. Do not run nested
  queries while iterating an open `*sql.Rows`.
- Some response checksums are verified by the game client. Their exact
  concatenation order lives in `internal/httpapi/handlers_leaderboard.go` and
  must be preserved.
