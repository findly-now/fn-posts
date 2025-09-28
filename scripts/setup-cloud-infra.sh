#!/bin/bash

# =============================================================================
# Cloud Infrastructure Setup Script
# =============================================================================
# This script helps set up the cloud infrastructure for the Posts service
# including Supabase, Google Cloud Storage, and Confluent Cloud.
#
# Prerequisites:
# - gcloud CLI installed and configured
# - confluent CLI installed and logged in
# - supabase CLI installed (optional, for automated setup)
#
# Usage: ./scripts/setup-cloud-infra.sh [environment]
# Environment: staging, production (default: staging)
# =============================================================================

set -euo pipefail

# Configuration
ENVIRONMENT=${1:-staging}
PROJECT_PREFIX="posts-service"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

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

    # Check gcloud CLI
    if ! command -v gcloud &> /dev/null; then
        log_error "gcloud CLI not found. Please install Google Cloud SDK."
        exit 1
    fi

    # Check confluent CLI
    if ! command -v confluent &> /dev/null; then
        log_error "confluent CLI not found. Please install Confluent CLI."
        exit 1
    fi

    # Check if logged in to gcloud
    if ! gcloud auth list --filter=status:ACTIVE --format="value(account)" | grep -q "."; then
        log_error "Not logged in to gcloud. Please run 'gcloud auth login'."
        exit 1
    fi

    log_success "All prerequisites met."
}

# Setup Google Cloud Storage
setup_gcs() {
    log_info "Setting up Google Cloud Storage..."

    local project_id
    project_id=$(gcloud config get-value project)

    if [[ -z "$project_id" ]]; then
        log_error "No active Google Cloud project. Please run 'gcloud config set project PROJECT_ID'."
        exit 1
    fi

    local bucket_name="${PROJECT_PREFIX}-${ENVIRONMENT}"
    local service_account_name="${PROJECT_PREFIX}-${ENVIRONMENT}"
    local key_file="${service_account_name}-key.json"

    log_info "Creating GCS bucket: ${bucket_name}"

    # Create bucket
    if ! gsutil ls -b "gs://${bucket_name}" &> /dev/null; then
        gsutil mb "gs://${bucket_name}"
        gsutil iam ch allUsers:objectViewer "gs://${bucket_name}"
        log_success "Created GCS bucket: ${bucket_name}"
    else
        log_warning "GCS bucket already exists: ${bucket_name}"
    fi

    # Create service account
    log_info "Creating service account: ${service_account_name}"

    if ! gcloud iam service-accounts describe "${service_account_name}@${project_id}.iam.gserviceaccount.com" &> /dev/null; then
        gcloud iam service-accounts create "${service_account_name}" \
            --display-name="Posts Service ${ENVIRONMENT}" \
            --description="Service account for Posts service ${ENVIRONMENT} environment"

        # Grant permissions
        gcloud projects add-iam-policy-binding "${project_id}" \
            --member="serviceAccount:${service_account_name}@${project_id}.iam.gserviceaccount.com" \
            --role="roles/storage.admin"

        log_success "Created service account: ${service_account_name}"
    else
        log_warning "Service account already exists: ${service_account_name}"
    fi

    # Create key file
    log_info "Creating service account key..."
    gcloud iam service-accounts keys create "${key_file}" \
        --iam-account="${service_account_name}@${project_id}.iam.gserviceaccount.com"

    log_success "Service account key created: ${key_file}"
    log_warning "Keep this key file secure and don't commit it to version control!"

    # Output configuration
    cat <<EOF

GCS Configuration:
==================
GCS_PROJECT_ID=${project_id}
GCS_BUCKET_NAME=${bucket_name}
GOOGLE_APPLICATION_CREDENTIALS=./${key_file}

EOF
}

# Setup Confluent Cloud
setup_confluent() {
    log_info "Setting up Confluent Cloud..."

    # Check if logged in
    if ! confluent kafka cluster list &> /dev/null; then
        log_error "Not logged in to Confluent Cloud. Please run 'confluent login'."
        exit 1
    fi

    local cluster_name="${PROJECT_PREFIX}-${ENVIRONMENT}"
    local topic_name="posts.events"

    # Create cluster
    log_info "Creating Kafka cluster: ${cluster_name}"

    local cluster_id
    cluster_id=$(confluent kafka cluster list -o json | jq -r ".[] | select(.name == \"${cluster_name}\") | .id" 2>/dev/null || echo "")

    if [[ -z "$cluster_id" ]]; then
        confluent kafka cluster create "${cluster_name}" \
            --cloud "aws" \
            --region "us-west-2" \
            --type "basic"

        # Wait for cluster to be ready
        log_info "Waiting for cluster to be ready..."
        sleep 30

        cluster_id=$(confluent kafka cluster list -o json | jq -r ".[] | select(.name == \"${cluster_name}\") | .id")
        log_success "Created Kafka cluster: ${cluster_name} (${cluster_id})"
    else
        log_warning "Kafka cluster already exists: ${cluster_name} (${cluster_id})"
    fi

    # Use the cluster
    confluent kafka cluster use "${cluster_id}"

    # Create topic
    log_info "Creating Kafka topic: ${topic_name}"

    if ! confluent kafka topic describe "${topic_name}" &> /dev/null; then
        confluent kafka topic create "${topic_name}" \
            --partitions 6 \
            --config retention.ms=604800000
        log_success "Created Kafka topic: ${topic_name}"
    else
        log_warning "Kafka topic already exists: ${topic_name}"
    fi

    # Create API key
    log_info "Creating API key..."
    local api_key_output
    api_key_output=$(confluent api-key create --resource "${cluster_id}" -o json)
    local api_key
    local api_secret
    api_key=$(echo "${api_key_output}" | jq -r '.key')
    api_secret=$(echo "${api_key_output}" | jq -r '.secret')

    # Get bootstrap servers
    local bootstrap_servers
    bootstrap_servers=$(confluent kafka cluster describe "${cluster_id}" -o json | jq -r '.endpoint' | sed 's/SASL_SSL:\/\///')

    log_success "Created API key: ${api_key}"

    # Output configuration
    cat <<EOF

Confluent Cloud Configuration:
==============================
CONFLUENT_BOOTSTRAP_SERVERS=${bootstrap_servers}
CONFLUENT_API_KEY=${api_key}
CONFLUENT_API_SECRET=${api_secret}
KAFKA_TOPIC=${topic_name}

EOF

    log_warning "Store the API secret securely - it won't be shown again!"
}

