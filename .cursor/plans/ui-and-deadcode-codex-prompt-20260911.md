# Codex 提词：死代码清理 + PO 全局设计系统治理（合并版）

> 仓库：`workbench-claude-po`
> 分支：`release/po-integrate-main-202609`
> 来源：Gemini 死代码审查（已抽查）+ Areith UI 治理扫描（Theme Contract / button-system / UserPicker）
> 规范：`docs/engineering/frontend.md`（Theme Contract）、`docs/engineering/ui-audit-20260910.md`
> 全局按钮：`web/static/css/components/button-system.css`（`layout/base.html` 已引入）
> 全局选人：`window.initUserPicker`（`web/static/js/ui.js`）
> 本地验：`WORKBENCH_MODE=dev WORKBENCH_APP_ADDR=:8090 WORKBENCH_SESSION_COOKIENAME=wb_session_po_8090`，账号 `<本地测试账号>` / `<本地测试密码>`
> **不要 push**；**不要改 `docs/PRD/`**；**不要动交付弹窗**（`po_deliver_modal.html` / `po-deliver.js` / `po-deliver.css` 由另一条 Codex 任务独占）。
> 读完先输出「将改文件清单 + 风险 + Track 顺序」，再动手；每 Track 做完自检验收勾选。

本提词两大部分互相独立，可分 commit，但**同一次任务内都要做完**（除非某条被验收方口头砍掉）：

| 部分 | 性质 | 目标 |
|------|------|------|
| **Part I 死代码** | 删/收敛不可达代码 | 减维护面、消竞态遗留 |
| **Part II 设计系统** | 视觉/组件对齐全局 | token、按钮、选人、去页面 dark 拷贝；**零产品行为变更** |

---

## 0. 明确排除（勿改）

- 发起交付弹窗全链路：`web/templates/components/po_deliver_modal.html`、`web/static/js/po/po-deliver.js`、`web/static/css/po/po-deliver.css`
- 催办业务逻辑（`urge.js` 收件人解析、预览文案）——仅允许 CSS 去 dark 拷贝时触碰 `po-urge.css`
- 禅道 PATH_INFO / 外链新标签 / 星标 CSRF `appFetch` / 提测四步**产品流程**（CSS token 化可以动样式）
- Admin `web/templates/components/ui/*`（部门/菜单仍在用，**不是**死代码）
- `.wip/demand-edit/`（WIP 停放，勿删勿提交）
- `web/templates/layout/scripts.html`：`base.html` 仍 `{{ template "scripts" . }}`，这是**空扩展槽**，**不要删**（Gemini 称死代码不准确）

---

# Part I — 死代码清理

## I-A. 删除 `SaveDemandFollow`（P0，安全）

### 现状（已核实）
- 生产关注/取关已走禅道 AJAX：`FollowDemandObject` / `UnfollowDemandObject`；压制 mailto 走 `EnsureDemandUnfollowed`。
- `SaveDemandFollow`（`internal/module/po/repofollow.go`）仅剩 `repo_rw_test.go` 引用；含旧式 SELECT-before-INSERT 竞态。
- `RepoSaveDemandFollowReq` 仍被 `EnsureDemandUnfollowed` 使用 → **保留类型，只删方法**。

### 必做
1. 删除 `Repo.SaveDemandFollow` 方法体与注释。
2. 改写/删除 `TestSaveDemandFollow_UsesWriteDB`：改为断言 `EnsureDemandUnfollowed`（或等价写入路径）走 writeDB；勿再调用已删方法。
3. 全文 `rg SaveDemandFollow` 仅允许出现在历史注释外的零调用（或仅 Ensure 相关 Req 名）。

### 验收
- [ ] `rg 'func \(r \*Repo\) SaveDemandFollow' internal` = 0
- [ ] 相关 Go 测试通过；星标取关冒烟仍 OK（勿回归到本地 zt_starinfo 直写）

---

## I-B. 下线「承建团队树」全套（P0，前端调用=0）

