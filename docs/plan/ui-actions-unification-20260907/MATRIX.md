# 页面与问题 MATRIX

> Stage 1 输出：本任务范围内的当前基线、页面/规则/操作矩阵，以及未明业务契约的现有证据。
> 标注：所有数字、行号、颜色值均为本轮观察位置；进入 stage 2+ 必须按符号复核。
> 仅记录，不发起修改。

## 1. 盘点基准

- **分支**：`Claude-PO`（`git branch --show-current` 输出）。未在 `main` / `master` 上工作。
- **HEAD / 工作区 WIP 状态**：
  - 当前为 `* Claude-PO`，未切分支。
  - 工作区有 45 个 dirty tracked（其中 M）和 50+ 个 untracked（??）文件，分散在以下三类：
    - 与本任务无直接关系的并行 WIP（独立审计、共享数据库、个人页对齐等规划包）：
      - `docs/operations/dev-vm-topology.md`
      - `docs/plan/architecture-review-20260906/`
      - `docs/plan/personal-pages-v104-alignment/`
      - `docs/plan/shared-database-optimization-20260906/P1B-VM-*-LOCAL.md`、`P1B-VM-DBA-BUNDLE.md`
      - `docs/plan/shared-database-optimization-20260906/P1B-VM-EXECUTION-RESULT.md` (M)
    - PO 模块相关文件（属本任务 scope 候选，但当前已是 in-flight 改动，可能与 stage 2/3 并发冲突，必须按行号复核）：
      - Go：`internal/module/po/{form,form_detail,formnotice,handler,handler_detail,repo_detail,repo_rw_test,repoboard,repodone,repokpi,reponotice,repovaluestream,service,service_detail,service_detail_tabs,servicedone,servicenotice}.go`、`internal/module/po/login/service.go`、`internal/module/schedule/repo.go`、`internal/pkg/zentao/{url_test,zentao}.go`
      - 模板：`web/templates/po/{home,done,follow,notice,workboard}.html`
      - 前端：`web/static/css/{po/{board,demand-detail,done,follow,personal-workspace},schedule/scheduleintegrated}.css`、`web/static/js/po/{demand-detail-render,done,follow,home,notice,personal-list,todos,workboard}.js`
      - 测试：`tests/unit/frontend/demand-detail.test.js`、`scripts/quality-baseline/file-length.tsv`
    - 新建（未列入本任务 scope，stage 2+ 不应触碰）：`internal/module/po/{form_done,form_weekly,handler_weekly,home_focus_repo,home_focus_test,html_sanitize,html_sanitize_test,kpi_blocked_test,notice_mark_all_test,po_work_scope,repo_weekly,repo_weekly_enrich,repodone_authz,repodone_enrich,service_detail_authz,service_detail_authz_test,service_detail_exec,service_weekly,service_weekly_util,servicedone_authz_test}.go`、`web/static/css/po/{workboard-addon,po/shell,po/homecompact}.css`、`web/static/js/po/{demand-detail-parent,demand-detail-richtext,follow-drawer,html-sanitize,workboard-issue,workboard-modal}.js`、`tests/unit/frontend/{home-focus,html-sanitize,notice-filters}.test.js`
- **服务运行状态（端口 / 是否在线）**：
  - `curl -s -o /dev/null -w "%{http_code}\n" http://127.0.0.1:8093/login` → `200`，服务在线。
  - 未尝试启动；按约束不擅自启动（避免与可能在 init 的另一个 worker 冲突）。
- **make check 当前基线**（执行于 2026-09-07）：
  - 退出码 0；所有必需门禁通过：
    - `gofmt` 通过
    - `go test ./...` 通过（含 `internal/server`、`tests/integration`，cached）
    - `go vet ./...` 通过（0 个 baseline diagnostic）
    - `whitespace gate passed`
    - `file-length regression gate passed`（existing debt: 18 个超 500 行的文件）
    - `hard-pattern non-growth gate passed`（0 个 hard 命中）
    - `secret regression gate passed`（0 个 fingerprint）
    - `architecture boundary regression gate passed`（existing debt: 1 个文件）
  - 提示信息中含 35 条 advisory pattern（`DIRECT_PAGE_FETCH`、`NEW_WINDOW`、`LOCAL_ESCAPE_HTML`、`LOCAL_PAGINATION`、`INIT_FUNCTION`），仅告警、不失败。
  - 报告所有依赖门禁脚本均 pass（auth-errors、csrf-tokens、PersonalList、DemandDetailRender 单元 / 行为测试套件均输出 PASS）。
