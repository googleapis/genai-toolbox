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

A skill package consists of a `SKILL.md` file (with required YAML frontmatter) and a set of Node.js scripts. Each tool defined in your toolset maps to a corresponding script in the generated skill.

### Command Signature

The `skills-generate` command follows this signature:

```bash
toolbox [--tools-file <path> | --prebuilt <name>] skills-generate \
  --name <skill-name> \
  --toolset <toolset-name> \
  --description <description> \
  --output-dir <output-directory>
```

> **Note:** The `<skill-name>` must follow the Agent Skill naming convention: it must contain only lowercase alphanumeric characters and hyphens, cannot start or end with a hyphen, and cannot contain consecutive hyphens (e.g., `my-skill`, `data-processing`).

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

## Output Directory

By default, skills are generated in the `skills` directory. You can specify a different output directory using the `--output-dir` flag.

## Shared Node.js Scripts

The `skills-generate` command generates shared Node.js scripts (`.js`) that work across different platforms (Linux, macOS, Windows). This ensures that the generated skills are portable.

## Installing the Generated Skill in Gemini CLI

Once you have generated a skill, you can install it into the Gemini CLI using the `gemini skills install` command.

### Installation Command

Provide the path to the directory containing the generated skill:

```bash
gemini skills install /path/to/generated-skills/my-skill
```
