package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"deployment-controller/internal/config"
	"deployment-controller/internal/database"
	"deployment-controller/internal/handlers"

	"github.com/gin-gonic/gin"
)

func main() {
	// Setup logger
	logger := setupLogger()

	// Load configuration
	cfg, err := config.Load("")
	if err != nil {
		logger.Error("Failed to load configuration", "error", err)
		os.Exit(1)
	}

	// Set Gin mode based on log level
	if cfg.Server.LogLevel == "debug" {
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}

	// Initialize database
	db, err := database.New(cfg)
	if err != nil {
		logger.Error("Failed to initialize database", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	logger.Info("Database connection established", "max_conns", cfg.Database.MaxConns)

	// Initialize handlers
	h := handlers.New(db, logger)

	// Setup router
	router := setupRouter(h, cfg, logger)

	// Create HTTP server
	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Server.Port),
		Handler:      router,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in a goroutine
	go func() {
		logger.Info("Starting server", "port", cfg.Server.Port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("Failed to start server", "error", err)
			os.Exit(1)
		}
	}()

	logger.Info("Deployment Controller started successfully", "port", cfg.Server.Port)

	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down server...")

	// The context is used to inform the server it has 30 seconds to finish
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		logger.Error("Server forced to shutdown", "error", err)
		os.Exit(1)
	}

	logger.Info("Server exited")
}

func setupLogger() *slog.Logger {
	// Create JSON logger for production
	opts := &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}

	handler := slog.NewJSONHandler(os.Stdout, opts)
	logger := slog.New(handler)
	slog.SetDefault(logger)

	return logger
}

func setupRouter(h *handlers.Handler, cfg *config.Config, logger *slog.Logger) *gin.Engine {
	router := gin.New()

	// Middleware
	router.Use(gin.Recovery())
	router.Use(requestLoggingMiddleware(logger))

	// Optional bearer token authentication
	if cfg.Security.BearerToken != "" {
		router.Use(authMiddleware(cfg.Security.BearerToken, logger))
	}

	// CORS middleware
	router.Use(corsMiddleware())

	// Health check endpoint (no auth required)
	router.GET("/healthz", h.HealthCheck)

	// API routes
	v1 := router.Group("/api/v1")
	{
		// Deployment endpoints
		v1.POST("/push", h.Push)
		v1.GET("/deployments", h.GetDeployments)
		v1.GET("/deployments/:id", h.GetDeployment)
		v1.PATCH("/deployments/:id/status", h.UpdateDeploymentStatus)

		// Registry endpoints
		v1.POST("/registry", h.StoreRegistryCredential)
		v1.GET("/registry", h.GetRegistryCredential)

		// Stats endpoint
		v1.GET("/stats", h.GetStats)
	}

	return router
}

func requestLoggingMiddleware(logger *slog.Logger) gin.HandlerFunc {
	return gin.LoggerWithConfig(gin.LoggerConfig{
		Formatter: func(param gin.LogFormatterParams) string {
			logger.Info("HTTP Request",
				"method", param.Method,
				"path", param.Path,
				"status", param.StatusCode,
				"latency", param.Latency,
				"ip", param.ClientIP,
				"user_agent", param.Request.UserAgent(),
			)
			return ""
		},
		Output: os.Stdout,
	})
}

func authMiddleware(bearerToken string, logger *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Skip auth for health check
		if c.Request.URL.Path == "/healthz" {
			c.Next()
			return
		}

		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			logger.Warn("Missing Authorization header", "path", c.Request.URL.Path)
			c.JSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"error":   "Authorization header required",
			})
			c.Abort()
			return
		}

		if !strings.HasPrefix(authHeader, "Bearer ") {
			logger.Warn("Invalid Authorization header format", "path", c.Request.URL.Path)
			c.JSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"error":   "Invalid Authorization header format",
			})
			c.Abort()
			return
		}

		token := strings.TrimPrefix(authHeader, "Bearer ")
		if token != bearerToken {
			logger.Warn("Invalid bearer token", "path", c.Request.URL.Path)
			c.JSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"error":   "Invalid bearer token",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

func corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PATCH, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}
