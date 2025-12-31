# Gomion CLI – v0 Product Requirements Document (PRD)

## 1. Purpose and Scope (v0)

This PRD defines the **v0 scope** for the Gomion CLI:

- Provide a **repeatable, automatable way to discover Go modules** under one or more filesystem roots.
- Generate **repo-local configuration** in `.gomion/config.json` that records which modules exist in each repo and what high-level role each plays.
- Support **batch adoption** of existing repos without requiring an interactive TUI.

v0 is intentionally narrow and foundational. It does **not** include release orchestration, dependency management, drop-ins (e.g., doterr), or TUIs. Those are captured in the Vision document and will be covered in future PRDs.


## 2. Terminology

- **Repo** – A Git repository (identified by a `.git` directory at or above a path).
- **Module** – A Go module, identified by a `go.mod` file inside a repo.
- **Repo root** – The directory containing the `.git` directory for a repo.
- **Root path / root** – A filesystem directory under which Gomion should search for repos and modules.
- **Managed repo** – A repo that has a `.gomion/config.json` file at its repo root.
- **Unmanaged repo** – A repo with Go modules but no `.gomion/config.json`.


## 3. Goals (v0)

1. **Discover Go modules across the filesystem** under user-specified roots, grouped by repo.
2. **Identify unmanaged repos** (those without `.gomion/config.json`) and expose their module `go.mod` paths.
3. **Generate minimal repo config** in `.gomion/config.json` that records:
   - Per-module **relative path** from repo root.
   - A short, unique **name** for each module in the repo.
   - A coarse **role** for each module (e.g., `lib`, `cli`, `test`).
4. Support **batch onboarding** of existing repos via a file-based workflow:
   - `gomion scan` → text file of candidate modules.
   - User edits file to choose which modules/repos to adopt.
   - `gomion init --file <path>` → writes `.gomion/config.json` for those repos.
5. Ensure that all config needed by CI/CD for v0 (module awareness and roles) is **repo-local**, not machine-local.


## 4. Non-goals (v0)

The following are explicitly **out of scope** for v0 and will be addressed in future PRDs:

- Release planning, tagging, and changelog management.
- Dependency policy (e.g., “latest” vs pinned versions), or manipulation of `go.mod`/`go.work` contents.
- Drop-in / "non-dependency dependency" management (e.g., doterr embed files).
- Workspace naming or cross-repo orchestration beyond basic scanning.
- TUIs for adoption or management (batch adoption is file-based in v0).
- Any modification of user code or `go.mod` other than creating `.gomion/config.json`.


## 5. Assumptions and Constraints

- Gomion v0 is **Go-centric**: modules are identified via `go.mod` files.
- The user may have **many repos** and wants a low-friction way to onboard them without touching each one manually.
- v0 must be safe: no modification of git history, no touching of `go.mod` contents, no code edits.
- v0 should avoid long-running operations: scanning typical development trees under a few top-level `--root` paths should complete within a few seconds. If scan time grows toward 60+ seconds, the design should be revisited.
- Config that affects CI/CD behavior **must live in the repo**, under `.gomion/config.json`. Machine-level state is purely ergonomic/caching and not required by CI.


## 6. Repo Config Format (v0)

### 6.1 File location

- Repo-local config lives at:

  - `<repo-root>/.gomion/config.json`

- The `.gomion` directory is created by Gomion as needed.

### 6.2 Minimal schema for v0

For v0, the **only required field** is a `modules` map. `groups`, `dependencies`, `dropins`, etc. are explicitly deferred.

Example:

```json
{
  "modules": {
    "./apppkg": {
      "name": "apppkg",
      "role": ["lib"]
    },
    "./cmd": {
      "name": "gomion-cli",
      "role": ["cli"]
    },
    "./test": {
      "name": "test",
      "role": ["test"]
    }
  }
}
```

#### 6.2.1 `modules` map

- **Key**: repo-relative path to the module directory, using `./` notation:
  - Root module → `"./"` or a specific path like `"./apppkg"` (project convention).
  - Submodules under `cmd/` → e.g., `"./cmd"` or `"./cmd/gomion-cli"` (depending on layout).
  - Submodules under `test/` → e.g., `"./test"`.
- **Value**: module metadata object with fields:

  - `name` (string; required)
    - Short, human-friendly name for the module.
    - Must be **unique per repo** for modules Gomion is expected to manage.
    - Used later by other features (e.g., grouping, referencing modules in UI).

  - `role` (array of strings; required, non-empty)
    - Coarse classification of the module’s role, e.g.:
      - `"lib"` – library / package module.
      - `"cli"` – command-line executable module.
      - `"test"` – test-only or harness module.
    - A module may have **multiple roles** (e.g., `"cli"` + `"daemon"` in future versions), but v0 only needs `lib`, `cli`, `test`.

- **Validation rules (v0):**
  - If `.gomion/config.json` exists, Gomion v0 should:
    - Validate JSON syntax.
    - Validate `modules` is an object.
    - Validate each module’s `name` is a string and `role` is a non-empty array of strings.
    - Ensure `name` values are unique per repo for Gomion-managed modules.
  - v0 may treat unrecognized fields as opaque and ignore them (to ease future schema evolution).


## 7. CLI Commands (v0)

v0 defines two core commands: `gomion scan` and `gomion init`.

### 7.1 `gomion scan`

**Purpose:** Discover Go modules under one or more roots, restricted to repos that do **not** yet have `.gomion/config.json`. Output the full filesystem path to each `go.mod` on its own line.

**Usage (v0):**

```bash
gomion scan --root <dir> [--root <dir> ...]
```

