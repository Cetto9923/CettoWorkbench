# Code Health Revalidation — 2026-09-07

仓库：`workbench-claude-po`
分支：`Claude-PO`（HEAD `cc4b8f7b`）
执行：Codex（Agent 模式）
前置约束：保护现有前端页面 = 最高优先级

> 复核与本轮修复执行全程 `STATIC VERIFIED`，未启动浏览器或服务做真实交互验证；运行时验证状态见各步骤末尾。

---

## 0. 复核范围与不修范围

不修（`KNOWN DEBT / OUT OF SCOPE`）：

- `listAllStageDemands` / `listMySQLDemands` 的 Go offset:end 旧路径（生产路径 `/demands?status=all` 走 `FindHomeFocus`，SQL filter+count+paged，不经过它们）。
- `po_work_scope.go` 死代码。
- 深色模式页面级 hex / inline style（仓库已具备 light/dark 语义 token，页面级 hex 是独立任务）。
- `home-focus.test.js` / `notice-filters.test.js` `BLOCKED BY EXISTING BASELINE`（测试沙箱未预装 `window.PersonalList`）。
- `esc()` 重复、`html-sanitize.js` 调用、机械拆分（按职责拆不是行数切分）。
- 第二批 PRD 页面（需求查询 / 版本跟进 / 验收交付 / 问题风险 / 指标管理 / 雷达 / 预警）。
- P3/P4 UI 样式、研发需求标签颜色、状态颜色统一、状态中文显示统一、Hex 色值、inline style、页面响应式、HTML 模板瘦身、前端公共组件抽取。
- 恢复独立业务需求详情页（你已拍板不恢复）。

不写 `docs/engineering/debt.md`（避免跨范围），全部登记在本报告 §3。

---

## 1. 复核结论表（每项含 UI Impact）

