package handlers

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"example.com/gourmetkan/internal/services"
	"example.com/gourmetkan/internal/util"
)

type RestaurantDetail struct {
	ID          int
	Name        string
	Description string
	Address     string
	MapsURL     string
	Distance    string
}

func (h *Handler) NewRestaurant(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	session, ok := h.requireLogin(w, r)
	if !ok {
		return
	}
	bases, _ := h.baseService.ListBases()
	base, _ := h.getSelectedBase(r)
	data := TemplateData{
		Bases:          toBaseOptions(bases),
		SelectedBaseID: base.ID,
		CSRFToken:      csrfTokenOrEmpty(session),
	}
	h.render(w, "restaurants_new.html", data)
}

func (h *Handler) CreateRestaurant(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	session, ok := h.requireLogin(w, r)
	if !ok {
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	if !h.verifyCSRF(r, session) {
		http.Error(w, "invalid csrf", http.StatusForbidden)
		return
	}

	name := strings.TrimSpace(r.FormValue("name"))
	description := strings.TrimSpace(r.FormValue("description"))
	mapsURL := strings.TrimSpace(r.FormValue("maps_url"))
	address := strings.TrimSpace(r.FormValue("address"))
	latStr := strings.TrimSpace(r.FormValue("latitude"))
	lngStr := strings.TrimSpace(r.FormValue("longitude"))

	errors := map[string]string{}
	if !util.ValidateRequiredText(name, 1, 100) {
		errors["name"] = "店名は1〜100文字で入力してください。"
	}
	if !util.ValidateOptionalText(description, 500) {
		errors["description"] = "説明は500文字以内で入力してください。"
	}
	if !util.ValidateOptionalText(address, 200) {
		errors["address"] = "住所は200文字以内で入力してください。"
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
		data := TemplateData{
			Bases:          toBaseOptions(bases),
			SelectedBaseID: base.ID,
			CSRFToken:      csrfTokenOrEmpty(session),
			Errors:         errors,
		}
		h.render(w, "restaurants_new.html", data)
		return
	}

	_, err := h.restaurantService.CreateRestaurant(services.Restaurant{
		Name:        name,
		Description: description,
		Latitude:    latitude,
		Longitude:   longitude,
		Address:     address,
		MapsURL:     mapsURL,
		CreatedBy:   session.UserID,
	})
	if err != nil {
		http.Error(w, "create error", http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/", http.StatusFound)
}

func (h *Handler) RestaurantDetail(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	id, err := extractID(r.URL.Path)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	rest, err := h.restaurantService.GetRestaurant(id)
	if err != nil || rest == nil {
		http.NotFound(w, r)
		return
	}
	base, _ := h.getSelectedBase(r)
	distanceKm := util.HaversineDistanceKm(base.Latitude, base.Longitude, rest.Latitude, rest.Longitude)

	page := 1
	if p := r.URL.Query().Get("page"); p != "" {
		if parsed, err := strconv.Atoi(p); err == nil && parsed > 0 {
			page = parsed
		}
	}
	limit := 20
	offset := (page - 1) * limit
	reviews, err := h.reviewService.ListReviews(rest.ID, limit, offset)
	if err != nil {
		http.Error(w, "review error", http.StatusInternalServerError)
		return
	}

	session, _ := h.getSession(r)
	bases, _ := h.baseService.ListBases()
	detail := RestaurantDetail{
		ID:          rest.ID,
		Name:        rest.Name,
		Description: rest.Description,
		Address:     rest.Address,
		MapsURL:     rest.MapsURL,
		Distance:    util.FormatDistanceKm(distanceKm),
	}
	data := TemplateData{
		Bases:          toBaseOptions(bases),
		SelectedBaseID: base.ID,
		CSRFToken:      csrfTokenOrEmpty(session),
		Restaurant:     detail,
		Reviews:        reviews,
	}
	h.render(w, "restaurants_show.html", data)
}

func (h *Handler) CreateReview(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	session, ok := h.requireLogin(w, r)
	if !ok {
		return
	}
	if !h.verifyCSRF(r, session) {
		http.Error(w, "invalid csrf", http.StatusForbidden)
		return
	}
	id, err := extractID(strings.TrimSuffix(r.URL.Path, "/reviews"))
	if err != nil {
		http.NotFound(w, r)
		return
	}
	rating, err := strconv.Atoi(r.FormValue("rating"))
	if err != nil || rating < 1 || rating > 5 {
		http.Error(w, "invalid rating", http.StatusBadRequest)
		return
	}
	comment := strings.TrimSpace(r.FormValue("comment"))
	if !util.ValidateRequiredText(comment, 1, 1000) {
		http.Error(w, "invalid comment", http.StatusBadRequest)
		return
	}
	if err := h.reviewService.CreateReview(services.Review{
		RestaurantID: id,
		UserID:       session.UserID,
		Rating:       rating,
		Comment:      comment,
	}); err != nil {
		http.Error(w, "review error", http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/restaurants/"+strconv.Itoa(id), http.StatusFound)
}

func extractID(path string) (int, error) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) < 2 {
		return 0, fmt.Errorf("invalid path")
	}
	return strconv.Atoi(parts[1])
}
