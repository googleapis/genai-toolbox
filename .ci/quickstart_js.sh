#!/bin/bash

# This script runs tests for JavaScript samples.

set -e

# The cleanup function is guaranteed to run when a test subshell exits.
cleanup() {
  psql -c "DROP TABLE IF EXISTS hotels;"
}

# The main function orchestrates the entire test run for all JS samples.
main() {
  echo "--- Running JavaScript Sample Tests for Each Framework ---"
  
  local frameworks=("genAI" "genkit" "langchain" "llamaindex")

  for framework in "${frameworks[@]}"; do
    local framework_dir="../docs/en/getting-started/quickstart/js/${framework}"
    
    if [ ! -d "${framework_dir}" ]; then
        echo -e "\nSkipping framework '${framework}': directory not found."
        continue
    fi


    find "${framework_dir}" -name "node_modules" -prune -o -type f -name "package.json" -print | while read -r pkg_file; do
      local sample_dir
      sample_dir=$(dirname "${pkg_file}")

      (
        cd "${sample_dir}"
        
        trap cleanup EXIT

        npm install --silent

        psql <<-SQL
          CREATE TABLE hotels(
            id INTEGER NOT NULL PRIMARY KEY, name VARCHAR NOT NULL, location VARCHAR NOT NULL,
            price_tier VARCHAR NOT NULL, checkin_date DATE NOT NULL, checkout_date DATE NOT NULL,
            booked BIT NOT NULL
          );
          INSERT INTO hotels(id, name, location, price_tier, checkin_date, checkout_date, booked)
          VALUES
            (1, 'Hilton Basel', 'Basel', 'Luxury', '2024-04-22', '2024-04-20', B'0'),
            (2, 'Marriott Zurich', 'Zurich', 'Upscale', '2024-04-14', '2024-04-21', B'0'),
            (3, 'Hyatt Regency Basel', 'Basel', 'Upper Upscale', '2024-04-02', '2024-04-20', B'0'),
            (4, 'Radisson Blu Lucerne', 'Lucerne', 'Midscale', '2024-04-24', '2024-04-05', B'0'),
            (5, 'Best Western Bern', 'Bern', 'Upper Midscale', '2024-04-23', '2024-04-01', B'0'),
            (6, 'InterContinental Geneva', 'Geneva', 'Luxury', '2024-04-23', '2024-04-28', B'0'),
            (7, 'Sheraton Zurich', 'Zurich', 'Upper Upscale', '2024-04-27', '2024-04-02', B'0'),
            (8, 'Holiday Inn Basel', 'Basel', 'Upper Midscale', '2024-04-24', '2024-04-09', B'0'),
            (9, 'Courtyard Zurich', 'Zurich', 'Upscale', '2024-04-03', '2024-04-13', B'0'),
            (10, 'Comfort Inn Bern', 'Bern', 'Midscale', '2024-04-04', '2024-04-16', B'0');
SQL

        echo "Running test script..."
        npm test
      )
    done
  done

  echo -e "\n--- All JavaScript Framework Tests Passed ---"
}

main "$@"
