# Phase 2 最终报告 — 治理框架冻结后的业务完整性收口

日期：2026-09-13 · 分支 `release/po-integrate-main-202609` · worktree `/Users/yuyan9923/GitHub/workbench-claude-po`

> 本报告对应 Phase 2 指令第「十三」节的 8 项交付。三个 P0 全部修复，`make check` 与 `make check-gates` 均 EXIT=0。

---

## 1. 三个 P0 是否全部修复

| P0 | 问题 | 结论 | 证据 |
|---|---|---|---|
| P0-1 | 指标模块前后端两套不相交字典 + `localStorage` 假落库 | **PASS** | 后端 `catalog.go` 21 条唯一 code 成为唯一 SSOT；前端删 `defaultMetrics`/`localStorage`，统一 `fetch("/metrics/api")` |
| P0-2 | 死端点 `GET /metrics/radar/data` | **PASS** | 前端二次确认 0 消费后全链删除（路由 + `RadarData` + `Radar()` + `RadarSummary`） |
| P0-3 | 版本窗口「已关联需求/产品」仍可删除 → 孤儿 `zt_demandwindow` | **PASS** | `service.Delete` 加 `CountWindowAssociations` 守卫，`>0` 即拒删；3 个行为测试 |

三个 P0 **全部 PASS**，无 `BLOCKED BY EXISTING BASELINE`。

---

## 2. SSOT / Invariant

**指标元数据 SSOT** = `internal/module/metrics/catalog.go`，21 条唯一 code，字段统一为
`code/name/category/unit/description/sourceType/direction/target/danger/order/enabled`。

**数据来源分层不变量**：`SourceZentao`（10 条，`repo.Snapshot` 真实 SQL 聚合）vs
`SourceExternal`（11 条，FineReport/DevOps/门禁，未接入）。外部指标未接入时
`value="—" + status="unknown"`，**禁止模拟值/随机数/`monthSeed`/`orgSeed`/`0` 冒充**；
`unknown` 不参与 radar 综合评分/均值/排名/趋势（`computeCategoryScores` 只统计 normal+warn+danger）。

**两阈值模型不变量**：`target`=达标线、`danger`=危险线，`warning` 是二者间的派生中间态
（不设独立字段）；`evaluateStatus` 按 `direction` 判定。`story.doneRate`：`direction=up`、
`target=90`、`danger=60` → value≥90 normal / 60≤value<90 warn / value<60 danger。

**分类归一不变量**：`gate.passRate` 分类统一为「研发质量」（消除「开发质量」并存）。

**删除守卫不变量（P0-3）**：`zt_demandwindow` 或 `zt_versionwindowproduct` 任一存在软删记录
即 `CountWindowAssociations > 0` → 拒绝删除，杜绝孤儿行。

---

## 3. Scope Gate 落地

`AGENTS.md` 新增 `## Scope Gate` 章节（位于「Mandatory pre-flight」之后、「MUST rules」之前），
**未新增任何规则文档**。内容为：Implement 前输出固定格式 Scope Contract
（目标/必须改变/允许影响/明确不处理 OUT_OF_SCOPE/预计修改/预计不修改/验收条件/发现的额外问题），
并定义 trivial 变更（typo/单行修正/纯格式）豁免。范围外发现标 `OUT_OF_SCOPE` 只报不改。
MUST 规则仍保持 1–16，未被稀释。

---

## 4. 删除了什么

**后端（P0-1/P0-2）**：
- `internal/module/metrics/radar_service.go` — 整文件删除（5 分类 `Radar()` 派生实现）
- `internal/module/metrics/repo.go` — 删 `RadarSummary`
- `internal/module/metrics/handler.go` — 删 `GET /radar/data` 路由与 `RadarData` 方法
- `internal/module/metrics/form.go` — 删 `RadarCategoryItem`/`RadarCategoryScore`/`RadarResp`
- `internal/module/metrics/service.go` — 删旧 12 条硬编码 `Code:` 及 `MetricStatusByRate/ByCount/MetricTextForRate/ForCount`、旧 `buildItems`

