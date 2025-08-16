#!/bin/bash

# Unit Test Runner - Redis Implementation
# Professional unit test execution with detailed reporting

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
MAGENTA='\033[0;35m'
CYAN='\033[0;36m'
BOLD='\033[1m'
NC='\033[0m' # No Color

# Script configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"

# Default options
VERBOSE=false
COVERAGE=false
RACE=false
SHORT=false
PACKAGE_FILTER=""
TEST_FILTER=""
TIMEOUT="30s"
PARALLEL=false

print_header() {
    echo -e "${CYAN}╔══════════════════════════════════════════════════════════════════════════╗${NC}"
    echo -e "${CYAN}║${NC} ${BOLD}$1${NC} ${CYAN}║${NC}"
    echo -e "${CYAN}╚══════════════════════════════════════════════════════════════════════════╝${NC}"
}

print_success() {
    echo -e "${GREEN}✓${NC} $1"
}

print_error() {
    echo -e "${RED}✗${NC} $1"
}

print_info() {
    echo -e "${CYAN}ℹ${NC} $1"
}

show_usage() {
    echo -e "${BOLD}Unit Test Runner${NC}"
    echo ""
    echo "Usage: $0 [OPTIONS]"
    echo ""
    echo "Options:"
    echo "  -v, --verbose      Enable verbose output"
    echo "  -c, --coverage     Generate coverage report"
    echo "  -r, --race         Enable race condition detection"
    echo "  -s, --short        Run tests in short mode"
    echo "  -p, --package      Filter by package pattern (e.g., 'engine')"
    echo "  -t, --test         Filter by test name pattern"
    echo "  --timeout          Set test timeout (default: 30s)"
    echo "  --parallel         Run tests in parallel"
    echo "  -h, --help         Show this help message"
    echo ""
    echo "Examples:"
    echo "  $0                           # Run all unit tests"
    echo "  $0 -v -c                     # Verbose with coverage"
    echo "  $0 -p engine                 # Test only engine package"
    echo "  $0 -t TestParseCommand       # Test specific function"
    echo "  $0 --race --timeout 60s      # With race detection and custom timeout"
}

parse_arguments() {
    while [[ $# -gt 0 ]]; do
        case $1 in
            -v|--verbose)
                VERBOSE=true
                shift
                ;;
            -c|--coverage)
                COVERAGE=true
                shift
                ;;
            -r|--race)
                RACE=true
                shift
                ;;
            -s|--short)
                SHORT=true
                shift
                ;;
            -p|--package)
                PACKAGE_FILTER="$2"
                shift 2
                ;;
            -t|--test)
                TEST_FILTER="$2"
                shift 2
                ;;
            --timeout)
                TIMEOUT="$2"
                shift 2
                ;;
            --parallel)
                PARALLEL=true
                shift
                ;;
            -h|--help)
                show_usage
                exit 0
                ;;
            *)
                echo -e "${RED}Unknown option: $1${NC}"
                show_usage
                exit 1
                ;;
        esac
    done
}

build_test_command() {
    local cmd="go test"
    
    if [ "$VERBOSE" = true ]; then
        cmd="$cmd -v"
    fi
    
    if [ "$RACE" = true ]; then
        cmd="$cmd -race"
    fi
    
    if [ "$SHORT" = true ]; then
        cmd="$cmd -short"
    fi
    
    if [ "$PARALLEL" = true ]; then
        cmd="$cmd -parallel 4"
    fi
    
    cmd="$cmd -timeout=$TIMEOUT"
    
    if [ "$COVERAGE" = true ]; then
        mkdir -p "$PROJECT_ROOT/coverage_results"
        cmd="$cmd -coverprofile=$PROJECT_ROOT/coverage_results/unit_coverage.out"
    fi
    
    if [ -n "$TEST_FILTER" ]; then
        cmd="$cmd -run=$TEST_FILTER"
    fi
    
    echo "$cmd"
}

