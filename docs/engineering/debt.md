# Existing engineering debt

> **HISTORICAL SNAPSHOT — not an authoritative live count.**
> The line counts, probe results, and "current" figures below were captured at
> the dates shown and are now historical. The live machine-readable source of
> truth is `scripts/quality-baseline/file-length.tsv` (exact non-growth
> baseline) together with the current `make check` / `make check-gates` output.
> Revalidate against those before treating any number here as current.

## Governance in progress — 2026-09-13

Current execution: [code health record](../plan/code-health-20260913/README.md).
Status is PARTIAL: deterministic PO/comment cleanup and two picker test migrations
are implemented. The user-authorized missing picker dependency repair reuses the
existing UI escape helper; frontend regression and 36 quality-gate self-tests pass.
Two remaining test migrations and production file splits are not complete.
`make check` currently stops when the length scanner encounters an unstaged deleted
tracked file; production over-limit files also remain. Demo/audit originals remain
`BLOCKED BY EXISTING BASELINE`. No baseline was expanded and browser acceptance
has not been certified.

## Baseline snapshot — 2026-09-10

`scripts/check-file-length.sh` was probed on commit `cc564036` (branch
`release/po-integrate-main-202609`) before any edits in this debt-cleanup
wave. It **exited with code 1** and printed 38 issues. This commit only
records the snapshot; the gate remains `BLOCKED BY EXISTING BASELINE` until a
follow-up wave either shrinks files below 500 lines or the
`file-length.tsv` baseline mechanism is amended (currently rejected as
"baseline expansion rejected: new over-500-line files cannot be added to
baseline").

Issue categories as captured by the probe:

1. **New over-limit files (7)** — never registered in the historical
   baseline. None are touched by this wave.
   - `internal/module/po/servicefollow.go` (507)
   - `internal/pkg/zentao/client.go` (553)
   - `web/static/css/metrics.css` (683)
   - `web/static/css/po/demand-detail.css` (667)
   - `web/static/css/po/follow.css` (691)
   - `web/static/css/po/personal-workspace.css` (598)
   - `web/static/js/po/notice.js` (568)
2. **Baseline files that grew during WIP (4)** — present in
   `file-length.tsv` but currently larger than the recorded count. These
   are existing-WIP files (per the pre-flight `git status`): the wave does
   not modify them.
   - `web/static/css/components/components.css` (595 → 616)
   - `web/static/css/layout/layout.css` (531 → 643)
   - `web/static/css/schedule/schedule.css` (580 → 596)
   - `web/static/js/ui.js` (775 → 819)
3. **Baseline files that shrank (2)** — `scheduleintegrated.css`
   (879 → 878) and `web/templates/schedule/index.html` (742 → 741).
   The wave deliberately does not ratchet the baseline; shrink without
   lowering the number keeps the gate failing until the owning change is
   ready.
4. **`docs/Demo/` artifacts (≈22)** — **resolved, historical entry.** Two
   independent changes removed this debt class: (a) the current
   `check-file-length.sh` restricts its scope to a `case` whitelist of
   `cmd/ internal/ web/templates/ web/static/js/ web/static/css/ tests/`,
   which excludes `docs/` entirely, so no Demo path is scanned; and (b) the
   Demo tree itself was migrated out of this repository on 2026-09-13 to
   `/Users/yuyan9923/GitHub/CRCBWorkbench/docs/Demo` (175 files,
   SHA-256-verified 1:1), matching `main`, which does not track `docs/Demo`.
   No Demo path remains in the gate scope, in `file-length.tsv`, or in
   `patterns.tsv`. See `docs/plan/code-health-20260913/README.md` for the
   migration record.

This snapshot is the only artifact produced by commit `chore(debt):
register 2026-09-10 file-length baseline failure`. The next wave must
either fix files, amend the baseline mechanism, or carve out a tracked
ignore — none of which is authorized by the current task.

## Current reading note — 2026-09-07

The Phase 1 table below is a **historical inventory**, not a current audit or an
execution order. Do not reopen completed cards from this table alone. Revalidate
each entry against the active task and exact working tree.

Current findings and evidence: [Agent governance audit](../plan/agent-governance-audit-20260907/REPORT.md).
Priority follow-up: reconcile mixed WIP/build failures; repair actual capability
checking; restore reachable regression coverage; remove home unbounded ID
pagination; define/test shared ZenTao write concurrency and runtime budgets.
No business repair was performed by that audit.

Known superseded inventory statements, verified from current source/gates:
- Secret scanner has zero fingerprints; this does not prove credential rotation.
- Explicit schedule capability middleware and corresponding tests now exist.
- Isolated integration and frontend/E2E files exist; their coverage and execution
  must be evaluated, not described as absent.
- Todo/notice SQL query modules exist. Old in-memory helpers alone do not prove
  that current request paths call them; trace callers before filing a regression.
- Current gofmt baseline is zero. Vet/application compilation is blocked by
  current WIP; do not repeat an old malformed-tag diagnosis without revalidation.

## Historical Phase 1 inventory

Phase 1 does not modify the business implementation below. Severity reflects
risk, not authorization to repair. Machine-readable non-growth baselines live in
`scripts/quality-baseline/`.

| Location | Existing problem | Severity | Phase 2 recommendation |
|---|---|---:|---|
| `configs/config.yaml` | Three tracked password fields contain non-placeholder values; values are not reproduced here. | P0 | Rotate credentials, commit placeholders, verify env/secret injection, then remove fingerprints. |
| `internal/module/user/service.go`, `internal/pkg/encode/encode.go` | User create/reset uses MD5 for password writes. | P0 | Confirm ZenTao auth contract, isolate compatibility, design migration, prohibit new MD5 writes. |
| `internal/model/basemodel.go:17` | Malformed GORM struct tag makes `go vet ./...` fail. | P1 | Correct it in a focused change and remove exact vet baseline. |
| `internal/module/user/handler.go` | POST/form/Flash/redirect conflicts with old JSON contract; file exceeds 500 lines. | P1 | Decide/migrate contracts with browser tests, then split by capability. |
| `internal/module/schedule/handler.go` | Schedule routes have group authentication but no explicit capability permission. | P1 | Add capability permissions, Service object checks, and authorization tests. |
| `internal/module/dept/service.go` | Service reaches through `repo.db` and owns a GORM transaction/callback, crossing the Service -> Repo boundary. | P1 | Move transaction-capable persistence behind Repo methods while leaving business orchestration in Service. |
| `internal/module/po/servicetodo.go`, `internal/module/po/repotodo.go` | Todo loads broad sets, filters/sorts/pages in Go, and reloads account dictionary up to three times. | P1 | Create bounded cross-object query plan; SQL-filter/count/page; load dictionary once. |
| `internal/module/po/reponotice.go` | Notifications load all rows, then filter/count/page in memory. | P1 | Move filter/count/page and bounded aggregates to SQL. |
| `internal/module/po/service.go`, `internal/module/po/servicefollow.go`, `internal/module/po/repokpi.go` | Homepage stage loops and per-KPI queries create page query fan-out. | P1 | Measure query plan, consolidate compatible aggregates/ID sets, add contract tests. |
| `internal/module/schedule/service_window.go` | Window lists query capacity, consumed hours, count/stats per window. | P1 | Batch aggregates/capacity inputs and assemble without per-window SQL. |
| `internal/module/user/service.go` | Batch create checks account existence once per row. | P1 | Fetch conflicts once and rely on verified transaction/uniqueness. |
| `internal/module/user/` | Former Golden Reference violates security, protocol, size, schema-boundary rules. | P1 | Keep non-authoritative; promote only a future all-green candidate. |
| `scripts/quality-baseline/file-length.tsv` entries | Multiple first-party Go/JS/CSS/HTML files exceed 500 lines; two exceed 1,100. | P1 | Split one capability at a time with before/after tests. |
| `web/static/js/po/*.js`, schedule scripts | Escape, pagination, direct fetch/error, and new-window behavior repeat locally. | P2 | Inventory contracts; consolidate only stable identical behavior; add browser coverage. |
| Repository tests | Unit tests are sparse; no explicit integration/E2E suites; SQL/permission contracts largely unprotected. | P2 | Add isolated integration fixtures and focused E2E; prioritize rules/permissions/query contracts. |
| `scripts/quality-baseline/gofmt.tsv` entries | Six tracked Go files have formatting debt. | P2 | Format when deliberately touched and remove exact fingerprint. |
| `go.mod:71` | `replace workbench => /home/wds/repo/workbench` couples module resolution to one developer's absolute filesystem path. | P2 | In Phase 2, confirm why the replacement exists and whether it can be removed or replaced portably. |

The complete >500-line list is stored once in the exact non-growth baseline.
