// Copyright 2025 Google LLC
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

package mysqlcommon

import (
	"database/sql"
	"encoding/json"
	"reflect"
)

// ConvertToType handles casting mysql returns to the right type
// types for mysql driver: https://github.com/go-sql-driver/mysql/blob/v1.9.3/fields.go
// all numeric type or unknown type will be return as is.
func ConvertToType(t *sql.ColumnType, v any) (any, error) {
	switch t.ScanType() {
	case reflect.TypeOf(""), reflect.TypeOf([]byte{}), reflect.TypeOf(sql.NullString{}):
		// unmarshal JSON data before returning to prevent double marshaling
		if t.DatabaseTypeName() == "JSON" {
			// unmarshal JSON data before storing to prevent double marshaling
			var unmarshaledData any
			err := json.Unmarshal(v.([]byte), &unmarshaledData)
			if err != nil {
				return nil, err
			}
			return unmarshaledData, nil
		}
		return string(v.([]byte)), nil
	default:
		return v, nil
	}
}
