# Squire — Multi-Purpose CLI Tool for Go-centric Developers

## Features, Requirements, Background, and Design Notes

This document captures the early requirements, background, and design decisions for **Squire**, a Go-centric developer CLI that centralizes project tooling and workflows.

It’s meant as a long-lived reference for future you (and future contributors), not as user-facing documentation.

---

## 1. High-level vision

Squire is a **single CLI** that:

- Envisions potentially supporting multiple languages, but starts with Go.
- Orchestrates **Go workspaces**, `go.work`, and `replace` usage in a multi-repo, multi-module world.
- Encodes and enforces a _**ClearPath**_ coding style via a dedicated linter.
- Manages a new category of **“non-dependency dependencies”** like [`go-doterr`](https://github.com/mikeschinkel/go-doterr/blob/main/README.md), where a single Go file is embedded into many packages instead of pulled as a normal module dependency. See [more details](https://github.com/mikeschinkel/go-doterr/blob/3ed692ed7bf5b3f3af0c1db5305cfd2012780e10/CLI_PRD.md)
- Provides an opinionated but configurable layer around:
  - Testing (`squire go test`)
  - Linting (`squire go lint`)
  - Building & releasing (`squire go build` + GoReleaser scaffolding)
  - Fuzzing (`squire go fuzz`, inspired by the existing infinite fuzz script)
- Eventually becomes the **home** for many of the one-off Go tools you’ve written (`ssot`, `readme-merge`, etc.), so they are discoverable and reusable.

The project is intended to be **open-source**, but unapologetically **opinionated** around Go and your preferred workflows.

---

## 2. Core problems Squire is meant to solve

### 2.1. go.work + replace orchestration

Empirically:

- `go.work` alone is **not** sufficient for real multi-repo setups.
- `replace` alone is fragile, noisy in `go.mod`, and painful to keep in sync.
- In practice, complex workspaces often need **both**.

Squire treats:

- `go.work` as **“which module directories are in this workspace”**, and
- `replace` as **“how specific module paths are resolved in this repo”**,

and acts as the orchestrator that keeps them consistent with a higher-level concept of:

- **Workspace** (user-level concept)
- **Repo** (git root)
- **Module** (`go.mod` root)
- **Owned module** (user controls its versions/behavior vs external deps)

Key requirements:

- Squire understands **owned modules** per:
  - workspace,
  - repo, and
  - module.
- It must support **go.work** but also:
  - Manage its `use` entries.
  - Keep `go.work` and `go.work.sum` aligned with workspace config.
  - Ensure they are appropriately ignored or tracked in git, depending on policy.
- It must be able to **add/remove dev overrides** (work+replace) with a single command pair (names TBD; current placeholders are “apply/unapply”).

### 2.2. ClearPath coding style

ClearPath is a personal Go style you want to formalize and enforce via a linter. High-level characteristics include (non-exhaustive):

- Prefer a single return at the end of a function, with clean-up code near that return.
- Use of `goto end` for clean-up rather than multiple early returns, to keep control flow visually “straight.”
- Minimal indentation and clear, obvious paths through code.

Requirements:

- Squire will eventually provide a **ClearPath linter** (e.g. `squire go lint --style=clearpath`) that can be run:
  - locally,
  - in CI,
  - and potentially as an editor/IDE hook.
- The style rules are **opinionated**, and target **your** conventions first; others can adopt or not.
- ClearPath is first-class in Squire’s docs and design, not an afterthought.

### 2.3. Non-dependency dependencies (`go-doterr`)

`go-doterr` is the prototype of a category you want to promote: a **single-file embeddable utility** that’s copied into a project (or package) instead of imported as a normal dependency. 

Key properties:

- One canonical file (e.g. `doterr.go`) that:
  - Is pure stdlib.
  - Exports only `error` + minimal interfaces (`KV`) to remain interoperable.
- Primary usage is **embedding** the file, not `go get`.
- This avoids:
  - Supply-chain risk from third-party module updates.
  - The friction and noise of vendoring entire dependency trees.
- You still want tooling to:
  - Keep copies up-to-date.
  - Avoid drift.
  - Manage template versions.

For a design that informs this design, See [here](https://github.com/mikeschinkel/go-doterr/blob/3ed692ed7bf5b3f3af0c1db5305cfd2012780e10/CLI_PRD.md)

Squire’s design incorporates that:

- Squire will subsume the doterr CLI’s responsibilities:
  - Manage a `doterr.go.tmpl` template and render per package.
  - Maintain per-repo config (`doterr.json` or equivalent) that defines whether each package uses:
    - `copy` (embed file),
    - `dot` (dot-import),
    - or `ignore`.
  - Provide `apply`, `check`, and `init` flows with TUI for interactive selection.
- The **doterr use-case** is the canonical example of Squire’s “non-dependency dependency” support, but the pattern should be general enough to apply to other single-file tools in the future.

---

## 3. Configuration and the “one watch” principle

You explicitly want to avoid “man with two watches never sure.” That means:

- **One canonical source of truth** for each kind of configuration.
- Other files are either:
  - Derived artifacts, or
  - Minimal overlays, not competing sources.

Some implications:

- For **Go experiments** and other low-level build toggles:
  - You prefer **inline directives** in `go.mod` as canonical, e.g.:

    ```go
    //squire:goexperiments=arenas,regabiwrappers
    ```

  - Squire parses those and sets `GOEXPERIMENT` when running `go` commands, fuzzing, etc.
  - `.squire` config should not become a second, conflicting “watch” for the same information.

- For **doterr** and similar embed tools:
  - Use `.squire/squire.json` instead of `doterr.json` as the canonical source for which packages use `copy` vs `dot` vs `ignore`.
  - Generated files and `go:generate` directives follow from that.

- For **workspaces**:
  - User-level config (e.g. `~/.config/squire/config.json`) is the canonical source of:
    - Workspace names and roots.
    - Owned-module patterns.
    - Possibly the “current” workspace.
  - Workspace root `go.work` files are the concrete representation for the Go tool, but Squire shouldn’t require you to maintain both manually.

Design note: **.squire vs inline config**

- `.squire` (directory or file) is expected to hold Squire-specific metadata and state.
- Inline directives (like `//squire:goexperiments=...`) live in code and are treated as the canonical configuration where they exist.
- Squire must be very deliberate about:
  - Which things are configured via `.squire`, and
  - Which things are configured via inline comments.

The overarching guideline: **pick one primary place per concern, and have Squire keep derived views in sync.**

---

## 4. Workspaces, repos, and modules

Conceptual hierarchy:

- **Workspace** – a logical grouping of repos and Go modules (e.g. all XMLUI projects).
- **Repo** – a git repository; may contain one or more Go modules.
- **Module** – a `go.mod` unit; a repo may have multiple (e.g. `./cmd`, `./test`, `./pkg`).
- **Owned module** – a module that:
  - Lives under your control (e.g. `github.com/mikeschinkel/*`),
  - Or is marked as owned in config (per workspace/repo/module).

Requirements:

- Squire should support **user-level workspace config**, likely under `~/.config/squire/…`, listing:
  - Workspaces and their root directories.
  - Patterns for “owned” module paths.
- In each repo, Squire should support a **.squire** directory (e.g. `.squire/squire.json`) for repo/module-specific metadata (role, release behavior, etc.), but **not** for things that can be reliably inferred from `go.mod` or inline directives.
- Squire must always be able to discover:
  - Repos via `.git`.
  - Modules via `go.mod`.
- Squire should expose `squire workspace` commands, including:
  - `squire workspace discover` with a **TUI** to scan known roots, group repos into workspaces, and adjust settings interactively.
  - `squire workspace use <name>` to set the current workspace.

---

## 5. Interactive TUI requirements

Some parts of Squire are **inherently interactive** and should be implemented as TUIs:

1. **Workspace discovery (`squire workspace discover`)**
   - Scans directories (e.g. `$HOME/Projects`, etc.) for `go.mod` + `.git`.
   - Presents candidate repos/workspaces in a TUI.
   - Lets you:
     - Group repos into workspaces.
     - Rename workspaces.
     - Mark repos/modules as owned/not owned.
     - Save results into user config.

2. **Doterr / embed-tool configuration**
   - Equivalent to the doterr CLI TUI described in the PRD:
     - List packages.
     - Toggle mode: `dot` → `copy` → `ignore`.
     - Manage defaults and overrides. 

3. **GoReleaser scaffolding**
   - When generating/updating release workflows:
     - TUI to choose:
       - OS/arch matrix.
       - Whether the module is library vs CLI.
       - How to trigger releases (tags, manual, etc.).
     - TUI to confirm or adjust guessed GitHub owner/repo, project name, etc.

In all cases:

- Non-interactive flags (e.g. `--non-interactive`, `--auto=<mode>`) should exist for CI and scripting.
- TUI frameworks like `bubbletea` are acceptable choices, but this doc doesn’t lock in a specific library.

---

## 6. Go-specific command layer (`squire go`)

Squire’s Go backend will provide a family of subcommands (names are not final):

- `squire go test`
  - Runs tests across owned modules in the current workspace / repo.
  - Respects Go experiments (via directives + `GOEXPERIMENT`).
  - Can be extended to integrate fuzzing, coverage, etc.

- `squire go lint`
  - Wraps `golangci-lint` (or similar) plus:
    - ClearPath linter.
    - Optional policy linters like gomodguard (for orgs that want it).

- `squire go build`
  - Builds CLI modules (`role: app`) according to Squire’s conventions.
  - Integrates with GoReleaser scaffolding for release builds.

- `squire go fuzz`
  - Infinite fuzz testing behavior based on the existing `infinite-fuzz.sh` script:
    - Discover `Fuzz*` functions.
    - Run `go test -run=^$ -fuzz=^FuzzName$` in loops.
    - Handle signals and clean shutdown.
  - Reuses experiments setup where needed.

- `squire go dev-*` (names TBD)
  - Commands to **toggle dev overrides** (go.work + replace) on/off.
  - Should handle:
    - Adding appropriate `use` entries to `go.work`.
    - Adding/removing Squire-managed `replace` blocks.
    - Leaving user-managed `replace` directives untouched, unless explicitly asked to adopt or normalize them.

Implementation detail (internal design, not UX):

- Using a **language backend** interface (e.g. `LanguageBackend`) is the right abstraction, but there’s no urgency to implement multi-language dispatch until a second language is actually added.
- Config should still anticipate multiple languages (e.g. `currentLanguage: "go"`; per-language blocks), even if only Go is implemented initially.

---

## 7. GoReleaser & CI workflows

Requirements around release automation:

- **Tagging and releasing** should be done via **GitHub Actions**, not via Squire directly.
  - Squire can generate or update workflows, but the actual tag/release happens in CI.
- The **release workflow** should:
  - Run `squire go test` / lint / vet first.
  - Only tag & release on success.
  - Use GoReleaser for binary releases when appropriate.

Squire should provide:

- `squire go release init`
  - TUI to configure:
    - CLI vs library module.
    - GoReleaser config defaults (entrypoint, binary name, OS/arch).
  - Generate:
    - A sensible `.goreleaser.yaml` (or reuse `goreleaser init` and then refine).
    - `.github/workflows/release.yml` wired to tags and GoReleaser.
- Optionally:
  - `squire go release --local` to run the Release flow locally for sanity before pushing tags.

---

## 8. Multi-language future: “current language”

Even though Go is the primary focus, the design should leave room for other languages (e.g. Zig) to reuse Squire’s concepts.

Planned approach:

- **Config supports languages up front**:

  ```jsonc
  {
    "currentLanguage": "go",
    "languages": {
      "go":  { "enabled": true },
      "zig": { "enabled": false }
    }
  }
````

* `squire test` and similar commands:

    * Default to `currentLanguage`.
    * Allow overrides: `squire test --lang=zig` or shorthand like `--zig`.
* Internally:

    * Language backends implement the common operations (`Test`, `Build`, `Lint`, etc.).
    * Go backend will exist first; others added only when needed.

No need to implement the full registry immediately, but the **configuration model** should not preclude it.

---

## 9. Templates & scaffolding

Squire should eventually help scaffold:

* **Go CLI projects**, using your own `go-cliutil` instead of Cobra/urfave-cli.

    * `squire go new-cli <name>`:

        * Creates `cmd/<name>/main.go` with a `RunCLI()` pattern.
        * Sets up `go.mod` if needed.
        * Optionally creates `.squire` config marking the module as `role: app` and enabling GoReleaser scaffolding.

* **Template-based project layouts**:

    * Where templates already include desired imports and wiring, there’s no need for a separate “packages to add” list.
    * Squire should avoid duplicating this – templates should be the primary source of truth.

---

## 10. Policy & governance (future)

While you personally don’t need org-level tools like gomodguard today, it’s worth noting:

* **gomodguard** exists and can enforce allow/block lists for modules in `go.mod`. It’s useful for teams controlling which dependencies are allowed.
* You previously designed a tool for **license-based module governance** (e.g. allow MIT, forbid GPL). That could be another Squire module in the future.

Squire should keep these in mind as **future features**:

* A policy layer in `.squire` (per workspace/org) that:

    * Expresses license and dependency rules.
    * Optionally generates or populates configs for tools like gomodguard / golangci-lint.
* CI integration via `squire go deps check` or similar.

---

## 11. Integration with existing tools & libraries

Squire is intended to sit on top of, not replace:

* `go` command and standard tooling.
* `GoReleaser` for binaries.
* `goyek` for task automation (Squire can expose reusable Go APIs that goyek flows call).
* `go-doterr` for embeddable error handling.
* Existing one-off tools (`ssot`, `readme-merge`) which can be pulled into Squire as subcommands over time.

Design preferences:

* Squire should be usable both as:

    * A **CLI** (`cmd/squire`), and
    * A **library** (Go packages that goyek flows or other programs can import).
* No gRPC or daemon process is required up front; everything can run in a single process via library calls.

---

## 12. Open questions / decisions to revisit

Some questions intentionally left open for future you (or contributors) to decide:

1. **Command verbs** for dev overrides:

    * Current placeholders: “apply” / “unapply”.
    * Candidates: `dev-on` / `dev-off`, `link` / `unlink`, `activate` / `deactivate`, etc.
    * Decision should prioritize clear meaning over cleverness.

2. **Canonical location for certain configs**:

    * Experiments: inline comment in `go.mod` vs `.squire`.
    * How much Squire metadata (e.g. roles, release settings) belongs in `.squire` vs other files.

3. **How aggressively to adopt/normalize existing `replace` directives**:

    * Default stance is conservative: only touch Squire-tagged sections.
    * A more aggressive “adopt everything” mode may be useful when standardizing a fleet of repos.

4. **Scope of non-Go features**:

    * How much functionality from `ssot`, `readme-merge`, and others should be pulled in.
    * Whether to keep some tools as separate CLIs vs subcommands.

5. **Language backends**:

    * When (and whether) to introduce a second language backend (e.g. Zig) and what minimum features are required to justify it.

---

This document should evolve as Squire becomes real code. For now, it serves as the shared memory of what you were trying to achieve: one CLI, one opinionated orchestrator, handling Go workspaces, ClearPath, non-dependency dependencies like `go-doterr`, experiments, CI/release, and eventually more – without ever giving you “two watches” for the same concept.

