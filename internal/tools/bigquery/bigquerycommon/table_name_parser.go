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

package bigquerycommon

import (
	"fmt"
	"strings"
	"unicode"
)

// parserState defines the state of the SQL parser's state machine.
type parserState int

const (
	stateNormal parserState = iota
	// String states
	stateInSingleQuoteString
	stateInDoubleQuoteString
	stateInTripleSingleQuoteString
	stateInTripleDoubleQuoteString
	stateInRawSingleQuoteString
	stateInRawDoubleQuoteString
	stateInRawTripleSingleQuoteString
	stateInRawTripleDoubleQuoteString
	// Comment states
	stateInSingleLineCommentDash
	stateInSingleLineCommentHash
	stateInMultiLineComment
)

// SQL statement verbs
const (
	verbCreate = "create"
	verbAlter  = "alter"
	verbDrop   = "drop"
	verbSelect = "select"
	verbInsert = "insert"
	verbUpdate = "update"
	verbDelete = "delete"
	verbMerge  = "merge"
)

var tableFollowsKeywords = map[string]bool{
	"from":   true,
	"join":   true,
	"update": true,
	"into":   true, // INSERT INTO, MERGE INTO
	"table":  true, // CREATE TABLE, ALTER TABLE
	"model":  true, // ML.GET_INSIGHTS(MODEL ...)
	"view":   true, // DROP VIEW ...
	"using":  true, // MERGE ... USING
	"insert": true, // INSERT my_table
	"merge":  true, // MERGE my_table
}

var tableContextExitKeywords = map[string]bool{
	"where":     true,
	"group":     true, // GROUP BY
	"order":     true, // ORDER BY
	"having":    true,
	"limit":     true,
	"window":    true,
	"union":     true,
	"intersect": true,
	"except":    true,
	"on":        true, // JOIN ... ON
	"set":       true, // UPDATE ... SET
	"when":      true, // MERGE ... WHEN
}

var sqlStatementVerbs = map[string]bool{
	verbCreate: true,
	verbAlter:  true,
	verbDrop:   true,
	verbSelect: true,
	verbInsert: true,
	verbUpdate: true,
	verbDelete: true,
	verbMerge:  true,
}

var schemaOperationVerbs = map[string]bool{
	verbCreate: true,
	verbAlter:  true,
	verbDrop:   true,
}

// hasPrefix checks if the runes starting at offset match the given prefix.
func hasPrefix(r []rune, offset int, prefix string) bool {
	if offset+len(prefix) > len(r) {
		return false
	}
	for i := 0; i < len(prefix); i++ {
		if r[offset+i] != rune(prefix[i]) {
			return false
		}
	}
	return true
}

// hasPrefixFold checks if the runes starting at offset match the given prefix, ignoring case (ASCII only).
func hasPrefixFold(r []rune, offset int, prefix string) bool {
	if offset+len(prefix) > len(r) {
		return false
	}
	for i := 0; i < len(prefix); i++ {
		rChar := r[offset+i]
		pChar := rune(prefix[i])
		if rChar >= 'A' && rChar <= 'Z' {
			rChar += 32
		}
		if pChar >= 'A' && pChar <= 'Z' {
			pChar += 32
		}
		if rChar != pChar {
			return false
		}
	}
	return true
}

// TableParser parses a SQL query and returns a list of table IDs that it references.
// It is intended as a conservative fallback for when a dry run cannot be performed or analyzed.
func TableParser(sql, defaultProjectID string) ([]string, error) {
	tableIDSet := make(map[string]struct{})
	visitedSQLs := make(map[string]struct{})
	aliases := make(map[string]struct{})
	if _, err := parseSQL(sql, defaultProjectID, tableIDSet, visitedSQLs, aliases, false); err != nil {
		return nil, err
	}

	tableIDs := make([]string, 0, len(tableIDSet))
	for id := range tableIDSet {
		isAlias := false
		parts := strings.Split(id, ".")
		for j := 0; j < len(parts); j++ {
			suffix := strings.ToLower(strings.Join(parts[j:], "."))
			if _, ok := aliases[suffix]; ok {
				isAlias = true
				break
			}
		}
		if !isAlias {
			tableIDs = append(tableIDs, id)
		}
	}
	return tableIDs, nil
}

