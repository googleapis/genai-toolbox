// Copyright 2025 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"fmt"
	"log"

	"github.com/googleapis/genai-toolbox/internal/util"
)

func main() {
	// Example queries to demonstrate SQL validation
	queries := []string{
		"SELECT id, name FROM users WHERE active = true LIMIT 10",                    // Safe query
		"SELECT * FROM users",                                                        // Query with suggestions
		"SELECT id FROM users WHERE id = 1 OR 1=1",                                  // Suspicious query
		"DROP TABLE users",                                                           // Dangerous query
		"SELECT id FROM users -- This is a comment",                                 // Query with comments
		"SELECT id FROM users UNION SELECT id FROM admins",                          // Query with UNION
		"",                                                                           // Empty query
	}

	fmt.Println("SQL Query Validation Examples")
	fmt.Println("=============================")

	for i, query := range queries {
		fmt.Printf("\n%d. Query: %q\n", i+1, query)
		
		// Validate the query
		result := util.ValidateSQLQuery(query)
		
		// Display results
		if result.IsValid {
			fmt.Println("   ✅ Valid query")
		} else {
			fmt.Println("   ❌ Invalid query")
		}
		
		if len(result.Warnings) > 0 {
			fmt.Println("   ⚠️  Warnings:")
			for _, warning := range result.Warnings {
				fmt.Printf("      - %s\n", warning)
			}
		}
		
		if len(result.Suggestions) > 0 {
			fmt.Println("   💡 Suggestions:")
			for _, suggestion := range result.Suggestions {
				fmt.Printf("      - %s\n", suggestion)
			}
		}
	}

	fmt.Println("\n\nSQL Query Sanitization Examples")
	fmt.Println("=================================")

	// Example queries for sanitization
	sanitizeQueries := []string{
		"  SELECT   id,   name   FROM   users  ",
		"SELECT id FROM users -- This is a comment",
		"SELECT id FROM users /* Block comment */",
		"SELECT    id    FROM    users",
	}

	for i, query := range sanitizeQueries {
		fmt.Printf("\n%d. Original: %q\n", i+1, query)
		sanitized := util.SanitizeSQLQuery(query)
		fmt.Printf("   Sanitized: %q\n", sanitized)
	}

	// Example of integrating validation into a tool
	fmt.Println("\n\nIntegration Example")
	fmt.Println("===================")
	
	userQuery := "SELECT * FROM users WHERE id = 1 OR 1=1"
	fmt.Printf("User query: %q\n", userQuery)
	
	// Validate before execution
	validationResult := util.ValidateSQLQuery(userQuery)
	if !validationResult.IsValid {
		log.Printf("Query rejected due to security concerns:")
		for _, warning := range validationResult.Warnings {
			log.Printf("  - %s", warning)
		}
		return
	}
	
	// If valid, proceed with execution
	fmt.Println("Query is safe to execute")

	// Performance Analysis Examples
	fmt.Println("\n\nSQL Query Performance Analysis Examples")
	fmt.Println("=======================================")

	performanceQueries := []string{
		"SELECT id, name FROM users WHERE active = true LIMIT 10",                    // Well-optimized
		"SELECT * FROM users u JOIN profiles p ON u.id = p.user_id",                 // With JOINs
		"SELECT id FROM users WHERE UPPER(name) = 'JOHN'",                           // Function in WHERE
		"SELECT * FROM users WHERE name LIKE '%john%'",                              // LIKE with wildcard
		"SELECT department, COUNT(*) FROM users GROUP BY department HAVING COUNT(*) > 10", // Complex query
	}

	for i, query := range performanceQueries {
		fmt.Printf("\n%d. Query: %q\n", i+1, query)
		
		// Analyze performance
		analysis := util.AnalyzeQueryPerformance(query)
		
		fmt.Printf("   📊 Complexity Score: %d/10\n", analysis.ComplexityScore)
		fmt.Printf("   💰 Estimated Cost: %s\n", analysis.EstimatedCost)
		
		if len(analysis.Issues) > 0 {
			fmt.Println("   ⚠️  Performance Issues:")
			for _, issue := range analysis.Issues {
				fmt.Printf("      - %s\n", issue)
			}
		}
		
		if len(analysis.Optimizations) > 0 {
			fmt.Println("   🚀 Optimization Suggestions:")
			for _, opt := range analysis.Optimizations {
				fmt.Printf("      - %s\n", opt)
			}
		}
	}
}
