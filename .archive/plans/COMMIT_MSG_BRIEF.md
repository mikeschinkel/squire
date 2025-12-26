# Squire — LLM Commit Message Generation (CLI MVP)  
**Implementation brief for Codex**

Date: 2025-12-23  
Scope: Add an MVP feature to Squire that generates a git commit message from the **staged diff** by invoking an installed LLM CLI (**Claude Code** and/or **OpenAI Codex CLI**) from Go. This MVP must be provider-agnostic via config and internal interfaces, and must be easy to evolve later to direct APIs and/or a local LLM server.

---

## 0) Goals and non-goals

### Goals
- **Generate a draft commit message** for the currently-selected repo (leaf repo in `squire next`) based only on **staged** changes.
- Use **installed LLM CLIs** for fastest MVP:
  - `claude` (Claude Code CLI)
  - `codex` (OpenAI Codex CLI)
- Make the implementation **configurable** (no hardcoded provider).
- Be safe/robust for interactive use:
  - timeouts
  - stderr capture and helpful errors
  - cap diff size (avoid huge prompts)
- Output should be **structured** for reliable parsing (JSON `{subject, body}`), but allow plain text fallback.

### Non-goals (for MVP)
- Direct cloud APIs (OpenAI/Anthropic SDKs)
- Local model server integration (Ollama/etc.) beyond “future-ready hooks”
- Automatic `git commit` (optional later). For MVP, just generate and display / write message to file; optionally open `$EDITOR`.

---

## 1) UX: where this appears in Squire

### 1.1 Interactive flow: `squire next`
Your existing interactive prompt shows:

```
Options: [s]tatus  [a]dd  [u]nstage  [q]uit
```

Add a new option:

- **`[m]essage`** (or `[c]ommitmsg`) → generate a commit message for the **staged** diff in the leaf repo.

Suggested UX:
1. User picks `m`.
2. Squire computes staged diff (and lightweight status context).
3. Squire calls the configured LLM driver.
4. Squire shows:
   - Subject line
   - Body
5. Provide follow-up options:
   - `[e]dit` → opens `$EDITOR` on a temp file containing the message
   - `[w]rite` → writes message to `.git/SQUIRE_COMMITMSG` (or a temp file) for later `git commit -F ...`
   - `[b]ack` → returns to the options menu

**Important behavior:** ignore untracked/un-staged files by default. Only staged diff is considered unless config enables otherwise.

### 1.2 Optional non-interactive subcommand (recommended)
Add a hidden/low-profile command for scripting and for future CI integration:

- `squire commitmsg [--repo <dir>] [--format json|text] [--output <file>]`

This can be used internally by `squire next` or independently. It also makes unit/integration testing easier.

---

## 2) Configuration

### 2.1 Config structure
Add an `ai_provider` section in Squire config (user-level and/or repo-level override). JSON example:

```jsonc
{
  "ai_provider": {
    "enabled": true,

    // Provider names for MVP:
    // "claude_cli" | "codex_cli"
    "provider": "claude_cli",

    // Optional: override executable names/paths
    "claude_exe": "claude",
    "codex_exe": "codex",

    // Optional: a file containing the system prompt / agent prompt
    // (You said you already have a detailed Claude agent prompt; point to it here.)
    "system_prompt_file": "~/.config/squire/prompts/commitmsg_agent.txt",

    // Prompt safety / limits
    "max_diff_bytes": 200000,
    "timeout_seconds": 60,

    // Output preference
    "output": "json", // "json" preferred; "text" fallback

    // Conventional commits toggle (pass as a hint; no enforcement beyond prompt)
    "conventional_commits": true,

    // Env keys to remove when spawning child process (to avoid accidentally switching to API billing)
    "strip_env": ["ANTHROPIC_API_KEY", "OPENAI_API_KEY"]
  }
}
```

### 2.2 Config loading & path expansion
- Support `~` expansion for `system_prompt_file`.
- Provide defaults when fields absent:
  - provider: `claude_cli` if `claude` exists in PATH; else `codex_cli` if `codex` exists; else error.
  - max_diff_bytes: `200000`
  - timeout_seconds: `60`
  - output: `json`
- If `enabled` is false, hide or disable the menu option.

---

## 3) Core design: internal interfaces

Create a small internal package, e.g.:

- `squirepkg/commitmsg`

### 3.1 Types
```go
type Request struct {
	RepoDir          string
	Branch           string
	StagedFiles      []string
	UntrackedFiles   []string // for display only in MVP; not sent by default
	StagedDiff       string

	Conventional     bool
	MaxDiffBytes     int
	MaxSubjectRunes  int // optional (e.g. 72)
}

type Result struct {
	Subject string `json:"subject"`
	Body    string `json:"body"`
	Raw     string // optional raw output for debugging
}

type Generator interface {
	Generate(ctx context.Context, req Request) (Result, error)
}
```

