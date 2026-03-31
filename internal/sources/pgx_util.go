// Copyright 2026 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package sources

import (
	"context"
	"fmt"

	"github.com/googleapis/genai-toolbox/internal/sqlcommenter"
	"github.com/googleapis/genai-toolbox/internal/util/orderedmap"
	"github.com/jackc/pgx/v5"
)

// PgxQueryer abstracts connection pools and wrapper classes that can execute native Pgx queries.
type PgxQueryer interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
}

// RunSQLWithPgxQueryer executes a standard SQL statement with SQLCommenter telemetry
// across any driver natively supporting the pgx execution interface.
func RunSQLWithPgxQueryer(ctx context.Context, queryer PgxQueryer, statement string, params []any, driver string) (any, error) {
	// Inject the database driver into the context for SQLCommenter
	ctx = sqlcommenter.WithDBDriver(ctx, driver)
	// Decorate the statement with SQLCommenter metadata from the context
	statement = sqlcommenter.AppendComment(ctx, statement)

	results, err := queryer.Query(ctx, statement, params...)
	if err != nil {
		return nil, fmt.Errorf("unable to execute query: %w", err)
	}
	defer results.Close()

	fields := results.FieldDescriptions()
	var out []any
	for results.Next() {
		values, err := results.Values()
		if err != nil {
			return nil, fmt.Errorf("unable to parse row: %w", err)
		}
		row := orderedmap.Row{}
		for i, f := range fields {
			row.Add(f.Name, values[i])
		}
		out = append(out, row)
	}
	// this will catch actual query execution errors
	if err := results.Err(); err != nil {
		return nil, fmt.Errorf("unable to execute query: %w", err)
	}
	return out, nil
}
