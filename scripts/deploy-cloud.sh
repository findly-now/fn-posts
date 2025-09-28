#!/bin/bash

# =============================================================================
# Cloud Deployment Script
# =============================================================================
# This script deploys the Posts service to various cloud platforms
# with proper configuration for cloud infrastructure.
#
# Prerequisites:
# - Cloud infrastructure already set up (run setup-cloud-infra.sh first)
# - Environment file (.env.[environment]) configured
# - Docker installed (for container deployment)
# - Platform-specific CLI tools installed
#
# Usage: ./scripts/deploy-cloud.sh [platform] [environment]
# Platform: docker, gcp-run, k8s (default: docker)
# Environment: staging, production (default: staging)
# =============================================================================

set -euo pipefail

# Configuration
PLATFORM=${1:-docker}
ENVIRONMENT=${2:-staging}
PROJECT_PREFIX="posts-service"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "${SCRIPT_DIR}/.." && pwd)"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Logging functions
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Check prerequisites
check_prerequisites() {
    log_info "Checking prerequisites..."

    # Check if environment file exists
    local env_file=".env.${ENVIRONMENT}"
    if [[ ! -f "${ROOT_DIR}/${env_file}" ]]; then
        log_error "Environment file not found: ${env_file}"
        log_info "Please run: ./scripts/setup-cloud-infra.sh ${ENVIRONMENT}"
        exit 1
    fi

    # Check Docker
    if ! command -v docker &> /dev/null; then
        log_error "Docker not found. Please install Docker."
        exit 1
    fi

    # Platform-specific checks
    case ${PLATFORM} in
        "gcp-run")
            if ! command -v gcloud &> /dev/null; then
                log_error "gcloud CLI not found. Please install Google Cloud SDK."
                exit 1
            fi
            ;;
        "k8s")
            if ! command -v kubectl &> /dev/null; then
                log_error "kubectl not found. Please install kubectl."
                exit 1
            fi
            ;;
    esac

    log_success "All prerequisites met."
}

# Build Docker image
build_docker_image() {
    log_info "Building Docker image..."

    local image_name="${PROJECT_PREFIX}:${ENVIRONMENT}"
    local dockerfile_path="${ROOT_DIR}/Dockerfile"

    if [[ ! -f "${dockerfile_path}" ]]; then
        log_error "Dockerfile not found: ${dockerfile_path}"
        exit 1
    fi

    cd "${ROOT_DIR}"
    docker build -t "${image_name}" .

    log_success "Built Docker image: ${image_name}"
}

