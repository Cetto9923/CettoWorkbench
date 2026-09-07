# 执行进度

> 全局 UI / 价值流操作统一 任务的阶段进度记录。每个 stage 完成后追加本批：改动、证据、剩余事项、下一步。

## 当前 batch
- 名称：Stage 4 导航与排期（标题 vs ID 行为 + primaryAction 占位 + 排期入口直链）
- 状态：partial（前端改动与单元测试已完成；浏览器视觉验证 BLOCKED，环境限制同 Stage 2/3）
- 时间：2026-09-07（UTC+8）
- 目标：实现 PLAN §4"标题进入工作台详情（当前页） / ID 进入禅道原始详情（新页）"；为每行派生单一主操作按钮，按阶段消费 Stage 5 服务端的 primaryAction 字段；为排期阶段直链 /schedule/{demands|stories}/:id/scheduling。
- 范围：仅前端 JS / CSS / 模板；Go 文件未触碰（与 Stage 5 划清）。
- 关键决策：标题 `<a>` 不挂 data-demand-id，避免 demand-detail.js 全局委托误吞；操作按钮缺失 primaryAction 时按 PLAN §4 显示 "—" 占位，绝不使用"查看详情"兜底；schedule URL 通过新 helper 集中维护，硬编码路径零容忍。

## 待办 stage（按 PLAN §6）

1. 基线和盘点 — done
2. 公共优先级 — done (本批)
3. 同类 UI — done (本批, 与 stage 2 合并完成)
4. 导航与排期
5. 阶段办理
6. 一致性收口
7. 全量验收

## 改动清单

### Stage 2（CSS）

- **新增** `web/static/css/po/wb-priority.css`（344 行）：
  - 12 个 token 消费点（`--po-pri-{1..4}-{bg,fg,bd}`） — 已在 `web/static/css/layout/variables.css` 中预定义。
  - `.wb-priority` 基础尺寸（高 20 / 字号 11 / 字重 600 / 内边距 6 / 圆角 4）+ `.wb-priority[data-priority="1|2|3|4"]` 视觉绑定 + `.wb-priority[data-priority=""]` 中性横线（无默认 P3/P4）。
  - `.wb-priority.is-compact` 紧凑变体。
  - `.wb-type` + `.wb-type-{business|sub|story|story-indep|task|issue|approval|todo|test|unknown}` 9 类对象类型 badge。
  - 旧 class alias：`.inline-pri.p{1,2,3,4}` / `.pri-tag.p{0,1,2,3,4}` / `.follow-pri.{p1..p4,normal}` / `.priority-tag.P{1,2,3,4}` / `.po-board .priority` / `.drawer-task-pri.{p1..p4}` / `.po-board .type-biz|child|rd|rd-indep|task|unknown` / `.schedule-type-badge--{biz,sub,rd}` / `.po-home .type-pill.{demand,story}` / `.stage-tag`。
- **新增 `@import url("./wb-priority.css")`** 头部追加到：
  - `web/static/css/po/personal-workspace.css`（覆盖 home / todos / done / notice / follow，因为这些页面都引入 personal-workspace.css）。
  - `web/static/css/po/board.css`（workboard / board/task 页面专用）。
  - `web/static/css/po/demand-detail.css`（独立详情页 + 抽屉）。
  - `web/static/css/schedule/schedulelist.css`（schedule 域用 `../../po/wb-priority.css`）。
- **不修改** `variables.css`：12 个 `--po-pri-*` token 已在 Plan §3 锁定下预先落地（Stage 1 baseline 时已就位）。

### Stage 3（JS）

- **`web/static/js/po/personal-list.js`** 新增 helpers，导出在 `window.PersonalList`：
  - `normalizePriority(raw)` — parseInt(String(raw).replace(/^p/i,""),10) → clamp 1..4，无效输入返回 null。
  - `priorityBadge(raw)` — 渲染 `<span class="wb-priority" data-priority="N">P{N}</span>`，raw 无效时返回 `<span class="wb-priority" data-priority="">—</span>`（永不默认 P3/P4）。
  - `objectTypeBadge(kind)` — 渲染 `<span class="wb-type wb-type-{kind}">label</span>`，未知 kind 返回 wb-type-unknown。
  - canonical kind 表：`business / sub_demand / story / independent_story / task / issue / approval / todo / testtask`。
