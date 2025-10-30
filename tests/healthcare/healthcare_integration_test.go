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
	healthcareSourceKind                  = "healthcare"
	getDatasetToolKind                    = "get-healthcare-dataset"
	listFHIRStoresToolKind                = "list-fhir-stores"
	listDICOMStoresToolKind               = "list-dicom-stores"
	getFHIRStoreToolKind                  = "get-fhir-store"
	getFHIRStoreMetricsToolKind           = "get-fhir-store-metrics"
	getFHIRResourceToolKind               = "get-fhir-resource"
	fhirPatientSearchToolKind             = "fhir-patient-search"
	fhirPatientEverythingToolKind         = "fhir-patient-everything"
	fhirFetchPageToolKind                 = "fhir-fetch-page"
	getDICOMStoreToolKind                 = "get-dicom-store"
	getDICOMStoreMetricsToolKind          = "get-dicom-store-metrics"
	searchDICOMStudiesToolKind            = "search-dicom-studies"
	searchDICOMSeriesToolKind             = "search-dicom-series"
	searchDICOMInstancesToolKind          = "search-dicom-instances"
	retrieveRenderedDICOMInstanceToolKind = "retrieve-rendered-dicom-instance"
	healthcareProject                     = os.Getenv("HEALTHCARE_PROJECT")
	healthcareRegion                      = os.Getenv("HEALTHCARE_REGION")
	healthcareDataset                     = os.Getenv("HEALTHCARE_DATASET")
	healthcarePrepopulatedDICOMStore      = os.Getenv("HEALTHCARE_PREPOPULATED_DICOM_STORE")
)

type DICOMInstance struct {
	study, series, instance string
}

var (
	singleFrameDICOMInstance = DICOMInstance{
		study:    "1.2.840.113619.2.176.3596.3364818.7819.1259708454.105",
		series:   "1.2.840.113619.2.176.3596.3364818.7819.1259708454.108",
		instance: "1.2.840.113619.2.176.3596.3364818.7271.1259708501.876",
	}
	multiFrameDICOMInstance = DICOMInstance{
		study:    "1.2.826.0.1.3680043.9.5704.649259287",
		series:   "1.2.826.0.1.3680043.9.5704.983743739",
		instance: "1.2.826.0.1.3680043.9.5704.983743739.2",
	}
)

func getHealthcareVars(t *testing.T) map[string]any {
	switch "" {
	case healthcareProject:
		t.Fatal("'HEALTHCARE_PROJECT' not set")
	case healthcareRegion:
		t.Fatal("'HEALTHCARE_REGION' not set")
	case healthcareDataset:
		t.Fatal("'HEALTHCARE_DATASET' not set")
	case healthcarePrepopulatedDICOMStore:
		t.Fatal("'HEALTHCARE_PREPOPULATED_DICOM_STORE' not set")
	}
	return map[string]any{
		"kind":    healthcareSourceKind,
		"project": healthcareProject,
		"region":  healthcareRegion,
		"dataset": healthcareDataset,
	}
}

