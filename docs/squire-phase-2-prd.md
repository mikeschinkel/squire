# Gomion CLI – Phase 2 PRD

**Phase:** 2 – Module Discovery & Dependency Ordering  
**Status:** Draft  
**Owner:** Mike / Gomion CLI  
**Predecessor:** v0 PRD (scan + init + config)  
**Date:** 2025-12-07

---
## 0. Non-negotiable requirements

Before starting you MOST read (or have read) <repo_root>/CLAUDE.md and then you must follow all instructions in _"CRITICAL INSTRUCTIONS"_ section.

## 1. Purpose & Scope

Phase 2 adds **module discovery** and **dependency-safe ordering** across the Gomion universe.

The goal is to give Gomion an internal, well-defined way to:

1. Discover all Go modules that Gomion cares about, using:
   - existing `.gomion/config.json` (from Phase 1 / v0), and
   - each module's `go.mod`.
2. Build an in-memory view of:
   - what modules exist,
   - what kind of modules they are (lib / cmd / test),
   - which other Gomion-managed modules they depend on.
3. Produce a **dependency-safe ordering** of those modules so that:
   - if module B depends on module A, A will always appear **before** B in the ordered list.

Phase 2 **does not** implement tagging, releases, changelogs, universes, or cross-repo orchestration. It only supplies the core module + dependency model and ordering that future phases will build on.

All public types and functions described here live in the existing **`gompkg`** package (no new top-level package, no `/internal`).

---

## 2. Background

Phase 1/v0 established:

- `gomion scan` to discover Go modules and `gomion init` to generate `.gomion/config.json` per repo.
- `.gomion/config.json` records per-module metadata such as:
  - relative directory (`./`, `./cmd`, `./test`, etc.),
  - human-readable name,
  - roles (e.g. `lib`, `cli`, `test`).
- An ephemeral file such as `go.mod.files.txt` **may** have been used at init-time but is not part of the long-term API or data model.

Gomion now needs to understand **how modules depend on each other** in order to:

- reason about **safe commit / release order** for multi-repo setups,
- support future features like release planning, drop-in updates, and multi-repo orchestration.

This PRD focuses on defining **in-memory types and behavior** for module discovery and ordering, without yet exposing user-facing release commands.

---

## 3. Goals

1. **Canonical module model**  
   Define a `Module` type in `gompkg` that represents a Gomion-managed Go module and captures:
   - location,
   - kind (lib / cmd / test),
   - whether it is considered versioned,
   - its dependencies on other Gomion-managed modules.

2. **Module aggregation**  
   Define a `ModuleSet` type in `gompkg` that aggregates a collection of `Module` values and provides:
   - discovery from a root directory,
   - accessors and ordering operations.

3. **Dependency-safe ordering**  
   Implement a method that returns modules in an order where:
   - for any `Module` M, all modules in `M.Dependencies` appear **earlier** in the returned list.

4. **Heuristic versioned/private classification**  
   Implement a simple heuristic for `Module.Versioned`:
   - test modules are **not** versioned,
   - all other modules are considered versioned (for now).

5. **No config schema changes required**  
   Phase 2 must work with existing `.gomion/config.json` files created by v0. It may **derive** additional information (e.g. `ModulePath`) from `go.mod`, but does not persist it.

6. **JSON-only, no TOML/YAML**  
   All persistent config remains JSON (`.gomion/config.json`) using only Go's standard library. TOML/YAML may be considered later under build tags, but are out of scope here.

---

## 4. Non-Goals

The following are explicitly **out of scope** for Phase 2:

1. **Tagging or releasing**  
   No `git tag`, `git push`, or release-creation behavior. The output of Phase 2 is **read-only** information about modules and their ordering.

2. **Changelog or annotation system**  
   No `.gomion/changelog/` or release-note modeling in this phase.

3. **Universe / multi-root orchestration UX**  
   No `gomion each` or user-facing universe commands. Phase 2 only needs enough to work from a given root directory and the repos Gomion already considers "attached".

