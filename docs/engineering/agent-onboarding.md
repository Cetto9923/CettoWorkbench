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
3. Relevant engineering guides: `architecture.md`, `database.md`, `frontend.md`, `testing.md`, `quality.md`.
4. Existing dirty or untracked WIP. Never discard or overwrite unrelated work.

## 3. Load the canonical boundaries

Read the MUST rules and Golden Reference policy in `AGENTS.md`; this guide does
not maintain a second abbreviated rule list. Refer to [spec-index.md](spec-index.md)
for the full specification index. Then read the focused engineering references
selected by the task, `module-index.md`, and the current task card. If a decision
gate is unresolved, report that exact gate rather than inventing a business or
schema contract.

## 4. Pre-Handoff Gates

Before reporting completion or handoff, always run:

```sh
git diff --check
make check
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
- Its tool/model runtime and conversation ID.
- Exact git SHA and branch verified via CLI.
- Explicit list of specification files read during the current session via tool calls.
- Specific applicable MUST rules governing the task.
- Clean pre-edit and post-edit verification logs with exit codes.
- Honest reporting of blocked decisions or legacy exceptions without inventing contracts.
