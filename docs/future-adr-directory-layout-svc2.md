
# ADR: Rename “module” directory to `<app>mod` and use `<app>pkg` for stable domain API

* **Status:** Accepted
* **Date:** 2025-12-26

## Context

The repo is a multi-module CLI layout (modules: `cmd`, `test`, and the reusable app module). The directory previously called `<app>pkg` actually represented the app’s reusable **Go module** (contained `go.mod`) and primarily held bootstrapping/wiring code (`ParseConfig`, `ParseOptions`, `RunCLI`, `Run(RunArgs)`). Naming it `<app>pkg` was misleading and also forced “services” naming decisions to compensate.

## Decision

* Rename the reusable module directory from `<app>pkg` to **`<app>mod`** (it is the app’s reusable Go module).
* Use **`<app>pkg`** as the stable, library-ish domain API package for domain functionality.
* Keep **`<app>svc`** unused for now and reserve it for a future “application services/use-cases” layer if needed.

## Rationale

* `mod` aligns with Go terminology: the directory is literally a module with a `go.mod`.
* `pkg` aligns with the intended meaning: reusable domain/library API.
* This naming clarifies layering: executable (`cmd`) calls module entrypoints (`<app>mod.Run*`), while other code can import `<app>pkg` for domain functionality.
* Import cycles are avoided by enforcing one-way dependencies: `<app>pkg` never imports commands/UI.

## Consequences

* Establish dependency rules:

    * `<app>pkg` must not import `<app>cmds`, `<app>cliui`, `<app>tui`, or `<app>mod`.
    * `<app>mod` may import `<app>pkg` and UI/command packages as needed for wiring.
* If orchestration grows, introduce `<app>svc` later as a non-UI “use-case” layer between commands and `<app>pkg`.

If you want, I can also rewrite your original tree to reflect this rename (showing where `appcfg`, `appcmds`, `appcliui`, `apptui`, `apppkg` live under `appmod`) in a way that makes the dependency arrows obvious.
