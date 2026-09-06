# P1b-vm Preflight — Permission Matrix

Date: 2026-09-06
Slice: **P1b-vm Preflight Closeout** (still no CREATE USER / GRANT / REVOKE).
Code HEAD at closeout base: `8f5c234a…` → follow-up commit after this edit.
Canonical privilege cells: `grants-workbench-runtime.sql` + `P1B-VM-GRANT-CHECKLIST.md`.

Status after closeout: see final report (`READY` / `BLOCKED`).

This document does **not** authorize CREATE USER / GRANT / REVOKE.

---

## A. Environment facts (dev/acceptance VM — not production)

| Item | Value | Evidence |
|---|---|---|
| Label | **开发/验收 VM** (PD `vm-zentao`) | Tunnel + SSH; **not** production |
| Engine / version | MySQL `8.0.46-0ubuntu0.24.04.4` | `SELECT VERSION()` |
| `@@hostname` | `ubuntu-gnu-linux-24-04-3` | read-only |
| Server `@@port` | `3306` | read-only |
| Acceptance client path | Mac Workbench → `127.0.0.1:13306` tunnel → VM MySQL | `lsof` on `./tmp/workbench_dev` → `127.0.0.1:13306` |
| Database (this grant template) | `zentaopms` | probe + Mac `config.dev.yaml` |
| Isolation / binlog | RR / ROW / FULL | `@@…` |
| Probe account | `zentao@127.0.0.1` | password **[REDACTED]** |
| Current grants | `USAGE` on `*.*`; **ALL PRIVILEGES** on `zentaopms.*` + `workbench.*` | `SHOW GRANTS` |
| Runtime DDL in Go | **none** | `AutoMigrate(` / DDL search empty |

Password values are never recorded here.

---

## B. Current account facts

| Role | Account | Password | Source | Shared? |
|---|---|---|---|---|
| Mac Workbench `database` / `databaseReadonly` | `zentao` | **[REDACTED]** | gitignore `config.dev.yaml`; tunnel `13306`; db `zentaopms` | same identity name as ZenTao |
| Tracked Git `configs/config.yaml` | placeholders | placeholders | not runtime | N/A |
| VM `/opt/workbench/current` | `zentao` | **[REDACTED]** | `database.host=127.0.0.1:3306`, **`dbname=workbench`** (different schema — **out of this zentaopms grant template** unless Review expands) | — |
| ZenTao live PHP | `zentao` | **[REDACTED]** | `/var/www/zentaopms/config/my.php`: host `10.211.55.2`, db `zentaopms` | **SHARED name / DO NOT REVOKE** |

**Hard rule:** old `zentao` account — **NO REVOKE / NO ALTER / NO DROP / NO PASSWORD ROTATION** in P1b-vm execution. Gap about historical retirement is a **legacy-account retirement** concern, not a blocker for creating new Workbench users.

---

## C. Target account model

| Account | Pool | Privileges |
|---|---|---|
| `workbench_ro@<APP_HOST>` | readonly pool | SELECT only (§D) |
| `workbench_rw@<APP_HOST>` | main pool | SELECT (§E) + minimal DML (§F) |

Denied forever for both: CREATE/ALTER/DROP/TRUNCATE/GRANT OPTION/SUPER/RELOAD/FILE/PROCESS/CREATE USER/ALL PRIVILEGES/schema-wide packs.

### APP_HOST_RESOLUTION

```text
current evidence:
  Mac ./tmp/workbench_dev → 127.0.0.1:13306 (tunnel)
  CURRENT_USER()=zentao@127.0.0.1 on zentaopms
expected execution connection path (this slice):
  Mac Workbench → SSH tunnel → VM MySQL (zentaopms)
resolved host: 127.0.0.1
confidence: HIGH for that path
banned: default '%'
note: keep placeholder <APP_HOST> in SQL until execution window substitutes the resolved value
```

---

## D. workbench_ro — SELECT (31)

See checklist. Write privileges: **none**.

---

## E. workbench_rw — SELECT (33)

See checklist. Includes schedule/auth reads + DML tables that need SELECT for upsert / `INSERT…SELECT`.

INSERT-only without RW SELECT (intentional): `zt_action`, `zt_login_failures`, `zt_storyspec`, `zt_taskspec`.

---

## F. workbench_rw — DML (minimal)

| table | I | U | D | notes |
|---|---|---|---|---|
| `zt_operation_logs` | Y | | | middleware INSERT |
| `zt_login_failures` | Y | | **N** | `RecordFailure` only; `CleanOldFailures` **no live caller** |
| `zt_login_logs` | Y | | | |
| `zt_roles` | Y | Y | | soft delete via UPDATE |
| `zt_role_permissions` | Y | Y | | soft Delete + INSERT |
| `zt_gf_user_roles` | Y | Y | | soft Delete + INSERT |
| `zt_menus` | Y | Y | | soft deletedAt |
| `zt_depts` | Y | Y | | soft deletedAt; not in install.sql |
| `zt_versionwindow` | Y | Y | | soft delete |
| `zt_versionwindowproduct` | Y | Y | Y | Unscoped rebuild |
| `zt_demandwindow` | Y | | Y | hard DELETE + INSERT |
| `zt_workbench_notify_reads` | Y | | | writeDB INSERT; RO also SELECT |
| `zt_user` | Y | Y | | temporary ZT |
| `zt_starinfo` | Y | Y | | SaveDemandFollow |
| `zt_story` | Y | Y | | schedule |
| `zt_storyspec` | Y | | | schedule |
| `zt_task` | Y | Y | | schedule |
| `zt_taskspec` | Y | | | schedule |
| `zt_action` | Y | | | schedule |
| `zt_planstory` | Y | | Y | link/unlink |
| `zt_projectstory` | Y | Y | | UPSERT |
| `zt_productplan` | Y | | | auto plan |
| `zt_demand` | | Y | | scheduling fields |

**Counts (must match SQL + checklist):**

```text
RO SELECT: 31
RW SELECT: 33
RW INSERT: 22
RW UPDATE: 13
RW DELETE: 3   # versionwindowproduct, demandwindow, planstory
```

---

## G. Current vs target delta (design only)

Create new users → switch Workbench → leave `zentao` untouched. Do not REVOKE schema ALL from the shared account in this window.

---

## H. Execution prerequisites

| Item | Status |
|---|---|
| Matrix ↔ SQL ↔ checklist identical | required (closeout) |
| APP_HOST resolved for tunnel path | `127.0.0.1` (HIGH) |
| `zt_operation_logs` / `zt_depts` / `zt_workbench_notify_reads` on VM | **exists** |
| Runtime DDL | none |
| Old `zentao` untouched | hard rule |
| VM `/opt/workbench` db=`workbench` | **out of scope** unless Review expands grants to that schema |
| ZenTao host `10.211.55.2` | verified account name `zentao`; retirement later |

P1c candidate remains documentation-only.
