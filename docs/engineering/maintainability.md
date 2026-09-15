# Maintainability and anti-bloat

This document expands `AGENTS.md` MUST 11 and MUST 17; it does not override them.
**MUST** is a gate, **SHOULD** is the default with a recorded local reason,
**MAY** is permitted.

## Why this exists

Production operators **cannot** use AI assistants to debug or operate Workbench.
Code written by Cursor, Codex, Antigravity/Gemini, Claude Code, or any other
agent MUST remain understandable and fixable by a human who was not in the
session that produced it. Clever density, speculative "future-proof" layers, and
silent compatibility shims are defects, not quality.

## Fail Fast

- Invalid input, missing permission, missing dependency, or impossible state MUST
  surface as an explicit error (HTTP/JSON envelope, logged cause, or test
  failure). Do not reinterpret failure as empty success, zero counts, or a
  stubbed UI that looks healthy.
- Nested `try` / catch-all / multi-layer fallback that hides the first real error
  is forbidden for new code. A fallback needs present-day evidence that the
  primary path fails in production for a known reason.
- "Best effort" partial responses need an explicit unavailable/error field, never
  a silent omission that looks complete.

## YAGNI and delete-over-wrap

- Do not add features, config knobs, feature flags, environment switches, or
  compat shims that the current task did not require.
- Do not introduce a shared manager / engine / provider / adapter / "common"
  helper for a single call site. Extract only after a stable repeated pattern
  exists (see `AGENTS.md` MUST 9 and MUST 11).
- Prefer deleting unused in-scope code over commenting it out, wrapping it, or
  leaving parallel dead paths. Out-of-scope dead code is reported, not "cleaned"
  opportunistically.
- Prefer narrowing types and branches over adding defensive copies for states the
  domain cannot reach.

## Human-debuggable shape

- Names and control flow SHOULD reveal intent without requiring the generating
  chat. Prefer one obvious path over four clever ones.
- Comments MUST stay true and non-redundant (`architecture.md`). Do not leave
  session narration ("AI added this", "temporary") in committed code.
- File splits follow responsibility and the 300/500 line gate; splitting alone
  does not make bloated logic acceptable.
- Prefer patterns already used in this repository over novel styles from model
  training data.

## Dehydration tasks (behavior-preserving slimming)

When the user asks to dehydrate / slim / 脱水 a module:

1. Scope Gate first: allowed files, forbidden files, acceptance commands.
2. Behavior MUST stay unchanged unless the task explicitly authorizes a behavior
   change. No product "improvements" smuggled into a dehydration.
3. Diff SHOULD reduce net lines in allowed files; a net increase needs a
   present-day reason recorded in the handoff.
4. Remove: unused exports, unreachable branches, duplicate helpers, speculative
   options, commented-out blocks you own in-scope, redundant wrappers that only
   forward once.
5. Do not: rename broadly, reformat unrelated files, invent new abstractions, or
   touch parallel WIP lanes.
6. Handoff MUST list deleted symbols, any behavior risk, and the acceptance
   commands that were run (`git diff --check`, `make check`, and task-specific
   tests/browser checks).

## Tool adapters

Cursor, Codex, Claude Code, Antigravity/Gemini, Copilot, and Cline all load
`AGENTS.md` via thin adapters (`docs/engineering/agent-compatibility.md`). No
adapter MAY redefine or weaken this document. Global user-level instruction
files outside the repo MUST defer to this repository's `AGENTS.md` when the
workspace is Workbench.
