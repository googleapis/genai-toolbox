---
title: "Generate Agent Skills"
type: docs
weight: 10
description: >
  How to generate agent skills from a toolset.
---

The `skills-generate` command allows you to convert a **toolset** into an **Agent Skill**. A toolset is a collection of tools, and the generated skill will contain metadata and execution scripts for all tools within that toolset, complying with the [Agent Skill specification](https://agentskills.io/specification).

## Before you begin

1. Make sure you have the `toolbox` executable in your PATH.
2. Make sure you have [Node.js](https://nodejs.org/) installed on your system.

## Generating a Skill from a Toolset

A skill package consists of a `SKILL.md` file (with required YAML frontmatter) and a set of Node.js scripts. Each tool defined in your toolset maps to a corresponding script in the generated Node.js scripts (`.js`) that work across different platforms (Linux, macOS, Windows).


### Command Usage

The basic syntax for the command is:

```bash
toolbox <tool-source> skills-generate \
  --name <skill-name> \
  --toolset <toolset-name> \
  --description <description> \
  --output-dir <output-directory>
```

- `<tool-source>`: Can be `--tools-file`, `--tools-files`, `--tools-folder`, and `--prebuilt`. See the [CLI Reference](../reference/cli.md) for details.
- `--name`: Name of the generated skill.
- `--description`: Description of the generated skill.
- `--toolset`: (Optional) Name of the toolset to convert into a skill. If not provided, all tools will be included.
- `--output-dir`: (Optional) Directory to output generated skills (default: "skills").

{{< notice note >}}
**Note:** The `<skill-name>` must follow the Agent Skill [naming convention](https://agentskills.io/specification): it must contain only lowercase alphanumeric characters and hyphens, cannot start or end with a hyphen, and cannot contain consecutive hyphens (e.g., `my-skill`, `data-processing`).
{{< /notice >}}

### Example: Custom Tools File

1. Create a `tools.yaml` file with a toolset and some tools:

   ```yaml
   tools:
     tool_a:
       description: "First tool"
       run:
         command: "echo 'Tool A'"
     tool_b:
       description: "Second tool"
       run:
         command: "echo 'Tool B'"
   toolsets:
     my_toolset:
       tools:
         - tool_a
         - tool_b
   ```

2. Generate the skill:

   ```bash
   toolbox --tools-file tools.yaml skills-generate \
     --name "my-skill" \
     --toolset "my_toolset" \
     --description "A skill containing multiple tools" \
     --output-dir "generated-skills"
   ```

3. The generated skill directory structure:

   ```text
   generated-skills/
   └── my-skill/
       ├── SKILL.md
       ├── assets/
       │   ├── tool_a.yaml
       │   └── tool_b.yaml
       └── scripts/
           ├── tool_a.js
           └── tool_b.js
   ```

   In this example, the skill contains two Node.js scripts (`tool_a.js` and `tool_b.js`), each mapping to a tool in the original toolset.

### Example: Prebuilt Configuration

You can also generate skills from prebuilt toolsets:

```bash
toolbox --prebuilt alloydb-postgres-admin skills-generate \
  --name "alloydb-postgres-admin" \
  --description "skill for performing administrative operations on alloydb"
```

## Installing the Generated Skill in Gemini CLI

Once you have generated a skill, you can install it into the Gemini CLI using the `gemini skills install` command.

### Installation Command

Provide the path to the directory containing the generated skill:

```bash
gemini skills install /path/to/generated-skills/my-skill
```

Alternatively, use ~/.gemini/skills as the `--output-dir` to generate the skill straight to the Gemini CLI.