func TestHealthcareToolEndpoints(t *testing.T) {
	sourceConfig := getHealthcareVars(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	healthcareService, err := newHealthcareService(ctx)
	if err != nil {
		t.Fatalf("failed to create healthcare service: %v", err)
	}

	fhirStoreID := "fhir-store-" + uuid.New().String()
	dicomStoreID := "dicom-store-" + uuid.New().String()

	patient1ID, patient2ID, teardown := setupHealthcareResources(t, healthcareService, healthcareDataset, fhirStoreID, dicomStoreID)
	defer teardown(t)

	toolsFile := getToolsConfig(sourceConfig)
	toolsFile = addClientAuthSourceConfig(t, toolsFile)

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
	runGetFHIRStoreToolInvokeTest(t, fhirStoreID, fhirStoreWant)
	runGetFHIRStoreMetricsToolInvokeTest(t, fhirStoreID, `"metrics"`)
	runGetFHIRResourceToolInvokeTest(t, fhirStoreID, "Patient", patient1ID, `"id":"`+patient1ID+`"`)
	runFHIRPatientSearchToolInvokeTest(t, fhirStoreID, patient1ID, patient2ID)
	runFHIRPatientEverythingToolInvokeTest(t, fhirStoreID, patient1ID, `"resourceType":"Bundle"`)

	nextURL := getNextPageURLForPatientEverything(t, fhirStoreID, patient2ID)
	runFHIRFetchPageToolInvokeTest(t, nextURL, `"total":1`)

	runGetDICOMStoreToolInvokeTest(t, dicomStoreID, dicomStoreWant)
	runGetDICOMStoreMetricsToolInvokeTest(t, healthcarePrepopulatedDICOMStore, `"structuredStorageSizeBytes"`)
	runSearchDICOMStudiesToolInvokeTest(t, healthcarePrepopulatedDICOMStore)
	runSearchDICOMSeriesToolInvokeTest(t, healthcarePrepopulatedDICOMStore)
	runSearchDICOMInstancesToolInvokeTest(t, healthcarePrepopulatedDICOMStore)
	runRetrieveRenderedDICOMInstanceToolInvokeTest(t, healthcarePrepopulatedDICOMStore)
}

func TestHealthcareToolWithStoreRestriction(t *testing.T) {
	sourceConfig := getHealthcareVars(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
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

	_, _, teardownAllowedStores := setupHealthcareResources(t, healthcareService, healthcareDataset, allowedFHIRStoreID, allowedDICOMStoreID)
	defer teardownAllowedStores(t)
	_, _, teardownDisallowedStores := setupHealthcareResources(t, healthcareService, healthcareDataset, disallowedFHIRStoreID, disallowedDICOMStoreID)
	defer teardownDisallowedStores(t)

	// Configure source with dataset restriction.
	sourceConfig["allowedFhirStores"] = []string{allowedFHIRStoreID}
	sourceConfig["allowedDicomStores"] = []string{allowedDICOMStoreID}

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

func createFHIRResource(t *testing.T, service *healthcare.Service, fhirStoreName, resourceType string, resourceBody io.Reader) (string, string) {
	resp, err := service.Projects.Locations.Datasets.FhirStores.Fhir.Create(fhirStoreName, resourceType, resourceBody).Do()
	if err != nil {
		t.Fatalf("failed to create FHIR resource: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("failed to create FHIR resource, status: %d, body: %s", resp.StatusCode, string(body))
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	id := result["id"].(string)
	return id, fmt.Sprintf("%s/%s", resourceType, id)
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

func setupHealthcareResources(t *testing.T, service *healthcare.Service, datasetID, fhirStoreID, dicomStoreID string) (string, string, func(*testing.T)) {
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

	// Create Patient 1
	patient1Body := bytes.NewBuffer([]byte(`{
		"resourceType":"Patient",
		"name":[{"use":"official","family":"Smith","given":["John"]}],
		"birthDate":"1980-01-01",
		"gender":"male",
		"address":[{"use":"home","line":["123 Main St"],"city":"san fransisco","state":"CA","postalCode":"12345","country":"USA"}],
		"active":true,
		"deceasedBoolean":false,
		"telecom":[{"system":"phone","value":"555-1234","use":"home"},{"system":"email","value":"john@foo.com","use":"work"}],
		"gender":"male",
		"identifier":[{"system":"http://hospital.org","value":"1234567"}],
		"communication":[{"language":{"coding":[{"system":"urn:ietf:bcp:47","code":"en","display":"English"}]},"preferred":true}]
	}`))
	patient1ID, patient1Name := createFHIRResource(t, service, fhirStore.Name, "Patient", patient1Body)

	// Create Observation for Patient 1
	observation1Body := bytes.NewBuffer([]byte(fmt.Sprintf(`
	{
		"resourceType": "Observation",
		"status": "final",
		"code": { "coding": [{ "system": "http://loinc.org", "code": "29463-7", "display": "Body Weight" }] },
		"subject": { "reference": "%s" },
		"valueQuantity": { "value": 185, "unit": "lbs", "system": "http://unitsofmeasure.org", "code": "[lb_av]" }
	}`, patient1Name)))
	createFHIRResource(t, service, fhirStore.Name, "Observation", observation1Body)

	// Create Patient 2
	patient2Body := bytes.NewBuffer([]byte(`{"resourceType":"Patient","name":[{"use":"official","family":"Doe","given":["Jane"]}]}`))
	patient2ID, patient2Name := createFHIRResource(t, service, fhirStore.Name, "Patient", patient2Body)

	// Create 101 Observations for Patient 2
	for i := 0; i < 101; i++ {
		observation2Body := bytes.NewBuffer([]byte(fmt.Sprintf(`
		{
			"resourceType": "Observation",
			"status": "final",
			"code": { "coding": [{ "system": "http://loinc.org", "code": "8302-2", "display": "Body Height" }] },
			"subject": { "reference": "%s" },
			"valueQuantity": { "value": 68, "unit": "in", "system": "http://unitsofmeasure.org", "code": "[in_i]" }
		}`, patient2Name)))
		createFHIRResource(t, service, fhirStore.Name, "Observation", observation2Body)
	}

	teardown := func(t *testing.T) {
		if _, err := service.Projects.Locations.Datasets.FhirStores.Delete(fhirStore.Name).Do(); err != nil {
			t.Logf("failed to delete fhir store: %v", err)
		}
		if _, err := service.Projects.Locations.Datasets.DicomStores.Delete(dicomStore.Name).Do(); err != nil {
			t.Logf("failed to delete dicom store: %v", err)
		}
	}
	return patient1ID, patient2ID, teardown
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
			"my-get-fhir-store-tool": map[string]any{
				"kind":        getFHIRStoreToolKind,
				"source":      "my-instance",
				"description": "Tool to get a FHIR store",
			},
			"my-get-fhir-store-metrics-tool": map[string]any{
				"kind":        getFHIRStoreMetricsToolKind,
				"source":      "my-instance",
				"description": "Tool to get FHIR store metrics",
			},
			"my-get-fhir-resource-tool": map[string]any{
				"kind":        getFHIRResourceToolKind,
				"source":      "my-instance",
				"description": "Tool to get FHIR resource",
			},
			"my-fhir-patient-search-tool": map[string]any{
				"kind":        fhirPatientSearchToolKind,
				"source":      "my-instance",
				"description": "Tool to search for patients",
			},
			"my-fhir-patient-everything-tool": map[string]any{
				"kind":        fhirPatientEverythingToolKind,
				"source":      "my-instance",
				"description": "Tool for patient everything",
			},
			"my-fhir-fetch-page-tool": map[string]any{
				"kind":        fhirFetchPageToolKind,
				"source":      "my-instance",
				"description": "Tool to fetch a page of FHIR resources",
			},
			"my-get-dicom-store-tool": map[string]any{
				"kind":        getDICOMStoreToolKind,
				"source":      "my-instance",
				"description": "Tool to get a DICOM store",
			},
			"my-get-dicom-store-metrics-tool": map[string]any{
				"kind":        getDICOMStoreMetricsToolKind,
				"source":      "my-instance",
				"description": "Tool to get DICOM store metrics",
			},
			"my-search-dicom-studies-tool": map[string]any{
				"kind":        searchDICOMStudiesToolKind,
				"source":      "my-instance",
				"description": "Tool to search DICOM studies",
			},
			"my-search-dicom-series-tool": map[string]any{
				"kind":        searchDICOMSeriesToolKind,
				"source":      "my-instance",
				"description": "Tool to search DICOM series",
			},
			"my-search-dicom-instances-tool": map[string]any{
				"kind":        searchDICOMInstancesToolKind,
				"source":      "my-instance",
				"description": "Tool to search DICOM instances",
			},
			"my-retrieve-rendered-dicom-instance-tool": map[string]any{
				"kind":        retrieveRenderedDICOMInstanceToolKind,
				"source":      "my-instance",
				"description": "Tool to retrieve rendered DICOM instance",
			},
			"my-client-auth-get-dataset-tool": map[string]any{
				"kind":        getDatasetToolKind,
				"source":      "my-client-auth-source",
				"description": "Tool to get a healthcare dataset",
			},
			"my-client-auth-list-fhir-stores-tool": map[string]any{
				"kind":        listFHIRStoresToolKind,
				"source":      "my-client-auth-source",
				"description": "Tool to list FHIR stores",
			},
			"my-client-auth-list-dicom-stores-tool": map[string]any{
				"kind":        listDICOMStoresToolKind,
				"source":      "my-client-auth-source",
				"description": "Tool to list DICOM stores",
			},
			"my-client-auth-get-fhir-store-tool": map[string]any{
				"kind":        getFHIRStoreToolKind,
				"source":      "my-client-auth-source",
				"description": "Tool to get a FHIR store",
			},
			"my-client-auth-get-fhir-store-metrics-tool": map[string]any{
				"kind":        getFHIRStoreMetricsToolKind,
				"source":      "my-client-auth-source",
				"description": "Tool to get FHIR store metrics",
			},
			"my-client-auth-get-fhir-resource-tool": map[string]any{
				"kind":        getFHIRResourceToolKind,
				"source":      "my-client-auth-source",
				"description": "Tool to get FHIR resource",
			},
			"my-client-auth-fhir-patient-search-tool": map[string]any{
				"kind":        fhirPatientSearchToolKind,
				"source":      "my-client-auth-source",
				"description": "Tool to search for patients",
			},
			"my-client-auth-fhir-patient-everything-tool": map[string]any{
				"kind":        fhirPatientEverythingToolKind,
				"source":      "my-client-auth-source",
				"description": "Tool for patient everything",
			},
			"my-client-auth-fhir-fetch-page-tool": map[string]any{
				"kind":        fhirFetchPageToolKind,
				"source":      "my-client-auth-source",
				"description": "Tool to fetch a page of FHIR resources",
			},
			"my-client-auth-get-dicom-store-tool": map[string]any{
				"kind":        getDICOMStoreToolKind,
				"source":      "my-client-auth-source",
				"description": "Tool to get a DICOM store",
			},
			"my-client-auth-get-dicom-store-metrics-tool": map[string]any{
				"kind":        getDICOMStoreMetricsToolKind,
				"source":      "my-client-auth-source",
				"description": "Tool to get DICOM store metrics",
			},
			"my-client-auth-search-dicom-studies-tool": map[string]any{
				"kind":        searchDICOMStudiesToolKind,
				"source":      "my-client-auth-source",
				"description": "Tool to search DICOM studies",
			},
			"my-client-auth-search-dicom-series-tool": map[string]any{
				"kind":        searchDICOMSeriesToolKind,
				"source":      "my-client-auth-source",
				"description": "Tool to search DICOM series",
			},
			"my-client-auth-search-dicom-instances-tool": map[string]any{
				"kind":        searchDICOMInstancesToolKind,
				"source":      "my-client-auth-source",
				"description": "Tool to search DICOM instances",
			},
			"my-client-auth-retrieve-rendered-dicom-instance-tool": map[string]any{
				"kind":        retrieveRenderedDICOMInstanceToolKind,
				"source":      "my-client-auth-source",
				"description": "Tool to retrieve rendered DICOM instance",
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
			"my-auth-get-fhir-store-tool": map[string]any{
				"kind":        getFHIRStoreToolKind,
				"source":      "my-instance",
				"description": "Tool to get a FHIR store",
				"authRequired": []string{
					"my-google-auth",
				},
			},
			"my-auth-get-fhir-store-metrics-tool": map[string]any{
				"kind":        getFHIRStoreMetricsToolKind,
				"source":      "my-instance",
				"description": "Tool to get FHIR store metrics",
				"authRequired": []string{
					"my-google-auth",
				},
			},
			"my-auth-get-fhir-resource-tool": map[string]any{
				"kind":        getFHIRResourceToolKind,
				"source":      "my-instance",
				"description": "Tool to get FHIR resource",
				"authRequired": []string{
					"my-google-auth",
				},
			},
			"my-auth-fhir-patient-search-tool": map[string]any{
				"kind":        fhirPatientSearchToolKind,
				"source":      "my-instance",
				"description": "Tool to search for patients",
				"authRequired": []string{
					"my-google-auth",
				},
			},
			"my-auth-fhir-patient-everything-tool": map[string]any{
				"kind":        fhirPatientEverythingToolKind,
				"source":      "my-instance",
				"description": "Tool for patient everything",
				"authRequired": []string{
					"my-google-auth",
				},
			},
			"my-auth-fhir-fetch-page-tool": map[string]any{
				"kind":        fhirFetchPageToolKind,
				"source":      "my-instance",
				"description": "Tool to fetch a page of FHIR resources",
				"authRequired": []string{
					"my-google-auth",
				},
			},
			"my-auth-get-dicom-store-tool": map[string]any{
				"kind":        getDICOMStoreToolKind,
				"source":      "my-instance",
				"description": "Tool to get a DICOM store",
				"authRequired": []string{
					"my-google-auth",
				},
			},
			"my-auth-get-dicom-store-metrics-tool": map[string]any{
				"kind":        getDICOMStoreMetricsToolKind,
				"source":      "my-instance",
				"description": "Tool to get DICOM store metrics",
				"authRequired": []string{
					"my-google-auth",
				},
			},
			"my-auth-search-dicom-studies-tool": map[string]any{
				"kind":        searchDICOMStudiesToolKind,
				"source":      "my-instance",
				"description": "Tool to search DICOM studies",
				"authRequired": []string{
					"my-google-auth",
				},
			},
			"my-auth-search-dicom-series-tool": map[string]any{
				"kind":        searchDICOMSeriesToolKind,
				"source":      "my-instance",
				"description": "Tool to search DICOM series",
				"authRequired": []string{
					"my-google-auth",
				},
			},
			"my-auth-search-dicom-instances-tool": map[string]any{
				"kind":        searchDICOMInstancesToolKind,
				"source":      "my-instance",
				"description": "Tool to search DICOM instances",
				"authRequired": []string{
					"my-google-auth",
				},
			},
			"my-auth-retrieve-rendered-dicom-instance-tool": map[string]any{
				"kind":        retrieveRenderedDICOMInstanceToolKind,
				"source":      "my-instance",
				"description": "Tool to retrieve rendered DICOM instance",
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

func addClientAuthSourceConfig(t *testing.T, config map[string]any) map[string]any {
	sources, ok := config["sources"].(map[string]any)
	if !ok {
		t.Fatalf("unable to get sources from config")
	}
	sources["my-client-auth-source"] = map[string]any{
		"kind":           healthcareSourceKind,
		"project":        healthcareProject,
		"region":         healthcareRegion,
		"dataset":        healthcareDataset,
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
		{
			name:          "invoke my-client-auth-get-dataset-tool with client auth",
			api:           "http://127.0.0.1:5000/api/tool/my-client-auth-get-dataset-tool/invoke",
			requestHeader: map[string]string{"Authorization": accessToken},
			requestBody:   bytes.NewBuffer([]byte(`{}`)),
			want:          want,
			isErr:         false,
		},
		{
			name:          "invoke my-client-auth-get-dataset-tool without auth token",
			api:           "http://127.0.0.1:5000/api/tool/my-client-auth-get-dataset-tool/invoke",
			requestHeader: map[string]string{},
			requestBody:   bytes.NewBuffer([]byte(`{}`)),
			isErr:         true,
		},
		{
			name:          "invoke my-client-auth-get-dataset-tool with invalid auth token",
			api:           "http://127.0.0.1:5000/api/tool/my-client-auth-get-dataset-tool/invoke",
			requestHeader: map[string]string{"my-google-auth_token": idToken},
			requestBody:   bytes.NewBuffer([]byte(`{}`)),
			isErr:         true,
		},
	}
	for _, tc := range invokeTcs {
		t.Run(tc.name, func(t *testing.T) {
			got, status := runTest(t, tc.api, tc.requestHeader, tc.requestBody)
			if tc.isErr {
				if status == http.StatusOK {
					t.Errorf("expected error but got success")
				}
				return
			}
			if status != http.StatusOK {
				t.Errorf("expected status OK but got %d", status)
			} else if !strings.Contains(got, tc.want) {
				t.Errorf("expected result to contain %q but got %q", tc.want, got)
			}
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
		{
			name:          "invoke my-client-auth-list-fhir-stores-tool with client auth",
			api:           "http://127.0.0.1:5000/api/tool/my-client-auth-list-fhir-stores-tool/invoke",
			requestHeader: map[string]string{"Authorization": accessToken},
			requestBody:   bytes.NewBuffer([]byte(`{}`)),
			want:          want,
			isErr:         false,
		},
		{
			name:          "invoke my-client-auth-list-fhir-stores-tool without auth token",
			api:           "http://127.0.0.1:5000/api/tool/my-client-auth-list-fhir-stores-tool/invoke",
			requestHeader: map[string]string{},
			requestBody:   bytes.NewBuffer([]byte(`{}`)),
			isErr:         true,
		},
		{
			name:          "invoke my-client-auth-list-fhir-stores-tool with invalid auth token",
			api:           "http://127.0.0.1:5000/api/tool/my-client-auth-list-fhir-stores-tool/invoke",
			requestHeader: map[string]string{"my-google-auth_token": idToken},
			requestBody:   bytes.NewBuffer([]byte(`{}`)),
			isErr:         true,
		},
	}
	for _, tc := range invokeTcs {
		t.Run(tc.name, func(t *testing.T) {
			got, status := runTest(t, tc.api, tc.requestHeader, tc.requestBody)
			if tc.isErr {
				if status == http.StatusOK {
					t.Errorf("expected error but got success")
				}
				return
			}
			if status != http.StatusOK {
				t.Errorf("expected status OK but got %d", status)
			} else if !strings.Contains(got, tc.want) {
				t.Errorf("expected result to contain %q but got %q", tc.want, got)
			}
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
		{
			name:          "invoke my-client-auth-list-dicom-stores-tool with client auth",
			api:           "http://127.0.0.1:5000/api/tool/my-client-auth-list-dicom-stores-tool/invoke",
			requestHeader: map[string]string{"Authorization": accessToken},
			requestBody:   bytes.NewBuffer([]byte(`{}`)),
			want:          want,
			isErr:         false,
		},
		{
			name:          "invoke my-client-auth-list-dicom-stores-tool without auth token",
			api:           "http://127.0.0.1:5000/api/tool/my-client-auth-list-dicom-stores-tool/invoke",
			requestHeader: map[string]string{},
			requestBody:   bytes.NewBuffer([]byte(`{}`)),
			isErr:         true,
		},
		{
			name:          "invoke my-client-auth-list-dicom-stores-tool with invalid auth token",
			api:           "http://127.0.0.1:5000/api/tool/my-client-auth-list-dicom-stores-tool/invoke",
			requestHeader: map[string]string{"my-google-auth_token": idToken},
			requestBody:   bytes.NewBuffer([]byte(`{}`)),
			isErr:         true,
		},
	}
	for _, tc := range invokeTcs {
		t.Run(tc.name, func(t *testing.T) {
			got, status := runTest(t, tc.api, tc.requestHeader, tc.requestBody)
			if tc.isErr {
				if status == http.StatusOK {
					t.Errorf("expected error but got success")
				}
				return
			}
			if status != http.StatusOK {
				t.Errorf("expected status OK but got %d", status)
			} else if !strings.Contains(got, tc.want) {
				t.Errorf("expected result to contain %q but got %q", tc.want, got)
			}
		})
	}
}

func runGetFHIRStoreToolInvokeTest(t *testing.T, fhirStoreID, want string) {
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
			name:          "invoke my-get-fhir-store-tool",
			api:           "http://127.0.0.1:5000/api/tool/my-get-fhir-store-tool/invoke",
			requestHeader: map[string]string{},
			requestBody:   bytes.NewBuffer([]byte(`{"storeID":"` + fhirStoreID + `"}`)),
			want:          want,
			isErr:         false,
		},
		{
			name:          "invoke my-auth-get-fhir-store-tool with auth",
			api:           "http://127.0.0.1:5000/api/tool/my-auth-get-fhir-store-tool/invoke",
			requestHeader: map[string]string{"my-google-auth_token": idToken},
			requestBody:   bytes.NewBuffer([]byte(`{"storeID":"` + fhirStoreID + `"}`)),
			want:          want,
			isErr:         false,
		},
		{
			name:          "invoke my-auth-get-fhir-store-tool with client auth",
			api:           "http://127.0.0.1:5000/api/tool/my-get-fhir-store-tool/invoke",
			requestHeader: map[string]string{"Authorization": accessToken},
			requestBody:   bytes.NewBuffer([]byte(`{"storeID":"` + fhirStoreID + `"}`)),
			want:          want,
			isErr:         false,
		},
		{
			name:          "invoke my-auth-get-fhir-store-tool without auth token",
			api:           "http://127.0.0.1:5000/api/tool/my-auth-get-fhir-store-tool/invoke",
			requestHeader: map[string]string{},
			requestBody:   bytes.NewBuffer([]byte(`{"storeID":"` + fhirStoreID + `"}`)),
			isErr:         true,
		},
		{
			name:          "invoke my-auth-get-fhir-store-tool with invalid auth token",
			api:           "http://127.0.0.1:5000/api/tool/my-auth-get-fhir-store-tool/invoke",
			requestHeader: map[string]string{"Authorization": "invalid-token"},
			requestBody:   bytes.NewBuffer([]byte(`{"storeID":"` + fhirStoreID + `"}`)),
			isErr:         true,
		},
		{
			name:          "invoke my-get-fhir-store-tool with invalid storeID",
			api:           "http://127.0.0.1:5000/api/tool/my-get-fhir-store-tool/invoke",
			requestHeader: map[string]string{},
			requestBody:   bytes.NewBuffer([]byte(`{"storeID":"invalid-store"}`)),
			isErr:         true,
		},
		{
			name:          "invoke my-client-auth-get-fhir-store-tool with client auth",
			api:           "http://127.0.0.1:5000/api/tool/my-client-auth-get-fhir-store-tool/invoke",
			requestHeader: map[string]string{"Authorization": accessToken},
			requestBody:   bytes.NewBuffer([]byte(`{"storeID":"` + fhirStoreID + `"}`)),
			want:          want,
			isErr:         false,
		},
		{
			name:          "invoke my-client-auth-get-fhir-store-tool without auth token",
			api:           "http://127.0.0.1:5000/api/tool/my-client-auth-get-fhir-store-tool/invoke",
			requestHeader: map[string]string{},
			requestBody:   bytes.NewBuffer([]byte(`{"storeID":"` + fhirStoreID + `"}`)),
			isErr:         true,
		},
		{
			name:          "invoke my-client-auth-get-fhir-store-tool with invalid auth token",
			api:           "http://127.0.0.1:5000/api/tool/my-client-auth-get-fhir-store-tool/invoke",
			requestHeader: map[string]string{"my-google-auth_token": idToken},
			requestBody:   bytes.NewBuffer([]byte(`{"storeID":"` + fhirStoreID + `"}`)),
			isErr:         true,
		},
	}
	for _, tc := range invokeTcs {
		t.Run(tc.name, func(t *testing.T) {
			got, status := runTest(t, tc.api, tc.requestHeader, tc.requestBody)
			if tc.isErr {
				if status == http.StatusOK {
					t.Errorf("expected error but got success")
				}
				return
			}
			if status != http.StatusOK {
				t.Errorf("expected status OK but got %d", status)
			} else if !strings.Contains(got, tc.want) {
				t.Errorf("expected result to contain %q but got %q", tc.want, got)
			}
		})
	}
}

func runGetFHIRStoreMetricsToolInvokeTest(t *testing.T, fhirStoreID, want string) {
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
			name:          "invoke my-get-fhir-store-metrics-tool",
			api:           "http://127.0.0.1:5000/api/tool/my-get-fhir-store-metrics-tool/invoke",
			requestHeader: map[string]string{},
			requestBody:   bytes.NewBuffer([]byte(`{"storeID":"` + fhirStoreID + `"}`)),
			want:          want,
			isErr:         false,
		},
		{
			name:          "invoke my-auth-get-fhir-store-metrics-tool with auth",
			api:           "http://127.0.0.1:5000/api/tool/my-auth-get-fhir-store-metrics-tool/invoke",
			requestHeader: map[string]string{"my-google-auth_token": idToken},
			requestBody:   bytes.NewBuffer([]byte(`{"storeID":"` + fhirStoreID + `"}`)),
			want:          want,
			isErr:         false,
		},
		{
			name:          "invoke my-auth-get-fhir-store-metrics-tool with client auth",
			api:           "http://127.0.0.1:5000/api/tool/my-get-fhir-store-metrics-tool/invoke",
			requestHeader: map[string]string{"Authorization": accessToken},
			requestBody:   bytes.NewBuffer([]byte(`{"storeID":"` + fhirStoreID + `"}`)),
			want:          want,
			isErr:         false,
		},
		{
			name:          "invoke my-auth-get-fhir-store-metrics-tool without auth token",
			api:           "http://127.0.0.1:5000/api/tool/my-auth-get-fhir-store-metrics-tool/invoke",
			requestHeader: map[string]string{},
			requestBody:   bytes.NewBuffer([]byte(`{"storeID":"` + fhirStoreID + `"}`)),
			isErr:         true,
		},
		{
			name:          "invoke my-auth-get-fhir-store-metrics-tool with invalid auth token",
			api:           "http://127.0.0.1:5000/api/tool/my-auth-get-fhir-store-metrics-tool/invoke",
			requestHeader: map[string]string{"Authorization": "invalid-token"},
			requestBody:   bytes.NewBuffer([]byte(`{"storeID":"` + fhirStoreID + `"}`)),
			isErr:         true,
		},
		{
			name:          "invoke my-get-fhir-store-metrics-tool with invalid storeID",
			api:           "http://127.0.0.1:5000/api/tool/my-get-fhir-store-metrics-tool/invoke",
			requestHeader: map[string]string{},
			requestBody:   bytes.NewBuffer([]byte(`{"storeID":"invalid-store"}`)),
			isErr:         true,
		},
		{
			name:          "invoke my-client-auth-get-fhir-store-metrics-tool with client auth",
			api:           "http://127.0.0.1:5000/api/tool/my-client-auth-get-fhir-store-metrics-tool/invoke",
			requestHeader: map[string]string{"Authorization": accessToken},
			requestBody:   bytes.NewBuffer([]byte(`{"storeID":"` + fhirStoreID + `"}`)),
			want:          want,
			isErr:         false,
		},
		{
			name:          "invoke my-client-auth-get-fhir-store-metrics-tool without auth token",
			api:           "http://127.0.0.1:5000/api/tool/my-client-auth-get-fhir-store-metrics-tool/invoke",
			requestHeader: map[string]string{},
			requestBody:   bytes.NewBuffer([]byte(`{"storeID":"` + fhirStoreID + `"}`)),
			isErr:         true,
		},
		{
			name:          "invoke my-client-auth-get-fhir-store-metrics-tool with invalid auth token",
			api:           "http://127.0.0.1:5000/api/tool/my-client-auth-get-fhir-store-metrics-tool/invoke",
			requestHeader: map[string]string{"my-google-auth_token": idToken},
			requestBody:   bytes.NewBuffer([]byte(`{"storeID":"` + fhirStoreID + `"}`)),
			isErr:         true,
		},
	}
	for _, tc := range invokeTcs {
		t.Run(tc.name, func(t *testing.T) {
			got, status := runTest(t, tc.api, tc.requestHeader, tc.requestBody)
			if tc.isErr {
				if status == http.StatusOK {
					t.Errorf("expected error but got success")
				}
				return
			}
			if status != http.StatusOK {
				t.Errorf("expected status OK but got %d", status)
			} else if !strings.Contains(got, tc.want) {
				t.Errorf("expected result to contain %q but got %q", tc.want, got)
			}
		})
	}
}

func runGetFHIRResourceToolInvokeTest(t *testing.T, storeID, resType, resID, want string) {
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
			name:          "invoke my-get-fhir-resource-tool",
			api:           "http://127.0.0.1:5000/api/tool/my-get-fhir-resource-tool/invoke",
			requestHeader: map[string]string{},
			requestBody:   bytes.NewBuffer([]byte(`{"storeID":"` + storeID + `", "resourceType":"` + resType + `", "resourceID":"` + resID + `"}`)),
			want:          want,
			isErr:         false,
		},
		{
			name:          "invoke my-auth-get-fhir-resource-tool with auth",
			api:           "http://127.0.0.1:5000/api/tool/my-auth-get-fhir-resource-tool/invoke",
			requestHeader: map[string]string{"my-google-auth_token": idToken},
			requestBody:   bytes.NewBuffer([]byte(`{"storeID":"` + storeID + `", "resourceType":"` + resType + `", "resourceID":"` + resID + `"}`)),
			want:          want,
			isErr:         false,
		},
		{
			name:          "invoke my-auth-get-fhir-resource-tool with client auth",
			api:           "http://127.0.0.1:5000/api/tool/my-get-fhir-resource-tool/invoke",
			requestHeader: map[string]string{"Authorization": accessToken},
			requestBody:   bytes.NewBuffer([]byte(`{"storeID":"` + storeID + `", "resourceType":"` + resType + `", "resourceID":"` + resID + `"}`)),
			want:          want,
			isErr:         false,
		},
		{
			name:          "invoke my-auth-get-fhir-resource-tool without auth token",
			api:           "http://127.0.0.1:5000/api/tool/my-auth-get-fhir-resource-tool/invoke",
			requestHeader: map[string]string{},
			requestBody:   bytes.NewBuffer([]byte(`{"storeID":"` + storeID + `", "resourceType":"` + resType + `", "resourceID":"` + resID + `"}`)),
			isErr:         true,
		},
		{
			name:          "invoke my-auth-get-fhir-resource-tool with invalid auth token",
			api:           "http://127.0.0.1:5000/api/tool/my-auth-get-fhir-resource-tool/invoke",
			requestHeader: map[string]string{"Authorization": "invalid-token"},
			requestBody:   bytes.NewBuffer([]byte(`{"storeID":"` + storeID + `", "resourceType":"` + resType + `", "resourceID":"` + resID + `"}`)),
			isErr:         true,
		},
		{
			name:          "invoke my-get-fhir-resource-tool with non-existent resource",
			api:           "http://127.0.0.1:5000/api/tool/my-get-fhir-resource-tool/invoke",
			requestHeader: map[string]string{},
			requestBody:   bytes.NewBuffer([]byte(`{"storeID":"` + storeID + `", "resourceType":"` + resType + `", "resourceID":"foo"}`)),
			isErr:         true,
		},
		{
			name:          "invoke my-get-fhir-resource-tool with missing required id parameter",
			api:           "http://127.0.0.1:5000/api/tool/my-get-fhir-resource-tool/invoke",
			requestHeader: map[string]string{},
			requestBody:   bytes.NewBuffer([]byte(`{"storeID":"` + storeID + `", "resourceType":"` + resType + `"}`)),
			isErr:         true,
		},
		{
			name:          "invoke my-client-auth-get-fhir-resource-tool with client auth",
			api:           "http://127.0.0.1:5000/api/tool/my-client-auth-get-fhir-resource-tool/invoke",
			requestHeader: map[string]string{"Authorization": accessToken},
			requestBody:   bytes.NewBuffer([]byte(`{"storeID":"` + storeID + `", "resourceType":"` + resType + `", "resourceID":"` + resID + `"}`)),
			want:          want,
			isErr:         false,
		},
		{
			name:          "invoke my-client-auth-get-fhir-resource-tool without auth token",
			api:           "http://127.0.0.1:5000/api/tool/my-client-auth-get-fhir-resource-tool/invoke",
			requestHeader: map[string]string{},
			requestBody:   bytes.NewBuffer([]byte(`{"storeID":"` + storeID + `", "resourceType":"` + resType + `", "resourceID":"` + resID + `"}`)),
			isErr:         true,
		},
		{
			name:          "invoke my-client-auth-get-fhir-resource-tool with invalid auth token",
			api:           "http://127.0.0.1:5000/api/tool/my-client-auth-get-fhir-resource-tool/invoke",
			requestHeader: map[string]string{"my-google-auth_token": idToken},
			requestBody:   bytes.NewBuffer([]byte(`{"storeID":"` + storeID + `", "resourceType":"` + resType + `", "resourceID":"` + resID + `"}`)),
			isErr:         true,
		},
	}
	for _, tc := range invokeTcs {
		t.Run(tc.name, func(t *testing.T) {
			got, status := runTest(t, tc.api, tc.requestHeader, tc.requestBody)
			if tc.isErr {
				if status == http.StatusOK {
					t.Errorf("expected error but got success")
				}
				return
			}
			if status != http.StatusOK {
				t.Errorf("expected status OK but got %d", status)
			} else if !strings.Contains(got, tc.want) {
				t.Errorf("expected result to contain %q but got %q", tc.want, got)
			}
		})
	}
}

func runFHIRPatientSearchToolInvokeTest(t *testing.T, fhirStoreID string, patientIDs ...string) {
	idToken, err := tests.GetGoogleIdToken(tests.ClientId)
	if err != nil {
		t.Fatalf("error getting Google ID token: %s", err)
	}

	accessToken, err := sources.GetIAMAccessToken(t.Context())
	if err != nil {
		t.Fatalf("error getting access token from ADC: %s", err)
	}
	accessToken = "Bearer " + accessToken
	want := `"total":` + fmt.Sprintf(`%d`, len(patientIDs))

	invokeTcs := []struct {
		name          string
		api           string
		requestHeader map[string]string
		requestBody   io.Reader
		want          string
		isErr         bool
	}{
		{
			name:          "invoke my-fhir-patient-search-tool",
			api:           "http://127.0.0.1:5000/api/tool/my-fhir-patient-search-tool/invoke",
			requestHeader: map[string]string{},
			requestBody:   bytes.NewBuffer([]byte(`{"storeID":"` + fhirStoreID + `"}`)),
			want:          want,
			isErr:         false,
		},
		{
			name:          "invoke my-auth-fhir-patient-search-tool with auth",
			api:           "http://127.0.0.1:5000/api/tool/my-auth-fhir-patient-search-tool/invoke",
			requestHeader: map[string]string{"my-google-auth_token": idToken},
			requestBody:   bytes.NewBuffer([]byte(`{"storeID":"` + fhirStoreID + `"}`)),
			want:          want,
			isErr:         false,
		},
		{
			name:          "invoke my-auth-fhir-patient-search-tool with client auth",
			api:           "http://127.0.0.1:5000/api/tool/my-fhir-patient-search-tool/invoke",
			requestHeader: map[string]string{"Authorization": accessToken},
			requestBody:   bytes.NewBuffer([]byte(`{"storeID":"` + fhirStoreID + `"}`)),
			want:          want,
			isErr:         false,
		},
		{
			name:          "invoke my-auth-fhir-patient-search-tool without auth token",
			api:           "http://127.0.0.1:5000/api/tool/my-auth-fhir-patient-search-tool/invoke",
			requestHeader: map[string]string{},
			requestBody:   bytes.NewBuffer([]byte(`{"storeID":"` + fhirStoreID + `"}`)),
			isErr:         true,
		},
		{
			name:          "invoke my-auth-fhir-patient-search-tool with invalid auth token",
			api:           "http://127.0.0.1:5000/api/tool/my-auth-fhir-patient-search-tool/invoke",
			requestHeader: map[string]string{"Authorization": "invalid-token"},
			requestBody:   bytes.NewBuffer([]byte(`{"storeID":"` + fhirStoreID + `"}`)),
			isErr:         true,
		},
		{
			name:          "invoke my-auth-fhir-patient-search-tool with invalid auth token",
			api:           "http://127.0.0.1:5000/api/tool/my-auth-fhir-patient-search-tool/invoke",
			requestHeader: map[string]string{"Authorization": "invalid-token"},
			requestBody:   bytes.NewBuffer([]byte(`{"storeID":"` + fhirStoreID + `"}`)),
			isErr:         true,
		},
		{
			name:          "invoke my-fhir-patient-search-tool wrong parameter type",
			api:           "http://127.0.0.1:5000/api/tool/my-fhir-patient-search-tool/invoke",
			requestHeader: map[string]string{},
			requestBody:   bytes.NewBuffer([]byte(`{"storeID":"` + fhirStoreID + `", "name":true}`)),
			isErr:         true,
		},
		{
			name:          "invoke my-fhir-patient-search-tool filters",
			api:           "http://127.0.0.1:5000/api/tool/my-fhir-patient-search-tool/invoke",
			requestHeader: map[string]string{},
			requestBody:   bytes.NewBuffer([]byte(`{"storeID":"` + fhirStoreID + `", "name":"john","gender":"male","state":"CA","active":"true","birthDateRange":"1970-01-01/2000-12-31"}`)),
			want:          patientIDs[0],
			isErr:         false,
		},
		{
			name:          "invoke my-fhir-patient-search-tool filters 2",
			api:           "http://127.0.0.1:5000/api/tool/my-fhir-patient-search-tool/invoke",
			requestHeader: map[string]string{},
			requestBody:   bytes.NewBuffer([]byte(`{"storeID":"` + fhirStoreID + `", "givenName":"john","addressSubstring":"main st","email":"john@foo.com","phone":"555-1234","language":"en","deceased":"false"}`)),
			want:          patientIDs[0],
			isErr:         false,
		},
		{
			name:          "invoke my-fhir-patient-search-tool filters 3",
			api:           "http://127.0.0.1:5000/api/tool/my-fhir-patient-search-tool/invoke",
			requestHeader: map[string]string{},
			requestBody:   bytes.NewBuffer([]byte(`{"storeID":"` + fhirStoreID + `", "familyName":"smith","country":"USA","addressUse":"home","postalCode":"12345","identifier":"1234567"}`)),
			want:          patientIDs[0],
			isErr:         false,
		},
		{
			name:          "invoke my-fhir-patient-search-tool zero matches",
			api:           "http://127.0.0.1:5000/api/tool/my-fhir-patient-search-tool/invoke",
			requestHeader: map[string]string{},
			requestBody:   bytes.NewBuffer([]byte(`{"storeID":"` + fhirStoreID + `", "gender":"unknown"}`)),
			want:          `"total":0`,
			isErr:         false,
		},
		{
			name:          "invoke my-fhir-patient-search-tool match second patient only",
			api:           "http://127.0.0.1:5000/api/tool/my-fhir-patient-search-tool/invoke",
			requestHeader: map[string]string{},
			requestBody:   bytes.NewBuffer([]byte(`{"storeID":"` + fhirStoreID + `","familyName":"Doe"}`)),
			want:          patientIDs[1],
			isErr:         false,
		},
		{
			name:          "invoke my-client-auth-fhir-patient-search-tool with client auth",
			api:           "http://127.0.0.1:5000/api/tool/my-client-auth-fhir-patient-search-tool/invoke",
			requestHeader: map[string]string{"Authorization": accessToken},
			requestBody:   bytes.NewBuffer([]byte(`{"storeID":"` + fhirStoreID + `"}`)),
			want:          want,
			isErr:         false,
		},
		{
			name:          "invoke my-client-auth-fhir-patient-search-tool without auth token",
			api:           "http://127.0.0.1:5000/api/tool/my-client-auth-fhir-patient-search-tool/invoke",
			requestHeader: map[string]string{},
			requestBody:   bytes.NewBuffer([]byte(`{"storeID":"` + fhirStoreID + `"}`)),
			isErr:         true,
		},
		{
			name:          "invoke my-client-auth-fhir-patient-search-tool with invalid auth token",
			api:           "http://127.0.0.1:5000/api/tool/my-client-auth-fhir-patient-search-tool/invoke",
			requestHeader: map[string]string{"my-google-auth_token": idToken},
			requestBody:   bytes.NewBuffer([]byte(`{"storeID":"` + fhirStoreID + `"}`)),
			isErr:         true,
		},
	}
	for _, tc := range invokeTcs {
		t.Run(tc.name, func(t *testing.T) {
			got, status := runTest(t, tc.api, tc.requestHeader, tc.requestBody)
			if tc.isErr {
				if status == http.StatusOK {
					t.Errorf("expected error but got success")
				}
				return
			}
			if status != http.StatusOK {
				t.Errorf("expected status OK but got %d", status)
			} else if !strings.Contains(got, tc.want) {
				t.Errorf("expected result to contain %q but got %q", tc.want, got)
			}
		})
	}
}

func runFHIRPatientEverythingToolInvokeTest(t *testing.T, fhirStoreID, patientID, want string) {
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
			name:          "invoke my-fhir-patient-everything-tool",
			api:           "http://127.0.0.1:5000/api/tool/my-fhir-patient-everything-tool/invoke",
			requestHeader: map[string]string{},
			requestBody:   bytes.NewBuffer([]byte(`{"storeID":"` + fhirStoreID + `", "patientID":"` + patientID + `"}`)),
			want:          want,
			isErr:         false,
		},
		{
			name:          "invoke my-auth-fhir-patient-everything-tool with auth",
			api:           "http://127.0.0.1:5000/api/tool/my-auth-fhir-patient-everything-tool/invoke",
			requestHeader: map[string]string{"my-google-auth_token": idToken},
			requestBody:   bytes.NewBuffer([]byte(`{"storeID":"` + fhirStoreID + `", "patientID":"` + patientID + `"}`)),
			want:          want,
			isErr:         false,
		},
		{
			name:          "invoke my-auth-fhir-patient-everything-tool with client auth",
			api:           "http://127.0.0.1:5000/api/tool/my-fhir-patient-everything-tool/invoke",
			requestHeader: map[string]string{"Authorization": accessToken},
			requestBody:   bytes.NewBuffer([]byte(`{"storeID":"` + fhirStoreID + `", "patientID":"` + patientID + `"}`)),
			want:          want,
			isErr:         false,
		},
		{
			name:          "invoke my-auth-fhir-patient-everything-tool without auth token",
			api:           "http://127.0.0.1:5000/api/tool/my-auth-fhir-patient-everything-tool/invoke",
			requestHeader: map[string]string{},
			requestBody:   bytes.NewBuffer([]byte(`{"storeID":"` + fhirStoreID + `", "patientID":"` + patientID + `"}`)),
			isErr:         true,
		},
		{
			name:          "invoke my-auth-fhir-patient-everything-tool with invalid auth token",
			api:           "http://127.0.0.1:5000/api/tool/my-auth-fhir-patient-everything-tool/invoke",
			requestHeader: map[string]string{"Authorization": "invalid-token"},
			requestBody:   bytes.NewBuffer([]byte(`{"storeID":"` + fhirStoreID + `", "patientID":"` + patientID + `"}`)),
			isErr:         true,
		},
		{
			name:          "invoke my-fhir-patient-everything-tool with non-existent patient",
			api:           "http://127.0.0.1:5000/api/tool/my-fhir-patient-everything-tool/invoke",
			requestHeader: map[string]string{},
			requestBody:   bytes.NewBuffer([]byte(`{"storeID":"` + fhirStoreID + `", "patientID":"foo"`)),
			isErr:         true,
		},
		{
			name:          "invoke my-fhir-patient-everything-tool with invalid since filter format",
			api:           "http://127.0.0.1:5000/api/tool/my-fhir-patient-everything-tool/invoke",
			requestHeader: map[string]string{},
			requestBody:   bytes.NewBuffer([]byte(`{"storeID":"` + fhirStoreID + `", "patientID":"` + patientID + `","sinceFilter":"October 10th, 2023"}`)),
			isErr:         true,
		},
		{
			name:          "invoke my-fhir-patient-everything-tool with type and since filters",
			api:           "http://127.0.0.1:5000/api/tool/my-fhir-patient-everything-tool/invoke",
			requestHeader: map[string]string{},
			requestBody:   bytes.NewBuffer([]byte(`{"storeID":"` + fhirStoreID + `", "patientID":"` + patientID + `","sinceFilter":"2023-01-01T00:00:00Z","resourceTypesFilter":["Observation"]}`)),
			want:          `"total":2`,
			isErr:         false,
		},
		{
			name:          "invoke my-fhir-patient-everything-tool with type and since keeps only patient",
			api:           "http://127.0.0.1:5000/api/tool/my-fhir-patient-everything-tool/invoke",
			requestHeader: map[string]string{},
			requestBody:   bytes.NewBuffer([]byte(`{"storeID":"` + fhirStoreID + `", "patientID":"` + patientID + `","sinceFilter":"` + time.Now().Format(time.RFC3339) + `","resourceTypesFilter":["Observation","Encounter"]}`)),
			want:          `"total":1`,
			isErr:         false,
		},
		{
			name:          "invoke my-fhir-patient-everything-tool with type keeps only patient",
			api:           "http://127.0.0.1:5000/api/tool/my-fhir-patient-everything-tool/invoke",
			requestHeader: map[string]string{},
			requestBody:   bytes.NewBuffer([]byte(`{"storeID":"` + fhirStoreID + `", "patientID":"` + patientID + `","resourceTypesFilter":["Encounter","Claim"]}`)),
			want:          `"total":1`,
			isErr:         false,
		},
		{
			name:          "invoke my-client-auth-fhir-patient-everything-tool with client auth",
			api:           "http://127.0.0.1:5000/api/tool/my-client-auth-fhir-patient-everything-tool/invoke",
			requestHeader: map[string]string{"Authorization": accessToken},
			requestBody:   bytes.NewBuffer([]byte(`{"storeID":"` + fhirStoreID + `", "patientID":"` + patientID + `"}`)),
			want:          want,
			isErr:         false,
		},
		{
			name:          "invoke my-client-auth-fhir-patient-everything-tool without auth token",
			api:           "http://127.0.0.1:5000/api/tool/my-client-auth-fhir-patient-everything-tool/invoke",
			requestHeader: map[string]string{},
			requestBody:   bytes.NewBuffer([]byte(`{"storeID":"` + fhirStoreID + `", "patientID":"` + patientID + `"}`)),
			isErr:         true,
		},
		{
			name:          "invoke my-client-auth-fhir-patient-everything-tool with invalid auth token",
			api:           "http://127.0.0.1:5000/api/tool/my-client-auth-fhir-patient-everything-tool/invoke",
			requestHeader: map[string]string{"my-google-auth_token": idToken},
			requestBody:   bytes.NewBuffer([]byte(`{"storeID":"` + fhirStoreID + `", "patientID":"` + patientID + `"}`)),
			isErr:         true,
		},
	}
	for _, tc := range invokeTcs {
		t.Run(tc.name, func(t *testing.T) {
			got, status := runTest(t, tc.api, tc.requestHeader, tc.requestBody)
			if tc.isErr {
				if status == http.StatusOK {
					t.Errorf("expected error but got success")
				}
				return
			}
			if status != http.StatusOK {
				t.Errorf("expected status OK but got %d", status)
			} else if !strings.Contains(got, tc.want) {
				t.Errorf("expected result to contain %q but got %q", tc.want, got)
			}
		})
	}
}

func runFHIRFetchPageToolInvokeTest(t *testing.T, pageURL, want string) {
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
			name:          "invoke my-fhir-fetch-page-tool",
			api:           "http://127.0.0.1:5000/api/tool/my-fhir-fetch-page-tool/invoke",
			requestHeader: map[string]string{},
			requestBody:   bytes.NewBuffer([]byte(`{"pageURL":"` + pageURL + `"}`)),
			want:          want,
			isErr:         false,
		},
		{
			name:          "invoke my-auth-fhir-fetch-page-tool with auth",
			api:           "http://127.0.0.1:5000/api/tool/my-auth-fhir-fetch-page-tool/invoke",
			requestHeader: map[string]string{"my-google-auth_token": idToken},
			requestBody:   bytes.NewBuffer([]byte(`{"pageURL":"` + pageURL + `"}`)),
			want:          want,
			isErr:         false,
		},
		{
			name:          "invoke my-auth-fhir-fetch-page-tool with client auth",
			api:           "http://127.0.0.1:5000/api/tool/my-fhir-fetch-page-tool/invoke",
			requestHeader: map[string]string{"Authorization": accessToken},
			requestBody:   bytes.NewBuffer([]byte(`{"pageURL":"` + pageURL + `"}`)),
			want:          want,
			isErr:         false,
		},
		{
			name:          "invoke my-auth-fhir-fetch-page-tool without auth token",
			api:           "http://127.0.0.1:5000/api/tool/my-auth-fhir-fetch-page-tool/invoke",
			requestHeader: map[string]string{},
			requestBody:   bytes.NewBuffer([]byte(`{"pageURL":"` + pageURL + `"}`)),
			isErr:         true,
		},
		{
			name:          "invoke my-auth-fhir-fetch-page-tool with invalid auth token",
			api:           "http://127.0.0.1:5000/api/tool/my-auth-fhir-fetch-page-tool/invoke",
			requestHeader: map[string]string{"Authorization": "invalid-token"},
			requestBody:   bytes.NewBuffer([]byte(`{"pageURL":"` + pageURL + `"}`)),
			isErr:         true,
		},
		{
			name:          "invoke my-fhir-fetch-page-tool with invalid url",
			api:           "http://127.0.0.1:5000/api/tool/my-fhir-fetch-page-tool/invoke",
			requestHeader: map[string]string{},
			requestBody:   bytes.NewBuffer([]byte(`{"pageURL":"google.com"}`)),
			isErr:         true,
		},
		{
			name:          "invoke my-client-auth-fhir-fetch-page-tool with client auth",
			api:           "http://127.0.0.1:5000/api/tool/my-client-auth-fhir-fetch-page-tool/invoke",
			requestHeader: map[string]string{"Authorization": accessToken},
			requestBody:   bytes.NewBuffer([]byte(`{"pageURL":"` + pageURL + `"}`)),
			want:          want,
			isErr:         false,
		},
		{
			name:          "invoke my-client-auth-fhir-fetch-page-tool without auth token",
			api:           "http://127.0.0.1:5000/api/tool/my-client-auth-fhir-fetch-page-tool/invoke",
			requestHeader: map[string]string{},
			requestBody:   bytes.NewBuffer([]byte(`{"pageURL":"` + pageURL + `"}`)),
			isErr:         true,
		},
		{
			name:          "invoke my-client-auth-fhir-fetch-page-tool with invalid auth token",
			api:           "http://127.0.0.1:5000/api/tool/my-client-auth-fhir-fetch-page-tool/invoke",
			requestHeader: map[string]string{"my-google-auth_token": idToken},
			requestBody:   bytes.NewBuffer([]byte(`{"pageURL":"` + pageURL + `"}`)),
			isErr:         true,
		},
	}
	for _, tc := range invokeTcs {
		t.Run(tc.name, func(t *testing.T) {
			got, status := runTest(t, tc.api, tc.requestHeader, tc.requestBody)
			if tc.isErr {
				if status == http.StatusOK {
					t.Errorf("expected error but got success")
				}
				return
			}
			if status != http.StatusOK {
				t.Errorf("expected status OK but got %d", status)
			} else if !strings.Contains(got, tc.want) {
				t.Errorf("expected result to contain %q but got %q", tc.want, got)
			}
		})
	}
}

func getNextPageURLForPatientEverything(t *testing.T, fhirStoreID, patientID string) string {
	api := "http://127.0.0.1:5000/api/tool/my-fhir-patient-everything-tool/invoke"
	reqBody := fmt.Sprintf(`{"storeID": "%s", "patientID": "%s"}`, fhirStoreID, patientID)
	resp, bodyBytes := tests.RunRequest(t, http.MethodPost, api, bytes.NewBuffer([]byte(reqBody)), map[string]string{"Content-type": "application/json"})
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("response status code is not 200, got %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var body map[string]interface{}
	err := json.Unmarshal(bodyBytes, &body)
	if err != nil {
		t.Fatalf("error parsing response body")
	}

	resultStr, ok := body["result"].(string)
	if !ok {
		t.Fatalf("unable to find result in response body")
	}

	var resultJSON map[string]interface{}
	if err := json.Unmarshal([]byte(resultStr), &resultJSON); err != nil {
		t.Fatalf("failed to unmarshal result string: %v", err)
	}

	links, ok := resultJSON["link"].([]interface{})
	if !ok {
		t.Fatalf("no link field in result")
	}

	for _, l := range links {
		link := l.(map[string]interface{})
		if relation, ok := link["relation"].(string); ok && relation == "next" {
			if url, ok := link["url"].(string); ok {
				return url
			}
		}
	}
	t.Fatalf("next link not found in patient everything response")
	return ""
}

func runTest(t *testing.T, api string, requestHeader map[string]string, requestBody io.Reader) (string, int) {
	resp, bodyBytes := tests.RunRequest(t, http.MethodPost, api, requestBody, requestHeader)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", resp.StatusCode
	}

	var body map[string]interface{}
	err := json.Unmarshal(bodyBytes, &body)
	if err != nil {
		t.Fatalf("error parsing response body")
	}

	got, ok := body["result"].(string)
	if !ok {
		t.Fatalf("unable to find result in response body")
	}
	return got, http.StatusOK
}

func runListFHIRStoresWithRestriction(t *testing.T, allowedFHIRStore, disallowedFHIRStore string) {
	api := "http://127.0.0.1:5000/api/tool/list-fhir-stores-restricted/invoke"
	got, status := runTest(t, api, map[string]string{"Content-type": "application/json"}, bytes.NewBuffer([]byte(`{}`)))
	if status != http.StatusOK {
		t.Fatalf("expected status OK but got %d", status)
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
	got, status := runTest(t, api, map[string]string{"Content-type": "application/json"}, bytes.NewBuffer([]byte(`{}`)))
	if status != http.StatusOK {
		t.Fatalf("expected status OK but got %d", status)
	}

	if !strings.Contains(got, allowedDICOMStore) {
		t.Fatalf("expected %q to contain %q, but it did not", got, allowedDICOMStore)
	}
	if strings.Contains(got, disallowedDICOMStore) {
		t.Fatalf("expected %q to NOT contain %q, but it did", got, disallowedDICOMStore)
	}
}

func runGetDICOMStoreToolInvokeTest(t *testing.T, dicomStoreID, want string) {
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
			name:          "invoke my-get-dicom-store-tool",
			api:           "http://127.0.0.1:5000/api/tool/my-get-dicom-store-tool/invoke",
			requestHeader: map[string]string{},
			requestBody:   bytes.NewBuffer([]byte(`{"storeID":"` + dicomStoreID + `"}`)),
			want:          want,
			isErr:         false,
		},
		{
			name:          "invoke my-auth-get-dicom-store-tool with auth",
			api:           "http://127.0.0.1:5000/api/tool/my-auth-get-dicom-store-tool/invoke",
			requestHeader: map[string]string{"my-google-auth_token": idToken},
			requestBody:   bytes.NewBuffer([]byte(`{"storeID":"` + dicomStoreID + `"}`)),
			want:          want,
			isErr:         false,
		},
		{
			name:          "invoke my-auth-get-dicom-store-tool with client auth",
			api:           "http://127.0.0.1:5000/api/tool/my-get-dicom-store-tool/invoke",
			requestHeader: map[string]string{"Authorization": accessToken},
			requestBody:   bytes.NewBuffer([]byte(`{"storeID":"` + dicomStoreID + `"}`)),
			want:          want,
			isErr:         false,
		},
		{
			name:          "invoke my-auth-get-dicom-store-tool without auth token",
			api:           "http://127.0.0.1:5000/api/tool/my-auth-get-dicom-store-tool/invoke",
			requestHeader: map[string]string{},
			requestBody:   bytes.NewBuffer([]byte(`{"storeID":"` + dicomStoreID + `"}`)),
			isErr:         true,
		},
		{
			name:          "invoke my-auth-get-dicom-store-tool with invalid auth token",
			api:           "http://127.0.0.1:5000/api/tool/my-auth-get-dicom-store-tool/invoke",
			requestHeader: map[string]string{"Authorization": "invalid-token"},
			requestBody:   bytes.NewBuffer([]byte(`{"storeID":"` + dicomStoreID + `"}`)),
			isErr:         true,
		},
		{
			name:          "invoke my-get-dicom-store-tool with invalid storeID",
			api:           "http://127.0.0.1:5000/api/tool/my-get-dicom-store-tool/invoke",
			requestHeader: map[string]string{},
			requestBody:   bytes.NewBuffer([]byte(`{"storeID":"invalid-store"}`)),
			isErr:         true,
		},
		{
			name:          "invoke my-client-auth-get-dicom-store-tool with client auth",
			api:           "http://127.0.0.1:5000/api/tool/my-client-auth-get-dicom-store-tool/invoke",
			requestHeader: map[string]string{"Authorization": accessToken},
			requestBody:   bytes.NewBuffer([]byte(`{"storeID":"` + dicomStoreID + `"}`)),
			want:          want,
			isErr:         false,
		},
		{
			name:          "invoke my-client-auth-get-dicom-store-tool without auth token",
			api:           "http://127.0.0.1:5000/api/tool/my-client-auth-get-dicom-store-tool/invoke",
			requestHeader: map[string]string{},
			requestBody:   bytes.NewBuffer([]byte(`{"storeID":"` + dicomStoreID + `"}`)),
			isErr:         true,
		},
		{
			name:          "invoke my-client-auth-get-dicom-store-tool with invalid auth token",
			api:           "http://127.0.0.1:5000/api/tool/my-client-auth-get-dicom-store-tool/invoke",
			requestHeader: map[string]string{"my-google-auth_token": idToken},
			requestBody:   bytes.NewBuffer([]byte(`{"storeID":"` + dicomStoreID + `"}`)),
			isErr:         true,
		},
	}
	for _, tc := range invokeTcs {
		t.Run(tc.name, func(t *testing.T) {
			got, status := runTest(t, tc.api, tc.requestHeader, tc.requestBody)
			if tc.isErr {
				if status == http.StatusOK {
					t.Errorf("expected error but got success")
				}
				return
			}
			if status != http.StatusOK {
				t.Errorf("expected status OK but got %d", status)
			} else if !strings.Contains(got, tc.want) {
				t.Errorf("expected result to contain %q but got %q", tc.want, got)
			}
		})
	}
}

func runGetDICOMStoreMetricsToolInvokeTest(t *testing.T, dicomStoreID, want string) {
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
			name:          "invoke my-get-dicom-store-metrics-tool",
			api:           "http://127.0.0.1:5000/api/tool/my-get-dicom-store-metrics-tool/invoke",
			requestHeader: map[string]string{},
			requestBody:   bytes.NewBuffer([]byte(`{"storeID":"` + dicomStoreID + `"}`)),
			want:          want,
			isErr:         false,
		},
		{
			name:          "invoke my-auth-get-dicom-store-metrics-tool with auth",
			api:           "http://127.0.0.1:5000/api/tool/my-auth-get-dicom-store-metrics-tool/invoke",
			requestHeader: map[string]string{"my-google-auth_token": idToken},
			requestBody:   bytes.NewBuffer([]byte(`{"storeID":"` + dicomStoreID + `"}`)),
			want:          want,
			isErr:         false,
		},
		{
			name:          "invoke my-auth-get-dicom-store-metrics-tool with client auth",
			api:           "http://127.0.0.1:5000/api/tool/my-get-dicom-store-metrics-tool/invoke",
			requestHeader: map[string]string{"Authorization": accessToken},
			requestBody:   bytes.NewBuffer([]byte(`{"storeID":"` + dicomStoreID + `"}`)),
			want:          want,
			isErr:         false,
		},
		{
			name:          "invoke my-auth-get-dicom-store-metrics-tool without auth token",
			api:           "http://127.0.0.1:5000/api/tool/my-auth-get-dicom-store-metrics-tool/invoke",
			requestHeader: map[string]string{},
			requestBody:   bytes.NewBuffer([]byte(`{"storeID":"` + dicomStoreID + `"}`)),
			isErr:         true,
		},
		{
			name:          "invoke my-auth-get-dicom-store-metrics-tool with invalid auth token",
			api:           "http://127.0.0.1:5000/api/tool/my-auth-get-dicom-store-metrics-tool/invoke",
			requestHeader: map[string]string{"Authorization": "invalid-token"},
			requestBody:   bytes.NewBuffer([]byte(`{"storeID":"` + dicomStoreID + `"}`)),
			isErr:         true,
		},
		{
			name:          "invoke my-get-dicom-store-metrics-tool with invalid storeID",
			api:           "http://127.0.0.1:5000/api/tool/my-get-dicom-store-metrics-tool/invoke",
			requestHeader: map[string]string{},
			requestBody:   bytes.NewBuffer([]byte(`{"storeID":"invalid-store"}`)),
			isErr:         true,
		},
		{
			name:          "invoke my-client-auth-get-dicom-store-metrics-tool with client auth",
			api:           "http://127.0.0.1:5000/api/tool/my-client-auth-get-dicom-store-metrics-tool/invoke",
			requestHeader: map[string]string{"Authorization": accessToken},
			requestBody:   bytes.NewBuffer([]byte(`{"storeID":"` + dicomStoreID + `"}`)),
			want:          want,
			isErr:         false,
		},
		{
			name:          "invoke my-client-auth-get-dicom-store-metrics-tool without auth token",
			api:           "http://127.0.0.1:5000/api/tool/my-client-auth-get-dicom-store-metrics-tool/invoke",
			requestHeader: map[string]string{},
			requestBody:   bytes.NewBuffer([]byte(`{"storeID":"` + dicomStoreID + `"}`)),
			isErr:         true,
		},
		{
			name:          "invoke my-client-auth-get-dicom-store-metrics-tool with invalid auth token",
			api:           "http://127.0.0.1:5000/api/tool/my-client-auth-get-dicom-store-metrics-tool/invoke",
			requestHeader: map[string]string{"my-google-auth_token": idToken},
			requestBody:   bytes.NewBuffer([]byte(`{"storeID":"` + dicomStoreID + `"}`)),
			isErr:         true,
		},
	}
	for _, tc := range invokeTcs {
		t.Run(tc.name, func(t *testing.T) {
			got, status := runTest(t, tc.api, tc.requestHeader, tc.requestBody)
			if tc.isErr {
				if status == http.StatusOK {
					t.Errorf("expected error but got success")
				}
				return
			}
			if status != http.StatusOK {
				t.Errorf("expected status OK but got %d", status)
			} else if !strings.Contains(got, tc.want) {
				t.Errorf("expected result to contain %q but got %q", tc.want, got)
			}
		})
	}
}

