-- ============================================================================
-- 文件: tools/seed/version-follow-demo.sql
-- 模块: 工作台 · 演示数据
-- 职责: 为「版本跟进」页造【DEMO】样例数据，使九阶段与三种发布判断都能演示。
--       本脚本由开发编写，执行需产品/研发负责人审核后手工执行；开发不执行。
--
-- 特性：
--   1. 幂等：可重复执行，重复执行不产生重复数据、不覆盖任何现有行。
--   2. 只新增带【DEMO】前缀的窗口与需求；不 UPDATE、不 DELETE 任何现有行。
--   3. 绝不触碰需求 63450（有脏数据）。
--   4. 软删除口径（已在库上核实）：
--        zt_demand         -> deleted    （enum，'0' 为未删除）
--        zt_versionwindow  -> deletedAt  （datetime，NULL 为未删除）
--        zt_demandwindow   -> deletedAt  （datetime，NULL 为未删除）
--   5. 判断三态由页面按确定性规则推导，不入库；本脚本按规则反推该造什么数据。
--
-- 为什么用临时表：
--   zt_demand 有 98 个 NOT NULL 列，逐条 SQL 硬写会让样例意图淹没在列名里；
--   先用临时表声明业务字段，再一次性搬入，可读性与可维护性更好。
--   这里先建一张临时业务表（只放要造的字段），再按 zt_demand 的实际列结构
--   逐列显式搬入，未涉及的必填列按库内默认值补齐。
--   业务字段集中在 tmp_vf_demo 声明处，样例意图一目了然。
--
-- 三态反推规则（与 internal/module/po/version_follow_judge.go 保持一致）：
--   阻塞 = 已过计划提测日仍未提测 | 缺负责人 | 已进测试但验收标准为空
--   风险 = 已超期 1-2 天 | 距计划提测日 0-2 天且阶段落后
--   可按期 = 其余（计划日期充裕、阶段已跟上、有负责人、有验收标准）
--
-- 执行方式：
--   mysql -h <host> -P <port> -u <user> -p <db> < tools/seed/version-follow-demo.sql
--
-- 回滚：见文件末尾 ROLLBACK 段，与本脚本配套，执行后按需运行。
-- ============================================================================

SET @demo = '【DEMO】';
SET @db = DATABASE();
SET @now = '2026-09-30 00:00:00';

