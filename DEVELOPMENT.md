# fn-posts Development Guide

**Document Ownership**: This document OWNS all development workflows for the Posts domain service.

## Prerequisites

- **Go** 1.21+
- **Docker Desktop** 20.10+
- **Make** installed
- **Cloud credentials** configured (see [../fn-docs/CLOUD-SETUP.md](../fn-docs/CLOUD-SETUP.md))

## Environment Setup

```bash
# Clone and navigate
git clone <repository-url>
cd fn-posts

# Setup environment (choose based on development approach)
cp .env.cloud.example .env    # For cloud-based development (recommended)
# OR
cp .env.local.example .env    # For local Docker development
# Edit .env with your credentials

# Install dependencies
go mod download

# Start development environment
make docker-up

# Run database migrations
make migrate-up

# Start service with live reload
make dev
```

## Development Commands

### Service Operations
```bash
make run                    # Start service
make dev                    # Run with live reload (air)
make build                  # Build binary
make docker-up              # Start local infrastructure
make docker-down            # Stop local infrastructure
```

### Database Operations
```bash
make migrate-up             # Apply database schema from script.sql
make migrate-down           # Drop database objects
psql "$DATABASE_URL" -f script.sql  # Apply schema directly
```

### Testing
```bash
make e2e-test              # Complete E2E test suite
make e2e-up                # Start test environment
make e2e-down              # Stop test environment
make e2e-test-only         # Run tests only (env must be running)
```

### Code Quality
```bash
make fmt                   # Format code
make lint                  # Run golangci-lint
make tidy                  # Tidy dependencies
make vet                   # Run go vet
```

## Domain-Driven Design Architecture

### Aggregate Root: Post
The Post entity manages all domain invariants:

```go
type Post struct {
    id           PostID        // Required unique identifier
    title        string        // Required: 1-100 characters
    description  string        // Optional: up to 500 characters
    photos       []Photo       // Required: 1-10 photos
    location     Location      // Required: lat/lng coordinates
    searchRadius Distance      // Required: search area in meters
    itemType     ItemType      // Required: lost or found
    status       PostStatus    // Required: active/resolved/expired/deleted
    organizationID OrgID       // Required: tenant isolation
    userID       UserID        // Required: post creator
    createdAt    time.Time     // Auto-generated
    updatedAt    time.Time     // Auto-generated
}
```

### Business Rules Enforced
- **Photos**: Minimum 1, maximum 10 photos per post
- **Location**: Valid GPS coordinates with reasonable search radius
- **Status Transitions**: active → resolved/expired/deleted only
- **Organization Isolation**: Posts isolated by organizationID

### Repository Pattern
```go
// Domain interface (internal/domain/repository.go)
type PostRepository interface {
    Create(post *Post) error
    FindByID(id PostID) (*Post, error)
    FindByRadius(location Location, radius Distance) ([]*Post, error)
    UpdateStatus(id PostID, status PostStatus) error
}

// Infrastructure implementation (internal/repository/postgres.go)
type PostgresPostRepository struct {
    db *sql.DB
}
```

## API Development

### Endpoint Structure
```
GET    /api/posts           # List posts with geospatial filtering
POST   /api/posts           # Create new post
GET    /api/posts/{id}      # Get specific post
PUT    /api/posts/{id}      # Update post
DELETE /api/posts/{id}      # Delete post
```

### Request/Response Examples
```bash
# Create post
curl -X POST http://localhost:8080/api/posts \
  -H "Content-Type: application/json" \
  -d '{
    "title": "Lost iPhone 14",
    "description": "Black iPhone with blue case",
    "itemType": "lost",
    "location": {"lat": 37.7749, "lng": -122.4194},
    "searchRadius": 1000,
    "photos": ["base64..."]
  }'

# Search by location
curl "http://localhost:8080/api/posts?lat=37.7749&lng=-122.4194&radius=5000"
```

## Event Publishing

### Domain Events
All state changes publish events to Kafka:

```go
// Published events
type PostCreatedEvent struct {
    PostID       string    `json:"post_id"`
    UserID       string    `json:"user_id"`
    Title        string    `json:"title"`
    ItemType     string    `json:"item_type"`
    Location     Location  `json:"location"`
    PhotoURLs    []string  `json:"photo_urls"`
    Timestamp    time.Time `json:"timestamp"`
}

// Event publishing
func (s *PostService) CreatePost(cmd CreatePostCommand) (*Post, error) {
    post := domain.NewPost(cmd)

    if err := s.repo.Create(post); err != nil {
        return nil, err
    }

    // Publish domain event
    event := PostCreatedEvent{
        PostID:    post.ID().String(),
        UserID:    post.UserID().String(),
        Title:     post.Title(),
        ItemType:  post.ItemType().String(),
        Location:  post.Location(),
        PhotoURLs: post.PhotoURLs(),
        Timestamp: time.Now(),
    }

    s.events.Publish("post.created", event)
    return post, nil
}
```

## Database Schema

### Schema Management
Posts service uses a single `script.sql` file for database setup (similar to fn-notifications approach):

