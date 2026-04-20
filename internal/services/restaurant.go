package services

import (
	"database/sql"
	"fmt"
	"strings"
)

type Restaurant struct {
	ID          int
	Name        string
	Description string
	PhotoPath   string
	PhotoPaths  []string
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

type Tag struct {
	ID   int
	Name string
}

func NewRestaurantService(db *sql.DB) *RestaurantService {
	return &RestaurantService{db: db}
}

func (s *RestaurantService) ListRestaurants() ([]Restaurant, error) {
	rows, err := s.db.Query(`
	SELECT id, name, description, COALESCE(photo_path, ''), latitude, longitude, address, maps_url, created_by, created_at
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
			&restaurant.PhotoPath,
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
	SELECT id, name, description, COALESCE(photo_path, ''), latitude, longitude, address, maps_url, created_by, created_at
        FROM restaurants
        WHERE id = ?
    `, id).Scan(
		&restaurant.ID,
		&restaurant.Name,
		&restaurant.Description,
		&restaurant.PhotoPath,
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
		INSERT INTO restaurants (name, description, photo_path, latitude, longitude, address, maps_url, created_by)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, input.Name, input.Description, input.PhotoPath, input.Latitude, input.Longitude, input.Address, input.MapsURL, input.CreatedBy)
	if err != nil {
		return 0, fmt.Errorf("create restaurant: %w", err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("last insert id: %w", err)
	}
	return int(id), nil
}

func (s *RestaurantService) UpdateRestaurant(input Restaurant) error {
	result, err := s.db.Exec(`
        UPDATE restaurants
		SET name = ?, description = ?, photo_path = ?, latitude = ?, longitude = ?, address = ?, maps_url = ?, updated_at = CURRENT_TIMESTAMP
        WHERE id = ?
	`, input.Name, input.Description, input.PhotoPath, input.Latitude, input.Longitude, input.Address, input.MapsURL, input.ID)
	if err != nil {
		return fmt.Errorf("update restaurant: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("rows affected: %w", err)
	}
	if affected == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (s *RestaurantService) DeleteRestaurant(id int, createdBy int) error {
	result, err := s.db.Exec(`
		DELETE FROM restaurants
		WHERE id = ? AND created_by = ?
	`, id, createdBy)
	if err != nil {
		return fmt.Errorf("delete restaurant: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("rows affected: %w", err)
	}
	if affected == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (s *RestaurantService) ListRestaurantPhotos(restaurantID int) ([]string, error) {
	rows, err := s.db.Query(`
		SELECT path
		FROM restaurant_photos
		WHERE restaurant_id = ?
		ORDER BY sort_order ASC, id ASC
	`, restaurantID)
	if err != nil {
		return nil, fmt.Errorf("list restaurant photos: %w", err)
	}
	defer rows.Close()

	paths := make([]string, 0)
	for rows.Next() {
		var path string
		if err := rows.Scan(&path); err != nil {
			return nil, fmt.Errorf("scan restaurant photo: %w", err)
		}
		paths = append(paths, path)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows restaurant photo: %w", err)
	}
	return paths, nil
}

func (s *RestaurantService) ReplaceRestaurantPhotos(restaurantID int, photoPaths []string) error {
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	if _, err := tx.Exec("DELETE FROM restaurant_photos WHERE restaurant_id = ?", restaurantID); err != nil {
		return fmt.Errorf("clear restaurant photos: %w", err)
	}

	if len(photoPaths) > 0 {
		stmt, err := tx.Prepare("INSERT INTO restaurant_photos (restaurant_id, path, sort_order) VALUES (?, ?, ?)")
		if err != nil {
			return fmt.Errorf("prepare restaurant photos: %w", err)
		}
		defer stmt.Close()

		for i, path := range photoPaths {
			if _, err := stmt.Exec(restaurantID, path, i); err != nil {
				return fmt.Errorf("insert restaurant photo: %w", err)
			}
		}
	}

	firstPath := ""
	if len(photoPaths) > 0 {
		firstPath = photoPaths[0]
	}
	if _, err := tx.Exec("UPDATE restaurants SET photo_path = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?", firstPath, restaurantID); err != nil {
		return fmt.Errorf("sync restaurant photo_path: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit tx: %w", err)
	}
	return nil
}

func (s *RestaurantService) UpsertTag(name string) (Tag, error) {
	var tag Tag
	err := s.db.QueryRow("SELECT id, name FROM tags WHERE name = ?", name).Scan(&tag.ID, &tag.Name)
	if err == nil {
		return tag, nil
	}
	if err != sql.ErrNoRows {
		return Tag{}, fmt.Errorf("get tag: %w", err)
	}
	result, err := s.db.Exec("INSERT INTO tags (name) VALUES (?)", name)
	if err != nil {
		return Tag{}, fmt.Errorf("create tag: %w", err)
	}
	createdID, err := result.LastInsertId()
	if err != nil {
		return Tag{}, fmt.Errorf("tag id: %w", err)
	}
	return Tag{ID: int(createdID), Name: name}, nil
}

func (s *RestaurantService) AttachTags(restaurantID int, tagIDs []int) error {
	if len(tagIDs) == 0 {
		return nil
	}
	stmt, err := s.db.Prepare("INSERT OR IGNORE INTO restaurant_tags (restaurant_id, tag_id) VALUES (?, ?)")
	if err != nil {
		return fmt.Errorf("prepare tags: %w", err)
	}
	defer stmt.Close()
	for _, tagID := range tagIDs {
		if _, err := stmt.Exec(restaurantID, tagID); err != nil {
			return fmt.Errorf("attach tag: %w", err)
		}
	}
	return nil
}

func (s *RestaurantService) ReplaceTags(restaurantID int, tagIDs []int) error {
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	if _, err := tx.Exec("DELETE FROM restaurant_tags WHERE restaurant_id = ?", restaurantID); err != nil {
		return fmt.Errorf("clear tags: %w", err)
	}

	if len(tagIDs) > 0 {
		stmt, err := tx.Prepare("INSERT INTO restaurant_tags (restaurant_id, tag_id) VALUES (?, ?)")
		if err != nil {
			return fmt.Errorf("prepare tags: %w", err)
		}
		defer stmt.Close()

		for _, tagID := range tagIDs {
			if _, err := stmt.Exec(restaurantID, tagID); err != nil {
				return fmt.Errorf("replace tag: %w", err)
			}
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit tx: %w", err)
	}
	return nil
}

func (s *RestaurantService) TagsForRestaurant(restaurantID int) ([]Tag, error) {
	rows, err := s.db.Query(`
        SELECT t.id, t.name
        FROM tags t
        INNER JOIN restaurant_tags rt ON rt.tag_id = t.id
        WHERE rt.restaurant_id = ?
        ORDER BY t.name ASC
    `, restaurantID)
	if err != nil {
		return nil, fmt.Errorf("list tags: %w", err)
	}
	defer rows.Close()

	var tags []Tag
	for rows.Next() {
		var tag Tag
		if err := rows.Scan(&tag.ID, &tag.Name); err != nil {
			return nil, fmt.Errorf("scan tag: %w", err)
		}
		tags = append(tags, tag)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows tag: %w", err)
	}
	return tags, nil
}

func (s *RestaurantService) TagsForRestaurants(restaurants []Restaurant) (map[int][]string, error) {
	ids := make([]int, 0, len(restaurants))
	for _, rest := range restaurants {
		ids = append(ids, rest.ID)
	}
	result := make(map[int][]string)
	if len(ids) == 0 {
		return result, nil
	}
	placeholders := make([]string, 0, len(ids))
	args := make([]interface{}, 0, len(ids))
	for _, id := range ids {
		placeholders = append(placeholders, "?")
		args = append(args, id)
	}
	query := "SELECT rt.restaurant_id, t.name FROM restaurant_tags rt INNER JOIN tags t ON t.id = rt.tag_id WHERE rt.restaurant_id IN (" + strings.Join(placeholders, ",") + ") ORDER BY t.name ASC"
	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("list tag map: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var restaurantID int
		var name string
		if err := rows.Scan(&restaurantID, &name); err != nil {
			return nil, fmt.Errorf("scan tag map: %w", err)
		}
		result[restaurantID] = append(result[restaurantID], name)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows tag map: %w", err)
	}
	return result, nil
}

func (s *RestaurantService) ListTags() ([]Tag, error) {
	rows, err := s.db.Query("SELECT id, name FROM tags ORDER BY name ASC")
	if err != nil {
		return nil, fmt.Errorf("list tags: %w", err)
	}
	defer rows.Close()

	var tags []Tag
	for rows.Next() {
		var tag Tag
		if err := rows.Scan(&tag.ID, &tag.Name); err != nil {
			return nil, fmt.Errorf("scan tag: %w", err)
		}
		tags = append(tags, tag)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows tag: %w", err)
	}
	return tags, nil
}

func (s *RestaurantService) ListRestaurantsByTag(tagName string) ([]Restaurant, error) {
	rows, err := s.db.Query(`
	SELECT r.id, r.name, r.description, COALESCE(r.photo_path, ''), r.latitude, r.longitude, r.address, r.maps_url, r.created_by, r.created_at
        FROM restaurants r
        INNER JOIN restaurant_tags rt ON rt.restaurant_id = r.id
        INNER JOIN tags t ON t.id = rt.tag_id
        WHERE t.name = ?
        ORDER BY r.created_at DESC
    `, tagName)
	if err != nil {
		return nil, fmt.Errorf("list restaurants by tag: %w", err)
	}
	defer rows.Close()

	var restaurants []Restaurant
	for rows.Next() {
		var restaurant Restaurant
		if err := rows.Scan(
			&restaurant.ID,
			&restaurant.Name,
			&restaurant.Description,
			&restaurant.PhotoPath,
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
