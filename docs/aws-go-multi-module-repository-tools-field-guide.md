# aws go multi module repository tools field guide

---

## Repo overview

* **Repo:** `github.com/awslabs/aws-go-multi-module-repository-tools` ([GitHub][1])
* **Purpose:** Toolkit AWS uses to manage a very large multi-module Go repo (AWS SDK v2): dependency updates (internal + external), release calculation, tagging, changelog generation, smithy codegen integration, and “run this across all modules” scripting. ([GitHub][1])
* **Layout (high-level):**

    * `cmd/*` – individual CLI tools.
    * `changelog`, `gomod`, `git`, `manifest`, `release`, `internal/semver` – shared libraries used by those tools. ([GitHub][1])
    * Root helpers like `config.go`, `repo_root.go`, `walk.go`, `go_module_metadata.go`. ([GitHub][1])

---

## Core configuration: `modman.toml`

* **File:** `modman.toml` at repo root. ([GitHub][1])
* **Role:** Drives behavior of dependency and release tools.

Key sections:

1. **`[dependencies]`**

    * Map: Go module path → version string.
    * Used by `updaterequires` to rewrite `require` entries in all `go.mod` files that depend on that module.
    * Example:

      ````toml
      [dependencies]
      "github.com/aws/smithy-go" = "v1.4.0"
      ``` :contentReference[oaicite:5]{index=5}  
 
      ````
2. **`[modules."path"]`**

    * Key = module dir relative to repo root (`"."` for root module). ([GitHub][1])
    * Values control release behavior. Example:

      ```toml
      [modules."feature/service/shinything"]
      no_tag = true
      ```
    * Used primarily by `calculaterelease` and related release tools to decide which modules to tag and how. ([GitHub][1])

For Gomion PRDs, `modman.toml` is the conceptual precursor to `.gomion/config.json` plus a dependency-policy file.

---

## Changelog & annotation system

### Library packages

* **`changelog` package**

    * Data structures + helpers for changelog annotations: types, templates, reading/writing `.changelog` files. ([rpm.pbone.net][2])

### On-disk format

* **`.changelog/` directory at repo root**

    * Stores per-change annotations (YAML/TOML/JSON-ish; exact schema is in `changelog/annotation.go`). ([GitHub][1])
    * Annotations drive semver bump computation and human-readable changelogs.

### Commands

All under `cmd/`:

1. **`cmd/changelog`** ([GitHub][1])

    * Multi-subcommand CLI for managing annotation files in `.changelog/`.
    * Subcommands (seen in RPM listing): `create`, `edit`, `list`, `view`, `remove`.
    * Uses `editor.go` to spawn `$EDITOR` for editing annotations.

2. **`cmd/generatechangelog`** ([GitHub][1])

    * Consumes:

        * A **release description** + associated annotations.
    * Produces:

        * `CHANGELOG.md` entries per module.
        * A summarized release statement at repo root (for release notes).
    * Key files: `main.go`, `summary.go`, `template.go`, tests. ([rpm.pbone.net][3])

3. **`cmd/annotatestablegen`** ([GitHub][1])

    * Specialized tool for AWS’s smithy pipeline.
    * For newly generated modules that are marked “stable”, creates appropriate changelog annotations automatically.

**What to steal for Gomion:**

* The **annotation model** and file layout (`.changelog/`).
* The idea of a **separate “generate changelog” step** that merges annotations into human-readable docs.
* The editor-driven flow (spawn `$EDITOR` to refine annotations).

---

## Dependency management & go.mod editing

### Library

* **`gomod` package**

    * Models modules and their relationships (`module.go`, `module_tree.go`) and has `diff.go` / `version.go` utilities. ([rpm.pbone.net][2])

### Commands

1. **`cmd/updaterequires`** ([GitHub][1])

    * Reads `[dependencies]` from `modman.toml`.
    * Walks all modules in the repo.
    * For any module that depends on a configured external module, rewrites `require` lines in `go.mod` to the configured version.
    * Supports both inter-repo and external dependencies.

2. **`cmd/editmoduledependency`** ([rpm.pbone.net][3])

    * More targeted tool: edit a specific module’s dependency requirement (likely interactive/one-off).
    * Has its own `README.md` describing usage.

3. **`cmd/moduleversion`** ([rpm.pbone.net][3])

    * Prints/derives the effective version for a module based on git tags + semver; useful for scripts.

4. **`cmd/makerelative`** ([GitHub][1])

    * Generates `replace` directives in `go.mod` so **internal AWS SDK modules** point at local paths instead of proxy versions.
    * Used for “develop with cloned monorepo” scenarios.
    * Works in concert with `gomod` and `modman` metadata.

5. **`cmd/gomodgen`** ([GitHub][1])

    * Specific to AWS smithy codegen:

        * Copies smithy-go build artifacts into the SDK repo.
        * Generates `go.mod` for those generated modules based on a `generated.json` descriptor. ([GitHub][1])

**What to steal for Gomion:**

* Strategies for **bulk updating `require` entries** (via a policy map).
* The concept of **makerelative/clearrelative** to manage dev-time local `replace` vs release-time clean `go.mod`.
* A representation of module trees and version diffs usable for release planning and dependency inspection.

---

## Release planning & tagging

### Library packages

* **`release` package** – core logic for computing per-module releases using semver + annotations + git. ([rpm.pbone.net][2])
* **`manifest` package** – defines the “release manifest” format shared between `calculaterelease` and `tagrelease`. ([rpm.pbone.net][2])
* **`internal/semver`** – semver handling utilities. ([rpm.pbone.net][2])
* **`git` package** – wrappers for `git diff`, `git tag`, add/commit, etc. ([rpm.pbone.net][3])

### Commands

1. **`cmd/calculaterelease`** ([GitHub][1])

    * Scans the repo to:

        * Detect **new / changed modules** (using `gomod` + git).
        * Associate each change with its **changelog annotations**.
    * Uses internal semver rules plus annotations and `modman` module config (e.g., `no_tag`) to compute the next version per module.
    * Emits a **release manifest** describing module→version mapping and other metadata; used by other tools.

2. **`cmd/tagrelease`** ([GitHub][1])

    * Reads the release manifest from `calculaterelease`.
    * Commits pending changes and **creates tags** for each module.
    * Tag format follows Go multi-module semver conventions (e.g., `modName/vX.Y.Z`), via the `git` + `release` packages.

3. **`cmd/updatemodulemeta`** ([GitHub][1])

    * Regenerates `go_module_metadata.go` for each module.
    * Embeds runtime metadata such as the module path and tagged version, so applications/libraries can introspect their own version at runtime.

4. **`cmd/generatechangelog`** (already covered above) is also part of the release pipeline: turns annotations + manifest into human-readable CHANGELOGs.

**What to steal for Gomion:**

* The **two-stage release pipeline**:

    * 1. Compute a manifest (plan).
    * 2. Apply it (tagging + metadata + changelog).
* The use of **annotations + semver rules + git diff** to determine bump type.
* The idea of a per-module **metadata file** for runtime version awareness.
* A structured release manifest format as the central artifact for CI/CD.

---

## Cross-module scripting

* **`cmd/eachmodule`** ([GitHub][1])

    * Enumerates modules in the repo and runs an arbitrary shell command in each module directory.
    * Used as a generic “for each Go module” loop primitive in scripts.

This is essentially their primitive for what you’re thinking of as `gomion each`.

---

## Common utilities in the root

* **`config.go`** – parses `modman.toml` into a struct model used by release/dependency tools. ([GitHub][1])
* **`repo_root.go`** – logic to find the git repo root from an arbitrary working directory. ([GitHub][1])
* **`walk.go`** – repo tree walking helper used by multiple commands. ([riscv-koji.fedoraproject.org][4])
* **`editor.go`** – generic “open a temp file in `$EDITOR` and read back the result” helper, used by changelog tools. ([GitHub][1])
* **`go_module_metadata.go`** – shared type definitions for module metadata generated by `updatemodulemeta`. ([GitHub][1])

---

## TL;DR for future PRDs

When we write Gomion PRDs that “cannibalize” this repo, the key concepts to remember are:

* **`modman.toml`** as:

    * A central place for dependency policy (`[dependencies]`).
    * A per-module config map (`[modules."path"]`) controlling release behavior.
* **Changelog annotations** as a separate data stream in `.changelog/`, edited by humans but machine-consumed.
* **Release workflow split into compute vs apply** (`calculaterelease` → manifest → `tagrelease`).
* **Dependency update tooling** (`updaterequires`, `makerelative`) as a pattern for large multi-module repos.
* **Drop-in pieces** we might reuse/port: `changelog` package, `gomod` module tree/diff logic, `release`/`manifest` semantics, `git` wrappers, distributed `go_module_metadata` pattern, and `eachmodule` as a core “apply over modules” primitive.

If you paste this summary into a new chat, I’ll have everything I need to write Gomion PRDs that draw from the AWS tools without having to re-open their repo.

[1]: https://github.com/awslabs/aws-go-multi-module-repository-tools "GitHub - awslabs/aws-go-multi-module-repository-tools"
[2]: https://rpm.pbone.net/info_idpl_125120109_distro_fedoraother_com_golang-github-awslabs-aws-multi-module-repository-tools-devel-0-0.7.20221016gitb6ea859.fc41.noarch.rpm.html?utm_source=chatgpt.com "RPM Fedora Other golang-github-awslabs-aws-multi-module- ..."
[3]: https://rpm.pbone.net/content_idpl_94292848_distro_mageiacauldron_com_golang-github-awslabs-aws-multi-module-repository-tools-devel-0-0.1.mga9.noarch.rpm.html?utm_source=chatgpt.com "RPM Search mageiacauldron golang-github-awslabs-aws-multi ..."
[4]: https://riscv-koji.fedoraproject.org/koji/rpminfo?buildrootOrder=-id&buildrootStart=100&fileOrder=-name&rpmID=20493&utm_source=chatgpt.com "golang-github-awslabs-aws-multi-module-repository ... - RISC-V Koji"
