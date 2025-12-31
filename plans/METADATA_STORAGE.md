# ADR: Branch-Independent Dependency Branch Metadata for Go Module Analysis

## Status

**Accepted**

## Date

2025-12-18

## Context

We are building tooling that analyzes Go module dependency graphs across multiple Git repositories.
For correctness, the tooling must validate that **each dependency is checked out on the expected branch and remote** before analysis can proceed.

Key constraints and realities:

* Dependencies are expressed via `go.mod` files.
* Each repository may itself depend on other repositories.
* There is **no global “root” repository** — every repository is authoritative only for its *direct* dependencies.
* Git branches are **not reliable as a canonical metadata store**:

    * Branches may be deleted, rebased, or disconnected.
* Git **worktrees may be used**, meaning:

    * Multiple branches may be checked out simultaneously.
    * Branch checkout state is per-directory, not per-repository.
* The tooling **must not persist or invent Git state**.

    * Git remains the source of truth for checkout state.
* The tooling **does not require storing observed checkout state** — only validating it at runtime.

We need a way to store **branch-independent expectations** about dependency checkouts that:

* Are local to a repository
* Are not affected by branch switching
* Can be mirrored to a versioned, shareable form
* Work correctly with or without Git worktrees

---

## Decision

### 1. Authority Model

**Each repository is authoritative for the expected branch and remote of its direct dependencies.**

There is no global root and no upward declaration (“who depends on me”).
Authority flows **outward along dependency edges**, matching Go module semantics.

---

### 2. What Is Stored (and What Is Not)

#### Stored (branch-independent expectations)

For each direct dependency, we store:

* Dependency module path
* Expected Git remote
* Expected Git branch
* Filesystem path used to satisfy the dependency (directory anchor)

These are **expectations**, not observed state.

#### Not stored

* Current checkout branch
* HEAD state
* Worktree enumeration
* Any derived Git state

Observed checkout state is always queried live from Git.

---

### 3. Storage Locations

We use **two complementary storage mechanisms**:

#### A. `.git/config` (local, branch-independent)

Used to store **authoritative local expectations**.

Characteristics:

* Shared across all worktrees
* Not affected by branch switching
* Local to the repository
* Fast and reliable

Example (illustrative):

```ini
[gomcfg.dependency "github.com/example/foo"]
remote = origin
branch = dev
path = ../foo
```

`.git/config` is canonical for **local intent**.

---

#### B. Versioned JSON on a well-known branch (shared, auditable)

A JSON file stored on a designated, well-known branch (name TBD) mirrors the same dependency expectations.

Characteristics:

* Versioned and reviewable
* Shareable across clones
* Recoverable if `.git/config` is lost
* Human-readable

This file is the **canonical shared representation**.

Tooling is responsible for keeping the JSON and `.git/config` in sync and reporting divergence.

---

### 4. Worktree Compatibility

Dependency resolution is **explicitly directory-anchored**.

For each dependency:

* The tooling resolves the concrete filesystem path associated with the dependency.
* That path corresponds to exactly one Git worktree (if worktrees are used).
* Git is queried **only for that directory** to determine current branch state.

This design:

* Avoids reasoning about “any worktree” or “all worktrees”
* Avoids ambiguity
* Works equally well with or without worktrees

---

### 5. Validation Model

Validation is performed at runtime:

1. Load branch-independent dependency expectations.
2. For each dependency:

    * Resolve its directory path.
    * Query Git for the current branch at that path.
3. Compare actual branch vs expected branch.
4. If a mismatch is detected:

    * Emit a clear, actionable error.
    * Abort analysis.

No Git state is persisted by the tooling.

---

## Consequences

### Positive

* Fully compatible with Git worktrees
* No reliance on fragile branch state
* Clear separation of:

    * **Expectation** (stored)
    * **State** (observed live)
* Scales naturally to deep dependency graphs (“turtles all the way down”)
* Resilient to branch deletion or rebasing
* Simple and honest validation logic

### Trade-offs

* Requires explicit directory tracking for dependencies
* Requires tooling to manage sync between `.git/config` and versioned JSON
* Does not support declaring expectations “upward” (by design)

---

## Explicit Non-Goals

The following are **intentionally out of scope**:

* Persisting current checkout branch
* Enforcing a single branch per repository
* Enumerating or managing Git worktrees
* Allowing dependencies to declare expectations about their dependers
* Using Git notes or refs for metadata storage

---

## Summary

This decision establishes a **branch-independent, worktree-safe mechanism** for declaring and validating dependency branch expectations in Go module graphs.

By anchoring dependencies to directories and separating expectation from observed state, the design remains robust, minimal, and aligned with Git’s actual data model.

---
