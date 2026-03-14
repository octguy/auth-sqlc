package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/octguy/auth-sqlc/internal/handler"
	"github.com/octguy/auth-sqlc/internal/middleware"
	"github.com/octguy/auth-sqlc/internal/service"
)

// RegisterRoutes mounts all routes onto the Gin engine. (register the routes to the engine not feature Register)
func RegisterRoutes(r *gin.Engine, authHandler *handler.AuthHandler, authSvc service.AuthService) {
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	v1 := r.Group("/api/v1")

	// Public routes — no auth required
	auth := v1.Group("/auth")
	{
		auth.POST("/register", authHandler.Register)
		auth.POST("/login", authHandler.Login)
	}

	// Protected routes — valid JWT required
	protected := v1.Group("/auth")
	protected.Use(middleware.JWTAuth(authSvc))
	{
		protected.GET("/profile", authHandler.GetProfile)
	}
}
