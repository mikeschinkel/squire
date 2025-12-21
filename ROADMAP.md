# Squire Roadmap

This document tracks planned features and enhancements for Squire.

## Status Key

- 🔴 **Not Started** - Feature planned but not yet begun
- 🟡 **In Progress** - Feature currently being implemented
- 🟢 **Completed** - Feature implemented and merged
- 🟠 **On Hold** - Feature pending or in-progress, but on-hold.

---

## Planned Features

### External Module Dependencies (`--all` flag for `requires-tree`)

**Status:** 🔴 Not Started
**Priority:** Medium
**Related Command:** `squire requires-tree`

**Description:**

Add support for the `--all` flag to the `requires-tree` command to include external (non-Squire-managed) modules in the dependency tree visualization.

**Current Behavior:**

The tree currently only shows Squire-managed modules (those discovered via `DiscoverModules` from `.squire/config.json`). External dependencies are not included in the visualization.

**Desired Behavior:**

When `--all` is specified:
- Include external Go modules that are required by Squire-managed modules
- Show the complete dependency tree, not just internal modules
- Use `go list -m -json` or similar tooling to discover external module metadata
- Handle cases where external module information is unavailable gracefully

**Implementation Notes:**

1. Extend `retinue.ModuleSet` to support external modules
2. Add discovery logic to query external dependencies via Go toolchain
3. Consider depth limits to avoid overly deep trees
4. Handle network/proxy issues when resolving external modules
5. Optionally cache external module metadata for performance

**Acceptance Criteria:**

- `squire requires-tree . --all` includes both internal and external modules
- External modules are visually distinguishable (or documented as such)
- Command handles missing/unreachable external modules without crashing
- Performance is acceptable for typical Go projects (hundreds of dependencies)

**Related Documentation:**

- PRD: `docs/squire-cli-tree-command-prd.md` (Section 6.2)
- Phase 2 PRD: `docs/squire-phase-2-prd.md`

---

### API Stability Management

**Status:** 🔴 Not Started
**Priority:** High
**Related ADRs:** `go-dt/adrs/adr-2025-12-20-stability-levels.md`, `go-doterr/adrs/adr-2025-12-20-error-sentinel-strategy.md`

**Description:**

Implement cross-repository stability management features to orchestrate API stability contracts across all Squire-managed Go packages. This includes automated changelog generation from Contract: annotations, cross-repo stability validation, and coordinated deprecation management.

**Goals:**

1. **Automated Changelog Generation** - Extract Contract: annotations from code and generate CHANGELOG entries
2. **Cross-Repo Orchestration** - Validate stability across all dependent modules
3. **Deprecation Tracking** - Track and coordinate RemoveAfter dates across packages
4. **Breaking Change Coordination** - Warn if changes in one package affect consumers

#### Feature: Changelog Generation from Contract: Annotations

**Description:**

Automatically generate CHANGELOG.md entries by parsing Contract: annotations from Go source code across all Squire-managed modules.

**Desired Behavior:**

```bash
# Generate changelog for specific version
squire changelog generate v1.7.0

# Preview changelog without writing
squire changelog preview v1.7.0

# Update existing changelog with new entries
squire changelog update --since v1.6.0

# Generate changelog across all modules
squire changelog generate-all v1.7.0
```

**Generated Output Example:**

```markdown
## v1.7.0 (2024-07-15)

### Added
- `ErrNotFound` - General not found error (stable)
- `ProcessAsync` - Asynchronous processing (experimental)

### Changed
- `RetryConfig` - Added exponential backoff options (evolving)

### Deprecated
- `ErrRecordNotFound` - Use `ErrNotFound` instead
  - Removal date: No earlier than 2026-01-15 (18 month notice)
  - Target version: v2.0.0

### Fixed
- Configuration validation now handles edge cases (stable)

### Internal
- `testHelper` - Updated test infrastructure (internal)
```

**Implementation Notes:**

1. **Parse Contract: Annotations**
   - Scan Go files for Contract: comment blocks
   - Extract: Stability, Since, RemoveAfter, UseInstead, Note
   - Group by stability level

