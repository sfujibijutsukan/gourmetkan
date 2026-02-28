package handlers

import (
	"strings"

	"example.com/gourmetkan/internal/services"
)

type BaseOption struct {
	ID   int
	Name string
}

type TagOption struct {
	Name string
}

func toBaseOptions(bases []services.Base) []BaseOption {
	options := make([]BaseOption, 0, len(bases))
	for _, base := range bases {
		options = append(options, BaseOption{ID: base.ID, Name: base.Name})
	}
	return options
}

func toTagOptions(tags []services.Tag) []TagOption {
	options := make([]TagOption, 0, len(tags))
	for _, tag := range tags {
		options = append(options, TagOption{Name: tag.Name})
	}
	return options
}

func normalizeTagName(value string) string {
	trimmed := strings.TrimSpace(value)
	trimmed = strings.Trim(trimmed, "#")
	return trimmed
}

func dedupeTags(names []string) []string {
	seen := make(map[string]bool)
	result := make([]string, 0, len(names))
	for _, name := range names {
		if name == "" {
			continue
		}
		key := strings.ToLower(name)
		if seen[key] {
			continue
		}
		seen[key] = true
		result = append(result, name)
	}
	return result
}

func parseTagList(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	parts := strings.FieldsFunc(raw, func(r rune) bool {
		return r == ',' || r == '、' || r == ';'
	})
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		result = append(result, normalizeTagName(part))
	}
	return dedupeTags(result)
}

func csrfTokenOrEmpty(session *SessionInfo) string {
	if session == nil {
		return ""
	}
	return session.CSRFToken
}
