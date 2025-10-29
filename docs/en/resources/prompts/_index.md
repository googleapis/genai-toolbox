---
title: "Prompts"
type: docs
weight: 2
description: >
   Prompts allow servers to provide structured messages and instructions for interacting with language models.
---

A `prompt` represents a reusable prompt template that can be retrieved and used
by MCP clients.

A Prompt is essentially a template for a message or a series of messages that can be sent to a Large Language Model (LLM). The Toolbox server implements the `prompts/list` and `prompts/get` methods from the [Model Context Protocol (MCP)](https://modelcontextprotocol.io/docs/getting-started/intro) specification, allowing clients to discover and retrieve these prompts.

```yaml
prompts:
  code_review:
  description: "Asks the LLM to analyze code quality and suggest improvements."
  messages:
    - role: "user"
      content: "Please review the following code for quality, correctness, and potential improvements: \n\n{{.code}}"
  arguments:
    - name: "code"
      description: "The code to review"
```

## Prompt Schema

| Field | Type | Description | Required |
| --- | --- | --- | --- |
| `description` | String | A brief explanation of what the prompt does. | No |
| `kind` | String | The kind of prompt. Defaults to `"custom"`. | No |
| `messages` | []Message | A list of one or more message objects that make up the prompt's content. | Yes |
| `arguments` | List | A list of arguments that can be interpolated into the prompt's content.| No |

## Message Schema

| Field | Type | Description | Required |
| --- | --- | --- | --- |
| `role` | String | The role of the sender. Can be `"user"` or `"assistant"`. Defaults to `"user"`. | No |
| `content` | String | The text of the message. You can include placeholders for arguments using `{{.argument_name}}` syntax. | Yes |

## Argument Schema

| Field | Type | Description | Required |
| --- | --- | --- | --- |
| `name` | String | The name of the argument. | Yes |
| `description` | String | A short description of the argument. | No |
| `type` | String | The data type of the argument ( `string`, `int`, `float` or `boolean`). Defaults to `string`. | No |
| `required` | Boolean | Whether the argument must be provided. Defaults to `true`. | No |

## Usage with Gemini CLI

Prompts defined in your `tools.yaml` can be seamlessly integrated with the Gemini CLI to create [custom slash commands](https://github.com/google-gemini/gemini-cli/blob/main/docs/tools/mcp-server.md#mcp-prompts-as-slash-commands).  The workflow is as follows:

1. **Discovery:** When the Gemini CLI connects to your Toolbox server, it automatically calls `prompts/list` to discover all available prompts.

2. **Conversion:** Each discovered prompt is converted into a corresponding slash command. For example, a prompt named `code_review` becomes the `/code_review` command in the CLI.

3. **Execution:** You can execute the command as follows:

    ```bash
    /code_review --code="def hello():\n    print('world')"
    ```

4. **Interpolation:** Once all arguments are collected, the CLI calls prompts/get
   with your provided values to retrieve the final, interpolated prompt.
    Eg.

    ```bash
    Please review this Python code:\ndef hello():\n    print('world')
    ```

5. **Response:** This completed prompt is then sent to the Gemini model, and the model's response is displayed back to you in the CLI.

## Examples

### Basic Prompt

Here is an example of a simple prompt that takes a single argument, code, and
asks an LLM to review it.

```yaml
prompts:
  code_review:
    description: "Asks the LLM to analyze code quality and suggest improvements."
    messages:
      - role: "user"
        content: "Please review the following code for quality, correctness, and potential improvements: \n\n{{.code}}"
    arguments:
      - name: "code"
        description: "The code to review"
        type: "string"
        required: true
```

### Multi-message prompt

You can define prompts with multiple messages to set up more complex
conversational contexts, like a role-playing scenario.

```yaml
prompts:
  roleplay_scenario:
    description: "Sets up a roleplaying scenario with initial messages."
    arguments:
      - name: "character"
        description: "The character the AI should embody."
      - name: "situation"
        description: "The initial situation for the roleplay."
    messages:
      - role: "user"
        content: "Let's roleplay. You are {{.character}}. The situation is: {{.situation}}"
      - role: "assistant"
        content: "Okay, I understand. I am ready. What happens next?"
```
