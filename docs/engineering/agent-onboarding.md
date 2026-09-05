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

## 3. Core Architectural Boundaries

- **Layers:** Handler -> Service -> Repo -> Database.
  - Handler binds protocol input, calls Service, and writes HTTP response.
  - Service owns business logic, authorization decisions, and transactions.
  - Repo owns database queries and data mapping.
  - Handlers MUST NOT access the DB directly; Repos MUST NOT decide permissions.
- **Query Discipline:** SQL filter -> SQL sort -> SQL count -> SQL pagination. Never use `SELECT *`, unindexed scans, or unbounded in-memory filtering.
- **Schema Ownership:** Workbench-owned tables follow Workbench conventions; ZenTao tables follow ZenTao schema. Do not invent columns without real schema DDL.
- **Security:** No secrets in Git; no MD5/SHA password writes.

## 4. Pre-Handoff Gates

Before reporting completion or handoff, always run:

```sh
git diff --check
make check
```

## 5. Delivery Truth

Use `done`, `complete`, `verified`, or `可交付` ONLY when every required gate and acceptance test passes. If any gate fails or is skipped, report `partial`, `failed`, or `blocked`.
