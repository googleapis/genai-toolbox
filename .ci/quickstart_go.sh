#!/bin/bash

set -e
set -u

TABLE_NAME="hotels"
QUICKSTART_GO_DIR="docs/en/getting-started/quickstart/go"
TOOLBOX_SETUP_DIR="/workspace/toolbox_setup"

apt-get update && apt-get install -y postgresql-client curl wget

if [ ! -d "$QUICKSTART_GO_DIR" ]; then
  exit 1
fi

# The "openAI" framework is temporarily excluded from the test run because a
# valid API key is not yet available.
#
# To re-enable testing for this framework once an API key is configured,
# comment out the first line and uncomment the second line below.

frameworks=("genAI" "genkit" "langchain")
# frameworks=("genAI" "genkit" "langchain" "openAI")

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

for framework in "${frameworks[@]}"; do
    FW_DIR="${QUICKSTART_GO_DIR}/${framework}"

    if [ ! -d "$FW_DIR" ]; then
        echo -e "\nSkipping framework '${framework}': directory not found."
        continue
    fi

  (
    set -e
    FW_NAME=$(basename "$FW_DIR")

    cleanup_fw() {
      psql -h "$PGHOST" -p "$PGPORT" -U "$DB_USER" -d "$DATABASE_NAME" -c "DROP TABLE IF EXISTS $TABLE_NAME;"
    }
    trap cleanup_fw EXIT

    psql -h "$PGHOST" -p "$PGPORT" -U "$DB_USER" -d "$DATABASE_NAME" -c "DROP TABLE IF EXISTS $TABLE_NAME;"

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
    
    if [ -f "go.mod" ]; then
      go mod tidy
    else
      echo "Warning: go.mod not found. Skipping."
    fi

    go test ./...

  )
done

echo ""
echo "--- All Go quickstart tests completed ---"