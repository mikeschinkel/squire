# Squire Vision Document – Long-Term Vision (Non‑v0 Scope)

This Vision document is meant to **replace all prior informal Squire design notes** (the four earlier Squire docs and the standalone `doterr` PRD) for everything **outside the v0 PRD**.

- The **v0 PRD** covers only the initial `scan` / `init` functionality and the minimal `modules{}` config needed to support it.
- This **Vision** captures the broader problem space, long‑term goals, design constraints, and feature families we intend to tackle in **future PRDs**.

It is intentionally descriptive, not prescriptive at the level of flags and field names (that is PRD territory). It should be read as the high‑level map for where Squire is going.

---

## 1. Problem Space & Context

Squire is intended to support a developer with:

- Many Git repos in directories like `~/Projects/xmlui`, `~/Projects/go-pkgs`, etc.
- Repos that often contain **multiple Go modules**:
  - A package module (e.g., `apppkg`).
  - One or more `cmd/*` modules for CLIs / daemons.
  - One or more `test` modules for integration or harness tests.
- A mixture of **patterns**:
  - **Lockstep app repos** where lib + CLI + tests should share a version.
  - **Independent library families** (e.g., dt/dtx/dtglob/appinfo) that live together but version separately.
  - **Root-of-tree application repos** (future "root_repos") that sit on top of a graph of shared packages.
  - **Drop‑in embeds** like `go-doterr` that are copied into repos as single files, not imported as normal module dependencies.

Current pain points include:

- Manually orchestrating **releases across many repos and modules**.
- Deciding when a change is patch/minor/major based on memory and intuition.
- Keeping `go.mod`, `replace` directives, `go.work`, and GoReleaser config synchronized.
- Maintaining "code hygiene" (tests, docs, ClearPath discipline) before releases.
- Managing **non‑dependency dependencies** (drop‑ins) like `doterr` that are intentionally copied instead of imported.

Squire’s long‑term mission is to make that multi‑repo, multi‑module world **safer, more observable, and harder to screw up**, while staying Go‑centric and low‑magic.

---

## 2. High‑Level Goals (Beyond v0)

1. **Poly‑repo, multi‑module orchestration**
   - Understand repos and modules across the filesystem and how they relate.
   - Treat selected repos as **root_repos** for applications or services.
   - Provide commands that operate over entire dependency trees ("everything this app depends on") rather than single modules.

2. **Release pipelines with minimal human error**
   - Automate mechanical steps (detecting changed modules, computing version bumps, tagging, wiring go.mod, kicking CI).
   - Use both **human input** (annotations) and **objective signals** (API diffs) to suggest semver bumps.
   - Gate releases on testing, code hygiene, and policy, so "I forgot" is less likely.

3. **Code hygiene and ClearPath enforcement**
   - Encode ClearPath‑style rules (single return plus cleanup region, explicit error handling, controlled use of `goto end`, etc.).
   - Provide commands that assert the codebase is in a releasable, maintainable state (tests + lint + ClearPath).

4. **Non‑dependency dependencies (drop‑ins)**
   - Treat drop‑ins like `go-doterr` as a first‑class pattern with dedicated commands.
   - Support modes like `dot`, `copy`, `ignore` as described in the doterr PRD.
   - Make updating drop‑ins an explicit, conscious action, separate from normal dependency bumps.

5. **Stable, CI‑friendly configuration model**
   - Encode all behavior CI needs in **repo‑local** files (`.squire/config.json` plus any Squire‑managed subdirs like changelog or dropins).
   - Restrict machine‑local config (`~/.config/squire/`) to ergonomics, performance, and user‑specific preferences.

6. **Longevity and maintainability**
   - Avoid dependencies on fast‑moving frameworks.
   - Prefer plain Go, file‑based protocols, and simple, explicit JSON/TOML/YAML formats.
   - Keep Squire usable for 10+ years across machines and environments.

---

## 3. Core Concepts (Extended)

These concepts are the foundation for all future PRDs.

### 3.1 Repo‑local module metadata

- Each repo has a `.squire/config.json` at its root.
- v0 defines only a `modules{}` map; future PRDs will extend this file,
  but **repo‑local config remains the canonical source of truth** for:
  - Which modules exist in the repo.
  - What their **names** and **roles** are.
  - How they participate in releases, dependencies, and drop‑ins.

Key conventions that carry forward:

- `modules` keys are **repo‑relative paths** like `"./apppkg"`, `"./cmd"`, `"./test"`.
- Each module has a unique `name` within its repo.
- `role` is a list of tags such as:
  - `"lib"` – library / package module.
  - `"cli"` – command‑line entrypoint.
  - `"test"` – test harness.
  - Future roles could include `"daemon"`, `"support"`, etc.
