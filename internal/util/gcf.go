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

package util

import (
	"encoding/json"
	"reflect"

	gcf "github.com/blackwell-systems/gcf-go"
	"github.com/googleapis/mcp-toolbox/internal/util/orderedmap"
)

// EncodeGCFToolResult encodes a slice of tool results as a single Graph Compact
// Format (https://gcformat.com) generic-profile block, returning the wire and true,
// or "" and false when the caller should keep the per-row JSON. The substitution is
// conservative: GCF is offered only when it encodes without error, is strictly
// smaller than the JSON the rows would otherwise produce (never-grow), and decodes
// back to the same rows (lossless), so enabling it never grows or alters a result.
//
// Query results arrive as orderedmap.Row values, which gcf-go cannot reflect
// directly: it encodes native Go types and does not honor json.Marshaler. Each Row
// is converted to a gcf.OrderedMap, which preserves the original query column order.
// Elements that are not rows are passed through unchanged.
func EncodeGCFToolResult(results []any) (string, bool) {
	converted := make([]any, len(results))
	for i, r := range results {
		converted[i] = convertRow(r)
	}

	// EncodeGenericChecked returns the numeric-domain error rather than panicking,
	// so an out-of-domain value falls back to JSON instead of failing the call.
	wire, err := gcf.EncodeGenericChecked(converted)
	if err != nil {
		return "", false
	}

	// Never-grow: compare against the JSON the rows would otherwise produce (one
	// json.Marshal per row, matching the per-row TextContent blocks of the JSON path).
	jsonLen := 0
	for _, r := range results {
		b, err := json.Marshal(r)
		if err != nil {
			return "", false
		}
		jsonLen += len(b)
	}
	if len(wire) >= jsonLen {
		return "", false
	}

	// Fail-safe: require a lossless round-trip back to the converted rows, so a value
	// GCF cannot carry exactly (e.g. a non-scalar column type) is caught and the JSON
	// is kept rather than a wire that dropped or altered it.
	decoded, err := gcf.DecodeGeneric(wire)
	if err != nil || !gcfValuesEqual(decoded, converted) {
		return "", false
	}

	return wire, true
}

func convertRow(v any) any {
	switch row := v.(type) {
	case orderedmap.Row:
		return columnsToOrdered(row.Columns)
	case *orderedmap.Row:
		if row == nil {
			return nil
		}
		return columnsToOrdered(row.Columns)
	default:
		return v
	}
}

func columnsToOrdered(cols []orderedmap.Column) *gcf.OrderedMap {
	m := gcf.NewOrderedMap()
	for _, c := range cols {
		m.Set(c.Name, c.Value)
	}
	return m
}

// gcfValuesEqual reports whether a gcf-decoded value equals the value that produced
// it. Object comparison is order-aware: query column order is significant and is
// preserved on the wire, so a reordering would be a change, not a match.
func gcfValuesEqual(a, b any) bool {
	switch av := a.(type) {
	case *gcf.OrderedMap:
		bv, ok := b.(*gcf.OrderedMap)
		if !ok || av.Len() != bv.Len() {
			return false
		}
		ak, bk := av.Keys(), bv.Keys()
		for i := range ak {
			if ak[i] != bk[i] {
				return false
			}
			avv, _ := av.Get(ak[i])
			bvv, _ := bv.Get(bk[i])
			if !gcfValuesEqual(avv, bvv) {
				return false
			}
		}
		return true
	case []any:
		bv, ok := b.([]any)
		if !ok || len(av) != len(bv) {
			return false
		}
		for i := range av {
			if !gcfValuesEqual(av[i], bv[i]) {
				return false
			}
		}
		return true
	default:
		return scalarEqual(a, b)
	}
}

// scalarEqual compares two leaf values. Same-type integers and floats compare
// exactly (so a large int64 is not masked through a float); mixed numeric widths
// compare by value; everything else (strings, bools, nil, and any type GCF could
// not round-trip) compares with reflect.DeepEqual.
func scalarEqual(a, b any) bool {
	switch av := a.(type) {
	case int64:
		if bv, ok := b.(int64); ok {
			return av == bv
		}
	case float64:
		if bv, ok := b.(float64); ok {
			return av == bv
		}
	}
	if af, aok := toFloat(a); aok {
		if bf, bok := toFloat(b); bok {
			return af == bf
		}
	}
	return reflect.DeepEqual(a, b)
}

func toFloat(v any) (float64, bool) {
	switch n := v.(type) {
	case int:
		return float64(n), true
	case int32:
		return float64(n), true
	case int64:
		return float64(n), true
	case float32:
		return float64(n), true
	case float64:
		return n, true
	default:
		return 0, false
	}
}
