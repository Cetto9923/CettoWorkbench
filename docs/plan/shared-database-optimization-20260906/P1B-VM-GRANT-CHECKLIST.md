# P1b-vm GRANT Checklist

Generated to keep **code/docs/SQL** counts identical. Source of truth for privilege cells: `grants-workbench-runtime.sql` (HEAD after closeout).

Do **not** execute GRANT from this file.

## Privilege matrix

| table | RO SELECT | RW SELECT | RW INSERT | RW UPDATE | RW DELETE |
|---|---:|---:|---:|---:|---:|
| `zt_action` | Y |  | Y |  |  |
| `zt_approvalnode` | Y |  |  |  |  |
| `zt_approvalobject` | Y |  |  |  |  |
| `zt_bug` | Y |  |  |  |  |
| `zt_case` | Y |  |  |  |  |
| `zt_charter` | Y |  |  |  |  |
| `zt_company` |  | Y |  |  |  |
| `zt_config` |  | Y |  |  |  |
| `zt_demand` | Y | Y |  | Y |  |
| `zt_demandappraise` | Y |  |  |  |  |
| `zt_demandclarify` | Y | Y |  |  |  |
| `zt_demandmanagerreview` | Y |  |  |  |  |
| `zt_demandpool` | Y | Y |  |  |  |
| `zt_demanduserstory` |  | Y |  |  |  |
| `zt_demandwindow` |  | Y | Y |  | Y |
| `zt_dept` | Y | Y |  |  |  |
| `zt_depts` |  | Y | Y | Y |  |
| `zt_file` | Y | Y |  |  |  |
| `zt_gf_user_roles` |  | Y | Y | Y |  |
| `zt_holiday` |  | Y |  |  |  |
| `zt_issue` | Y |  |  |  |  |
| `zt_login_failures` |  |  | Y |  |  |
| `zt_login_logs` |  | Y | Y |  |  |
| `zt_menus` |  | Y | Y | Y |  |
| `zt_notify` | Y | Y |  |  |  |
| `zt_operation_logs` |  | Y | Y |  |  |
| `zt_planchange` | Y |  |  |  |  |
| `zt_planstory` |  | Y | Y |  | Y |
| `zt_product` | Y | Y |  |  |  |
| `zt_productplan` |  | Y | Y |  |  |
| `zt_project` | Y | Y |  |  |  |
| `zt_projectbuildguide` | Y |  |  |  |  |
| `zt_projectproduct` |  | Y |  |  |  |
| `zt_projectstory` |  | Y | Y | Y |  |
| `zt_projectweekly` | Y |  |  |  |  |
| `zt_review` | Y |  |  |  |  |
| `zt_risk` | Y |  |  |  |  |
| `zt_role_permissions` |  | Y | Y | Y |  |
| `zt_roles` |  | Y | Y | Y |  |
| `zt_starinfo` | Y | Y | Y | Y |  |
| `zt_story` | Y | Y | Y | Y |  |
| `zt_storyspec` |  |  | Y |  |  |
| `zt_task` | Y | Y | Y | Y |  |
| `zt_taskspec` |  |  | Y |  |  |
| `zt_team` | Y | Y |  |  |  |
| `zt_teamgroup` | Y | Y |  |  |  |
| `zt_testtask` | Y |  |  |  |  |
| `zt_todo` | Y |  |  |  |  |
| `zt_user` | Y | Y | Y | Y |  |
| `zt_versionwindow` |  | Y | Y | Y |  |
| `zt_versionwindowproduct` |  | Y | Y | Y | Y |
| `zt_workbench_notify_reads` | Y | Y | Y |  |  |

## Counts (must match SQL)

```text
RO SELECT: 31
RW SELECT: 33
RW INSERT: 22
RW UPDATE: 13
RW DELETE: 3
```

## APP_HOST_RESOLUTION

```text
current evidence:
  - Mac ./tmp/workbench_dev ESTABLISHED to 127.0.0.1:13306 (SSH tunnel to VM MySQL)
  - Probe CURRENT_USER()=zentao@127.0.0.1 against zentaopms
  - Tunnel listener: 127.0.0.1:13306
expected execution connection path (this Review slice):
  Mac Workbench → SSH tunnel → VM MySQL (zentaopms)
resolved host: 127.0.0.1
confidence: HIGH for that path
banned: default "%"
alternate topologies (out of this grant template scope unless Review expands):
  - /opt/workbench/current on VM uses database.dbname=workbench (different schema)
  - live ZenTao /var/www/zentaopms/config/my.php → host 10.211.55.2 user zentao db zentaopms
```

## Schema prerequisites (current VM zentaopms)

| table | exists | note |
|---|---|---|
| `zt_operation_logs` | YES | required columns present; in install.sql (P1a) |
| `zt_depts` | YES | not in install.sql → clean-install debt |
| `zt_workbench_notify_reads` | YES | not in install.sql → clean-install debt |

```text
current VM execution: not blocked
clean-install provisioning debt: open
```

## zt_login_failures DELETE decision

```text
CleanOldFailures live caller: NONE (definition only in login/repo.go)
DELETE grant: NO
INSERT grant: YES (RecordFailure)
```