### 3.2 Generator selection
Implement:

- `func NewGenerator(cfg LLMConfig) (Generator, error)`

Select based on `cfg.Provider`:
- `"claude_cli"` → `ClaudeCLIGenerator`
- `"codex_cli"` → `CodexCLIGenerator`

---

## 4) Git data collection (objective context)

Add a helper in your git wrapper layer (whatever exists today), or implement minimal commands:

### 4.1 Required commands
- `git rev-parse --abbrev-ref HEAD` → branch
- `git diff --cached` → staged diff
- `git diff --cached --name-only` → staged file list
- optionally `git status --porcelain=v2 -b` → structured status summary (useful for debugging/logging)

### 4.2 Diff size enforcement
Before calling the LLM:
- If `len(stagedDiff) > cfg.MaxDiffBytes`, fall back to:
  - `git diff --cached --stat`
  - and/or truncate diff with a clear marker:
    - “(diff truncated to N bytes)”
- Prefer **diffstat** over truncating mid-hunk if you can (but truncation is acceptable in MVP).

### 4.3 Secret redaction (light MVP)
Do a cheap redaction pass on the diff before sending it:
- Replace any PEM blocks:
  - `-----BEGIN ...-----` to `-----END ...-----`
- Replace obvious tokens if detected (simple regexes).
Keep it minimal; the primary defense is “only staged changes” + “don’t stage secrets”.

---

## 5) Prompt construction

### 5.1 Output contract
Prefer **JSON output** for reliable parsing:

```json
{"subject":"...","body":"..."}
```

No extra keys.

### 5.2 Standard instruction content (model-agnostic)
Build a prompt that:
- States the repo/branch
- States “ONLY staged diff below; ignore untracked/un-staged”
- Asks for concise subject (<= 72 chars)
- Suggests conventional commits if enabled

Example “user prompt” body:

```
You are generating a git commit message.

Rules:
- Use ONLY the staged diff provided below.
- Do NOT mention untracked files.
- Subject line: imperative mood, <= 72 chars, no trailing period.
- If a body is needed, use short bullet points.
- Output ONLY valid JSON: {"subject":"...","body":"..."} with no extra keys.

Context:
- Repo: <repoDir>
- Branch: <branch>
- Conventional commits: <true/false>

---- STAGED DIFF ----
<diff here>
```

### 5.3 System prompt / agent prompt
If `system_prompt_file` is set:
- pass it to the CLI driver using that tool’s flag:
  - Claude: `--system-prompt-file <path>`
  - Codex: (no exact equivalent; include content at top of prompt as a prefix in MVP)

---

## 6) CLI driver: Claude Code (`claude`)

### 6.1 Invocation requirements
Use `exec.CommandContext`, no shell.

Preferred flags for MVP:
- `-p` (print-only / non-interactive)
- `--output-format json`
- `--json-schema <schema>` (inline string is acceptable)
- `--system-prompt-file <path>` if configured
- `--disallowedTools Edit` (optional safety)
- prompt string as final arg
- staged diff piped via stdin

### 6.2 JSON schema
Inline schema:

```json
{
  "type":"object",
  "properties":{
    "subject":{"type":"string"},
    "body":{"type":"string"}
  },
  "required":["subject","body"],
  "additionalProperties":false
}
```

### 6.3 Environment handling
To “piggyback” on the user’s subscription login:
- Ensure the CLI uses its existing auth on the machine.
- Avoid accidentally switching to API-billed mode if the user has an API key in env:
  - remove `ANTHROPIC_API_KEY` if `strip_env` includes it.

### 6.4 Output parsing
- Read stdout.
- `json.Unmarshal` into `Result`.
- Trim whitespace.
- If JSON parsing fails:
  - save raw output for display
  - return a helpful error suggesting switching `llm.output` to `"text"` or verifying CLI flags.

---

## 7) CLI driver: OpenAI Codex CLI (`codex`)

### 7.1 Invocation requirements
Use `codex exec` for scripting.

Recommended MVP approach:
- Provide **the entire prompt (including diff)** on stdin by passing `-` as the prompt argument.
- Use `--cd <repoDir>` so Codex runs with repo context (even if not used in MVP).
- Use `--color never` to avoid ANSI control codes in output capture.
- Use `--output-last-message <file>` to write only the final model message to a file.

Example args:
```
codex exec - --cd <repoDir> --color never --output-last-message <tmpFile>
```

Then read `<tmpFile>`.

### 7.2 Structured output (optional in MVP)
Codex supports `--output-schema <jsonSchemaFile>`. If you implement this, write a temp schema file and pass its path, then parse JSON from stdout or output file.

If you want faster MVP: request JSON in the prompt and parse it from `--output-last-message` content.

### 7.3 Environment handling
To “piggyback” on subscription auth:
- use existing `codex` login on machine
- optionally strip `OPENAI_API_KEY` if configured, to avoid API-billed mode. 

