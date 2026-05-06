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

package odata

import (
	"encoding/xml"
	"fmt"
	"strings"
)

// ODataMetadata represents the parsed $metadata XML document.
type ODataMetadata struct {
	Version      string
	EntityTypes  map[string]EntityType
	EntitySets   map[string]EntitySet // Maps EntitySet Name to EntityType Name
	FunctionImps map[string]FunctionImport
}

type EntityType struct {
	Name       string
	Properties []Property
}

type Property struct {
	Name      string
	Type      string
	Nullable  bool
	MaxLength string
	SAPLabel  string
}

type EntitySet struct {
	Name       string
	EntityType string // e.g., "API_SALES_ORDER_SRV.A_SalesOrderType"
}

type FunctionImport struct {
	Name       string
	HttpMethod string
	ReturnType string
	Parameters []FunctionParameter
}

type FunctionParameter struct {
	Name string
	Type string
	Mode string
}

// Edmx structure to easily marshal standard OData V2 and V4 schemas
type edmxRoot struct {
	XMLName      xml.Name     `xml:"Edmx"`
	Version      string       `xml:"Version,attr"`
	DataServices dataServices `xml:"DataServices"`
}

type dataServices struct {
	DataServiceVersion string   `xml:"DataServiceVersion,attr"`
	Schemas            []schema `xml:"Schema"`
}

type schema struct {
	Namespace       string             `xml:"Namespace,attr"`
	EntityTypes     []xmlEntityType    `xml:"EntityType"`
	EntityContainer xmlEntityContainer `xml:"EntityContainer"`
}

type xmlEntityType struct {
	Name       string        `xml:"Name,attr"`
	Properties []xmlProperty `xml:"Property"`
}

type xmlProperty struct {
	Name      string `xml:"Name,attr"`
	Type      string `xml:"Type,attr"`
	Nullable  string `xml:"Nullable,attr"`
	MaxLength string `xml:"MaxLength,attr"`
	Label     string `xml:"label,attr"` // sap:label
}

type xmlEntityContainer struct {
	Name            string              `xml:"Name,attr"`
	EntitySets      []xmlEntitySet      `xml:"EntitySet"`
	FunctionImports []xmlFunctionImport `xml:"FunctionImport"`
}

type xmlEntitySet struct {
	Name       string `xml:"Name,attr"`
	EntityType string `xml:"EntityType,attr"`
}

type xmlFunctionImport struct {
	Name       string                 `xml:"Name,attr"`
	HttpMethod string                 `xml:"HttpMethod,attr"`
	ReturnType string                 `xml:"ReturnType,attr"`
	Parameters []xmlFunctionParameter `xml:"Parameter"`
}

type xmlFunctionParameter struct {
	Name string `xml:"Name,attr"`
	Type string `xml:"Type,attr"`
	Mode string `xml:"Mode,attr"`
}

// ParseMetadata parses an OData $metadata XML byte slice and extracts version, entities, and functions.
func ParseMetadata(data []byte) (*ODataMetadata, error) {
	var root edmxRoot
	err := xml.Unmarshal(data, &root)
	if err != nil {
		return nil, fmt.Errorf("failed to parse OData XML: %w", err)
	}

	meta := &ODataMetadata{
		EntityTypes:  make(map[string]EntityType),
		EntitySets:   make(map[string]EntitySet),
		FunctionImps: make(map[string]FunctionImport),
	}

	// 1. Determine Version (V2 vs V4)
	if root.DataServices.DataServiceVersion == "2.0" || strings.HasPrefix(root.DataServices.DataServiceVersion, "2.") {
		meta.Version = "2.0"
	} else if root.Version == "4.0" || root.DataServices.DataServiceVersion == "4.0" {
		meta.Version = "4.0"
	} else {
		// Default to 2.0 if ambiguous but usually SAP provides it here.
		meta.Version = "2.0"
	}

	// 2. Extract Types and Functions
	for _, sch := range root.DataServices.Schemas {
		// Populate EntityTypes
		for _, et := range sch.EntityTypes {
			t := EntityType{
				Name:       et.Name,
				Properties: make([]Property, 0, len(et.Properties)),
			}
			for _, prop := range et.Properties {
				nullable := true
				if strings.ToLower(prop.Nullable) == "false" {
					nullable = false
				}
				t.Properties = append(t.Properties, Property{
					Name:      prop.Name,
					Type:      prop.Type,
					Nullable:  nullable,
					MaxLength: prop.MaxLength,
					SAPLabel:  prop.Label,
				})
			}
			meta.EntityTypes[et.Name] = t
		}

		// Populate EntitySets & FunctionImports
		for _, es := range sch.EntityContainer.EntitySets {
			meta.EntitySets[es.Name] = EntitySet{
				Name:       es.Name,
				EntityType: es.EntityType,
			}
		}

		for _, fi := range sch.EntityContainer.FunctionImports {
			f := FunctionImport{
				Name:       fi.Name,
				HttpMethod: fi.HttpMethod,
				ReturnType: fi.ReturnType,
				Parameters: make([]FunctionParameter, 0, len(fi.Parameters)),
			}
			for _, p := range fi.Parameters {
				f.Parameters = append(f.Parameters, FunctionParameter{
					Name: p.Name,
					Type: p.Type,
					Mode: p.Mode,
				})
			}
			meta.FunctionImps[fi.Name] = f
		}
	}

	return meta, nil
}

// GetEntityTypeForSet returns the EntityType definition for a given EntitySet name.
func (m *ODataMetadata) GetEntityTypeForSet(setName string) (*EntityType, error) {
	es, ok := m.EntitySets[setName]
	if !ok {
		// Sometimes people specify the type directly instead of the set
		if et, ok := m.EntityTypes[setName]; ok {
			return &et, nil
		}
		return nil, fmt.Errorf("EntitySet %q not found in metadata", setName)
	}

	// EntityType is usually namespaced, e.g. "API_SALES_ORDER_SRV.A_SalesOrderType"
	parts := strings.Split(es.EntityType, ".")
	typeName := parts[len(parts)-1]

	et, ok := m.EntityTypes[typeName]
	if !ok {
		return nil, fmt.Errorf("EntityType %q for Set %q not found in metadata", typeName, setName)
	}
	return &et, nil
}