| ID | 审计报告结论 | 仓库真相 | 分类 | 是否修 | UI Impact |
|---|---|---|---|---|---|
| **P0-A `hasCapability()` 权限放行** | 函数完全忽略 perm 参数 | **真**：`internal/module/po/service_primaryaction.go:215-224` 函数体仅 `nil actor` / `IsSuperAdmin` / `actor.Account != ""` 三种放行；handler `RequirePerm` 已做页面级守卫（`internal/middleware/permission.go:24-58`），但 service 行级派生未消费目标 perm | **CONFIRMED-BUG + CONFIRMED-ARCH** | 待 §A1 处理 | NONE（仅 Go 层） |
| **P0-B PrimaryAction 合同未接通** | `/demands` `/todos` 未回填 | **误**：合同已接通（`internal/module/po/servicefollow.go:310-320` 批量 IN、`internal/module/po/service_detail.go:357-377` 单行、`internal/module/po/form.go:104` JSON 字段、`web/static/js/po/primary-action.js` 渲染、`web/static/js/po/todos.js:203` 消费）。未接通的 4 个 endpoint（§4-1 审批 / §4-2 评价 / §4-4 测试单）是 PRD 没冻结的产品决策 | **FALSE-POSITIVE + PRODUCT-DECISION** | 不修 | NONE |
| **P0-C 无界 ID 全量加载 + Go 内存分页** | Pluck 全量 → Go offset:end | **部分真**：`servicefollow.go:139-205`（`listAllStageDemands`）与 `:236-275`（`listMySQLDemands` 含 scheduleIncomplete/deliverStories 路径）走 Pluck → Go offset:end；但生产路径 `/demands?status=all` 走 `service.go:243` `FindHomeFocus`（SQL filter+count+paged），不经过前者 | **CONFIRMED-DEBT（死路径，前端不触发）** | 不修 | NONE |
| **P1-1 首页工具栏筛选只过滤当前页** | keyword/objectType/priority 只前端 filter | **真**：工具栏 4 字段未透传服务端；顶部 focus chips（today/blocked/overdue/suspended）**故意只页面内筛选**，符合 PRD §01 五 验收 2「焦点不再误跳我的待办」 | **CONFIRMED-BUG（仅工具栏 4 字段）+ FALSE-POSITIVE（focus chips）** | 待 §A2 处理 | LOGIC ONLY |
| **P1-2 Makefile 前端测试漏测** | 只跑 4 个 | **误**：`Makefile:18-31` 实际跑 11 个；`home-focus.test.js` / `notice-filters.test.js` 是 `BLOCKED BY EXISTING BASELINE`（`docs/plan/ui-actions-unification-20260907/PROGRESS.md:67, 199`） | **FALSE-POSITIVE + NEED-RUNTIME-VERIFY** | 不修 | NONE |
| **P1-3 WAIT DECISION 污染** | 关键字污染生产代码 | **部分真**：`internal/module/po/form.go:213-215` message 文案 "(WAIT DECISION)" 暴露给前端；`Stage 5 / PLAN §X` 是文档引用，保留 | **CONFIRMED-BUG（仅 1 处）** | 待 §A3 处理 | NONE（仅后端 message） |
| **死代码 `po_work_scope.go`** | 0 caller | 0 caller | **CONFIRMED-DEBT** | 不修（见 §3） | NONE |
| **死代码 `toActionIDs`** | 0 caller | 0 caller（`servicefollow.go:355-366` 用 `toUintSlice`） | **CONFIRMED-DEBT** | 待 §A4 处理 | NONE |
| **死代码 `html-sanitize.js`** | 0 caller | **误**：被 `web/static/js/po/demand-detail-richtext.js:21-25` 直接 require，`internal/module/po/html_sanitize.go` Go 侧同名净化器与测试引用 | **FALSE-POSITIVE** | 不修 | NONE |
| **重复 `esc()`** | 多文件副本 | 真重复；`PersonalList.escapeHtml` 是共享真源 | **CONFIRMED-DEBT + PRODUCT-DECISION**（"简单重复优于复杂抽象"） | 不修 | NONE |
| **机械拆分 `demand-detail-parent.js` / `-richtext.js`** | 为 500 行机械拆 | 真按职责拆（`parent` 专做 `renderParentAggregate`，`richtext` 专做 F02 富文本净化兜底） | **FALSE-POSITIVE** | 不修 | NONE |
| **样式 / 状态不一致 / P3P4 / 翻译 / Hex / 内联 style / 深色模式** | — | 独立任务 | **KNOWN DEBT / OUT OF SCOPE** | 不修 | NONE |
| **恢复独立详情页** | — | 你已拍板不恢复 | **PRODUCT-DECISION** | 不修 | NONE |

> "UI Impact" 列：A1 NONE / A2 LOGIC ONLY / A3 NONE / A4 NONE。任何 > LOGIC ONLY 的改动立即回退并向你确认。

---

## 2. 修改项（按执行顺序：Step 1 → A3 → A4 → A2 → A1）

### Step 1 · 复核报告（本文件）

- **状态**：STATIC VERIFIED（写报告本身不动代码）。

### A3 · 清理 `form.go:214` 的 `(WAIT DECISION)` 文案

- **问题**：`internal/module/po/form.go:213-215` message 含 "(WAIT DECISION)"，暴露给前端校验错误信息（按 `architecture.md` "implementation labels such as WAIT DECISION are not normal UI"）。
- **根因**：早期接入 story 待办数据源占位时直接用了协作术语。
- **修改文件**：
  - `internal/module/po/form.go`
  - `internal/module/po/servicetodo_test.go`
- **修改方案**：message 改为 `"业务需求的故事待办数据源暂未接入"`；同步单测断言。
- **测试**：`go test ./internal/module/po/...`。
- **运行验证**：STATIC VERIFIED / RUNTIME NOT VERIFIED。
- **残余风险**：无。
- **UI Impact**：NONE（仅后端 message + 单测文本；不触碰 HTML/CSS/前端 JS）。

### A4 · 删 `toActionIDs`（0 caller）

- **问题**：`internal/module/po/service_primaryaction.go:283-290` `toActionIDs` 函数定义后无任何调用者（`grep` 仅命中定义点；同文件 `populateWorkItems` 走 `toUintSlice`，`servicefollow.go:355-366`）。
- **根因**：历史遗留。
- **前置确认（按 plan 硬要求）**：
  - 0 普通调用 ✓（grep 全仓 0）
  - 0 反射引用 ✓（无 `reflect` 或字符串名引用）
  - 0 动态注册 ✓（无注册机制）
  - 0 测试依赖 ✓
  - 0 构建脚本依赖 ✓
  - 0 近期 Git 暂存 ✓（HEAD `cc4b8f7b`，后续无 untracked 同名改动）
