# Squire Overview

Squire is a Go-centric, multi-purpose CLI that consolidates project tooling, multi-repo workflows, and one-off utilities into a single, cohesive command-line experience. It is a Swiss Army Knife for Go development that orchestrates tasks normally done via Makefiles, shell scripts, or multi-step workflows. Squire is not a replacement for native go commands; it provides higher-level orchestration.

Current status: early development (Dec 2025). The concept and architecture are defined, with several implemented components and multiple active plans.

---

## Purpose and Vision

Squire is designed to support a developer working across many Git repos and Go modules, including:

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
- Managed repo: a repo with .squire/config.json.
- Unmanaged repo: a repo without .squire/config.json.
- Owned module: a module under active control (via workspace patterns or repo/module overrides).

#### Maybe no longer used

- Workspace / Universe (term evolving): a user-level grouping of repos/modules.

### "Requires" Terminology (ADR-2025-12-14)

Use the term "requires" consistently across code, config, CLI, and docs.

- Config: `"requires"` fields _(not deps)_.
- Types: Requires, RepoRequirement, etc.

### "Active" development and "In-flux" state

_"Active"_ development is a term applies to a Go package that is not "in-flux" _(see below)_ and is also a dependency of the app we are working on _(the classification of "active" is alway relevant to an executable or package that depends on it.)_ Squire looks for candidates for "active" within the directories listed in the `"scan_dirs"`setting in the user config, e.g. `["~/Projects"]` could be a value for `"scan_dirs"`.  

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
- squirepkg/ - Core library
- test/ - Test module (avoids circular dependencies)

Each module has its own go.mod. Use go work sync after modifying dependencies.

### Command Registration Pattern

Commands auto-register via init() using go-cliutil:

```go
package squirecmds

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
    config := c.Config.(*common.Config)
    c.Writer.Printf("Executing...\n")
    return nil
}
```

Key points:

- Commands embed *cliutil.CmdBase.
- Commands implement Handle() error.
- Commands register via init().
- Commands are discovered via blank import: _ "github.com/mikeschinkel/squire/squirepkg/squirecmds".

### Execution Flow

```
cmd/main.go
  -> squirepkg.RunCLI()
    -> Parse CLI args (cliutil.ParseCLIOptions)
    -> Load configs (squirecfg.LoadRootConfigV1)
    -> Transform to runtime config
    -> Setup logger/writer
    -> squirepkg.Run()
      -> Create CmdRunner
      -> Parse command from args
      -> Execute command.Handle()
```

### Core Package Responsibilities

- squirepkg/run_cli.go: entry point, options/config parsing, initialization
- squirepkg/run.go: command runner orchestration
- squirepkg/config.go: config transformation (squirecfg -> common)
- squirepkg/parse_options.go: options transformation

Common packages:

- squirepkg/common: Config, Options, constants, singleton logger/writer
- squirepkg/squirecfg: configuration file structures, go-cfgstore loading
- squirepkg/squirecmds: command implementations

### Design Patterns

- Blank import for command registration.
- Initializer registry for logger/writer setup.
- Multi-source config merging (user -> repo -> module -> CLI flags).
- Deterministic output using ordered maps (dtx.OrderedMap) and sorted iteration.

### Error Handling and Conventions

- Use ClearPath-style goto end cleanup patterns when appropriate.
- Package-level singleton access via common.EnsureLogger().
- Exit codes (run_cli.go):
  - 1: options parsing error
  - 2: config loading error
  - 3: config parsing error
  - 4: known runtime error
  - 5: unknown runtime error
  - 6: logger setup error

---

## Required Dependencies and Conventions

Squire must use these packages when applicable:

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

Squire uses a three-layer model:

1. User-level config: ~/.config/squire/config.json
2. Repo-level config: .squire/config.json or .squire.json in repo root
3. Module-level config: .squire/config.json or .squire.json in module dir

Merging order: user -> repo -> module -> CLI flags (later overrides earlier).

### v0 Repo Config Schema

Location: <repo-root>/.squire/config.json

Required field: modules map. Example:

