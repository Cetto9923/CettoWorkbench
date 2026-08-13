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

CREATE TABLE IF NOT EXISTS `zt_role_permissions` (
  `id` bigint NOT NULL AUTO_INCREMENT,
  `roleId` bigint NOT NULL,
  `permCode` varchar(64) NOT NULL DEFAULT '',
  `createdBy` bigint NOT NULL DEFAULT '0',
  `updatedBy` bigint NOT NULL DEFAULT '0',
  `createdDate` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updatedDate` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  `deletedAt` datetime DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_role_perm_deleted` (`roleId`,`permCode`,`deletedAt`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE `zt_roles` (
  `id` bigint NOT NULL AUTO_INCREMENT,
  `code` varchar(64) NOT NULL,
  `name` varchar(64) NOT NULL,
  `description` varchar(255) NOT NULL DEFAULT '',
  `isBuiltin` tinyint(1) NOT NULL DEFAULT '0',
  `isActive` tinyint(1) NOT NULL DEFAULT '1',
  `sortOrder` int NOT NULL DEFAULT '0',
  `createdBy` bigint NOT NULL DEFAULT '0',
  `updatedBy` bigint NOT NULL DEFAULT '0',
  `createdDate` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updatedDate` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  `deleted` tinyint(1) NOT NULL DEFAULT '0',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_zt_roles_code_deleted` (`code`,`deleted`),
  UNIQUE KEY `uk_zt_roles_name_deleted` (`name`,`deleted`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE `zt_gf_user_roles` (
  `id` bigint NOT NULL AUTO_INCREMENT,
  `userId` bigint NOT NULL,
  `roleId` bigint NOT NULL,
  `tenantId` bigint NOT NULL DEFAULT '0',
  `createdBy` bigint NOT NULL DEFAULT '0',
  `updatedBy` bigint NOT NULL DEFAULT '0',
  `createdDate` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updatedDate` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  `deleted` tinyint(1) NOT NULL DEFAULT '0',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_zt_userroles_user_role_deleted` (`tenantId`,`userId`,`roleId`,`deleted`),
  KEY `idx_zt_userroles_user` (`tenantId`,`userId`),
  KEY `idx_zt_userroles_role` (`tenantId`,`roleId`)
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
 (2,	1,	'工作台首页',	'fa-home',	'/home',	'po:home',	'C',	1),
 (3,	1,	'排期工作台',	'fa-calendar-check',	'/schedule',	'po:schedule',	'C',	2);

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