2. **Classify Changes**
   - New symbols with current version in Since field → "Added"
   - Changed symbols (signature/type changes) → "Changed"
   - Stability: deprecated → "Deprecated"
   - Stability: obsolete → "Deprecated" (but note won't be removed)
   - Bug fixes (from commit messages or annotations) → "Fixed"
   - Stability: internal → "Internal" section

3. **Format Generation**
   - Follow Keep a Changelog format
   - Include removal dates and target versions for deprecated items
   - Note stability levels in parentheses
   - Link to migration guides

4. **Integration Points**
   - Use git diff to detect changes between versions
   - Parse commit messages for context
   - Cross-reference with go-nextver breaking change detection
   - Support semver-based section ordering

**Acceptance Criteria:**

- `squire changelog generate` creates valid CHANGELOG.md entries
- All Contract: annotations are captured
- Deprecated items show removal dates and alternatives
- Changelog follows Keep a Changelog format
- Works across all Squire-managed modules
- Can preview without writing
- Can update existing changelog

#### Feature: Contract Enforcement Tooling

**Description:**

Automated validation and enforcement of Contract: annotations in Go source code.

**Planned Capabilities:**

- Extract Contract: blocks from Go source files into JSON index
- Validate that Deprecated items include RemoveAfter and Replacement metadata
- Use golang.org/x/exp/apidiff to detect breaking changes between versions
- Fail CI builds when Stable contracts are broken without major version bump
- Cross-module validation to ensure coordinated deprecations

**Integration with doterr:**

- Detect MsgErr usage and suggest sentinel promotion
- Validate error sentinel naming against ADR conventions
- Track error sentinel stability across packages

**Implementation Notes:**

1. **Contract Extraction**
   - Parse Go source files for Contract: comment blocks
   - Extract: Stability, Since, RemoveAfter, UseInstead, Note fields
   - Generate JSON index for all contracts across modules

2. **Validation Logic**
   - Check Deprecated items have RemoveAfter and Replacement
   - Verify time-based constraints are met before breaking changes
   - Ensure stability level transitions follow policy

3. **Breaking Change Detection**
   - Integrate with golang.org/x/exp/apidiff
   - Compare current code against previous tagged version
   - Flag violations of stability contracts

4. **Error Sentinel Analysis (doterr-specific)**
   - Scan for MsgErr() usage patterns
   - Identify frequently-used ad-hoc error messages
   - Suggest promotion to sentinel errors
   - Validate sentinel names follow ADR naming conventions

**Acceptance Criteria:**

- Extracts Contract: blocks from all Go files
- Validates deprecation metadata completeness
- Detects breaking changes in Stable symbols
- Provides actionable error messages for violations
- Integrates with CI/CD pipelines
- Supports doterr-specific analysis

#### Feature: Cross-Repo Stability Validation

**Description:**

Validate API stability across all dependent Squire-managed modules to ensure breaking changes are coordinated.

**Desired Behavior:**

```bash
# Check stability across all modules
squire stability check

# Check if specific module has breaking changes affecting others
squire stability check --module=go-doterr

# Show removal dates across all modules
squire stability removal-dates

# Check if ready to remove deprecated items
squire stability ready-to-remove
```

**Example Output:**

```
Stability Status Across Modules:

go-dt:
  ✅ No stability violations
  📅 1 item ready for removal (RemoveAfter passed)
     - ErrOldName (RemoveAfter: 2024-01-01, passed 530 days ago)

go-doterr:
  ⚠️  2 items with upcoming removals
     - ErrRecordNotFound (RemoveAfter: 2025-07-01, 197 days remaining)
     - ErrBadInput (RemoveAfter: 2025-09-15, 273 days remaining)

  ❌ 1 breaking change detected
     - ErrUserNotFound changed before 6 month notice (evolving)
     - Since: v1.5.0 (2024-05-01), only 4 months elapsed

go-cliutil:
  ⚠️  Depends on go-dt.ErrOldName (deprecated, will be removed)
  💡 Update to use go-dt.ErrNotFound before v2.0.0

Summary:
- 3 modules checked
- 1 breaking change
- 2 upcoming removals
- 1 dependent package needs updating
```

**Implementation Notes:**

1. **Module Discovery**
   - Use Squire's existing module discovery
   - Build dependency graph
   - Identify cross-module dependencies

2. **Stability Analysis**
   - Parse Contract: annotations from all modules
   - Run go-nextver stability checks per module
   - Aggregate results across modules
   - Track cross-module symbol usage

3. **Dependency Impact Analysis**
   - Detect when Module A uses deprecated symbol from Module B
   - Calculate removal timeline impact
   - Warn about cascading breaking changes

4. **Coordination Logic**
   - Ensure RemoveAfter dates are coordinated across dependencies
   - Warn if Module B removes symbol before Module A updates
   - Suggest migration order

**Acceptance Criteria:**

- Validates stability across all Squire-managed modules
- Detects cross-module dependencies on deprecated symbols
- Warns about uncoordinated RemoveAfter dates
- Provides actionable migration guidance
- Integrates with go-nextver for per-module checks

#### Feature: RemoveAfter Date Coordination

**Description:**

Coordinate RemoveAfter dates across packages to ensure dependencies have time to migrate before symbols are removed.

**Desired Behavior:**

```bash
# Show all RemoveAfter dates in dependency order
squire stability timeline

# Check if RemoveAfter dates are safe
squire stability validate-timeline

# Suggest RemoveAfter date for new deprecation
squire stability suggest-removal-date --symbol=ErrOldName --module=go-dt
```

**Example Output:**

```
Removal Timeline (Dependency-Safe Order):

2025-07-01:
  ✅ go-dt.ErrOldName (no dependents)

2025-09-15:
  ⚠️  go-doterr.ErrBadInput
     Depends on: go-dt.ErrInvalid (stable, no removal planned)
     ✅ Safe to remove

2026-01-15:
  ❌ go-doterr.ErrRecordNotFound
     Used by: go-cliutil (3 occurrences)
     💡 go-cliutil should migrate to go-doterr.ErrNotFound
     💡 Recommend: Extend RemoveAfter to 2026-06-15 (give consumers 6 months)
```

**Implementation Notes:**

1. **Timeline Visualization**
   - Collect all RemoveAfter dates
   - Sort by date
   - Group by module
   - Show dependency relationships

2. **Dependency Checking**
   - Parse imports across modules
   - Track symbol usage
   - Identify consumers of deprecated symbols

3. **Safety Validation**
   - Ensure removing package has no dependents still using symbol
   - Or dependents have deprecated their usage with earlier RemoveAfter
   - Or dependents have migrated away

4. **Suggestion Logic**
   - Calculate minimum safe RemoveAfter based on dependents
   - Add buffer time for consumer migration
   - Respect stability level minimums (18mo for stable, 6mo for evolving)

**Acceptance Criteria:**

- Shows removal timeline in dependency order
- Detects unsafe removal dates
- Suggests safe RemoveAfter dates
- Accounts for transitive dependencies
- Provides migration guidance

#### Feature: Breaking Change Reports

**Description:**

Generate comprehensive reports of breaking changes across all modules, with impact analysis.

**Desired Behavior:**

```bash
# Report breaking changes between versions
squire breaking-changes --from=v1.6.0 --to=v1.7.0

# Report across all modules
squire breaking-changes --from=v1.6.0 --to=v1.7.0 --all-modules

# Check if version bump is safe
squire version-check v1.7.0

# Suggest next version based on changes
squire suggest-version
```

**Example Output:**

```
Breaking Changes Analysis: v1.6.0 → v1.7.0

Module: go-doterr
  ❌ 2 BREAKING changes detected

  1. ErrUserNotFound removed
     - Stability: deprecated
     - RemoveAfter: 2026-01-01 (199 days remaining)
     - ❌ Cannot remove yet
     - Impact: Breaking for any consumer checking errors.Is()

  2. ProcessData signature changed
     - Stability: stable
     - Since: v1.5.0 (2024-05-01)
     - Only 14 months elapsed (minimum: 18 months)
     - ❌ Changed too soon
     - Impact: All callers must update

Module: go-dt
  ✅ No breaking changes
  ✅ 2 new symbols added (backward compatible)

Cross-Module Impact:
  ⚠️  go-cliutil uses go-doterr.ProcessData
     Must be updated if go-doterr changes are released

Semver Recommendation:
  ❌ Cannot release as v1.7.0 (breaking changes present)
  💡 Either:
     1. Fix breaking changes, then release as v1.7.0
     2. Release as v2.0.0 (but RemoveAfter dates not yet reached)
  💡 Recommended: Fix issues, release as v1.7.0
```

**Implementation Notes:**

1. **Per-Module Analysis**
   - Use go-nextver for each module
   - Collect breaking changes
   - Categorize by severity

2. **Cross-Module Impact**
   - Track symbol usage across modules
   - Identify affected dependents
   - Calculate blast radius

3. **Semver Recommendation**
   - Analyze changes (breaking vs compatible)
   - Check time-based constraints
   - Suggest appropriate version bump
   - Flag if version number doesn't match changes

4. **Integration with go-nextver**
   - Aggregate per-module results
   - Add cross-module analysis layer
   - Provide unified report

**Acceptance Criteria:**

- Detects breaking changes across all modules
- Analyzes cross-module impact
- Provides semver recommendations
- Flags time-based constraint violations
- Suggests fixes or alternative versions

#### Integration Points

**With go-nextver:**
- Use go-nextver for per-package stability validation
- Aggregate results across all Squire-managed modules
- Add cross-module dependency analysis layer

**With Git:**
- Parse commit messages for changelog context
- Track changes between versions
- Coordinate tagging across modules

**With Module Discovery:**
- Leverage existing `DiscoverModules` functionality
- Use dependency graph from `requires-tree`
- Track module relationships

#### Future Enhancements

1. **Interactive Migration Planning**
   - TUI for planning deprecations across modules
   - Preview timeline changes
   - Simulate impact of RemoveAfter adjustments

2. **Automated PRs**
   - Generate PR descriptions with stability info
   - Auto-update changelogs on merge
   - Create follow-up issues for deprecations

3. **Stability Dashboard**
   - Web UI showing stability status
   - Upcoming removal calendar
   - Migration progress tracking

4. **Release Coordination**
   - Suggest release order based on dependencies
   - Coordinate version bumps across modules
   - Automate tagging workflow

**Related Documentation:**

- `go-dt/adrs/adr-2025-12-20-stability-levels.md` - General stability levels
- `go-doterr/adrs/adr-2025-12-20-error-sentinel-strategy.md` - Error-specific stability
- `go-nextver/PLAN.md` - Stability compliance checks
- `docs/squire-phase-2-prd.md` - Multi-repo orchestration

---

### `tree` command

1. `TREE_PLAN.md` contains in-flight ideas that are currently on-hold but will be resumed at some point.

## Completed Features

### Module Discovery & Dependency Ordering

**Status:** 🟢 Completed
**Completed:** Phase 2

- Module discovery from `.squire/config.json`
- Dependency-safe ordering via topological sort
- Module classification (lib/cmd/test)
- Versioned vs non-versioned heuristics

### Basic Dependency Tree Visualization

**Status:** 🟢 Completed
**Completed:** 2025-12-11

- ASCII tree rendering of internal module dependencies
- `--show-dirs` flag for directory-based labels
- `--show-all` flag for module path + location
- Markdown embedding with `--embed`, `--before`, `--after` flags
- Flag validation for mutual exclusivity

---

## Future Considerations

These are ideas for future exploration, not yet committed to the roadmap:

- **Release Planning & Tagging** - Automated versioning and git tag management
- **Changelog Generation** - Automatic changelog creation from commit history
- **Multi-Repo Orchestration** - Commands that operate across multiple related repositories
- **ClearPath Linter** - Custom linting rules for ClearPath coding style
- **GitHub Workflow Management** - Ensure standard workflows across all repos
- **GoReleaser Integration** - Scaffolding and automation for compiled binaries
- **TUI Mode** - Interactive terminal UI for complex operations
