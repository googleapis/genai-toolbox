#!/usr/bin/env bash
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

set -eo pipefail

if [ $# -lt 5 ]; then
  echo "Error: Missing required arguments." >&2
  echo "Usage: $0 <project_id> <region> <image_version> <cluster_name> <service_account>" >&2
  exit 1
fi

PROJECT_ID="$1"
REGION="$2"
IMAGE_VERSION="$3"
CLUSTER_NAME="$4"
SERVICE_ACCOUNT="$5"

echo "=========================================================="
echo "Recreating Dataproc cluster in project: ${PROJECT_ID}"
echo "Region:          ${REGION}"
echo "Image Version:   ${IMAGE_VERSION}"
echo "Cluster Name:    ${CLUSTER_NAME}"
echo "Service Account: ${SERVICE_ACCOUNT}"
echo "=========================================================="

# Check if the cluster exists and get its status, capturing output to handle NOT_FOUND
echo "Checking if cluster '${CLUSTER_NAME}' exists..."
set +e
STATE=$(gcloud dataproc clusters describe "${CLUSTER_NAME}" --region="${REGION}" --project="${PROJECT_ID}" --format="value(status.state)" 2>&1)
DESCRIBE_STATUS=$?
set -e

if [ ${DESCRIBE_STATUS} -eq 0 ]; then
  if [ "${STATE}" = "DELETING" ]; then
    echo "Cluster '${CLUSTER_NAME}' is currently in DELETING state. Waiting for deletion to complete..."
    while gcloud dataproc clusters describe "${CLUSTER_NAME}" --region="${REGION}" --project="${PROJECT_ID}" >/dev/null 2>&1; do
      echo "Waiting 10 seconds for cluster deletion..."
      sleep 10
    done
    echo "Cluster '${CLUSTER_NAME}' has been successfully deleted."
  else
    echo "Cluster '${CLUSTER_NAME}' exists (state: ${STATE}). Deleting it..."
    gcloud dataproc clusters delete "${CLUSTER_NAME}" \
      --region="${REGION}" \
      --project="${PROJECT_ID}" \
      --quiet
    echo "Cluster '${CLUSTER_NAME}' deleted successfully."
  fi
elif echo "${STATE}" | grep -q "NOT_FOUND"; then
  echo "Cluster '${CLUSTER_NAME}' does not exist. Skipping deletion."
else
  echo "Error querying cluster existence: ${STATE}" >&2
  exit ${DESCRIBE_STATUS}
fi

# Create the cluster
echo "Creating Dataproc cluster '${CLUSTER_NAME}'..."
gcloud dataproc clusters create "${CLUSTER_NAME}" \
  --region="${REGION}" \
  --project="${PROJECT_ID}" \
  --image-version="${IMAGE_VERSION}" \
  --service-account="${SERVICE_ACCOUNT}" \
  --scopes=cloud-platform \
  --no-address \
  --network=default \
  --master-machine-type=n4-standard-2 \
  --worker-machine-type=n4-standard-2 \
  --num-workers=2

echo "Cluster '${CLUSTER_NAME}' created successfully."

# Integration tests require at least one job to exist on the test cluster
echo "Submitting a test Spark job to cluster '${CLUSTER_NAME}'..."
gcloud dataproc jobs submit spark \
  --project="${PROJECT_ID}" \
  --region="${REGION}" \
  --cluster="${CLUSTER_NAME}" \
  --class=org.apache.spark.examples.SparkPi \
  --jars=file:///usr/lib/spark/examples/jars/spark-examples.jar \
  -- 100