// parseSQL is the core recursive function that processes SQL strings.
// It uses a state machine to find table names and recursively parse EXECUTE IMMEDIATE.
func parseSQL(sql, defaultProjectID string, tableIDSet map[string]struct{}, visitedSQLs map[string]struct{}, aliases map[string]struct{}, inSubquery bool) (int, error) {
	// Prevent infinite recursion.
	if _, ok := visitedSQLs[sql]; ok {
		return len(sql), nil
	}
	visitedSQLs[sql] = struct{}{}

	state := stateNormal
	expectingTable, expectingAlias, expectingCTE := false, false, false
	var lastTableKeyword, lastToken, statementVerb string
	runes := []rune(sql)

	for i := 0; i < len(runes); {
		char := runes[i]

		switch state {
		case stateNormal:
			if hasPrefix(runes, i, "--") {
				state = stateInSingleLineCommentDash
				i += 2
				continue
			}
			if char == '#' {
				state = stateInSingleLineCommentHash
				i++
				continue
			}
			if hasPrefix(runes, i, "/*") {
				state = stateInMultiLineComment
				i += 2
				continue
			}
			if char == ',' {
				if lastTableKeyword == "from" {
					expectingTable = true
					expectingAlias = false
				} else if statementVerb == "with" {
					expectingCTE = true
					expectingAlias = false
				}
				i++
				continue
			}
			if char == '(' {
				if expectingTable || expectingCTE || lastToken == "as" {
					consumed, err := parseSQL(string(runes[i+1:]), defaultProjectID, tableIDSet, visitedSQLs, aliases, true)
					if err != nil {
						return 0, err
					}
					i += consumed + 1
					if lastTableKeyword != "from" {
						expectingTable = false
					}
					expectingAlias = true
					expectingCTE = false
					continue
				}
			}
			if char == ')' {
				if inSubquery {
					return i + 1, nil
				}
			}
			if char == ';' {
				statementVerb = ""
				lastToken = ""
				expectingTable = false
				expectingAlias = false
				expectingCTE = false
				i++
				continue
			}

			// Raw strings must be checked before regular strings.
			if hasPrefixFold(runes, i, "r'''") {
				state = stateInRawTripleSingleQuoteString
				i += 4
				continue
			}
			if hasPrefixFold(runes, i, `r"""`) {
				state = stateInRawTripleDoubleQuoteString
				i += 4
				continue
			}
			if hasPrefixFold(runes, i, "r'") {
				state = stateInRawSingleQuoteString
				i += 2
				continue
			}
			if hasPrefixFold(runes, i, `r"`) {
				state = stateInRawDoubleQuoteString
				i += 2
				continue
			}
			if hasPrefix(runes, i, "'''") {
				state = stateInTripleSingleQuoteString
				i += 3
				continue
			}
			if hasPrefix(runes, i, `"""`) {
				state = stateInTripleDoubleQuoteString
				i += 3
				continue
			}
			if char == '\'' {
				state = stateInSingleQuoteString
				i++
				continue
			}
			if char == '"' {
				state = stateInDoubleQuoteString
				i++
				continue
			}

			if unicode.IsLetter(char) || char == '`' || char == '_' {
				parts, consumed, err := parseIdentifierSequence(runes[i:])
				if err != nil {
					return 0, err
				}
				if consumed == 0 {
					i++
					continue
				}

				keyword := strings.ToLower(parts[0])
				fullID := strings.ToLower(strings.Join(parts, "."))

				// Security check for restricted statements
				if keyword == "immediate" && lastToken == "execute" {
					return 0, fmt.Errorf("EXECUTE IMMEDIATE is not allowed when dataset restrictions are in place")
				}
				if (lastToken == "create" || lastToken == "create or" || lastToken == "create or replace") &&
					(keyword == "procedure" || keyword == "function" || keyword == "table function") {
					tokenToReport := strings.ToUpper(lastToken)
					if tokenToReport == "" {
						tokenToReport = "CREATE"
					}
					return 0, fmt.Errorf("unanalyzable statements like '%s %s' are not allowed", tokenToReport, strings.ToUpper(keyword))
				}
				if keyword == "call" {
					return 0, fmt.Errorf("CALL is not allowed when dataset restrictions are in place")
				}
				if schemaOperationVerbs[statementVerb] &&
					(keyword == "schema" || keyword == "dataset") {
					return 0, fmt.Errorf("dataset-level operations like '%s %s' are not allowed", strings.ToUpper(statementVerb), strings.ToUpper(keyword))
				}

				if lastToken == "execute" && keyword == "immediate" {
					// Found EXECUTE IMMEDIATE. The first expression must be the SQL string.
					// Search for the next string literal.
					sqlConsumed, err := findAndParseSQLString(runes[i+consumed:], defaultProjectID, tableIDSet, visitedSQLs, aliases)
					if err != nil {
						return 0, err
					}
					i += consumed + sqlConsumed
					lastToken = "execute immediate"
					continue
				}

				// Resolve aliases and identify table references.
				isKnownAlias := false
				if _, ok := aliases[fullID]; ok {
					isKnownAlias = true
				}
				if !isKnownAlias && len(parts) > 1 {
					if _, ok := aliases[strings.ToLower(parts[0])]; ok {
						isKnownAlias = true
					}
				}

				if expectingCTE {
					aliases[fullID] = struct{}{}
					aliases[strings.ToLower(parts[0])] = struct{}{}
					expectingCTE = false
				} else if expectingAlias {
					if len(parts) == 1 && (tableContextExitKeywords[keyword] || tableFollowsKeywords[keyword] || keyword == "select" || keyword == "with") {
						expectingAlias = false
					} else {
						aliases[fullID] = struct{}{}
						aliases[strings.ToLower(parts[0])] = struct{}{}
						expectingAlias = false
						isKnownAlias = true
					}
				}

				// Re-check aliases after potential registration.
				if !isKnownAlias {
					if _, ok := aliases[fullID]; ok {
						isKnownAlias = true
					}
				}

				if expectingTable && !isKnownAlias {
					if len(parts) >= 2 {
						tableID, err := formatTableID(parts, defaultProjectID)
						if err != nil {
							return 0, err
						}
						if tableID != "" {
							// If it's a system function (AI.FORECAST, etc.), don't treat it as a table.
							isSystem := false
							p := strings.Split(tableID, ".")
							if len(p) == 3 && IsSystemResource(p[1], p[2]) {
								isSystem = true
							}
							if !isSystem {
								tableIDSet[tableID] = struct{}{}
							}
						}
					}
					// For most keywords, we expect only one table.
					if lastTableKeyword != "from" {
						expectingTable = false
					}
					expectingAlias = true
				}

				// Update state machine based on the current keyword.
				if len(parts) == 1 {
					if keyword == "with" {
						expectingCTE = true
						statementVerb = "with"
					} else if keyword == "as" {
						if statementVerb != "with" {
							expectingAlias = true
						}
						expectingTable = false
					} else if _, ok := tableFollowsKeywords[keyword]; ok {
						expectingTable = true
						lastTableKeyword = keyword
						expectingAlias = false
					} else if _, ok := tableContextExitKeywords[keyword]; ok {
						expectingTable = false
						lastTableKeyword = ""
						expectingAlias = false
					}
					if lastToken == "create" && keyword == "or" {
						lastToken = "create or"
					} else if lastToken == "create or" && keyword == "replace" {
						lastToken = "create or replace"
					} else {
						lastToken = keyword
					}
					// Also track statement verb for schema checks
					if sqlStatementVerbs[keyword] {
						if statementVerb == "" || statementVerb == "with" {
							statementVerb = keyword
						}
					}
				} else {
					lastToken = ""
				}
				i += consumed
				continue
			}
			i++
		case stateInSingleQuoteString:
			if char == '\\' {
				i += 2
				continue
			}
			if char == '\'' {
				state = stateNormal
			}
			i++
		case stateInDoubleQuoteString:
			if char == '\\' {
				i += 2
				continue
			}
			if char == '"' {
				state = stateNormal
			}
			i++
		case stateInTripleSingleQuoteString:
			if hasPrefix(runes, i, "'''") {
				state = stateNormal
				i += 3
			} else {
				i++
			}
		case stateInTripleDoubleQuoteString:
			if hasPrefix(runes, i, `"""`) {
				state = stateNormal
				i += 3
			} else {
				i++
			}
		case stateInSingleLineCommentDash, stateInSingleLineCommentHash:
			if char == '\n' {
				state = stateNormal
			}
			i++
		case stateInMultiLineComment:
			if hasPrefix(runes, i, "*/") {
				state = stateNormal
				i += 2
			} else {
				i++
			}
		case stateInRawSingleQuoteString:
			if char == '\'' {
				state = stateNormal
			}
			i++
		case stateInRawDoubleQuoteString:
			if char == '"' {
				state = stateNormal
			}
			i++
		case stateInRawTripleSingleQuoteString:
			if hasPrefix(runes, i, "'''") {
				state = stateNormal
				i += 3
			} else {
				i++
			}
		case stateInRawTripleDoubleQuoteString:
			if hasPrefix(runes, i, `"""`) {
				state = stateNormal
				i += 3
			} else {
				i++
			}
		}
	}
	if inSubquery {
		return 0, fmt.Errorf("unclosed subquery parenthesis")
	}
	return len(runes), nil
}

