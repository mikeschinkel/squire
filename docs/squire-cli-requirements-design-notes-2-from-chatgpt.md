# Gomion CLI – Requirements & Design Notes

This document summarizes current requirements, background, and design directions for the **Gomion** CLI.
It is intended as an internal project artifact to use when refining Gomion’s specification and architecture.

---

## 1. High-Level Vision

Gomion is a **Go-centric developer CLI** that:

* Orchestrates **multi-module, multi-repo Go development**, particularly:

  * `go.work` workspaces
  * `replace` directives in `go.mod`
* Encodes and enforces the **ClearPath** coding style via a linter.
* Manages a new class of **“non-dependency dependencies”**, exemplified by [`go-doterr`](https://github.com/mikeschinkel/go-doterr), using a single-file embed model instead of traditional module dependencies.
* Provides opinionated automation around:

  * Testing, linting, building, fuzzing
  * CI workflows and GoReleaser
  * Scaffolding new Go projects (especially CLIs using `go-cliutil`)
* Eventually becomes the **central toolbox** for many one-off utilities (e.g., `ssot`, `readme-merge`) so they are not forgotten or re-implemented.

The project is intended to be **open source** and opinionated, but structured so other developers and organizations can adopt, extend, and contribute.

---

## 2. Core Problems Gomion Is Addressing

### 2.1. Real-world `go.work` + `replace` workflows

Empirically:

* `go.work` alone is insufficient in complex, multi-repo setups.
* `replace` alone is noisy and difficult to manage.
* In practice, many workflows require both to cooperate.

Gomion’s core job is to orchestrate:

* `` – defines which local module directories participate in a workspace.
* `` – defines how module paths are resolved for a particular `go.mod` (e.g., local fork paths, non-workspace locations).

Key goals:

* Treat `go.work` and `replace` as **orthogonal knobs**, not mutually exclusive.
* Provide **high-level commands** to:

  * “Turn dev overrides on/off” (names TBD) for a workspace or repo.
  * Ensure `go.work` is consistent with the workspace definition.
  * Add/remove Gomion-managed `replace` blocks safely, without touching user-managed ones.
* Avoid brittle manual editing of `go.mod` and `go.work` for routine tasks.

### 2.2. ClearPath coding style

ClearPath is a personal Go style you want to formalize:

* Prefer **one return at the end** of a function.
* Use `goto end` and a shared clean-up block instead of multiple early returns.
* Keep control flow visually straightforward (“clear path”), with minimal nesting and surprises.

Requirements:

* Gomion will eventually provide a **ClearPath linter** as part of `gomion go lint` (or equivalent).
* The linter will focus on structural rules that support ClearPath:

  * Control flow patterns.
  * Return placement.
  * Label usage (`end:`).
* This should be optional but first-class, not a hidden add-on.

### 2.3. Non-dependency dependencies (e.g., `go-doterr`)

[`go-doterr`](https://github.com/mikeschinkel/go-doterr) is the prototype for a category of libraries that:

* Are designed to be **embedded as a single Go file** (e.g., `doterr.go`) into a project or package instead of imported as a traditional dependency.
* Use only the standard library, and expose a small, conventional interface (e.g., `error`, a `KV` type) to stay interoperable.
* Avoid:

  * Supply-chain risk from updating third-party modules.
  * Complexity and friction from vendoring entire dependency graphs.

Conceptually:

* A **“non-dependency dependency”** is a reusable Go file dropped into many repositories.
* Gomion’s job is to **manage and update** those embedded copies across a workspace when requested.

Existing `go-doterr` design work includes:

* A dedicated CLI PRD (with concepts like `dot`, `copy`, and `ignore` modes per package).
* TUI for per-package configuration and bulk operations.

Gomion should **subsume this CLI**:

* Represent doterr settings under Gomion config (see §4).
* Provide commands to:

  * Initialize doterr usage in a repo.
  * Apply doterr template(s) (i.e., generate or update `doterr.go`).
  * Check for drift across packages.
  * Run TUIs for interactive configuration.

`go-doterr` becomes the canonical example of Gomion’s non-dependency dependency support pattern.

---

## 3. “One Watch” Principle for Configuration

Guiding principle:

> “Man with one watch knows the time; man with two watches never sure.”

For any given concern, there should be a **single canonical source of truth**. Everything else is either:

* A **derived artifact**, or
* A **view** that can be regenerated from the canonical source.

Implications:

* **Experiments & low-level build toggles**:

  * Canonical config should live in the source tree, e.g. in `go.mod` as a directive comment:

    ```go
    //gomion:goexperiments=arenas,regabiwrappers
    ```
  * Gomion reads this and sets `GOEXPERIMENT` when invoking `go test`, `go build`, fuzzing, etc.
  * `.gomion` should **not** become a second “watch” for the same data.

* **Doterr usage**:

  * Canonical config is in Gomion’s config (e.g. a `doterr` section) describing package modes: `copy`, `dot`, `ignore`.
  * Generated `doterr.go` files and `go:generate` lines are derived from that.

* **Workspaces**:

  * Canonical definitions (names, roots, owned module patterns) live in user-level Gomion config (e.g. `~/.config/gomion/config.json`).
  * `go.work` files are the concrete representation used by the Go tool, managed by Gomion according to the workspace definition.

In short: Gomion must avoid duplicating configuration between `.gomion`, inline comments, and other files. There should be **one authoritative place per concept**.

---

## 4. Configuration Layers & Ownership

### 4.1. Hierarchy

Conceptual hierarchy Gomion works with:

* **Workspace** – logical grouping of repos & modules (e.g., XMLUI universe).
* **Repo** – a git repository.
* **Module** – a `go.mod` root.
* **Owned module** – a module that is under your control or explicitly marked as such.

### 4.2. Configuration locations

* **User level** (global):

  * Location: `~/.config/gomion/config.json` (or similar).
  * Canonical info:

    * Workspaces:

      * Name (e.g. `"xmlui"`).
      * Root path.
      * Patterns for “owned” module paths (e.g. `github.com/mikeschinkel/*`, `github.com/xmlui-org/*`).
    * Current workspace.
    * Language configuration (see §9).

* **Repo / module level**:

  * Location: `.gomion/gommod/gomion.json` (directory-based, consistent with `go-cfgstore`).
  * Contains **Gomion-specific metadata**, for example:

    * Module role(s): `library`, `app`, `test-support`.
    * Release behavior:

      * Whether to use GoReleaser.
      * Which GoReleaser config file to use, etc.
    * Doterr settings (packages and their modes).
    * Future extensions (license policy overrides, project-specific flows, etc.).
  * It should **not** duplicate things that can be derived from `go.mod` (module path, versions) or inline directives.

### 4.3. Owned modules

“Owned” modules must be configurable per:

* **Workspace** – via path patterns (e.g. `github.com/mikeschinkel/*`).
* **Repo** – overrides or special cases defined in `.gomion/gommod/gomion.json`.
* **Module** – per-module entries in `.gomion/gommod/gomion.json` (e.g., test module vs CLI module).

Resolution precedence:

1. Module-level config (if present).
2. Repo-level config.
3. Workspace patterns.
4. Default: not owned.

Owned status influences:

* Whether Gomion treats a module as part of dev overrides (go.work + replace).
* Whether it participates in workspace-wide test/lint/release flows.
* Whether Gomion expects GitHub workflows & GoReleaser setups for that module.

---

## 5. Workspace Management & Discovery

### 5.1. Workspace commands

Gomion should provide `gomion workspace` subcommands to manage workspaces:

* `gomion workspace init <name> --root <dir>`:

  * Define a new workspace in user config.
  * Optionally create an initial `go.work` at root.

* `gomion workspace use <name>`:

  * Set the current workspace in user config.

* `gomion workspace list` / `status`:

  * Show defined workspaces and which one is current.

### 5.2. Workspace discovery (TUI)

`workspace discover` is inherently interactive and should offer a TUI:

* Command: `gomion workspace discover`.
* Behavior:

  1. Scan configurable roots (e.g. `$HOME/Projects`, `$HOME/src`, etc.) for directories containing both `.git` and `go.mod`.
  2. Cluster repos into tentative workspaces based on directory structure.
  3. Present candidates in a TUI:

     * Create / rename / remove workspaces.
     * Assign repos to workspaces.
     * Mark repos/modules as owned or not.
  4. Persist resulting workspace config to `~/.config/gomion/config.json`.

A non-interactive mode (`--non-interactive`) should be available for CI or scripted usage.

---

## 6. `go.work` and `replace` Orchestration

### 6.1. `go.work` management

Gomion’s responsibilities include:

* Ensuring the workspace root `go.work` is consistent with:

  * User-defined workspace, and
  * Owned modules within that workspace.

Typical command: `gomion go sync-work` (name TBD):

* Ensure all owned modules that should be part of the workspace appear in `use` directives.
* Optionally remove `use` entries for missing/obsolete modules.
* Ensure `go.work`/`go.work.sum` are either:

  * Ignored by git (default), or
  * Tracked intentionally, according to workspace policy.

### 6.2. Dev overrides (names TBD)

Gomion should provide the ability to “toggle dev mode” for a workspace/repo:

* “Dev ON” (placeholder name):

  * Ensure workspace `go.work` is in place and includes the right `use` entries.
  * Add Gomion-managed `replace` directives as needed (e.g., for owned modules outside workspace root).
  * Only operate inside **clearly delimited Gomion blocks**, e.g.:

    ```go
    // gomion:replaces begin
    replace github.com/mikeschinkel/foo => /Users/mike/Projects/ws/foo
    replace github.com/mikeschinkel/bar => /Users/mike/Projects/ws/bar
    // gomion:replaces end

    ```
