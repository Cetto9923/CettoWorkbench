# Phase 1 rules audit

> Historical Phase 1 evidence, not a current specification or current test result.
> Use [spec-index.md](spec-index.md) for effective rules; revalidate findings
> against the task's revision before copying them into a new repair plan.

This conflict matrix records rule defects; it does not authorize business edits.

| Priority | Conflict class | Confirmed old problem | Resolution |
|---|---|---|---|
| P0 | Rule/security | Rules prohibited MD5 while mandatory `user` reference creates/resets ZenTao users with MD5. | Prohibit new weak hashes; isolate legacy compatibility; `user` is non-authoritative. |
| P0 | Rule/schema | `BaseModel`/`deletedAt`/audit fields were stated for all business models while ZenTao uses historical fields. | Mandatory Workbench-owned versus ZenTao ownership split. |
| P0 | Agent truth | `AGENTS.md` delegated truth to Cursor files while Claude loaded three competing sources. | `AGENTS.md` is highest; adapters only point to it. |
| P1 | Rule/reality | All writes were declared JSON/PUT/DELETE while `user` uses POST, HTML, Flash, and redirects. | One contract for new fetch mutations; old form endpoints are legacy, not examples. |
| P1 | Golden Reference | `user` was mandatory to copy despite protocol, hashing, size, and schema conflicts. | No Golden Reference until a candidate passes current rules/gates. |
| P1 | Architecture | Rigid CRUD symmetry was applied to action/workbench flows. | CRUD shape only for genuine CRUD; complex flows follow capabilities. |
| P1 | Performance | No rule stopped load/filter/page, N+1, repeated dictionaries, or page query fan-out. | Database rules and debt list govern all four. |
| P1 | Enforcement | File limit, forbidden APIs, architecture, secrets, vet, and status words had no gate. | `make check`, exact finding baselines, advisory detectors, and CI enforce only mechanically reliable regressions. |
| P1 | Scanner truth | Pattern counts allowed one old finding to hide one new finding, while conditional patterns and same-line route checks were treated as unconditional violations. | Stable per-finding fingerprints enforce hard non-growth; contextual patterns and route candidates are advisory. |
| P1 | Completion | PO plan called M1 `verified` while documenting required `go vet` failure. | Required failures force partial/blocked status. |
| P1 | Secrets | Tracked config has non-placeholder sensitive fields and no secret scan. | Values untouched; masked fingerprint gate and debt record added. |
| P2 | Naming | Underscore ban contradicted `*_test.go` and idiomatic capability/layer names. | Lower-case capability/layer underscores allowed. |
| P2 | Tests | Unit/integration/E2E/real-DB boundaries were unclear. | Four explicit test/fixture contracts. |
| P2 | Frontend | Shared rules did not prevent repeated escape, pagination, fetch/error, navigation code. | Frontend boundary and regression patterns added. |
| P2 | Maintainability | Large header blocks and generated symmetry emphasized form over responsibility. | Responsibility boundaries and gates are mandatory; prose ceremony is not. |

At audit time: Go tests and JS syntax passed; six tracked Go files were not
gofmt-clean; `go vet` had one malformed struct-tag diagnostic; multiple
first-party files exceeded 500 lines. Exact debt is in `debt.md` and baselines.
