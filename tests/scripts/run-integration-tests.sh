#!/bin/bash

# Integration Test Runner - Redis Implementation
# Professional integration test execution with environment management

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
SETUP_ENV=true
CLEANUP=true
RACE=false
SHORT=false
TIMEOUT="60s"
TEST_FILTER=""
PORT_BASE=6380
PARALLEL_SERVERS=false

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

print_warning() {
    echo -e "${YELLOW}⚠${NC} $1"
}

print_info() {
    echo -e "${CYAN}ℹ${NC} $1"
}

print_step() {
    echo -e "${BLUE}▶${NC} $1"
}

show_usage() {
    echo -e "${BOLD}Integration Test Runner${NC}"
    echo ""
    echo "Usage: $0 [OPTIONS]"
    echo ""
    echo "Options:"
    echo "  -v, --verbose         Enable verbose output"
    echo "  -r, --race            Enable race condition detection"
    echo "  -s, --short           Run tests in short mode"
    echo "  -t, --test            Filter by test name pattern"
    echo "  --timeout             Set test timeout (default: 60s)"
    echo "  --port                Base port for test servers (default: 6380)"
    echo "  --no-setup            Skip environment setup"
    echo "  --no-cleanup          Skip cleanup after tests"
    echo "  --parallel-servers    Allow parallel server instances"
    echo "  -h, --help            Show this help message"
    echo ""
    echo "Examples:"
    echo "  $0                           # Run all integration tests"
    echo "  $0 -v --race                 # Verbose with race detection"
    echo "  $0 -t TestServerIntegration  # Test specific integration"
    echo "  $0 --port 7000 --short       # Custom port in short mode"
}

parse_arguments() {
    while [[ $# -gt 0 ]]; do
        case $1 in
            -v|--verbose)
                VERBOSE=true
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
            -t|--test)
                TEST_FILTER="$2"
                shift 2
                ;;
            --timeout)
                TIMEOUT="$2"
                shift 2
                ;;
            --port)
                PORT_BASE="$2"
                shift 2
                ;;
            --no-setup)
                SETUP_ENV=false
                shift
                ;;
            --no-cleanup)
                CLEANUP=false
                shift
                ;;
            --parallel-servers)
                PARALLEL_SERVERS=true
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

check_port_available() {
    local port=$1
    if command -v ss >/dev/null 2>&1; then
        ! ss -tuln | grep -q ":$port "
    elif command -v netstat >/dev/null 2>&1; then
        ! netstat -tuln 2>/dev/null | grep -q ":$port "
    else
        # Fallback: try to connect
        ! timeout 1 bash -c "echo >/dev/tcp/localhost/$port" 2>/dev/null
    fi
}

