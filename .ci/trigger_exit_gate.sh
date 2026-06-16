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
#
# Uploads the OSS Exit Gate publishing manifest, which triggers external
# publication of whichever package set was just pushed to the internal
# Artifact Registry.
#
# Usage:
#   bash .ci/trigger_exit_gate.sh <registry>
#     registry: "npm" or "pypi" — determines the GCS subpath Exit Gate watches
#
# Required env vars (supplied by Cloud Build):
#   BUILD_ID  Cloud Build's auto-provided unique build ID

set -eo pipefail

REGISTRY="${1:?registry argument required: npm or pypi}"
case "${REGISTRY}" in
  npm|pypi) ;;
  *)
    echo "ERROR: unsupported registry '${REGISTRY}' (expected: npm or pypi)" >&2
    exit 1
    ;;
esac

# OSS Exit Gate constants — owned by Exit Gate, not build configuration.
readonly EXIT_GATE_PROJECT="mcp-toolbox"
readonly MANIFEST_BUCKET="oss-exit-gate-prod-projects-bucket"

VERSION="v$(cat ./cmd/version.txt)"
MANIFEST="${VERSION}-${BUILD_ID}.json"
MANIFEST_PATH="gs://${MANIFEST_BUCKET}/${EXIT_GATE_PROJECT}/${REGISTRY}/manifests/${MANIFEST}"

echo '{"publish_all": true}' > "${MANIFEST}"

gcloud storage cp "${MANIFEST}" "${MANIFEST_PATH}"

echo "Manifest uploaded: ${MANIFEST_PATH}"
echo "Exit Gate will email a confirmation when the external publish completes."
