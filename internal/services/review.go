package services

import (
	"database/sql"
	"fmt"
)

type Review struct {
	ID           int
	RestaurantID int
	UserID       int
	Username     string
	Rating       int
	Comment      string
	PhotoPath    string
	PhotoPaths   []string
	CreatedAt    string
}

type ReviewService struct {
	db *sql.DB
}

func NewReviewService(db *sql.DB) *ReviewService {
	return &ReviewService{db: db}
}

func (s *ReviewService) ListReviews(restaurantID int, limit, offset int) ([]Review, error) {
	rows, err := s.db.Query(`
	SELECT reviews.id, reviews.restaurant_id, reviews.user_id, users.username, reviews.rating, reviews.comment, COALESCE(reviews.photo_path, ''), reviews.created_at
        FROM reviews
        JOIN users ON users.id = reviews.user_id
        WHERE reviews.restaurant_id = ?
        ORDER BY reviews.created_at DESC
        LIMIT ? OFFSET ?
    `, restaurantID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("list reviews: %w", err)
	}
	defer rows.Close()

	var reviews []Review
	for rows.Next() {
		var review Review
		if err := rows.Scan(&review.ID, &review.RestaurantID, &review.UserID, &review.Username, &review.Rating, &review.Comment, &review.PhotoPath, &review.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan review: %w", err)
		}
		reviews = append(reviews, review)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows review: %w", err)
	}
	return reviews, nil
}

func (s *ReviewService) CreateReview(review Review) (int, error) {
	result, err := s.db.Exec(`
		INSERT INTO reviews (restaurant_id, user_id, rating, comment, photo_path)
		VALUES (?, ?, ?, ?, ?)
	`, review.RestaurantID, review.UserID, review.Rating, review.Comment, review.PhotoPath)
	if err != nil {
		return 0, fmt.Errorf("create review: %w", err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("review id: %w", err)
	}
	return int(id), nil
}

func (s *ReviewService) GetReview(id int) (*Review, error) {
	var review Review
	err := s.db.QueryRow(`
		SELECT id, restaurant_id, user_id, rating, comment, COALESCE(photo_path, ''), created_at
		FROM reviews
		WHERE id = ?
	`, id).Scan(
		&review.ID,
		&review.RestaurantID,
		&review.UserID,
		&review.Rating,
		&review.Comment,
		&review.PhotoPath,
		&review.CreatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get review: %w", err)
	}
	return &review, nil
}

func (s *ReviewService) UpdateReview(review Review) error {
	result, err := s.db.Exec(`
		UPDATE reviews
		SET rating = ?, comment = ?, photo_path = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = ? AND user_id = ?
	`, review.Rating, review.Comment, review.PhotoPath, review.ID, review.UserID)
	if err != nil {
		return fmt.Errorf("update review: %w", err)
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

func (s *ReviewService) DeleteReview(id int, userID int) error {
	result, err := s.db.Exec(`
		DELETE FROM reviews
		WHERE id = ? AND user_id = ?
	`, id, userID)
	if err != nil {
		return fmt.Errorf("delete review: %w", err)
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

func (s *ReviewService) ListReviewPhotos(reviewID int) ([]string, error) {
	rows, err := s.db.Query(`
		SELECT path
		FROM review_photos
		WHERE review_id = ?
		ORDER BY sort_order ASC, id ASC
	`, reviewID)
	if err != nil {
		return nil, fmt.Errorf("list review photos: %w", err)
	}
	defer rows.Close()

	paths := make([]string, 0)
	for rows.Next() {
		var path string
		if err := rows.Scan(&path); err != nil {
			return nil, fmt.Errorf("scan review photo: %w", err)
		}
		paths = append(paths, path)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows review photo: %w", err)
	}
	return paths, nil
}

func (s *ReviewService) ReplaceReviewPhotos(reviewID int, photoPaths []string) error {
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	if _, err := tx.Exec("DELETE FROM review_photos WHERE review_id = ?", reviewID); err != nil {
		return fmt.Errorf("clear review photos: %w", err)
	}

	if len(photoPaths) > 0 {
		stmt, err := tx.Prepare("INSERT INTO review_photos (review_id, path, sort_order) VALUES (?, ?, ?)")
		if err != nil {
			return fmt.Errorf("prepare review photos: %w", err)
		}
		defer stmt.Close()

		for i, path := range photoPaths {
			if _, err := stmt.Exec(reviewID, path, i); err != nil {
				return fmt.Errorf("insert review photo: %w", err)
			}
		}
	}

	firstPath := ""
	if len(photoPaths) > 0 {
		firstPath = photoPaths[0]
	}
	if _, err := tx.Exec("UPDATE reviews SET photo_path = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?", firstPath, reviewID); err != nil {
		return fmt.Errorf("sync review photo_path: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit tx: %w", err)
	}
	return nil
}

func (s *ReviewService) AverageRating(restaurantID int) (float64, int, error) {
	var avg sql.NullFloat64
	var count int
	if err := s.db.QueryRow(`
        SELECT AVG(rating), COUNT(*)
        FROM reviews
        WHERE restaurant_id = ?
    `, restaurantID).Scan(&avg, &count); err != nil {
		return 0, 0, fmt.Errorf("avg rating: %w", err)
	}
	if !avg.Valid {
		return 0, 0, nil
	}
	return avg.Float64, count, nil
}
