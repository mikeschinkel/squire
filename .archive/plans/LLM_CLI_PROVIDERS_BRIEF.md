# Gomion LLM CLI Provider Integration — Design Brief (for Claude Code)

**Date:** December 24, 2025  
**Audience:** Claude Code (implementation agent) + Gomion maintainers  
**Purpose:** Implement a clean, extensible way for Gomion (a Go CLI) to invoke *external* AI coding CLIs (Claude Code, Codex, Letta Code) **without Gomion owning API keys**, while keeping the door open for additional tools later.

---

## 1) Executive summary

Gomion needs to call out to AI “coding assistant” CLIs as subprocesses from Go. The CLIs themselves should handle authentication (user subscription login, local credential storage, etc.) so Gomion does **not** store or manage keys. For the MVP, implement a provider abstraction with **three first-class providers**:

1. **Claude Code**
2. **Codex**
3. **Letta Code**

Additionally, implement an **External CLI provider** (template-based) as an escape hatch for future tools and for covering unknown flag shapes (especially important for Letta Code if its CLI surface differs across versions).

Later phases can add dedicated providers for OpenCode and GitHub Copilot; even later, Charm’s Mods and much later Crush.

---

## 2) Goals

### MVP goals (must have)
- Provide a Go API and CLI-level integration in Gomion to:
  - Send a prompt (and optional context) to a selected AI CLI tool.
  - Capture results reliably (stdout/stderr, exit code).
  - Support **non-interactive/headless** invocation where possible.
  - Support **streaming output** in Gomion’s newline-oriented UI.
- Gomion must not require users to copy/paste or store API keys for MVP providers.
- Implement three first-class providers: **claude**, **codex**, **letta**.
- Include a configurable, generic provider: **external-cli**.
- Provide robust diagnostics and predictable output parsing.

### Nice-to-have (still MVP if quick)
- “Structured output” mode: Gomion can request JSON output if a provider supports it; otherwise Gomion wraps text output into a JSON envelope for downstream parsing/logging.
- Basic model selection support where providers allow it.
- Configurable timeouts and cancellation via context.

---

## 3) Non-goals (MVP)
- No “multi-turn agent session” state management inside Gomion (beyond passing a prompt).
- No token/usage accounting beyond what providers explicitly return.
- No deep tool-permission orchestration in Gomion (providers may offer their own controls).
- No attempt to standardize the ecosystem; Gomion adapts.

---

## 4) Provider priority roadmap

### MVP
- Claude Code
- Codex
- Letta Code

### Later
- OpenCode
- GitHub Copilot (`gh copilot …`)
- Charm `crush`

### Maybe
- Charm `mods`

---

## 5) Core requirements

### 5.1 Invocation model
Gomion will execute provider CLIs via subprocess from Go. Requirements:
- Must support:
  - `context.Context` for cancellation
  - timeout (configurable)
  - cwd control
  - environment injection (minimal; typically pass-through)
- Must not echo secrets (even if provider uses env vars).
- Must capture:
  - exit code
  - stdout (stream + final)
  - stderr (stream + final)
  - duration
  - the exact command line (with redaction rules)

### 5.2 Input model
A request may include:
- `Prompt` (string)
- `Mode` (e.g., “chat”, “explain”, “commit-message”, “fix-tests”; provider may ignore)
- `Files` (paths to include or pass as context; provider-specific)
- `Stdin`:
  - None
  - Prompt on stdin
  - Arbitrary data piped (diff, logs, etc.)

### 5.3 Output model
Gomion needs a consistent internal response type:

```go
type Response struct {
    Provider   string            // "claude" | "codex" | "letta" | "external-cli"
    Model      string            // if known
    Format     string            // "text" | "json"
    Text       string            // filled if Format=="text" OR if json wrapping includes text
    JSON       []byte            // raw json if provider returns json; optional
    ExitCode   int
    DurationMS int64
    StdoutRaw  []byte            // always captured
    StderrRaw  []byte            // always captured
    Meta       map[string]string // provider-specific bits (version, flags used, etc.)
}
```