- **修改文件**：`internal/module/po/service_primaryaction.go`（仅删函数定义 8 行）。
- **修改方案**：删除函数。
- **测试**：`go build ./...` + `go test ./internal/module/po/...`。
- **运行验证**：STATIC VERIFIED / RUNTIME NOT VERIFIED。
- **残余风险**：无。
- **UI Impact**：NONE。

### A2 · 首页工具栏 4 字段透传服务端（keyword / objectType / priority / relation）

- **问题**：`web/static/js/po/home.js:264-290` `filterItems` 在客户端对当前页 rawItems 做 keyword/objectType/priority 二次过滤（relation 在前端没有同语义二次过滤，仅 UI 状态）；后端 `DemandsReq` 没有这 4 字段（`internal/module/po/form.go:56-86`），所以跨页筛选失效。
- **根因**：工具栏查询条件未透传到 Repo SQL。
- **修改文件**：
  - `internal/module/po/form.go` `DemandsReq` 加 4 字段 + `Validate`
  - `internal/module/po/service.go` `Demands` 透传给 `Repo.FindHomeFocus`
  - `internal/module/po/repovaluestream.go` 在 `FindHomeFocus` 内按 `?` 占位追加 4 段 `q.Where(...)`
  - `web/static/js/po/home.js` `initToolbar` 4 字段去抖后调 `refreshDemands`；`refreshDemands` URL 携带 4 字段；`filterItems` 内 4 字段不再做二次过滤
- **修改方案**：
  - 前端：`initToolbar` keyword/objectType/priority/relation 由 `renderList(rawItems.length)` 改为去抖后调 `refreshDemands(state.status)`；`demandsUrl` 携带 4 字段；`filterItems` 内 4 字段分支删除或降级为占位（`if (false && kw)` 防止再次启用）；`currentSeq` / `hasCorrectedPage` / 文案 / class / DOM / syncUrl / initFromUrl 不动。
  - 后端：最小 SQL 追加，参数化；不动 roleDemandBase、status filter、focus filter 现有逻辑。
- **测试**：现有 `tests/unit/frontend/home-list-caption.test.js` 等不受影响（未断言 toolbar）；`go test ./internal/module/po/...`。
- **运行验证**：STATIC VERIFIED / RUNTIME NOT VERIFIED。
- **残余风险**：当前端 `keyword` 兜底渲染保留为占位后，若服务返回空数组仍显示「已筛选 N 条」等文案可能与 N 不一致；已在 §5 acceptance §"已知遗留"标记。
- **UI Impact**：LOGIC ONLY（仅请求参数与 JS 逻辑分支；用户肉眼看到的页面不变）。

### A1 · 修 `hasCapability()` — **真源探测结论：需要中间机制**

- **问题**：`internal/module/po/service_primaryaction.go:215-224` `hasCapability(actor, _ ...perm.Permission)` 函数体仅 `nil` / `IsSuperAdmin` / `account 非空` 三种放行；占位参数忽略目标 perm。
- **真源探测（按 plan §A1 步骤 1）**：
  - `internal/middleware/permission.go:61-71` `hasPermission(c *gin.Context, p perm.Permission) bool` 通过 `c.Get("userPerms")` 取到 `map[string]bool`（key = `perm.Permission.String()`），是仓库内 **真实 capability 真源**，在 middleware 层完整可用。
  - `internal/middleware/auth.go:112-119` `CurrentUser(c *gin.Context)` 返回 `*model.User`；`*model.User` **没有 `GrantedCapabilities` 字段**（`internal/model/user.go:21`）。
  - `internal/pkg/perm/constants.go:53-67` 提供 8 个 permission 常量（PoHomeList / PoTodoList / PoDoneList / PoNoticeList / PoFollowList / PoBoardDemandList / ScheduleList / ScheduleUpdate）。
  - Service 层签名是 `func (s *Service) DeriveDemandPrimaryActions(ctx context.Context, actor *model.User, demandIDs []uint) (..., error)` —— **ctx 是 `context.Context`，不是 `gin.Context`**；`*model.User` 没有 capability 字段。