- **git diff --check 当前基线**：`exit=0`（无错误）。

## 2. 路由与页面清单

来源：`internal/module/po/handler.go`、`internal/module/schedule/handler.go`、模板 `web/templates/po/*.html` 与 `web/templates/schedule/index.html`。

| 路由 | 模板 | page_js | page_css | 阶段 chip | 数据 priority | action 按钮 | 备注 |
|---|---|---|---|---|---|---|---|
| `GET /home` | `web/templates/po/home.html` | `personal-list.js`、`demand-detail-render.js`、`demand-detail.js`、`home.js` | `personal-workspace.css`、`home.css`、`demand-detail.css` | 顶部 `home-vs-mini-card`（10 阶段，含 `data-vs-status`）+ 右侧 `data-stage-target` focus-card | 表格 `<span class="inline-pri p*">`，home.js:205 由 pri 大写化小写后赋值；空值显示 `—` 套 `p3`（U01/U04） | 行内 "查看详情"（home.js:180/199）：业务需求 button 含 `data-demand-id`，研发需求/ID 直接 `target="_blank"` 跳禅道；A01 违规点 | 包含版本窗口卡片 |
| `GET /demands` (JSON) | —（API） | — | — | — | JSON 字段 | — | `/demands` 返回需求详情列表 |
| `GET /demands/:id/detail` (JSON) | —（API） | — | — | — | summary.priority | — | 详情 JSON 供 drawer |
| `GET /demands/:id` | `web/templates/po/demand-detail.html` | `demand-detail-render.js`、`demand-detail.js` | `demand-detail.css` | dd-flow 9 阶段（dd-flow-stage） | 顶部 `<span class="dd-tag">`（无 P1/P2/P3/P4 区分，U05） | dd-iconbtn（顶部 ↗ 新窗 / × 关闭） | 独立访问详情页 |
| `GET /todos` | `web/templates/po/todos.html` | `personal-list.js`、`demand-detail-render.js`、`demand-detail.js`、`todos.js` | `personal-workspace.css`、`todos.css`、`demand-detail.css` | 表格 `<td class="todos-col-stage">` 中 `<span class="relation-tag">`（无专用 stage chip） | `<span class="inline-pri p*">`（todos.js:183），todos.css 给 p1 红、p2 橙、p3/p4 绿、normal 灰（U01 异常） | `<a class="table-action-btn primary">` 行内，按 `item.url` 渲染；无 url 时为 `—` | 顶部 5 个 quick-chip（P1 chip 之一） |
| `GET /done` | `web/templates/po/done.html` | `personal-list.js`、`demand-detail-render.js`、`demand-detail.js`、`done.js` | `demand-detail.css`、`personal-workspace.css`、`done.css` | 无；表格 9 列仅记录业务动作 | 无（无 priority 字段） | `<button class="wb-done-detail-btn">`（done.js:259）开抽屉 | 顶部 4 KPI 卡 + 11 chip（JS 动态注入）+ 抽屉 `#doneDrawerMask` |
| `GET /notice` | `web/templates/po/notice.html` | `personal-list.js`、`notice.js` | `personal-workspace.css`、`notice.css` | 顶部 quick-chip（5 类：all/unread/action/abnormal/today） | 无 priority | `<a class="table-action-btn primary">` 去处理 / `<a class="table-action-btn">` 查看详情（notice.js:145-147） | — |
| `GET /follow` | `web/templates/po/follow.html` | `demand-detail-render.js`、`demand-detail.js`、`follow-drawer.js`、`follow.js` | `personal-workspace.css`、`follow.css`、`demand-detail.css` | 顶部 5 类 tab（全部 / 业务需求 / 项目周报 / 风险 / 测试单） | follow.js:279 仅按 P1–P4 大写匹配，缺省 `normal`；CSS 端 **无 `.follow-pri` 规则**（U05，dead class） | `<button class="follow-btn">`（查看详情/取消关注，follow.js:299-300） | 双视图：项目周报视图（pw-）、业务需求视图（follow-）；抽屉 `#pwDrawerMask` |
| `GET /board/demand`、`GET /board/task` | `web/templates/po/workboard.html` | `demand-detail-render.js`、`demand-detail.js`、`workboard-issue.js`、`workboard-modal.js`、`workboard.js` | `board.css`、`demand-detail.css`、`workboard-addon.css` | 需求树 5 列 stage-card（U03：`.po-board .priority` 固定红色，无 P1–P4 区分） | workboard.js:65 `priTag()` 仅输出 `<span class="priority">` 单 class，所有优先级同色（U03）；workboard.js:674 `drawerPriClass()` parseInt 后 P1 红 / P2 橙 / 其余灰（U04） | 模态：`openKanbanCreateModal('issue' / 'task')`、`openKanbanTeamModal()`；任务看板就绪/进行中/已完成 | 顶部 KPI stats、`#taskDrawer`、`#issueDrawer` |
| `GET /schedule` | `web/templates/schedule/index.html` | —（详见 schedule JS 套件） | `schedule.css`、`schedulelist.css`、`schedulewindow.css`、`schedulemodal.css`、`schedulefilter.css`、`scheduleintegrated.css` | `.stage-tag` + `--draft / --final`（schedulelist.css:341-359） | 仅 `.pri-tag.p0/p1/p2`（U02：`formatPriority` 输出 p1–p4 但 CSS 缺 p3/p4 规则） | `.action-btn` 一族 | 排期工作台首页（版本窗口 + 业务/独立研需列表） |