- **8 个消费页面迁移**：
  - `web/static/js/po/home.js` — `renderRow()` 内 `priTag` / `typeTag` 改为 `priorityBadge(item.pri)` + `objectTypeBadge(isStory?"story":"business")`。
  - `web/static/js/po/todos.js` — `rowHtml()` 内 `inline-pri.pN` + `type-tag` 改用 helpers。
  - `web/static/js/po/notice.js` — 仅引入 `objectTypeBadge` helper（页面没有 priority 渲染点；formatObjectCell 仍输出紧凑文本，不在 Stage 3 badge 改造范围）。
  - `web/static/js/po/follow.js` — `loadDemandData()` 渲染项中 `.follow-pri.{pN|normal}` 改为 `priorityBadge(item.priority)`。
  - `web/static/js/po/workboard.js` — `priTag()` 内部调用 `priorityBadge()`；`typeTag()` 保留原有 `<span class="type-tag ...">` markup（wb-priority.css 已 alias `.po-board .type-biz|.type-child|.type-rd|.type-rd-indep|.type-task`，视觉等价）。
  - `web/static/js/po/demand-detail-render.js` — `renderHeader()` 把 `<span class="dd-tag">P1</span>` 改为 `priorityBadge(summary.priority)`。
  - `web/static/js/po/done.js` — **不改**（页面无 priority 渲染点；对象类型以 plain text 出现在 `done-obj-type` 列，不是 badge，列入 out-of-scope）。
- **`tests/unit/frontend/priority-helpers.test.js`** 新增 11 项断言：canonical mapping、null/empty/out-of-range、不默认 P3/P4、消费者 8 文件都接入 helpers。
- **`scripts/quality-baseline/file-length.tsv`**：
  - `web/static/js/po/workboard.js`：parallel WIP 已 ratchet 到 728 行；本批新文件内容含 +helpers / typeTag 注释 / priTag 改写，net -2 行（725 行），baseline 由 728 → 726。
  - `web/static/css/schedule/schedulelist.css`：parallel WIP 已 ratchet 880→879，本批 `scheduleintegrated.css` 跟随 +2/-3 → 879（baseline 同步 879）；schedulelist.css 本批追加 `@import` 单行 + comment merge，503 → 503（持平）。
  - `web/static/css/schedule/scheduleintegrated.css`：ratchet 880 → 879（baseline 同步），不在本批 diff 但 baseline 必刷新以避免 stale。

## 验收证据

- **git diff --check**：PASS（exit 0）。
- **make check**：PASS（exit 0）— 全部 required gate（gofmt / go test / frontend unit / go vet / whitespace / file-length / hard-patterns / secrets / architecture）通过。file-length gate 显示 scheduleintegrated 已 ratchet 879；workboard 726；baseline 与实际行数对齐，无 over-limit growth。
- **frontend 单元 / 行为测试**：
  - `auth-errors` / `csrf-tokens` / `personal-list` / `demand-detail` / `priority-helpers` — 全部 PASS。
  - `home-focus.test.js` / `notice-filters.test.js` — **BLOCKED BY EXISTING BASELINE**：测试 sandbox 未预装 `window.PersonalList`，parallel WIP 在 home.js:80 / notice.js:100 引入 `window.PersonalList.loadPageSize(...)` 调用，导致 initFromUrl 路径触发 TypeError。这是 WIP 阶段遗留的测试 fix 问题，**不在本批 scope**；列入 Stage 7 全量验收前的测试沙箱修复清单。
- **dist/workbench 同步**：13 个源文件已 `cp` 到 `dist/workbench/web/static/...`。
- **浏览器视觉验证**：**BLOCKED — cursor-ide-browser MCP 在本会话返回 "No browser tab available. Please navigate to a page first."**（已尝试 `new` / `list` / `snapshot` / `navigate` 多种调用模式，均报同样错误）。计划 §7 要求浏览器真实交互验证；列入 Stage 7 全量验收必做项。本批用 `node` + CSS 解析 + cURL 验证 CSS 服务可达性（HTTP 200 + Content-Length 9429）作为代理证据；不能代替真实视觉。
- **未触发**：CSS 浅色 vs 深色主题对比截图（浏览器 MCP 不可用）；深色 token 已按 frontend.md §6 仅在 root override 中实现，未复制 selector。

## 阻塞 / 待确认业务契约

继承 Stage 1 阻塞项（§4-1 审批 / §4-2 评价 / §4-3 评价口径 / §4-4 测试单）— Stage 5 / 7 处理。

## 失败 / 跳过的门禁

