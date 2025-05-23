// Copyright 2024 Google LLC
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

package prebuiltconfigs

import (
	"testing"
)

func TestGetPrebuiltToolYAMLs(t *testing.T) {
	test_name := "test load prebuilt configs"
	expectedKeys := []string{
		"alloydb",
		"cloudsqlpg",
		"postgres",
		"spanner",
	}
	t.Run(test_name, func(t *testing.T) {
		configs, err := GetPrebuiltToolYAMLs()
		if err != nil {
			t.Fatalf("unexpected error: %s", err)
		}
		foundExpectedKeys := make(map[string]bool)

		if len(expectedKeys) != len(configs) {
			t.Fatalf("Failed to load all prebuilt tools.")
		}

		for _, expectedKey := range expectedKeys {
			_, ok := configs[expectedKey]
			if !ok {
				t.Fatalf("Prebuilt tools for '%s' was NOT FOUND in the loaded map.", expectedKey)
			} else {
				foundExpectedKeys[expectedKey] = true // Mark as found
			}
		}

	})
}
