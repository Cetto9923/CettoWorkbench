# Performance debt plan — board, done, schedule

> Scope: Plan §5 轨 E (PERF-1 / PERF-2 / PERF-3 / PERF-4 / PERF-5)
> Source: GLM code-review report `code-review-2026-09-10.html` §7
> Created: 2026-09-10 · Branch `release/po-integrate-main-202609`
> State: **design only** — no SQL changes in this wave.

This document records the agreed scope, the queries to consolidate,
the contracts that must hold before and after, and the verification
plan. It is intentionally separate from `debt.md` so the staged
implementation (W3+) can ship incremental plans that re-verify
each piece against the same evidence rather than against a single
ambitious rewrite.

## Scope map

| ID | Surface | Current shape | Target shape | Wave |
|----|---------|---------------|--------------|------|
| PERF-1 | 看板指标卡 (board/metrics) | per-metric SQL fan-out (5N) | 1–2 GROUP BY queries, owners IN-batched | W3 |
| PERF-2 | 已办详情页 (done/detail) | ~12 SQL/page, object context per row | batched object lookup, optional facet/approval cache | W3 |
| PERF-3 | 看板树 (board/demand/items) | children/stories unbounded | bounded children/stories with documented upper limit | W3 |
| PERF-4 | 时间比较 + FIND_IN_SET | function-wrapped columns; FIND_IN_SET in WHERE | range comparisons on indexed columns; FIND_IN_SET = long-term debt | W3+ |
| PERF-5 | misc (PERF-1/2/3/4 之外) | various | log only, no scheduled work | backlog |

Each row maps to **one focused change with before/after SQL count
and rows-examined evidence**; they are **not bundled**.

## Why design-only here

- The cleanup wave (B/C/D) must not bake a SQL change into a
  release before the user's WIP settles.
- W3+ waves run against an isolated environment (per
  `docs/engineering/database.md` §"Evidence"), not the live DB.
- Performance claims require measured before/after SQL counts,
  rows examined/returned, representative plans, and pool wait —
  none of which can be collected from this branch's working tree
  without taking the system off the user's current flow.

## PERF-1 — board metrics fan-out

### Current shape (from GLM §7 PERF-1)

`FindBoardTeamMetrics` runs **per-metric** queries against the
shared ZenTao DB:

- one query per metric name × teamgroup;
- repeated dictionary reads for owner names;
- no IN-batching across teamgroups.

Roughly **5N** SQL per page request, where N is the number of
metric cards visible.

### Target shape

1. Aggregate metrics in **at most two GROUP BY queries**, one for
   numeric counts (delivered / in-progress / blocked / over-2-week
   not scheduled) and one for ratios (predictability, gate pass
   rate, defect closure rate, defect response time).
2. Owner names loaded **once per request** in a single `WHERE
   account IN (...)` query; the result map is shared with other
   page handlers via a request-scoped cache (not process-global).
3. The metric calculation lives in the **Service layer**; the
   Repo only returns the raw aggregations. Calculation in SQL is
   allowed only when it eliminates a separate round-trip; any
   SQL that recomputes what the page could compute in Go is a
   layer-placement defect.

### Acceptance contract

- `make check` baseline unaffected (no new advisories).
- `go test ./internal/module/board/...` passes, including a new
  contract test that asserts:
  - number of SQL calls <= 2 per metric page (verified with the
    existing `DATA-DOG/go-sqlmock` test fixture, or a new
    `testdata/` integration case if sqlmock is insufficient);
  - returned metric values match the historical numbers for a
    captured dataset.
- A `design-evidence/` note recording before/after:
  - SQL count per request
  - rows examined/returned per query
  - representative EXPLAIN plan
  - p50/p95 latency in the isolated env
  - dataset cardinality / owner count
- No behavior change visible to a logged-in user: same metric
  numbers, same refresh cadence, same UI.

## PERF-2 — done-detail SQL fan-out

### Current shape (from GLM §7 PERF-2)

`DoneDetail` issues roughly **12 SQL queries per page** to assemble
the object's approval / sidebar / facet / status context. Most are
single-row lookups issued sequentially.

### Target shape

1. Group queries by **target entity type** (action, demand,
   story, task, product, dept) and batch by primary key.
2. Facet / approval lookups become a single joined query or are
   loaded in parallel (errgroup) with a documented total budget.
3. Sidebar / label / owner dictionaries load once and are reused
   across the request lifetime.

