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

# Deletes the resources an admin evalset created. The toolbox has no delete tool,
# so this cannot be a scenario and runs as its own Cloud Build step instead.
#
# Scope is deliberately narrow: one instance, on-demand backups only, and only
# those started after this build's evals began. Scheduled backups are never
# matched.

set -euo pipefail

marker="/workspace/eval-start-${TOOLBOX_PREBUILT}"
if [[ ! -f "${marker}" ]]; then
  echo "evals did not run; nothing to tear down"
  exit 0
fi
start=$(cat "${marker}")

echo "deleting on-demand backups on ${TEARDOWN_INSTANCE} since ${start}"

ids=$(gcloud sql backups list \
  --project="${TEARDOWN_PROJECT}" \
  --instance="${TEARDOWN_INSTANCE}" \
  --filter="type=ON_DEMAND AND windowStartTime>=${start}" \
  --format='value(id)')

if [[ -z "${ids}" ]]; then
  echo "no backups to delete"
  exit 0
fi

for id in ${ids}; do
  echo "deleting backup ${id}"
  gcloud sql backups delete "${id}" \
    --project="${TEARDOWN_PROJECT}" \
    --instance="${TEARDOWN_INSTANCE}" \
    --quiet
done
