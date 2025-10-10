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

// To run these tests, set the following environment variables:
// HEALTHCARE_PROJECT: Google Cloud project ID for healthcare resources.
// HEALTHCARE_REGION: Google Cloud region for healthcare resources.

package healthcare

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/googleapis/genai-toolbox/internal/sources"
	"github.com/googleapis/genai-toolbox/internal/testutils"
	"github.com/googleapis/genai-toolbox/tests"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/healthcare/v1"
	"google.golang.org/api/option"
)

var (
	healthcareSourceKind    = "healthcare"
	getDatasetToolKind      = "get-healthcare-dataset"
	listFHIRStoresToolKind  = "list-fhir-stores"
	listDICOMStoresToolKind = "list-dicom-stores"
	healthcareProject       = os.Getenv("HEALTHCARE_PROJECT")
	healthcareRegion        = os.Getenv("HEALTHCARE_REGION")
	healthcareDataset       = os.Getenv("HEALTHCARE_DATASET")
)

func verifyHealthcareVars(t *testing.T) {
	switch "" {
	case healthcareProject:
		t.Fatal("'HEALTHCARE_PROJECT' not set")
	case healthcareRegion:
		t.Fatal("'HEALTHCARE_REGION' not set")
	case healthcareDataset:
		t.Fatal("'HEALTHCARE_DATASET' not set")
	}
}

func TestHealthcareToolEndpoints(t *testing.T) {
	verifyHealthcareVars(t)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	healthcareService, err := newHealthcareService(ctx)
	if err != nil {
		t.Fatalf("failed to create healthcare service: %v", err)
	}

	fhirStoreID := "fhir-store-" + uuid.New().String()
	dicomStoreID := "dicom-store-" + uuid.New().String()

	teardown := setupHealthcareResources(t, ctx, healthcareService, healthcareDataset, fhirStoreID, dicomStoreID)
	defer teardown(t)

	sourceConfig := map[string]any{
		"kind":    healthcareSourceKind,
		"project": healthcareProject,
		"region":  healthcareRegion,
		"dataset": healthcareDataset,
	}

	toolsFile := getToolsConfig(sourceConfig)
	toolsFile = addClientAuthSourceConfig(t, toolsFile, healthcareDataset)

	var args []string
	cmd, cleanup, err := tests.StartCmd(ctx, toolsFile, args...)
	if err != nil {
		t.Fatalf("command initialization returned an error: %s", err)
	}
	defer cleanup()

	waitCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	out, err := testutils.WaitForString(waitCtx, regexp.MustCompile(`Server ready to serve`), cmd.Out)
	if err != nil {
		t.Logf("toolbox command logs: %s", out)
		t.Fatalf("toolbox didn't start successfully: %s", err)
	}

	datasetWant := fmt.Sprintf(`"name":"projects/%s/locations/%s/datasets/%s"`, healthcareProject, healthcareRegion, healthcareDataset)
	fhirStoreWant := fmt.Sprintf(`"name":"projects/%s/locations/%s/datasets/%s/fhirStores/%s"`, healthcareProject, healthcareRegion, healthcareDataset, fhirStoreID)
	dicomStoreWant := fmt.Sprintf(`"name":"projects/%s/locations/%s/datasets/%s/dicomStores/%s"`, healthcareProject, healthcareRegion, healthcareDataset, dicomStoreID)

	runGetDatasetToolInvokeTest(t, datasetWant)
	runListFHIRStoresToolInvokeTest(t, fhirStoreWant)
	runListDICOMStoresToolInvokeTest(t, dicomStoreWant)
}

