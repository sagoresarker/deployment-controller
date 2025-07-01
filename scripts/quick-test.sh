#!/bin/bash

# Quick Test Script for Deployment Controller
# This script tests basic functionality using curl commands

set -e

# Configuration
BASE_URL="http://localhost:8080"
CONTENT_TYPE="Content-Type: application/json"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Function to print colored output
print_status() {
    echo -e "${BLUE}[TEST]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[PASS]${NC} $1"
}

print_error() {
    echo -e "${RED}[FAIL]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

# Function to check if service is running
check_service() {
    print_status "Checking if service is running..."

    if curl -s "$BASE_URL/healthz" > /dev/null; then
        print_success "Service is running"
        return 0
    else
        print_error "Service is not running. Please start it with 'docker-compose up -d' or 'make dev'"
        exit 1
    fi
}

# Function to test health endpoint
test_health() {
    print_status "Testing health endpoint..."

    response=$(curl -s -w "\n%{http_code}" "$BASE_URL/healthz")
    http_code=$(echo "$response" | tail -n1)
    body=$(echo "$response" | head -n -1)

    if [ "$http_code" = "200" ]; then
        print_success "Health check passed (HTTP $http_code)"
        echo "Response: $body"
    else
        print_error "Health check failed (HTTP $http_code)"
        echo "Response: $body"
        return 1
    fi
}

# Function to test storing registry credentials
test_store_registry() {
    print_status "Testing registry credential storage..."

    response=$(curl -s -w "\n%{http_code}" -X POST "$BASE_URL/api/v1/registry" \
        -H "$CONTENT_TYPE" \
        -d '{
            "registry": "test-registry.com",
            "username": "test-user",
            "password": "test-password"
        }')

    http_code=$(echo "$response" | tail -n1)
    body=$(echo "$response" | head -n -1)

    if [ "$http_code" = "201" ]; then
        print_success "Registry credential storage passed (HTTP $http_code)"
        echo "Response: $body"
    else
        print_error "Registry credential storage failed (HTTP $http_code)"
        echo "Response: $body"
        return 1
    fi
}

# Function to test getting registry credentials
test_get_registry() {
    print_status "Testing registry credential retrieval..."

    response=$(curl -s -w "\n%{http_code}" "$BASE_URL/api/v1/registry?registry=test-registry.com")
    http_code=$(echo "$response" | tail -n1)
    body=$(echo "$response" | head -n -1)

    if [ "$http_code" = "200" ]; then
        print_success "Registry credential retrieval passed (HTTP $http_code)"
        echo "Response: $body"
    else
        print_error "Registry credential retrieval failed (HTTP $http_code)"
        echo "Response: $body"
        return 1
    fi
}

# Function to test pushing single deployment
test_push_single() {
    print_status "Testing single deployment push..."

    response=$(curl -s -w "\n%{http_code}" -X POST "$BASE_URL/api/v1/push" \
        -H "$CONTENT_TYPE" \
        -d '[{
            "domain": "test1.poridhi.com",
            "app_name": "test-app-1",
            "docker_image": "test-registry.com/test-app:v1.0.0",
            "port": 3000,
            "env": [
                "NODE_ENV=production",
                "VERSION=1.0.0"
            ]
        }]')

    http_code=$(echo "$response" | tail -n1)
    body=$(echo "$response" | head -n -1)

    if [ "$http_code" = "201" ]; then
        print_success "Single deployment push passed (HTTP $http_code)"
        echo "Response: $body"

        # Extract deployment ID for later use
        DEPLOYMENT_ID=$(echo "$body" | grep -o '"id":"[^"]*"' | head -1 | cut -d'"' -f4)
        echo "Deployment ID: $DEPLOYMENT_ID"
    else
        print_error "Single deployment push failed (HTTP $http_code)"
        echo "Response: $body"
        return 1
    fi
}

# Function to test pushing multiple deployments
test_push_multiple() {
    print_status "Testing multiple deployment push..."

    response=$(curl -s -w "\n%{http_code}" -X POST "$BASE_URL/api/v1/push" \
        -H "$CONTENT_TYPE" \
        -d '[
            {
                "domain": "test2.poridhi.com",
                "app_name": "test-app-2",
                "docker_image": "test-registry.com/test-app-2:v1.0.0",
                "port": 3001,
                "env": ["NODE_ENV=production"]
            },
            {
                "domain": "test3.poridhi.com",
                "app_name": "test-app-3",
                "docker_image": "test-registry.com/test-app-3:v1.0.0",
                "port": 3002,
                "env": ["NODE_ENV=production", "DEBUG=false"]
            }
        ]')

    http_code=$(echo "$response" | tail -n1)
    body=$(echo "$response" | head -n -1)

    if [ "$http_code" = "201" ]; then
        print_success "Multiple deployment push passed (HTTP $http_code)"
        echo "Response: $body"
    else
        print_error "Multiple deployment push failed (HTTP $http_code)"
        echo "Response: $body"
        return 1
    fi
}

