# fn-posts

Photo-first lost & found posts service with <15 second reporting workflow and privacy-first design, deployed on Google Kubernetes Engine (GKE).

**Technology**: Go + Gin + PostgreSQL/PostGIS + Google Cloud Storage + Kafka

## Service Overview

The fn-posts service is the core domain service managing lost & found posts with photo uploads, geospatial search, and privacy-first event streaming. It runs as a stateless deployment on GKE with auto-scaling based on CPU/memory utilization.

### Key Capabilities
- **Photos**: 1-10 photos required per post (stored in GCS)
- **Geospatial**: PostGIS radius-based search with spatial indexes
- **Events**: Self-contained Kafka events with complete context
- **Privacy**: Zero PII in events, encrypted contact exchange
- **Performance**: <15 second end-to-end workflow

## Prerequisites

### Local Development
```bash
# Required tools
go 1.21+
docker 20.10+
kubectl 1.28+
helm 3.12+
gcloud CLI

# GCP Service Account with permissions:
# - Cloud SQL Client
# - Storage Object Admin
# - Pub/Sub Editor
# - Artifact Registry Reader
```

### GKE Setup
```bash
# Authenticate with GCP
gcloud auth login
gcloud config set project findly-now-${ENV}

# Get GKE cluster credentials
gcloud container clusters get-credentials findly-cluster --region us-central1

# Verify connection
kubectl get nodes
```

## Environment Variables

### Development (GKE Dev)
```bash
# Database
DATABASE_URL=postgresql://posts_user:${DB_PASSWORD}@10.x.x.x:5432/posts_db?sslmode=require
DB_MAX_CONNECTIONS=25
DB_MAX_IDLE=5

# Google Cloud Storage
GCS_BUCKET=findly-photos-dev
GCS_PROJECT_ID=findly-now-dev
GOOGLE_APPLICATION_CREDENTIALS=/secrets/gcp/key.json

# Kafka
KAFKA_BROKERS=kafka-broker-1.dev:9092,kafka-broker-2.dev:9092
KAFKA_TOPIC_PREFIX=dev.

# Service Configuration
PORT=8080
LOG_LEVEL=debug
ENVIRONMENT=development
```

### Production (GKE Prod)
```bash
# Database (Cloud SQL Proxy)
DATABASE_URL=postgresql://posts_user:${DB_PASSWORD}@localhost:5432/posts_db?sslmode=require
DB_MAX_CONNECTIONS=100
DB_MAX_IDLE=20

# Google Cloud Storage
GCS_BUCKET=findly-photos-prod
GCS_PROJECT_ID=findly-now-prod
GOOGLE_APPLICATION_CREDENTIALS=/secrets/gcp/key.json

# Kafka
KAFKA_BROKERS=kafka-prod-1:9092,kafka-prod-2:9092,kafka-prod-3:9092
KAFKA_TOPIC_PREFIX=prod.
KAFKA_COMPRESSION=snappy

# Service Configuration
PORT=8080
LOG_LEVEL=info
ENVIRONMENT=production
```

## CI/CD Pipeline

### GitHub Actions Workflow
```yaml
# .github/workflows/deploy.yml
name: Deploy to GKE

on:
  push:
    branches: [main, develop]

jobs:
  deploy:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3

      - name: Setup Cloud SDK
        uses: google-github-actions/setup-gcloud@v1

      - name: Build and Push Image
        run: |
          docker build -t gcr.io/$PROJECT_ID/fn-posts:$GITHUB_SHA .
          docker push gcr.io/$PROJECT_ID/fn-posts:$GITHUB_SHA

      - name: Deploy to GKE
        run: |
          helm upgrade --install fn-posts ./charts/fn-posts \
            --set image.tag=$GITHUB_SHA \
            --namespace=findly \
            --values=./charts/fn-posts/values.$ENV.yaml
```

## Deployment Commands

### Helm Deployment
```bash
# Development deployment
helm upgrade --install fn-posts ./charts/fn-posts \
  --namespace=findly-dev \
  --values=./charts/fn-posts/values.dev.yaml \
  --set image.tag=latest

# Production deployment (with approval)
helm upgrade --install fn-posts ./charts/fn-posts \
  --namespace=findly-prod \
  --values=./charts/fn-posts/values.prod.yaml \
  --set image.tag=v1.2.3 \
  --wait --timeout=10m

# Rollback if needed
helm rollback fn-posts --namespace=findly-prod

# Check deployment status
kubectl rollout status deployment/fn-posts -n findly-prod
```

### Manual Kubernetes Commands
```bash
# Apply ConfigMap and Secrets
kubectl apply -f k8s/configmap.yaml -n findly-dev
kubectl create secret generic fn-posts-secrets --from-env-file=.env -n findly-dev

# Scale deployment
kubectl scale deployment fn-posts --replicas=5 -n findly-prod

# Update image
kubectl set image deployment/fn-posts fn-posts=gcr.io/findly-now-prod/fn-posts:v1.2.3 -n findly-prod
```

