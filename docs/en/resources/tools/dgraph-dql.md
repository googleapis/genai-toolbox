---
title: "dgraph-dql"
type: docs
weight: 1
description: > 
  A "dgraph-dql" tool executes a pre-defined DQL statement against a Dgraph
  database.
---

## About

A `dgraph-dql` tool executes a pre-defined DQL statement against a Dgraph
database. It's compatible with any of the following sources:

- [dgraph](../sources/dgraph.md)

To run a statement as a query, you need to set the config `isQuery=true`. For
upserts or mutations, set `isQuery=false`. You can also configure timeout for a
query.

> **Note:** This tool uses parameterized queries to prevent SQL injections.
> Query parameters can be used as substitutes for arbitrary expressions.
> Parameters cannot be used as substitutes for identifiers, column names, table
> names, or other parts of the query.

## Example

{{< tabpane persist="header" >}}
{{< tab header="Query" lang="yaml" >}}

tools:
  search_user:
    kind: dgraph-dql
    source: my-dgraph-source
    statement: |
      query all($role: string){
        users(func: has(name)) @filter(eq(role, $role) AND ge(age, 30) AND le(age, 50)) {
          uid
          name
          email
          role
          age
        }
      }
    isQuery: true
    timeout: 20s
    description: |
     Use this tool to retrieve the details of users who are admins and are between 30 and 50 years old.
     The query returns the user's name, email, role, and age.
     This can be helpful when you want to fetch admin users within a specific age range.
     Example: Fetch admins aged between 30 and 50:
      [
        {
          "name": "Alice",
          "role": "admin",
          "age": 35
        },
        {
          "name": "Bob",
          "role": "admin",
          "age": 45
        }
      ]
    parameters:
      - name: $role
        type: string
        description: admin

{{< /tab >}}
{{< tab header="Mutation" lang="yaml" >}}

tools:
  dgraph-manage-user-instance:
    kind: dgraph-dql
    source: my-dgraph-source
    isQuery: false
    statement: |
           {
            set {
            _:user1 <name> $user1 .
            _:user1 <email> $email1 .
            _:user1 <role> "admin" .
            _:user1 <age> "35" .

            _:user2 <name> $user2 .
            _:user2 <email> $email2 .
            _:user2 <role> "admin" .
            _:user2 <age> "45" .
            }
           }
    description: |
     Use this tool to insert or update user data into the Dgraph database.
     The mutation adds or updates user details like name, email, role, and age.
     Example: Add users Alice and Bob as admins with specific ages.
    parameters:
      - name: user1
        type: string
        description: Alice
      - name: email1
        type: string
        description: alice@email.com
      - name: user2
        type: string
        description: Bob
      - name: email2
        type: string
        description: bob@email.com

{{< /tab >}}
{{< /tabpane >}}

## Reference

| **field**   |                  **type**                  | **required** | **description**                                                                              |
|-------------|:------------------------------------------:|:------------:|----------------------------------------------------------------------------------------------|
| kind        |                   string                   |     true     | Must be "dgraph-dql".                                                                        |
| source      |                   string                   |     true     | Name of the source the dql query should execute on.                                          |
| description |                   string                   |     true     | Description of the tool that is passed to the LLM.                                           |
| statement   |                   string                   |     true     | dql statement to execute                                                                     |
| isQuery     |                  boolean                   |    false     | To run statement as query set true otherwise false                                           |
| timeout     |                   string                   |    false     | To set timeout for query                                                                     |
| parameters  | [parameters](_index#specifying-parameters) |    false     | List of [parameters](_index#specifying-parameters) that will be used with the dql statement. |
