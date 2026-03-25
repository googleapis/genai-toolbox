---
title: "HF Inference OAI"
type: docs
weight: 2
description: >
  Use OpenAI-compatible embedding endpoints, including local servers and Hugging Face routed providers.
---

## About

`hf-inference-oai` is for embedding endpoints that expose an OpenAI-compatible
`/v1/embeddings` API. This includes local servers such as `llama.cpp`, and
Hugging Face routed provider endpoints that use the same request and response
format.

## Example

```yaml
kind: embeddingModels
name: local-jina
type: hf-inference-oai
baseUrl: http://127.0.0.1:8080
model: jinaai/jina-embeddings-v5-text-small-clustering-GGUF
```

Routed provider example:

```yaml
kind: embeddingModels
name: qwen-provider
type: hf-inference-oai
baseUrl: https://router.huggingface.co/scaleway
model: qwen3-embedding-8b
apiKey: ${HF_TOKEN}
```

## Reference

| **field** | **type** | **required** | **description** |
|-----------|:--------:|:------------:|-----------------|
| type      |  string  |     true     | Must be `hf-inference-oai`. |
| baseUrl   |  string  |     true     | Base URL for the OpenAI-compatible embeddings endpoint. |
| model     |  string  |     true     | Model name sent in the `/v1/embeddings` request. |
| apiKey    |  string  |    false     | Bearer token used for authentication. |
