# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## ⚠️ CRITICAL INSTRUCTIONS - MUST READ FIRST

**These are non-negotiable requirements. Read and follow these instructions at the start of every session before proceeding with any tasks.**

### Coding Conventions (READ FIRST)
Read these files to understand and follow coding conventions. Use pre-existing functionality from the packages below instead of reimplementing from scratch in a less maintainable form:

- `~/.claude/CLAUDE-golang.md`
- `~/.claude/CLAUDE-golang-error-handling.md`
- `~/.claude/CLAUDE-golang-file-handling.md`
- `~/.claude/CLAUDE-testing-best-practices.md`

### Required Packages (USE THESE)
You MUST use these packages whenever applicable instead of standard library or third-party alternatives:

**For production code:**
- `~/Projects/go-pkgs/go-doterr/README.md` - Error handling
- `~/Projects/go-pkgs/go-dt/README.md` - Data types (paths, etc.)
- `~/Projects/go-pkgs/go-dt/dtx/README.md` - Data type extensions
- `~/Projects/go-pkgs/go-cfgstore/README.md` - Configuration management
- `~/Projects/go-pkgs/go-cliutil/README.md` - CLI utilities and command handling
- `~/Projects/go-pkgs/go-logutil/README.md` - Logging utilities

**For writing tests:**
- `~/Projects/go-pkgs/go-testutil/README.md` - Testing utilities
- `~/Projects/go-pkgs/go-fsfix/README.md` - Filesystem fixtures for tests
- `~/Projects/go-pkgs/go-jsontest/README.md` - JSON testing utilities
- `~/Projects/go-pkgs/go-jsontest/pipefuncs/README.md` - JSON pipeline functions

**For localsvr work (not directly CLI):**
- `~/Projects/go-pkgs/go-sqlparams/README.md` - SQL parameter handling
- `~/Projects/go-pkgs/go-pathvars/README.md` - Path variable extraction/routing
- `~/Projects/go-pkgs/go-jsonxtractr/README.md` - JSON extraction utilities
- `~/Projects/go-pkgs/go-rfc9457/README.md` - RFC 9457 problem details
- `~/Projects/go-pkgs/go-pathvars/examples/basic_routing/README.md` - Basic routing examples
- `~/Projects/go-pkgs/go-pathvars/examples/rest_api/README.md` - REST API examples

---

## Project Purpose

Squire is a "Swiss Army Knife" CLI for Go development that consolidates many one-off tools into a single, cohesive CLI. It automates tasks currently requiring Makefiles, shell scripts, or manual multi-step workflows. It is NOT a replacement for native `go` commands - it provides higher-level orchestration for multi-repo workflows, development automation, and workspace management.

**Primary Goals**:
1. **Consolidate one-off tools** - Keep all custom Go development tools in one CLI to avoid forgetting about them and accidentally recreating them
2. **Replace project Makefiles** - Provide consistent commands across projects (see temp/Makefile for typical tasks)
3. **Multi-repo orchestration** - Manage dependencies across multiple repositories in active development
4. **Workspace awareness** - Operations work on configured workspace without constant path references
5. **Release automation** - Integrate with GoReleaser and GitHub Actions for testing, linting, tagging, and releasing

**Key Features**:
- Manage `replace` directives in go.mod for multi-repo development (go.work alone is insufficient in practice)
- Orchestrate tests/lints/vets across dependency trees
- Workspace-aware operations via `~/.config/squire/config.json`
- Replace project-specific Makefiles with consistent CLI commands
- GitHub workflow management (ensure all repos have test.yml and release.yml)
- Version tagging integrated with CI/CD (via GitHub Actions)
- GoReleaser integration for compiled binaries

**Design Principle**: Only add commands that would otherwise require Makefiles, scripts, or multiple manual steps. Do not duplicate native `go` commands.

**Open Source Vision**: Designed for others to adopt, use, and contribute to - not just a personal tool.

## Development Commands

### Building
```bash
# Build all modules in workspace
go build ./...

# Build specific module
cd cmd && go build

# Build and install
go install ./cmd/...
```

### Testing
```bash
# Run tests in current module
go test -v ./...

# Run with race detection and coverage
go test -v -race -coverprofile=coverage.txt -covermode=atomic ./...

# Test all workspace modules
go work sync && go test ./...
```

