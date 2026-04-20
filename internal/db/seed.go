package db

import (
	"database/sql"
	"fmt"
)

type baseSeed struct {
	Name      string
	Latitude  float64
	Longitude float64
}

var defaultBases = []baseSeed{
	{Name: "立命館大学 OIC（大阪いばらきキャンパス）", Latitude: 34.810888, Longitude: 135.561172},
	{Name: "立命館大学 BKC（びわこ・くさつキャンパス）", Latitude: 34.982189, Longitude: 135.96272},
	{Name: "立命館大学 衣笠キャンパス（KIC）", Latitude: 35.0325428, Longitude: 135.7240146},
}

func EnsureBaseSeed(db *sql.DB) error {
	var count int
	if err := db.QueryRow("SELECT COUNT(*) FROM bases").Scan(&count); err != nil {
		return fmt.Errorf("count bases: %w", err)
	}
	if count > 0 {
		return nil
	}

	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("begin seed: %w", err)
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare("INSERT INTO bases (name, latitude, longitude) VALUES (?, ?, ?)")
	if err != nil {
		return fmt.Errorf("prepare seed: %w", err)
	}
	defer stmt.Close()

	for _, base := range defaultBases {
		if _, err := stmt.Exec(base.Name, base.Latitude, base.Longitude); err != nil {
			return fmt.Errorf("seed base %s: %w", base.Name, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit seed: %w", err)
	}
	return nil
}
