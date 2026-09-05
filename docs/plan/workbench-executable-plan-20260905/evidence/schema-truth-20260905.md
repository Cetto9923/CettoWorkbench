# D1 / V0 Database Schema Truth & Audit Evidence

- **采集时间**: 2026-09-05 13:05 CST
- **环境来源**: 本地 MySQL 实例（`MacBook-Air-809.local:3306`）
- **数据库**: `zentaopms` (MySQL 8.4.0, utf8mb4 / utf8mb4_0900_ai_ci, sql_mode: NO_AUTO_VALUE_ON_ZERO, time_zone: SYSTEM)
- **凭据处理**: 本地免密/只读审计，不记录任何密码、token 或敏感 DSN。

---

## 一、RBAC / Schedule Grant 核心事实回答

1. **`zt_roles` 真实软删除字段**:
   - `deleted tinyint(1) NOT NULL DEFAULT '0'`
   - 同时具备 `isActive tinyint(1) NOT NULL DEFAULT '1'`
   - 唯一键：`uk_zt_roles_code_deleted (code, deleted)`, `uk_zt_roles_name_deleted (name, deleted)`

2. **`zt_gf_user_roles` 真实软删除字段**:
   - `deleted tinyint(1) NOT NULL DEFAULT '0'`
   - 唯一键：`uk_zt_userroles_user_role_deleted (tenantId, userId, roleId, deleted)`

3. **`zt_role_permissions` 真实软删除字段**:
   - `deletedAt datetime DEFAULT NULL`
   - 唯一键：`uk_role_perm_deleted (roleId, permCode, deletedAt)`
   - **注意**：同一套 RBAC 内部三张表软删除语义不一致（`roles` 与 `user_roles` 为 `deleted tinyint`，`role_permissions` 为 `deletedAt datetime`）。

4. **`tenantId` 是否真实存在**:
   - `zt_gf_user_roles`: **存在** (`tenantId bigint NOT NULL DEFAULT '0'`)
   - `zt_roles`: **不存在**
   - `zt_role_permissions`: **不存在**
   - `zt_operation_logs`: **存在** (`tenantId bigint NOT NULL DEFAULT '0'`)
   - `zt_dept` (禅道真实表): **不存在**

5. **当前有效角色拥有的 schedule capability grant**:
   - 数据库中 `zt_roles` 为 **0 行**
   - `zt_role_permissions` 为 **0 行**
   - `zt_gf_user_roles` 为 **0 行**
   - **结论**：没有任何角色拥有任何 schedule capability 授权。

6. **当前数据库中是否存在指定权限码**:
   - `po:schedule`: **存在于 `zt_menus`**（`id=3, title='排期工作台', path='/schedule', perm='po:schedule'`）
   - `schedule:list`: **不存在**（在 `zt_menus`, `zt_role_permissions`, `zt_grouppriv` 中均为 0）
   - `schedule:create`: **不存在**
   - `schedule:update`: **不存在**
   - `schedule:delete`: **不存在**

7. **`/schedule` 菜单当前实际 perm**:
   - 实际字段值为 `'po:schedule'`

8. **deleted / inactive / revoked grant 数据库语义与加载器行为**:
   - 数据库语义：`zt_roles.deleted = 1` 为已删，`isActive = 0` 为停用；`zt_gf_user_roles.deleted = 1` 为已解除关系；`zt_role_permissions.deletedAt IS NOT NULL` 为已撤销权限。
   - 代码加载器现状（`internal/pkg/perm/loader.go`）：
     - 查询语句仅包含 `Where("r.isActive = ?", true)`
     - **未过滤 `r.deleted = 0`**
     - **未过滤 `ur.deleted = 0`**
     - **未过滤 `rp.deletedAt IS NULL`**
     - 此为确认缺陷（E09：权限查询未过滤删除边）。

---

## 二、四层 Truth Matrix (Live DB vs install.sql vs Model vs Query)

