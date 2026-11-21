package httpapi

import "net/http"

func (s *Server) handlePing(w http.ResponseWriter, r *http.Request) {
	writeText(w, http.StatusOK, "1")
}

func (s *Server) handleGetSeasonEvent(w http.ResponseWriter, r *http.Request) {
	writeText(w, http.StatusOK, "0")
}

func (s *Server) handleGetServerText(w http.ResponseWriter, r *http.Request) {
	// "0" makes the client show its localized server-down message.
	writeText(w, http.StatusOK, "0")
}

func (s *Server) handleHerssiGetAll(w http.ResponseWriter, r *http.Request) {
	// Four region status characters; '0' means offline.
	writeText(w, http.StatusOK, "0000")
}

func (s *Server) handleSteamBanCheck(w http.ResponseWriter, r *http.Request) {
	// The game ignores the body; return a non-negative value.
	writeText(w, http.StatusOK, "1")
}
