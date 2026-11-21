package httpapi

import (
	"fmt"
	"net/http"
	"strings"

	"herosiege/internal/lobby"
)

func (s *Server) handleServerAdd(w http.ResponseWriter, r *http.Request) {
	acc, err := s.requireAccount(r.Context(), r)
	if err != nil {
		writeText(w, http.StatusOK, "-1")
		return
	}

	server := lobby.Server{
		AccountID:      acc.ID,
		Name:           formValue(r, "name"),
		Password:       formValue(r, "password"),
		CurrentPlayers: parseInt64(formValue(r, "current_players")),
		MaxPlayers:     parseInt64(formValue(r, "max_players")),
		SiegeMode:      parseInt64(formValue(r, "siege_mode")),
		Hardcore:       parseInt64(formValue(r, "hardcore")),
		Dedicated:      parseInt64(formValue(r, "dedicated")),
		Region:         parseInt64(formValue(r, "region")),
		Season:         parseInt64(formValue(r, "season")),
		Level:          parseInt64(formValue(r, "level")),
		ItemLevel:      parseInt64(formValue(r, "item_level")),
		Wormhole:       parseInt64(formValue(r, "wormhole")),
		Public:         parseInt64(formValue(r, "public")),
	}

	if err := s.lobby.Upsert(r.Context(), server, remoteIP(r)); err != nil {
		writeText(w, http.StatusOK, "-1")
		return
	}
	writeText(w, http.StatusOK, "1")
}

func (s *Server) handleServerUpdate(w http.ResponseWriter, r *http.Request) {
	acc, err := s.requireAccount(r.Context(), r)
	if err != nil {
		writeText(w, http.StatusOK, "-1")
		return
	}

	if err := s.lobby.Touch(r.Context(), acc.ID, parseInt64(formValue(r, "current_players"))); err != nil {
		writeText(w, http.StatusOK, "-1")
		return
	}
	writeText(w, http.StatusOK, "1")
}

func (s *Server) handleServerDelete(w http.ResponseWriter, r *http.Request) {
	acc, err := s.requireAccount(r.Context(), r)
	if err != nil {
		writeText(w, http.StatusOK, "-1")
		return
	}

	if err := s.lobby.Delete(r.Context(), acc.ID); err != nil {
		writeText(w, http.StatusOK, "-1")
		return
	}
	writeText(w, http.StatusOK, "1")
}

func (s *Server) handleServerList(w http.ResponseWriter, r *http.Request) {
	siegeMode := parseInt64(formValue(r, "siege_mode"))
	hardcore := parseInt64(formValue(r, "hardcore"))
	region := parseInt64(formValue(r, "region"))

	servers, err := s.lobby.List(r.Context(), siegeMode, hardcore, region)
	if err != nil {
		writeText(w, http.StatusOK, "-1")
		return
	}
	if len(servers) == 0 {
		writeText(w, http.StatusOK, "0")
		return
	}

	fields := []string{fmt.Sprintf("%d", len(servers))}
	for _, srv := range servers {
		fields = append(fields, fmt.Sprintf("%s|%s|%s|%d|%d|%d|%d|%d",
			srv.IP,
			srv.Name,
			srv.Password,
			srv.CurrentPlayers,
			srv.MaxPlayers,
			srv.Dedicated,
			srv.AccountID,
			srv.Season,
		))
	}
	writeText(w, http.StatusOK, strings.Join(fields, "§"))
}

func (s *Server) handleServerMatchmaking(w http.ResponseWriter, r *http.Request) {
	acc, err := s.requireAccount(r.Context(), r)
	if err != nil {
		writeText(w, http.StatusOK, "-1")
		return
	}

	srv, err := s.lobby.Matchmaking(
		r.Context(),
		acc.ID,
		parseInt64(formValue(r, "hardcore")),
		parseInt64(formValue(r, "region")),
		parseInt64(formValue(r, "season")),
	)
	if err != nil {
		writeText(w, http.StatusOK, "-1")
		return
	}
	if srv == nil {
		writeText(w, http.StatusOK, "0")
		return
	}

	writeText(w, http.StatusOK, fmt.Sprintf("1|%s|%d", srv.IP, srv.AccountID))
}
