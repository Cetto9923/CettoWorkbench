-- V2 key/value preferences. Kept separate to preserve the legacy preferredRoles table.
CREATE TABLE IF NOT EXISTS `zt_wb_profile_pref_kv` (
 `id` bigint NOT NULL AUTO_INCREMENT,
 `account` varchar(64) NOT NULL DEFAULT '',
 `prefKey` varchar(64) NOT NULL DEFAULT '',
 `prefValue` text NOT NULL,
 `createdAt` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
 `updatedAt` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
 PRIMARY KEY (`id`), UNIQUE KEY `uk_account_key` (`account`,`prefKey`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- V2 agile team to organization mapping; retain history by switching status instead of deleting rows.
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
