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

/*
Package classifier provides tools to classify Cypher queries as either read-only or write operations.

It uses a keyword-based and procedure-based approach to determine the query's nature.
The main entry point is the `Classify` method on a `QueryClassifier` object. The classifier
is designed to be conservative, defaulting to classifying unknown procedures as write
operations to ensure safety in read-only environments.

It can handle:
- Standard Cypher keywords (MATCH, CREATE, MERGE, etc.).
- Multi-word keywords (DETACH DELETE, ORDER BY).
- Comments and string literals, which are ignored during classification.
- Procedure calls (CALL db.labels), with predefined lists of known read/write procedures.
- Subqueries (CALL { ... }), checking for write operations within the subquery block.
*/
package classifier

import (
	"regexp"
	"sort"
	"strings"
)

// QueryType represents the classification of a Cypher query as either read or write.
type QueryType int

const (
	// ReadQuery indicates a query that only reads data.
	ReadQuery QueryType = iota
	// WriteQuery indicates a query that modifies data.
	WriteQuery
)

// String provides a human-readable representation of the QueryType.
func (qt QueryType) String() string {
	if qt == ReadQuery {
		return "READ"
	}
	return "WRITE"
}

// QueryClassification represents the detailed result of a query classification.
type QueryClassification struct {
	// Type is the overall classification of the query (READ or WRITE).
	Type QueryType
	// Confidence is a score from 0.0 to 1.0 indicating the classifier's certainty.
	// 1.0 is fully confident. Lower scores may be assigned for ambiguous cases,
	// like unknown procedures.
	Confidence float64
	// WriteTokens is a list of keywords or procedures found that indicate a write operation.
	WriteTokens []string
	// ReadTokens is a list of keywords or procedures found that indicate a read operation.
	ReadTokens []string
	// HasSubquery is true if the query contains a `CALL { ... }` block.
	HasSubquery bool
	// Error holds any error that occurred during classification, though this is not
	// currently used in the implementation.
	Error error
}

// QueryClassifier contains the logic and data for classifying Cypher queries.
// It should be instantiated via the NewQueryClassifier() function.
type QueryClassifier struct {
	writeKeywords map[string]struct{}
	readKeywords  map[string]struct{}
	// writeProcedures is a map of known write procedure prefixes for quick lookup.
	writeProcedures map[string]struct{}
	// readProcedures is a map of known read procedure prefixes for quick lookup.
	readProcedures         map[string]struct{}
	multiWordWriteKeywords []string
	multiWordReadKeywords  []string
	commentPattern         *regexp.Regexp
	stringLiteralPattern   *regexp.Regexp
	procedureCallPattern   *regexp.Regexp
	subqueryPattern        *regexp.Regexp
	whitespacePattern      *regexp.Regexp
	tokenSplitPattern      *regexp.Regexp
}

