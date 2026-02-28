package handlers

import "example.com/gourmetkan/internal/services"

type BaseOption struct {
	ID   int
	Name string
}

func toBaseOptions(bases []services.Base) []BaseOption {
	options := make([]BaseOption, 0, len(bases))
	for _, base := range bases {
		options = append(options, BaseOption{ID: base.ID, Name: base.Name})
	}
	return options
}

func csrfTokenOrEmpty(session *SessionInfo) string {
	if session == nil {
		return ""
	}
	return session.CSRFToken
}