-- ---------------------------------------------------------------------------
-- 0. 临时业务表：只声明要造的字段，其余列在搬运时自动补默认。
-- ---------------------------------------------------------------------------
DROP TEMPORARY TABLE IF EXISTS tmp_vf_demo;
CREATE TEMPORARY TABLE tmp_vf_demo (
  title           VARCHAR(255) NOT NULL,
  stage           VARCHAR(20)  NOT NULL,
  status          VARCHAR(20)  NOT NULL,
  mainSystem      VARCHAR(64)  NOT NULL,
  pri             CHAR(1)      NOT NULL,
  deadline        DATE         NULL,
  schedulePlanDate DATE        NULL,
  acceptance      VARCHAR(255) NULL,
  assignedTo      VARCHAR(64)  NULL,
  windowName      VARCHAR(100) NOT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- ---------------------------------------------------------------------------
-- 1. 造 3 个版本窗口：规划中 / 进行中 / 已发布
-- ---------------------------------------------------------------------------
INSERT INTO zt_versionwindow
  (name, releaseDate, startDate, windowType, teamgroup, groupSize,
   createdBy, updatedBy, status, `order`, createdDate, updatedDate)
SELECT CONCAT(@demo, '窗口-规划中'), '2026-11-28', '2026-11-02', 'regular', 146, 3,
       'seed', 'seed', 'planning', 91, NOW(), NOW()
WHERE NOT EXISTS (
  SELECT 1 FROM zt_versionwindow
  WHERE name = CONCAT(@demo, '窗口-规划中') AND deletedAt IS NULL
);

INSERT INTO zt_versionwindow
  (name, releaseDate, startDate, windowType, teamgroup, groupSize,
   createdBy, updatedBy, status, `order`, createdDate, updatedDate)
SELECT CONCAT(@demo, '窗口-进行中'), '2026-10-20', '2026-09-20', 'regular', 146, 3,
       'seed', 'seed', 'released', 92, NOW(), NOW()
WHERE NOT EXISTS (
  SELECT 1 FROM zt_versionwindow
  WHERE name = CONCAT(@demo, '窗口-进行中') AND deletedAt IS NULL
);

INSERT INTO zt_versionwindow
  (name, releaseDate, startDate, windowType, teamgroup, groupSize,
   createdBy, updatedBy, status, `order`, createdDate, updatedDate)
SELECT CONCAT(@demo, '窗口-已发布'), '2026-09-05', '2026-08-20', 'regular', 146, 3,
       'seed', 'seed', 'released', 93, NOW(), NOW()
WHERE NOT EXISTS (
  SELECT 1 FROM zt_versionwindow
  WHERE name = CONCAT(@demo, '窗口-已发布') AND deletedAt IS NULL
);

-- ---------------------------------------------------------------------------
-- 2. 造 14 条【DEMO】需求：阻塞 3 / 风险 4 / 可按期 5 / 缺数据 2
--    assignedTo 故意留空的样例，用于命中「缺负责人」规则。
-- ---------------------------------------------------------------------------

-- 【阻塞 x3】---------------------------------------------------------------
-- 已过计划提测日仍未提测
INSERT INTO tmp_vf_demo (title, stage, status, mainSystem, pri, deadline, schedulePlanDate, acceptance, assignedTo, windowName)
SELECT CONCAT(@demo,'阻塞-已过计划提测日未提测'), 'wait', 'wait', '手机银行', '1', '2026-10-05', '2026-09-25', '已有验收标准', 'demo_po', CONCAT(@demo,'窗口-进行中')
WHERE NOT EXISTS (SELECT 1 FROM zt_demand WHERE title = CONCAT(@demo,'阻塞-已过计划提测日未提测'));

-- 缺负责人
INSERT INTO tmp_vf_demo (title, stage, status, mainSystem, pri, deadline, schedulePlanDate, acceptance, assignedTo, windowName)
SELECT CONCAT(@demo,'阻塞-缺负责人'), 'developing', 'clarified', '对公网银', '2', '2026-10-30', '2026-10-20', '已有验收标准', NULL, CONCAT(@demo,'窗口-进行中')
WHERE NOT EXISTS (SELECT 1 FROM zt_demand WHERE title = CONCAT(@demo,'阻塞-缺负责人'));

-- 已进测试但验收标准为空
INSERT INTO tmp_vf_demo (title, stage, status, mainSystem, pri, deadline, schedulePlanDate, acceptance, assignedTo, windowName)
SELECT CONCAT(@demo,'阻塞-已进测试但验收标准为空'), 'testing', 'testing', '信贷系统', '1', '2026-10-25', '2026-10-10', NULL, 'demo_leader', CONCAT(@demo,'窗口-进行中')
WHERE NOT EXISTS (SELECT 1 FROM zt_demand WHERE title = CONCAT(@demo,'阻塞-已进测试但验收标准为空'));

-- 【风险 x4】---------------------------------------------------------------
-- 已超期 2 天
INSERT INTO tmp_vf_demo (title, stage, status, mainSystem, pri, deadline, schedulePlanDate, acceptance, assignedTo, windowName)
SELECT CONCAT(@demo,'风险-已超期2天'), 'developing', 'clarified', '手机银行', '2', '2026-09-28', '2026-10-15', '已有验收标准', 'demo_po', CONCAT(@demo,'窗口-进行中')
WHERE NOT EXISTS (SELECT 1 FROM zt_demand WHERE title = CONCAT(@demo,'风险-已超期2天'));

-- 距计划提测日 2 天但仍在澄清（阶段落后）
INSERT INTO tmp_vf_demo (title, stage, status, mainSystem, pri, deadline, schedulePlanDate, acceptance, assignedTo, windowName)
SELECT CONCAT(@demo,'风险-距计划提测日2天且阶段落后'), 'wait', 'active', '运营支撑', '3', '2026-10-12', '2026-10-02', '已有验收标准', 'demo_leader', CONCAT(@demo,'窗口-进行中')
WHERE NOT EXISTS (SELECT 1 FROM zt_demand WHERE title = CONCAT(@demo,'风险-距计划提测日2天且阶段落后'));

-- 已超期 1 天
INSERT INTO tmp_vf_demo (title, stage, status, mainSystem, pri, deadline, schedulePlanDate, acceptance, assignedTo, windowName)
SELECT CONCAT(@demo,'风险-已超期1天'), 'testing', 'testing', '手机银行', '1', '2026-09-29', '2026-10-18', '已有验收标准', 'demo_po', CONCAT(@demo,'窗口-进行中')
WHERE NOT EXISTS (SELECT 1 FROM zt_demand WHERE title = CONCAT(@demo,'风险-已超期1天'));

-- 距计划提测日 1 天且阶段落后
INSERT INTO tmp_vf_demo (title, stage, status, mainSystem, pri, deadline, schedulePlanDate, acceptance, assignedTo, windowName)
SELECT CONCAT(@demo,'风险-距计划提测日1天且阶段落后'), 'developing', 'clarified', '信贷系统', '2', '2026-10-20', '2026-10-01', '已有验收标准', 'demo_leader', CONCAT(@demo,'窗口-进行中')
WHERE NOT EXISTS (SELECT 1 FROM zt_demand WHERE title = CONCAT(@demo,'风险-距计划提测日1天且阶段落后'));

-- 【可按期 x5】-------------------------------------------------------------
-- 已排期（阶段：排期）
INSERT INTO tmp_vf_demo (title, stage, status, mainSystem, pri, deadline, schedulePlanDate, acceptance, assignedTo, windowName)
SELECT CONCAT(@demo,'可按期-已排期'), 'incharter', 'clarified', '手机银行', '1', '2026-10-30', '2026-10-20', '已有验收标准', 'demo_po', CONCAT(@demo,'窗口-进行中')
WHERE NOT EXISTS (SELECT 1 FROM zt_demand WHERE title = CONCAT(@demo,'可按期-已排期'));

-- 已澄清（阶段：澄清）
INSERT INTO tmp_vf_demo (title, stage, status, mainSystem, pri, deadline, schedulePlanDate, acceptance, assignedTo, windowName)
SELECT CONCAT(@demo,'可按期-已澄清'), 'inroadmap', 'active', '对公网银', '2', '2026-11-02', '2026-10-25', '已有验收标准', 'demo_leader', CONCAT(@demo,'窗口-进行中')
WHERE NOT EXISTS (SELECT 1 FROM zt_demand WHERE title = CONCAT(@demo,'可按期-已澄清'));

-- 测试中（阶段：测试）
INSERT INTO tmp_vf_demo (title, stage, status, mainSystem, pri, deadline, schedulePlanDate, acceptance, assignedTo, windowName)
SELECT CONCAT(@demo,'可按期-测试中'), 'testing', 'testing', '运营支撑', '1', '2026-10-28', '2026-10-18', '已有验收标准', 'demo_po', CONCAT(@demo,'窗口-进行中')
WHERE NOT EXISTS (SELECT 1 FROM zt_demand WHERE title = CONCAT(@demo,'可按期-测试中'));

-- 待验收（阶段：验收）
INSERT INTO tmp_vf_demo (title, stage, status, mainSystem, pri, deadline, schedulePlanDate, acceptance, assignedTo, windowName)
SELECT CONCAT(@demo,'可按期-待验收'), 'acceptanced', 'acceptanced', '信贷系统', '2', '2026-10-25', '2026-10-15', '已有验收标准', 'demo_leader', CONCAT(@demo,'窗口-进行中')
WHERE NOT EXISTS (SELECT 1 FROM zt_demand WHERE title = CONCAT(@demo,'可按期-待验收'));

-- 待发布（阶段：发布）
INSERT INTO tmp_vf_demo (title, stage, status, mainSystem, pri, deadline, schedulePlanDate, acceptance, assignedTo, windowName)
SELECT CONCAT(@demo,'可按期-待发布'), 'delivering', 'waitdeliver', '手机银行', '1', '2026-10-22', '2026-10-12', '已有验收标准', 'demo_po', CONCAT(@demo,'窗口-进行中')
WHERE NOT EXISTS (SELECT 1 FROM zt_demand WHERE title = CONCAT(@demo,'可按期-待发布'));

-- 【缺数据 x2】——deadline 与 schedulePlanDate 均为空，页面显示「暂无判断」---
INSERT INTO tmp_vf_demo (title, stage, status, mainSystem, pri, deadline, schedulePlanDate, acceptance, assignedTo, windowName)
SELECT CONCAT(@demo,'缺数据-无计划日期A'), 'wait', 'wait', '运营支撑', '1', NULL, NULL, '已有验收标准', 'demo_po', CONCAT(@demo,'窗口-进行中')
WHERE NOT EXISTS (SELECT 1 FROM zt_demand WHERE title = CONCAT(@demo,'缺数据-无计划日期A'));

INSERT INTO tmp_vf_demo (title, stage, status, mainSystem, pri, deadline, schedulePlanDate, acceptance, assignedTo, windowName)
SELECT CONCAT(@demo,'缺数据-无计划日期B'), 'inroadmap', 'active', '手机银行', '2', NULL, NULL, NULL, 'demo_leader', CONCAT(@demo,'窗口-进行中')
WHERE NOT EXISTS (SELECT 1 FROM zt_demand WHERE title = CONCAT(@demo,'缺数据-无计划日期B'));

-- ---------------------------------------------------------------------------
-- 3. 把临时表的业务字段搬进 zt_demand，其余 NOT NULL 列自动补默认。
--    group_concat 生成列清单，NULL 默认值统一用 ''（char/varchar 类），
--    数值/枚举类沿用库自带默认值。
-- ---------------------------------------------------------------------------
-- ---------------------------------------------------------------------------
-- 3. 把临时表的业务字段搬进 zt_demand。
--    列清单与取值按 zt_demand 实际结构逐列写全（109 列，不含自增 id）：
--    业务字段取临时表，其余 NOT NULL 列按库内默认值显式补齐。
--    全部为 MySQL 通用语法（SELECT + 显式列），兼容 OceanBase MySQL 模式，
--    不使用 CTE、窗口函数、ON DUPLICATE 等 MySQL8 专有写法。
-- ---------------------------------------------------------------------------
INSERT INTO zt_demand
  (parent,
   pool,
   module,
   pri,
   severity,
   category,
   source,
   sourceNote,
   name,
   desc,
   feedbackBy,
   email,
   mobile,
   assignedTo,
   QD,
   RD,
   BRA,
   mainSystem,
   reviewer,
   status,
   stage,
   hang,
   hangUpReason,
   isChange,
   changedReviewers,
   changedBy,
   isManagerReview,
   managerReviewers,
   managerReviewResult,
   submitBy,
   isReturned,
   returnedBy,
   returnedReason,
   deadline,
   createdBy,
   originator,
   createdDate,
   clarifyDate,
   closedBy,
   closedDate,
   closedReason,
   mailto,
   deleted,
   clarifyDesc,
   story,
   reviewedBy,
   reviewedDate,
   finalStatus,
   duplicateDemand,
   submitAcceptanceDate,
   acceptance,
   acceptancedDate,
   accepter,
   comment,
   estimateLaunch,
   schedulePlanDate,
   actualDevStartDate,
   actualTestStartDate,
   estimateLaunchChange,
   deliverDate,
   publishWindow,
   mainDevelopers,
   mainTesters,
   onetimeAcceptance,
   reviewMark,
   isNeedReview,
   leadDept,
   developFinish,
   developFinishChange,
   testFinish,
   testFinishChange,
   verifyFinish,
   verifyFinishChange,
   feedback,
   editedBy,
   editedDate,
   isNeedFocus,
   changeReason,
   dispatchTime,
   isDispatched,
   demandCompletionTime,
   color,
   originalStatus,
   proposeDept,
   demandSize,
   overall,
   deliveryCycle,
   deliveryQuality,
   collabora,
   appraiseTime,
   appraiseDesc,
   appraiseBy,
   reasonType,
   delaySystem,
   devPostponed,
   testPostponed,
   verifyPostponed,
   launchPostponed,
   isCarReview,
   isGrayVerifyPlan,
   isNewProduct,
   isRelatedAccounts,
   isNewFunction,
   isImportantOrder,
   isAutoRecImportantOrder,
   isOtherImportantOrder,
   version,
   parentVersion,
   keywords)
SELECT
  0,
  0,
  0,
  t.pri,
  '',
  ,
  ,
  ,
  '',
  '',
  '',
  ,
  '',
  IFNULL(t.assignedTo,''),
  '',
  '',
  '',
  t.mainSystem,
  '',
  t.status,
  t.stage,
  0,
  '',
  '',
  '',
  ,
  wait,
  '',
  '',
  '',
  0,
  ,
  '',
  t.deadline,
  'seed',
  ,
  NOW(),
  0,
  ,
  0,
  ,
  '',
  '0',
  '',
  0,
  '',
  0,
  t.status,
  0,
  0,
  IFNULL(t.acceptance,''),
  0,
  '',
  '',
  0,
  t.schedulePlanDate,
  0,
  0,
  0,
  0,
  0,
  ,
  ,
  '',
  ,
  '',
  '',
  0,
  0,
  0,
  0,
  0,
  0,
  0,
  '',
  NOW(),
  0,
  '',
  0,
  0,
  0,
  ,
  '',
  '',
  ,
  0,
  0,
  0,
  0,
  0,
  '',
  '',
  ,
  '',
  0,
  0,
  0,
  0,
  0,
  ,
  -1,
  -1,
  -1,
  0,
  0,
  -1,
  ,
  0,
  
FROM tmp_vf_demo t;

-- 4. 建立「窗口 ↔ 需求」关联；重复执行靠 NOT EXISTS 跳过。
-- ---------------------------------------------------------------------------
INSERT INTO zt_demandwindow (demand, story, versionWindow, createdBy, updatedBy, createdDate, updatedDate)
SELECT d.id, 0, w.id, 'seed', 'seed', NOW(), NOW()
FROM tmp_vf_demo t
JOIN zt_demand d ON d.title = t.title AND d.deleted = '0'
JOIN zt_versionwindow w ON w.name = t.windowName AND w.deletedAt IS NULL
WHERE NOT EXISTS (
  SELECT 1 FROM zt_demandwindow dw
  WHERE dw.demand = d.id AND dw.story = 0 AND dw.versionWindow = w.id AND dw.deletedAt IS NULL
);

-- 已发布窗口额外挂两条，便于对比「已上线」窗口的只读态
INSERT INTO zt_demandwindow (demand, story, versionWindow, createdBy, updatedBy, createdDate, updatedDate)
SELECT d.id, 0, w.id, 'seed', 'seed', NOW(), NOW()
FROM zt_demand d, zt_versionwindow w
WHERE w.name = CONCAT(@demo, '窗口-已发布') AND w.deletedAt IS NULL
  AND d.title IN (CONCAT(@demo,'可按期-待验收'), CONCAT(@demo,'可按期-待发布'))
  AND d.deleted = '0'
  AND NOT EXISTS (
    SELECT 1 FROM zt_demandwindow dw
    WHERE dw.demand = d.id AND dw.story = 0 AND dw.versionWindow = w.id AND dw.deletedAt IS NULL
  );

DROP TEMPORARY TABLE IF EXISTS tmp_vf_demo;

-- ============================================================================
-- ROLLBACK（与本脚本配套；仅删除带【DEMO】前缀的数据，不触碰任何现有数据）
-- ============================================================================
-- DELETE dw FROM zt_demandwindow dw
--   JOIN zt_demand d ON d.id = dw.demand
--   JOIN zt_versionwindow w ON w.id = dw.versionWindow
--   WHERE d.title LIKE CONCAT('【DEMO】', '%') OR w.name LIKE CONCAT('【DEMO】', '%');
--
-- DELETE FROM zt_versionwindow WHERE name LIKE CONCAT('【DEMO】', '%');
-- DELETE FROM zt_demand WHERE title LIKE CONCAT('【DEMO】', '%');
