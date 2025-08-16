#!/bin/bash

# Redis Test Runner - Professional Testing Suite
# Main entry point for all Redis implementation tests

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
SCRIPTS_DIR="$PROJECT_ROOT/tests/scripts"

# Function to print colored output
print_banner() {
    echo -e "${CYAN}╔══════════════════════════════════════════════════════════════════════════╗${NC}"
    echo -e "${CYAN}║${NC} ${MAGENTA}${BOLD}$1${NC} ${CYAN}║${NC}"
    echo -e "${CYAN}╚══════════════════════════════════════════════════════════════════════════╝${NC}"
}

print_header() {
    echo -e "${BLUE}▶${NC} ${BOLD}$1${NC}"
}

print_success() {
    echo -e "${GREEN}✓${NC} $1"
}

print_error() {
    echo -e "${RED}✗${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}⚠${NC} $1"
}

print_info() {
    echo -e "${CYAN}ℹ${NC} $1"
}

print_separator() {
    echo -e "${CYAN}────────────────────────────────────────────────────────────────────────────${NC}"
}

# Function to show usage
show_usage() {
    print_banner "Redis Test Runner - Usage Guide"
    echo ""
    echo -e "${BOLD}USAGE:${NC}"
    echo "  $0 [COMMAND] [OPTIONS]"
    echo ""
    echo -e "${BOLD}COMMANDS:${NC}"
    echo -e "  ${GREEN}all${NC}              Run complete test suite (unit + integration + extensive + benchmarks)"
    echo -e "  ${GREEN}unit${NC}             Run unit tests only"
    echo -e "  ${GREEN}integration${NC}      Run integration tests only"
    echo -e "  ${GREEN}extensive${NC}       Run extensive test suite"
    echo -e "  ${GREEN}benchmarks${NC}      Run performance benchmarks"
    echo -e "  ${GREEN}stress${NC}          Run stress tests"
    echo -e "  ${GREEN}quick${NC}           Run quick test suite (unit + basic integration)"
    echo -e "  ${GREEN}coverage${NC}        Run tests with coverage analysis"
    echo -e "  ${GREEN}ci${NC}              Run CI/CD pipeline tests"
    echo -e "  ${GREEN}validate${NC}        Validate test environment and dependencies"
    echo ""
    echo -e "${BOLD}OPTIONS:${NC}"
    echo -e "  ${YELLOW}-v, --verbose${NC}    Enable verbose output"
    echo -e "  ${YELLOW}-q, --quiet${NC}      Enable quiet mode (minimal output)"
    echo -e "  ${YELLOW}-f, --fail-fast${NC}  Stop on first failure"
    echo -e "  ${YELLOW}-t, --timeout${NC}    Set test timeout (default: 60s)"
    echo -e "  ${YELLOW}-p, --parallel${NC}   Run tests in parallel where possible"
    echo -e "  ${YELLOW}-c, --clean${NC}      Clean previous test artifacts"
    echo -e "  ${YELLOW}--short${NC}          Run tests in short mode"
    echo -e "  ${YELLOW}--race${NC}           Enable race condition detection"
    echo -e "  ${YELLOW}--profile${NC}        Enable CPU/memory profiling"
    echo ""
    echo -e "${BOLD}EXAMPLES:${NC}"
    echo "  $0 all                    # Run complete test suite"
    echo "  $0 unit --verbose         # Run unit tests with verbose output"
    echo "  $0 stress --short         # Run stress tests in short mode"
    echo "  $0 benchmarks --profile   # Run benchmarks with profiling"
    echo "  $0 quick --race           # Run quick tests with race detection"
    echo ""
    echo -e "${BOLD}ENVIRONMENT VARIABLES:${NC}"
    echo "  TEST_TIMEOUT             Test timeout in seconds (default: 60)"
    echo "  TEST_VERBOSE             Enable verbose mode (true/false)"
    echo "  TEST_PARALLEL            Enable parallel execution (true/false)"
    echo ""
}

# Function to validate environment
validate_environment() {
    print_header "Validating Test Environment"
    
    # Check if we're in the right directory
    if [ ! -f "$PROJECT_ROOT/go.mod" ]; then
        print_error "go.mod not found. Please run from the project root directory."
        exit 1
    fi
    
    # Check Go installation
    if ! command -v go &> /dev/null; then
        print_error "Go is not installed or not in PATH"
        exit 1
    fi
    
    local go_version=$(go version)
    print_success "Go installation: $go_version"
    
    # Check project structure
    local required_dirs=("tests/unit" "tests/integration" "tests/extensive" "tests/benchmarks")
    for dir in "${required_dirs[@]}"; do
        if [ -d "$PROJECT_ROOT/$dir" ]; then
            print_success "Directory exists: $dir"
        else
            print_warning "Directory missing: $dir"
        fi
    done
    
    # Check scripts
    local scripts=("run_extensive_tests.sh" "run_stress_tests.sh")
    for script in "${scripts[@]}"; do
        if [ -x "$SCRIPTS_DIR/$script" ]; then
            print_success "Script available: $script"
        else
            print_warning "Script missing or not executable: $script"
        fi
    done
    
    print_success "Environment validation completed"
}