find_available_ports() {
    local count=${1:-3}
    local ports=()
    local current_port=$PORT_BASE
    
    while [ ${#ports[@]} -lt $count ]; do
        if check_port_available $current_port; then
            ports+=($current_port)
        fi
        ((current_port++))
        
        # Safety check to avoid infinite loop
        if [ $current_port -gt $((PORT_BASE + 100)) ]; then
            print_error "Could not find $count available ports starting from $PORT_BASE"
            return 1
        fi
    done
    
    echo "${ports[@]}"
}

setup_test_environment() {
    if [ "$SETUP_ENV" = false ]; then
        return 0
    fi
    
    print_step "Setting up integration test environment"
    
    # Create temporary directories
    export TEST_TEMP_DIR=$(mktemp -d -t redis-integration-XXXXXX)
    export TEST_LOG_DIR="$TEST_TEMP_DIR/logs"
    export TEST_DATA_DIR="$TEST_TEMP_DIR/data"
    
    mkdir -p "$TEST_LOG_DIR" "$TEST_DATA_DIR"
    print_success "Created temporary directories: $TEST_TEMP_DIR"
    
    # Find available ports
    local available_ports=($(find_available_ports 5))
    if [ ${#available_ports[@]} -lt 5 ]; then
        print_error "Could not find enough available ports"
        return 1
    fi
    
    export TEST_REDIS_PORT=${available_ports[0]}
    export TEST_REDIS_PORT_2=${available_ports[1]}
    export TEST_REDIS_PORT_3=${available_ports[2]}
    export TEST_REPLICA_PORT=${available_ports[3]}
    export TEST_MASTER_PORT=${available_ports[4]}
    
    print_success "Allocated ports: ${available_ports[*]}"
    
    # Set other environment variables
    export TEST_REDIS_HOST="localhost"
    export TEST_TIMEOUT="30s"
    export TEST_MAX_CONNECTIONS="100"
    
    if [ "$SHORT" = true ]; then
        export TEST_SHORT_MODE="true"
        export TEST_TIMEOUT="10s"
    fi
    
    print_success "Environment setup completed"
}

cleanup_test_environment() {
    if [ "$CLEANUP" = false ]; then
        return 0
    fi
    
    print_step "Cleaning up integration test environment"
    
    # Kill any remaining test processes
    if [ -n "$TEST_TEMP_DIR" ] && [ -d "$TEST_TEMP_DIR" ]; then
        # Look for PID files and kill processes
        find "$TEST_TEMP_DIR" -name "*.pid" -type f | while read pidfile; do
            if [ -f "$pidfile" ]; then
                local pid=$(cat "$pidfile" 2>/dev/null)
                if [ -n "$pid" ] && kill -0 "$pid" 2>/dev/null; then
                    print_info "Killing process $pid"
                    kill "$pid" 2>/dev/null || true
                    sleep 1
                    kill -9 "$pid" 2>/dev/null || true
                fi
            fi
        done
        
        # Remove temporary directory
        rm -rf "$TEST_TEMP_DIR"
        print_success "Removed temporary directory: $TEST_TEMP_DIR"
    fi
    
    # Kill any processes using our test ports
    local test_ports=(
        "$TEST_REDIS_PORT" "$TEST_REDIS_PORT_2" "$TEST_REDIS_PORT_3"
        "$TEST_REPLICA_PORT" "$TEST_MASTER_PORT"
    )
    
    for port in "${test_ports[@]}"; do
        if [ -n "$port" ]; then
            local pids=$(lsof -ti:$port 2>/dev/null || true)
            if [ -n "$pids" ]; then
                print_info "Killing processes on port $port: $pids"
                echo "$pids" | xargs kill -9 2>/dev/null || true
            fi
        fi
    done
    
    print_success "Environment cleanup completed"
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
    
    cmd="$cmd -timeout=$TIMEOUT"
    
    if [ -n "$TEST_FILTER" ]; then
        cmd="$cmd -run=$TEST_FILTER"
    fi
    
    echo "$cmd"
}

validate_integration_setup() {
    print_step "Validating integration test setup"
    
    # Check if integration test directory exists
    if [ ! -d "$PROJECT_ROOT/tests/integration" ]; then
        print_error "Integration test directory not found: $PROJECT_ROOT/tests/integration"
        return 1
    fi
    
    # Check for test files
    local test_files=($(find "$PROJECT_ROOT/tests/integration" -name "*_test.go" -type f))
    if [ ${#test_files[@]} -eq 0 ]; then
        print_error "No integration test files found"
        return 1
    fi
    
    print_success "Found ${#test_files[@]} integration test files"
    
    # Check if main application can be built
    if ! go build -o /tmp/redis-test-build ./app >/dev/null 2>&1; then
        print_error "Failed to build main application"
        return 1
    fi
    
    rm -f /tmp/redis-test-build
    print_success "Application builds successfully"
    
    return 0
}

run_integration_tests() {
    local cmd=$(build_test_command)
    local test_package="./tests/integration/..."
    
    print_info "Command: $cmd $test_package"
    print_info "Environment variables:"
    [ -n "$TEST_REDIS_PORT" ] && print_info "  TEST_REDIS_PORT=$TEST_REDIS_PORT"
    [ -n "$TEST_TEMP_DIR" ] && print_info "  TEST_TEMP_DIR=$TEST_TEMP_DIR"
    [ -n "$TEST_TIMEOUT" ] && print_info "  TEST_TIMEOUT=$TEST_TIMEOUT"
    
    local start_time=$(date +%s)
    
    if $cmd $test_package; then
        local end_time=$(date +%s)
        local duration=$((end_time - start_time))
        print_success "Integration tests completed in ${duration}s"
        return 0
    else
        local end_time=$(date +%s)
        local duration=$((end_time - start_time))
        print_error "Integration tests failed after ${duration}s"
        return 1
    fi
}

show_test_summary() {
    echo ""
    print_header "Integration Test Summary"
    
    # Show available tests
    local test_files=($(find "$PROJECT_ROOT/tests/integration" -name "*_test.go" -type f))
    print_info "Found ${#test_files[@]} integration test files:"
    
    for file in "${test_files[@]}"; do
        local rel_path=${file#$PROJECT_ROOT/}
        local test_count=$(grep -c "^func Test" "$file" 2>/dev/null || echo "0")
        echo -e "  ${YELLOW}$rel_path${NC} ($test_count tests)"
    done
    
    echo ""
    print_info "Configuration:"
    [ "$VERBOSE" = true ] && echo -e "  ${GREEN}✓${NC} Verbose output"
    [ "$RACE" = true ] && echo -e "  ${GREEN}✓${NC} Race condition detection"
    [ "$SHORT" = true ] && echo -e "  ${GREEN}✓${NC} Short mode"
    [ "$SETUP_ENV" = true ] && echo -e "  ${GREEN}✓${NC} Environment setup"
    [ "$CLEANUP" = true ] && echo -e "  ${GREEN}✓${NC} Automatic cleanup"
    [ -n "$TEST_FILTER" ] && echo -e "  ${GREEN}✓${NC} Test filter: $TEST_FILTER"
    echo -e "  ${GREEN}✓${NC} Timeout: $TIMEOUT"
    echo -e "  ${GREEN}✓${NC} Base port: $PORT_BASE"
}

trap_cleanup() {
    echo ""
    print_warning "Received interrupt signal, cleaning up..."
    cleanup_test_environment
    exit 1
}

main() {
    cd "$PROJECT_ROOT"
    
    # Set up signal handlers
    trap trap_cleanup INT TERM
    
    parse_arguments "$@"
    
    print_header "Redis Integration Test Runner"
    show_test_summary
    
    echo ""
    if ! validate_integration_setup; then
        exit 1
    fi
    
    echo ""
    if ! setup_test_environment; then
        exit 1
    fi
    
    echo ""
    print_header "Executing Integration Tests"
    
    local success=false
    if run_integration_tests; then
        success=true
    fi
    
    echo ""
    cleanup_test_environment
    
    if [ "$success" = true ]; then
        print_success "All integration tests passed! 🎉"
        exit 0
    else
        print_error "Integration tests failed! ❌"
        exit 1
    fi
}

main "$@"
