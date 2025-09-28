# fn-posts

**Photo-first lost & found posts with sub-15 second reporting workflow.**

## Purpose

Manages lost and found item posts through a unified API, enabling quick item reporting and geospatial discovery with photo-first UX.

**Technology**: Go + Gin + PostgreSQL/PostGIS + Google Cloud Storage + Kafka

## Quick Start

```bash
# 1. Setup environment (choose cloud or local)
cp .env.cloud.example .env    # For cloud development
# OR
cp .env.local.example .env    # For local development
# Edit .env with your credentials (see fn-docs/CLOUD-SETUP.md)

# 2. Start service
make dev

# 3. Test
curl http://localhost:8080/health
make e2e-test
```

## Business Rules

- **Photos**: 1-10 photos required per post
- **Lifecycle**: active → resolved/expired/deleted
- **Location**: PostGIS radius-based search
- **Events**: Published for all state changes

## Documentation

- **[DEVELOPMENT.md](./DEVELOPMENT.md)** - Complete development guide
- **[../fn-docs/](../fn-docs/)** - Architecture and standards