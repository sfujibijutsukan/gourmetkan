package util

import (
	"fmt"
	"math"
)

const earthRadiusKm = 6371.0

func HaversineDistanceKm(lat1, lng1, lat2, lng2 float64) float64 {
	lat1Rad := toRadians(lat1)
	lat2Rad := toRadians(lat2)
	deltaLat := toRadians(lat2 - lat1)
	deltaLng := toRadians(lng2 - lng1)

	a := math.Sin(deltaLat/2)*math.Sin(deltaLat/2) +
		math.Cos(lat1Rad)*math.Cos(lat2Rad)*math.Sin(deltaLng/2)*math.Sin(deltaLng/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
	return earthRadiusKm * c
}

func toRadians(deg float64) float64 {
	return deg * math.Pi / 180
}

func FormatDistanceKm(km float64) string {
	if km < 1 {
		meters := int(math.Round(km * 1000))
		return fmt.Sprintf("%d m", meters)
	}
	return fmt.Sprintf("%.1f km", km)
}
