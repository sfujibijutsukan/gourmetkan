package handlers

import (
	"context"
	"database/sql"
	"fmt"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	"example.com/gourmetkan/internal/services"
	"example.com/gourmetkan/internal/util"
)

type RestaurantDetail struct {
	ID             int
	Name           string
	Description    string
	PhotoPath      string
	PhotoPaths     []string
	Address        string
	MapsURL        string
	Latitude       float64
	Longitude      float64
	Distance       string
	Tags           []string
	Average        float64
	ReviewCount    int
	AveragePercent int
	CanEdit        bool
}

type RestaurantListItem struct {
	ID          int
	Name        string
	Description string
	PhotoPath   string
	DistanceKm  float64
	Distance    string
	Tags        []string
}

type ReviewDisplay struct {
	ID            int
	RestaurantID  int
	Username      string
	Rating        int
	RatingPercent int
	Comment       string
	PhotoPath     string
	PhotoPaths    []string
	CanManage     bool
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
	user, _ := h.userService.GetUserByID(session.UserID)
	data := TemplateData{
		Bases:          toBaseOptions(bases),
		SelectedBaseID: base.ID,
		User:           user,
		CSRFToken:      csrfTokenOrEmpty(session),
		PresetTags:     presetTags,
		AvailableTags:  toTagOptionsExcludingPreset(allTags, presetTags),
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
	if err := r.ParseMultipartForm(8 << 20); err != nil {
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
	photoPaths, photoErr := util.SaveUploadedImages(r, "photos", "static/uploads", util.DefaultMaxUploadBytes, util.DefaultMaxUploadFiles)
	photoPath := ""
	if len(photoPaths) > 0 {
		photoPath = photoPaths[0]
	}

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
	if photoErr != nil {
		errors["photo"] = "画像は5MB以内の JPG/PNG/GIF/WebP を指定してください。"
	}
	if len(photoPaths) > util.DefaultMaxUploadFiles {
		errors["photo"] = "画像は最大8枚までアップロードできます。"
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
		_ = util.DeleteUploadedImages(photoPaths)
		bases, _ := h.baseService.ListBases()
		base, _ := h.getSelectedBase(r)
		allTags, _ := h.restaurantService.ListTags()
		user, _ := h.userService.GetUserByID(session.UserID)
		selectedSet := make(map[string]bool)
		for _, tag := range parsedTags {
			selectedSet[tag] = true
		}
		data := TemplateData{
			Bases:          toBaseOptions(bases),
			SelectedBaseID: base.ID,
			User:           user,
			CSRFToken:      csrfTokenOrEmpty(session),
			Errors:         errors,
			Restaurant: RestaurantDetail{
				Name:        name,
				Description: description,
				Address:     address,
				MapsURL:     mapsURL,
				Latitude:    latitude,
				Longitude:   longitude,
				PhotoPath:   photoPath,
				PhotoPaths:  photoPaths,
			},
			PresetTags:     presetTags,
			AvailableTags:  toTagOptionsExcludingPreset(allTags, presetTags),
			SelectedTagSet: selectedSet,
			TagInput:       freeform,
		}
		h.render(w, "restaurants_new.html", data)
		return
	}

	createdID, err := h.restaurantService.CreateRestaurant(services.Restaurant{
		Name:        name,
		Description: description,
		PhotoPath:   photoPath,
		Latitude:    latitude,
		Longitude:   longitude,
		Address:     address,
		MapsURL:     mapsURL,
		CreatedBy:   session.UserID,
	})
	if err != nil {
		_ = util.DeleteUploadedImages(photoPaths)
		http.Error(w, "create error", http.StatusInternalServerError)
		return
	}
	if err := h.restaurantService.ReplaceRestaurantPhotos(createdID, photoPaths); err != nil {
		_ = util.DeleteUploadedImages(photoPaths)
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
	session, _ := h.getSession(r)
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
	reviewDisplays := make([]ReviewDisplay, 0, len(reviews))
	for _, review := range reviews {
		reviewPhotoPaths, err := h.reviewService.ListReviewPhotos(review.ID)
		if err != nil {
			http.Error(w, "review error", http.StatusInternalServerError)
			return
		}
		reviewPhotoPath := ""
		if len(reviewPhotoPaths) > 0 {
			reviewPhotoPath = reviewPhotoPaths[0]
		}
		reviewDisplays = append(reviewDisplays, ReviewDisplay{
			ID:            review.ID,
			RestaurantID:  review.RestaurantID,
			Username:      review.Username,
			Rating:        review.Rating,
			RatingPercent: review.Rating * 20,
			Comment:       review.Comment,
			PhotoPath:     reviewPhotoPath,
			PhotoPaths:    reviewPhotoPaths,
			CanManage:     session != nil && session.UserID == review.UserID,
		})
	}
	avgRating, reviewCount, err := h.reviewService.AverageRating(rest.ID)
	if err != nil {
		http.Error(w, "review error", http.StatusInternalServerError)
		return
	}
	starAverage := math.Round(avgRating*2) / 2
	tagRows, err := h.restaurantService.TagsForRestaurant(rest.ID)
	if err != nil {
		http.Error(w, "tag error", http.StatusInternalServerError)
		return
	}
	var tagNames []string
	for _, tag := range tagRows {
		tagNames = append(tagNames, tag.Name)
	}
	restaurantPhotoPaths, err := h.restaurantService.ListRestaurantPhotos(rest.ID)
	if err != nil {
		http.Error(w, "restaurant error", http.StatusInternalServerError)
		return
	}
	restaurantPhotoPath := ""
	if len(restaurantPhotoPaths) > 0 {
		restaurantPhotoPath = restaurantPhotoPaths[0]
	}

	bases, _ := h.baseService.ListBases()
	var user interface{}
	if session != nil {
		user, _ = h.userService.GetUserByID(session.UserID)
	}
	detail := RestaurantDetail{
		ID:             rest.ID,
		Name:           rest.Name,
		Description:    rest.Description,
		PhotoPath:      restaurantPhotoPath,
		PhotoPaths:     restaurantPhotoPaths,
		Address:        rest.Address,
		MapsURL:        rest.MapsURL,
		Latitude:       rest.Latitude,
		Longitude:      rest.Longitude,
		Distance:       util.FormatDistanceKm(distanceKm),
		Tags:           tagNames,
		Average:        avgRating,
		ReviewCount:    reviewCount,
		AveragePercent: int(math.Round(starAverage / 5 * 100)),
		CanEdit:        session != nil,
	}
	data := TemplateData{
		Bases:          toBaseOptions(bases),
		SelectedBaseID: base.ID,
		User:           user,
		CSRFToken:      csrfTokenOrEmpty(session),
		Restaurant:     detail,
		Reviews:        reviewDisplays,
	}
	h.render(w, "restaurants_show.html", data)
}

func (h *Handler) EditRestaurant(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	session, ok := h.requireLogin(w, r)
	if !ok {
		return
	}
	id, err := extractID(strings.TrimSuffix(r.URL.Path, "/edit"))
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
	bases, _ := h.baseService.ListBases()
	allTags, _ := h.restaurantService.ListTags()
	user, _ := h.userService.GetUserByID(session.UserID)
	tagRows, _ := h.restaurantService.TagsForRestaurant(rest.ID)
	restaurantPhotoPaths, _ := h.restaurantService.ListRestaurantPhotos(rest.ID)
	restaurantPhotoPath := ""
	if len(restaurantPhotoPaths) > 0 {
		restaurantPhotoPath = restaurantPhotoPaths[0]
	}
	selectedSet := make(map[string]bool)
	for _, tag := range tagRows {
		selectedSet[tag.Name] = true
	}
	data := TemplateData{
		Bases:          toBaseOptions(bases),
		SelectedBaseID: base.ID,
		User:           user,
		CSRFToken:      csrfTokenOrEmpty(session),
		Restaurant: RestaurantDetail{
			ID:          rest.ID,
			Name:        rest.Name,
			Description: rest.Description,
			PhotoPath:   restaurantPhotoPath,
			PhotoPaths:  restaurantPhotoPaths,
			Address:     rest.Address,
			MapsURL:     rest.MapsURL,
			Latitude:    rest.Latitude,
			Longitude:   rest.Longitude,
		},
		PresetTags:     presetTags,
		AvailableTags:  toTagOptionsExcludingPreset(allTags, presetTags),
		SelectedTagSet: selectedSet,
	}
	h.render(w, "restaurants_edit.html", data)
}

func (h *Handler) UpdateRestaurant(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	session, ok := h.requireLogin(w, r)
	if !ok {
		return
	}
	if err := r.ParseMultipartForm(8 << 20); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	if !h.verifyCSRF(r, session) {
		http.Error(w, "invalid csrf", http.StatusForbidden)
		return
	}

	id, err := extractID(strings.TrimSuffix(r.URL.Path, "/update"))
	if err != nil {
		http.NotFound(w, r)
		return
	}
	rest, err := h.restaurantService.GetRestaurant(id)
	if err != nil || rest == nil {
		http.NotFound(w, r)
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
	existingPhotoPaths, err := h.restaurantService.ListRestaurantPhotos(rest.ID)
	if err != nil {
		http.Error(w, "update error", http.StatusInternalServerError)
		return
	}
	photoPaths := append([]string(nil), existingPhotoPaths...)
	removePhoto := r.FormValue("remove_photo") == "1"
	removeSelected := r.Form["remove_photos"]
	newPhotoPaths, photoErr := util.SaveUploadedImages(r, "photos", "static/uploads", util.DefaultMaxUploadBytes, util.DefaultMaxUploadFiles)
	if removePhoto {
		removeSelected = photoPaths
	}
	removeSet := make(map[string]bool, len(removeSelected))
	for _, path := range removeSelected {
		removeSet[path] = true
	}
	keptPhotoPaths := make([]string, 0, len(photoPaths))
	for _, path := range photoPaths {
		if !removeSet[path] {
			keptPhotoPaths = append(keptPhotoPaths, path)
		}
	}
	photoPaths = append(keptPhotoPaths, newPhotoPaths...)
	photoPath := ""
	if len(photoPaths) > 0 {
		photoPath = photoPaths[0]
	}
	if removePhoto {
		// remove_photo keeps compatibility with previous UI and removes all current photos.
	}
	if len(photoPaths) > util.DefaultMaxUploadFiles {
		photoErr = fmt.Errorf("too many photos")
	}

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
	if photoErr != nil {
		errors["photo"] = "画像は5MB以内の JPG/PNG/GIF/WebP を指定してください。"
	}
	if len(photoPaths) > util.DefaultMaxUploadFiles {
		errors["photo"] = "画像は最大8枚までアップロードできます。"
	}

	latProvided := latStr != ""
	lngProvided := lngStr != ""
	latitude := rest.Latitude
	longitude := rest.Longitude
	locationSet := true
	if latProvided || lngProvided {
		if !(latProvided && lngProvided) {
			errors["latitude"] = "緯度経度は両方入力してください。"
			locationSet = false
		} else {
			lat, err1 := strconv.ParseFloat(latStr, 64)
			lng, err2 := strconv.ParseFloat(lngStr, 64)
			if err1 != nil || err2 != nil || !util.ValidateLatitude(lat) || !util.ValidateLongitude(lng) {
				errors["latitude"] = "緯度経度が不正です。"
				locationSet = false
			} else {
				latitude = lat
				longitude = lng
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
		_ = util.DeleteUploadedImages(newPhotoPaths)
		base, _ := h.getSelectedBase(r)
		bases, _ := h.baseService.ListBases()
		allTags, _ := h.restaurantService.ListTags()
		user, _ := h.userService.GetUserByID(session.UserID)
		selectedSet := make(map[string]bool)
		for _, tag := range parsedTags {
			selectedSet[tag] = true
		}
		data := TemplateData{
			Bases:          toBaseOptions(bases),
			SelectedBaseID: base.ID,
			User:           user,
			CSRFToken:      csrfTokenOrEmpty(session),
			Errors:         errors,
			Restaurant: RestaurantDetail{
				ID:          rest.ID,
				Name:        name,
				Description: description,
				PhotoPath:   photoPath,
				PhotoPaths:  photoPaths,
				Address:     address,
				MapsURL:     mapsURL,
				Latitude:    latitude,
				Longitude:   longitude,
			},
			PresetTags:     presetTags,
			AvailableTags:  toTagOptionsExcludingPreset(allTags, presetTags),
			SelectedTagSet: selectedSet,
			TagInput:       freeform,
		}
		h.render(w, "restaurants_edit.html", data)
		return
	}

	if err := h.restaurantService.UpdateRestaurant(services.Restaurant{
		ID:          rest.ID,
		Name:        name,
		Description: description,
		PhotoPath:   photoPath,
		Latitude:    latitude,
		Longitude:   longitude,
		Address:     address,
		MapsURL:     mapsURL,
	}); err != nil {
		if err == sql.ErrNoRows {
			http.NotFound(w, r)
			return
		}
		http.Error(w, "update error", http.StatusInternalServerError)
		return
	}
	if err := h.restaurantService.ReplaceRestaurantPhotos(rest.ID, photoPaths); err != nil {
		_ = util.DeleteUploadedImages(newPhotoPaths)
		http.Error(w, "update error", http.StatusInternalServerError)
		return
	}
	removedPaths := make([]string, 0, len(existingPhotoPaths))
	finalSet := make(map[string]bool, len(photoPaths))
	for _, path := range photoPaths {
		finalSet[path] = true
	}
	for _, path := range existingPhotoPaths {
		if !finalSet[path] {
			removedPaths = append(removedPaths, path)
		}
	}
	if len(removedPaths) > 0 {
		_ = util.DeleteUploadedImages(removedPaths)
	}

	tagIDs := make([]int, 0, len(parsedTags))
	for _, tagName := range parsedTags {
		tag, err := h.restaurantService.UpsertTag(tagName)
		if err != nil {
			http.Error(w, "update error", http.StatusInternalServerError)
			return
		}
		tagIDs = append(tagIDs, tag.ID)
	}
	if err := h.restaurantService.ReplaceTags(rest.ID, tagIDs); err != nil {
		http.Error(w, "update error", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/restaurants/"+strconv.Itoa(rest.ID), http.StatusFound)
}

func (h *Handler) DeleteRestaurant(w http.ResponseWriter, r *http.Request) {
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
	id, err := extractID(strings.TrimSuffix(r.URL.Path, "/delete"))
	if err != nil {
		http.NotFound(w, r)
		return
	}
	rest, err := h.restaurantService.GetRestaurant(id)
	if err != nil || rest == nil {
		http.NotFound(w, r)
		return
	}
	if rest.CreatedBy != session.UserID {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	photoPaths, _ := h.restaurantService.ListRestaurantPhotos(id)
	if err := h.restaurantService.DeleteRestaurant(id, session.UserID); err != nil {
		if err == sql.ErrNoRows {
			http.NotFound(w, r)
			return
		}
		http.Error(w, "delete error", http.StatusInternalServerError)
		return
	}
	if len(photoPaths) > 0 {
		_ = util.DeleteUploadedImages(photoPaths)
	}
	http.Redirect(w, r, "/", http.StatusFound)
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
	if err := r.ParseMultipartForm(8 << 20); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
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
	photoPaths, photoErr := util.SaveUploadedImages(r, "photos", "static/uploads", util.DefaultMaxUploadBytes, util.DefaultMaxUploadFiles)
	photoPath := ""
	if len(photoPaths) > 0 {
		photoPath = photoPaths[0]
	}
	if photoErr != nil {
		http.Error(w, "invalid photo", http.StatusBadRequest)
		return
	}
	createdReviewID, err := h.reviewService.CreateReview(services.Review{
		RestaurantID: id,
		UserID:       session.UserID,
		Rating:       rating,
		Comment:      comment,
		PhotoPath:    photoPath,
	})
	if err != nil {
		_ = util.DeleteUploadedImages(photoPaths)
		http.Error(w, "review error", http.StatusInternalServerError)
		return
	}
	if err := h.reviewService.ReplaceReviewPhotos(createdReviewID, photoPaths); err != nil {
		_ = util.DeleteUploadedImages(photoPaths)
		http.Error(w, "review error", http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/restaurants/"+strconv.Itoa(id), http.StatusFound)
}

func (h *Handler) EditReview(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	session, ok := h.requireLogin(w, r)
	if !ok {
		return
	}
	reviewID, err := extractID(strings.TrimSuffix(r.URL.Path, "/edit"))
	if err != nil {
		http.NotFound(w, r)
		return
	}
	review, err := h.reviewService.GetReview(reviewID)
	if err != nil || review == nil {
		http.NotFound(w, r)
		return
	}
	if review.UserID != session.UserID {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	rest, err := h.restaurantService.GetRestaurant(review.RestaurantID)
	if err != nil || rest == nil {
		http.NotFound(w, r)
		return
	}
	base, _ := h.getSelectedBase(r)
	bases, _ := h.baseService.ListBases()
	user, _ := h.userService.GetUserByID(session.UserID)
	data := TemplateData{
		Bases:          toBaseOptions(bases),
		SelectedBaseID: base.ID,
		User:           user,
		CSRFToken:      csrfTokenOrEmpty(session),
		Restaurant: RestaurantDetail{
			ID:   rest.ID,
			Name: rest.Name,
		},
		Review: ReviewDisplay{
			ID:           review.ID,
			RestaurantID: review.RestaurantID,
			Rating:       review.Rating,
			Comment:      review.Comment,
			PhotoPath:    review.PhotoPath,
			PhotoPaths:   review.PhotoPaths,
		},
	}
	reviewPhotoPaths, _ := h.reviewService.ListReviewPhotos(review.ID)
	reviewPhotoPath := ""
	if len(reviewPhotoPaths) > 0 {
		reviewPhotoPath = reviewPhotoPaths[0]
	} else {
		reviewPhotoPath = review.PhotoPath
	}
	data.Review = ReviewDisplay{
		ID:           review.ID,
		RestaurantID: review.RestaurantID,
		Rating:       review.Rating,
		Comment:      review.Comment,
		PhotoPath:    reviewPhotoPath,
		PhotoPaths:   reviewPhotoPaths,
	}
	h.render(w, "reviews_edit.html", data)
}

func (h *Handler) UpdateReview(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	session, ok := h.requireLogin(w, r)
	if !ok {
		return
	}
	if err := r.ParseMultipartForm(8 << 20); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	if !h.verifyCSRF(r, session) {
		http.Error(w, "invalid csrf", http.StatusForbidden)
		return
	}
	reviewID, err := extractID(strings.TrimSuffix(r.URL.Path, "/update"))
	if err != nil {
		http.NotFound(w, r)
		return
	}
	review, err := h.reviewService.GetReview(reviewID)
	if err != nil || review == nil {
		http.NotFound(w, r)
		return
	}
	if review.UserID != session.UserID {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	rest, err := h.restaurantService.GetRestaurant(review.RestaurantID)
	if err != nil || rest == nil {
		http.NotFound(w, r)
		return
	}

	rating, err := strconv.Atoi(r.FormValue("rating"))
	if err != nil || rating < 1 || rating > 5 {
		http.Error(w, "invalid rating", http.StatusBadRequest)
		return
	}
	comment := strings.TrimSpace(r.FormValue("comment"))
	existingPhotoPaths, err := h.reviewService.ListReviewPhotos(review.ID)
	if err != nil {
		http.Error(w, "review error", http.StatusInternalServerError)
		return
	}
	photoPaths := append([]string(nil), existingPhotoPaths...)
	removePhoto := r.FormValue("remove_photo") == "1"
	removeSelected := r.Form["remove_photos"]
	removeSet := make(map[string]bool, len(removeSelected))
	for _, path := range removeSelected {
		removeSet[path] = true
	}
	if removePhoto {
		for _, path := range photoPaths {
			removeSet[path] = true
		}
	}
	keptPhotoPaths := make([]string, 0, len(photoPaths))
	for _, path := range photoPaths {
		if !removeSet[path] {
			keptPhotoPaths = append(keptPhotoPaths, path)
		}
	}
	photoPaths = keptPhotoPaths
	photoPath := ""
	if len(photoPaths) > 0 {
		photoPath = photoPaths[0]
	}
	newPhotoPaths, photoErr := util.SaveUploadedImages(r, "photos", "static/uploads", util.DefaultMaxUploadBytes, util.DefaultMaxUploadFiles)
	photoPaths = append(photoPaths, newPhotoPaths...)
	if len(photoPaths) > 0 {
		photoPath = photoPaths[0]
	}

	errors := map[string]string{}
	if !util.ValidateRequiredText(comment, 1, 1000) {
		errors["comment"] = "コメントは1〜1000文字で入力してください。"
	}
	if photoErr != nil {
		errors["photo"] = "画像は5MB以内の JPG/PNG/GIF/WebP を指定してください。"
	}
	if len(photoPaths) > util.DefaultMaxUploadFiles {
		errors["photo"] = "画像は最大8枚までアップロードできます。"
	}
	if len(errors) > 0 {
		_ = util.DeleteUploadedImages(newPhotoPaths)
		base, _ := h.getSelectedBase(r)
		bases, _ := h.baseService.ListBases()
		user, _ := h.userService.GetUserByID(session.UserID)
		data := TemplateData{
			Bases:          toBaseOptions(bases),
			SelectedBaseID: base.ID,
			User:           user,
			CSRFToken:      csrfTokenOrEmpty(session),
			Errors:         errors,
			Restaurant: RestaurantDetail{
				ID:   rest.ID,
				Name: rest.Name,
			},
			Review: ReviewDisplay{
				ID:           review.ID,
				RestaurantID: review.RestaurantID,
				Rating:       rating,
				Comment:      comment,
				PhotoPath:    photoPath,
				PhotoPaths:   photoPaths,
			},
		}
		h.render(w, "reviews_edit.html", data)
		return
	}

	if err := h.reviewService.UpdateReview(services.Review{
		ID:        review.ID,
		UserID:    session.UserID,
		Rating:    rating,
		Comment:   comment,
		PhotoPath: photoPath,
	}); err != nil {
		if err == sql.ErrNoRows {
			http.NotFound(w, r)
			return
		}
		http.Error(w, "review error", http.StatusInternalServerError)
		return
	}
	if err := h.reviewService.ReplaceReviewPhotos(review.ID, photoPaths); err != nil {
		_ = util.DeleteUploadedImages(newPhotoPaths)
		http.Error(w, "review error", http.StatusInternalServerError)
		return
	}
	removedPaths := make([]string, 0, len(existingPhotoPaths))
	finalSet := make(map[string]bool, len(photoPaths))
	for _, path := range photoPaths {
		finalSet[path] = true
	}
	for _, path := range existingPhotoPaths {
		if !finalSet[path] {
			removedPaths = append(removedPaths, path)
		}
	}
	if len(removedPaths) > 0 {
		_ = util.DeleteUploadedImages(removedPaths)
	}

	http.Redirect(w, r, "/restaurants/"+strconv.Itoa(review.RestaurantID), http.StatusFound)
}

func (h *Handler) DeleteReview(w http.ResponseWriter, r *http.Request) {
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
	reviewID, err := extractID(strings.TrimSuffix(r.URL.Path, "/delete"))
	if err != nil {
		http.NotFound(w, r)
		return
	}
	review, err := h.reviewService.GetReview(reviewID)
	if err != nil || review == nil {
		http.NotFound(w, r)
		return
	}
	if review.UserID != session.UserID {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	photoPaths, _ := h.reviewService.ListReviewPhotos(reviewID)
	if err := h.reviewService.DeleteReview(reviewID, session.UserID); err != nil {
		if err == sql.ErrNoRows {
			http.NotFound(w, r)
			return
		}
		http.Error(w, "review error", http.StatusInternalServerError)
		return
	}
	if len(photoPaths) > 0 {
		_ = util.DeleteUploadedImages(photoPaths)
	}
	http.Redirect(w, r, "/restaurants/"+strconv.Itoa(review.RestaurantID), http.StatusFound)
}

func extractID(path string) (int, error) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) < 2 {
		return 0, fmt.Errorf("invalid path")
	}
	return strconv.Atoi(parts[1])
}
