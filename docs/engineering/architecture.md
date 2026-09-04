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
transaction orchestration. It accepts `context.Context`, not `gin.Context`.

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