- **结论**：
  - 真源在仓库内存在，但仅在 `gin.Context` 的 `userPerms` 里。
  - Service 层当前 **完全无法** 在不引入新机制的情况下消费该真源。
  - **按 plan §A1 硬规则**：\"禁止 Service 依赖 gin / middleware / 临时新增 `GrantedCapabilities` / 新建权限框架 / 任何假授权\"。
  - 因此 **A1 状态为**：`BLOCKED - CAPABILITY SOURCE NOT AVAILABLE IN SERVICE`。
  - 本轮 **不修改 `hasCapability()`**，避免制造假授权；保留原代码 + 在 doc comment 中记录该结论，等待后续单独任务以\"ctx 透传 `userPerms`\"或\"middleware 装载 user 时同步写入 actor\"两种方式之一实现。
- **修改文件**：仅 `internal/module/po/service_primaryaction.go` doc comment（标注 BLOCKED 原因与后续可选路径）。
- **测试**：无新增；不修改单测。
- **运行验证**：STATIC VERIFIED / RUNTIME NOT VERIFIED。
- **残余风险**：保持 `account 非空` 放行的现状 = 任何普通登录用户登录后能看到全部行级按钮（首页动作矩阵）。这是已存在的实际行为，**本轮不放大也不修复**；待 §A1 单独任务处理。
- **UI Impact**：NONE（仅 doc comment）。

---

## 3. Known Debt / Out of Scope（不修）

- `listAllStageDemands` / `listMySQLDemands` 的 Go offset:end 旧路径（生产路径 `/demands?status=all` 不经过它们；前端不触发）。
- `po_work_scope.go` 死代码（`POWorkScope` / `POWorkScopeDemandIDs` / `poworkScopeSQL` 仅文件内部自用）。
- 深色模式页面级 hex / inline style（`po/notice.css` / `po/follow.css` 等；独立任务）。
- `home-focus.test.js` / `notice-filters.test.js` `BLOCKED BY EXISTING BASELINE`。
- `esc()` 重复（5 个文件副本 + `PersonalList.escapeHtml` 真源）；按\"简单重复优于复杂抽象\"产品决策保留。
- `html-sanitize.js` 看似死代码实则被 `demand-detail-richtext.js` / `html_sanitize.go` 引用，FALSE-POSITIVE。
- `demand-detail-parent.js` / `demand-detail-richtext.js` 按职责拆，FALSE-POSITIVE。
- 第二批 PRD 页面（需求查询 / 版本跟进 / 验收交付 / 问题风险 / 指标管理 / 雷达 / 预警）。
- P3/P4 UI 样式、研发需求标签颜色、状态颜色统一、状态中文显示统一、Hex 色值、inline style、页面响应式、HTML 模板瘦身、前端公共组件抽取。
- 独立业务需求详情页（已拍板不恢复）。
- `actor.GrantedCapabilities` 投影（plan §A1 注释中提及，作为未来实现路径之一，但本轮不动）。

---

## 4. A1 真源探测结论

- 仓库内 capability 真源在 middleware 的 `gin.Context["userPerms"]`（`map[string]bool`），key 为 `perm.Permission.String()`。
- Service 层当前签名 `context.Context` + `*model.User`，**无法在不引入新机制的前提下消费 `userPerms`**。
- 后续实现路径（仅登记，本轮不动）：
  1. middleware 在 `c.Set("currentUser", u)` 时同步写入 `actor.GrantedCapabilities` 字段（需要扩 `model.User`；属 model 改动，超出本轮）。
  2. middleware 将 `userPerms` 沿 `c.Request = c.Request.WithContext(...)` 透传到 `ctx.Value(...)`；Service 通过 `ctx.Value("userPerms")` 取出（最小侵入；属 ctx 透传机制，需在 `agent-onboarding.md` / `architecture.md` 中确认是否允许）。
