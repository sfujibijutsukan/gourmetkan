package handlers

import (
	"net/http"
	"sort"
	"strings"

	"example.com/gourmetkan/internal/services"
	"example.com/gourmetkan/internal/util"
)

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
	selectedTag := strings.TrimSpace(r.URL.Query().Get("tag"))
	var restaurants []services.Restaurant
	if selectedTag != "" {
		restaurants, err = h.restaurantService.ListRestaurantsByTag(selectedTag)
	} else {
		restaurants, err = h.restaurantService.ListRestaurants()
	}
	if err != nil {
		http.Error(w, "restaurant error", http.StatusInternalServerError)
		return
	}

	tagMap, err := h.restaurantService.TagsForRestaurants(restaurants)
	if err != nil {
		http.Error(w, "tag error", http.StatusInternalServerError)
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
			Tags:        tagMap[rest.ID],
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
	allTags, _ := h.restaurantService.ListTags()
	data := TemplateData{
		Bases:          toBaseOptions(bases),
		SelectedBaseID: base.ID,
		User:           user,
		Restaurants:    items,
		CSRFToken:      csrfTokenOrEmpty(session),
		AvailableTags:  toTagOptions(allTags),
		SelectedTag:    selectedTag,
	}
	h.render(w, "index.html", data)
}
