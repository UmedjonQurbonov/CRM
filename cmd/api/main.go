package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "github.com/UmedjonQurbonov/CRM/docs"
	"github.com/UmedjonQurbonov/CRM/internal/config"
	"github.com/UmedjonQurbonov/CRM/internal/platform/database"
	"github.com/UmedjonQurbonov/CRM/internal/platform/redis"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	httpSwagger "github.com/swaggo/http-swagger/v2"
)

// @title Retail POS & Inventory API
// @version 1.0
// @description High-performance retail POS and inventory management API with RBAC, atomic checkout, and analytics.
// @termsOfService http://swagger.io/terms/

// @contact.name API Support
// @contact.email support@example.com

// @license.name Apache 2.0
// @license.url http://www.apache.org/licenses/LICENSE-2.0.html

// @host localhost:8080
// @BasePath /
// @schemes http

// HealthResponse represents the health check response body.
type HealthResponse struct {
	Status    string    `json:"status" example:"ok"`
	Database  string    `json:"database" example:"connected"`
	Redis     string    `json:"redis" example:"connected"`
	Timestamp time.Time `json:"timestamp"`
}

func main() {
	// 1. Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("failed to load configuration: %v", err)
	}

	log.Printf("Starting CRM API server in %s mode...", cfg.Server.Env)

	// 2. Initialize PostgreSQL connection pool
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	dbPool, err := database.NewPostgresPool(ctx, cfg.Database)
	if err != nil {
		log.Fatalf("failed to connect to PostgreSQL: %v", err)
	}
	defer dbPool.Close()
	log.Printf("Connected to PostgreSQL on %s:%s/%s", cfg.Database.Host, cfg.Database.Port, cfg.Database.Name)

	// 3. Run database migrations
	log.Println("Executing database migrations...")
	if err := database.RunMigrations(dbPool); err != nil {
		log.Fatalf("failed to execute migrations: %v", err)
	}
	log.Println("Database migrations completed successfully.")

	// 4. Initialize Redis client (optional check for health report)
	redisStatus := "connected"
	redisClient, err := redis.NewClient(ctx, cfg.Redis)
	if err != nil {
		log.Printf("Warning: Redis connection failed (%v). Continuing startup...", err)
		redisStatus = "unavailable"
	} else {
		defer redisClient.Close()
		log.Printf("Connected to Redis on %s", cfg.Redis.Addr)
	}

	// 5. Setup Chi Router & Middleware
	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(60 * time.Second))

	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS", "PATCH"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: false,
		MaxAge:           300,
	}))

	// Healthcheck endpoint
	r.Get("/health", handleHealthCheck(dbPool, redisStatus))

	// Swagger documentation endpoint
	r.Get("/swagger/*", httpSwagger.WrapHandler)

	// 6. HTTP Server Setup & Graceful Shutdown
	srv := &http.Server{
		Addr:         ":" + cfg.Server.Port,
		Handler:      r,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
		IdleTimeout:  cfg.Server.IdleTimeout,
	}

	shutdownChan := make(chan os.Signal, 1)
	signal.Notify(shutdownChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		log.Printf("CRM API server listening on port :%s", cfg.Server.Port)
		log.Printf("Swagger UI available at http://localhost:%s/swagger/index.html", cfg.Server.Port)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("HTTP server error: %v", err)
		}
	}()

	// Wait for termination signal
	sig := <-shutdownChan
	log.Printf("Received signal %s. Initiating graceful shutdown...", sig)

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server gracefully stopped.")
}

// @Summary Health Check
// @Description Returns the operational status of the service and database
// @Tags System
// @Produce json
// @Success 200 {object} HealthResponse
// @Failure 503 {object} HealthResponse
// @Router /health [get]
func handleHealthCheck(dbPool interface {
	Ping(ctx context.Context) error
}, redisStatus string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		dbStatus := "connected"
		statusCode := http.StatusOK

		if err := dbPool.Ping(r.Context()); err != nil {
			dbStatus = fmt.Sprintf("unreachable: %v", err)
			statusCode = http.StatusServiceUnavailable
		}

		resp := HealthResponse{
			Status:    "ok",
			Database:  dbStatus,
			Redis:     redisStatus,
			Timestamp: time.Now().UTC(),
		}

		if statusCode != http.StatusOK {
			resp.Status = "degraded"
		}

		w.WriteHeader(statusCode)
		_ = json.NewEncoder(w).Encode(resp)
	}
}