# Validate environment configuration
validate_environment() {
    log_info "Validating environment configuration..."

    local env_file=".env.${ENVIRONMENT}"
    source "${ROOT_DIR}/${env_file}"

    # Check required cloud provider variables
    local missing_vars=()

    if [[ "${STORAGE_PROVIDER:-}" != "gcs" ]]; then
        missing_vars+=("STORAGE_PROVIDER should be 'gcs'")
    fi

    if [[ "${EVENT_PROVIDER:-}" != "confluent" ]]; then
        missing_vars+=("EVENT_PROVIDER should be 'confluent'")
    fi

    if [[ -z "${GCS_PROJECT_ID:-}" ]]; then
        missing_vars+=("GCS_PROJECT_ID")
    fi

    if [[ -z "${CONFLUENT_BOOTSTRAP_SERVERS:-}" ]]; then
        missing_vars+=("CONFLUENT_BOOTSTRAP_SERVERS")
    fi

    if [[ -z "${DATABASE_URL:-}" ]] || [[ "${DATABASE_URL}" == *"[PASSWORD]"* ]]; then
        missing_vars+=("DATABASE_URL (properly configured)")
    fi

    if [[ ${#missing_vars[@]} -gt 0 ]]; then
        log_error "Missing or invalid environment variables:"
        for var in "${missing_vars[@]}"; do
            echo "  - ${var}"
        done
        log_info "Please update ${env_file} with proper values."
        exit 1
    fi

    log_success "Environment configuration is valid."
}

# Deploy with Docker
deploy_docker() {
    log_info "Deploying with Docker..."

    local image_name="${PROJECT_PREFIX}:${ENVIRONMENT}"
    local container_name="${PROJECT_PREFIX}-${ENVIRONMENT}"
    local env_file=".env.${ENVIRONMENT}"

    # Stop existing container
    if docker ps -q -f name="${container_name}" | grep -q .; then
        log_info "Stopping existing container..."
        docker stop "${container_name}"
        docker rm "${container_name}"
    fi

    # Run new container
    log_info "Starting new container..."
    docker run -d \
        --name "${container_name}" \
        --env-file "${ROOT_DIR}/${env_file}" \
        -p 8080:8080 \
        --restart unless-stopped \
        "${image_name}"

    log_success "Container deployed: ${container_name}"

    # Show container status
    docker ps -f name="${container_name}"

    # Show logs
    log_info "Container logs (last 20 lines):"
    docker logs --tail 20 "${container_name}"
}

# Deploy to Google Cloud Run
deploy_gcp_run() {
    log_info "Deploying to Google Cloud Run..."

    local service_name="${PROJECT_PREFIX}-${ENVIRONMENT}"
    local image_name="gcr.io/$(gcloud config get-value project)/${service_name}:latest"
    local env_file=".env.${ENVIRONMENT}"

    # Tag and push image
    log_info "Pushing image to Google Container Registry..."
    docker tag "${PROJECT_PREFIX}:${ENVIRONMENT}" "${image_name}"
    docker push "${image_name}"

    # Read environment variables
    local env_vars=()
    while IFS='=' read -r key value; do
        # Skip comments and empty lines
        [[ $key =~ ^#.*$ ]] && continue
        [[ -z "$key" ]] && continue

        # Add to environment variables array
        env_vars+=("--set-env-vars=${key}=${value}")
    done < "${ROOT_DIR}/${env_file}"

    # Deploy to Cloud Run
    log_info "Deploying to Cloud Run..."
    gcloud run deploy "${service_name}" \
        --image="${image_name}" \
        --platform=managed \
        --region=us-central1 \
        --allow-unauthenticated \
        --port=8080 \
        --memory=512Mi \
        --cpu=1 \
        --max-instances=10 \
        "${env_vars[@]}"

    # Get service URL
    local service_url
    service_url=$(gcloud run services describe "${service_name}" \
        --platform=managed \
        --region=us-central1 \
        --format="value(status.url)")

    log_success "Deployed to Cloud Run: ${service_url}"
}

# Deploy to Kubernetes
deploy_k8s() {
    log_info "Deploying to Kubernetes..."

    local app_name="${PROJECT_PREFIX}-${ENVIRONMENT}"
    local image_name="${PROJECT_PREFIX}:${ENVIRONMENT}"
    local env_file=".env.${ENVIRONMENT}"

    # Create namespace if it doesn't exist
    kubectl create namespace "${ENVIRONMENT}" --dry-run=client -o yaml | kubectl apply -f -

    # Create secret from environment file
    log_info "Creating Kubernetes secret..."
    kubectl create secret generic "${app_name}-config" \
        --from-env-file="${ROOT_DIR}/${env_file}" \
        --namespace="${ENVIRONMENT}" \
        --dry-run=client -o yaml | kubectl apply -f -

    # Generate Kubernetes manifests
    local k8s_dir="${ROOT_DIR}/k8s"
    mkdir -p "${k8s_dir}"

    # Deployment manifest
    cat > "${k8s_dir}/deployment.yaml" <<EOF
apiVersion: apps/v1
kind: Deployment
metadata:
  name: ${app_name}
  namespace: ${ENVIRONMENT}
  labels:
    app: ${app_name}
    environment: ${ENVIRONMENT}
spec:
  replicas: 2
  selector:
    matchLabels:
      app: ${app_name}
  template:
    metadata:
      labels:
        app: ${app_name}
        environment: ${ENVIRONMENT}
    spec:
      containers:
      - name: posts-service
        image: ${image_name}
        ports:
        - containerPort: 8080
        envFrom:
        - secretRef:
            name: ${app_name}-config
        resources:
          requests:
            memory: "256Mi"
            cpu: "250m"
          limits:
            memory: "512Mi"
            cpu: "500m"
        livenessProbe:
          httpGet:
            path: /health
            port: 8080
          initialDelaySeconds: 30
          periodSeconds: 10
        readinessProbe:
          httpGet:
            path: /health
            port: 8080
          initialDelaySeconds: 5
          periodSeconds: 5
---
apiVersion: v1
kind: Service
metadata:
  name: ${app_name}-service
  namespace: ${ENVIRONMENT}
  labels:
    app: ${app_name}
spec:
  selector:
    app: ${app_name}
  ports:
  - port: 80
    targetPort: 8080
    protocol: TCP
  type: ClusterIP
---
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: ${app_name}-ingress
  namespace: ${ENVIRONMENT}
  labels:
    app: ${app_name}
  annotations:
    kubernetes.io/ingress.class: "nginx"
    cert-manager.io/cluster-issuer: "letsencrypt-prod"
spec:
  tls:
  - hosts:
    - ${app_name}.yourdomain.com
    secretName: ${app_name}-tls
  rules:
  - host: ${app_name}.yourdomain.com
    http:
      paths:
      - path: /
        pathType: Prefix
        backend:
          service:
            name: ${app_name}-service
            port:
              number: 80
EOF

    # Apply manifests
    log_info "Applying Kubernetes manifests..."
    kubectl apply -f "${k8s_dir}/deployment.yaml"

    # Wait for deployment
    log_info "Waiting for deployment to be ready..."
    kubectl wait --for=condition=available deployment/${app_name} \
        --namespace="${ENVIRONMENT}" \
        --timeout=300s

    # Show deployment status
    kubectl get pods -l app="${app_name}" --namespace="${ENVIRONMENT}"

    log_success "Deployed to Kubernetes namespace: ${ENVIRONMENT}"
}

# Run database migrations
run_migrations() {
    log_info "Running database migrations..."

    local env_file=".env.${ENVIRONMENT}"

    # Source environment variables
    source "${ROOT_DIR}/${env_file}"

    # Check if migrations directory exists
    if [[ ! -d "${ROOT_DIR}/migrations" ]]; then
        log_warning "No migrations directory found, skipping migrations."
        return
    fi

    # Run migrations using a temporary container
    docker run --rm \
        --env-file "${ROOT_DIR}/${env_file}" \
        -v "${ROOT_DIR}/migrations:/migrations" \
        postgres:15 \
        sh -c 'for file in /migrations/*.up.sql; do echo "Running $file"; psql "$DATABASE_URL" -f "$file"; done'

    log_success "Database migrations completed."
}

# Health check
health_check() {
    log_info "Performing health check..."

    local url=""
    local max_attempts=30
    local attempt=1

    case ${PLATFORM} in
        "docker")
            url="http://localhost:8080/health"
            ;;
        "gcp-run")
            local service_name="${PROJECT_PREFIX}-${ENVIRONMENT}"
            url=$(gcloud run services describe "${service_name}" \
                --platform=managed \
                --region=us-central1 \
                --format="value(status.url)")/health
            ;;
        "k8s")
            # Port forward for health check
            kubectl port-forward "service/${PROJECT_PREFIX}-${ENVIRONMENT}-service" 8080:80 \
                --namespace="${ENVIRONMENT}" &
            local port_forward_pid=$!
            sleep 5
            url="http://localhost:8080/health"
            ;;
    esac

    log_info "Checking health endpoint: ${url}"

    while [[ $attempt -le $max_attempts ]]; do
        if curl -s -f "${url}" > /dev/null 2>&1; then
            log_success "Health check passed!"

            # Cleanup port forward for k8s
            if [[ ${PLATFORM} == "k8s" ]]; then
                kill $port_forward_pid 2>/dev/null || true
            fi

            return 0
        fi

        log_info "Attempt ${attempt}/${max_attempts}: Health check failed, retrying in 10 seconds..."
        sleep 10
        ((attempt++))
    done

    # Cleanup port forward for k8s
    if [[ ${PLATFORM} == "k8s" ]]; then
        kill $port_forward_pid 2>/dev/null || true
    fi

    log_error "Health check failed after ${max_attempts} attempts."
    return 1
}

# Main execution
main() {
    log_info "Deploying Posts service to ${PLATFORM} (${ENVIRONMENT})"

    check_prerequisites
    validate_environment
    build_docker_image
    run_migrations

    case ${PLATFORM} in
        "docker")
            deploy_docker
            ;;
        "gcp-run")
            deploy_gcp_run
            ;;
        "k8s")
            deploy_k8s
            ;;
        *)
            log_error "Unsupported platform: ${PLATFORM}"
            log_info "Supported platforms: docker, gcp-run, k8s"
            exit 1
            ;;
    esac

    # Wait a moment for service to start
    sleep 10

    # Perform health check
    if health_check; then
        log_success "Deployment completed successfully!"

        case ${PLATFORM} in
            "docker")
                echo "Service is running at: http://localhost:8080"
                ;;
            "gcp-run")
                local service_url
                service_url=$(gcloud run services describe "${PROJECT_PREFIX}-${ENVIRONMENT}" \
                    --platform=managed \
                    --region=us-central1 \
                    --format="value(status.url)")
                echo "Service is running at: ${service_url}"
                ;;
            "k8s")
                echo "Service is running in Kubernetes namespace: ${ENVIRONMENT}"
                echo "Use port forwarding to access: kubectl port-forward service/${PROJECT_PREFIX}-${ENVIRONMENT}-service 8080:80 --namespace=${ENVIRONMENT}"
                ;;
        esac
    else
        log_error "Deployment completed but health check failed."
        exit 1
    fi
}

# Show usage if no arguments
if [[ $# -eq 0 ]]; then
    cat <<EOF
Usage: $0 [platform] [environment]

Platforms:
  docker      Deploy using Docker containers (default)
  gcp-run     Deploy to Google Cloud Run
  k8s         Deploy to Kubernetes

Environments:
  staging     Staging environment (default)
  production  Production environment

Examples:
  $0                          # Deploy to Docker (staging)
  $0 docker production        # Deploy to Docker (production)
  $0 gcp-run staging          # Deploy to Google Cloud Run (staging)
  $0 k8s production           # Deploy to Kubernetes (production)

Prerequisites:
  1. Run setup-cloud-infra.sh first
  2. Configure .env.[environment] file
  3. Install required CLI tools for target platform

EOF
    exit 0
fi

# Run main function
main "$@"