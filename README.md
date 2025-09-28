# Posts Domain

**Photo-first lost & found posts with sub-15 second reporting workflow.**

## Domain Vision

The Posts Domain manages lost and found item posts through a unified API, enabling quick item reporting and geospatial discovery with photo-first UX and sub-15 second workflows.

## Quick Start

```bash
# 1. Install dependencies
go mod download

# 2. Set up environment
cp .env.example .env

# 3. Start the service
make dev

# 4. Access the API
curl http://localhost:8080/health
```

## Architecture

**Technology Stack:**
- **API**: Go + Gin framework
- **Database**: Supabase (PostgreSQL + PostGIS)
- **Storage**: Google Cloud Storage with global CDN
- **Events**: Confluent Cloud (managed Kafka)
- **Pattern**: Domain-Driven Design with Clean Architecture

## Domain Objects

See domain entities, value objects, and events in [`/internal/domain/`](./internal/domain/)

## Business Rules

**Photos**: All posts require 1-10 photos in JPEG/PNG/WebP format
**Lifecycle**: Posts transition: `active` → `resolved` | `expired` | `deleted`
**Location**: PostGIS integration for radius-based search with privacy controls
**Events**: Domain events published for `created`, `updated`, `resolved`, `deleted`

## Documentation

- **[CLAUDE.md](./CLAUDE.md)** - Development guidelines and DDD patterns
- **[fn-docs](https://github.com/your-org/fn-docs)** - Architecture decisions and product vision