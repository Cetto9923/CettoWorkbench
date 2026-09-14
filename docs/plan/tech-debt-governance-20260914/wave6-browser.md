# Wave 6 — browser / live acceptance

**Date:** 2026-09-14
**Verdict:** **PARTIAL** — unauthenticated + static assets verified; authenticated dept/metrics/home paths **BLOCKED** (no credentials in session; `:8090` returned 404 for `/metrics/manage` and `/admin/depts` without login cookie — may also be an older binary for those routes).

## Verified

| Check | Result |
|-------|--------|
| `http://127.0.0.1:8090/login` loads | OK — form, `app.js`, `auth/login.js` present |
| `window.appFetch` on login | function (auth layout loads `app.js`) |
| `window.showToast` on login | **undefined** — confirms retained `typeof showToast` guard / no `ui.js` |
| `/static/css/metrics/shared.css` + `metrics.css` shim | HTTP 200 |
| Unit tests (dept CreateEnablingAncestors, listMySQLDemands) | PASS |
| `make check` / architecture / file-length / patterns | run in post-flight |

## Not claimed

- Authenticated metrics manage/radar visual regression after CSS split
- Dept create-with-ancestors UI path
- Home schedule/deliver pagination live query counts
- MD5 / password flows (decision-only; no hash rewrite)

Do **not** treat green `make check` as P0 password debt cleared.

## Gate note (2026-09-14)

Scoped `git diff --check` on Wave 0–5 paths: **pass**.
Full `make check` / `check-whitespace`: **BLOCKED BY EXISTING BASELINE** — trailing whitespace in unrelated `.cursor/plans/release-remaining-master-plan-20260911.md` (pre-existing WIP; not touched by this governance batch).
