# Tech debt governance — 2026-09-14

Execution record for the post-review governance plan (Waves 0–6).

| Wave | Deliverable | Status |
|------|-------------|--------|
| 0 | Live [`docs/engineering/debt.md`](../../engineering/debt.md) | Done |
| 1 | [password-hash-decision.md](password-hash-decision.md) (+ ADR 0001) | Done (no hash rewrite) |
| 2 | dept Service→Repo; `architecture.tsv` = 0 | Done |
| 3 | [wave3-query-revalidate.md](wave3-query-revalidate.md) | Done |
| 4 | `metrics.css` → `web/static/css/metrics/*` + ratchet | Done (other >500 remain) |
| 5 | [wave5-advisory.md](wave5-advisory.md) | Done (inventory 102→91) |
| 6 | [wave6-browser.md](wave6-browser.md) | PARTIAL (auth pages blocked) |

Scope rules: Delete > Merge > Replace; one module cluster per PR where possible;
do not weaken gates; MD5 storage change requires separate product/security auth.
