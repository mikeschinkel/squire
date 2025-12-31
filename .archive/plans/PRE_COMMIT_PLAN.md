# PRE-commit Analysis for Gomion Next - Comprehensive Plan

## Problem Statement

The `gomion next` workflow needs intelligent commit assistance with API change analysis, but there's a fundamental chicken-and-egg problem:
- POST-commit tools (go-nextver, retinue) require clean repos to analyze
- We need the analysis BEFORE creating the commit
- Users need to understand impact of staged changes to write appropriate commit messages

**Solution:** Analyze staged changes (not committed) using:
1. **Staged file export** - Extract staged content to temp directory (NEW - doesn't exist anywhere)
2. **CachedWorktree pattern** - Safely checkout baseline tags without touching user's repo (reuse from go-nextver)
3. **Domain-based analysis functions** - API, AST, and test signal analysis in goutils package
4. **Template-driven prompts** - Externalized, customizable AI prompts
5. **Multi-commit assistance** - Help users split changes into logical commits

## Architecture Overview

### Package Responsibilities

**New Packages:**
- `gompkg/gitutils` - Git operations (extract from next_cmd.go)
  - CachedWorktree management (adapted from go-nextver pattern)
  - Staged file export (NEW implementation)
  - Repository queries (tags, branches, status)
  - Filesystem locking for concurrent access

- `gompkg/gomionscliui` - Terminal UI/display logic (extract from next_cmd.go)
  - Interactive menus
  - Progress indicators
  - Formatted output (tables, boxes, summaries)
  - User input handling

- `gompkg/precommit` - PRE-commit analysis orchestration
  - Direct calls to analysis functions (no pluggable interface for execution)
  - Result aggregation
  - Cache management
  - Analysis persistence (for exit/reenter workflows)
  - Common types: AnalysisResult interface, OutputFormat, Verdict

**Renamed/Broadened Packages:**
- `gompkg/goutils` - Go language utilities (rename/broaden from gomodutils)
  - go.mod operations (existing gomodutils functionality)
  - API compatibility analysis (wraps apidiffr)
  - AST-level code analysis
  - Test signal detection (new tests, changed tests, removed tests)
  - Cohesive domain: all Go language tooling

**Enhanced Existing Packages:**
- `gompkg/commitmsg` - Commit message generation (already exists)
  - Template-driven prompt construction
  - Integration with analysis results
  - Multi-format output support

- `gompkg/askai` - Generic AI interaction (already exists)
  - Provider abstraction
  - Timeout/retry logic
  - No domain coupling

**Keep Separate (No Changes):**
- `gompkg/apidiffr` - API compatibility engine (independent library, wrapped by goutils)
- `gompkg/retinue` - POST-commit version tagging (different use case)

### Core Workflow

```
User stages changes
    ↓
gomion next [module]
    ↓
next_cmd.go (orchestration only)
    ├─> gitutils.GetStagedFiles()
    ├─> gitutils.FindBaselineTag()
    ├─> gitutils.OpenCachedWorktree(baselineTag)
    ├─> gitutils.ExportStagedFiles(→ tempDir)
    ├─> precommit.Analyze(baselineDir, stagedDir)
    │   ├─> goutils.AnalyzeAPICompatibility() → APICompatResult
    │   ├─> goutils.AnalyzeASTDiff() → ASTDiffResult
    │   └─> goutils.AnalyzeTestSignals() → TestSignalsResult
    ├─> precommit.PersistResults(cacheFile)
    ├─> Format results for display using AnalysisResult.AnalysisSummary(ANSIEscapedFormat)
    ├─> gomionscliui.DisplayVerdictSummary()
    ├─> Format results for AI using AnalysisResult.AnalysisSummary(MarkdownFormat)
    ├─> commitmsg.BuildPrompt(template, analyzerResults)
    ├─> askai.Agent.Ask(prompt)
    ├─> gomionscliui.ShowCommitMessageMenu()
    │   ├─> [v]iew full report → gomionscliui.DisplayFullReport()
    │   ├─> [s]plit commits → precommit.SuggestGroupings()
    │   ├─> [e]dit → editor
    │   ├─> [r]egenerate → loop
    │   └─> [y]es commit → git commit
    └─> Cleanup temp directories
```

## Phase 1: Extract & Refactor Existing Code

### 1.1 Create gitutils Package

**File:** `gompkg/gitutils/repo.go`

Extract from next_cmd.go:
- Git command execution helpers
- Repository root finding
- Branch/tag queries
- Staged file listing

**New types:**
```go
type Repo struct {
    Dir     dt.DirPath  // Repo root (may differ from module dir)
    ModDir  dt.DirPath  // Module directory within repo
}

func OpenRepo(moduleDir dt.DirPath) (*Repo, error)
func (r *Repo) GetStagedFiles() ([]dt.RelFilepath, error)
func (r *Repo) FindBaselineTag() (string, error)
func (r *Repo) CurrentBranch() (string, error)
```

**File:** `gompkg/gitutils/cached_worktree.go`

Adapt from go-nextver's pattern:
```go
type CachedWorktree struct {
    Dir     dt.DirPath
    repoDir dt.DirPath
    release func() error  // Lock cleanup
}

func (r *Repo) OpenCachedWorktree(ctx context.Context) (*CachedWorktree, error)
func (c *CachedWorktree) Checkout(ref string) error
func (c *CachedWorktree) Close() error
```

**Implementation details:**
- Cache location: `~/.cache/gomion/worktrees/<repo-hash>/`
- Filesystem locking: `~/.cache/gomion/locks/<repo-hash>.lock`
- Clone once, reuse across invocations
- Fetch latest before each checkout

**File:** `gompkg/gitutils/export_staged.go` (NEW)

This functionality does NOT exist anywhere - must implement:

```go
type ExportStagedArgs struct {
    Repo       *Repo
    DestDir    dt.DirPath
    StagedFiles []dt.RelFilepath  // From GetStagedFiles()
}

func ExportStagedFiles(ctx context.Context, args ExportStagedArgs) error
```

**Implementation:**
1. For each staged file: `git show :path/to/file`
   - Note: `:path` is git index syntax (`:` prefix means staged version)
2. Write output to `destDir/path/to/file` (preserve structure)
3. Handle deleted files gracefully (skip - they won't exist in staged)
4. Handle binary files (copy bytes as-is)
5. Filter to only module directory files (ignore nested modules)

**Critical edge cases:**
- Partial staging: `git show :path` returns staged version (correct)
- Deleted files: Command fails, skip and continue
- New files: Works correctly (they're in index)
- Renamed files: Appears as delete + add (both handled correctly)

**File:** `gompkg/gitutils/doterr.go`

Drop in from another package - NEVER edit.

### 1.2 Create gomionscliui Package

**File:** `gompkg/gomionscliui/menu.go`

Extract from next_cmd.go:
```go
type MenuOption struct {
    Key         rune
    Label       string
    Description string
}

type MenuArgs struct {
    Prompt  string
    Options []MenuOption
    Writer  io.Writer
}

func ShowMenu(args MenuArgs) (rune, error)
func ShowMenuWithNumbers(args MenuArgs) (int, error)  // Digit-based, not letters
```

**File:** `gompkg/gomionscliui/display.go`

Extract display formatting from next_cmd.go:
```go
func DisplayBox(title string, content string, w io.Writer)
func DisplayTable(headers []string, rows [][]string, w io.Writer)
func DisplayVerdictSummary(result precommit.AnalysisResult, w io.Writer)
func DisplayFullReport(result precommit.AnalysisResult, w io.Writer)
func DisplayProgress(message string, w io.Writer)
```

**File:** `gompkg/gomionscliui/doterr.go`

Drop in from another package.

## Phase 2: Implement Analysis Functions in goutils

### 2.1 Common Types and Interface

**File:** `gompkg/precommit/types.go` (or `gompkg/goutils/types.go`)

```go
// OutputFormat specifies the output format for analysis summaries
type OutputFormat string

const (
    MarkdownFormat    OutputFormat = "markdown"      // For AI prompts
    TextFormat        OutputFormat = "text"          // Plain text (logs, files)
    ANSIEscapedFormat OutputFormat = "ansi_escaped"  // Terminal display (colors, bold, etc.)
)

// AnalysisResult is implemented by all analysis result types for formatting
type AnalysisResult interface {
    AnalysisSummary(format OutputFormat) string
}

// Verdict indicates compatibility assessment
type Verdict string

const (
    VerdictBreaking         Verdict = "breaking"
    VerdictLikelyCompatible Verdict = "likely-compatible"  // NEVER claim absolute "compatible"
    VerdictMaybeCompatible  Verdict = "maybe-compatible"
    VerdictUnknown          Verdict = "unknown"
    VerdictNoChanges        Verdict = "no-changes"
)
```

### 2.2 API Compatibility Analysis

**File:** `gompkg/goutils/api_compat.go`

Wraps existing apidiffr package with bespoke result type:

```go
// APIChange represents a single API change
type APIChange struct {
    Type        string  // "removed", "added", "modified"
    Entity      string  // "func", "method", "type", "field"
    Signature   string  // Full signature
    Description string  // Human-readable description
}

// APICompatResult contains API compatibility analysis results
type APICompatResult struct {
    Verdict         Verdict
    BaselineTag     string
    BreakingChanges []APIChange
    Additions       []APIChange
    Modifications   []APIChange
}

// AnalysisSummary implements AnalysisResult interface
func (r APICompatResult) AnalysisSummary(format OutputFormat) string {
    switch format {
    case MarkdownFormat:
        return r.formatAsMarkdown()
    case ANSIEscapedFormat:
        return r.formatAsANSI()
    case TextFormat:
        return r.formatAsPlainText()
    default:
        return r.formatAsPlainText()
    }
}

// AnalyzeAPICompatibility analyzes API compatibility between baseline and staged code
func AnalyzeAPICompatibility(ctx context.Context, baseline, staged dt.DirPath) (APICompatResult, error) {
    var result APICompatResult
    var err error

    // Call apidiffr.Compare(baseline, staged)
    // Parse report
    // Categorize changes into breaking/additions/modifications
    // Determine verdict:
    //   - Breaking if any removals or incompatible changes
    //   - LikelyCompatible if only additions
    //   - MaybeCompatible if uncertain

    return result, err
}
```

**Verdict logic:**
- Breaking: Function/method removed, signature changed incompatibly, field removed
- Likely Compatible: Only additions (new funcs, new fields, new methods)
- Maybe Compatible: Changes we can't categorize definitively
- Unknown: No baseline to compare, or apidiffr failed

**Example markdown output:**
```markdown
## API Compatibility Analysis

**Baseline:** v1.2.3
**Verdict:** BREAKING

### Breaking Changes
- `func (Auth) Login() error` - REMOVED
- `type Credentials` - field `OldToken` REMOVED

### Non-Breaking Changes
- `func (Auth) LoginWithToken(token string) error` - ADDED
- `type Credentials` - field `NewToken` ADDED
```

### 2.3 AST Analysis

**File:** `gompkg/goutils/ast_diff.go`

Analyzes Go code changes at AST level with bespoke result type:

```go
// TypeChange represents a change to a type definition
type TypeChange struct {
    TypeName    string
    ChangeType  string  // "added", "removed", "modified"
    Description string
}

// FuncChange represents a change to a function
type FuncChange struct {
    FuncName    string
    ChangeType  string  // "added", "removed", "modified"
    Signature   string
    Description string
}

// ASTDiffResult contains AST-level analysis results
type ASTDiffResult struct {
    Verdict         Verdict
    TypeChanges     []TypeChange
    FuncChanges     []FuncChange
    DocChanges      []string  // Significant doc comment changes
    StructTagChanges []string  // Struct tag modifications
}

// AnalysisSummary implements AnalysisResult interface
func (r ASTDiffResult) AnalysisSummary(format OutputFormat) string {
    switch format {
    case MarkdownFormat:
        return r.formatAsMarkdown()
    case ANSIEscapedFormat:
        return r.formatAsANSI()
    case TextFormat:
        return r.formatAsPlainText()
    default:
        return r.formatAsPlainText()
    }
}

// AnalyzeASTDiff analyzes AST-level changes between baseline and staged code
func AnalyzeASTDiff(ctx context.Context, baseline, staged dt.DirPath) (ASTDiffResult, error) {
    var result ASTDiffResult
    var err error

    // Parse both directories using go/parser
    // Compare ASTs
    // Detect:
    //   - New types/functions (additions)
    //   - Removed types/functions (deletions)
    //   - Changed signatures
    //   - Changed struct tags
    //   - Changed doc comments

    return result, err
}
```

**Detection capabilities:**
1. **Type changes:** New types, removed types, field modifications
2. **Function changes:** New funcs, removed funcs, signature changes
3. **Interface changes:** New methods, removed methods
4. **Doc comment changes:** Helps AI understand intent
5. **Struct tag changes:** May affect serialization/validation

**Example markdown output:**
```markdown
## AST Analysis

### Type Changes
- `type User struct` - NEW FIELD: `Email string` (json tag: "email")
- `type Auth struct` - REMOVED FIELD: `LegacyToken string`

### Function Changes
- `func NewUser(name string, email string) *User` - NEW
- `func ValidateEmail(email string) error` - NEW

### Documentation Changes
- `type User` - Updated doc comment to mention email requirement
```

### 2.4 Test Signal Analysis

**File:** `gompkg/goutils/test_signals.go`

Detects test changes to inform commit messages with bespoke result type:

```go
// TestSignalsResult contains test change analysis results
type TestSignalsResult struct {
    Verdict        Verdict
    NewTests       []string  // Newly added test functions
    ModifiedTests  []string  // Changed test functions
    RemovedTests   []string  // Deleted test functions
    NewTestCount   int       // Count of new tests
    CoverageSignal string    // "good", "poor", "unknown"
}

// AnalysisSummary implements AnalysisResult interface
func (r TestSignalsResult) AnalysisSummary(format OutputFormat) string {
    switch format {
    case MarkdownFormat:
        return r.formatAsMarkdown()
    case ANSIEscapedFormat:
        return r.formatAsANSI()
    case TextFormat:
        return r.formatAsPlainText()
    default:
        return r.formatAsPlainText()
    }
}

// AnalyzeTestSignals detects test changes between baseline and staged code
func AnalyzeTestSignals(ctx context.Context, baseline, staged dt.DirPath) (TestSignalsResult, error) {
    var result TestSignalsResult
    var err error

    // Find *_test.go files in both dirs
    // Detect:
    //   - New test functions
    //   - Removed test functions
    //   - Changed test names/structure
    // Signals:
    //   - Tests added for new code → good coverage
    //   - Tests removed → breaking change likely
    //   - Tests modified → behavior change

    return result, err
}
```

**Example markdown output:**
```markdown
## Test Analysis

### New Tests
- `TestUserEmailValidation` - NEW test for email validation
- `TestNewUserWithEmail` - NEW test for constructor

### Modified Tests
- `TestUserCreation` - Updated to include email parameter

### Coverage Signals
- New functionality is well-tested (2 new tests added)
```

## Phase 3: PRE-commit Analysis Orchestration

### 3.1 Analysis Coordinator

**File:** `gompkg/precommit/analyze.go`

```go
type Results struct {
    Timestamp      time.Time
    BaselineTag    string
    ModulePath     string
    OverallVerdict Verdict

    // Individual analysis results (bespoke types)
    API   goutils.APICompatResult
    AST   goutils.ASTDiffResult
    Tests goutils.TestSignalsResult
}

type AnalyzeArgs struct {
    ModuleDir dt.DirPath
    CacheKey  string  // For persistence
}

func Analyze(ctx context.Context, args AnalyzeArgs) (Results, error) {
    var result Results
    var repo *gitutils.Repo
    var cachedWT *gitutils.CachedWorktree
    var stagedDir dt.DirPath
    var err error

    // Open repo
    repo, err = gitutils.OpenRepo(args.ModuleDir)
    if err != nil {
        goto end
    }

    // Find baseline
    result.BaselineTag, err = repo.FindBaselineTag()
    if err != nil {
        // Not fatal - set verdict to unknown
        result.OverallVerdict = VerdictUnknown
        goto end
    }

    // Open cached worktree and checkout baseline
    cachedWT, err = repo.OpenCachedWorktree(ctx)
    if err != nil {
        goto end
    }
    defer cachedWT.Close()

    err = cachedWT.Checkout(result.BaselineTag)
    if err != nil {
        goto end
    }

    // Export staged files to temp directory
    stagedDir, err = createTempStagedDir()
    if err != nil {
        goto end
    }
    defer os.RemoveAll(stagedDir.String())

    err = gitutils.ExportStagedFiles(ctx, gitutils.ExportStagedArgs{
        Repo:    repo,
        DestDir: stagedDir,
    })
    if err != nil {
        goto end
    }

    // Call each analysis function directly (bespoke handling)
    result.API, err = goutils.AnalyzeAPICompatibility(ctx, cachedWT.Dir, stagedDir)
    if err != nil {
        // Log but continue
    }

    result.AST, err = goutils.AnalyzeASTDiff(ctx, cachedWT.Dir, stagedDir)
    if err != nil {
        // Log but continue
    }

    result.Tests, err = goutils.AnalyzeTestSignals(ctx, cachedWT.Dir, stagedDir)
    if err != nil {
        // Log but continue
    }

    // Compute overall verdict (bespoke logic based on specific fields)
    result.OverallVerdict = computeOverallVerdict(&result)

    // Persist for exit/reenter workflows
    err = persistResult(&result, args.CacheKey)

end:
    return result, err
}

// computeOverallVerdict uses bespoke logic to determine overall verdict
func computeOverallVerdict(r *Results) Verdict {
    // Breaking takes precedence
    if r.API.Verdict == VerdictBreaking {
        return VerdictBreaking
    }
    if r.AST.Verdict == VerdictBreaking {
        return VerdictBreaking
    }

    // If all are likely compatible
    if r.API.Verdict == VerdictLikelyCompatible &&
       r.AST.Verdict == VerdictLikelyCompatible {
        return VerdictLikelyCompatible
    }

    // Default to maybe compatible
    return VerdictMaybeCompatible
}

// FormatForAI generates markdown for AI prompts (generic formatting)
func (r Results) FormatForAI() string {
    var combined string

    // Use AnalysisResult interface for generic iteration
    for _, analyzer := range []AnalysisResult{r.API, r.AST, r.Tests} {
        combined += analyzer.AnalysisSummary(MarkdownFormat)
        combined += "\n\n"
    }

    return combined
}

// FormatForTerminal generates ANSI-escaped output for terminal (generic formatting)
func (r Results) FormatForTerminal() string {
    var combined string

    for _, analyzer := range []AnalysisResult{r.API, r.AST, r.Tests} {
        combined += analyzer.AnalysisSummary(ANSIEscapedFormat)
        combined += "\n\n"
    }

    return combined
}
```

**File:** `gompkg/precommit/persist.go`

```go
func persistResult(result *Results, cacheKey string) error
func loadPersistedResult(cacheKey string) (*Results, error)
func clearPersistedResult(cacheKey string) error
```

Cache location: `~/.cache/gomion/analysis/<module-hash>-<staged-hash>.json`

Purpose: Allow user to exit `gomion next`, do other work, return later and resume with same analysis.

### 3.2 Commit Grouping Suggestions

**File:** `gompkg/precommit/grouping.go`

```go
type CommitGroup struct {
    Title       string
    Files       []dt.RelFilepath
    Rationale   string
    Suggested   bool  // AI suggested vs user-defined
}

type GroupingArgs struct {
    StagedFiles []dt.RelFilepath
    Analysis    Results
    AIAgent     *askai.Agent
}

func SuggestGroupings(ctx context.Context, args GroupingArgs) ([]CommitGroup, error) {
    // Build AI prompt with:
    //   - List of staged files
    //   - Analysis results
    //   - Request: "Suggest logical commit groupings"
    // Parse AI response into CommitGroup structs
    // Return suggestions
}

func InteractiveRestage(groups []CommitGroup, repo *gitutils.Repo) error {
    // For each group:
    //   1. Show files in group
    //   2. Ask user: [a]ccept, [m]odify, [s]kip
    //   3. If accept: unstage all, restage this group, commit
    //   4. If modify: interactive file selection
    //   5. If skip: move to next group
    // After all groups, show remaining unstaged files
}
```

## Phase 4: Template-Driven Prompts

### 4.1 Prompt Templates

**File:** `gompkg/commitmsg/templates/default.tmpl`

```
Generate a git commit message for these staged changes.

{{- if .ConventionalCommits }}
Use conventional commit format (type: subject).
{{- end }}

{{- if .MaxSubjectChars }}
Keep subject line under {{ .MaxSubjectChars }} characters.
{{- end }}

Add a blank line and bullet points in the body if needed.
Output ONLY the commit message, no explanations or markdown formatting.

Branch: {{ .Branch }}

{{- if .AnalysisResult }}

--- API ANALYSIS ---
{{ .AnalysisResult.SummaryMarkdown }}
{{- end }}

--- STAGED DIFF ---
{{ .StagedDiff }}
```

**File:** `gompkg/commitmsg/templates/breaking.tmpl`

```
Generate a git commit message for these BREAKING CHANGES.

Use conventional commits with BREAKING CHANGE footer.
Subject should start with type and include "!" for breaking (e.g., "feat!:" or "refactor!:").

{{- if .AnalysisResult }}

The following breaking changes were detected:
{{ .AnalysisResult.API.Markdown }}
{{- end }}

Explain the breaking changes in the commit body.
Include migration guidance if applicable.

--- STAGED DIFF ---
{{ .StagedDiff }}
```

### 4.2 Template Rendering

**File:** `gompkg/commitmsg/generator.go` (update existing)

```go
import "text/template"

type PromptData struct {
    ConventionalCommits bool
    MaxSubjectChars     int
    Branch              string
    StagedDiff          string
    AnalysisResult      *precommit.AnalysisResult
}

func BuildPrompt(req Request, analysis *precommit.AnalysisResult) (string, error) {
    // Select template based on verdict
    var tmplPath string
    if analysis != nil && analysis.OverallVerdict == common.VerdictBreaking {
        tmplPath = "templates/breaking.tmpl"
    } else {
        tmplPath = "templates/default.tmpl"
    }

    // Load and parse template
    tmpl, err := template.ParseFiles(tmplPath)
    if err != nil {
        return "", err
    }

    // Render with data
    data := PromptData{
        ConventionalCommits: req.ConventionalCommits,
        MaxSubjectChars:     req.MaxSubjectChars,
        Branch:              req.Branch,
        StagedDiff:          req.StagedDiff,
        AnalysisResult:      analysis,
    }

    var buf bytes.Buffer
    err = tmpl.Execute(&buf, data)
    return buf.String(), err
}
```

**Template customization:**
- User can override templates in `~/.config/gomion/templates/`
- Project can override in `.gomion/templates/`
- Hierarchy: project > user > built-in

## Phase 5: Integration with next_cmd.go

### 5.1 Refactored next_cmd.go

The command file should be THIN - only orchestration:

```go
func (c *NextCmd) generateCommitMessage(moduleDir dt.DirPath) error {
    var results precommit.Results
    var message string
    var err error

    // Run analysis (calls goutils functions directly)
    results, err = precommit.Analyze(ctx, precommit.AnalyzeArgs{
        ModuleDir: moduleDir,
        CacheKey:  computeCacheKey(moduleDir),
    })
    if err != nil {
        // Log warning but continue
        c.Logger.Warn("Analysis failed", "error", err)
    }

    // Display verdict summary using ANSI-formatted output
    summary := results.FormatForTerminal()
    c.Writer.Write([]byte(summary))

    // Generate commit message with AI using markdown-formatted results
    message, err = c.callClaudeForCommitMsg(moduleDir, &results)
    if err != nil {
        goto end
    }

    // Interactive menu
    err = c.handleCommitMessageActions(moduleDir, message, &results)

end:
    return err
}

func (c *NextCmd) handleCommitMessageActions(
    moduleDir dt.DirPath,
    message string,
    results *precommit.Results,
) error {
    for {
        choice, err := gomionscliui.ShowMenu(gomionscliui.MenuArgs{
            Prompt: "What would you like to do?",
            Options: []gomionscliui.MenuOption{
                {Key: 'y', Label: "Use this commit message"},
                {Key: 'e', Label: "Edit in editor"},
                {Key: 'r', Label: "Regenerate with AI"},
                {Key: 'v', Label: "View full analysis report"},
                {Key: 's', Label: "Split into multiple commits"},
                {Key: 'b', Label: "Back to main menu"},
            },
            Writer: c.Writer,
        })

        switch choice {
        case 'y':
            return gitCommit(moduleDir, message)
        case 'e':
            message = editInEditor(message)
        case 'r':
            message, _ = c.callClaudeForCommitMsg(moduleDir, results)
        case 'v':
            gomionscliui.DisplayFullReport(*results, c.Writer)
        case 's':
            return c.handleMultiCommitFlow(moduleDir, results)
        case 'b':
            return nil
        }
    }
}

func (c *NextCmd) handleMultiCommitFlow(
    moduleDir dt.DirPath,
    results *precommit.Results,
) error {
    // Get AI-suggested groupings
    repo, _ := gitutils.OpenRepo(moduleDir)
    stagedFiles, _ := repo.GetStagedFiles()

    groups, err := precommit.SuggestGroupings(ctx, precommit.GroupingArgs{
        StagedFiles: stagedFiles,
        Analysis:    *results,
        AIAgent:     c.aiAgent,
    })

    // Interactive restaging
    return precommit.InteractiveRestage(groups, repo)
}
```

All the git logic, display logic, and analysis logic is now in dedicated packages.

## Implementation Sequence

### Phase 1: Foundation (1-2 days)
1. Create gitutils package structure
2. Extract git operations from next_cmd.go to gitutils
3. Implement CachedWorktree (adapt from go-nextver)
4. Implement ExportStagedFiles (NEW)
5. Unit tests for gitutils

### Phase 2: UI Extraction (1 day)
1. Create gomionscliui package
2. Extract display logic from next_cmd.go
3. Implement menu system with numbers (not letters)
4. Unit tests for display functions

### Phase 3: Analysis Functions in goutils (1-2 days)
1. Rename gomodutils to goutils (or keep name and broaden scope)
2. Create common types (AnalysisResult interface, OutputFormat, Verdict)
3. Implement AnalyzeAPICompatibility() with APICompatResult
4. Implement AnalyzeASTDiff() with ASTDiffResult
5. Implement AnalyzeTestSignals() with TestSignalsResult
6. Implement AnalysisSummary() formatters for each result type
7. Unit tests for each analysis function

### Phase 4: Analysis Orchestration (1 day)
1. Create precommit package
2. Implement Analyze() coordinator (direct function calls, no pluggable interface)
3. Implement FormatForAI() and FormatForTerminal() methods
4. Implement result persistence
5. Unit tests for orchestration

### Phase 5: Templates (1 day)
1. Create template files
2. Implement template rendering in commitmsg
3. Support template customization hierarchy
4. Test template rendering

### Phase 6: Multi-Commit Flow (1-2 days)
1. Implement commit grouping suggestions
2. Implement interactive restaging
3. Integration with AI for grouping suggestions
4. End-to-end testing

### Phase 7: Integration (1 day)
1. Refactor next_cmd.go to use new packages
2. Wire up all components
3. End-to-end testing with real repos
4. Polish UX based on testing

### Phase 8: Testing & Polish (1 day)
1. Integration tests with go-dt, go-cliutil, gomion
2. Test with breaking changes, compatible changes, no baseline
3. Test multi-commit workflows
4. Test template customization
5. Documentation updates

**Total Estimated Time:** 8-10 days

## Critical Files

### New Files (Create)
```
gompkg/gitutils/
  ├── repo.go                    - Repository operations
  ├── cached_worktree.go         - Safe baseline checkouts (from go-nextver pattern)
  ├── export_staged.go           - NEW staged file export (git show :path)
  ├── lock.go                    - Filesystem locking
  └── doterr.go                  - Error handling (drop-in)

gompkg/gomionscliui/
  ├── menu.go                    - Interactive menus (number-based)
  ├── display.go                 - Formatted output
  └── doterr.go                  - Error handling (drop-in)

gompkg/goutils/               - Rename/broaden from gomodutils
  ├── mod_*.go                   - Existing go.mod operations (from gomodutils)
  ├── types.go                   - AnalysisResult interface, OutputFormat, Verdict
  ├── api_compat.go              - API compatibility analysis (wraps apidiffr)
  ├── ast_diff.go                - AST-level code analysis
  ├── test_signals.go            - Test signal detection
  └── doterr.go                  - Error handling (drop-in)

gompkg/precommit/
  ├── types.go                   - Results struct, common types
  ├── analyze.go                 - Analysis coordinator (direct function calls)
  ├── persist.go                 - Result persistence
  ├── grouping.go                - Commit grouping suggestions
  └── doterr.go                  - Error handling (drop-in)

gompkg/commitmsg/templates/
  ├── default.tmpl               - Standard commit message template
  ├── breaking.tmpl              - Breaking change template
  └── grouping.tmpl              - Multi-commit template
```

### Modified Files
```
gompkg/gomodutils/ → gompkg/goutils/
  - Rename package (or keep name but broaden scope)
  - Add analysis functions: api_compat.go, ast_diff.go, test_signals.go
  - Add types.go for AnalysisResult interface and OutputFormat
  - Existing mod_*.go files stay (go.mod operations)

gompkg/gomioncmds/next_cmd.go
  - Extract git operations → gitutils
  - Extract display logic → gomionscliui
  - Call precommit.Analyze() directly (no analyzer list)
  - Use Results.FormatForAI() and Results.FormatForTerminal()
  - Keep only orchestration
  - Add multi-commit flow handling

gompkg/commitmsg/generator.go
  - Add template rendering
  - Support precommit.Results parameter
  - Support template customization
```

### Unchanged Files (Reference Only)
```
gompkg/apidiffr/              - Independent library (wrapped by goutils)
gompkg/retinue/               - POST-commit tagging (different use case)
gompkg/askai/                 - Generic AI interaction (already refactored)
```

## Edge Cases & Error Handling

### 1. No Baseline Tag
**Scenario:** First release, no previous tag exists

**Behavior:**
- Verdict: "unknown"
- Display: "API Analysis: Unknown (no baseline tag found - first release?)"
- Don't pass to AI (no meaningful analysis possible)
- Still allow commit (not blocking)

### 2. Nested Modules
**Challenge:** go-dt has nested `dtx/` module

**Solution:**
- Find nested modules: `find . -name go.mod -not -path "./go.mod"`
- Filter staged files to exclude nested module paths
- Only analyze current module's packages

### 3. Binary Files
**Scenario:** Image, PDF, or compiled binary in staging

**Behavior:**
- `git show :path` returns binary content
- Write bytes as-is to temp directory
- Analyzers skip binary files (only analyze .go files)

### 4. Deleted Files
**Scenario:** File removed from repo (in staging as deletion)

**Behavior:**
- `git show :path` fails (file not in index)
- Skip file during export (expected)
- Baseline will have it, staged won't (detected as removal)

### 5. Partial Staging
**Scenario:** File has some hunks staged, others unstaged

**Behavior:**
- `git show :path` returns staged version (correct)
- Analysis based on what will be committed
- Unstaged changes ignored (expected)

### 6. Concurrent Gomion Invocations
**Challenge:** User runs `gomion next` in two terminals

**Solution:**
- Filesystem locking in gitutils (from go-nextver pattern)
- Lock file: `~/.cache/gomion/locks/<repo-hash>.lock`
- Second invocation waits or fails gracefully

### 7. Cache Staleness
**Challenge:** Baseline tag updated in upstream, cached version stale

**Solution:**
- Fetch before each checkout in CachedWorktree
- Detect if baseline tag has been force-pushed (warn user)
- Allow `--no-cache` flag to bypass cached worktree

### 8. Large Diffs
**Challenge:** Massive diff exceeds AI token limits

**Solution:**
- Truncate diff intelligently (keep headers, summarize body)
- Rely more on AST analysis (structured, compact)
- Warn user: "Diff too large, analysis may be incomplete"

### 9. Template Errors
**Challenge:** User's custom template has syntax errors

**Behavior:**
- Parse template during load
- If error: warn user, fallback to built-in template
- Don't block commit generation

### 10. Analysis Persistence Conflicts
**Challenge:** User stages changes, exits, modifies staging, returns

**Solution:**
- Cache key includes hash of staged file list
- Different staging = different cache key = fresh analysis
- Old cache files cleaned up after 24h

## Success Criteria

1. **Chicken-and-egg solved:** Can analyze staged changes without committing
2. **Non-invasive:** Read-only operations on user's repo (all writes to cache)
3. **Domain-based architecture:** Go utilities in goutils, git utilities in gitutils
4. **Bespoke + Generic:** Specific result types for specific handling, common interface for formatting
5. **Multi-format output:** Markdown for AI, ANSI-escaped for terminal, plain text for logs
6. **Informative:** Clear verdict summaries, detailed reports on demand
7. **AI-enhanced:** Rich context passed to AI for better commit messages
8. **Multi-commit support:** Intelligent grouping suggestions
9. **Customizable:** Template-driven prompts, user can override
10. **Type-safe:** Uses go-dt types throughout
11. **ClearPath compliant:** Uses goto end, embedded doterr, no fmt.Errorf
12. **Fast:** Cached worktrees avoid repeated clones, persistent analysis avoids re-running

## Future Enhancements (Not in Initial Implementation)

1. **Incremental analysis:** Only re-analyze changed files on regenerate
2. **Historical analysis:** Compare against multiple previous versions
3. **Custom analyzers:** Plugin system for project-specific analysis
4. **Version suggestions:** Recommend major/minor/patch bump based on verdict
5. **Release notes generation:** Aggregate commit messages into release notes
6. **GitHub integration:** Auto-create PR with analysis report
7. **Config options:** Skip internal packages, adjust sensitivity, custom templates
8. **Performance analysis:** Detect benchmark changes, performance regressions
9. **Dependency analysis:** Detect new dependencies added in staging
10. **Security analysis:** Flag use of unsafe packages, known vulnerabilities

## Migration Path

This implementation makes go-nextver obsolete for the PRE-commit use case:
- **Before:** go-nextver analyzed committed code (POST-commit)
- **After:** gomion analyzes staged changes (PRE-commit)
- **Pattern reuse:** CachedWorktree concept proven in go-nextver
- **New capability:** ExportStagedFiles extends pattern to uncommitted changes

**POST-commit analysis** remains in retinue engine:
- Used for version tagging after commit
- Different use case from PRE-commit assistance

**Once complete:**
- go-nextver can be archived/deprecated
- All functionality integrated into gomion with better UX
- Single tool for entire workflow: stage → analyze → commit → tag → release
