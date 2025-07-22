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

package classifier

import (
	"reflect"
	"sort"
	"testing"
)

// assertElementsMatch checks if two string slices have the same elements, ignoring order.
// It serves as a replacement for testify's assert.ElementsMatch.
func assertElementsMatch(t *testing.T, expected, actual []string, msg string) {
	// t.Helper() marks this function as a test helper.
	// When t.Errorf is called from this function, the line number of the calling code is reported, not the line number inside this helper.
	t.Helper()
	if len(expected) == 0 && len(actual) == 0 {
		return // Both are empty or nil, they match.
	}

	// Create copies to sort, leaving the original slices unmodified.
	expectedCopy := make([]string, len(expected))
	actualCopy := make([]string, len(actual))
	copy(expectedCopy, expected)
	copy(actualCopy, actual)

	sort.Strings(expectedCopy)
	sort.Strings(actualCopy)

	// reflect.DeepEqual provides a robust comparison for complex types, including sorted slices.
	if !reflect.DeepEqual(expectedCopy, actualCopy) {
		t.Errorf("%s: \nexpected: %v\n     got: %v", msg, expected, actual)
	}
}

func TestQueryClassifier_Classify(t *testing.T) {
	classifier := NewQueryClassifier()

	tests := []struct {
		name          string
		query         string
		expectedType  QueryType
		expectedWrite []string
		expectedRead  []string
		minConfidence float64
	}{
		// Read queries
		{
			name:          "simple MATCH query",
			query:         "MATCH (n:Person) RETURN n",
			expectedType:  ReadQuery,
			expectedRead:  []string{"MATCH", "RETURN"},
			expectedWrite: []string{},
			minConfidence: 1.0,
		},
		{
			name:          "complex read query",
			query:         "MATCH (p:Person)-[:KNOWS]->(f) WHERE p.age > 30 RETURN p.name, count(f) ORDER BY p.name SKIP 10 LIMIT 5",
			expectedType:  ReadQuery,
			expectedRead:  []string{"MATCH", "WHERE", "RETURN", "ORDER_BY", "SKIP", "LIMIT"},
			expectedWrite: []string{},
			minConfidence: 1.0,
		},
		{
			name:          "UNION query",
			query:         "MATCH (n:Person) RETURN n.name UNION MATCH (m:Company) RETURN m.name",
			expectedType:  ReadQuery,
			expectedRead:  []string{"MATCH", "RETURN", "UNION", "MATCH", "RETURN"},
			expectedWrite: []string{},
			minConfidence: 1.0,
		},

		// Write queries
		{
			name:          "CREATE query",
			query:         "CREATE (n:Person {name: 'John', age: 30})",
			expectedType:  WriteQuery,
			expectedWrite: []string{"CREATE"},
			expectedRead:  []string{},
			minConfidence: 1.0,
		},
		{
			name:          "MERGE query",
			query:         "MERGE (n:Person {id: 123}) ON CREATE SET n.created = timestamp()",
			expectedType:  WriteQuery,
			expectedWrite: []string{"MERGE", "CREATE", "SET"},
			expectedRead:  []string{},
			minConfidence: 1.0,
		},
		{
			name:          "DETACH DELETE query",
			query:         "MATCH (n:Person) DETACH DELETE n",
			expectedType:  WriteQuery,
			expectedWrite: []string{"DETACH_DELETE"},
			expectedRead:  []string{"MATCH"},
			minConfidence: 0.9,
		},

		// Procedure calls
		{
			name:          "read procedure",
			query:         "CALL db.labels() YIELD label RETURN label",
			expectedType:  ReadQuery,
			expectedRead:  []string{"RETURN", "CALL db.labels"},
			expectedWrite: []string{},
			minConfidence: 1.0,
		},
		{
			name:          "unknown procedure conservative",
			query:         "CALL custom.procedure.doSomething()",
			expectedType:  WriteQuery,
			expectedWrite: []string{"CALL custom.procedure.dosomething"},
			expectedRead:  []string{},
			minConfidence: 0.8,
		},
		{
			name:          "unknown read-like procedure",
			query:         "CALL custom.procedure.getUsers()",
			expectedType:  ReadQuery,
			expectedRead:  []string{"CALL custom.procedure.getusers"},
			expectedWrite: []string{},
			minConfidence: 1.0,
		},

		// Subqueries
		{
			name:          "read subquery",
			query:         "CALL { MATCH (n:Person) RETURN n } RETURN n",
			expectedType:  ReadQuery,
			expectedRead:  []string{"MATCH", "RETURN", "RETURN"},
			expectedWrite: []string{},
			minConfidence: 1.0,
		},
		{
			name:          "write subquery",
			query:         "CALL { CREATE (n:Person) RETURN n } RETURN n",
			expectedType:  WriteQuery,
			expectedWrite: []string{"CREATE", "WRITE_IN_SUBQUERY"},
			expectedRead:  []string{"RETURN", "RETURN"},
			minConfidence: 0.9,
		},

		// Multiline Queries
		{
			name: "multiline read query with comments",
			query: `
				// Find all people and their friends
				MATCH (p:Person)-[:KNOWS]->(f:Friend)
				/*
				  Where the person is older than 25
				*/
				WHERE p.age > 25
				RETURN p.name, f.name
			`,
			expectedType:  ReadQuery,
			expectedWrite: []string{},
			expectedRead:  []string{"MATCH", "WHERE", "RETURN"},
			minConfidence: 1.0,
		},
		{
			name: "multiline write query",
			query: `
				MATCH (p:Person {name: 'Alice'})
				CREATE (c:Company {name: 'Neo4j'})
				CREATE (p)-[:WORKS_FOR]->(c)
			`,
			expectedType:  WriteQuery,
			expectedWrite: []string{"CREATE", "CREATE"},
			expectedRead:  []string{"MATCH"},
			minConfidence: 0.9,
		},

		// Complex Subqueries
		{
			name: "nested read subquery",
			query: `
				CALL {
					MATCH (p:Person)
					RETURN p
				}
				CALL {
					MATCH (c:Company)
					RETURN c
				}
				RETURN p, c
			`,
			expectedType:  ReadQuery,
			expectedWrite: []string{},
			expectedRead:  []string{"MATCH", "RETURN", "MATCH", "RETURN", "RETURN"},
			minConfidence: 1.0,
		},
		{
			name: "subquery with write and outer read",
			query: `
				MATCH (u:User {id: 1})
				CALL {
					WITH u
					CREATE (p:Post {content: 'New post'})
					CREATE (u)-[:AUTHORED]->(p)
					RETURN p
				}
				RETURN u.name, p.content
			`,
			expectedType:  WriteQuery,
			expectedWrite: []string{"CREATE", "CREATE", "WRITE_IN_SUBQUERY"},
			expectedRead:  []string{"MATCH", "WITH", "RETURN", "RETURN"},
			minConfidence: 0.9,
		},
		{
			name: "subquery with read passing to outer write",
			query: `
				CALL {
					MATCH (p:Product {id: 'abc'})
					RETURN p
				}
				WITH p
				SET p.lastViewed = timestamp()
			`,
			expectedType:  WriteQuery,
			expectedWrite: []string{"SET"},
			expectedRead:  []string{"MATCH", "RETURN", "WITH"},
			minConfidence: 0.9,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := classifier.Classify(tt.query)

			if tt.expectedType != result.Type {
				t.Errorf("Query type mismatch: expected %v, got %v", tt.expectedType, result.Type)
			}
			if result.Confidence < tt.minConfidence {
				t.Errorf("Confidence too low: expected at least %f, got %f", tt.minConfidence, result.Confidence)
			}
			assertElementsMatch(t, tt.expectedWrite, result.WriteTokens, "Write tokens mismatch")
			assertElementsMatch(t, tt.expectedRead, result.ReadTokens, "Read tokens mismatch")
		})
	}
}

