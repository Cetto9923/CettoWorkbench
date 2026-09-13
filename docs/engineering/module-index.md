# Module Index

Freshness note (2026-09-07): schema labels below are the original source-only
inventory. The repository now contains
[V0 evidence](../plan/workbench-executable-plan-20260905/evidence/schema-truth-20260905.md)
for a dated instance; that is not proof for every later Mac/VM/production target.
Use table-specific evidence and verify environment identity after migrations.
Do not reopen V0 solely from an unchanged inventory label.

This index provides a structural map of the modules under `internal/module/`.

> [!NOTE]
> Listing here documents current source reality; it does **not** certify any legacy module as a golden architectural reference. Listed table names come from models/repository queries, not live DDL; table ownership is marked `SCHEMA UNVERIFIED` until production schema DDL is confirmed via Phase V0.

| Module | Responsibility | Public Services | Module-wide Dependencies (not Service fields) | Selected Table References (not exhaustive ownership) | Status | Source Location |
|---|---|---|---|---|---|---|
| `dept` | Department tree structure and CRUD | `dept.Service` | `dept.Repo`, `gorm.DB` | `zt_depts` | SOURCE VERIFIED / SCHEMA UNVERIFIED | `internal/module/dept/` |
| `login` | Authentication, session creation, logout | `login.Service` | `login.Repo`, `gorm.DB`, `scs.SessionManager` | `zt_user` | SOURCE VERIFIED / SCHEMA UNVERIFIED | `internal/module/login/` |
| `loginlog` | User login audit logs | `loginlog.Service` | `loginlog.Repo`, `gorm.DB` | `zt_login_logs` | SOURCE VERIFIED / SCHEMA UNVERIFIED | `internal/module/loginlog/` |
| `operationlog` | User operation audit logging | `operationlog.Service` | `operationlog.Repo`, `gorm.DB` | `zt_operation_logs` | SOURCE VERIFIED / SCHEMA UNVERIFIED | `internal/module/operationlog/` |
| `menu` | Navigation menu tree construction | `menu.Service` | `menu.Repo`, `gorm.DB` | `zt_menus` | SOURCE VERIFIED / SCHEMA UNVERIFIED | `internal/module/menu/` |
| `role` | Role definitions and capability assignments | `role.Service` | `role.Repo`, `gorm.DB` | `zt_roles`, `zt_role_permissions` | SOURCE VERIFIED / SCHEMA UNVERIFIED | `internal/module/role/` |
| `user` | User management (legacy non-authoritative) | `user.Service` | `user.Repo`, `gorm.DB` | `zt_user`, `zt_gf_user_roles` | SOURCE VERIFIED / SCHEMA UNVERIFIED (Legacy) | `internal/module/user/` |
| `schedule` | Window scheduling, demand allocation | `schedule.Service` | `schedule.Repo`, `gorm.DB`, `zentao` | `zt_demand`, `zt_task` | SOURCE VERIFIED / SCHEMA UNVERIFIED | `internal/module/schedule/` |
| `po` | PO value stream, board metrics, todos, notices | `po.Service` | `po.Repo`, `schedule.Service`, `user.Service` | `zt_demand`, `zt_task` | SOURCE VERIFIED / SCHEMA UNVERIFIED | `internal/module/po/` |
| `agileteam` | Agile team topology, membership adjustments, confirm/reject | `agileteam.Service` | `agileteam.Repo`, `gorm.DB` | `zt_teamgroup`, `zt_team`, `zt_wb_agileteam_adjustment`, `zt_wb_agileteam_adjustment_item`, `zt_wb_agileteam_history`, `zt_user` | SOURCE VERIFIED / SCHEMA UNVERIFIED | `internal/module/agileteam/` |
| `metrics` | Metrics definitions, read-only ZenTao snapshots, radar helpers | `metrics.Service` | `metrics.Repo`, `gorm.DB` | `zt_story`, `zt_bug`, `zt_task` | SOURCE VERIFIED / SCHEMA UNVERIFIED | `internal/module/metrics/` |
| `profile` | Self-service profile, password, preferred roles, main team | `profile.Service` | `profile.Repo`, `gorm.DB` | `zt_user`, `zt_dept`, `zt_teamgroup`, `zt_team`, `zt_wb_profile_prefs`, `zt_gf_user_roles`, `zt_roles` | SOURCE VERIFIED / SCHEMA UNVERIFIED | `internal/module/profile/` |
| `query` | Read-only business/RD demand list views | `query.Service` | `query.Repo`, `gorm.DB` | `zt_demand`, `zt_story`, `zt_user`, `zt_product` | SOURCE VERIFIED / SCHEMA UNVERIFIED | `internal/module/query/` |
| `debug` | SQL performance telemetry (superadmin only) | `debug.Service` | `debug.Repo`, filesystem logs | None (file logs only) | SOURCE VERIFIED | `internal/module/debug/` |

- **Shared Frontend**: Source-inspected shared client-side capabilities are documented in [shared-frontend.md](shared-frontend.md).
