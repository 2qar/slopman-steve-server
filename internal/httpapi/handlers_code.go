package httpapi

import (
	"errors"
	"fmt"
	"net/http"

	"herosiege/internal/redeem"
)

func (s *Server) handleCodeRedeem(w http.ResponseWriter, r *http.Request) {
	code := formValue(r, "code")
	if code == "" {
		writeText(w, http.StatusOK, "0")
		return
	}

	accountID := parseInt64(formValue(r, "account_id"))
	red, err := s.redeem.Redeem(r.Context(), code, accountID)
	if err != nil {
		if errors.Is(err, redeem.ErrCodeNotFound) || errors.Is(err, redeem.ErrCodeUsed) {
			writeText(w, http.StatusOK, "0")
			return
		}
		writeText(w, http.StatusOK, "0")
		return
	}

	switch red.Kind {
	case 1:
		writeText(w, http.StatusOK, fmt.Sprintf("1/%d", red.Amount))
	case 2:
		writeText(w, http.StatusOK, fmt.Sprintf("2/%d/%s/%d/%d/%d", red.Slot, red.Name, red.Class, red.Level, red.Hardcore))
	default:
		writeText(w, http.StatusOK, "0")
	}
}
