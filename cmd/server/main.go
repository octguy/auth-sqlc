package main

import (
	"context"
	"fmt"

	"log"

	"github.com/gin-gonic/gin"
	"github.com/octguy/auth-sqlc/api"
	"github.com/octguy/auth-sqlc/config"
	db "github.com/octguy/auth-sqlc/db/sqlc"
	"github.com/octguy/auth-sqlc/internal/database"
	"github.com/octguy/auth-sqlc/internal/handler"
	"github.com/octguy/auth-sqlc/internal/repository"
	"github.com/octguy/auth-sqlc/internal/service"
)

func main() {
	// 1. Load config from environment variables
	cfg := config.Load() // Load configuration from .env file
	ctx := context.Background()

	gin.SetMode(cfg.GinMode) // Set Gin mode based on configuration

	// 2. Connect to PostgreSQL
	pool, err := database.Connect(ctx, cfg.DSN())
	if err != nil {
		log.Fatalf("Database connection failed: %v", err)
	}
	defer pool.Close()
	log.Println("Connected to PostgreSQL")

	// 3. Wire: sqlc Queries → repository → service → handler (Dependency Injection)
	queries := db.New(pool) // sqlc-generated queries
	userRepo := repository.NewUserRepository(queries)
	authSvc := service.NewAuthService(userRepo, cfg.JWTSecret, cfg.TokenTTL)
	authHandler := handler.NewAuthHandler(authSvc)

	// 4. Set up Gin and register all routes
	r := gin.Default()
	api.RegisterRoutes(r, authHandler, authSvc)

	// 5. Start the server
	addr := fmt.Sprintf(":%s", cfg.Port)
	log.Printf("Server listening on %s (mode: %s)", addr, cfg.GinMode)

	if err := r.Run(addr); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
