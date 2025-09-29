//go:build wireinject
// +build wireinject

package internal

import (
	"database/sql"

	"github.com/google/wire"
	"github.com/jsarabia/fn-posts/internal/config"
	"github.com/jsarabia/fn-posts/internal/domain"
	"github.com/jsarabia/fn-posts/internal/handler"
	"github.com/jsarabia/fn-posts/internal/repository"
	"github.com/jsarabia/fn-posts/internal/service"
)

// Application represents the complete application with all dependencies
type Application struct {
	PostHandler  *handler.PostHandler
	PhotoHandler *handler.PhotoHandler
	Config       *config.Config
}

// InitializeApplication sets up the complete application using Wire
func InitializeApplication(db *sql.DB, cfg *config.Config) (*Application, error) {
	wire.Build(
		// Repositories
		repository.NewPostgresPostRepository,
		repository.NewPostgresPhotoRepository,

		// Services
		service.NewEventService,
		service.NewStorageService,
		service.NewPostService,

		// Handlers
		handler.NewPostHandler,
		handler.NewPhotoHandler,

		// Providers
		provideStorageConfig,
		provideKafkaConfig,
		provideStorageInterface,
		providePostRepository,
		providePhotoRepository,
		provideEventPublisher,

		// Application
		wire.Struct(new(Application), "*"),
	)
	return &Application{}, nil
}

// Providers for configuration and interfaces

func provideStorageConfig(cfg *config.Config) config.StorageConfig {
	return cfg.StorageConfig
}

func provideKafkaConfig(cfg *config.Config) config.KafkaConfig {
	return cfg.KafkaConfig
}

func provideStorageInterface(storageService *service.StorageService) handler.StorageInterface {
	return storageService
}

func providePostRepository(repo *repository.PostgresPostRepository) domain.PostRepository {
	return repo
}

func providePhotoRepository(repo *repository.PostgresPhotoRepository) domain.PhotoRepository {
	return repo
}

func provideEventPublisher(eventService *service.EventService) domain.EventPublisher {
	return eventService
}