- Later PRDs may add additional per‑module fields (e.g., `version_strategy`, or drop‑in participation) but `name` + `role` stays central.

### 3.2 Version strategy and release groups

We anticipate two primary versioning behaviors:

- **Independent** – a module versions on its own (e.g., dt, dtx, dtglob, appinfo).
- **Lockstep** – a set of modules share a version (e.g., `apppkg` + `squire-cli` + tests).

Rather than baking this into the v0 schema, future PRDs will:

- Add a **per‑module strategy** field (e.g., `"independent"`, `"lockstep"`).
- Introduce **release groups** that identify modules that should version together:

  ```jsonc
  {
    "groups": [
      {
        "name": "squire",
        "strategy": "lockstep",
        "modules": ["apppkg", "squire-cli", "test"]
      }
    ]
  }
  ```

  - `name` – identifier for the group (e.g., `"squire"`).
  - `strategy` – how this group versions (`"lockstep"` initially; others may be added later).
  - `modules` – module **names** (not paths) participating in the group.

Release groups act as **versioning units** in release planning: one version decision per group.

### 3.3 Root repos vs universes (future "workspace" concept)

- A **root_repo** is a repo that:
  - Serves as the root of a dependency tree (e.g., the main CLI or service).
  - Has a `.squire/config.json` describing its modules and, eventually, its release groups and dependency policies.
- A **universe** (name TBD, used instead of "workspace" to avoid confusion with `go.work`) is a user‑level construct:
  - A set of root_repos (and their trees) that you are actively working on.
  - Known and configured in machine‑local config under `~/.config/squire/`.

Future PRDs will describe commands like:

- `squire each` – run a command against each module or repo in the current universe.
- `squire status` – summarize changes, dependency health, and release readiness across the universe.

### 3.4 Drop‑ins (non‑dependency dependencies)

Drop‑ins like `go-doterr` are **not** treated as normal Go module deps. Instead:

- They are described in `.squire/config.json` under a future `dropins` section, e.g.:

  ```jsonc
  {
    "dropins": {
      "doterr": {
        "source": "github.com/mikeschinkel/go-doterr@v0.3.1",
        "template": "1",          // template track/major
        "default_mode": "copy",   // `dot` | `copy` | `ignore`
        "targets": [
          {"module": "apppkg", "path": "internal/doterr.go", "mode": "copy"}
        ]
      }
    }
  }
  ```

- Modes mirror the doterr PRD:
  - `dot` – use a dot import pattern.
  - `copy` – copy the doterr implementation into the target path.
  - `ignore` – explicitly opt out.

The CLI surface should treat drop‑ins as a separate family, for example:

- `squire dropin init doterr`
- `squire dropin status doterr`
- `squire dropin update doterr`
- `squire dropin check doterr`

Updates to drop‑ins are always **explicit** and **rare**, and must never be silently triggered as part of normal dependency updates.

### 3.5 Changelog annotations

Squire will use a structured changelog system, conceptually similar to AWS’s `.changelog` files but tailored to this ecosystem:

- Annotation files stored in a Squire‑managed area (e.g., `.squire/changelog/`).
- Each annotation records:
  - Which modules (by `name`) are affected.
  - Change type: `breaking | feature | fix | doc | internal`.
  - Short summary and optional details.
  - Optional metadata (issue IDs, PR links).

These annotations will:

- Feed into release planning (see §4.1).
- Be combined into human‑readable changelogs for modules and repos.
- Potentially be drafted by Squire from diffs and API changes, with optional AI assistance.

We have explicitly **postponed** deciding exactly where changelog creation is enforced (on commit, on release planning, etc.) until after real‑world experience with Squire. The enforcement point is a future PRD decision, but the **existence** of structured annotations and their use in release workflows is part of this Vision.

### 3.6 Machine‑local state, ignore lists, and caches

Machine‑local config under `~/.config/squire/` will eventually include:

- Paths to roots (`~/Projects/xmlui`, `~/Projects/go-pkgs`, …) the user cares about.
- Definitions of universes/stacks (user‑named sets of root_repos).
- An **ignore list** of repos that should never be Squire‑managed (so scans can omit them).
- Caches of scan results to speed up operations while respecting the constraint that CI cannot depend on them.

Machine‑local state is always **optional and reconstructible**; CI must rely only on repo‑local config.

---

## 4. Future Capability Families (for Later PRDs)

### 4.1 Release orchestration

Target commands (names tentative):

