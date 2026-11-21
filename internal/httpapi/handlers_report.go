package httpapi

import "net/http"

func (s *Server) handleSubmitReport(w http.ResponseWriter, r *http.Request) {
	_, _ = s.requireAccount(r.Context(), r)
	writeText(w, http.StatusOK, "1")
}

func (s *Server) handleBan(w http.ResponseWriter, r *http.Request) {
	// The original backend stored cheating reports. The game never parses the
	// response body, so acknowledging the report is sufficient.
	_, _ = s.requireAccount(r.Context(), r)
	writeText(w, http.StatusOK, "1")
}

func (s *Server) handleAdminBan(w http.ResponseWriter, r *http.Request) {
	_, _ = s.requireAccount(r.Context(), r)
	writeText(w, http.StatusOK, "1")
}
