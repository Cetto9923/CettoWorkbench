# P1b-vm DBA Bundle — Run on the VM, Return SHOW GRANTS

This bundle is meant to be **executed by you on the VM** (or by a DBA in your seat). Agent does **not** run `sudo mysql`. The bundle does **not** print, log, or transmit passwords in clear text. Agent will read only:

- The **SHOW GRANTS** outputs (sanitized)
- A copy of the GRANT SQL it applied (no real password inside)

---

## 0. Pre-flight (read-only)

```bash
# On VM terminal
sudo -v
sudo mysql -uroot -e "SELECT VERSION(), CURRENT_USER(), @@hostname, @@transaction_isolation;" zentaopms
```

Expected: `VERSION() = 8.0.46-0ubuntu0.24.04.4` (or similar 8.0.x), `CURRENT_USER() = root@localhost`, host matches `ubuntu-gnu-linux-24-04-3`.

If any line fails, **stop** and paste the error.

## 1. Backup current grants (no secrets)

```bash
sudo mysql -uroot -B -e "SHOW GRANTS" > /tmp/grants_before_P1b.txt 2>/dev/null
for u in "'zentao'@'127.0.0.1'" "'zentao'@'localhost'" \
         "'workbench_rw'@'127.0.0.1'" "'workbench_ro'@'127.0.0.1'"; do
  echo "=== $u ===" | sudo tee -a /tmp/grants_before_P1b.txt >/dev/null
  sudo mysql -uroot -B -e "SHOW GRANTS FOR $u" \
    2>/dev/null | sudo tee -a /tmp/grants_before_P1b.txt >/dev/null
done
sudo chown $USER /tmp/grants_before_P1b.txt
ls -l /tmp/grants_before_P1b.txt
```

## 2. Generate two strong passwords (local files, 600)

```bash
umask 077
openssl rand -base64 24 | tr -d '/+=' | head -c 32 > ~/.wb_rw_pw
openssl rand -base64 24 | tr -d '/+=' | head -c 32 > ~/.wb_ro_pw
chmod 600 ~/.wb_rw_pw ~/.wb_ro_pw
ls -l ~/.wb_rw_pw ~/.wb_ro_pw
```

## 3. Download reviewed GRANT template (no secrets inside)

```bash
mkdir -p ~/p1bvm && cd ~/p1bvm
curl -fsSL https://raw.githubusercontent.com/Cetto9923/workbench-claude-po/Claude-PO/docs/plan/shared-database-optimization-20260906/grants-workbench-runtime.sql -o grants.sql
sed -i 's|<APP_HOST>|127.0.0.1|g' grants.sql
head -3 grants.sql && echo "..." && wc -l grants.sql
```

The 13 commented-out lines (CREATE USER placeholders + FLUSH + DO-NOT list) should remain commented. The remaining GRANT lines start with `GRANT SELECT`/`GRANT INSERT/UPDATE/DELETE` and target `127.0.0.1`.

## 4. Create the two users (passwords from 600 files, no echo)

```bash
set +o history
RW_PW=$(cat ~/.wb_rw_pw)
RO_PW=$(cat ~/.wb_ro_pw)
sudo mysql -uroot <<SQL
CREATE USER 'workbench_rw'@'127.0.0.1' IDENTIFIED BY '${RW_PW}';
CREATE USER 'workbench_ro'@'127.0.0.1' IDENTIFIED BY '${RO_PW}';
SQL
unset RW_PW RO_PW
set -o history
```

If both users already exist from a previous attempt, **stop**. Do not ALTER USER or DROP. Paste current `SHOW GRANTS` for both to Agent.

## 5. Apply reviewed GRANT SQL

```bash
sudo mysql -uroot < ~/p1bvm/grants.sql
```

## 6. Verify (paste these outputs to Agent)

```bash
sudo mysql -uroot -B -e "SHOW GRANTS FOR 'workbench_ro'@'127.0.0.1';"
sudo mysql -uroot -B -e "SHOW GRANTS FOR 'workbench_rw'@'127.0.0.1';"
sudo mysql -uroot -B -e "SHOW GRANTS FOR 'zentao'@'127.0.0.1';"   # baseline
sudo mysql -uroot -B -e "SHOW GRANTS FOR 'zentao'@'localhost';" # baseline
```

Expected counts (must match exactly):

```text
workbench_ro: 31 GRANT SELECT
workbench_rw: 33 SELECT + 22 INSERT + 13 UPDATE + 3 DELETE
zentao:      UNCHANGED (no REVOKE/ALTER/DROP/rotation)
```

## 7. Bring the two password files to the Mac (Agent does NOT see content)

```bash
# On Mac terminal
scp parallels@vm-zentao:.wb_rw_pw .wb_rw_pw
scp parallels@vm-zentao:.wb_ro_pw .wb_ro_pw
chmod 600 .wb_rw_pw .wb_ro_pw
ls -l .wb_rw_pw .wb_ro_pw
```

Agent will reference these files via `WORKBENCH_*` env at Canary time. Passwords are never pasted into chat, Markdown, Git, or shell history.

## 8. Cleanup (after Canary switch & smoke pass)

```bash
# On VM
shred -u ~/.wb_rw_pw ~/.wb_ro_pw
rm -rf ~/p1bvm

# On Mac, AFTER smoke success:
shred -u .wb_rw_pw .wb_ro_pw
```

## What Agent will do next

- Validate 31 / 33 / 22 / 13 / 3 from your SHOW GRANTS output
- On Mac: build `./tmp/workbench_p1b` on a free port using `WORKBENCH_*` env that reads the two password files (env-only, no tracked YAML change)
- Run startup / login / PO read / PO follow+notice / schedule fixture write smoke
- Run negative tests via tunnel as `workbench_ro`/`workbench_rw`
- Update `P1B-VM-EXECUTION-RESULT.md` and PLANS, commit, push
- If any smoke step fails or any required GRANT is missing, Agent **stops** and rolls back Canary without touching the old `zentao` account

## Hard rules (still in force)

- No REVOKE / ALTER / DROP / password rotation on `zentao`
- No P1c / P1d / P2 in this round
- No tracked YAML changes
- Agent does not read or log the content of the two password files
