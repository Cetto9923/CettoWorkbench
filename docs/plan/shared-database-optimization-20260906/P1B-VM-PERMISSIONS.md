# P1b-vm Preflight — Permission Matrix

Date: 2026-09-06
Slice: **P1b-vm Preflight only** (read-only probe + design).
Code HEAD at authoring: `3597431f85a783849defc46d273f29d929646b5f`
Status: **READY FOR P1b-vm EXECUTION REVIEW** with blockers listed in §H.

This document does **not** authorize CREATE USER / GRANT / REVOKE. See `grants-workbench-runtime.sql` (template) and `P1B-VM-RUNBOOK.md`.

---

## A. Environment facts (dev/acceptance VM — not production)

| Item | Value | Evidence |
|---|---|---|
| Label | **开发/验收 VM** (PD `vm-zentao` via SSH tunnel) | Tunnel `127.0.0.1:13306` → MySQL; **not** claimed as production |
| Engine / version | MySQL `8.0.46-0ubuntu0.24.04.4` | `SELECT VERSION()` |
| `@@hostname` | `ubuntu-gnu-linux-24-04-3` | read-only |
| Server `@@port` | `3306` | read-only |
| Client path | Mac `127.0.0.1:13306` (ssh -L) | `lsof` |
| Database | `zentaopms` | `SELECT DATABASE()` |
| `@@read_only` | `0` | does **not** prove production primary |
| Isolation | `REPEATABLE-READ` | `@@transaction_isolation` |
| Binlog | `ROW` / `FULL` | `@@binlog_format` / `@@binlog_row_image` |
| Probe account | `zentao@127.0.0.1` (`CURRENT_USER`) | password **[REDACTED]** |
| Session `USER()` | `zentao@localhost` | tunnel presentation |
| Current grants | `USAGE` on `*.*`; **`ALL PRIVILEGES` on `zentaopms.*` and `workbench.*`** | `SHOW GRANTS` — schema-level ALL, not global `*.*` ALL |
| Triggers / views on Workbench-used tables | none found | `information_schema.TRIGGERS` / non-BASE TABLE for scoped list |
| Runtime DDL in Go | **none** | `grep AutoMigrate(`/`CREATE TABLE` in `internal` `cmd` — empty |

Password values are never recorded in this file.

---

## B. Current account facts

| Role | Account | Password | Source | Shared? |
|---|---|---|---|---|
| Workbench `database` (effective local/dev) | `zentao` | **[REDACTED]** | gitignore `configs/config.dev.yaml` when `WORKBENCH_MODE=dev`; host `127.0.0.1:13306`, db `zentaopms` | Likely **SHARED** with ZenTao |
| Workbench `databaseReadonly` (same file) | `zentao` (same endpoint as primary in this env) | **[REDACTED]** | same | Same instance |
| Tracked `configs/config.yaml` | placeholders `CHANGE_ME` / `changeme` | placeholders | Git — **not** runtime | N/A |
| ZenTao PHP (local tree) | `zentao` @ `127.0.0.1` | **[REDACTED]** | `csrcb20-gitfox/zentao/config/my.php` | Treat as **SHARED / DO NOT REVOKE** |
| ZenTao PHP **on VM filesystem** | **UNVERIFIED this session** | — | SSH `vm-zentao` → **No route to host**; could not read deployed `my.php` | **BLOCKER** before execution |

**Policy:** Do **not** REVOKE / ALTER / DROP `zentao@…`. Create new Workbench-only accounts and switch Workbench via `WORKBENCH_*` env.

---

## C. Target account model

| Account | Pool | Privileges |
|---|---|---|
| `workbench_ro@<APP_HOST>` | `databaseReadonly` → `po.NewRepo` **read** (`Repo.db`) | **SELECT only** on §D tables |
| `workbench_rw@<APP_HOST>` | `database` (main) | **SELECT** on §E tables + **minimal DML** on §F |

Denied for both (never grant):

