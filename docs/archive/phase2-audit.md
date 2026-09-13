# Phase 2 Full Repository Audit

> 历史审查快照：下文的“当前”、严重级别、验收结果和例外建议仅属于原审查范围，
> 不代表当前源码、部署或新增授权。现行规范见 [spec-index.md](spec-index.md)，
> 新审查按 [quality.md](quality.md) 的证据与定级要求复核；保留原文用于追溯。

> 状态：Phase 2 Audit 完成。
> Phase 1 Engineering Governance Baseline **已正式冻结**。本报告不修改 Rules / Gates / Baselines / AGENTS.md / docs/engineering/*.md 的规范定义。
> 本报告仅审计 + 给出 Phase 3 Governance Waves 建议。

## 1. Executive Summary

**当前是否适合继续高速 AI 开发：YES WITH CONDITIONS。**

理由：
- **架构边界层面**：Phase 1 的 Handler → Service → Repo + Scanner 仍然有效。`make check` 全绿，CI Run 33894968843 success，8 个 gate 全部闭环。
- **但发现 1 个 P0 安全漏洞**（`/debug/sqlperf` 无 auth）+ **1 个 P1 capability permission gap**（Schedule 路由缺 RequirePerm，**已降级**：无业务定义证据证明越权严重性前不升 P0）+ 1 个 P0-CANDIDATE-SECRET（明文凭证，是否仍有效未验证）+ 1 个 P2 dead/inert portability debt（`go.mod` self-replace）+ 多个 P1 性能/正确性 在 dept/user/po 模块已记录但根因可归并。
- **绝大多数发现属于"已有 P0/P1 baseline 已知债务"**，根因可归并为 4 个：
  1. **Debug 路由完全无 auth** → P0-AUTH-01
  2. **Schedule capability permission 缺失** → P1-AUTH-02（降级）
  3. **Service 越权直持 `repo.db`**（dept 模块 1 处 + dept 内部 callback `*gorm.DB`）→ P1-ARCH-01
  4. **N+1 / 全量加载 / 内存分页 / Query Fan-out** 集中在 PO + Schedule → P1-PERF-01..04
- **AI Failure Mode 防护充分**：硬 gate 防住了 SELECT *、文件超长、新增 hard pattern。新增的 advisory 类别（route permission / memory pagination / new window / weak hash / SQL wildcard in SQL files）虽然没有 fail-CI，但有可见 drift。
- **业务规则重复**：PO value stage 在 SQL/Service/Frontend 三处各算一次计数 — 这是真实业务复杂度（kind+id 去重）而非 AI 错写，但**确实应该整理"业务真源"为目标**，具体实现留给 Phase 3 架构评审（不预先决定新建 wb_value_stage 表）。
- **超长文件 20 个**：14 个是 templates/CSS/JS，**绝大部分应 ACCEPTABLE FOR NOW**；schedule/form.go 1126 行（54 个 struct，纯 DTO）属于规范内可保留。**不设独立"文件治理 Wave"**。

**结论**：可以继续 Phase 3，但**必须先修 P0**（debug auth）+ 必须**在 Schedule 写新功能前冻结 capability 模型并补齐 RequirePerm**。

## 2. Evidence & Validation

实际执行的命令：
```bash
git status                      # working tree clean + 1 untracked .exec.log
git branch --show-current       # Claude-PO ✅
git rev-parse HEAD               # c366a32b64a2fcf2e3568ab9e9e6ef143152d026
git remote -v                    # 3 remotes; origin 是公司 origin, 不动
make check                       # ✅ 全绿（gofmt / vet / test / file-length / pattern / secret / architecture / whitespace）
go test ./...                    # po + schedule + pkg/zentao 有 test; 其余 17/22 packages 无 test
git diff --check                 # 通过
grep / sed / 静态分析全仓代码
make check 报告：
  - go vet: 1 existing diagnostic (basemodel.go:17)
  - whitespace: passed
  - file-length: 20 existing debt (ratchet, 不允许新增/缩小)
  - hard-pattern: 0 existing finding (non-growth ✅)
  - advisory pattern: 67 findings (0 new this audit)
  - secret: 3 existing fingerprint (configs/config.yaml)
  - architecture boundary: 1 existing debt (dept/service.go)
```

**未被运行环境验证的部分**：
- EXPLAIN 真实 SQL
- Slow query log
- Runtime pprof / 真实数据规模
- Browser E2E（needs headless browser）

这些标 `NOT VERIFIED — RUNTIME EVIDENCE REQUIRED` 于各 finding 中。

## 3. Findings Summary

| Severity | Count | MUST | SHOULD | OPTIONAL |
|---|---|---|---|---|
| P0 | 1 | 1 | 0 | 0 |
| P0-CANDIDATE | 1 | 1 | 0 | 0 |
| P1 | 9 | 7 | 2 | 0 |
| P2 | 10 | 0 | 10 | 0 |
| P3 | 1 | 0 | 0 | 1 |
| GOVERNANCE-GAP | 3 | (本阶段不修) | | |

> **口径**：P0 = `P0-AUTH-01`（debug 无 auth，唯一明确证据的 P0 安全漏洞）。P0-CANDIDATE = `P0-CANDIDATE-SECRET`（明文凭证；凭证是否仍有效未验证 → 实际严重性待验证）。P1 = 9 个（含 P1-AUTH-02）。P2 = 10 个（新增 P2-PORT-01）。

**Root Cause 归并**（避免文件级 findings 散开）：

- RC-1 **Debug 路由完全无 auth** → P0-AUTH-01
- RC-2 **Schedule capability permission 缺失** → P1-AUTH-02（**降级**：无业务定义证据证明"任意登录用户构成生产越权"前，不升 P0）
- RC-3 **Service 越权直持 `*gorm.DB`** → P1-ARCH-01
- RC-4 **N+1 / 全量加载 / Query Fan-out** → P1-PERF-01 / P1-PERF-02 / P1-PERF-03 / P1-PERF-04
- RC-5 **Cross-Repo Transaction 设计空白** → 没有真实用例，归入 GOVERNANCE-GAP
- RC-6 **未使用 actor = capability-wide 资源** → 全部合理
- RC-7 **真使用 actor 但语义不够清晰** → 仅 schedule `Create`/`UpdateWindow` 是 capability-wide，已合规
- RC-8 **`.go.mod` self-replace** → P2-PORT-01（已降级）
- RC-9 **测试缺口** → P2-TEST-01

## 4. P0 / P1 Findings

### P0-AUTH-01 / Debug 路由完全无 auth

- **Severity**: P0
- **MUST / SHOULD**: MUST
- **Category**: Security / Permission
- **Files**:
  - `internal/server/routes.go` (debugGroup := r.Group("/debug"))
  - `internal/module/debug/handler.go:30-34`
- **Evidence**:
  ```
  internal/server/routes.go:	debugGroup := r.Group("/debug")
  internal/server/routes.go:	{
  internal/server/routes.go:		if deps.SqlPerfHandler != nil {
  internal/server/routes.go:			deps.SqlPerfHandler.RegisterRoutes(debugGroup)
  internal/server/routes.go:		}
  internal/server/routes.go:	}
  ```
  没有 `Use(middleware.RequireLogin(...))` 也没有任何 `RequirePerm`。
  Handler 内只注册了：
  ```
  g.GET("", h.List)
  g.GET("/requests", h.Requests)
  ```
  对应到 SQL 性能日志读取 endpoint `/debug/sqlperf` 和 `/debug/sqlperf/requests`。
  `internal/module/debug/repo.go` 暴露的字段：
  ```
  Time / Level / RequestID / Method / Route / Elapsed / ElapsedMS / SQLCount
  ```
  字段不含 SQL 文本，但 Route 列表本身泄露了**所有业务路由**；结合 Elapsed / SQLCount 可推断业务热点。
- **Root Cause**: debug 模块在 boot 时挂在根 group `/debug`，没有 ApplyRequireLogin；与 admin/po/schedule 等其他 group 的"先 RequireLogin 再 RegisterRoutes"模式不一致。
- **Impact**: 任何匿名 HTTP 请求都能读 `sql.log` 摘要；与项目"研发管理工作台"定位严重不符。
- **Recommended Direction**: 把 debug group 移到 admin/schedule 同样 RequireLogin 的中间件层；或加 IP allowlist。
- **Dependency**: 需要同时核查是否有其它直接挂在根 group 的 handler。
- **Regression Risk**: 低 — debug 是 dev 工具，加 login 后只是真实开发人员仍可访问。

### P1-AUTH-02 / Schedule Capability Permission Gap

- **Severity**: P1
- **MUST / SHOULD**: MUST
- **Category**: Security / Permission
- **Files**:
  - `internal/module/schedule/handler.go:100-120` (15 routes)
  - `scripts/quality-baseline/patterns.tsv` (17 ROUTE_PERMISSION_REVIEW advisory entries, 14 在 schedule)
- **Evidence**:
  ```
  internal/module/schedule/handler.go:102:	g := rg.Group("/schedule")
  internal/module/schedule/handler.go:103:	g.Use(middleware.ActiveNav("/schedule"))
  internal/module/schedule/handler.go:105:		g.GET("", h.Index)                       // 无 RequirePerm
  internal/module/schedule/handler.go:115:		g.POST("/windows", h.CreateWindow)       // 无 RequirePerm
  internal/module/schedule/handler.go:119:		g.DELETE("/windows/:id", h.DeleteWindow) // 无 RequirePerm
  ...
  ```
  对比 `internal/module/user/handler.go:55-67` 全部都有 `middleware.RequirePerm(...)`。
  Schedule 已有 RequireLogin（来自 group middleware 层），未确认"任意登录用户当前是否构成生产越权"需业务定义证据。
- **Root Cause**: Schedule 模块历史上只有 group-level `RequireLogin`，从未强制 capability 路由权限。
- **Impact**: 路由层 capability 缺失；**严重程度取决于业务是否定义"谁应该能写 Schedule"**。未提供业务定义证据前不假设越权严重性。
- **Recommended Direction**:
  - Phase 3 Wave 1 先冻结 capability 模型（候选粒度：read/list / scheduling write / window manage），由产品 / 业务权限模型决定。
  - capability 模型冻结后，再补 RequirePerm 到 15 个 route。
  - 加 `internal/pkg/perm` capability 常量。
- **Dependency**: 产品/业务确认 capability 粒度。
- **Regression Risk**: 低 — 加 RequirePerm 是收紧行为；现有 RequireLogin 仍生效。

### P1-ARCH-01 / Service 越权直持 `*gorm.DB`

- **Severity**: P1
- **MUST / SHOULD**: MUST
- **Category**: Architecture Boundary
- **Files**:
  - `internal/module/dept/service.go` (Phase 1 baseline `architecture.tsv` 已记录 1 处)
  - `internal/module/dept/service.go` callback 内部 `tx.WithContext(ctx).Model(...)` (第 80-110 行范围内)
- **Evidence**:
  ```
  internal/module/dept/service.go:  if err := s.repo.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
  internal/module/dept/service.go:      if txErr = tx.WithContext(ctx).Model(&model.Dept{}).Where(...)
  ```
  `check-architecture.sh` 当前只检测 service.go 文件中的 `.db.(WithContext|Table|Model|Raw|Exec)` 与 `*gorm.DB`。callback 内的 `tx *gorm.DB` 没在 baseline 里（因为 callback 内的 tx 形参本身是合法事务句柄，由 Repo 内部事务创建）。
  **但 Service 层第一次拿 `s.repo.db` 越权本身仍存在。**
- **Root Cause**: dept 模块在实现 ancestor cascade update 时需要事务原子性，但没走 Repo.Transaction 而直接持 Repo 内部的 `db` 字段。
- **Impact**: 阻碍其它模块复用 dept Repo；arm architecture scanner 被 baseline 锁住，**新文件同样的写法仍会被 scanner 抓到**（baseline 只豁免这一处）。
- **Recommended Direction**: 把 ancestor cascade update 移到 `dept.Repo.UpdateWithAncestor(ctx, ...)`，Service 通过 `s.repo.UpdateWithAncestor` 调用。
- **Dependency**: 需要明确 Repo 是否要支持 `tx *gorm.DB` 形参 callback 模式（当前 schedule 已有 `r.Transaction(ctx, func(txRepo *Repo) error)`）。
- **Regression Risk**: 中 — cascade update 行为不能改；需要 before/after 测试。

### P1-PERF-01 / PO Home 阶段汇总 Query Fan-out（≥29 SQL / page）

- **Severity**: P1
- **MUST / SHOULD**: SHOULD
- **Category**: Performance
- **Files**:
  - `internal/module/po/service.go:55-118` (Home 函数)
  - `internal/module/po/servicefollow.go` (countAllStageUniq)
  - `internal/module/po/repokpi.go` (4 个 CountKPI*)
  - `internal/module/po/repovaluestream.go` (CountRoleDemands/CountScheduleStories/CountDeliverStories)
- **Evidence**:
  - 9 个 value stream stages，每个调 `s.repo.CountRoleDemands(ctx, account, filter)` = 9 SQL
  - 部分 stage 含 `scheduleIncomplete` → 额外 `CountScheduleStories` = 1 SQL
  - 部分 stage 含 `deliverStories` → 额外 `CountDeliverStories` = 1 SQL
  - `countAllStageUniq` 调 `FindRoleDemandIDs` 8 次 + `FindScheduleStoryIDs` ≈ 10 SQL
  - `fillKPICounts` 调 4 个 `CountKPI*` = 4 SQL
  - `ListHomeVersionWindows` 内部 schedule module 调 ≈ 4 SQL
  - **合计 ≥ 29 SQL / page render**（static estimate，未 runtime 验证）
- **Root Cause**: 业务上每个 stage 是独立"kind+status"组合，没办法一个 SQL 算出；"全部"是 union 去重更没法单 SQL。**这是真实业务复杂度**，不是 AI 错写。
- **Impact**: PO 首页冷启动 ≥ 29 SQL（static estimate）。实际 P50/P95/P99 **NOT VERIFIED — RUNTIME EVIDENCE REQUIRED**。
- **Recommended Direction (Candidate)**:
  - 不预先承诺具体 cache 方案 / 固定 TTL / 固定 LIMIT / SQL 实现细节。
  - 必须先采集 runtime evidence（real row count, EXPLAIN, slow query log, representative dataset）后才能选最终方案。
  - 候选方向（**待 runtime evidence 验证后决定**）：
    - **reduce repeated aggregation / query fan-out**（减少重复聚合 / 查询扇出）。
    - **evaluate consolidated metric query**（评估合并的指标查询是否能共用 base filter）。
    - **evaluate caching only if runtime evidence proves repeated expensive computation**（仅当 runtime evidence 证明存在重复昂贵计算时，才评估启用 caching）。
  - 具体 SQL 结构 / cache TTL / 阈值由 Phase 3 Wave 3 决定，**不在 Phase 2.1 预设**。
- **Dependency**: 不依赖其它 finding；依赖 Phase 3 Wave 3 先采集 runtime evidence。
- **Regression Risk**: 中 — 改 SQL 必须配 contract test 锁住 stage 计数结果。

### P1-PERF-02 / Notice 全量加载 + 内存分页 + 内存 filter

- **Severity**: P1
- **MUST / SHOULD**: MUST
- **Category**: Performance
- **Files**:
  - `internal/module/po/reponotice.go:42-72` (FindNotices)
  - `internal/module/po/reponotice.go` (filterNoticeRows / noticeQuickCounts / noticeCategoryCounts)
  - `internal/module/po/servicenotice.go:19-39` (NoticeList)
- **Evidence**:
  ```
  internal/module/po/reponotice.go:func (r *Repo) FindNotices(...) {
  internal/module/po/reponotice.go:	allRows, err := r.findNoticeRows(ctx, account)   // 全量拉
  internal/module/po/reponotice.go:	resp.Total, resp.Unread, resp.Action, resp.Abnormal, resp.Today = noticeQuickCounts(allRows)  // 内存算计数
  internal/module/po/reponotice.go:	filteredRows := filterNoticeRows(allRows, req)    // 内存 filter
  internal/module/po/reponotice.go:	start := (req.Page - 1) * req.PageSize
  internal/module/po/reponotice.go:	end := start + req.PageSize
  internal/module/po/reponotice.go:	... // 内存分页
  ```
  `findNoticeRows` 是 `SELECT ... FROM zt_notify AS n LEFT JOIN zt_action ...` 无 Limit。
- **Root Cause**: 通知表 (zt_notify) 与 ZenTao 共享，Schema Ownership 是 ZenTao；Workbench 这边要算 8 个 bucket count（Total/Unread/Action/Abnormal/Today/Categories/Filtered/Paged Items），如果每个都用独立 SQL，**会从 1 次变成 8+ 次反而慢**。当前设计是"一次拉取，多次复用"——这是一种合理 trade-off，**但前提是通知表有上界**。
- **Impact**: 取决于 `zt_notify` 实际行数。NOT VERIFIED — RUNTIME EVIDENCE REQUIRED（需 SELECT COUNT(*) FROM zt_notify WHERE toList LIKE '%account%'）。
- **Recommended Direction (Candidate)**:
  - **严禁** "先 LIMIT 5000 再计算 Total/Unread/Category" 类建议。QuickView 和 Category count 必须保持全量语义正确；任何"先截断再算"都会改变 Total/Unread/Category 等汇总字段的含义。
  - 必须先采集 runtime evidence（real row count per account, EXPLAIN, slow query log）才能选最终方案。
  - 候选方向（**待 runtime evidence 验证后决定**）：
    - A. **数据天然有界**（每个账户通知数 < runtime evidence 显示的阈值） → 保持当前简单实现。
    - B. **数据较大**（每个账户通知数超过阈值） → SQL aggregate 负责 summary / count（Total/Unread/Action/Abnormal/Today/Categories），SQL pagination 负责 Items；保持 QuickView 全量语义正确。
  - 阈值本身由 runtime evidence 决定，**不在 Phase 2.1 预设**。
- **Dependency**: 真实 EXPLAIN + 真实通知数据量。
- **Regression Risk**: 中-高 — QuickView/Items/Categories 三个 UI 接口契约要锁。

### P1-PERF-03 / Window Card N × 1 Query Fan-out

- **Severity**: P1
- **MUST / SHOULD**: SHOULD
- **Category**: Performance
- **Files**:
  - `internal/module/schedule/service_window.go:27-120` (ListWindowCards)
  - `internal/module/schedule/service_window.go:300-340` (ListWindows)
  - `internal/module/schedule/repo.go:255` (GetWindowConsumedHours)
- **Evidence**:
  ```
  internal/module/schedule/service_window.go:96:		capacityHours, err := s.CalcCapacity(...)
  internal/module/schedule/service_window.go:106:		consumed, err := s.repo.GetWindowConsumedHours(ctx, window.ID)
  ```
  ListWindowCards / ListWindows 对每个 window 调一次 `GetWindowConsumedHours`（1 SQL / window）。
- **Root Cause**: window 数量是产品决策（同一时间能并发几个版本窗口），但目前 `r.repo.FindAll(ctx)` 无 LIMIT。
- **Impact**: 取决于活跃 window 数。当前典型 5-15 个窗口 = 5-15 SQL；上限取决于团队规模。NOT VERIFIED — RUNTIME EVIDENCE REQUIRED。
- **Recommended Direction (Candidate)**:
  - **不预设固定时间窗口**（如 `releaseDate >= now - 3 month`）。
  - 候选方向（**待 runtime evidence 验证后决定**）：
    - 一次性 batch aggregation（按真实 window 数据规模评估）。
    - 根据业务展示范围，评估必要的数据范围约束（具体范围由 runtime evidence 决定）。
  - 不在 Phase 2.2 预设任何具体阈值或时间窗口。
- **Dependency**: 不依赖。
- **Regression Risk**: 低 — 纯聚合优化。

### P1-PERF-04 / 字典加载重复调用（10+ 处 `loadAccountDisplayMap`）

- **Severity**: P1
- **MUST / SHOULD**: SHOULD
- **Category**: Performance / Maintainability
- **Files**:
  - `internal/module/po/service.go:177`
  - `internal/module/po/repotodoapproval.go:73`
  - `internal/module/po/repofollow.go:104,183`
  - `internal/module/po/repotodoextra.go:95,127,160,185,202`
  - `internal/module/po/repodone.go:277`
- **Evidence**: 见上 — 同一个 Request 生命周期内可能 10+ 次调 `loadAccountDisplayMap`。
- **Root Cause**: 各 Repo 函数独立调，没有 request-scoped cache。
- **Impact**: 同一请求多次重复读用户表。NOT VERIFIED — RUNTIME EVIDENCE REQUIRED（具体看实际多 SQL 数）。
- **Recommended Direction (Candidate)**:
  - **目标**（固定）：同一 request 内 `account display map` 最多加载一次。
  - **实现**（**不预先决定**）：
    - Phase 3 架构评审候选：explicit load + pass / lightweight request-scoped loader / 复用仓库现有成熟能力。
    - **严禁** `ctx.Value("displayMap")` 作为推荐实现——`context.Context` 不应成为业务字典缓存的隐藏依赖。
    - **严禁** 提前造框架。
- **Dependency**: 不依赖。
- **Regression Risk**: 低 — 纯缓存。

### P1-PORT-01 / `go.mod` Self-Replace（**降级到 P2-PORT-01**：见后文）

- **Severity**: P1（本次审计初判） → **本次精修降级为 P2-PORT-01**（理由见 §P2 表）
- **MUST / SHOULD**: 见 P2-PORT-01 行
- **Category**: Portability / Build
- **Files**:
  - `go.mod:71` `replace workbench => /home/wds/repo/workbench`
- **Evidence**:
  ```
  go.mod:1:module workbench
  go.mod:71:replace workbench => /home/wds/repo/workbench
  ```
  这是 **self-replace**：当前 module 自己叫 `workbench`，replace 指向 `/home/wds/repo/workbench`（Linux 绝对路径，在 macOS 上不存在）。
  当前 `make check` 全绿、`go build ./...` 成功——因为 **Go toolchain 在 build 当前 module 时会忽略 self-replace**（避免循环）。
  GitHub Actions Run 33894968843 成功，说明当前 CI runner 兼容此配置。
- **Root Cause**: 历史原因（开发者 wds 本地 fork 调试用），未清理。`/home/wds/repo/workbench` 在当前 macOS 开发机不存在，但 Go toolchain 自洽处理。
- **降级理由**：
  - 已知"当前 CI / 本地 build / make check 全部 success"是 **已验证证据**。
  - 没有证据显示"build baseline 不稳"或"必须 Wave 0 先修"。
  - 没有证据显示"GitHub runner 可能存在等价路径"——这是猜测，应删除。
  - 实际影响范围 = 其它开发者 clone / `go mod tidy` / IDE module resolution（**理论**影响）。
- **Impact**:
  - 其他开发者 clone 此 repo 后 `go mod tidy` 可能报错或修改 go.mod（**理论**，未实际验证）。
  - IDE（Goland / VSCode + gopls）的 module resolution 行为可能不一致（**理论**，未实际验证）。
  - 当前 `make check` / `go build` / GitHub Actions 均 success（**已验证**）。
- **Recommended Direction (Deferred P2)**:
  - **不进入主 Phase 3 Wave**。放入 **Deferred P2 Backlog**，由后续 portability 任务（不在 Phase 3 范围）处理。
  - 待办任务可包括：确认这条 replace 是否真的 dead；本地 + CI 各跑一次 `make check` + `go test ./...` + `go build ./cmd/server`；删除并验证仍可 build。
- **Dependency**: 无（删除本身即可）。
- **Regression Risk**: 中 — 需要 CI runner 上验证删除后还能 build。

### P1-USER-01 / BatchCreate N Query ExistsByAccount + 无 Transaction

- **Severity**: P1
- **MUST / SHOULD**: SHOULD
- **Category**: Correctness
- **Files**:
  - `internal/module/user/service.go` BatchCreate（line 附近的 `for _, item := range req.Users` 循环）
- **Evidence**: 循环内每次调 `s.repo.ExistsByAccount` 检查，**没有包在事务里**；后续 Create 也没在事务里。
- **Root Cause**: 批量创建用户时，如果两个 account 在两次检查之间 race condition，可能都通过检查导致 duplicate。
- **Impact**: 实际并发低时问题不显，但理论上是 correctness gap。
- **Recommended Direction (Candidate)**:
  - **严禁** 默认建议"加 unique index 双保险"。必须先确认目标 user 表是 ZenTao-owned 还是 Workbench-owned，以及现有 unique constraint。
  - 如果是 ZenTao 原生表：**未经明确 schema 决策不得新增 Workbench unique index**。
  - Phase 3 候选方向（**待 schema 验证后决定**）：
    - A. batch conflict query（一次性 `SELECT 1 FROM user WHERE account IN (?)` 拉冲突集）。
    - B. transaction atomicity（包在事务里）。
    - C. existing schema constraint verification（确认现有 DB 层 unique index 行为）。
- **Dependency**: 目标表 ownership（ZenTao vs Workbench）+ 现有 unique constraint（NOT VERIFIED — runtime EXPLAIN / schema inspection）。
- **Regression Risk**: 低 — 事务化 + batch conflict query 是收紧行为，不改变业务输出。

### P1-MD5-01 / ZenTao Compatible MD5 Password Write

- **Severity**: P1
- **MUST / SHOULD**: MUST
- **Category**: Security (legacy compatibility)
- **Files**:
  - `internal/module/user/service.go:74,144,188` （PasswordHash: encode.MD5(...)）
  - `internal/pkg/encode/encode.go` MD5 实现
- **Evidence**: 见上 — 共 3 处调用。
- **Root Cause**: ZenTao 兼容需要；ZenTao 用户的 password 是 MD5 存在 zt_user.password。
- **Impact**: ZenTao login 兼容必须保留；**不能直接禁**。但**新写的代码不能复制**这个模式（Phase 1 baseline 已记录 6 个 `WEAK_PASSWORD_HASH` advisory fingerprints）。
- **Recommended Direction**:
  - 在 encode 包加注释"This is ZenTao legacy compatibility. Do NOT use in Workbench-owned code."
  - 新建 Workbench-owned 表（如未来加 `wb_user`）必须用 bcrypt/argon2。
- **Dependency**: 取决于产品是否计划迁移 user 表到 Workbench-owned schema。
- **Regression Risk**: 中-高 — 不能破坏 ZenTao login。

### P2 findings（高 9 个）

| ID | Severity | Title | Files |
|---|---|---|---|
| P2-PORT-01 | P2 | `go.mod` self-replace (`workbench => /home/wds/repo/workbench`) dead/inert portability debt（**已验证**：本地 + CI build / test / make check 全绿；理论影响 = 其它开发者 clone / `go mod tidy` / IDE module resolution）。**本 Finding 不进入任何 Wave**；放入 **Deferred P2 Backlog**，由后续 portability 任务处理。 | `go.mod:71` |
| P2-TEST-01 | P2 | 17/22 packages 零测试 | `internal/module/{dept,login,debug,menu,role,operationlog,loginlog,user,internal/server,internal/pkg/{...12 个}}` |
| P2-FORM-01 | P2 | `internal/module/schedule/form.go` 1126 行（DTO 集合） | `schedule/form.go` |
| P2-FORM-02 | P2 | `internal/module/schedule/handler_demand.go` 665 行 | `schedule/handler_demand.go` |
| P2-FORM-03 | P2 | `internal/module/user/handler.go` 544 行 | `user/handler.go` |
| P2-CSS-01 | P2 | `web/static/css/schedule/scheduleintegrated.css` 880 行 | 同 |
| P2-CSS-02 | P2 | `web/static/css/components/components.css` 595 行 | 同 |
| P2-CSS-03 | P2 | `web/static/css/schedule/schedulelist.css` 503 行等 4 个 | 同 |
| P2-JS-01 | P2 | `web/static/js/schedule/scheduleintegrated.js` 842 行 + `ui.js` 775 行 | 同 |

> **口径说明**：
> - **`configs/config.yaml` 明文凭证**由 `P0-CANDIDATE-SECRET` 单一 Severity 口径承担（见 §Security Matrix + §6）。**不再独立列 P2-ENV-01**，避免同一风险双计数。
> - `P2-PORT-01` (`go.mod` self-replace) **不进入主 Phase 3 Wave**；放入 **Deferred P2 Backlog**，由后续 portability 任务（不在 Phase 3 范围）处理。

### P3 findings（低 1 个）

| ID | Severity | Title |
|---|---|---|
| P3-CMT-01 | P3 | 每个 Go 文件顶部 8 行 ASCII banner 注释（无信息量） |

## 5. Authorization Model Matrix

| Module | Capability (典型) | Authentication | Capability Permission | Object Scope Exists? | Service Enforcement | Conclusion |
|---|---|---|---|---|---|---|
| po | PoHomeList / PoTodoList / PoDoneList / PoNoticeList / PoNoticeUpdate / PoFollowList / PoFollowUpdate / PoBoardDemandList / PoBoardTaskList | ✅ RequireLogin | ✅ RequirePerm on all 14 routes | ✅ actor.Account 真用作"我的数据"过滤 | ✅ Home / Demands / Todo / Done / Follow / Notice 全部按 account 过滤 | **合规** ✅ |
| schedule | (Phase 3 需新建) ScheduleList / ScheduleWindowManage / ScheduleSchedulingEdit | ✅ RequireLogin (group level) | ⚠️ 0 RequirePerm, 仅 advisory；**先冻结 capability 模型再补 RequirePerm** | 部分真用（Delete window 用 CreatedBy 检查；SaveScheduling 用 account 验证产品访问权） | 真用 + 部分 capability-wide 资源（Create/Update Window 是 admin-level） | **缺口在路由层但需业务定义证据** → P1-AUTH-02 |
| user | UserList / UserCreate / UserUpdate / UserDelete / UserResetPassword | ✅ RequireLogin (admin group) | ✅ RequirePerm on all routes | ❌ admin-wide resource；actor 仅作为 audit 字段 | `_ = actor` 6 处 | **合规**（admin-wide resource 无需 object-scope） ✅ |
| dept | DeptList / DeptCreate / DeptEdit / DeptDelete | ✅ RequireLogin (admin) | ✅ RequirePerm on all routes | ❌ admin-wide；actor 仅 audit | `_ = actor` 6 处 | **合规** ✅ |
| menu | MenuList / MenuCreate / MenuEdit / MenuDelete | ✅ RequireLogin (admin) | ✅ RequirePerm on all routes | ❌ admin-wide | `_ = actor` 6 处 | **合规** ✅ |
| role | RoleList / RoleCreate / RoleEdit / RoleDelete | ✅ RequireLogin (admin) | ✅ RequirePerm on all routes | ❌ admin-wide | 全部 RequirePerm | **合规** ✅ |
| login | (own session) | ✅ RequireLogin | ❌ 不需要 capability（self） | N/A | Logout 用 session 而非 actor | **合规** ✅ |
| loginlog | LoginLogList | ✅ RequireLogin (admin) | ✅ RequirePerm | ❌ admin-wide audit log | `_ = actor` | **合规** ✅ |
| operationlog | OperationLogList | ✅ RequireLogin (admin) | ✅ RequirePerm | ❌ admin-wide audit log | (仅读 list) | **合规** ✅ |
| debug | (none — 完全无 auth) | ❌ **完全无 RequireLogin** | ❌ | N/A | N/A | **P0-AUTH-01** 🚨 |

**结论**：所有 admin-wide 资源的 `_ = actor` 都合规（不需要 object-scope）。**唯一真问题是 debug 无 auth + Schedule 缺 capability permission**。

## 6. Security / Permission Matrix

| Risk Surface | Where | Status | Reference |
|---|---|---|---|
| 匿名访问 SQL 性能日志 | `/debug/sqlperf` | **P0** | P0-AUTH-01 |
| Schedule 写操作无 capability permission | `/schedule/*` 15 routes | **P1**（Schedule 有 RequireLogin；当前**未经证据证明**"任意登录用户可执行高危 Schedule 写操作且属于明确生产越权"。**严禁**未提供业务定义证据就提前升 P0。Phase 3 Wave 必须**先冻结 capability 模型**，再补 RequirePerm） | P1-AUTH-02 |
| User BatchCreate race condition | `/admin/users/batch/create` | P1 | P1-USER-01 |
| 明文凭证 committed in git history | `configs/config.yaml` | **P0-CANDIDATE-SECRET / MUST ROTATE**（**已知**，3 fingerprint baseline；当前未验证凭证是否仍有效 → 必须按 quality.md:54 单独授权 rotate + placeholder，git history 由用户单独授权处理） | quality.md:50-54 |
| MD5 password write (ZenTao 兼容) | user/service.go + encode/encode.go | P1 (must keep for ZenTao compat) | P1-MD5-01 |
| SQL 字符串拼接 | (无 — 全部走 `?` placeholder) | ✅ 安全 | 见 Q4-M |
| `template.HTML` unsafe | (无) | ✅ 安全 | — |

## 7. Business Rule Duplication Matrix

| Concept | Implementation A | Implementation B | Implementation C | Consistency | Recommended Truth Source |
|---|---|---|---|---|---|
| PO Value Stream Stage | `mysqlStageFilters` in `internal/module/po/repovaluestream.go` | `valueStreamStages` in `internal/module/po/service.go` | (Frontend) `web/templates/po/home.html` 循环 valueStreamStages | PARTIAL — Service 已知 9 stage 顺序与 label；SQL filter 是另一份映射；前端再展示一次 | **目标**（固定）：PO Value Stage 必须有一个唯一可测试的 domain truth source，SQL / Service / Frontend 不得各自重新定义 stage semantics。**候选方向**（不预设）：现有 Go domain mapping / query builder / resolver / constants/rules module。只有业务证明需要"运行时可配置阶段"时才讨论新建配置表。 |
| Today / Overdue / Blocked / Suspended | `repokpi.go` 4 个独立 CountKPI 方法（4 SQL） | Service 顺序填 | Frontend 5 KPI card | CONSISTENT（schema 已固定）但 fan-out | **目标**（固定）：减少重复 base-filter 查询；SQL / Service / Frontend 不再各自重新定义 metric semantics。**候选方向**（不预设）：是否合并为单个多 metric SQL 查询由 Phase 3 runtime evidence 决定。 |
| Window Teamgroup 名称（含 parent path） | `service_window.go:55-90` 拼 `parent / name` | `service.go:91` 同样拼 | Frontend `web/templates/schedule/index.html` 展示 | CONSISTENT（算法一致） | **目标**（固定）：统一为单一可复用 formatting capability，避免跨 Service 重复字符串拼接逻辑。**候选方向**（不预设）：具体位置（`包 /`、internal/module/schedule/format、函数提取）由 Phase 3 架构评审决定；命名与 package 由架构评审选定。 |
| Account Display Name | `loadAccountDisplayMap` 拉 `account → displayName` | 10+ 处调用 | Frontend 直接展示 | CONSISTENT（cache 不一致 → 见 P1-PERF-04） | **目标**（固定）：同一 request 内不应重复加载相同 account mapping。**候选方向**（不预设）：实现方式（explicit load + pass / lightweight request-scoped loader / 复用仓库现有成熟能力）由 Phase 3 架构评审决定。**严禁** `ctx.Value` 作为推荐实现；**严禁**提前造框架。 |
| 7 维 AND 公式 (Todo) | `servicetodo.go:43-58` Go filter | `repotodo.go` SQL 部分过滤 | (Frontend) 7 个 chip filter UI | CONSISTENT — 业务规则来源：PRD V10.1 02 节（注释明确） | **目标**（固定）：评估可安全下推 DB 的 filtering / aggregation，**不得改变业务语义**。**候选方向**（不预设）：能在 SQL 内等价的 filter 必须下推；涉及业务规则模糊的部分保留 Go 计算。具体哪些维可下推由 Phase 3 runtime evidence 决定。 |
| Notice QuickView bucket | `reponotice.go:noticeQuickCounts` 内存算 | `noticeCategoryCounts` 内存分类 | Frontend bucket UI | CONSISTENT | **目标**（固定）：QuickView 全量语义正确，summary 与 items 不得因优化而错算。**候选方向**（不预设）：数据天然有界 → 保持现状；数据较大 → SQL aggregate 负责 summary / count，SQL pagination 负责 items。是否切换阈值由 Phase 3 runtime evidence 决定。 |

## 8. Page Query Plans

### 8.1 PO Home

- **Service Entry**: `Service.Home(ctx, actor)` `internal/module/po/service.go:56`
- **Repo Calls**: `CountRoleDemands` ×9, `CountScheduleStories` ×1, `CountDeliverStories` ×1, `FindRoleDemandIDs` ×8, `FindScheduleStoryIDs` ×2, `CountKPIToday/Overdue/Suspended/Blocked` ×4, `ListHomeVersionWindows` (内部 ≈4 SQL)
- **Static SQL Fan-out Estimate**: ≥29 SQL
- **Potential Full Read**: NO（CountRoleDemands 是 COUNT, not Find）
- **Potential N+1**: NO direct; fan-out 是多 stage 多 filter
- **Repeated Lookup**: account 字典（Home 不直接显示 user name，所以 OK）
- **Go Memory Processing**: NO（仅汇总 int64）
- **Pagination Position**: NO（首页不分页）
- **Risk**: P1-PERF-01
- **Recommended Direction**: 多 stage UNION ALL 化 "全部" 计算；KPI 多 metric 一次 SQL；短期 in-memory cache

### 8.2 Todo

- **Service Entry**: `Service.TodoList(ctx, actor, req)` `internal/module/po/servicetodo.go:22`
- **Repo Calls**: `FindTodoItems` 拉全量
- **Static SQL Fan-out Estimate**: 1 SQL
- **Potential Full Read**: **YES** — `findTodoItems` 无 LIMIT
- **Potential N+1**: NO
- **Repeated Lookup**: account 字典（多处 `loadAccountDisplayMap`）
- **Go Memory Processing**: YES — 7 维 AND filter + summary + group + tab/focus + sort + paginate 全 Go
- **Pagination Position**: 全内存分页 (`items[start:end]`)
- **Risk**: P1 — 真实业务复杂度（7 维 AND），但**应该把 SQL filter 做满**
- **Recommended Direction**: Phase 3 评估：能否把 7 维 AND 中至少 5 维下推到 SQL，剩 2 维（业务规则模糊）走 Go。

### 8.3 Done

- **Service Entry**: `Service.DoneList` `internal/module/po/servicedone.go:20`
- **Repo Calls**: `FindDoneActions` (SQL 分页) + `CountDoneActions` (summary)
- **Static SQL Fan-out Estimate**: 2 SQL
- **Potential Full Read**: NO（Page/PageSize 进 SQL）
- **Potential N+1**: NO
- **Repeated Lookup**: account 字典（1 处 `loadAccountDisplayMap`）
- **Go Memory Processing**: NO
- **Pagination Position**: SQL-side ✅
- **Risk**: 低
- **Recommended Direction**: 现有结构符合规范；**作为好代码样板**。

### 8.4 Notice

- 见 P1-PERF-02。

### 8.5 Follow

- **Service Entry**: `Service.FollowList` `internal/module/po/servicefollow.go:23`
- **Repo Calls**: `FindFollowItems` (推测 SQL 分页)
- **Static SQL Fan-out Estimate**: 1 SQL
- **Recommended Direction**: 检查 Repo 实现确认 SQL 分页。

### 8.6 Schedule

- **Service Entry**: 多个（Index / ListWindows / ListWindowCards / ListBizDemands / ListIndependentStories / GetMatchingPlans / GetStoryTasks 等）
- **Static SQL Fan-out Estimate**:
  - `ListWindowCards`: 1 (FindAll) + 1-2 (FindTeamgroupsByIDs) + N (GetWindowConsumedHours) — **N × 1 fan-out**
  - `Index`: 多个列表
- **Recommended Direction**: P1-PERF-03

### 8.7 Workboard / Board

- **Service Entry**: `Service.BoardTask/BoardDemand/BoardIssues/BoardGroupMetrics` `internal/module/po/serviceboard.go`
- **Recommended Direction**: 抽查实际 SQL 数（Phase 3）

### 8.8 Demand Query

- 同 Schedule 8.6 子项

### 8.9 Version Window

- 见 8.6 `ListWindowCards` / `ListWindows` / `GetByID`

## 9. Architecture Boundary Review

### Handler

- ✅ 所有 Handler 不直接 import `gorm.io/gorm` 或 `"database/sql"`（验证：`grep -rn "gorm.io/gorm\|database/sql" internal/module/**/*handler*.go` 仅 login 内部用到 `c.ShouldBindQuery` 等 gin helper）
- ✅ Handler 不写业务状态决策（业务状态全在 Service）
- ✅ Handler 不写权限决策（RequirePerm 在路由注册时声明）

### Service

- ❌ **dept/service.go**: `s.repo.db.WithContext(ctx).Transaction(...)` 越权 — **P1-ARCH-01**
- ⚠️ dept callback 内部 `tx *gorm.DB` 直接使用 — 由 `s.repo.db` 越权传递，需要在 P1-ARCH-01 修复时一并修
- ✅ 其它 Service（po / schedule / user / role / menu / login / operationlog / loginlog）不持有 `repo.db`

### Repo

- ✅ 所有 Repo 函数接 `context.Context` 而非 `gin.Context`（验证：`grep -rn "gin.Context" internal/module/**/*repo*.go` 无结果）
- ✅ Repo 不 import `github.com/gin-gonic/gin`
- ✅ Schedule `Repo.Transaction` 模式良好：`r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error { return fn(&Repo{db: tx}) })` —— tx Repo 是新 Repo 实例

### Transaction

- **真实 Cross-Repo 原子需求 = 0**（见 RC-5 / GOVERNANCE-GAP-01）
- schedule 5 处 `s.repo.Transaction` 都是单 Repo 内部多表
- dept 1 处 `s.repo.db.WithContext(ctx).Transaction` 是 Service 越权，应该改写为 `dept.Repo.UpdateWithAncestor(...)` 内部事务
- user `BatchCreate` 没有事务 — P1-USER-01

### Cross-Service

- ✅ PO → Schedule → User 跨模块 Service 调用 (`internal/module/po/service.go:46-51`)，**in-process orchestration** 是合理架构
- ❌ 不要拆微服务

### Gate Coverage

- ✅ 现有 `check-architecture.sh` 检测：Handler-DB-Access / Repo-HTTP-Dep / Service-DB-Access
- ⚠️ **潜在盲区（GOVERNANCE-GAP-02）**：
  - `Service.db.Create / Find / First / Take / Save / Updates / Delete / Count / Row / Rows` 写法（不仅是 `WithContext / Table / Model / Raw / Exec`）
  - `Service` 直接 `import "github.com/gin-gonic/gin"`（**未见实际违反**，但 scanner 不检）
  - `Handler` 直接绕 Service 调 Repo（**未见实际违反**，但 scanner 不检）
- 当前代码真实违反数 = 0（除 dept service.go 1 处已 baseline）
- 是否值得 Phase 3 加 scanner？**取决于 dept 修复后是否有其它 violation**。如果 dept 修复后扫描全绿，**不立即加 scanner**，governance-gap 记录即可。

## 10. Oversized File Review

| File | Lines | Type | Verdict | Capability Boundary |
|---|---|---|---|---|
| `internal/module/po/repodone.go` | 504 | Repo (Done Action capability) | ACCEPTABLE FOR NOW | Done Action 数据访问 — 单能力 |
| `internal/module/schedule/form.go` | 1126 | DTO 集合 (54 struct types) | **ACCEPTABLE FOR NOW**（继续 Do Not Fix，**不设独立 Wave 治理**） | 排期模块请求/响应结构 — 一个能力 |
| `internal/module/schedule/handler_demand.go` | 665 | Handler (排期业需) | ACCEPTABLE FOR NOW（Phase 3 Wave 1 改 RequirePerm 时若责任边界确阻碍维护再顺带拆） | 排期业务相关路由 |
| `internal/module/schedule/handler.go` | 515 | Handler (排期基础) | ACCEPTABLE FOR NOW（同上） | 排期基础路由 |
| `internal/module/user/handler.go` | 544 | Handler (User 模块) | ACCEPTABLE FOR NOW（**不在 Wave 1 顺手拆**，避免越权；按真实 capability boundary 决定） | User CRUD + 批量 + 导出 + 重置密码 |
| `web/static/css/components/components.css` | 595 | Shared component CSS | ACCEPTABLE FOR NOW | 共享组件样式 — 单能力 |
| `web/static/css/layout/layout.css` | 531 | Layout CSS | ACCEPTABLE FOR NOW | 布局样式 — 单能力 |
| `web/static/css/schedule/schedule.css` | 580 | Schedule CSS | ACCEPTABLE FOR NOW（Wave 6 前端 consolidation 若需再顺带拆） | 排期页面样式 |
| `web/static/css/schedule/scheduleintegrated.css` | 880 | Schedule 一体化 | ACCEPTABLE FOR NOW（同上） | 排期一体化样式 |
| `web/static/css/schedule/schedulelist.css` | 503 | Schedule list | ACCEPTABLE FOR NOW（同上） | 排期列表样式 |
| `web/static/css/schedule/schedulemodal.css` | 645 | Schedule modal | ACCEPTABLE FOR NOW（同上） | 排期弹窗样式 |
| `web/static/js/components/form.js` | 513 | Shared form JS | ACCEPTABLE FOR NOW | 共享 form 渲染 |
| `web/static/js/debug/sqlperf.js` | 550 | Debug JS | ACCEPTABLE FOR NOW | 调试工具 |
| `web/static/js/dept/list.js` | 655 | Dept list JS | ACCEPTABLE FOR NOW | 部门列表 |
| `web/static/js/po/workboard.js` | 660 | PO workboard JS | ACCEPTABLE FOR NOW | PO 看板 |
| `web/static/js/schedule/scheduleintegrated.js` | 842 | Schedule 一体化 JS | ACCEPTABLE FOR NOW | 排期一体化 |
| `web/static/js/schedule/scheduleintegratedtasks.js` | 556 | Schedule 任务 JS | ACCEPTABLE FOR NOW | 排期任务 |
| `web/static/js/ui.js` | 775 | UI shared | ACCEPTABLE FOR NOW | 共享 UI 工具 |
| `web/templates/dept/edit.html` | 513 | Dept edit template | ACCEPTABLE FOR NOW | 单页面 |
| `web/templates/schedule/index.html` | 1176 | Schedule index template | ACCEPTABLE FOR NOW（Wave 6 前端 consolidation 若真实 capability boundary 阻碍维护再顺带拆） | 排期首页（业务复杂） |

> **总体原则**：所有超长文件 = **ACCEPTABLE FOR NOW**。文件拆分**不进入任何独立 Wave 治理**。只有在修复真实 P0/P1/P2 根因过程中，若该文件结构**确实阻碍维护**，才**顺带**拆。

## 11. Test Coverage Gap Matrix

| Module / Capability | Current Test | Missing Contract | Risk | Suggested Test Type |
|---|---|---|---|---|
| po/service.Home | home_test.go (basic) | 阶段计数准确性、KPI 准确性 | High | Integration (DB) + Contract SQL |
| po/service.TodoList | servicetodo_test.go | 7 维 AND 公式 SQL 化后需回归 | High | Integration |
| po/repo.FindNotices | reponotice_test.go | QuickView / Items 一致性 | High | Integration |
| po/repo.FindTodoItems | repotodo_test.go | 边界 case（keyword LIKE escape, NULL filter） | Medium | Unit + Integration |
| po/repo.FindDoneActions | repodone_test.go | time range / 7 bucket | Medium | Integration |
| schedule/repo | priority_test.go + searchkeyword_test.go + stagefilter_test.go + servicebizdemandfilter_test.go | window txn, capacity calc | Medium | Integration |
| schedule/service.SaveScheduling | (无) | atomic 写窗口 + story + task | **High** | Integration (required — multi-table) |
| schedule/service.Delete | (无) | owner check | Medium | Unit |
| schedule/repo.Transaction | (无) | rollback behavior | Medium | Integration |
| user/service.BatchCreate | (无) | exists 优化 + 事务 | High (P1-USER-01) | Integration |
| user/service MD5 compat | (无) | encode.MD5 兼容 ZenTao | High | Unit (input/output table) |
| dept/service.Update | (无) | ancestor cascade | High (P1-ARCH-01 修后) | Integration |
| login/service.Logout | (无) | session clear | Medium | HTTP integration |
| menu/service | (无) | CRUD 链路 | Low | Unit |
| role/service.AssignPerms | (无) | permission matrix 写入 | Medium | Integration |
| operationlog | (无) | 写入/读取 | Low | Integration |
| loginlog | (无) | 写入/读取 | Low | Integration |
| debug/sqlperf | (无) | 文件解析 / filter | Low | Unit |
| pkg/zentao (URL etc) | url_test.go | ZenTao URL 拼接 | Low | Unit |
| pkg/perm | (无) | capability 解析 | Medium | Unit |
| pkg/ratelimit | (无) | limit / window | Low | Unit |
| pkg/session | (无) | session lifecycle | Medium | Integration |
| pkg/sqllog | (无) | log 解析 | Low | Unit |
| pkg/upload | (无) | 上传 | Low | Integration |

## 12. Frontend Duplication Review

| Capability | Stable | Duplication Sites | Verdict |
|---|---|---|---|
| fetch / error / 401 handling | YES | `app.js` (shared) + `web/static/js/po/*` + `web/static/js/schedule/*` 共 12 处 direct fetch | SHOULD EXTRACT (Wave 6) — 同质代码，应统一走 `app.js` 共享层 |
| CSRF token | YES | (隐含在 shared fetch) | OK |
| toast | YES | (shared) | OK |
| modal | YES | (shared) | OK |
| pagination | YES | 5 个 `LOCAL_PAGINATION` advisory findings | SHOULD EXTRACT (Wave 6) — 当前重复 renderPagination |
| escapeHtml | YES | 7 个 `LOCAL_ESCAPE_HTML` advisory findings | SHOULD EXTRACT (Wave 6) |
| loading/empty/error | YES | shared | OK |
| form | YES | `web/static/js/components/form.js` | OK |
| new window | N/A | 12 个 `NEW_WINDOW` advisory findings | REVIEW — 当前都是产品决策？需手工确认 |
| dropdown | YES | shared | OK |

**Frontend Rule 守住：不要 Framework 重写。** 只抽 stable duplication（pagination / escapeHtml / fetch wrapper），不做大型重构。

## 13. Dependency / Portability Review

| Issue | File | Severity | Risk if Fix | Recommended Direction |
|---|---|---|---|---|
| `replace workbench => /home/wds/repo/workbench` | `go.mod:71` | P2 (P2-PORT-01, **Deferred P2 Backlog**) | 中 — 删除后需验证 CI runner 仍能 build | 后续 portability 任务处理（不在 Phase 3 主 Wave） |
| 明文凭证 in git history | `configs/config.yaml` | P0-CANDIDATE-SECRET（凭证有效性未验证） | 高 — 改 config 不影响凭证；改 history 需重写或 BFG | Wave 0B（user-assisted）: Hermes 提供 rotation checklist；实际 rotate 由用户执行 |
| GitHub Actions runner 假设 | `.github/workflows/quality.yml` | OK | — | 继续 |
| Module 路径 | `module workbench` | OK | — | — |
| go version | `go.mod` go directive | OK | — | — |

## 14. Governance Gate Gaps

**本阶段不修。只记录。**

### GOVERNANCE-GAP-01 / Cross-Repo Transaction Pattern 未定义

- 当前 Phase 1 Rules 明确禁止发明 UnitOfWork / TransactionManager；但同时没说"如果真有跨 Repo 原子需求怎么办"。
- 现状：所有 5 处 schedule `s.repo.Transaction` 都是单 Repo；dept 用 `s.repo.db` 越权（应该改回 Repo 内部事务）；user BatchCreate 无事务。
- **实际跨 Repo 原子需求 = 0**（NOT VERIFIED — runtime verification required for future modules）。
- 建议：保持当前规则不变；如果 Phase 3 出现真需求，单独建一个 PR 讨论。
- **不在本阶段修。**

### GOVERNANCE-GAP-02 / Architecture Scanner 覆盖盲区

- `check-architecture.sh` 当前检测：
  - Handler: `gorm.io/gorm` / `database/sql` / `.db` / `.DB`
  - Repo: `gin-gonic/gin` / `*gin.Context`
  - Service: `.db.(WithContext|Table|Model|Raw|Exec)` / `*gorm.DB`
- 潜在盲区：
  - Service 内 `.db.Create / Find / First / Take / Save / Updates / Delete / Count / Row / Rows`
  - Service 直接 `import "github.com/gin-gonic/gin"`
  - Handler 直接绕 Service 调 Repo (如直接调用 repo package)
- 当前代码真实违反数 = 0（除 dept service.go 1 处已 baseline）
- **是否值得加 scanner？** 取决于 Phase 3 P1-ARCH-01 修复后是否仍出现同样问题。如果 dept 修后全绿，**不立即加 scanner**；如果未来再现，加 scanner。
- **不在本阶段修。**

### GOVERNANCE-GAP-03 / SQL String Interpolation Detector 缺失

- `database.md:35` 禁止，但 scanner 没覆盖。
- 当前代码真实违反数 = 0（所有 LIKE 用参数化 `%` + `?` + `%`）
- 建议：保留 prose 规则；如果 Phase 3 出现真实 SQL injection，加 scanner。
- **不在本阶段修。**

## 15. Do Not Fix

**关键：明确不修的代码（防止 Phase 3 AI 看到不漂亮就重构）。**

1. **ZenTao 兼容 MD5 password 写**（user/service.go:74,144,188 + encode/encode.go）
   - **原因**：ZenTao 真实 schema 决定；破坏它 = 现有 ZenTao 用户无法 login。
   - **不修**。但加注释 + 新建 Workbench-owned 表时必须用 bcrypt/argon2。

2. **`internal/module/schedule/form.go` 1126 行**
   - **原因**：纯 DTO 集合（54 struct），**一个能力**（排期模块请求/响应结构）。机械拆 = 拆出 54 个文件 = 反可维护性恶化。
   - **不拆**（**继续 Do Not Fix**，不进入任何 Wave）。

3. **`web/static/css/components/components.css` 等 shared CSS / JS**
   - **原因**：本身就是 shared 主题，多 CSS file 拆分要等 Phase 3 真正做 frontend consolidation 时一起做。
   - **不拆**。

4. **`internal/module/po/servicetodo.go` 7 维 AND 公式内存分页**
   - **原因**：业务规则来自 PRD V10.1 02 节（注释明确）；这是真实业务复杂度。
   - **不立即全部 SQL-side 化**。Phase 3 Wave 3 可在 runtime evidence 支持下 partial SQL-side 化（push 可等价部分到 SQL，剩余模糊业务规则保留 Go）。

5. **`ScheduleModule` 跨模块 Service 调用**（po → schedule）
   - **原因**：in-process orchestration 合适；拆 RPC = distributed transaction 债。
   - **不拆**。

6. **ZenTao 表 legacy semantics**（`deleted`, `fromDemand`, `action`）
   - **原因**：ZenTao schema 决定；Workbench 不能改 ZenTao 表语义。
   - **不改**。

7. **`scripts/check-patterns.sh` 当前 Advisory 规则不全**（IN_MEMORY_PAGINATION / ROUTE_PERMISSION_REVIEW / WEAK_PASSWORD_HASH / NEW_WINDOW 等都是 advisory）
   - **原因**：这些规则都需要 context 判断，advisory 是正确选择；硬编码为 hard 会误报。
   - **不立即改 hard**。但 Phase 3 可以加"模块 missing-RequirePerm 数量"作为硬上限（Claude Opus 4.8 提议）。

8. **`make check` 当前 8 个 gate**
   - **原因**：覆盖了真实机器可判断的问题；`-race`、`-cover`、SQL EXPLAIN 等需要 runtime 或 CI 才能跑，不在本地 gate 范围。
   - **不加**。

9. **`dept/service.go` callback 内部 `tx *gorm.DB`**（在修复 P1-ARCH-01 时一并改为 Repo callback 即可）
   - **不单独修**。

10. **`internal/module/po/repokpi.go` 4 个独立 CountKPI 方法**
    - **原因**：每个方法自身是 SQL aggregate，单元测试容易。Phase 3 Wave 2 可以合并为 1 个多 metric SQL，**但当前实现不算错误**。
    - **不立即合并**。

## 16. Phase 3 Governance Waves

**重要**：Wave 顺序**基于真实 audit findings 决定**，不是固定模板。

> **Wave 顺序原则**（来自本审计发现的根因结构）：
> 1. 修 P0 安全漏洞（debug auth）；
> 2. 冻结 Schedule capability 模型 + 补齐 RequirePerm；
> 3. 修架构越权 + 事务原子性（dept / user）；
> 4. 性能 / Query Plan（先采集 runtime evidence，再做决策）；
> 5. 业务规则真源整理（仅处理已证明重复定义的）；
> 6. 按风险补测试（非 coverage %）；
> 7. 前端 stable duplication consolidation。
>
> **不设独立"文件治理 Wave"**：文件拆分不是独立治理目标，只有在修复真实 P0/P1/P2 根因时若责任边界确实阻碍维护，**顺带**拆。

| Wave | Goal | Findings | Root Cause | Scope | Dependencies | Risk | Acceptance | Do Not Touch |
|---|---|---|---|---|---|---|---|---|
| **Wave 0A — Debug Authentication Emergency** | 给 `/debug/sqlperf` 与 `/debug/sqlperf/requests` 加 RequireLogin 中间件，修复 P0-AUTH-01 | P0-AUTH-01 | debug 模块 boot 时挂在根 group，无 RequireLogin / RequirePerm | `internal/server/routes.go` (debug group 加 RequireLogin); `internal/middleware/auth.go` 复用现有 Login 中间件；必要的 auth regression test | 无 — Wave 0A 独立执行 | 中 — 收紧访问；可能影响开发体验 | anonymous 请求 → 401 / login redirect（按现有 HTTP contract）；authenticated developer / admin 可按既有策略访问；debug 数据语义不变；`make check` green | 业务代码（除 auth middleware 注册）; Rules / AGENTS.md / Phase 1 baseline; secret rotation（归 Wave 0B）; go.mod（归 Deferred P2 Backlog） |
| **Wave 0B — Credential Remediation (user-assisted)** | 协助用户识别凭证风险并提供 rotation checklist；**不自行 rotate / 不自行写凭证 / 不声称外部 rotate 完成** | P0-CANDIDATE-SECRET | `configs/config.yaml` 含明文凭证（3 fingerprint baseline）；git history 已有 commit；凭证是否仍有效未验证 | Hermes 可做：①识别需轮换的 field（仅 path/field name，不输出 value）；②给 placeholder / env 化配置建议；③验证 config loader 支持 env 注入；④检查 git history exposure（commits / tags）；⑤给 rotation checklist。**Hermes 不可以做**：①猜测或生成新凭证；②自行声称已完成外部系统 rotate；③未经用户确认修改生产凭证；④未经用户授权 rewrite git history | Hermes 任务：无依赖；实际 rotate 需用户授权后由用户执行 | 高（影响所有环境） | Hermes 交付 rotation checklist（路径列表 / 字段名 / env 模板 / git history 处理建议）；用户后续执行实际 rotate 并确认；rotate 后 Phase 1 secrets baseline 同步更新（**由用户授权后**） | 任何真实凭证内容；任何未经授权的外部 rotate 操作；git history 改写（除非用户明确授权）；Rules / AGENTS.md / Phase 1 baseline（rotate 不影响 baseline 自身） |
| **Wave 1 — Schedule Permission Model** | 先冻结 capability 模型（候选粒度：read/list / scheduling write / window manage），再补齐 15 个 route 的 RequirePerm | P1-AUTH-02 | Schedule 模块历史上没强制 capability 路由权限；当前 RequireLogin group level 仍在生效 | `internal/pkg/perm/perm.go` (加 Schedule* capabilities); `internal/module/schedule/handler.go` (15 routes RequirePerm); 产品/业务对 capability 粒度的确认 | Wave 0A 完；产品/业务授权 | 中 — 加 RequirePerm 是收紧 | 未授权用户访问 /schedule/* 写操作返回 403；admin UI 验证；业务功能回归测试 | **不顺带拆 user/handler.go**（P2-FORM-03 不是 Wave 1 目标）；业务代码（除 auth）; Rules / AGENTS.md / Phase 1 baseline |
| **Wave 2 — Architecture & Transaction Correctness** | 修 P1-ARCH-01（dept 越权直持 `repo.db`）+ P1-USER-01（BatchCreate 事务原子性 + 现有 schema constraint 验证） | P1-ARCH-01, P1-USER-01 | dept 模块 ancestor cascade 实现时未走 Repo 内部事务；user BatchCreate 缺事务 + schema constraint 未验证 | `internal/module/dept/repo.go` (加 UpdateWithAncestor 等事务方法); `internal/module/dept/service.go` (改调用); `internal/module/user/service.go` BatchCreate 包事务; `internal/module/user/repo.go` (加一次性 batch conflict query) | Wave 1 完（避免双改 auth 影响 verification） | 中 | dept Update ancestor cascade 与原行为一致；user batch 在并发下不产生 duplicate user；目标表 ownership 与 unique constraint 已验证（**不预设方案**） | ZenTao schema 修改（未经授权不得修改）; 业务代码（除 dept / user service + repo）; Rules / AGENTS.md |
| **Wave 3 — Query Plan & Performance** | 先采集 runtime evidence（real row count, EXPLAIN, slow query, representative dataset），再处理 Home / Todo / Notice / Window / 字典查找 | P1-PERF-01, P1-PERF-02, P1-PERF-03, P1-PERF-04 | 业务复杂度 + 当前实现 trade-off（见各 finding） | `internal/module/po/reponotice.go` + `servicenotice.go`; `internal/module/po/repovaluestream.go` + `servicefollow.go`; `internal/module/po/repokpi.go`; `internal/module/schedule/service_window.go` + `repo.go`; account display map 加载机制 | Wave 2 完 | 中-高（SQL 改易破坏契约） | runtime evidence 已采集（real row count / EXPLAIN / slow query / trace）; 业务契约不变（回归测试通过）; query count / latency 相对 measured baseline 有改进; 无新增 N+1 / 无界读取; `make check` 全绿; 相关回归测试通过 | **不预先承诺 cache 方案 / CTAS / 固定 latency threshold / 固定 LIMIT / 固定时间窗口**; **不预先决定 `ctx.Value` 字典缓存实现**; **不预先决定 SQL aggregate / pagination 拆分阈值**（先 evidence 再方案）; 业务代码（除非相关 SQL）; Rules / AGENTS.md |
| **Wave 4 — Business Rule Truth Consolidation** | 只处理已经证明存在重复定义的业务规则；不新建配置表除非业务明确需要 runtime configurable rule | P2-BR-* | 业务规则在不同 layer 重复定义 | PO Value Stage truth source（按 §7 目标 + 候选方向）; Window Teamgroup Display 统一为单一可复用 formatting capability（按 §7 候选方向）; account display map 加载机制（如果 Wave 3 未做） | Wave 3 完（依赖其运行时 evidence） | 中 | 业务输出与重构前一致（contract test 锁住）; 相关回归测试通过; SQL / Service / Frontend 不再各自重新定义同一业务语义 | **不新建 Workbench-owned `wb_value_stage` 表**（除非业务明确需要）; **不进行架构抽象**; 业务规则新增; Rules / AGENTS.md |
| **Wave 5 — Risk-Based Testing** | 按 Test Coverage Gap Matrix 的 High / Medium risk 补，不追 coverage % | P2-TEST-01 | 17/22 packages 零测试；当前测试侧重 happy path | `internal/module/po/`, `schedule/`, `user/`, `dept/` 关键 service + repo 测试；`pkg/perm` / `pkg/session` 单元测试 | Wave 1-4 完（修复后补 regression test） | 低 | 按矩阵中 High / Medium risk 全部补齐（不是 coverage %）; 业务规则 / permission / SQL contract 都有 test 保护 | coverage % 数字; 业务代码（除新增 test 文件）; Rules / AGENTS.md |
| **Wave 6 — Frontend Stable Duplication** | 只抽真正稳定重复能力（fetch wrapper / pagination / escapeHtml 等）；不进行 Framework 重写 | P2-FE-* | 多处重复 local 实现（advisory pattern 已记录） | `web/static/js/app.js` + `components/`; `web/static/js/po/*.js`; `web/static/js/schedule/*.js`; 相关 CSS | Wave 3 完（避免双改影响 perf 验证） | 中 | browser E2E 验证; advisory count 显著降低; UI 业务行为不变 | **不重写为前端 Framework**（Vue / React / 组件库等）; **不顺带为行数拆大文件**（P2-FORM-* / P2-CSS-* / P2-JS-* 在此 Wave 仅当存在真实 capability boundary 才拆）; Rules / AGENTS.md |

### Deferred P2 Backlog（不在 Phase 3 主 Wave 内）

| Finding | Title | 处理建议 |
|---|---|---|
| P2-PORT-01 | `go.mod` self-replace | 后续 portability 任务处理：确认 replace 是否 dead → 删除并验证 build → 更新 baseline。**不阻塞 Phase 3 主 Wave 启动**。 |

### Wave 间依赖图

```
Wave 0A (Debug Authentication)               ← 唯一明确 P0，可独立启动
Wave 0B (Credential Remediation, user-assisted) ← 不阻塞 Wave 0A，可并行
  └─> Wave 1 (Schedule Permission Model + capability 冻结)
        └─> Wave 2 (Architecture & Transaction: dept / user)
              └─> Wave 3 (Query Plan: 先 evidence 再方案)
                    └─> Wave 4 (Business Rule Consolidation)
        └─> Wave 5 (Risk-Based Testing, 修复后回归)
Wave 6 (Frontend Duplication) ─独立，可在 Wave 2 之后任何时候启动

Deferred P2 Backlog:
  P2-PORT-01 (go.mod self-replace) ← 不阻塞主 Wave，后续 portability 任务
```

### Wave 0A / 1 必须先做的原因

- Wave 0A：debug 无 auth 是当前代码里**唯一**有明确证据的 P0 安全漏洞。**Hermes 可独立完成**，不需要等待 Wave 0B（credential remediation）。
- Wave 1：Schedule capability 模型冻结是 Phase 3 的关键"必须业务决策点"——必须由产品 / 业务确认粒度才能继续，否则 Phase 3 后续 wave 都可能因 capability 变化而返工。

### 用户授权后才能启动 Wave

- Wave 0A 不需要额外授权即可启动（修复明确 P0）。
- Wave 0B 涉及 secret rotate → 需要用户单独授权（按 quality.md:54）；**Hermes 不会自行声称完成外部 rotate**。
- Wave 1 涉及 capability permission 拆分 → 需要产品/业务确认 Schedule capability 粒度。
- Wave 3 必须先采集 runtime evidence → 需要 staging / prod-like 数据环境（NOT VERIFIED in Phase 2）。
- Wave 5 不强求 coverage %，只按风险补。
- Wave 6 仅抽 stable duplication，不进行 Framework 重写。

### 通用 Acceptance 模板（所有 Wave 适用）

```
- business contract unchanged（业务输出与重构前一致）
- relevant regression tests pass（相关回归测试通过）
- query count / latency improves against measured baseline（相对 measured baseline 有改进；不预设具体阈值）
- no new N+1 / unbounded read（不引入新的 N+1 / 无界读取）
- make check green
- task-specific browser/API verification（任务特定的 browser/API 验证）
```

> **严禁** 使用未经证据支持的硬数字（P95 < 300ms / SQL < 15 / byte-equal / 固定 LIMIT 等）作为 Acceptance criterion。

---

## 17. Phase 2.1 Final Reconciliation（精修核对）

本节为 Phase 2.1 Audit Report Finalization 完成的精修核对。

### 17.1 修正后的 Severity Summary

| Severity | Count | Findings |
|---|---|---|
| **P0** | 1 | P0-AUTH-01（debug 无 auth，唯一明确证据 P0） |
| **P0-CANDIDATE** | 1 | P0-CANDIDATE-SECRET（凭证是否仍有效未验证 → MUST ROTATE；Wave 0B user-assisted） |
| **P1** | 8 | P1-AUTH-02（Schedule capability, 降级）、P1-ARCH-01、P1-PERF-01..04、P1-USER-01、P1-MD5-01 |
| **P2** | 9 | P2-PORT-01（go.mod self-replace，**Deferred P2 Backlog，不进入主 Wave**）、P2-TEST-01、P2-FORM-01..03、P2-CSS-01..03、P2-JS-01 |
| **P3** | 1 | P3-CMT-01 |
| **GOVERNANCE-GAP** | 3 | Cross-Repo Transaction / Architecture Scanner 盲区 / SQL Interpolation Detector |

> **口径一致性**：`configs/config.yaml` 明文凭证由 `P0-CANDIDATE-SECRET` 单一承担；不再有独立的 P2-ENV-01。

### 17.2 最终 Phase 3 Wave 顺序

```
Wave 0A — Debug Authentication Emergency             ← 唯一明确 P0，可独立启动
Wave 0B — Credential Remediation (user-assisted)      ← 不阻塞 Wave 0A，可并行
  └─> Wave 1 — Schedule Permission Model (capability 冻结 + RequirePerm)
        └─> Wave 2 — Architecture & Transaction Correctness (dept / user)
              └─> Wave 3 — Query Plan & Performance (先 evidence 再方案)
                    └─> Wave 4 — Business Rule Truth Consolidation
        └─> Wave 5 — Risk-Based Testing (修复后回归)
Wave 6 — Frontend Stable Duplication ─独立

Deferred P2 Backlog:
  P2-PORT-01 (go.mod self-replace)                    ← 不阻塞主 Wave
```

**Wave 0A / 1 必须先做**（唯一明确 P0 + 关键业务决策点）。

### 17.3 相比原报告删除的 speculative recommendation

| 类别 | 删除项 |
|---|---|
| 性能方案 | `30s cache` / `CTAS` / `P95 < 300ms` / `SQL count < 15` / 固定 LIMIT 5000 / 固定时间窗口（如 `releaseDate >= now - 3 month`） |
| 实现细节 | `ctx.Value("displayMap")` 字典缓存实现 / `wb_value_stage` 新表 / `pkg/format.TeamgroupDisplay` 固定位置 / Schedule capability 三段式粒度（仅作为候选，未预设） |
| 默认方案 | `加 unique index 双保险`（改为待 schema ownership 验证后决定）/ `byte-equal` Acceptance |
| Wave 设计 | `Wave 7 — Oversized File Selective Split` 独立 Wave（删除）；Wave 0 拆分后**只**包含 debug auth（Wave 0A）+ credential remediation user-assisted（Wave 0B） |
| go.mod 错误描述 | "build baseline 不稳" / "必须 Wave 0 先修" / "GitHub runner 可能存在等价路径" / "Wave 0 删除 replace" |
| Auth 错误结论 | Schedule 升 P0（无业务证据 → 保持 P1） |
| 收口口径冲突 | 同时 P0 secret + P2-ENV-01（统一为 P0-CANDIDATE-SECRET，P2 不重复计数）；P2-PORT-01（go.mod）移出主 Wave → Deferred P2 Backlog |
| Secret 假完成 | Wave 0B 明文禁止 Hermes 自行 rotate / 生成凭证 / 声称外部 rotate 完成；只能交付 checklist，用户执行实际 rotate |

### 17.4 阻塞 Phase 3 的审计矛盾

**无**。

所有修正后：
- Findings Summary P0/P1/P2 数量与正文一致 ✅
- Security Matrix 与 Findings Summary 一致 ✅
- Do Not Fix（schedule/form.go 等）与 Wave Scope 不冲突 ✅
- 不存在同一个问题同时 P0 和 P2（secret 由 P0-CANDIDATE-SECRET 单一承担）✅
- 不存在 "NOT VERIFIED" 同时又给出硬性能指标 ✅
- 不存在修改 ZenTao schema 的未经授权建议 ✅
- 不存在为了文件行数拆分的 Wave ✅
- `go.mod` 全文只出现 P2-PORT-01（已在 Findings / P2 表 / Deferred P2 Backlog 统一）✅
- Wave 0A / 0B 明确分工，Wave 0A 不需等待 0B ✅
- Secret rotation 不被 Agent 假完成（Wave 0B 仅交付 checklist）✅
- 不修改任何业务代码 ✅

---

## Phase 2 Audit Baseline Frozen.
## Phase 3 Wave 0A is authorized to start.

Phase 2 审计阶段结束。**不进入 Phase 3**。Wave 0A 启动授权由用户决定；Wave 0B 必须用户单独授权实际 rotate 操作。

1. **真实 EXPLAIN** — 需要真实 DB 环境（NOT VERIFIED — RUNTIME EVIDENCE REQUIRED）
2. **真实数据量** — 需要 SELECT COUNT(*) 看各表行数
3. **真实 runtime profile** — 需要真实流量或 staging 环境
4. **Browser E2E** — 需要 headless browser + 真实 fixture
5. **GitHub Actions runner 上 `make check` 删除 replace 后是否能 build** — Wave 0 需要 CI 验证

这些必须在 Phase 3 各 wave 内执行，本审计阶段不执行。
