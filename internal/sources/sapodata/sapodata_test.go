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

package sapodata

import (
	"testing"

	"github.com/goccy/go-yaml"
	"github.com/googleapis/mcp-toolbox/internal/sources"
)

func TestSourceRegistration(t *testing.T) {
	// Verify it registers okay
	yamlDef := []byte(`
name: my_test_sap
type: sap-odata
baseUrl: https://example.com/sap/opu/odata
timeout: 10s
auth:
  type: basic
  username: user
  password: pwd
`)

	var c Config
	if err := yaml.Unmarshal(yamlDef, &c); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if c.Name != "my_test_sap" || c.BaseURL != "https://example.com/sap/opu/odata" {
		t.Errorf("Config unmarshaled incorrectly: %+v", c)
	}
	if c.Auth.Type != "basic" || c.Auth.Username != "user" {
		t.Errorf("Auth unmarshaled incorrectly: %+v", c.Auth)
	}
}

func TestParseMetadata(t *testing.T) {
	xmlData := []byte(`
<edmx:Edmx Version="1.0" xmlns:edmx="http://schemas.microsoft.com/ado/2007/06/edmx" xmlns:m="http://schemas.microsoft.com/ado/2007/08/dataservices/metadata" xmlns:sap="http://www.sap.com/Protocols/SAPData">
    <edmx:DataServices m:DataServiceVersion="2.0">
        <Schema Namespace="API_SALES_ORDER_SRV" xmlns="http://schemas.microsoft.com/ado/2008/09/edm">
            <EntityType Name="A_SalesOrderType" sap:label="Sales Order">
                <Property Name="SalesOrder" Type="Edm.String" Nullable="false" MaxLength="10" sap:label="Sales Order ID" />
                <Property Name="TotalNetAmount" Type="Edm.Decimal" Nullable="true" sap:label="Net Amount" />
                <Property Name="CreationDate" Type="Edm.DateTime" sap:label="Created On" />
            </EntityType>
            <EntityContainer Name="API_SALES_ORDER_SRV_Entities" m:IsDefaultEntityContainer="true">
                <EntitySet Name="A_SalesOrder" EntityType="API_SALES_ORDER_SRV.A_SalesOrderType" />
                <FunctionImport Name="RejectSalesOrder" ReturnType="API_SALES_ORDER_SRV.A_SalesOrderType" HttpMethod="POST">
                    <Parameter Name="SalesOrder" Type="Edm.String" Mode="In" />
                </FunctionImport>
            </EntityContainer>
        </Schema>
    </edmx:DataServices>
</edmx:Edmx>`)

	meta, err := ParseMetadata(xmlData)
	if err != nil {
		t.Fatalf("ParseMetadata failed: %v", err)
	}

	if meta.Version != "2.0" {
		t.Errorf("Expected version 2.0, got %s", meta.Version)
	}

	es, ok := meta.EntitySets["A_SalesOrder"]
	if !ok {
		t.Fatalf("EntitySet A_SalesOrder not found")
	}

	et, err := meta.GetEntityTypeForSet(es.Name)
	if err != nil {
		t.Fatalf("GetEntityTypeForSet failed: %v", err)
	}

	if et.Name != "A_SalesOrderType" {
		t.Errorf("Expected A_SalesOrderType, got %s", et.Name)
	}

	if len(et.Properties) != 3 {
		t.Errorf("Expected 3 properties, got %d", len(et.Properties))
	}

	// Check specific property
	if et.Properties[0].Name != "SalesOrder" || et.Properties[0].Type != "Edm.String" || et.Properties[0].SAPLabel != "Sales Order ID" {
		t.Errorf("Property 0 mapped incorrectly: %+v", et.Properties[0])
	}
	if et.Properties[0].Nullable {
		t.Errorf("Property 0 nullable should be false")
	}

	fi, ok := meta.FunctionImps["RejectSalesOrder"]
	if !ok {
		t.Fatalf("FunctionImport RejectSalesOrder not found")
	}

	if fi.HttpMethod != "POST" || len(fi.Parameters) != 1 || fi.Parameters[0].Name != "SalesOrder" {
		t.Errorf("FunctionImport mapped incorrectly: %+v", fi)
	}
}

// ensure Source interface is fully implemented
var _ sources.Source = &Source{}
