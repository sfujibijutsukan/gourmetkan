package handlers

import (
	"net/http"
	"strings"
)

func (h *Handler) RestaurantRouter(w http.ResponseWriter, r *http.Request) {
	if strings.HasSuffix(r.URL.Path, "/reviews") {
		h.CreateReview(w, r)
		return
	}
	if r.URL.Path == "/restaurants" {
		h.CreateRestaurant(w, r)
		return
	}
	h.RestaurantDetail(w, r)
}