```bash
# Apply complete schema
psql "$DATABASE_URL" -f script.sql

# Or use makefile
make migrate-up
```

### Posts Table
```sql
CREATE TABLE posts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    title VARCHAR(200) NOT NULL,
    description TEXT,
    location GEOMETRY(POINT, 4326),
    radius_meters INTEGER DEFAULT 1000,
    status post_status DEFAULT 'active',
    type post_type NOT NULL,
    user_id UUID NOT NULL,
    organization_id UUID,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- PostGIS spatial index for performance
CREATE INDEX idx_posts_location ON posts USING GIST (location);
CREATE INDEX idx_posts_status_org ON posts (status, organization_id);
```

### Post Photos Table
```sql
CREATE TABLE post_photos (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    post_id UUID NOT NULL REFERENCES posts(id) ON DELETE CASCADE,
    url TEXT NOT NULL,
    thumbnail_url TEXT,
    display_order INTEGER NOT NULL DEFAULT 1,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);
```

## Testing Guidelines

### E2E Test Structure
```go
func TestCreatePost_E2E(t *testing.T) {
    // Initialize real adapters
    testDB := testhelpers.SetupPostgresDB(t)
    testGCS := testhelpers.SetupGCS(t)
    testKafka := testhelpers.SetupKafka(t)

    // Initialize repositories
    postRepo := postgres.NewPostRepository(testDB)
    photoRepo := postgres.NewPhotoRepository(testDB)

    // Initialize services
    photoService := service.NewPhotoService(photoRepo, testGCS)
    eventService := service.NewEventService(testKafka)
    postService := service.NewPostService(postRepo, photoService, eventService)

    // Test complete workflow
    command := CreatePostCommand{
        Title:        "Lost iPhone",
        Photos:       [][]byte{validPhotoData},
        Location:     Location{Lat: 37.7749, Lng: -122.4194},
        SearchRadius: 1000,
        ItemType:     Lost,
        UserID:       UserID("user123"),
        OrgID:        OrgID("org456"),
    }

    post, err := postService.CreatePost(command)

    // Assertions
    assert.NoError(t, err)
    assert.NotEmpty(t, post.ID())
    assert.Len(t, post.PhotoURLs(), 1)

    // Verify event published
    events := testKafka.GetPublishedEvents("post.created")
    assert.Len(t, events, 1)
}
```

## Error Handling

### Custom Error Types
```go
type PostError struct {
    Code    string
    Message string
    Cause   error
}

var (
    ErrPostNotFound     = PostError{Code: "POST_NOT_FOUND", Message: "Post not found"}
    ErrTooManyPhotos    = PostError{Code: "TOO_MANY_PHOTOS", Message: "Maximum 10 photos allowed"}
    ErrInvalidLocation  = PostError{Code: "INVALID_LOCATION", Message: "Invalid GPS coordinates"}
    ErrUnauthorized     = PostError{Code: "UNAUTHORIZED", Message: "User not authorized for organization"}
)

// HTTP error mapping
func (h *PostHandler) handleError(c *gin.Context, err error) {
    switch {
    case errors.Is(err, ErrPostNotFound):
        c.JSON(404, gin.H{"error": err.Error()})
    case errors.Is(err, ErrTooManyPhotos), errors.Is(err, ErrInvalidLocation):
        c.JSON(400, gin.H{"error": err.Error()})
    case errors.Is(err, ErrUnauthorized):
        c.JSON(403, gin.H{"error": err.Error()})
    default:
        c.JSON(500, gin.H{"error": "Internal server error"})
    }
}
```

## Performance Guidelines

### PostGIS Query Optimization
```go
// Optimized spatial query with proper indexing
func (r *PostgresPostRepository) FindByRadius(location Location, radius Distance) ([]*Post, error) {
    query := `
        SELECT id, title, description, item_type, status,
               ST_X(location::geometry) as lng, ST_Y(location::geometry) as lat,
               search_radius, organization_id, user_id, created_at, updated_at,
               ST_Distance(location, ST_SetSRID(ST_MakePoint($1, $2), 4326)) as distance
        FROM posts
        WHERE status = 'active'
          AND ST_DWithin(location, ST_SetSRID(ST_MakePoint($1, $2), 4326), $3)
        ORDER BY distance
        LIMIT 100`

    rows, err := r.db.Query(query, location.Lng, location.Lat, radius.Meters)
    // ... handle results
}
```

### Photo Upload Optimization
```go
// Concurrent photo uploads to GCS
func (s *PhotoService) UploadPhotos(photos [][]byte) ([]string, error) {
    var wg sync.WaitGroup
    urls := make([]string, len(photos))
    errors := make([]error, len(photos))

    for i, photo := range photos {
        wg.Add(1)
        go func(index int, data []byte) {
            defer wg.Done()
            url, err := s.storage.Upload(data)
            urls[index] = url
            errors[index] = err
        }(i, photo)
    }

    wg.Wait()

    // Check for errors
    for _, err := range errors {
        if err != nil {
            return nil, err
        }
    }

    return urls, nil
}
```

---

*For architecture and cross-service standards, see [../fn-docs/](../fn-docs/)*