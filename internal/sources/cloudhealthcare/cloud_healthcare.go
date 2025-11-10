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

package cloudhealthcare

import (
	"context"
	"fmt"
	"net/http"

	"github.com/goccy/go-yaml"
	"github.com/googleapis/genai-toolbox/internal/sources"
	"github.com/googleapis/genai-toolbox/internal/util"
	"go.opentelemetry.io/otel/trace"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/healthcare/v1"
	"google.golang.org/api/option"
)

const SourceKind string = "cloud-healthcare"

// validate interface
var _ sources.SourceConfig = Config{}

type HealthcareServiceCreator func(tokenString string) (*healthcare.Service, error)

func init() {
	if !sources.Register(SourceKind, newConfig) {
		panic(fmt.Sprintf("source kind %q already registered", SourceKind))
	}
}

func newConfig(ctx context.Context, name string, decoder *yaml.Decoder) (sources.SourceConfig, error) {
	actual := Config{Name: name}
	if err := decoder.DecodeContext(ctx, &actual); err != nil {
		return nil, err
	}
	return actual, nil
}

type Config struct {
	// Healthcare configs
	Name               string   `yaml:"name" validate:"required"`
	Kind               string   `yaml:"kind" validate:"required"`
	Project            string   `yaml:"project" validate:"required"`
	Region             string   `yaml:"region" validate:"required"`
	Dataset            string   `yaml:"dataset" validate:"required"`
	AllowedFHIRStores  []string `yaml:"allowedFhirStores"`
	AllowedDICOMStores []string `yaml:"allowedDicomStores"`
	UseClientOAuth     bool     `yaml:"useClientOAuth"`
}

func (c Config) SourceConfigKind() string {
	return SourceKind
}

func (c Config) Initialize(ctx context.Context, tracer trace.Tracer) (sources.Source, error) {
	var service *healthcare.Service
	var serviceCreator HealthcareServiceCreator
	var tokenSource oauth2.TokenSource

	svc, tok, err := initHealthcareConnection(ctx, tracer, c.Name)
	if err != nil {
		return nil, fmt.Errorf("error creating service from ADC: %w", err)
	}
	if c.UseClientOAuth {
		serviceCreator, err = newHealthcareServiceCreator(ctx, tracer, c.Name)
		if err != nil {
			return nil, fmt.Errorf("error constructing service creator: %w", err)
		}
	} else {
		service = svc
		tokenSource = tok
	}

	dsName := fmt.Sprintf("projects/%s/locations/%s/datasets/%s", c.Project, c.Region, c.Dataset)
	if _, err = svc.Projects.Locations.Datasets.FhirStores.Get(dsName).Do(); err != nil {
		if gerr, ok := err.(*googleapi.Error); ok && gerr.Code == http.StatusNotFound {
			return nil, fmt.Errorf("dataset '%s' not found", dsName)
		}
		return nil, fmt.Errorf("failed to verify existence of dataset '%s': %w", dsName, err)
	}

	allowedFHIRStores := make(map[string]struct{})
	for _, store := range c.AllowedFHIRStores {
		name := fmt.Sprintf("%s/fhirStores/%s", dsName, store)
		_, err := svc.Projects.Locations.Datasets.FhirStores.Get(name).Do()
		if err != nil {
			if gerr, ok := err.(*googleapi.Error); ok && gerr.Code == http.StatusNotFound {
				return nil, fmt.Errorf("allowedFhirStore '%s' not found in dataset '%s'", store, dsName)
			}
			return nil, fmt.Errorf("failed to verify allowedFhirStore '%s' in datasest '%s': %w", store, dsName, err)
		}
		allowedFHIRStores[store] = struct{}{}
	}
	allowedDICOMStores := make(map[string]struct{})
	for _, store := range c.AllowedDICOMStores {
		name := fmt.Sprintf("%s/dicomStores/%s", dsName, store)
		_, err := svc.Projects.Locations.Datasets.DicomStores.Get(name).Do()
		if err != nil {
			if gerr, ok := err.(*googleapi.Error); ok && gerr.Code == http.StatusNotFound {
				return nil, fmt.Errorf("allowedDicomStore '%s' not found in dataset '%s'", store, dsName)
			}
			return nil, fmt.Errorf("failed to verify allowedDicomFhirStore '%s' in datasest '%s': %w", store, dsName, err)
		}
		allowedDICOMStores[store] = struct{}{}
	}
	s := &Source{
		name:               c.Name,
		kind:               SourceKind,
		project:            c.Project,
		region:             c.Region,
		dataset:            c.Dataset,
		service:            service,
		serviceCreator:     serviceCreator,
		tokenSource:        tokenSource,
		allowedFHIRStores:  allowedFHIRStores,
		allowedDICOMStores: allowedDICOMStores,
		useClientOAuth:     c.UseClientOAuth,
	}
	return s, nil
}

