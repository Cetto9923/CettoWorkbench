-- REVIEW TEMPLATE ONLY
-- DO NOT EXECUTE AUTOMATICALLY
-- P1b-vm PRE-FLIGHT
--
-- Purpose: create dedicated Workbench runtime accounts with per-table privileges.
-- Database (VM evidence): zentaopms
-- Accounts (placeholders):
--   'workbench_rw'@'<APP_HOST>'
--   'workbench_ro'@'<APP_HOST>'
-- Password: set by DBA outside Git as <GENERATE_STRONG_PASSWORD_OUTSIDE_GIT>
--
-- Absolute bans in this template:
--   no ALL PRIVILEGES
--   no GRANT OPTION
--   no REVOKE / DROP / ALTER of ZenTao shared accounts
--   no real passwords
--
-- Replace <APP_HOST> with the MySQL host identity of the Workbench process
-- (often '127.0.0.1' or '%' depending on how the app connects).

-- =============================================================================
-- 0) CREATE USERS (DBA injects passwords; do not commit secrets)
-- =============================================================================
-- CREATE USER 'workbench_rw'@'<APP_HOST>' IDENTIFIED BY '<GENERATE_STRONG_PASSWORD_OUTSIDE_GIT>';
-- CREATE USER 'workbench_ro'@'<APP_HOST>' IDENTIFIED BY '<GENERATE_STRONG_PASSWORD_OUTSIDE_GIT>';

-- =============================================================================
-- 1) workbench_ro — SELECT only (PO readonly pool / Repo.db)
-- =============================================================================
GRANT SELECT ON `zentaopms`.`zt_action` TO 'workbench_ro'@'<APP_HOST>';
GRANT SELECT ON `zentaopms`.`zt_approvalnode` TO 'workbench_ro'@'<APP_HOST>';
GRANT SELECT ON `zentaopms`.`zt_approvalobject` TO 'workbench_ro'@'<APP_HOST>';
GRANT SELECT ON `zentaopms`.`zt_bug` TO 'workbench_ro'@'<APP_HOST>';
GRANT SELECT ON `zentaopms`.`zt_case` TO 'workbench_ro'@'<APP_HOST>';
GRANT SELECT ON `zentaopms`.`zt_charter` TO 'workbench_ro'@'<APP_HOST>';
GRANT SELECT ON `zentaopms`.`zt_demand` TO 'workbench_ro'@'<APP_HOST>';
GRANT SELECT ON `zentaopms`.`zt_demandappraise` TO 'workbench_ro'@'<APP_HOST>';
GRANT SELECT ON `zentaopms`.`zt_demandclarify` TO 'workbench_ro'@'<APP_HOST>';
GRANT SELECT ON `zentaopms`.`zt_demandmanagerreview` TO 'workbench_ro'@'<APP_HOST>';
GRANT SELECT ON `zentaopms`.`zt_demandpool` TO 'workbench_ro'@'<APP_HOST>';
GRANT SELECT ON `zentaopms`.`zt_dept` TO 'workbench_ro'@'<APP_HOST>';
GRANT SELECT ON `zentaopms`.`zt_file` TO 'workbench_ro'@'<APP_HOST>';
GRANT SELECT ON `zentaopms`.`zt_issue` TO 'workbench_ro'@'<APP_HOST>';
GRANT SELECT ON `zentaopms`.`zt_notify` TO 'workbench_ro'@'<APP_HOST>';
GRANT SELECT ON `zentaopms`.`zt_planchange` TO 'workbench_ro'@'<APP_HOST>';
GRANT SELECT ON `zentaopms`.`zt_product` TO 'workbench_ro'@'<APP_HOST>';
GRANT SELECT ON `zentaopms`.`zt_project` TO 'workbench_ro'@'<APP_HOST>';
GRANT SELECT ON `zentaopms`.`zt_projectbuildguide` TO 'workbench_ro'@'<APP_HOST>';
GRANT SELECT ON `zentaopms`.`zt_projectweekly` TO 'workbench_ro'@'<APP_HOST>';
GRANT SELECT ON `zentaopms`.`zt_review` TO 'workbench_ro'@'<APP_HOST>';
GRANT SELECT ON `zentaopms`.`zt_risk` TO 'workbench_ro'@'<APP_HOST>';
GRANT SELECT ON `zentaopms`.`zt_starinfo` TO 'workbench_ro'@'<APP_HOST>';
GRANT SELECT ON `zentaopms`.`zt_story` TO 'workbench_ro'@'<APP_HOST>';
GRANT SELECT ON `zentaopms`.`zt_task` TO 'workbench_ro'@'<APP_HOST>';
GRANT SELECT ON `zentaopms`.`zt_team` TO 'workbench_ro'@'<APP_HOST>';
GRANT SELECT ON `zentaopms`.`zt_teamgroup` TO 'workbench_ro'@'<APP_HOST>';
GRANT SELECT ON `zentaopms`.`zt_testtask` TO 'workbench_ro'@'<APP_HOST>';
GRANT SELECT ON `zentaopms`.`zt_todo` TO 'workbench_ro'@'<APP_HOST>';
GRANT SELECT ON `zentaopms`.`zt_user` TO 'workbench_ro'@'<APP_HOST>';
GRANT SELECT ON `zentaopms`.`zt_workbench_notify_reads` TO 'workbench_ro'@'<APP_HOST>';

