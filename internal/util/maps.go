package util

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"time"
)

var (
	mapsAtPattern    = regexp.MustCompile(`@(-?\d+\.\d+),(-?\d+\.\d+)`)
	mapsQueryPattern = regexp.MustCompile(`q=(-?\d+\.\d+),(-?\d+\.\d+)`)
)

type MapLocation struct {
	Latitude  float64
	Longitude float64
}

func ParseMapLocation(raw string) (MapLocation, bool) {
	if raw == "" {
		return MapLocation{}, false
	}
	decoded, err := url.QueryUnescape(raw)
	if err != nil {
		decoded = raw
	}

	if match := mapsAtPattern.FindStringSubmatch(decoded); len(match) == 3 {
		lat, err1 := strconv.ParseFloat(match[1], 64)
		lng, err2 := strconv.ParseFloat(match[2], 64)
		if err1 == nil && err2 == nil {
			return MapLocation{Latitude: lat, Longitude: lng}, true
		}
	}

	if match := mapsQueryPattern.FindStringSubmatch(decoded); len(match) == 3 {
		lat, err1 := strconv.ParseFloat(match[1], 64)
		lng, err2 := strconv.ParseFloat(match[2], 64)
		if err1 == nil && err2 == nil {
			return MapLocation{Latitude: lat, Longitude: lng}, true
		}
	}

	return MapLocation{}, false
}

func ExpandShortURL(ctx context.Context, raw string) (string, error) {
	if raw == "" {
		return "", fmt.Errorf("empty url")
	}
	parsed, err := url.Parse(raw)
	if err != nil {
		return "", fmt.Errorf("parse url: %w", err)
	}
	if parsed.Scheme == "" {
		parsed.Scheme = "https"
	}

	client := &http.Client{
		Timeout: 2 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 3 {
				return http.ErrUseLastResponse
			}
			return nil
		},
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodHead, parsed.String(), nil)
	if err != nil {
		return "", fmt.Errorf("new request: %w", err)
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("request: %w", err)
	}
	defer resp.Body.Close()

	finalURL := resp.Request.URL.String()
	return finalURL, nil
}
