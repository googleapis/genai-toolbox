#!/bin/bash

set -e
set -u

# --- Configuration ---
: "${GCP_PROJECT:?Error: GCP_PROJECT environment variable not set.}"
: "${DATABASE_NAME:?Error: DATABASE_NAME environment variable not set.}"
: "${DB_USER:?Error: DB_USER environment variable not set.}"
: "${GOOGLE_API_KEY:?Error: GOOGLE_API_KEY environment variable not set.}"
: "${PGHOST:?Error: PGHOST environment variable not set.}"
: "${PGPORT:?Error: PGPORT environment variable not set.}"
: "${PGPASSWORD:?Error: PGPASSWORD environment variable not set.}"

TABLE_NAME="hotels"
QUICKSTART_JS_DIR="docs/en/getting-started/quickstart/js"

echo "Google API Key is set (first 4 chars): $(echo "${GOOGLE_API_KEY}" | head -c 4)"
export PGPASSWORD

if [ ! -d "$QUICKSTART_JS_DIR" ]; then
  echo "Error: Quickstart directory not found at '$QUICKSTART_JS_DIR'"
  exit 1
fi

for FW_DIR in "$QUICKSTART_JS_DIR"/*/; do
  if [ ! -d "$FW_DIR" ]; then
    continue
  fi

  (
    set -e
    FW_NAME=$(basename "$FW_DIR")

    cleanup_fw() {
      psql -h "$PGHOST" -p "$PGPORT" -U "$DB_USER" -d "$DATABASE_NAME" -c "DROP TABLE IF EXISTS $TABLE_NAME;"
    }
    trap cleanup_fw EXIT

    cd "$FW_DIR"

    psql -h "$PGHOST" -p "$PGPORT" -U "$DB_USER" -d "$DATABASE_NAME" <<EOF
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

    if [ -f "package.json" ]; then
      npm install
    else
      echo "Warning: package.json not found. Skipping."
    fi

    npm test

  )
done

echo "--- All JavaScript quickstart tests completed ---"
