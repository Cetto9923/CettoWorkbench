-- Workbench 自有表。只用于经审核的安装/升级，不在应用运行期执行。
CREATE TABLE IF NOT EXISTS `zt_wb_agileteam_adjustment` (
  `id` BIGINT NOT NULL AUTO_INCREMENT PRIMARY KEY,
  `teamgroupId` INT UNSIGNED NOT NULL,
  `adjustNo` VARCHAR(32) NOT NULL,
  `status` VARCHAR(20) NOT NULL DEFAULT 'pending',
  `reason` VARCHAR(500) NOT NULL DEFAULT '',
  `submittedBy` VARCHAR(30) NOT NULL,
  `confirmedBy` VARCHAR(30) NOT NULL DEFAULT '',
  `confirmedDate` DATETIME NULL,
  `rejectedBy` VARCHAR(30) NOT NULL DEFAULT '',
  `rejectedDate` DATETIME NULL,
  `rejectReason` VARCHAR(500) NOT NULL DEFAULT '',
  `createdBy` VARCHAR(30) NOT NULL DEFAULT '',
  `createdDate` DATETIME NOT NULL,
  `updatedBy` VARCHAR(30) NOT NULL DEFAULT '',
  `updatedDate` DATETIME NOT NULL,
  `deletedAt` DATETIME NULL,
  KEY `idx_adjustment_group_status` (`teamgroupId`, `status`, `deletedAt`),
  KEY `idx_adjustment_no` (`adjustNo`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS `zt_wb_agileteam_adjustment_item` (
  `id` BIGINT NOT NULL AUTO_INCREMENT PRIMARY KEY,
  `adjustmentId` BIGINT NOT NULL,
  `account` VARCHAR(30) NOT NULL,
  `actionType` VARCHAR(20) NOT NULL,
  `role` VARCHAR(64) NOT NULL DEFAULT '',
  `prevRole` VARCHAR(64) NOT NULL DEFAULT '',
  `availableHours` DOUBLE NOT NULL DEFAULT 0,
  `prevHours` DOUBLE NOT NULL DEFAULT 0,
  `createdBy` VARCHAR(30) NOT NULL DEFAULT '',
  `createdDate` DATETIME NOT NULL,
  `updatedBy` VARCHAR(30) NOT NULL DEFAULT '',
  `updatedDate` DATETIME NOT NULL,
  `deletedAt` DATETIME NULL,
  KEY `idx_adjustment_item` (`adjustmentId`, `deletedAt`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS `zt_wb_agileteam_history` (
  `id` BIGINT NOT NULL AUTO_INCREMENT PRIMARY KEY,
  `teamgroupId` INT UNSIGNED NOT NULL,
  `eventType` VARCHAR(32) NOT NULL,
  `adjustmentId` BIGINT NULL,
  `summary` VARCHAR(500) NOT NULL DEFAULT '',
  `actor` VARCHAR(30) NOT NULL DEFAULT '',
  `createdBy` VARCHAR(30) NOT NULL DEFAULT '',
  `createdDate` DATETIME NOT NULL,
  `updatedBy` VARCHAR(30) NOT NULL DEFAULT '',
  `updatedDate` DATETIME NULL,
  `deletedAt` DATETIME NULL,
  KEY `idx_agile_history` (`teamgroupId`, `createdDate`, `id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

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
