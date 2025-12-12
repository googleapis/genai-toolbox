---
title: "Gemini Embedding"
type: docs
weight: 1
description: >
  Use Google's Gemini models to generate high-performance text embeddings for vector databases.
---

## Getting Started

Google Gemini provides state-of-the-art embedding models that convert text into
high-dimensional vectors. To use Gemini embeddings, you must first obtain an API
Key from the [Google AI Studio][ai-studio].

Once you have your key, configure the `embeddingModels` section in your
`tools.yaml` with the model identifier (e.g., `gemini-embedding-001`) and the
dimensionality of the vector output.

[ai-studio]: https://aistudio.google.com/app/apikey

## Behavior

### Automatic Vectorization

When a tool parameter is configured with `embeddedBy: <your-gemini-model-name>`,
the Toolbox intercepts the raw text input from the client and sends it to the
Gemini API. The resulting numerical array is then formatted before being passed
to your database source.

### Dimension Matching

The `dimension` field must match the expected size of your database column
(e.g., a `vector(768)` column in PostgreSQL). This setting is supported by newer
models since 2024 only. You cannot set this value if using the earlier model
(`models/embedding-001`).

## Example

```yaml
embeddingModels:
  gemini-model:
    kind: gemini
    model: gemini-embedding-001
    apiKey: ${GOOGLE_API_KEY}
    dimension: 768
```

{{< notice tip >}}
Use environment variable replacement with the format ${ENV_NAME}
instead of hardcoding your secrets into the configuration file.
{{< /notice >}}

## Reference

| **field** | **type** | **required** | **description**                                              |
|-----------|:--------:|:------------:|--------------------------------------------------------------|
| kind      |  string  |     true     | Must be `gemini`.                                            |
| model     |  string  |     true     | The Gemini model ID to use (e.g., `gemini-embedding-001`).   |
| apiKey    |  string  |     true     | Your API Key from Google AI Studio.                          |
| dimension | integer  |    false     | The number of dimensions in the output vector (e.g., `768`). |
