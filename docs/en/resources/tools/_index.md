---
title: "Tools"
type: docs
weight: 2
description: > 
  Tools define actions an agent can take -- such as reading and writing to a 
  source.
---

A tool represents an action your agent can take, such as running a SQL
statement. You can define Tools as a map in the `tools` section of your
`tools.yaml` file. Typically, a tool will require a source to act on:

```yaml
tools:
 search_flights_by_number:
    kind: postgres-sql
    source: my-pg-instance
    statement: |
      SELECT * FROM flights
      WHERE airline = $1
      AND flight_number = $2
      LIMIT 10
    description: |
      Use this tool to get information for a specific flight.
      Takes an airline code and flight number and returns info on the flight.
      Do NOT use this tool with a flight id. Do NOT guess an airline code or flight number.
      An airline code is a code for an airline service consisting of a two-character
      airline designator and followed by a flight number, which is a 1 to 4 digit number.
      For example, if given CY 0123, the airline is "CY", and flight_number is "123".
      Another example for this is DL 1234, the airline is "DL", and flight_number is "1234".
      If the tool returns more than one option choose the date closest to today.
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
```

## Specifying Parameters

Parameters for each Tool will define what inputs the agent will need to provide
to invoke them. Parameters should be pass as a list of Parameter objects:

```yaml
    parameters:
      - name: airline
        type: string
        description: Airline unique 2 letter identifier
      - name: flight_number
        type: string
        description: 1 to 4 digit number
```

### Basic Parameters

Basic parameters types include `string`, `integer`, `float`, `boolean` types. In
most cases, the description will be provided to the LLM as context on specifying
the parameter.

```yaml
    parameters:
      - name: airline
        type: string
        description: Airline unique 2 letter identifier
```

| **field**   | **type**        | **required** | **description**                                                             |
|-------------|:---------------:|:------------:|-----------------------------------------------------------------------------|
| name        |  string         |     true     | Name of the parameter.                                                      |
| type        |  string         |     true     | Must be one of "string", "integer", "float", "boolean" "array"              |
| default     |  parameter type |     false    | Default value of the parameter. If provided, the parameter is not required. |
| description |  string         |     true     | Natural language description of the parameter to describe it to the agent.  |

### Array Parameters

The `array` type is a list of items passed in as a single parameter.
To use the `array` type, you must also specify what kind of items are
in the list using the items field:

```yaml
    parameters:
      - name: preferred_airlines
        type: array
        description: A list of airline, ordered by preference.
        items:
          name: name
          type: string
          description: Name of the airline.
    statement: |
      SELECT * FROM airlines WHERE preferred_airlines = ANY($1);
```

| **field**   |     **type**     | **required** | **description**                                                             |
|-------------|:----------------:|:------------:|-----------------------------------------------------------------------------|
| name        |      string      |     true     | Name of the parameter.                                                      |
| type        |      string      |     true     | Must be "array"                                                             |
| default     |  parameter type  |     false    | Default value of the parameter. If provided, the parameter is not required. |
| description |      string      |     true     | Natural language description of the parameter to describe it to the agent.  |
| items       | parameter object |     true     | Specify a Parameter object for the type of the values in the array.         |

{{< notice note >}}
Items in array should not have a default value. If provided, it will be ignored.
{{< /notice >}}

### Map Parameters

The map type is a collection of key-value pairs. It can be configured in two ways:

- Generic Map: By default, it accepts values of any primitive type (string, number, boolean), allowing for mixed data.
- Typed Map: By setting the valueType field, you can enforce that all values
  within the map must be of the same specified type.

#### Generic Map (Mixed Value Types)

This is the default behavior when valueType is omitted. It's useful for passing a flexible group of settings.

```yaml
    parameters:
          - name: execution_context
            type: map
            description: A flexible set of key-value pairs for the execution environment.
```

#### Typed Map

Specify valueType to ensure all values in the map are of the same type. An error
will be thrown in case of value type mismatch.

```yaml
 parameters:
      - name: user_scores
        type: map
        description: A map of user IDs to their scores. All scores must be integers.
        valueType: integer # This enforces the value type for all entries.
```

### Authenticated Parameters

Authenticated parameters are automatically populated with user
information decoded from [ID
tokens](../authsources/#specifying-id-tokens-from-clients) that are passed in
request headers. They do not take input values in request bodies like other
parameters. To use authenticated parameters, you must configure the tool to map
the required [authServices](../authservices) to specific claims within the
user's ID token.

```yaml
  tools:
    search_flights_by_user_id:
        kind: postgres-sql
        source: my-pg-instance
        statement: |
          SELECT * FROM flights WHERE user_id = $1
        parameters:
          - name: user_id
            type: string
            description: Auto-populated from Google login
            authServices:
              # Refer to one of the `authServices` defined
              - name: my-google-auth
              # `sub` is the OIDC claim field for user ID
                field: sub
```

| **field** | **type** | **required** | **description**                                                                         |
|-----------|:--------:|:------------:|-----------------------------------------------------------------------------------------|
| name      |  string  |     true     | Name of the [authServices](../authservices) used to verify the OIDC auth token. |
| field     |  string  |     true     | Claim field decoded from the OIDC token used to auto-populate this parameter.           |

### Template Parameters

Template parameters types include `string`, `integer`, `float`, `boolean` types.
In most cases, the description will be provided to the LLM as context on
specifying the parameter. Template parameters will be inserted into the SQL
statement before executing the prepared statement. They will be inserted without
quotes, so to insert a string using template parameters, quotes must be
explicitly added within the string.

Template parameter arrays can also be used similarly to basic parameters, and array
items must be strings. Once inserted into the SQL statement, the outer layer of
quotes will be removed. Therefore to insert strings into the SQL statement, a
set of quotes must be explicitly added within the string.

{{< notice warning >}}
Because template parameters can directly replace identifiers, column names, and
table names, they are prone to SQL injections. Basic parameters are preferred
for performance and safety reasons.
{{< /notice >}}

```yaml
tools:
 select_columns_from_table:
    kind: postgres-sql
    source: my-pg-instance
    statement: |
      SELECT {{array .columnNames}} FROM {{.tableName}}
    description: |
      Use this tool to list all information from a specific table.
      Example:
      {{
          "tableName": "flights",
          "columnNames": ["id", "name"]
      }}
    templateParameters:
      - name: tableName
        type: string
        description: Table to select from
      - name: columnNames
        type: array
        description: The columns to select
        items:
          name: column
          type: string
          description: Name of a column to select
```

| **field**   | **type**         | **required**  | **description**                                                                     |
|-------------|:----------------:|:-------------:|-------------------------------------------------------------------------------------|
| name        |  string          |     true      | Name of the template parameter.                                                     |
| type        |  string          |     true      | Must be one of "string", "integer", "float", "boolean" "array"                      |
| description |  string          |     true      | Natural language description of the template parameter to describe it to the agent. |
| items       | parameter object |true (if array)| Specify a Parameter object for the type of the values in the array (string only).   |

## Authorized Invocations

You can require an authorization check for any Tool invocation request by
specifying an `authRequired` field. Specify a list of
[authServices](../authservices) defined in the previous section.

```yaml
tools:
  search_all_flight:
      kind: postgres-sql
      source: my-pg-instance
      statement: |
        SELECT * FROM flights
      # A list of `authServices` defined previously
      authRequired:
        - my-google-auth
        - other-auth-service
```

## Kinds of tools