func TestHealthcareToolWithStoreRestriction(t *testing.T) {
	verifyHealthcareVars(t)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	healthcareService, err := newHealthcareService(ctx)
	if err != nil {
		t.Fatalf("failed to create healthcare service: %v", err)
	}

	// Create stores
	allowedFHIRStoreID := "fhir-store-allowed-" + uuid.New().String()
	allowedDICOMStoreID := "dicom-store-allowed-" + uuid.New().String()
	disallowedFHIRStoreID := "fhir-store-disallowed-" + uuid.New().String()
	disallowedDICOMStoreID := "dicom-store-disallowed-" + uuid.New().String()

	teardownAllowedStores := setupHealthcareResources(t, ctx, healthcareService, healthcareDataset, allowedFHIRStoreID, allowedDICOMStoreID)
	defer teardownAllowedStores(t)
	teardownDisallowedStores := setupHealthcareResources(t, ctx, healthcareService, healthcareDataset, disallowedFHIRStoreID, disallowedDICOMStoreID)
	defer teardownDisallowedStores(t)

	// Configure source with dataset restriction.
	sourceConfig := map[string]any{
		"kind":    healthcareSourceKind,
		"project": healthcareProject,
		"region":  healthcareRegion,
		"dataset": healthcareDataset,
		"allowedFhirStores": []string{
			allowedFHIRStoreID,
		},
		"allowedDicomStores": []string{
			allowedDICOMStoreID,
		},
	}

	// Configure tool
	toolsConfig := map[string]any{
		"list-fhir-stores-restricted": map[string]any{
			"kind":        "list-fhir-stores",
			"source":      "my-instance",
			"description": "Tool to list fhir stores",
		},
		"list-dicom-stores-restricted": map[string]any{
			"kind":        "list-dicom-stores",
			"source":      "my-instance",
			"description": "Tool to list dicom stores",
		},
	}

	// Create config file
	config := map[string]any{
		"sources": map[string]any{
			"my-instance": sourceConfig,
		},
		"tools": toolsConfig,
	}

	// Start server
	cmd, cleanup, err := tests.StartCmd(ctx, config)
	if err != nil {
		t.Fatalf("command initialization returned an error: %s", err)
	}
	defer cleanup()

	waitCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	out, err := testutils.WaitForString(waitCtx, regexp.MustCompile(`Server ready to serve`), cmd.Out)
	if err != nil {
		t.Logf("toolbox command logs: \n%s", out)
		t.Fatalf("toolbox didn't start successfully: %s", err)
	}

	// Run tests
	runListFHIRStoresWithRestriction(t, allowedFHIRStoreID, disallowedFHIRStoreID)
	runListDICOMStoresWithRestriction(t, allowedDICOMStoreID, disallowedDICOMStoreID)
}

func newHealthcareService(ctx context.Context) (*healthcare.Service, error) {
	creds, err := google.FindDefaultCredentials(ctx, healthcare.CloudHealthcareScope)
	if err != nil {
		return nil, fmt.Errorf("failed to find default credentials: %w", err)
	}

	healthcareService, err := healthcare.NewService(ctx, option.WithCredentials(creds))
	if err != nil {
		return nil, fmt.Errorf("failed to create healthcare service: %w", err)
	}
	return healthcareService, nil
}

func setupHealthcareResources(t *testing.T, ctx context.Context, service *healthcare.Service, datasetID, fhirStoreID, dicomStoreID string) func(*testing.T) {
	datasetName := fmt.Sprintf("projects/%s/locations/%s/datasets/%s", healthcareProject, healthcareRegion, datasetID)
	var err error

	// Create FHIR store
	fhirStore := &healthcare.FhirStore{Version: "R4"}
	if fhirStore, err = service.Projects.Locations.Datasets.FhirStores.Create(datasetName, fhirStore).FhirStoreId(fhirStoreID).Do(); err != nil {
		t.Fatalf("failed to create fhir store: %v", err)
	}

	// Create DICOM store
	dicomStore := &healthcare.DicomStore{}
	if dicomStore, err = service.Projects.Locations.Datasets.DicomStores.Create(datasetName, dicomStore).DicomStoreId(dicomStoreID).Do(); err != nil {
		t.Fatalf("failed to create dicom store: %v", err)
	}

	return func(t *testing.T) {
		if _, err := service.Projects.Locations.Datasets.FhirStores.Delete(fhirStore.Name).Do(); err != nil {
			t.Logf("failed to delete fhir store: %v", err)
		}
		if _, err := service.Projects.Locations.Datasets.DicomStores.Delete(dicomStore.Name).Do(); err != nil {
			t.Logf("failed to delete dicom store: %v", err)
		}
	}
}

