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

package neo4jdbschema

import (
	"fmt"
	"sort"

	"github.com/goccy/go-yaml"
)

func convertToStringSlice(slice []any) []string {
	result := make([]string, len(slice))
	for i, v := range slice {
		result[i] = fmt.Sprintf("%v", v)
	}
	return result
}

func getStringValue(val any) string {
	if val == nil {
		return ""
	}
	return fmt.Sprintf("%v", val)
}

func mapToAPOCSchema(schemaMap map[string]any) (*APOCSchemaResult, error) {
	schemaBytes, err := yaml.Marshal(schemaMap)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal schema map: %w", err)
	}
	var entities map[string]APOCEntity
	if err = yaml.Unmarshal(schemaBytes, &entities); err != nil {
		return nil, fmt.Errorf("failed to unmarshal schema map into entities: %w", err)
	}
	return &APOCSchemaResult{Value: entities}, nil
}

// processAPOCSchema converts APOC schema to our format
func processAPOCSchema(apocSchema *APOCSchemaResult) ([]NodeLabel, *Statistics) {
	var nodeLabels []NodeLabel

	stats := &Statistics{
		NodesByLabel:        make(map[string]int64),
		RelationshipsByType: make(map[string]int64),
		PropertiesByLabel:   make(map[string]int64),
		PropertiesByRelType: make(map[string]int64),
	}

	for name, entity := range apocSchema.Value {
		if entity.Type == "node" {
			// Process node label
			nodeLabel := NodeLabel{
				Name:       name,
				Count:      entity.Count,
				Properties: []PropertyInfo{}, // Initialize properties
			}

			// Convert properties
			for propName, propInfo := range entity.Properties {
				prop := PropertyInfo{
					Name:      propName,
					Types:     []string{propInfo.Type},
					Unique:    propInfo.Unique,
					Indexed:   propInfo.Indexed,
					Mandatory: propInfo.Existence,
				}
				nodeLabel.Properties = append(nodeLabel.Properties, prop)
			}

			// Sort properties by name
			sort.Slice(nodeLabel.Properties, func(i, j int) bool {
				return nodeLabel.Properties[i].Name < nodeLabel.Properties[j].Name
			})

			nodeLabels = append(nodeLabels, nodeLabel)

			// Update statistics
			stats.NodesByLabel[name] = entity.Count
			stats.TotalNodes += entity.Count

			cnt := int64(len(entity.Properties)) * entity.Count
			stats.PropertiesByLabel[name] = cnt
			stats.TotalProperties += cnt
		}
	}

	// Sort by count descending
	sort.Slice(nodeLabels, func(i, j int) bool {
		return nodeLabels[i].Count > nodeLabels[j].Count
	})

	// If maps or lists are empty, set them to nil for cleaner JSON output
	if len(nodeLabels) == 0 {
		nodeLabels = nil
	}
	if len(stats.NodesByLabel) == 0 {
		stats.NodesByLabel = nil
	}
	if len(stats.PropertiesByLabel) == 0 {
		stats.PropertiesByLabel = nil
	}
	return nodeLabels, stats
}

// processNoneAPOCSchema converts non APOC schema data into our format
func processNoneAPOCSchema(
	nodeCounts map[string]int64,
	nodePropsMap map[string]map[string]map[string]bool,
	relCounts map[string]int64,
	relPropsMap map[string]map[string]map[string]bool,
	relConnectivity map[string]struct {
		startNode string
		endNode   string
		count     int64
	},
) ([]NodeLabel, []Relationship, *Statistics) {
	nodeLabels := make([]NodeLabel, 0, len(nodePropsMap))
	stats := &Statistics{
		NodesByLabel:        make(map[string]int64, len(nodeCounts)),
		RelationshipsByType: make(map[string]int64, len(relCounts)),
		PropertiesByLabel:   make(map[string]int64),
		PropertiesByRelType: make(map[string]int64),
	}

	// Process node labels
	processedLabels := make(map[string]bool)

	// First, process labels with properties
	for label, props := range nodePropsMap {
		count := nodeCounts[label]
		properties := make([]PropertyInfo, 0, len(props))

		for key, types := range props {
			typeList := make([]string, 0, len(types))
			for tp := range types {
				typeList = append(typeList, tp)
			}
			sort.Strings(typeList)
			properties = append(properties, PropertyInfo{
				Name:  key,
				Types: typeList,
			})
		}

		sort.Slice(properties, func(i, j int) bool {
			return properties[i].Name < properties[j].Name
		})

		nodeLabels = append(nodeLabels, NodeLabel{
			Name:       label,
			Count:      count,
			Properties: properties,
		})

		stats.NodesByLabel[label] = count
		stats.TotalNodes += count
		stats.PropertiesByLabel[label] = int64(len(properties))
		stats.TotalProperties += int64(len(properties)) * count
		processedLabels[label] = true
	}

	// Then, include labels that have counts but no properties sampled
	for label, count := range nodeCounts {
		if !processedLabels[label] {
			nodeLabels = append(nodeLabels, NodeLabel{
				Name:       label,
				Count:      count,
				Properties: []PropertyInfo{},
			})
			stats.NodesByLabel[label] = count
			stats.TotalNodes += count
		}
	}

	// Sort node labels by count (descending) then by name
	sort.Slice(nodeLabels, func(i, j int) bool {
		if nodeLabels[i].Count != nodeLabels[j].Count {
			return nodeLabels[i].Count > nodeLabels[j].Count
		}
		return nodeLabels[i].Name < nodeLabels[j].Name
	})

	// Process relationships
	relationships := make([]Relationship, 0, len(relCounts))

	for relType, count := range relCounts {
		properties := make([]PropertyInfo, 0)

		if props, exists := relPropsMap[relType]; exists {
			for key, types := range props {
				typeList := make([]string, 0, len(types))
				for tp := range types {
					typeList = append(typeList, tp)
				}
				sort.Strings(typeList)
				properties = append(properties, PropertyInfo{
					Name:  key,
					Types: typeList,
				})
			}

			sort.Slice(properties, func(i, j int) bool {
				return properties[i].Name < properties[j].Name
			})
		}

		conn := relConnectivity[relType]
		relationships = append(relationships, Relationship{
			Type:       relType,
			Count:      count,
			StartNode:  conn.startNode,
			EndNode:    conn.endNode,
			Properties: properties,
		})

		stats.RelationshipsByType[relType] = count
		stats.TotalRelationships += count
		stats.PropertiesByRelType[relType] = int64(len(properties))
		stats.TotalProperties += int64(len(properties)) * count
	}

	// Sort relationships by count (descending) then by type
	sort.Slice(relationships, func(i, j int) bool {
		if relationships[i].Count != relationships[j].Count {
			return relationships[i].Count > relationships[j].Count
		}
		return relationships[i].Type < relationships[j].Type
	})

	// If maps or lists are empty, set them to nil for cleaner JSON output
	if len(nodeLabels) == 0 {
		nodeLabels = nil
	}
	if len(stats.NodesByLabel) == 0 {
		stats.NodesByLabel = nil
	}
	if len(stats.RelationshipsByType) == 0 {
		stats.RelationshipsByType = nil
	}
	if len(stats.PropertiesByLabel) == 0 {
		stats.PropertiesByLabel = nil
	}
	if len(stats.PropertiesByRelType) == 0 {
		stats.PropertiesByRelType = nil
	}

	return nodeLabels, relationships, stats
}
