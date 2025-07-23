---
title: "mongodb-update-many"
type: docs
weight: 1
description: > 
  A "mongodb-update-many" tool updates all documents in a MongoDB collection that match a filter.
aliases:
- /resources/tools/mongodb-update-many
---

## About

A `mongodb-update-many` tool updates **all** documents within a specified MongoDB collection that match a given filter. It locates the documents using a `filterPayload` and applies the modifications defined in an `updatePayload`.

The tool returns an array of three integers: `[ModifiedCount, UpsertedCount, MatchedCount]`.

This tool is compatible with the following source kind:

* [`mongodb`](../sources/mongodb.md)

---

## Example

Here's an example configuration. This tool applies a discount to all items within a specific category and also marks them as being on sale.

```yaml
tools:
  apply_category_discount:
    kind: mongodb-update-many
    source: my-mongo-source
    description: Use this tool to apply a discount to all items in a given category.
    database: products
    collection: inventory
    filterPayload: |
        { "category": {{json .category_name}} }
    filterParams:
      - name: category_name
        type: string
        description: The category of items to update.
    updatePayload: |
        { 
          "$mul": { "price": {{json .discount_multiplier}} },
          "$set": { "on_sale": true }
        }
    updateParams:
      - name: discount_multiplier
        type: number
        description: The multiplier to apply to the price (e.g., 0.8 for a 20% discount).
    canonical: false
    upsert: false
```

| **field**       | **type** | **required** | **description**                                                                                                                                                                                                             |
|:----------------|:---------|:-------------|:----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| `kind`          | `string` | **true**     | Must be `mongodb-update-many`.                                                                                                                                                                                              |
| `source`        | `string` | **true**     | The name of the `mongodb` source to use.                                                                                                                                                                                    |
| `description`   | `string` | **true**     | A description of the tool that is passed to the LLM.                                                                                                                                                                        |
| `database`      | `string` | **true**     | The name of the MongoDB database containing the collection.                                                                                                                                                                 |
| `collection`    | `string` | **true**     | The name of the MongoDB collection in which to update documents.                                                                                                                                                            |
| `filterPayload` | `string` | **true**     | The MongoDB query filter document to select the documents for updating. It's written as a Go template, using `{{json .param_name}}` to insert parameters.                                                                   |
| `filterParams`  | `list`   | **true**     | A list of parameter objects that define the variables used in the `filterPayload`.                                                                                                                                          |
| `updatePayload` | `string` | **true**     | The MongoDB update document, It's written as a Go template, using `{{json .param_name}}` to insert parameters.                                                                                                              |
| `updateParams`  | `list`   | **true**     | A list of parameter objects that define the variables used in the `updatePayload`.                                                                                                                                          |
| `canonical`     | `bool`   | **true**     | Determines if the `filterPayload` and `updatePayload` strings are parsed using MongoDB's Canonical or Relaxed Extended JSON format. **Canonical** is stricter about type representation, while **Relaxed** is more lenient. |
| `upsert`        | `bool`   | `false`      | If `true`, a new document is created if no document matches the `filterPayload`. Defaults to `false`.                                                                                                                       |