```text
CREATE, ALTER, DROP, TRUNCATE, GRANT OPTION, SUPER, RELOAD, FILE, PROCESS, CREATE USER,
ALL PRIVILEGES, schema-wide *.* or zentaopms.* DML packs
```

---

## D. workbench_ro — SELECT inventory (PO `Repo.db`)

Count: **31** tables. Write privileges: **none**.

| table | ownership | evidence (representative) |
|---|---|---|
| `zt_action` | ZT | done / notice / demand detail |
| `zt_approvalnode` | ZT | todo approval |
| `zt_approvalobject` | ZT | todo approval |
| `zt_bug` | ZT | board metrics / todo / detail |
| `zt_case` | ZT | detail case counts |
| `zt_charter` | ZT | todo approval |
| `zt_demand` | ZT | value stream / board / todo / follow / detail |
| `zt_demandappraise` | ZT | value stream / todo |
| `zt_demandclarify` | ZT | scopes / detail |
| `zt_demandmanagerreview` | ZT | KPI blocked |
| `zt_demandpool` | ZT | detail JOIN |
| `zt_dept` | ZT | detail JOIN (**≠** `zt_depts`) |
| `zt_file` | ZT | demand files |
| `zt_issue` | ZT | board / todo |
| `zt_notify` | ZT | notice list / access |
| `zt_planchange` | ZT | todo approval |
| `zt_product` | ZT | board / detail |
| `zt_project` | ZT | follow / approval |
| `zt_projectbuildguide` | ZT | todo approval |
| `zt_projectweekly` | ZT | follow reports |
| `zt_review` | ZT | todo approval |
| `zt_risk` | ZT | todo |
| `zt_starinfo` | MIX | follow list (**SELECT** on RO; writes on RW) |
| `zt_story` | ZT | board / value stream / detail |
| `zt_task` | ZT | board / todo / detail |
| `zt_team` | ZT | board / access |
| `zt_teamgroup` | ZT | board |
| `zt_testtask` | ZT | todo |
| `zt_todo` | ZT | personal todos |
| `zt_user` | ZT | display names |
| `zt_workbench_notify_reads` | WB-style (not in install.sql) | notice unread JOIN |

`fetchObjectNames` uses a `table` argument; verified call sites only: `zt_demand`, `zt_task`, `zt_bug`, `zt_story`.

---

## E. workbench_rw — SELECT inventory (main pool)

Count: **34** tables (admin + schedule + write-path helpers).

Includes all DML tables below that need SELECT for upsert / `INSERT…SELECT` / auth, plus schedule/auth read-only tables:

`zt_operation_logs`, `zt_login_logs`, `zt_roles`, `zt_role_permissions`, `zt_gf_user_roles`, `zt_menus`, `zt_depts`, `zt_versionwindow`, `zt_versionwindowproduct`, `zt_demandwindow`, `zt_workbench_notify_reads`, `zt_user`, `zt_starinfo`, `zt_story`, `zt_task`, `zt_planstory`, `zt_projectstory`, `zt_productplan`, `zt_demand`, `zt_product`, `zt_project`, `zt_projectproduct`, `zt_holiday`, `zt_config`, `zt_company`, `zt_teamgroup`, `zt_team`, `zt_demandpool`, `zt_dept`, `zt_demandclarify`, `zt_demanduserstory`, `zt_file`, `zt_notify`.

(`zt_login_failures` has no row SELECT in login module; Migrator column probe only — no SELECT grant required for current code.)

---

## F. workbench_rw — DML matrix (minimal)

Soft-delete via `UPDATE` / GORM `DeletedAt` → grant **UPDATE**, not DELETE, unless code uses `Unscoped` / hard `DELETE`.