4. **Config schema extensions**  
   No new required fields in `.gomion/config.json`. Future fields such as `"versioned": false` may be added later as overrides, but are not required for Phase 2.

5. **Public release of dependency-ordering CLI**  
   A debug CLI command (e.g. `gomion deps order`) is optional. The primary deliverable is the **library API** in `gompkg` plus tests.

6. **Using `internal` packages**  
   Do not introduce new `/internal` packages in Phase 2. The API lives in `gompkg` and can later be refactored if needed.

---

## 5. Terminology

- **Module** – a Go module with its own `go.mod` file that Gomion knows about.
- **ModuleKind** – classification of a module for Gomion's purposes:
  - `LibModuleKind` – general library module.
  - `CmdModuleKind` – command module (typically under `cmd/` with a `main` package).
  - `TestModuleKind` – test/support module (typically under `test/` or `tests/`).
- **Versioned module** – a module that Gomion considers as a potential **tagged** unit for future releases.
- **Non-versioned ("private") module** – a module that Gomion generally does not treat as an independently versioned artifact (e.g. tests).
- **ModuleSet** – an in-memory collection of `Module` values plus indexes for dependency reasoning.

---

## 6. Data Model

All types below live in `gompkg`.

### 6.1 ModuleKind

```go
// ModuleKind classifies the role of a module in Gomion's universe.
type ModuleKind int

const (
    UnknownModuleKind ModuleKind = iota
    LibModuleKind
    CmdModuleKind
    TestModuleKind
)
```

- `LibModuleKind` – normal library modules (default when not obviously `cmd` or `test`).
- `CmdModuleKind` – modules rooted under a `cmd/` directory that contain a `main` package.
- `TestModuleKind` – modules under `test/` or `tests/` directories.

(Exact detection rules are defined in §7.)

### 6.2 Module

```go
// Module represents a single Go module that Gomion knows about.
// It includes its location, semantic kind, and dependencies on other modules.
type Module struct {
    // RepoRoot is the filesystem root for the repo containing this module.
    // (Absolute or canonicalized path; details TBD but must be consistent
    // with how Phase 1 identifies a "root repo".)
    RepoRoot string

    // RelDir is the module's path relative to RepoRoot, e.g. "./", "./cmd", "./test".
    RelDir string

    // ModulePath is the Go module path from this module's go.mod "module" directive,
    // e.g. "github.com/mikeschinkel/go-dt".
    ModulePath string

    // Kind classifies the module as lib/cmd/test.
    Kind ModuleKind

    // Versioned indicates whether this module is considered a versionable unit.
    // Phase 2 computes this via heuristics (see §7); no config override yet.
    Versioned bool

    // Requires lists the Go module paths of Gomion-managed modules that
    // this module depends on (from its go.mod "require" directives).
    //
    // Only requires that are themselves known Gomion-managed modules are included.
    Requires []string
}
```

### 6.3 ModuleSet

```go
// ModuleSet represents a collection of Modules Gomion knows about,
// along with internal indexes for dependency reasoning.
type ModuleSet struct {
    Modules []*Module

    // unexported indexes
    byPath map[string]*Module
}
```

- `Modules` contains all discovered modules.
- `byPath` provides fast lookup by `ModulePath` for building dependency relationships and ordering.

`ModuleSet` is conceptually "the set of modules we are reasoning over" – typically all modules in the repos attached to a given root directory.

---

## 7. Behavior & Algorithms

### 7.1 Module discovery – `DiscoverModules`

**Signature (initial):**

```go
// DiscoverModules discovers Gomion-managed modules starting at rootDir.
//
// rootDir is typically a path inside a "root repo" that Gomion knows how
// to locate based on existing behavior (Phase 1 / v0). DiscoverModules
// uses .gomion/config.json plus each module's go.mod to build a ModuleSet.
func DiscoverModules(rootDir string) (*ModuleSet, error)
```

