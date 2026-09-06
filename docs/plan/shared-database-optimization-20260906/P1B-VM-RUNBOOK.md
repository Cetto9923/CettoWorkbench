# P1b-vm Runbook (execution — after Review)

Companion artifacts:

- `P1B-VM-PERMISSIONS.md`
- `P1B-VM-GRANT-CHECKLIST.md`
- `grants-workbench-runtime.sql` — **REVIEW TEMPLATE ONLY / DO NOT EXECUTE AUTOMATICALLY**

Target for **this** runbook: **开发/验收** path = Mac Workbench → SSH tunnel → VM MySQL database **`zentaopms`**.

Not covered unless Review expands: `/opt/workbench/current` using schema **`workbench`**.

---

## EXECUTION GATE

All must be true before any CREATE USER / GRANT:

- [ ] Permission matrix counts == GRANT SQL == `P1B-VM-GRANT-CHECKLIST.md`
- [ ] `<APP_HOST>` substituted with resolved host (`127.0.0.1` for tunnel path); **no `%`**
- [ ] Account names confirmed: `workbench_rw` / `workbench_ro`
- [ ] Target database confirmed: `zentaopms`
- [ ] Required tables exist on that DB: `zt_operation_logs`, `zt_depts`, `zt_workbench_notify_reads` (+ other grant targets)
- [ ] No runtime DDL in application
- [ ] New passwords generated outside Git by DBA/secure store
- [ ] Old `zentao` account: **no REVOKE / ALTER / DROP / rotation**
- [ ] Rollback credentials for current Workbench identity still available
- [ ] Human architecture Review approved execution

If any box fails → **stop**.

---

## Step 0 — Capture baseline (read-only)

1. Record Workbench config source (`WORKBENCH_MODE`, env vs file) and open TCP path (`lsof`).
2. `SHOW GRANTS` for current users (passwords **[REDACTED]** in notes).
3. Confirm ZenTao PHP user remains `zentao` (already verified on VM); still **do not modify it**.
4. Snapshot engine / isolation / binlog / database name.
5. Confirm `zt_operation_logs` exists; if missing, migration account applies install SQL **before** no-DDL runtime users.

---

## Step 1 — Create Workbench-only users (DBA)

```text
workbench_rw@127.0.0.1   # or other resolved APP_HOST
workbench_ro@127.0.0.1
```

Passwords: `<GENERATE_STRONG_PASSWORD_OUTSIDE_GIT>`. Never reuse ZenTao account.

---

## Step 2 — Apply reviewed grants

1. Copy SQL template; replace `<APP_HOST>` with resolved host only.
2. Static check: no ALL PRIVILEGES / GRANT OPTION / REVOKE / DROP USER / ALTER USER / real passwords / `%` host.
3. DBA executes under privileged migration identity.
4. `SHOW GRANTS` for both new users.

---

## Step 3 — Point Workbench at new accounts

Env only (`WORKBENCH_DATABASE_*` / `WORKBENCH_DATABASEREADONLY_*`). Do not write secrets into tracked YAML.

---

## Step 4 — Restart Workbench

Expect schema check pass; no CREATE/ALTER; no permission denied on boot.

---

## Step 5 — Smoke (fixture data only)

Startup / login / PO read / PO write (follow + notice read) / schedule read / schedule write on disposable fixtures.

---

## Step 6 — Negative tests

`workbench_ro`: SELECT ok; INSERT/UPDATE/DELETE/CREATE denied.
`workbench_rw`: whitelist DML ok; DDL denied; INSERT into SELECT-only table (e.g. `zt_product`) denied. Prefer transaction rollback / fixtures.

---

## Step 7 — Observe

ZenTao still on unchanged `zentao`. Workbench logs clean.

---

## Step 8 — Rollback

Repoint `WORKBENCH_*` to previous still-valid credentials; restart. Fix missing GRANTs on new accounts — do not restore ALL PRIVILEGES as default. Leave ZenTao untouched.

---

## Explicit non-goals

No REVOKE of `zentao@…`; no P1c/P1d/P2; no Git password rewrite.