补充：

- **板内 draw 触发器（plan §3 A01 关键点）**：`web/static/js/po/demand-detail.js:188-208` 注册全局 `click` 委托，匹配 `[data-demand-id]` 或 `[data-open-demand-detail]`。home.js:189/215 在 `<button>` 与 `<a class="table-title-link">` 上都打了 `data-demand-id`，导致 ID 与标题点击都被 draw 吞掉，无法跳禅道原始详情。
- **与现有 follow.html 资源**：项目周报视图的 4 KPI、过滤器、7 列表格与侧滑 `#pwDrawerMask` 由 `follow-drawer.js` 接管。

## 3. 优先级规则现状

> 颜色按现状记录，**未做合并 / 删除**。

| 文件 | 选择器 | P1 | P2 | P3 | P4 | 备注 |
|---|---|---|---|---|---|---|
| `web/static/css/po/home.css` | `.po-home .inline-pri.p1` / `.p2` / `.p3` / `.p4` | `#fee2e2 / #b91c1c` | `#fef3c7 / #b45309` | `#f1f5f9 / #64748b`（p3、p4 共用） | 同 p3 | 高度 18px、字号 10px、字重 700、`ui-monospace`。home.css:101-108 |
| `web/static/css/po/todos.css` | `.po-todos .inline-pri.p1` / `.p2` / `.p3` / `.p4` / `.normal` | `#fee2e2 / #b91c1c` | `#ffedd5 / #c2410c` | `#f0fdf4 / #15803d`（p3、p4 共用绿） | 同 p3 | p3/p4 绿色，与 home/todos/page 其它文件不一致（U01）。todos.css:104-130 |
| `web/static/css/components/table.css` | `.table .priority-tag.P1` / `.P2` / `.P3` / `.P4` | `--color-danger-fill` (`#dc2626`)、`--color-text-on-accent`（实色底） | `--color-warning-fill`（`#f5cd2d`，实色） | `--color-info-fill`（`#2563eb`，实色） | `--color-surface-muted` / `--color-text-secondary` | 实色底 + 白字；与浅底标签体系不统一（U01）。table.css:405-441 |
| `web/static/css/schedule/schedulelist.css` | `.pri-tag.p0` / `.p1` / `.p2` | `var(--red)` | `var(--orange)` | **无规则** | **无规则** | schedule backend `formatPriority` 输出 `p1`–`p4`（bizdemand_view.go:141-151，>4 截到 4），CSS 仅 p0/p1/p2；p3/p4 退化默认色（U02）。schedulelist.css:308-324 |
| `web/static/css/po/board.css` | `.po-board .priority` | 固定 `var(--red-bg) / var(--red) / var(--red-bd)` | 同左 | 同左 | 同左 | 所有优先级同红色（U03）。board.css:134 |
| `web/static/css/po/board.css` | `.drawer-task-pri.p1` / `.p2` / `.p3` | `var(--red)` 系 | `var(--orange)` 系 | `#f1f5f9 / #cbd5e1 / var(--t2)` | 无 | 任务抽屉内有 P3 但缺 P4；统一性弱 |
| `web/static/css/po/demand-detail.css` | `.dd-tag`（统一基础）、`.dd-tag.blue` / `.green` / `.red` | — | — | — | — | 无 P1/P2/P3/P4 区分，priority 显示为统一灰底（U05）。demand-detail.css:51-58, 269-271 |
| `web/static/css/po/follow.css` | `.follow-pri`（JS 引用） | **未定义** | **未定义** | **未定义** | **未定义** | follow.js:279-292 输出 `.follow-pri .p1/.p2/.p3/.p4/.normal`；CSS 端完全缺失（U05 / dead class） |
| `web/static/css/po/personal-workspace.css` | — | — | — | — | — | 文件仅消费 token `--po-bg / --po-t1`，未涉及 priority（U07 证据：本任务不动 personal-workspace 共享样式） |
| `web/static/css/po/done.css`、`po/notice.css`、`po/homecompact.css`、`po/workboard-addon.css` | — | — | — | — | — | 全部无 priority 规则 |
| `web/static/css/layout/variables.css` | 全局 token | — | — | — | — | 仅 `--po-red / --po-red-bg / --po-orange / --po-orange-bg / --po-blue / --po-blue-bg / --color-danger-fill / --color-warning-fill / --color-info-fill` 等可复用；**无 `--po-priority-*` 或 `--color-priority-*` token**。variables.css:117-129 |