func runSearchDICOMStudiesToolInvokeTest(t *testing.T, dicomStoreID string) {
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
			name:          "invoke my-search-dicom-studies-tool",
			api:           "http://127.0.0.1:5000/api/tool/my-search-dicom-studies-tool/invoke",
			requestHeader: map[string]string{},
			requestBody:   bytes.NewBuffer([]byte(`{"storeID":"` + dicomStoreID + `"}`)),
			want:          singleFrameDICOMInstance.study,
			isErr:         false,
		},
		{
			name:          "invoke my-auth-search-dicom-studies-tool with auth",
			api:           "http://127.0.0.1:5000/api/tool/my-auth-search-dicom-studies-tool/invoke",
			requestHeader: map[string]string{"my-google-auth_token": idToken},
			requestBody:   bytes.NewBuffer([]byte(`{"storeID":"` + dicomStoreID + `"}`)),
			want:          multiFrameDICOMInstance.study,
			isErr:         false,
		},
		{
			name:          "invoke my-auth-search-dicom-studies-tool with client auth",
			api:           "http://127.0.0.1:5000/api/tool/my-search-dicom-studies-tool/invoke",
			requestHeader: map[string]string{"Authorization": accessToken},
			requestBody:   bytes.NewBuffer([]byte(`{"storeID":"` + dicomStoreID + `"}`)),
			want:          multiFrameDICOMInstance.study,
			isErr:         false,
		},
		{
			name:          "invoke my-auth-search-dicom-studies-tool without auth token",
			api:           "http://127.0.0.1:5000/api/tool/my-auth-search-dicom-studies-tool/invoke",
			requestHeader: map[string]string{},
			requestBody:   bytes.NewBuffer([]byte(`{"storeID":"` + dicomStoreID + `"}`)),
			isErr:         true,
		},
		{
			name:          "invoke my-auth-search-dicom-studies-tool with invalid auth token",
			api:           "http://127.0.0.1:5000/api/tool/my-auth-search-dicom-studies-tool/invoke",
			requestHeader: map[string]string{"Authorization": "invalid-token"},
			requestBody:   bytes.NewBuffer([]byte(`{"storeID":"` + dicomStoreID + `"}`)),
			isErr:         true,
		},
		{
			name:          "invoke my-search-dicom-studies-tool with invalid storeID",
			api:           "http://127.0.0.1:5000/api/tool/my-search-dicom-studies-tool/invoke",
			requestHeader: map[string]string{},
			requestBody:   bytes.NewBuffer([]byte(`{"storeID":"invalid-store"}`)),
			isErr:         true,
		},
		{
			name:          "invoke my-client-auth-search-dicom-studies-tool with client auth",
			api:           "http://127.0.0.1:5000/api/tool/my-client-auth-search-dicom-studies-tool/invoke",
			requestHeader: map[string]string{"Authorization": accessToken},
			requestBody:   bytes.NewBuffer([]byte(`{"storeID":"` + dicomStoreID + `"}`)),
			want:          singleFrameDICOMInstance.study,
			isErr:         false,
		},
		{
			name:          "invoke my-client-auth-search-dicom-studies-tool without auth token",
			api:           "http://127.0.0.1:5000/api/tool/my-client-auth-search-dicom-studies-tool/invoke",
			requestHeader: map[string]string{},
			requestBody:   bytes.NewBuffer([]byte(`{"storeID":"` + dicomStoreID + `"}`)),
			isErr:         true,
		},
		{
			name:          "invoke my-client-auth-search-dicom-studies-tool with invalid auth token",
			api:           "http://127.0.0.1:5000/api/tool/my-client-auth-search-dicom-studies-tool/invoke",
			requestHeader: map[string]string{"my-google-auth_token": idToken},
			requestBody:   bytes.NewBuffer([]byte(`{"storeID":"` + dicomStoreID + `"}`)),
			isErr:         true,
		},
		{
			name:          "invoke my-search-dicom-studies-tool with patient name and fuzzy matching",
			api:           "http://127.0.0.1:5000/api/tool/my-search-dicom-studies-tool/invoke",
			requestHeader: map[string]string{},
			requestBody:   bytes.NewBuffer([]byte(`{"storeID":"` + dicomStoreID + `", "PatientName":"Andrew", "fuzzymatching":true}`)),
			want:          multiFrameDICOMInstance.study,
			isErr:         false,
		},
		{
			name:          "invoke my-search-dicom-studies-tool with patient id filter",
			api:           "http://127.0.0.1:5000/api/tool/my-search-dicom-studies-tool/invoke",
			requestHeader: map[string]string{},
			requestBody:   bytes.NewBuffer([]byte(`{"storeID":"` + dicomStoreID + `", "PatientID":"Joelle-del"}`)),
			want:          singleFrameDICOMInstance.study,
			isErr:         false,
		},
	}
	for _, tc := range invokeTcs {
		t.Run(tc.name, func(t *testing.T) {
			got, status := runTest(t, tc.api, tc.requestHeader, tc.requestBody)
			if tc.isErr {
				if status == http.StatusOK {
					t.Errorf("expected error but got success")
				}
				return
			}
			if status != http.StatusOK {
				t.Errorf("expected status OK but got %d", status)
			} else if !strings.Contains(got, tc.want) {
				t.Errorf("expected result to contain %q but got %q", tc.want, got)
			}
		})
	}
}

