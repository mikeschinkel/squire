# Gomion Roadmap

This document tracks planned and completed features.

## Status Key

- 🔴 **Not Started** — Planned but not yet begun
- 🟡 **In Progress** — Currently being implemented
- 🟢 **Completed** — Implemented and merged
- 🟠 **On Hold** — Paused or blocked

---

## Active Development

### Interactive Commit Workflow with AI
**Status:** 🟡 In Progress
**Priority:** High

AI-powered commit message generation with BubbleTea editor.

**Completed:**
- ✅ Interactive menu in `gomion next` for dirty repos
- ✅ `[s]tatus`, `sta[g]e`, `[u]nstage`, `[c]ommit-msg` commands
- ✅ AI generation via Claude Code CLI
- ✅ Message review: `[y]es`, `[r]egenerate`, `[e]dit`, `[b]ack`

**Remaining:**
- 🔴 BubbleTea commit message editor (structured title/body, live preview, drafts)
- 🔴 Configurable LLM providers (Claude/ChatGPT)
- 🔴 Diff size limits and truncation
- 🔴 Standalone `gomion commitmsg` command

**Docs:** `COMMIT_MSG_BRIEF.md`

### File Staging TUI
**Status:** 🟡 In Progress
**Priority:** High

Interactive file selection and disposition management for commits.

**Completed:**
- ✅ Tree view with file dispositions (commit/omit/ignore/exclude)
- ✅ Table view for directory contents
- ✅ File content viewer with diff highlighting
- ✅ Commit plan persistence to `.git/info/gomion/commit-plan.json`
- ✅ Auto-save with debouncing (3 seconds)
- ✅ Manual save (Cmd-S/Ctrl-S)

**Planned Enhancements:**
- Toggle module node on/off in tree
- Sort table columns
- Multi-select in table
- Variable-width filename column
- AM/PM time format for Modified column
- Copy filepath to clipboard
- Stop wrap-around scrolling
- Apply colors to cell values
- Display deleted files (old content)
- Line numbers in file viewer
- Side-by-side diff view with highlighting
- Syntax highlighting for code
- Markdown preview (toggleable)
- Visual save indicator

**Docs:** `gommod/FEATURES_TO_ADD.md`

---

## Planned Features

### Enhanced File Viewer and Information Display
**Status:** 🔴 Not Started
**Priority:** Medium

Comprehensive rethink of the right-pane file viewer to display rich file metadata and context.

**Current Limitations:**
- Limited space for verbose metadata
- No way to show git ignore/exclude attribution
- No file history or blame information
- No expandable file information

**Proposed Enhancements:**

**1. File Information Overlay/Popup**
- Hotkey (e.g., `i` for "info") to show detailed file information
- ESC to clear overlay
- Display:
  - Git status (modified, staged, untracked, ignored, excluded)
  - Ignore/exclude attribution:
    - `excluded: .git/info/exclude:12 "*.log"`
    - `ignored: .gitignore:8 "dist/"`
    - `ignored: global:3 ".DS_Store"`
  - File metadata (size, modified time, permissions)
  - Git history (last commit, author, age)
  - Module/package information (Go package name, imports)

**2. Enhanced File Viewer**
- Better syntax highlighting
- Line numbers (toggleable)
- Minimap for long files (optional)
- Jump to line
- Search within file
- Collapsible sections for long files

**3. Diff View Improvements**
- Side-by-side diff with better highlighting
- Inline diff mode
- Word-level diff highlighting
- Show which hunks are staged/unstaged
- Context expansion (show more lines around changes)

**4. Interactive Actions from Viewer**
- Quick actions menu (hotkey, maybe `a` for "actions")
- Copy file path
- Open in external editor
- View git log for file
- View git blame
- Un-ignore/un-exclude from viewer

**5. Multiple View Modes**
- Raw view (current)
- Diff view (side-by-side or inline)
- Markdown preview (for .md files)
- Image preview (for image files)
- JSON formatted view
- Toggle between modes with keybindings

**UI Design Considerations:**
- Popup/overlay: 1/2 to 2/3 viewport size, centered
- Preserve context (file list visible behind overlay)
- Smooth animations for open/close
- Keyboard-driven navigation
- Help text at bottom showing available actions

**Implementation Notes:**
- Leverage existing lipgloss styling
- Use bubbletea viewport for scrolling
- Async loading for heavy operations (git log, blame)
- Cache expensive operations

**Related Features:**
- In-app ignore/exclude editor (separate overlay)
- Show excluded/ignored files feature
- File disposition workflow

### API Stability Management
**Status:** 🔴 Not Started
**Priority:** High

Cross-repository stability management using Contract: annotations.

**Features:**
- Automated changelog generation from annotations
- Contract enforcement tooling (validate deprecations, breaking changes)
- Cross-repo stability validation
- RemoveAfter date coordination across dependencies
- Breaking change reports with impact analysis

**Integrations:** golang.org/x/exp/apidiff, go-tuipoc

**Docs:** `go-dt/adrs/adr-2025-12-20-stability-levels.md`, `go-doterr/adrs/adr-2025-12-20-error-sentinel-strategy.md`

### Policy File Sync & Drop-in Manager
**Status:** 🔴 Not Started
**Priority:** Medium

Declarative rules for distributing policy files across repos.

**Features:**
- User-level and project-level policy rules
- Actions: copy, template, merge, drop-in (language-aware)
- Dry-run mode with diff output
- Per-repo overrides

