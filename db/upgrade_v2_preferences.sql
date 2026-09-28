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