# Function to parse command line arguments
parse_arguments() {
    COMMAND=""
    VERBOSE=false
    QUIET=false
    FAIL_FAST=false
    TIMEOUT="60s"
    PARALLEL=false
    CLEAN=false
    SHORT=false
    RACE=false
    PROFILE=false
    
    while [[ $# -gt 0 ]]; do
        case $1 in
            all|unit|integration|extensive|benchmarks|stress|quick|coverage|ci|validate)
                COMMAND="$1"
                shift
                ;;
            -v|--verbose)
                VERBOSE=true
                shift
                ;;
            -q|--quiet)
                QUIET=true
                shift
                ;;
            -f|--fail-fast)
                FAIL_FAST=true
                shift
                ;;
            -t|--timeout)
                TIMEOUT="$2"
                shift 2
                ;;
            -p|--parallel)
                PARALLEL=true
                shift
                ;;
            -c|--clean)
                CLEAN=true
                shift
                ;;
            --short)
                SHORT=true
                shift
                ;;
            --race)
                RACE=true
                shift
                ;;
            --profile)
                PROFILE=true
                shift
                ;;
            -h|--help)
                show_usage
                exit 0
                ;;
            *)
                print_error "Unknown option: $1"
                show_usage
                exit 1
                ;;
        esac
    done
    
    if [ -z "$COMMAND" ]; then
        print_error "No command specified"
        show_usage
        exit 1
    fi
}

# Function to build test flags
build_test_flags() {
    local flags=""
    
    if [ "$VERBOSE" = true ]; then
        flags="$flags -v"
    fi
    
    if [ "$SHORT" = true ]; then
        flags="$flags -short"
    fi
    
    if [ "$RACE" = true ]; then
        flags="$flags -race"
    fi
    
    if [ "$FAIL_FAST" = true ]; then
        flags="$flags -failfast"
    fi
    
    flags="$flags -timeout=$TIMEOUT"
    
    echo "$flags"
}