```json
{
  "modules": {
    "./apppkg": {"name": "apppkg", "role": ["lib"]},
    "./cmd": {"name": "squire-cli", "role": ["cli"]},
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
//squire:goexperiments=arenas,regabiwrappers
```

Squire reads these directives and sets GOEXPERIMENT when running go commands. Experiments are treated as must-have for now.

---

## CLI Commands and Behavior

### v0 Commands

#### squire scan

Purpose: discover go.mod files under roots for unmanaged repos.

Usage:

```
squire scan --root <dir> [--root <dir> ...]
```

Behavior:

- Expand ~ in roots and output absolute paths.
- Find go.mod files.
- Determine repo root via .git.
- Skip repos with .squire/config.json.
- Output each included go.mod path on its own line.

#### squire init

Purpose: create .squire/config.json for selected repos.

Usage:

```
squire init --file <path>
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
- Write .squire/config.json.

Errors:

- Missing file -> non-zero exit.
- Missing go.mod entries -> report and continue, non-zero exit.
- No valid paths -> non-zero exit.

### Requires Tree Command

Command (working name):

```
squire requires tree [<dir>] [flags]
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
<!-- squire:embed-requires-tree -->
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

- squire plan: graph display (inspection). Must remain backward compatible.
- squire next / process: engine-based leaf selection and verdicts. Output is machine-parseable.

Example output (next/process):

```
/Users/mikeschinkel/Projects/go-pkgs/go-dt|withheld|no baseline tag found (first release?)
```

### Future Go Commands

- squire go test: run tests across owned modules, using GOEXPERIMENT directives.
- squire go lint: wraps golangci-lint plus ClearPath linter.
- squire go build: build CLI modules; integrate with GoReleaser.
- squire go fuzz: infinite fuzz testing based on existing scripts.
- squire go dev-on/dev-off (names TBD): toggle go.work + replace overrides.
- squire go release init: TUI for GoReleaser and release workflow scaffolding.

### Workspace Commands (Future)

- squire workspace set <name>
- squire workspace list
- squire workspace add <name>
- squire workspace discover (TUI)

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

### Deterministic Output

Use ordered maps (dtx.OrderedMap) and sorting to ensure stable output across runs.

---

## Retinue Engine and Current Status

### Engine API (in progress)

- VerdictType enum: breaking, likely_breaking, maybe_not_breaking, withheld, unknown.
- EngineResult: leaf module dir, repo dir, repo modules, LocalTagNotPushed, verdict, verdict reason, in-flux dependencies.
- EngineArgs: StartDir, RepoDirs, Config, Logger, Writer.
- ReleaseEngine.Run(ctx): normalize paths, scan repos, build graph, find leaf, check tags, compute verdict.

### Current Status (from PLAN.md)

Completed:

- apidiffr, gitutils, modutils migrated into squire.
- Engine API and core flow implemented.
- Leaf selection and in-flux checks partially implemented.

In progress:

- Remove unused variables in engine.go.
- Resolve modutils/retinue overlap (merge into retinue recommended).
- Complete in-flux detection (git dirty, replace directives, tag push checks).
- Implement verdict computation via apidiffr and baseline tag discovery.
- Wire plan to engine without changing output.
- Create minimal process command.
- Testing against real repos.

---

## go-tuipoc Integration Plan (Next Version Analysis)

Purpose: integrate go-tuipoc version analysis into squire.

Key points:

- go-tuipoc determines next semver for a single module based on API changes since last tag.
- squire orchestrates multi-repo release order.

Missing pieces to migrate:

- analyzer.go, baseline.go, judgment.go, result.go, precondition.go, review/tests.go, cli.go

Integration phases:

1. Copy core logic into squirepkg/tuipoc and adapt imports.
2. Integrate into retinue engine and process command output.
3. Add squire version command with --json output.
4. Later: stability contract validation (from go-tuipoc PLAN).

---

## PRE-commit Analysis Plan (Staged Changes)

Purpose: analyze staged changes before commit to generate commit messages and verdicts.

Key elements:

- Use staged file export to temp directory (git show :path).
- Use cached worktree for baseline tag comparisons.
- Perform API compatibility analysis, AST diff analysis, and test signal analysis.
- Aggregate results into precommit.Results with Verdict.
- Support persistence of analysis in cache (~/.cache/squire/analysis/...).
- Support AI-driven commit grouping and multi-commit workflows.

