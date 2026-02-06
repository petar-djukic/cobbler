#!/usr/bin/env bash
#
# Run a benchmark fixture end-to-end and report pass/fail.
#
# Usage: ./scripts/run-benchmark.sh <benchmark-directory>
#
# Example: ./scripts/run-benchmark.sh benchmarks/hello-world
#
# The script:
# 1. Reads benchmark.yaml for metadata and pass criteria
# 2. Creates a temp working directory
# 3. Copies the fixture spec (e.g., VISION.md) into it
# 4. Invokes Claude with a prompt built from the spec
# 5. Checks pass criteria: expected files exist, build succeeds, output matches
# 6. Reports pass/fail and cleans up
#

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Globals
BENCHMARK_DIR=""
WORK_DIR=""
BENCHMARK_NAME=""
BENCHMARK_LANG=""
SPEC_FILE=""
EXPECTED_FILES=""
BUILD_CMD=""
BUILD_EXIT=""
RUN_CMD=""
RUN_STDOUT=""
RUN_EXIT=""

usage() {
    echo "Usage: $0 <benchmark-directory>"
    echo ""
    echo "Example: $0 benchmarks/hello-world"
    exit 1
}

log_info() {
    echo -e "${YELLOW}[INFO]${NC} $1"
}

log_pass() {
    echo -e "${GREEN}[PASS]${NC} $1"
}

log_fail() {
    echo -e "${RED}[FAIL]${NC} $1"
}

# Parse a simple YAML value: parse_yaml_value "key" "file"
# Works for simple key: value pairs (no nested structures)
parse_yaml_value() {
    local key="$1"
    local file="$2"
    grep "^[[:space:]]*${key}:" "$file" | head -1 | sed "s/^[[:space:]]*${key}:[[:space:]]*//" | sed 's/^"//' | sed 's/"$//' | sed "s/^'//" | sed "s/'$//"
}

