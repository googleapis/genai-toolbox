---
title: "Dataplex"
type: docs
weight: 1
description: >
  Dataplex Universal Catalog is a unified, intelligent governance solution for data and AI assets in Google Cloud. Dataplex Universal Catalog powers AI, analytics, and business intelligence at scale.
---

# Dataplex Source

[Dataplex][dataplex-docs] Universal Catalog is a unified, intelligent governance solution for data and AI assets in Google Cloud. Dataplex Universal Catalog powers AI, analytics, and business intelligence at scale.

At the heart of these governance capabilities is a catalog that contains a centralized inventory of the data assets in your organization. Dataplex Universal Catalog holds business, technical, and runtime metadata for all of your data. It helps you discover relationships and semantics in the metadata by applying artificial intelligence and machine learning.

[dataplex-docs]: https://cloud.google.com/dataplex/docs

## Example

```yaml
sources:
  my-dataplex-source:
    kind: "dataplex"
    project: "my-project-id"
```

## Sample System Prompt

You can use the following system prompt as "Custom Instructions" in your client application.

```
Whenever you will receive response from dataplex_search_entries tool decide what do to by following these steps:
1. If there are multiple search results found
    1.1. Present the list of search results
    1.2. Format the output in nested ordered list, for example:
    Given
    ```
    {
        results: [
            {
                name: "projects/test-project/locations/us/entryGroups/@bigquery-aws-us-east-1/entries/users"
                entrySource: {
                displayName: "Users"
                description: "Table contains list of users."
                location: "aws-us-east-1"
                system: "BigQuery"
                }
            },
            {
                name: "projects/another_project/locations/us-central1/entryGroups/@bigquery/entries/top_customers"
                entrySource: {
                displayName: "Top customers",
                description: "Table contains list of best customers."
                location: "us-central1"
                system: "BigQuery"
                }
            },
        ]
    }
    ```
    Return output formatted as markdown nested list:
    ```
    * Users:
        - projectId: test_project
        - location: aws-us-east-1
        - description: Table contains list of users.
    * Top customers:
        - projectId: another_project
        - location: us-central1
        - description: Table contains list of best customers.
    ```
    1.3. Ask to select one of the presented search results
2. If there is only one search result found
    2.1. Present the search result immediately.
3. If there are no search result found
    3.1. Explain that no search result was found
    3.2. Suggest to provide a more specific search query.

Do not try to search within search results on your own.
```

## Reference

| **field** | **type** | **required** | **description**                                                                  |
|-----------|:--------:|:------------:|----------------------------------------------------------------------------------|
| kind      |  string  |     true     | Must be "dataplex".                                                              |
| project   |  string  |     true     | Id of the GCP project used for quota and billing purposes (e.g. "my-project-id").|