New packages (planned):

- squirepkg/gitutils: git operations, cached worktrees, staged export, repo queries, locks.
- squirepkg/squirescliui: terminal UI and display helpers.
- squirepkg/precommit: pre-commit analysis orchestration.
- squirepkg/goutils (rename/broaden gomodutils): Go language utilities, analysis functions.

Commit message generation will be template-driven with user and project overrides:

- Built-in templates (default, breaking).
- User templates: ~/.config/squire/templates/
- Project templates: .squire/templates/
- Override order: project -> user -> built-in.

Multi-commit flow:

- AI suggests commit groupings.
- Interactive restaging per group.
- Handle remaining unstaged files.

Edge cases:

- No baseline tag -> verdict unknown, allow commit.
- Nested modules -> exclude nested module paths.
- Binary files -> copy bytes; analyzers skip.
- Deleted files -> skip staged export for deleted file.
- Partial staging -> staged version used.
- Concurrent invocations -> lock files.
- Cache staleness -> fetch before checkout; support --no-cache.
- Large diffs -> truncate or diffstat.
- Template errors -> fallback to built-in.
- Analysis persistence conflicts -> cache key includes staged hash.

---

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

Interactive flow in squire next:

- Option [m]essage or [c]ommitmsg.
- Generate from staged diff only.
- Show subject/body.
- Options: edit, write to file, back, regenerate.

Non-interactive command (planned):

```
squire commitmsg [--repo <dir>] [--format json|text] [--output <file>]
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

Purpose: invoke external AI CLIs without Squire owning API keys.

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

## GoModule Consolidation Plan

Goal: unify gomodutils.Module and retinue.GoModule into a single gomodutils.Module, with retinue.ModuleExt for Squire-specific logic.

Key decisions:

- Keep gomodutils.Module as canonical type.
- Rename GoModGraph -> Graph in gomodutils.
- Graph and Repo move to gomodutils.
- Module works standalone or with SetGraph().
- retinue.ModuleExt wraps Module and implements IsInFlux.

Implementation highlights:

- Module gains Graph, repo, modfile, loaded fields.
- New methods: SetGraph, Dir, Key, Repo, RequireDirs, HasReplaceDirectives.
- AnalyzeStatus becomes method.
- Guard methods enforce Load and SetGraph where needed.

---

## Roadmap (Planned Features and Status)

### Policy File Sync and Drop-in Manager

Status: not started. Priority: medium.

- squire sync to distribute policy files across repos.
- Declarative rules from user or project config.
- Supports copy, template, merge, drop-in actions.

### External Module Dependencies for requires-tree

Status: not started. Priority: medium.

- Implement --all to include external modules.
- Discover via go list -m -json.
- Handle network/proxy failures gracefully.

### API Stability Management

Status: not started. Priority: high.

Includes:

- Changelog generation from Contract annotations.
- Contract enforcement tooling.
- Cross-repo stability validation.
- RemoveAfter timeline coordination.
- Breaking change reports.
- Integration with go-tuipoc.

### Interactive Commit Workflow with LLM Generation

Status: in progress. Priority: high.

- Interactive menu in squire next is implemented.
- AI commit message generation is in progress.
- BubbleTea commit editor not started.
- Configurable LLM providers not started.
- Draft management not started.
- Standalone squire commitmsg command not started.

---

## Release Automation and GitHub Integration

- Tagging and releasing are done via GitHub Actions, not Squire directly.
- Release workflow should run tests, lint, vet before tagging.
- GoReleaser used for binary releases.
- Use go-github SDK, not gh CLI.
- Ensure .github/workflows/test.yml and release.yml exist for managed repos.

---

## go.work and replace Orchestration

Squire treats go.work and replace directives as orthogonal knobs:

- go.work: which local module directories participate in workspace.
- replace: how module paths are resolved for a specific go.mod.

Commands (names TBD):

- dev-on/dev-off to add or remove Squire-managed replace blocks.
- adopt mode for existing replace directives.
- ensure go.work and go.work.sum are managed or ignored per policy.

---

## Language-Aware Commands

Design supports a current language with possible overrides:

```json
{
  "currentLanguage": "go",
  "languages": {"go": {"enabled": true}, "zig": {"enabled": false}}
}
```

Commands like squire test/lint/build should default to current language, with --lang or --zig overrides. Language backend registry is planned when a second language is needed.

---

## TUI Requirements

TUIs are expected for:

- workspace discover
- drop-in configuration
- GoReleaser scaffolding
- interactive release workflows

Non-interactive flags should exist for CI and scripting. BubbleTea is an acceptable framework.

---

## Non-Dependency Dependencies (Drop-ins)

Squire treats embeddable single-file utilities (go-doterr) as a first-class pattern.

Features:

- squire embed add <url>
- squire embed update <name>
- squire embed update --all
- squire embed list

Drop-ins are configured in .squire/config.json (future dropins section) with modes:

- dot
- copy
- ignore

Updates are explicit and not tied to normal dependency bumps.

---

## External References: AWS Multi-Module Tools

Squire draws inspiration from AWS go multi-module repository tools:

- modman.toml for dependency policy and per-module release behavior.
- .changelog/ annotation files and generatechangelog workflow.
- release manifest pipeline (calculaterelease -> tagrelease).
- makerelative/clearrelative for replace directives.
- eachmodule for running commands across modules.
- go_module_metadata generation.

---

## ADRs and Conventions

### ADR-2025-12-21: Branch Metadata Storage

- Each repo is authoritative for expectations about its direct dependencies (branch, remote, path).
- Expectations stored in .git/config and mirrored to JSON on a well-known branch (shared).
- Directory-anchored resolution for worktrees.
- Validate at runtime; do not store observed Git state.

### ADR-2025-12-25: appsvc Directory Layout

Use appsvc as the package for app-specific workflows/services called by commands.

Layout example:

```
app
  cmd
  test
  apppkg
    app
    appcfg
    appcmds
    appcliui
    apptui
    appsvc