### 涉及（整条删）
- 路由：`GET /follow/project-weeklies/teams`、`GET /watches/project-weeklies/teams`（`handler.go`）
- Handler：`ProjectWeeklyTeams`（`handler_weekly.go`）
- Form：`ProjectWeeklyTeamsReq/Resp`、`ProjectWeeklyTeamNode`（`form_weekly.go`）
- Service：`ProjectWeeklyTeams`（`service_weekly.go`）
- Repo：`FindProjectWeeklyTeamLeaves`（`repo_weekly.go`）
- Util：`buildProjectWeeklyDeptTree`、`collectDeptIDsFromPaths`、`ensureDeptInTree`（`service_weekly_util.go`）及仅被其使用的私有类型/测试

### 注意
- **不要**误删仍在用的周报列表 `/follow/project-weeklies`、详情/history、`FindMineProjectWeeklyProjects`、scope=`mine|watched|all`。
- 若 `teamIDs` 查询参数仍被列表 API 接收但前端永不传：可保留参数兼容，或一并标注释「预留」；**本轮优先删树接口链路**，列表 filter 字段若仍被 service 使用则保留。

### 验收
- [ ] 前端/后端 `rg 'project-weeklies/teams|ProjectWeeklyTeams|FindProjectWeeklyTeamLeaves'` 无生产引用
- [ ] `/follow` 周报 Tab 列表/搜索仍可用

---

## I-C. 删除旧周报查询（P0）

1. **`FindFollowedProjectReports`**（`repofollow.go`）+ `servicefollow.go` 中 `FollowTabProjectReport` 分支 + form 中 `FollowTabProjectReport` 常量/校验（前端 Tab 仅 `demand`/`weekly`，不再走 `/follow/items?tab=project_report`）。
2. **`FindWatchedProjectWeeklyProjects`**（`repo_weekly.go`）：已是 `FindMineProjectWeeklyProjects(..., "watched", ...)` 包装且调用点为 0 → 删除包装，调用方（若有）直接用 `FindMine…`。

### 验收
- [ ] `rg 'FindFollowedProjectReports|FollowTabProjectReport|FindWatchedProjectWeeklyProjects' internal` = 0（或仅 CHANGELOG）
- [ ] 周报仍走 `/follow/project-weeklies`

---

## I-D. 清理待办/通知旧助手（P1）

- `demandTodoRelation`（`repotodo.go`）+ `repotodo_test.go` 中仅测它的用例 → 删函数；测试改为覆盖真实 SQL/Service 行为，或删除过时单测并注明。
- `sameNoticeDay`（`reponotice.go`）：若仅测试对照 SQL「今日」范围，可改为测试导出的范围 helper，或内联到测试文件；**勿删仍被生产调用的 queryQuickNoticeCounts 等**。

### 验收
- [ ] 生产 `.go` 中无未引用的上述两函数；测试绿。

---

## I-E. 价值流无过滤包装（P1）

- `FindRoleDemandIDs` / `FindScheduleStoryIDs` / `FindDeliverStoryIDs`（`repovaluestream.go`）生产已用 `*WithFilters`；当前主要被 `home_perf_integration_test.go` 当夹具。
- **做法**：删除无过滤包装，集成测试改为直接调 `*WithFilters` + 空 `DemandsReq{}`（或等价）。确认无其它调用后再删。

### 验收
- [ ] `rg 'FindRoleDemandIDs\(|FindScheduleStoryIDs\(|FindDeliverStoryIDs\(' internal` 仅剩 WithFilters 或测试已改完。

---

## I-F. 下线兼容路由 `/workbench/api/*` 与 `/watches/*`（P1，需谨慎）

### 现状（已核实）
- 前端 JS/HTML 对 `/workbench/api/`、`/watches/` 调用计数为 **0**；已迁 `/done/*`、`/follow/project-weeklies*`。

### 必做
1. 从 `handler.go` 删除 `wbApi := rg.Group("/workbench/api")` 及其下 done/watches 路由注册。
2. 在 commit message / 简短笔记写明：**已确认仓库内无调用**；若贵司有仓外脚本依赖需回滚该 commit。
3. 不要误删现行 `/done`、`/follow` 路由。

