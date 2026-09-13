# 指标模块三套字典语义对账表（P0-1 实施前）

时间：2026-09-13　HEAD：`3507193e`
对账对象：后端 `service.go buildItems`(12) vs 前端 `metrics-manage.js defaultMetrics`(14) vs 前端 `metrics-radar.js` 硬编码(13)。
结论：三套集合**不等价**，唯一 code 并集 **21 个**。本表为「实施前对账」，**不删除任何指标**。

## 一、并集分组

### A 组 — 仅后端定义（7 条，前端未用）

| code | 名称 | 分类 | 数据源 | backend | manage | radar | 重复? | 保留 |
|---|---|---|---|---|---|---|---|---|
| story.closedOnTimeRate | 按时关闭率 | 交付效率 | 禅道 zt_story | ✓ | ✗ | ✗ | 否 | ✅ 保留 |
| task.overdue | 逾期任务数 | 交付效率 | 禅道 zt_task | ✓ | ✗ | ✗ | 否 | ✅ 保留 |
| bug.p1p2 | 致命/严重缺陷数 | 研发质量 | 禅道 zt_bug | ✓ | ✗ | ✗ | 否 | ✅ 保留 |
| bug.resolutionRate | 缺陷解决率 | 研发质量 | 禅道 zt_bug | ✓ | ✗ | ✗ | 否 | ✅ 保留 |
| norm.owner | 责任人必填覆盖率 | 规范执行 | 门禁（待同步） | ✓ | ✗ | ✗ | 否 | ✅ 保留 |
| task.total | 任务总量 | 效能管理 | 禅道 zt_task | ✓ | ✗ | ✗ | 否 | ✅ 保留 |
| task.open | 未完成任务 | 效能管理 | 禅道 zt_task | ✓ | ✗ | ✗ | 否 | ✅ 保留 |

### B 组 — 仅前端定义（9 条，后端缺失；真实业务能力，不能误删）

| code | 名称 | 分类 | 数据源 | backend | manage | radar | 重复? | 保留 |
|---|---|---|---|---|---|---|---|---|
| delivery.cycle | 交付周期 | 交付效率 | FineReport(JbuB)/禅道 zt_demand.teamGroup | ✗ | ✓ | ✓ | 否 | ✅ 补入后端 |
| implement.cycle | 实施周期 | 交付效率 | FineReport(JbuB)/禅道 zt_demand.teamGroup | ✗ | ✓ | ✓ | 否 | ✅ 补入后端 |
| story.overIteration | 超两迭代周期占比 | 需求治理 | FineReport(hNBB)/禅道看板 | ✗ | ✓ | ✓ | 否 | ✅ 补入后端 |
| story.unscheduled | 超2周未排期单数 | 需求治理 | FineReport(hNBB)/禅道 zt_demand | ✗ | ✓ | ✓ | 否 | ✅ 补入后端 |
| gate.passRate | 质量门禁通过率 | 开发质量* | DevOps/门禁平台 | ✗ | ✓ | ✓ | 否 | ✅ 补入后端 |
| bug.closeRate | 缺陷关闭率 | 研发质量 | 禅道 zt_bug/看板成员 | ✗ | ✓ | ✓ | 否 | ✅ 补入后端 |
| bug.responseRate | 缺陷响应效率 | 研发质量 | 禅道 zt_bug | ✗ | ✓ | ✓ | 否 | ✅ 补入后端 |
| story.delayedLaunch | 上线延期数 | 需求治理 | FineReport(hNBB)/禅道需求 | ✗ | ✓ | ✓ | 否 | ✅ 补入后端 |
| sp.deviationRate | SP偏差率 | 效能管理 | FineReport(Y3VO)/禅道工时 | ✗ | ✓ | ✓ | 否 | ✅ 补入后端 |

> *`gate.passRate` 在 manage 里分类写作「开发质量」，与后端 `ValidCategories` 的「研发质量」不一致，需归一。

### C 组 — 多套重叠（5 条，需收敛为后端唯一定义）

| code | 名称 | backend | manage | radar | 重复点 |
|---|---|---|---|---|---|
| story.total | 研发需求总量 | ✓ | ✓ | ✗ | 2 套：数据源 manage 写 zt_demand/zt_story，后端 zt_story |
| story.active | 进行中需求 | ✓ | ✓ | ✓ | 3 套：阈值漂移（见下）|
| story.doneRate | 研发需求完成率 | ✓ | ✓ | ✓ | 3 套：**判定语义相反**（见下）|
| bug.open | 未关闭缺陷 | ✓ | ✓ | ✓ | 3 套：team 模式阈值不一致 |
| norm.completeness | 业务需求富文本完整性 | ✓ | ✓ | ✓ | 3 套：radar 名称/阈值漂移 |

## 二、重叠项漂移清单（收敛时必须统一，以业务确认值为准）

| code | 漂移项 | 后端 | manage | radar |
|---|---|---|---|---|
| story.active | dangerThreshold | 60 | 45 | group 45 / team 500 |
| story.active | target | ≤30 | ≤30个 | group ≤30 / team ≤300 |
| story.doneRate | dangerThreshold | 90（达标线）| 60（危险线）| 60（危险线）|
| story.doneRate | status 判定 | up: ≥90 正常 | up: <60 危险 | up: <60 危险 |
| bug.open | dangerThreshold | 10 | 10 | group 10 / team 100 |
| norm.completeness | name | 业务需求富文本完整性 | 同后端 | 规范执行完整性 |
| norm.completeness | dangerThreshold | 95 | 95 | 70 |

> **关键：`story.doneRate` 的 `dangerThreshold` 语义在前后端完全相反** —— 后端把 danger 当「达标线」(up 方向 ≥90 才算 normal)，前端把 danger 当「危险线」(up 方向 <60 才算 danger)。这是 status 判定算法的分裂，不是单纯数值漂移。

## 三、对账结论（供实施）

1. **唯一 code 并集 21 条** = A 组 7（仅后端）+ B 组 9（仅前端，FineReport/DevOps 效能）+ C 组 5（重叠）。
2. **SSOT 归属**：指标元数据（code/name/category/unit/description/order/enabled/source type）**统一由后端负责**；21 条都应进后端统一目录。
3. **数据来源分层**（不把计算都搬进 service.go）：
   - 禅道聚合（A 组 7 + C 组 5）：后端 `repo.Snapshot` 已有真实 SQL。
   - FineReport/DevOps 效能（B 组 9）：数据源在外部系统，后端只登记**元数据定义**，值来源标注「外部（未接）」，暂显「暂无数据」——**不伪造、不用前端模拟值兜底**。
4. **前端收敛**：manage 删 `defaultMetrics` + localStorage 假落库；radar 删硬编码目录 + 模拟值 fallback；两者统一读后端 metadata API。localStorage 仅保留纯用户偏好（排序/展示开关）。
5. **`/metrics/radar/data`**：零消费者 → 调用链确认后删除，不人为制造消费者。
6. **本轮不动**：FineReport 数据源接入、雷达 SVG 重画、指标模块整体重构——记范围外。
