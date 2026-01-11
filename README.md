# Gomion **/ˈɡɒm.jən/**

> Multi-repo Go development tool that automates workflows, manages workspaces, and consolidates one-off utilities.

[![License: Apache 2.0](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](LICENSE)
[![Go Version](https://img.shields.io/badge/Go-1.25%2B-00ADD8?logo=go)](https://go.dev/)

---

## What is Gomion?

Gomion is a CLI tool that solves daily friction in Go development across multiple repositories and modules. It replaces repetitive Makefiles, manages multi-repo workflows where `go.work` alone isn't enough, and consolidates scattered tools into one discoverable place.

**Core Philosophy**: Orchestrate workflows that would otherwise require Makefiles or shell scripts. Don't duplicate what `go` already does well.

## How to Pronounce

**GOM‑yun** — Short "o" (as in _pom_) + _yun_. Two syllables, first stressed.

**Not** _go‑MY‑on_ or _GO‑mee‑on_.

## Why Gomion?

Multi-repo Go development has recurring pain points:

- **Multi-repo chaos**: `go.work` and `replace` directives drift out of sync
- **Makefile proliferation**: Every project has similar `test`, `lint`, `build` targets
- **Forgotten tools**: One-off utilities get lost and accidentally recreated
- **Workspace friction**: Constantly referencing paths across related projects

Gomion provides workspace-aware orchestration and tool consolidation.

## Current Status

🚧 **Early Development** (Dec 2025) — Core concepts established, initial features implemented.

## Key Concepts

### Workspaces

Logical groupings of related Go modules. Define once, work without constant path references.

### In-Flux State

A module is "in-flux" (not ready for release) if ANY of:
1. Dirty working tree (uncommitted/untracked changes)
2. Commits not tagged
3. Tags not pushed

**Normal during development** — Most modules are in-flux. The workflow systematically releases them one-by-one.

### Leaf Algorithm

Find which in-flux module can be released next:
- Among all in-flux modules, find one whose dependencies are all clean
- Release bottom-up through the dependency tree
- Repeat until nothing is in-flux

### Release Automation Goal

Automate the manual multi-repo release workflow:
1. Find in-flux modules → 2. Find the leaf → 3. Prepare for release (tidy, vet, lint, test) → 4. Commit → 5. Tag and release → 6. Repeat

## Design Principles

1. **Orchestration Over Duplication** — Don't replace `go` commands; orchestrate them
2. **One Source of Truth** — Discover from `go.mod`/`go.work`/git, don't duplicate
3. **Minimal Configuration** — Store only what can't be discovered
4. **Workspace Awareness** — Work on configured workspaces without constant paths

## Planned Features

- Workspace management and discovery (with TUI)
- Multi-repo testing, linting, building across dependency trees
- Dev mode toggling (`go.work` + `replace` directive management)
- Interactive commit workflow with AI-generated messages
- GitHub workflow integration and release automation
- GoReleaser integration for binary releases
- API stability management across modules
- ClearPath custom linter

See [ROADMAP.md](ROADMAP.md) for details.

## Installation

_Installation instructions will be added once initial release is available._

For now, build from source:

```bash
git clone https://github.com/mikeschinkel/gomion.git
cd gomion
go install ./cmd/...
```

## Project Structure

Go workspace with three modules:

- **`cmd/`** — CLI entry point
- **`gommod/`** — Core library (importable by other tools)
- **`test/`** — Test module (avoids circular dependencies)

## Documentation

- **[ROADMAP.md](ROADMAP.md)** — Planned features and status
- **[CLAUDE.md](CLAUDE.md)** — Architectural guide for AI-assisted development
- **[DONE.md](DONE.md)** — Recently completed work
- **[PERFORMANCE.md](PERFORMANCE.md)** — BubbleTea performance patterns
- **[LICENSE](LICENSE)** — Apache License 2.0

## Contributing

Gomion welcomes contributions but is still in early development.

**Before contributing**:
- Read [ROADMAP.md](ROADMAP.md) to understand vision and design
- Read [CLAUDE.md](CLAUDE.md) for architectural patterns
- Open an issue to discuss significant changes

## License

Apache License, Version 2.0 — See [LICENSE](LICENSE) for details.

## Author

**Mike Schinkel** — [@mikeschinkel](https://github.com/mikeschinkel)

## Built With

- [go-cliutil](https://github.com/mikeschinkel/go-cliutil) — Command framework
- [go-cfgstore](https://github.com/mikeschinkel/go-cfgstore) — Configuration management
- [go-dt](https://github.com/mikeschinkel/go-dt) — Type-safe data types

---

**Note**: Opinionated tool prioritizing the author's workflows, but designed to be useful to others with similar multi-repo pain points.