## Health Check Endpoints

### Liveness Probe
```bash
GET /health/live
# Returns 200 if service is running
# Used by Kubernetes to restart unhealthy pods

curl http://fn-posts.findly.com/health/live
```

### Readiness Probe
```bash
GET /health/ready
# Returns 200 if all dependencies are connected:
# - PostgreSQL connection pool
# - Kafka producer ready
# - GCS bucket accessible

curl http://fn-posts.findly.com/health/ready
```

### Detailed Health Status
```bash
GET /health
# Returns detailed status of all components
{
  "status": "healthy",
  "timestamp": "2024-01-15T10:30:00Z",
  "checks": {
    "database": "connected",
    "kafka": "connected",
    "gcs": "accessible",
    "memory": "78MB/256MB",
    "goroutines": 42
  }
}
```

## Monitoring & Logging

### Google Cloud Logging
```bash
# View logs
gcloud logging read "resource.type=k8s_container \
  AND resource.labels.namespace_name=findly-prod \
  AND resource.labels.container_name=fn-posts" \
  --limit=100 --format=json

# Stream logs
kubectl logs -f deployment/fn-posts -n findly-prod

# Logs with timestamp
kubectl logs deployment/fn-posts -n findly-prod --timestamps=true --tail=100
```

### Metrics & Alerts
```bash
# Prometheus metrics endpoint
GET /metrics

# Key metrics to monitor:
- fn_posts_request_duration_seconds
- fn_posts_active_connections
- fn_posts_photo_upload_duration_seconds
- fn_posts_kafka_publish_errors_total
- fn_posts_database_query_duration_seconds
```

### Grafana Dashboards
- **Service Health**: CPU, memory, request rate, error rate
- **Business Metrics**: Posts created, photos uploaded, search queries
- **Dependencies**: Database connections, Kafka lag, GCS latency

## Troubleshooting

### Common Issues

#### 1. Pod CrashLoopBackOff
```bash
# Check logs
kubectl logs pod/fn-posts-xxx -n findly-prod --previous

# Common causes:
# - Missing environment variables
# - Database connection failure
# - Invalid GCP credentials

# Fix: Update ConfigMap/Secrets and restart
kubectl rollout restart deployment/fn-posts -n findly-prod
```

#### 2. Database Connection Errors
```bash
# Test connection from pod
kubectl exec -it deployment/fn-posts -n findly-prod -- psql $DATABASE_URL

# Check Cloud SQL Proxy
kubectl logs deployment/cloud-sql-proxy -n findly-prod

# Verify network policies
kubectl get networkpolicies -n findly-prod
```

#### 3. Photo Upload Failures
```bash
# Check GCS permissions
gsutil iam get gs://findly-photos-prod

# Test from pod
kubectl exec -it deployment/fn-posts -n findly-prod -- \
  gsutil cp test.jpg gs://findly-photos-prod/test/

# Verify service account
kubectl get secret gcp-service-account -n findly-prod -o yaml
```

#### 4. Kafka Publishing Issues
```bash
# Check Kafka connectivity
kubectl exec -it deployment/fn-posts -n findly-prod -- \
  kafkacat -b kafka-prod-1:9092 -L

# Monitor consumer lag
kafka-consumer-groups --bootstrap-server kafka-prod-1:9092 \
  --group fn-matcher --describe
```

#### 5. Memory/CPU Issues
```bash
# Check resource usage
kubectl top pod -n findly-prod | grep fn-posts

# Update resource limits
kubectl set resources deployment/fn-posts \
  --requests=memory=256Mi,cpu=100m \
  --limits=memory=512Mi,cpu=500m \
  -n findly-prod
```

### Emergency Procedures

```bash
# Quick rollback
helm rollback fn-posts --namespace=findly-prod

# Scale to zero (maintenance)
kubectl scale deployment/fn-posts --replicas=0 -n findly-prod

# Force restart all pods
kubectl rollout restart deployment/fn-posts -n findly-prod

# Emergency access to pod
kubectl exec -it deployment/fn-posts -n findly-prod -- /bin/sh
```

## Development Commands

```bash
# Local development with hot reload
make dev

# Run E2E tests
make e2e-test

# Build Docker image
make docker-build

# Push to registry
make docker-push TAG=v1.2.3

# Database migrations
make migrate-up
```

## Documentation

- **[CLAUDE.md](./CLAUDE.md)** - DDD implementation guidelines
- **[fn-docs/](../fn-docs/)** - System architecture and standards
- **[fn-infra/](../fn-infra/)** - Kubernetes manifests and Helm charts