func getToolsConfig(sourceConfig map[string]any) map[string]any {
	config := map[string]any{
		"sources": map[string]any{
			"my-instance": sourceConfig,
		},
		"tools": map[string]any{
			"my-get-dataset-tool": map[string]any{
				"kind":        getDatasetToolKind,
				"source":      "my-instance",
				"description": "Tool to get a healthcare dataset",
			},
			"my-list-fhir-stores-tool": map[string]any{
				"kind":        listFHIRStoresToolKind,
				"source":      "my-instance",
				"description": "Tool to list FHIR stores",
			},
			"my-list-dicom-stores-tool": map[string]any{
				"kind":        listDICOMStoresToolKind,
				"source":      "my-instance",
				"description": "Tool to list DICOM stores",
			},
			"my-auth-get-dataset-tool": map[string]any{
				"kind":        getDatasetToolKind,
				"source":      "my-instance",
				"description": "Tool to get a healthcare dataset with auth",
				"authRequired": []string{
					"my-google-auth",
				},
			},
			"my-auth-list-fhir-stores-tool": map[string]any{
				"kind":        listFHIRStoresToolKind,
				"source":      "my-instance",
				"description": "Tool to list FHIR stores with auth",
				"authRequired": []string{
					"my-google-auth",
				},
			},
			"my-auth-list-dicom-stores-tool": map[string]any{
				"kind":        listDICOMStoresToolKind,
				"source":      "my-instance",
				"description": "Tool to list DICOM stores with auth",
				"authRequired": []string{
					"my-google-auth",
				},
			},
		},
		"authServices": map[string]any{
			"my-google-auth": map[string]any{
				"kind":     "google",
				"clientId": tests.ClientId,
			},
		},
	}
	return config
}

func addClientAuthSourceConfig(t *testing.T, config map[string]any, datasetID string) map[string]any {
	sources, ok := config["sources"].(map[string]any)
	if !ok {
		t.Fatalf("unable to get sources from config")
	}
	sources["my-client-auth-source"] = map[string]any{
		"kind":           healthcareSourceKind,
		"project":        healthcareProject,
		"region":         healthcareRegion,
		"dataset":        datasetID,
		"useClientOAuth": true,
	}
	config["sources"] = sources
	return config
}

func runGetDatasetToolInvokeTest(t *testing.T, want string) {
	idToken, err := tests.GetGoogleIdToken(tests.ClientId)
	if err != nil {
		t.Fatalf("error getting Google ID token: %s", err)
	}

	accessToken, err := sources.GetIAMAccessToken(t.Context())
	if err != nil {
		t.Fatalf("error getting access token from ADC: %s", err)
	}
	accessToken = "Bearer " + accessToken

	invokeTcs := []struct {
		name          string
		api           string
		requestHeader map[string]string
		requestBody   io.Reader
		want          string
		isErr         bool
	}{
		{
			name:          "invoke my-get-dataset-tool",
			api:           "http://127.0.0.1:5000/api/tool/my-get-dataset-tool/invoke",
			requestHeader: map[string]string{},
			requestBody:   bytes.NewBuffer([]byte(`{}`)),
			want:          want,
			isErr:         false,
		},
		{
			name:          "invoke my-auth-get-dataset-tool with auth",
			api:           "http://127.0.0.1:5000/api/tool/my-auth-get-dataset-tool/invoke",
			requestHeader: map[string]string{"my-google-auth_token": idToken},
			requestBody:   bytes.NewBuffer([]byte(`{}`)),
			want:          want,
			isErr:         false,
		},
		{
			name:          "invoke my-auth-get-dataset-tool with client auth",
			api:           "http://127.0.0.1:5000/api/tool/my-get-dataset-tool/invoke",
			requestHeader: map[string]string{"Authorization": accessToken},
			requestBody:   bytes.NewBuffer([]byte(`{}`)),
			want:          want,
			isErr:         false,
		},
		{
			name:          "invoke my-auth-get-dataset-tool without auth token",
			api:           "http://127.0.0.1:5000/api/tool/my-auth-get-dataset-tool/invoke",
			requestHeader: map[string]string{},
			requestBody:   bytes.NewBuffer([]byte(`{}`)),
			isErr:         true,
		},
		{
			name:          "invoke my-auth-get-dataset-tool with invalid auth token",
			api:           "http://127.0.0.1:5000/api/tool/my-auth-get-dataset-tool/invoke",
			requestHeader: map[string]string{"Authorization": "invalid-token"},
			requestBody:   bytes.NewBuffer([]byte(`{}`)),
			isErr:         true,
		},
	}
	for _, tc := range invokeTcs {
		t.Run(tc.name, func(t *testing.T) {
			runTest(t, tc.api, tc.requestHeader, tc.requestBody, tc.want, tc.isErr)
		})
	}
}

