# CLAUDE.md - Posts Domain Development Guidelines

## Domain-Driven Design (DDD) Principles

This microservice strictly follows DDD patterns:

- **Aggregate Root**: `Post` entity manages all domain invariants
- **Entities**: `Post`, `Photo`, `Location` with unique identities
- **Value Objects**: `PostID`, `PhotoID`, `UserID`, `OrganizationID`, `Coordinates`
- **Domain Events**: Published for all state changes (`PostCreated`, `PostResolved`, etc.)
- **Repository Pattern**: Abstract interfaces in domain, concrete implementations in infrastructure

## Testing Strategy

**E2E Tests Only**: All tests initialize repositories with actual adapters (database, storage)
- Test files: `tests/e2e/*.go`
- Run: `go test ./tests/e2e/...`
- Docker environment: `tests/e2e/docker-compose.e2e.yml`

## Code Standards

- **Minimal Comments**: Comments only for complex business rules
- **Clean Architecture**: Domain → Service → Handler layers
- **Event-Driven**: Kafka events for all domain state changes

## Development Commands

```bash
# Run tests
go test ./tests/e2e/...

# Build
go build -o main cmd/main.go

# Lint (if available)
golangci-lint run

# Format
go fmt ./...
```

## Domain Objects Location

All domain objects are in `/internal/domain/`:
- `post.go` - Post aggregate root
- `photo.go` - Photo entity
- `location.go` - Location entity
- `value_objects.go` - Value objects and DTOs
- `event.go` - Domain events
- `repository.go` - Repository interfaces

## Architecture Decisions

For detailed architecture decisions and product vision, see the [fn-docs repository](https://github.com/your-org/fn-docs).