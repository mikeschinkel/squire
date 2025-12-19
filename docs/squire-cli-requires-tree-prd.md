# Squire CLI – Requires Tree PRD

**Feature:** Requires Tree (dependency tree visualization + markdown embedding)

**Status:** Draft  
**Owner:** Mike / Squire CLI  
**Related phases:**
- Phase 2 – Module Discovery & Dependency Ordering (uses `squirepkg.Module`, `ModuleSet`, `DiscoverModules`, `OrderModules`)

**Date:** 2025-12-08

---

## 1. Purpose & Scope

Add a **tree-style dependency visualization** rooted at the current repo, plus the ability to embed that tree into a markdown document.

At a high level, this feature provides:

1. A **text tree** similar in spirit to the `tree` bash command, but for **Go module dependencies** instead of directories.
2. The tree is rooted at the **current repo’s modules** and branches out through the modules they depend on.
3. A `--all` flag to expand the tree to include **all dependent modules**, not just those that Squire manages.
4. An `--embed=<markdown_file>` mode that inserts the rendered tree into a markdown file near a special marker:
   - `<!-- squire:embed-requires-tree -->`
   - Placement controlled by `--before` or `--after` (mutually exclusive, required when `--embed` is used).

The tree is primarily a **read-only visualization** and documentation aid; it does not perform tagging or release actions.

---

## 2. Goals

1. **Visualize dependency relationships**
   - Provide a CLI command that prints a tree showing how the current repo’s modules depend on other modules.
   - Use ASCII-art style (or similar), suitable for code-block embedding in markdown.

2. **Reuse/extend existing discovery logic**
   - Leverage the Phase 2 module model (`Module`, `ModuleSet`, `DiscoverModules`) for Squire-managed modules.
   - Add the ability to optionally include non-Squire modules when `--all` is specified.

3. **Markdown embedding workflow**
   - Allow users to embed the dependency tree into documentation files.
   - Support a stable marker (`<!-- squire:embed-requires-tree -->`) and explicit placement (`--before` or `--after`).
   - Embed as a fenced code block: ` ```text` + tree + ` ``` `.

4. **On-the-fly operation**
   - Compute the tree from the current state of `go.mod` files and Squire metadata at the time the command is run.
   - No pre-generated tree artifacts required.

---

## 3. Non-Goals

1. **Graphical / web UI**
   - No HTML/graphviz/web visualizations in this feature. Output is plain text only.

2. **Changelog or release planning**
   - The tree is informational only; it does not propose or apply release plans.

3. **Config schema changes**
   - No new required fields in `.squire/config.json` are needed for this feature.

4. **Idempotent embed management** (for now)
   - Phase 1 of this feature may not attempt to detect and replace an existing embedded tree; it may simply insert a new code block before/after the marker on each run.
   - Smarter replacement logic can be added later if desired.

5. **Universe multi-root UX**
   - The initial design focuses on the current repo and any attached repos Squire already handles via `DiscoverModules`.

---

## 4. CLI Interface

### 4.1 Command name

**Working name:**

```bash
squire requires tree [<dir>] [flags]
```

- `requires` subcommand namespace reflects that we are working with **module requirements** (from `go.mod`).
- `tree` is the subcommand that produces the tree visualization.

> If the top-level subcommand naming changes later (e.g. this moves under another namespace), the internal behavior and options here should still apply.

### 4.2 Arguments & flags

- `dir` (optional positional)
  - Defaults to `.`
  - Must be any directory inside a Squire-managed repo; used as the starting point for discovery.

- `--all`
  - When **absent** (default):
    - The tree only shows modules Squire manages (`ModuleSet`), i.e., modules discovered by `DiscoverModules`.
  - When **present**:
    - The tree also includes external modules (non-Squire modules) that are required by any module in the tree.
    - See §6 for how external dependencies are discovered.

- `--show-dirs`
  - Affects how each node label is rendered.
  - For **local (Squire-managed) modules**:
    - Show the module’s **relative directory** from the repo root instead of the Go module path, e.g. `./`, `./cmd`, `./test`.
  - For **non-local modules** (external modules when `--all` is used):
    - Show the **module path** (since a local directory is not applicable).