**Inputs:**

- `rootDir` – any directory under a Gomion-managed repo.
- `.gomion/config.json` – per-repo configuration created by `gomion init`.
- `go.mod` – per-module go.mod files referenced by the config.

**Required behavior:**

1. Locate the **root repo** containing `rootDir` using the same logic v0 uses (e.g. search up for `.git` and `.gomion`).

2. Read that repo's `.gomion/config.json`.

3. For each module entry in config:
   - Determine its module directory based on `RelDir`.
   - Read its `go.mod` and parse the `module` directive to obtain `ModulePath`.
   - Determine `Kind`:
     - If module is under a `test/` or `tests/` directory → `TestModuleKind`.
     - Else if module is under a `cmd/` directory (e.g. `./cmd`, `./cmd/foo`) and contains a `main` package → `CmdModuleKind`.
     - Else → `LibModuleKind`.
   - Determine `Versioned`:
     - `Kind == TestModuleKind` → `Versioned = false`.
     - All other kinds → `Versioned = true`.

4. Collect the modules into a `ModuleSet`:
   - Append each `Module` to `ModuleSet.Modules`.
   - Populate `byPath[module.ModulePath] = &module`.

5. Build the `Requires` field for each module:
   - For each module, read its `go.mod` again (or reuse the parsed representation) and inspect `require` directives.
   - For each `require` module path `P`:
     - If `P` exists in `byPath`, append `P` to `module.Requires`.
   - Ignore requires not known/managed by Gomion.

6. Return a populated `*ModuleSet`.

**Notes:**

- `go.mod.files.txt` is **not** used. It was an ephemeral helper for `gomion init` and must not be required for Phase 2.
- No new fields are written back to `.gomion/config.json`.
- Any I/O errors (missing `go.mod`, malformed config, etc.) should produce descriptive errors.

### 7.2 Dependency-safe ordering – `OrderModules`

**Signature:**

```go
// OrderModules returns all modules in the set in a dependency-safe order.
//
// For any module M in the returned slice, all modules whose ModulePath is in
// M.Dependencies appear earlier in the slice.
func (ms *ModuleSet) OrderModules() ([]*Module, error)
```

**Behavior:**

- Implements a dependency-first ordering over the modules in `ModuleSet`.
- If module B lists module A in its `Dependencies`, A must appear before B in the returned slice.
- All modules in `ms.Modules` must appear exactly once in the returned slice.
- If a dependency cycle is detected among Gomion-managed modules, return an error.

**Algorithm sketch (informal):**

1. Build a working map of remaining modules keyed by `ModulePath`.
2. Build a count of how many unresolved dependencies each module has.
3. Repeatedly:
   - Find modules with zero unresolved dependencies.
   - Append them to the output list and remove them from the working map.
   - For each removed module, decrement the dependency count of modules that depended on it.
4. If at any point there are remaining modules but none have zero unresolved dependencies, a cycle exists → return an error.

(Implementation may use any standard topological-sort approach; the behavior above is the contract.)

**Versioned vs non-versioned:**

- `OrderModules` should include **all** modules in the ordering, regardless of `Versioned` status.
- Callers that only care about versioned modules can filter by `Module.Versioned` after ordering.

---

## 8. CLI (Optional for Phase 2)

An optional debug-only CLI command can be added to help validate implementation and aid manual workflows:

```bash
gomion deps order [<dir>]
```

**Semantics:**

- `dir` defaults to `.` if omitted.
- Internally calls `DiscoverModules(dir)` and `OrderModules`.
- Prints an ordered list of modules, including:
  - Repo root,
  - RelDir,
  - ModulePath,
  - Kind,
  - Versioned flag.

Example output (illustrative):

