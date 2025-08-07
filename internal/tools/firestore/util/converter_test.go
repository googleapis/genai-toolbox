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
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/genproto/googleapis/type/latlng"
)

func TestJSONToFirestoreValue_ComplexDocument(t *testing.T) {
	// This is the exact JSON format provided by the user
	jsonData := `{
		"name": {
			"stringValue": "Acme Corporation"
		},
		"establishmentDate": {
			"timestampValue": "2000-01-15T10:30:00Z"
		},
		"location": {
			"geoPointValue": {
				"latitude": 34.052235,
				"longitude": -118.243683
			}
		},
		"active": {
			"booleanValue": true
		},
		"employeeCount": {
			"integerValue": "1500"
		},
		"annualRevenue": {
			"doubleValue": 1234567.89
		},
		"website": {
			"stringValue": "https://www.acmecorp.com"
		},
		"contactInfo": {
			"mapValue": {
				"fields": {
					"email": {
						"stringValue": "info@acmecorp.com"
					},
					"phone": {
						"stringValue": "+1-555-123-4567"
					},
					"address": {
						"mapValue": {
							"fields": {
								"street": {
									"stringValue": "123 Business Blvd"
								},
								"city": {
									"stringValue": "Los Angeles"
								},
								"state": {
									"stringValue": "CA"
								},
								"zipCode": {
									"stringValue": "90012"
								}
							}
						}
					}
				}
			}
		},
		"products": {
			"arrayValue": {
				"values": [
					{
						"stringValue": "Product A"
					},
					{
						"stringValue": "Product B"
					},
					{
						"mapValue": {
							"fields": {
								"productName": {
									"stringValue": "Product C Deluxe"
								},
								"version": {
									"integerValue": "2"
								},
								"features": {
									"arrayValue": {
										"values": [
											{
												"stringValue": "Feature X"
											},
											{
												"stringValue": "Feature Y"
											}
										]
									}
								}
							}
						}
					}
				]
			}
		},
		"notes": {
			"nullValue": null
		},
		"lastUpdated": {
			"timestampValue": "2025-07-30T11:47:59.000Z"
		},
		"binaryData": {
			"bytesValue": "SGVsbG8gV29ybGQh"
		}
	}`

	// Parse JSON
	var data interface{}
	err := json.Unmarshal([]byte(jsonData), &data)
	require.NoError(t, err)

	// Convert to Firestore format
	result, err := JSONToFirestoreValue(data, nil)
	require.NoError(t, err)

	// Verify the result is a map
	resultMap, ok := result.(map[string]interface{})
	require.True(t, ok, "Result should be a map")

	// Verify string values
	assert.Equal(t, "Acme Corporation", resultMap["name"])
	assert.Equal(t, "https://www.acmecorp.com", resultMap["website"])

	// Verify timestamp
	establishmentDate, ok := resultMap["establishmentDate"].(time.Time)
	require.True(t, ok, "establishmentDate should be time.Time")
	expectedDate, _ := time.Parse(time.RFC3339, "2000-01-15T10:30:00Z")
	assert.Equal(t, expectedDate, establishmentDate)

	// Verify geopoint
	location, ok := resultMap["location"].(*latlng.LatLng)
	require.True(t, ok, "location should be *latlng.LatLng")
	assert.Equal(t, 34.052235, location.Latitude)
	assert.Equal(t, -118.243683, location.Longitude)

	// Verify boolean
	assert.Equal(t, true, resultMap["active"])

	// Verify integer (should be int64)
	employeeCount, ok := resultMap["employeeCount"].(int64)
	require.True(t, ok, "employeeCount should be int64")
	assert.Equal(t, int64(1500), employeeCount)

	// Verify double
	annualRevenue, ok := resultMap["annualRevenue"].(float64)
	require.True(t, ok, "annualRevenue should be float64")
	assert.Equal(t, 1234567.89, annualRevenue)

	// Verify nested map
	contactInfo, ok := resultMap["contactInfo"].(map[string]interface{})
	require.True(t, ok, "contactInfo should be a map")
	assert.Equal(t, "info@acmecorp.com", contactInfo["email"])
	assert.Equal(t, "+1-555-123-4567", contactInfo["phone"])

	// Verify nested nested map
	address, ok := contactInfo["address"].(map[string]interface{})
	require.True(t, ok, "address should be a map")
	assert.Equal(t, "123 Business Blvd", address["street"])
	assert.Equal(t, "Los Angeles", address["city"])
	assert.Equal(t, "CA", address["state"])
	assert.Equal(t, "90012", address["zipCode"])

	// Verify array
	products, ok := resultMap["products"].([]interface{})
	require.True(t, ok, "products should be an array")
	assert.Len(t, products, 3)
	assert.Equal(t, "Product A", products[0])
	assert.Equal(t, "Product B", products[1])

	// Verify complex item in array
	product3, ok := products[2].(map[string]interface{})
	require.True(t, ok, "products[2] should be a map")
	assert.Equal(t, "Product C Deluxe", product3["productName"])
	version, ok := product3["version"].(int64)
	require.True(t, ok, "version should be int64")
	assert.Equal(t, int64(2), version)

	features, ok := product3["features"].([]interface{})
	require.True(t, ok, "features should be an array")
	assert.Len(t, features, 2)
	assert.Equal(t, "Feature X", features[0])
	assert.Equal(t, "Feature Y", features[1])

	// Verify null value
	assert.Nil(t, resultMap["notes"])

	// Verify bytes
	binaryData, ok := resultMap["binaryData"].([]byte)
	require.True(t, ok, "binaryData should be []byte")
	expectedBytes, _ := base64.StdEncoding.DecodeString("SGVsbG8gV29ybGQh")
	assert.Equal(t, expectedBytes, binaryData)
}

