package auth

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"example.com/gourmetkan/internal/util"
)

type Config struct {
	BaseURL            string
	GitHubClientID     string
	GitHubClientSecret string
	CookieSecure       bool
	SessionTTL         time.Duration
}

type Service struct {
	config Config
}

func NewService(cfg Config) *Service {
	return &Service{config: cfg}
}

type GitHubUser struct {
	ID        int    `json:"id"`
	Login     string `json:"login"`
	AvatarURL string `json:"avatar_url"`
}

type TokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	Scope       string `json:"scope"`
}

func (s *Service) BuildLoginURL(state string) string {
	values := url.Values{}
	values.Set("client_id", s.config.GitHubClientID)
	values.Set("redirect_uri", s.config.BaseURL+"/auth/github/callback")
	values.Set("state", state)
	values.Set("scope", "read:user")
	return "https://github.com/login/oauth/authorize?" + values.Encode()
}

func (s *Service) ExchangeCode(ctx context.Context, code string) (TokenResponse, error) {
	values := url.Values{}
	values.Set("client_id", s.config.GitHubClientID)
	values.Set("client_secret", s.config.GitHubClientSecret)
	values.Set("code", code)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://github.com/login/oauth/access_token", strings.NewReader(values.Encode()))
	if err != nil {
		return TokenResponse{}, fmt.Errorf("new request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return TokenResponse{}, fmt.Errorf("request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return TokenResponse{}, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return TokenResponse{}, fmt.Errorf("token exchange failed: %s", string(body))
	}

	var token TokenResponse
	if err := json.Unmarshal(body, &token); err != nil {
		return TokenResponse{}, fmt.Errorf("decode token: %w", err)
	}
	if token.AccessToken == "" {
		return TokenResponse{}, errors.New("missing access token")
	}
	return token, nil
}

func (s *Service) FetchGitHubUser(ctx context.Context, token string) (GitHubUser, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.github.com/user", nil)
	if err != nil {
		return GitHubUser{}, fmt.Errorf("new request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/json")

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return GitHubUser{}, fmt.Errorf("request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return GitHubUser{}, fmt.Errorf("github user failed: %s", string(body))
	}

	var user GitHubUser
	if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
		return GitHubUser{}, fmt.Errorf("decode user: %w", err)
	}
	return user, nil
}

func CreateSession(db *sql.DB, userID int, ttl time.Duration) (string, string, time.Time, error) {
	sessionID, err := util.RandomToken(32)
	if err != nil {
		return "", "", time.Time{}, err
	}
	csrfToken, err := util.RandomToken(32)
	if err != nil {
		return "", "", time.Time{}, err
	}
	expiresAt := time.Now().Add(ttl)
	_, err = db.Exec("INSERT INTO sessions (id, user_id, csrf_token, expires_at) VALUES (?, ?, ?, ?)", sessionID, userID, csrfToken, expiresAt)
	if err != nil {
		return "", "", time.Time{}, fmt.Errorf("insert session: %w", err)
	}
	return sessionID, csrfToken, expiresAt, nil
}

func GetSession(db *sql.DB, sessionID string) (int, string, time.Time, error) {
	var userID int
	var csrfToken string
	var expiresAt time.Time
	err := db.QueryRow("SELECT user_id, csrf_token, expires_at FROM sessions WHERE id = ?", sessionID).Scan(&userID, &csrfToken, &expiresAt)
	if err == sql.ErrNoRows {
		return 0, "", time.Time{}, nil
	}
	if err != nil {
		return 0, "", time.Time{}, fmt.Errorf("get session: %w", err)
	}
	return userID, csrfToken, expiresAt, nil
}

func DeleteSession(db *sql.DB, sessionID string) error {
	_, err := db.Exec("DELETE FROM sessions WHERE id = ?", sessionID)
	if err != nil {
		return fmt.Errorf("delete session: %w", err)
	}
	return nil
}

func StoreOAuthState(db *sql.DB, state string, ttl time.Duration) error {
	expiresAt := time.Now().Add(ttl)
	_, err := db.Exec("INSERT INTO oauth_states (state, expires_at) VALUES (?, ?)", state, expiresAt)
	if err != nil {
		return fmt.Errorf("insert state: %w", err)
	}
	return nil
}

func ConsumeOAuthState(db *sql.DB, state string) (bool, error) {
	result, err := db.Exec("DELETE FROM oauth_states WHERE state = ? AND expires_at > CURRENT_TIMESTAMP", state)
	if err != nil {
		return false, fmt.Errorf("consume state: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("rows affected: %w", err)
	}
	return rows > 0, nil
}
