#!/bin/bash
# .ci/test_prompts_with_coverage.sh
#
# This script runs a specific prompt integration test, calculates its
# code coverage, and checks if it meets a minimum threshold.
#
# It is called with one argument: the type of the prompt.
# Example usage: .ci/test_prompts_with_coverage.sh "custom"

# Exit immediately if a command fails.
set -e

# --- 1. Define Variables ---

# The first argument is the prompt type (e.g., "custom").
PROMPT_TYPE=$1
COVERAGE_THRESHOLD=50 # Minimum coverage percentage required.

if [ -z "$PROMPT_TYPE" ]; then
    echo "Error: No prompt type provided. Please call this script with an argument."
    echo "Usage: .ci/test_prompts_with_coverage.sh <prompt_type>"
    exit 1
fi

# Construct names based on the prompt type.
TEST_BINARY="./prompt.${PROMPT_TYPE}.test"
TEST_NAME="$(tr '[:lower:]' '[:upper:]' <<< ${PROMPT_TYPE:0:1})${PROMPT_TYPE:1} Prompts"
COVERAGE_FILE="coverage.prompts-${PROMPT_TYPE}.out"


# --- 2. Run Integration Tests ---

echo "--- Running integration tests for ${TEST_NAME} ---"

# Safety check for the binary's existence.
if [ ! -f "$TEST_BINARY" ]; then
    echo "Error: Test binary not found at ${TEST_BINARY}. Aborting."
    exit 1
fi

# Execute the test binary and generate the coverage file.
# If the tests fail, the 'set -e' command will cause the script to exit here.
if ! ./"${TEST_BINARY}" -test.v -test.coverprofile="${COVERAGE_FILE}"; then
  echo "Error: Tests for ${TEST_NAME} failed. Exiting."
  exit 1
fi

echo "--- Tests for ${TEST_NAME} passed successfully ---"


# --- 3. Calculate and Check Coverage ---

echo "Calculating coverage for ${TEST_NAME}..."

# Calculate the total coverage percentage from the generated file.
# The '2>/dev/null' suppresses warnings if the coverage file is empty.
total_coverage=$(go tool cover -func="${COVERAGE_FILE}" 2>/dev/null | grep "total:" | awk '{print $3}')

if [ -z "$total_coverage" ]; then
    echo "Warning: Could not calculate coverage for ${TEST_NAME}. The coverage report might be empty."
    total_coverage="0%"
fi

echo "${TEST_NAME} total coverage: $total_coverage"

# Remove the '%' sign for numerical comparison.
coverage_numeric=$(echo "$total_coverage" | sed 's/%//')

# Check if the coverage is below the defined threshold.
if awk -v coverage="$coverage_numeric" -v threshold="$COVERAGE_THRESHOLD" 'BEGIN {exit !(coverage < threshold)}'; then
    echo "Coverage failure: ${TEST_NAME} total coverage (${total_coverage}) is below the ${COVERAGE_THRESHOLD}% threshold."
    exit 1
else
    echo "Coverage for ${TEST_NAME} is sufficient."
fi