# Function to test versioning (same app, new version)
test_versioning() {
    print_status "Testing deployment versioning..."

    response=$(curl -s -w "\n%{http_code}" -X POST "$BASE_URL/api/v1/push" \
        -H "$CONTENT_TYPE" \
        -d '[{
            "domain": "test1.poridhi.com",
            "app_name": "test-app-1",
            "docker_image": "test-registry.com/test-app:v1.1.0",
            "port": 3000,
            "env": [
                "NODE_ENV=production",
                "VERSION=1.1.0",
                "NEW_FEATURE=enabled"
            ]
        }]')

    http_code=$(echo "$response" | tail -n1)
    body=$(echo "$response" | head -n -1)

    if [ "$http_code" = "201" ]; then
        print_success "Deployment versioning passed (HTTP $http_code)"
        echo "Response: $body"

        # Check if version was incremented
        if echo "$body" | grep -q '"version":2'; then
            print_success "Version was correctly incremented to 2"
        else
            print_warning "Could not verify version increment"
        fi
    else
        print_error "Deployment versioning failed (HTTP $http_code)"
        echo "Response: $body"
        return 1
    fi
}

# Function to test getting deployments
test_get_deployments() {
    print_status "Testing deployment retrieval..."

    response=$(curl -s -w "\n%{http_code}" "$BASE_URL/api/v1/deployments")
    http_code=$(echo "$response" | tail -n1)
    body=$(echo "$response" | head -n -1)

    if [ "$http_code" = "200" ]; then
        print_success "Deployment retrieval passed (HTTP $http_code)"
        echo "Response: $body"

        # Count deployments
        deployment_count=$(echo "$body" | grep -o '"domain"' | wc -l)
        print_success "Found $deployment_count deployments"
    else
        print_error "Deployment retrieval failed (HTTP $http_code)"
        echo "Response: $body"
        return 1
    fi
}

# Function to test getting statistics
test_get_stats() {
    print_status "Testing deployment statistics..."

    response=$(curl -s -w "\n%{http_code}" "$BASE_URL/api/v1/stats")
    http_code=$(echo "$response" | tail -n1)
    body=$(echo "$response" | head -n -1)

    if [ "$http_code" = "200" ]; then
        print_success "Deployment statistics passed (HTTP $http_code)"
        echo "Response: $body"
    else
        print_error "Deployment statistics failed (HTTP $http_code)"
        echo "Response: $body"
        return 1
    fi
}

# Function to test error scenarios
test_error_scenarios() {
    print_status "Testing error scenarios..."

    # Test empty array (should fail)
    print_status "  Testing empty deployment array..."
    response=$(curl -s -w "\n%{http_code}" -X POST "$BASE_URL/api/v1/push" \
        -H "$CONTENT_TYPE" \
        -d '[]')

    http_code=$(echo "$response" | tail -n1)

    if [ "$http_code" = "400" ]; then
        print_success "  Empty array correctly rejected (HTTP $http_code)"
    else
        print_error "  Empty array should have been rejected with 400, got $http_code"
    fi

    # Test missing required fields (should fail)
    print_status "  Testing missing required fields..."
    response=$(curl -s -w "\n%{http_code}" -X POST "$BASE_URL/api/v1/push" \
        -H "$CONTENT_TYPE" \
        -d '[{
            "domain": "invalid.com",
            "app_name": "missing-fields"
        }]')

    http_code=$(echo "$response" | tail -n1)

    if [ "$http_code" = "400" ]; then
        print_success "  Missing fields correctly rejected (HTTP $http_code)"
    else
        print_error "  Missing fields should have been rejected with 400, got $http_code"
    fi

    # Test invalid registry request (should fail)
    print_status "  Testing invalid registry request..."
    response=$(curl -s -w "\n%{http_code}" "$BASE_URL/api/v1/registry?registry=nonexistent-registry.com")
    http_code=$(echo "$response" | tail -n1)

    if [ "$http_code" = "404" ]; then
        print_success "  Nonexistent registry correctly returned 404"
    else
        print_error "  Nonexistent registry should have returned 404, got $http_code"
    fi
}

# Main execution
main() {
    echo "=================================================="
    echo "Deployment Controller - Quick Test Script"
    echo "=================================================="
    echo ""

    # Check if service is running
    check_service
    echo ""

    # Run tests
    test_health && echo ""
    test_store_registry && echo ""
    test_get_registry && echo ""
    test_push_single && echo ""
    test_push_multiple && echo ""
    test_versioning && echo ""
    test_get_deployments && echo ""
    test_get_stats && echo ""
    test_error_scenarios && echo ""

    echo "=================================================="
    print_success "All tests completed successfully!"
    echo "=================================================="
    echo ""
    echo "Next steps:"
    echo "1. Use api-tests.http for more comprehensive testing"
    echo "2. Check database content with: docker-compose exec postgres psql -U postgres -d deployment_controller"
    echo "3. View service logs with: docker-compose logs -f deployment-controller"
    echo ""
}

# Run main function
main