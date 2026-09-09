# Agent entry points and verification

Updated 2026-09-09. Authority remains `AGENTS.md`; this is a loading guide.
Settings can disable/override discovery. File existence is not compliance.

## Repository coverage

- **Codex:** root `AGENTS.md`. This session supplied it and the Agent read it;
  new sessions must verify their own context.
- **Claude Code:** `CLAUDE.md` imports `AGENTS.md` and points to onboarding.
  [Official imports](https://code.claude.com/docs/en/memory).
- **Cursor:** alwaysApply `.cursor/rules/engineering-entry.mdc` plus focused
  rules. `.cursorrules` remains a thin legacy pointer.
- **Gemini CLI:** `GEMINI.md` imports `AGENTS.md`. Inspect `/memory show` and use
  the installed version's documented reload command after changes.
  [Official context guide](https://geminicli.com/docs/cli/gemini-md/).
- **Jules:** root `AGENTS.md`. Remote sessions need a revision containing it.
  [Official entry](https://jules.google/docs/).
- **GitHub Copilot:** `.github/copilot-instructions.md` points to constitution
  and onboarding. Surfaces differ; inspect the active context.
  [Official customization](https://docs.github.com/en/copilot/concepts/prompting/response-customization).
- **Windsurf:** root `AGENTS.md`; no duplicate constitution is needed.
  [Official discovery](https://docs.windsurf.com/zh/windsurf/cascade/agents-md).
- **Cline:** `.clinerules/00-workbench.md`; verify the workspace rule is enabled.
  [Official rules](https://docs.cline.bot/customization/cline-rules).
- **Antigravity:** `.agents/rules/workbench-engineering.md` uses
  `trigger: always_on` to route preflight, concurrent-WIP protection and delivery
  checks to the existing constitution. The local Antigravity 2.12.2 bundled
  customization guide describes unconditional loading for this trigger and
  hierarchical discovery of `AGENTS.md` / `GEMINI.md`; the project rule is an
  explicit execution reminder, not a replacement constitution.
  [Official workspace rules](https://antigravity.google/docs/rules-workflows/).
  Configuration verified 2026-09-09; current-session loading and cold-start
  behavior remain unverified. Do not restart or interrupt an active developer
  merely to test discovery. At its next safe checkpoint, ask it to read this
  rule and complete the read-only preflight below; verify observed file reads
  and the Rules panel, not just an assurance of compliance.
- **WorkBuddy, other tools:** installed-version automatic loading
  was not established here. Use the startup instruction below and verify actual
  reads. Add a thin adapter only after verifying its documented mechanism.
  `.workbuddy/memory/` is historical context, not the engineering entry.

## Cold-start acceptance

For each tool/version used by the team, open a fresh session at repository root:

1. Request read-only preflight. Verify reads of `AGENTS.md`, onboarding and
   task-focused docs; inspect loaded-context UI where available. Self-reported
   compliance alone is insufficient. Do not request hidden prompts/credentials.
2. Verify real branch/HEAD/WIP and applicable MUST rules, including access scope
   and authentication versus capability. IDs unavailable to the tool stay unknown.
3. Inspect a no-edit plan for a hypothetical shared write: field ownership,
   write-time rechecks, lock order, finite work and isolated evidence must appear.
   This proves comprehension, not future obedience.
4. For real changes inspect diff, required checks and negative tests. Record
   tool/version/date/revision, observed reads and pass/fail/unknown in task
   evidence. Repeat after runtime/settings/adapter changes.

This audit prepares entry files; it does not prove fresh-session loading in
every product. Remote Agents cannot read uncommitted local files. Commit/push
requires the user's request; this audit changes no global settings or remote policy.

## Portable startup instruction

```text
Before editing, read AGENTS.md and docs/engineering/agent-onboarding.md from this
repository. Load focused engineering docs and the current task's confirmed
business source. Report branch/HEAD, WIP and exact files read. Tool memory,
prior COMPLETE status and reference projects are not present authorization.
Preserve other contributors' work. Inspect capability/object checks and shared
database concurrency when affected. Run actual required/affected tests. Report
failures and unverified evidence without claiming completion. Do not weaken
rules or baselines to pass. Start with read-only preflight.
```