### 验收
- [ ] `rg '"/workbench/api"|Group\("/workbench|"/watches/' internal/module/po/handler.go` = 0
- [ ] `/done`、`/follow` 页面冒烟 OK

---

## I-G. 待办 `TodoListReq.tab` 死分支（P1）

### 现状
- 前端 `todos.js`：`objectType` 是唯一对象维度，**不传 `tab`**。
- 后端空 tab → 默认 `TodoTabAll`；`TodoTabApproval/Risk/Personal` 的「暂未接入」分支生产不可达。

### 必做（二选一，优先 A）
- **A（推荐）**：删除 `tab` 字段及 Tab 枚举校验死分支；列表只认 `objectType` + 现有其它 filter。若 Service 内部仍用 Tab 聚合，改为固定 `all` 语义或直接按 objectType 查询，**行为与现在「不传 tab」一致**。
- **B**：保留字段兼容但删除 Approval/Risk/Personal 错误分支，未知 tab 直接忽略为 all；并在 form 注释标明 deprecated。

### 验收
- [ ] 待办页分类芯片切换、计数、列表与改前一致
- [ ] 不传 tab 时无 4xx

---

## I-H. 前端静态死资源（P0）

1. **删除整个目录** `web/static/workbench/`（profile 副本；生产用 `web/static/{css,js}/po/po-profile.*`）。
2. **`html-sanitize.js` 收敛（不要当纯删除）**
   - 现状：无模板 `<script>` 加载；`demand-detail-richtext.js` 优先用全局 `HtmlSanitize`，否则 Node `require`，否则内联 fallback → 三套白名单。
   - 必做：在详情页（或所有会渲染富文本的 PO 页）于 `demand-detail-richtext.js` **之前**引入 `/static/js/po/html-sanitize.js`；确认浏览器走共享实现。
   - 有余力：删掉 `sanitizeRichTextFallback` 内联大段，仅保留「无 sanitizer 则 esc」兜底；单测 `html-sanitize.test.js` 继续测共享文件。
3. **`layout/scripts.html`：不动**（见排除项）。

### 验收
- [ ] `web/static/workbench/` 不存在
- [ ] 详情富文本页 Network/源码可见加载 `html-sanitize.js`；XSS 白名单行为不弱化
- [ ] 前端相关单测通过

---

# Part II — 全局设计系统治理（排除交付弹窗）

> 与 Part I 正交：Part I 删死代码；Part II 改「还在用但未对齐全局」的样式/选人。

## II-A. 选人统一 UserPicker（优先）

### 现状
- 澄清人员：已 `initUserPicker`
- 排期集成：RD / QD / 验收人仍 `initAutocomplete`（`scheduleintegrated.js` + `scheduleintegratedshared.js`）
- 交付验收人：另一任务负责 → **本轮勿改**

### 必做
凡「选人」改 `initUserPicker`；产品/版本等非人员仍 `initAutocomplete`。item 形状保持兼容。

### 验收
- [ ] 排期三处选人不再直接 `initAutocomplete`
- [ ] 排期弹窗可搜姓名/工号；交付三文件 diff 为空

---

## II-B. 澄清弹窗样式并入全局

`web/static/css/po/demand-clarify.css` ≈ **95 处纯硬编码色**；footer 私有 `clarify-btn-*`。

1. 颜色 → `variables.css` semantic token；禁止新增 `--dark-*` / `--clarify-dark-*`。
2. Footer/行内按钮 → `action-btn` / `action-btn primary`（对齐催办 / button-system）；优先改模板 class。
3. **禁止**新增 `html[data-theme="dark"] .clarify-…` 拷贝。
4. JS/接口/UserPicker 不动。

### 验收
- [ ] 澄清 CSS 纯 `#hex` 接近 0
- [ ] Light/Dark 澄清弹窗可读；保存澄清冒烟 OK

---

## II-C. 提测弹窗样式并入全局

`testtask.css` 私有 `--tt-*` hex + `po-testtask-btn`。

