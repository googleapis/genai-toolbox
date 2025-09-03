#!/bin/bash

set -e
set -u


# --- Configuration ---
: "${GCP_PROJECT:?Error: GCP_PROJECT environment variable not set.}"
: "${CLOUD_SQL_INSTANCE:?Error: CLOUD_SQL_INSTANCE environment variable not set.}"
: "${DATABASE_NAME:?Error: DATABASE_NAME environment variable not set.}"
: "${DB_USER:?Error: DB_USER environment variable not set.}"
: "${GOOGLE_API_KEY:?Error: GOOGLE_API_KEY environment variable not set.}"

TABLE_NAME="hotels"
QUICKSTART_PYTHON_DIR="docs/en/getting-started/quickstart/python"


if [ ! -d "$QUICKSTART_PYTHON_DIR" ]; then
  echo "Error: Quickstart directory not found at '$QUICKSTART_PYTHON_DIR'"
  exit 1
fi

echo "--- Starting test run for Python orchestrators ---"
for ORCH_DIR in "$QUICKSTART_PYTHON_DIR"/*/; do
  if [ ! -d "$ORCH_DIR" ]; then
    continue
  fi

  (
    set -e
    ORCH_NAME=$(basename "$ORCH_DIR")

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
  id            INTEGER NOT NULL PRIMARY KEY,
  name          VARCHAR NOT NULL,
  location      VARCHAR NOT NULL,
  price_tier    VARCHAR NOT NULL,
  checkin_date  DATE    NOT NULL,
  checkout_date DATE    NOT NULL,
  booked        BIT     NOT NULL
);

INSERT INTO $TABLE_NAME (id, name, location, price_tier, checkin_date, checkout_date, booked)
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
EOF
    echo "Table '$TABLE_NAME' created and data inserted."

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
