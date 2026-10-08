-- 侧栏菜单配置化：为 zt_menus 增加「规划中占位 / 窄轨短名 / 额外高亮地址」三列，
-- 并按现网侧栏结构写入种子数据。需单独审批后执行，只操作 zt_menus，不修改禅道原生表。
-- 兼容 OceanBase / MySQL 8.0；可重复执行。

-- 1. 幂等加列：先查 information_schema，已存在则跳过。
SET @db = DATABASE();

SET @sql = (SELECT IF(COUNT(*) = 0,
  'ALTER TABLE `zt_menus` ADD COLUMN `planned` TINYINT NOT NULL DEFAULT 0 COMMENT ''1=规划中占位菜单，渲染为不可点占位''',
  'DO 0')
  FROM information_schema.COLUMNS
  WHERE TABLE_SCHEMA = @db AND TABLE_NAME = 'zt_menus' AND COLUMN_NAME = 'planned');
PREPARE stmt FROM @sql; EXECUTE stmt; DEALLOCATE PREPARE stmt;

SET @sql = (SELECT IF(COUNT(*) = 0,
  'ALTER TABLE `zt_menus` ADD COLUMN `shortTitle` VARCHAR(16) NOT NULL DEFAULT '''' COMMENT ''一级菜单窄轨短名，为空时用 title''',
  'DO 0')
  FROM information_schema.COLUMNS
  WHERE TABLE_SCHEMA = @db AND TABLE_NAME = 'zt_menus' AND COLUMN_NAME = 'shortTitle');
PREPARE stmt FROM @sql; EXECUTE stmt; DEALLOCATE PREPARE stmt;

SET @sql = (SELECT IF(COUNT(*) = 0,
  'ALTER TABLE `zt_menus` ADD COLUMN `activePaths` VARCHAR(255) NOT NULL DEFAULT '''' COMMENT ''额外高亮地址前缀，逗号分隔''',
  'DO 0')
  FROM information_schema.COLUMNS
  WHERE TABLE_SCHEMA = @db AND TABLE_NAME = 'zt_menus' AND COLUMN_NAME = 'activePaths');
PREPARE stmt FROM @sql; EXECUTE stmt; DEALLOCATE PREPARE stmt;

-- 2. 旧顶栏种子（个人入口 / 工作区 / 首页 / 需求排期 / 工作看板）软删除。
--    新树以「我的工作台」等六个一级分组重建，这些行不再参与导航。
UPDATE `zt_menus` SET `deletedAt` = NOW(3)
WHERE `id` IN (1, 2, 3, 4, 5) AND `deletedAt` IS NULL;

-- 3. 侧栏种子：一级分组 type='M'（parentId=0），二级 type='C'。
--    planned=1 为规划中占位（无 path，渲染为灰色不可点）。
--    activePaths 为除主 path 之外、同样要高亮该菜单的地址前缀。
INSERT INTO `zt_menus`
  (`id`, `parentId`, `title`, `shortTitle`, `icon`, `path`, `perm`, `type`, `sort`, `planned`, `activePaths`)
VALUES
  -- 一级：我的工作台
  (100, 0, '我的工作台', '工作台', 'fa-desktop', '', '', 'M', 10, 0, ''),
  (101, 100, '首页',       '', 'fa-house',        '/home',    '',        'C', 1, 0, ''),
  (102, 100, '我的待办',   '', 'fa-list-check',   '/todos',   '',        'C', 2, 0, ''),
  (103, 100, '我的已办',   '', 'fa-clock-rotate-left', '/done', '',       'C', 3, 0, ''),
  (104, 100, '通知中心',   '', 'fa-bell',         '/notice',  '',        'C', 4, 0, ''),
  (105, 100, '我的关注',   '', 'fa-star',         '/follow',  '',        'C', 5, 0, ''),
  -- 一级：需求规划
  (110, 0, '需求规划', '规划', 'fa-calendar-check', '', '', 'M', 20, 0, ''),
  (111, 110, '需求排期', '', 'fa-calendar-check', '/schedule', '', 'C', 1, 0, ''),
  (112, 110, '需求查询', '', 'fa-magnifying-glass', '/query',  '', 'C', 2, 0, '/demands/'),
  (113, 110, '版本跟进', '', 'fa-boxes-packing', '/version-follow', 'po:schedule', 'C', 3, 0, ''),
  -- 一级：团队协作
  (120, 0, '团队协作', '协作', 'fa-users', '', '', 'M', 30, 0, ''),
  (121, 120, '工作看板', '', 'fa-table-cells-large', '/board/demand', '', 'C', 1, 0, '/board/task'),
  (122, 120, '敏捷小组', '', 'fa-users', '/agileteam', '', 'C', 2, 0, '/pmo'),
  (123, 120, '迭代管理', '', 'fa-arrows-spin', '', '', 'C', 3, 1, ''),
  (124, 120, '团队健康', '', 'fa-heart-pulse', '', '', 'C', 4, 1, ''),
  -- 一级：项目管理（仅超管可见，对齐现网 IsSuperAdmin 判断）
  (130, 0, '项目管理', '项目', 'fa-diagram-project', '', 'project:list', 'M', 40, 0, ''),
  (131, 130, '项目总览', '', 'fa-folder-tree', '', '', 'C', 1, 1, ''),
  (132, 130, '计划里程碑', '', 'fa-flag-checkered', '', '', 'C', 2, 1, ''),
  -- 一级：治理分析
  (140, 0, '治理分析', '治理', 'fa-chart-pie', '', '', 'M', 50, 0, ''),
  (141, 140, '问题风险', '', 'fa-shield-alt', '/issues/risk', '', 'C', 1, 0, '/issue-risk'),
  (142, 140, '指标雷达', '', 'fa-chart-line', '/metrics/radar', '', 'C', 2, 0, ''),
  (143, 140, '指标管理', '', 'fa-sliders', '/metrics/manage', '', 'C', 3, 0, ''),
  (144, 140, '质效预警', '', 'fa-triangle-exclamation', '', '', 'C', 4, 1, ''),
  -- 一级：组织管理（按现有 adminLinks 的权限判断）
  (150, 0, '组织管理', '管理', 'fa-user-shield', '', '', 'M', 60, 0, ''),
  (151, 150, '角色管理', '', 'fa-user-shield', '/admin/roles', 'role:list', 'C', 1, 0, ''),
  (152, 150, '菜单管理', '', 'fa-bars', '/admin/menus', 'menu:list', 'C', 2, 0, ''),
  (153, 150, '操作日志', '', 'fa-clock-rotate-left', '/admin/operation-logs', 'operationlog:list', 'C', 3, 0, ''),
  (154, 150, '登录日志', '', 'fa-right-to-bracket', '/admin/login-logs', 'loginlog:list', 'C', 4, 0, '')
ON DUPLICATE KEY UPDATE
  `parentId`    = VALUES(`parentId`),
  `title`       = VALUES(`title`),
  `shortTitle`  = VALUES(`shortTitle`),
  `icon`        = VALUES(`icon`),
  `path`        = VALUES(`path`),
  `perm`        = VALUES(`perm`),
  `type`        = VALUES(`type`),
  `sort`        = VALUES(`sort`),
  `planned`     = VALUES(`planned`),
  `activePaths` = VALUES(`activePaths`),
  `deletedAt`   = NULL;
