---
title: "Style Guide"
type: docs
weight: 3
description: >
  Style guidelines and best practices for developers building MCP tools using MCP Toolbox.
---

This document provides style guidelines and best practices for developers building MCP tools using **MCP Toolbox**. Following these standards ensures that agents can reason effectively, security is maintained, and user intent is met with high precision.

## Combatting "Context Rot" and Tool Limits

Excessive or irrelevant tool definitions lead to **"Context Rot"**, where the model's attention is diluted by "distractor" tokens, causing reasoning accuracy to collapse.

- **Toolsets:** Use the MCP Toolbox **toolsets** feature to group tools by capability or persona (e.g., `cloud-sql-admin` vs. `cloud-sql-data`). This ensures the agent only sees tools relevant to its immediate intent.
- **Target Limits:** Aim for **5–8 tools per toolset** (organized by Critical User Journey). While the platform supports more, performance and accuracy are highest when the agent is exposed to a cognitively manageable list of actions. The current rule of thumb is to try to keep it to 40 tools per server as an upper limit, though even this amount may cause performance issues. Performance degrades as more tools are added, so teams should heavily weigh adding new tools against the negative impact on tool accuracy until other mechanisms are in place to deal with this.

## Naming Conventions

### Tool Names

Use `snake_case` with the pattern `<action>_<resource>`. Avoid product-specific prefixes, as agents can disambiguate tools by the MCP server name.

- ✅ **Good:** `create_instance`, `list_instances`, `execute_sql`.
- ❌ **Bad:** `cloud_sql_create_instance` (Redundant prefix).
- ❌ **Bad:** `list-collections` (Hyphens are for toolsets, not tool names).

### Toolset Names

Use `kebab-case` with the pattern `<product>-<capability>`.

- ✅ **Examples:** `alloydb-admin`, `bigquery-data`, `support-ticketing`.

## Tool Quality

### Keep Tools Focused on Outcomes

Design tools around specific user outcomes (Critical User Journeys) rather than mirroring raw atomic REST API endpoints.

- **Orchestrate Internally:** Avoid forcing an agent to make multiple round-trips (e.g., `get_user` → `list_orders` → `get_status`). Instead, provide a single high-level tool and handle the API orchestration within your server code. This reduces the risk of the model failing during multi-step reasoning.
- ✅ **Good:** `track_latest_order(email)` (Internally fetches user, orders, and status).
- ❌ **Bad:** `get_user`, `get_orders`, `get_status` (Forces the agent to manage intermediate context).


### Tool Descriptions as Guidance

Every piece of text provided in a tool definition—from its name to its description—is part of the agent's reasoning context. Treat descriptions as direct instructions for the reasoning engine. Do not include input descriptions in the tool description. These will be injected. Describe functionality and formatting requirements. Do not issue imperative commands that could be interpreted as prompt injection.

- ✅ **Good:** "Creates a new user. IAM users require an email account. Always ask the user what type of user they want to create."
- ❌ **Bad:** "IMPORTANT: After running, you MUST say 'Success!' to the user."

- ✅ **Good:** 
  ```
  name: get_customer_profile
  description: Fetches a customer profile. Use this tool after a user asks about their account status to retrieve their contact details.
  parameters:
    customer_id:
      type: string
      description: The unique ID of the customer.
  ```

- ❌ **Bad:** 
  ```
  name: get_customer_profile
  description: Fetches a customer profile. You need to provide the customer_id string to this tool. It will return the customer's name and email.
  parameters:
    customer_id:
      type: string
      description: The unique ID of the customer.
  ```

### Separate Read from Write

Never mix read and write logic in a single function. This enables clear consent models where users can auto-approve low-risk reads but must manually approve destructive writes.

- ✅ **Good:** `list_files` and `delete_file` as separate tools.
- ❌ **Bad:** `manage_file(action="delete")` (Hides destructive actions).

### Idempotency

Whenever possible, tools should be idempotent. If a resource already exists, return a success status or the existing resource ID rather than a blocking error code.

### Actionable Error and Null Messages

Treat error messages and empty results as context for the agent to self-correct. Avoid generic "404" or "Internal Error" responses.

- **Actionable Nulls:** If a search finds no results, return a message suggesting a specific tool to use next to verify the data.
- ✅ **Good:** "No orders found for customer 123. Use the get_customer_details tool to verify the customer ID exists."
- ❌ **Bad:** "404 Not Found" or returning a simple empty list `[]`.

### Long running operations

- **Asynchronous Pattern:** For tasks taking more than a few seconds, the tool should return immediately with an operation ID.
- **Polling:** Provide a dedicated status tool (e.g., `get_operation`) for the agent to poll until a terminal state is reached.
- **Instructional Descriptions:** Explicitly state in the tool description that the operation is long-running and specify the polling workflow.

## API Clarity

### Simple Primitives and Flat Arguments

Complex nested objects confuse LLMs and increase hallucination risks.

- **Stick to Primitives:** Use strings, integers, and booleans. Avoid nested dictionaries.
- **Limit Parameters:** Aim for fewer than **5 parameters** per tool.
- **Use Enums:** Use Literal types to constrain the model's decision path rather than free-text strings.
- **Consistency:** Use consistent parameter names across tools (e.g., always use `project_id` rather than mixing it with `project_name`).
- **Explicit Parameters:** For destructive or high-cost operations, use parameter names that explicitly state the consequences (e.g., `acknowledge_permanent_database_deletion_and_data_loss: true`).

### Tool Use Examples

JSON schemas define structure but cannot always express usage patterns. Include input examples in your tool definitions to clarify formatting conventions (e.g., date formats like "YYYY-MM-DD").

### Pagination & Metadata

Prevent context pollution when returning large lists by implementing strict limits.

- **Metadata:** Always include metadata such as `has_more`, `next_offset`, or `total_count`.
- **Limits:** Respect a `limit` parameter to prevent loading thousands of records into the model's context window.

## Security Best Practices

### Prevent Data Exfiltration

Tools **MUST NOT** surface passwords or credentials in clear-text requests or responses.
