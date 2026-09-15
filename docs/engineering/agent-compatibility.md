# Agent entry points

`AGENTS.md` is the only normative rule source; this table lists each tool's
discovery file. File existence does not prove loading.

| Tool | Entry | Notes |
|---|---|---|
| Codex | root `AGENTS.md` | User-level `~/.codex/AGENTS.md` MUST defer to the repo file when present |
| Claude Code | `CLAUDE.md` | imports `AGENTS.md` |
| Cursor | `.cursor/rules/engineering-entry.mdc` | alwaysApply |
| Antigravity / Gemini | `GEMINI.md` | imports `AGENTS.md`; Antigravity uses the Gemini adapter path |
| GitHub Copilot | `.github/copilot-instructions.md` | surfaces differ |
| Cline | `.clinerules/00-workbench.md` | verify the workspace rule is enabled |
| Jules / Windsurf | root `AGENTS.md` | remote sessions need a revision containing it |

All adapters are thin pointers that define no independent rules. Verify actual
`AGENTS.md` reads in a fresh session; self-reported compliance is insufficient.
Remote Agents cannot read uncommitted local files — commit/push only when the
user asks.

User-level extras (Obsidian context, model memories, global AGENTS) MUST NOT
override repository MUST rules. If a global file conflicts, stop and surface the
conflict.