func runListFHIRStoresToolInvokeTest(t *testing.T, want string) {
	idToken, err := tests.GetGoogleIdToken(tests.ClientId)
	if err != nil {
		t.Fatalf("error getting Google ID token: %s", err)
	}

	accessToken, err := sources.GetIAMAccessToken(t.Context())
	if err != nil {
		t.Fatalf("error getting access token from ADC: %s", err)
	}
	accessToken = "Bearer " + accessToken

	invokeTcs := []struct {
		name          string
		api           string
		requestHeader map[string]string
		requestBody   io.Reader
		want          string
		isErr         bool
	}{
		{
			name:          "invoke my-list-fhir-stores-tool",
			api:           "http://127.0.0.1:5000/api/tool/my-list-fhir-stores-tool/invoke",
			requestHeader: map[string]string{},
			requestBody:   bytes.NewBuffer([]byte(`{}`)),
			want:          want,
			isErr:         false,
		},
		{
			name:          "invoke my-auth-list-fhir-stores-tool with auth",
			api:           "http://127.0.0.1:5000/api/tool/my-auth-list-fhir-stores-tool/invoke",
			requestHeader: map[string]string{"my-google-auth_token": idToken},
			requestBody:   bytes.NewBuffer([]byte(`{}`)),
			want:          want,
			isErr:         false,
		},
		{
			name:          "invoke my-auth-list-fhir-stores-tool with client auth",
			api:           "http://127.0.0.1:5000/api/tool/my-list-fhir-stores-tool/invoke",
			requestHeader: map[string]string{"Authorization": accessToken},
			requestBody:   bytes.NewBuffer([]byte(`{}`)),
			want:          want,
			isErr:         false,
		},
		{
			name:          "invoke my-auth-list-fhir-stores-tool without auth token",
			api:           "http://127.0.0.1:5000/api/tool/my-auth-list-fhir-stores-tool/invoke",
			requestHeader: map[string]string{},
			requestBody:   bytes.NewBuffer([]byte(`{}`)),
			isErr:         true,
		},
		{
			name:          "invoke my-auth-list-fhir-stores-tool with invalid auth token",
			api:           "http://127.0.0.1:5000/api/tool/my-auth-list-fhir-stores-tool/invoke",
			requestHeader: map[string]string{"Authorization": "invalid-token"},
			requestBody:   bytes.NewBuffer([]byte(`{}`)),
			isErr:         true,
		},
	}
	for _, tc := range invokeTcs {
		t.Run(tc.name, func(t *testing.T) {
			runTest(t, tc.api, tc.requestHeader, tc.requestBody, tc.want, tc.isErr)
		})
	}
}

func runListDICOMStoresToolInvokeTest(t *testing.T, want string) {
	idToken, err := tests.GetGoogleIdToken(tests.ClientId)
	if err != nil {
		t.Fatalf("error getting Google ID token: %s", err)
	}

	accessToken, err := sources.GetIAMAccessToken(t.Context())
	if err != nil {
		t.Fatalf("error getting access token from ADC: %s", err)
	}
	accessToken = "Bearer " + accessToken

	invokeTcs := []struct {
		name          string
		api           string
		requestHeader map[string]string
		requestBody   io.Reader
		want          string
		isErr         bool
	}{
		{
			name:          "invoke my-list-dicom-stores-tool",
			api:           "http://127.0.0.1:5000/api/tool/my-list-dicom-stores-tool/invoke",
			requestHeader: map[string]string{},
			requestBody:   bytes.NewBuffer([]byte(`{}`)),
			want:          want,
			isErr:         false,
		},
		{
			name:          "invoke my-auth-list-dicom-stores-tool with auth",
			api:           "http://127.0.0.1:5000/api/tool/my-auth-list-dicom-stores-tool/invoke",
			requestHeader: map[string]string{"my-google-auth_token": idToken},
			requestBody:   bytes.NewBuffer([]byte(`{}`)),
			want:          want,
			isErr:         false,
		},
		{
			name:          "invoke my-auth-list-dicom-stores-tool with client auth",
			api:           "http://127.0.0.1:5000/api/tool/my-list-dicom-stores-tool/invoke",
			requestHeader: map[string]string{"Authorization": accessToken},
			requestBody:   bytes.NewBuffer([]byte(`{}`)),
			want:          want,
			isErr:         false,
		},
		{
			name:          "invoke my-auth-list-dicom-stores-tool without auth token",
			api:           "http://127.0.0.1:5000/api/tool/my-auth-list-dicom-stores-tool/invoke",
			requestHeader: map[string]string{},
			requestBody:   bytes.NewBuffer([]byte(`{}`)),
			isErr:         true,
		},
		{
			name:          "invoke my-auth-list-dicom-stores-tool with invalid auth token",
			api:           "http://127.0.0.1:5000/api/tool/my-auth-list-dicom-stores-tool/invoke",
			requestHeader: map[string]string{"Authorization": "invalid-token"},
			requestBody:   bytes.NewBuffer([]byte(`{}`)),
			isErr:         true,
		},
	}
	for _, tc := range invokeTcs {
		t.Run(tc.name, func(t *testing.T) {
			runTest(t, tc.api, tc.requestHeader, tc.requestBody, tc.want, tc.isErr)
		})
	}
}