// NewQueryClassifier creates and initializes a new QueryClassifier instance.
// It pre-compiles regular expressions and populates the internal lists of
// known Cypher keywords and procedures.
func NewQueryClassifier() *QueryClassifier {
	c := &QueryClassifier{
		writeKeywords:        make(map[string]struct{}),
		readKeywords:         make(map[string]struct{}),
		writeProcedures:      make(map[string]struct{}),
		readProcedures:       make(map[string]struct{}),
		commentPattern:       regexp.MustCompile(`(?m)//.*?$|/\*[\s\S]*?\*/`),
		stringLiteralPattern: regexp.MustCompile(`'[^']*'|"[^"]*"`),
		procedureCallPattern: regexp.MustCompile(`(?i)\bCALL\s+([a-zA-Z0-9_.]+)`),
		subqueryPattern:      regexp.MustCompile(`(?i)\bCALL\s*\{`),
		whitespacePattern:    regexp.MustCompile(`\s+`),
		tokenSplitPattern:    regexp.MustCompile(`[\s,(){}[\]]+`),
	}

	// Lists of known keywords that perform write operations.
	writeKeywordsList := []string{
		"CREATE", "MERGE", "DELETE", "DETACH DELETE", "SET", "REMOVE", "FOREACH",
		"CREATE INDEX", "DROP INDEX", "CREATE CONSTRAINT", "DROP CONSTRAINT",
	}
	// Lists of known keywords that perform read operations.
	readKeywordsList := []string{
		"MATCH", "OPTIONAL MATCH", "WITH", "WHERE", "RETURN", "ORDER BY", "SKIP", "LIMIT",
		"UNION", "UNION ALL", "UNWIND", "CASE", "WHEN", "THEN", "ELSE", "END",
		"SHOW", "PROFILE", "EXPLAIN",
	}
	// A list of procedure prefixes known to perform write operations.
	writeProceduresList := []string{
		"apoc.create", "apoc.merge", "apoc.refactor", "apoc.atomic", "apoc.trigger",
		"apoc.periodic.commit", "apoc.load.jdbc", "apoc.load.json", "apoc.load.csv",
		"apoc.export", "apoc.import", "db.create", "db.drop", "db.index.create",
		"db.constraints.create", "dbms.security.create", "gds.graph.create", "gds.graph.drop",
	}
	// A list of procedure prefixes known to perform read operations.
	readProceduresList := []string{
		"apoc.meta", "apoc.help", "apoc.version", "apoc.text", "apoc.math", "apoc.coll",
		"apoc.path", "apoc.algo", "apoc.date", "db.labels", "db.propertyKeys",
		"db.relationshipTypes", "db.schema", "db.indexes", "db.constraints",
		"dbms.components", "dbms.listConfig", "gds.graph.list", "gds.util",
	}

	c.populateKeywords(writeKeywordsList, c.writeKeywords, &c.multiWordWriteKeywords)
	c.populateKeywords(readKeywordsList, c.readKeywords, &c.multiWordReadKeywords)
	c.populateProcedures(writeProceduresList, c.writeProcedures)
	c.populateProcedures(readProceduresList, c.readProcedures)

	return c
}

// populateKeywords processes a list of keyword strings, separating them into
// single-word and multi-word lists for easier processing later.
// Multi-word keywords (e.g., "DETACH DELETE") are sorted by length descending
// to ensure longer matches are replaced first.
func (c *QueryClassifier) populateKeywords(keywords []string, keywordMap map[string]struct{}, multiWord *[]string) {
	for _, kw := range keywords {
		if strings.Contains(kw, " ") {
			*multiWord = append(*multiWord, kw)
		}
		// Replace spaces with underscores for unified tokenization.
		keywordMap[strings.ReplaceAll(kw, " ", "_")] = struct{}{}
	}
	// Sort multi-word keywords by length (longest first) to prevent
	// partial matches, e.g., replacing "CREATE OR REPLACE" before "CREATE".
	sort.SliceStable(*multiWord, func(i, j int) bool {
		return len((*multiWord)[i]) > len((*multiWord)[j])
	})
}

// populateProcedures adds a list of procedure prefixes to the given map.
func (c *QueryClassifier) populateProcedures(procedures []string, procedureMap map[string]struct{}) {
	for _, proc := range procedures {
		procedureMap[strings.ToLower(proc)] = struct{}{}
	}
}

