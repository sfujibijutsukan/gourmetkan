package services

import (
	"database/sql"
	"fmt"
)

type Review struct {
	ID           int
	RestaurantID int
	UserID       int
	Rating       int
	Comment      string
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
        SELECT id, restaurant_id, user_id, rating, comment, created_at
        FROM reviews
        WHERE restaurant_id = ?
        ORDER BY created_at DESC
        LIMIT ? OFFSET ?
    `, restaurantID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("list reviews: %w", err)
	}
	defer rows.Close()

	var reviews []Review
	for rows.Next() {
		var review Review
		if err := rows.Scan(&review.ID, &review.RestaurantID, &review.UserID, &review.Rating, &review.Comment, &review.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan review: %w", err)
		}
		reviews = append(reviews, review)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows review: %w", err)
	}
	return reviews, nil
}

func (s *ReviewService) CreateReview(review Review) error {
	_, err := s.db.Exec(`
        INSERT INTO reviews (restaurant_id, user_id, rating, comment)
        VALUES (?, ?, ?, ?)
    `, review.RestaurantID, review.UserID, review.Rating, review.Comment)
	if err != nil {
		return fmt.Errorf("create review: %w", err)
	}
	return nil
}
