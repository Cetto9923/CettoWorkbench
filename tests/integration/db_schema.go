//go:build integration

// =============================================================================
// 文件: tests/integration/db_schema.go
// 模块: 性能基线测试
// 类型: test
// 职责: 定义隔离测试库所需最小 ZenTao / Workbench Schema DDL。
// 依赖: gorm.io/gorm
// =============================================================================

package integration

import (
	"context"
	"fmt"

	"gorm.io/gorm"
)

// InitMinimalSchema 在隔离测试数据库中创建测试所需的最小表结构。
func InitMinimalSchema(ctx context.Context, db *gorm.DB) error {
	_ = db.Exec("SET SESSION sql_mode = 'NO_AUTO_VALUE_ON_ZERO'").Error

	tables := []string{
		"zt_workbench_notify_reads", "zt_notify", "zt_action", "zt_user", "zt_demandclarify",
		"zt_demand", "zt_task", "zt_story", "zt_bug", "zt_versionwindowproduct", "zt_demandwindow",
		"zt_versionwindow", "zt_planstory", "zt_dept", "zt_approvalnode", "zt_approvalobject",
		"zt_charter", "zt_project", "zt_planchange", "zt_projectbuildguide", "zt_review",
		"zt_testtask", "zt_issue", "zt_risk", "zt_todo", "zt_demandappraise",
		"zt_company", "zt_demandmanagerreview",
		"zt_product", "zt_productplan", "zt_teamgroup",
		"zt_holiday", "zt_config",
	}
	for _, t := range tables {
		_ = db.Exec(fmt.Sprintf("DROP TABLE IF EXISTS %s", t)).Error
	}

	ddls := []string{
		`CREATE TABLE IF NOT EXISTS zt_notify (
			id int unsigned NOT NULL AUTO_INCREMENT,
			objectType varchar(50) NOT NULL DEFAULT '',
			objectID int unsigned NOT NULL DEFAULT '0',
			action int unsigned NOT NULL DEFAULT '0',
			toList text,
			subject text,
			data text,
			createdBy char(30) NOT NULL DEFAULT '',
			createdDate datetime DEFAULT NULL,
			sendTime datetime DEFAULT NULL,
			status varchar(10) NOT NULL DEFAULT '',
			PRIMARY KEY (id)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`,

		`CREATE TABLE IF NOT EXISTS zt_action (
			id int unsigned NOT NULL AUTO_INCREMENT,
			objectType varchar(30) NOT NULL DEFAULT '',
			objectID int unsigned NOT NULL DEFAULT '0',
			product text,
			project int unsigned NOT NULL DEFAULT '0',
			execution int unsigned NOT NULL DEFAULT '0',
			actor varchar(100) NOT NULL DEFAULT '',
			action varchar(80) NOT NULL DEFAULT '',
			date datetime DEFAULT NULL,
			comment text,
			extra text,
			` + "`read`" + ` enum('0','1') NOT NULL DEFAULT '0',
			PRIMARY KEY (id)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`,

		`CREATE TABLE IF NOT EXISTS zt_workbench_notify_reads (
			id bigint unsigned NOT NULL AUTO_INCREMENT,
			notify bigint unsigned NOT NULL,
			account varchar(64) NOT NULL,
			readAt datetime DEFAULT CURRENT_TIMESTAMP,
			PRIMARY KEY (id),
			UNIQUE KEY uk_notify_account (notify, account)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`,

		`CREATE TABLE IF NOT EXISTS zt_user (
			id mediumint unsigned NOT NULL AUTO_INCREMENT,
			dept mediumint unsigned NOT NULL DEFAULT '0',
			account char(30) NOT NULL DEFAULT '',
			realname varchar(100) NOT NULL DEFAULT '',
			role char(30) NOT NULL DEFAULT '',
			deleted enum('0','1') NOT NULL DEFAULT '0',
			locked datetime DEFAULT NULL,
			PRIMARY KEY (id),
			UNIQUE KEY account (account)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`,

		`CREATE TABLE IF NOT EXISTS zt_demand (
			id int NOT NULL AUTO_INCREMENT,
			parent mediumint NOT NULL DEFAULT '0',
			pool int NOT NULL DEFAULT '0',
			pri char(30) NOT NULL DEFAULT '',
			name varchar(255) NOT NULL DEFAULT '',
			` + "`desc`" + ` longtext,
			assignedTo char(30) NOT NULL DEFAULT '',
			distributedBy varchar(30) NOT NULL DEFAULT '',
			QD char(30) NOT NULL DEFAULT '',
			RD char(30) NOT NULL DEFAULT '',
			BRA char(30) NOT NULL DEFAULT '',
			accepter char(30) NOT NULL DEFAULT '',
			status char(30) NOT NULL DEFAULT '',
			stage enum('wait','distributed','inroadmap','incharter','developing','delivering','delivered','closed') DEFAULT 'wait',
			hang enum('0','1') DEFAULT '0',
			hangUpReason longtext,
			deadline date DEFAULT NULL,
			developFinish date DEFAULT NULL,
			testFinish date DEFAULT NULL,
			verifyFinish date DEFAULT NULL,
			estimateLaunch date DEFAULT NULL,
			deliverDate date DEFAULT NULL,
			mainDevelopers text,
			overall varchar(30) NOT NULL DEFAULT '',
			createdBy char(30) NOT NULL DEFAULT '',
			createdDate datetime DEFAULT NULL,
			deleted enum('0','1') NOT NULL DEFAULT '0',
			product varchar(255) NOT NULL DEFAULT '',
			PRIMARY KEY (id)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`,

		`CREATE TABLE IF NOT EXISTS zt_demandclarify (
			id int NOT NULL AUTO_INCREMENT,
			demand mediumint NOT NULL DEFAULT '0',
			product varchar(255) NOT NULL DEFAULT '',
			PM longtext,
			PRIMARY KEY (id),
			KEY idx_demand (demand)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`,

		`CREATE TABLE IF NOT EXISTS zt_demandappraise (
			id int NOT NULL AUTO_INCREMENT,
			demand int NOT NULL DEFAULT '0',
			appraiseBy varchar(30) NOT NULL DEFAULT '',
			appraiseTime datetime DEFAULT NULL,
			PRIMARY KEY (id),
			KEY idx_demand (demand)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`,

		`CREATE TABLE IF NOT EXISTS zt_company (
			id int NOT NULL AUTO_INCREMENT,
			name varchar(100) NOT NULL DEFAULT '',
			admins text,
			PRIMARY KEY (id)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`,

		`CREATE TABLE IF NOT EXISTS zt_demandmanagerreview (
			id int NOT NULL AUTO_INCREMENT,
			demand int NOT NULL DEFAULT '0',
			resultStatus text,
			PRIMARY KEY (id),
			KEY idx_demand (demand)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`,

		`CREATE TABLE IF NOT EXISTS zt_task (
			id mediumint unsigned NOT NULL AUTO_INCREMENT,
			project mediumint unsigned NOT NULL DEFAULT '0',
			parent mediumint NOT NULL DEFAULT '0',
			story mediumint unsigned NOT NULL DEFAULT '0',
			name varchar(255) NOT NULL DEFAULT '',
			type varchar(20) NOT NULL DEFAULT '',
			pri tinyint unsigned NOT NULL DEFAULT '0',
			estimate float unsigned NOT NULL DEFAULT '0',
			consumed float unsigned NOT NULL DEFAULT '0',
			deadline date DEFAULT NULL,
			status enum('wait','doing','done','pause','cancel','closed') NOT NULL DEFAULT 'wait',
			assignedTo varchar(30) NOT NULL DEFAULT '',
			openedBy varchar(30) NOT NULL DEFAULT '',
			deleted enum('0','1') NOT NULL DEFAULT '0',
			PRIMARY KEY (id)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`,

		`CREATE TABLE IF NOT EXISTS zt_story (
			id mediumint unsigned NOT NULL AUTO_INCREMENT,
			product mediumint unsigned NOT NULL DEFAULT '0',
			fromDemand mediumint unsigned NOT NULL DEFAULT '0',
			title varchar(255) NOT NULL DEFAULT '',
			pri tinyint unsigned NOT NULL DEFAULT '3',
			status enum('','changing','active','draft','closed','reviewing','launched','developing') NOT NULL DEFAULT '',
			stage enum('','wait','inroadmap','incharter','planned','projected','designing','designed','developing','developed','testing','tested','verified','rejected','delivering','delivered','released','closed') DEFAULT 'wait',
			assignedTo varchar(30) NOT NULL DEFAULT '',
			openedBy varchar(30) NOT NULL DEFAULT '',
			deliverDate date DEFAULT NULL,
			developFinish date DEFAULT NULL,
			testFinish date DEFAULT NULL,
			verifyFinish date DEFAULT NULL,
			type varchar(30) NOT NULL DEFAULT 'story',
			sourceType varchar(255) NOT NULL DEFAULT '',
			deleted enum('0','1') NOT NULL DEFAULT '0',
			PRIMARY KEY (id)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`,

		`CREATE TABLE IF NOT EXISTS zt_bug (
			id mediumint NOT NULL AUTO_INCREMENT,
			product mediumint unsigned NOT NULL DEFAULT '0',
			story mediumint unsigned NOT NULL DEFAULT '0',
			title varchar(255) NOT NULL DEFAULT '',
			pri tinyint unsigned NOT NULL DEFAULT '0',
			severity tinyint NOT NULL DEFAULT '0',
			status enum('active','resolved','closed') NOT NULL DEFAULT 'active',
			assignedTo varchar(30) NOT NULL DEFAULT '',
			deadline date DEFAULT NULL,
			deleted enum('0','1') NOT NULL DEFAULT '0',
			PRIMARY KEY (id)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`,

		`CREATE TABLE IF NOT EXISTS zt_approvalnode (
			id mediumint NOT NULL AUTO_INCREMENT,
			approval mediumint NOT NULL DEFAULT '0',
			type enum('review','cc') NOT NULL DEFAULT 'review',
			title varchar(255) NOT NULL DEFAULT '',
			account char(30) NOT NULL DEFAULT '',
			status varchar(20) NOT NULL DEFAULT 'wait',
			result varchar(10) NOT NULL DEFAULT '',
			date datetime DEFAULT NULL,
			PRIMARY KEY (id)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`,

		`CREATE TABLE IF NOT EXISTS zt_approvalobject (
			id int NOT NULL AUTO_INCREMENT,
			approval int NOT NULL DEFAULT '0',
			objectType char(30) NOT NULL DEFAULT '',
			objectID mediumint NOT NULL DEFAULT '0',
			extra varchar(255) NOT NULL DEFAULT '',
			PRIMARY KEY (id)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`,

		`CREATE TABLE IF NOT EXISTS zt_charter (
			id mediumint unsigned NOT NULL AUTO_INCREMENT,
			project mediumint unsigned NOT NULL DEFAULT '0',
			deleted enum('0','1') NOT NULL DEFAULT '0',
			PRIMARY KEY (id)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`,

		`CREATE TABLE IF NOT EXISTS zt_project (
			id mediumint unsigned NOT NULL AUTO_INCREMENT,
			name varchar(255) NOT NULL DEFAULT '',
			deleted enum('0','1') NOT NULL DEFAULT '0',
			PRIMARY KEY (id)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`,

		`CREATE TABLE IF NOT EXISTS zt_planchange (
			id mediumint unsigned NOT NULL AUTO_INCREMENT,
			project mediumint unsigned NOT NULL DEFAULT '0',
			title varchar(255) NOT NULL DEFAULT '',
			PRIMARY KEY (id)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`,

		`CREATE TABLE IF NOT EXISTS zt_projectbuildguide (
			id mediumint unsigned NOT NULL AUTO_INCREMENT,
			projectID mediumint unsigned NOT NULL DEFAULT '0',
			name varchar(255) NOT NULL DEFAULT '',
			deleted enum('0','1') NOT NULL DEFAULT '0',
			PRIMARY KEY (id)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`,

		`CREATE TABLE IF NOT EXISTS zt_review (
			id mediumint unsigned NOT NULL AUTO_INCREMENT,
			project mediumint unsigned NOT NULL DEFAULT '0',
			title varchar(255) NOT NULL DEFAULT '',
			deadline date DEFAULT NULL,
			deleted enum('0','1') NOT NULL DEFAULT '0',
			PRIMARY KEY (id)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`,

		`CREATE TABLE IF NOT EXISTS zt_testtask (
			id mediumint unsigned NOT NULL AUTO_INCREMENT,
			project mediumint unsigned NOT NULL DEFAULT '0',
			name varchar(255) NOT NULL DEFAULT '',
			status varchar(30) NOT NULL DEFAULT '',
			pri tinyint unsigned NOT NULL DEFAULT '0',
			end date DEFAULT NULL,
			owner varchar(30) NOT NULL DEFAULT '',
			deleted enum('0','1') NOT NULL DEFAULT '0',
			PRIMARY KEY (id)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`,

		`CREATE TABLE IF NOT EXISTS zt_issue (
			id mediumint unsigned NOT NULL AUTO_INCREMENT,
			project mediumint unsigned NOT NULL DEFAULT '0',
			title varchar(255) NOT NULL DEFAULT '',
			status varchar(30) NOT NULL DEFAULT '',
			pri tinyint unsigned NOT NULL DEFAULT '0',
			deadline date DEFAULT NULL,
			assignedTo varchar(30) NOT NULL DEFAULT '',
			createdBy varchar(30) NOT NULL DEFAULT '',
			deleted enum('0','1') NOT NULL DEFAULT '0',
			PRIMARY KEY (id)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`,

		`CREATE TABLE IF NOT EXISTS zt_risk (
			id mediumint unsigned NOT NULL AUTO_INCREMENT,
			project mediumint unsigned NOT NULL DEFAULT '0',
			name varchar(255) NOT NULL DEFAULT '',
			status varchar(30) NOT NULL DEFAULT '',
			pri tinyint unsigned NOT NULL DEFAULT '0',
			plannedClosedDate date DEFAULT NULL,
			assignedTo varchar(30) NOT NULL DEFAULT '',
			createdBy varchar(30) NOT NULL DEFAULT '',
			deleted enum('0','1') NOT NULL DEFAULT '0',
			PRIMARY KEY (id)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`,

		`CREATE TABLE IF NOT EXISTS zt_todo (
			id mediumint unsigned NOT NULL AUTO_INCREMENT,
			account char(30) NOT NULL DEFAULT '',
			name varchar(255) NOT NULL DEFAULT '',
			` + "`type`" + ` varchar(20) NOT NULL DEFAULT '',
			status varchar(20) NOT NULL DEFAULT '',
			` + "`date`" + ` date DEFAULT NULL,
			begin smallint unsigned NOT NULL DEFAULT '0',
			` + "`end`" + ` smallint unsigned NOT NULL DEFAULT '0',
			pri tinyint unsigned NOT NULL DEFAULT '0',
			assignedTo varchar(30) NOT NULL DEFAULT '',
			deleted enum('0','1') NOT NULL DEFAULT '0',
			PRIMARY KEY (id)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`,

		`CREATE TABLE IF NOT EXISTS zt_versionwindow (
			id bigint unsigned NOT NULL AUTO_INCREMENT,
			name varchar(100) NOT NULL,
			releaseDate date NOT NULL,
			startDate date DEFAULT NULL,
			teamgroup mediumint unsigned NOT NULL DEFAULT '0',
			groupSize int unsigned NOT NULL DEFAULT '1',
			createdBy varchar(30) NOT NULL DEFAULT '',
			updatedBy varchar(30) NOT NULL DEFAULT '',
			status varchar(20) NOT NULL DEFAULT 'planning',
			` + "`order`" + ` int NOT NULL DEFAULT '0',
			createdDate datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updatedDate datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
			deletedAt datetime(3) DEFAULT NULL,
			PRIMARY KEY (id)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`,

		`CREATE TABLE IF NOT EXISTS zt_versionwindowproduct (
			id bigint unsigned NOT NULL AUTO_INCREMENT,
			versionWindow bigint unsigned NOT NULL,
			product mediumint unsigned NOT NULL,
			plan mediumint unsigned DEFAULT NULL,
			planSynced tinyint(1) NOT NULL DEFAULT '0',
			createdBy varchar(30) NOT NULL DEFAULT '',
			updatedBy varchar(30) NOT NULL DEFAULT '',
			createdDate datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updatedDate datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
			deletedAt datetime(3) DEFAULT NULL,
			PRIMARY KEY (id),
			UNIQUE KEY uk_versionWindow_product (versionWindow, product)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`,

		`CREATE TABLE IF NOT EXISTS zt_demandwindow (
			id bigint unsigned NOT NULL AUTO_INCREMENT,
			demand mediumint unsigned NOT NULL,
			story mediumint unsigned NOT NULL DEFAULT '0',
			versionWindow bigint unsigned NOT NULL,
			createdBy varchar(30) NOT NULL DEFAULT '',
			updatedBy varchar(30) NOT NULL DEFAULT '',
			createdDate datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updatedDate datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
			deletedAt datetime(3) DEFAULT NULL,
			PRIMARY KEY (id),
			UNIQUE KEY uk_demand_story (demand, story)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`,

		`CREATE TABLE IF NOT EXISTS zt_planstory (
			PKID int unsigned NOT NULL AUTO_INCREMENT,
			plan mediumint unsigned NOT NULL,
			story mediumint unsigned NOT NULL,
			` + "`order`" + ` mediumint NOT NULL DEFAULT '0',
			PRIMARY KEY (PKID),
			UNIQUE KEY plan_story (plan, story)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`,

		`CREATE TABLE IF NOT EXISTS zt_dept (
			id mediumint unsigned NOT NULL AUTO_INCREMENT,
			name char(60) NOT NULL DEFAULT '',
			parent mediumint unsigned NOT NULL DEFAULT '0',
			path char(255) NOT NULL DEFAULT '',
			grade tinyint unsigned NOT NULL DEFAULT '0',
			` + "`order`" + ` smallint unsigned NOT NULL DEFAULT '0',
			PRIMARY KEY (id)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`,

		`CREATE TABLE IF NOT EXISTS zt_product (
			id mediumint unsigned NOT NULL AUTO_INCREMENT,
			name varchar(255) NOT NULL DEFAULT '',
			code varchar(255) NOT NULL DEFAULT '',
			status varchar(30) NOT NULL DEFAULT '',
			PO varchar(30) NOT NULL DEFAULT '',
			QD varchar(30) NOT NULL DEFAULT '',
			RD varchar(30) NOT NULL DEFAULT '',
			createdBy varchar(30) NOT NULL DEFAULT '',
			whitelist text,
			deleted enum('0','1') NOT NULL DEFAULT '0',
			PRIMARY KEY (id)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`,

		`CREATE TABLE IF NOT EXISTS zt_productplan (
			id mediumint unsigned NOT NULL AUTO_INCREMENT,
			product mediumint unsigned NOT NULL DEFAULT '0',
			title varchar(255) NOT NULL DEFAULT '',
			begin date DEFAULT NULL,
			` + "`end`" + ` date DEFAULT NULL,
			status varchar(30) NOT NULL DEFAULT '',
			` + "`order`" + ` varchar(30) NOT NULL DEFAULT '',
			closedReason varchar(255) NOT NULL DEFAULT '',
			createdBy varchar(30) NOT NULL DEFAULT '',
			createdDate datetime DEFAULT NULL,
			deleted enum('0','1') NOT NULL DEFAULT '0',
			PRIMARY KEY (id)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`,

		`CREATE TABLE IF NOT EXISTS zt_teamgroup (
			id mediumint unsigned NOT NULL AUTO_INCREMENT,
			name varchar(255) NOT NULL DEFAULT '',
			parent mediumint unsigned NOT NULL DEFAULT '0',
			path varchar(255) NOT NULL DEFAULT '',
			deleted enum('0','1') NOT NULL DEFAULT '0',
			PRIMARY KEY (id)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`,

		`CREATE TABLE IF NOT EXISTS zt_holiday (
			id mediumint unsigned NOT NULL AUTO_INCREMENT,
			name varchar(30) NOT NULL DEFAULT '',
			type enum('holiday','working') NOT NULL DEFAULT 'holiday',
			` + "`desc`" + ` text,
			` + "`year`" + ` char(4) NOT NULL DEFAULT '',
			` + "`begin`" + ` date DEFAULT NULL,
			` + "`end`" + ` date DEFAULT NULL,
			PRIMARY KEY (id)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`,

		`CREATE TABLE IF NOT EXISTS zt_config (
			id mediumint unsigned NOT NULL AUTO_INCREMENT,
			owner varchar(30) NOT NULL DEFAULT '',
			module varchar(30) NOT NULL DEFAULT '',
			section varchar(30) NOT NULL DEFAULT '',
			` + "`key`" + ` varchar(30) NOT NULL DEFAULT '',
			value text,
			PRIMARY KEY (id)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`,
	}

	for _, ddl := range ddls {
		if err := db.WithContext(ctx).Exec(ddl).Error; err != nil {
			return fmt.Errorf("failed to exec ddl: %w", err)
		}
	}
	return nil
}
