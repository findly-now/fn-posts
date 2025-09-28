#!/bin/bash

# =============================================================================
# Environment Validation Script
# =============================================================================
# This script validates the environment configuration for the Posts service
# to ensure all required cloud services are properly configured.
#
# Usage: ./scripts/validate-env.sh [environment]
# Environment: staging, production (default: staging)
# =============================================================================

set -euo pipefail

# Configuration
ENVIRONMENT=${1:-staging}
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "${SCRIPT_DIR}/.." && pwd)"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Counters
TESTS_PASSED=0
TESTS_FAILED=0
WARNINGS=0

# Logging functions
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[PASS]${NC} $1"
    ((TESTS_PASSED++))
}

log_warning() {
    echo -e "${YELLOW}[WARN]${NC} $1"
    ((WARNINGS++))
}

log_error() {
    echo -e "${RED}[FAIL]${NC} $1"
    ((TESTS_FAILED++))
}

# Test functions
test_env_file_exists() {
    local env_file=".env.${ENVIRONMENT}"

    if [[ -f "${ROOT_DIR}/${env_file}" ]]; then
        log_success "Environment file exists: ${env_file}"
        return 0
    else
        log_error "Environment file not found: ${env_file}"
        return 1
    fi
}

test_required_variables() {
    local env_file=".env.${ENVIRONMENT}"
    local required_vars=(
        "PORT"
        "ENVIRONMENT"
        "POSTGRES_URL"
        "BUCKET_PROJECT_ID"
        "BUCKET_NAME"
        "KAFKA_BOOTSTRAP_SERVERS"
        "KAFKA_API_KEY"
        "JWT_SECRET"
    )

    source "${ROOT_DIR}/${env_file}"

    local missing_vars=()

    for var in "${required_vars[@]}"; do
        if [[ -z "${!var:-}" ]]; then
            missing_vars+=("${var}")
        fi
    done

    if [[ ${#missing_vars[@]} -eq 0 ]]; then
        log_success "All required environment variables are set"
    else
        log_error "Missing required environment variables: ${missing_vars[*]}"
        return 1
    fi
}

test_cloud_services_configuration() {
    local env_file=".env.${ENVIRONMENT}"
    source "${ROOT_DIR}/${env_file}"

    # Test Google Cloud Storage configuration
    if [[ -z "${BUCKET_PROJECT_ID:-}" ]]; then
        log_error "BUCKET_PROJECT_ID is required for Google Cloud Storage"
        return 1
    fi
    if [[ -z "${BUCKET_NAME:-}" ]]; then
        log_error "BUCKET_NAME is required for Google Cloud Storage"
        return 1
    fi
    log_success "Google Cloud Storage configuration is valid"

    # Test Confluent Cloud Kafka configuration
    if [[ -z "${KAFKA_BOOTSTRAP_SERVERS:-}" ]]; then
        log_error "KAFKA_BOOTSTRAP_SERVERS is required for Confluent Cloud"
        return 1
    fi
    if [[ -z "${KAFKA_API_KEY:-}" ]]; then
        log_error "KAFKA_API_KEY is required for Confluent Cloud"
        return 1
    fi
    if [[ -z "${KAFKA_API_SECRET:-}" ]]; then
        log_error "KAFKA_API_SECRET is required for Confluent Cloud"
        return 1
    fi
    log_success "Confluent Cloud Kafka configuration is valid"

    # Test Supabase configuration
    if [[ -z "${POSTGRES_URL:-}" ]]; then
        log_error "POSTGRES_URL is required for Supabase"
        return 1
    fi
    log_success "Supabase PostgreSQL configuration is valid"
}

test_database_connection() {
    local env_file=".env.${ENVIRONMENT}"
    source "${ROOT_DIR}/${env_file}"

    if [[ "${DATABASE_URL}" == *"[PASSWORD]"* ]] || [[ "${DATABASE_URL}" == *"[PROJECT]"* ]]; then
        log_error "DATABASE_URL contains placeholder values - please update with actual values"
        return 1
    fi

    # Test database connection
    if command -v psql &> /dev/null; then
        if timeout 10 psql "${DATABASE_URL}" -c "SELECT version();" &> /dev/null; then
            log_success "Database connection successful"
        else
            log_error "Database connection failed"
            return 1
        fi
    else
        log_warning "psql not found - skipping database connection test"
    fi
}

test_gcs_configuration() {
    local env_file=".env.${ENVIRONMENT}"
    source "${ROOT_DIR}/${env_file}"

    # Check if service account key file exists
    if [[ -n "${GOOGLE_APPLICATION_CREDENTIALS:-}" ]]; then
        if [[ -f "${ROOT_DIR}/${GOOGLE_APPLICATION_CREDENTIALS}" ]]; then
            log_success "GCS service account key file found"
        else
            log_error "GCS service account key file not found: ${GOOGLE_APPLICATION_CREDENTIALS}"
            return 1
        fi
    fi

    # Test GCS access if gcloud is available
    if command -v gsutil &> /dev/null; then
        if gsutil ls "gs://${BUCKET_NAME}" &> /dev/null; then
            log_success "GCS bucket access successful"
        else
            log_error "GCS bucket access failed: ${BUCKET_NAME}"
            return 1
        fi
    else
        log_warning "gsutil not found - skipping GCS access test"
    fi
}

test_confluent_configuration() {
    local env_file=".env.${ENVIRONMENT}"
    source "${ROOT_DIR}/${env_file}"

    # Check placeholder values
    if [[ "${KAFKA_BOOTSTRAP_SERVERS}" == *"[BOOTSTRAP_SERVERS]"* ]]; then
        log_error "KAFKA_BOOTSTRAP_SERVERS contains placeholder values"
        return 1
    fi

    if [[ "${KAFKA_API_KEY}" == *"[API_KEY]"* ]]; then
        log_error "KAFKA_API_KEY contains placeholder values"
        return 1
    fi

    # Test Confluent Cloud access if confluent CLI is available
    if command -v confluent &> /dev/null; then
        # Set temporary config for testing
        export CONFLUENT_CLOUD_API_KEY="${KAFKA_API_KEY}"
        export CONFLUENT_CLOUD_API_SECRET="${KAFKA_API_SECRET}"

        if timeout 10 confluent kafka topic list &> /dev/null; then
            log_success "Confluent Cloud access successful"
        else
            log_error "Confluent Cloud access failed"
            return 1
        fi
    else
        log_warning "confluent CLI not found - skipping Confluent Cloud access test"
        fi
    fi
}

test_security_configuration() {
    local env_file=".env.${ENVIRONMENT}"
    source "${ROOT_DIR}/${env_file}"

    # Check JWT secret strength
    if [[ ${#JWT_SECRET} -lt 32 ]]; then
        log_warning "JWT_SECRET is less than 32 characters - consider using a stronger secret"
    else
        log_success "JWT_SECRET has adequate length"
    fi

    # Check for default/weak values
    if [[ "${JWT_SECRET}" == "your-secret-key" ]] || [[ "${JWT_SECRET}" == "your-secret-key-change-in-production" ]]; then
        log_error "JWT_SECRET is using default value - please generate a secure secret"
        return 1
    fi

    # Check environment-specific security
    if [[ "${ENVIRONMENT}" == "production" ]]; then
        if [[ "${LOG_LEVEL:-info}" == "debug" ]]; then
            log_warning "LOG_LEVEL is set to debug in production - consider using 'info' or 'warn'"
        fi

        if [[ "${FEATURE_ANALYTICS_ENABLED:-true}" != "true" ]]; then
            log_warning "Analytics disabled in production - consider enabling for monitoring"
        fi
    fi
}

test_feature_flags() {
    local env_file=".env.${ENVIRONMENT}"
    source "${ROOT_DIR}/${env_file}"

    local feature_flags=(
        "FEATURE_ANALYTICS_ENABLED"
        "FEATURE_REAL_TIME_UPDATES"
        "FEATURE_IMAGE_OPTIMIZATION"
        "FEATURE_THUMBNAIL_GENERATION"
    )

    for flag in "${feature_flags[@]}"; do
        local value="${!flag:-}"
        if [[ "${value}" == "true" ]] || [[ "${value}" == "false" ]]; then
            log_success "Feature flag ${flag} has valid value: ${value}"
        else
            log_warning "Feature flag ${flag} has invalid value: '${value}' (should be 'true' or 'false')"
        fi
    done
}

test_port_configuration() {
    local env_file=".env.${ENVIRONMENT}"
    source "${ROOT_DIR}/${env_file}"

    # Check if port is a valid number
    if [[ "${PORT}" =~ ^[0-9]+$ ]]; then
        if [[ ${PORT} -ge 1024 ]] && [[ ${PORT} -le 65535 ]]; then
            log_success "PORT configuration is valid: ${PORT}"
        else
            log_warning "PORT ${PORT} is outside recommended range (1024-65535)"
        fi
    else
        log_error "PORT is not a valid number: ${PORT}"
        return 1
    fi
}

# Summary function
print_summary() {
    echo
    echo "======================================"
    echo "Validation Summary"
    echo "======================================"
    echo -e "Environment: ${BLUE}${ENVIRONMENT}${NC}"
    echo -e "Tests Passed: ${GREEN}${TESTS_PASSED}${NC}"
    echo -e "Tests Failed: ${RED}${TESTS_FAILED}${NC}"
    echo -e "Warnings: ${YELLOW}${WARNINGS}${NC}"
    echo

    if [[ ${TESTS_FAILED} -eq 0 ]]; then
        log_success "All validation tests passed!"
        if [[ ${WARNINGS} -gt 0 ]]; then
            log_warning "Please review the warnings above"
        fi
        echo "The environment is ready for deployment."
        return 0
    else
        log_error "Some validation tests failed!"
        echo "Please fix the issues above before deployment."
        return 1
    fi
}

# Main execution
main() {
    log_info "Validating environment configuration: ${ENVIRONMENT}"
    echo

    # Run all tests
    test_env_file_exists || exit 1
    test_required_variables
    test_cloud_services_configuration
    test_database_connection
    test_gcs_configuration
    test_confluent_configuration
    test_security_configuration
    test_feature_flags
    test_port_configuration

    # Print summary and exit with appropriate code
    print_summary
}

# Show usage if help requested
if [[ "${1:-}" == "--help" ]] || [[ "${1:-}" == "-h" ]]; then
    cat <<EOF
Usage: $0 [environment]

Validates the environment configuration for the Posts service.

Environments:
  staging     Staging environment (default)
  production  Production environment

Tests performed:
  - Environment file exists
  - Required variables are set
  - Provider configuration is valid
  - Database connection works
  - Cloud service access is functional
  - Security configuration is adequate
  - Feature flags are properly configured

Examples:
  $0                # Validate staging environment
  $0 production     # Validate production environment

EOF
    exit 0
fi

# Run main function
main "$@"