# Setup Supabase
setup_supabase() {
    log_info "Setting up Supabase..."

    local project_name="${PROJECT_PREFIX}-${ENVIRONMENT}"

    cat <<EOF

Supabase Setup (Manual Steps Required):
=======================================

1. Go to https://supabase.com/dashboard
2. Create a new project named: ${project_name}
3. Choose a region close to your users
4. Generate a strong database password
5. Wait for the project to be provisioned

6. Enable PostGIS extension:
   - Go to Database > Extensions
   - Search for "postgis" and enable it

7. Get your connection details:
   - Go to Settings > Database
   - Copy the connection string

8. Configure environment variables:
   DATABASE_URL=postgresql://postgres:[PASSWORD]@db.[PROJECT].supabase.co:5432/postgres?sslmode=require
   SUPABASE_URL=https://[PROJECT].supabase.co
   SUPABASE_ANON_KEY=[ANON_KEY from Settings > API]

EOF
}

# Generate environment file
generate_env_file() {
    log_info "Generating environment file..."

    local env_file=".env.${ENVIRONMENT}"

    cat > "${env_file}" <<EOF
# Posts Service - ${ENVIRONMENT} Environment
# Generated on $(date)

# =============================================================================
# SERVER CONFIGURATION
# =============================================================================
PORT=8080
ENVIRONMENT=${ENVIRONMENT}

# =============================================================================
# CLOUD PROVIDER CONFIGURATION
# =============================================================================
STORAGE_PROVIDER=gcs
EVENT_PROVIDER=confluent

# =============================================================================
# SUPABASE CONFIGURATION
# =============================================================================
# TODO: Update these values from your Supabase project
DATABASE_URL=postgresql://postgres:[PASSWORD]@db.[PROJECT].supabase.co:5432/postgres?sslmode=require
SUPABASE_URL=https://[PROJECT].supabase.co
SUPABASE_ANON_KEY=[ANON_KEY]

# =============================================================================
# GOOGLE CLOUD STORAGE CONFIGURATION
# =============================================================================
# TODO: Update project ID from setup output above
GCS_PROJECT_ID=[PROJECT_ID]
GCS_BUCKET_NAME=${PROJECT_PREFIX}-${ENVIRONMENT}
GOOGLE_APPLICATION_CREDENTIALS=./${PROJECT_PREFIX}-${ENVIRONMENT}-key.json

# =============================================================================
# CONFLUENT CLOUD CONFIGURATION
# =============================================================================
# TODO: Update these values from setup output above
CONFLUENT_BOOTSTRAP_SERVERS=[BOOTSTRAP_SERVERS]
CONFLUENT_API_KEY=[API_KEY]
CONFLUENT_API_SECRET=[API_SECRET]
KAFKA_TOPIC=posts.events

# =============================================================================
# AUTHENTICATION & SECURITY
# =============================================================================
JWT_SECRET=$(openssl rand -base64 32)
JWT_EXPIRY=24h

# =============================================================================
# FEATURE FLAGS
# =============================================================================
FEATURE_ANALYTICS_ENABLED=true
FEATURE_REAL_TIME_UPDATES=true
FEATURE_IMAGE_OPTIMIZATION=true
FEATURE_THUMBNAIL_GENERATION=true

EOF

    log_success "Generated environment file: ${env_file}"
    log_warning "Please update the TODO values in ${env_file} with actual configuration!"
}

# Main execution
main() {
    log_info "Setting up cloud infrastructure for environment: ${ENVIRONMENT}"

    check_prerequisites

    echo
    log_info "Starting infrastructure setup..."

    # Setup cloud services
    setup_gcs
    setup_confluent
    setup_supabase

    # Generate environment file
    generate_env_file

    echo
    log_success "Cloud infrastructure setup completed!"
    log_info "Next steps:"
    echo "  1. Complete manual Supabase setup (see instructions above)"
    echo "  2. Update .env.${ENVIRONMENT} with actual values"
    echo "  3. Run database migrations: make migrate-up"
    echo "  4. Deploy the service: ./scripts/deploy-cloud.sh ${ENVIRONMENT}"
    echo
    log_warning "Remember to:"
    echo "  - Store service account keys securely"
    echo "  - Add .env.${ENVIRONMENT} to .gitignore"
    echo "  - Use proper secret management in production"
}

# Run main function
main "$@"