| table | S | I | U | D | ownership | callers / notes | reclaim |
|---|---|---|---|---|---|---|---|
| `zt_operation_logs` | Y | Y | | | WB | middleware `Create`; admin list SELECT | keep WB |
| `zt_login_failures` | | Y | | Y | WB | `RecordFailure`; `CleanOldFailures` (dead caller, still in source) | keep WB |
| `zt_login_logs` | Y | Y | | | WB | `InsertLoginLog` / list | keep WB |
| `zt_roles` | Y | Y | Y | | WB | soft `deleted=1` via UPDATE | keep WB |
| `zt_role_permissions` | Y | Y | Y | | WB | replace = soft Delete + INSERT | keep WB |
| `zt_gf_user_roles` | Y | Y | Y | | WB | replace = soft Delete + INSERT | keep WB |
| `zt_menus` | Y | Y | Y | | WB | soft `deletedAt` UPDATE | keep WB |
| `zt_depts` | Y | Y | Y | | UNVERIFIED | soft `deletedAt`; not in install.sql | keep / clarify |
| `zt_versionwindow` | Y | Y | Y | | WB | soft delete | keep WB |
| `zt_versionwindowproduct` | Y | Y | Y | Y | WB | `Unscoped` rebuild | keep WB |
| `zt_demandwindow` | Y | Y | | Y | WB | hard delete + INSERT | keep WB |
| `zt_workbench_notify_reads` | Y | Y | | | WB-style | `SaveNoticeRead*` writeDB; RO also SELECT | keep WB |
| `zt_user` | Y | Y | Y | | ZT | admin user module soft delete UPDATE | **P2** |
| `zt_starinfo` | Y | Y | Y | | MIX | `SaveDemandFollow` writeDB | **P2** / PO |
| `zt_story` | Y | Y | Y | | ZT | schedule save | **P2** |
| `zt_storyspec` | | Y | | | ZT | schedule save | **P2** |
| `zt_task` | Y | Y | Y | | ZT | schedule / `*ForStory` | **P2** |
| `zt_taskspec` | | Y | | | ZT | schedule save | **P2** |
| `zt_action` | | Y | | | ZT | schedule / link actions | **P2** |
| `zt_planstory` | Y | Y | | Y | ZT | link / unlink | **P2** |
| `zt_projectstory` | Y | Y | Y | | ZT | UPSERT | **P2** |
| `zt_productplan` | Y | Y | | | ZT | auto-create plan | **P2** |
| `zt_demand` | Y | | Y | | ZT | `UpdateDemandScheduling` | **P2** |

**INSERT counts:** 20 tables with INSERT.
**UPDATE counts:** 14 tables with UPDATE.
**DELETE counts:** 4 tables with hard DELETE (`zt_login_failures`, `zt_versionwindowproduct`, `zt_demandwindow`, `zt_planstory`).

---

## G. Current vs target delta (design only — do not REVOKE old account)

| Current (`zentao@127.0.0.1`) | Target Workbench accounts |
|---|---|
| `ALL PRIVILEGES` on `zentaopms.*` and `workbench.*` | No schema-wide ALL |
| Implicit CREATE/ALTER/DROP on schema | **No DDL** |
| Single shared identity with ZenTao | Dedicated `workbench_rw` + `workbench_ro` |
| Same account for RO + RW pools (this env) | Split identities matching `NewRepo(dbReadonly, db)` |

Implementation path: **create new users → switch Workbench env → verify → leave `zentao` untouched**.

---

## H. BLOCKERS before execution

1. **ZenTao deployed config on VM not readable this session** (SSH no route). Local `my.php` + Workbench `config.dev.yaml` both use `zentao` → treat shared, but confirm on VM before any future discussion of retiring `zentao`.
2. **`<APP_HOST>`** for MySQL user host must match how Workbench connects (tunnel vs socket vs remote).
3. **`zt_depts` / `zt_workbench_notify_reads`** still absent from `db/install.sql` — privileges OK; schema ownership still UNVERIFIED / WB-style.
4. Empty-DB / new env must run **migration account** install for `zt_operation_logs` (and other WB tables) before runtime without DDL.
5. This env’s RO and RW currently share one MySQL endpoint — accounts still should be split so a future true replica cannot receive PO writes.

No Go code change required to proceed to execution review.

---

## I. P1c candidate (do not implement now)

After grants land, add a regression scanner for new ZenTao direct DML outside the temporary whitelist.
