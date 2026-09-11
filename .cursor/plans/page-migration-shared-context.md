# 页面迁移共享上下文

仓库：`workbench-claude-po`
分支：`Claude-PO`（HEAD `f2fa3d31`）
工作区 dirty 列表（与本任务无关，**不要触碰**）：
- modified: `internal/bootstrap/bootstrap.go` / `internal/constants/templates.go` / `internal/module/po/handler.go` / `internal/server/routes.go` / `web/static/js/po/workboard.js` / `web/templates/layout/sidebar.html`
- untracked: `.run/` / `.workbuddy/` / `docs/PRD/` / `docs/CODE-HEALTH-AUDIT-REPORT.md` / `docs/CODE-HEALTH-REVALIDATION-20260907.md` / `internal/module/metrics/` / `internal/module/query/` / `internal/module/po/form_issue_risk.go` 等 4 个 issue_risk 文件 / `web/static/css/{metrics.css,po/issue-risk.css,query/}` / `web/static/js/{metrics.js,po/issue-risk.js}` / `web/templates/{metrics/,po/issue-risk.html,query/}`

**本任务目标模块**（untracked 部分，必须落到 plan 与 commit）：
- `internal/module/query/` （4 文件）
- `internal/module/metrics/` （4 文件）
- `internal/module/po/form_issue_risk.go` / `handler_issue_risk.go` / `repo_issue_risk.go` / `service_issue_risk.go`
- 对应模板与前端

## 已确认壳信息（**所有 Agent 必须复用，不要造第二套**）

### 模板常量（已定义）
- `TEMPLATE_PO_HOME = "po/home"`
- `TEMPLATE_PO_TODOS = "po/todos"`
- `TEMPLATE_PO_DONE = "po/done"`
- `TEMPLATE_PO_NOTICE = "po/notice"`
- `TEMPLATE_PO_FOLLOW = "po/follow"`
- `TEMPLATE_PO_BOARD_DEMAND = "po/workboard"` / `TEMPLATE_PO_BOARD_TASK = "po/workboard"`
- `TEMPLATE_QUERY_INDEX = "query/index"` ✓ 已存在
- ⚠️ `/metrics/manage` 与 `/metrics/radar` 在 `templates.go` 中**尚未定义 TEMPLATE_METRICS_MANAGE / TEMPLATE_METRICS_RADAR**，需要补上（修改 `internal/constants/templates.go`，仅增加常量）
- ⚠️ `TEMPLATE_PO_BOARD_*` 已存在但 `/issues/risk` 是否需要 `TEMPLATE_PO_ISSUE_RISK` 常量待定；可考虑沿用 `render.Page` 直接传字符串 `"po/issue-risk"`（现有 handler_issue_risk.go:12 就是这么做的，无常量）

### Sidebar 链接（已就位，**不要再改 sidebar.html**）
- `/query` 已 active 匹配
- `/issues/risk` 已 active 匹配
- `/metrics/manage` 已 active 匹配
- `/metrics/radar` 已 active 匹配
- sidebar 的"研发工作台"分区已包含 4 个目标链接

### 路由挂载（`internal/server/routes.go:97-103`）
- `queryHandler` 已在 PO group（`""`，无前缀）下挂载，`RegisterRoutes` 定义在 `internal/module/query/handler.go:30-35`：
  ```go
  g := rg.Group("/query")
  g.Use(middleware.ActiveNav("/query"))
  g.GET("", middleware.RequirePerm(perm.ScheduleList), h.Index)
  g.GET("/items", middleware.RequirePerm(perm.ScheduleList), h.Items)
  ```
- **问题**：`perm.ScheduleList` 用作 query 页面入口权限不合适（语义不对）；**但不能改 perm 常量**（会破坏 `TestAllPermissionsComplete`）。
  - **方案**：`/query` 是查询页面，沿用 `perm.ScheduleList`（用户已有该权限才看到排期，进而能查；语义勉强说得通："能排期就允许查"）；不新增常量。如确需新增，**单独立任务**走 baseline 流程。

### `/issues/risk` 路由（**已挂载**）
- `internal/module/po/handler.go:81-84` 已经挂载 `/issues/risk` 与 `/issues/risk/items`（兼容别名 `/issue-risk` 与 `/issue-risk/items` 也同时挂载）；权限 `perm.PoBoardDemandList`。
- 不需要新增路由；如需对齐 sidebar 链接可以保持现状。
- **注意**：`PoBoardDemandList` 与 `/board/*` 同权，语义上对 issue/risk 略弱（issue/risk 是否应单独 capability 是开放问题）；**不新增 perm 常量**，沿用现有。

