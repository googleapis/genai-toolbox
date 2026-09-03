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

# Seeds the tables the alloydb-postgres evalset reads. The integration tests
# create their tables with generated names and drop them on the way out, so a
# scenario that asks what is in the database has nothing to find otherwise.
#
# Idempotent, and the tables are left in place: they are fixtures shared by
# every run, which is why there is no matching teardown script.

set -euo pipefail

: "${ALLOYDB_POSTGRES_PROJECT:?}" "${ALLOYDB_POSTGRES_REGION:?}"
: "${ALLOYDB_POSTGRES_CLUSTER:?}" "${ALLOYDB_POSTGRES_INSTANCE:?}"
: "${ALLOYDB_POSTGRES_DATABASE:?}" "${ALLOYDB_POSTGRES_USER:?}"
: "${ALLOYDB_POSTGRES_PASSWORD:?}"

# The toolbox reaches the instance through the AlloyDB connector, which psql
# cannot speak, so the address has to come from the admin API.
host=$(gcloud alloydb instances describe "${ALLOYDB_POSTGRES_INSTANCE}" \
  --cluster="${ALLOYDB_POSTGRES_CLUSTER}" \
  --region="${ALLOYDB_POSTGRES_REGION}" \
  --project="${ALLOYDB_POSTGRES_PROJECT}" \
  --format='value(publicIpAddress)')

if [[ -z "${host}" ]]; then
  echo "no public IP on ${ALLOYDB_POSTGRES_INSTANCE}; cannot seed" >&2
  exit 1
fi

echo "seeding ${ALLOYDB_POSTGRES_DATABASE} on ${host}"

# ON_ERROR_STOP so a failed statement is not reported as a successful seed.
PGPASSWORD="${ALLOYDB_POSTGRES_PASSWORD}" psql \
  --host="${host}" \
  --username="${ALLOYDB_POSTGRES_USER}" \
  --dbname="${ALLOYDB_POSTGRES_DATABASE}" \
  --set=ON_ERROR_STOP=1 \
  --quiet <<'SQL'
CREATE SCHEMA IF NOT EXISTS eval_fixtures;

CREATE TABLE IF NOT EXISTS eval_fixtures.customers (
  id          bigserial PRIMARY KEY,
  email       text NOT NULL UNIQUE,
  country     text NOT NULL,
  created_at  timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS eval_fixtures.orders (
  id           bigserial PRIMARY KEY,
  customer_id  bigint NOT NULL REFERENCES eval_fixtures.customers(id),
  status       text NOT NULL,
  total_cents  bigint NOT NULL,
  placed_at    timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS orders_customer_id_idx
  ON eval_fixtures.orders (customer_id);

-- Guarded so reruns neither duplicate rows nor grow the tables without bound.
INSERT INTO eval_fixtures.customers (email, country)
SELECT format('user%s@example.com', i),
       (ARRAY['US','GB','DE','JP'])[1 + i % 4]
FROM generate_series(1, 5000) AS i
WHERE NOT EXISTS (SELECT 1 FROM eval_fixtures.customers);

INSERT INTO eval_fixtures.orders (customer_id, status, total_cents)
SELECT 1 + (i % 5000),
       (ARRAY['placed','shipped','refunded'])[1 + i % 3],
       (i * 37) % 500000
FROM generate_series(1, 20000) AS i
WHERE NOT EXISTS (SELECT 1 FROM eval_fixtures.orders);

-- Dead tuples, so list_top_bloated_tables has something to rank. No VACUUM
-- afterwards, which is the point.
UPDATE eval_fixtures.orders
SET status = status
WHERE id % 2 = 0;

-- One object per kind the evalset asks about, so list_views, list_sequences,
-- list_triggers and list_stored_procedure each return something. The bigserial
-- columns above already supply sequences; this one is named for legibility.
CREATE OR REPLACE VIEW eval_fixtures.recent_orders AS
SELECT o.id, c.email, o.status, o.total_cents, o.placed_at
FROM eval_fixtures.orders o
JOIN eval_fixtures.customers c ON c.id = o.customer_id
WHERE o.placed_at > now() - interval '30 days';

CREATE SEQUENCE IF NOT EXISTS eval_fixtures.invoice_number_seq;

CREATE OR REPLACE FUNCTION eval_fixtures.stamp_placed_at()
RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
  NEW.placed_at := coalesce(NEW.placed_at, now());
  RETURN NEW;
END;
$$;

DROP TRIGGER IF EXISTS orders_stamp_placed_at ON eval_fixtures.orders;
CREATE TRIGGER orders_stamp_placed_at
  BEFORE INSERT ON eval_fixtures.orders
  FOR EACH ROW EXECUTE FUNCTION eval_fixtures.stamp_placed_at();

CREATE OR REPLACE PROCEDURE eval_fixtures.archive_refunded()
LANGUAGE plpgsql AS $$
BEGIN
  DELETE FROM eval_fixtures.orders WHERE status = 'refunded' AND false;
END;
$$;

-- A publication, so list_publication_tables is not empty. Publications alone
-- retain no WAL; only replication slots do, which is why none is created.
DO $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_publication WHERE pubname = 'eval_orders_pub') THEN
    CREATE PUBLICATION eval_orders_pub FOR TABLE eval_fixtures.orders;
  END IF;
END;
$$;

-- list_table_stats and the bloat estimate both read planner statistics, which
-- are otherwise whatever autovacuum last happened to collect.
ANALYZE eval_fixtures.customers;
ANALYZE eval_fixtures.orders;
SQL

# Separate call, and deliberately without ON_ERROR_STOP: pg_stat_statements has
# to be in shared_preload_libraries, so CREATE EXTENSION can fail for reasons
# the seed cannot fix. list_query_stats reads it, and its scenario accepts an
# unavailable extension as an answer.
PGPASSWORD="${ALLOYDB_POSTGRES_PASSWORD}" psql \
  --host="${host}" \
  --username="${ALLOYDB_POSTGRES_USER}" \
  --dbname="${ALLOYDB_POSTGRES_DATABASE}" \
  --quiet \
  --command="CREATE EXTENSION IF NOT EXISTS pg_stat_statements;" ||
  echo "pg_stat_statements unavailable; list_query_stats will return nothing"

echo "seed complete"
