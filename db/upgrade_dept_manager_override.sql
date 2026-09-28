-- =============================================================================
-- 文件: db/upgrade_dept_manager_override.sql
-- 模块: 部门管理
-- 职责: 科技本部部门负责人补缺覆盖表（针对 52 科技部本部及其子孙部门）
-- 规范: 100% 幂等性设计，支持安全重复执行。
-- =============================================================================

CREATE TABLE IF NOT EXISTS `zt_wb_dept_manager_override` (
  `id` BIGINT NOT NULL AUTO_INCREMENT COMMENT '主键',
  `dept` INT NOT NULL DEFAULT 0 COMMENT '部门ID（zt_dept.id）',
  `account` VARCHAR(30) NOT NULL DEFAULT '' COMMENT '补缺部门负责人账号（zt_user.account）',
  `remark` VARCHAR(255) NOT NULL DEFAULT '' COMMENT '补缺说明/备注',
  `createdBy` VARCHAR(30) NOT NULL DEFAULT '' COMMENT '创建人账号',
  `createdDate` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `updatedBy` VARCHAR(30) NOT NULL DEFAULT '' COMMENT '更新人账号',
  `updatedDate` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  `deleted` ENUM('0', '1') NOT NULL DEFAULT '0' COMMENT '删除标记：0-有效，1-已删除',
  PRIMARY KEY (`id`),
  KEY `idx_dept_deleted` (`dept`, `deleted`),
  KEY `idx_account_deleted` (`account`, `deleted`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='科技本部部门负责人补缺覆盖表';
