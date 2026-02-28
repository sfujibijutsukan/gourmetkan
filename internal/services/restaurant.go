package services

import (
	"database/sql"
	"fmt"
)

type Restaurant struct {
	ID          int
	Name        string
	Description string
	Latitude    float64
	Longitude   float64
	Address     string
	MapsURL     string
	CreatedBy   int
	CreatedAt   string
}

type RestaurantService struct {
	db *sql.DB
}

func NewRestaurantService(db *sql.DB) *RestaurantService {
	return &RestaurantService{db: db}
}

func (s *RestaurantService) ListRestaurants() ([]Restaurant, error) {
	rows, err := s.db.Query(`
        SELECT id, name, description, latitude, longitude, address, maps_url, created_by, created_at
        FROM restaurants
        ORDER BY created_at DESC
    `)
	if err != nil {
		return nil, fmt.Errorf("list restaurants: %w", err)
	}
	defer rows.Close()

	var restaurants []Restaurant
	for rows.Next() {
		var restaurant Restaurant
		if err := rows.Scan(
			&restaurant.ID,
			&restaurant.Name,
			&restaurant.Description,
			&restaurant.Latitude,
			&restaurant.Longitude,
			&restaurant.Address,
			&restaurant.MapsURL,
			&restaurant.CreatedBy,
			&restaurant.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan restaurant: %w", err)
		}
		restaurants = append(restaurants, restaurant)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows restaurant: %w", err)
	}
	return restaurants, nil
}

func (s *RestaurantService) GetRestaurant(id int) (*Restaurant, error) {
	var restaurant Restaurant
	err := s.db.QueryRow(`
        SELECT id, name, description, latitude, longitude, address, maps_url, created_by, created_at
        FROM restaurants
        WHERE id = ?
    `, id).Scan(
		&restaurant.ID,
		&restaurant.Name,
		&restaurant.Description,
		&restaurant.Latitude,
		&restaurant.Longitude,
		&restaurant.Address,
		&restaurant.MapsURL,
		&restaurant.CreatedBy,
		&restaurant.CreatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get restaurant: %w", err)
	}
	return &restaurant, nil
}

func (s *RestaurantService) CreateRestaurant(input Restaurant) (int, error) {
	result, err := s.db.Exec(`
        INSERT INTO restaurants (name, description, latitude, longitude, address, maps_url, created_by)
        VALUES (?, ?, ?, ?, ?, ?, ?)
    `, input.Name, input.Description, input.Latitude, input.Longitude, input.Address, input.MapsURL, input.CreatedBy)
	if err != nil {
		return 0, fmt.Errorf("create restaurant: %w", err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("last insert id: %w", err)
	}
	return int(id), nil
}
