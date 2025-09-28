package main

import (
	"context"
	"database/sql"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	_ "github.com/lib/pq"
	"github.com/jsarabia/fn-posts/internal/config"
	"github.com/jsarabia/fn-posts/internal/handler"
	"github.com/jsarabia/fn-posts/internal/repository"
	"github.com/jsarabia/fn-posts/internal/service"
)

func main() {
	cfg := config.Load()

	// Initialize database connection
	db, err := sql.Open("postgres", cfg.PostgresURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Test database connection
	if err := db.Ping(); err != nil {
		log.Fatalf("Failed to ping database: %v", err)
	}
	log.Println("Connected to database successfully")

	// Initialize repositories
	postRepo := repository.NewPostgresPostRepository(db)
	photoRepo := repository.NewPostgresPhotoRepository(db)

	// Initialize storage service (use test storage in test environment)
	var storageService interface {
		UploadPhoto(ctx context.Context, file multipart.File, header *multipart.FileHeader, postID uuid.UUID, organizationID *uuid.UUID) (*service.UploadResult, error)
		DeletePhoto(ctx context.Context, filename string) error
		GetPhotoURL(filename string) string
		GenerateThumbnail(ctx context.Context, originalURL string, postID uuid.UUID, organizationID *uuid.UUID) (string, error)
	}

	if cfg.Environment == "test" {
		storageService = service.NewTestStorageService(cfg.StorageConfig)
		log.Println("Using test storage service")
	} else {
		var err error
		storageService, err = service.NewStorageService(cfg.StorageConfig)
		if err != nil {
			log.Fatalf("Failed to initialize storage service: %v", err)
		}
		log.Println("Using GCS storage service")
	}

	// Initialize event service
	eventService, err := service.NewEventService(cfg.KafkaConfig)
	if err != nil {
		log.Fatalf("Failed to initialize event service: %v", err)
	}
	defer eventService.Close()

	// Initialize post service
	postService := service.NewPostService(postRepo, photoRepo, eventService)

	// Setup router
	router := gin.Default()

	// Health check endpoint
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":  "healthy",
			"service": "posts-domain",
		})
	})

	// API routes
	api := router.Group("/api/v1")
	handler.SetupRoutes(api, postService, storageService)

	srv := &http.Server{
		Addr:    ":" + cfg.Port,
		Handler: router,
	}

	// Start server in a goroutine
	go func() {
		log.Printf("Starting server on port %s", cfg.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatal("Server forced to shutdown:", err)
	}

	log.Println("Server exited")
}