// Classify analyzes a Cypher query string and returns a QueryClassification result.
// It is the main method for this package.
//
// The process is as follows:
// 1. Normalize the query by removing comments and extra whitespace.
// 2. Replace string literals to prevent keywords inside them from being classified.
// 3. Unify multi-word keywords (e.g., "DETACH DELETE" becomes "DETACH_DELETE").
// 4. Extract all procedure calls (e.g., `CALL db.labels`).
// 5. Tokenize the remaining query string.
// 6. Check tokens and procedures against known read/write lists.
// 7. If a subquery `CALL { ... }` exists, check its contents for write operations.
// 8. Assign a final classification and confidence score.
//
// Usage example:
//
//	classifier := NewQueryClassifier()
//	query := "MATCH (n:Person) WHERE n.name = 'Alice' SET n.age = 30"
//	result := classifier.Classify(query)
//	fmt.Printf("Query is a %s query with confidence %f\n", result.Type, result.Confidence)
//	// Output: Query is a WRITE query with confidence 0.900000
//	fmt.Printf("Write tokens found: %v\n", result.WriteTokens)
//	// Output: Write tokens found: [SET]
func (c *QueryClassifier) Classify(query string) QueryClassification {
	result := QueryClassification{
		Type:       ReadQuery, // Default to read, upgrade to write if write tokens are found.
		Confidence: 1.0,
	}

	normalizedQuery := c.normalizeQuery(query)
	if normalizedQuery == "" {
		return result // Return default for empty queries.
	}

	// Early check for subqueries to set the flag.
	result.HasSubquery = c.subqueryPattern.MatchString(normalizedQuery)
	procedures := c.extractProcedureCalls(normalizedQuery)

	// Sanitize the query by replacing string literals to avoid misinterpreting their contents.
	sanitizedQuery := c.stringLiteralPattern.ReplaceAllString(normalizedQuery, "STRING_LITERAL")
	// Unify multi-word keywords to treat them as single tokens.
	unifiedQuery := c.unifyMultiWordKeywords(sanitizedQuery)
	tokens := c.extractTokens(unifiedQuery)

	// Classify based on standard keywords.
	for _, token := range tokens {
		upperToken := strings.ToUpper(token)

		if _, isWrite := c.writeKeywords[upperToken]; isWrite {
			result.WriteTokens = append(result.WriteTokens, upperToken)
			result.Type = WriteQuery
		} else if _, isRead := c.readKeywords[upperToken]; isRead {
			result.ReadTokens = append(result.ReadTokens, upperToken)
		}
	}

	// Classify based on procedure calls.
	for _, proc := range procedures {
		if c.isWriteProcedure(proc) {
			result.WriteTokens = append(result.WriteTokens, "CALL "+proc)
			result.Type = WriteQuery
		} else if c.isReadProcedure(proc) {
			result.ReadTokens = append(result.ReadTokens, "CALL "+proc)
		} else {
			// CONSERVATIVE APPROACH: If a procedure is not in a known list,
			// we guess its type. If it looks like a read (get, list), we treat it as such.
			// Otherwise, we assume it's a write operation with lower confidence.
			if strings.Contains(proc, ".get") || strings.Contains(proc, ".list") ||
				strings.Contains(proc, ".show") || strings.Contains(proc, ".meta") {
				result.ReadTokens = append(result.ReadTokens, "CALL "+proc)
			} else {
				result.WriteTokens = append(result.WriteTokens, "CALL "+proc)
				result.Type = WriteQuery
				result.Confidence = 0.8 // Lower confidence for unknown procedures.
			}
		}
	}

	// If a subquery exists, explicitly check its contents for write operations.
	if result.HasSubquery && c.hasWriteInSubquery(unifiedQuery) {
		result.Type = WriteQuery
		// Add a specific token to indicate the reason for the write classification.
		found := false
		for _, t := range result.WriteTokens {
			if t == "WRITE_IN_SUBQUERY" {
				found = true
				break
			}
		}
		if !found {
			result.WriteTokens = append(result.WriteTokens, "WRITE_IN_SUBQUERY")
		}
	}

	// If a query contains both read and write operations (e.g., MATCH ... DELETE),
	// it's a write query. We lower the confidence slightly to reflect the mixed nature.
	if len(result.WriteTokens) > 0 && len(result.ReadTokens) > 0 {
		result.Confidence = 0.9
	}

	return result
}

// unifyMultiWordKeywords replaces multi-word keywords in a query with a single,
// underscore-separated token. This simplifies the tokenization process.
// Example: "DETACH DELETE" becomes "DETACH_DELETE".
func (c *QueryClassifier) unifyMultiWordKeywords(query string) string {
	upperQuery := strings.ToUpper(query)
	// Combine all multi-word keywords for a single pass.
	allMultiWord := append(c.multiWordWriteKeywords, c.multiWordReadKeywords...)

	for _, kw := range allMultiWord {
		placeholder := strings.ReplaceAll(kw, " ", "_")
		upperQuery = strings.ReplaceAll(upperQuery, kw, placeholder)
	}
	return upperQuery
}

// normalizeQuery cleans a query string by removing comments and collapsing
// all whitespace into single spaces.
func (c *QueryClassifier) normalizeQuery(query string) string {
	// Remove single-line and multi-line comments.
	query = c.commentPattern.ReplaceAllString(query, " ")
	// Collapse consecutive whitespace characters into a single space.
	query = c.whitespacePattern.ReplaceAllString(query, " ")
	return strings.TrimSpace(query)
}

