---
title: "Securing Toolbox with Model Armor"
type: docs
weight: 1
description: >
  Protect your agents and tools against prompt injection and sensitive data
  leakage by screening traffic with Google Cloud Model Armor.
---

## About

[Google Cloud Model Armor](https://docs.cloud.google.com/model-armor/overview) is
an LLM-agnostic service that screens prompts and responses to defend AI
applications against prompt injection, jailbreaks, and sensitive data leakage.
Pairing it with MCP Toolbox lets you enforce these protections on both the
prompts your users send and the requests your agent makes to tools, without
trusting the model to police itself.

Model Armor inspects traffic at two points:

- **Ingress (client → agent):** the user prompt is screened *before* it reaches
  the agent, blocking prompt injection and jailbreak attempts.
- **Egress (agent → tool / anywhere):** the agent's tool requests and the
  responses returned by those tools are screened to mask or block sensitive data
  (PII, secrets) before it is processed or surfaced.

```mermaid
flowchart LR
    User([User])
    Agent[Agent / LLM]
    Tool[(MCP Toolbox<br/>tool + data source)]

    subgraph MA["Google Cloud Model Armor"]
        In{{Ingress filter}}
        Out{{Egress filter}}
    end

    User -->|prompt| In
    In -->|sanitized prompt| Agent
    Agent -->|tool call + args| Out
    Out -->|sanitized request| Tool
    Tool -->|raw response| Out
    Out -->|sanitized response| Agent
    Agent -->|answer| User

    In -.->|block: prompt injection / jailbreak| User
    Out -.->|mask / block: PII, secrets| Agent

    classDef armor fill:#e8f0fe,stroke:#1a73e8,color:#174ea6;
    class In,Out armor;
```

In short, Model Armor inspects the user prompt before it reaches the agent, and
inspects every tool request and response before and after the tool runs.

{{< notice note >}}
Like other [pre- and post-processing](../configuration/pre-post-processing/)
guardrails, these checks live in your orchestration layer (LangChain, ADK, Agent
Gateway), not in the Toolbox SDK itself. Toolbox tools are designed to work
cleanly with this kind of interception.
{{< /notice >}}

{{< notice tip >}}
This guide covers the Python + LangChain integration. Examples for ADK, Agent
Gateway, Google Cloud MCP servers, and other languages (Go, Node.js, Java) are
being added in follow-up guides.
{{< /notice >}}

## Requirements

1. **Enable the API.** Enable `modelarmor.googleapis.com` in your Google Cloud
   project.
2. **Grant IAM roles.** The identity that runs your agent needs
   `roles/modelarmor.user` to invoke sanitization. To create and manage
   templates, you also need `roles/modelarmor.admin`. See the
   [access control reference](https://docs.cloud.google.com/model-armor/overview)
   for the full list of roles.

## Step 1: Configure a Model Armor template

Model Armor applies its filters through a **template**. Create one that enforces
both Sensitive Data Protection (SDP) and prompt injection / jailbreak detection:

```bash
gcloud model-armor templates create my-mcp-template \
    --location=us-central1 \
    --project=YOUR_PROJECT_ID \
    --basic-config-filter-enforcement=enabled \
    --pi-and-jailbreak-filter-settings-enforcement=enabled \
    --pi-and-jailbreak-filter-settings-confidence-level=MEDIUM_AND_ABOVE
```

{{< notice note >}}
Basic SDP automatically scans for high-confidence secrets such as credit card
numbers, API keys, and passwords. For granular PII detection and masking, use an
advanced SDP configuration with `--advanced-config-inspect-template` and
`--advanced-config-deidentify-template`. See
[Sanitize prompts and responses](https://docs.cloud.google.com/model-armor/sanitize-prompts-responses)
for details.
{{< /notice >}}

## Step 2: Secure ingress and egress

{{< tabpane persist=header >}}
{{% tab header="Python (LangChain)" text=true %}}

If your agent uses LangChain, the `langchain-google-community` package provides
runnables and middleware that screen prompts and responses with Model Armor.

1. Install the dependencies:

    ```bash
    pip install "langchain>=1.0" "langchain-google-community>=3.0.4" langchain-google-genai toolbox-langchain
    ```

2. Create an **ingress** sanitizer for user prompts and an **egress** sanitizer
   for responses. Set `fail_open=False` so execution is blocked when a threat is
   detected:

    ```python
    from langchain_google_community.model_armor import (
        ModelArmorSanitizePromptRunnable,
        ModelArmorSanitizeResponseRunnable,
    )

    PROJECT_ID = "YOUR_PROJECT_ID"
    LOCATION = "us-central1"
    TEMPLATE_ID = "my-mcp-template"

    # Ingress: screen the user prompt before it reaches the model.
    sanitize_prompt = ModelArmorSanitizePromptRunnable(
        project=PROJECT_ID,
        location=LOCATION,
        template_id=TEMPLATE_ID,
        fail_open=False,
    )

    # Egress: screen the response before it returns to the user.
    sanitize_response = ModelArmorSanitizeResponseRunnable(
        project=PROJECT_ID,
        location=LOCATION,
        template_id=TEMPLATE_ID,
        fail_open=False,
    )
    ```

3. For a simple chain, place the sanitizers around the model. A blocked prompt or
   response raises `ValueError`:

    ```python
    from langchain_google_genai import ChatGoogleGenerativeAI

    llm = ChatGoogleGenerativeAI(model="gemini-2.5-flash")
    chain = sanitize_prompt | llm | sanitize_response

    try:
        result = chain.invoke("Summarize today's bookings.")
        print(result.content)
    except ValueError as e:
        print(f"Model Armor blocked the content: {e}")
    ```

4. For an agent that calls Toolbox tools, wrap the sanitizers in
   `ModelArmorMiddleware` and pass it to `create_agent`. This screens the
   intermediate tool calls and responses (agent-to-tool egress) in addition to
   the user-facing prompt and answer:

    {{< notice note >}}
  `create_agent` and the `middleware` parameter are part of
  [LangChain v1.0](https://docs.langchain.com/oss/python/releases/langchain-v1)
  (`langchain>=1.0`). On older releases, agents were built with
  `create_react_agent` / `create_tool_calling_agent`, which do not support
  middleware. Upgrade to v1.0 to use this pattern.
    {{< /notice >}}

    ```python
    import asyncio

    from langchain.agents import create_agent
    from langchain_google_community.model_armor import ModelArmorMiddleware
    from langchain_google_genai import ChatGoogleGenerativeAI
    from toolbox_langchain import ToolboxClient


    async def main():
        async with ToolboxClient("http://127.0.0.1:5000") as client:
            tools = await client.aload_toolset("my-toolset")

            model_armor = ModelArmorMiddleware(
                prompt_sanitizer=sanitize_prompt,
                response_sanitizer=sanitize_response,
            )

            agent = create_agent(
                model=ChatGoogleGenerativeAI(model="gemini-2.5-flash"),
                tools=tools,
                middleware=[model_armor],
            )

            response = await agent.ainvoke(
                {"messages": [{"role": "user", "content": "Find hotels in Basel."}]}
            )
            print(response["messages"][-1].content)


    if __name__ == "__main__":
        asyncio.run(main())
    ```

For more on the middleware, see the
[Model Armor LangChain integration](https://docs.cloud.google.com/model-armor/model-armor-langchain-integration).

{{% /tab %}}
{{< /tabpane >}}

## Additional Resources

- [Model Armor overview](https://docs.cloud.google.com/model-armor/overview)
- [Sanitize prompts and responses](https://docs.cloud.google.com/model-armor/sanitize-prompts-responses)
- [Model Armor LangChain integration](https://docs.cloud.google.com/model-armor/model-armor-langchain-integration)
- [Codelab: Secure your agent with Model Armor](https://codelabs.developers.google.com/secure-agent-modelarmor#9)
- [Toolbox pre- and post-processing](../configuration/pre-post-processing/)
