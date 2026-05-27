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

INSERT IGNORE INTO zt_menus (id, parentId, title, icon, path, perm, type, sort) VALUES
  (1, 0, '工作台首页', 'bi-speedometer2', '/admin/dashboard', '', 'C', 1);