-- =============================================================================
-- 2) workbench_rw — SELECT on all tables the main pool reads
-- =============================================================================
-- Workbench-owned / admin
GRANT SELECT ON `zentaopms`.`zt_operation_logs` TO 'workbench_rw'@'<APP_HOST>';
GRANT SELECT ON `zentaopms`.`zt_login_logs` TO 'workbench_rw'@'<APP_HOST>';
GRANT SELECT ON `zentaopms`.`zt_roles` TO 'workbench_rw'@'<APP_HOST>';
GRANT SELECT ON `zentaopms`.`zt_role_permissions` TO 'workbench_rw'@'<APP_HOST>';
GRANT SELECT ON `zentaopms`.`zt_gf_user_roles` TO 'workbench_rw'@'<APP_HOST>';
GRANT SELECT ON `zentaopms`.`zt_menus` TO 'workbench_rw'@'<APP_HOST>';
GRANT SELECT ON `zentaopms`.`zt_depts` TO 'workbench_rw'@'<APP_HOST>';
GRANT SELECT ON `zentaopms`.`zt_versionwindow` TO 'workbench_rw'@'<APP_HOST>';
GRANT SELECT ON `zentaopms`.`zt_versionwindowproduct` TO 'workbench_rw'@'<APP_HOST>';
GRANT SELECT ON `zentaopms`.`zt_demandwindow` TO 'workbench_rw'@'<APP_HOST>';
GRANT SELECT ON `zentaopms`.`zt_workbench_notify_reads` TO 'workbench_rw'@'<APP_HOST>';

-- ZenTao / schedule / auth reads on main pool
GRANT SELECT ON `zentaopms`.`zt_user` TO 'workbench_rw'@'<APP_HOST>';
GRANT SELECT ON `zentaopms`.`zt_starinfo` TO 'workbench_rw'@'<APP_HOST>';
GRANT SELECT ON `zentaopms`.`zt_story` TO 'workbench_rw'@'<APP_HOST>';
GRANT SELECT ON `zentaopms`.`zt_task` TO 'workbench_rw'@'<APP_HOST>';
GRANT SELECT ON `zentaopms`.`zt_planstory` TO 'workbench_rw'@'<APP_HOST>';
GRANT SELECT ON `zentaopms`.`zt_projectstory` TO 'workbench_rw'@'<APP_HOST>';
GRANT SELECT ON `zentaopms`.`zt_productplan` TO 'workbench_rw'@'<APP_HOST>';
GRANT SELECT ON `zentaopms`.`zt_demand` TO 'workbench_rw'@'<APP_HOST>';
GRANT SELECT ON `zentaopms`.`zt_product` TO 'workbench_rw'@'<APP_HOST>';
GRANT SELECT ON `zentaopms`.`zt_project` TO 'workbench_rw'@'<APP_HOST>';
GRANT SELECT ON `zentaopms`.`zt_projectproduct` TO 'workbench_rw'@'<APP_HOST>';
GRANT SELECT ON `zentaopms`.`zt_holiday` TO 'workbench_rw'@'<APP_HOST>';
GRANT SELECT ON `zentaopms`.`zt_config` TO 'workbench_rw'@'<APP_HOST>';
GRANT SELECT ON `zentaopms`.`zt_company` TO 'workbench_rw'@'<APP_HOST>';
GRANT SELECT ON `zentaopms`.`zt_teamgroup` TO 'workbench_rw'@'<APP_HOST>';
GRANT SELECT ON `zentaopms`.`zt_team` TO 'workbench_rw'@'<APP_HOST>';
GRANT SELECT ON `zentaopms`.`zt_demandpool` TO 'workbench_rw'@'<APP_HOST>';
GRANT SELECT ON `zentaopms`.`zt_dept` TO 'workbench_rw'@'<APP_HOST>';
GRANT SELECT ON `zentaopms`.`zt_demandclarify` TO 'workbench_rw'@'<APP_HOST>';
GRANT SELECT ON `zentaopms`.`zt_demanduserstory` TO 'workbench_rw'@'<APP_HOST>';
GRANT SELECT ON `zentaopms`.`zt_file` TO 'workbench_rw'@'<APP_HOST>';
GRANT SELECT ON `zentaopms`.`zt_notify` TO 'workbench_rw'@'<APP_HOST>';

