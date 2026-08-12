---
title: "odata"
type: docs
weight: 1
description: >
  An "odata" tool executes CRUD operations and function imports against an OData service.
---

## About

An `odata` tool dynamically maps OData entities and metadata to perform operations like `READ`, `CREATE`, `UPDATE`, `DELETE`, and `FUNCTION_IMPORT`. 

When executing `READ` operations, it automatically parses the OData `$filter`, `$select`, `$top`, `$skip`, and `$skiptoken` pagination parameters, exposing metadata properties directly to the LLM for high-quality context and seamless discovery.

## Compatible Sources

{{< compatible-sources >}}

## Example

An example of a `CREATE` tool to create a sales order header:

```yaml
kind: tool
name: create_sales_order_header
type: odata
source: odata-sales-order-srv
entitySet: A_SalesOrder
operation: CREATE
description: Create a new sales order header entity.
bodyParams:
  - name: SalesOrderType
    description: Type of the sales order (e.g. "OR")
    type: string
  - name: SalesOrganization
    description: Sales organization (e.g. "1010")
    type: string
  - name: SoldToParty
    description: Sold-to party customer number
    type: string
```

An example of a `READ` tool to query sales order headers:

```yaml
kind: tool
name: read_sales_order_header
type: odata
source: odata-sales-order-srv
entitySet: A_SalesOrder
operation: READ
description: Retrieve and query sales order headers.
```

## Reference

| **field** | **type** | **required** | **description** |
| :--- | :---: | :---: | :--- |
| type | string | true | Must be "odata". |
| source | string | true | The name of the configured OData source. |
| entitySet | string | true | The entity set name (e.g., `A_SalesOrder`) or function import name. |
| operation | string | true | The operation type to perform. One of: `READ`, `CREATE`, `UPDATE`, `DELETE`, `FUNCTION_IMPORT`. |
| description | string | true | Description of the tool provided to the LLM. |
| contentType | string | false | Override for the request `Content-Type` header. Defaults to `'application/json'`. |
| queryParams | list | false | Custom query parameters specific to this tool. |
| bodyParams | list | false | Request body parameters (required for `CREATE` and `UPDATE` operations). |
| authRequired | array[string] | false | List of required authentication services to invoke the tool. |