## 4. 对象类型规则现状

> 仅记录相同语义对象在不同页面的视觉差异。

| 文件 | 选择器 | 业务需求 | 业务子需求 | 研发需求 | 备注 |
|---|---|---|---|---|---|
| `web/static/css/po/home.css` | `.po-home .type-pill.story` / `.demand` | `.type-pill.demand`（推断默认蓝/灰，home.css:94） | — | `#faf5ff / #7e22ce`（紫） | home.js:210-212：`研需` 紫、`业需` 默认色 |
| `web/static/css/po/board.css` | `.type-biz` / `.type-child` / `.type-rd` / `.type-rd-indep` / `.type-task` / `.type-unknown` | `#fff7ed / #fed7aa / #c2410c`（橙） | `#f5f3ff / #ddd6fe / #5b21b6`（紫） | `#eef3ff / #bfdbfe / #1e40af`（蓝/海军蓝） | board.css:128-133；与 home 紫色研需冲突（U06）。研发需求在排期侧则用红色（见下） |
| `web/static/css/schedule/schedulelist.css` | `.schedule-type-badge--biz` / `--sub` / `--rd` | `#eff6ff / #2563eb / #dbeafe`（蓝） | `#f5f3ff / #7c3aed / #ede9fe`（紫） | `#fef2f2 / #dc2626 / #fee2e2`（红） | schedulelist.css:245-261；研发需求红色与 home 紫色冲突（U06） |
| `web/static/css/po/notice.css`、`po/done.css`、`po/follow.css` | — | — | — | — | 通知/已办/关注表均不显式带对象类型 chip |
| `web/static/css/po/personal-workspace.css` | — | — | — | — | 共享样式未定义对象类型色板 |

## 5. 操作按钮现状

> 类型 = `内` 表示内部页面/抽屉（不新开窗）、`外` 表示 `target="_blank" rel="noopener noreferrer"` 跳禅道原始详情。