func runSearchDICOMSeriesToolInvokeTest(t *testing.T, dicomStoreID string) {
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
			name:          "invoke my-search-dicom-series-tool",
			api:           "http://127.0.0.1:5000/api/tool/my-search-dicom-series-tool/invoke",
			requestHeader: map[string]string{},
			requestBody:   bytes.NewBuffer([]byte(`{"storeID":"` + dicomStoreID + `"}`)),
			want:          singleFrameDICOMInstance.series,
			isErr:         false,
		},
		{
			name:          "invoke my-auth-search-dicom-series-tool with auth",
			api:           "http://127.0.0.1:5000/api/tool/my-auth-search-dicom-series-tool/invoke",
			requestHeader: map[string]string{"my-google-auth_token": idToken},
			requestBody:   bytes.NewBuffer([]byte(`{"storeID":"` + dicomStoreID + `"}`)),
			want:          multiFrameDICOMInstance.series,
			isErr:         false,
		},
		{
			name:          "invoke my-auth-search-dicom-series-tool with client auth",
			api:           "http://127.0.0.1:5000/api/tool/my-search-dicom-series-tool/invoke",
			requestHeader: map[string]string{"Authorization": accessToken},
			requestBody:   bytes.NewBuffer([]byte(`{"storeID":"` + dicomStoreID + `"}`)),
			want:          multiFrameDICOMInstance.series,
			isErr:         false,
		},
		{
			name:          "invoke my-auth-search-dicom-series-tool without auth token",
			api:           "http://127.0.0.1:5000/api/tool/my-auth-search-dicom-series-tool/invoke",
			requestHeader: map[string]string{},
			requestBody:   bytes.NewBuffer([]byte(`{"storeID":"` + dicomStoreID + `"}`)),
			isErr:         true,
		},
		{
			name:          "invoke my-auth-search-dicom-series-tool with invalid auth token",
			api:           "http://127.0.0.1:5000/api/tool/my-auth-search-dicom-series-tool/invoke",
			requestHeader: map[string]string{"Authorization": "invalid-token"},
			requestBody:   bytes.NewBuffer([]byte(`{"storeID":"` + dicomStoreID + `"}`)),
			isErr:         true,
		},
		{
			name:          "invoke my-search-dicom-series-tool with invalid storeID",
			api:           "http://127.0.0.1:5000/api/tool/my-search-dicom-series-tool/invoke",
			requestHeader: map[string]string{},
			requestBody:   bytes.NewBuffer([]byte(`{"storeID":"invalid-store"}`)),
			isErr:         true,
		},
		{
			name:          "invoke my-client-auth-search-dicom-series-tool with client auth",
			api:           "http://127.0.0.1:5000/api/tool/my-client-auth-search-dicom-series-tool/invoke",
			requestHeader: map[string]string{"Authorization": accessToken},
			requestBody:   bytes.NewBuffer([]byte(`{"storeID":"` + dicomStoreID + `"}`)),
			want:          singleFrameDICOMInstance.series,
			isErr:         false,
		},
		{
			name:          "invoke my-client-auth-search-dicom-series-tool without auth token",
			api:           "http://127.0.0.1:5000/api/tool/my-client-auth-search-dicom-series-tool/invoke",
			requestHeader: map[string]string{},
			requestBody:   bytes.NewBuffer([]byte(`{"storeID":"` + dicomStoreID + `"}`)),
			isErr:         true,
		},
		{
			name:          "invoke my-client-auth-search-dicom-series-tool with invalid auth token",
			api:           "http://127.0.0.1:5000/api/tool/my-client-auth-search-dicom-series-tool/invoke",
			requestHeader: map[string]string{"my-google-auth_token": idToken},
			requestBody:   bytes.NewBuffer([]byte(`{"storeID":"` + dicomStoreID + `"}`)),
			isErr:         true,
		},
		{
			name:          "invoke my-search-dicom-series-tool with study date and referring physician name filters",
			api:           "http://127.0.0.1:5000/api/tool/my-search-dicom-series-tool/invoke",
			requestHeader: map[string]string{},
			requestBody:   bytes.NewBuffer([]byte(`{"storeID":"` + dicomStoreID + `", "StudyDate":"20170101-20171231", "ReferringPhysicianName":"Frederick^Bryant^^Ph.D."}`)),
			want:          multiFrameDICOMInstance.series,
			isErr:         false,
		},
		{
			name:          "invoke my-search-dicom-series-tool with series instance uid",
			api:           "http://127.0.0.1:5000/api/tool/my-search-dicom-series-tool/invoke",
			requestHeader: map[string]string{},
			requestBody:   bytes.NewBuffer([]byte(`{"storeID":"` + dicomStoreID + `", "SeriesInstanceUID":"1.2.840.113619.2.176.3596.3364818.7819.1259708454.108"}`)),
			want:          singleFrameDICOMInstance.series,
			isErr:         false,
		},
	}
	for _, tc := range invokeTcs {
		t.Run(tc.name, func(t *testing.T) {
			got, status := runTest(t, tc.api, tc.requestHeader, tc.requestBody)
			if tc.isErr {
				if status == http.StatusOK {
					t.Errorf("expected error but got success")
				}
				return
			}
			if status != http.StatusOK {
				t.Errorf("expected status OK but got %d", status)
			} else if !strings.Contains(got, tc.want) {
				t.Errorf("expected result to contain %q but got %q", tc.want, got)
			}
		})
	}
}