---

## 8) Integration into `squire next` interactive menu

### 8.1 Add menu option
In the menu loop that currently handles:
- status
- add
- unstage
- quit

Add:
- message / commitmsg

### 8.2 Implementation steps for the handler
When user chooses `m`:
1. Determine current repo dir (leaf-most repo reported by `squire next`).
2. Collect staged diff and branch.
3. If no staged changes:
   - print “No staged changes; nothing to generate.”
4. Call `llmcommit.NewGenerator(cfg)` and then `Generate(...)`.
5. Display result and offer follow-on actions:
   - `[e]dit`: write a temp file and open `$EDITOR`
   - `[w]rite`: write to `.git/SQUIRE_COMMITMSG`
   - `[b]ack`

### 8.3 Editing behavior
Implement `OpenInEditor(path string)`:
- Determine editor from `$GIT_EDITOR`, `$VISUAL`, `$EDITOR` (in that order), fallback to `vi`.
- Spawn editor with stdio connected to current terminal.
- After editor exits, read the file back and show updated message.

### 8.4 ClearPath / style constraints
Follow repository style conventions:
- Avoid single-line compound `if` initializers.
- Prefer your ClearPath `goto end` pattern where cleanup/error propagation is needed.
- Keep errors wrapped with context.

---

## 9) Error handling and observability

### 9.1 Timeouts
Use `context.WithTimeout` based on `timeout_seconds`.

### 9.2 stderr capture
Always capture stderr and include it in errors (but do not spam on success).

### 9.3 Diagnostics flag
Add a hidden or config-controlled debug mode:
- prints which provider was selected
- prints the exact CLI args (redacting secrets)
- prints prompt size and whether truncation happened

---

## 10) File layout suggestion

Add/modify files (names are suggestions; adapt to repo conventions):

- `internal/config/llm.go`  
  - `type LLMConfig struct { ... }`
  - defaults + validation

- `squirepkg/commitmsg/types.go`  
  - `Request`, `Result`, `Generator`

- `squirepkg/commitmsg/factory.go`  
  - `NewGenerator(cfg LLMConfig)`

- `squirepkg/commitmsg/claude_cli.go`  
  - `ClaudeCLIGenerator`

- `squirepkg/commitmsg/codex_cli.go`  
  - `CodexCLIGenerator`

- `gitutil/staged.go` (or wherever git helpers live)  
  - staged diff and staged files retrieval

- `cmd/squire/next.go` (or wherever the interactive loop lives)  
  - add menu option + handler

- `cmd/squire/commitmsg.go` (optional command)

- `osutil/editor.go`  
  - `OpenInEditor`

---

## 11) Acceptance criteria (definition of done)

1. With staged changes in a repo, running `squire next`, choosing `m` produces a commit message draft.
2. Draft is based on **staged diff only**.
3. Provider is selectable by config:
   - `claude_cli` works if `claude` is installed and authenticated
   - `codex_cli` works if `codex` is installed and authenticated
4. Output is parsed as JSON `{subject, body}` by default; error is helpful if parsing fails.
5. If diff is too large, tool truncates or uses diffstat and informs user.
6. Errors show stderr and how to resolve (missing CLI, auth not set up, etc.).
7. Unit tests exist for:
   - prompt builder (includes key rules)
   - env stripping helper
   - output parsing (JSON success + failure modes)
8. No shell invocation (no `bash -lc`); use `exec.CommandContext`.

---

## 12) Suggested incremental implementation plan

1. Add config struct + loading + defaults.
2. Add `commitmsg` package with types + factory.
3. Implement Claude CLI generator (most deterministic with JSON schema support).
4. Integrate into `squire next` menu with `[m]essage`.
5. Add editor flow (`[e]dit`) + write flow (`[w]rite`).
6. Implement Codex CLI generator as alternative provider.
7. Add optional `squire commitmsg` subcommand.
8. Add tests and docs.

---

## 13) Quick manual test checklist (developer)

In a repo with staged changes:

- `claude` path:
  - `claude -p --output-format json --json-schema '...' '...' < <(git diff --cached)`
  - verify JSON
- `codex` path:
  - pipe prompt+diff: `... | codex exec - --cd . --color never --output-last-message /tmp/msg`
  - verify output in `/tmp/msg`

Then test in Squire:
- run `squire next`
- choose `m`
- choose `e` to edit
- choose `w` to write to commit message file

---

## 14) Notes for future evolution (not MVP)

- Add direct API providers (OpenAI Responses API, Anthropic Messages API).
- Add local model server provider (Ollama OpenAI-compatible endpoint).
- Add multi-candidate generation (N alternatives).
- Add commit execution: `git commit -F <file> -e` after user confirms.
- Add structured “reasoning” field to help review (keep optional and off by default).

---
