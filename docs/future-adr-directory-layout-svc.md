# ADR: Use `appsvc` for domain/workflow logic package

* **Status:** Accepted
* **Date:** 2025-12-25

## Context

The CLI app repository uses a starter layout that separates reusable code (`apppkg/*`) from app-specific packages to avoid Go import cycles. We needed a short (2–3 char) suffix for the package that holds the app’s core “engine” logic (workflows/rules/orchestration) called by CLI commands.

Candidate suffixes considered included `dom` (domain), `biz` (business rules), `dm` (domain model), and `svc` (services).

Constraints and preferences:

* Avoid import cycles while keeping code organized.
* Avoid requiring import aliasing when composing multiple packages with similar roles.
* The mental model is **“commands call services.”**
* “Business rules” terminology feels mismatched for technical tooling.
* The package may be a pragmatic “bucket” whose primary purpose is to break cycles; truly reusable/cross-cutting utilities can be extracted into separate packages later (e.g., `gitutils`).

## Decision

Use **`appsvc`** (suffix `svc`) as the package for the app’s core workflows/policy/orchestration logic that commands invoke.

```
app - Name of app, e.g. `gomnion` or `gomion`
├── cmd — Executable lives here
├── test — Integration tests live here
└── apppkg — Reusable package
    ├── app — No-dependency package for shared types, consts, and vars
    ├── appcfg — Serializable configuration models using built-in data types (string, int, etc.)
    ├── appcmds — Command line commands
    ├── appcliui — Console UI for commands
    ├── apptui — Full screen text UI for commands  (optional)
    └── appsvc — Domain model/business rule functionality, Parse functions for config
```
## Rationale

* `svc` matches the intended layering: **commands → services**.
* `svc` reads cleanly when concatenated (`gomionsvc`, `gomionsvc`) and avoids the unintended “kingdom”/“dominion” parsing that occurs with `dom`.
* `svc` avoids the “business” connotation of `biz` while still conveying “operations the tool provides.”
* The package can function as a pragmatic boundary to prevent cycles; when code is clearly reusable or cross-cutting, it can be factored into dedicated packages without changing the overall convention.

## Consequences

* `appcmds` should depend on `appsvc`, not vice versa.
* UI packages (`appcliui`, `apptui`) must not be imported by `appsvc`.
* Over time, reusable helpers may be extracted from `appsvc` into focused packages (e.g., `fsutils`, `awsutils`, etc) as needed, but `appsvc` remains the primary entry point for the app’s operational workflows.
