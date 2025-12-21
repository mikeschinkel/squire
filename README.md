# Squire

> A Multi-purpose CLI _(a.k.a. a "Swiss Army Knife")_ CLI for Go developers that consolidates project tooling, multi-repo workflows, and one-off utilities into a single, cohesive command-line experience.

[![License: Apache 2.0](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](LICENSE)
[![Go Version](https://img.shields.io/badge/Go-1.25%2B-00ADD8?logo=go)](https://go.dev/)

---

## What is Squire?

Squire is a command-line tool designed to solve the daily friction points of Go development across multiple repositories and modules. It replaces repetitive `Makefiles`, manages the reality of multi-repo development where `go.work` alone isn't enough, and brings together scattered one-off tools into one discoverable place.

**Core Philosophy**: Only add commands that orchestrate multiple steps or automate workflows that would otherwise require a `Makefile` or shell scripts. Do not duplicate what `go` already does well.

## Why Squire?

Over years of Go development, common pain points emerge:

- **Multi-repo chaos**: Working on several related modules means juggling both `go.work` and `replace` directives, which drift out of sync
- **`Makefile` proliferation**: Every project has a similar `Makefile` with `test`, `lint`, `build`, `ci` targets
- **Forgotten tools**: One-off utilities get written, then forgotten and accidentally recreated months later
- **Workspace friction**: Constantly referencing paths when running commands across a set of related projects

Squire plans to address these by providing workspace-aware orchestration and consolidating tooling in one place.

## Current Status

🚧 **Early Development** - Squire is brand new in Dec 2025 and now under active initial development. The concept is established, but few features are implemented yet.

## Planned Features

- **Workspace Management**: Define and work with logical groupings of Go modules
- **Multi-Repo Orchestration**: Test, lint, and build across module dependency trees
- **Dev Mode Toggling**: Switch between local development (with `replace` directives) and release mode
- **Embedded Dependencies**: Manage single-file utilities that are copied into projects rather than imported
- **GitHub Workflow Integration**: Ensure repos have proper test and release workflows
- **GoReleaser Integration**: Scaffold and manage binary release configurations
- **Interactive TUI**: For complex operations like workspace discovery and configuration
- **ClearPath Linting**: Custom linter for opinionated Go coding style

See [FEATURES.md](FEATURES.md) for comprehensive details about planned features, requirements, and design decisions.

## Documentation

- **[FEATURES.md](FEATURES.md)** - Detailed requirements, background, and design decisions (for contributors and future reference)
- **[CLAUDE.md](CLAUDE.md)** - Architectural guide for AI-assisted development with Claude Code
- **[LICENSE](LICENSE)** - Apache License 2.0

## Installation

_Installation instructions will be added once initial release is available._

For now, to build from source:

```bash
git clone https://github.com/mikeschinkel/squire.git
cd squire
go install ./cmd/...
```

## Quick Start

_Quick start guide will be added as features are implemented._

## Project Structure

This project uses Go workspaces with three modules:

- **`cmd/`** - CLI entry point
- **`squirepkg/`** - Core library (importable by other tools)
- **`test/`** - Test module

## Contributing

Squire is designed to be open source and welcomes contributions! However, it's still in early development.

**Before contributing**:
- Read [FEATURES.md](FEATURES.md) to understand the vision and design philosophy
- Read [CLAUDE.md](CLAUDE.md) for architectural patterns and conventions
- Open an issue to discuss significant changes before implementing

## Design Principles

1. **One Source of Truth**: Don't duplicate information that can be discovered from `go.mod`, `go.work`, or other canonical sources
2. **Minimal Configuration**: Store only what cannot be discovered automatically
3. **Workspace Awareness**: Operations should work on configured workspaces without constant path references
4. **Orchestration Over Duplication**: Don't replace `go` commands; orchestrate them for complex workflows

## License

This project is licensed under the Apache License, Version 2.0 - see the [LICENSE](LICENSE) file for details.

## Author

**Mike Schinkel**
- GitHub: [@mikeschinkel](https://github.com/mikeschinkel)

## Acknowledgments

Built with:
- [go-cliutil](https://github.com/mikeschinkel/go-cliutil) - Command framework
- [go-cfgstore](https://github.com/mikeschinkel/go-cfgstore) - Configuration management
- [go-dt](https://github.com/mikeschinkel/go-dt) - Type-safe data types

---

**Note**: This is an opinionated tool that prioritizes the author's workflows and preferences, but is designed to be useful to others who share similar pain points in Go development.
