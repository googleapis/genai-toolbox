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
package util

import (
	"testing"
)

func TestValidateSQLQuery(t *testing.T) {
	tests := []struct {
		name     string
		query    string
		expected SQLValidationResult
	}{
		{
			name:  "Valid SELECT query",
			query: "SELECT id, name FROM users WHERE id = 1",
			expected: SQLValidationResult{
				IsValid:     true,
				Warnings:    []string{},
				Suggestions: []string{"Consider adding a LIMIT clause to prevent large result sets"},
			},
		},
		{
			name:  "Empty query",
			query: "",
			expected: SQLValidationResult{
				IsValid:  false,
				Warnings: []string{"Query is empty"},
			},
		},
		{
			name:  "Query with DROP statement",
			query: "DROP TABLE users",
			expected: SQLValidationResult{
				IsValid:  false,
				Warnings: []string{"Query contains DROP statement"},
			},
		},
		{
			name:  "Query with DELETE statement",
			query: "DELETE FROM users WHERE id = 1",
			expected: SQLValidationResult{
				IsValid:  false,
				Warnings: []string{"Query contains DELETE statement"},
			},
		},
		{
			name:  "Query with suspicious OR condition",
			query: "SELECT * FROM users WHERE id = 1 OR 1=1",
			expected: SQLValidationResult{
				IsValid:  false,
				Warnings: []string{"Query contains suspicious OR condition"},
			},
		},
		{
			name:  "Query with SELECT *",
			query: "SELECT * FROM users",
			expected: SQLValidationResult{
				IsValid:     true,
				Warnings:    []string{},
				Suggestions: []string{"Consider specifying column names instead of using SELECT *", "Consider adding a WHERE clause to limit the result set", "Consider adding a LIMIT clause to prevent large result sets"},
			},
		},
		{
			name:  "Query with comments",
			query: "SELECT * FROM users -- This is a comment",
			expected: SQLValidationResult{
				IsValid:  false,
				Warnings: []string{"Query contains SQL comments"},
			},
		},
		{
			name:  "Query with UNION",
			query: "SELECT id FROM users UNION SELECT id FROM admins",
			expected: SQLValidationResult{
				IsValid:  false,
				Warnings: []string{"Query contains UNION statement"},
			},
		},
		{
			name:  "Safe query with LIMIT",
			query: "SELECT id, name FROM users WHERE active = true LIMIT 10",
			expected: SQLValidationResult{
				IsValid:     true,
				Warnings:    []string{},
				Suggestions: []string{},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ValidateSQLQuery(tt.query)
			
			if result.IsValid != tt.expected.IsValid {
				t.Errorf("ValidateSQLQuery() IsValid = %v, want %v", result.IsValid, tt.expected.IsValid)
			}
			
			if len(result.Warnings) != len(tt.expected.Warnings) {
				t.Errorf("ValidateSQLQuery() Warnings length = %d, want %d", len(result.Warnings), len(tt.expected.Warnings))
			}
			
			if len(result.Suggestions) != len(tt.expected.Suggestions) {
				t.Errorf("ValidateSQLQuery() Suggestions length = %d, want %d", len(result.Suggestions), len(tt.expected.Suggestions))
			}
		})
	}
}

