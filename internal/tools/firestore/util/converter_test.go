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
	"bytes"
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"
	"time"

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
	if err != nil {
		t.Fatalf("Failed to unmarshal JSON: %v", err)
	}

	// Convert to Firestore format
	result, err := JSONToFirestoreValue(data, nil)
	if err != nil {
		t.Fatalf("Failed to convert JSON to Firestore value: %v", err)
	}

	// Verify the result is a map
	resultMap, ok := result.(map[string]interface{})
	if !ok {
		t.Fatalf("Result should be a map, got %T", result)
	}

	// Verify string values
	if resultMap["name"] != "Acme Corporation" {
		t.Errorf("Expected name 'Acme Corporation', got %v", resultMap["name"])
	}
	if resultMap["website"] != "https://www.acmecorp.com" {
		t.Errorf("Expected website 'https://www.acmecorp.com', got %v", resultMap["website"])
	}

	// Verify timestamp
	establishmentDate, ok := resultMap["establishmentDate"].(time.Time)
	if !ok {
		t.Fatalf("establishmentDate should be time.Time, got %T", resultMap["establishmentDate"])
	}
	expectedDate, _ := time.Parse(time.RFC3339, "2000-01-15T10:30:00Z")
	if !establishmentDate.Equal(expectedDate) {
		t.Errorf("Expected date %v, got %v", expectedDate, establishmentDate)
	}

	// Verify geopoint
	location, ok := resultMap["location"].(*latlng.LatLng)
	if !ok {
		t.Fatalf("location should be *latlng.LatLng, got %T", resultMap["location"])
	}
	if location.Latitude != 34.052235 {
		t.Errorf("Expected latitude 34.052235, got %v", location.Latitude)
	}
	if location.Longitude != -118.243683 {
		t.Errorf("Expected longitude -118.243683, got %v", location.Longitude)
	}

	// Verify boolean
	if resultMap["active"] != true {
		t.Errorf("Expected active true, got %v", resultMap["active"])
	}

	// Verify integer (should be int64)
	employeeCount, ok := resultMap["employeeCount"].(int64)
	if !ok {
		t.Fatalf("employeeCount should be int64, got %T", resultMap["employeeCount"])
	}
	if employeeCount != int64(1500) {
		t.Errorf("Expected employeeCount 1500, got %v", employeeCount)
	}

	// Verify double
	annualRevenue, ok := resultMap["annualRevenue"].(float64)
	if !ok {
		t.Fatalf("annualRevenue should be float64, got %T", resultMap["annualRevenue"])
	}
	if annualRevenue != 1234567.89 {
		t.Errorf("Expected annualRevenue 1234567.89, got %v", annualRevenue)
	}

	// Verify nested map
	contactInfo, ok := resultMap["contactInfo"].(map[string]interface{})
	if !ok {
		t.Fatalf("contactInfo should be a map, got %T", resultMap["contactInfo"])
	}
	if contactInfo["email"] != "info@acmecorp.com" {
		t.Errorf("Expected email 'info@acmecorp.com', got %v", contactInfo["email"])
	}
	if contactInfo["phone"] != "+1-555-123-4567" {
		t.Errorf("Expected phone '+1-555-123-4567', got %v", contactInfo["phone"])
	}

	// Verify nested nested map
	address, ok := contactInfo["address"].(map[string]interface{})
	if !ok {
		t.Fatalf("address should be a map, got %T", contactInfo["address"])
	}
	if address["street"] != "123 Business Blvd" {
		t.Errorf("Expected street '123 Business Blvd', got %v", address["street"])
	}
	if address["city"] != "Los Angeles" {
		t.Errorf("Expected city 'Los Angeles', got %v", address["city"])
	}
	if address["state"] != "CA" {
		t.Errorf("Expected state 'CA', got %v", address["state"])
	}
	if address["zipCode"] != "90012" {
		t.Errorf("Expected zipCode '90012', got %v", address["zipCode"])
	}

	// Verify array
	products, ok := resultMap["products"].([]interface{})
	if !ok {
		t.Fatalf("products should be an array, got %T", resultMap["products"])
	}
	if len(products) != 3 {
		t.Errorf("Expected 3 products, got %d", len(products))
	}
	if products[0] != "Product A" {
		t.Errorf("Expected products[0] 'Product A', got %v", products[0])
	}
	if products[1] != "Product B" {
		t.Errorf("Expected products[1] 'Product B', got %v", products[1])
	}

	// Verify complex item in array
	product3, ok := products[2].(map[string]interface{})
	if !ok {
		t.Fatalf("products[2] should be a map, got %T", products[2])
	}
	if product3["productName"] != "Product C Deluxe" {
		t.Errorf("Expected productName 'Product C Deluxe', got %v", product3["productName"])
	}
	version, ok := product3["version"].(int64)
	if !ok {
		t.Fatalf("version should be int64, got %T", product3["version"])
	}
	if version != int64(2) {
		t.Errorf("Expected version 2, got %v", version)
	}

	features, ok := product3["features"].([]interface{})
	if !ok {
		t.Fatalf("features should be an array, got %T", product3["features"])
	}
	if len(features) != 2 {
		t.Errorf("Expected 2 features, got %d", len(features))
	}
	if features[0] != "Feature X" {
		t.Errorf("Expected features[0] 'Feature X', got %v", features[0])
	}
	if features[1] != "Feature Y" {
		t.Errorf("Expected features[1] 'Feature Y', got %v", features[1])
	}

	// Verify null value
	if resultMap["notes"] != nil {
		t.Errorf("Expected notes to be nil, got %v", resultMap["notes"])
	}

	// Verify bytes
	binaryData, ok := resultMap["binaryData"].([]byte)
	if !ok {
		t.Fatalf("binaryData should be []byte, got %T", resultMap["binaryData"])
	}
	expectedBytes, _ := base64.StdEncoding.DecodeString("SGVsbG8gV29ybGQh")
	if !bytes.Equal(binaryData, expectedBytes) {
		t.Errorf("Expected bytes %v, got %v", expectedBytes, binaryData)
	}
}

func TestJSONToFirestoreValue_IntegerFromString(t *testing.T) {
	// Test that integerValue as string gets converted to int64
	data := map[string]interface{}{
		"integerValue": "1500",
	}

	result, err := JSONToFirestoreValue(data, nil)
	if err != nil {
		t.Fatalf("Failed to convert: %v", err)
	}

	intVal, ok := result.(int64)
	if !ok {
		t.Fatalf("Result should be int64, got %T", result)
	}
	if intVal != int64(1500) {
		t.Errorf("Expected 1500, got %v", intVal)
	}
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
	if !ok {
		t.Fatalf("Expected map, got %T", jsonRepresentation)
	}

	// Time should be converted to string
	metadata, ok := jsonMap["metadata"].(map[string]interface{})
	if !ok {
		t.Fatalf("metadata should be a map, got %T", jsonMap["metadata"])
	}
	_, ok = metadata["created"].(string)
	if !ok {
		t.Errorf("created should be a string, got %T", metadata["created"])
	}
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
				if err == nil {
					t.Errorf("Expected error but got none")
				} else if tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("Expected error containing '%s', got '%v'", tt.errMsg, err)
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
			}
		})
	}
}
