# Claude Code adapter

Before editing, load and follow `@AGENTS.md`; it is the highest and canonical
engineering source. Then read the relevant `@docs/engineering/` document and
the matching file-scoped `@.cursor/rules/*.mdc` rule.

Do not treat this adapter, historical code, or a task document as a competing
rule source. Run the pre-flight, keep the diff task-scoped, and finish with
`git diff --check` plus `make check`. Report any failed or skipped gate exactly
as required by `AGENTS.md`.
