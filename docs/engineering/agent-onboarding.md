# Agent Onboarding Guide

This document outlines the mandatory engineering protocol for all AI coding agents (Claude Code, Cursor, Gemini, Codex, Jules, etc.) working on the Workbench codebase.

## 1. Truth Chains

Engineering decisions and business requirements follow separate truth chains:

- **Engineering Truth:** `AGENTS.md` -> `docs/engineering/` -> file-scoped `.cursor/rules/*.mdc`. It governs architecture, security, quality, layers, and engineering constraints.
- **Business Truth:** The current user's explicit requirement -> confirmed PRD/decision -> implementation evidence. It governs what features should do.

Engineering rules MUST NOT invent business behavior. Business requirements MUST NOT bypass MUST-level engineering gates.

## 2. Mandatory Pre-Flight Checklist

Before editing any file, confirm and report:

1. Repository root and safe branch:
   ```sh
   pwd
   git rev-parse --show-toplevel
   git branch --show-current
   git status --short
   ```
   **STOP immediately if current branch is `main` or `master`.**
2. Task goal, allowed change scope, and forbidden scope.
3. Use `spec-index.md` to distinguish current rules, implementation catalogs and
   historical evidence; load the task-relevant engineering guides from it.
4. Existing dirty or untracked WIP. Never discard or overwrite unrelated work.

## 3. Load the canonical boundaries

Read the MUST rules and Golden Reference policy in `AGENTS.md`; this guide does
not maintain a second abbreviated rule list. Refer to [spec-index.md](spec-index.md)
for the full specification index. Then read the focused engineering references
selected by the task, `module-index.md`, and the current task card. If a decision
gate is unresolved, report that exact gate rather than inventing a business or
schema contract.

## 4. Pre-Handoff Gates

Before reporting completion or handoff, run:

```sh
git diff --check
make check
```

For rules, gates or baseline changes, also run:

```sh
bash scripts/test-quality-gates.sh
```

## 5. Delivery Truth

Use `done`, `complete`, `verified`, or `可交付` ONLY when every required gate and acceptance test passes. If any gate fails or is skipped, report `partial`, `failed`, or `blocked`.

## 6. Agent loading evidence

Adapter files being present does not prove a tool loaded them. There is a fundamental
distinction between:
1. **File existence:** Adapter files (e.g. `GEMINI.md`, `.cursorrules`, `CLAUDE.md`) existing in the repository tree; and
2. **Cold-start loading verification:** The active AI agent explicitly executing read/view tools on `AGENTS.md` and the required `docs/engineering/*.md` guides in the current session, and producing verifiable pre-edit evidence.

Before accepting an agent handoff, the agent must output and verify:
- Tool/model runtime and conversation ID if exposed; otherwise unknown. Never
  infer the exact model or session ID from an adapter filename.
- Exact git SHA and branch verified via CLI.
- Explicit list of specification files read during the current session via tool calls.
- Specific applicable MUST rules governing the task.
- Clean pre-edit and post-edit verification logs with exit codes.
- Honest reporting of blocked decisions or legacy exceptions without inventing contracts.

## 7. Continuing work safely

After context reset, tool switch or another contributor's edits, reload the
current task and affected rules; inspect branch/HEAD/WIP again. Do not rerun
COMPLETE cards or apply stale plans without a current task reason. Verify the
current PRD/decision source; tool memories and reference code are not business
authority. A user-confirmed decision to retain an implemented behavior stays
limited to that scope, not an automatic precedence rule for all code.

Compare affected file content before writing and starting/final WIP inventories
(hashes where useful) before handoff. Reconcile external changes before editing
the affected scope; independent work can proceed. Do not reset/stash/restore
all WIP to get a clean review. Read-only audits can identify separate snapshots
and report drift. Do not claim one moving tree is a stable reviewed revision.

## 8. Minimal handoff record

Use a short task record, not a copied constitution or mandatory giant report:

```text
Scope / allowed files / forbidden actions:
Repository / branch / HEAD / WIP snapshot:
Rules actually read / current confirmed business source:
Changed behavior or audit finding / file:symbol:
DB ownership / environment authorization / concurrent writers (if affected):
Queries / lock order / bounds / idempotency evidence (if affected):
Commands / exit codes / executed cases / skipped gates:
External changes / unresolved risks / remaining decisions:
Docs, application, live acceptance and CI status separately:
```

Run added/affected tests not enumerated by `make check`. The gate self-test in
section 4 is required for rule/gate/baseline changes; see `quality.md`. Report
early failures and later unexecuted gates. A failing audit does not authorize
business-code repair. No secrets, business payloads or raw DB diagnostics in logs.

Tool loading procedure: [agent-compatibility.md](agent-compatibility.md).
