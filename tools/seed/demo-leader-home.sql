-- ============================================================================
-- 文件: tools/seed/demo-leader-home.sql
-- 模块: 工作台 · 演示数据
-- 职责: 让 demo_leader（【DEMO】科技主管-张健）在首页能看到自己的需求。
--       本脚本只写不执行；执行由产品/研发审核后手工进行。
--
-- 背景（第 6 轮 T2 取证结论）：
--   首页需求列表的可见性只认「个人级关系」，见
--   internal/module/po/home_focus_repo.go 的 homeFocusDemandBase：
--     createdBy / originator / assignedTo / QD / RD / BRA
--     + 澄清 PM / 评审人 / 经理评审人 / accepter
--   其中不含「团队长管辖部门」。而详情页 demandauthz.Evaluate 第 3 步会把
--   「管辖部门内干系人的需求」判为 AccessRelated。两者口径不一致，导致
--   demo_leader 在详情页可读可写、首页却是 0 行。
--
--   本轮不改代码、不放宽任何账号可见范围（该口径差异留待产品裁决），
--   只补「demo_leader 本人直接相关」的需求，让演示账号首页有数据，
--   不依赖团队长管辖放行。
--
-- 特性：
--   1. 幂等：可重复执行，不产生重复数据、不覆盖任何现有行。
--   2. 只新增带【DEMO】前缀的需求；不 UPDATE、不 DELETE 任何现有行。
--   3. 绝不触碰需求 63450 / 63449（演示脏数据），脚本全程未引用这两个 ID。
--   4. 软删除口径：zt_demand.deleted 是 enum('0','1')，写入用字符串 '0'。
--   5. 列为 zt_demand 实际结构逐列写全（109 列，不含自增 id），
--      语法为 MySQL 通用写法，兼容 OceanBase MySQL 模式，
--      不使用 CTE、窗口函数、ON DUPLICATE 等专有语法。
--
-- 造 3 条需求，分别落在 assignedTo / createdBy / RD 三个关系列上，
-- 覆盖首页可见性 WHERE 的不同分支；阶段取已澄清待排期，便于演示排期按钮。
--
-- 执行方式：
--   mysql -h <host> -P <port> -u <user> -p <db> < tools/seed/demo-leader-home.sql
-- ============================================================================

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
  IFNULL(t.rd,''),
  IFNULL(t.bra,''),
  t.mainSystem,
  '',
  'clarified',
  'incharter',
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
  IFNULL(t.originator,''),
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
  'wait',
  0,
  0,
  '已有验收标准',
  0,
  '',
  '',
  0,
  '2026-10-20',
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
  
FROM (
  SELECT '【DEMO】科技主管-张健负责-风险台账自动化' AS title, '运营支撑' AS mainSystem,
         '2' AS pri, '2026-11-06' AS deadline, 'demo_leader' AS assignedTo,
         '' AS bra, '' AS rd, 'demo_biz' AS originator
  UNION ALL SELECT '【DEMO】科技主管-张健发起-监管报送口径统一', '手机银行',
         '2', '2026-11-13', '', '', '', 'demo_leader'
  UNION ALL SELECT '【DEMO】科技主管-张健担任RD-数据中台监控补齐', '信贷系统',
         '3', '2026-11-20', '', '', 'demo_leader', 'demo_biz'
) AS t
WHERE NOT EXISTS (
  SELECT 1 FROM zt_demand d
  WHERE d.title LIKE '【DEMO】科技主管-张健%' AND d.deleted = '0'
);

-- ============================================================================
-- ROLLBACK（与本脚本配套；只删除本脚本新增的【DEMO】需求）
-- 63450 / 63449 永不受影响。
-- ============================================================================
-- DELETE FROM zt_demand
--   WHERE title LIKE '【DEMO】科技主管-张健%'
--     AND id NOT IN (63450, 63449);
