package neo4jdbschema

import (
	"fmt"
	"sort"
	"strings"

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

// ProcessAPOCSchema converts APOC schema to our format
func processAPOCSchema(apocSchema *APOCSchemaResult) ([]NodeLabel, []Relationship, *Statistics) {
	var nodeLabels []NodeLabel
	var relationships []Relationship

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
			stats.PropertiesByLabel[name] = int64(len(entity.Properties)) * entity.Count

			// Find most common relationship patterns
			if len(entity.Relationships) > 0 {
				for relType, relInfo := range entity.Relationships {
					if relInfo.Direction == "out" && len(relInfo.Labels) > 0 {
						// Check if we need to update relationship info
						for i, rel := range relationships {
							if rel.Type == relType {
								// Update with a pattern if this is more common
								if relInfo.Count > 0 {
									relationships[i].StartNode = name
									if len(relInfo.Labels) > 0 {
										relationships[i].EndNode = relInfo.Labels[0]
									}
								}
								break
							}
						}
					}
				}
			}
		} else if entity.Type == "relationship" {
			// Process relationship type
			rel := Relationship{
				Type:       name,
				Count:      entity.Count,
				Properties: []PropertyInfo{}, // Initialize properties
			}

			// Convert properties
			for propName, propInfo := range entity.Properties {
				prop := PropertyInfo{
					Name:    propName,
					Types:   []string{propInfo.Type},
					Unique:  propInfo.Unique,
					Indexed: propInfo.Indexed,
				}
				rel.Properties = append(rel.Properties, prop)
			}

			// Sort properties by name
			sort.Slice(rel.Properties, func(i, j int) bool {
				return rel.Properties[i].Name < rel.Properties[j].Name
			})

			relationships = append(relationships, rel)

			// Update statistics
			stats.RelationshipsByType[name] = entity.Count
			stats.TotalRelationships += entity.Count
			stats.PropertiesByRelType[name] = int64(len(entity.Properties)) * entity.Count
		}
	}

	// Calculate total properties
	for _, count := range stats.PropertiesByLabel {
		stats.TotalProperties += count
	}
	for _, count := range stats.PropertiesByRelType {
		stats.TotalProperties += count
	}

	// Sort by count descending
	sort.Slice(nodeLabels, func(i, j int) bool {
		return nodeLabels[i].Count > nodeLabels[j].Count
	})
	sort.Slice(relationships, func(i, j int) bool {
		return relationships[i].Count > relationships[j].Count
	})

	// Update relationship patterns by checking node relationships
	updateRelationshipPatterns(relationships, apocSchema)

	return nodeLabels, relationships, stats
}

// updateRelationshipPatterns finds common patterns for relationships
func updateRelationshipPatterns(relationships []Relationship, apocSchema *APOCSchemaResult) {
	// Track patterns for each relationship type
	patternCounts := make(map[string]map[string]int64) // relType -> pattern -> count

	for nodeName, entity := range apocSchema.Value {
		if entity.Type != "node" {
			continue
		}

		for relType, relInfo := range entity.Relationships {
			if relInfo.Direction == "out" && len(relInfo.Labels) > 0 && relInfo.Count > 0 {
				pattern := fmt.Sprintf("%s->%s", nodeName, relInfo.Labels[0])

				if patternCounts[relType] == nil {
					patternCounts[relType] = make(map[string]int64)
				}
				patternCounts[relType][pattern] += relInfo.Count
			}
		}
	}

	// Update relationships with the most common pattern
	for i, rel := range relationships {
		if patterns, exists := patternCounts[rel.Type]; exists {
			var maxPattern string
			var maxCount int64

			for pattern, count := range patterns {
				if count > maxCount {
					maxCount = count
					maxPattern = pattern
				}
			}

			if maxPattern != "" {
				parts := strings.Split(maxPattern, "->")
				if len(parts) == 2 {
					relationships[i].StartNode = parts[0]
					relationships[i].EndNode = parts[1]
				}
			}
		}
	}
}
