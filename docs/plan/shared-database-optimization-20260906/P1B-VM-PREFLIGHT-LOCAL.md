# P1b-vm Preflight Re-base (local MySQL 9.6)

Date: 2026-09-06 (Asia/Shanghai)
Status: **Preflight only** — no CREATE USER / GRANT executed.

## Why this re-base

VM `vm-zentao` (PD) was retired between slices. The acceptance path now targets **Mac-local MySQL 9.6** instead of the VM-hosted MySQL 8.0.46 previously documented in `P1B-VM-PERMISSIONS.md` and `P1B-VM-EXECUTION-RESULT.md`. This file refreshes the read-only fact pack and flags what must be re-validated before granting.

This is **not** the production instance. Hostname `MacBook-Air-809.local`. No external ZenTao web is running here.

## A. Environment facts

| Item | Value |
|---|---|
| Engine | MySQL `9.6.0` |
| `@@hostname` / `@@port` | `MacBook-Air-809.local` / `3306` |
| `@@read_only` | `0` |
| Isolation / binlog | `REPEATABLE-READ` / `ROW` / `FULL` |
| Probe identity | `root@localhost` |
| Current grants on `root@localhost` | global `*.*` ALL with GRANT OPTION + full system-admin dynamic privileges |
| Database | `zentaopms` |
| Database engine family | still `InnoDB` (default) — partial diff with VM 8.0; no schema compatibility check performed here |
| Workbench dev process | running locally on `:8093` (Mac) — confirmed live during probe |

## B. Account inventory

| Account | Host | Notes |
|---|---|---|
| `root` | `localhost` | full DBA / system-admin; do not use as runtime identity |
| `zentao` | `127.0.0.1` | present (carried over from VM migration); current schema-level grants on `zentaopms`: **ALL PRIVILEGES** (incl. CREATE/ALTER/DROP). No ZenTao web process running on this host observed during probe. |
| `zentao` | `localhost` | same as above, second host row |
| `test_sha2` | `127.0.0.1` | unrelated; left untouched |
| `mysql.infoschema` / `mysql.session` / `mysql.sys` | system; not modified |

`workbench_rw` / `workbench_ro` do **not** exist yet.

Implication: on this host there is **no active external ZenTao consumer** for `zentao@…`. The shared-account protection rule still applies (no REVOKE / ALTER / DROP / password rotation) until you confirm otherwise.

## C. Schema pre-flight

| Check | Result |
|---|---|
| All 52 grant-target tables present in `zentaopms` | **YES** (52 / 52) |
| `zt_operation_logs` exists | YES |
| `zt_depts` exists | YES |
| `zt_workbench_notify_reads` exists | YES |
| Triggers on target tables | none |
| Non-BASE-TABLE objects (views / external) | none in scope |

## D. Engine delta vs the previously reviewed VM (8.0.46 → 9.6)

What stayed the same:
- `GRANT SELECT, INSERT, UPDATE, DELETE` per-table semantics.
- `mysql.user` columns used by the contract (host / user / auth plugin).

What is different and **must be re-checked** before granting:
- `information_schema.user_privileges` no longer filters per-schema in the same way; the previous read-only probe used `GRANTEE` filters — Agent had to re-query to capture real grants here. **No logic change required**, but reconfirmation per slice is required.
- 9.6 default auth plugin differs across hosts in 9.0+; we will explicitly use `caching_sha2_password` (current `zentao` style) unless you say otherwise.
- 9.6 default character set / collation behavior may change `CREATE TABLE IF NOT EXISTS` outcomes in `install.sql`; not in scope for P1b-vm (no DDL this round).

## E. Path forward

The grant contract (31 / 33 / 22 / 13 / 3) is engine-agnostic enough to apply on 9.6 with the same SQL template (`grants-workbench-runtime.sql`). What changes is:

- **Environment** = "Mac local MySQL 9.6 (post-VM migration)", not "PD VM MySQL 8.0.46".
- **`<APP_HOST>`** = `127.0.0.1` (unchanged; just direct on Mac now, not via SSH tunnel).
- **Rollback credentials** = local `root@localhost` (this is the same identity we used to probe). Documented here so we can always re-run the backup step.
- **Existing `zentao`** stays untouched; we still do not know whether a future local ZenTao web will reuse it. Do **not** touch until you confirm.

## F. Decision (2026-09-06) — Option A

User chose **Option A**:

| Item | Decision |
|---|---|
| Workbench acceptance DB | **Mac local** `127.0.0.1:3306` / `zentaopms` |
| Runtime identity (transition) | keep **`zentao`** via `WORKBENCH_MODE=dev` → `configs/config.dev.yaml` |
| P1b-vm (`workbench_rw` / `workbench_ro` CREATE USER + GRANT) | **PAUSED** — do not execute until re-authorized after Option A is stable |
| Topology fact pack | `docs/operations/dev-vm-topology.md` (local MySQL; VM historical) |

Option A smoke (Agent, unauthenticated): process `:8093` with `WORKBENCH_MODE=dev`, TCP to `:3306`, `GET /login` = 200, PO/schedule paths 303→login. Logged-in browser acceptance remains a human check.

## G. Still open before granting (when P1b-vm resumes)

1. Confirm this Mac is the acceptance environment for the grant window (VM evidence stays historical).
2. Human/DBA runs the local DBA bundle (`P1B-VM-DBA-BUNDLE.md`) with `<APP_HOST>=127.0.0.1`; Agent does not CREATE USER / GRANT.
3. Password generation: local 600 file, `openssl rand`, never paste into chat/Git.

No DB privilege has been changed by this Preflight or by Option A.
