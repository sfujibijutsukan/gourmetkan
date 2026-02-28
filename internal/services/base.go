package services

import (
	"database/sql"
	"fmt"
)

type Base struct {
	ID        int
	Name      string
	Latitude  float64
	Longitude float64
}

type BaseService struct {
	db *sql.DB
}

func NewBaseService(db *sql.DB) *BaseService {
	return &BaseService{db: db}
}

func (s *BaseService) ListBases() ([]Base, error) {
	rows, err := s.db.Query("SELECT id, name, latitude, longitude FROM bases ORDER BY id ASC")
	if err != nil {
		return nil, fmt.Errorf("list bases: %w", err)
	}
	defer rows.Close()

	var bases []Base
	for rows.Next() {
		var base Base
		if err := rows.Scan(&base.ID, &base.Name, &base.Latitude, &base.Longitude); err != nil {
			return nil, fmt.Errorf("scan base: %w", err)
		}
		bases = append(bases, base)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows base: %w", err)
	}
	return bases, nil
}

func (s *BaseService) GetBaseByID(id int) (*Base, error) {
	var base Base
	err := s.db.QueryRow("SELECT id, name, latitude, longitude FROM bases WHERE id = ?", id).
		Scan(&base.ID, &base.Name, &base.Latitude, &base.Longitude)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get base: %w", err)
	}
	return &base, nil
}