discover_test_packages() {
    local packages=()
    
    if [ -n "$PACKAGE_FILTER" ]; then
        # Find packages matching the filter
        for dir in "$PROJECT_ROOT"/tests/unit/*/; do
            if [ -d "$dir" ]; then
                local pkg_name=$(basename "$dir")
                if [[ "$pkg_name" == *"$PACKAGE_FILTER"* ]]; then
                    packages+=("./tests/unit/$pkg_name")
                fi
            fi
        done
        
        # Also check direct files in unit directory
        if ls "$PROJECT_ROOT"/tests/unit/*"$PACKAGE_FILTER"*_test.go >/dev/null 2>&1; then
            packages+=("./tests/unit")
        fi
    else
        # All unit test packages
        packages+=("./tests/unit/...")
    fi
    
    echo "${packages[@]}"
}

run_unit_tests() {
    local cmd=$(build_test_command)
    local packages=($(discover_test_packages))
    
    if [ ${#packages[@]} -eq 0 ]; then
        print_error "No test packages found"
        if [ -n "$PACKAGE_FILTER" ]; then
            print_info "Package filter: $PACKAGE_FILTER"
        fi
        return 1
    fi
    
    print_info "Test packages: ${packages[*]}"
    print_info "Command: $cmd ${packages[*]}"
    
    local start_time=$(date +%s)
    
    if $cmd "${packages[@]}"; then
        local end_time=$(date +%s)
        local duration=$((end_time - start_time))
        print_success "Unit tests completed in ${duration}s"
        
        if [ "$COVERAGE" = true ] && [ -f "$PROJECT_ROOT/coverage_results/unit_coverage.out" ]; then
            generate_coverage_report
        fi
        
        return 0
    else
        local end_time=$(date +%s)
        local duration=$((end_time - start_time))
        print_error "Unit tests failed after ${duration}s"
        return 1
    fi
}

generate_coverage_report() {
    local coverage_file="$PROJECT_ROOT/coverage_results/unit_coverage.out"
    local html_file="$PROJECT_ROOT/coverage_results/unit_coverage.html"
    
    if [ -f "$coverage_file" ]; then
        # Generate HTML report
        go tool cover -html="$coverage_file" -o="$html_file"
        
        # Show coverage summary
        local coverage_summary=$(go tool cover -func="$coverage_file")
        echo ""
        print_header "Coverage Report"
        echo "$coverage_summary"
        
        local total_coverage=$(echo "$coverage_summary" | grep total | awk '{print $3}')
        print_success "Total coverage: $total_coverage"
        print_info "HTML report: $html_file"
    fi
}

show_test_summary() {
    echo ""
    print_header "Unit Test Summary"
    
    # Show available tests
    local test_files=($(find "$PROJECT_ROOT/tests/unit" -name "*_test.go" -type f))
    print_info "Found ${#test_files[@]} test files:"
    
    for file in "${test_files[@]}"; do
        local rel_path=${file#$PROJECT_ROOT/}
        local test_count=$(grep -c "^func Test" "$file" 2>/dev/null || echo "0")
        echo -e "  ${YELLOW}$rel_path${NC} ($test_count tests)"
    done
    
    echo ""
    print_info "Options used:"
    [ "$VERBOSE" = true ] && echo -e "  ${GREEN}✓${NC} Verbose output"
    [ "$COVERAGE" = true ] && echo -e "  ${GREEN}✓${NC} Coverage analysis"
    [ "$RACE" = true ] && echo -e "  ${GREEN}✓${NC} Race condition detection"
    [ "$SHORT" = true ] && echo -e "  ${GREEN}✓${NC} Short mode"
    [ "$PARALLEL" = true ] && echo -e "  ${GREEN}✓${NC} Parallel execution"
    [ -n "$PACKAGE_FILTER" ] && echo -e "  ${GREEN}✓${NC} Package filter: $PACKAGE_FILTER"
    [ -n "$TEST_FILTER" ] && echo -e "  ${GREEN}✓${NC} Test filter: $TEST_FILTER"
    echo -e "  ${GREEN}✓${NC} Timeout: $TIMEOUT"
}

main() {
    cd "$PROJECT_ROOT"
    
    parse_arguments "$@"
    
    print_header "Redis Unit Test Runner"
    show_test_summary
    
    echo ""
    print_header "Executing Unit Tests"
    
    if run_unit_tests; then
        echo ""
        print_success "All unit tests passed! 🎉"
        exit 0
    else
        echo ""
        print_error "Unit tests failed! ❌"
        exit 1
    fi
}

main "$@"