| 页面 | 触发器 | 类型 | URL 模式 | 何时禁用 | 备注 |
|---|---|---|---|---|---|
| `/home`（home.js:180/199） | 表格行末 `查看详情`（业务需求 button + `data-demand-id`） | 内 | `/demands/:id/detail`（drawer） | item.url 空 → 渲染 `<span class="home-unavailable">暂无详情</span>`；研发需求直接渲染 `<a target="_blank">` 跳禅道 | A01：业务需求 title 链接被全局 draw 委托吞掉，无法直接进禅道 |
| `/home`（home.js:215） | `table-title-link` 含 `data-demand-id` | 内外冲突 | `<a target="_blank" href="zentaoUrl">` 但被 demand-detail.js 全局委托 `e.preventDefault()` 拦截 → 实际进入 drawer | — | U04 + A01 核心证据 |
| `/todos`（todos.js:188） | `table-action-btn primary` 链接 | 外 | `item.url`（禅道 URL） | `item.url` 空 → 渲染 `—` | action label 由 todos.js 派生（基于 `action`/`stage`） |
| `/done`（done.js:259） | `wb-done-detail-btn`（按 actionId） | 内 | `/done/detail/:actionId` → 抽屉 `#doneDrawerMask` | — | 行内 "查看记录"，抽屉底部有 "在禅道打开对象 ↗"（done.html:135） |
| `/notice`（notice.js:144-149） | `table-action-btn primary`（去处理） / `table-action-btn`（查看详情） | 外 | `item.url`（禅道 URL） | `item.url` 空 → `<span class="notice-no-op">—</span>` | "去处理" 在 `item.needAction === true && item.url` 时出现 |
| `/follow`（follow.js:299-300） | `follow-btn`（查看详情 / 取消关注） | 内 / 内 | drawer + `PUT /follow/demand/:id` | — | "查看详情" 走 drawer，"取消关注" 走 JSON 接口 |
| `/board/...` | 顶部 `.action` 一族 | — | `openKanbanCreateModal(...)`、`onclick="location.href='/schedule'"` 等 | — | 创建模态：`+ 提问题 / + 建任务 / 调整成员` |
| `/schedule` | `.action-btn` | — | 业务：`/demands/:id/scheduling`（独立研需：`/stories/:id/scheduling`） | — | 行内 "维护任务 / 排期" |

观察：

- 全站 action 按钮尺寸、高度、padding 不统一（home/todos 的 `table-action-btn` 24×4 边，done/notice 的 `table-action-btn` 26×8 边，board 的 `.action` 28×11 边，schedule 的 `.action-btn` 各异）——**plan §3 同类 UI 公共尺寸** 待统一。
- 全局 `[data-demand-id]` 点击委托在 demand-detail.js:188-208 内部 regex `US\d+|\d+` 校验，未区分业务需求 vs 研发需求；同时 home.js 在研发需求链接上根本不带 `data-demand-id`，所以理论可跳禅道；但只要继承业务需求的相同 ID 形态，仍可能被吞。

## 6. 共享组件现状

- **PersonalList (`web/static/js/po/personal-list.js`)**
  - 提供：`createController`（fetch + 时序保护 + 401/403/500 区分 + AbortController）、`renderPagination`（最多 7 槽位 + 省略号 + 每页条数）、`loadPageSize / savePageSize`（localStorage 白名单）。
  - 消费者：`home.js:278`、`todos.js`、`notice.js`、`done.js`、`follow.js`。
  - 状态：成熟（make check 中 PASS 全部用例）；不直接持有 priority 渲染。
- **demand-detail 抽屉 (`web/static/js/po/demand-detail.js` + `demand-detail-render.js`)**
  - 注册：全局 `document.click` 委托匹配 `[data-demand-id]`、`[data-open-demand-detail]`、`.demand-grid .node-title-line`（含 `.code` 文本 US 数字匹配）。
  - 渲染：5 Tab（overview/requirement/execution/delivery/history）+ 9 阶段价值流 + 关系链。
  - 状态：成熟，独立页 `demand-detail.html` 已具备直访能力。
