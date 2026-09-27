-- Workbench V2: 敏捷团队到禅道组织团队挂靠关系。
-- 仅创建 Workbench 自有表；由部署负责人在备份并确认目标库后单独执行。
-- 本文件不会由应用启动时自动执行。
CREATE TABLE IF NOT EXISTS `zt_wb_agileteam_orgmap` (
    `id`           BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    `teamgroupId`  MEDIUMINT UNSIGNED NOT NULL COMMENT '禅道敏捷团队/小组 ID；父级挂靠可由子级继承',
    `deptId`       MEDIUMINT UNSIGNED NOT NULL COMMENT '禅道最底层组织部门 ID',
    `status`       VARCHAR(16) NOT NULL DEFAULT 'active' COMMENT 'active/inactive，按快照保留变更历史',
    `createdBy`    VARCHAR(30) NOT NULL DEFAULT '',
    `updatedBy`    VARCHAR(30) NOT NULL DEFAULT '',
    `createdDate`  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `updatedDate`  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (`id`),
    KEY `idx_agileteam_orgmap_current` (`teamgroupId`, `status`),
    KEY `idx_agileteam_orgmap_dept` (`deptId`, `status`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='Workbench敏捷团队组织挂靠历史';
