-- =============================================================================
-- 文件: db/01_main_base.sql
-- 来源: Main 分支基线（禅道产品团队开发）
-- 职责: 初始化工作台基础表结构与基础种子数据：
--       1. 登录防爆破与审计（zt_login_failures, zt_login_logs）
--       2. RBAC 基础角色权限（zt_roles, zt_role_permissions, zt_gf_user_roles）
--       3. 基础菜单导航（zt_menus）
--       4. 排期版本窗口核心表（zt_versionwindow, zt_versionwindowproduct, zt_demandwindow）
-- 适用: 数据源库全新初始化或恢复时的第 1 步执行
-- 规范: 100% 幂等性设计，支持安全重复执行无副作用
-- =============================================================================

SET NAMES utf8mb4;
SET FOREIGN_KEY_CHECKS = 0;

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

-- 7. Main 基础菜单种子数据（幂等插入/更新）
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

-- 8. Main 超级管理员角色与基础权限种子（幂等初始化）
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

-- 默认将 admin (userId=1) 关联到超级管理员
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

SET FOREIGN_KEY_CHECKS = 1;