// findAndParseSQLString scans for the first string literal and parses its content as SQL.
func findAndParseSQLString(runes []rune, defaultProjectID string, tableIDSet map[string]struct{}, visitedSQLs map[string]struct{}, aliases map[string]struct{}) (int, error) {
	for i := 0; i < len(runes); {
		if hasPrefix(runes, i, "'''") {
			end := indexRunes(runes[i+3:], "'''")
			if end != -1 {
				sqlContent := string(runes[i+3 : i+3+end])
				if _, err := parseSQL(sqlContent, defaultProjectID, tableIDSet, visitedSQLs, aliases, false); err != nil {
					return 0, err
				}
				return i + 3 + end + 3, nil
			}
		}
		if hasPrefix(runes, i, `"""`) {
			end := indexRunes(runes[i+3:], `"""`)
			if end != -1 {
				sqlContent := string(runes[i+3 : i+3+end])
				if _, err := parseSQL(sqlContent, defaultProjectID, tableIDSet, visitedSQLs, aliases, false); err != nil {
					return 0, err
				}
				return i + 3 + end + 3, nil
			}
		}
		if runes[i] == '\'' {
			// Find end of single-quoted string, respecting backslash escapes.
			for j := i + 1; j < len(runes); j++ {
				if runes[j] == '\\' {
					j++
					continue
				}
				if runes[j] == '\'' {
					sqlContent := string(runes[i+1 : j])
					if _, err := parseSQL(sqlContent, defaultProjectID, tableIDSet, visitedSQLs, aliases, false); err != nil {
						return 0, err
					}
					return j + 1, nil
				}
			}
		}
		if runes[i] == '"' {
			for j := i + 1; j < len(runes); j++ {
				if runes[j] == '\\' {
					j++
					continue
				}
				if runes[j] == '"' {
					sqlContent := string(runes[i+1 : j])
					if _, err := parseSQL(sqlContent, defaultProjectID, tableIDSet, visitedSQLs, aliases, false); err != nil {
						return 0, err
					}
					return j + 1, nil
				}
			}
		}
		i++
	}
	return len(runes), nil
}

