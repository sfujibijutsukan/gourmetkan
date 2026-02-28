package handlers

import (
	"database/sql"
	"html/template"
	"net/http"
	"time"

	"example.com/gourmetkan/internal/auth"
	"example.com/gourmetkan/internal/services"
)

type Config struct {
	BaseURL      string
	CookieSecure bool
	SessionTTL   time.Duration
}

type Router struct {
	mux *http.ServeMux
}

func NewRouter(cfg Config, authService *auth.Service, baseService *services.BaseService, restaurantService *services.RestaurantService, reviewService *services.ReviewService, userService *services.UserService, db *sql.DB) http.Handler {
	r := &Router{mux: http.NewServeMux()}
	handlers := &Handler{
		cfg:               cfg,
		authService:       authService,
		baseService:       baseService,
		restaurantService: restaurantService,
		reviewService:     reviewService,
		userService:       userService,
		db:                db,
	}
	r.mux.HandleFunc("/", handlers.Index)
	r.mux.HandleFunc("/auth/github/login", handlers.GitHubLogin)
	r.mux.HandleFunc("/auth/github/callback", handlers.GitHubCallback)
	r.mux.HandleFunc("/auth/logout", handlers.Logout)
	r.mux.HandleFunc("/bases/select", handlers.SelectBase)
	r.mux.HandleFunc("/restaurants/new", handlers.NewRestaurant)
	r.mux.HandleFunc("/restaurants", handlers.CreateRestaurant)
	r.mux.HandleFunc("/restaurants/", handlers.RestaurantRouter)
	r.mux.HandleFunc("/random", handlers.RandomRestaurant)
	r.mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("./static"))))
	return r.mux
}

type Handler struct {
	cfg               Config
	authService       *auth.Service
	baseService       *services.BaseService
	restaurantService *services.RestaurantService
	reviewService     *services.ReviewService
	userService       *services.UserService
	db                *sql.DB
	templates         map[string]*template.Template
}
