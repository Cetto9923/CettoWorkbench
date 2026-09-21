-- =============================================================================
-- 文件: db/install.sql
-- 职责: 工作台全量数据库安装脚本（包含 01_main_base.sql 与 02_po_incremental.sql）
-- 说明:
--   1. 若希望分层初始化，请按顺序执行：
--      - 第一步：db/01_main_base.sql（Main 基线，禅道产品团队开发）
--      - 第二步：db/02_po_incremental.sql（当前开发分支 PO 角色增量）
--   2. 若全新环境希望单文件一次性初始化，直接执行本脚本即可。
-- 规范: 100% 幂等性设计，支持安全重复执行。
-- =============================================================================

SET NAMES utf8mb4;
SET FOREIGN_KEY_CHECKS = 0;

-- -----------------------------------------------------------------------------
-- 第一部分：Main 基线表结构与基础数据（禅道产品团队开发）
-- -----------------------------------------------------------------------------

-- 1. 登录失败防爆破记录表
CREATE TABLE IF NOT EXISTS `zt_login_failures` (
  `id` BIGINT NOT NULL AUTO_INCREMENT,
  `account` VARCHAR(64) NOT NULL,
  `ip` VARCHAR(45) NOT NULL,
  `failedAt` DATETIME NOT NULL,
  `createdDate` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `idx_zt_loginfailures_account_time` (`account`, `failedAt`),
  KEY `idx_zt_loginfailures_ip_time` (`ip`, `failedAt`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='工作台登录失败记录';

-- 2. 登录审计日志表
CREATE TABLE IF NOT EXISTS `zt_login_logs` (
  `id` BIGINT NOT NULL AUTO_INCREMENT,
  `account` VARCHAR(64) NOT NULL,
  `userId` BIGINT NULL,
  `ip` VARCHAR(45) NOT NULL,
  `userAgent` VARCHAR(512) NOT NULL,
  `success` TINYINT(1) NOT NULL DEFAULT 0,
  `failReason` VARCHAR(64) NOT NULL DEFAULT '',
  `createdDate` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `idx_zt_loginlogs_account_time` (`account`, `createdDate`),
  KEY `idx_zt_loginlogs_ip_time` (`ip`, `createdDate`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='工作台登录审计日志';

-- 3. 系统角色表
CREATE TABLE IF NOT EXISTS `zt_roles` (
  `id` BIGINT NOT NULL AUTO_INCREMENT,
  `code` VARCHAR(64) NOT NULL,
  `name` VARCHAR(64) NOT NULL,
  `description` VARCHAR(255) NOT NULL DEFAULT '',
  `isBuiltin` TINYINT(1) NOT NULL DEFAULT '0',
  `isActive` TINYINT(1) NOT NULL DEFAULT '1',
  `sortOrder` INT NOT NULL DEFAULT '0',
  `createdBy` VARCHAR(30) NOT NULL DEFAULT '',
  `createdDate` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updatedBy` VARCHAR(30) NOT NULL DEFAULT '',
  `updatedDate` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  `deletedAt` DATETIME DEFAULT NULL,
  `deleted` TINYINT(1) NOT NULL DEFAULT '0',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_zt_roles_code` (`code`,`deletedAt`),
  KEY `idx_zt_roles_active` (`isActive`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='工作台系统角色表';

-- 4. 角色权限关联表
CREATE TABLE IF NOT EXISTS `zt_role_permissions` (
  `id` BIGINT NOT NULL AUTO_INCREMENT,
  `roleId` BIGINT NOT NULL,
  `permCode` VARCHAR(64) NOT NULL DEFAULT '',
  `createdBy` VARCHAR(30) NOT NULL DEFAULT '',
  `createdDate` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updatedBy` VARCHAR(30) NOT NULL DEFAULT '',
  `updatedDate` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  `deletedAt` DATETIME DEFAULT NULL,
  `deleted` TINYINT(1) NOT NULL DEFAULT '0',
  PRIMARY KEY (`id`),
  KEY `idx_zt_role_permissions_role` (`roleId`),
  KEY `idx_zt_role_permissions_perm` (`permCode`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='工作台角色权限关联表';

-- 5. 用户角色关联表
CREATE TABLE IF NOT EXISTS `zt_gf_user_roles` (
  `id` BIGINT NOT NULL AUTO_INCREMENT,
  `userId` BIGINT NOT NULL,
  `roleId` BIGINT NOT NULL,
  `createdBy` VARCHAR(30) NOT NULL DEFAULT '',
  `createdDate` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updatedBy` VARCHAR(30) NOT NULL DEFAULT '',
  `updatedDate` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  `deletedAt` DATETIME DEFAULT NULL,
  `deleted` TINYINT(1) NOT NULL DEFAULT '0',
  PRIMARY KEY (`id`),
  KEY `idx_zt_gf_user_roles_user` (`userId`),
  KEY `idx_zt_gf_user_roles_role` (`roleId`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='工作台用户角色关联表';

-- 6. 工作台菜单表
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
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='工作台菜单配置';

-- 7. Main 基础菜单种子数据
INSERT INTO `zt_menus` (`id`, `parentId`, `title`, `icon`, `path`, `perm`, `type`, `sort`) VALUES
 (1, 0, '个人入口', '', '', '', 'M', 100),
 (2, 1, '首页', 'fa-home', '/home', 'po:home', 'C', 1),
 (4, 0, '工作区', '', '', '', 'M', 200),
 (3, 4, '需求排期', 'fa-calendar-check', '/schedule', 'po:schedule', 'C', 1)
ON DUPLICATE KEY UPDATE
 `parentId` = VALUES(`parentId`),
 `title`    = VALUES(`title`),
 `icon`     = VALUES(`icon`),
 `path`     = VALUES(`path`),
 `perm`     = VALUES(`perm`),
 `type`     = VALUES(`type`),
 `sort`     = VALUES(`sort`);

-- 8. Main 超级管理员角色与权限种子
INSERT INTO `zt_roles` (`id`, `code`, `name`, `description`, `isBuiltin`, `isActive`, `sortOrder`) VALUES
 (1, 'super_admin', '超级管理员', '系统内置超级管理员，拥有系统所有能力', 1, 1, 0)
ON DUPLICATE KEY UPDATE
 `name` = VALUES(`name`),
 `description` = VALUES(`description`);

INSERT INTO `zt_role_permissions` (`roleId`, `permCode`, `createdBy`) VALUES
 (1, 'auth:logout', '1'),
 (1, 'user:list', '1'),
 (1, 'user:create', '1'),
 (1, 'user:update', '1'),
 (1, 'user:delete', '1'),
 (1, 'user:assignrole', '1'),
 (1, 'user:resetpassword', '1'),
 (1, 'operationlog:list', '1'),
 (1, 'loginlog:list', '1'),
 (1, 'role:list', '1'),
 (1, 'role:create', '1'),
 (1, 'role:edit', '1'),
 (1, 'role:delete', '1'),
 (1, 'menu:list', '1'),
 (1, 'menu:create', '1'),
 (1, 'menu:edit', '1'),
 (1, 'menu:delete', '1'),
 (1, 'dept:list', '1'),
 (1, 'dept:create', '1'),
 (1, 'dept:edit', '1'),
 (1, 'dept:delete', '1'),
 (1, 'schedule:list', '1'),
 (1, 'schedule:create', '1'),
 (1, 'schedule:update', '1'),
 (1, 'schedule:delete', '1'),
 (1, 'po:home', '1'),
 (1, 'po:schedule', '1')
ON DUPLICATE KEY UPDATE `updatedDate` = NOW();

-- 默认将 admin (id=1) 关联到超级管理员
INSERT INTO `zt_gf_user_roles` (`id`, `userId`, `roleId`, `createdBy`) VALUES
 (1, 1, 1, '1')
ON DUPLICATE KEY UPDATE `updatedDate` = NOW();

-- 9. 版本窗口核心主表
CREATE TABLE IF NOT EXISTS `zt_versionwindow` (
    `id`            BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    `name`          VARCHAR(100) NOT NULL COMMENT '窗口名称，根据预计上线日期自动生成，如 26-0701窗口',
    `releaseDate`   DATE NOT NULL COMMENT '预计上线日期（=窗口结束日期）',
    `startDate`     DATE DEFAULT NULL COMMENT '窗口开始日期，用户手填',
    `teamgroup`     MEDIUMINT UNSIGNED NOT NULL COMMENT '关联敏捷小组，对应 zt_teamgroup.id',
    `groupSize`     INT UNSIGNED NOT NULL DEFAULT 1 COMMENT '小组人数，用于容量计算（工作日×7h×人数）',
    `createdBy`     VARCHAR(30) NOT NULL COMMENT '创建人账号，对应 zt_team.account / zt_user.account',
    `updatedBy`     VARCHAR(30) NOT NULL DEFAULT '' COMMENT '最后更新人账号',
    `status`        VARCHAR(20) NOT NULL DEFAULT 'planning' COMMENT 'current/next/planning/released',
    `order`         INT NOT NULL DEFAULT 0,
    `createdDate`   DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `updatedDate`   DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    `deletedAt`     DATETIME(3) DEFAULT NULL,
    INDEX `idx_teamgroup` (`teamgroup`),
    INDEX `idx_releaseDate` (`releaseDate`),
    INDEX `idx_createdBy` (`createdBy`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='版本窗口';

-- 10. 版本窗口关联产品/系统表
CREATE TABLE IF NOT EXISTS `zt_versionwindowproduct` (
    `id`            BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    `versionWindow` BIGINT UNSIGNED NOT NULL COMMENT '版本窗口ID',
    `product`       MEDIUMINT UNSIGNED NOT NULL COMMENT '关联产品/系统，对应 zt_product.id',
    `plan`          MEDIUMINT UNSIGNED DEFAULT NULL COMMENT '匹配到的禅道计划ID，对应 zt_productplan.id，NULL表示待建计划',
    `planSynced`    TINYINT(1) NOT NULL DEFAULT 0 COMMENT '计划是否已同步到禅道（0=待同步/待建，1=已同步）',
    `createdBy`     VARCHAR(30) NOT NULL DEFAULT '' COMMENT '创建人账号',
    `updatedBy`     VARCHAR(30) NOT NULL DEFAULT '' COMMENT '最后更新人账号',
    `createdDate`   DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `updatedDate`   DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    `deletedAt`     DATETIME(3) DEFAULT NULL,
    INDEX `idx_versionWindow` (`versionWindow`),
    INDEX `idx_product` (`product`),
    UNIQUE KEY `uk_versionWindow_product` (`versionWindow`, `product`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='版本窗口关联产品/系统';

-- 10.1 Workbench 自有版本窗口基础里程碑表
CREATE TABLE IF NOT EXISTS `zt_wb_versionwindow_milestone` (
    `versionWindow` BIGINT UNSIGNED NOT NULL COMMENT '版本窗口ID，对应 zt_versionwindow.id',
    `planTestDone`  DATE DEFAULT NULL COMMENT '预计提测/开发完成日期',
    `testDone`      DATE DEFAULT NULL COMMENT '预计测试完成日期',
    `acceptDone`    DATE DEFAULT NULL COMMENT '预计验收完成日期',
    `createdBy`     VARCHAR(30) NOT NULL DEFAULT '' COMMENT '创建人账号',
    `updatedBy`     VARCHAR(30) NOT NULL DEFAULT '' COMMENT '最后更新人账号',
    `createdDate`   DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `updatedDate`   DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    `deletedAt`     DATETIME(3) DEFAULT NULL,
    PRIMARY KEY (`versionWindow`),
    INDEX `idx_deletedAt` (`deletedAt`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='Workbench版本窗口基础里程碑';

-- 11. 业务需求-窗口关联表
CREATE TABLE IF NOT EXISTS `zt_demandwindow` (
    `id`            BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    `demand`        MEDIUMINT UNSIGNED NOT NULL COMMENT '业务需求ID，对应 zt_demand.id',
    `story`         MEDIUMINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '研发需求ID，0=业需级关联（固定值），对应 zt_story.id',
    `versionWindow` BIGINT UNSIGNED NOT NULL COMMENT '版本窗口ID，对应 zt_versionwindow.id',
    `createdBy`     VARCHAR(30) NOT NULL DEFAULT '' COMMENT '创建人账号',
    `updatedBy`     VARCHAR(30) NOT NULL DEFAULT '' COMMENT '最后更新人账号',
    `createdDate`   DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `updatedDate`   DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    `deletedAt`     DATETIME(3) DEFAULT NULL,
    INDEX `idx_demand` (`demand`),
    INDEX `idx_story` (`story`),
    INDEX `idx_versionWindow` (`versionWindow`),
    UNIQUE KEY `uk_demand_story` (`demand`, `story`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='业务需求-窗口关联（业需级单值：demand+story(=0) 唯一）';

-- -----------------------------------------------------------------------------
-- 第二部分：当前分支 PO 角色工作台增量扩展
-- -----------------------------------------------------------------------------

-- 12. 工作台操作审计日志表
CREATE TABLE IF NOT EXISTS `zt_operation_logs` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `tenantId` BIGINT NOT NULL DEFAULT 0,
  `userId` BIGINT UNSIGNED NOT NULL DEFAULT 0,
  `account` VARCHAR(64) NOT NULL DEFAULT '',
  `method` VARCHAR(10) NOT NULL DEFAULT '',
  `path` VARCHAR(255) NOT NULL DEFAULT '',
  `query` VARCHAR(500) NOT NULL DEFAULT '',
  `body` TEXT,
  `ip` VARCHAR(64) NOT NULL DEFAULT '',
  `userAgent` VARCHAR(255) NOT NULL DEFAULT '',
  `statusCode` BIGINT NOT NULL DEFAULT 0,
  `createdAt` DATETIME(3) DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `idx_account_createdAt` (`account`, `createdAt`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='工作台操作审计日志';

-- 13. 工作台通知已读记录表
CREATE TABLE IF NOT EXISTS `zt_workbench_notify_reads` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `notify` BIGINT UNSIGNED NOT NULL COMMENT '关联 zt_notify.id',
  `account` VARCHAR(64) NOT NULL DEFAULT '' COMMENT '已读用户账号',
  `readAt` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '已读时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_workbench_notify_account` (`notify`, `account`),
  KEY `idx_workbench_notify_account` (`account`, `readAt`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='工作台通知已读记录';

-- 14. 工作台部门管理表
CREATE TABLE IF NOT EXISTS `zt_depts` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `tenantId` BIGINT NOT NULL DEFAULT 0,
  `parentId` BIGINT UNSIGNED NOT NULL DEFAULT 0,
  `name` VARCHAR(64) NOT NULL DEFAULT '',
  `leader` VARCHAR(64) NOT NULL DEFAULT '',
  `phone` VARCHAR(32) NOT NULL DEFAULT '',
  `email` VARCHAR(64) NOT NULL DEFAULT '',
  `status` TINYINT UNSIGNED NOT NULL DEFAULT 0,
  `sort` INT NOT NULL DEFAULT 0,
  `createdAt` DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `updatedAt` DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  `deletedAt` DATETIME(3) DEFAULT NULL,
  `deleted` TINYINT(1) NOT NULL DEFAULT 0,
  PRIMARY KEY (`id`),
  KEY `idx_zt_depts_parent` (`parentId`),
  KEY `idx_zt_depts_status` (`status`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='工作台部门管理';

-- 15. 初始化 PO 角色
INSERT INTO `zt_roles` (`id`, `code`, `name`, `description`, `isBuiltin`, `isActive`, `sortOrder`, `createdBy`) VALUES
 (2, 'po', '产品负责人', '负责需求梳理、价值流跟踪、排期推进与跨团队协同', 1, 1, 1, '1')
ON DUPLICATE KEY UPDATE
 `name` = VALUES(`name`),
 `description` = VALUES(`description`);

-- 16. 扩充 PO 权限点
INSERT INTO `zt_role_permissions` (`roleId`, `permCode`, `createdBy`) VALUES
 -- 超级管理员扩展 PO 权限
 (1, 'po:todo', '1'),
 (1, 'po:done', '1'),
 (1, 'po:notice', '1'),
 (1, 'po:notice:update', '1'),
 (1, 'po:follow', '1'),
 (1, 'po:follow:update', '1'),
 (1, 'po:boarddemand:list', '1'),
 (1, 'po:boardtask:list', '1'),
 (1, 'po:demandreview', '1'),
 -- PO 角色专属业务权限
 (2, 'auth:logout', '1'),
 (2, 'po:home', '1'),
 (2, 'po:todo', '1'),
 (2, 'po:done', '1'),
 (2, 'po:notice', '1'),
 (2, 'po:notice:update', '1'),
 (2, 'po:follow', '1'),
 (2, 'po:follow:update', '1'),
 (2, 'po:boarddemand:list', '1'),
 (2, 'po:boardtask:list', '1'),
 (2, 'po:demandreview', '1'),
 (2, 'po:schedule', '1'),
 (2, 'schedule:list', '1')
ON DUPLICATE KEY UPDATE `updatedDate` = NOW();

-- 确保示例 PO 用户 wangweijia 处于启用状态 (deleted = '0')
UPDATE `zt_user` SET `deleted` = '0' WHERE `account` = 'wangweijia';

-- 默认将内置 PO 用户 wangweijia (若存在) 绑定到 PO 角色
INSERT INTO `zt_gf_user_roles` (`userId`, `roleId`, `createdBy`)
SELECT id, 2, '1' FROM `zt_user` WHERE `account` = 'wangweijia'
ON DUPLICATE KEY UPDATE `updatedDate` = NOW();

-- 默认将示例 PO 用户 wangweijia 赋予禅道端 PO 组权限（group 7: PO, group 22: 基础权限）
INSERT IGNORE INTO `zt_usergroup` (`account`, `group`) VALUES ('wangweijia', 7), ('wangweijia', 22);

-- 17. 扩充工作台前端二级导航菜单
INSERT INTO `zt_menus` (`id`, `parentId`, `title`, `icon`, `path`, `perm`, `type`, `sort`) VALUES
 -- 个人入口 (parentId = 1)
 (10, 1, '我的待办', 'fa-tasks',          '/todos',         'po:todo',              'C', 2),
 (11, 1, '我的已办', 'fa-check-circle',    '/done',          'po:done',              'C', 3),
 (12, 1, '通知中心', 'fa-bell',            '/notice',        'po:notice',            'C', 4),
 (13, 1, '我的关注', 'fa-star',            '/follow',        'po:follow',            'C', 5),
 -- 工作区 (parentId = 4)
 (20, 4, '工作看板', 'fa-columns',         '/board/demand',  'po:boarddemand:list',  'C', 2),
 (21, 4, '综合查询', 'fa-search',          '/query',         'po:home',              'C', 3)
ON DUPLICATE KEY UPDATE
 `parentId` = VALUES(`parentId`),
 `title`    = VALUES(`title`),
 `icon`     = VALUES(`icon`),
 `path`     = VALUES(`path`),
 `perm`     = VALUES(`perm`),
 `type`     = VALUES(`type`),
 `sort`     = VALUES(`sort`);

SET FOREIGN_KEY_CHECKS = 1;
