package main

import (
	"encoding/json"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"log"
	"os"
	"path/filepath"
	"reflect"
	"strings"
)

const (
	sourcesDir = "internal/sources"
	outputFile = "internal/server/static/data/source_templates.json"
)

func main() {
	log.SetFlags(0)

	results := make(map[string][]string)

	absSourcesDir, err := filepath.Abs(sourcesDir)
	if err != nil {
		log.Fatalf("Failed to get absolute path for sources_dir: %v", err)
	}

	entries, err := os.ReadDir(absSourcesDir)
	if err != nil {
		log.Fatalf("Error reading sources directory %s: %v", absSourcesDir, err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			sourceName := entry.Name()
			goFilePath := filepath.Join(absSourcesDir, sourceName, sourceName+".go")

			if _, err := os.Stat(goFilePath); err == nil {
				log.Printf("Processing: %s", goFilePath)
				fields, err := parseConfigFields(goFilePath)
				if err != nil {
					log.Printf("Error parsing %s: %v", goFilePath, err)
					continue
				}
				if len(fields) > 0 {
					results[sourceName] = fields
				} else {
					log.Printf("Warning: No relevant fields found in 'Config' struct in %s", goFilePath)
				}
			} else if os.IsNotExist(err) {
				// This is fine, not all subdirs might follow the pattern
			} else {
				log.Printf("Error stating file %s: %v", goFilePath, err)
			}
		}
	}

	if len(results) == 0 {
		log.Printf("No source configs found matching the pattern in %s", absSourcesDir)
	}

	jsonData, err := json.MarshalIndent(results, "", "  ")
	if err != nil {
		log.Fatalf("Error marshaling JSON: %v", err)
	}

	absOutputFile, err := filepath.Abs(outputFile)
	if err != nil {
		log.Fatalf("Failed to get absolute path for output_file: %v", err)
	}

	// Ensure the output directory exists
	outputDir := filepath.Dir(absOutputFile)
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		log.Fatalf("Error creating output directory %s: %v", outputDir, err)
	}

	err = os.WriteFile(absOutputFile, jsonData, 0644)
	if err != nil {
		log.Fatalf("Error writing output file %s: %v", absOutputFile, err)
	}

	fmt.Printf("Successfully generated %s\n", absOutputFile)
}

func parseConfigFields(filename string) ([]string, error) {
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, filename, nil, 0)
	if err != nil {
		return nil, err
	}

	var fields []string
	foundConfig := false

	ast.Inspect(node, func(n ast.Node) bool {
		if foundConfig {
			return false
		}
		typeSpec, ok := n.(*ast.TypeSpec)
		if !ok || typeSpec.Name.Name != "Config" {
			return true
		}

		structType, ok := typeSpec.Type.(*ast.StructType)
		if !ok {
			return true
		}
		foundConfig = true // Mark as found

		for _, field := range structType.Fields.List {
			if len(field.Names) == 0 {
				continue
			}
			fieldName := field.Names[0].Name
			if fieldName == "Name" {
				continue
			}

			yamlTagName := ""
			if field.Tag != nil {
				tagVal := strings.Trim(field.Tag.Value, "`")
				tags := reflect.StructTag(tagVal)
				if yamlTag, ok := tags.Lookup("yaml"); ok {
					yamlTagName = strings.Split(yamlTag, ",")[0]
				}
			}

			if yamlTagName != "" && yamlTagName != "-" {
				fields = append(fields, yamlTagName)
			}
		}
		return false
	})

	return fields, nil
}
