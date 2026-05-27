-- goframework schema install script
-- Usage:
--   mysql -u <user> -p <database> < db/install.sql
--
-- 约定：zt_user / zt_login_failures / zt_login_logs / zt_operation_logs 的登录标识列
-- 一律为 account（小驼峰）。勿使用 username 列名，以免与 ORM、手写 Where 不一致。
-- 若已有库仍为列名 username，备份后可按需执行（示例，长度以实际为准）：
--   ALTER TABLE zt_user CHANGE COLUMN username account VARCHAR(64) NOT NULL;

CREATE TABLE IF NOT EXISTS zt_user (
  id BIGINT NOT NULL AUTO_INCREMENT,
  account VARCHAR(30) NOT NULL,
  email VARCHAR(60) NOT NULL,
  password VARCHAR(60) NOT NULL,
  realname VARCHAR(30) NOT NULL,
  gender ENUM('f', 'm') NOT NULL DEFAULT 'm',
  position VARCHAR(30) NOT NULL,
  manager BIGINT NOT NULL DEFAULT 0,
  phone VARCHAR(20) NOT NULL,
  createdBy VARCHAR(30) NOT NULL,
  createdDate DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updatedBy VARCHAR(30) NOT NULL,
  updatedDate DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  deleted TINYINT(1) NOT NULL DEFAULT 0,
  isSuperAdmin TINYINT(1) NOT NULL DEFAULT 0,
  isActive TINYINT(1) NOT NULL DEFAULT 1,
  deptID BIGINT NOT NULL DEFAULT 0,
  lastLoginDate DATETIME NULL,
  lastLoginIP VARCHAR(45) NULL,
  tenantId BIGINT NOT NULL DEFAULT 0,
  PRIMARY KEY (id),
  UNIQUE KEY uk_zt_user_account (account),
  UNIQUE KEY uk_zt_user_email (email),
  KEY idx_zt_user_deleted (deleted),
  KEY idx_zt_user_tenant (tenantId)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

INSERT IGNORE INTO zt_user (account,email,password,realname,gender,position,manager,phone,createdBy,updatedBy,isSuperAdmin,isActive,tenantId) VALUES ('admin','admin@example.com','$2a$12$n.AfkpRXoMScpH1a6CQZM.TgsYEzrXqJWm.f7nyCGWjbbnl8mfT3G','超级管理员','m','Super Admin',0,'','system','system',0,1,0);

CREATE TABLE IF NOT EXISTS zt_login_failures (
  id BIGINT NOT NULL AUTO_INCREMENT,
  account VARCHAR(64) NOT NULL,
  ip VARCHAR(45) NOT NULL,
  failedAt DATETIME NOT NULL,
  createdDate DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (id),
  KEY idx_zt_loginfailures_account_time (account, failedAt),
  KEY idx_zt_loginfailures_ip_time (ip, failedAt)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS zt_login_logs (
  id BIGINT NOT NULL AUTO_INCREMENT,
  account VARCHAR(64) NOT NULL,
  userId BIGINT NULL,
  ip VARCHAR(45) NOT NULL,
  userAgent VARCHAR(512) NOT NULL,
  success TINYINT(1) NOT NULL DEFAULT 0,
  failReason VARCHAR(64) NOT NULL DEFAULT '',
  createdDate DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  tenantId BIGINT NOT NULL DEFAULT 0,
  PRIMARY KEY (id),
  KEY idx_zt_loginlogs_account_time (account, createdDate),
  KEY idx_zt_loginlogs_ip_time (ip, createdDate)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS zt_roles (
  id BIGINT NOT NULL AUTO_INCREMENT,
  code VARCHAR(64) NOT NULL,
  name VARCHAR(64) NOT NULL,
  description VARCHAR(255) NOT NULL DEFAULT '',
  isBuiltin TINYINT(1) NOT NULL DEFAULT 0,
  isActive TINYINT(1) NOT NULL DEFAULT 1,
  sortOrder INT NOT NULL DEFAULT 0,
  tenantId BIGINT NOT NULL DEFAULT 0,
  createdBy BIGINT NOT NULL DEFAULT 0,
  updatedBy BIGINT NOT NULL DEFAULT 0,
  createdDate DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updatedDate DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  deleted TINYINT(1) NOT NULL DEFAULT 0,
  PRIMARY KEY (id),
  UNIQUE KEY uk_zt_roles_code_deleted (tenantId, code, deleted),
  UNIQUE KEY uk_zt_roles_name_deleted (tenantId, name, deleted),
  KEY idx_zt_roles_tenant (tenantId),
  KEY idx_zt_roles_active (tenantId, isActive),
  KEY idx_zt_roles_sort (tenantId, sortOrder)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS zt_user_roles (
  id BIGINT NOT NULL AUTO_INCREMENT,
  userId BIGINT NOT NULL,
  roleId BIGINT NOT NULL,
  tenantId BIGINT NOT NULL DEFAULT 0,
  createdBy BIGINT NOT NULL DEFAULT 0,
  updatedBy BIGINT NOT NULL DEFAULT 0,
  createdDate DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updatedDate DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  deleted TINYINT(1) NOT NULL DEFAULT 0,
  PRIMARY KEY (id),
  UNIQUE KEY uk_zt_userroles_user_role_deleted (tenantId, userId, roleId, deleted),
  KEY idx_zt_userroles_user (tenantId, userId),
  KEY idx_zt_userroles_role (tenantId, roleId)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS zt_role_permissions (
  id BIGINT NOT NULL AUTO_INCREMENT,
  roleId BIGINT NOT NULL,
  permCode VARCHAR(64) NOT NULL DEFAULT '',
  tenantId BIGINT NOT NULL DEFAULT 0,
  createdBy BIGINT NOT NULL DEFAULT 0,
  updatedBy BIGINT NOT NULL DEFAULT 0,
  createdDate DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updatedDate DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  deleted TINYINT(1) NOT NULL DEFAULT 0,
  PRIMARY KEY (id),
  UNIQUE KEY uk_role_perm_deleted (tenantId, roleId, permCode, deleted),
  KEY idx_zt_rolepermissions_role (tenantId, roleId),
  KEY idx_perm (tenantId, permCode)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS `zt_menus` (
  `id`        BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `parentId`  BIGINT UNSIGNED NOT NULL DEFAULT 0,
  `title`     VARCHAR(64) NOT NULL DEFAULT '',
  `icon`      VARCHAR(64) NOT NULL DEFAULT '',
  `path`      VARCHAR(255) NOT NULL DEFAULT '',
  `perm`      VARCHAR(64) NOT NULL DEFAULT '',
  `type`      CHAR(1) NOT NULL DEFAULT 'C',
  `sort`      INT NOT NULL DEFAULT 0,
  `createdAt` DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `updatedAt` DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  `deletedAt` DATETIME(3) DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_zt_menus_parent_title_perm` (`parentId`, `title`, `perm`),
  KEY `idx_parentId` (`parentId`),
  KEY `idx_zt_menus_perm` (`perm`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS `zt_depts` (
  `id`        BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `tenantId`  BIGINT NOT NULL DEFAULT 0,
  `parentId`  BIGINT UNSIGNED NOT NULL DEFAULT 0,
  `name`      VARCHAR(64) NOT NULL DEFAULT '',
  `leader`    VARCHAR(64) NOT NULL DEFAULT '' COMMENT '负责人',
  `phone`     VARCHAR(32) NOT NULL DEFAULT '' COMMENT '联系电话',
  `email`     VARCHAR(64) NOT NULL DEFAULT '' COMMENT '邮箱',
  `status`    TINYINT(1) NOT NULL DEFAULT 0 COMMENT '部门状态（0正常 1停用）',
  `sort`      INT NOT NULL DEFAULT 0,
  `createdAt` DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `updatedAt` DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  `deletedAt` DATETIME(3) DEFAULT NULL,
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;


CREATE TABLE IF NOT EXISTS `zt_dict_types` (
  `id`        BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `tenantId`  BIGINT NOT NULL DEFAULT 0,
  `code`      VARCHAR(64) NOT NULL DEFAULT '',
  `name`      VARCHAR(64) NOT NULL DEFAULT '',
  `remark`    VARCHAR(255) NOT NULL DEFAULT '',
  `createdAt` DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `updatedAt` DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  `deletedAt` DATETIME(3) DEFAULT NULL,
  UNIQUE KEY `uk_code_deleted` (`code`, `deletedAt`),
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS `zt_dict_items` (
  `id`         BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `tenantId`   BIGINT NOT NULL DEFAULT 0,
  `typeCode`   VARCHAR(64) NOT NULL DEFAULT '',
  `label`      VARCHAR(64) NOT NULL DEFAULT '',
  `value`      VARCHAR(64) NOT NULL DEFAULT '',
  `sort`       INT NOT NULL DEFAULT 0,
  `createdAt`  DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `updatedAt`  DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  `deletedAt`  DATETIME(3) DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `idx_typeCode` (`typeCode`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- -----------------------------------------------------------------------------
-- RBAC 种子数据（可重复执行：依赖唯一约束 + INSERT IGNORE）
-- 默认 admin 为非超管，通过「管理员」角色获得全量权限，便于联调 RequirePerm。
-- -----------------------------------------------------------------------------

INSERT IGNORE INTO zt_roles (code, name, description, isBuiltin, isActive, sortOrder, tenantId, createdBy, updatedBy, deleted)
VALUES
  ('site_admin', '管理员', '第一期功能全量权限', 1, 1, 10, 0, 0, 0, 0),
  ('read_only', '只读', '仅列表类权限', 1, 1, 20, 0, 0, 0, 0);

INSERT IGNORE INTO zt_role_permissions (roleId, permCode, tenantId, createdBy, updatedBy, deleted)
SELECT r.id, pc.permCode, 0, 0, 0, 0
FROM zt_roles r
JOIN (
  SELECT 'auth:logout' AS permCode UNION ALL
  SELECT 'user:list' UNION ALL
  SELECT 'user:create' UNION ALL
  SELECT 'user:update' UNION ALL
  SELECT 'user:delete' UNION ALL
  SELECT 'user:assignrole' UNION ALL
  SELECT 'user:resetpassword' UNION ALL
  SELECT 'operationlog:list' UNION ALL
  SELECT 'loginlog:list' UNION ALL
  SELECT 'role:list' UNION ALL
  SELECT 'role:create' UNION ALL
  SELECT 'role:edit' UNION ALL
  SELECT 'role:delete' UNION ALL
  SELECT 'menu:list' UNION ALL
  SELECT 'menu:create' UNION ALL
  SELECT 'menu:edit' UNION ALL
  SELECT 'menu:delete' UNION ALL
  SELECT 'dept:list' UNION ALL
  SELECT 'dept:create' UNION ALL
  SELECT 'dept:edit' UNION ALL
  SELECT 'dept:delete' UNION ALL
  SELECT 'dict:list' UNION ALL
  SELECT 'dict:create' UNION ALL
  SELECT 'dict:edit' UNION ALL
  SELECT 'dict:delete' UNION ALL
  SELECT 'cronjob:list' UNION ALL
  SELECT 'cronjob:edit' UNION ALL
  SELECT 'cronjob:trigger' UNION ALL
  SELECT 'backup:list' UNION ALL
  SELECT 'backup:create' UNION ALL
  SELECT 'backup:download'
) pc ON 1 = 1
WHERE r.code = 'site_admin' AND r.deleted = 0;

INSERT IGNORE INTO zt_role_permissions (roleId, permCode, tenantId, createdBy, updatedBy, deleted)
SELECT r.id, pc.permCode, 0, 0, 0, 0
FROM zt_roles r
JOIN (
  SELECT 'user:list' AS permCode UNION ALL
  SELECT 'operationlog:list' UNION ALL
  SELECT 'loginlog:list'
) pc ON 1 = 1
WHERE r.code = 'read_only' AND r.deleted = 0;

INSERT IGNORE INTO zt_user_roles (userId, roleId, tenantId, createdBy, updatedBy, deleted)
SELECT u.id, r.id, 0, 0, 0, 0
FROM zt_user u
JOIN zt_roles r ON r.code = 'site_admin' AND r.deleted = 0
WHERE u.account = 'admin';

-- TODO 最后刷新菜单
INSERT IGNORE INTO zt_menus (id, parentId, title, icon, path, perm, type, sort) VALUES
  (1, 0, '仪表盘', 'bi-speedometer2', '/admin/dashboard', '', 'C', 1),
  (2, 0, '系统管理', 'bi-gear', '', '', 'M', 100),
  (3, 2, '用户', 'bi-people', '/admin/users', 'user:list', 'C', 1),
  (13, 3, '分配角色', '', '', 'user:assignrole', 'F', 106),
  (14, 3, '重置密码', '', '', 'user:resetpassword', 'F', 107),
  (4, 2, '角色管理', 'bi-person-badge', '/admin/roles', 'role:list', 'C', 2),
  (5, 2, '菜单管理', 'bi-diagram-3', '/admin/menus', 'menu:list', 'C', 3),
  (6, 2, '部门管理', 'bi-diagram-2', '/admin/depts', 'dept:list', 'C', 4),
  (7, 2, '字典管理', 'bi-card-list', '/admin/dict-types', 'dict:list', 'C', 5),
  (15, 7, '字典新增', '', '', 'dict:create', 'F', 502),
  (16, 7, '字典编辑', '', '', 'dict:edit', 'F', 503),
  (17, 7, '字典删除', '', '', 'dict:delete', 'F', 504),
  (9, 2, '通信日志', 'bi-journal-text', '/admin/operation-logs', 'operationlog:list', 'C', 7),
  (10, 2, '登录日志', 'bi-box-arrow-in-right', '/admin/login-logs', 'loginlog:list', 'C', 8),
  (11, 2, '定时任务', 'bi-clock-history', '/admin/cron-jobs', 'cronjob:list', 'C', 9),
  (18, 11, '任务编辑', '', '', 'cronjob:edit', 'F', 902),
  (19, 11, '任务触发', '', '', 'cronjob:trigger', 'F', 903),
  (12, 2, '数据库备份', 'bi-hdd-stack', '/admin/backups', 'backup:list', 'C', 10),
  (20, 12, '备份创建', '', '', 'backup:create', 'F', 1002),
  (21, 12, '备份下载', '', '', 'backup:download', 'F', 1003);

-- 线上库增量补丁：补充按钮级菜单（可重复执行）
INSERT IGNORE INTO zt_menus (parentId, title, icon, path, perm, type, sort) VALUES
  (3, '分配角色', '', '', 'user:assignrole', 'F', 106),
  (3, '重置密码', '', '', 'user:resetpassword', 'F', 107),
  (7, '字典新增', '', '', 'dict:create', 'F', 502),
  (7, '字典编辑', '', '', 'dict:edit', 'F', 503),
  (7, '字典删除', '', '', 'dict:delete', 'F', 504),
  (11, '任务编辑', '', '', 'cronjob:edit', 'F', 902),
  (11, '任务触发', '', '', 'cronjob:trigger', 'F', 903),
  (12, '备份创建', '', '', 'backup:create', 'F', 1002),
  (12, '备份下载', '', '', 'backup:download', 'F', 1003);

INSERT IGNORE INTO zt_depts (id, tenantId, parentId, name, sort) VALUES
  (1, 0, 0, '总部', 1);

CREATE TABLE IF NOT EXISTS `zt_operation_logs` (
  `id`          BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `tenantId`    BIGINT NOT NULL DEFAULT 0,
  `userId`      BIGINT UNSIGNED NOT NULL DEFAULT 0,
  `account`     VARCHAR(64) NOT NULL DEFAULT '',
  `method`      VARCHAR(10) NOT NULL DEFAULT '',
  `path`        VARCHAR(255) NOT NULL DEFAULT '',
  `query`       VARCHAR(500) NOT NULL DEFAULT '',
  `body`        TEXT,
  `ip`          VARCHAR(64) NOT NULL DEFAULT '',
  `userAgent`   VARCHAR(255) NOT NULL DEFAULT '',
  `statusCode`  INT NOT NULL DEFAULT 0,
  `createdAt`   DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  PRIMARY KEY (`id`),
  KEY `idx_userId` (`userId`),
  KEY `idx_createdAt` (`createdAt`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS `zt_cron_jobs` (
  `id`         BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `name`       VARCHAR(64) NOT NULL DEFAULT '',
  `spec`       VARCHAR(64) NOT NULL DEFAULT '',
  `isEnabled`  TINYINT(1) NOT NULL DEFAULT 1,
  `remark`     VARCHAR(255) NOT NULL DEFAULT '',
  `lastRunAt`  DATETIME(3) DEFAULT NULL,
  `createdAt`  DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `updatedAt`  DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_name` (`name`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS `zt_cron_logs` (
  `id`        BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `jobName`   VARCHAR(64) NOT NULL DEFAULT '',
  `startAt`   DATETIME(3) NOT NULL,
  `endAt`     DATETIME(3) DEFAULT NULL,
  `success`   TINYINT(1) NOT NULL DEFAULT 0,
  `message`   TEXT,
  `createdAt` DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  PRIMARY KEY (`id`),
  KEY `idx_jobName` (`jobName`),
  KEY `idx_startAt` (`startAt`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS `zt_backup_records` (
  `id`          BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `filename`    VARCHAR(255) NOT NULL DEFAULT '',
  `sizeByte`    BIGINT NOT NULL DEFAULT 0,
  `createdBy`   BIGINT NOT NULL DEFAULT 0,
  `createdDate` DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `updatedBy`   BIGINT NOT NULL DEFAULT 0,
  `updatedDate` DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  `deleted`     TINYINT(1) NOT NULL DEFAULT 0,
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;