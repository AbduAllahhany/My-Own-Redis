#!/bin/bash

# Redis Extensive Test Runner
# This script runs all extensive Redis tests with proper configuration

set -e

echo "🔧 Redis Extensive Test Suite"
echo "=============================="

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Function to print colored output
print_status() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[PASS]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

print_error() {
    echo -e "${RED}[FAIL]${NC} $1"
}

# Change to project directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
cd "$PROJECT_ROOT"

print_status "Current directory: $(pwd)"

# Check if Go is installed
if ! command -v go &> /dev/null; then
    print_error "Go is not installed or not in PATH"
    exit 1
fi

print_status "Go version: $(go version)"

# Check if go.mod exists
if [ ! -f "go.mod" ]; then
    print_error "go.mod not found. Make sure you're in the correct directory."
    exit 1
fi

# Create coverage directory if it doesn't exist
mkdir -p coverage_results

# Function to run a specific test suite
run_test_suite() {
    local test_path="$1"
    local test_name="$2"
    local timeout="${3:-30s}"
    
    print_status "Running $test_name tests..."
    
    if go test -timeout="$timeout" -v "$test_path" 2>&1 | tee "coverage_results/${test_name}_test.log"; then
        print_success "$test_name tests completed successfully"
        return 0
    else
        print_error "$test_name tests failed"
        return 1
    fi
}

# Function to run tests with coverage
run_with_coverage() {
    local test_path="$1"
    local test_name="$2"
    local timeout="${3:-30s}"
    local coverage_file="coverage_results/${test_name}_coverage.out"
    
    print_status "Running $test_name tests with coverage..."
    
    if go test -timeout="$timeout" -v -coverprofile="$coverage_file" "$test_path" 2>&1 | tee "coverage_results/${test_name}_test.log"; then
        print_success "$test_name tests completed successfully"
        
        # Generate coverage report
        if [ -f "$coverage_file" ]; then
            coverage_percent=$(go tool cover -func="$coverage_file" | grep total | awk '{print $3}')
            print_status "$test_name coverage: $coverage_percent"
        fi
        return 0
    else
        print_error "$test_name tests failed"
        return 1
    fi
}

# Track test results
total_tests=0
passed_tests=0

echo ""
print_status "Starting extensive Redis tests..."
echo ""

# Test 1: Comprehensive Redis Tests
total_tests=$((total_tests + 1))
if run_with_coverage "./tests/extensive" "comprehensive" "60s"; then
    passed_tests=$((passed_tests + 1))
fi

echo ""

# Test 2: Core functionality (if exists)
if [ -d "tests/unit" ]; then
    total_tests=$((total_tests + 1))
    if run_test_suite "./tests/unit" "unit" "30s"; then
        passed_tests=$((passed_tests + 1))
    fi
    echo ""
fi

# Test 3: Integration tests (if exists)
if [ -d "tests/integration" ]; then
    total_tests=$((total_tests + 1))
    if run_test_suite "./tests/integration" "integration" "60s"; then
        passed_tests=$((passed_tests + 1))
    fi
    echo ""
fi

# Test 4: Benchmarks (quick run)
if [ -d "tests/benchmarks" ]; then
    total_tests=$((total_tests + 1))
    print_status "Running benchmark tests (short mode)..."
    if go test -short -bench=. -benchtime=1s "./tests/benchmarks" 2>&1 | tee "coverage_results/benchmark_test.log"; then
        print_success "Benchmark tests completed successfully"
        passed_tests=$((passed_tests + 1))
    else
        print_error "Benchmark tests failed"
    fi
    echo ""
fi

# Summary
echo "=============================="
print_status "Test Summary"
echo "=============================="
print_status "Total test suites: $total_tests"
print_success "Passed: $passed_tests"

if [ $passed_tests -lt $total_tests ]; then
    failed_tests=$((total_tests - passed_tests))
    print_error "Failed: $failed_tests"
fi

echo ""

# Generate combined coverage report if available
if ls coverage_results/*_coverage.out 1> /dev/null 2>&1; then
    print_status "Generating combined coverage report..."
    
    # Combine coverage files
    echo "mode: set" > coverage_results/combined_coverage.out
    for f in coverage_results/*_coverage.out; do
        if [ -f "$f" ]; then
            tail -n +2 "$f" >> coverage_results/combined_coverage.out
        fi
    done
    
    # Generate HTML coverage report
    go tool cover -html=coverage_results/combined_coverage.out -o coverage_results/coverage.html
    print_success "Coverage report generated: coverage_results/coverage.html"
    
    # Show overall coverage
    overall_coverage=$(go tool cover -func=coverage_results/combined_coverage.out | grep total | awk '{print $3}')
    print_status "Overall coverage: $overall_coverage"
fi

echo ""

# Performance summary
if [ -f "coverage_results/comprehensive_test.log" ]; then
    print_status "Performance Summary:"
    grep -E "(PASS|FAIL).*[0-9]+\.[0-9]+s" coverage_results/comprehensive_test.log | tail -5
fi

echo ""

# Exit with appropriate code
if [ $passed_tests -eq $total_tests ]; then
    print_success "All test suites passed! 🎉"
    exit 0
else
    print_error "Some test suites failed. Check logs for details."
    exit 1
fi
