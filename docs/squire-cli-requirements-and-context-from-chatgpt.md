# Squire CLI — Project Overview and Context

## 1. Purpose

**Squire** is a personal developer-assistant CLI designed to act as a **companion tool for Go developers**, including the author’s broader XMLUI and Go package ecosystems.
It replaces the author’s earlier MCP server prototype with a simpler, more maintainable, and debuggable command-line tool.

Squire’s guiding metaphor:

> The “Squire” serves the “Knight” (the developer) — handling repetitive and organizational work so the Knight can focus on battles that matter.

---

## 2. Background and Motivation

Originally, the author built a **Claude MCP server** (`scout-mcp`) to expose custom tools (e.g., code scanners) to Claude Desktop.
After discovering that **Claude Code** already accesses and edits local files faster and more effectively, maintaining a full MCP server was unnecessary overhead.

However, one **MCP tool proved uniquely valuable**:

* Scanning a Go codebase to identify:

  * Files and exported symbols lacking comments.
  * Directories missing `README.md` files.

To retain that functionality—and make future extensions easier to debug and maintain—the author is porting that logic to a **standalone CLI app**.

---

## 3. Current State and Source Reference

* Existing code: [`check_docs_tool.go`](https://github.com/mikeschinkel/scout-mcp/blob/main/mcptools/check_docs_tool.go) and test file.
* Repository: [`mikeschinkel/scout-mcp`](https://github.com/mikeschinkel/scout-mcp).

Squire will extract this logic, expand it, and eventually deprecate the MCP server layer.

---

## 4. Core Design Principles

| Principle                       | Description                                                              |
| ------------------------------- | ------------------------------------------------------------------------ |
| **Local-first**                 | All analysis runs locally; no external API dependency.                   |
| **Composable CLI**              | Each feature is its own subcommand.                                      |
| **JSON/NDJSON output**          | Allows easy integration with AI tools (Claude Code, etc.).               |
| **Scriptable & human-friendly** | Output can be machine-parsed or pretty-printed for terminal use.         |
| **Cross-platform**              | Fully functional on macOS, Linux, and Windows.                           |
| **Go-native**                   | Written in Go, leveraging `go/ast`, `go/packages`, and standard tooling. |

---

## 5. Current and Planned Features

### ✅ Phase 1 — Extract and Modernize

* **Command:** `squire scan docs`
  Scans Go packages and lists all exported symbols missing doc comments.

* **Command:** `squire scan readmes`
  Finds directories containing Go packages without `README.md`.

* **Output formats:** `text` (default), `json`, `ndjson`.

---

### 🧩 Phase 2 — Extend Capabilities

#### A. Package Relationship Visualization

* **Command:** `squire map imports --mermaid`
  Generates a Mermaid-compatible dependency graph to visualize package relationships.

#### B. ClearPath Compliance (Code Hygiene)

* **Command:** `squire lint clearpath`
  Scans for violations of the author’s *ClearPath* style rules (e.g., avoid early returns, prefer `goto end` for cleanup consistency).

#### C. Future Linter Enhancements

* Detect large functions, inconsistent naming, or missing doc.go.
* Identify directories that lack examples or tests.

---

### ⚙️ Phase 3 — Potential Integrations

| Integration             | Purpose                                                                                             |
| ----------------------- | --------------------------------------------------------------------------------------------------- |
| **Claude Code**         | Can call `squire ... --json` to produce actionable output for automated documentation improvements. |
| **JetBrains (GoLand)**  | Configurable external tool for doc hygiene scans or dependency visualization.                       |
| **GitHub Actions**      | Use as a CI job to enforce documentation and README presence.                                       |
| **XMLUI Dev Toolchain** | Squire can assist developers working on XMLUI-based apps or packages.                               |

---

## 6. CLI Surface Overview

Example structure (subject to iteration):

```
squire
├── scan
│   ├── docs         # missing doc comments
│   └── readmes      # missing README.md files
├── map
│   └── imports      # package dependency map
├── lint
│   └── clearpath    # ClearPath style enforcement
├── gen
│   ├── readme       # stub README.md templates
│   └── docs         # generate doc comment stubs
```

Each command supports:

* `--root` to specify the root directory.
* `--format json|ndjson|text`.
* `--exclude` and `--include` patterns.
* `--exit-code` (for CI/CD integration).

---

## 7. Future-Proofing and Compatibility

* The binary name will be **`squire`**, despite an existing archived project (`mitchellh/squire`), since that project is long defunct and domain overlap is minimal.
* If conflicts arise, an alias such as `sqr` may be added.
* The architecture will make it easy to extend with new static-analysis or project-maintenance commands.
* Each feature is encapsulated under a subpackage in `internal/`.

---

## 8. Naming Justification

The name **Squire** was chosen for thematic and practical reasons:

* Complements the author’s earlier project **Scout** (`scout-mcp`).
* Conveys helpfulness, reliability, and organization (“a squire keeps the knight’s gear in order”).
* Short, easy to remember, and already part of a consistent ecosystem (e.g., XMLUI, Scout, ClearPath).

---

## 9. Related and Supporting Projects

| Project                 | Role                                                                                        |
| ----------------------- | ------------------------------------------------------------------------------------------- |
| **scout-mcp**           | Predecessor (MCP-based version).                                                            |
| **xmlui-test-server**   | Backend framework where Squire’s code scanning and DB schema tools may later be integrated. |
| **ClearPath**           | Coding style guidelines and planned lint rules.                                             |
| **XMLUI CLI ecosystem** | Target platform for Squire’s reuse and integration.                                         |

---

## 10. Guiding Themes and Non-Goals

### Themes

* Help developers **stay organized**, **reduce friction**, and **document their work**.
* Support both **human** and **AI-driven** workflows.
* Encourage maintainability over complexity.

### Non-goals

* Not a general-purpose MCP framework.
* Not a replacement for `golangci-lint` or `go vet`.
* Not a build or deployment tool — strictly hygiene, visualization, and project assistance.

---

## 11. Initial Implementation Plan

1. Extract `check_docs_tool.go` logic → `internal/scan/docs.go`.
2. Add `internal/scan/readmes.go`.
3. Implement CLI via stdlib `flag` or a minimal command dispatcher.
4. Add structured output (JSON/NDJSON).
5. Provide test fixtures and unit tests.
6. Add Makefile and CI workflow for build/test.

---

## 12. Example JSON Output Schema

```json
{
  "version": "1",
  "issues": [
    {
      "kind": "missing_doc",
      "pkg": "github.com/example/foo/bar",
      "file": "bar/baz.go",
      "line": 42,
      "symbol": "DoThing",
      "symbol_kind": "func",
      "exported": true,
      "message": "Exported function lacks a doc comment."
    },
    {
      "kind": "missing_readme",
      "dir": "pkg/qux",
      "message": "Directory has no README.md"
    }
  ]
}
```

---

## 13. Dependencies and Tooling

| Category      | Tools                                                         |
| ------------- | ------------------------------------------------------------- |
| Language      | Go 1.25+                                                      |
| Testing       | `go test`, `go vet`                                           |
| Linting       | optional `golangci-lint` integration                          |
| Visualization | Mermaid (for dependency graphs)                               |
| Build System  | Standard Makefile (`build`, `test`, `lint`, `cover`, `clean`) |

---

## 14. Long-Term Vision

**Squire** will grow into a modular helper toolkit for developers across the author’s Go ecosystem:

* Assist with **documentation hygiene**.
* Support **visual introspection** (package graphs, structure maps).
* Enforce **ClearPath-style consistency**.
* Optionally, integrate with **XMLUI’s** database migration and code generation workflows.