**Use Cases:**
- Enforce consistent linting across repos
- Share CI workflows and templates
- Apply Go drop-in files (e.g., `doterr.go`)

**Docs:** `go-doterr/cmd/sync-doterr/docs/PRD_COMPARISON.md`

### External Module Dependencies
**Status:** 🔴 Not Started
**Priority:** Medium

Add `--all` flag to `requires-tree` to include external modules.

**Features:**
- Complete dependency tree (internal + external)
- Use `go list -m -json` for discovery
- Visual distinction for external modules
- Graceful handling of unreachable modules

**Docs:** `docs/gomion-cli-tree-command-prd.md` (Section 6.2)

### Multi-Language Support
**Status:** 🔴 Not Started
**Priority:** Low

Design supports future languages (Zig, Rust, etc.).

**Config:**
```json
{
  "currentLanguage": "go",
  "languages": {
    "go": {"enabled": true},
    "zig": {"enabled": false}
  }
}
```

**Commands:** `gomion test --lang=zig` or `gomion test --zig`

**Implementation:** Language backend registry (deferred until 2nd language)

### ClearPath Linter
**Status:** 🔴 Not Started
**Priority:** Medium

Custom linter for ClearPath coding style.

**Style Characteristics:**
- Single return at end of function
- `goto end` for cleanup (not multiple early returns)
- Minimal indentation
- Clear, obvious control flow

**Command:** `gomion lint --clearpath`

### GitHub Workflow Integration
**Status:** 🔴 Not Started
**Priority:** Medium

Ensure repos have standard test and release workflows.

**Features:**
- Template-based workflow generation
- Release workflow runs tests/lint/vet before tagging
- GoReleaser integration for binaries
- Workflow validation across repos

---

## Completed Features

### Commit Plan Persistence
**Status:** 🟢 Completed (Jan 2026)

File disposition persistence across TUI sessions.

**Features:**
- Save/load commit plan to `.git/info/gomion/commit-plan.json`
- Two-layer type system (gomcfg scalars, gompkg domain types)
- Auto-save with 3-second debouncing
- Manual save (Cmd-S/Ctrl-S)
- Async I/O via tea.Cmd

**Files:** `gommod/gomcfg/commit_plan_v1.go`, `gommod/gompkg/commit_plan.go`

### Module Discovery & Dependency Ordering
**Status:** 🟢 Completed (Phase 2)

- Module discovery from `.gomion/config.json`
- Dependency-safe topological sort
- Module classification (lib/cmd/test)
- Versioned vs non-versioned heuristics

### Basic Dependency Tree Visualization
**Status:** 🟢 Completed (Dec 2025)

- ASCII tree rendering of internal module dependencies
- `--show-dirs` flag for directory labels
- `--show-all` flag for module path + location
- Markdown embedding with `--embed`, `--before`, `--after` flags

**Command:** `gomion requires-tree`

**Docs:** `docs/gomion-cli-tree-command-prd.md`

---

## BubbleTea Performance: Keep `Update()` Fast

### Core Rule

**`Update()` must be fast.** All slow/blocking work runs in `tea.Cmd`, not inline in `Update()`.

This is not an optimization — it's the intended Bubble Tea architecture.

---

### Pattern

#### 1. `Update()` handles intent, not work

On user input:
- Update lightweight state (cursor, flags, "loading")
- Return immediately
- Kick off slow work via `tea.Cmd`

#### 2. Slow work runs in `tea.Cmd`

Examples:
- Git commands
- Filesystem scans
- Network calls
- Expensive computation

Returns a `Msg` when finished.

#### 3. Results re-enter via `Msg`

- `Update()` handles the result
- Stores output in model
- Clears loading state
- Triggers re-render

---

### Flow

```
KeyMsg
  ↓
Update():
  - record intent
  - set loading=true
  - return Cmd
  ↓
Cmd executes slow work
  ↓
Cmd returns ResultMsg
  ↓
Update():
  - store results
  - loading=false
  ↓
View()
```

---

### Example

```go
type gitResultMsg struct {
	out []byte
	err error
}

func runGit(args ...string) tea.Cmd {
	return func() tea.Msg {
		cmd := exec.Command("git", args...)
		out, err := cmd.CombinedOutput()
		return gitResultMsg{out: out, err: err}
	}
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "r" {
			m.loading = true
			return m, runGit("status", "--porcelain=v1")
		}
	case gitResultMsg:
		m.loading = false
		if msg.err != nil {
			m.err = msg.err
		} else {
			m.output = string(msg.out)
		}
	}
	return m, nil
}
```

---

### What NOT to Do

❌ **Never call `exec.Command()` directly in `Update()`**
❌ **Never do filesystem I/O in `Update()`**
❌ **Never parse large data in `Update()`**

✅ **Always use `tea.Cmd` for slow work**
✅ **Always return fast from `Update()`**
✅ **Always handle results via `Msg`**

---

_For details: [Bubble Tea Tutorial](https://github.com/charmbracelet/bubbletea/tree/master/tutorials)_

---

## Future Considerations

Ideas not yet committed to roadmap:

- **In-App Ignore/Exclude Editor** — Modal overlay text editor (1/2 to 2/3 viewport size) for editing `.gitignore` and `.git/info/exclude` files within the TUI, with syntax highlighting and live pattern matching preview
- BubbleTea Performance Optimization
- Release planning & tagging automation
- Changelog generation from commit history
- License management (filter dependencies by license)
- GitHub Actions coordination
- TUI dashboard for stability status
- Interactive migration planning TUI
