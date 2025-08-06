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

package bigqueryexecutesql

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
	// Comment states
	stateInSingleLineCommentDash
	stateInSingleLineCommentHash
	stateInMultiLineComment
)

var tableFollowsKeywords = map[string]bool{
	"from":   true,
	"join":   true,
	"update": true,
	"into":   true, // INSERT INTO, MERGE INTO
	"table":  true, // CREATE TABLE, ALTER TABLE
	"using":  true, // MERGE ... USING
	"insert": true, // INSERT my_table
	"merge":  true, // MERGE my_table
	"call":   true, // CALL my_procedure
}

var tableContextExitKeywords = map[string]bool{
	"where":  true,
	"group":  true, // GROUP BY
	"having": true,
	"order":  true, // ORDER BY
	"limit":  true,
	"window": true,
	"on":     true, // JOIN ... ON
	"set":    true, // UPDATE ... SET
	"when":   true, // MERGE ... WHEN
}

// TableParser is the main entry point for parsing a SQL string to find all referenced table IDs.
// It handles multi-statement SQL, comments, and recursive parsing of EXECUTE IMMEDIATE statements.
func TableParser(sql, defaultProjectID string) ([]string, error) {
	tableIDSet := make(map[string]struct{})
	visitedSQLs := make(map[string]struct{})
	if err := parseSQL(sql, defaultProjectID, tableIDSet, visitedSQLs); err != nil {
		return nil, err
	}

	tableIDs := make([]string, 0, len(tableIDSet))
	for id := range tableIDSet {
		tableIDs = append(tableIDs, id)
	}
	return tableIDs, nil
}