func runSearchDICOMInstancesToolInvokeTest(t *testing.T, dicomStoreID string) {
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
			name:          "invoke my-search-dicom-instances-tool",
			api:           "http://127.0.0.1:5000/api/tool/my-search-dicom-instances-tool/invoke",
			requestHeader: map[string]string{},
			requestBody:   bytes.NewBuffer([]byte(`{"storeID":"` + dicomStoreID + `"}`)),
			want:          singleFrameDICOMInstance.instance,
			isErr:         false,
		},
		{
			name:          "invoke my-auth-search-dicom-instances-tool with auth",
			api:           "http://127.0.0.1:5000/api/tool/my-auth-search-dicom-instances-tool/invoke",
			requestHeader: map[string]string{"my-google-auth_token": idToken},
			requestBody:   bytes.NewBuffer([]byte(`{"storeID":"` + dicomStoreID + `"}`)),
			want:          multiFrameDICOMInstance.instance,
			isErr:         false,
		},
		{
			name:          "invoke my-auth-search-dicom-instances-tool with client auth",
			api:           "http://127.0.0.1:5000/api/tool/my-search-dicom-instances-tool/invoke",
			requestHeader: map[string]string{"Authorization": accessToken},
			requestBody:   bytes.NewBuffer([]byte(`{"storeID":"` + dicomStoreID + `"}`)),
			want:          multiFrameDICOMInstance.instance,
			isErr:         false,
		},
		{
			name:          "invoke my-auth-search-dicom-instances-tool without auth token",
			api:           "http://127.0.0.1:5000/api/tool/my-auth-search-dicom-instances-tool/invoke",
			requestHeader: map[string]string{},
			requestBody:   bytes.NewBuffer([]byte(`{"storeID":"` + dicomStoreID + `"}`)),
			isErr:         true,
		},
		{
			name:          "invoke my-auth-search-dicom-instances-tool with invalid auth token",
			api:           "http://127.0.0.1:5000/api/tool/my-auth-search-dicom-instances-tool/invoke",
			requestHeader: map[string]string{"Authorization": "invalid-token"},
			requestBody:   bytes.NewBuffer([]byte(`{"storeID":"` + dicomStoreID + `"}`)),
			isErr:         true,
		},
		{
			name:          "invoke my-search-dicom-instances-tool with invalid storeID",
			api:           "http://127.0.0.1:5000/api/tool/my-search-dicom-instances-tool/invoke",
			requestHeader: map[string]string{},
			requestBody:   bytes.NewBuffer([]byte(`{"storeID":"invalid-store"}`)),
			isErr:         true,
		},
		{
			name:          "invoke my-client-auth-search-dicom-instances-tool with client auth",
			api:           "http://127.0.0.1:5000/api/tool/my-client-auth-search-dicom-instances-tool/invoke",
			requestHeader: map[string]string{"Authorization": accessToken},
			requestBody:   bytes.NewBuffer([]byte(`{"storeID":"` + dicomStoreID + `"}`)),
			want:          singleFrameDICOMInstance.instance,
			isErr:         false,
		},
		{
			name:          "invoke my-client-auth-search-dicom-instances-tool without auth token",
			api:           "http://127.0.0.1:5000/api/tool/my-client-auth-search-dicom-instances-tool/invoke",
			requestHeader: map[string]string{},
			requestBody:   bytes.NewBuffer([]byte(`{"storeID":"` + dicomStoreID + `"}`)),
			isErr:         true,
		},
		{
			name:          "invoke my-client-auth-search-dicom-instances-tool with invalid auth token",
			api:           "http://127.0.0.1:5000/api/tool/my-client-auth-search-dicom-instances-tool/invoke",
			requestHeader: map[string]string{"my-google-auth_token": idToken},
			requestBody:   bytes.NewBuffer([]byte(`{"storeID":"` + dicomStoreID + `"}`)),
			isErr:         true,
		},
		{
			name:          "invoke my-search-dicom-instances-tool with modality filter",
			api:           "http://127.0.0.1:5000/api/tool/my-search-dicom-instances-tool/invoke",
			requestHeader: map[string]string{},
			requestBody:   bytes.NewBuffer([]byte(`{"storeID":"` + dicomStoreID + `", "modality":"SM"}`)),
			want:          multiFrameDICOMInstance.instance,
			isErr:         false,
		},
		{
			name:          "invoke my-search-dicom-instances-tool with include attribute",
			api:           "http://127.0.0.1:5000/api/tool/my-search-dicom-instances-tool/invoke",
			requestHeader: map[string]string{},
			requestBody:   bytes.NewBuffer([]byte(`{"storeID":"` + dicomStoreID + `", "includefield":["52009230"]}`)),
			want:          `"52009230"`,
			isErr:         false,
		},
	}
	for _, tc := range invokeTcs {
		t.Run(tc.name, func(t *testing.T) {
			got, status := runTest(t, tc.api, tc.requestHeader, tc.requestBody)
			if tc.isErr {
				if status == http.StatusOK {
					t.Errorf("expected error but got success")
				}
				return
			}
			if status != http.StatusOK {
				t.Errorf("expected status OK but got %d", status)
			} else if !strings.Contains(got, tc.want) {
				t.Errorf("expected result to contain %q but got %q", tc.want, got)
			}
		})
	}
}