**Behavior:**

1. For each `--root` argument:
   - Expand `~` to the user’s home directory.
   - Resolve to an absolute path.
   - Recursively walk the directory tree.
2. When a `go.mod` is found:
   - Walk up the directory tree until a `.git` directory is found; that directory is the **repo root**.
   - Check whether `<repo-root>/.gomion/config.json` exists.
   - If `.gomion/config.json` **exists**, skip this `go.mod` (repo is already managed).
   - If it does **not** exist, include this `go.mod` in the scan results.
3. Output each included `go.mod` path on its own line to stdout.
   - Paths should be **absolute** by default.
   - `~` may optionally be accepted in input roots but should not appear in output.

**Options (v0):**

- `--root <dir>` (repeatable; required at least once)
  - One or more root directories to scan under.
- Future options (to be defined later, not part of v0 behavior):
  - `--all` – include modules from managed repos as well.
  - `--format json` – structured output.

**Examples:**

```bash
# Scan two roots and write candidate go.mod paths to a file
gomion scan \
  --root ~/Projects/xmlui \
  --root ~/Projects/go-pkgs \
  > scan.txt
```


### 7.2 `gomion init`

**Purpose:** Create `.gomion/config.json` files for repos that do not already have one, based on a curated list of `go.mod` paths.

v0 focuses on a **batch adoption mode** driven by a file. A single-repo `gomion init` (run from within a repo) may be added later but is not required for v0.

**Usage (v0):**

```bash
gomion init --file <path>
```

**Input file format:**

- Plain text file.
- Each non-empty, non-comment line is a path to a `go.mod` file.
  - Lines starting with `#` are treated as comments and ignored.
  - Paths may be absolute or start with `~`.

Example:

```text
# cfgstore repo (all modules)
~/Projects/go-pkgs/go-cfgstore/examples/basic_usage/go.mod
~/Projects/go-pkgs/go-cfgstore/go.mod
~/Projects/go-pkgs/go-cfgstore/test/go.mod

# xmlui servers
~/Projects/xmlui/localsvr/xmluisvr/go.mod
~/Projects/xmlui/mcpsvr/xmluimcp/go.mod
```

**Behavior:**

1. Read the `--file` line by line.
   - Ignore empty lines and comments.
   - Expand `~` to `$HOME`.
   - Normalize each path to an absolute path.
2. For each `go.mod` path:
   - Validate the file exists; if not, record an error.
   - Walk up to find the nearest `.git` directory; its directory is the **repo root**.
3. Group all `go.mod` paths by repo root.
4. For each repo root group:
   - If `<repo-root>/.gomion/config.json` already exists:
     - v0 default behavior: **skip** and emit a warning.
     - A future flag (e.g., `--update` or `--force`) may alter this behavior; out of scope for v0.
   - Otherwise:
     - Compute each module’s **repo-relative path**, using `./` notation.
     - Derive module `name` values:
       - Root module: by default, use either the repo basename or the last path element (exact rule TBD but consistent per implementation).
       - Other modules: use the last path segment of the module directory by default.
     - Infer `role` values using simple heuristics:
       - Paths under `cmd/` → `"cli"`.
       - Paths under `test/` → `"test"`.
       - All others → `"lib"`.
     - Construct a `modules` map and write it to `<repo-root>/.gomion/config.json`.
       - Create the `.gomion` directory if it does not exist.
5. Report a summary:
   - Number of repos initialized.
   - Number of repos skipped due to existing `.gomion/config.json`.
   - Any errors encountered (e.g., missing `go.mod` paths in the file).

**Error handling:**

- If the `--file` does not exist or is unreadable, exit non-zero with a clear message.
- If a listed `go.mod` path does not exist, report the line and path, continue processing others, and exit non-zero.
- If no valid `go.mod` paths remain after filtering, exit non-zero with a message.
- If a repo already has `.gomion/config.json`, log a warning indicating it was skipped.

**Examples:**

```bash
# Typical adoption flow
gomion scan \
  --root ~/Projects/xmlui \
  --root ~/Projects/go-pkgs \
  > scan.txt

# Manually edit scan.txt → init.txt to keep only the repos you want to adopt now
$EDITOR init.txt

# Initialize .gomion/config.json for those repos
gomion init --file init.txt
```


## 8. User Stories (v0)

1. **As a developer with many existing Go repos**, I want to run a single command to discover all repos/modules that are not yet Gomion-managed so I can decide which to onboard.

2. **As a developer**, I want to edit a simple text file listing module `go.mod` paths and then have Gomion create `.gomion/config.json` files for all selected repos in one batch operation.

3. **As a developer**, I want Gomion to infer reasonable defaults for module names and roles, so I do not need to hand-author JSON for each repo.

4. **As a CI maintainer**, I want `.gomion/config.json` to live inside each repo and fully describe its modules/roles for v0 features, so CI jobs can run Gomion without any dependence on machine-local configuration.


## 9. Open Questions (to resolve in later PRDs)

These are deliberately **not** required for v0 but should be tracked:

- Should Gomion allow a single-repo `gomion init` (no `--file`) that infers modules from the current directory’s repo?
- Should scan support additional filters (e.g., ignore repos based on a user-local ignore list in `~/.config/gomion/`)?
- Exact rules and options for naming modules (`name` field) when there are multiple modules under `cmd/`, `test/`, etc.
- How to handle modules discovered in a repo that are **not** represented in `modules{}` (e.g., warn vs strict error modes) in later features.
- Whether to allow `.gomion/config.json` to be created in non-Go repos for future multi-language support.


---