// IsAnyTableExplicitlyReferenced performs a lexical audit of the SQL to see if any of the target tables
// are explicitly named as identifiers. It correctly ignores names inside comments or strings.
func IsAnyTableExplicitlyReferenced(sql, defaultProjectID string, targetTableIDs []string) (bool, error) {
	targets := make(map[string]struct{})
	for _, id := range targetTableIDs {
		targets[strings.ToLower(id)] = struct{}{}
	}

	runes := []rune(sql)
	state := stateNormal

	for i := 0; i < len(runes); {
		char := runes[i]

		switch state {
		case stateNormal:
			if hasPrefix(runes, i, "--") {
				state = stateInSingleLineCommentDash
				i += 2
				continue
			}
			if char == '#' {
				state = stateInSingleLineCommentHash
				i++
				continue
			}
			if hasPrefix(runes, i, "/*") {
				state = stateInMultiLineComment
				i += 2
				continue
			}

			if unicode.IsLetter(char) || char == '`' || char == '_' {
				parts, consumed, err := parseIdentifierSequence(runes[i:])
				if err != nil {
					return false, err
				}
				if consumed > 0 {
					fullID := strings.ToLower(strings.Join(parts, "."))
					for target := range targets {
						// Exact match or as a prefix for column references.
						if fullID == target || strings.HasPrefix(fullID, target+".") {
							return true, nil
						}
						// Match without any backticks.
						cleanFullID := strings.ReplaceAll(fullID, "`", "")
						cleanTarget := strings.ReplaceAll(target, "`", "")
						if cleanFullID == cleanTarget || strings.HasPrefix(cleanFullID, cleanTarget+".") {
							return true, nil
						}
						// Try matching with the default project ID prefix.
						if defaultProjectID != "" {
							cleanDefaultProjectID := strings.ReplaceAll(strings.ToLower(defaultProjectID), "`", "")
							withDefault := cleanDefaultProjectID + "." + cleanFullID
							if withDefault == cleanTarget || strings.HasPrefix(withDefault, cleanTarget+".") {
								return true, nil
							}
						}
					}
					i += consumed
					continue
				}
			}

			// Handle various BigQuery string literal formats.
			if hasPrefixFold(runes, i, "r'''") {
				state = stateInRawTripleSingleQuoteString
				i += 4
				continue
			}
			if hasPrefixFold(runes, i, `r"""`) {
				state = stateInRawTripleDoubleQuoteString
				i += 4
				continue
			}
			if hasPrefixFold(runes, i, "r'") {
				state = stateInRawSingleQuoteString
				i += 2
				continue
			}
			if hasPrefixFold(runes, i, `r"`) {
				state = stateInRawDoubleQuoteString
				i += 2
				continue
			}
			if hasPrefix(runes, i, "'''") {
				state = stateInTripleSingleQuoteString
				i += 3
				continue
			}
			if hasPrefix(runes, i, `"""`) {
				state = stateInTripleDoubleQuoteString
				i += 3
				continue
			}
			if char == '\'' {
				state = stateInSingleQuoteString
				i++
				continue
			}
			if char == '"' {
				state = stateInDoubleQuoteString
				i++
				continue
			}

		case stateInSingleQuoteString:
			if char == '\\' {
				i += 2
				continue
			}
			if char == '\'' {
				state = stateNormal
			}
		case stateInDoubleQuoteString:
			if char == '\\' {
				i += 2
				continue
			}
			if char == '"' {
				state = stateNormal
			}
		case stateInTripleSingleQuoteString:
			if hasPrefix(runes, i, "'''") {
				state = stateNormal
				i += 3
				continue
			}
		case stateInTripleDoubleQuoteString:
			if hasPrefix(runes, i, `"""`) {
				state = stateNormal
				i += 3
				continue
			}
		case stateInSingleLineCommentDash, stateInSingleLineCommentHash:
			if char == '\n' {
				state = stateNormal
			}
		case stateInMultiLineComment:
			if hasPrefix(runes, i, "*/") {
				state = stateNormal
				i += 2
				continue
			}
		case stateInRawSingleQuoteString:
			if char == '\'' {
				state = stateNormal
			}
		case stateInRawDoubleQuoteString:
			if char == '"' {
				state = stateNormal
			}
		case stateInRawTripleSingleQuoteString:
			if hasPrefix(runes, i, "'''") {
				state = stateNormal
				i += 3
				continue
			}
		case stateInRawTripleDoubleQuoteString:
			if hasPrefix(runes, i, `"""`) {
				state = stateNormal
				i += 3
				continue
			}
		}
		i++
	}

	return false, nil
}