# Function to clean test artifacts
clean_artifacts() {
    if [ "$CLEAN" = true ]; then
        print_header "Cleaning Test Artifacts"
        rm -rf "$PROJECT_ROOT/coverage_results/"*.out
        rm -rf "$PROJECT_ROOT/coverage_results/"*.html
        rm -rf "$PROJECT_ROOT/coverage_results/"*.log
        rm -rf "$PROJECT_ROOT"/*.prof
        print_success "Test artifacts cleaned"
    fi
}

# Function to create coverage directory
ensure_coverage_dir() {
    mkdir -p "$PROJECT_ROOT/coverage_results"
}

# Function to run unit tests
run_unit_tests() {
    print_header "Running Unit Tests"
    local flags=$(build_test_flags)
    
    if go test $flags ./tests/unit/...; then
        print_success "Unit tests passed"
        return 0
    else
        print_error "Unit tests failed"
        return 1
    fi
}

# Function to run integration tests
run_integration_tests() {
    print_header "Running Integration Tests"
    local flags=$(build_test_flags)
    
    if go test $flags ./tests/integration/...; then
        print_success "Integration tests passed"
        return 0
    else
        print_error "Integration tests failed"
        return 1
    fi
}

# Function to run extensive tests
run_extensive_tests() {
    print_header "Running Extensive Test Suite"
    
    if [ -x "$SCRIPTS_DIR/run_extensive_tests.sh" ]; then
        if bash "$SCRIPTS_DIR/run_extensive_tests.sh"; then
            print_success "Extensive tests passed"
            return 0
        else
            print_error "Extensive tests failed"
            return 1
        fi
    else
        print_error "Extensive test script not found"
        return 1
    fi
}

# Function to run benchmarks
run_benchmarks() {
    print_header "Running Performance Benchmarks"
    local flags=""
    local benchtime="3s"
    
    if [ "$SHORT" = true ]; then
        benchtime="1s"
    fi
    
    if [ "$VERBOSE" = true ]; then
        flags="$flags -v"
    fi
    
    if go test $flags -bench=. -benchtime=$benchtime ./tests/benchmarks/...; then
        print_success "Benchmarks completed"
        return 0
    else
        print_error "Benchmarks failed"
        return 1
    fi
}

# Function to run stress tests
run_stress_tests() {
    print_header "Running Stress Tests"
    
    if [ -x "$SCRIPTS_DIR/run_stress_tests.sh" ]; then
        local args=""
        if [ "$SHORT" = false ]; then
            args="--full"
        fi
        
        if bash "$SCRIPTS_DIR/run_stress_tests.sh" $args; then
            print_success "Stress tests passed"
            return 0
        else
            print_error "Stress tests failed"
            return 1
        fi
    else
        print_error "Stress test script not found"
        return 1
    fi
}

# Function to run quick tests
run_quick_tests() {
    print_header "Running Quick Test Suite"
    
    local success=true
    
    run_unit_tests || success=false
    
    if [ "$success" = true ]; then
        print_header "Running Basic Integration Tests"
        local flags=$(build_test_flags)
        flags="$flags -short"
        
        if go test $flags ./tests/integration/...; then
            print_success "Quick integration tests passed"
        else
            print_error "Quick integration tests failed"
            success=false
        fi
    fi
    
    if [ "$success" = true ]; then
        print_success "Quick test suite passed"
        return 0
    else
        print_error "Quick test suite failed"
        return 1
    fi
}

# Function to run coverage analysis
run_coverage_tests() {
    print_header "Running Coverage Analysis"
    ensure_coverage_dir
    
    local coverage_file="$PROJECT_ROOT/coverage_results/coverage.out"
    local html_file="$PROJECT_ROOT/coverage_results/coverage.html"
    
    if go test -coverprofile="$coverage_file" ./...; then
        if [ -f "$coverage_file" ]; then
            # Generate HTML report
            go tool cover -html="$coverage_file" -o="$html_file"
            
            # Show coverage summary
            local coverage=$(go tool cover -func="$coverage_file" | grep total | awk '{print $3}')
            print_success "Coverage analysis completed: $coverage"
            print_info "HTML report: $html_file"
            return 0
        else
            print_error "Coverage file not generated"
            return 1
        fi
    else
        print_error "Coverage tests failed"
        return 1
    fi
}

# Function to run CI tests
run_ci_tests() {
    print_header "Running CI/CD Pipeline Tests"
    
    # Set CI-specific flags
    RACE=true
    FAIL_FAST=true
    TIMEOUT="120s"
    
    local success=true
    
    # Run validation
    validate_environment || success=false
    
    # Run unit tests with race detection
    if [ "$success" = true ]; then
        run_unit_tests || success=false
    fi
    
    # Run integration tests
    if [ "$success" = true ]; then
        run_integration_tests || success=false
    fi
    
    # Run short benchmarks
    if [ "$success" = true ]; then
        SHORT=true
        run_benchmarks || success=false
    fi
    
    if [ "$success" = true ]; then
        print_success "CI/CD pipeline tests passed"
        return 0
    else
        print_error "CI/CD pipeline tests failed"
        return 1
    fi
}

# Function to run all tests
run_all_tests() {
    print_header "Running Complete Test Suite"
    
    local success=true
    local start_time=$(date +%s)
    
    # Clean artifacts if requested
    clean_artifacts
    
    # Run tests in order
    run_unit_tests || success=false
    print_separator
    
    run_integration_tests || success=false
    print_separator
    
    run_extensive_tests || success=false
    print_separator
    
    run_benchmarks || success=false
    print_separator
    
    if [ "$SHORT" = false ]; then
        run_stress_tests || success=false
        print_separator
    fi
    
    local end_time=$(date +%s)
    local duration=$((end_time - start_time))
    
    if [ "$success" = true ]; then
        print_success "Complete test suite passed in ${duration}s"
        return 0
    else
        print_error "Complete test suite failed after ${duration}s"
        return 1
    fi
}

# Main execution
main() {
    cd "$PROJECT_ROOT"
    
    parse_arguments "$@"
    
    print_banner "Redis Implementation Test Runner"
    print_info "Project: $(basename "$PROJECT_ROOT")"
    print_info "Command: $COMMAND"
    
    if [ "$QUIET" = false ]; then
        print_info "Go version: $(go version | awk '{print $3}')"
        print_info "Test timeout: $TIMEOUT"
        [ "$VERBOSE" = true ] && print_info "Verbose mode: enabled"
        [ "$RACE" = true ] && print_info "Race detection: enabled"
        [ "$SHORT" = true ] && print_info "Short mode: enabled"
    fi
    
    print_separator
    
    case $COMMAND in
        validate)
            validate_environment
            ;;
        unit)
            run_unit_tests
            ;;
        integration)
            run_integration_tests
            ;;
        extensive)
            run_extensive_tests
            ;;
        benchmarks)
            run_benchmarks
            ;;
        stress)
            run_stress_tests
            ;;
        quick)
            run_quick_tests
            ;;
        coverage)
            run_coverage_tests
            ;;
        ci)
            run_ci_tests
            ;;
        all)
            run_all_tests
            ;;
        *)
            print_error "Unknown command: $COMMAND"
            show_usage
            exit 1
            ;;
    esac
    
    local exit_code=$?
    
    print_separator
    if [ $exit_code -eq 0 ]; then
        print_success "Test execution completed successfully! 🎉"
    else
        print_error "Test execution failed! ❌"
    fi
    
    exit $exit_code
}

# Run main function with all arguments
main "$@"