### `/metrics/manage` 与 `/metrics/radar` 路由
- `bootstrap.go:137` 已经构造 `metricsHandler`，`routes.go:100-102` 已经挂载到 PO group。
- `internal/module/metrics/handler.go:22-27` 已经挂 `/metrics/manage` / `/metrics/radar` / `/metrics/api` 三个 GET；**但都没有挂 `middleware.RequirePerm`**（仅 `RequireLogin`），违反 AGENTS §3。
- **必须改**：把 `RequireLogin` 替换为 `RequirePerm(perm.PoHomeList)` 或 `perm.ScheduleList`；**不新增 perm 常量**。
- 不动 `routes.go` / `bootstrap.go`（dirty）；**只在 `internal/module/metrics/handler.go` 内加 middleware**。

### 已复用的前端组件 / helper（**必须使用，禁止重新造**）
- `web/static/js/po/personal-list.js` 暴露：`escapeHtml` / `priorityBadge` / `objectTypeBadge` / `objectTypeBadgeFromKind` / `renderPagination` / `createController` / `loadPageSize` / `savePageSize`
- `web/static/css/po/personal-workspace.css` 暴露：`.workspace-panel` / `.category-tabs` / `.workspace-toolbar` / `.workspace-result-bar` / `.workspace-pagination` / `.workspace-search` / `.workspace-table` / `.workspace-header` / `.ir-*` 等
- 模板基线：`po/issue-risk.html` 已经使用这些类；后续两个页面照搬骨架。

### 现有 reference 页面（**模仿它们**）
- `web/templates/po/issue-risk.html` —— 已基本完整；Agent 只需核对 CRCBWorkbench 是否有额外细节要补
- `web/templates/po/todos.html` / `po/follow.html` / `po/notice.html` —— 列表 + 筛选 + 分页的成熟模板骨架
- `web/templates/po/home.html` —— 首页 KPI + 价值流联动样式

### 数据库与 SQL 约定（**强制**）
- `internal/module/query/repo.go` 当前已有 listStories（业务需求 + 研发需求），分页走 SQL count + limit/offset；**基本符合规范**。
- `internal/module/metrics/repo.go` 待 Agent 调研。
- `internal/module/po/repo_issue_risk.go` 已用 `q.Count(&total)` + `Offset/Limit`，符合规范。
- **禁止**：把全量数据加载后 Go/JS 内存过滤或分页。

### ID 命名（**强制，PRD §17 / 已落地的 front-end helper**）
- 业务需求：禅道原始数字 ID + `US` 前缀（`US{id}`）
- 研发需求：纯数字 ID（无前缀，与禅道一致）
- 其它对象（任务/Bug/Issue/Risk）：纯数字 ID

### 状态中文映射（**必须使用 `priorityBadge` 与已有的渲染**）
- 业务需求 status：`draft/wait/refuse/active/clarified/developing/testing/waitacceptance/acceptanced/waitdeliver/delivered/released/closed/suspended` 等已在 `home.js:90-95` 定义中文 label
- Story status：`draft/reviewing/active/changing/closed` 在 `home.js:97-99`
- Issue/Risk status 已在 `service_issue_risk.go:72` 定义

### 加载 / 空 / 错误态（**强制**）
- 加载：`workspace-result-bar` + `<td class="state-placeholder">正在拉取...</td>`
- 空：`<td colspan="N" class="state-placeholder">当前条件下暂无 XXX</td>`
- 错误：`<td colspan="N" class="state-placeholder error">加载失败，请重试</td>`

## 与本任务完全无关，**不要触碰**

- `web/static/js/po/workboard.js`（dirty，与本任务无关）
- `web/templates/layout/sidebar.html`（dirty，但内部已含 4 个目标 active 匹配，**不要改**）
- `internal/bootstrap/bootstrap.go` / `internal/server/routes.go`（dirty，**禁止改动**）
- `internal/module/po/handler.go`（dirty，**禁止改动路由表**；4 个目标路由已挂载）
- `docs/CODE-HEALTH-AUDIT-REPORT.md` / `docs/CODE-HEALTH-REVALIDATION-20260907.md`（审计报告，独立任务）
- `.run/` / `.workbuddy/` / `docs/PRD/`（工具产物）

## 执行约束

1. 保护现有页面 — 不要改 sidebar、layout、shell、其它已成熟页面。
2. 模板必须使用 `workspace-panel` / `category-tabs` / `workspace-toolbar` / `workspace-result-bar` / `workspace-table` / `workspace-pagination` / `workspace-search` 等已定义 class，**禁止引入新 class 名**（避免分裂）。
3. JS 优先复用 `personal-list.js` 暴露的 helper；自实现 `esc()` 仅作 fallback（参见 `home.js:11`）。
4. SQL filter → sort → count → pagination，**禁止内存过滤**。
5. ID 前缀规则：业务需求 `US`，其它对象纯数字。
6. 状态/优先级必须经过 `priorityBadge` / `objectTypeBadge` / `issueRiskStatusLabel` 之类映射，**禁止英文 raw 状态输出**。
7. 每个模块最后给出"改动文件清单 + 行为说明 + 测试结果"。