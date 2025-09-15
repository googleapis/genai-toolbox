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

TABLE_NAME="hotels_python"
QUICKSTART_PYTHON_DIR="docs/en/getting-started/quickstart/python"
SQL_FILE=".ci/setup_hotels_sample.sql"
DEPS_FILE=".ci/quickstart_dependencies.json"

install_system_packages() {
  apt-get update
  JQ_VERSION=$(jq -r '.apt.jq' "$DEPS_FILE")
  apt-get install -y "jq=${JQ_VERSION}"

  mapfile -t install_list < <(jq -r '.apt | to_entries | .[] | select(.key != "jq" and .value != null) | "\(.key)=\(.value)"' "$DEPS_FILE")

  if (( ${#install_list[@]} > 0 )); then
    apt-get install -y "${install_list[@]}"
  fi
}

start_cloud_sql_proxy() {
  CLOUD_SQL_PROXY_VERSION=$(jq -r '.cloud_sql_proxy' "$DEPS_FILE")
  wget "https://storage.googleapis.com/cloud-sql-connectors/cloud-sql-proxy/${CLOUD_SQL_PROXY_VERSION}/cloud-sql-proxy.linux.amd64" -O /usr/local/bin/cloud-sql-proxy
  chmod +x /usr/local/bin/cloud-sql-proxy
  cloud-sql-proxy "${CLOUD_SQL_INSTANCE}" &
  PROXY_PID=$!
  sleep 5
}

setup_toolbox() {
  TOOLBOX_YAML="/tools.yaml"
  echo "${TOOLS_YAML_CONTENT}" > "$TOOLBOX_YAML"
  if [ ! -f "$TOOLBOX_YAML" ]; then echo "Failed to create tools.yaml"; exit 1; fi
  curl -L "https://storage.googleapis.com/genai-toolbox/v${VERSION}/linux/amd64/toolbox" -o "/toolbox"
  chmod +x "/toolbox"
  if [ ! -f "/toolbox" ]; then echo "Failed to download toolbox"; exit 1; fi
  /toolbox --tools-file "$TOOLBOX_YAML" &
  TOOLBOX_PID=$!
  sleep 2
}

cleanup_all() {
  echo "--- Final cleanup: Shutting down processes and dropping table ---"
  kill $TOOLBOX_PID || true
  kill $PROXY_PID || true
}
trap cleanup_all EXIT

setup_orch_table() {
  export TABLE_NAME
  envsubst < "$SQL_FILE" | psql -h "$PGHOST" -p "$PGPORT" -U "$DB_USER" -d "$DATABASE_NAME"
}


run_orch_test() {
  local orch_dir="$1"
  local orch_name
  orch_name=$(basename "$orch_dir")
  (
    set -e
    setup_orch_table
    cd "$orch_dir"
    local VENV_DIR=".venv"
    python3 -m venv "$VENV_DIR"
    source "$VENV_DIR/bin/activate"
    pip install -r requirements.txt
    echo "--- Running tests for $orch_name ---"
    cd ..
    ORCH_NAME="$orch_name" pytest
    rm -rf "$VENV_DIR"
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

if [[ ! -f "$SQL_FILE" ]]; then
  exit 1
fi

for ORCH_DIR in "$QUICKSTART_PYTHON_DIR"/*/; do
  if [ ! -d "$ORCH_DIR" ]; then
    continue
  fi
  run_orch_test "$ORCH_DIR"
done
