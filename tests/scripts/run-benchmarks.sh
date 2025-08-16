#!/bin/bash

# Benchmark Test Runner - Redis Implementation
# Professional benchmark execution with performance analysis

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
BENCHTIME="3s"
COUNT=1
BENCH_FILTER=""
OUTPUT_FILE=""
MEMORY_PROFILE=false
CPU_PROFILE=false
COMPARE_MODE=false
BASELINE_FILE=""
SHORT=false
STRESS_ONLY=false

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
    echo -e "${BOLD}Benchmark Test Runner${NC}"
    echo ""
    echo "Usage: $0 [OPTIONS]"
    echo ""
    echo "Options:"
    echo "  -v, --verbose         Enable verbose output"
    echo "  -t, --benchtime       Duration for each benchmark (default: 3s)"
    echo "  -c, --count           Number of times to run each benchmark (default: 1)"
    echo "  -b, --bench           Filter benchmarks by pattern"
    echo "  -o, --output          Save results to file"
    echo "  --cpu-profile         Enable CPU profiling"
    echo "  --mem-profile         Enable memory profiling"
    echo "  --compare             Compare with baseline file"
    echo "  --baseline            Baseline file for comparison"
    echo "  --short               Run benchmarks in short mode"
    echo "  --stress-only         Run only stress benchmarks"
    echo "  -h, --help            Show this help message"
    echo ""
    echo "Examples:"
    echo "  $0                                    # Run all benchmarks"
    echo "  $0 -v -t 5s                          # Verbose, 5 second benchtime"
    echo "  $0 -b BenchmarkSet                   # Only SET benchmarks"
    echo "  $0 --cpu-profile --mem-profile       # With profiling"
    echo "  $0 --stress-only --short             # Only stress tests, short mode"
    echo "  $0 -o results.txt --compare          # Save and compare results"
}

parse_arguments() {
    while [[ $# -gt 0 ]]; do
        case $1 in
            -v|--verbose)
                VERBOSE=true
                shift
                ;;
            -t|--benchtime)
                BENCHTIME="$2"
                shift 2
                ;;
            -c|--count)
                COUNT="$2"
                shift 2
                ;;
            -b|--bench)
                BENCH_FILTER="$2"
                shift 2
                ;;
            -o|--output)
                OUTPUT_FILE="$2"
                shift 2
                ;;
            --cpu-profile)
                CPU_PROFILE=true
                shift
                ;;
            --mem-profile)
                MEMORY_PROFILE=true
                shift
                ;;
            --compare)
                COMPARE_MODE=true
                shift
                ;;
            --baseline)
                BASELINE_FILE="$2"
                COMPARE_MODE=true
                shift 2
                ;;
            --short)
                SHORT=true
                shift
                ;;
            --stress-only)
                STRESS_ONLY=true
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

setup_benchmark_environment() {
    print_step "Setting up benchmark environment"
    
    # Create benchmark results directory
    mkdir -p "$PROJECT_ROOT/benchmark_results"
    
    # Set environment variables for consistent benchmarking
    export GOMAXPROCS=$(nproc)
    export GOGC=100
    
    # Set benchmark-specific settings
    if [ "$SHORT" = true ]; then
        BENCHTIME="1s"
    fi
    
    if [ "$STRESS_ONLY" = true ] && [ -z "$BENCH_FILTER" ]; then
        BENCH_FILTER="Stress"
    fi
    
    print_success "Environment configured"
    print_info "GOMAXPROCS: $GOMAXPROCS"
    print_info "Benchtime: $BENCHTIME"
    print_info "Count: $COUNT"
}

build_benchmark_command() {
    local cmd="go test"
    
    if [ "$VERBOSE" = true ]; then
        cmd="$cmd -v"
    fi
    
    cmd="$cmd -bench=."
    
    if [ -n "$BENCH_FILTER" ]; then
        cmd="$cmd -bench=$BENCH_FILTER"
    fi
    
    cmd="$cmd -benchtime=$BENCHTIME"
    cmd="$cmd -count=$COUNT"
    
    if [ "$MEMORY_PROFILE" = true ]; then
        cmd="$cmd -memprofile=benchmark_results/mem.prof"
    fi
    
    if [ "$CPU_PROFILE" = true ]; then
        cmd="$cmd -cpuprofile=benchmark_results/cpu.prof"
    fi
    
    if [ "$SHORT" = true ]; then
        cmd="$cmd -short"
    fi
    
    # Don't run normal tests, only benchmarks
    cmd="$cmd -run=^$"
    
    echo "$cmd"
}

