package handler

import (
	"context"
	"mime/multipart"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jsarabia/fn-posts/internal/service"
)

// StorageInterface defines the storage service interface
type StorageInterface interface {
	UploadPhoto(ctx context.Context, file multipart.File, header *multipart.FileHeader, postID uuid.UUID, organizationID *uuid.UUID) (*service.UploadResult, error)
	DeletePhoto(ctx context.Context, filename string) error
	GetPhotoURL(filename string) string
	GenerateThumbnail(ctx context.Context, originalURL string, postID uuid.UUID, organizationID *uuid.UUID) (string, error)
}

func SetupRoutes(router *gin.RouterGroup, postService *service.PostService, storageService StorageInterface) {
	// Initialize handlers
	postHandler := NewPostHandler(postService)
	photoHandler := NewPhotoHandler(postService, storageService)

	// Posts routes
	posts := router.Group("/posts")
	{
		posts.POST("", postHandler.CreatePost)
		posts.GET("", postHandler.ListPosts)
		posts.GET("/nearby", postHandler.SearchNearbyPosts)
		posts.GET("/:id", postHandler.GetPost)
		posts.PUT("/:id", postHandler.UpdatePost)
		posts.PATCH("/:id/status", postHandler.UpdatePostStatus)
		posts.DELETE("/:id", postHandler.DeletePost)

		// Photo routes (sub-resource of posts)
		posts.POST("/:postId/photos", photoHandler.UploadPhoto)
		// posts.DELETE("/:postId/photos/:photoId", photoHandler.DeletePhoto) // TODO: Fix route conflict
	}

	// Users routes
	users := router.Group("/users")
	{
		users.GET("/:userId/posts", postHandler.GetUserPosts)
	}
}
