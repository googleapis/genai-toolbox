---
title: "Custom"
type: docs
weight: 1
description: > 
  Custom prompts defined by the user.
---

Custom prompts are defined by the user to be exposed through their MCP server.
They are the default type for prompts.

## Examples

### Basic Prompt

Here is an example of a simple prompt that takes a single argument, code, and
asks an LLM to review it.

```yaml
prompts:
  code_review:
    description: "Asks the LLM to analyze code quality and suggest improvements."
    messages:
      - content: "Please review the following code for quality, correctness, and potential improvements: \n\n{{.code}}"
    arguments:
      - name: "code"
        description: "The code to review"
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

## Reference

### Prompt Schema

| **field** | **type** | **required** | **description** |
| --- | --- | --- | --- |
| description | string | No | A brief explanation of what the prompt does. |
| kind | string | No | The kind of prompt. Defaults to `"custom"`. |
| messages | []Message | Yes | A list of one or more message objects that make up the prompt's content. |
| arguments | []Argument | No | A list of arguments that can be interpolated into the prompt's content.|

### Message Schema

| **field** | **type** | **required** | **description** |
| --- | --- | --- | --- |
| role | string | No | The role of the sender. Can be `"user"` or `"assistant"`. Defaults to `"user"`. |
| content | string | Yes | The text of the message. You can include placeholders for arguments using `{{.argument_name}}` syntax. |

### Argument Schema

An argument is a [Parameter](../../tools/_index.md#specifying-parameters) with a single change:

- The type for an argument is not required but defaults to `string`.