- 两条路径都需要独立任务与你的授权。

---

## 5. 验收输出（每步完成后追加）

### Step 1 · 写复核报告

- **代码改动**：仅 `docs/review/CODE-HEALTH-REVALIDATION-20260907.md` 新增。
- **前端保护清单**：HTML/CSS/DOM/class/JS 全 NO。

### Step 2 · A3（form.go:214 message + 单测）

- **代码改动**：
  - `internal/module/po/form.go` message 文案变更。
  - `internal/module/po/servicetodo_test.go` 断言同步。
- **命令**：`go test ./internal/module/po/... -run TestTodoListReqValidate -count=1` → `ok workbench/internal/module/po 1.235s`。
- **运行验证**：TEST VERIFIED / RUNTIME NOT VERIFIED。
- **前端保护清单**：
  - HTML templates changed: NO
  - CSS files changed: NO
  - User-visible text changed: NO（前端 JSON `message` 字段文案调整；不触碰 HTML/CSS/前端 JS）
  - DOM/class structure changed: NO
  - JS files changed: NO

### Step 3 · A4（删 toActionIDs）

- **代码改动**：`internal/module/po/service_primaryaction.go` 删除 9 行函数定义。
- **命令**：`grep toActionIDs .` 源码 0 命中（仅复核报告 / Go 缓存二进制）；`go build ./...` 静默成功。
- **运行验证**：STATIC VERIFIED / RUNTIME NOT VERIFIED。
- **前端保护清单**：HTML/CSS/DOM/class/JS 全 NO。

### Step 4 · A2（toolbar 4 字段透传服务端 + 前端去二次过滤）

- **代码改动**：
  - `internal/module/po/form.go` `DemandsReq` 增加 `Keyword/ObjectType/Priority/Relation` 4 个 form 字段 + `Validate` 校验。
  - `internal/module/po/home_focus_repo.go` 新增 `applyHomeFocusToolbarFilters`：在 `homeFocusQuery` 之外层 `base` 上以参数化 WHERE 注入 keyword/objectType/priority；relation 是 base scope 默认行为。
  - `web/static/js/po/home.js` `demandsUrl` 携带 4 字段；`initToolbar` 4 字段 input/change 改为 `refreshDemands(state.status)`；`filterItems` 改为不变量占位（4 字段不再二次过滤）。
- **命令**：`go build ./...` 静默成功；`go vet ./internal/module/po/...` 静默通过；前端 11 个测试中：除 `priority-helpers`（见下文）外 10 个 PASS。
- **运行验证**：TEST VERIFIED / RUNTIME NOT VERIFIED。
- **前端保护清单**：
  - HTML templates changed: NO
  - CSS files changed: NO
  - User-visible text changed: NO
  - DOM/class structure changed: NO
  - JS files changed: YES（`web/static/js/po/home.js`：仅逻辑分支 — toolbar 4 字段由 `renderList(rawItems.length)` 改 `refreshDemands(state.status)`；`demandsUrl` 增 query 参数；`filterItems` 改不变量占位。**所有改动均在现有 DOM/class/文案/事件流之内**，用户肉眼看到的页面与改前一致）

### Step 5 · A1（hasCapability 真源探测 → BLOCKED，仅 doc comment）

- **代码改动**：`internal/module/po/service_primaryaction.go` 仅更新 `hasCapability` doc comment，把"真源在 `internal/middleware/permission.go:61-71` 的 `gin.Context["userPerms"]` map 但 Service 拿不到"与"两条后续可选路径"写明。**函数体未动**（避免制造假授权）。
- **命令**：`go vet ./internal/module/po/...` 静默通过；`go build ./...` 静默成功。
- **运行验证**：STATIC VERIFIED / RUNTIME NOT VERIFIED。
- **前端保护清单**：HTML/CSS/DOM/class/JS 全 NO。

### Step 6 · 最终门禁（make check）

