package main

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/octguy/auth-sqlc/config"

	"log"
)

func main() {
	cfg := config.Load() // Load configuration from .env file

	gin.SetMode(cfg.GinMode) // Set Gin mode based on configuration

	r := gin.Default() // Create a new Gin router

	addr := fmt.Sprintf(":%s", cfg.Port)
	log.Printf("Server starting on %s (mode: %s)\n", addr, cfg.GinMode)

	if err := r.Run(addr); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
