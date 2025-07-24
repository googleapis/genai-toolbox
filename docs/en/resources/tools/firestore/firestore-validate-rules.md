---
title: "firestore-validate-rules"
type: docs
weight: 1
description: > 
  A "firestore-validate-rules" tool validates Firestore security rules syntax and semantic correctness without deploying them. It provides detailed error reporting with source positions and code snippets.
aliases:
- /resources/tools/firestore-validate-rules
---

## Overview

The `firestore-validate-rules` tool validates Firestore security rules syntax and semantic correctness without deploying them. It provides detailed error reporting with source positions and code snippets.

## Configuration

```yaml
tools:
  firestore-validate-rules:
    kind: firestore-validate-rules
    source: <firestore-source-name>
    description: "Checks the provided Firestore Rules source for syntax and validation errors"
```

## Authentication

This tool requires authentication if the source requires authentication.

## Parameters

| Parameter | Type   | Required | Description |
|-----------|--------|----------|-------------|
| source    | string | Yes      | The Firestore Rules source code to validate |

## Response

The tool returns a `ValidationResult` object containing:

```json
{
  // Whether the rules are valid
  "valid": boolean,
  // Number of issues found       
  "issueCount": number,
  // Human-readable formatted issues 
  "formattedIssues": string,
  // Array of raw issue objects
  "rawIssues": [
    {
      "sourcePosition": {
        "fileName": string,
        "line": number,
        "column": number,
        "currentOffset": number,
        "endOffset": number
      },
      "description": string,
      // e.g., "ERROR", "WARNING"
      "severity": string
    }
  ]
}
```

## Example Usage

### Validate simple rules

```json
{
  "source": "rules_version = '2';\nservice cloud.firestore {\n  match /databases/{database}/documents {\n    match /{document=**} {\n      allow read, write: if true;\n    }\n  }\n}"
}
```

### Example response for valid rules

```json
{
  "valid": true,
  "issueCount": 0,
  "formattedIssues": "✓ No errors detected. Rules are valid."
}
```

### Example response with errors

```json
{
  "valid": false,
  "issueCount": 1,
  "formattedIssues": "Found 1 issue(s) in rules source:\n\nERROR: Unexpected token ';' [Ln 4, Col 32]\n```\n      allow read, write: if true;;\n                               ^\n```",
  "rawIssues": [
    {
      "sourcePosition": {
        "line": 4,
        "column": 32,
        "currentOffset": 105,
        "endOffset": 106
      },
      "description": "Unexpected token ';'",
      "severity": "ERROR"
    }
  ]
}
```

## Error Handling

The tool will return errors for:
- Missing or empty `source` parameter
- API errors when calling the Firebase Rules service
- Network connectivity issues

## Use Cases

1. **Pre-deployment validation**: Validate rules before deploying to production
2. **CI/CD integration**: Integrate rules validation into your build pipeline
3. **Development workflow**: Quickly check rules syntax while developing
4. **Error debugging**: Get detailed error locations with code snippets

## Related Tools

- [firestore-get-rules]({{< ref "firestore-get-rules" >}}): Retrieve current active rules
- [firestore-query-collection]({{< ref "firestore-query-collection" >}}): Test rules by querying collections
