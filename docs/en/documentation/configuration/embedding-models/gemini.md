---
title: "Gemini Embedding"
type: docs
weight: 1
description: >
  Use Google's Gemini models to generate high-performance text embeddings for
  vector databases.
---

## About

Google Gemini provides state-of-the-art embedding models that convert text into
high-dimensional vectors.

### Authentication

Toolbox supports two authentication modes:

1.  **Google AI (API Key):** Used if you
    provide `apiKey` (or set `GOOGLE_API_KEY`/`GEMINI_API_KEY` environment
    variables). This uses the [Google AI Studio][ai-studio] backend.
2.  **Vertex AI (ADC):** Used if provided `project` and `location` (or set
    `GOOGLE_CLOUD_PROJECT`/`GOOGLE_CLOUD_LOCATION` environment variables). This uses [Application
    Default Credentials (ADC)][adc].

We recommend using an API key for quick testing and using Vertex AI with ADC for
production environments.

[adc]: https://cloud.google.com/docs/authentication#adc
[api-key]: https://ai.google.dev/gemini-api/docs/api-key#api-keys
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
(`models/embedding-001`). Check out [available Gemini models][modellist] for
more information.

[modellist]:
  https://docs.cloud.google.com/vertex-ai/generative-ai/docs/embeddings/get-text-embeddings#supported-models

## Example

### Using Google AI

Google AI uses API Key for authentication. You can get an API key from [Google
AI Studio][ai-studio].

```yaml
kind: embeddingModel
name: gemini-model
type: gemini
model: gemini-embedding-001
apiKey: ${GOOGLE_API_KEY}
dimension: 768
```

### Using Vertex AI

Vertex AI uses Application Default Credentials (ADC) for authentication. Learn
how to set up ADC [here][adc].

```yaml
kind: embeddingModel
name: gemini-model
type: gemini
model: gemini-embedding-001
project: ${GOOGLE_CLOUD_PROJECT}
location: us-central1
dimension: 768
```

[adc]: https://docs.cloud.google.com/docs/authentication/provide-credentials-adc

{{< notice tip >}} Use environment variable replacement with the format
${ENV_NAME} instead of hardcoding your secrets into the configuration file.
{{< /notice >}}

## Reference

| **field**   | **type** | **required** | **description**                                                                                                                                      |
| ----------- | :------: | :----------: | ---------------------------------------------------------------------------------------------------------------------------------------------------- |
| type        |  string  |     true     | Must be `gemini`.                                                                                                                                    |
| model       |  string  |     true     | The Gemini model ID to use (e.g., `gemini-embedding-001`).                                                                                             |
| dimension   | integer  |    false     | The number of dimensions in the output vector (e.g., `768`).                                                                                         |