// parseSQL is the core recursive function that processes SQL strings.
// It uses a state machine to find table names and recursively parse EXECUTE IMMEDIATE.
func parseSQL(sql, defaultProjectID string, tableIDSet map[string]struct{}, visitedSQLs map[string]struct{}) error {
	// Prevent infinite recursion.
	if _, ok := visitedSQLs[sql]; ok {
		return nil
	}
	visitedSQLs[sql] = struct{}{}

	state := stateNormal
	expectingTable := false
	var lastTableKeyword string
	runes := []rune(sql)

	for i := 0; i < len(runes); {
		char := runes[i]
		remaining := sql[i:]

		switch state {
		case stateNormal:
			if strings.HasPrefix(remaining, "--") {
				state = stateInSingleLineCommentDash
				i += 2
				continue
			}
			if strings.HasPrefix(remaining, "#") {
				state = stateInSingleLineCommentHash
				i++
				continue
			}
			if strings.HasPrefix(remaining, "/*") {
				state = stateInMultiLineComment
				i += 2
				continue
			}
			if strings.HasPrefix(remaining, "'''") {
				state = stateInTripleSingleQuoteString
				i += 3
				continue
			}
			if strings.HasPrefix(remaining, `"""`) {
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

			if unicode.IsLetter(char) || char == '`' {
				parts, consumed, err := parseIdentifierSequence(remaining)
				if err != nil {
					return err
				}
				if consumed == 0 {
					i++
					continue
				}

				if len(parts) == 1 {
					keyword := strings.ToLower(parts[0])
					if keyword == "execute" {
						// Check if the next token is "IMMEDIATE", allowing for flexible whitespace.
						// This is a best-effort check that does not handle comments between the keywords.
						nextRemaining := sql[i+consumed:]
						trimmedNext := strings.TrimLeftFunc(nextRemaining, unicode.IsSpace)
						if len(trimmedNext) >= 9 && strings.EqualFold(trimmedNext[:9], "immediate") {
							// Check for a word boundary to avoid matching prefixes like "IMMEDIATELY".
							if len(trimmedNext) == 9 || (!unicode.IsLetter(rune(trimmedNext[9])) && !unicode.IsNumber(rune(trimmedNext[9])) && trimmedNext[9] != '_') {
								return fmt.Errorf("parsing SQL with EXECUTE IMMEDIATE is not supported")
							}
						}
					}

					if _, ok := tableFollowsKeywords[keyword]; ok {
						expectingTable = true
						lastTableKeyword = keyword
					} else if _, ok := tableContextExitKeywords[keyword]; ok {
						expectingTable = false
						lastTableKeyword = ""
					}
				} else if len(parts) >= 2 {
					// This is a multi-part identifier. If we were expecting a table, this is it.
					// This also handles cases like `UPDATE my.table SET...` where the keyword is not followed by a space.
					if expectingTable {
						tableID, err := formatTableID(parts, defaultProjectID)
						if err != nil {
							return err
						}
						if tableID != "" {
							tableIDSet[tableID] = struct{}{}
						}
						// For most keywords, we expect only one table.
						if lastTableKeyword != "from" {
							expectingTable = false
						}
					}
				}

				i += consumed
				continue
			}
			i++

		case stateInSingleQuoteString:
			if char == '\'' {
				state = stateNormal
			}
			i++
		case stateInDoubleQuoteString:
			if char == '"' {
				state = stateNormal
			}
			i++
		case stateInTripleSingleQuoteString:
			if strings.HasPrefix(remaining, "'''") {
				state = stateNormal
				i += 3
			} else {
				i++
			}
		case stateInTripleDoubleQuoteString:
			if strings.HasPrefix(remaining, `"""`) {
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
			if strings.HasPrefix(remaining, "*/") {
				state = stateNormal
				i += 2
			} else {
				i++
			}
		}
	}
	return nil
}

// parseIdentifierSequence parses a sequence of dot-separated identifiers.
// It returns the parts of the identifier, the number of characters consumed, and an error.
func parseIdentifierSequence(s string) ([]string, int, error) {
	var parts []string
	var totalConsumed int

	for {
		remaining := s[totalConsumed:]
		trimmed := strings.TrimLeftFunc(remaining, unicode.IsSpace)
		totalConsumed += len(remaining) - len(trimmed)
		current := s[totalConsumed:]

		if len(current) == 0 {
			break
		}

		var part string
		var consumed int

		if current[0] == '`' {
			end := strings.Index(current[1:], "`")
			if end == -1 {
				return nil, 0, fmt.Errorf("unclosed backtick identifier")
			}
			part = current[1 : end+1]
			consumed = end + 2
		} else if len(current) > 0 && unicode.IsLetter(rune(current[0])) {
			end := strings.IndexFunc(current, func(r rune) bool {
				return !unicode.IsLetter(r) && !unicode.IsNumber(r) && r != '_' && r != '-'
			})
			if end == -1 {
				part = current
				consumed = len(current)
			} else {
				part = current[:end]
				consumed = end
			}
		} else {
			break
		}

		if current[0] == '`' && strings.Contains(part, ".") {
			// This handles cases like `project.dataset.table` but not `project.dataset`.table.
			// If the character after the quoted identifier is not a dot, we treat it as a full name.
			if len(current) <= consumed || current[consumed] != '.' {
				parts = append(parts, strings.Split(part, ".")...)
				totalConsumed += consumed
				break
			}
		}

		parts = append(parts, strings.Split(part, ".")...)
		totalConsumed += consumed

		if len(s) <= totalConsumed || s[totalConsumed] != '.' {
			break
		}
		totalConsumed++ // consume the dot
	}
	return parts, totalConsumed, nil
}

func formatTableID(parts []string, defaultProjectID string) (string, error) {
	if len(parts) < 2 || len(parts) > 3 {
		// Not a table identifier (could be a CTE, column, etc.).
		// Return the consumed length so the main loop can skip this identifier.
		return "", nil
	}

	var tableID string
	if len(parts) == 3 { // project.dataset.table
		tableID = strings.Join(parts, ".")
	} else { // dataset.table
		if defaultProjectID == "" {
			return "", fmt.Errorf("query contains table '%s' without project ID, and no default project ID is provided", strings.Join(parts, "."))
		}
		tableID = fmt.Sprintf("%s.%s", defaultProjectID, strings.Join(parts, "."))
	}

	return tableID, nil
}