func newHealthcareServiceCreator(ctx context.Context, tracer trace.Tracer, name string) (func(string) (*healthcare.Service, error), error) {
	userAgent, err := util.UserAgentFromContext(ctx)
	if err != nil {
		return nil, err
	}
	return func(tokenString string) (*healthcare.Service, error) {
		return initHealthcareConnectionWithOAuthToken(ctx, tracer, name, userAgent, tokenString)
	}, nil
}

func initHealthcareConnectionWithOAuthToken(ctx context.Context, tracer trace.Tracer, name string, userAgent string, tokenString string) (*healthcare.Service, error) {
	ctx, span := sources.InitConnectionSpan(ctx, tracer, SourceKind, name)
	defer span.End()
	// Construct token source
	token := &oauth2.Token{
		AccessToken: string(tokenString),
	}
	ts := oauth2.StaticTokenSource(token)

	// Initialize the Healthcare service with tokenSource
	service, err := healthcare.NewService(ctx, option.WithUserAgent(userAgent), option.WithTokenSource(ts))
	if err != nil {
		return nil, fmt.Errorf("failed to create Healthcare service: %w", err)
	}
	service.UserAgent = userAgent
	return service, nil
}

func initHealthcareConnection(ctx context.Context, tracer trace.Tracer, name string) (*healthcare.Service, oauth2.TokenSource, error) {
	ctx, span := sources.InitConnectionSpan(ctx, tracer, SourceKind, name)
	defer span.End()

	cred, err := google.FindDefaultCredentials(ctx, healthcare.CloudHealthcareScope)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to find default Google Cloud credentials with scope %q: %w", healthcare.CloudHealthcareScope, err)
	}

	userAgent, err := util.UserAgentFromContext(ctx)
	if err != nil {
		return nil, nil, err
	}

	service, err := healthcare.NewService(ctx, option.WithUserAgent(userAgent), option.WithCredentials(cred))
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create Healthcare service: %w", err)
	}
	service.UserAgent = userAgent
	return service, cred.TokenSource, nil
}

var _ sources.Source = &Source{}

type Source struct {
	name               string `yaml:"name"`
	kind               string `yaml:"kind"`
	project            string
	region             string
	dataset            string
	service            *healthcare.Service
	serviceCreator     HealthcareServiceCreator
	tokenSource        oauth2.TokenSource
	allowedFHIRStores  map[string]struct{}
	allowedDICOMStores map[string]struct{}
	useClientOAuth     bool
}

func (s *Source) SourceKind() string {
	return SourceKind
}

func (s *Source) Project() string {
	return s.project
}

func (s *Source) Region() string {
	return s.region
}

func (s *Source) DatasetID() string {
	return s.dataset
}

func (s *Source) Service() *healthcare.Service {
	return s.service
}

func (s *Source) ServiceCreator() HealthcareServiceCreator {
	return s.serviceCreator
}

func (s *Source) TokenSource() oauth2.TokenSource {
	return s.tokenSource
}

func (s *Source) AllowedFHIRStores() map[string]struct{} {
	if len(s.allowedFHIRStores) == 0 {
		return nil
	}
	return s.allowedFHIRStores
}

func (s *Source) AllowedDICOMStores() map[string]struct{} {
	if len(s.allowedDICOMStores) == 0 {
		return nil
	}
	return s.allowedDICOMStores
}

func (s *Source) IsFHIRStoreAllowed(storeID string) bool {
	if len(s.allowedFHIRStores) == 0 {
		return true
	}
	_, ok := s.allowedFHIRStores[storeID]
	return ok
}

func (s *Source) IsDICOMStoreAllowed(storeID string) bool {
	if len(s.allowedDICOMStores) == 0 {
		return true
	}
	_, ok := s.allowedDICOMStores[storeID]
	return ok
}

func (s *Source) UseClientAuthorization() bool {
	return s.useClientOAuth
}