# Parse a YAML list under a key: parse_yaml_list "parent.child" "file"
# Returns items one per line
parse_yaml_list() {
    local key="$1"
    local file="$2"
    # Find the key, then grab subsequent lines starting with "- " until we hit a non-list line
    awk -v key="$key:" '
        $0 ~ "^[[:space:]]*" key {found=1; next}
        found && /^[[:space:]]*-[[:space:]]/ {
            sub(/^[[:space:]]*-[[:space:]]*/, "")
            gsub(/^"/, ""); gsub(/"$/, "")
            gsub(/'"'"'/, "")
            print
            next
        }
        found && /^[[:space:]]*[a-zA-Z_]/ {exit}
    ' "$file"
}

# Parse nested YAML value: parse_yaml_nested "criteria.build.command" "file"
parse_yaml_nested() {
    local path="$1"
    local file="$2"
    local keys
    IFS='.' read -ra keys <<< "$path"

    local indent=0
    local in_section=1
    local result=""

    # Simple approach: find the deepest key at appropriate indentation
    local depth=${#keys[@]}
    local target_key="${keys[$((depth-1))]}"

    # Build a pattern to find the nested value
    # For "criteria.build.command", we look for "command:" under "build:" under "criteria:"
    local pattern=""
    for ((i=0; i<depth-1; i++)); do
        pattern="${pattern}.*${keys[$i]}:"
    done

    # Use awk to navigate the structure
    awk -v keys="${path}" '
    BEGIN {
        n = split(keys, k, ".")
        depth = 0
        target_depth = n
    }
    {
        # Calculate current indentation (number of leading spaces)
        match($0, /^[[:space:]]*/)
        indent = RLENGTH

        # Check if this line matches current target key
        if (depth < target_depth) {
            pattern = "^[[:space:]]*" k[depth+1] ":"
            if ($0 ~ pattern) {
                depth++
                if (depth == target_depth) {
                    # Extract value after the key
                    sub(/^[[:space:]]*[^:]+:[[:space:]]*/, "")
                    gsub(/^"/, ""); gsub(/"$/, "")
                    gsub(/\\n/, "\n")
                    print
                    exit
                }
            }
        }
    }
    ' "$file"
}

read_benchmark_yaml() {
    local yaml_file="$BENCHMARK_DIR/benchmark.yaml"

    if [ ! -f "$yaml_file" ]; then
        log_fail "benchmark.yaml not found in $BENCHMARK_DIR"
        exit 1
    fi

    BENCHMARK_NAME=$(parse_yaml_value "name" "$yaml_file")
    BENCHMARK_LANG=$(parse_yaml_value "language" "$yaml_file")
    SPEC_FILE=$(parse_yaml_nested "spec.file" "$yaml_file")

    # Parse expected files list
    EXPECTED_FILES=$(parse_yaml_list "files" "$yaml_file")

    # Parse criteria
    BUILD_CMD=$(parse_yaml_nested "criteria.build.command" "$yaml_file")
    BUILD_EXIT=$(parse_yaml_nested "criteria.build.exit_code" "$yaml_file")
    RUN_CMD=$(parse_yaml_nested "criteria.run.command" "$yaml_file")
    RUN_STDOUT=$(parse_yaml_nested "criteria.run.stdout" "$yaml_file")
    RUN_EXIT=$(parse_yaml_nested "criteria.run.exit_code" "$yaml_file")

    log_info "Benchmark: $BENCHMARK_NAME ($BENCHMARK_LANG)"
    log_info "Spec file: $SPEC_FILE"
    log_info "Expected files: $(echo $EXPECTED_FILES | tr '\n' ' ')"
    log_info "Build: $BUILD_CMD (expect exit $BUILD_EXIT)"
    log_info "Run: $RUN_CMD (expect exit $RUN_EXIT)"
}

setup_work_dir() {
    WORK_DIR=$(mktemp -d)
    log_info "Work directory: $WORK_DIR"

    # Copy spec file to work directory
    local spec_path="$BENCHMARK_DIR/$SPEC_FILE"
    if [ ! -f "$spec_path" ]; then
        log_fail "Spec file not found: $spec_path"
        cleanup
        exit 1
    fi

    cp "$spec_path" "$WORK_DIR/"
    log_info "Copied $SPEC_FILE to work directory"
}

build_prompt() {
    local spec_content
    spec_content=$(cat "$WORK_DIR/$SPEC_FILE")

    cat <<EOF
## Benchmark Task: $BENCHMARK_NAME

You are generating code for a benchmark fixture. Read the specification below and produce the required files.

### Specification

$spec_content

### Instructions

1. Create all required files in the current directory
2. Follow the specification exactly
3. The code must build and run successfully
4. Do not create any files beyond what is specified

### Required Output Files

$(echo "$EXPECTED_FILES" | while read -r f; do echo "- $f"; done)

Begin implementation now. Create the files as specified.
EOF
}

run_claude() {
    local prompt="$1"

    log_info "Invoking Claude..."

    cd "$WORK_DIR"

    # Use the same pattern as do-work.sh: echo prompt | claude --dangerously-skip-permissions -p
    if ! echo "$prompt" | claude --dangerously-skip-permissions -p >/dev/null 2>&1; then
        log_fail "Claude invocation failed"
        return 1
    fi

    log_info "Claude completed"
    return 0
}

check_expected_files() {
    log_info "Checking expected files..."
    local all_exist=true

    echo "$EXPECTED_FILES" | while read -r file; do
        if [ -z "$file" ]; then
            continue
        fi
        if [ -f "$WORK_DIR/$file" ]; then
            log_pass "File exists: $file"
        else
            log_fail "File missing: $file"
            echo "MISSING" > "$WORK_DIR/.check_result"
        fi
    done

    if [ -f "$WORK_DIR/.check_result" ]; then
        rm "$WORK_DIR/.check_result"
        return 1
    fi
    return 0
}

check_build() {
    if [ -z "$BUILD_CMD" ]; then
        log_info "No build criteria, skipping"
        return 0
    fi

    log_info "Running build: $BUILD_CMD"

    cd "$WORK_DIR"
    local actual_exit=0
    eval "$BUILD_CMD" >/dev/null 2>&1 || actual_exit=$?

    if [ "$actual_exit" -eq "$BUILD_EXIT" ]; then
        log_pass "Build succeeded (exit code $actual_exit)"
        return 0
    else
        log_fail "Build failed (exit code $actual_exit, expected $BUILD_EXIT)"
        return 1
    fi
}

check_run() {
    if [ -z "$RUN_CMD" ]; then
        log_info "No run criteria, skipping"
        return 0
    fi

    log_info "Running: $RUN_CMD"

    cd "$WORK_DIR"
    local actual_output
    local actual_exit=0
    actual_output=$(eval "$RUN_CMD" 2>&1) || actual_exit=$?

    # Check exit code
    if [ "$actual_exit" -ne "$RUN_EXIT" ]; then
        log_fail "Run exit code mismatch (got $actual_exit, expected $RUN_EXIT)"
        return 1
    fi
    log_pass "Run exit code: $actual_exit"

    # Check stdout if specified
    if [ -n "$RUN_STDOUT" ]; then
        # Convert \n in expected to actual newline for comparison
        local expected_output
        expected_output=$(printf '%b' "$RUN_STDOUT")

        if [ "$actual_output" = "$expected_output" ]; then
            log_pass "Output matches expected"
        else
            log_fail "Output mismatch"
            echo "  Expected: $(echo "$expected_output" | cat -v)"
            echo "  Got:      $(echo "$actual_output" | cat -v)"
            return 1
        fi
    fi

    return 0
}

cleanup() {
    if [ -n "$WORK_DIR" ] && [ -d "$WORK_DIR" ]; then
        log_info "Cleaning up $WORK_DIR"
        rm -rf "$WORK_DIR"
    fi
}

# Ensure cleanup on exit
trap cleanup EXIT

main() {
    if [ $# -lt 1 ]; then
        usage
    fi

    BENCHMARK_DIR="$1"

    # Convert to absolute path if relative
    if [[ "$BENCHMARK_DIR" != /* ]]; then
        BENCHMARK_DIR="$(pwd)/$BENCHMARK_DIR"
    fi

    if [ ! -d "$BENCHMARK_DIR" ]; then
        log_fail "Benchmark directory not found: $BENCHMARK_DIR"
        exit 1
    fi

    echo ""
    echo "========================================"
    echo "Running Benchmark"
    echo "========================================"
    echo ""

    read_benchmark_yaml
    setup_work_dir

    local prompt
    prompt=$(build_prompt)

    if ! run_claude "$prompt"; then
        log_fail "BENCHMARK FAILED: Claude invocation error"
        exit 1
    fi

    echo ""
    echo "----------------------------------------"
    echo "Checking Pass Criteria"
    echo "----------------------------------------"
    echo ""

    local failed=false

    if ! check_expected_files; then
        failed=true
    fi

    if ! check_build; then
        failed=true
    fi

    if ! check_run; then
        failed=true
    fi

    echo ""
    echo "========================================"
    if [ "$failed" = true ]; then
        log_fail "BENCHMARK FAILED: $BENCHMARK_NAME"
        exit 1
    else
        log_pass "BENCHMARK PASSED: $BENCHMARK_NAME"
        exit 0
    fi
}

main "$@"
