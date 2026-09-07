# Architecture

This document expands `AGENTS.md`; it does not override it. **MUST** is a gate,
**SHOULD** is the default with a recorded reason for deviation, and **MAY** is a
permitted option.

## System boundary

Workbench is a Go SSR monolith using Gin, GORM, MySQL, server templates, and
browser JavaScript. A feature being complicated is not evidence for a new
service, framework, or microservice. New infrastructure abstractions require a
current repeated need or an actual boundary the existing monolith cannot hold.

```text
HTTP -> Handler -> Service -> Repo -> MySQL
                    |
                    +-> another Service (explicit orchestration)
HTTP <- template/JSON <- Handler
```

A Handler MUST bind protocol input, obtain request identity, call a Service, and
translate the result to HTML/JSON/download responses. It MUST NOT query the DB,
own business transitions, or rely on hidden controls for authorization.

A Service MUST own business rules, object-level read/write authorization, and
the authorization decisions the business model requires, and transaction
orchestration. Object-level authorization is one form those decisions may take
when access depends on ownership, scope, membership, tenant, state,
assignment, or participation; it is not a default for capability-wide
administrative or reference resources. The Service accepts `context.Context`,
not `gin.Context`.

A Repo MUST own database access, query construction, projections, and bounded
persistence operations. It MUST NOT import Gin, decide permission, reinterpret
business status, or make a UI decision. Complex parameterized SQL is valid here.

## Transaction boundary

A Service decides business atomicity: which writes must succeed or fail as one
operation, and therefore where the transaction boundary belongs. The Repo or
another persistence-layer implementation executes the database transaction.
A Service MUST NOT hold, expose, or operate a `*gorm.DB` directly.

This repository does not yet have a certified, uniform cross-Repo transaction
pattern. When an atomic write spans Repos, an Agent MUST first inspect the
current implementations and obtain architecture confirmation. It MUST NOT
invent a `UnitOfWork`, `TransactionManager`, DB accessor, or similar abstraction
as part of an ordinary feature task. This rule defines ownership only; it does
not introduce a transaction framework.

Templates display explicit page data. JavaScript coordinates interaction and
rendering. Neither is an authority for permissions, workflow transitions,
business totals, or data visibility. Server enforcement remains mandatory.

## Modules and references

Simple CRUD modules MAY use a consistent Handler/Service/Repo shape. Complex
actions, dashboards, and workbenches SHOULD be split by capability and may expose
domain-specific method sets. Symmetry is useful only when responsibilities match.

No current module is an authoritative Golden Reference. `internal/module/user/`
is `legacy/reference-not-authoritative`: it mixes form redirects with the former
all-JSON claim, writes ZenTao-compatible MD5 passwords, exceeds the file limit,
and demonstrates ZenTao schema behavior that must not be copied to Workbench
tables. Promotion requires current rule conformance, all gates, and a recorded
decision.

## Files and abstractions

Names SHOULD reveal capability and layer: `todo_handler.go`, `todo_service.go`,
`todo_repo.go`, `todo_service_test.go`. A small module MAY retain `handler.go`,
`service.go`, and `repo.go`. Lower-case underscores are idiomatic and allowed.

First-party Go/JS/CSS/HTML files SHOULD be at most 300 lines and MUST be at most
500. Split by business responsibility, not line ranges. The precise legacy list
is a non-growth baseline; it is debt, not an exception for new files.

Use the simplest current solution. Before adding a manager, engine, provider,
adapter, framework, or common layer, record the repeated behavior or boundary it
resolves. Bug fixes target root causes; fallback branches require evidence.

## Content and structure contract

- Comments MUST describe current behavior and its non-obvious reason, invariant
  or source. Claims such as "batched", "atomic", "authorized", "no N+1",
  "unique truth" and "verified" need matching call-path/test evidence. Copied
  dates, task IDs and upstream filenames are not business confirmation.
- Do not mandate banners in every file. Prefer concise Go doc comments on
  exported contracts and local explanations for surprising logic. Update
  comments affected by a change; remove orphan comments the change creates.
  Do not clean unrelated comments or translate entire modules as a side task.
- A file name declares responsibility. Repo receivers and SQL belong in Repo
  files, even if used by one Service. Authorization/state decisions belong in
  Service; Repo may apply a Service-selected scope predicate and return facts.
  Renaming a file does not cure a layer violation. Service construction must not
  reach through Repo DB fields; assemble dependencies at the composition boundary.
- Split by capability/layer, including callers, script order, tests and exports.
  Do not minify, delete useful comments, or make numbered chunks to pass the line
  limit. New directories need a present responsibility. Do not rename legacy
  files en masse. Small pure domain helpers are valid when they centralize
  repeated behavior and have no DB/HTTP dependency.
- "Single source" requires a consumer search: identify all callers, remove only
  superseded in-scope code, and test parity. An unused helper beside independent
  SQL/mapping implementations is not a completed consolidation.
- Plans, execution logs and review evidence belong in the current `docs/plan/`
  task directory; topology belongs in `docs/operations/`; fixtures in `testdata/`.
  Generated local artifacts stay ignored. Do not put session narratives into
  production code or duplicate business PRDs in tool memory folders.
- Relevant design/status changes update their index and dated evidence.
  Historical audits retain original scope; mark superseded claims explicitly.
  `COMPLETE` refers to a revision/evidence scope, not later WIP, another DB, or
  a running binary by default.

## Capability facts and errors

An authenticated route does not imply all actions on that page are authorized.
Capability arguments MUST be used; use the existing permission source, then
Service object/state checks. UI action descriptors project those rules and do
not replace authorization. Do not invent capability codes or substitute a
page-read capability for a mutation capability.

DB/network failures propagate as errors or explicitly partial responses.
Missing, forbidden, not applicable, empty, unavailable and zero are different
states. Fallbacks require a contract and telemetry; `None` or `0` with nil error
must not conceal infrastructure failure. Unknown business rules stay in task
decision records; implementation labels such as WAIT DECISION are not normal UI.