func runTest(t *testing.T, api string, requestHeader map[string]string, requestBody io.Reader, want string, isErr bool) {
	req, err := http.NewRequest(http.MethodPost, api, requestBody)
	if err != nil {
		t.Fatalf("unable to create request: %s", err)
	}
	req.Header.Add("Content-type", "application/json")
	for k, v := range requestHeader {
		req.Header.Add(k, v)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("unable to send request: %s", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		if isErr {
			return
		}
		bodyBytes, _ := io.ReadAll(resp.Body)
		t.Fatalf("response status code is not 200, got %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var body map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&body)
	if err != nil {
		t.Fatalf("error parsing response body")
	}

	got, ok := body["result"].(string)
	if !ok {
		t.Fatalf("unable to find result in response body")
	}

	if !strings.Contains(got, want) {
		t.Fatalf("expected %q to contain %q, but it did not", got, want)
	}
}

func runListFHIRStoresWithRestriction(t *testing.T, allowedFHIRStore, disallowedFHIRStore string) {
	api := "http://127.0.0.1:5000/api/tool/list-fhir-stores-restricted/invoke"
	req, err := http.NewRequest(http.MethodPost, api, bytes.NewBuffer([]byte(`{}`)))
	if err != nil {
		t.Fatalf("unable to create request: %s", err)
	}
	req.Header.Add("Content-type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("unable to send request: %s", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		t.Fatalf("response status code is not 200, got %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var body map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&body)
	if err != nil {
		t.Fatalf("error parsing response body")
	}

	got, ok := body["result"].(string)
	if !ok {
		t.Fatalf("unable to find result in response body")
	}

	if !strings.Contains(got, allowedFHIRStore) {
		t.Fatalf("expected %q to contain %q, but it did not", got, allowedFHIRStore)
	}
	if strings.Contains(got, disallowedFHIRStore) {
		t.Fatalf("expected %q to NOT contain %q, but it did", got, disallowedFHIRStore)
	}
}

func runListDICOMStoresWithRestriction(t *testing.T, allowedDICOMStore, disallowedDICOMStore string) {
	api := "http://127.0.0.1:5000/api/tool/list-dicom-stores-restricted/invoke"
	req, err := http.NewRequest(http.MethodPost, api, bytes.NewBuffer([]byte(`{}`)))
	if err != nil {
		t.Fatalf("unable to create request: %s", err)
	}
	req.Header.Add("Content-type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("unable to send request: %s", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		t.Fatalf("response status code is not 200, got %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var body map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&body)
	if err != nil {
		t.Fatalf("error parsing response body")
	}

	got, ok := body["result"].(string)
	if !ok {
		t.Fatalf("unable to find result in response body")
	}

	if !strings.Contains(got, allowedDICOMStore) {
		t.Fatalf("expected %q to contain %q, but it did not", got, allowedDICOMStore)
	}
	if strings.Contains(got, disallowedDICOMStore) {
		t.Fatalf("expected %q to NOT contain %q, but it did", got, disallowedDICOMStore)
	}
}
