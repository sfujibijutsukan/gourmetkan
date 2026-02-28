package util

import "strings"

func ValidateRequiredText(value string, min, max int) bool {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return false
	}
	if len([]rune(trimmed)) < min {
		return false
	}
	if len([]rune(trimmed)) > max {
		return false
	}
	return true
}

func ValidateOptionalText(value string, max int) bool {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return true
	}
	return len([]rune(trimmed)) <= max
}

func ValidateLatitude(lat float64) bool {
	return lat >= -90 && lat <= 90
}

func ValidateLongitude(lng float64) bool {
	return lng >= -180 && lng <= 180
}