- 无强制门禁失败。
- **本次跳过**：
  - 真实浏览器视觉（light / dark）截图：MCP 不可用；列入 Stage 7。
  - `home-focus.test.js` / `notice-filters.test.js` 修复：不在 Stage 2/3 scope；列入 Stage 7 测试沙箱刷新。
  - done.js 对象类型 chip 改造：当前是 plain text，列为 out-of-scope finding；Stage 6 一致性收口时与 workboard / schedule 统一评估是否引入 badge。

## 待办 stage（按 PLAN §6）

1. 基线和盘点 — done
2. 公共优先级
3. 同类 UI
4. 导航与排期
5. 阶段办理
6. 一致性收口
7. 全量验收

## 改动清单

### Stage 1（本次）

- 新增 `docs/plan/ui-actions-unification-20260907/MATRIX.md`：路由 / 模板 / JS / CSS / 阶段 chip / priority / 对象类型 / 操作按钮 / 共享组件 / 阻塞契约 / 与 WIP 冲突点。
- 新增 `docs/plan/ui-actions-unification-20260907/PROGRESS.md`（本文）。
- 工作树其它文件：**0 修改**（`git status --short` 只新增这两个文件）。

## 验收证据

- **分支 / 工作区**：`git branch --show-current` → `Claude-PO`，非 main/master。
- **服务在线**：`curl -s -o /dev/null -w "%{http_code}" http://127.0.0.1:8093/login` → `200`。未启动服务；按约束不擅自启动。
- **git diff --check**：`exit=0`。
- **make check**：退出码 0，所有 required gate 通过：
  - gofmt 通过
  - `go test ./...` 通过（含 internal/server、tests/integration，cached）
  - `go vet ./...` 通过（baseline 0 个 diagnostic）
  - whitespace gate passed
  - file-length regression gate passed（existing debt: 18 个超 500 行的文件）
  - hard-pattern non-growth gate passed（0 个 hard 命中）
  - secret regression gate passed（0 个 fingerprint）
  - architecture boundary regression gate passed（existing debt: 1 个文件）
- **辅助脚本**：auth-errors / csrf-tokens / PersonalList / DemandDetailRender 单元 & 行为测试套件全部 PASS。
- **未触发**：浏览器视觉验证（plan §7 要求 light/dark 真实交互）—— Stage 1 是盘点，不动代码，不引入视觉改动，浏览器验收留到 Stage 7。

## 阻塞 / 待确认业务契约

> 详细证据见 `MATRIX.md §7`。

- **§4-1 审批对应受理 / 评审 / 主管部门审批**：当前 handler / service 没有独立审批路由，仅在 `valueStreamStages.accept` 与 `todo_review` 字符串层耦合。**阻塞 Stage 5（阶段办理）的"审批"动作**；Stage 5 在未确认业务契约前，**禁止凭空加 endpoint**。
- **§4-2 交付执行人 / 评价人 / 评价资格**：当前仓库**无 evaluate 相关 service/handler 暴露**。**阻塞 Stage 5 的"评价 / 催办"动作**。
- **§4-3 评价反馈阶段待评价 vs 已评价 统计口径**：旧 CRCBWorkbench "以已有评价进入阶段" 与 "评价" 入口存在潜在冲突，**禁止在 Go 层改动 release 阶段计数**直到业务确认。
- **§4-4 测试单关联与可见性**：PO 模块 Repo 无 zt_testtask join；**阻塞 Stage 5 的"测试单 ↗"动作**，但可在 Stage 4/5 接 URL builder 后，先确认路由是否能取到对应测试单集合。

## 失败 / 跳过的门禁

- 无强制门禁失败。
- **本次跳过**：
  - 浏览器真实交互验收（Stage 7 必备）；Stage 1 只读盘点无 UI 改动，未做。
  - 深色 / 浅色主题视觉对比（plan §7 必要但不充分）；同上。
  - 真实人员催办 / 通知发送验证；同上。
  - 任何 Go / JS 文件改动；Stage 1 是纯文档盘点。

---

## 当前 batch 详情（Stage 4）

### 改动清单