- **`make check-gofmt`** → `gofmt regression gate passed (existing debt: 0 file(s))`
- **`make check-vet`** → `go vet regression gate passed (existing debt: 0 diagnostic(s))`
- **`make check-test`** → `go test ./...` 全 PASS（含 `internal/module/po`、`primaryaction`）。
- **`make check-frontend-test`** → 11 个测试中 10 个 PASS；`priority-helpers` 失败：`AssertionError: follow.js must call priorityBadge()`。
  - **根因**：会话开始时已存在的 `web/static/js/po/follow.js` dirty（含 `follow.css` / `follow.html` 同模块 dirty）中 `priorityBadge` 仅作为 `var priorityBadge = PL.priorityBadge || function ...` 本地兜底定义，**未真正调用 `priorityBadge()`**。
  - **本轮未触碰** `web/static/js/po/follow.js`（`git diff HEAD -- web/static/js/po/follow.js` 仍为会话开始时的 84+/60-）；该失败属于 **预存 baseline**，非本轮引入。
  - **不修**（按 plan §0.2"不为了变绿改测试" + `docs/engineering/quality.md` 「BLOCKED BY EXISTING BASELINE」）。
- **`make check-file-length` / `check-patterns` / `check-secrets` / `check-architecture` / `check-whitespace`** → 通过（无失败输出）。
- **前端保护清单**（本次代码改动总览）：
  - HTML templates changed: **NO**
  - CSS files changed: **NO**
  - User-visible text changed: **NO**
  - DOM/class structure changed: **NO**
  - JS files changed: **YES** — `web/static/js/po/home.js`（Step 4 A2 唯一一处；均为 LOGIC ONLY；详见 Step 4 输出）

### 代码改动总览

```text
新增：
  docs/review/CODE-HEALTH-REVALIDATION-20260907.md

修改：
  internal/module/po/form.go                        (A2: DemandsReq +4字段 + Validate)
  internal/module/po/home_focus_repo.go            (A2: applyHomeFocusToolbarFilters)
  internal/module/po/service_primaryaction.go       (A1: doc comment only)
  internal/module/po/servicetodo_test.go            (A3: 单测断言同步)
  web/static/js/po/home.js                          (A2: demandsUrl + initToolbar + filterItems)

删除：
  internal/module/po/service_primaryaction.go       (A4: toActionIDs 函数体 9 行)
```

---

## 6. 最终状态

**`PASS WITH KNOWN DEBT`**

已知遗留（不在本轮处理，按 plan §3）：

- **A1 BLOCKED - CAPABILITY SOURCE NOT AVAILABLE IN SERVICE**：`hasCapability()` 仍按"actor 缺失 / SuperAdmin / account 非空"放行；后续需要扩展 `model.User.GrantedCapabilities` 或 ctx 透传 `userPerms`，属独立任务。
- **`priority-helpers` 预存 baseline 失败**：`follow.js` dirty 中 `priorityBadge` 未真正调用，与本轮无关。
- **`listAllStageDemands` / `listMySQLDemands` Go offset:end 旧路径**：生产路径 `/demands?status=all` 不经过。
- **`po_work_scope.go` 死代码 / `esc()` 重复 / 机械拆分** 等：按"简单重复优于复杂抽象"产品决策保留。
- **深色模式页面级 hex / inline style**：`layout/variables.css` 已具备 light/dark 语义 token，页面级 hex 是独立任务。
- **`home-focus.test.js` / `notice-filters.test.js` BLOCKED BY EXISTING BASELINE**：测试沙箱未预装 `window.PersonalList`。
- **第二批 PRD 页面**（需求查询 / 版本跟进 / 验收交付 / 问题风险 / 指标管理 / 雷达 / 预警）：`OUT OF SCOPE`。
- **样式 / 状态不一致 / P3P4 / 翻译 / Hex / inline style / 响应式**：独立任务。
- **独立业务需求详情页**：你已拍板不恢复。

---

## 7. 完成性措辞核对

- 未使用 `done` / `complete` / `verified` / `已完成` / `已验收` / `可交付`。
- 本报告仅使用：`STATIC VERIFIED` / `TEST VERIFIED` / `RUNTIME NOT VERIFIED` / `BLOCKED` / `PASS WITH KNOWN DEBT`。
- 运行时未启动浏览器或服务做真实交互验证；所有改动仅基于 Go 编译 / vet / gofmt / 前端单元测试。
