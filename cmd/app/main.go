package main

import (
	"context"
	"database/sql"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "github.com/mattn/go-sqlite3"

	"example.com/gourmetkan/internal/auth"
	"example.com/gourmetkan/internal/db"
	"example.com/gourmetkan/internal/handlers"
	"example.com/gourmetkan/internal/services"
)

func main() {
	cfg, err := loadConfig()
	if err != nil {
		log.Fatalf("config error: %v", err)
	}

	database, err := sql.Open("sqlite3", cfg.DatabasePath)
	if err != nil {
		log.Fatalf("db open: %v", err)
	}
	defer database.Close()

	if err := db.EnsureSchema(database); err != nil {
		log.Fatalf("schema: %v", err)
	}
	if err := db.EnsureBaseSeed(database); err != nil {
		log.Fatalf("seed: %v", err)
	}

	authService := auth.NewService(auth.Config{
		BaseURL:            cfg.BaseURL,
		GitHubClientID:     cfg.GitHubClientID,
		GitHubClientSecret: cfg.GitHubClientSecret,
		CookieSecure:       cfg.CookieSecure,
		SessionTTL:         cfg.SessionTTL,
	})
	baseService := services.NewBaseService(database)
	restaurantService := services.NewRestaurantService(database)
	reviewService := services.NewReviewService(database)
	userService := services.NewUserService(database)

	router := handlers.NewRouter(
		handlers.Config{
			BaseURL:      cfg.BaseURL,
			CookieSecure: cfg.CookieSecure,
			SessionTTL:   cfg.SessionTTL,
		},
		authService,
		baseService,
		restaurantService,
		reviewService,
		userService,
		database,
	)

	server := &http.Server{
		Addr:              cfg.ListenAddr,
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		log.Printf("listening on %s", cfg.ListenAddr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server: %v", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		log.Printf("shutdown error: %v", err)
	}
}