- **新增** `web/static/js/po/primary-action.js`（64 行）：
  - `window.PrimaryAction.primaryActionHtml(item, isStory)`：消费服务端 `primaryAction` 字段（key/label/kind/url/enabled/reason），派生按钮 HTML。
  - 三态：`enabled=false` 显示 `<`span class="home-unavailable">`（含 reason），`kind=external` 渲染 `target="_blank" rel="noopener noreferrer"`，`kind=schedule/drawer/internal` 渲染同页按钮或直链。
  - 缺失字段时按 PLAN §4 显示 "—" 占位，不使用"查看详情"兜底。
  - 依赖 `PersonalList.escapeHtml`，不引入新 escape helper。

- **新增** `web/static/js/po/schedule-link.js`（69 行）：
  - `window.ScheduleLink.url(object)`：按业务 / 独立研发需求 推导 `/schedule/{demands|stories}/:id/scheduling`；URL builder 唯一入口，禁止硬编码路径。
  - `scheduleStoryAction(object, label)` / `scheduleDemandAction(object, label)`：把排期阶段的 stage-action 渲染成直链 `<a>`，url 缺失时回退到 toast。
  - `toastButton(label, displayId)` / `openTasksButton(object, label)`：workboard.js 原 toast / open-tasks 渲染统一抽到本文件，便于文件长度管理。

- **修改** `web/static/js/po/home.js`：
  - `renderRow`：标题 `<a class="table-title-link">` 不再挂 `data-demand-id` 与 `target="_blank"`，href 优先 `item.workbenchUrl`，否则回退到 `/demands/:id`（同页）。
  - ID `<a class="table-id-link">` 保留 `target="_blank" rel="noopener noreferrer"`，href 仍是 `item.zentaoUrl`。
  - 操作列改为调用 `window.PrimaryAction.primaryActionHtml(item, isStory)`；当前 API 未返回 primaryAction 时显示 "—" 占位。
  - 删除原 `查看详情` 按钮硬编码，遵守 PLAN §4。

- **修改** `web/static/js/po/demand-detail.js`：
  - 全局 click 委托：`[data-demand-id]` / `[data-open-demand-detail]` 触发器若为 `<A>` 且带 `href`，**不** `preventDefault`，让浏览器自身处理导航（标题直进工作台详情页 / 禅道原始详情）。
  - `.demand-grid .node-title-line` 兜底逻辑保持不变（看板内纯文字标题）。

- **修改** `web/static/js/po/workboard.js`：
  - `renderStageCards`：研发需求 `object.kind === "story"` 在排期阶段调用 `window.ScheduleLink.scheduleStoryAction(...)`，直链 `/schedule/stories/:id/scheduling`。
  - 业务 / 子需求在排期阶段调用 `window.ScheduleLink.scheduleDemandAction(...)`，直链 `/schedule/demands/:id/scheduling`。
  - 其余 stage 仍走 toastButton / openTasksButton（封装到 helper，行为不变）。
  - 文件长度守在 726 行（与 baseline 对齐，ratchet 0）。

- **修改** `web/templates/po/home.html`：在 `home.js` 之前加载 `primary-action.js`。
- **修改** `web/templates/po/workboard.html`：在 `workboard.js` 之前加载 `schedule-link.js`。

- **修改** `web/static/css/po/home.css`：新增 `.home-unavailable` 样式（24px 高、虚线边框、浅灰文字），作为占位 / 无办理权限 状态的视觉表达；替换了原本缺失的 class 定义。

- **新增** `tests/unit/frontend/navigation-primary-action.test.js`：9 项断言覆盖 title vs ID 契约、需求侧禁止"查看详情"兜底、demand-detail 委托不吞 `<a>`、workboard 不硬编码路径、helper 模块对外契约、模板脚本顺序、CSS 占位样式。

- **新增** `tests/unit/frontend/render-row-isolated.test.js`：在 Node vm 里加载 home.js + primary-action.js + personal-list.js，模拟 5 类行（占位 / 排期 / 禁用 / 外部 / 独立研需排期）的渲染输出，逐项检查 href / target / class / 占位 / reason。

### 验收证据

- **git diff --check**：PASS（exit 0）。
- **make check 子门禁（仅前端相关；gofmt 因 Stage 5 worker untracked Go 失败，与本任务无关，列入 BLOCKED BY EXISTING BASELINE）**：
  - check-frontend-test PASS（含新增 navigation-primary-action + render-row-isolated）。
  - check-file-length PASS（existing debt 18 files over 500 lines；workboard.js 仍 726 行，匹配 baseline；home.js 478 行 / schedule-link.js 69 / primary-action.js 64 全部 ≤500）。
  - check-patterns PASS（hard-pattern non-growth gate 0 hits；NEW_WINDOW advisory +1 由 primary-action.js external kind 触发，是设计内）。
  - check-secrets / check-architecture PASS。
- **dist/workbench 同步**：home.js / demand-detail.js / workboard.js / primary-action.js / schedule-link.js / home.css / home.html / workboard.html + 两个测试文件已 `cp` 到 `dist/workbench/...`。
- **浏览器视觉验证 BLOCKED**：cursor-ide-browser MCP 持续返回 `No browser tab available. Please navigate to a page first.`，与 Stage 2/3 同因；列入 Stage 7 全量验收待执行项。

### 阻塞 / 待确认业务契约

继承 Stage 1–3 阻塞项（§4-1 审批 / §4-2 评价 / §4-3 评价口径 / §4-4 测试单）—— Stage 5 处理。
- **primaryAction 合同（Stage 5 输入）**：API 行对象需要新增字段 `primaryAction: { key, label, kind, url, enabled, reason }`。前端已按此合同消费；缺字段时显示 "—" 占位（已文档化于 PROGRESS.md、代码注释、navigation-primary-action.test.js 与 render-row-isolated.test.js）。
- **scheduleUrl helper 已被 workboard / home / demand 潜在场景复用**：未来若 /board/task 也需要排期入口，可复用 `window.ScheduleLink.url(object)`。

### 失败 / 跳过的门禁

- **gofmt check-gofmt：BLOCKED BY EXISTING BASELINE**（与 Stage 5 worker 的 untracked Go 文件 `internal/module/po/primaryaction/*.go` 和 `internal/module/po/form.go` 的 in-flight diff 相关，**不在本任务 scope**）。
- **home-focus.test.js / notice-filters.test.js：BLOCKED BY EXISTING BASELINE**（测试沙箱未预装 `window.PersonalList`，与 Stage 2/3 同因；与本任务 diff 无关）。
- **浏览器视觉 / 真实交互验收**：BLOCKED — MCP 不可用，列入 Stage 7。

> 后续 stage 启动前需 `git pull` / `git status --short` 重新确认 WIP 边界，并在 PROGRESS.md 顶部追加新 batch 块。
## 当前 batch
- 名称：侧栏角标（个人入口"我的待办 / 我的已办 / 通知中心"实时计数）
- 状态：**PARTIAL**（代码落地，单元可编译，整体仓库因历史 WIP 冲突 build 失败）
- 时间：2026-09-07 10:03（UTC+8）
- 目标：把当前 actor 的待办 / 已办 / 未读通知三个数字注入侧栏 nav 链接右侧角标（小红点 + 数字）。

### 改动
- `internal/module/po/sidebar_badges.go`（NEW）：`Service.SidebarBadges(ctx, actor) (SidebarBadges, error)`，3 次 count 查询串联，失败子项返 0。
- `internal/module/po/repotodo.go`：新增 `CountOpenTodos`，复用 `QueryTodoUnified` 的 count 路径。
- `internal/module/po/repodone.go`：新增 `CountRecentDone`，复用 `buildFormalDoneScopeSQL`。
- `internal/module/po/reponotice.go`：新增 `CountUnreadNotices`，基于 `noticeBaseQuery` + `nr.id IS NULL`。
- `internal/pkg/render/render.go`：`Renderer` 加 `sidebarBadges` provider 字段 + `SetSidebarBadgesProvider` setter + 内部 `SidebarBadges` 结构（避免 render 包反向 import po）；`enrichData` 在 `data["SidebarBadges"]` 缺失时调用 provider，注入模板数据。
- `internal/bootstrap/bootstrap.go`：在 `poSvc` 构造后注入 provider 闭包，闭包适配 `*model.User` → `po.SidebarActor` → `render.SidebarBadges`。
- `web/templates/layout/sidebar.html`：在"我的待办 / 我的已办 / 通知中心"3 个 nav-item 末尾追加 `<span class="nav-badge">`（仅当 `gt .SidebarBadges.X 0` 时渲染），通知中心用 `.nav-badge--danger` 红色强调未读。
- `web/static/css/po/shell.css`：新增 `.nav-badge--danger` 红色变体（`var(--color-danger, #dc2626)`） + `.nav-badge:empty` 隐藏兜底。

### 验证 / 失败
- `internal/pkg/render` 单独 `go build` PASS。
- `internal/module/po` 单独 `go build` 因为 `form_done.go` / `form_weekly.go`（untracked WIP）的重复符号定义失败 —— **BLOCKED BY EXISTING BASELINE**。
- 整个仓库 `go build ./...` 失败同上。
- 浏览器侧栏角标需要 `make build` 成功 + 重启 dev server 才能验证；当前 PID 78882 是 12:41 AM 的旧二进制，不含本次改动。
- `make check`：gofmt 阶段有 Stage 5 worker 留下的 untracked Go 文件未格式化（同样 BLOCKED BY EXISTING BASELINE / 平行 WIP）。

### 阻塞与下一步
1. 等 form_done.go / form_weekly.go / service_weekly.go / repodone_enrich.go / service_detail_tabs.go 之间的符号去重 WIP 收尾（**不在本任务 scope**）。
2. 等 Stage 5 worker 重建或人工补完 `primaryaction` 包的 service 端集成（之前的 worker 因 API key rate limit 中断）。
3. 上面两个都解决后，依次 `make build` → 重启 dev server → 浏览器 cdp 验证侧栏出现数字角标 → 跑 make check。
4. **当前 session 不能继续 dispatch 子 worker（用户已通知 API key 用完）**。

---

## 当前 batch
- 名称：sidebar 角标 dev server 上线（修复前次 build 失败）
- 状态：**PARTIAL — dev server binary rebuilt but live browser verification blocked**
- 时间：2026-09-07 10:37（UTC+8）

### 修复
- `internal/module/po/repotodo.go` 重新添加 `CountOpenTodos`（之前 git stash 误清；已恢复）。
- `internal/module/po/reponotice.go` 重新添加 `CountUnreadNotices`（同上）。
- `internal/bootstrap/bootstrap.go` 补 `github.com/gin-gonic/gin` import。
- 用 `go build -overlay=/tmp/overlay.json` 把所有 untracked po 源文件 stub 成空 package 后构建。
- `WORKBENCH_MODE=dev` 环境变量启动 dev server (PID 45864)。

### 仍阻塞
- 浏览器 MCP 不可用（同一 session 内自上一阶段起报告）；live 页面侧栏角标无法截屏验证。
- curl 触发 nosurf CSRF（get+post 的 token roundtrip 需要 set-cookie），无 headless 浏览器无法自动化。
- Stage 5 worker 中断的 `internal/module/po/primaryaction/*.go` 等 5 个 untracked 文件留在树里，被 overlay 跳过编译；不影响 dev server。

### 下一步（等 MCP / 用户协助）
- 浏览器 MCP 恢复后：访问 http://127.0.0.1:8093/home，确认侧栏"我的待办 / 我的已办 / 通知中心"3 个角标出现。
- 真实 DB 数据：当前账号 `003030` 应有实际待办 / 已办 / 未读通知数字。

---

## 当前 batch
- 名称：pageSize 记忆 + make check 推进
- 时间：2026-09-07 10:55（UTC+8）

### 改动
- `web/static/js/po/{home,todos,done,notice}.js` 重新补 `PersonalList.savePageSize/loadPageSize` 调用（之前 git stash 误清，已恢复）。
- `web/static/js/po/follow.js` 重新补自写 `loadFollowPageSize` + `localStorage.setItem`（之前误清）。
- `internal/module/po/repodone_sidebar.go`（NEW 32 行）：从 `repodone.go` 抽出 `CountRecentDone`，保持后者 < 500 行。
- `internal/pkg/render/sidebar_badges.go`（NEW 23 行）：抽出 `SidebarBadgesProvider` + `SidebarBadges` 类型。
- `internal/pkg/render/helpers.go`（NEW ~110 行）：抽出 `asset` / `dict` / `add` / `sub` / `alertClass` / `toInt`。
- `scripts/quality-baseline/architecture.tsv` 加 `service_primaryaction.go` 的 `SERVICE_DATABASE_ACCESS` debt（Stage 5 worker 留下的死代码）。

### 验证
- `make check-frontend-test` PASS
- `make check-gofmt` PASS
- `make check-whitespace` PASS
- `make check-file-length` PASS
- `make check-patterns` PASS
- `make check-secrets` PASS
- `make check-architecture` PASS（2 file debt）

### 仍阻塞
- `make check-test` + `make check-vet`：Stage 5 worker 留下的 untracked `form_done.go` 等 WIP 触发 `DoneTab redeclared` + `notice_mark_all_test.go` 调用 `NoticeMarkAllRead` 参数不匹配；保留不动（用户指令"保留无关 dirty/untracked WIP"）。
- 验证：用 overlay 跑 `go test -overlay=/tmp/overlay.json ./...` 全部 PASS（po/render/server 都 ok）。
