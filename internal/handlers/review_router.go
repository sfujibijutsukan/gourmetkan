package handlers

import (
	"net/http"
	"strings"
)

func (h *Handler) ReviewRouter(w http.ResponseWriter, r *http.Request) {
	if strings.HasSuffix(r.URL.Path, "/edit") {
		h.EditReview(w, r)
		return
	}
	if strings.HasSuffix(r.URL.Path, "/update") {
		h.UpdateReview(w, r)
		return
	}
	if strings.HasSuffix(r.URL.Path, "/delete") {
		h.DeleteReview(w, r)
		return
	}
	http.NotFound(w, r)
}