func TestQueryClassifier_AbuseCases(t *testing.T) {
	classifier := NewQueryClassifier()

	tests := []struct {
		name          string
		query         string
		expectedType  QueryType
		expectedWrite []string
		expectedRead  []string
	}{
		{
			name:          "write keyword in a string literal",
			query:         `MATCH (n) WHERE n.name = 'MERGE (m)' RETURN n`,
			expectedType:  ReadQuery,
			expectedWrite: []string{},
			expectedRead:  []string{"MATCH", "WHERE", "RETURN"},
		},
		{
			name:          "incomplete SET clause",
			query:         `MATCH (n) SET`,
			expectedType:  WriteQuery,
			expectedWrite: []string{"SET"},
			expectedRead:  []string{"MATCH"},
		},
		{
			name:          "keyword as a node label",
			query:         `MATCH (n:CREATE) RETURN n`,
			expectedType:  ReadQuery,
			expectedWrite: []string{}, // 'CREATE' should be seen as an identifier, not a keyword
			expectedRead:  []string{"MATCH", "RETURN"},
		},
		{
			name:          "unbalanced parentheses",
			query:         `MATCH (n:Person RETURN n`,
			expectedType:  ReadQuery,
			expectedWrite: []string{},
			expectedRead:  []string{"MATCH", "RETURN"},
		},
		{
			name:          "unclosed curly brace in subquery",
			query:         `CALL { MATCH (n) CREATE (m)`,
			expectedType:  WriteQuery,
			expectedWrite: []string{"CREATE", "WRITE_IN_SUBQUERY"},
			expectedRead:  []string{"MATCH"},
		},
		{
			name:          "semicolon inside a query part",
			query:         `MATCH (n;Person) RETURN n`,
			expectedType:  ReadQuery,
			expectedWrite: []string{},
			expectedRead:  []string{"MATCH", "RETURN"},
		},
		{
			name:         "jumbled keywords without proper syntax",
			query:        `RETURN CREATE MATCH DELETE`,
			expectedType: WriteQuery,
			// The classifier's job is to find the tokens, not validate the syntax.
			// It should find both read and write tokens.
			expectedWrite: []string{"CREATE", "DELETE"},
			expectedRead:  []string{"RETURN", "MATCH"},
		},
		{
			name: "write in a nested subquery",
			query: `
				CALL {
					MATCH (a)
					CALL {
						CREATE (b:Thing)
					}
					RETURN a
				}
				RETURN "done"
			`,
			expectedType:  WriteQuery,
			expectedWrite: []string{"CREATE", "WRITE_IN_SUBQUERY"},
			expectedRead:  []string{"MATCH", "RETURN", "RETURN"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// This defer-recover block ensures the test fails gracefully if the Classify function panics,
			// which was the goal of the original assert.NotPanics call.
			defer func() {
				if r := recover(); r != nil {
					t.Fatalf("The code panicked on test '%s': %v", tt.name, r)
				}
			}()

			result := classifier.Classify(tt.query)
			if tt.expectedType != result.Type {
				t.Errorf("Query type mismatch: expected %v, got %v", tt.expectedType, result.Type)
			}
			if tt.expectedWrite != nil {
				assertElementsMatch(t, tt.expectedWrite, result.WriteTokens, "Write tokens mismatch")
			}
			if tt.expectedRead != nil {
				assertElementsMatch(t, tt.expectedRead, result.ReadTokens, "Read tokens mismatch")
			}
		})
	}
}

func TestNormalizeQuery(t *testing.T) {
	classifier := NewQueryClassifier()
	t.Run("single line comment", func(t *testing.T) {
		input := "MATCH (n) // comment\nRETURN n"
		expected := "MATCH (n) RETURN n"
		result := classifier.normalizeQuery(input)
		if expected != result {
			t.Errorf("normalizeQuery failed:\nexpected: %q\n     got: %q", expected, result)
		}
	})
}
