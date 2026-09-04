# Claude Code adapter

Before editing, load and follow the two truth chains defined by `@AGENTS.md`.
Engineering truth is `@AGENTS.md` -> relevant `@docs/engineering/` documents ->
matching file-scoped `@.cursor/rules/*.mdc`. Business truth is the current user's
explicit requirement/current task -> the current valid PRD, prototype, or
confirmed business decision -> implementation evidence.

This adapter is not a competing truth source. Engineering rules do not invent
business behavior, and business requirements do not bypass MUST-level
engineering gates. Run the pre-flight, keep the diff task-scoped, and finish
with `git diff --check` plus `make check`. Report any failed or skipped gate
exactly as required by `AGENTS.md`.