If a provider can emit JSON, store it in `JSON` and optionally derive `Text` from a known field. If not, set `Format="text"` and store output in `Text`.

### 5.4 Configuration and selection
Gomion must allow selecting a provider via:
- CLI switch (e.g., `--llm-provider claude`)
- config file default

Provider config must include:
- path to binary (optional; default to binary name in PATH)
- default args
- per-provider options (model, output format, etc.)
- timeouts

---

## 6) Proposed package layout (Go)

**Recommendation (adjust to repo conventions):**
- `internal/llm/` – core abstractions + runner
- `internal/llm/providers/` – first-class providers
- `internal/llm/external/` – template-driven provider
- `internal/llm/contract/` – shared request/response types

Example structure:

```
internal/llm/
  contract/
    types.go
  runner/
    runner.go
  providers/
    claude/
      provider.go
    codex/
      provider.go
    letta/
      provider.go
    externalcli/
      provider.go
```

---

## 7) Provider abstraction

### 7.1 Interface
```go
type Provider interface {
    Name() string
    Detect(ctx context.Context) (Detection, error) // binary exists, version, capabilities
    Capabilities(ctx context.Context) (Caps, error)
    BuildCommand(req Request) (CmdSpec, error)
    ParseResult(exec ExecResult) (Response, error)
}
```

Where:
- `CmdSpec` includes `Path`, `Args`, `Env`, `Cwd`, and `StdinMode`.
- `ExecResult` includes stdout/stderr bytes, exit code, duration, etc.
- `Caps` communicates:
  - supports JSON output
  - supports streaming
  - supports model selection
  - supports “headless” prompt mode (no TTY)
  - supports file context args (if known)

### 7.2 Execution runner
Runner responsibilities:
- resolve provider by name
- call `Detect()` optionally to produce nicer errors
- run subprocess with context cancellation
- handle streaming to Gomion inline UI (write stdout incrementally)
- apply timeout
- collect stdout/stderr + exit code
- call `ParseResult()`

---

## 8) First-class provider behavior

### 8.1 Claude Code provider
**Key implementation points:**
- Prefer headless mode if available (e.g., “print prompt and exit” pattern).
- Prefer JSON output if available; otherwise text.
- Keep flags in one place and version-gate if needed.
- Ensure Gomion can run it without TTY.

**Detect():**
- `claude --version` (or equivalent; tolerate non-zero if that tool prints version elsewhere)

**BuildCommand():**
- If JSON requested and supported: add output-format flag.
- For text: use prompt flag mode.
- Provide a config option to add extra allowed tools / permission mode if user wants.

**ParseResult():**
- If JSON, store raw bytes.
- Else store text.

> Note: actual flags and semantics may change; implementation should allow overrides via config.

### 8.2 Codex provider
**Key implementation points:**
- Use `codex exec` for headless.
- Support `--json` / schema output if requested.
- Avoid auto-approval by default; keep “dangerous auto” behind explicit config.

**Detect():**
- `codex --version` (or `codex -V` if supported; check in detect)

**BuildCommand():**
- Primary: `codex exec <prompt>`
- Add `--json` if requested
- Optional: `--model` if set in config or request

### 8.3 Letta Code provider
**Key implementation points:**
- Letta CLI surface is treated as **version-sensitive**; do not hardcode assumptions without verifying.
- Implement provider to:
  - discover help/commands at runtime (or at detect-time) to validate a usable command exists
  - support a configured invocation template if built-in defaults fail

**Detect():**
- Try:
  - `letta --version`
  - `letta --help`
- Capture and store the help output (trimmed) in `Detection` for debugging.

**BuildCommand():**
- Prefer a headless “run/prompt” mode if supported.
- Otherwise fall back to the **external-cli template** mechanism (see below) but keep provider name “letta”.