| 表名 | Live 真实字段/主键/软删除 | db/install.sql | GORM 模型定义 | 查询/写入层现状 | 分类结论 |
|---|---|---|---|---|---|
| `zt_roles` | `deleted tinyint(1)`, 无 `tenantId`, 无 `deletedAt` | 结构一致 | `BaseModel` 定义了 `deletedAt gorm.DeletedAt` (column: `deletedAt`) | `loader.go` 未查 `r.deleted = 0` | **LIVE CODE CONFLICT** (模型与加载器均与真实字段冲突) |
| `zt_gf_user_roles` | `deleted tinyint(1)`, 含 `tenantId` | 结构一致 | `BaseModel` 定义了 `deletedAt` | `loader.go` 未查 `ur.deleted = 0`，未带 `tenantId` | **LIVE CODE CONFLICT** (模型期望 `deletedAt`，实际为 `deleted`) |
| `zt_role_permissions` | `deletedAt datetime DEFAULT NULL`, 无 `tenantId` | 结构一致 | `BaseModel` 匹配 `deletedAt` | `loader.go` 未查 `rp.deletedAt IS NULL` | **LIVE CODE CONFLICT** (加载器漏查软删边) |
| `zt_menus` | `deletedAt datetime(3)`, `/schedule` 行 `perm='po:schedule'` | 结构与 seed 一致 | 模型匹配 | 常量定义为 `schedule:list`，与菜单 seed 不符 | **REPOSITORY CONTRACT CONFLICT** |
| `zt_versionwindow` | `id uint64`, `deletedAt datetime(3)` | 结构一致 | `VersionWindow` 字段完全匹配 | `repo_window.go` 正常 | **MATCH** |
| `zt_versionwindowproduct`| `id uint64`, `deletedAt datetime(3)` | 结构一致 | `VersionWindowProduct` 完全匹配 | `repo_window.go` 正常 | **MATCH** |
| `zt_demandwindow` | `demand uint`, `story uint`, `versionWindow uint64`, `deletedAt datetime(3)` | 结构一致 | `DemandWindow` 完全匹配 | `repodemandwindow.go` 正常 | **MATCH** |
| `zt_dept` (vs `zt_depts`) | 表名为 `zt_dept`；字段: `id, name, parent, path, grade, order, position, function, manager, strucNo, superStrucNo`。**无 `deletedAt`，无 `status`，无 `tenantId`，无 `parentId`** | 未定义 | 模型为 `Dept`，表名为 `zt_depts`，含 `parentId, leader, phone, email, status, sort, deletedAt` | `dept/repo.go` 查询 `FROM zt_depts WHERE deletedAt IS NULL`，在真实库必抛表不存在错误 | **CRITICAL LIVE CODE CONFLICT** (表名与全部业务列均不存在) |
| `zt_demand` | 禅道表：`deleted enum('0','1')`, `product varchar(255)`, `parent mediumint` | 未定义 | `zentao/demand.go` 匹配 | 业务 SQL 显式用 `deleted = '0'` 过滤 | **MATCH / STALE REPOSITORY DDL** |
| `zt_story` | 禅道表：`deleted enum('0','1')`, `product mediumint`, `fromDemand mediumint` | 未定义 | `zentao/story.go` 匹配 | 业务 SQL 显式用 `deleted = '0'` 过滤 | **MATCH / STALE REPOSITORY DDL** |
| `zt_task` | 禅道表：`deleted enum('0','1')`, `story mediumint`, `parent mediumint` | 未定义 | 写入 struct 匹配 | 业务 SQL 显式用 `deleted = '0'` 过滤 | **MATCH / STALE REPOSITORY DDL** |
| `zt_project` | 禅道表：`deleted enum('0','1')` | 未定义 | 读投影匹配 | 业务 SQL 匹配 | **MATCH / STALE REPOSITORY DDL** |
| `zt_projectproduct` | 禅道关系表：PK (`project, product, branch`)，无软删除 | 未定义 | 投影匹配 | 业务 SQL 匹配 | **MATCH / STALE REPOSITORY DDL** |
| `zt_projectstory` | 禅道关系表：PK (`PKID`), UK (`project, story`) | 未定义 | 投影匹配 | 业务 SQL 匹配 | **MATCH / STALE REPOSITORY DDL** |
| `zt_planstory` | 禅道关系表：PK (`PKID`), UK (`plan, story`) | 未定义 | 投影匹配 | 业务 SQL 匹配 | **MATCH / STALE REPOSITORY DDL** |
| `zt_product` | 禅道表：`deleted enum('0','1')` | 未定义 | 投影匹配 | 业务 SQL 匹配 | **MATCH / STALE REPOSITORY DDL** |
| `zt_productplan` | 禅道表：`deleted enum('0','1')` | 未定义 | 投影匹配 | 业务 SQL 匹配 | **MATCH / STALE REPOSITORY DDL** |
| `zt_action` | 禅道审计表：`read enum('0','1')`，无 `deleted` | 未定义 | 投影匹配 | 业务 SQL 匹配 | **MATCH / STALE REPOSITORY DDL** |
| `zt_user` | 禅道表：`deleted enum('0','1')`, `locked datetime`, `account char(30)` | 未定义 | `model.User` 匹配 | `AfterFind` 计算 `IsActive` 匹配 | **MATCH / STALE REPOSITORY DDL** |
| `zt_operation_logs` | 工作台表：`id, tenantId, userId, account, method, path, query, body, ip, userAgent, statusCode, createdAt` | 未定义 | `model.OperationLog` 匹配 | 业务写入匹配 | **MATCH / STALE REPOSITORY DDL** |
| `zt_notify` | 禅道通知表：`status varchar(10)` | 未定义 | 投影匹配 | `reponotice.go` 匹配 | **MATCH / STALE REPOSITORY DDL** |
| `zt_workbench_notify_reads` | 工作台表：`id, notify, account, readAt`, UK (`notify, account`) | 未定义 | 投影匹配 | `reponotice.go` 匹配 | **MATCH / STALE REPOSITORY DDL** |
| `zt_demandclarify` | 禅道澄清表：`demand mediumint, PM longtext` | 未定义 | 投影匹配 | `repo_scheduling.go` 匹配 | **MATCH / STALE REPOSITORY DDL** |

