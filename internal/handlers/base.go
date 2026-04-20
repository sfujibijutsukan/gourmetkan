package handlers

import (
	"context"
	"net/http"
	"strconv"
	"strings"
	"time"

	"example.com/gourmetkan/internal/services"
	"example.com/gourmetkan/internal/util"
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

func (h *Handler) NewBase(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	bases, _ := h.baseService.ListBases()
	base, _ := h.getSelectedBase(r)
	session, _ := h.getSession(r)
	selectedID := 0
	if base != nil {
		selectedID = base.ID
	}
	data := TemplateData{
		Bases:          toBaseOptions(bases),
		SelectedBaseID: selectedID,
		CSRFToken:      csrfTokenOrEmpty(session),
	}
	h.render(w, "bases_new.html", data)
}

func (h *Handler) CreateBase(w http.ResponseWriter, r *http.Request) {
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

	name := strings.TrimSpace(r.FormValue("name"))
	latStr := strings.TrimSpace(r.FormValue("latitude"))
	lngStr := strings.TrimSpace(r.FormValue("longitude"))
	mapsURL := strings.TrimSpace(r.FormValue("maps_url"))

	errors := map[string]string{}
	if !util.ValidateRequiredText(name, 1, 100) {
		errors["name"] = "拠点名は1〜100文字で入力してください。"
	}
	latProvided := latStr != ""
	lngProvided := lngStr != ""
	var latitude, longitude float64
	locationSet := false
	if latProvided || lngProvided {
		if !(latProvided && lngProvided) {
			errors["latitude"] = "緯度経度は両方入力してください。"
		} else {
			lat, err1 := strconv.ParseFloat(latStr, 64)
			lng, err2 := strconv.ParseFloat(lngStr, 64)
			if err1 != nil || err2 != nil || !util.ValidateLatitude(lat) || !util.ValidateLongitude(lng) {
				errors["latitude"] = "緯度経度が不正です。"
			} else {
				latitude = lat
				longitude = lng
				locationSet = true
			}
		}
	}

	if mapsURL != "" {
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()
		expanded, err := util.ExpandShortURL(ctx, mapsURL)
		if err == nil {
			mapsURL = expanded
		}
		if loc, ok := util.ParseMapLocation(mapsURL); ok {
			latitude = loc.Latitude
			longitude = loc.Longitude
			locationSet = true
		}
	}

	if !locationSet {
		errors["latitude"] = "緯度経度が取得できませんでした。"
	}

	if len(errors) > 0 {
		bases, _ := h.baseService.ListBases()
		base, _ := h.getSelectedBase(r)
		session, _ := h.getSession(r)
		selectedID := 0
		if base != nil {
			selectedID = base.ID
		}
		data := TemplateData{
			Bases:          toBaseOptions(bases),
			SelectedBaseID: selectedID,
			CSRFToken:      csrfTokenOrEmpty(session),
			Errors:         errors,
		}
		h.render(w, "bases_new.html", data)
		return
	}

	baseID, err := h.baseService.CreateBase(services.Base{
		Name:      name,
		Latitude:  latitude,
		Longitude: longitude,
	})
	if err != nil {
		http.Error(w, "create error", http.StatusInternalServerError)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     baseCookieName,
		Value:    strconv.Itoa(baseID),
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
