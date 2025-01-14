# Tools

Tools represent an action your agent can take, such as running a SQL statement.
You can define Tools as a map in the `sources` section of your `tools.yaml`
file. Typically, a tool will require a source to act on:

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
```

## Kinds of Tools

We currently support the following types of kinds of tools:

* [postgres-sql](./postgres-sql.md) - Run a PostgreSQL statement against a
  PostgreSQL-compatible database.
* [spanner](./spanner.md) - Run a Spanner (either googlesql or postgresql)
  statement againts Spanner database.
* [neo4j-cypher](./neo4j-cypher.md) - Run a Cypher statement against a
  Neo4j database.


## Specifying Parameters

Parameters for each Tool will define what inputs the Agent will need to provide
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

| **field**   | **type** | **required** | **description**                                                            |
|-------------|:--------:|:------------:|----------------------------------------------------------------------------|
| name        |  string  |     true     | Name of the parameter.                                                     |
| type        |  string  |     true     | Must be one of "string", "integer", "float", "boolean" "array"             |
| description |  string  |     true     | Natural language description of the parameter to describe it to the agent. |

### Array Parameters

The `array` type is a list of items passed in as a single parameter. This `type`
requires another Parameter to be specified under the `items` field:

```yaml
    parameters:
      - name: preffered_airlines
        type: array
        description: A list of airline, ordered by preference. 
        items:
          name: name 
          type: string
          description: Name of the airline. 
```

| **field**   |     **type**     | **required** | **description**                                                            |
|-------------|:----------------:|:------------:|----------------------------------------------------------------------------|
| name        |      string      |     true     | Name of the parameter.                                                     |
| type        |      string      |     true     | Must be "array"                                                            |
| description |      string      |     true     | Natural language description of the parameter to describe it to the agent. |
| items       | parameter object |     true     | Specify a Parameter object for the type of the values in the array.        |

### Authenticated Parameters

Authenticated parameters automatically populate their values with user
information decoded from your [ID tokens](../authSources/README.md#id-token)
passed in from request headers. They do not take input values in request bodies
like other parameters. Instead, specify your configured
[authSources](../authSources/README.md) and corresponding claim fields in ID
tokens to tell Toolbox which values they should be auto-populated with.

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
            authSources:
              # Refer to one of the `authSources` defined
              - name: my-google-auth
              # `sub` is the OIDC claim field for user ID
                field: sub
```

| **field**   | **type** | **required** | **description**                                                            |
|-------------|:--------:|:------------:|----------------------------------------------------------------------------|
| name        |  string  |     true     | Name of the [authSources](../authSources/README.md) used to verify the OIDC auth token.                |
| field       |  string  |     true     | Claim field decoded from the OIDC token used to auto-populate this parameter.|

## Authorized Tool Call

You can require an authorization check for any Tool invocation request by
specifying an `authRequired` field. Specify a list of
[authSources](../authSources/README.md) defined in the previous section.

```yaml
tools:
  search_all_flight:
      kind: postgres-sql
      source: my-pg-instance
      statement: |
        SELECT * FROM flights
      # A list of `authSources` defined previously
      authRequired:
        - my-google-auth
        - other-auth-service
```