// extractTokens splits a query string into a slice of individual tokens.
// It splits on whitespace and various punctuation marks.
func (c *QueryClassifier) extractTokens(query string) []string {
	tokens := c.tokenSplitPattern.Split(query, -1)
	// Filter out empty strings that can result from the split.
	result := make([]string, 0, len(tokens))
	for _, token := range tokens {
		if token != "" {
			result = append(result, token)
		}
	}
	return result
}

// extractProcedureCalls finds all procedure calls (e.g., `CALL db.labels`)
// in the query and returns a slice of their names.
func (c *QueryClassifier) extractProcedureCalls(query string) []string {
	matches := c.procedureCallPattern.FindAllStringSubmatch(query, -1)
	procedures := make([]string, 0, len(matches))
	for _, match := range matches {
		if len(match) > 1 {
			procedures = append(procedures, strings.ToLower(match[1]))
		}
	}
	return procedures
}

// isWriteProcedure checks if a given procedure name matches any of the known
// write procedure prefixes.
func (c *QueryClassifier) isWriteProcedure(procedure string) bool {
	procedure = strings.ToLower(procedure)
	for wp := range c.writeProcedures {
		if strings.HasPrefix(procedure, wp) {
			return true
		}
	}
	return false
}

// isReadProcedure checks if a given procedure name matches any of the known
// read procedure prefixes.
func (c *QueryClassifier) isReadProcedure(procedure string) bool {
	procedure = strings.ToLower(procedure)
	for rp := range c.readProcedures {
		if strings.HasPrefix(procedure, rp) {
			return true
		}
	}
	return false
}

// hasWriteInSubquery detects if a write keyword exists within a `CALL { ... }` block.
// It correctly handles nested braces to find the content of the top-level subquery.
func (c *QueryClassifier) hasWriteInSubquery(unifiedQuery string) bool {
	loc := c.subqueryPattern.FindStringIndex(unifiedQuery)
	if loc == nil {
		return false
	}

	// The search starts from the beginning of the `CALL {` match.
	subqueryContent := unifiedQuery[loc[0]:]
	openBraces := 0
	startIndex := -1
	endIndex := -1

	// Find the boundaries of the first complete `{...}` block.
	for i, char := range subqueryContent {
		if char == '{' {
			if openBraces == 0 {
				startIndex = i + 1
			}
			openBraces++
		} else if char == '}' {
			openBraces--
			if openBraces == 0 {
				endIndex = i
				break
			}
		}
	}

	var block string
	if startIndex != -1 {
		if endIndex != -1 {
			// A complete `{...}` block was found.
			block = subqueryContent[startIndex:endIndex]
		} else {
			// An opening brace was found but no closing one; this indicates a
			// likely syntax error, but we check the rest of the string anyway.
			block = subqueryContent[startIndex:]
		}

		// Check if any write keyword exists as a whole word within the subquery block.
		for writeOp := range c.writeKeywords {
			// Use regex to match the keyword as a whole word to avoid partial matches
			// (e.g., finding "SET" in "ASSET").
			re := regexp.MustCompile(`\b` + writeOp + `\b`)
			if re.MatchString(block) {
				return true
			}
		}
	}

	return false
}

// AddWriteProcedure allows users to dynamically add a custom procedure prefix to the
// list of known write procedures. This is useful for environments with custom plugins.
// The pattern is matched using `strings.HasPrefix`.
//
// Usage example:
//
//	classifier := NewQueryClassifier()
//	classifier.AddWriteProcedure("my.custom.writer")
//	result := classifier.Classify("CALL my.custom.writer.createUser()")
//	// result.Type will be WriteQuery
func (c *QueryClassifier) AddWriteProcedure(pattern string) {
	if pattern != "" {
		c.writeProcedures[strings.ToLower(pattern)] = struct{}{}
	}
}

// AddReadProcedure allows users to dynamically add a custom procedure prefix to the
// list of known read procedures.
// The pattern is matched using `strings.HasPrefix`.
//
// Usage example:
//
//	classifier := NewQueryClassifier()
//	classifier.AddReadProcedure("my.custom.reader")
//	result := classifier.Classify("CALL my.custom.reader.getData()")
//	// result.Type will be ReadQuery
func (c *QueryClassifier) AddReadProcedure(pattern string) {
	if pattern != "" {
		c.readProcedures[strings.ToLower(pattern)] = struct{}{}
	}
}
