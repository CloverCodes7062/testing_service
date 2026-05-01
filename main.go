package main

import (
	"context"
	"log"
	"os"

	"github.com/labstack/echo/v4"
	echoMiddleware "github.com/labstack/echo/v4/middleware"

	"testing_service/internal/auth"
	"testing_service/internal/db"
	"testing_service/internal/handlers"
	customMiddleware "testing_service/internal/middleware"
)

func main() {
	// Init DB (non-fatal if DATABASE_URL not set — allows running without DB for M1/M2)
	if dbURL := os.Getenv("DATABASE_URL"); dbURL != "" {
		if err := db.InitDB(context.Background(), dbURL); err != nil {
			log.Printf("Warning: DB init failed: %v", err)
		}
	}

	e := echo.New()
	e.HideBanner = true

	// Global middleware
	e.Use(echoMiddleware.Logger())
	e.Use(echoMiddleware.Recover())
	e.Use(customMiddleware.SecurityHeaders())
	e.Use(customMiddleware.CacheNoStore())
	e.Use(customMiddleware.CustomCORS())
	e.Use(customMiddleware.GlobalRateLimit())

	// Public routes
	e.GET("/health", handlers.HealthHandler)
	e.GET("/weatherData", handlers.WeatherHandler)

	// Auth routes (with cache no-store)
	authGroup := e.Group("/auth")
	authGroup.Use(customMiddleware.CacheNoStore())
	authGroup.POST("/register", handlers.Register)
	authGroup.POST("/login", handlers.Login)
	authGroup.POST("/refresh", handlers.Refresh)
	authGroup.POST("/reset-password/request", handlers.ResetPasswordRequest)
	authGroup.POST("/reset-password/confirm", handlers.ResetPasswordConfirm)

	// User routes
	e.GET("/users", handlers.ListUsers, auth.RequireAuth("admin"), customMiddleware.CacheNoStore())
	e.POST("/users", handlers.CreateUser, auth.RequireAuth("admin"), customMiddleware.CacheNoStore())

	// Incident routes
	e.GET("/incidents", handlers.ListIncidents, auth.RequireAuth("member"), customMiddleware.CacheNoStore())
	e.GET("/incidents/stats", handlers.IncidentStats, auth.RequireAuth("member"), customMiddleware.CacheNoStore())
	e.GET("/incidents/:id", handlers.GetIncident, auth.RequireAuth("member"), customMiddleware.CacheNoStore())
	e.POST("/incidents", handlers.CreateIncident, auth.RequireAuth("manager"), customMiddleware.CacheNoStore())

	// Admin routes
	e.GET("/admin/config", handlers.AdminConfig, auth.RequireAuth("admin"), customMiddleware.CacheNoStore())

	// Team routes
	e.POST("/teams", handlers.CreateTeam, auth.RequireAuth("manager"), customMiddleware.CacheNoStore())

	port := os.Getenv("PORT")
	if port == "" {
		port = "3000"
	}
	e.Logger.Fatal(e.Start(":" + port))
}
