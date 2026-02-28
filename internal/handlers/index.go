package handlers

import (
	"net/http"
	"sort"

	"example.com/gourmetkan/internal/util"
)

type RestaurantListItem struct {
	ID          int
	Name        string
	Description string
	DistanceKm  float64
	Distance    string
}

func (h *Handler) Index(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	base, err := h.getSelectedBase(r)
	if err != nil || base == nil {
		http.Error(w, "base error", http.StatusInternalServerError)
		return
	}
	restaurants, err := h.restaurantService.ListRestaurants()
	if err != nil {
		http.Error(w, "restaurant error", http.StatusInternalServerError)
		return
	}

	items := make([]RestaurantListItem, 0, len(restaurants))
	for _, rest := range restaurants {
		distanceKm := util.HaversineDistanceKm(base.Latitude, base.Longitude, rest.Latitude, rest.Longitude)
		items = append(items, RestaurantListItem{
			ID:          rest.ID,
			Name:        rest.Name,
			Description: rest.Description,
			DistanceKm:  distanceKm,
			Distance:    util.FormatDistanceKm(distanceKm),
		})
	}
	sort.Slice(items, func(i, j int) bool {
		return items[i].DistanceKm < items[j].DistanceKm
	})

	bases, _ := h.baseService.ListBases()
	session, _ := h.getSession(r)
	var user interface{}
	if session != nil {
		user, _ = h.userService.GetUserByID(session.UserID)
	}
	data := TemplateData{
		Bases:          toBaseOptions(bases),
		SelectedBaseID: base.ID,
		User:           user,
		Restaurants:    items,
		CSRFToken:      csrfTokenOrEmpty(session),
	}
	h.render(w, "index.html", data)
}
