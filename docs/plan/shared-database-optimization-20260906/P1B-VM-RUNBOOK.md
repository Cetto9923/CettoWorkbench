# P1b-vm Runbook (execution — after Review)

**Preflight only produced this file.** Do **not** run CREATE USER / GRANT until architecture Review signs off.

Companion artifacts:

- `P1B-VM-PERMISSIONS.md` — matrix and environment facts
- `grants-workbench-runtime.sql` — **REVIEW TEMPLATE ONLY / DO NOT EXECUTE AUTOMATICALLY**

Target environment: **开发/验收 VM** (not production unless separately authorized).

---

## Step 0 — Capture baseline (read-only)

1. Record Workbench config source actually used (`WORKBENCH_MODE`, env vs file).
2. `SHOW GRANTS` for current Workbench and ZenTao DB users (password **[REDACTED]** in notes).
3. Confirm ZenTao PHP DB user on the **VM filesystem** (`config/my.php` or env). If still unverified, **stop** — do not touch `zentao@…`.
4. Snapshot: engine version, `@@transaction_isolation`, `@@binlog_format`, database name.
5. Confirm `zt_operation_logs` exists (P1a). If missing, migration account runs `db/install.sql` fragment **before** switching to no-DDL runtime users.

---

## Step 1 — Create Workbench-only users (DBA)

Create:

```text
workbench_rw@<APP_HOST>
workbench_ro@<APP_HOST>
```

Passwords: generate outside Git (`<GENERATE_STRONG_PASSWORD_OUTSIDE_GIT>`).
**Do not** reuse the ZenTao application account.

---

## Step 2 — Apply reviewed grants

1. Replace `<APP_HOST>` in `grants-workbench-runtime.sql`.
2. Human Review confirms: no `ALL PRIVILEGES`, no `GRANT OPTION`, no REVOKE of ZenTao account, no real secrets.
3. DBA executes CREATE USER (if not done) + GRANT statements under a privileged migration/admin identity.
4. `FLUSH PRIVILEGES` if required by site practice.
5. `SHOW GRANTS FOR 'workbench_rw'@'<APP_HOST>';` and same for `_ro`.

---

## Step 3 — Point Workbench at new accounts

Use existing env overrides only (do **not** write secrets back into tracked YAML):

```text
WORKBENCH_DATABASE_USER=workbench_rw
WORKBENCH_DATABASE_PASSWORD=<REDACTED>
WORKBENCH_DATABASE_HOST=...
WORKBENCH_DATABASE_PORT=...
WORKBENCH_DATABASE_DBNAME=zentaopms

WORKBENCH_DATABASEREADONLY_USER=workbench_ro
WORKBENCH_DATABASEREADONLY_PASSWORD=<REDACTED>
WORKBENCH_DATABASEREADONLY_HOST=...
WORKBENCH_DATABASEREADONLY_PORT=...
WORKBENCH_DATABASEREADONLY_DBNAME=zentaopms
```

Keep `WORKBENCH_MODE=dev` → `config.dev.yaml` for non-secret host/port if desired; passwords must come from env when placeholders are in Git.

---

## Step 4 — Restart Workbench

Restart the VM/local process under the new credentials.
Expect: process starts; `ensureOperationLogSchema` passes; logs show **no** CREATE/ALTER attempts and no permission denied on boot.

---

## Step 5 — Smoke test (VM fixture data only)

### Startup

- [ ] App listens
- [ ] Schema check OK
- [ ] No DDL SQL in traces

### Login

- [ ] Successful login
- [ ] Login log row appears (`zt_login_logs`)
- [ ] Failed login records (`zt_login_failures`) with disposable account attempt

### PO read (RO pool)

- [ ] Home / todos / done / notice / follow pages load

### PO write (RW pool)

- [ ] Follow demand → unfollow
- [ ] Mark one notice read
- [ ] Mark all notices read

### Schedule read

- [ ] Schedule pages / window list / demand-story views

### Schedule write (disposable fixture only)

- [ ] Create/update a **test** version window
- [ ] Save scheduling on fixture objects
- [ ] Task maintenance on fixture story

Never use production business rows.

---

## Step 6 — Negative / least-privilege tests

Prefer disposable fixture + `START TRANSACTION` … `ROLLBACK` when safe.

### As `workbench_ro`

- [ ] `SELECT` on an allowed table → success
- [ ] `INSERT` / `UPDATE` / `DELETE` → denied
- [ ] `CREATE TABLE` → denied

### As `workbench_rw`

- [ ] Allowed DML on whitelist table → success (rollback / fixture)
- [ ] `CREATE TABLE` / `ALTER TABLE` / `DROP TABLE` → denied
- [ ] `INSERT` into a SELECT-only ZenTao table (e.g. `zt_product` if still SELECT-only) → denied

If transaction rollback is unsafe for a path, use isolated fixture rows documented here — do not mutate real business data.

---

## Step 7 — Observe

- [ ] ZenTao Web still works on its **unchanged** DB account
- [ ] Workbench logs clean for 1 agreed soak window
- [ ] No accidental writes to readonly identity

---

## Step 8 — Rollback

1. Point `WORKBENCH_*` back to the previous **still-valid** Workbench credentials (if not revoked).
2. Restart Workbench.
3. **Do not** write root / leaked passwords into Git.
4. Fix missing GRANTs on the new accounts; do **not** restore `ALL PRIVILEGES` as the default fix.
5. Leave ZenTao account untouched.

---

## Explicit non-goals

- No REVOKE of `zentao@…` in this change window
- No P1c scanner, P1d RC, P2 API work
- No password rotation of historical Git secrets unless separately authorized