-- =============================================================================
-- 3) workbench_rw — DML (minimal; soft-delete paths use UPDATE, not DELETE)
-- =============================================================================
-- WB: operationlog middleware INSERT only
GRANT INSERT ON `zentaopms`.`zt_operation_logs` TO 'workbench_rw'@'<APP_HOST>';

-- WB: login failures INSERT + DELETE (CleanOldFailures exists in source; currently no live caller)
GRANT INSERT, DELETE ON `zentaopms`.`zt_login_failures` TO 'workbench_rw'@'<APP_HOST>';

-- WB: login logs INSERT
GRANT INSERT ON `zentaopms`.`zt_login_logs` TO 'workbench_rw'@'<APP_HOST>';

-- WB: roles soft-delete via UPDATE deleted=
GRANT INSERT, UPDATE ON `zentaopms`.`zt_roles` TO 'workbench_rw'@'<APP_HOST>';

-- WB: role_permissions replace = GORM soft Delete (UPDATE deletedAt) + INSERT
GRANT INSERT, UPDATE ON `zentaopms`.`zt_role_permissions` TO 'workbench_rw'@'<APP_HOST>';

-- WB: gf_user_roles replace = GORM soft Delete + INSERT
GRANT INSERT, UPDATE ON `zentaopms`.`zt_gf_user_roles` TO 'workbench_rw'@'<APP_HOST>';

-- WB: menus soft-delete via UPDATE deletedAt
GRANT INSERT, UPDATE ON `zentaopms`.`zt_menus` TO 'workbench_rw'@'<APP_HOST>';

-- UNVERIFIED ownership (Workbench-style); soft-delete via UPDATE deletedAt
GRANT INSERT, UPDATE ON `zentaopms`.`zt_depts` TO 'workbench_rw'@'<APP_HOST>';

-- WB: version windows soft-delete UPDATE; products hard DELETE on rebuild
GRANT INSERT, UPDATE ON `zentaopms`.`zt_versionwindow` TO 'workbench_rw'@'<APP_HOST>';
GRANT INSERT, UPDATE, DELETE ON `zentaopms`.`zt_versionwindowproduct` TO 'workbench_rw'@'<APP_HOST>';

-- WB: demandwindow hard DELETE + INSERT
GRANT INSERT, DELETE ON `zentaopms`.`zt_demandwindow` TO 'workbench_rw'@'<APP_HOST>';

-- WB-style notify reads: INSERT...SELECT (needs SELECT already granted above)
GRANT INSERT ON `zentaopms`.`zt_workbench_notify_reads` TO 'workbench_rw'@'<APP_HOST>';

-- Temporary ZenTao / MIX DML (P2 later reclaim)
GRANT INSERT, UPDATE ON `zentaopms`.`zt_user` TO 'workbench_rw'@'<APP_HOST>';
GRANT INSERT, UPDATE ON `zentaopms`.`zt_starinfo` TO 'workbench_rw'@'<APP_HOST>';
GRANT INSERT, UPDATE ON `zentaopms`.`zt_story` TO 'workbench_rw'@'<APP_HOST>';
GRANT INSERT ON `zentaopms`.`zt_storyspec` TO 'workbench_rw'@'<APP_HOST>';
GRANT INSERT, UPDATE ON `zentaopms`.`zt_task` TO 'workbench_rw'@'<APP_HOST>';
GRANT INSERT ON `zentaopms`.`zt_taskspec` TO 'workbench_rw'@'<APP_HOST>';
GRANT INSERT ON `zentaopms`.`zt_action` TO 'workbench_rw'@'<APP_HOST>';
GRANT INSERT, DELETE ON `zentaopms`.`zt_planstory` TO 'workbench_rw'@'<APP_HOST>';
GRANT INSERT, UPDATE ON `zentaopms`.`zt_projectstory` TO 'workbench_rw'@'<APP_HOST>';
GRANT INSERT ON `zentaopms`.`zt_productplan` TO 'workbench_rw'@'<APP_HOST>';
GRANT UPDATE ON `zentaopms`.`zt_demand` TO 'workbench_rw'@'<APP_HOST>';

-- =============================================================================
-- 4) Apply
-- =============================================================================
-- FLUSH PRIVILEGES;

-- DO NOT:
--   REVOKE ... FROM 'zentao'@'...';
--   DROP USER 'zentao'@'...';
--   ALTER USER 'zentao'@'...';
--   GRANT CREATE/ALTER/DROP/TRUNCATE/GRANT OPTION/...