discover_benchmark_files() {
    local files=()
    
    # Find all benchmark test files
    while IFS= read -r -d '' file; do
        files+=("$file")
    done < <(find "$PROJECT_ROOT/tests/benchmarks" -name "*_test.go" -type f -print0 2>/dev/null)
    
    if [ ${#files[@]} -eq 0 ]; then
        print_error "No benchmark files found in tests/benchmarks/"
        return 1
    fi
    
    print_info "Found ${#files[@]} benchmark files:"
    for file in "${files[@]}"; do
        local rel_path=${file#$PROJECT_ROOT/}
        local bench_count=$(grep -c "^func Benchmark" "$file" 2>/dev/null || echo "0")
        echo -e "  ${YELLOW}$rel_path${NC} ($bench_count benchmarks)"
    done
    
    return 0
}

run_benchmarks() {
    local cmd=$(build_benchmark_command)
    local benchmark_package="./tests/benchmarks/..."
    
    print_info "Command: $cmd $benchmark_package"
    
    local start_time=$(date +%s)
    local temp_output=$(mktemp)
    
    if [ -n "$OUTPUT_FILE" ]; then
        local full_output_path="$PROJECT_ROOT/benchmark_results/$OUTPUT_FILE"
        print_info "Saving results to: $full_output_path"
        
        if $cmd $benchmark_package | tee "$temp_output" > "$full_output_path"; then
            local success=true
        else
            local success=false
        fi
    else
        if $cmd $benchmark_package | tee "$temp_output"; then
            local success=true
        else
            local success=false
        fi
    fi
    
    local end_time=$(date +%s)
    local duration=$((end_time - start_time))
    
    if [ "$success" = true ]; then
        print_success "Benchmarks completed in ${duration}s"
        analyze_benchmark_results "$temp_output"
        
        if [ "$COMPARE_MODE" = true ]; then
            compare_with_baseline "$temp_output"
        fi
        
        rm -f "$temp_output"
        return 0
    else
        print_error "Benchmarks failed after ${duration}s"
        rm -f "$temp_output"
        return 1
    fi
}

analyze_benchmark_results() {
    local results_file="$1"
    
    if [ ! -f "$results_file" ]; then
        return
    fi
    
    print_step "Analyzing benchmark results"
    
    # Extract benchmark results
    local bench_lines=$(grep "^Benchmark" "$results_file" || true)
    
    if [ -z "$bench_lines" ]; then
        print_warning "No benchmark results found"
        return
    fi
    
    # Count benchmarks
    local total_benchmarks=$(echo "$bench_lines" | wc -l)
    print_info "Total benchmarks run: $total_benchmarks"
    
    # Find fastest and slowest benchmarks
    echo ""
    print_header "Performance Summary"
    
    # Parse results and show top performers
    echo "$bench_lines" | while read -r line; do
        if [[ $line =~ ^Benchmark([^[:space:]]+)[[:space:]]+([0-9]+)[[:space:]]+([0-9.]+)[[:space:]]+(ns/op|μs/op|ms/op) ]]; then
            local name="${BASH_REMATCH[1]}"
            local iterations="${BASH_REMATCH[2]}"
            local time="${BASH_REMATCH[3]}"
            local unit="${BASH_REMATCH[4]}"
            
            # Convert to nanoseconds for comparison
            local time_ns="$time"
            case "$unit" in
                "μs/op") time_ns=$(echo "$time * 1000" | bc -l 2>/dev/null || echo "$time") ;;
                "ms/op") time_ns=$(echo "$time * 1000000" | bc -l 2>/dev/null || echo "$time") ;;
            esac
            
            echo -e "  ${YELLOW}$name${NC}: $iterations iterations, $time $unit"
        fi
    done
    
    # Show memory allocation info if available
    local mem_lines=$(grep "B/op" "$results_file" || true)
    if [ -n "$mem_lines" ]; then
        echo ""
        print_info "Memory allocation summary:"
        echo "$mem_lines" | head -5 | while read -r line; do
            echo "  $line"
        done
    fi
}

compare_with_baseline() {
    local current_results="$1"
    
    if [ -z "$BASELINE_FILE" ]; then
        # Try to find the most recent baseline
        local latest_baseline=$(find "$PROJECT_ROOT/benchmark_results" -name "baseline_*.txt" -type f | sort | tail -1)
        if [ -n "$latest_baseline" ]; then
            BASELINE_FILE=$(basename "$latest_baseline")
            print_info "Using latest baseline: $BASELINE_FILE"
        else
            print_warning "No baseline file specified and none found"
            return
        fi
    fi
    
    local baseline_path="$PROJECT_ROOT/benchmark_results/$BASELINE_FILE"
    
    if [ ! -f "$baseline_path" ]; then
        print_warning "Baseline file not found: $baseline_path"
        return
    fi
    
    print_step "Comparing with baseline: $BASELINE_FILE"
    
    # Simple comparison (could be enhanced with benchcmp tool)
    local current_benchmarks=$(grep "^Benchmark" "$current_results" | wc -l)
    local baseline_benchmarks=$(grep "^Benchmark" "$baseline_path" | wc -l)
    
    echo -e "  Current run: $current_benchmarks benchmarks"
    echo -e "  Baseline:    $baseline_benchmarks benchmarks"
    
    if [ $current_benchmarks -gt $baseline_benchmarks ]; then
        print_success "Added $((current_benchmarks - baseline_benchmarks)) new benchmarks"
    elif [ $current_benchmarks -lt $baseline_benchmarks ]; then
        print_warning "Removed $((baseline_benchmarks - current_benchmarks)) benchmarks"
    else
        print_success "Same number of benchmarks as baseline"
    fi
}

generate_profile_reports() {
    if [ "$CPU_PROFILE" = true ] && [ -f "$PROJECT_ROOT/benchmark_results/cpu.prof" ]; then
        print_step "Generating CPU profile report"
        if command -v go >/dev/null 2>&1; then
            go tool pprof -text "$PROJECT_ROOT/benchmark_results/cpu.prof" > "$PROJECT_ROOT/benchmark_results/cpu_profile.txt" 2>/dev/null || true
            print_success "CPU profile saved to benchmark_results/cpu_profile.txt"
        fi
    fi
    
    if [ "$MEMORY_PROFILE" = true ] && [ -f "$PROJECT_ROOT/benchmark_results/mem.prof" ]; then
        print_step "Generating memory profile report"
        if command -v go >/dev/null 2>&1; then
            go tool pprof -text "$PROJECT_ROOT/benchmark_results/mem.prof" > "$PROJECT_ROOT/benchmark_results/mem_profile.txt" 2>/dev/null || true
            print_success "Memory profile saved to benchmark_results/mem_profile.txt"
        fi
    fi
}

save_as_baseline() {
    if [ -n "$OUTPUT_FILE" ]; then
        local baseline_file="$PROJECT_ROOT/benchmark_results/baseline_$(date +%Y%m%d_%H%M%S).txt"
        local output_path="$PROJECT_ROOT/benchmark_results/$OUTPUT_FILE"
        
        if [ -f "$output_path" ]; then
            cp "$output_path" "$baseline_file"
            print_success "Saved current results as baseline: $(basename $baseline_file)"
        fi
    fi
}

show_benchmark_summary() {
    echo ""
    print_header "Benchmark Configuration Summary"
    
    print_info "Configuration:"
    [ "$VERBOSE" = true ] && echo -e "  ${GREEN}✓${NC} Verbose output"
    [ "$SHORT" = true ] && echo -e "  ${GREEN}✓${NC} Short mode"
    [ "$STRESS_ONLY" = true ] && echo -e "  ${GREEN}✓${NC} Stress benchmarks only"
    [ "$CPU_PROFILE" = true ] && echo -e "  ${GREEN}✓${NC} CPU profiling"
    [ "$MEMORY_PROFILE" = true ] && echo -e "  ${GREEN}✓${NC} Memory profiling"
    [ "$COMPARE_MODE" = true ] && echo -e "  ${GREEN}✓${NC} Baseline comparison"
    [ -n "$BENCH_FILTER" ] && echo -e "  ${GREEN}✓${NC} Benchmark filter: $BENCH_FILTER"
    [ -n "$OUTPUT_FILE" ] && echo -e "  ${GREEN}✓${NC} Output file: $OUTPUT_FILE"
    echo -e "  ${GREEN}✓${NC} Benchtime: $BENCHTIME"
    echo -e "  ${GREEN}✓${NC} Count: $COUNT"
}

main() {
    cd "$PROJECT_ROOT"
    
    parse_arguments "$@"
    
    print_header "Redis Benchmark Test Runner"
    show_benchmark_summary
    
    echo ""
    if ! discover_benchmark_files; then
        exit 1
    fi
    
    echo ""
    setup_benchmark_environment
    
    echo ""
    print_header "Executing Benchmarks"
    
    if run_benchmarks; then
        echo ""
        generate_profile_reports
        save_as_baseline
        
        echo ""
        print_success "All benchmarks completed successfully! 🎉"
        
        if [ "$CPU_PROFILE" = true ] || [ "$MEMORY_PROFILE" = true ]; then
            print_info "Profile reports saved in benchmark_results/"
        fi
        
        exit 0
    else
        echo ""
        print_error "Benchmarks failed! ❌"
        exit 1
    fi
}

main "$@"
