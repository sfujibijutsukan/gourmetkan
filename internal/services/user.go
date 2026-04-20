package services

import (
	"database/sql"
	"fmt"
)

type User struct {
	ID        int
	GitHubID  string
	Username  string
	AvatarURL string
}

type UserService struct {
	db *sql.DB
}

func NewUserService(db *sql.DB) *UserService {
	return &UserService{db: db}
}

func (s *UserService) UpsertGitHubUser(githubID, username, avatarURL string) (User, error) {
	_, err := s.db.Exec(`
        INSERT INTO users (github_id, username, avatar_url)
        VALUES (?, ?, ?)
        ON CONFLICT(github_id) DO UPDATE SET username = excluded.username, avatar_url = excluded.avatar_url, updated_at = CURRENT_TIMESTAMP
    `, githubID, username, avatarURL)
	if err != nil {
		return User{}, fmt.Errorf("upsert user: %w", err)
	}

	var user User
	err = s.db.QueryRow("SELECT id, github_id, username, avatar_url FROM users WHERE github_id = ?", githubID).
		Scan(&user.ID, &user.GitHubID, &user.Username, &user.AvatarURL)
	if err != nil {
		return User{}, fmt.Errorf("fetch user: %w", err)
	}
	return user, nil
}

func (s *UserService) GetUserByID(id int) (*User, error) {
	var user User
	err := s.db.QueryRow("SELECT id, github_id, username, avatar_url FROM users WHERE id = ?", id).
		Scan(&user.ID, &user.GitHubID, &user.Username, &user.AvatarURL)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get user: %w", err)
	}
	return &user, nil
}