- `--show-all`
  - Expands the node label to include **both** module identity and location when available.
  - For local modules:
    - Show `modulePath (~/<relative-path-from-$HOME>)`, where the filesystem path is rendered with a leading `~` instead of the full `$HOME` prefix, e.g.:
      - `github.com/mikeschinkel/go-dt (~ /Projects/go-dt)`.
  - For external modules:
    - Show the module path, optionally followed by any resolved local cache path (if known), also rendered with a leading `~`.
  - `--show-all` implies the same information that `--show-dirs` provides, but always includes the module path as the primary identifier.

- `--embed=<markdown_file>`
  - If provided, Squire will embed the generated tree into the given markdown file instead of (or in addition to) printing to stdout.
  - Must be a path to an existing markdown file.
  - Requires exactly one of `--before` or `--after`.

- `--before`
  - Only valid when `--embed` is provided.
  - Indicates that the tree code block should be inserted **before** the marker `<!-- squire:embed-requires-tree -->` in the target file.

- `--after`
  - Only valid when `--embed` is provided.
  - Indicates that the tree code block should be inserted **after** the marker.

**Mutual exclusivity rules:**

- If `--embed` is **not** specified:
  - `--before` and `--after` are invalid and must cause an error.

- If `--embed` **is** specified:
  - Exactly one of `--before` or `--after` is required.
  - Providing both is an error.
  - Providing neither is an error.

---


## 5. Tree Output Format

### 5.1 Basic structure

The tree is rendered as plain text suitable for a ` ```text ` fenced code block.

**Example (internal-only, no `--all`):**

```text
github.com/xmlui/cli
├─ github.com/mikeschinkel/go-dt
│  ├─ github.com/mikeschinkel/go-doterr
│  └─ github.com/mikeschinkel/go-logutil
└─ github.com/xmlui/localsvr
   └─ github.com/mikeschinkel/go-dt
```

Rules:

1. Each node line should show, at minimum, the **module path** (`Module.ModulePath`).
2. The top-level root(s) are the module(s) belonging to the current repo:
   - For repos with a single module, there is a single root.
   - For repos with multiple modules, each module in that repo is a root; it is acceptable to render them as separate top-level entries.
3. Child entries show **direct dependencies** of a module.
4. Indentation and branch characters should follow a consistent ASCII style:
   - `├─` for non-final children.
   - `└─` for final children.
   - Indentation with `│` and spaces for clarity.

### 5.2 Internal-only vs `--all`

- **Internal-only (default)**:
  - The tree is built using only Squire-managed modules, i.e. modules returned in `ModuleSet`.
  - For any dependency not managed by Squire, the dependency may be:
    - omitted, or
    - optionally summarized as a stub (TBD; see §6.3).

- **With `--all`**:
  - External modules are included as nodes.
  - The tree attempts to follow external dependencies as well, subject to practical constraints (see §6).

### 5.3 Multiple occurrences

When the same module appears multiple times via different paths in the dependency tree:

- The tree should **avoid infinite recursion**.
- For subsequent occurrences of a module already shown higher in the tree, acceptable behaviors include:
  - Re-showing it as a leaf node (no further expansion), or
  - Showing a special notation indicating it was already expanded elsewhere (e.g. `… (already shown)`).

The choice for Phase 1 of this feature can be simplest-first: re-show the module as a leaf without re-expanding its children.

---

## 6. Dependency Discovery

### 6.1 Internal dependencies (Squire-managed)

For internal modules, the tree uses the Phase 2 `ModuleSet`:

- `DiscoverModules(dir)` builds a `ModuleSet` for the current repo (and any attached repos as defined by Squire’s universe rules).
- For each `Module`, the `Dependencies` field lists module paths of other Squire-managed modules that this module requires.
- The internal-only tree walks these `Dependencies` starting from the current repo’s module(s).

### 6.2 External dependencies (`--all`)

When `--all` is specified, the tree must extend beyond Squire-managed modules:

1. For each module (internal or external) in the current frontier:
   - Inspect its `go.mod` (for internal modules) or use `go list` / other means (for external modules) to obtain the list of `require` entries.

2. For each `require`d module path `P`:
   - If `P` is a Squire-managed module (`ModuleSet.byPath[P]` exists), follow it as usual.
   - If `P` is not a Squire-managed module:
     - Include `P` as an external node in the tree.
     - Optionally, recursively expand `P`’s own requirements, if reasonably obtainable via tooling (e.g. `go list -m -json`), subject to:
       - reasonable depth limits,
       - error reporting when external metadata is unavailable.

3. External modules do not need the full `Module` struct; a simpler internal representation may be used, but the **rendered tree** should not distinguish stylistically between internal and external nodes unless there is a strong UX reason to do so.

### 6.3 Practical considerations / constraints

- If external dependency resolution fails (network, `GOPROXY` issues, etc.), the tree should still render internal modules and any external modules it can resolve.
- It is acceptable, in the first implementation, to:
  - include external modules as **one-level leaves** (no further expansion), or
  - have an internal depth limit for external expansion.

The PRD does not mandate how deep external dependencies must be expanded; the key requirement is that `--all` includes external module nodes and not just internal ones.

---

## 7. Markdown Embedding Behavior

### 7.1 Marker and placement

- The marker comment is:

  ```html
  <!-- squire:embed-requires-tree -->
  ```

- `--embed=<markdown_file>` instructs Squire to:
  1. Read the specified markdown file.
  2. Search for the marker comment.
  3. Generate the dependency tree as described above.
  4. Wrap the tree in a fenced code block:

     ```text
     ```text
     <tree>
     ```
     ```

     (The actual embed will be ` ```text` + newline + tree + newline + ` ``` `.)

  5. Insert the code block **before** or **after** the marker, depending on flags.

- If the marker is not found in the file, return an error.

### 7.2 `--before` / `--after` rules

- When `--embed` is present:
  - Exactly one of `--before` or `--after` must be provided.
  - If both are provided, error.
  - If neither is provided, error.

- When `--embed` is not present:
  - Use of `--before` or `--after` is an error.

### 7.3 Idempotency (initial behavior)

For the initial implementation:

- The command is **not required** to detect and replace existing tree blocks.
- Each invocation may insert a new ` ```text` block before/after the marker.

