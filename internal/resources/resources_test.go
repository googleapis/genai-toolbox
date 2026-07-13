// Copyright 2026 Google LLC
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

package resources

import (
	"bytes"
	"context"
	"errors"
	"testing"

	"github.com/goccy/go-yaml"
	"github.com/googleapis/mcp-toolbox/internal/util"
)

type mockResourceConfig struct {
	BaseConfig `yaml:",inline"`
	CustomProp string `yaml:"customProp"`
}

func (m mockResourceConfig) ResourceConfigType() string {
	return "mock"
}

func (m mockResourceConfig) Initialize(ctx context.Context) (Resource, error) {
	return nil, nil
}

func mockFactory(ctx context.Context, name string, decoder *yaml.Decoder) (ResourceConfig, error) {
	cfg := mockResourceConfig{}
	cfg.Name = name
	cfg.Type = "mock"
	if err := decoder.DecodeContext(ctx, &cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}

func mockFailingFactory(ctx context.Context, name string, decoder *yaml.Decoder) (ResourceConfig, error) {
	return nil, errors.New("factory error")
}

func TestRegister(t *testing.T) {
	registry = make(map[string]ResourceConfigFactory) // reset registry

	if !Register("mock", mockFactory) {
		t.Errorf("Expected Register to return true for new type")
	}

	if Register("mock", mockFactory) {
		t.Errorf("Expected Register to return false for duplicate type")
	}
}

func TestDecodeConfig(t *testing.T) {
	registry = make(map[string]ResourceConfigFactory) // reset registry
	Register("mock", mockFactory)
	Register("failing", mockFailingFactory)

	t.Run("Success", func(t *testing.T) {
		yamlBytes := []byte("customProp: value\nuri: mock://test")
		decoder := yaml.NewDecoder(bytes.NewReader(yamlBytes))
		ctx := context.Background()
		cfg, err := DecodeConfig(ctx, "mock", "testMock", decoder)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		mockCfg, ok := cfg.(mockResourceConfig)
		if !ok {
			t.Fatalf("Expected mockResourceConfig, got %T", cfg)
		}

		if mockCfg.Name != "testMock" {
			t.Errorf("Expected Name 'testMock', got %q", mockCfg.Name)
		}
		if mockCfg.Type != "mock" {
			t.Errorf("Expected Type 'mock', got %q", mockCfg.Type)
		}
		if mockCfg.URI != "mock://test" {
			t.Errorf("Expected URI 'mock://test', got %q", mockCfg.URI)
		}
		if mockCfg.CustomProp != "value" {
			t.Errorf("Expected CustomProp 'value', got %q", mockCfg.CustomProp)
		}
	})

	t.Run("UnknownType", func(t *testing.T) {
		yamlBytes := []byte("customProp: value\nuri: mock://test")
		decoder := yaml.NewDecoder(bytes.NewReader(yamlBytes))
		ctx := context.Background()
		_, err := DecodeConfig(ctx, "unknown", "test", decoder)
		if err == nil {
			t.Fatalf("Expected error for unknown type, got nil")
		}
	})

	t.Run("FactoryError", func(t *testing.T) {
		yamlBytes := []byte("customProp: value\nuri: mock://test")
		decoder := yaml.NewDecoder(bytes.NewReader(yamlBytes))
		ctx := context.Background()
		_, err := DecodeConfig(ctx, "failing", "test", decoder)
		if err == nil {
			t.Fatalf("Expected error from failing factory, got nil")
		}
	})
}

func TestGetBaseDirFromContext(t *testing.T) {
	ctx := context.Background()

	t.Run("EmptyContext", func(t *testing.T) {
		if GetBaseDirFromContext(ctx) != "" {
			t.Errorf("Expected empty string for empty context")
		}
	})

	t.Run("NilContext", func(t *testing.T) {
		//lint:ignore SA1012 intentional nil context test
		if GetBaseDirFromContext(nil) != "" {
			t.Errorf("Expected empty string for nil context")
		}
	})

	t.Run("ValidString", func(t *testing.T) {
		ctxWithDir := context.WithValue(ctx, BaseDirKey, "/test/dir")
		if GetBaseDirFromContext(ctxWithDir) != "/test/dir" {
			t.Errorf("Expected '/test/dir', got %q", GetBaseDirFromContext(ctxWithDir))
		}
	})

	t.Run("InvalidType", func(t *testing.T) {
		ctxWithInt := context.WithValue(ctx, BaseDirKey, 12345)
		if GetBaseDirFromContext(ctxWithInt) != "" {
			t.Errorf("Expected empty string when base dir is not a string type")
		}
	})
}

func TestBaseConfig_YAML(t *testing.T) {
	yamlStr := `
name: testName
type: testType
uri: file:///test
description: A test description
title: A Test Title
mimeType: text/plain
annotations:
  key1: value1
`
	var cfg BaseConfig
	if err := yaml.Unmarshal([]byte(yamlStr), &cfg); err != nil {
		t.Fatalf("Failed to unmarshal BaseConfig: %v", err)
	}

	if cfg.Name != "testName" {
		t.Errorf("Expected Name 'testName', got %q", cfg.Name)
	}
	if cfg.Type != "testType" {
		t.Errorf("Expected Type 'testType', got %q", cfg.Type)
	}
	if cfg.URI != "file:///test" {
		t.Errorf("Expected URI 'file:///test', got %q", cfg.URI)
	}
	if cfg.Description != "A test description" {
		t.Errorf("Expected Description 'A test description', got %q", cfg.Description)
	}
	if cfg.Title != "A Test Title" {
		t.Errorf("Expected Title 'A Test Title', got %q", cfg.Title)
	}
	if cfg.MimeType != "text/plain" {
		t.Errorf("Expected MimeType 'text/plain', got %q", cfg.MimeType)
	}
	if val, ok := cfg.Annotations["key1"].(string); !ok || val != "value1" {
		t.Errorf("Expected annotation key1=value1, got %v", cfg.Annotations["key1"])
	}
}

func TestStrictDecoding_Error(t *testing.T) {
	raw := map[string]any{
		"name":               "testResource",
		"type":               "mock",
		"invalidRandomField": true, // This should trigger the strict decoding error
	}

	decoder, err := util.NewStrictDecoder(raw)
	if err != nil {
		t.Fatalf("Failed to create strict decoder: %v", err)
	}

	ctx := context.Background()
	registry = make(map[string]ResourceConfigFactory)
	Register("mock", mockFactory)

	_, err = DecodeConfig(ctx, "mock", "testResource", decoder)
	if err == nil {
		t.Fatalf("Expected DecodeConfig to return an error for an unknown field 'invalidRandomField', but got nil")
	}
}