- `squire release plan`
  - Given a root_repo (and optional universe):
    - Determine which modules and release groups have changes since their last tagged version.
    - Use changelog annotations + API diffs to propose semver bumps per module/group.
    - Flag inconsistencies (e.g., annotations say patch but API changes are breaking).
    - Output a release manifest detailing proposed versions and notes.

- `squire release apply` (or `squire release tag`)
  - Consume the manifest and:
    - Run configured gates (tests, linters, ClearPath, etc.).
    - Apply Git tags following Go multi‑module conventions.
    - Optionally update generated metadata files or manifests.

Future PRDs will define:

- Exact manifest formats.
- Integration points with GoReleaser and CI/CD pipelines.
- Behavior for partial failures and retries.

### 4.2 Dependency policy & go.mod / go.work management

Squire will manage dep behavior beyond what `go mod tidy` does by default:

- A `dependencies` section in `.squire/config.json` to describe per‑dep policies, e.g.:
  - Follow `go.mod` as authored (default).
  - Track latest `v0.x.y` for a given module.
  - Use a different version for one module vs others (e.g., `cmd/foo` pinned to `v0.1.0` while other modules use latest).
- CLI commands like:
  - `squire deps list` – show dependencies across modules, grouped and filtered.
  - `squire deps update` – apply policies by editing `go.mod` in a controlled manner.
  - `squire deps makerelative` / `squire deps clearrelative` – manage local `replace` directives for intra‑repo / intra‑workspace development.

All such commands will adhere to:

- **Explicitness** – nothing mutates `go.mod` or `go.work` without the user asking for it.
- **Auditability** – changes are small, clear, and easy to review in diffs.

### 4.3 Code hygiene, ClearPath, and readiness checks

Squire will incorporate code hygiene checks including ClearPath rules:

- Potential commands:
  - `squire clearpath lint` – Enforce ClearPath rules (no early returns, structured cleanup via `goto end`, etc. where desired).
  - `squire code scan` – Run configured static checks (e.g., `go vet`, `golangci-lint`, custom analyzers) across Squire‑managed modules.
  - `squire code ready` – Aggregate health checks to assert a module/repo is ready for release.

Integration points:

- Release orchestration commands can require these checks to pass before tagging.
- Config may allow per‑module or per‑group control over which checks apply.

### 4.4 Docs, structure, and graphing

Future commands may also address documentation and structural quality:

- `squire docs scan` – Identify missing or stale docs (missing README, missing exported symbol comments, etc.).
- `squire deps graph` or `squire imports map` – Visualize module and package‑level dependencies.
- Helpers to generate or update metadata files consumed by external catalogs/registries.

These features are lower priority than release orchestration and drop‑ins but remain important for long‑term maintainability.

### 4.5 Universes / stacks and `squire each`

Once universes/stacks are defined, Squire can:

- Provide `squire each` to run arbitrary commands over multiple repos/modules, e.g.:

  ```bash
  squire each go test ./...
  squire each git status
  ```

- Use universes in higher‑level commands:
  - `squire status` – aggregated view of which repos/modules have changes, failing tests, or pending releases.
  - `squire release plan` – operate over a whole universe rather than a single repo.

The exact data model for universes and ignore lists (e.g., patterns vs explicit paths) will be detailed in separate PRDs.

### 4.6 TUIs and richer UX

After a CLI‑only baseline is stable, TUIs can:

- Replace the `scan.txt → init.txt` workflow with an interactive adoption UI.
- Allow interactive editing of `.squire/config.json` (module roles, group membership, drop‑ins).
- Provide dashboards for status, errors, and release plans.

TUI design will emphasize:

- Non‑destructive previews.
- Clear indications of what will be written to disk.
- Keyboard‑friendly workflows for power users.

---

## 5. Architectural Principles (Summary)

These principles govern future design decisions and PRDs:

1. **Repo‑local truth for CI** – CI must be able to operate using only repo‑local files and Squire binaries.
2. **One canonical watch per concern** – Each concern (modules, drop‑ins, dependencies, changelog) has a single authorable source of truth.
3. **Minimal duplication** – Prefer inference or reference over copying data between files.
4. **Safety and explicitness** – Mutating operations require explicit commands and should be easily auditable.
5. **Extensible schema** – `.squire/config.json` is designed to grow new sections (`groups`, `dependencies`, `dropins`, etc.) without breaking existing repos.
6. **Go‑first, broadly applicable** – Optimize for Go ecosystems but allow patterns that could generalize to other languages in the future.

This Vision, together with the v0 PRD, is now the **canonical design reference** for Squire. All future PRDs should treat it as the starting point, refining and extending specific areas (release orchestration, drop‑ins, deps, ClearPath, universes, TUIs) without re‑introducing the four legacy docs or the standalone doterr PRD.

