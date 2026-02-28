package handlers

import (
	"net/http"
	"strconv"

	"example.com/gourmetkan/internal/services"
)

func (h *Handler) SelectBase(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	if session, _ := h.getSession(r); session != nil {
		if !h.verifyCSRF(r, session) {
			http.Error(w, "invalid csrf", http.StatusForbidden)
			return
		}
	}
	baseID, err := strconv.Atoi(r.FormValue("base_id"))
	if err != nil {
		http.Error(w, "invalid base", http.StatusBadRequest)
		return
	}

	base, err := h.baseService.GetBaseByID(baseID)
	if err != nil || base == nil {
		http.Error(w, "invalid base", http.StatusBadRequest)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     baseCookieName,
		Value:    strconv.Itoa(base.ID),
		Path:     "/",
		HttpOnly: false,
		Secure:   h.cfg.CookieSecure,
		SameSite: http.SameSiteLaxMode,
	})
	http.Redirect(w, r, "/", http.StatusFound)
}

func (h *Handler) getSelectedBase(r *http.Request) (*services.Base, error) {
	bases, err := h.baseService.ListBases()
	if err != nil {
		return nil, err
	}
	if len(bases) == 0 {
		return nil, nil
	}
	baseID := bases[0].ID
	if cookie, err := r.Cookie(baseCookieName); err == nil {
		if parsed, err := strconv.Atoi(cookie.Value); err == nil {
			baseID = parsed
		}
	}
	selected, err := h.baseService.GetBaseByID(baseID)
	if err != nil || selected == nil {
		selected = &bases[0]
	}
	return selected, nil
}
