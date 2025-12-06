# Squire CLI — General Purpose Project Overview

## 1. Purpose and Vision

**Squire** is a developer-assistant CLI tool designed to help Go developers maintain clean, well-documented, well-organized codebases. It originated from functionality developed for an MCP server, but will now exist as a standalone command-line tool that is easier to debug, maintain, and extend.

Squire acts metaphorically as a *squire* assisting the *knight* (the developer). Its job is to handle repetitive, organizational, and hygiene-related tasks that improve developer productivity and code clarity.

Squire is intentionally simple, local-first, written in Go, and optimized for both human use and AI-assisted workflows (e.g., Claude Code).

---

## 2. Motivations for Creating Squire

### 2.1 Replacement for MCP Server Logic

The earlier MCP server prototype (`scout-mcp`) contained a highly valuable tool: scanning a Go codebase for missing:

* Doc comments on exported symbols
* README files for directories

Debugging and extending an MCP server proved significantly more difficult than maintaining a regular CLI. Claude Code also outperformed the MCP integration for file updates. This justified extracting the logic into its own standalone tool.

### 2.2 Expand Scope Beyond Documentation Checks

Squire is intended to grow into a **multi-purpose helper tool** offering:

* Code hygiene and style enforcement (e.g., ClearPath)
* Dependency and package relationship visualization
* Documentation scaffold generation
* Project cleanup and organization tasks

Its purpose is broader than a linter but narrower than a full build or analysis tool—focused specifically on *developer efficiency and codebase upkeep*.

---

## 3. Core Feature Areas (High-Level)

### 3.1 Documentation Hygiene

* Detect exported Go symbols lacking doc comments.
* Detect directories missing `README.md`.
* Generate optional documentation scaffolds.
* Output formats suitable for human reading or AI parsing (JSON, NDJSON).

### 3.2 Package and Code Structure Visualization

* Generate a Go import dependency graph.
* Export graphs in Mermaid syntax for use in Markdown or documentation.
* Help developers understand codebase structure and identify package merging opportunities.

### 3.3 ClearPath Style Enforcement

Squire will optionally enforce **ClearPath**, a Go coding style developed to:

* Avoid early returns in favor of `goto end` for cleanup consistency.
* Encourage predictable control flow.
* Improve readability and maintainability.

ClearPath checks may include:

* Detection of early returns
* Missing `end:` blocks
* Variable naming consistency
* Possible control-flow violations

### 3.4 Future Growth Areas

Squire is envisioned as a modular tool. Possible future commands include:

* Code-generation helpers
* Cleanup functions (remove unused files, flag dead code)
* Project health reports
* Aggregating metadata across multiple repositories

---

## 4. CLI Philosophy and Design

### 4.1 Command Hierarchy

Squire organizes features into clear, intuitive subcommands:

```
squire
├── scan        # analyze codebase for hygiene issues
│   ├── docs
│   └── readmes
├── map         # generate structural visualizations
│   └── imports
├── lint        # enforce ClearPath and related rules
│   └── clearpath
├── gen         # scaffold docs, readmes, etc.
```

### 4.2 Output Philosophy

* **Human-first text output** when run interactively.
* **JSON and NDJSON** for Claude Code and tooling automation.
* **Mermaid** for graph visualizations.

### 4.3 Integration Targets

Squire is explicitly designed to integrate smoothly with:

* Claude Code (AI-assisted development)
* Makefiles and shell workflows
* GoLand (JetBrains external tools)
* GitHub Actions for hygiene enforcement

---

## 5. Naming Justification

The name **Squire** fits naturally because:

* It conveys the idea of a helpful assistant who keeps the Knight's (developer's) tools in order.
* It follows the thematic predecessor project **Scout**.
* It is memorable, succinct, and metaphorically appropriate.
* Existing projects named "squire" do not meaningfully overlap with this tool’s domain.

Even though another archived tool uses the same binary name, the risk is acceptable. If necessary, alternative binaries (`sqr`) can be offered later.

---

## 6. Non-Goals

To maintain clarity, Squire is *not* intended to be:

* A replacement for `go vet` or `golangci-lint`
* A build or deployment tool
* A dependency management system
* A full MCP replacement
* A framework tied specifically to XMLUI

It is purpose-built to enhance developer experience through code hygiene, structure understanding, and documentation assistance.

---

## 7. Roadmap Summary (High-Level)

### Phase 1 (Immediate)

* Extract existing doc-checking logic
* Build basic `scan docs` and `scan readmes`
* Implement JSON/NDJSON output

### Phase 2

* Add dependency graphing (`map imports`)
* Implement ClearPath checks
* Add templated README/doc generation

### Phase 3

* Broader hygiene tools
* Multi-repo insights
* Extend ClearPath rules
* Optional project-scaffolding helpers

---

## 8. Guiding Themes

* **Assist, don’t dictate** — Squire helps developers, not forces heavy rules.
* **Low friction** — everything should work with minimal configuration.
* **Human + AI synergy** — Squire acts as a bridge between a codebase and an AI assistant.
* **Go-first design** — leverage Go's tooling ecosystem.
* **Maintainability** — Squire is built for the author’s long-term personal use, extensible without heavy overhead.

---

This document provides a complete overview of Squire’s purpose, philosophy, scope, and high-level design. It is suitable as a project introduction or a context file to accompany further technical requirements and design discussions.
