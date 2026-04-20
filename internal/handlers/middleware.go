package handlers

import (
	"net/http"
	"time"

	"example.com/gourmetkan/internal/auth"
)

const (
	sessionCookieName = "session_id"
	baseCookieName    = "base_id"
)

type SessionInfo struct {
	UserID    int
	CSRFToken string
	ExpiresAt time.Time
}

func (h *Handler) getSession(r *http.Request) (*SessionInfo, error) {
	cookie, err := r.Cookie(sessionCookieName)
	if err != nil {
		return nil, nil
	}
	userID, csrfToken, expiresAt, err := auth.GetSession(h.db, cookie.Value)
	if err != nil {
		return nil, err
	}
	if userID == 0 {
		return nil, nil
	}
	if time.Now().After(expiresAt) {
		_ = auth.DeleteSession(h.db, cookie.Value)
		return nil, nil
	}
	return &SessionInfo{UserID: userID, CSRFToken: csrfToken, ExpiresAt: expiresAt}, nil
}

func (h *Handler) requireLogin(w http.ResponseWriter, r *http.Request) (*SessionInfo, bool) {
	session, err := h.getSession(r)
	if err != nil || session == nil {
		http.Redirect(w, r, "/auth/github/login", http.StatusFound)
		return nil, false
	}
	return session, true
}

func (h *Handler) verifyCSRF(r *http.Request, session *SessionInfo) bool {
	if session == nil {
		return false
	}
	if err := r.ParseForm(); err != nil {
		return false
	}
	token := r.FormValue("csrf_token")
	return token != "" && token == session.CSRFToken
}

func (h *Handler) setSessionCookie(w http.ResponseWriter, sessionID string, expiresAt time.Time) {
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    sessionID,
		Path:     "/",
		HttpOnly: true,
		Secure:   h.cfg.CookieSecure,
		SameSite: http.SameSiteLaxMode,
		Expires:  expiresAt,
	})
}

func (h *Handler) clearSessionCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   h.cfg.CookieSecure,
		SameSite: http.SameSiteLaxMode,
		Expires:  time.Unix(0, 0),
		MaxAge:   -1,
	})
}
