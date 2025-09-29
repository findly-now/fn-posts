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

	"github.com/gin-gonic/gin"
	"github.com/jsarabia/fn-posts/internal"
	"github.com/jsarabia/fn-posts/internal/config"
	_ "github.com/lib/pq"
)

func main() {
	cfg := config.Load()

	// Initialize database connection with connection pooling
	db, err := sql.Open("postgres", cfg.PostgresURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Configure connection pool for optimal performance
	db.SetMaxOpenConns(25)                 // Maximum number of open connections
	db.SetMaxIdleConns(5)                  // Maximum number of idle connections
	db.SetConnMaxLifetime(5 * time.Minute) // Maximum connection lifetime

	// Test database connection
	if err := db.Ping(); err != nil {
		log.Fatalf("Failed to ping database: %v", err)
	}
	log.Println("Connected to database successfully with connection pooling")

	// Initialize application using Wire
	app, err := internal.InitializeApplication(db, cfg)
	if err != nil {
		log.Fatalf("Failed to initialize application: %v", err)
	}
	log.Println("Application initialized successfully with Wire dependency injection")

	// Setup router
	router := gin.Default()

	// Health check endpoint
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":  "healthy",
			"service": "posts-domain",
		})
	})

	// API routes using Wire-injected handlers
	api := router.Group("/api")

	// Posts routes
	posts := api.Group("/posts")
	{
		posts.POST("", app.PostHandler.CreatePost)
		posts.GET("", app.PostHandler.ListPosts)
		posts.GET("/nearby", app.PostHandler.SearchNearbyPosts)
		posts.GET("/:id", app.PostHandler.GetPost)
		posts.PUT("/:id", app.PostHandler.UpdatePost)
		posts.PATCH("/:id/status", app.PostHandler.UpdatePostStatus)
		posts.DELETE("/:id", app.PostHandler.DeletePost)

		// Photo routes (sub-resource of posts)
		posts.POST("/:postId/photos", app.PhotoHandler.UploadPhoto)
	}

	// Users routes
	users := api.Group("/users")
	{
		users.GET("/:userId/posts", app.PostHandler.GetUserPosts)
	}

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