func TestJSONToFirestoreValue_IntegerFromString(t *testing.T) {
	// Test that integerValue as string gets converted to int64
	data := map[string]interface{}{
		"integerValue": "1500",
	}

	result, err := JSONToFirestoreValue(data, nil)
	require.NoError(t, err)

	intVal, ok := result.(int64)
	require.True(t, ok, "Result should be int64")
	assert.Equal(t, int64(1500), intVal)
}

func TestFirestoreValueToJSON_RoundTrip(t *testing.T) {
	// Test round-trip conversion
	original := map[string]interface{}{
		"name":   "Test",
		"count":  int64(42),
		"price":  19.99,
		"active": true,
		"tags":   []interface{}{"tag1", "tag2"},
		"metadata": map[string]interface{}{
			"created": time.Now(),
		},
		"nullField": nil,
	}

	// Convert to JSON representation
	jsonRepresentation := FirestoreValueToJSON(original)

	// Verify types are simplified
	jsonMap, ok := jsonRepresentation.(map[string]interface{})
	require.True(t, ok)

	// Time should be converted to string
	metadata, ok := jsonMap["metadata"].(map[string]interface{})
	require.True(t, ok)
	_, ok = metadata["created"].(string)
	assert.True(t, ok, "created should be a string")
}

func TestJSONToFirestoreValue_InvalidFormats(t *testing.T) {
	tests := []struct {
		name    string
		input   interface{}
		wantErr bool
		errMsg  string
	}{
		{
			name: "invalid integer value",
			input: map[string]interface{}{
				"integerValue": "not-a-number",
			},
			wantErr: true,
			errMsg:  "invalid integer value",
		},
		{
			name: "invalid timestamp",
			input: map[string]interface{}{
				"timestampValue": "not-a-timestamp",
			},
			wantErr: true,
			errMsg:  "invalid timestamp format",
		},
		{
			name: "invalid geopoint - missing latitude",
			input: map[string]interface{}{
				"geoPointValue": map[string]interface{}{
					"longitude": -118.243683,
				},
			},
			wantErr: true,
			errMsg:  "invalid geopoint value format",
		},
		{
			name: "invalid array format",
			input: map[string]interface{}{
				"arrayValue": "not-an-array",
			},
			wantErr: true,
			errMsg:  "invalid array value format",
		},
		{
			name: "invalid map format",
			input: map[string]interface{}{
				"mapValue": "not-a-map",
			},
			wantErr: true,
			errMsg:  "invalid map value format",
		},
		{
			name: "invalid bytes - not base64",
			input: map[string]interface{}{
				"bytesValue": "!!!not-base64!!!",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := JSONToFirestoreValue(tt.input, nil)
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