func TestSanitizeSQLQuery(t *testing.T) {
	tests := []struct {
		name     string
		query    string
		expected string
	}{
		{
			name:     "Remove extra whitespace",
			query:    "  SELECT   id,   name   FROM   users  ",
			expected: "SELECT id, name FROM users",
		},
		{
			name:     "Remove single line comments",
			query:    "SELECT id FROM users -- This is a comment",
			expected: "SELECT id FROM users",
		},
		{
			name:     "Remove block comments",
			query:    "SELECT id FROM users /* This is a block comment */",
			expected: "SELECT id FROM users",
		},
		{
			name:     "Remove multiple consecutive spaces",
			query:    "SELECT    id    FROM    users",
			expected: "SELECT id FROM users",
		},
		{
			name:     "Handle empty query",
			query:    "",
			expected: "",
		},
		{
			name:     "Handle query with only whitespace",
			query:    "   ",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SanitizeSQLQuery(tt.query)
			if result != tt.expected {
				t.Errorf("SanitizeSQLQuery() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestAnalyzeQueryPerformance(t *testing.T) {
	tests := []struct {
		name     string
		query    string
		expected QueryPerformanceAnalysis
	}{
		{
			name:  "Simple SELECT query",
			query: "SELECT id, name FROM users WHERE active = true",
			expected: QueryPerformanceAnalysis{
				ComplexityScore: 1,
				Issues:          []string{},
				Optimizations:   []string{"Consider adding a LIMIT clause to prevent large result sets"},
				EstimatedCost:   "Low",
			},
		},
		{
			name:  "Query with JOINs",
			query: "SELECT u.id, u.name, p.title FROM users u JOIN profiles p ON u.id = p.user_id",
			expected: QueryPerformanceAnalysis{
				ComplexityScore: 3, // 1 base + 2 for JOIN
				Issues:          []string{},
				Optimizations:   []string{"Consider using appropriate indexes for JOIN columns", "Consider adding a LIMIT clause to prevent large result sets"},
				EstimatedCost:   "Low",
			},
		},
		{
			name:  "Query with LIKE wildcard",
			query: "SELECT * FROM users WHERE name LIKE '%john%'",
			expected: QueryPerformanceAnalysis{
				ComplexityScore: 3, // 1 base + 1 for SELECT * + 2 for LIKE wildcard
				Issues:          []string{"LIKE queries with leading wildcards cannot use indexes efficiently"},
				Optimizations:   []string{"SELECT * can impact performance - specify only needed columns", "Consider full-text search or different query pattern", "Consider adding a LIMIT clause to prevent large result sets"},
				EstimatedCost:   "Low",
			},
		},
		{
			name:  "Query with functions in WHERE",
			query: "SELECT id FROM users WHERE UPPER(name) = 'JOHN'",
			expected: QueryPerformanceAnalysis{
				ComplexityScore: 3, // 1 base + 2 for function in WHERE
				Issues:          []string{"Functions in WHERE clause prevent index usage"},
				Optimizations:   []string{"Consider restructuring to avoid functions in WHERE clause", "Consider adding a LIMIT clause to prevent large result sets"},
				EstimatedCost:   "Low",
			},
		},
		{
			name:  "Query with multiple OR conditions",
			query: "SELECT id FROM users WHERE status = 'active' OR status = 'pending' OR status = 'verified' OR status = 'approved'",
			expected: QueryPerformanceAnalysis{
				ComplexityScore: 5, // 1 base + 4 for OR conditions
				Issues:          []string{},
				Optimizations:   []string{"Multiple OR conditions can be slow - consider UNION or different approach", "Consider adding a LIMIT clause to prevent large result sets"},
				EstimatedCost:   "Medium",
			},
		},
		{
			name:  "Query with GROUP BY and HAVING",
			query: "SELECT department, COUNT(*) FROM users GROUP BY department HAVING COUNT(*) > 10",
			expected: QueryPerformanceAnalysis{
				ComplexityScore: 5, // 1 base + 2 for GROUP BY + 2 for HAVING
				Issues:          []string{},
				Optimizations:   []string{"Consider adding indexes on GROUP BY columns", "HAVING clauses can be expensive - consider filtering in WHERE instead", "Consider adding a LIMIT clause to prevent large result sets"},
				EstimatedCost:   "Medium",
			},
		},
		{
			name:  "Empty query",
			query: "",
			expected: QueryPerformanceAnalysis{
				ComplexityScore: 1,
				Issues:          []string{"Query is empty"},
				Optimizations:   []string{},
				EstimatedCost:   "Low",
			},
		},
		{
			name:  "Well-optimized query",
			query: "SELECT id, name FROM users WHERE active = true AND created_at > '2023-01-01' ORDER BY name LIMIT 100",
			expected: QueryPerformanceAnalysis{
				ComplexityScore: 2, // 1 base + 1 for ORDER BY
				Issues:          []string{},
				Optimizations:   []string{"Ensure ORDER BY columns are indexed"},
				EstimatedCost:   "Low",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := AnalyzeQueryPerformance(tt.query)
			
			if result.ComplexityScore != tt.expected.ComplexityScore {
				t.Errorf("AnalyzeQueryPerformance() ComplexityScore = %d, want %d", result.ComplexityScore, tt.expected.ComplexityScore)
			}
			
			if result.EstimatedCost != tt.expected.EstimatedCost {
				t.Errorf("AnalyzeQueryPerformance() EstimatedCost = %s, want %s", result.EstimatedCost, tt.expected.EstimatedCost)
			}
			
			// Check that we have the expected number of issues and optimizations
			if len(result.Issues) != len(tt.expected.Issues) {
				t.Errorf("AnalyzeQueryPerformance() Issues length = %d, want %d", len(result.Issues), len(tt.expected.Issues))
			}
			
			if len(result.Optimizations) != len(tt.expected.Optimizations) {
				t.Errorf("AnalyzeQueryPerformance() Optimizations length = %d, want %d", len(result.Optimizations), len(tt.expected.Optimizations))
			}
		})
	}
}
