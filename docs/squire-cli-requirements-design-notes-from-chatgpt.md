# Gomion CLI – Requirements & Design Notes

This document summarizes current requirements, background, and design directions for the **Gomion** CLI.
It is intended as an internal project artifact to use when refining Gomion’s specification and architecture.

---

## 1. High-Level Vision

Gomion is a **Go-centric developer CLI** that:

- Orchestrates **multi-module, multi-repo Go development**, particularly:
  - `go.work` workspaces
  - `replace` directives in `go.mod`
- Encodes and enforces the **ClearPath** coding style via a linter.
- Manages a new class of **“non-dependency dependencies”**, exemplified by [`go-doterr`](https://github.com/mikeschinkel/go-doterr), using a single-file embed model instead of traditional module dependencies.
- Provides opinionated automation around:
  - Testing, linting, building, fuzzing
  - CI workflows and GoReleaser
  - Scaffolding new Go projects (especially CLIs using `go-cliutil`)
- Eventually becomes the **central toolbox** for many one-off utilities (e.g., `ssot`, `readme-merge`) so they are not forgotten or re-implemented.

The project is intended to be **open source** and opinionated, but structured so other developers and organizations can adopt, extend, and contribute.

---

## 2. Core Problems Gomion Is Addressing

### 2.1. Real-world `go.work` + `replace` workflows

Empirically:

- `go.work` alone is insufficient in complex, multi-repo setups.
- `replace` alone is noisy and difficult to manage.
- In practice, many workflows require both to cooperate.

Gomion’s core job is to orchestrate:

- **`go.work`** – defines which local module directories participate in a workspace.
- **`replace`** – defines how module paths are resolved for a particular `go.mod` (e.g., local fork paths, non-workspace locations).

Key goals:

- Treat `go.work` and `replace` as **orthogonal knobs**, not mutually exclusive.
- Provide **high-level commands** to:
  - “Turn dev overrides on/off” (names TBD) for a workspace or repo.
  - Ensure `go.work` is consistent with the workspace definition.
  - Add/remove Gomion-managed `replace` blocks safely, without touching user-managed ones.
- Avoid brittle manual editing of `go.mod` and `go.work` for routine tasks.

### 2.2. ClearPath coding style

ClearPath is a personal Go style you want to formalize:

- Prefer **one return at the end** of a function.
- Use `goto end` and a shared clean-up block instead of multiple early returns.
- Keep control flow visually straightforward (“clear path”), with minimal nesting and surprises.

Requirements:

- Gomion will eventually provide a **ClearPath linter** as part of `gomion go lint` (or equivalent).
- The linter will focus on structural rules that support ClearPath:
  - Control flow patterns.
  - Return placement.
  - Label usage (`end:`).
- This should be optional but first-class, not a hidden add-on.

### 2.3. Non-dependency dependencies (e.g., `go-doterr`)

[`go-doterr`](https://github.com/mikeschinkel/go-doterr) is the prototype for a category of libraries that:

- Are designed to be **embedded as a single Go file** (e.g., `doterr.go`) into a project or package instead of imported as a traditional dependency.
- Use only the standard library, and expose a small, conventional interface (e.g., `error`, a `KV` type) to stay interoperable.
- Avoid:
  - Supply-chain risk from updating third-party modules.
  - Complexity and friction from vendoring entire dependency graphs.

Conceptually:

- A **“non-dependency dependency”** is a reusable Go file dropped into many repositories.
- Gomion’s job is to **manage and update** those embedded copies across a workspace when requested.

Existing `go-doterr` design work includes:

- A dedicated CLI PRD (with concepts like `dot`, `copy`, and `ignore` modes per package).
- TUI for per-package configuration and bulk operations.

Gomion should **subsume this CLI**:

- Represent doterr settings under Gomion config (see §4).
- Provide commands to:
  - Initialize doterr usage in a repo.
  - Apply doterr template(s) (i.e., generate or update `doterr.go`).
  - Check for drift across packages.
  - Run TUIs for interactive configuration.

`go-doterr` becomes the canonical example of Gomion’s non-dependency dependency support pattern.

---

## 3. “One Watch” Principle for Configuration

Guiding principle:

> “Man with one watch knows the time; man with two watches never sure.”

For any given concern, there should be a **single canonical source of truth**. Everything else is either:

- A **derived artifact**, or
- A **view** that can be regenerated from the canonical source.

Implications:

- **Experiments & low-level build toggles**:
  - Canonical config should live in the source tree, e.g. in `go.mod` as a directive comment:
    ```go
    //gomion:goexperiments=arenas,regabiwrappers
    ```
  - Gomion reads this and sets `GOEXPERIMENT` when invoking `go test`, `go build`, fuzzing, etc.
  - `.gomion` should **not** become a second “watch” for the same data.

- **Doterr usage**:
  - Canonical config is in Gomion’s config (e.g. a `doterr` section) describing package modes: `copy`, `dot`, `ignore`.
  - Generated `doterr.go` files and `go:generate` lines are derived from that.

- **Workspaces**:
  - Canonical definitions (names, roots, owned module patterns) live in user-level Gomion config (e.g. `~/.config/gomion/config.json`).
  - `go.work` files are the concrete representation used by the Go tool, managed by Gomion according to the workspace definition.

In short: Gomion must avoid duplicating configuration between `.gomion`, inline comments, and other files. There should be **one authoritative place per concept**.

---

## 4. Configuration Layers & Ownership

### 4.1. Hierarchy

Conceptual hierarchy Gomion works with:

- **Workspace** – logical grouping of repos & modules (e.g., XMLUI universe).
- **Repo** – a git repository.
- **Module** – a `go.mod` root.
- **Owned module** – a module that is under your control or explicitly marked as such.

### 4.2. Configuration locations

- **User level** (global):
  - Location: `~/.config/gomion/config.json` (or similar).
  - Canonical info:
    - Workspaces:
      - Name (e.g. `"xmlui"`).
      - Root path.
      - Patterns for “owned” module paths (e.g. `github.com/mikeschinkel/*`, `github.com/xmlui-org/*`).
    - Current workspace.
    - Language configuration (see §9).

- **Repo / module level**:
  - Location: `.gomion/gommod/gomion.json` (directory-based, consistent with `go-cfgstore`).
  - Contains **Gomion-specific metadata**, for example:
    - Module role(s): `library`, `app`, `test-support`.
    - Release behavior:
      - Whether to use GoReleaser.
      - Which GoReleaser config file to use, etc.
    - Doterr settings (packages and their modes).
    - Future extensions (license policy overrides, project-specific flows, etc.).
  - It should **not** duplicate things that can be derived from `go.mod` (module path, versions) or inline directives.

### 4.3. Owned modules

“Owned” modules must be configurable per:

- **Workspace** – via path patterns (e.g. `github.com/mikeschinkel/*`).
- **Repo** – overrides or special cases defined in `.gomion/gommod/gomion.json`.
- **Module** – per-module entries in `.gomion/gommod/gomion.json` (e.g., test module vs CLI module).

Resolution precedence:

1. Module-level config (if present).
2. Repo-level config.
3. Workspace patterns.
4. Default: not owned.

Owned status influences:

- Whether Gomion treats a module as part of dev overrides (go.work + replace).
- Whether it participates in workspace-wide test/lint/release flows.
- Whether Gomion expects GitHub workflows & GoReleaser setups for that module.

---

## 5. Workspace Management & Discovery

### 5.1. Workspace commands

Gomion should provide `gomion workspace` subcommands to manage workspaces:

- `gomion workspace init <name> --root <dir>`:
  - Define a new workspace in user config.
  - Optionally create an initial `go.work` at root.

- `gomion workspace use <name>`:
  - Set the current workspace in user config.

- `gomion workspace list` / `status`:
  - Show defined workspaces and which one is current.

### 5.2. Workspace discovery (TUI)

`workspace discover` is inherently interactive and should offer a TUI:

- Command: `gomion workspace discover`.
- Behavior:
  1. Scan configurable roots (e.g. `$HOME/Projects`, `$HOME/src`, etc.) for directories containing both `.git` and `go.mod`.
  2. Cluster repos into tentative workspaces based on directory structure.
  3. Present candidates in a TUI:
     - Create / rename / remove workspaces.
     - Assign repos to workspaces.
     - Mark repos/modules as owned or not.
  4. Persist resulting workspace config to `~/.config/gomion/config.json`.

A non-interactive mode (`--non-interactive`) should be available for CI or scripted usage.

---

## 6. `go.work` and `replace` Orchestration

### 6.1. `go.work` management

Gomion’s responsibilities include:

- Ensuring the workspace root `go.work` is consistent with:
  - User-defined workspace, and
  - Owned modules within that workspace.

Typical command: `gomion go sync-work` (name TBD):

- Ensure all owned modules that should be part of the workspace appear in `use` directives.
- Optionally remove `use` entries for missing/obsolete modules.
- Ensure `go.work`/`go.work.sum` are either:
  - Ignored by git (default), or
  - Tracked intentionally, according to workspace policy.

### 6.2. Dev overrides (names TBD)

Gomion should provide the ability to “toggle dev mode” for a workspace/repo:

- “Dev ON” (placeholder name):
  - Ensure workspace `go.work` is in place and includes the right `use` entries.
  - Add Gomion-managed `replace` directives as needed (e.g., for owned modules outside workspace root).
  - Only operate inside **clearly delimited Gomion blocks**, e.g.:

    ```go
    // gomion:replaces begin
    replace github.com/mikeschinkel/foo => /Users/mike/Projects/ws/foo
    replace github.com/mikeschinkel/bar => /Users/mike/Projects/ws/bar
    // gomion:replaces end
    ```

- “Dev OFF” (placeholder name):
  - Remove or neutralize Gomion’s replace block(s).
  - Optionally leave `go.work` intact but encourage CI to use `GOWORK=off` or not commit `go.work`.

Gomion should also support an **adoption flow** (verb TBD) for projects that already have unmanaged `replace` lines:

- TUI to select which existing `replace` entries to adopt under Gomion management.
- Move chosen entries into Gomion’s `// gomion:replaces` region.

---

## 7. Go Experiments & Inline Directives

### 7.1. Current Go reality

Go experiments:

- Are controlled by the `GOEXPERIMENT` environment variable.
- Are not currently configured in `go.mod`.
- Are intended as toolchain-level knobs rather than module-level API.

However, for certain features (e.g. `jsonv2`), code may **require** particular experiments to be on to behave correctly or even compile.

### 7.2. Gomion’s approach

Gomion should:

- Treat inline comment directives in `go.mod` as canonical, e.g.:

  ```go
  //gomion:goexperiments=arenas,regabiwrappers
  ````markdown
  # Gomion CLI – Requirements & Design Notes

  This document summarizes current requirements, background, and design directions for the **Gomion** CLI.
  It is intended as an internal project artifact to use when refining Gomion’s specification and architecture.

  ---

  ## 1. High-Level Vision

  Gomion is a **Go-centric developer CLI** that:

  - Orchestrates **multi-module, multi-repo Go development**, particularly:
    - `go.work` workspaces
    - `replace` directives in `go.mod`
  - Encodes and enforces the **ClearPath** coding style via a linter.
  - Manages a new class of **“non-dependency dependencies”**, exemplified by [`go-doterr`](https://github.com/mikeschinkel/go-doterr), using a single-file embed model instead of traditional module dependencies.
  - Provides opinionated automation around:
    - Testing, linting, building, fuzzing
    - CI workflows and GoReleaser
    - Scaffolding new Go projects (especially CLIs using `go-cliutil`)
  - Eventually becomes the **central toolbox** for many one-off utilities (e.g., `ssot`, `readme-merge`) so they are not forgotten or re-implemented.

  The project is intended to be **open source** and opinionated, but structured so other developers and organizations can adopt, extend, and contribute.

  ---

  ## 2. Core Problems Gomion Is Addressing

  ### 2.1. Real-world `go.work` + `replace` workflows

  Empirically:

  - `go.work` alone is insufficient in complex, multi-repo setups.
  - `replace` alone is noisy and difficult to manage.
  - In practice, many workflows require both to cooperate.

  Gomion’s core job is to orchestrate:

  - **`go.work`** – defines which local module directories participate in a workspace.
  - **`replace`** – defines how module paths are resolved for a particular `go.mod` (e.g., local fork paths, non-workspace locations).

  Key goals:

  - Treat `go.work` and `replace` as **orthogonal knobs**, not mutually exclusive.
  - Provide **high-level commands** to:
    - “Turn dev overrides on/off” (names TBD) for a workspace or repo.
    - Ensure `go.work` is consistent with the workspace definition.
    - Add/remove Gomion-managed `replace` blocks safely, without touching user-managed ones.
  - Avoid brittle manual editing of `go.mod` and `go.work` for routine tasks.

  ### 2.2. ClearPath coding style

  ClearPath is a personal Go style you want to formalize:

  - Prefer **one return at the end** of a function.
  - Use `goto end` and a shared clean-up block instead of multiple early returns.
  - Keep control flow visually straightforward (“clear path”), with minimal nesting and surprises.

  Requirements:

  - Gomion will eventually provide a **ClearPath linter** as part of `gomion go lint` (or equivalent).
  - The linter will focus on structural rules that support ClearPath:
    - Control flow patterns.
    - Return placement.
    - Label usage (`end:`).
  - This should be optional but first-class, not a hidden add-on.

  ### 2.3. Non-dependency dependencies (e.g., `go-doterr`)

  [`go-doterr`](https://github.com/mikeschinkel/go-doterr) is the prototype for a category of libraries that:

  - Are designed to be **embedded as a single Go file** (e.g., `doterr.go`) into a project or package instead of imported as a traditional dependency.
  - Use only the standard library, and expose a small, conventional interface (e.g., `error`, a `KV` type) to stay interoperable.
  - Avoid:
    - Supply-chain risk from updating third-party modules.
    - Complexity and friction from vendoring entire dependency graphs.

  Conceptually:

  - A **“non-dependency dependency”** is a reusable Go file dropped into many repositories.
  - Gomion’s job is to **manage and update** those embedded copies across a workspace when requested.

  Existing `go-doterr` design work includes:

  - A dedicated CLI PRD (with concepts like `dot`, `copy`, and `ignore` modes per package).
  - TUI for per-package configuration and bulk operations.

  Gomion should **subsume this CLI**:

  - Represent doterr settings under Gomion config (see §4).
  - Provide commands to:
    - Initialize doterr usage in a repo.
    - Apply doterr template(s) (i.e., generate or update `doterr.go`).
    - Check for drift across packages.
    - Run TUIs for interactive configuration.

  `go-doterr` becomes the canonical example of Gomion’s non-dependency dependency support pattern.

  ---

  ## 3. “One Watch” Principle for Configuration

  Guiding principle:

  > “Man with one watch knows the time; man with two watches never sure.”

  For any given concern, there should be a **single canonical source of truth**. Everything else is either:

  - A **derived artifact**, or
  - A **view** that can be regenerated from the canonical source.

  Implications:

  - **Experiments & low-level build toggles**:
    - Canonical config should live in the source tree, e.g. in `go.mod` as a directive comment:
      ```go
      //gomion:goexperiments=arenas,regabiwrappers
      ```
    - Gomion reads this and sets `GOEXPERIMENT` when invoking `go test`, `go build`, fuzzing, etc.
    - `.gomion` should **not** become a second “watch” for the same data.

  - **Doterr usage**:
    - Canonical config is in Gomion’s config (e.g. a `doterr` section) describing package modes: `copy`, `dot`, `ignore`.
    - Generated `doterr.go` files and `go:generate` lines are derived from that.

  - **Workspaces**:
    - Canonical definitions (names, roots, owned module patterns) live in user-level Gomion config (e.g. `~/.config/gomion/config.json`).
    - `go.work` files are the concrete representation used by the Go tool, managed by Gomion according to the workspace definition.

  In short: Gomion must avoid duplicating configuration between `.gomion`, inline comments, and other files. There should be **one authoritative place per concept**.

  ---

  ## 4. Configuration Layers & Ownership

  ### 4.1. Hierarchy

  Conceptual hierarchy Gomion works with:

  - **Workspace** – logical grouping of repos & modules (e.g., XMLUI universe).
  - **Repo** – a git repository.
  - **Module** – a `go.mod` root.
  - **Owned module** – a module that is under your control or explicitly marked as such.

  ### 4.2. Configuration locations

  - **User level** (global):
    - Location: `~/.config/gomion/config.json` (or similar).
    - Canonical info:
      - Workspaces:
        - Name (e.g. `"xmlui"`).
        - Root path.
        - Patterns for “owned” module paths (e.g. `github.com/mikeschinkel/*`, `github.com/xmlui-org/*`).
      - Current workspace.
      - Language configuration (see §9).

  - **Repo / module level**:
    - Location: `.gomion/gommod/gomion.json` (directory-based, consistent with `go-cfgstore`).
    - Contains **Gomion-specific metadata**, for example:
      - Module role(s): `library`, `app`, `test-support`.
      - Release behavior:
        - Whether to use GoReleaser.
        - Which GoReleaser config file to use, etc.
      - Doterr settings (packages and their modes).
      - Future extensions (license policy overrides, project-specific flows, etc.).
    - It should **not** duplicate things that can be derived from `go.mod` (module path, versions) or inline directives.

  ### 4.3. Owned modules

  “Owned” modules must be configurable per:

  - **Workspace** – via path patterns (e.g. `github.com/mikeschinkel/*`).
  - **Repo** – overrides or special cases defined in `.gomion/gommod/gomion.json`.
  - **Module** – per-module entries in `.gomion/gommod/gomion.json` (e.g., test module vs CLI module).

  Resolution precedence:

  1. Module-level config (if present).
  2. Repo-level config.
  3. Workspace patterns.
  4. Default: not owned.

  Owned status influences:

  - Whether Gomion treats a module as part of dev overrides (go.work + replace).
  - Whether it participates in workspace-wide test/lint/release flows.
  - Whether Gomion expects GitHub workflows & GoReleaser setups for that module.

  ---

  ## 5. Workspace Management & Discovery

  ### 5.1. Workspace commands

  Gomion should provide `gomion workspace` subcommands to manage workspaces:

  - `gomion workspace init <name> --root <dir>`:
    - Define a new workspace in user config.
    - Optionally create an initial `go.work` at root.

  - `gomion workspace use <name>`:
    - Set the current workspace in user config.

  - `gomion workspace list` / `status`:
    - Show defined workspaces and which one is current.

  ### 5.2. Workspace discovery (TUI)

  `workspace discover` is inherently interactive and should offer a TUI:

  - Command: `gomion workspace discover`.
  - Behavior:
    1. Scan configurable roots (e.g. `$HOME/Projects`, `$HOME/src`, etc.) for directories containing both `.git` and `go.mod`.
    2. Cluster repos into tentative workspaces based on directory structure.
    3. Present candidates in a TUI:
       - Create / rename / remove workspaces.
       - Assign repos to workspaces.
       - Mark repos/modules as owned or not.
    4. Persist resulting workspace config to `~/.config/gomion/config.json`.

  A non-interactive mode (`--non-interactive`) should be available for CI or scripted usage.

  ---

  ## 6. `go.work` and `replace` Orchestration

  ### 6.1. `go.work` management

  Gomion’s responsibilities include:

  - Ensuring the workspace root `go.work` is consistent with:
    - User-defined workspace, and
    - Owned modules within that workspace.

  Typical command: `gomion go sync-work` (name TBD):

  - Ensure all owned modules that should be part of the workspace appear in `use` directives.
  - Optionally remove `use` entries for missing/obsolete modules.
  - Ensure `go.work`/`go.work.sum` are either:
    - Ignored by git (default), or
    - Tracked intentionally, according to workspace policy.

  ### 6.2. Dev overrides (names TBD)

  Gomion should provide the ability to “toggle dev mode” for a workspace/repo:

  - “Dev ON” (placeholder name):
    - Ensure workspace `go.work` is in place and includes the right `use` entries.
    - Add Gomion-managed `replace` directives as needed (e.g., for owned modules outside workspace root).
    - Only operate inside **clearly delimited Gomion blocks**, e.g.:

      ```go
      // gomion:replaces begin
      replace github.com/mikeschinkel/foo => /Users/mike/Projects/ws/foo
      replace github.com/mikeschinkel/bar => /Users/mike/Projects/ws/bar
      // gomion:replaces end
      ```

  - “Dev OFF” (placeholder name):
    - Remove or neutralize Gomion’s replace block(s).
    - Optionally leave `go.work` intact but encourage CI to use `GOWORK=off` or not commit `go.work`.

  Gomion should also support an **adoption flow** (verb TBD) for projects that already have unmanaged `replace` lines:

  - TUI to select which existing `replace` entries to adopt under Gomion management.
  - Move chosen entries into Gomion’s `// gomion:replaces` region.

  ---

  ## 7. Go Experiments & Inline Directives

  ### 7.1. Current Go reality

  Go experiments:

  - Are controlled by the `GOEXPERIMENT` environment variable.
  - Are not currently configured in `go.mod`.
  - Are intended as toolchain-level knobs rather than module-level API.

  However, for certain features (e.g. `jsonv2`), code may **require** particular experiments to be on to behave correctly or even compile.

  ### 7.2. Gomion’s approach

  Gomion should:

  - Treat inline comment directives in `go.mod` as canonical, e.g.:

    ```go
    //gomion:goexperiments=arenas,regabiwrappers
    ```

  * Parse these from `go.mod` (or other agreed-upon canonical place).
  * When running Go commands (`gomion go test`, `gomion go build`, `gomion go fuzz`, etc.), Gomion should:

    * Combine experiments from:

      * Workspace defaults (if any),
      * Repo/module directives.
    * Set `GOEXPERIMENT` accordingly when invoking `go`.

  Possible distinction:

  * Some experiments may be **“must-have”** (e.g. required for `jsonv2`), others “nice-to-have”.
  * Gomion could allow marking certain experiments as required vs optional, but that is a future refinement; for now the key is **canonical config via inline directives**, not `.gomion`.

  ---

  ## 8. Testing, Linting, Fuzzing & ClearPath

  ### 8.1. `gomion go test`

  High-level behavior:

  * Discover owned modules in the current workspace / repo.
  * For each module (or a subset as configured):

    * Run `go test ./...` with appropriate environment:

      * `GOEXPERIMENT` from directives.
      * Any other Gomion-managed flags.

  Options and future extensions:

  * Limiting to specific modules.
  * Integrating coverage reporting.
  * Aggregating results across modules.

  ### 8.2. `gomion go lint`

  Responsibilities:

  * Wrap existing linters (e.g., `golangci-lint`) with Gomion’s conventions.
  * Eventually include the **ClearPath linter** as a component:

    * Enforce single-return + `goto end` pattern where applicable.
    * Check for other ClearPath rules (to be defined in detail separately).

  Policy tools:

  * Gomion should be aware of tools like `gomodguard` (for dependency allow/block lists), and long term:

    * Integrate as needed for organizations.
    * But for now this is informational / future expansion.

  License-based dependency governance:

  * There’s prior work on tools that govern dependencies by license (e.g. allow Apache-2.0, disallow GPL).
  * Gomion may eventually include a dependency policy layer that:

    * Reads dependency metadata (including license).
    * Enforces allow/deny lists per workspace.

  ### 8.3. `gomion go fuzz` – infinite fuzz testing

  Gomion should absorb the functionality of the existing `infinite-fuzz-go` shell script:

  * Discover fuzz targets from `*_test.go` (functions `FuzzXxx`).
  * Run `go test -run=^$ -fuzz=^FuzzName$` in a loop per target.
  * Support:

    * Per-target selection (`--target`).
    * All targets by default.
    * Concurrency control (how many fuzzers at once).
    * Clean shutdown on signals (SIGINT, etc.).
  * Respect Go experiments when fuzzing.

  ---

  ## 9. Multi-Language Future & “Current Language”

  While Go is the primary focus, the design should anticipate adding other languages (e.g., Zig).

  Config concept:

  * A **current language**:

    ```jsonc
    {
      "currentLanguage": "go",
      "languages": {
        "go":  { "enabled": true },
        "zig": { "enabled": false }
      }
    }
    ```

  * `gomion test`, `gomion build`, etc. default to `currentLanguage`, but support:

    * `--lang=<name>`, e.g. `--lang=zig`.
    * Possibly convenience flags (`--zig`).

  Implementation:

  * Internally, Gomion can eventually have a **language backend registry** with a shared interface:

    * `Test`, `Build`, `Lint`, etc.
  * Initially, only the **Go backend** needs to exist; other backends are future work.

  ---

  ## 10. GoReleaser & CI Integration

  ### 10.1. Release model

  General approach:

  * Tagging and releasing is done by **GitHub Actions**, not directly by Gomion.
  * Gomion provides scaffolding and validation.

  Requirements:

  * A release workflow (`release.yml`) that:

    * Runs tests, lint, vet (potentially via `gomion go test` / `gomion go lint`).
    * Only proceeds to tagging / GoReleaser on success.
  * GoReleaser for binaries:

    * CLI modules (`role: app`) should have GoReleaser config generated/managed by Gomion.

  ### 10.2. TUI for GoReleaser scaffolding

  GoReleaser setup is interactive by nature and should have a TUI:

  * Command: e.g. `gomion go release init`.
  * TUI tasks:

    * Confirm which module(s) are CLI apps.
    * Choose OS/arch targets.
    * Confirm GitHub owner/repo and project name.
    * Decide how releases are triggered (tag push, manual, etc.).
  * Outputs:

    * `.goreleaser.yaml` (or variant).
    * `.github/workflows/release.yml`.

  Additionally, Gomion can later support:

  * Local runs: `gomion go release --local` for dry runs before pushing tags.

  ### 10.3. GitHub integration (without `gh` CLI)

  Gomion should talk directly to GitHub via API (e.g. `google/go-github`), not rely on the `gh` CLI:

  * For reading / verifying repo metadata.
  * Potentially for validating tags/releases.
  * Authentication via token (env var, config, or keychain).

  ---

  ## 11. TUIs in Gomion

  Certain tasks are inherently interactive and should offer TUIs:

  * **Workspace discovery** (see §5.2).
  * **Doterr (and similar embed tool) configuration**:

    * Package list with mode toggles (`copy`, `dot`, `ignore`).
  * **GoReleaser scaffolding** (see §10.2).
  * Potentially:

    * Adoption of existing `replace` directives.
    * High-level “workspace status” views.

  TUIs should:

  * Be optional: non-interactive flags exist for CI or automation.
  * Use a robust terminal UI framework (choice of library is an implementation detail, not locked in here).

  ---

  ## 12. Templates & Scaffolding

  For Go development, Gomion should:

  * Help scaffold **CLI projects** using `go-cliutil` instead of Cobra/urfave-cli:

    * Command like `gomion go new-cli <name>`:

      * Set up `cmd/<name>/main.go` with your `RunCLI()` pattern.
      * Configure `go.mod` as needed.
      * Optionally create `.gomion/gommod/gomion.json` with:

        * Module role `app`.
        * GoReleaser usage settings.

  * Support project templates:

    * The templates themselves can include imports of desired packages.
    * Gomion doesn’t need a separate “initial packages list” if templates serve that role.
    * Avoid duplicating configuration: templates are canonical for initial structure.

  ---

  ## 13. Integration with Existing Tools & Libraries

  Gomion is meant to **compose** with, not replace:

  * `go` toolchain and standard commands.
  * **GoReleaser** for binary releases.
  * **goyek** as an optional task automation framework:

    * Gomion can provide Go APIs that goyek flows call.
    * Projects can embed Gomion logic directly in goyek tasks.
  * **go-doterr** as a single-file embeddable error handling pattern.
  * Existing one-off utilities such as:

    * `ssot` (Single Source of Truth manager).
    * `readme-merge` (merged READMEs from multiple sources).
    * These tools can be folded into Gomion subcommands over time.

  Implementation preference:

  * Gomion should be both:

    * A **CLI app** (`cmd/gomion`).
    * A **library** (packages under `github.com/mikeschinkel/gomion/pkg/...`) reusable by goyek flows or other tools.
  * No need for a long-running daemon or gRPC initially; a single process per invocation is sufficient.

  ---

  ## 14. Future / Open Items

  A non-exhaustive list of items that remain open or are explicitly deferred:

  1. **Final verbs** for dev overrides and adoption:

     * Placeholders like `apply` / `unapply`, `dev-on` / `dev-off`, `adopt-replaces` are temporary.
     * Final naming should prioritize clarity and idiomatic feel.

  2. **Exact ClearPath rules**:

     * The ClearPath linter’s detailed rule set still needs to be documented and implemented.

  3. **Exact schema for `.gomion/gommod/gomion.json`**:

     * You intend to design JSON schemas by hand; auto-generated schemas are usually not aligned with your preferences.
     * Gomion’s design assumes such a file exists but leaves the schema details to be finalized separately.

  4. **License-based dependency governance**:

     * There is prior work for license-whitelist / blacklist policies (e.g. allow Apache-2.0, disallow GPL).
     * Gomion may integrate this as a future module.

  5. **Multi-language backends**:

     * Go backend is the only one required initially.
     * Zig or others may be added later once there’s a concrete need.

  6. **Depth of integration with tools like gomodguard**:

     * Currently noted as useful for organizations.
     * Gomion may later generate or manage gomodguard configuration for teams that want strict dependency policies.

  ---

  This document should serve as a reference point when refining Gomion’s requirements in other discussions or documents. It captures the current intent and direction: Gomion as a Go-centric orchestration CLI that manages workspaces, ClearPath style, non-dependency dependencies like `go-doterr`, experiments, CI/release, and more, while staying faithful to the “one watch” configuration principle.

  ```
  ::contentReference[oaicite:0]{index=0}
  ```
