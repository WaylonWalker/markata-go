#!/bin/bash
# Benchmark script for comparing markata-go performance between branches
# Usage: ./scripts/benchmark.sh [--base <branch>] [--compare <branch>] [--site <path>]

set -e

# Store script location
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

# Default values
BASE_BRANCH="main"
COMPARE_BRANCH=""
SITE_PATH="../waylonwalker.com-markata-go-migration"
NUM_RUNS=3

# Temp directory for binaries
TEMP_DIR=""

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Cleanup function
cleanup() {
    local exit_code=$?
    echo ""
    echo "Cleaning up..."
    
    # Restore original branch
    if [ -n "$ORIGINAL_BRANCH" ]; then
        cd "$PROJECT_ROOT"
        git checkout "$ORIGINAL_BRANCH" 2>/dev/null || true
    fi
    
    # Pop stash if we stashed changes
    if [ "$STASHED" = "true" ]; then
        echo "Restoring stashed changes..."
        git stash pop 2>/dev/null || true
    fi
    
    # Remove temp binaries
    if [ -n "$TEMP_DIR" ] && [ -d "$TEMP_DIR" ]; then
        rm -rf "$TEMP_DIR"
        echo "Removed temp binaries"
    fi
    
    exit $exit_code
}

trap cleanup EXIT

# Print usage
usage() {
    echo "Usage: $0 [OPTIONS]"
    echo ""
    echo "Options:"
    echo "  --base <branch>     Base branch to compare against (default: main)"
    echo "  --compare <branch>  Branch to compare (default: current branch)"
    echo "  --site <path>       Path to test site (default: ../waylonwalker.com-markata-go-migration)"
    echo "  --runs <n>          Number of runs for averaging (default: 3)"
    echo "  -h, --help          Show this help message"
    echo ""
    echo "Example:"
    echo "  $0 --base main --compare feature/faster-rendering"
}

# Parse arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        --base)
            BASE_BRANCH="$2"
            shift 2
            ;;
        --compare)
            COMPARE_BRANCH="$2"
            shift 2
            ;;
        --site)
            SITE_PATH="$2"
            shift 2
            ;;
        --runs)
            NUM_RUNS="$2"
            shift 2
            ;;
        -h|--help)
            usage
            exit 0
            ;;
        *)
            echo "Unknown option: $1"
            usage
            exit 1
            ;;
    esac
done

# Print header
echo "=========================================="
echo "  Markata-Go Branch Benchmark"
echo "=========================================="
echo ""

# Get current branch
cd "$PROJECT_ROOT"
ORIGINAL_BRANCH=$(git branch --show-current)
if [ -z "$COMPARE_BRANCH" ]; then
    COMPARE_BRANCH="$ORIGINAL_BRANCH"
fi

# Validate branches exist
echo "Validating branches..."
if ! git rev-parse --verify "$BASE_BRANCH" >/dev/null 2>&1; then
    echo -e "${RED}Error: Base branch '$BASE_BRANCH' does not exist${NC}"
    exit 1
fi

if ! git rev-parse --verify "$COMPARE_BRANCH" >/dev/null 2>&1; then
    echo -e "${RED}Error: Compare branch '$COMPARE_BRANCH' does not exist${NC}"
    exit 1
fi

