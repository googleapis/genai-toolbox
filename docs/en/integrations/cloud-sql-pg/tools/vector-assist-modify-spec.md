---
title: "vector-assist-modify-spec"
type: docs
weight: 1
description: >
  The "vector-assist-modify-spec" tool modifies an existing vector specification
  with new parameters or overrides, recalculating the generated SQL
  recommendations to match the updated requirements.
---

## About

The `vector-assist-modify-spec` tool modifies an existing vector specification (identified by a required `spec_id`) with new parameters or overrides. Upon modification, it automatically recalculates and refreshes the list of generated recommendations by `vector_assist.define-spec` to match the updated spec requirements. 

Use this tool when a user or agent wants to adjust or fine-tune the configuration of an already defined vector spec (such as changing the target recall, embedding model, or quantization) before actually executing the setup commands. Under the hood, this tool connects to the target database and executes the `vector_assist.modify_spec` function to generate the updated specifications.

## Compatible Sources

{{< compatible-sources >}}

## Requirements

{{< notice tip >}} 
Ensure that your target PostgreSQL database has the required `vector_assist` extension installed, and that the `vector_assist.modify_spec` function is available in order for this tool to execute successfully.
{{< /notice >}}

## Parameters

The tool takes the following input parameters:

| Parameter                | Type    | Description                                                            | Required |
| :----------------------- | :------ | :--------------------------------------------------------------------- | :------- |
| `spec_id`                | string  | Unique ID of the vector specification to modify.                       | Yes      |
| `table_name`             | string  | New table name for the vector workload setup.                          | No       |
| `schema_name`            | string  | New schema name containing the target table.                           | No       |
| `vector_column_name`     | string  | New name for the column containing vector embeddings.                  | No       |
| `text_column_name`       | string  | New name for the text column for vector search.                        | No       |
| `vector_index_type`      | string  | New vector index type ('hnsw', 'ivfflat', or 'scann').                 | No       |
| `embeddings_available`   | boolean | Update if vector embeddings already exist in the table.                | No       |
| `num_vectors`            | integer | Update the expected total number of vectors.                           | No       |
| `dimensionality`         | integer | Update the dimension of vectors or the embedding model.                | No       |
| `embedding_model`        | string  | Update the model used for generating vector embeddings.                | No       |
| `prefilter_column_names` | array   | Update the columns used for prefiltering vector queries.               | No       |
| `distance_func`          | string  | Update the distance function ('cosine', 'ip', 'l2', 'l1').             | No       |
| `quantization`           | string  | Update the quantization method ('none', 'halfvec', 'bit').             | No       |
| `memory_budget_kb`       | integer | Update maximum memory (in KB) for index building.                      | No       |
| `target_recall`          | float   | Update the target recall rate for the index.                           | No       |
| `target_top_k`           | integer | Update the number of top results (top-K) to retrieve.                  | No       |
| `tune_vector_index`      | boolean | Update whether automatic tuning is required for the index.             | No       |

> Note
> Parameters are marked as required or optional based on the vector assist function definitions. 
> The function may perform further validation on optional parameters to ensure all necessary 
> data is available before returning a response.

## Example

```yaml
kind: tool
name: modify_spec
type: vector-assist-modify-spec
source: my-database-source
description: "This tool modifies an existing vector specification (identified by a required spec_id) with new parameters or overrides. Upon modification, it automatically recalculates and refreshes the list of generated SQL recommendations to match the updated requirements. Use this tool when a user wants to adjust or fine-tune the configuration of an already defined vector spec (such as changing the target recall, embedding model, or quantization) before actually executing the setup commands."
```

## Reference

| **field**   | **type** | **required** | **description**                                      |
|-------------|:--------:|:------------:|------------------------------------------------------|
| type        |  string  |     true     | Must be "vector-assist-modify-spec".                 |
| source      |  string  |     true     | Name of the source the SQL should execute on.        |
| description |  string  |    false     | Description of the tool that is passed to the agent. |