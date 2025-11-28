# Toolbox

A CLI tool for running a toolbox server.

## Installation

You can install the toolbox globally:

```bash
npm install -g @toolbox-sdk/server
```

Or run it directly using npx:

```bash
npx @toolbox-sdk/server
```

## Configuration

The toolbox requires a `tools.yaml` file in the current working directory to define sources, tools, and prompts.

### Example `tools.yaml`

```yaml
sources:
  my-pg-source:
    kind: postgres
    host: 127.0.0.1
    port: 5432
    database: toolbox_db
    user: postgres
    password: password
tools:
  search-hotels-by-name:
    kind: postgres-sql
    source: my-pg-source
    description: Search for hotels based on name.
    parameters:
      - name: name
        type: string
        description: The name of the hotel.
    statement: SELECT * FROM hotels WHERE name ILIKE '%' || $1 || '%';
prompts:
  code-review:
    description: "Asks the LLM to analyze code quality and suggest improvements."
    messages:
      - role: "user"
        content: "Please review the following code for quality, correctness, and potential improvements: \n\n{{.code}}"
    arguments:
      - name: "code"
        description: "The code to review"
        required: true
```

To learn more on how to configure your toolbox, visit the [official docsite](https://googleapis.github.io/genai-toolbox/getting-started/configure/).

## Platform Support

The toolbox automatically handles platform-specific binaries. Supported platforms include:
- macOS (arm64, x64)
- Linux (x64)

Note: Windows is not supported yet.

## Resources

For more information, visit the 
- [MCP Toolbox repository](https://github.com/googleapis/genai-toolbox)
- [Official Documentation](https://googleapis.github.io/genai-toolbox/getting-started/introduction/)