// parseIdentifierSequence parses a sequence of dot-separated identifiers.
// It returns the parts of the identifier, the number of characters consumed, and an error.
func parseIdentifierSequence(runes []rune) ([]string, int, error) {
	var parts []string
	var totalConsumed int
	for {
		// Skip whitespace and comments before identifier part
		for {
			originalConsumed := totalConsumed
			for totalConsumed < len(runes) && unicode.IsSpace(runes[totalConsumed]) {
				totalConsumed++
			}
			if hasPrefix(runes, totalConsumed, "/*") {
				endIdx := indexRunes(runes[totalConsumed:], "*/")
				if endIdx != -1 {
					totalConsumed += endIdx + 2
				}
			} else if hasPrefix(runes, totalConsumed, "--") || (totalConsumed < len(runes) && runes[totalConsumed] == '#') {
				endIdx := indexRunes(runes[totalConsumed:], "\n")
				if endIdx != -1 {
					totalConsumed += endIdx + 1
				} else {
					totalConsumed = len(runes)
				}
			}
			if totalConsumed == originalConsumed {
				break
			}
		}
		if totalConsumed >= len(runes) {
			break
		}

		var part string
		var consumed int

		if runes[totalConsumed] == '`' {
			end := indexRunes(runes[totalConsumed+1:], "`")
			if end == -1 {
				return nil, 0, fmt.Errorf("unclosed backtick identifier")
			}
			part = string(runes[totalConsumed+1 : totalConsumed+end+1])
			consumed = end + 2
		} else if unicode.IsLetter(runes[totalConsumed]) || runes[totalConsumed] == '_' {
			end := totalConsumed
			for end < len(runes) && (unicode.IsLetter(runes[end]) || unicode.IsNumber(runes[end]) || runes[end] == '_' || runes[end] == '-') {
				end++
			}
			part = string(runes[totalConsumed:end])
			consumed = end - totalConsumed
		} else {
			break
		}

		parts = append(parts, strings.Split(part, ".")...)
		totalConsumed += consumed

		// Skip whitespace and comments between parts (before potential dot)
		for {
			originalConsumed := totalConsumed
			for totalConsumed < len(runes) && unicode.IsSpace(runes[totalConsumed]) {
				totalConsumed++
			}
			if hasPrefix(runes, totalConsumed, "/*") {
				endIdx := indexRunes(runes[totalConsumed:], "*/")
				if endIdx != -1 {
					totalConsumed += endIdx + 2
				}
			} else if hasPrefix(runes, totalConsumed, "--") || (totalConsumed < len(runes) && runes[totalConsumed] == '#') {
				endIdx := indexRunes(runes[totalConsumed:], "\n")
				if endIdx != -1 {
					totalConsumed += endIdx + 1
				} else {
					totalConsumed = len(runes)
				}
			}
			if totalConsumed == originalConsumed {
				break
			}
		}

		if totalConsumed >= len(runes) || runes[totalConsumed] != '.' {
			break
		}
		totalConsumed++
	}

	return parts, totalConsumed, nil
}

func formatTableID(parts []string, defaultProjectID string) (string, error) {
	if len(parts) == 4 && strings.Contains(parts[1], ":") {
		parts = []string{parts[0] + "." + parts[1], parts[2], parts[3]}
	}
	if len(parts) < 2 || len(parts) > 3 {
		// Not a table identifier (could be a CTE, column, etc.).
		return "", nil
	}

	if len(parts) == 3 { // project.dataset.table
		return strings.Join(parts, "."), nil
	}

	// dataset.table
	if defaultProjectID == "" {
		return "", fmt.Errorf("query contains table '%s' without project ID, and no default project ID is provided", strings.Join(parts, "."))
	}
	return fmt.Sprintf("%s.%s", defaultProjectID, strings.Join(parts, ".")), nil
}

func indexRunes(r []rune, sub string) int {
	subRunes := []rune(sub)
	if len(subRunes) == 0 {
		return 0
	}
	for i := 0; i <= len(r)-len(subRunes); i++ {
		match := true
		for j := 0; j < len(subRunes); j++ {
			if r[i+j] != subRunes[j] {
				match = false
				break
			}
		}
		if match {
			return i
		}
	}
	return -1
}