**前端（P0-1 收敛）**：
- `metrics-manage.js` — 删 `defaultMetrics`(14 条)、`localStorage`(`crcb_metrics_meta_config_v2`)、抽屉/保存逻辑
- `metrics-radar.js` — 删硬编码 13 条指标、`monthSeed`/`orgSeed` 模拟值、`loadBoardTeamgroups`
- `manage.html` — 删右侧抽屉 modal、新增指标按钮、启停筛选
- `radar.html` — 删视角切换 tab、`radarTeamgroupSelect`/`radarOrgTeamSelect`/`radarMonthSelect` 下拉

**债务基线净减**：
- `scripts/quality-baseline/file-length.tsv` — 删 2 条 stale（`metrics-manage.js` 611→176、`metrics-radar.js` 701→419，双双降至 500 线下），超限文件 29 → **27**

---

## 5. 新增了什么

- `internal/module/metrics/catalog.go` — 21 条指标 SSOT + `sourceType` 常量 + 两阈值模型（`valueFor`/`evaluateStatus`/`formatValue`）
- `internal/module/schedule/repo.go` — `CountWindowAssociations`（统计两表软删记录）
- `internal/module/schedule/service.go` — `Delete` 加关联守卫
- `internal/module/schedule/service_window_delete_test.go` — 3 个行为测试
- `internal/module/metrics/service_test.go` — 重写为 16 个测试（catalog 唯一性/来源分布/两阈值语义/外部不可用）
- 前端 `loadMetrics`（`fetch("/metrics/api?page=1&pageSize=100")`）+ `computeCategoryScores`（仅统计 normal+warn+danger）+ `renderMatrixGrid`（外部指标「暂无数据」+ unavailable gauge）
- `AGENTS.md` — Scope Gate 章节
- `docs/engineering/debt.md` — 顶部 HISTORICAL SNAPSHOT 声明

---

## 6. 是否平行实现

**无平行实现。** 前端第二套字典（`defaultMetrics` + `localStorage`）已物理删除，不是「新旧并存」；
radar 的服务端派生实现（`radar_service.go`）已整文件删除，统一收敛到 `/metrics/api`。不存在
`if newMode { } else { 旧 }` 分支、双注册、兼容兜底。P0-2 未造消费者，直接删除。

---

## 7. Tests

| 验证 | 结果 |
|---|---|
| `go build ./...` | EXIT=0 |
| `go vet ./...` | EXIT=0 |
| `git diff --check` | PASS（无 whitespace error） |
| `go test ./internal/module/metrics/` | **ok**（16 用例，含 `TestCatalog_UniqueCodesAndSize(==21)`、`TestCatalog_SourceTypeDistribution(zentao 10/external 11)`、`TestCatalog_DoneRateSemantics`、`TestValueFor_ExternalIsUnavailable`、`TestBuildItems_AssemblesCatalog`） |
| `go test ./internal/module/schedule/ -run TestDeleteWindow` | **ok**（`WithDemandWindowRejected`/`WithProductRejected`/`NoAssociationSucceeds` 3 用例） |
| `make check` | **EXIT=0**（hard-pattern non-growth 0、secret 0、file-length 27、architecture 1、frontend unit/behavior tests passed） |
| `make check-gates` | **EXIT=0**（38 项自测全过） |

---

## 8. 本轮范围外问题

1. **`radar_service.go` 幽灵条目（非用户可见）**：P0-2 删除文件后仍未 `git add` 进索引，
   `check-architecture.sh` 的 `git ls-files --cached` 命中该缺失文件，打印
   `grep: ... radar_service.go: No such file or directory`（**gate 仍 EXIT=0**）。
   与 `docs/Demo` 169 个删除同口径，待用户统一决定提交时消解。建议后续给
   `check-architecture.sh` 补 `[[ -f "$file" ]] || continue`（与 `check-secrets.sh` 已做的修复一致）。
2. **`DIRECT_PAGE_FETCH` advisory +2（非用户可见）**：前端收敛后 `metrics-manage.js:38`、
   `metrics-radar.js:40` 出现直接 `fetch`，属 advisory（非 hard）不阻塞门禁；全仓已有 63 条同类，
   是「页面直接 fetch」既有惯例的延续，非本轮新增技术债类别。

---

## Change Integrity Review 结论

除授权范围（3 P0 修复 + 前端收敛 + Scope Gate + debt.md 快照声明）外，**无其他用户可感知行为变化**。
radar/manage 页从「模拟值/假落库」变「后端真实数据 / 外部指标暂无数据」是授权内的诚实化；
版本窗口删除被拒是 P0-3 的预期行为变化。理想答案 **NONE** 成立。
