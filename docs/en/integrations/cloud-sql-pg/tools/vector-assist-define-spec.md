---
title: "vector-assist-define-spec"
type: docs
weight: 1
description: >
  The "vector-assist-define-spec" tool defines a new vector specification by
  capturing the user's intent and requirements for a vector search workload,
  generating SQL recommendations for setting up database, embeddings, and
  vector indexes.
---

## About

The `vector-assist-define-spec` tool defines a new vector specification by capturing the user's intent and requirements for a vector search workload. It generates a complete, ordered set of SQL recommendations required to set up the database, embeddings, and vector indexes. 

Use this tool at the very beginning of the vector setup process when an agent or user first wants to configure a table for vector search, generate embeddings, or create a new vector index. Under the hood, this tool connects to the target database and executes the `vector_assist.define_spec` function to generate the necessary specifications.

## Compatible Sources

{{< compatible-sources >}}

## Requirements

{{< notice tip >}} 
Ensure that your target PostgreSQL database has the required `vector_assist` extension installed, in order for this tool to execute successfully.
{{< /notice >}}

## Parameters

The tool takes the following input parameters:

| Parameter                | Type    | Description                                                            | Required |
| :----------------------- | :------ | :--------------------------------------------------------------------- | :------- |
| `table_name`             | string  | Target table name for setting up the vector workload.                  | Yes      |
| `schema_name`            | string  | Name of the schema containing the target table.                        | No       |
| `spec_id`                | string  | Unique ID for the vector specification; auto-generated if omitted.     | No       |
| `vector_column_name`     | string  | Name of the column containing the vector embeddings.                   | No       |
| `text_column_name`       | string  | Name of the text column for setting up vector search.                  | No       |
| `vector_index_type`      | string  | Type of vector index ('hnsw', 'ivfflat', or 'scann').                  | No       |
| `embeddings_available`   | boolean | Indicates if vector embeddings already exist in the table.             | No       |
| `num_vectors`            | integer | Expected total number of vectors in the dataset.                       | No       |
| `dimensionality`         | integer | Dimension of existing vectors or the chosen embedding model.           | No       |
| `embedding_model`        | string  | Model to be used for generating vector embeddings.                     | No       |
| `prefilter_column_names` | array   | List of columns to use for prefiltering vector queries.                | No       |
| `distance_func`          | string  | Distance function for comparing vectors ('cosine', 'ip', 'l2', 'l1').  | No       |
| `quantization`           | string  | Quantization method for vector indexes ('none', 'halfvec', 'bit').     | No       |
| `memory_budget_kb`       | integer | Maximum memory (in KB) the index can use during build.                 | No       |
| `target_recall`          | float   | Target recall rate for standard vector queries using this index.       | No       |
| `target_top_k`           | integer | Number of top results (top-K) to retrieve per query.                   | No       |
| `tune_vector_index`      | boolean | Indicates whether automatic tuning is required for the index.          | No       |

> Note
> Parameters are marked as required or optional based on the vector assist function definitions. 
> The function may perform further validation on optional parameters to ensure all necessary 
> data is available before returning a response.

## Example

```yaml
kind: tool
name: define_spec
type: vector-assist-define-spec
source: my-database-source
description: "This tool defines a new vector specification by capturing the user's intent and requirements for a vector search workload. This generates a complete, ordered set of SQL recommendations required to set up the database, embeddings, and vector indexes. Use this tool at the very beginning of the vector setup process when a user first wants to configure a table for vector search, generate embeddings, or create a new vector index."
```

## Reference

| **field**   | **type** | **required** | **description**                                      |
|-------------|:--------:|:------------:|------------------------------------------------------|
| type        |  string  |     true     | Must be "vector-assist-define-spec".                 |
| source      |  string  |     true     | Name of the source the SQL should execute on.        |
| description |  string  |    false     | Description of the tool that is passed to the agent. |