#!/bin/bash

# This script runs prompt tests based on the convention that tests
# are located in `tests/prompts/{subdir}`.
#
# Arguments:
# $1: Display name for logs (e.g., "Custom Prompts")
# $2: The unique subdirectory for the prompt test (e.g., custom)

set -e

DISPLAY_NAME="$1"
PROMPT_TEST_SUBDIR="$2" # e.g., "custom"

FULL_TEST_PATH="tests/prompts/${PROMPT_TEST_SUBDIR}"
TEST_BINARY=$(echo "${FULL_TEST_PATH}" | tr / .).test

COVERAGE_FILE="${TEST_BINARY%.test}_coverage.out"
FILTERED_COVERAGE_FILE="${TEST_BINARY%.test}_filtered_coverage.out"

export path="github.com/googleapis/genai-toolbox/internal/"

GREP_PATTERN="^mode:|${path}prompts/"

# Run integration test
if ! ./"${TEST_BINARY}" -test.v -test.coverprofile="${COVERAGE_FILE}"; then
  echo "Error: Tests for ${DISPLAY_NAME} failed. Exiting."
  exit 1
fi

# Filter the coverage file to only include the prompts package
if ! grep -E "${GREP_PATTERN}" "${COVERAGE_FILE}" > "${FILTERED_COVERAGE_FILE}"; then
  echo "Warning: Could not filter coverage for ${DISPLAY_NAME}. Filtered file might be empty or invalid."
fi

# Calculate coverage
echo "Calculating coverage for ${DISPLAY_NAME}..."
total_coverage=$(go tool cover -func="${FILTERED_COVERAGE_FILE}" 2>/dev/null | grep "total:" | awk '{print $3}')

echo "${DISPLAY_NAME} total coverage: $total_coverage"
coverage_numeric=$(echo "$total_coverage" | sed 's/%//')

# Check coverage threshold
if awk -v coverage="$coverage_numeric" 'BEGIN {exit !(coverage < 50)}'; then
    echo "Coverage failure: ${DISPLAY_NAME} total coverage($total_coverage) is below 50%."
    exit 1
else
    echo "Coverage for ${DISPLAY_NAME} is sufficient."
fi
