# Wave 3 — query debt revalidation

**Date:** 2026-09-14

| Claim | Verdict | Action |
|-------|---------|--------|
| PO todo in-memory page | FALSE | none |
| PO notice in-memory page | FALSE | none |
| home KPI / follow fan-out | FALSE (KPI/follow/stage cards) | none |
| schedule/deliver stage list ID materialize + Go page | STILL_TRUE → fixed | `FindStageMixedRefsPaged` + `listMySQLDemands` |
| schedule window list per-window SQL | FALSE (batch helpers) | none |
| user BatchCreate per-row ExistsByAccount | STILL_TRUE → fixed | `FindExistingAccounts` once |

Before/after (schedule/deliver list): unbounded ID Pluck ×1–2 + Go slice → `COUNT(*)` + `LIMIT/OFFSET` on UNION ALL.

Before/after (batch create): N× `ExistsByAccount` → 1× `account IN (?)`.
