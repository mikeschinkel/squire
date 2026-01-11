# Gomion Overview

Gomion is a Go-centric, multi-purpose CLI that consolidates project tooling, multi-repo workflows, and one-off utilities into a single, cohesive command-line experience. It is a Swiss Army Knife for Go development that orchestrates tasks normally done via Makefiles, shell scripts, or multi-step workflows. Gomion is not a replacement for native go commands; it provides higher-level orchestration.

Current status: early development (Dec 2025). The concept and architecture are defined, with several implemented components and multiple active plans.

---

## Purpose and Vision

Gomion is designed to support a developer working across many Git repos and Go modules, including:

- Repos with multiple Go modules (package modules, cmd/* modules, test modules).
- Lockstep app repos (lib + CLI + tests share a version) and independent library families (dt/dtx/dtglob/appinfo).
- Root-of-tree application repos that sit above dependency graphs of shared packages.
- Drop-in embeds like go-doterr that are copied into repos as single files, not imported as normal modules.

Primary goals:

1. Consolidate one-off tools into one CLI so they are discoverable and reused.
2. Replace project Makefiles with consistent CLI commands.
3. Orchestrate multi-repo workflows, including dependency graphs and release order.
4. Provide workspace-aware operations without constant path references.
5. Automate release workflows (testing, linting, tagging, GitHub Actions, GoReleaser).

The long-term mission is to make multi-repo, multi-module development safer, more observable, and harder to get wrong, while staying Go-centric and low-magic.

---

## Design Principles

- Orchestration over duplication: do not implement native go commands; orchestrate them.
- One source of truth per concern; avoid duplicating configuration.
- Explicit and auditable changes (no silent go.mod or go.work edits).
- Repo-local truth for CI; machine-local state is ergonomic only.
- Go-first design, but leave room for future language backends.
- Open-source, opinionated, and long-lived (10+ years).

---

## Terminology and Concepts

### Core Entities

- Repo: a Git repository (identified by .git).
- Module: a Go module with go.mod.
- Root repo: a repo that acts as the root of a dependency tree (application/service).
- Managed repo: a repo with .gomion/config.json.
- Unmanaged repo: a repo without .gomion/config.json.
- Owned module: a module under active control (via workspace patterns or repo/module overrides).

#### Maybe no longer used

- Workspace / Universe (term evolving): a user-level grouping of repos/modules.

### "Requires" Terminology (ADR-2025-12-14)

Use the term "requires" consistently across code, config, CLI, and docs.

- Config: `"requires"` fields _(not deps)_.
- Types: Requires, RepoRequirement, etc.

### "Active" development and "In-flux" state

_"Active"_ development is a term applies to a Go package that is not "in-flux" _(see below)_ and is also a dependency of the app we are working on _(the classification of "active" is alway relevant to an executable or package that depends on it.)_ Gomion looks for candidates for "active" within the directories listed in the `"scan_dirs"`setting in the user config, e.g. `["~/Projects"]` could be a value for `"scan_dirs"`.  

A module or repo is in-flux if any of the following are true:

1. Dirty working tree (untracked/staged/unstaged).
2. Commits not tagged.
3. Tags not pushed.
4. go.mod has `replace` directives _(when required by specific workflows)._

Tagged but unpushed is a special case that should stop with a clear warning _("Did you mean to push?")_ so downstream logic can assume tagged implies pushed.

Note that `replace` directives in `go.mod` are transient state, not intent.

### "Leaf" Module

A _"leaf"_ module is an in-flux module that has no in-flux dependencies. This is the next module that can be worked on. The leaf algorithm is dependency-order driven and intentionally selects in-flux modules, not clean ones.

---

## Repository Layout and Architecture

### Go Workspace Structure

This repository uses Go workspaces with three modules:

- cmd/ - CLI entry point
- gompkg/ - Core library
- test/ - Test module (avoids circular dependencies)

Each module has its own go.mod. Use go work sync after modifying dependencies.

### Command Registration Pattern

Commands auto-register via init() using go-cliutil:

```go
package gomioncmds

import "github.com/mikeschinkel/go-cliutil"

type MyCmd struct {
    *cliutil.CmdBase
}

func init() {
    cliutil.RegisterCommand(&MyCmd{
        CmdBase: cliutil.NewCmdBase(cliutil.CmdArgs{
            Name:        "mycommand",
            Usage:       "mycommand [options]",
            Description: "Brief description",
            Order:       10,
        }),
    })
}

func (c *MyCmd) Handle() error {
    config := c.Config.(*gomion.Config)
    c.Writer.Printf("Executing...\n")
    return nil
}
```

Key points:

- Commands embed *cliutil.CmdBase.
- Commands implement Handle() error.
- Commands register via init().
- Commands are discovered via blank import: _ "github.com/mikeschinkel/gomion/gompkg/gomioncmds".

### Execution Flow

```
cmd/gomion/main.go
  -> gommod.RunCLI()
    -> Parse CLI args (cliutil.ParseCLIOptions)
    -> Load configs (gomioncfg.LoadRootConfigV1)
    -> Transform to runtime config
    -> Setup logger/writer
    -> gommod.Run()
      -> Create CmdRunner
      -> Parse command from args
      -> Execute command.Handle()
```

### Core Package Responsibilities

- gompkg/run_cli.go: entry point, options/config parsing, initialization
- gompkg/run.go: command runner orchestration
- gompkg/config.go: config transformation (gomioncfg -> gomion)
- gompkg/parse_options.go: options transformation

Common packages:

- gompkg/gomion: Config, Options, constants, singleton logger/writer
- gompkg/gomioncfg: configuration file structures, go-cfgstore loading
- gompkg/gomioncmds: command implementations

### Design Patterns

- Blank import for command registration.
- Initializer registry for logger/writer setup.
- Multi-source config merging (user -> repo -> module -> CLI flags).
- Deterministic output using ordered maps (dtx.OrderedMap) and sorted iteration.

### Error Handling and Conventions

- Use ClearPath-style goto end cleanup patterns when appropriate.
- Package-level singleton access via gomion.EnsureLogger().
- Exit codes (run_cli.go):
  - 1: options parsing error
  - 2: config loading error
  - 3: config parsing error
  - 4: known runtime error
  - 5: unknown runtime error
  - 6: logger setup error

---

## Required Dependencies and Conventions

Gomion must use these packages when applicable:

### Production

- ~/Projects/go-pkgs/go-doterr/README.md - error handling
- ~/Projects/go-pkgs/go-dt/README.md - data types (paths, etc.)
- ~/Projects/go-pkgs/go-dt/dtx/README.md - data type extensions
- ~/Projects/go-pkgs/go-cfgstore/README.md - configuration management
- ~/Projects/go-pkgs/go-cliutil/README.md - CLI utilities and command handling
- ~/Projects/go-pkgs/go-logutil/README.md - logging utilities

### Tests

- ~/Projects/go-pkgs/go-testutil/README.md - testing utilities
- ~/Projects/go-pkgs/go-fsfix/README.md - filesystem fixtures for tests
- ~/Projects/go-pkgs/go-jsontest/README.md - JSON testing utilities
- ~/Projects/go-pkgs/go-jsontest/pipefuncs/README.md - JSON pipeline functions

### Local Server Work (not directly CLI)

- ~/Projects/go-pkgs/go-sqlparams/README.md - SQL parameter handling
- ~/Projects/go-pkgs/go-pathvars/README.md - path variable extraction/routing
- ~/Projects/go-pkgs/go-jsonxtractr/README.md - JSON extraction utilities
- ~/Projects/go-pkgs/go-rfc9457/README.md - RFC 9457 problem details
- ~/Projects/go-pkgs/go-pathvars/examples/basic_routing/README.md
- ~/Projects/go-pkgs/go-pathvars/examples/rest_api/README.md

---

## Configuration Model

### Scope and Layers

Gomion uses a three-layer model:

1. User-level config: ~/.config/gomion/config.json
2. Repo-level config: .gomion/config.json or .gomion.json in repo root
3. Module-level config: .gomion/config.json or .gomion.json in module dir

Merging order: user -> repo -> module -> CLI flags (later overrides earlier).

### v0 Repo Config Schema

Location: <repo-root>/.gomion/config.json

Required field: modules map. Example:

```json
{
  "modules": {
    "./apppkg": {"name": "apppkg", "role": ["lib"]},
    "./cmd": {"name": "gomion-cli", "role": ["cli"]},
    "./test": {"name": "test", "role": ["test"]}
  }
}
```

Rules:

- Keys are repo-relative paths with ./ notation.
- name is required and unique per repo.
- role is a non-empty array of strings (lib, cli, test in v0).
- Unrecognized fields are ignored (future compatibility).

### Planned Extensions (Vision)

- groups: release groups and lockstep versioning
- dependencies: dependency policy for go.mod updates
- dropins: non-dependency dependency management
- language config: currentLanguage and enabled languages

### Ownership and Resolution

Owned modules can be defined at:

- Workspace level (path patterns)
- Repo level (overrides)
- Module level (per-module overrides)

Resolution order: module -> repo -> workspace -> default (not owned).

### Go Experiments Directive

Use inline directives in go.mod as canonical config:

```go
//gomion:goexperiments=arenas,regabiwrappers
```

Gomion reads these directives and sets GOEXPERIMENT when running go commands. Experiments are treated as must-have for now.

---

## CLI Commands and Behavior

### v0 Commands

#### gomion scan

Purpose: discover go.mod files under roots for unmanaged repos.

Usage:

```
gomion scan --root <dir> [--root <dir> ...]
```

Behavior:

- Expand ~ in roots and output absolute paths.
- Find go.mod files.
- Determine repo root via .git.
- Skip repos with .gomion/config.json.
- Output each included go.mod path on its own line.

#### gomion init

Purpose: create .gomion/config.json for selected repos.

Usage:

```
gomion init --file <path>
```

Input file:

- Plain text.
- Each non-empty, non-comment line is a go.mod path.
- ~ expansion supported.

Behavior:

- Group go.mod paths by repo root.
- Skip repos already managed (warn).
- Derive module names and roles by heuristics:
  - cmd/ -> cli
  - test/ -> test
  - others -> lib
- Write .gomion/config.json.

Errors:

- Missing file -> non-zero exit.
- Missing go.mod entries -> report and continue, non-zero exit.
- No valid paths -> non-zero exit.

### Requires Tree Command

Command (working name):

```
gomion requires tree [<dir>] [flags]
```

Purpose:

- Render a dependency tree for current repo modules.
- Optionally embed the tree into a markdown file.

Flags:

- --all: include external modules.
- --show-dirs: show local module rel dirs instead of module path.
- --show-all: show module path plus location (implies show-dirs info).
- --embed=<markdown_file>: embed tree in markdown at marker.
- --before / --after: place tree before or after marker.

Marker:

```
<!-- gomion:embed-requires-tree -->
```

Embedding:

- Wrap in ```text fenced block.
- Insert before or after marker based on flags.
- Marker not found -> error.
- For now, repeated runs may append additional blocks (no replacement).

Tree format:

- ASCII tree using "|-" and "`-" style (implemented as "|-" with proper characters).
- Top-level roots are modules in current repo.
- Child entries are direct requires.
- Duplicate nodes may be shown as leaves to avoid recursion.

### plan and next/process

- gomion plan: graph display (inspection). Must remain backward compatible.
- gomion next: engine-based leaf selection and verdicts. Output is machine-parseable.


---

## Release Workflow and Engine Behavior

### Manual Workflow to Automate

1. Identify in-flux repos (dirty, untracked, untagged commits, unpushed tags).
2. Find leaf module with no in-flux dependencies.
3. Prepare module for release:
   - Remove replace directives
   - go mod tidy
   - vet, lint, test
4. Commit changes (use commit-msg generation and review).
5. Tag and release via GitHub Actions.
6. Repeat until all modules are released.

### Leaf Algorithm (Correct Model)

- Most modules are expected to be in-flux during development.
- Find in-flux modules whose dependencies are all clean.
- Release bottom-up, then repeat.


## Commit Message Generation (LLM CLI MVP)

Purpose: generate draft commit messages from staged diffs using installed LLM CLIs.

Config (ai_provider or llm block):

- enabled: true/false
- provider: claude_cli or codex_cli (later: letta, external-cli)
- claude_exe, codex_exe
- system_prompt_file
- max_diff_bytes, timeout_seconds
- output: json or text
- conventional_commits: true/false
- strip_env: keys to remove (ANTHROPIC_API_KEY, OPENAI_API_KEY)

Interactive flow in gomion next:

- Option [m]essage or [c]ommitmsg.
- Generate from staged diff only.
- Show subject/body.
- Options: edit, write to file, back, regenerate.

Non-interactive command (planned):

```
gomion commitmsg [--repo <dir>] [--format json|text] [--output <file>]
```

Claude CLI invocation (preferred):

- exec.CommandContext
- -p, --output-format json, --json-schema, --system-prompt-file
- pass prompt and diff via stdin

Codex CLI invocation:

- codex exec - --cd <repoDir> --color never --output-last-message <tmpFile>

Output should be JSON {subject, body} when possible.

---

## LLM CLI Provider Integration (Broader)

Purpose: invoke external AI CLIs without Gomion owning API keys.

MVP providers:

- Claude Code
- Codex
- Letta Code
- External CLI (template-driven)

Key requirements:

- Provider abstraction with Detect, Capabilities, BuildCommand, ParseResult.
- Runner supports context cancellation, timeouts, cwd control, streaming output.
- Response type includes provider, model, format, text, JSON, stdout/stderr, exit code, duration, meta.

External CLI provider config supports placeholders:

- {{prompt}}, {{cwd}}, {{repoRoot}}

Redaction rules for env and args with secrets.

---

## Roadmap (Planned Features and Status)

### Policy File Sync and Drop-in Manager

- gomion sync to distribute policy files across repos.
- Declarative rules from user or project config.
- Supports copy, template, merge, drop-in actions.

### API Stability Management

- Changelog generation from Contract annotations.
- Contract enforcement tooling.
- Cross-repo stability validation.
- RemoveAfter timeline coordination.
- Breaking change reports.
- Basic proof-of-concept for TUI in go-tuipoc.

---

## Release Automation and GitHub Integration

- Tagging and releasing can be done either by Gomion directly or by GitHub Actions depending on configuration.
- Release workflow should run tests, lint, vet before tagging.
- GoReleaser as an option used for binary releases.
- Optional commands to ensure appropriate .github/workflows exist for managed repos.

---

## go.work and replace Orchestration

Gomion treats go.work and replace directives as orthogonal knobs:

- go.work: which local module directories participate in workspace.
- replace: how module paths are resolved for a specific go.mod.

Commands (names TBD):

- Commands to add or remove Gomion-managed replace blocks.
- If needed, commands adopt mode for existing replace directives.
- Commands should ensure go.work/.sum are managed or ignored per policy.

---

## Language-Aware Commands

We hvae discussed this ideas as potential for future, but not finalized plans for them.

Design potentially supports a current language with possible overrides:

```json
{
  "currentLanguage": "go",
  "languages": {"go": {"enabled": true}, "zig": {"enabled": false}}
}
```

Commands like gomion test/lint/build should default to current language, with --lang or --zig overrides. Language backend registry is planned when a second language is needed.

---


## Naming and Miscellany

### Gomion Naming

Gomion is an opinionated tool but designed to be useful to others. It is open source, Apache 2.0 licensed

Gomion will soon be renamed to Gomion.

### Gomion Pronunciation

Gomion is pronounced GOM-yun (short "o" as in pom, then "yun"). Two syllables, first stressed. Avoid go-MY-on or GO-mee-on.

---


## What Not to Add

- Do not add commands that duplicate native go commands.
- Do not add gomion gomod; use go mod directly.
- Do add commands that orchestrate multiple go commands.
- Do add multi-repo orchestration commands.