# Resolve site path
if [[ "$SITE_PATH" != /* ]]; then
    SITE_PATH="$(cd "$PROJECT_ROOT" && cd "$SITE_PATH" 2>/dev/null && pwd)" || {
        echo -e "${RED}Error: Site path does not exist: $SITE_PATH${NC}"
        exit 1
    }
fi

if [ ! -f "$SITE_PATH/markata-go.toml" ]; then
    echo -e "${RED}Error: No markata-go.toml found in $SITE_PATH${NC}"
    exit 1
fi

echo ""
echo "Configuration:"
echo "  Base branch:    $BASE_BRANCH"
echo "  Compare branch: $COMPARE_BRANCH"
echo "  Test site:      $SITE_PATH"
echo "  Runs per test:  $NUM_RUNS"
echo ""

# Create temp directory for binaries
TEMP_DIR=$(mktemp -d)
BASE_BINARY="$TEMP_DIR/markata-go-base"
COMPARE_BINARY="$TEMP_DIR/markata-go-compare"

# Check for uncommitted changes
STASHED="false"
if [ -n "$(git status --porcelain)" ]; then
    echo -e "${YELLOW}Uncommitted changes detected, stashing...${NC}"
    git stash push -m "benchmark-script-temp-stash"
    STASHED="true"
fi

# Build binary for a branch
build_binary() {
    local branch=$1
    local output=$2
    
    echo -e "${BLUE}Building binary for $branch...${NC}"
    git checkout "$branch" --quiet
    go build -o "$output" ./cmd/markata-go
    echo -e "${GREEN}  Built: $output${NC}"
}

# Run benchmark (returns time in seconds)
run_benchmark() {
    local binary=$1
    local site_path=$2
    local clean=$3  # "cold" or "warm"
    
    cd "$site_path"
    
    if [ "$clean" = "cold" ]; then
        rm -rf output .build-cache.json html-cache 2>/dev/null || true
    fi
    
    # Run build and extract Duration from output
    local output
    output=$("$binary" build 2>&1)
    
    # Extract duration (e.g., "Duration: 17.34s" -> "17.34")
    local duration
    duration=$(echo "$output" | grep -oP 'Duration: \K[0-9.]+' || echo "0")
    
    echo "$duration"
}

# Run multiple benchmarks and calculate average
run_benchmarks() {
    local binary=$1
    local site_path=$2
    local clean=$3
    local runs=$4
    
    local times=""
    for ((i=1; i<=runs; i++)); do
        local time=$(run_benchmark "$binary" "$site_path" "$clean")
        times="$times $time"
        echo "    Run $i: ${time}s"
    done
    
    # Calculate average using awk
    echo "$times" | awk '{sum=0; for(i=1;i<=NF;i++) sum+=$i; printf "%.2f", sum/NF}'
}

# Count files in output directory
count_files() {
    local dir=$1
    find "$dir" -type f 2>/dev/null | wc -l
}

# Build binaries
echo "=========================================="
echo "  Building Binaries"
echo "=========================================="
echo ""

build_binary "$BASE_BRANCH" "$BASE_BINARY"
build_binary "$COMPARE_BRANCH" "$COMPARE_BINARY"

# Return to original branch (cleanup will handle full restore)
git checkout "$ORIGINAL_BRANCH" --quiet

echo ""
echo "=========================================="
echo "  Running Benchmarks"
echo "=========================================="

# Benchmark base branch
echo ""
echo -e "${BLUE}Benchmarking: $BASE_BRANCH${NC}"
echo "  Cold builds:"
BASE_COLD=$(run_benchmarks "$BASE_BINARY" "$SITE_PATH" "cold" "$NUM_RUNS")
echo "  Average: ${BASE_COLD}s"

echo "  Incremental builds:"
BASE_WARM=$(run_benchmarks "$BASE_BINARY" "$SITE_PATH" "warm" "$NUM_RUNS")
echo "  Average: ${BASE_WARM}s"

# Count files and save output for comparison
cd "$SITE_PATH"
BASE_FILE_COUNT=$(count_files "output")
mv output output-base 2>/dev/null || true

# Benchmark compare branch
echo ""
echo -e "${BLUE}Benchmarking: $COMPARE_BRANCH${NC}"
echo "  Cold builds:"
COMPARE_COLD=$(run_benchmarks "$COMPARE_BINARY" "$SITE_PATH" "cold" "$NUM_RUNS")
echo "  Average: ${COMPARE_COLD}s"

echo "  Incremental builds:"
COMPARE_WARM=$(run_benchmarks "$COMPARE_BINARY" "$SITE_PATH" "warm" "$NUM_RUNS")
echo "  Average: ${COMPARE_WARM}s"

# Count files and save output for comparison
COMPARE_FILE_COUNT=$(count_files "output")
mv output output-compare 2>/dev/null || true

echo ""
echo "=========================================="
echo "  Comparing Outputs"
echo "=========================================="
echo ""

# Compare outputs
OUTPUTS_MATCH="Yes"
DIFF_OUTPUT=$(diff -rq output-base output-compare 2>&1) || OUTPUTS_MATCH="No"

if [ "$OUTPUTS_MATCH" = "Yes" ]; then
    echo -e "${GREEN}Outputs are identical${NC}"
else
    echo -e "${RED}Outputs differ:${NC}"
    echo "$DIFF_OUTPUT" | head -20
    if [ $(echo "$DIFF_OUTPUT" | wc -l) -gt 20 ]; then
        echo "... (truncated)"
    fi
fi

# Clean up output directories
rm -rf output-base output-compare

# Calculate differences using awk
COLD_DIFF=$(awk "BEGIN {printf \"%.2f\", $COMPARE_COLD - $BASE_COLD}")
WARM_DIFF=$(awk "BEGIN {printf \"%.2f\", $COMPARE_WARM - $BASE_WARM}")

# Calculate percentage change using awk
COLD_PCT=$(awk "BEGIN {if ($BASE_COLD > 0) printf \"%.1f\", (($COMPARE_COLD - $BASE_COLD) / $BASE_COLD) * 100; else print \"N/A\"}")
WARM_PCT=$(awk "BEGIN {if ($BASE_WARM > 0) printf \"%.1f\", (($COMPARE_WARM - $BASE_WARM) / $BASE_WARM) * 100; else print \"N/A\"}")

# Format diff with sign
format_diff() {
    local val=$1
    if awk "BEGIN {exit !($val >= 0)}"; then
        echo "+$val"
    else
        echo "$val"
    fi
}

echo ""
echo "=========================================="
echo "  Results"
echo "=========================================="
echo ""
echo "Metric               | $BASE_BRANCH          | $COMPARE_BRANCH  | Diff"
echo "---------------------|-----------------|-----------------|----------------"
echo "Cold Build (avg)     | ${BASE_COLD}s         | ${COMPARE_COLD}s        | $(format_diff $COLD_DIFF)s (${COLD_PCT}%)"
echo "Incremental (avg)    | ${BASE_WARM}s         | ${COMPARE_WARM}s        | $(format_diff $WARM_DIFF)s (${WARM_PCT}%)"
echo "File Count           | $BASE_FILE_COUNT              | $COMPARE_FILE_COUNT             | $((COMPARE_FILE_COUNT - BASE_FILE_COUNT))"
echo "Outputs Match        | -               | -               | $OUTPUTS_MATCH"
echo ""

# Summary
echo "Summary:"
if awk "BEGIN {exit !($COLD_DIFF < 0)}"; then
    COLD_ABS=$(awk "BEGIN {printf \"%.2f\", $COLD_DIFF * -1}")
    echo -e "  ${GREEN}Cold builds: ${COLD_ABS}s faster (${COLD_PCT}%)${NC}"
elif awk "BEGIN {exit !($COLD_DIFF > 0)}"; then
    echo -e "  ${RED}Cold builds: ${COLD_DIFF}s slower (${COLD_PCT}%)${NC}"
else
    echo "  Cold builds: No change"
fi

if awk "BEGIN {exit !($WARM_DIFF < 0)}"; then
    WARM_ABS=$(awk "BEGIN {printf \"%.2f\", $WARM_DIFF * -1}")
    echo -e "  ${GREEN}Incremental builds: ${WARM_ABS}s faster (${WARM_PCT}%)${NC}"
elif awk "BEGIN {exit !($WARM_DIFF > 0)}"; then
    echo -e "  ${RED}Incremental builds: ${WARM_DIFF}s slower (${WARM_PCT}%)${NC}"
else
    echo "  Incremental builds: No change"
fi

echo ""

# Exit with error if outputs differ
if [ "$OUTPUTS_MATCH" = "No" ]; then
    echo -e "${RED}ERROR: Outputs differ between branches!${NC}"
    exit 1
fi

echo -e "${GREEN}Benchmark complete!${NC}"
