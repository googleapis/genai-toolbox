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

package common

import (
	"fmt"

	"github.com/googleapis/genai-toolbox/internal/tools"
)

// StoreKey is the key used to identify FHIR/DICOM store IDs in tool parameters.
const StoreKey = "storeID"

// ValidateAndFetchStoreID validates the provided storeID against the allowedStores.
// If only one store is allowed, it returns that storeID.
// If multiple stores are allowed, it checks if the storeID parameter is in the allowed list.
func ValidateAndFetchStoreID(params tools.ParamValues, allowedStores map[string]struct{}) (string, error) {
	if len(allowedStores) == 1 {
		for k := range allowedStores {
			return k, nil
		}
	}
	mapParams := params.AsMap()
	storeID, ok := mapParams[StoreKey].(string)
	if !ok {
		return "", fmt.Errorf("invalid or missing '%s' parameter; expected a string", StoreKey)
	}
	if len(allowedStores) > 0 {
		if _, ok := allowedStores[storeID]; !ok {
			return "", fmt.Errorf("store ID '%s' is not in the list of allowed stores", storeID)
		}
	}
	return storeID, nil
}
