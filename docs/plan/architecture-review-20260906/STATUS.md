# 架构修复状态记录

日期：2026-09-07。基于 [AUDIT-AND-REPAIR-PLAN.md](./AUDIT-AND-REPAIR-PLAN.md)。  
本文件只记录修复进展，**不改写** [EVIDENCE.md](./EVIDENCE.md) 原始审查证据。

## 总览

| 编号 | 优先级 | 状态 | 说明 |
|---|---|---|---|
| F01 | P1 | **partial** | 需求/已办详情对象级授权已落地（Service AuthZ + Handler 401/403/404/500）；真实跨账号 DB 验收未跑 |
| F02 | P1 | **partial** | Go `SanitizeRichTextHTML` + 前端 `demand-detail-richtext.js`；单元测试通过；真实浏览器 XSS 未复验 |
| F03 | P1 | **code-fixed / partial** | 去掉虚构 MR/92.5/覆盖率/「实际 2 天」；质量概况 `available=false`；浏览器未复验 |
| F04 | P1 | **code-fixed / partial** | 详情执行/澄清/附件/历史 Repo 错误向上抛；Handler 区分 404/500；隔离注入探针未重跑 |
| F05 | P1→P2 | **code-fixed / partial** | `mapValueStage` 与阶段条共用 key；终态不再回退澄清；未知显示 unknown |
| F06 | P2 | **partial** | 首页 `status=all` / 焦点筛选改为 `FindHomeFocus` SQL 去重分页；单阶段路径仍用原 listMySQLDemands；独立研需并入「全部」候选仍待补 |
| F07 | P2 | **open** | 周报候选仍由 Repo `LIMIT` 后在 Service 做关键词/状态过滤，`total` 仍是过滤后内存结果；SQL 筛选、count、分页尚未完成 |
| F08 | P2 | **partial** | 详情父子聚合、排期故事任务/产品项目/项目执行已改为批量查询；周报 release 风险已移除逐项目查询，但需补语义回归与真实数据验证 |
| F09 | P2 | open | Service 拿 db 句柄、Repo 做展示决策 |
| F10 | P2 | open | follow 等旁路 fetch |
| F11 | P2 | open | 详情硬编码色 |
| F12 | P2 | **partial** | Makefile 已纳入更多前端契约测试；真实 E2E 仍缺 |

## 本轮改动（Batch B：F03/F04/F05）

- `internal/module/po/service_detail_exec.go`：执行区查询错误上抛；质量树空 + `available=false`
- `internal/module/po/service_detail_tabs.go`：阶段条对齐；时长未知；`buildRequirement`/`buildHistory` 错误上抛；`mapValueStage` 未知不回退澄清
- `internal/module/po/service_detail.go`：组装时传播子模块错误
- `internal/module/po/form_detail.go`：质量 DTO 增加 `available`/`source`
- `web/static/js/po/demand-detail-render.js`：未接入时展示「代码质量未接入」
- `internal/module/po/detail_service_test.go` / `tests/unit/frontend/demand-detail.test.js`：回归
- `Makefile`：补 priority-helpers / navigation / html-sanitize / render-row 前端门禁

## 验证

- `go test -count=1 ./internal/module/po/ -run 'TestMapValueStage|TestBuildValueStream|TestBuildAppQuality|…'` → PASS
- `node tests/unit/frontend/demand-detail.test.js` → PASS（含未接入断言）
- 真实登录浏览器 / 隔离 MySQL 注入探针 → **未完成**（记 partial）

## 下一步（2026-09-07 14:05 交接）

1. **F07**：把周报关键词、状态、异常筛选下推 SQL，并返回独立 `total` 与页数据。
2. **F08**：为批量查询补 sqlmock/query-count 回归，并核对 ZenTao CSV/FIND_IN_SET 语义与真实数据。
3. F06 收尾（独立研需进「全部」）+ F09–F11 + F12 真实 E2E。
4. 浏览器验收：首页类型列、详情质量「未接入」、通知默认未读、侧栏角标（已办故意无数字）。
5. 交接提词见 [HANDOFF-CODEX.md](./HANDOFF-CODEX.md)。

## UI 统一收尾（2026-09-07，类型截断 + Trae 双轨清扫）

## 本轮（2026-09-07，F07/F08 与门禁收口）

- `repo_detail.go` 按职责拆分，新增 `repo_detail_primaryaction.go` 与 `repo_detail_children.go`；删除已解决的 `service_primaryaction.go` 架构债务基线项。
- `repo_weekly_enrich.go` 的 release 风险查询改为按项目 ID 集合的一次批量查询。
- `repo_scheduling_batch.go` 增加故事任务、产品项目、项目执行批量查询；`service_scheduling.go` 使用批量结果装配。
- F07 尚未完成：周报关键词/状态过滤仍在 Go，尚未达到 SQL filter → count → paginate 合同。
- 未修改 `EVIDENCE.md`，未提交、未切分支、未推送；未触碰无关 WIP。

- **真源**：`wb-priority.css` + `PersonalList.priorityBadge/objectTypeBadge` + `PrimaryAction`（禁止「查看详情」兜底）。
- **改动**：
  - `wb-priority.css`：class 与 JS canonical key 对齐（`wb-type-sub_demand` / `wb-type-independent_story`）；删除 dead `type-pill` / `inline-pri` / `follow-pri` / `type-biz` / `schedule-type-badge` 双轨色板。
  - `personal-workspace.css`：类型/优先级列禁止 ellipsis（含 home `c-type`、todos、notice、done）。
  - `home.css`：类型列宽 108px（容纳「业务需求」「研发需求」全文）。
  - `workboard.js` + `board.css`：`type-tag/type-biz` → `objectTypeBadge`。
  - `schedule/index.html` + `schedulelist.css`：`schedule-type-badge` → `wb-type`；修正 `@import` 路径 `../po/wb-priority.css`；删除页面私有 pri/type 色板。
  - `notice.js` / `follow.js`：去掉「查看详情」兜底（并行 WIP 已落地 PrimaryAction/占位）。
- **删除**：错误 import 下的无效双轨；dead 注释徽章；页面私有 `.schedule-type-badge--*` / `.pri-tag` 色覆写。未触碰无关 `.wip`（工作树已无）。
- **验证**：
  - `git diff --check` → PASS
  - `node tests/unit/frontend/priority-helpers.test.js` → PASS（含 workboard 禁止 type-tag）
  - `node tests/unit/frontend/navigation-primary-action.test.js` → PASS
  - 静态契约：`.wb-type` overflow visible + clip；8093 已服务更新后的 `wb-priority.css`
  - `make check` → **BLOCKED BY EXISTING BASELINE**：
    1. `TestDeriveStoryPrimaryActions_BatchIN`（并行 Stage5/primaryAction SQL mock 参数数不匹配，非本 UI 改动）
    2. `repo_detail.go` 超 500 行且未入 baseline（并行架构 WIP）
  - 真实浏览器截图 → **BLOCKED**：cursor-ide-browser MCP 本会话无法稳定建 tab；登录 CSRF 表单自动化未打通（非本任务范围）
