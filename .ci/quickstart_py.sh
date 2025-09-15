# Copyright 2025 Google LLC
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

#!/bin/bash

set -e
set -u

TABLE_NAME="hotels_python"
QUICKSTART_PYTHON_DIR="docs/en/getting-started/quickstart/python"
TOOLBOX_SETUP_DIR="/workspace/toolbox_setup"
SQL_FILE=".ci/setup_hotels_sample.sql"

install_system_packages() {
  apt-get update && apt-get install -y postgresql-client python3-venv curl wget
}

start_cloud_sql_proxy() {
  wget https://storage.googleapis.com/cloud-sql-connectors/cloud-sql-proxy/v2.10.0/cloud-sql-proxy.linux.amd64 -O /usr/local/bin/cloud-sql-proxy
  chmod +x /usr/local/bin/cloud-sql-proxy
  cloud-sql-proxy "${CLOUD_SQL_INSTANCE}" &
  PROXY_PID=$!
  sleep 5
}

setup_toolbox() {
  mkdir -p "${TOOLBOX_SETUP_DIR}"
  echo "${TOOLS_YAML_CONTENT}" > "${TOOLBOX_SETUP_DIR}/tools.yaml"
  if [ ! -f "${TOOLBOX_SETUP_DIR}/tools.yaml" ]; then echo "Failed to create tools.yaml"; exit 1; fi
  curl -L "https://storage.googleapis.com/genai-toolbox/v${VERSION}/linux/amd64/toolbox" -o "${TOOLBOX_SETUP_DIR}/toolbox"
  chmod +x "${TOOLBOX_SETUP_DIR}/toolbox"
  if [ ! -f "${TOOLBOX_SETUP_DIR}/toolbox" ]; then echo "Failed to download toolbox"; exit 1; fi
  cd "${TOOLBOX_SETUP_DIR}"
  ./toolbox --tools-file ./tools.yaml &
  TOOLBOX_PID=$!
  cd "/workspace"
  sleep 2
}

cleanup_all() {
  echo "--- Final cleanup: Shutting down processes and dropping table ---"
  kill $TOOLBOX_PID || true
  psql -h "$PGHOST" -p "$PGPORT" -U "$DB_USER" -d "$DATABASE_NAME" -c "DROP TABLE IF EXISTS $TABLE_NAME;"
  kill $PROXY_PID || true
}
trap cleanup_all EXIT

setup_orch_table() {
  envsubst < "$SQL_FILE" | psql -h "$PGHOST" -p "$PGPORT" -U "$DB_USER" -d "$DATABASE_NAME"
}


run_orch_test() {
  local orch_dir="$1"
  local orch_name
  orch_name=$(basename "$orch_dir")
  (
    set -e
    cd "$orch_dir"
    VENV_DIR=".venv"
    python3 -m venv "$VENV_DIR"
    source "$VENV_DIR/bin/activate"
    pip install -r requirements.txt
    echo "Running tests for $orch_name..."
    cd ..
    ORCH_NAME="$orch_name" pytest
    rm -rvf "$VENV_DIR"
    psql -h "$PGHOST" -p "$PGPORT" -U "$DB_USER" -d "$DATABASE_NAME" -c "TRUNCATE TABLE $TABLE_NAME;"
  )
}

# Main script execution
install_system_packages
start_cloud_sql_proxy

export PGHOST=127.0.0.1
export PGPORT=5432
export PGPASSWORD="$DB_PASSWORD"
export GOOGLE_API_KEY="$GOOGLE_API_KEY"

setup_toolbox

if [ ! -d "$QUICKSTART_PYTHON_DIR" ]; then
  exit 1
fi

if [[ -f "$SQL_FILE" ]]; then
  setup_orch_table

for ORCH_DIR in "$QUICKSTART_PYTHON_DIR"/*/; do
  if [ ! -d "$ORCH_DIR" ]; then
    continue
  fi
  run_orch_test "$ORCH_DIR"
done
