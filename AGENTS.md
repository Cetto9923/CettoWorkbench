# workbench · Agent entry

This file is the cross-tool entry for Codex and other agents. The project rules
remain single-sourced in the existing Cursor files; do not duplicate or weaken
them here.

Before changing code, read these files in order:

1. `.cursorrules`
2. `.cursor/rules/conventions.mdc`
3. The relevant module code and design document under `docs/`

Implementation constraints:

- Preserve the Go SSR architecture and the Handler → Service → Repo boundary.
- Use the existing module and shared component conventions before adding code.
- Keep changes page-scoped and evidence-backed; do not copy the legacy
  `CRCBWorkbench/po-v2` implementation wholesale.
- Treat `CRCBWorkbench/po-v2` as a UI/interaction reference only. Data,
  permissions, routes, and write behavior must come from this repository.
- Do not display placeholder metrics as real data.
- Verify every UI change in a real browser after automated checks pass.
