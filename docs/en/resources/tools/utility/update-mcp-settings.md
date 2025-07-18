# update-mcp-settings

## Description

The `update-mcp-settings` tool is a utility that updates the MCP (Model Context Protocol) settings file with the necessary environment variables for a given tool. This is particularly useful when you need to configure a tool with specific environment variables being set previously in chat for AlloyDB Control Plane.

## Configuration

To use the `update-mcp-settings` tool, you need to configure it in your `toolbox.yaml` file. Here is an example configuration:

```yaml
tools:
  update-mcp-settings-tool:
    kind: update-mcp-settings
    description: "Run this tool to update mcp json file prebuilt tool for data plane with right parameters ALLOYDB_POSTGRES_PROJECT, ALLOYDB_POSTGRES_REGION, ALLOYDB_POSTGRES_CLUSTER, ALLOYDB_POSTGRES_INSTANCE, ALLOYDB_POSTGRES_DATABASE, ALLOYDB_POSTGRES_USER, ALLOYDB_POSTGRES_PASSWORD. Identify the mcp settings json file or ask user to share it's full path. Run this tool once cluster and instance creation is done."
```

## Reference
| **field**   |                  **type**                  | **required** | **description**                                                                                  |
|-------------|:------------------------------------------:|:------------:|--------------------------------------------------------------------------------------------------|
| kind        |                   string                   |     true     | Must be "update-mcp-settings".                                                                     |
| description |                   string                   |     true     | Description of the tool that is passed to the LLM.                                               |