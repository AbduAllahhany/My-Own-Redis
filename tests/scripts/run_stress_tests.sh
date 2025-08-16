#!/bin/bash

# Stress Test Runner for Redis Implementation
# Runs comprehensive stress tests with detailed output

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
MAGENTA='\033[0;35m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

# Function to print colored output
print_header() {
    echo -e "${CYAN}╔════════════════════════════════════════════════════════════════╗${NC}"
    echo -e "${CYAN}║${NC} ${MAGENTA}$1${NC} ${CYAN}║${NC}"
    echo -e "${CYAN}╚════════════════════════════════════════════════════════════════╝${NC}"
}

print_section() {
    echo -e "${BLUE}[STRESS]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[PASS]${NC} $1"
}

print_info() {
    echo -e "${YELLOW}[INFO]${NC} $1"
}

# Change to project directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
cd "$PROJECT_ROOT"

print_header "Redis Stress Test Suite Runner"

echo ""
print_section "Available Stress Test Categories:"
echo ""

echo -e "${YELLOW}1. Core Stress Tests:${NC}"
echo "   • BenchmarkStressCore_MassiveKeys      - 100K+ keys performance"
echo "   • BenchmarkStressCore_RapidUpdates     - Rapid key update cycles"
echo "   • BenchmarkStressCore_MaxParallelism   - Maximum CPU parallelism"
echo "   • BenchmarkStressCore_MemoryChurn      - Memory allocation stress"

echo ""
echo -e "${YELLOW}2. Concurrency Stress Tests:${NC}"
echo "   • BenchmarkStress_MassiveParallelWrites - Heavy concurrent writes"
echo "   • BenchmarkStress_ReadHeavyWorkload     - 90% read, 10% write load"
echo "   • BenchmarkStress_ContentionLevel      - Lock contention on hot keys"
echo "   • BenchmarkStress_ThousandGoRoutines    - 1000 goroutines stress"

echo ""
echo -e "${YELLOW}3. Memory Stress Tests:${NC}"
echo "   • BenchmarkStress_MemoryPressure       - Large value memory stress"
echo "   • BenchmarkStress_LargeKeyOperations    - 1MB value operations"
echo "   • BenchmarkStress_RapidKeyChurn         - Rapid create/delete cycles"

echo ""
echo -e "${YELLOW}4. Realistic Workload Tests:${NC}"
echo "   • BenchmarkStress_MixedWorkload         - Real-world operation mix"
echo "   • BenchmarkStress_ExpirationStorm       - Heavy expiration handling"

echo ""
echo -e "${YELLOW}5. Lock Contention Tests:${NC}"
echo "   • BenchmarkRedis_HighConcurrency        - High concurrent operations"
echo "   • BenchmarkRedis_LockContention         - Severe lock contention"

echo ""
print_section "Running Quick Stress Test Overview..."
echo ""

# Run a quick overview of all stress tests
go test ./tests/benchmarks/ -bench=Stress -benchtime=1s | grep -E "(Benchmark|PASS|FAIL)" | while read line; do
    if [[ $line == Benchmark* ]]; then
        benchmark_name=$(echo $line | awk '{print $1}')
        ops_per_second=$(echo $line | awk '{print $2}')
        ns_per_op=$(echo $line | awk '{print $3}')
        echo -e "${GREEN}✓${NC} ${benchmark_name} - ${ops_per_second} ops, ${ns_per_op}"
    fi
done

echo ""
print_section "Detailed Stress Test Commands:"
echo ""

echo -e "${CYAN}Run all stress tests:${NC}"
echo "  go test ./tests/benchmarks/ -bench=Stress -benchtime=5s -v"

echo ""
echo -e "${CYAN}Run specific stress categories:${NC}"
echo "  go test ./tests/benchmarks/ -bench=BenchmarkStressCore -benchtime=3s"
echo "  go test ./tests/benchmarks/ -bench=BenchmarkStress_Memory -benchtime=3s"
echo "  go test ./tests/benchmarks/ -bench=BenchmarkStress_Concurrency -benchtime=3s"

echo ""
echo -e "${CYAN}Run with memory profiling:${NC}"
echo "  go test ./tests/benchmarks/ -bench=Stress -benchmem -memprofile=mem.prof"

echo ""
echo -e "${CYAN}Run with CPU profiling:${NC}"
echo "  go test ./tests/benchmarks/ -bench=Stress -cpuprofile=cpu.prof"

echo ""
echo -e "${CYAN}Continuous stress testing:${NC}"
echo "  watch -n 10 'go test ./tests/benchmarks/ -bench=Stress -benchtime=2s'"

echo ""
echo -e "${CYAN}Compare stress test results:${NC}"
echo "  go test ./tests/benchmarks/ -bench=Stress > stress_baseline.txt"
echo "  # Make changes, then:"
echo "  go test ./tests/benchmarks/ -bench=Stress > stress_current.txt"
echo "  benchcmp stress_baseline.txt stress_current.txt"

echo ""
print_section "Performance Targets for Stress Tests:"
echo ""

echo -e "${GREEN}Excellent Performance:${NC} < 100 ns/op"
echo -e "${YELLOW}Good Performance:${NC}     100-500 ns/op"
echo -e "${YELLOW}Acceptable:${NC}           500-1000 ns/op"
echo -e "${RED}Needs Optimization:${NC}   > 1000 ns/op"

echo ""
print_section "Stress Test Success Criteria:"
echo ""

echo "✓ No panics or crashes during execution"
echo "✓ Consistent performance across multiple runs"
echo "✓ Memory usage remains stable over time"
echo "✓ All operations complete successfully"
echo "✓ Performance within acceptable thresholds"

echo ""
print_info "To run the full stress test suite with comprehensive output:"
print_info "  ./run_stress_tests.sh --full"

echo ""
if [ "$1" = "--full" ]; then
    print_header "Running Full Stress Test Suite"
    echo ""
    
    print_section "Starting comprehensive stress testing..."
    go test ./tests/benchmarks/ -bench=Stress -benchtime=5s -benchmem -v
    
    echo ""
    print_success "Full stress test suite completed!"
fi

print_info "Stress testing infrastructure ready! 🚀"