- **排期入口**
  - 路由（已注册）：`GET /schedule/demands/:id/scheduling`、`POST /schedule/demands/:id/save-scheduling`、`GET /schedule/stories/:id/scheduling`、`POST /schedule/stories/:id/save-scheduling`（handler.go:108-111）。
  - 能力：业务 vs 独立研发 两套独立入口已存在；权限校验使用 `perm.ScheduleList`/`ScheduleUpdate`。
  - 入口缺失位置：home.js 没有排期按钮、workboard.js 仅跳 `/schedule`，未传 ID。
- **其它**
  - `web/static/js/po/workboard-modal.js`、`workboard-issue.js`：看板创建问题/任务/调整团队弹窗，独立于抽屉体系。
  - `web/static/js/po/html-sanitize.js` + `web/static/module/po/html_sanitize.go`：新 untracked 文件，不在本任务观察焦点。
  - `web/static/js/po/follow-drawer.js`：项目周报抽屉，与业务需求 `follow.js` 视图共享 `#pwDrawerMask`。

## 7. 阻塞与未明契约

按 PLAN §4 列举的 4 项业务契约，逐项给出当前仓库能 / 不能确认的事实证据。

### §4-1 审批对应受理 / 评审 / 主管部门审批

- **能确认的**：
    - PO ` /` handler 与 service 已定义 `valueStreamStages`（所有阶段名与中文 label，见 home.html 引用），包含 `accept` / `clarify` / `schedule` / `developing` / `testing` / `waitacceptance` / `acceptanced` / `publish` / `released`。
    - todo filter option `todo_review`（template todos.html:49）映射 "待受理 / 澄清"，但 label 合并。
  - **不能确认的**：
    - 审批（approve）与"受理"是否同一操作：未发现独立的 approve 路由或 capability (`grep -n "Approve\|approve\|审批"` 在 `internal/module/po/handler.go` 无匹配）。`stage:"accept"` 与 `action:"todo_review"` 字符串层耦合，并未落到独立 endpoint。
    - 主管部门审批人 / 状态流转：**无现存服务/路由**可直接证实（PLAN §4-1 的 "审批对应受理 / 评审 / 主管部门审批" 在当前阶段 5 之前仍是契约空缺）。

### §4-2 交付执行人 / 评价人 / 评价资格

- **能确认的**：
    - 表格中已展示 `责任人 / 负责人` 字段（home.js:229 `item.nextOwner || item.owner`）。
    - 已办页 (`/done`) 展示动作记录，可看见 "执行人 / 我做了什么"。
  - **不能确认的**：
    - 评价人字段是否存在、来自何表：`grep` 命中 `grep -rn "评价" internal/module/po` 仅有 done 列表的 `当前状态`；`Released` 阶段与 "评价反馈" 仍仅停留在 `valueStreamStages` 的字符串层。
    - 评价次数限制 / 业务与独立研发各自的数据来源：**无现存 endpoint 或表确认**。

### §4-3 评价反馈阶段待评价 vs 已评价 统计口径

- **能确认的**：
    - `released` 阶段作为最终阶段存在于 `valueStreamStages`。
    - 旧 CRCBWorkbench 中"以已有评价进入该阶段"是历史注释/原型描述，本任务**禁止以此为由在 Go 层改造**（PLAN §4-3）。
  - **不能确认的**：
    - 业务部门能否在"已评价"状态下继续做"待评价"操作：当前仓库**无任何 service/handler 暴露 evaluate 或 release-stage 计数**，也没有"已评价 / 待评价"区分口径。
    - 多人评价阻塞行为：**无证据**。

### §4-4 测试单关联与可见性

- **能确认的**：
    - `valueStreamStages` 含 `developing → testing`，但只是 UI 文案。
    - `follow` 页面 `data-tab="testtask"` 标 `二期`（phase2），暂未实现。
  - **不能确认的**：
    - 测试单（zt_testtask / zt_testcase）字段在 PO 模块中的关联 query：**当前 PO 模块 Repo 未发现测试单 join / 子查询**（`internal/module/po/repo*.go`）。
    - 批量取真实关联 / 跨对象 ID 套用风险：**无现存 endpoint** 触发该路径，所以也无法在测试套件中验证。

## 8. 与 WIP 冲突点

> stage 2+ 在动手前必须按 §1 文件清单复查 dirty/untracked；下列点为已知高风险。

- **Stage 2（公共优先级）的并行冲突**：
  - `web/static/css/po/home.css` (M)、`web/static/css/po/todos.css`（已存在 `inline-pri.p*` 但 untracked 中的 `todos.css` 状态需先 `git diff` 确认当前行号是否仍为 104 / 118）；同样，`web/static/css/po/board.css`、`.drawer-task-pri` 与 `demand-detail.css` 都在 M 列表。
  - `web/static/css/schedule/schedulelist.css` / `scheduleintegrated.css` 也已 M；stage 2 触及 `.pri-tag` 时必须先与当前 diff 合并。
  - `web/static/js/po/workboard.js:65 / 674`、`home.js:180-231`、`follow.js:279-292`、`demand-detail-render.js:48` 全部为 M，stage 2-5 修改 JS 优先级渲染时必须 `git diff` 对齐。
- **Stage 3（同类 UI）的并行冲突**：
  - `web/static/css/po/home.css`、`web/static/css/po/board.css`、`web/static/css/schedule/schedulelist.css` 同时被 M —— 对象类型色板（业务/业务子/研发）横跨这三处。
  - `web/static/css/po/done.css`、`web/static/css/po/follow.css`、`web/static/css/po/personal-workspace.css` 处于 M，需要在改动前比对 `git diff`。
- **Stage 4（导航与排期）的并行冲突**：
  - `web/static/js/po/demand-detail.js:188-208` 全局委托在 untracked 文件 `demand-detail-parent.js` / `demand-detail-richtext.js` 同时存在；如父 agent 已经迁移了全局委托入口，本任务的修改可能错位。**先 `cat internal/static/js/po/demand-detail-parent.js` 与 diff `demand-detail.js` 全部入口再做决定**。
  - `home.js:180-231` renderRow 已被改（M 66 lines）；改 renderRow 必须以最新版本为基准。
  - `internal/module/schedule/repo.go` 与 `internal/pkg/zentao/{zentao,url_test}.go` 已 M：URL builder 的 `ZentaoStoryViewURL` 等 API 已经新增；stage 4 跳转禅道应使用 builder，不要硬编码。
- **Stage 5（阶段办理）的并行冲突**：
  - `internal/module/po/service_detail*.go`、`internal/module/po/servicedone.go`、`internal/module/po/repodone.go`、`internal/module/po/handler_detail.go`、`internal/module/po/handler.go` 全部 M；写权限/authorize 路径同时在改动；**stage 5 在落 action_key / workbenchUrl / zentaoUrl 之前必须逐文件看 diff**。
  - `internal/module/po/form.go`、`form_detail.go`、`formnotice.go` M；表单提交链路上若有变更需对齐。
  - `internal/module/login/service.go` M；可能影响认证 context。
- **不可触碰的并行包**（与本任务无关，必须保留原状）：
  - 文档：shared-database-optimization、architecture-review、personal-pages-v104、operations/dev-vm-topology、ui-actions-unification 自身。
  - 新建 untracked Go/JS/CSS（PO 工作台增量 feature）：`form_done / form_weekly / handler_weekly / home_focus_repo / home_focus_test / html_sanitize / html_sanitize_test / kpi_blocked_test / notice_mark_all_test / po_work_scope / repo_weekly / repo_weekly_enrich / repodone_authz / repodone_enrich / service_detail_authz / service_detail_authz_test / service_detail_exec / service_weekly / service_weekly_util / servicedone_authz_test / home-focus.test.js / html-sanitize.test.js / notice-filters.test.js / workboard-addon.css / demand-detail-parent.js / demand-detail-richtext.js / follow-drawer.js / html-sanitize.js / workboard-issue.js / workboard-modal.js / shell.css / homecompact.css` —— 本任务**不动**这些文件。
- **Quality baseline 风险**：`scripts/quality-baseline/file-length.tsv` (M)；stage 6 收口后若引入超过 500 行的首方文件，**必须先回退并按 quality.md §5 与 baseline 走治理流程**，不要借机绕过。