func runRetrieveRenderedDICOMInstanceToolInvokeTest(t *testing.T, dicomStoreID string) {
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
		isErr         bool
	}{
		{
			name:          "invoke my-retrieve-rendered-dicom-instance-tool",
			api:           "http://127.0.0.1:5000/api/tool/my-retrieve-rendered-dicom-instance-tool/invoke",
			requestHeader: map[string]string{},
			requestBody:   bytes.NewBuffer([]byte(`{"storeID":"` + dicomStoreID + `", "StudyInstanceUID":"` + singleFrameDICOMInstance.study + `", "SeriesInstanceUID":"` + singleFrameDICOMInstance.series + `", "SOPInstanceUID":"` + singleFrameDICOMInstance.instance + `"}`)),
			isErr:         false,
		},
		{
			name:          "invoke my-auth-retrieve-rendered-dicom-instance-tool with auth",
			api:           "http://127.0.0.1:5000/api/tool/my-auth-retrieve-rendered-dicom-instance-tool/invoke",
			requestHeader: map[string]string{"my-google-auth_token": idToken},
			requestBody:   bytes.NewBuffer([]byte(`{"storeID":"` + dicomStoreID + `", "StudyInstanceUID":"` + singleFrameDICOMInstance.study + `", "SeriesInstanceUID":"` + singleFrameDICOMInstance.series + `", "SOPInstanceUID":"` + singleFrameDICOMInstance.instance + `"}`)),
			isErr:         false,
		},
		{
			name:          "invoke my-auth-retrieve-rendered-dicom-instance-tool with client auth",
			api:           "http://127.0.0.1:5000/api/tool/my-retrieve-rendered-dicom-instance-tool/invoke",
			requestHeader: map[string]string{"Authorization": accessToken},
			requestBody:   bytes.NewBuffer([]byte(`{"storeID":"` + dicomStoreID + `", "StudyInstanceUID":"` + singleFrameDICOMInstance.study + `", "SeriesInstanceUID":"` + singleFrameDICOMInstance.series + `", "SOPInstanceUID":"` + singleFrameDICOMInstance.instance + `"}`)),
			isErr:         false,
		},
		{
			name:          "invoke my-auth-retrieve-rendered-dicom-instance-tool without auth token",
			api:           "http://127.0.0.1:5000/api/tool/my-auth-retrieve-rendered-dicom-instance-tool/invoke",
			requestHeader: map[string]string{},
			requestBody:   bytes.NewBuffer([]byte(`{"storeID":"` + dicomStoreID + `", "StudyInstanceUID":"` + singleFrameDICOMInstance.study + `", "SeriesInstanceUID":"` + singleFrameDICOMInstance.series + `", "SOPInstanceUID":"` + singleFrameDICOMInstance.instance + `"}`)),
			isErr:         true,
		},
		{
			name:          "invoke my-auth-retrieve-rendered-dicom-instance-tool with invalid auth token",
			api:           "http://127.0.0.1:5000/api/tool/my-auth-retrieve-rendered-dicom-instance-tool/invoke",
			requestHeader: map[string]string{"Authorization": "invalid-token"},
			requestBody:   bytes.NewBuffer([]byte(`{"storeID":"` + dicomStoreID + `", "StudyInstanceUID":"` + singleFrameDICOMInstance.study + `", "SeriesInstanceUID":"` + singleFrameDICOMInstance.series + `", "SOPInstanceUID":"` + singleFrameDICOMInstance.instance + `"}`)),
			isErr:         true,
		},
		{
			name:          "invoke my-retrieve-rendered-dicom-instance-tool with invalid storeID",
			api:           "http://127.0.0.1:5000/api/tool/my-retrieve-rendered-dicom-instance-tool/invoke",
			requestHeader: map[string]string{},
			requestBody:   bytes.NewBuffer([]byte(`{"storeID":"invalid-store", "StudyInstanceUID":"` + singleFrameDICOMInstance.study + `", "SeriesInstanceUID":"` + singleFrameDICOMInstance.series + `", "SOPInstanceUID":"` + singleFrameDICOMInstance.instance + `"}`)),
			isErr:         true,
		},
		{
			name:          "invoke my-client-auth-retrieve-rendered-dicom-instance-tool with client auth",
			api:           "http://127.0.0.1:5000/api/tool/my-client-auth-retrieve-rendered-dicom-instance-tool/invoke",
			requestHeader: map[string]string{"Authorization": accessToken},
			requestBody:   bytes.NewBuffer([]byte(`{"storeID":"` + dicomStoreID + `", "StudyInstanceUID":"` + singleFrameDICOMInstance.study + `", "SeriesInstanceUID":"` + singleFrameDICOMInstance.series + `", "SOPInstanceUID":"` + singleFrameDICOMInstance.instance + `"}`)),
			isErr:         false,
		},
		{
			name:          "invoke my-client-auth-retrieve-rendered-dicom-instance-tool without auth token",
			api:           "http://127.0.0.1:5000/api/tool/my-client-auth-retrieve-rendered-dicom-instance-tool/invoke",
			requestHeader: map[string]string{},
			requestBody:   bytes.NewBuffer([]byte(`{"storeID":"` + dicomStoreID + `", "StudyInstanceUID":"` + singleFrameDICOMInstance.study + `", "SeriesInstanceUID":"` + singleFrameDICOMInstance.series + `", "SOPInstanceUID":"` + singleFrameDICOMInstance.instance + `"}`)),
			isErr:         true,
		},
		{
			name:          "invoke my-client-auth-retrieve-rendered-dicom-instance-tool with invalid auth token",
			api:           "http://127.0.0.1:5000/api/tool/my-client-auth-retrieve-rendered-dicom-instance-tool/invoke",
			requestHeader: map[string]string{"my-google-auth_token": idToken},
			requestBody:   bytes.NewBuffer([]byte(`{"storeID":"` + dicomStoreID + `", "StudyInstanceUID":"` + singleFrameDICOMInstance.study + `", "SeriesInstanceUID":"` + singleFrameDICOMInstance.series + `", "SOPInstanceUID":"` + singleFrameDICOMInstance.instance + `"}`)),
			isErr:         true,
		},
		{
			name:          "invoke my-retrieve-rendered-dicom-instance-tool second frame on single-frame instance",
			api:           "http://127.0.0.1:5000/api/tool/my-retrieve-rendered-dicom-instance-tool/invoke",
			requestHeader: map[string]string{},
			requestBody:   bytes.NewBuffer([]byte(`{"FrameNumber": 2, "storeID":"` + dicomStoreID + `", "StudyInstanceUID":"` + singleFrameDICOMInstance.study + `", "SeriesInstanceUID":"` + singleFrameDICOMInstance.series + `", "SOPInstanceUID":"` + singleFrameDICOMInstance.instance + `"}`)),
			isErr:         true,
		},
		{
			name:          "invoke my-retrieve-rendered-dicom-instance-tool second frame on multi-frame instance",
			api:           "http://127.0.0.1:5000/api/tool/my-retrieve-rendered-dicom-instance-tool/invoke",
			requestHeader: map[string]string{},
			requestBody:   bytes.NewBuffer([]byte(`{"FrameNumber": 2, "storeID":"` + dicomStoreID + `", "StudyInstanceUID":"` + multiFrameDICOMInstance.study + `", "SeriesInstanceUID":"` + multiFrameDICOMInstance.series + `", "SOPInstanceUID":"` + multiFrameDICOMInstance.instance + `"}`)),
			isErr:         false,
		},
	}
	for _, tc := range invokeTcs {
		t.Run(tc.name, func(t *testing.T) {
			_, status := runTest(t, tc.api, tc.requestHeader, tc.requestBody)
			if tc.isErr {
				if status == http.StatusOK {
					t.Errorf("expected error but got success")
				}
				return
			}
			if status != http.StatusOK {
				t.Errorf("expected status OK but got %d", status)
			}
		})
	}
}
