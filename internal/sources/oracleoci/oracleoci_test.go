//go:build oracleoci
// +build oracleoci

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

package oracleoci

import (
	"testing"
)

func TestSourceConfigKind(t *testing.T) {
	config := Config{}
	if got := config.SourceConfigKind(); got != "oracle-oci" {
		t.Errorf("SourceConfigKind() = %q, want %q", got, "oracle-oci")
	}
}

func TestSourceKind(t *testing.T) {
	source := Source{Kind: SourceKind}
	if got := source.SourceKind(); got != "oracle-oci" {
		t.Errorf("SourceKind() = %q, want %q", got, "oracle-oci")
	}
}

func TestConfigValidate_TnsAlias(t *testing.T) {
	config := Config{
		Name:     "test",
		TnsAlias: "mydb",
		User:     "admin",
		Password: "pass",
	}

	if err := config.validate(); err != nil {
		t.Errorf("validate() with tnsAlias failed: %v", err)
	}
}

func TestConfigValidate_HostAndServiceName(t *testing.T) {
	config := Config{
		Name:        "test",
		Host:        "localhost",
		Port:        1521,
		ServiceName: "ORCL",
		User:        "admin",
		Password:    "pass",
	}

	if err := config.validate(); err != nil {
		t.Errorf("validate() with host+serviceName failed: %v", err)
	}
}

func TestConfigValidate_ConnectionString(t *testing.T) {
	config := Config{
		Name:             "test",
		ConnectionString: "localhost:1521/ORCL",
		User:             "admin",
		Password:         "pass",
	}

	if err := config.validate(); err != nil {
		t.Errorf("validate() with connectionString failed: %v", err)
	}
}

func TestConfigValidate_NoConnectionMethod(t *testing.T) {
	config := Config{
		Name:     "test",
		User:     "admin",
		Password: "pass",
	}

	err := config.validate()
	if err == nil {
		t.Error("validate() should fail when no connection method provided")
	}
}

func TestConfigValidate_MultipleConnectionMethods(t *testing.T) {
	config := Config{
		Name:        "test",
		TnsAlias:    "mydb",
		Host:        "localhost",
		ServiceName: "ORCL",
		User:        "admin",
		Password:    "pass",
	}

	err := config.validate()
	if err == nil {
		t.Error("validate() should fail when multiple connection methods provided")
	}
}

func TestConfigValidate_HostWithoutServiceName(t *testing.T) {
	config := Config{
		Name:     "test",
		Host:     "localhost",
		User:     "admin",
		Password: "pass",
	}

	err := config.validate()
	if err == nil {
		t.Error("validate() should fail when host provided without serviceName")
	}
}

func TestConfigValidate_ServiceNameWithoutHost(t *testing.T) {
	config := Config{
		Name:        "test",
		ServiceName: "ORCL",
		User:        "admin",
		Password:    "pass",
	}

	err := config.validate()
	if err == nil {
		t.Error("validate() should fail when serviceName provided without host")
	}
}

func TestConfigValidate_EmptyTnsAlias(t *testing.T) {
	config := Config{
		Name:     "test",
		TnsAlias: "   ", // whitespace only
		User:     "admin",
		Password: "pass",
	}

	err := config.validate()
	if err == nil {
		t.Error("validate() should fail when tnsAlias is empty/whitespace")
	}
}

func TestConfigValidate_EmptyHost(t *testing.T) {
	config := Config{
		Name:        "test",
		Host:        "   ", // whitespace only
		ServiceName: "ORCL",
		User:        "admin",
		Password:    "pass",
	}

	err := config.validate()
	if err == nil {
		t.Error("validate() should fail when host is empty/whitespace")
	}
}

func TestConfigValidate_EmptyServiceName(t *testing.T) {
	config := Config{
		Name:        "test",
		Host:        "localhost",
		ServiceName: "   ", // whitespace only
		User:        "admin",
		Password:    "pass",
	}

	err := config.validate()
	if err == nil {
		t.Error("validate() should fail when serviceName is empty/whitespace")
	}
}