### Linting
```bash
# Run golangci-lint
go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.6.2 run ./... --timeout=5m
```

### Formatting
```bash
gofmt -s -w .
go vet ./...
```

### Dependencies
```bash
# Tidy all modules in workspace
go work sync
cd cmd && go mod tidy
cd ../squirepkg && go mod tidy
cd ../test && go mod tidy
```

### Running
```bash
# Run from source
go run ./cmd/main.go [command] [args]

# After install
squire [command] [args]
```

## Architecture

### Go Workspace Structure
This project uses Go workspaces (go.work) with three modules:
- **cmd/** - Thin binary entry point
- **squirepkg/** - Core library (importable by other projects)
- **test/** - Test module (avoids circular dependencies)

**Important**: Each module has its own go.mod. Use `go work sync` after modifying dependencies.

**Critical Reality**: go.work alone is insufficient for multi-repo development. In practice, you MUST use both go.work AND `replace` directives in go.mod to successfully build projects with local dependencies. Tools that assume otherwise (like x/mod/gohack or standard go work commands) don't handle real-world development workflows. Squire manages both go.work and go.mod replace directives together.

### Command Registration Pattern

Commands auto-register via `init()` using the go-cliutil framework:

```go
// squirepkg/squirecmds/mycommand_cmd.go
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
            Order:       10,  // Display order in help
        }),
    })
}

func (c *MyCmd) Handle() error {
    config := c.Config.(*common.Config)
    // Access: c.Writer, c.Logger, c.Args, c.AppInfo
    c.Writer.Printf("Executing...\n")
    return nil
}
```

**Key Points**:
1. Commands MUST embed `*cliutil.CmdBase`
2. Commands MUST implement `Handle() error`
3. Register in `init()` for auto-discovery
4. Import squirecmds package with blank import in run.go: `_ "github.com/mikeschinkel/squire/squirepkg/squirecmds"`

### Config System (Three Layers, Three Scopes)

**Configuration Scopes**:
Squire uses a hierarchical config system with three levels:
1. **User-level** (`~/.config/squire/config.json`) - Workspace definitions, global settings
2. **Repo-level** (`.squire/config.json` OR `.squire.json` in repo root) - Repo-specific settings
3. **Module-level** (`.squire/config.json` OR `.squire.json` in module dir) - Module-specific settings

**Note**: Config file location (directory vs. file) is managed by go-cfgstore and may be standardized as project evolves.

**Config merging**: User → Repo → Module → CLI flags (later overrides earlier)

**Workspace Configuration**:
The user-level config defines workspaces. Module information is discovered from go.mod files, not duplicated in config (following "one source of truth" principle).

**"Owned" modules**: Configurable per workspace, repo, and module. Owned modules are those under active development and should be managed by Squire (testing, linting, versioning, replace directive management, etc.).

**Layer 1: Raw Config** (squirecfg package)
- Loaded from user/repo/module `.squire/config.json` files
- Versioned schema (`RootConfigV1`)
- Managed by go-cfgstore (supports multi-location merging)

**Layer 2: Options** (common package)
- CLI flags parsed into `common.Options`
- Embeds `cliutil.CLIOptions` for standard flags
- Transformation: `squirecfg.Options` → `common.Options` via `ParseOptions()`

**Layer 3: Runtime Config** (common package)
- `common.Config` contains Options + AppInfo + Logger + Writer
- Injected into commands via CmdRunner
- Access in commands: `config := c.Config.(*common.Config)`

### Execution Flow

```
cmd/main.go
  → squirepkg.RunCLI()
    → Parse CLI args (cliutil.ParseCLIOptions)
    → Load configs (squirecfg.LoadRootConfigV1)
    → Transform to runtime config
    → Setup logger/writer
    → squirepkg.Run()
      → Create CmdRunner
      → Parse command from args
      → Execute command.Handle()
```

### Package Responsibilities

**squirepkg/**
- `run_cli.go` - Entry point, options/config parsing, initialization
- `run.go` - Core execution, command runner orchestration
- `config.go` - Config transformation (squirecfg → common)
- `parse_options.go` - Options transformation

**squirepkg/common/**
- Shared types: `Config`, `Options`, constants
- Singleton management: logger, writer

**squirepkg/squirecfg/**
- Configuration file structures
- Loading via go-cfgstore

**squirepkg/squirecmds/**
- All command implementations
- Each command in separate file: `*_cmd.go`

## Key Dependencies

### go-cliutil (Command Framework)
Location: `../../go-pkgs/go-cliutil/` (via replace directive)

Provides:
- Command registration and tree building
- Flag parsing
- Help generation
- CmdRunner orchestration

### go-cfgstore (Config Management)
Location: `../../go-pkgs/go-cfgstore/` (via replace directive)

Provides:
- Multi-location config file management
- Config merging (user-level overrides project-level)
- Versioned config schema support

### go-dt (Type-Safe Data Types)
Provides type-safe wrappers for:
- `dt.Filepath`, `dt.DirPath`, `dt.RelFilepath`
- `dt.Version`, `dt.URL`
- `appinfo.AppInfo` - Application metadata container

## Important Conventions

### Error Handling with goto
Use consistent pattern for cleanup:
```go
func DoSomething() (err error) {
    result, err := Step1()
    if err != nil {
        goto end
    }

    err = Step2(result)
    if err != nil {
        goto end
    }

end:
    return err
}
```

### Package-Level Singleton Access
```go
// squire/logger.go
logger := common.EnsureLogger()  // Panics if SetLogger() not called
```

### Exit Codes (run_cli.go)
- 1: Options parsing error
- 2: Config loading error
- 3: Config parsing error
- 4: Known runtime error
- 5: Unknown runtime error
- 6: Logger setup error

### Type Assertions for Context
Commands receive context via interfaces, type-assert to concrete types:
```go
config := c.Config.(*common.Config)
options := c.Options.(*common.Options)
```

## Design Patterns

### Blank Import for Side Effects
Commands auto-register via init():
```go
import _ "github.com/mikeschinkel/squire/squirepkg/squirecmds"
```

### Initializer Registry
Packages can register callbacks for logger/writer initialization:
```go
common.RegisterSetLoggerFunc(func(l *slog.Logger) {
    myPackageLogger = l
})
```

### Multi-Source Config Merging
Later configs override earlier ones:
```
Project config → User config → CLI flags
```

## Core Concepts

### Non-Dependency Dependencies (go-doterr Model)

**Problem**: External dependencies create security risks when you update code (3rd party could inject exploits). `go mod vendor` causes too many problems.

**Solution**: Embed model for single-file utilities
- Reference: https://github.com/mikeschinkel/go-doterr
- Drop in a single .go file from a dependency (no go.mod dependency)
- Squire can update one or all uses of that file in a workspace
- Not commonly used, but essential for specific use cases

**Squire Features**:
- `squire embed add <url>` - Add a single-file dependency to project
- `squire embed update <name>` - Update specific embedded file
- `squire embed update --all` - Update all embedded files in workspace
- `squire embed list` - Show all embedded dependencies in workspace
- Tracks original source URL and version for updates
- Verifies file hasn't been locally modified before updating

**Use Cases**:
- Utility packages like go-doterr (error handling)
- Single-file helpers that don't warrant a full dependency
- Security-sensitive code where you want to audit every line
- Stable utilities that rarely change

**Why This Matters**:
- Avoids supply chain attacks
- Full control over code in your repo
- Easy to audit (it's just a file)
- Can still benefit from upstream updates when you choose

### ClearPath Coding Style

**Concept**: A coding style that will be enforced via Squire linter

**Implementation**:
- `squire lint --clearpath` - Run ClearPath linter
- ClearPath linter will be developed as part of Squire
- Details TBD as style is formalized

**Note**: This is a custom linting feature specific to this coding philosophy.

### Scaffolding and Templates

**Template System**:
Squire will support project scaffolding with templates:
- Template-based .go files that include desired imports
- Effectively defines starter packages for new projects
- Avoids duplication (templates include imports, not separate package lists)
- Interactive TUI for template selection and customization

**Integration with GoReleaser**:
- `squire scaffold goreleaser` - Interactive setup of .goreleaser.yml
- Template-based workflow file generation
- Workspace-aware defaults

---

### go.work and replace Directive Management

**Reality of Multi-Repo Development**:
- go.work alone is insufficient - you MUST also use `replace` directives in go.mod
- Squire treats go.work and go.mod replace directives as **orthogonal knobs**
- Squire keeps them consistent with workspace/user/repo/module config

**Key Operations**:
- **Enable local dev mode**: Add modules to go.work + add replace directives to relevant go.mod files
- **Disable local dev mode**: Remove from go.work + remove replace directives (prepare for release)
- **Ensure consistency**: Verify go.work and replace directives match workspace config
- **Manage .gitignore**: Ensure go.work and go.work.sum are in .gitignore
- **Discover modules**: Read go.mod files to find module information (don't duplicate in config)

**go.mod Parsing**:
Use golang.org/x/mod/modfile package:
- `Parse()` - Full validation (use when go.mod should be buildable)
- `ParseLax()` - Syntax-only validation (use when go.mod may not be buildable yet, which is expected during local dev)

### GitHub Integration

**GitHub API Access**:
Use GitHub's Go SDK (google/go-github) rather than shelling out to `gh` CLI:
- No dependency on external `gh` binary
- No version compatibility issues
- Programmatic access to GitHub API

**Workflow Management**:
- Ensure all managed repos have `.github/workflows/test.yml`
- Ensure all managed repos have `.github/workflows/release.yml`
- Templates should be flexible/configurable for different project needs
- Release workflow should run tests/lint/vet before tagging
- Integration with GoReleaser for compiled binaries

**Version Tagging**:
- Versions are tagged by GitHub Actions (release.yml), not manually
- Tagging only after CI passes (tests, lint, vet)
- Potential future feature: tag without releasing (TBD)

### Workspace Management

**Workspace Concept**:
A workspace is a collection of related Go modules under active development. Setting a workspace allows commands to operate without constant path references.

**Operations**:
- `squire workspace set <name>` - Set active workspace
- `squire workspace list` - Show configured workspaces
- `squire workspace add <name>` - Create new workspace
- `squire workspace discover` - Discover modules in a directory tree
- Commands like `squire go test` operate on the active workspace

### Language-Aware Commands

**Multi-Language Design**:
Squire is designed with a "current language" concept to support future language extensions:
- Default language: Go (for now)
- Language can be set per workspace in config
- Language can be overridden via flag: `squire test --lang=zig` or `squire test --zig`
- Commands like `squire test`, `squire lint`, `squire build` adapt to current language

**Command Structure Options**:
Two possible patterns being considered:

1. **Language-aware top-level commands** (preferred):
   ```bash
   squire test              # Uses current language (Go by default)
   squire test --zig        # Override to Zig
   squire lint              # Uses current language
   squire build             # Uses current language
   ```

2. **Language-specific subcommands** (current structure):
   ```bash
   squire go test           # Go-specific
   squire go lint           # Go-specific
   squire zig test          # Zig-specific (future)
   ```

**Why Option 1 Matters**:
- Allows future support for Zig, Rust, etc. without breaking existing commands
- Commands remain intuitive: `squire test` just works regardless of language
- Language-specific behavior is abstracted behind common interface
- Reduces typing and cognitive load

**Implementation Note**:
Language-specific logic should be pluggable (similar to command registration pattern).

**Future: Language Backend Registry**:
When multi-language support is actually needed (not before):
- Implement a registry of language backends
- Each language registers handlers for test, lint, build, etc.
- Commands dispatch to appropriate language backend
- Similar to command registration pattern

**For Now**:
- Implement configuration for languages up-front
- Don't implement language backend registry until adding a 2nd language
- Keep Go-specific logic simple and direct

### Terminal User Interface (TUI)

**Interactive Commands**:
Some commands are interactive by nature and should offer a TUI:
- `squire workspace discover` - Interactively select/configure discovered modules
- GoReleaser scaffolding - Interactive setup of .goreleaser.yml
- Workspace management (selecting, configuring)
- Module selection for operations
- Conflict resolution (e.g., when replace directives conflict)
- Configuration wizards

**TUI Framework**: TBD (consider bubbletea or similar)

**Design Principle**: Commands that involve making multiple choices or configuring complex settings should provide an interactive TUI rather than requiring many CLI flags or config file editing.

### Go Experiments Tracking

**Concept**:
Squire should track Go experiments (GOEXPERIMENT) per workspace/module using directives in go.mod.

**Directive Format** (in go.mod):
```go
module github.com/example/mymodule
//squire:goexperiments=arenas,regabiwrappers
```

**Important**:
- This is a **Squire directive**, not a Go toolchain directive
- The Go toolchain will ignore these comments
- Squire will parse and apply these experiments when running commands
- Two types of experiments:
  1. **Must-have**: Required for code to work (e.g., jsonv2 needs specific experiments)
  2. **Nice-to-have**: Performance/behavior tweaks (e.g., GC experiments)
- For now, treat all experiments as must-have (no distinction in directives)

**Why in go.mod**:
- go.mod is the single source of truth for the module
- Keeps experiment config with the module it affects
- Avoids "man with two watches" problem of duplicating config

## What NOT to Add

Based on project philosophy:
- Do NOT create commands that duplicate native `go` commands
- Do NOT add `squire gomod` - use `go mod` directly
- DO add commands that orchestrate multiple `go` commands
- DO add commands that automate Makefile-style tasks
- DO add multi-repo orchestration commands

Example of what TO add:
- `squire go ci` - Run fmt + vet + lint + test (replaces `make ci`)
- `squire go test-all` - Run tests with coverage + race detection (replaces `make test`)
- `squire workspace set` - Configure workspace directory
- `squire workspace discover` - Discover modules in directory tree
- `squire deps ensure` - Verify all deps have passing tests/lint
- `squire replace enable` - Enable local dev mode (go.work + replace directives) [placeholder verb]
- `squire replace disable` - Disable local dev mode (remove overrides) [placeholder verb]

**Note on Verbs**: Commands like `replace enable/disable` use placeholder verbs. Better verbs will be chosen as the right terminology reveals itself. Consider alternatives like `dev-on/dev-off`, `link/unlink`, `adopt-replaces`, etc. The important thing is the functionality, not the current verb choice.

Example of what NOT to add:
- `squire go build` - Just use `go build`
- `squire go mod tidy` - Just use `go mod tidy`

## Important Principles

### One Source of Truth
Don't duplicate information that can be discovered:
- Module info comes from go.mod files, not config
- Dependency relationships from go.mod, not config
- Config should only store user preferences and settings

Following ancient Chinese proverb: "Man with one watch knows the time; man with two watches never sure."

### Minimal Configuration
Store only what cannot be discovered. Discover everything else from:
- go.mod files (modules, dependencies, versions)
- go.work files (workspace members)
- .git directories (repo roots)
- GitHub API (repos, releases, workflows)

### Parsing Strategy
When parsing go.mod:
- Use `modfile.ParseLax()` during local dev (syntax validation only)
- Use `modfile.Parse()` when go.mod should be buildable
- Don't assume go.mod is always valid - it won't be during active development

## Potential Integrations & Related Tools

### GoReleaser
For projects that produce compiled binaries:
- Integration with GoReleaser for building cross-platform binaries
- Release workflow triggers GoReleaser after tests pass
- Interactive TUI for scaffolding .goreleaser.yml configuration

### goyek (Task Runner)
Considering incorporating functionality from github.com/goyek/goyek:
- Alternative to Makefiles for task running
- May integrate or provide similar functionality
- Evaluate if/how to incorporate into Squire's design

### gomodguard
Reference: https://github.com/ryancurrah/gomodguard
- Tool for blocking/allowing specific Go modules (useful for organizations)
- Track for potential future integration
- Not a priority for sole developer use case, but may be valuable for teams

### License Management
Future consideration: Tool to manage dependencies by OSS license
- Allow only specific licenses (e.g., MIT but not GPL)
- Useful for organizations with legal/compliance requirements
- Track licenses of all dependencies in workspace
- Alert when new dependency uses prohibited license

### Related Tools to Review
- aws-go-multi-module-repository-tools - Multi-module management (already identified as promising)
- Other multi-repo/workspace management tools as they're discovered

### Tool Consolidation Philosophy
Squire is designed to consolidate one-off Go development tools into a single CLI:
- Prevents forgetting about previously built tools
- Avoids accidentally recreating existing functionality
- Makes tool discovery easier (all commands in one place)
- Reduces context switching between different CLIs

When adding new functionality, consider:
1. Does this replace repetitive Makefile tasks?
2. Is this multi-step orchestration that can't be done with a single `go` command?
3. Would this be useful across multiple projects?
4. Does this help manage multi-repo workflows?

If yes to any of these, it belongs in Squire.