1. `--tt-*` 映射到全局 token（如 `--tt-primary: var(--color-primary)`），清除页面内散落 `#fff` 等。
2. Footer 优先 `action-btn`；或保留类名但样式体 token 化并对齐 button-system。
3. 不改四步流程/API。
4. 禁止页面级 dark 拷贝。

### 验收
- [ ] 提测弹窗 Light/Dark 主色与全局 primary 一致；步进/草稿/提交不回归

---

## II-D. 去掉六页 `html[data-theme="dark"]` 拷贝（Theme Contract §5）

| 文件 | 约 dark 块 |
|------|-----------|
| `follow.css` | 46 |
| `wb-priority.css` | 25 |
| `po-urge.css` | 18 |
| `home.css` | 15 |
| `notice.css` | 12 |
| `done.css` | 5 |

方法：缺的语义补进 `variables.css` 的 light/dark **token 层** → 删除页面拷贝选择器。`wb-priority` 已有 `--po-pri-*` 则删冗余 dark。`follow` 的 `pw-action-btn` token 化，能并 `action-btn`/`table-action-btn` 则并。禁止 `filter: invert`。

### 验收
- [ ] 六文件 `rg 'html\[data-theme=.dark.\]' …` 为 0（极少数例外须注释+报告列出）
- [ ] `/follow` `/home` `/done` `/notice` + 催办 Dark 无白底残留；优先级色语义不变

---

## II-E. 个人资料按钮泄漏

`po-profile.css` 无作用域重写全局 `.action-btn`（靛蓝 `#6366f1`）。

1. 删除裸 `.action-btn` 重定义，改用 button-system。
2. 若需微调，必须 `.po-profile .action-btn`，且 token 对齐全局 primary。

### 验收
- [ ] `po-profile.css` 无顶层裸 `.action-btn {`
- [ ] `/profile` 按钮与全站一致

---

## II-F. 小脏点（有余力）

1. `demand-detail-render-execution.js` 内联 `style="color:#8a99ad…"` → class + token。
2. `schedule.css` 重复 `.batch-modal .action-btn`：能删则删，不确定则报告跳过。

---

# 提交切分建议

**Part I**
1. `chore(po): remove dead SaveDemandFollow; align follow write tests`
2. `chore(po): drop unused project-weekly teams tree API`
3. `chore(po): remove legacy followed project-report query helpers`
4. `chore(po): remove dead todo/notice helpers and unused value-stream wrappers`
5. `chore(po): drop unused /workbench/api and /watches compat routes`
6. `chore(po): simplify TodoListReq tab dead branches`
7. `chore(web): delete static/workbench duplicate; wire html-sanitize.js`

**Part II**
8. `fix(ui): schedule person fields use initUserPicker`
9. `fix(ui): clarify modal tokens + action-btn footer`
10. `fix(ui): testtask map private palette to semantic tokens`
11. `fix(ui): drop page-level dark selector copies`
12. `fix(ui): profile stop redefining global action-btn`

---

# 门禁与回报

### 门禁
- 波及 Go 包 `go test` 通过；相关前端单测（user-picker / html-sanitize / urge 等）通过
- 编译重启 `:8090`，冒烟：关注星标、周报列表、待办芯片、已办、通知、澄清、提测、排期选人、资料；切 Dark 看 II-B/C/D
- **确认交付弹窗三文件无 diff**

### 回报验收方
- 各 Track 完成情况与 `rg` 清零前后计数（dark 选择器、澄清/提测 pure hex、SaveDemandFollow 等）
- Part I 删除的路由/方法清单
- 未做项与原因
- 明确声明：仓外若有脚本打 `/workbench/api` 或 `/watches`，需回滚 I-F

---

# 禁止事项

- 禁止借机做产品功能/文案/接口语义大改（Part II 零行为；Part I 只删不可达）
- 禁止新建第二套主题或 `--tt-dark-*` / `--clarify-dark-*`
- 禁止改交付弹窗；禁止 push；禁止 commit `docs/PRD/`
- 禁止删除 Admin `ui/*`、`.wip/`、`layout/scripts.html`
- 禁止把 `html-sanitize.js` 直接删掉而不接线到页面（必须先收敛加载）
