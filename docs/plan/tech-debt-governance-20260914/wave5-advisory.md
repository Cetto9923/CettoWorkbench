# Wave 5 — advisory convergence

**Date:** 2026-09-14

## DIRECT_PAGE_FETCH

| File | Action |
|------|--------|
| `metrics-radar.js` / `metrics-manage.js` | → `window.appFetch` |
| `dept/list.js` (2 call sites) | → `window.appFetch` |
| `auth/login.js` | **retained** raw `fetch` — cookie race + documented; auth shell may lack full UI stack |

Other pages still using `window.appFetch \|\| fetch` remain advisory until follow-up (fallback still matches the scanner).

## ROUTE_PERMISSION_REVIEW

| Item | Action |
|------|--------|
| `metrics/handler.go` | Inlined `middleware.RequirePerm` (was `permMW` false positive) |
| `check-patterns.sh` | Also skip `RequireAnyPerm` on the same route line (real middleware, not a weaken) |
| Remaining advisories | Routes without capability on the registration line (e.g. debug/superadmin, group-level MW) — inventory shrunk via `--emit-baseline` |

## Baseline

`patterns.tsv`: 102 → 91 advisories (resolved entries removed; hard still 0).
