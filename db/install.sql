CREATE TABLE IF NOT EXISTS `zt_login_failures` (
  `id` BIGINT NOT NULL AUTO_INCREMENT,
  `account` VARCHAR(64) NOT NULL,
  `ip` VARCHAR(45) NOT NULL,
  `failedAt` DATETIME NOT NULL,
  `createdDate` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `idx_zt_loginfailures_account_time` (`account`, `failedAt`),
  KEY `idx_zt_loginfailures_ip_time` (`ip`, `failedAt`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS zt_login_logs (
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

INSERT INTO `zt_menus` (`id`, `parentId`, `title`, `icon`, `path`, `perm`, `type`, `sort`) VALUES
 (1,	0,	'PO专属',	'',	'',	'',	'M',	100),
 (2,	1,	'工作台首页',	'fa-home',	'/po/home',	'po:home',	'C',	1)
 (3,	1,	'排期工作台',	'fa-calendar-check',	'/po/schedule',	'po:schedule',	'C',	2);