```

Rules:

- appcmds depend on appsvc, not vice versa.
- UI packages must not be imported by appsvc.

---

## Development Commands

Build:

- go build ./...
- cd cmd && go build
- go install ./cmd/...

Test:

- go test -v ./...
- go test -v -race -coverprofile=coverage.txt -covermode=atomic ./...
- go work sync && go test ./...

Lint:

- go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.6.2 run ./... --timeout=5m

Format:

- gofmt -s -w .
- go vet ./...

Dependencies:

- go work sync
- go mod tidy in cmd/, squirepkg/, test/

Run:

- go run ./cmd/main.go [command] [args]
- squire [command] [args]

---

## Naming and Miscellany

### Squire Naming

Squire is an opinionated tool but designed to be useful to others. It is open source, Apache 2.0 licensed, and uses go-cliutil, go-cfgstore, and go-dt.

### Gomion Pronunciation

Gomion is pronounced GOM-yun (short "o" as in pom, then "yun"). Two syllables, first stressed. Avoid go-MY-on or GO-mee-on.

---

## Current Testing and Validation Tasks

From TREE_PLAN.md, testing after FlagSet migration:

- ./bin/squire help
- ./bin/squire help scan
- ./bin/squire help init
- ./bin/squire help requires-list
- ./bin/squire help requires-tree
- ./bin/squire scan --continue .
- ./bin/squire requires-list --format=json .
- ./bin/squire requires-tree --show-dirs .
- ./bin/squire requires-tree --embed=/tmp/test.md --before .

Expected:

- Main help shows clean command list (no command-specific flags).
- Command help shows OPTIONS header consistently.
- Flags parse and execute correctly.

---

## What Not to Add

- Do not add commands that duplicate native go commands.
- Do not add squire gomod; use go mod directly.
- Do add commands that orchestrate multiple go commands.
- Do add multi-repo orchestration commands.

Examples of desired commands:

- squire go ci
- squire go test-all
- squire workspace set
- squire workspace discover
- squire deps ensure
- squire replace enable/disable

---

## Acknowledgments and Dependencies

Squire is built with:

- go-cliutil
- go-cfgstore
- go-dt

It is authored by Mike Schinkel and licensed under Apache 2.0.
