---
title: "spanner-sql"
type: docs
weight: 1
description: >
  A "spanner-sql" tool executes a pre-defined SQL statement against a Google
  Cloud Spanner database.
---

## About

A `spanner-sql` tool executes a pre-defined SQL statement (either `googlesql` or
`postgresql`) against a Cloud Spanner database.

## Compatible Sources

{{< compatible-sources >}}

### GoogleSQL

For the `googlesql` dialect, the specified SQL statement is executed as a [data
manipulation language (DML)][gsql-dml] statements, and specified parameters will
inserted according to their name: e.g. `@name`.

> **Note:** This tool uses parameterized queries to prevent SQL injections.
> Query parameters can be used as substitutes for arbitrary expressions.
> Parameters cannot be used as substitutes for identifiers, column names, table
> names, or other parts of the query.

[gsql-dml]:
    https://cloud.google.com/spanner/docs/reference/standard-sql/dml-syntax

### PostgreSQL

For the `postgresql` dialect, the specified SQL statement is executed as a [prepared
statement][pg-prepare], and specified parameters will be inserted according to
their position: e.g. `$1` will be the first parameter specified, `$2` will be
the second parameter, and so on.

[pg-prepare]: https://www.postgresql.org/docs/current/sql-prepare.html

## Example

> **Note:** This tool uses parameterized queries to prevent SQL injections.
> Query parameters can be used as substitutes for arbitrary expressions.
> Parameters cannot be used as substitutes for identifiers, column names, table
> names, or other parts of the query.

{{< tabpane persist="header" >}}
{{< tab header="GoogleSQL" lang="yaml" >}}

kind: tool
name: search_flights_by_number
type: spanner-sql
source: my-spanner-instance
statement: |
  SELECT * FROM flights
  WHERE airline = @airline
  AND flight_number = @flight_number
  LIMIT 10
description: |
  Use this tool to get information for a specific flight.
  Takes an airline code and flight number and returns info on the flight.
  Do NOT use this tool with a flight id. Do NOT guess an airline code or flight number.
  A airline code is a code for an airline service consisting of two-character
  airline designator and followed by flight number, which is 1 to 4 digit number.
  For example, if given CY 0123, the airline is "CY", and flight_number is "123".
  Another example for this is DL 1234, the airline is "DL", and flight_number is "1234".
  If the tool returns more than one option choose the date closes to today.
  Example:
  {{
      "airline": "CY",
      "flight_number": "888",
  }}
  Example:
  {{
      "airline": "DL",
      "flight_number": "1234",
  }}
parameters:
  - name: airline
    type: string
    description: Airline unique 2 letter identifier
  - name: flight_number
    type: string
    description: 1 to 4 digit number

{{< /tab >}}
{{< tab header="PostgreSQL" lang="yaml" >}}

kind: tool
name: search_flights_by_number
type: spanner
source: my-spanner-instance
statement: |
  SELECT * FROM flights
  WHERE airline = $1
  AND flight_number = $2
  LIMIT 10
description: |
  Use this tool to get information for a specific flight.
  Takes an airline code and flight number and returns info on the flight.
  Do NOT use this tool with a flight id. Do NOT guess an airline code or flight number.
  A airline code is a code for an airline service consisting of two-character
  airline designator and followed by flight number, which is 1 to 4 digit number.
  For example, if given CY 0123, the airline is "CY", and flight_number is "123".
  Another example for this is DL 1234, the airline is "DL", and flight_number is "1234".
  If the tool returns more than one option choose the date closes to today.
  Example:
  {{
      "airline": "CY",
      "flight_number": "888",
  }}
  Example:
  {{
      "airline": "DL",
      "flight_number": "1234",
  }}
parameters:
  - name: airline
    type: string
    description: Airline unique 2 letter identifier
  - name: flight_number
    type: string
    description: 1 to 4 digit number

{{< /tab >}}
{{< /tabpane >}}


### Example with Vector Search

Spanner supports high-performance vector similarity search. When using an `embeddingModel` with a `spanner-sql` tool, the tool automatically converts text parameters into the native ARRAY<FLOAT64> format required by Spanner.

#### Define the Embedding Model
See [EmbeddingModels](../../../documentation/configuration/embedding-models/_index.md) for more information.

```yaml
kind: embeddingModel
name: gemini-model
type: gemini
model: gemini-embedding-001
apiKey: ${GOOGLE_API_KEY}
dimension: 768
```

#### Vector Ingestion Tool
This tool stores both the raw text and its vector representation. It uses `valueFromParam` to hide the vector conversion logic from the LLM, ensuring the Agent only has to provide the content once.

```yaml
kind: tool
name: insert_doc_spanner
type: spanner-sql
source: my-spanner-source
statement: |
  INSERT INTO vector_table (id, content, embedding)
  VALUES (1, @content, @text_to_embed)
description: |
  Index new documents for semantic search in Spanner.
parameters:
  - name: content
    type: string
    description: The text content to store.
  - name: text_to_embed
    type: string
    # Automatically copies 'content' and converts it to a FLOAT64 array
    valueFromParam: content
    embeddedBy: gemini-model
```

#### Vector Search Tool
This tool allows the Agent to perform a natural language search. The query string provided by the Agent is converted into a vector before the SQL is executed.

```yaml
kind: tool
name: search_docs_spanner
type: spanner-sql
source: my-spanner-source
statement: |
  SELECT 
    id, 
    content, 
    COSINE_DISTANCE(embedding, @query) AS distance 
  FROM 
    vector_table 
  ORDER BY 
    distance 
  LIMIT 1
description: |
  Search for documents in Spanner using natural language. 
  Returns the most semantically similar result.
parameters:
  - name: query
    type: string
    description: The search query to be converted to a vector.
    embeddedBy: gemini-model
```


### Example with Template Parameters

> **Note:** This tool allows direct modifications to the SQL statement,
> including identifiers, column names, and table names. **This makes it more
> vulnerable to SQL injections**. Using basic parameters only (see above) is
> recommended for performance and safety reasons. For more details, please check
> [templateParameters](../../../documentation/configuration/tools/_index.md#template-parameters).

```yaml
kind: tool
name: list_table
type: spanner
source: my-spanner-instance
statement: |
  SELECT * FROM {{.tableName}};
description: |
  Use this tool to list all information from a specific table.
  Example:
  {{
      "tableName": "flights",
  }}
templateParameters:
  - name: tableName
    type: string
    description: Table to select from
```

## Reference

| **field**          |                   **type**                   | **required** | **description**                                                                                                                        |
|--------------------|:--------------------------------------------:|:------------:|----------------------------------------------------------------------------------------------------------------------------------------|
| type               |                    string                    |     true     | Must be "spanner-sql".                                                                                                                 |
| source             |                    string                    |     true     | Name of the source the SQL should execute on.                                                                                          |
| description        |                    string                    |     true     | Description of the tool that is passed to the LLM.                                                                                     |
| statement          |                    string                    |     true     | SQL statement to execute on.                                                                                                           |
| parameters         |   [parameters](../../../documentation/configuration/tools/_index.md#specifying-parameters)    |    false     | List of [parameters](../../../documentation/configuration/tools/_index.md#specifying-parameters) that will be inserted into the SQL statement.                                          |
| readOnly           |                     bool                     |    false     | When set to `true`, the `statement` is run as a read-only transaction. Default: `false`.                                               |
| templateParameters | [templateParameters](../../../documentation/configuration/tools/_index.md#template-parameters) |    false     | List of [templateParameters](../../../documentation/configuration/tools/_index.md#template-parameters) that will be inserted into the SQL statement before executing prepared statement. |