```text
1. Repo: /Users/mike/Projects/go-doterr
   RelDir: ./
   ModulePath: github.com/mikeschinkel/go-doterr
   Kind: lib
   Versioned: true

2. Repo: /Users/mike/Projects/go-dt
   RelDir: ./
   ModulePath: github.com/mikeschinkel/go-dt
   Kind: lib
   Versioned: true

...

N. Repo: /Users/mike/Projects/cli
   RelDir: ./cmd
   ModulePath: github.com/xmlui/cli/cmd/xmluicli
   Kind: cmd
   Versioned: true
```

This command is primarily for:

- manual validation during Phase 2 implementation, and
- ad hoc debugging by the author.

It does **not** need to be positioned as a polished end-user feature in this phase.

---

## 9. Testing Strategy

Phase 2 should be covered by unit tests and small fixture repositories.

### 9.1 Fixture layout

Create one or more test fixtures representing a small multi-repo Gomion universe, e.g.:

- `testdata/universe/` with subdirectories:
  - `go-doterr/`
  - `go-dt/`
  - `go-logutil/`
  - `cli/`

Each fixture repo should contain:

- A minimal `.git` placeholder (if required by root-detection logic).
- `.gomion/config.json` consistent with Phase 1 behavior.
- One or more `go.mod` files with plausible dependencies among them.

### 9.2 Tests

1. **Discovery correctness**
   - Given a `rootDir` under a fixture repo, `DiscoverModules(rootDir)` should:
     - discover the expected number of modules,
     - correctly set `RepoRoot`, `RelDir`, `ModulePath`, and `Kind`,
     - correctly set `Versioned` based on heuristics,
     - include only internal Gomion modules in `Dependencies`.

2. **Ordering correctness**
   - Given the `ModuleSet` from a fixture universe, `OrderModules` should:
     - return all modules exactly once,
     - ensure that for every module, each `Dependencies[i]` appears earlier in the slice.

3. **Cycle detection**
   - Construct a synthetic fixture or in-memory ModuleSet with a deliberate cycle and verify that `OrderModules` returns an error.

4. **Versioned filtering**
   - Verify that filtering `OrderModules` output by `Module.Versioned` produces the expected subset (e.g. test modules excluded).

---

## 10. Extensibility & Future Work

Phase 2 is intentionally minimal but is designed to support future phases:

1. **Config overrides for Versioned**  
   Later, `.gomion/config.json` can grow an optional `versioned` boolean per module. The logic for `Versioned` would become:
   - compute heuristic (test → false, others → true), then
   - if `versioned` is present in config, override the heuristic.

2. **Repo-level release targets**  
   Future phases can introduce a `ReleaseTarget` concept that groups modules by `RepoRoot` using the ordered `[]*Module` as input. That layer can make repo-level decisions without changing `Module` or `ModuleSet`.

3. **Release planning and application**  
   Phase 3+ can use `OrderModules` to:
   - propose release order for versioned modules,
   - generate release manifests,
   - eventually drive tagging and CI/CD integrations.

4. **Universe-level operations**  
   Once universes are formally modeled, `DiscoverModules` may grow a variant that spans multiple root repos, but the basic `Module` and `ModuleSet` types should remain valid.

5. **ClearPath & lint integration**  
   Future ClearPath linters or style tools may reuse the same module discovery logic, keeping module understanding centralized in `gompkg`.

---

## 11. Acceptance Criteria

Phase 2 is complete when:

1. `gompkg` exposes `ModuleKind`, `Module`, `ModuleSet`, `DiscoverModules`, and `OrderModules` as described.
2. The implementation:
   - correctly discovers modules using existing `.gomion/config.json` and `go.mod` files,
   - classifies modules into lib/cmd/test and sets `Versioned` via the agreed heuristic,
   - computes `Dependencies` based only on Gomion-managed modules.
3. `OrderModules` returns a valid dependency-safe order in realistic fixture universes and detects cycles.
4. Tests cover the behaviors listed in §9 and pass reliably.
5. No changes are required to existing `.gomion/config.json` files created by Phase 1.
6. (Optional) `gomion deps order` exists as a debug command and produces intelligible output for a multi-repo fixture universe.

