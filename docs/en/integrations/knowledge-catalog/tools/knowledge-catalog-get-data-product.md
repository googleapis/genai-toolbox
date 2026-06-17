---
title: "dataplex-get-data-product"
type: docs
weight: 2
description: >
  A "dataplex-get-data-product" tool allows to retrieve a specific Data Product.
aliases:
  - /integrations/dataplex/tools/dataplex-get-data-product/
---

## About

A `dataplex-get-data-product` tool retrieves detailed metadata for a specific Data Product in Knowledge Catalog (formerly known as Dataplex).

`dataplex-get-data-product` requires the following parameters:

- `locationId` - The location ID (e.g., `us`, `us-central1`) where the Data Product is located.
- `dataProductId` - The unique ID of the Data Product.

## Compatible Sources

{{< compatible-sources >}}

## Requirements

### IAM Permissions

To retrieve a data product, your authenticated identity must have the following IAM permissions:
*   `dataplex.dataProducts.get` (usually included in `roles/dataplex.viewer` or `roles/dataplex.developer`).

Refer to the main [Knowledge Catalog Source Requirements](../source.md#requirements) for details on setting up Application Default Credentials (ADC).

## Example

```yaml
kind: tool
name: get_data_product
type: dataplex-get-data-product
source: my-dataplex-source
description: Use this tool to retrieve a Data Product.
```

## Reference

| **field**   | **type** | **required** | **description**                                    |
|-------------|:--------:|:------------:|----------------------------------------------------|
| type        |  string  |     true     | Must be "dataplex-get-data-product".               |
| source      |  string  |     true     | Name of the source the tool should execute on.     |
| description |  string  |     true     | Description of the tool that is passed to the LLM. |
