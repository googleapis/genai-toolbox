---
title: "Pre and Post processing"
type: docs
weight: 1
description: >
  Pre and Post processing in GenAI applications.
---

Pre and post processing allow developers to intercept and modify interactions between the agent and its tools or the user. This capability is essential for building robust, secure, and compliant agents.

## Types of Processing

### Pre-processing

Pre-processing occurs before a tool is executed or an agent processes a message. Key types include:

- **Input Sanitization & Redaction**: Detecting and masking sensitive information (like PII) in user queries or tool arguments to prevent it from being logged or sent to unauthorized systems.
- **Business Logic Validation**: Verifying that the proposed action complies with business rules (e.g., ensuring a requested hotel stay does not exceed 14 days, or checking if a user has sufficient permission).
- **Security Guardrails**: Analyzing inputs for potential prompt injection attacks or malicious payloads.

### Post-processing

Post-processing occurs after a tool has executed or the model has generated a response. Key types include:

- **Response Enrichment**: Injecting additional data into the tool output that wasn't part of the raw API response (e.g., calculating loyalty points earned based on the booking value).
- **Output Formatting**: Transforming raw data (like JSON or XML) into a more human-readable or model-friendly format to improve the agent's understanding.
- **Compliance Auditing**: Logging the final outcome of transactions, including the original request and the result, to a secure audit trail.

## Processing Scopes

Processing logic can be applied at different levels of the application:

### Tool Level

Wraps individual tool executions. This is best for logic specific to a single tool or a set of tools.

- **Scope**: Intercepts the raw inputs (arguments) to a tool and its outputs.
- **Use Cases**: Argument validation, output formatting, specific privacy rules for sensitive tools.

### Model Level

Intercepts individual calls to the Large Language Model (LLM).

- **Scope**: Intercepts the list of messages (prompt) sent to the model and the generation (response) received.
- **Use Cases**: Global PII redaction (across all tools/chat), prompt engineering/injection, token usage tracking, and hallucination detection.

### Agent Level

Wraps the high-level agent execution loop (e.g., a "turn" in the conversation).

- **Scope**: Intercepts the initial user input and the final agent response, enveloping one or more model calls and tool executions.
- **Use Cases**: User authentication, rate limiting, session management, and end-to-end audit logging.
