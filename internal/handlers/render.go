package handlers

import (
	"bytes"
	"html/template"
	"net/http"
	"path/filepath"
)

type TemplateData struct {
	Bases          []BaseOption
	SelectedBaseID int
	User           interface{}
	CSRFToken      string
	Restaurants    interface{}
	Restaurant     interface{}
	Reviews        interface{}
	Errors         map[string]string
	Notice         string
	RadiusKm       float64
}

func (h *Handler) render(w http.ResponseWriter, name string, data TemplateData) {
	if h.templates == nil {
		h.templates = make(map[string]*template.Template)
	}
	tmpl, ok := h.templates[name]
	if !ok {
		layoutPath := filepath.Join("templates", "layout.html")
		pagePath := filepath.Join("templates", name)
		parsed, err := template.ParseFiles(layoutPath, pagePath)
		if err != nil {
			http.Error(w, "template error", http.StatusInternalServerError)
			return
		}
		h.templates[name] = parsed
		tmpl = parsed
	}
	var buf bytes.Buffer
	if err := tmpl.ExecuteTemplate(&buf, "layout", data); err != nil {
		http.Error(w, "render error", http.StatusInternalServerError)
		return
	}
	_, _ = w.Write(buf.Bytes())
}
