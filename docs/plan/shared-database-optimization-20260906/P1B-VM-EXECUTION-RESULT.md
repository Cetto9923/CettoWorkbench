# P1b-vm Execution Result

Date: 2026-09-06 (Asia/Shanghai)
Status: **P1b-vm BLOCKED** (no CREATE USER / GRANT executed)

## Baseline

| Item | Value |
|---|---|
| Branch | `Claude-PO` |
| HEAD | `f1d1d71999b205e3bf3f846d0c1912ae072aa6aa` |
| Working tree at start | clean |
| GitHub CI | `regression-gates` = success; `integration-tests` = success (run 34017400345) |
| Environment | Mac acceptance → `127.0.0.1:13306` SSH tunnel → VM MySQL |
| Database | `zentaopms` |
| Engine | MySQL `8.0.46-0ubuntu0.24.04.4` |
| `@@hostname` / `@@port` | `ubuntu-gnu-linux-24-04-3` / `3306` |
| Isolation | `REPEATABLE-READ` |
| Probe identity | `zentao@127.0.0.1` (password **[REDACTED]**) |
| Target APP_HOST | `127.0.0.1` |
| Intended accounts | `workbench_rw@127.0.0.1`, `workbench_ro@127.0.0.1` |

## Pre-flight checks completed (read-only)

| Check | Result |
|---|---|
| Tunnel listening | YES |
| Database is `zentaopms` (not `workbench`) | YES |
| Grant-target tables (52) all exist | YES |
| Required tables `zt_operation_logs` / `zt_depts` / `zt_workbench_notify_reads` | EXISTS |
| Permission contract 31/33/22/13/3 | unchanged (not applied) |
| Legacy ZenTao account touched | **NO** |

## Accounts / grants

| Item | Result |
|---|---|
| `workbench_rw` created | **NO** |
| `workbench_ro` created | **NO** |
| GRANT applied | **NO** |
| Workbench switched to new accounts | **NO** |
| Canary / smoke | **NOT STARTED** |
| Rollback required | **NO** (nothing changed) |
| Legacy `zentao` modified | **NO** |

## Blocker

```text
BLOCKED: no authorized DBA identity
```

Evidence:

1. Current Workbench/probe account `zentao@127.0.0.1` has only:
   - `USAGE ON *.*`
   - `ALL PRIVILEGES ON zentaopms.*`
   - `ALL PRIVILEGES ON workbench.*`
2. Explicit probe: `CREATE USER ...` → **ERROR 1227** — needs `CREATE USER` privilege.
3. Cannot list `mysql.user` with `zentao` (ERROR 1142).
4. On VM (`parallels`):
   - `sudo mysql` requires interactive password (`sudo -n` fails).
   - `/etc/mysql/debian.cnf` exists but is **not readable** (`root:root` 600).
   - `mysql -uroot` via socket → Access denied (auth_socket / no password path).
5. Per execution rules: **no** skip-grant-tables, root password reset, auth-plugin changes, or other security bypasses.

## What is needed from operator

Provide an approved DBA/migration identity that can:

```text
CREATE USER
GRANT ... ON zentaopms.<table>
SHOW GRANTS
```

Examples (operator choice):

- passwordless/scripted `sudo mysql` for this window
- temporary DBA credential delivered via approved secret channel (not chat plaintext if avoidable)
- run CREATE USER + GRANT yourself, then re-authorize Agent for switch/smoke only

Then re-run P1b-vm Execution from Step 8 onward (accounts may already exist — Agent must STOP if pre-created and wait for confirmation).

## Out of scope confirmed

```text
production untouched
/opt/workbench untouched
workbench schema untouched
legacy zentao account untouched
P1c not started
P1d not started
P2 not started
permission matrix not expanded
```

## Conclusion

```text
P1b-vm BLOCKED / ROLLED BACK
```

(No DB privilege changes were made; rollback of application config was unnecessary.)
