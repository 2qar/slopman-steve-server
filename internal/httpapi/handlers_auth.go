package httpapi

import (
	"errors"
	"net/http"

	"herosiege/internal/account"
)

func (s *Server) handleLoginCheck(w http.ResponseWriter, r *http.Request) {
	reason := formValue(r, "reason")
	switch reason {
	case "register":
		s.handleRegister(w, r)
	case "login":
		s.handleLogin(w, r)
	default:
		writeText(w, http.StatusOK, loginError("Unknown reason"))
	}
}

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	username := formValue(r, "username")
	passwordMD5 := formValue(r, "password")

	acc, err := s.sessions.Login(r.Context(), username, passwordMD5)
	if err != nil {
		switch {
		case errors.Is(err, account.ErrBanned):
			writeText(w, http.StatusOK, loginError("Account banned"))
		case errors.Is(err, account.ErrInvalidCredential):
			writeText(w, http.StatusOK, loginError("Wrong username or password"))
		default:
			writeText(w, http.StatusOK, loginError("Server error"))
		}
		return
	}

	writeText(w, http.StatusOK, loginSuccess(acc))
}

func (s *Server) handleRegister(w http.ResponseWriter, r *http.Request) {
	username := formValue(r, "username")
	passwordMD5 := formValue(r, "password1")
	email := formValue(r, "email")
	steamID := formValue(r, "steam_id")

	acc, err := s.accounts.Register(r.Context(), username, passwordMD5, email, steamID)
	if err != nil {
		switch {
		case errors.Is(err, account.ErrUsernameTaken):
			writeText(w, http.StatusOK, registerError("Username already exists"))
		case errors.Is(err, account.ErrInvalidCredential):
			writeText(w, http.StatusOK, registerError("Invalid username or password"))
		default:
			writeText(w, http.StatusOK, registerError("Registration failed"))
		}
		return
	}

	writeText(w, http.StatusOK, "1|"+accountIDString(acc)+"|"+acc.Identifier+"|Account created")
}

func (s *Server) handleResendEmail(w http.ResponseWriter, r *http.Request) {
	// The game fires this request and discards the body. Validate the session
	// when it is present so the endpoint is still exercised in tests.
	if accountID := formValue(r, "account_id"); accountID != "" {
		_, _ = s.sessions.Validate(r.Context(), accountID, formValue(r, "identifier"))
	}
	writeText(w, http.StatusOK, "1|0|0|Email sent|1|0|0|1")
}

func loginError(message string) string {
	return "0|0|0|" + message + "|1|0|0|1"
}

func registerError(message string) string {
	return "0|0|0|" + message
}
