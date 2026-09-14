# Engineering debt ledger

> **Live ledger** (refreshed 2026-09-14, tech-debt governance Wave 0).
> Machine truth: `scripts/quality-baseline/*` + current `make check`.
> This file is the human entry; numbers here must match those baselines after revalidation.

Governance plan: Cursor plan「技术债治理方案」; execution notes under
`docs/plan/tech-debt-governance-20260914/` when present.

## Current open items

| Sev | Item | Live evidence | Status |
|-----|------|---------------|--------|
| P0 | ZenTao-compatible MD5 password write/verify | `patterns.tsv` `WEAK_PASSWORD_HASH` ×7 (`encode`, `user`, `profile`, `login`); see [password-hash decision](../plan/tech-debt-governance-20260914/password-hash-decision.md) | **DECIDED / migration pending** — no silent algorithm change |
| P1 | Service → DB boundary | `architecture.tsv` empty (0 files) after Wave 2 | **CLOSED** |
| P1 | Query / pagination debt | Revalidated 2026-09-14: todo/notice/window/KPI/follow OK; schedule/deliver list + user batch create fixed — [wave3](../plan/tech-debt-governance-20260914/wave3-query-revalidate.md) | **CLOSED** (residual advisories on unrelated paths remain P2) |
| P1 | Files >500 lines | `file-length.tsv` — 26 entries after Wave 4 removed `metrics.css` (2154→split under 500). Remaining hot: `schedule/form.go` 935, `scheduleintegrated.css` 929, `po-profile.js` 856… | **OPEN** (metrics cluster closed; continue by file) |
| P2 | Advisory inventory | `patterns.tsv` ~91 after Wave 5 shrink (was ~102); login `DIRECT_PAGE_FETCH` retained intentionally | **PARTIAL** |
| P2 | auth toast guard | `web/templates/layout/auth.html` does not load `ui.js`; `app.js` / `auth/login.js` keep `typeof showToast` | **RETAINED** (intentional; optional load of `ui.js` needs separate auth) |

## SUPERSEDED (do not reopen from old narrative)

| Former claim | Why superseded |
|--------------|----------------|
| Schedule routes lack capability middleware | `schedule/handler.go` uses `RequirePerm(perm.Schedule*)` |
| `configs/config.yaml` tracked real passwords | Placeholders `changeme` only; `secrets.tsv` empty |
| Malformed `basemodel` GORM tag / go-vet fail | `go-vet` / `gofmt` baselines empty or clean under current tree |
| `go.mod` absolute `replace workbench => /home/...` | No `replace` directive in current `go.mod` |
| `docs/Demo/` file-length / patterns debt | Demo tree removed from this repo; length scanner excludes `docs/` |
| Production JS `alert(` / esc-probe / badge–PAGE_SIZE thin wrappers / picker `catch→[]` | Slim 2026-09: production probes cleared; auth toast guard retained as above |
| “No schedule capability / no integration tests at all” (old audit prose) | Capability middleware + isolated test files exist; coverage quality is separate |

## Frozen / out of casual slim batches

- FollowScope constants (business-facing; comment-only cleanup already done)
- Full-repo file-header reformatting
- Metrics **business** semantics / radar formula rewrites without product auth
- Weakening AGENTS / Gate / expanding `file-length.tsv` to absorb growth
- Promoting `internal/module/user/` to Golden Reference

## Baseline pointers

| Gate artifact | Role |
|---------------|------|
| `scripts/quality-baseline/file-length.tsv` | Exact non-growth line counts for >500 files |
| `scripts/quality-baseline/architecture.tsv` | Exact Handler/Repo/Service boundary debt fingerprints |
| `scripts/quality-baseline/patterns.tsv` | Pattern hits (blocking + advisory) |
| `scripts/quality-baseline/secrets.tsv` | Secret fingerprints (must stay empty of real secrets) |
| `scripts/quality-baseline/gofmt.tsv` / `go-vet.txt` | Format / vet debt |

## Historical appendix

Older dated snapshots (2026-09-07 Phase 1 inventory, 2026-09-10 file-length probe prose,
2026-09-13 code-health PARTIAL notes) are **not** current counts. Prefer this open-items
table and the TSV baselines. Prior narrative remains in git history and
`docs/plan/code-health-20260913/` if needed for archaeology.
