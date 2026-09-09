-- =============================================================================
-- 文件: db/02_po_incremental.sql
-- 来源: Claude-PO 开发分支（PO 角色工作台增量）
-- 职责: 补充 PO 工作台当前分支所依赖的全部增量结构与业务数据：
--       1. 操作审计表 zt_operation_logs（受控迁移创建，彻底消除运行时动态建表）
--       2. 通知已读表 zt_workbench_notify_reads（支撑通知中心单条与批量已读）
--       3. 部门管理表 zt_depts（支撑后台部门管理与组织树）
--       4. PO 专属角色定义（zt_roles）与全套权限点（zt_role_permissions）
--       5. PO 工作台完整导航菜单项扩展（zt_menus：待办、已办、通知、关注、看板、查询）
-- 依赖: 必须在 db/01_main_base.sql 执行完毕后执行
-- 规范: 100% 幂等性设计，支持安全重复执行无副作用
-- =============================================================================

SET NAMES utf8mb4;
SET FOREIGN_KEY_CHECKS = 0;

-- 1. 工作台操作审计日志表（替代应用运行时的 DDL 自动建表）
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

-- 2. 工作台通知已读记录表（支撑通知中心“标为已读”与“全部已读”能力）
CREATE TABLE IF NOT EXISTS `zt_workbench_notify_reads` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `notify` BIGINT UNSIGNED NOT NULL COMMENT '关联 zt_notify.id',
  `account` VARCHAR(64) NOT NULL DEFAULT '' COMMENT '已读用户账号',
  `readAt` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '已读时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_workbench_notify_account` (`notify`, `account`),
  KEY `idx_workbench_notify_account` (`account`, `readAt`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='工作台通知已读记录';

-- 3. 工作台部门管理表（供 internal/module/dept 模块使用）
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

-- 4. 初始化 PO（产品负责人）角色定义
INSERT INTO `zt_roles` (`id`, `code`, `name`, `description`, `isBuiltin`, `isActive`, `sortOrder`, `createdBy`) VALUES
 (2, 'po', '产品负责人', '负责需求梳理、价值流跟踪、排期推进与跨团队协同', 1, 1, 1, '1')
ON DUPLICATE KEY UPDATE
 `name` = VALUES(`name`),
 `description` = VALUES(`description`);

-- 5. 扩充 PO 工作台全套权限点（同时给超级管理员 roleId=1 与 PO 角色 roleId=2 赋权）
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

-- 默认将示例 PO 用户 wangweijia (若存在) 绑定到 PO 角色
INSERT INTO `zt_gf_user_roles` (`userId`, `roleId`, `createdBy`)
SELECT id, 2, '1' FROM `zt_user` WHERE `account` = 'wangweijia'
ON DUPLICATE KEY UPDATE `updatedDate` = NOW();

-- 默认将示例 PO 用户 wangweijia 赋予禅道端 PO 组权限（group 7: PO, group 22: 基础权限）
INSERT IGNORE INTO `zt_usergroup` (`account`, `group`) VALUES ('wangweijia', 7), ('wangweijia', 22);

-- 6. 扩充工作台前端导航菜单（补全 PO 核心场景二级菜单）
INSERT INTO `zt_menus` (`id`, `parentId`, `title`, `icon`, `path`, `perm`, `type`, `sort`) VALUES
 -- 个人入口 (parentId = 1) 下的业务二级场景
 (10, 1, '我的待办', 'fa-tasks',          '/todos',         'po:todo',              'C', 2),
 (11, 1, '我的已办', 'fa-check-circle',    '/done',          'po:done',              'C', 3),
 (12, 1, '通知中心', 'fa-bell',            '/notice',        'po:notice',            'C', 4),
 (13, 1, '我的关注', 'fa-star',            '/follow',        'po:follow',            'C', 5),
 -- 工作区 (parentId = 4) 下的协同视图
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
