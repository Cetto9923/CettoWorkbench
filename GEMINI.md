# Gemini adapter

Before editing, load and follow the two truth chains defined by AGENTS.md.
Engineering truth is AGENTS.md -> relevant docs/engineering/ documents ->
matching file-scoped rules. Business truth is the current user's explicit
requirement/current task -> the current valid PRD, prototype, or confirmed
business decision -> implementation evidence.

This adapter is not a competing truth source. Engineering rules do not invent
business behavior, and business requirements do not bypass MUST-level
engineering gates.

## Mandatory Pre-Flight

1. Confirm repository root, branch, and status:
   ```sh
   pwd
   git rev-parse --show-toplevel
   git branch --show-current
   git status --short
   ```
   Confirm branch is NOT `main` or `master`.
2. Confirm current task goal, allowed change scope, and relevant docs under `docs/engineering/`.
3. Preserve existing dirty/untracked WIP. Never revert unrelated changes.

## Engineering Invariants

- Monolith architecture: Handler -> Service -> Repo -> Database.
- Zero schema hallucinations: never invent table columns or soft-delete semantics without verified schema DDL.
- List queries must follow: SQL filter -> SQL sort -> SQL count -> SQL pagination. Never use `SELECT *` or in-memory pagination.
- Security: never commit secrets; no MD5/SHA password hashing.

## Hand-off Gates

Before completing a task, always execute:

```sh
git diff --check
make check
```

Report all passed, failed, or skipped checks. Never claim completion if any gate fails.
