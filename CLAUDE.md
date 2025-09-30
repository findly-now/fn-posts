# fn-posts DDD Implementation Guidelines

**Document Ownership**: This document OWNS Posts domain DDD patterns, Go development standards, and privacy-first event architecture.

## DDD Architecture

### Aggregates & Bounded Contexts
- **Post** - Main aggregate root managing posts lifecycle and invariants
- **ContactExchangeRequest** - Secure contact sharing aggregate with encryption
- **Entities**: Photo, Location with unique identities
- **Value Objects**: PostID, UserID, PrivacySafeUser, ContactExchangeToken
- **Repository Pattern**: Interfaces in domain, implementations in infrastructure

### Domain Boundaries
- **Database Isolation**: Only access `posts_db`, never query other domains
- **Fat Events**: Complete context eliminates cross-service API dependencies
- **Privacy Layer**: No PII in events, encrypted contact exchange only

## Testing Strategy: E2E Only

```go
// E2E tests with real repositories
func TestCreatePost_E2E(t *testing.T) {
    // Initialize real adapters (no mocks)
    testDB := setupPostgresDB(t)
    testGCS := setupGCS(t)
    testKafka := setupKafka(t)

    postService := NewPostService(testDB, testGCS, testKafka)

    // Test complete workflow
    post, err := postService.CreatePost(validCommand)
    assert.NoError(t, err)

    // Verify events published with no PII
    events := testKafka.GetPublishedEvents("post.created")
    assertNoPIIInEvents(t, events)
}
```

**Location**: `tests/e2e/*.go`
**Command**: `make e2e-test`

## Privacy-First Event Architecture

### Fat Events Pattern
Events include complete context to eliminate cross-service calls:

```go
type PostCreatedEvent struct {
    EventID     string               `json:"event_id"`
    EventType   string               `json:"event_type"`
    Payload     PostCreatedPayload   `json:"payload"`
    Privacy     PrivacyContext       `json:"privacy"`
}

type PostCreatedPayload struct {
    Post         PostData         `json:"post"`
    User         PrivacySafeUser  `json:"user"`        // NO email/phone
    Organization OrganizationData `json:"organization"`
    AIAnalysis   AIMetadata       `json:"ai_analysis"`
}
```

### No PII Rule
```go
// ❌ NEVER do this
type BadEvent struct {
    Email string `json:"email"` // PII violation!
}

// ✅ Always do this
type GoodEvent struct {
    UserID      string                `json:"user_id"`      // ID only
    DisplayName string                `json:"display_name"` // "John D."
    ContactToken ContactExchangeToken `json:"contact_token"` // Encrypted
}
```

## Domain Object Structure

```
internal/domain/
├── post.go              # Post aggregate root
├── contact_exchange.go  # ContactExchangeRequest aggregate
├── photo.go             # Photo entity
├── location.go          # Location entity (PostGIS)
├── value_objects.go     # IDs, PrivacySafeUser, tokens
├── event.go             # Fat event structures
├── event_converters.go  # Domain to event transformations
├── repository.go        # Repository interfaces
└── errors.go           # Domain-specific errors
```

## Business Rules Enforcement

### Post Aggregate Rules
- Photos: 1-10 required per post
- Status transitions: active → resolved/expired/deleted only
- Organization isolation: Posts filtered by organizationID
- Location: Valid GPS coordinates with PostGIS indexing

### Contact Exchange Rules
- Owner approval required for contact sharing
- Encrypted contact info with 24h expiration
- Verification required for high-value items
- Complete audit trail for compliance

## Performance Standards
- **Post creation**: <15 seconds end-to-end
- **Geospatial queries**: <200ms (PostGIS optimized)
- **Event processing**: 50-100ms (vs 500-2000ms with thin events)

## Development Commands

See workspace root [CLAUDE.md](../CLAUDE.md) for complete command reference.

**Key Commands**:
```bash
make dev          # Live reload development
make e2e-test     # E2E test suite
make fmt          # Format code
```

For detailed architecture patterns and cross-service standards, see [fn-docs/](../fn-docs/).