---

## 三、真实 DDL 结构存档

### 1. `zt_roles`
```sql
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
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
```

### 2. `zt_gf_user_roles`
```sql
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
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
```

### 3. `zt_role_permissions`
```sql
CREATE TABLE `zt_role_permissions` (
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
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
```

### 4. `zt_menus`
```sql
CREATE TABLE `zt_menus` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `parentId` bigint unsigned NOT NULL DEFAULT '0',
  `title` varchar(64) NOT NULL DEFAULT '',
  `icon` varchar(64) NOT NULL DEFAULT '',
  `path` varchar(255) NOT NULL DEFAULT '',
  `perm` varchar(64) NOT NULL DEFAULT '',
  `type` char(1) NOT NULL DEFAULT 'C',
  `sort` int NOT NULL DEFAULT '0',
  `createdAt` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `updatedAt` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  `deletedAt` datetime(3) DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_zt_menus_parent_title_perm` (`parentId`,`title`,`perm`),
  KEY `idx_parentId` (`parentId`),
  KEY `idx_zt_menus_perm` (`perm`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
```

### 5. `zt_versionwindow`
```sql
CREATE TABLE `zt_versionwindow` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `name` varchar(100) NOT NULL COMMENT '窗口名称',
  `releaseDate` date NOT NULL COMMENT '预计上线日期',
  `startDate` date DEFAULT NULL COMMENT '窗口开始日期',
  `teamgroup` mediumint unsigned NOT NULL COMMENT '关联敏捷小组',
  `groupSize` int unsigned NOT NULL DEFAULT '1' COMMENT '小组人数',
  `createdBy` varchar(30) NOT NULL COMMENT '创建人账号',
  `updatedBy` varchar(30) NOT NULL DEFAULT '' COMMENT '最后更新人账号',
  `status` varchar(20) NOT NULL DEFAULT 'planning' COMMENT 'current/next/planning/released',
  `order` int NOT NULL DEFAULT '0',
  `createdDate` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updatedDate` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  `deletedAt` datetime(3) DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `idx_teamgroup` (`teamgroup`),
  KEY `idx_releaseDate` (`releaseDate`),
  KEY `idx_createdBy` (`createdBy`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
```

### 6. `zt_versionwindowproduct`
```sql
CREATE TABLE `zt_versionwindowproduct` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `versionWindow` bigint unsigned NOT NULL COMMENT '版本窗口ID',
  `product` mediumint unsigned NOT NULL COMMENT '关联产品ID',
  `plan` mediumint unsigned DEFAULT NULL COMMENT '关联禅道计划ID',
  `planSynced` tinyint(1) NOT NULL DEFAULT '0' COMMENT '计划是否已同步',
  `createdBy` varchar(30) NOT NULL DEFAULT '' COMMENT '创建人账号',
  `updatedBy` varchar(30) NOT NULL DEFAULT '' COMMENT '最后更新人账号',
  `createdDate` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updatedDate` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  `deletedAt` datetime(3) DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_versionWindow_product` (`versionWindow`,`product`),
  KEY `idx_versionWindow` (`versionWindow`),
  KEY `idx_product` (`product`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
```

### 7. `zt_demandwindow`
```sql
CREATE TABLE `zt_demandwindow` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `demand` mediumint unsigned NOT NULL COMMENT '业务需求ID',
  `story` mediumint unsigned NOT NULL DEFAULT '0' COMMENT '研发需求ID，0=业需级',
  `versionWindow` bigint unsigned NOT NULL COMMENT '版本窗口ID',
  `createdBy` varchar(30) NOT NULL DEFAULT '',
  `updatedBy` varchar(30) NOT NULL DEFAULT '',
  `createdDate` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updatedDate` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  `deletedAt` datetime(3) DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_demand_story` (`demand`,`story`),
  KEY `idx_demand` (`demand`),
  KEY `idx_story` (`story`),
  KEY `idx_versionWindow` (`versionWindow`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
```

### 8. `zt_dept` (真实禅道部门表，非 `zt_depts`)
```sql
CREATE TABLE `zt_dept` (
  `id` mediumint unsigned NOT NULL AUTO_INCREMENT,
  `name` char(60) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `parent` mediumint unsigned NOT NULL DEFAULT '0',
  `path` char(255) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `grade` tinyint unsigned NOT NULL DEFAULT '0',
  `order` smallint unsigned NOT NULL DEFAULT '0',
  `position` char(30) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `function` char(255) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `manager` char(30) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `strucNo` varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL,
  `superStrucNo` varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL,
  PRIMARY KEY (`id`),
  KEY `parent` (`parent`),
  KEY `path` (`path`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
```
