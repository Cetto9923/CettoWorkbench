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
 (2,	1,	'工作台首页',	'fa-home',	'/po/home',	'po:home',	'C',	1),
 (3,	1,	'排期工作台',	'fa-calendar-check',	'/po/schedule',	'po:schedule',	'C',	2);

CREATE TABLE IF NOT EXISTS `version_window` (
    `id`            BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    `name`          VARCHAR(100) NOT NULL COMMENT '窗口名称，根据预计上线日期自动生成，如 26-0701窗口',
    `release_date`  DATE NOT NULL COMMENT '预计上线日期（=窗口结束日期）',
    `start_date`    DATE DEFAULT NULL COMMENT '窗口开始日期，用户手填',
    `teamgroup_id`  MEDIUMINT UNSIGNED NOT NULL COMMENT '关联敏捷小组，对应 zt_teamgroup.id',
    `group_size`    INT UNSIGNED NOT NULL DEFAULT 1 COMMENT '小组人数，用于容量计算（工作日×7h×人数）',
    `created_by`    VARCHAR(30) NOT NULL COMMENT '创建人账号，对应 zt_team.account / zt_user.account',
    `status`        VARCHAR(20) NOT NULL DEFAULT 'planning' COMMENT 'current/next/planning/released',
    `sort_order`    INT NOT NULL DEFAULT 0,
    `deleted`       TINYINT(1) NOT NULL DEFAULT 0,
    `created_at`    DATETIME DEFAULT CURRENT_TIMESTAMP,
    `updated_at`    DATETIME DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    INDEX `idx_teamgroup` (`teamgroup_id`),
    INDEX `idx_release_date` (`release_date`),
    INDEX `idx_created_by` (`created_by`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='版本窗口';

CREATE TABLE IF NOT EXISTS `version_window_product` (
    `id`            BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    `window_id`     BIGINT UNSIGNED NOT NULL COMMENT '版本窗口ID',
    `product_id`    MEDIUMINT UNSIGNED NOT NULL COMMENT '关联产品/系统，对应 zt_product.id',
    `plan_id`       MEDIUMINT UNSIGNED DEFAULT NULL COMMENT '匹配到的禅道计划ID，对应 zt_productplan.id，NULL表示待建计划',
    `plan_synced`   TINYINT(1) NOT NULL DEFAULT 0 COMMENT '计划是否已同步到禅道（0=待同步/待建，1=已同步）',
    `created_at`    DATETIME DEFAULT CURRENT_TIMESTAMP,
    INDEX `idx_window` (`window_id`),
    INDEX `idx_product` (`product_id`),
    UNIQUE KEY `uk_window_product` (`window_id`, `product_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='版本窗口关联产品/系统';
