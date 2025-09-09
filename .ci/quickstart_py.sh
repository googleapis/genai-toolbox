#!/bin/bash

set -e
set -u

TABLE_NAME="hotels"
QUICKSTART_PYTHON_DIR="docs/en/getting-started/quickstart/python"
TOOLBOX_SETUP_DIR="/workspace/toolbox_setup"

apt-get update && apt-get install -y postgresql-client python3-venv curl wget

if [ ! -d "$QUICKSTART_PYTHON_DIR" ]; then
  exit 1
fi

wget https://storage.googleapis.com/cloud-sql-connectors/cloud-sql-proxy/v2.10.0/cloud-sql-proxy.linux.amd64 -O /usr/local/bin/cloud-sql-proxy
chmod +x /usr/local/bin/cloud-sql-proxy

cloud-sql-proxy "${CLOUD_SQL_INSTANCE}" &
PROXY_PID=$!

export PGHOST=127.0.0.1
export PGPORT=5432
export PGPASSWORD="$DB_PASSWORD"
export GOOGLE_API_KEY="$GOOGLE_API_KEY"

mkdir -p "${TOOLBOX_SETUP_DIR}"
echo "${TOOLS_YAML_CONTENT}" > "${TOOLBOX_SETUP_DIR}/tools.yaml"
if [ ! -f "${TOOLBOX_SETUP_DIR}/tools.yaml" ]; then echo "Failed to create tools.yaml"; exit 1; fi

curl -L "https://storage.googleapis.com/genai-toolbox/v${VERSION}/linux/amd64/toolbox" -o "${TOOLBOX_SETUP_DIR}/toolbox"
chmod +x "${TOOLBOX_SETUP_DIR}/toolbox"
if [ ! -f "${TOOLBOX_SETUP_DIR}/toolbox" ]; then echo "Failed to download toolbox"; exit 1; fi

echo "--- Starting Toolbox Server ---"
cd "${TOOLBOX_SETUP_DIR}"
./toolbox --tools-file ./tools.yaml &
TOOLBOX_PID=$!
cd "/workspace"
sleep 5

cleanup_all() {
  kill $TOOLBOX_PID || true
  kill $PROXY_PID || true
}
trap cleanup_all EXIT


for ORCH_DIR in "$QUICKSTART_PYTHON_DIR"/*/; do
  if [ ! -d "$ORCH_DIR" ]; then
    continue
  fi
  set -e
  ORCH_NAME=$(basename "$ORCH_DIR")

  # Drop table before test
  psql -h "$PGHOST" -p "$PGPORT" -U "$DB_USER" -d "$DATABASE_NAME" -c "DROP TABLE IF EXISTS $TABLE_NAME;"

  # Create table and insert data
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

  # Create venv in python folder for each orchestrator
  cd "$QUICKSTART_PYTHON_DIR"
  VENV_DIR=".venv"
  python3 -m venv "$VENV_DIR"
  source "$VENV_DIR/bin/activate"

  if [ -f "requirements.txt" ]; then
    pip install -r "$ORCH_DIR/requirements.txt"
  else
    echo "Warning: requirements.txt not found. Skipping."
  fi

  echo "Running tests for $ORCH_NAME..."
  ORCH_NAME="$ORCH_NAME" pytest

  # Deactivate and remove venv
  deactivate || true
  rm -rf "$VENV_DIR"

  cd -
done
