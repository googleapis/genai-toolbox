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

TABLE_NAME="hotels_go"
QUICKSTART_GO_DIR="docs/en/getting-started/quickstart/go"
SQL_FILE=".ci/quickstart_test/setup_hotels_sample.sql"

PROXY_PID=""
TOOLBOX_PID=""

install_system_packages() {
  apt-get update && apt-get install -y \
    postgresql-client \
    wget \
    gettext-base  \
    netcat-openbsd
}

start_cloud_sql_proxy() {
  wget "https://storage.googleapis.com/cloud-sql-connectors/cloud-sql-proxy/v2.10.0/cloud-sql-proxy.linux.amd64" -O /usr/local/bin/cloud-sql-proxy
  chmod +x /usr/local/bin/cloud-sql-proxy
  cloud-sql-proxy "${CLOUD_SQL_INSTANCE}" &
  PROXY_PID=$!

  for i in {1..30}; do
    if nc -z 127.0.0.1 5432; then
      echo "Cloud SQL Proxy is up and running."
      return
    fi
    sleep 1
  done

  echo "Cloud SQL Proxy failed to start within the timeout period."
  exit 1
}

setup_toolbox() {
  TOOLBOX_YAML="/tools.yaml"
  echo "${TOOLS_YAML_CONTENT}" > "$TOOLBOX_YAML"
  if [ ! -f "$TOOLBOX_YAML" ]; then echo "Failed to create tools.yaml"; exit 1; fi
  wget "https://storage.googleapis.com/genai-toolbox/v${VERSION}/linux/amd64/toolbox" -O "/toolbox"
  chmod +x "/toolbox"
  /toolbox --tools-file "$TOOLBOX_YAML" &
  TOOLBOX_PID=$!
  sleep 2
}

setup_orch_table() {
  export TABLE_NAME
  envsubst < "$SQL_FILE" | psql -h "$PGHOST" -p "$PGPORT" -U "$DB_USER" -d "$DATABASE_NAME"
}

run_orch_test() {
  local orch_dir="$1"
  local orch_name
  orch_name=$(basename "$orch_dir")
  
  if [ "$orch_name" == "openAI" ]; then
      echo -e "\nSkipping framework '${orch_name}': Temporarily excluded."
      return
  fi
  
  (
    set -e
    setup_orch_table

    echo "--- Preparing module for $orch_name ---"
    cd "$orch_dir"

    if [ -f "go.mod" ]; then
      go mod tidy
    fi

    cd ..

    export ORCH_NAME="$orch_name"

    echo "--- Running tests for $orch_name ---"
    go test -v ./...
  )
}

cleanup_all() {
  echo "--- Final cleanup: Shutting down processes and dropping table ---"
  if [ -n "$TOOLBOX_PID" ]; then
    kill $TOOLBOX_PID || true
  fi
  if [ -n "$PROXY_PID" ]; then
    kill $PROXY_PID || true
  fi
}
trap cleanup_all EXIT

# Main script execution
install_system_packages
start_cloud_sql_proxy

export PGHOST=127.0.0.1
export PGPORT=5432
export PGPASSWORD="$DB_PASSWORD"
export GOOGLE_API_KEY="$GOOGLE_API_KEY"

setup_toolbox

for ORCH_DIR in "$QUICKSTART_GO_DIR"/*/; do
  if [ ! -d "$ORCH_DIR" ]; then
    continue
  fi
  run_orch_test "$ORCH_DIR"
done
