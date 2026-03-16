---
title: "HF Inference"
type: docs
weight: 3
description: >
  Use Hugging Face's direct inference API for embedding models exposed via task endpoints.
---

## About

`hf-inference` is for Hugging Face's direct inference API when a model is
served through a task-specific endpoint such as
`/pipeline/feature-extraction`.

This is useful for models like `sentence-transformers/all-MiniLM-L6-v2`.

## Example

```yaml
kind: embeddingModels
name: mini-lm
type: hf-inference
model: sentence-transformers/all-MiniLM-L6-v2
apiKey: ${HF_TOKEN}
task: feature-extraction
```

## Reference

| **field** | **type** | **required** | **description** |
|-----------|:--------:|:------------:|-----------------|
| type      |  string  |     true     | Must be `hf-inference`. |
| model     |  string  |     true     | Hugging Face model ID. |
| apiKey    |  string  |    false     | Bearer token used for authentication. |
| baseUrl   |  string  |    false     | Base URL for the HF inference router. Defaults to `https://router.huggingface.co/hf-inference`. |
| task      |  string  |    false     | Task path to call. Currently `feature-extraction`. |
