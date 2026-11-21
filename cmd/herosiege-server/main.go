package main

import (
	"flag"
	"log"
	"net/http"
	"os"
	"time"

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

func main() {
	addr := flag.String("addr", ":8080", "HTTP listen address")
	dbPath := flag.String("db", "herosiege.db", "path to the SQLite database")
	flag.Parse()

	db, err := store.Open(*dbPath)
	if err != nil {
		log.Printf("open database: %v", err)
		os.Exit(1)
	}
	defer db.Close()

	accounts := account.NewManager(db)
	sessions := session.NewManager(accounts)
	saves := save.NewManager(db)
	guilds := guild.NewManager(db, saves)
	boards := leaderboard.NewManager(db)
	lobbyMgr := lobby.NewManager(db)
	redeemMgr := redeem.NewManager(db)

	server := httpapi.New(httpapi.Deps{
		DB:       db,
		Accounts: accounts,
		Sessions: sessions,
		Saves:    saves,
		Guilds:   guilds,
		Boards:   boards,
		Lobby:    lobbyMgr,
		Redeem:   redeemMgr,
	})

	httpServer := &http.Server{
		Addr:              *addr,
		Handler:           server.Handler(),
		ReadTimeout:       10 * time.Second,
		ReadHeaderTimeout: 5 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       120 * time.Second,
		MaxHeaderBytes:    1 << 20,
	}

	log.Printf("hero siege backend listening on %s (db: %s)", *addr, *dbPath)
	if err := httpServer.ListenAndServe(); err != nil {
		log.Printf("http server: %v", err)
		os.Exit(1)
	}
}
