#!/bin/bash
# Copyright 2026 Google LLC
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

set -e

# Configuration
SAMPLES_ROOT="docs/en/samples/pre_post_processing"
SQL_FILE=".ci/samples_test/setup_hotels_sample.sql"
TABLE_NAME="hotels_samples"
VERSION=$(cat ./cmd/version.txt)

# Process IDs & Logs
PROXY_PID=""
TOOLBOX_PID=""
PROXY_LOG="cloud_sql_proxy.log"
TOOLBOX_LOG="toolbox_server.log"

install_system_packages() {
  echo "Installing system packages..."
  apt-get update && apt-get install -y \
    postgresql-client \
    python3-venv \
    wget \
    gettext-base  \
    netcat-openbsd
}

start_cloud_sql_proxy() {
  echo "Starting Cloud SQL Proxy..."
  wget -q "https://storage.googleapis.com/cloud-sql-connectors/cloud-sql-proxy/v2.10.0/cloud-sql-proxy.linux.amd64" -O /usr/local/bin/cloud-sql-proxy
  chmod +x /usr/local/bin/cloud-sql-proxy
  cloud-sql-proxy "${CLOUD_SQL_INSTANCE}" > "$PROXY_LOG" 2>&1 &
  PROXY_PID=$!

  # Health check (Idea 3)
  for i in {1..30}; do
    if nc -z 127.0.0.1 5432; then
      echo "Cloud SQL Proxy is up and running."
      return
    fi
    sleep 1
  done
  echo "ERROR: Cloud SQL Proxy failed to start. Logs:"
  cat "$PROXY_LOG"
  exit 1
}

setup_toolbox() {
  echo "Setting up Toolbox server..."
  TOOLBOX_YAML="/tools.yaml"
  echo "${TOOLS_YAML_CONTENT}" > "$TOOLBOX_YAML"
  wget -q "https://storage.googleapis.com/genai-toolbox/v${VERSION}/linux/amd64/toolbox" -O "/toolbox"
  chmod +x "/toolbox"
  /toolbox --tools-file "$TOOLBOX_YAML" > "$TOOLBOX_LOG" 2>&1 &
  TOOLBOX_PID=$!
  
  # Health check (Idea 3)
  for i in {1..15}; do
    if nc -z 127.0.0.1 5000; then
      echo "Toolbox server is up and running."
      return
    fi
    sleep 1
  done
  echo "ERROR: Toolbox server failed to start. Logs:"
  cat "$TOOLBOX_LOG"
  exit 1
}

setup_db_table() {
  echo "Setting up database table $TABLE_NAME..."
  export TABLE_NAME
  envsubst < "$SQL_FILE" | psql -h 127.0.0.1 -p 5432 -U "$DB_USER" -d "$DATABASE_NAME"
}

run_python_sample() {
  local dir=$1
  local orch_name=$(basename "$dir")
  echo "--- Testing Python sample: $orch_name ---"
  
  (
    cd "$dir"
    python3 -m venv .venv
    source .venv/bin/activate
    pip install -q -r requirements.txt pytest
    
    # Idea 5: Use native pytest instead of shell grep
    echo "Running native pytest for $orch_name..."
    export ORCH_NAME="$orch_name"
    export PYTHONPATH="../../" # Path to docs/en/samples/pre_post_processing
    pytest ../agent_test.py
    local exit_code=$?
    
    if [ $exit_code -ne 0 ]; then
      echo "ERROR: Pytest failed for $orch_name"
      exit $exit_code
    fi
    
    echo "SUCCESS: $orch_name"
    rm -rf .venv
  )
}

run_js_sample() {
  local dir=$1
  local orch_name=$(basename "$dir")
  echo "--- Testing JS sample: $orch_name ---"
  (
    cd "$dir"
    npm install -q
    node agent.js > output.log 2>&1
    local exit_code=$?

    if [ $exit_code -ne 0 ]; then
      echo "ERROR: JS Sample failed for $orch_name"
      cat output.log
      exit $exit_code
    fi

    echo "SUCCESS: $orch_name"
    rm -rf node_modules output.log
  )
}

cleanup() {
  echo "Cleaning up background processes..."
  [ -n "$TOOLBOX_PID" ] && kill "$TOOLBOX_PID" || true
  [ -n "$PROXY_PID" ] && kill "$PROXY_PID" || true
}
trap cleanup EXIT

# Execution Flow
install_system_packages
start_cloud_sql_proxy

export PGHOST=127.0.0.1
export PGPORT=5432
export PGPASSWORD="$DB_PASSWORD"
export GOOGLE_API_KEY="$GOOGLE_API_KEY"

setup_toolbox
setup_db_table

# Arguments
TARGET_LANG=$1

# Discovery and run samples
echo "Discovering samples in $SAMPLES_ROOT for language: ${TARGET_LANG:-all}"

# Run Python samples
if [[ -z "$TARGET_LANG" || "$TARGET_LANG" == "python" ]]; then
    find "$SAMPLES_ROOT/python" -name "agent.py" | while read -r agent_file; do
        run_python_sample "$(dirname "$agent_file")"
    done
fi

# Run JS samples
if [[ -z "$TARGET_LANG" || "$TARGET_LANG" == "js" ]]; then
    find "$SAMPLES_ROOT/js" -name "agent.js" | while read -r agent_file; do
        run_js_sample "$(dirname "$agent_file")"
    done
fi

# Run Go samples
if [[ -z "$TARGET_LANG" || "$TARGET_LANG" == "go" ]]; then
    find "$SAMPLES_ROOT/go" -name "agent.go" | while read -r agent_file; do
        echo "Go testing coming soon..."
    done
fi
