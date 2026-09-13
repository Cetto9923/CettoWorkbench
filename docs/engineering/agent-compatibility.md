# Agent entry points

`AGENTS.md` is the only normative rule source; this table lists each tool's
discovery file. File existence does not prove loading.

| Tool | Entry | Notes |
|---|---|---|
| Codex / Jules / Windsurf | root `AGENTS.md` | remote sessions need a revision containing it |
| Claude Code | `CLAUDE.md` | imports `AGENTS.md` |
| Cursor | `.cursor/rules/engineering-entry.mdc` | alwaysApply |
| Gemini CLI | `GEMINI.md` | imports `AGENTS.md` |
| GitHub Copilot | `.github/copilot-instructions.md` | surfaces differ |
| Cline | `.clinerules/00-workbench.md` | verify the workspace rule is enabled |

All adapters are thin pointers that define no independent rules. Verify actual
`AGENTS.md` reads in a fresh session; self-reported compliance is insufficient.
Remote Agents cannot read uncommitted local files — commit/push only when the
user asks.