### Acceptance contract

Same shape as PERF-1, applied to `done/detail`. The contract test
must assert **<= 4 SQL queries per request** for the detail page
when the object exists, plus the rows-examined comparison note.

## PERF-3 — board tree bound

### Current shape (from GLM §7 PERF-3)

`loadDemand` / `loadTasks` walk the demand tree without
`LIMIT`, so children/stories can grow without bound for a hot
teamgroup.

### Target shape

1. Add `LIMIT ?` (configurable, default 200) on children and
   stories at the Repo layer; surface a `truncated: bool` flag in
   the response so the UI can warn.
2. Document the bound in `docs/engineering/spec-index.md` →
   `module-index.md` for `internal/module/board/`.
3. The bound is a **contract**, not a target — the page must keep
   working at the bound (no missing rows visible without the
   `truncated` flag).

### Acceptance contract

- New unit test: tree of 1000 stories under one demand returns
  ≤ 200 children rows + `truncated: true`.
- New contract test: with truncation flag set, the UI exposes a
  banner; the test asserts the banner element is in the rendered
  HTML.

## PERF-4 — function-wrapped columns + FIND_IN_SET

### Current shape (from GLM §7 PERF-4)

Some pages compare time columns via function calls (e.g. `DATE()`
wrapping) and use `FIND_IN_SET` for `path` lookups. Both defeat
indexes.

### Target shape

1. Replace `DATE(col) = ?` predicates with **half-open range**
   comparisons `col >= ? AND col < ?`; these are index-friendly.
2. For `FIND_IN_SET(id, path)` lookups, document the trade-off
   (full scan) and link to the long-term replacement using a
   join on a normalized ancestor table; this remains **debt, not
   a fix in this wave**.

### Acceptance contract

- EXPLAIN shows the range predicate uses the relevant index.
- The `FIND_IN_SET` calls remain in `debt.md` with an owner and a
  deprecation horizon; no replacement is required for W3+.

## PERF-5 — backlog

Logged only. No scheduled work in W3+. Examples from the GLM
report that fall into this bucket:

- cosmetic latency from JSON serialization of large trees
- dashboard cache TTL tuning
- per-request logger overhead in hot paths

These are **not** part of any current wave and are not required
for `make check` green.

## Cross-cutting rules (apply to every PERF change)

1. **No new N+1.** A method named "batch" with one query per ID
   is still N+1; consolidate or document why a single SQL cannot
   replace it.
2. **No goroutine fan-out.** Parallel queries are a last resort
   when the SQL plan is already optimal; if you reach for
   goroutines, the SQL is wrong.
3. **No unauthorized caching.** Replica topology and ZenTao
   permissions must be re-verified before any cache shortcut.
4. **No behavior regression.** A perf change that drops a column,
   renames a label, or reorders a list is a **separate change**,
   not a side effect.
5. **Every new test has an execution entry.** Per AGENTS.md
   MUST 16, run the test before claiming done; report the exit
   code and the case coverage.
6. **No drive-by refactor.** Touch only the queries named in the
   scope map row; report other findings, do not fix them.

## Verification chain (per change)

```
1. Capture baseline:
   - SQL count (count of * calls per request, not just statements)
   - EXPLAIN plan + rows examined
   - p50/p95 latency in isolated env (N=200 mixed read/write)
   - Pool wait (gorm/sql.DB.Stats().WaitCount delta)
2. Implement scoped change (one surface only).
3. Re-measure under the same load; capture before/after note in
   design-evidence/<id>-2026MMDD.md.
4. Run targeted contract test; report exit code.
5. Run `make check` and `go test ./...`; report exit codes.
6. Update debt.md if any new entry becomes debt; do NOT silently
   resolve debt (MUST 12).
```

## References

- `code-review-2026-09-10.html` §7 — original findings
- `docs/plan/agent-governance-audit-20260907/REPORT.md` — repo
  fan-out history
- `docs/engineering/database.md` §"Shared ZenTao database and
  concurrent writes" — finite work, lock order, evidence
- `docs/engineering/quality.md` — delivery truth, baseline rules
- `docs/engineering/debt.md` — current debt register
- Plan §5 — original scope and reclassification

## Reversal / supersession

This document is superseded when a PERF change ships and its
evidence note replaces the matching row in the scope map with a
link to the design-evidence/ file. The scope map itself is the
running register; do not duplicate it in `debt.md`.