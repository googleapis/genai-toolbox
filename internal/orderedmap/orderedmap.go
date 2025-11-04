// Copyright 2024 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package orderedmap

import (
	"bytes"
	"encoding/json"
)

// Column represents a single column in a row.
type Column struct {
	Name  string
	Value any
}

// Row represents a row of data with columns in a specific order.
type Row struct {
	Columns []Column
}

// Add adds a new column to the row.
func (r *Row) Add(name string, value any) {
	r.Columns = append(r.Columns, Column{Name: name, Value: value})
}

// MarshalJSON implements the json.Marshaler interface for the Row struct.
// It marshals the row into a JSON object, preserving the order of the columns.
func (r Row) MarshalJSON() ([]byte, error) {
	var buf bytes.Buffer
	buf.WriteString("{")
	for i, col := range r.Columns {
		if i > 0 {
			buf.WriteString(",")
		}
		// Marshal the key
		key, err := json.Marshal(col.Name)
		if err != nil {
			return nil, err
		}
		buf.Write(key)
		buf.WriteString(":")
		// Marshal the value
		val, err := json.Marshal(col.Value)
		if err != nil {
			return nil, err
		}
		buf.Write(val)
	}
	buf.WriteString("}")
	return buf.Bytes(), nil
}