**ParseResult():**
- Best-effort:
  - if JSON is returned, store it
  - else treat as text

---

## 9) External CLI provider (template-driven)

This is essential for:
- unknown future tools (OpenCode, gh copilot, mods, crush)
- user-customized flag layouts
- Letta CLI changes without a code release

### 9.1 Template config fields
Support a config block like:

```json
{
  "llm": {
    "provider": "external-cli",
    "externalCli": {
      "name": "mytool",
      "path": "mytool",
      "args": ["run", "--json", "--prompt", "{{prompt}}"],
      "cwd": "{{repoRoot}}",
      "stdin": "none", // none|prompt|data
      "env": {
        "MYTOOL_MODE": "headless"
      },
      "expectsJson": true
    }
  }
}
```

Placeholders:
- `{{prompt}}` – escaped prompt token
- `{{cwd}}` or `{{repoRoot}}`
- `{{file:<path>}}` – optional advanced: load file contents into arg (future)

### 9.2 Redaction rules
External provider config may include secrets in env var values; Gomion must redact:
- any env var keys matching `*_KEY`, `*_TOKEN`, `*_SECRET`, `PASSWORD`, etc.
- any arg values flagged as secret in config (optional future)


---

## 10) Testing strategy

### 10.1 Fake CLI stubs
Create small test binaries/scripts (in `testdata/fakecli/`) that emulate:
- success text output
- success json output
- non-zero exit codes
- huge outputs / streaming behavior
- stderr warnings but stdout results
- timeouts / hangs

Run provider tests by pointing config `path` to those fakes.

### 10.2 Golden tests
For each provider:
- BuildCommand() golden snapshot for a few requests
- ParseResult() tests with recorded outputs

### 10.3 Integration smoke test (optional)
If CI environment supports it, include a skipped-by-default test that runs `--version` for installed CLIs.

---

## 11) Error handling + UX

- If provider binary is missing: show actionable error:
  - name of binary
  - how to install (if known) OR “install it and ensure it’s in PATH”
- If provider returns non-zero:
  - include stderr tail in error message
  - include a hint: “run with --debug to see full command + logs”
- If JSON parse fails when JSON expected:
  - fall back to text wrapping, but mark `Format="text"` and set a meta key `json_parse_error=true`

---

## 12) Observability & debug logging

Add `--debug-llm` (or reuse global debug) to log:
- provider chosen
- detected provider version (if available)
- final command (redacted)
- cwd
- timeout
- duration
- exit code
- bytes read from stdout/stderr

---

## 13) Implementation checklist (MVP)

### Core
- [ ] Add `Request` / `Response` types
- [ ] Implement subprocess runner with streaming support
- [ ] Implement provider registry and selection
- [ ] Implement config parsing for provider settings

### Providers
- [ ] Claude Code provider: detect + headless invocation + parse
- [ ] Codex provider: detect + `exec` invocation + parse
- [ ] Letta Code provider: detect + headless invocation if possible; fallback to template; parse

### Generic
- [ ] External CLI provider with templates + placeholders
- [ ] Redaction rules

### CLI UX
- [ ] `gomion ai run` command + flags
- [ ] Inline UI streaming integration (newline-oriented UI)

### Tests
- [ ] Fake CLIs + unit tests
- [ ] Golden tests for command building
- [ ] Timeout/cancel tests

---

## 13) Notes for Claude Code (agent implementing this)

- Do **not** assume Letta flags are stable; build in flexibility.
- Keep provider-specific flags behind config as much as possible.
- Prefer “simple request → output” for MVP. Multi-turn sessions can be later.
- Ensure no auth material is stored by Gomion. The provider CLIs own auth.

---

## 15) Deliverables (what to commit)

- Go source code implementing the above
- Config schema/doc updates for new provider settings
- Unit tests with fake CLIs
- Minimal README section: “AI Provider Setup” + examples
