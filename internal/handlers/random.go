package handlers

import (
	"math/rand"
	"net/http"
	"strconv"
	"time"

	"example.com/gourmetkan/internal/util"
)

func (h *Handler) RandomRestaurant(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	base, err := h.getSelectedBase(r)
	if err != nil || base == nil {
		http.Error(w, "base error", http.StatusInternalServerError)
		return
	}
	radiusKm := 2.0
	if value := r.URL.Query().Get("radius_km"); value != "" {
		if parsed, err := strconv.ParseFloat(value, 64); err == nil && parsed > 0 {
			radiusKm = parsed
		}
	}

	restaurants, err := h.restaurantService.ListRestaurants()
	if err != nil {
		http.Error(w, "restaurant error", http.StatusInternalServerError)
		return
	}

	candidates := make([]int, 0)
	for _, rest := range restaurants {
		distance := util.HaversineDistanceKm(base.Latitude, base.Longitude, rest.Latitude, rest.Longitude)
		if distance <= radiusKm {
			candidates = append(candidates, rest.ID)
		}
	}
	if len(candidates) == 0 {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}
	rnd := rand.New(rand.NewSource(time.Now().UnixNano()))
	picked := candidates[rnd.Intn(len(candidates))]
	http.Redirect(w, r, "/restaurants/"+strconv.Itoa(picked), http.StatusFound)
}
