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

# Deletes the backups the cloud-sql-postgres-admin evalset creates. The toolbox
# has no delete tool, so this cannot be a scenario.
#
# Two callers: EvalBench's tear_down_script hook, which does the deleting, and
# a scheduled sweep for what a killed eval leaks before the hook can run.
#
# Scope is deliberately narrow: one instance, on-demand backups only. Scheduled
# backups are never matched.

set -euo pipefail

: "${TEARDOWN_PROJECT:?}" "${TEARDOWN_INSTANCE:?}"

# A flag rather than $1, because EvalBench passes the session directory there.
if [[ -n "${TEARDOWN_SWEEP:-}" ]]; then
  # No marker to bound against, so spare anything recent enough to belong to a
  # build still in flight.
  cutoff=$(date -u -d "-${TEARDOWN_SWEEP_AGE_HOURS:-6} hours" +%Y-%m-%dT%H:%M:%SZ)
  filter="type=ON_DEMAND AND enqueuedTime<${cutoff}"
  echo "sweeping on-demand backups on ${TEARDOWN_INSTANCE} older than ${cutoff}"
else
  marker="/workspace/eval-start-${TOOLBOX_PREBUILT}"
  if [[ ! -f "${marker}" ]]; then
    echo "evals did not run; nothing to tear down"
    exit 0
  fi
  cutoff=$(cat "${marker}")
  filter="type=ON_DEMAND AND enqueuedTime>=${cutoff}"
  echo "deleting on-demand backups on ${TEARDOWN_INSTANCE} since ${cutoff}"
  # Same marker the eval steps use. EvalBench discards this script's exit code,
  # so under the hook the marker is the only thing that reddens the build.
  trap '[ $? -eq 0 ] || touch "/workspace/failed-teardown-${TOOLBOX_PREBUILT}"' EXIT
fi

ids=$(gcloud sql backups list \
  --project="${TEARDOWN_PROJECT}" \
  --instance="${TEARDOWN_INSTANCE}" \
  --filter="${filter}" \
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
