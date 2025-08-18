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

package util

import (
	"encoding/base64"
	"fmt"
	"time"
	"strings"

	"cloud.google.com/go/firestore"
	"google.golang.org/genproto/googleapis/type/latlng"
)

// JSONToFirestoreValue converts a JSON value with type information to a Firestore-compatible value
// The input should be a map with a single key indicating the type (e.g., "stringValue", "integerValue")
// If a client is provided, referenceValue types will be converted to *firestore.DocumentRef
func JSONToFirestoreValue(value interface{}, client *firestore.Client) (interface{}, error) {
	if value == nil {
		return nil, nil
	}

	switch v := value.(type) {
	case map[string]interface{}:
		// Check for typed values
		if len(v) == 1 {
			for key, val := range v {
				switch key {
				case "nullValue":
					return nil, nil
				case "booleanValue":
					return val, nil
				case "stringValue":
					return val, nil
				case "integerValue":
					// Convert to int64
					switch num := val.(type) {
					case float64:
						return int64(num), nil
					case int:
						return int64(num), nil
					case int64:
						return num, nil
					case string:
						// Parse string representation if needed
						var i int64
						if _, err := fmt.Sscanf(num, "%d", &i); err == nil {
							return i, nil
						}
					}
					return nil, fmt.Errorf("invalid integer value: %v", val)
				case "doubleValue":
					// Convert to float64
					switch num := val.(type) {
					case float64:
						return num, nil
					case int:
						return float64(num), nil
					case int64:
						return float64(num), nil
					}
					return nil, fmt.Errorf("invalid double value: %v", val)
				case "bytesValue":
					// Decode base64 string to bytes
					if str, ok := val.(string); ok {
						return base64.StdEncoding.DecodeString(str)
					}
					return nil, fmt.Errorf("bytes value must be a base64 encoded string")
				case "timestampValue":
					// Parse timestamp
					if str, ok := val.(string); ok {
						t, err := time.Parse(time.RFC3339Nano, str)
						if err != nil {
							return nil, fmt.Errorf("invalid timestamp format: %w", err)
						}
						return t, nil
					}
					return nil, fmt.Errorf("timestamp value must be a string")
				case "geoPointValue":
					// Convert to LatLng
					if geoMap, ok := val.(map[string]interface{}); ok {
						lat, latOk := geoMap["latitude"].(float64)
						lng, lngOk := geoMap["longitude"].(float64)
						if latOk && lngOk {
							return &latlng.LatLng{
								Latitude:  lat,
								Longitude: lng,
							}, nil
						}
					}
					return nil, fmt.Errorf("invalid geopoint value format")
				case "arrayValue":
					// Convert array
					if arrayMap, ok := val.(map[string]interface{}); ok {
						if values, ok := arrayMap["values"].([]interface{}); ok {
							result := make([]interface{}, len(values))
							for i, item := range values {
								converted, err := JSONToFirestoreValue(item, client)
								if err != nil {
									return nil, fmt.Errorf("array item %d: %w", i, err)
								}
								result[i] = converted
							}
							return result, nil
						}
					}
					return nil, fmt.Errorf("invalid array value format")
				case "mapValue":
					// Convert map
					if mapMap, ok := val.(map[string]interface{}); ok {
						if fields, ok := mapMap["fields"].(map[string]interface{}); ok {
							result := make(map[string]interface{})
							for k, v := range fields {
								converted, err := JSONToFirestoreValue(v, client)
								if err != nil {
									return nil, fmt.Errorf("map field %q: %w", k, err)
								}
								result[k] = converted
							}
							return result, nil
						}
					}
					return nil, fmt.Errorf("invalid map value format")
				case "referenceValue":
					// Convert to DocumentRef if client is provided
					if strVal, ok := val.(string); ok {
						if client != nil && isValidDocumentPath(strVal) {
							return client.Doc(strVal), nil
						}
						// Return the path as string if no client or invalid path
						return strVal, nil
					}
					return nil, fmt.Errorf("reference value must be a string")
				default:
					// If not a typed value, treat as regular map
					return convertPlainMap(v, client)
				}
			}
		}
		// Regular map without type annotation
		return convertPlainMap(v, client)
	default:
		// Plain values (for backward compatibility)
		return value, nil
	}
}

// convertPlainMap converts a plain map to Firestore format
func convertPlainMap(m map[string]interface{}, client *firestore.Client) (map[string]interface{}, error) {
	result := make(map[string]interface{})
	for k, v := range m {
		converted, err := JSONToFirestoreValue(v, client)
		if err != nil {
			return nil, fmt.Errorf("field %q: %w", k, err)
		}
		result[k] = converted
	}
	return result, nil
}

// FirestoreValueToJSON converts a Firestore value to a simplified JSON representation
// This removes type information and returns plain values
func FirestoreValueToJSON(value interface{}) interface{} {
	if value == nil {
		return nil
	}

	switch v := value.(type) {
	case time.Time:
		return v.Format(time.RFC3339Nano)
	case *latlng.LatLng:
		return map[string]interface{}{
			"latitude":  v.Latitude,
			"longitude": v.Longitude,
		}
	case []byte:
		return base64.StdEncoding.EncodeToString(v)
	case []interface{}:
		result := make([]interface{}, len(v))
		for i, item := range v {
			result[i] = FirestoreValueToJSON(item)
		}
		return result
	case map[string]interface{}:
		result := make(map[string]interface{})
		for k, val := range v {
			result[k] = FirestoreValueToJSON(val)
		}
		return result
	case *firestore.DocumentRef:
		return v.Path
	default:
		return value
	}
}

// isValidDocumentPath checks if a string is a valid Firestore document path
// Valid paths have an even number of segments (collection/doc/collection/doc...)
func isValidDocumentPath(path string) bool {
	if path == "" {
		return false
	}

	// Split the path by '/' and check if it has an even number of segments
	segments := splitPath(path)
	return len(segments) > 0 && len(segments)%2 == 0
}

// splitPath splits a path by '/' while handling empty segments correctly
func splitPath(path string) []string {
    rawSegments := strings.Split(path, "/")
    var segments []string
     for _, s := range rawSegments {
         if s != "" {
             segments = append(segments, s)
         }
     }
     return segments
 }
