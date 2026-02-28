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
	Tags        []string
}

type RestaurantListItem struct {
	ID          int
	Name        string
	Description string
	DistanceKm  float64
	Distance    string
	Tags        []string
}

var presetTags = []string{"ラーメン", "居酒屋", "寿司", "焼肉", "カフェ", "定食", "中華", "イタリアン", "カレー"}

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
	allTags, _ := h.restaurantService.ListTags()
	data := TemplateData{
		Bases:          toBaseOptions(bases),
		SelectedBaseID: base.ID,
		CSRFToken:      csrfTokenOrEmpty(session),
		PresetTags:     presetTags,
		AvailableTags:  toTagOptions(allTags),
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
	selectedTags := r.Form["tags"]
	freeform := strings.TrimSpace(r.FormValue("tag_input"))

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
	parsedTags := make([]string, 0)
	for _, tag := range selectedTags {
		parsedTags = append(parsedTags, normalizeTagName(tag))
	}
	parsedTags = append(parsedTags, parseTagList(freeform)...)
	parsedTags = dedupeTags(parsedTags)
	if len(parsedTags) > 10 {
		errors["tags"] = "タグは10個以内で入力してください。"
	}
	for _, tag := range parsedTags {
		if !util.ValidateRequiredText(tag, 1, 20) {
			errors["tags"] = "タグは1〜20文字で入力してください。"
			break
		}
	}

	if len(errors) > 0 {
		bases, _ := h.baseService.ListBases()
		base, _ := h.getSelectedBase(r)
		allTags, _ := h.restaurantService.ListTags()
		selectedSet := make(map[string]bool)
		for _, tag := range parsedTags {
			selectedSet[tag] = true
		}
		data := TemplateData{
			Bases:          toBaseOptions(bases),
			SelectedBaseID: base.ID,
			CSRFToken:      csrfTokenOrEmpty(session),
			Errors:         errors,
			PresetTags:     presetTags,
			AvailableTags:  toTagOptions(allTags),
			SelectedTagSet: selectedSet,
			TagInput:       freeform,
		}
		h.render(w, "restaurants_new.html", data)
		return
	}

	createdID, err := h.restaurantService.CreateRestaurant(services.Restaurant{
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
	if len(parsedTags) > 0 {
		tagIDs := make([]int, 0, len(parsedTags))
		for _, tagName := range parsedTags {
			tag, err := h.restaurantService.UpsertTag(tagName)
			if err != nil {
				http.Error(w, "create error", http.StatusInternalServerError)
				return
			}
			tagIDs = append(tagIDs, tag.ID)
		}
		if err := h.restaurantService.AttachTags(createdID, tagIDs); err != nil {
			http.Error(w, "create error", http.StatusInternalServerError)
			return
		}
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
	tagRows, err := h.restaurantService.TagsForRestaurant(rest.ID)
	if err != nil {
		http.Error(w, "tag error", http.StatusInternalServerError)
		return
	}
	var tagNames []string
	for _, tag := range tagRows {
		tagNames = append(tagNames, tag.Name)
	}

	session, _ := h.getSession(r)
	bases, _ := h.baseService.ListBases()
	var user interface{}
	if session != nil {
		user, _ = h.userService.GetUserByID(session.UserID)
	}
	detail := RestaurantDetail{
		ID:          rest.ID,
		Name:        rest.Name,
		Description: rest.Description,
		Address:     rest.Address,
		MapsURL:     rest.MapsURL,
		Distance:    util.FormatDistanceKm(distanceKm),
		Tags:        tagNames,
	}
	data := TemplateData{
		Bases:          toBaseOptions(bases),
		SelectedBaseID: base.ID,
		User:           user,
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
