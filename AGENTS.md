# workbench engineering constitution

`AGENTS.md` is the repository-wide canonical engineering source for Codex,
Claude Code, Cursor, and other coding agents. Tool adapters may summarize how
to load it, but MUST NOT redefine or weaken it. Conflicts are resolved in this
order: this file, `docs/engineering/`, file-scoped `.cursor/rules/*.mdc`, then
task/design documents. Existing code is evidence, not a rule.

## Project and scope

This repository is a Go/Gin/GORM/MySQL server-rendered monolith. Keep it a
monolith unless an approved task explicitly changes that decision. The normal
flow is Handler -> Service -> Repo -> database; templates and browser JavaScript
are presentation and interaction layers.

Rules use these meanings:

- **MUST**: correctness, security, architecture, performance, or delivery gate.
- **SHOULD**: preferred default; deviations need a concrete local reason.
- **MAY**: permitted choice, not a requirement.

Only the current task scope may be changed. Unrelated findings MUST go to
`docs/engineering/debt.md`; never fix them opportunistically. Preserve unrelated
dirty/untracked work. Do not create, switch, merge, rebase, or push branches
unless the user explicitly asks. Never write on `main` or `master`.

## Mandatory pre-flight

Before editing, confirm: safe current branch; task goal and allowed diff; relevant
module/design docs; data source and schema ownership; whether any reference or
shared component is actually applicable; DB/permission/performance impact; and
the commands that prove acceptance. Stop and surface unresolved architecture or
data ambiguity instead of inventing a contract.

## MUST rules

1. **Layers.** Handlers bind protocol input, call Services, and render/return HTTP
   responses. Services own business rules, object authorization, and transaction
   orchestration. Repos own database access and query shaping. Repos MUST NOT
   decide permissions or business state; Handlers MUST NOT access the DB; browser
   code MUST NOT become the source of business truth.
2. **Module shape.** CRUD conventions apply only to genuinely simple CRUD.
   Action/workbench modules follow capabilities and business flows; they MUST NOT
   be forced into symmetric CRUD methods or copied from another module.
3. **Authorization.** Protected routes require authentication and the applicable
   capability permission. Object-level authorization is repeated in Service
   methods for reads and writes. A hidden button is never authorization.
4. **Write protocol.** New browser mutations use CSRF-protected `fetch`, JSON,
   the real HTTP method (`POST`/`PUT`/`DELETE`), and a consistent JSON envelope.
   File downloads and explicitly documented external callbacks are exceptions.
   Existing form/Flash/redirect endpoints are legacy contracts, not examples.
5. **Queries.** List data defaults to SQL filter -> SQL sort -> SQL count -> SQL
   pagination. No unbounded large-table load followed by Go filtering/pagination,
   avoidable N+1, repeated same-request dictionary reads, query-per-metric fan-out
   without a page query-plan review, `SELECT *`, unnecessary large fields, or SQL
   string interpolation. Parameterize values.
6. **Schema ownership.** Workbench-owned tables follow Workbench audit/soft-delete
   conventions. Existing ZenTao tables follow their real schema (`deleted`,
   `fromDemand`, `action`, and other historical fields). Never reshape ZenTao
   semantics to fit `BaseModel` or Workbench naming conventions.
7. **Security.** Never commit a real password, token, API key, secret, or private
   key. Config in Git contains placeholders only; runtime values come from env or
   an approved secret mechanism. New password writes MUST NOT use MD5/SHA family
   hashes. Legacy ZenTao verification compatibility must be isolated and may not
   be copied into Workbench-owned authentication.
8. **Files.** Use idiomatic lower-case Go names. Capability/layer names such as
   `todo_handler.go`, `todo_service.go`, `todo_repo.go`, and
   `todo_service_test.go` are valid. Split by responsibility, not arbitrary line
   chunks. First-party Go/JS/CSS/HTML files SHOULD stay within 300 lines and MUST
   not exceed 500; exact legacy exceptions may shrink but may not grow.
9. **Frontend.** Reuse established shared fetch/error, toast, modal, pagination,
   dropdown, form, CSRF, and loading/empty/error-state capabilities. Extract a
   shared abstraction only after a stable repeated pattern exists. Internal
   navigation opens in the current page by default; a new window needs an
   explicit product reason and safe `noopener` handling.
10. **Testing.** Go unit tests live beside the package as `*_test.go`. DB/HTTP or
    multi-module integration tests use the integration system; browser flows use
    E2E; fixtures use `testdata/`. Do not disguise a real-DB test as a unit test or
    move all Go tests into a generic `tests/` directory.
11. **Simplicity.** Prefer the smallest maintainable change and existing mature
    components. A new framework, manager, engine, provider, adapter, or common
    layer needs present-day duplication/boundary evidence. Fix root causes; do not
    stack fallbacks and special cases. No drive-by refactor.
12. **Delivery truth.** `done`, `verified`, `ready`, `complete`, or `可交付` may be
    used only when every required gate and task-specific acceptance gate passes.
    Otherwise report `partial`, `failed`, or `blocked`. A known baseline failure
    is reported as `BLOCKED BY EXISTING BASELINE`, never explained away.

## Golden Reference policy

There is currently **no authoritative module Golden Reference**. In particular,
`internal/module/user/` is `legacy/reference-not-authoritative`: it conflicts
with current rules in password hashing, write protocol, file size, and ZenTao
schema assumptions. Agents MAY study a module for a narrowly verified pattern,
but MUST validate it against this constitution, the real schema, and machine
gates before reuse. A module can be promoted only after all current rules and
required gates pass and the decision is recorded in `docs/engineering/`.

## Required references and gates

Read the focused documents under `docs/engineering/` before affected work:
`architecture.md`, `database.md`, `frontend.md`, `testing.md`, and `quality.md`.
Cursor file rules load these boundaries by file type. Before handoff, run:

```sh
git diff --check
make check
```

For UI changes, automated gates are necessary but not sufficient: verify the
real authenticated browser interaction, visual result, Console/network errors,
and the actually loaded asset revision.

## Mandatory post-flight

Confirm the diff is in scope; no duplicated implementation or layer violation
was added; query changes do not introduce N+1, in-memory pagination, or repeated
same-request reads; no secret was introduced; required automated and acceptance
gates ran. Report every failed/skipped gate and do not claim completion when one
is missing.
