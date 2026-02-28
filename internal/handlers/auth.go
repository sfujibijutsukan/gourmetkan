package handlers

import (
	"net/http"
	"strconv"
	"time"

	"example.com/gourmetkan/internal/auth"
	"example.com/gourmetkan/internal/util"
)

func (h *Handler) GitHubLogin(w http.ResponseWriter, r *http.Request) {
	state, err := util.RandomToken(16)
	if err != nil {
		http.Error(w, "state error", http.StatusInternalServerError)
		return
	}
	if err := auth.StoreOAuthState(h.db, state, 10*time.Minute); err != nil {
		http.Error(w, "state error", http.StatusInternalServerError)
		return
	}
	loginURL := h.authService.BuildLoginURL(state)
	http.Redirect(w, r, loginURL, http.StatusFound)
}

func (h *Handler) GitHubCallback(w http.ResponseWriter, r *http.Request) {
	state := r.URL.Query().Get("state")
	code := r.URL.Query().Get("code")
	if state == "" || code == "" {
		http.Error(w, "invalid callback", http.StatusBadRequest)
		return
	}
	ok, err := auth.ConsumeOAuthState(h.db, state)
	if err != nil || !ok {
		http.Error(w, "invalid state", http.StatusBadRequest)
		return
	}

	ctx := r.Context()
	token, err := h.authService.ExchangeCode(ctx, code)
	if err != nil {
		http.Error(w, "token error", http.StatusBadRequest)
		return
	}
	user, err := h.authService.FetchGitHubUser(ctx, token.AccessToken)
	if err != nil {
		http.Error(w, "user error", http.StatusBadRequest)
		return
	}

	stored, err := h.userService.UpsertGitHubUser(strconv.Itoa(user.ID), user.Login, user.AvatarURL)
	if err != nil {
		http.Error(w, "user error", http.StatusInternalServerError)
		return
	}
	sessionID, _, expiresAt, err := auth.CreateSession(h.db, stored.ID, h.cfg.SessionTTL)
	if err != nil {
		http.Error(w, "session error", http.StatusInternalServerError)
		return
	}
	h.setSessionCookie(w, sessionID, expiresAt)
	http.Redirect(w, r, "/", http.StatusFound)
}

func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	session, _ := h.getSession(r)
	if session != nil {
		if !h.verifyCSRF(r, session) {
			http.Error(w, "invalid csrf", http.StatusForbidden)
			return
		}
		if cookie, err := r.Cookie(sessionCookieName); err == nil {
			_ = auth.DeleteSession(h.db, cookie.Value)
		}
	}
	h.clearSessionCookie(w)
	http.Redirect(w, r, "/", http.StatusFound)
}