Future improvements may include an optional `--replace` mode that:

- looks for a previous ` ```text` block adjacent to the marker, and
- replaces it atomically with a fresh tree render.

---

## 8. Error Handling

The command should fail with clear messages when:

1. `dir` is not inside a Squire-managed repo.
2. `.squire/config.json` is missing or malformed for the relevant repo.
3. `go.mod` files required for internal modules cannot be read or parsed.
4. `--embed` is provided but the target file does not exist or is not readable/writable.
5. `--embed` is provided but the marker `<!-- squire:embed-requires-tree -->` is not found.
6. `--embed` is provided without a valid combination of `--before` / `--after`.

In all failure cases, the command must:

- not partially edit the markdown file, and
- exit with a non-zero status code.

---

## 9. Testing Strategy

### 9.1 Tree rendering tests

- Use fixture repos (similar to Phase 2 testdata) with known module structures.
- Tests should verify that:
  - The internal-only tree matches expected ASCII output for a simple universe.
  - The `--all` variant includes at least one external module in the tree when present.
  - The tree avoids infinite recursion when modules depend on each other in a cycle (in such a case, a cycle error may be raised instead of a partial tree).

### 9.2 Embedding tests

- Create markdown fixtures containing the marker at various positions:
  - middle of file,
  - top of file,
  - bottom of file.
- Verify that:
  - `--embed`, `--before` inserts a code block **before** the marker.
  - `--embed`, `--after` inserts **after**.
  - The inserted block has the exact ` ```text` wrapping and contents.
  - Subsequent runs append additional blocks (current behavior) without corrupting the file.

### 9.3 Flag validation tests

- Ensure error conditions for invalid combinations of flags (`--embed` with no marker, `--before` without `--embed`, etc.) are hit and reported clearly.

---

## 10. Acceptance Criteria

This feature is considered complete when:

1. A CLI command (working name `squire requires tree`) exists and:
   - prints an internal-only dependency tree when called without `--all` and without `--embed`.
   - includes external dependencies in the tree when `--all` is specified.

2. The tree is rendered as deterministic, readable ASCII suitable for markdown ` ```text` blocks.

3. `--embed=<markdown_file> --before` and `--embed=<markdown_file> --after` insert the tree into the given file at the correct position relative to the marker without corrupting the file.

4. The implementation leverages Phase 2’s `Module` and `ModuleSet` for internal modules and does not require changes to `.squire/config.json`.

5. Unit tests cover tree rendering, embedding, and flag validation as described above, and pass reliably.

