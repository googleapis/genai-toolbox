#!/bin/bash

set -e
set -u

# This script is designed to be called from a CI/CD pipeline to run Python
# quickstart tests for multiple orchestration frameworks.
#
# For each orchestrator framework, it performs the following steps in an
# isolated subshell with its own cleanup trap:
# 1. Creates a temporary table in a Cloud SQL database.
# 2. Creates a dedicated Python virtual environment.
# 3. Installs dependencies.
# 4. Runs the tests.
# 5. Cleans up the virtual environment and the database table.

# --- Configuration ---
: "${GCP_PROJECT:?Error: GCP_PROJECT environment variable not set.}"
: "${CLOUD_SQL_INSTANCE:?Error: CLOUD_SQL_INSTANCE environment variable not set.}"
: "${DATABASE_NAME:?Error: DATABASE_NAME environment variable not set.}"
: "${DB_USER:?Error: DB_USER environment variable not set.}"

TABLE_NAME="hotel_table"
QUICKSTART_PYTHON_DIR="docs/en/getting-started/quickstart/python"

# --- Main Loop for Orchestrators ---
if [ ! -d "$QUICKSTART_PYTHON_DIR" ]; then
  echo "Error: Quickstart directory not found at '$QUICKSTART_PYTHON_DIR'"
  exit 1
fi

echo "--- Starting test run for Python orchestrators ---"
for ORCH_DIR in "$QUICKSTART_PYTHON_DIR"/*/; do
  # Ensure it's a directory before processing
  if [ ! -d "$ORCH_DIR" ]; then
    continue
  fi

  ( # Start a subshell for each orchestrator to encapsulate its environment and cleanup.
    set -e
    ORCH_NAME=$(basename "$ORCH_DIR")

    # This cleanup function will be called when the subshell exits.
    cleanup_orch() {
      echo "--- Cleaning up for $ORCH_NAME ---"
      
      echo "Dropping temporary table '$TABLE_NAME' for $ORCH_NAME..."
      gcloud sql connect "$CLOUD_SQL_INSTANCE" --project="$GCP_PROJECT" --user="$DB_USER" --database="$DATABASE_NAME" --quiet <<< "DROP TABLE IF EXISTS $TABLE_NAME;"
      
      # The venv is created inside the orchestrator directory.
      if [ -d ".venv" ]; then
          echo "Removing virtual environment for $ORCH_NAME..."
          rm -rf ".venv"
      fi
      echo "--- Cleanup for $ORCH_NAME complete ---"
    }
    trap cleanup_orch EXIT

    echo "--- Processing orchestrator: $ORCH_NAME ---"
    
    # Change into the orchestrator's directory.
    cd "$ORCH_DIR"

    # 1. Database Setup for this orchestrator
    echo "Creating temporary table '$TABLE_NAME' for $ORCH_NAME..."
    gcloud sql connect "$CLOUD_SQL_INSTANCE" --project="$GCP_PROJECT" --user="$DB_USER" --database="$DATABASE_NAME" --quiet <<EOF
CREATE TABLE $TABLE_NAME (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    location VARCHAR(255),
    rating INT
);
EOF
    echo "Table '$TABLE_NAME' created."

    # 2. Virtual Environment Creation
    VENV_DIR=".venv"
    echo "Creating Python virtual environment in '$(pwd)/$VENV_DIR'..."
    python3 -m venv "$VENV_DIR"
    source "$VENV_DIR/bin/activate"

    # 3. Dependency Installation
    echo "Installing dependencies for $ORCH_NAME..."
    if [ -f "requirements.txt" ]; then
      pip install -r requirements.txt
    else
      echo "Warning: requirements.txt not found. Skipping."
    fi

    # 4. Test Execution
    echo "Running tests for $ORCH_NAME..."
    pytest

    echo "--- Finished processing $ORCH_NAME ---"
    # Cleanup for this orchestrator will be triggered by the trap on subshell exit.
  )
done

echo "--- All Python quickstart tests completed ---"
