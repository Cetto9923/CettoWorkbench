# GLM 代码审查修复 Plan · 2026-09-10

> 基线报告：`code-review-2026-09-10.html`（HEAD `49801427` + WIP）
> 仓库：`workbench-claude-po` · 分支：`release/po-integrate-main-202609`
> 原则：**关注页产品语义先交付；工程债分波；P0 以「真阻断 make check / 真数据风险」为准，不按报告字面全标紧急。**
> 约束：不改 `docs/PRD/`；不 force-push；共库不擅自 DDL；改密 MD5 必须先决策再动。

---

## 0. 总策略与并行轨

```
轨 A（产品）  我的关注：业需默认未关闭 + 周报 mine（参与∪关注）
轨 B（门禁）  make check 变绿（单测夹具/断言 + 行数债处置）
轨 C（共库写）排期事务上限 + 锁序统一 + story 归属谓词
轨 D（清债）  死代码/兼容路由/小正确性/UI 漏项
轨 E（性能）  看板指标扇出、已办批量化（不挡 A）
```

**顺序建议**

| 周次 | 主做 | 可并行 |
|------|------|--------|
| W0（立即） | 轨 A 产品提词落地（另文 `follow-page-gemini-prompt`） | 轨 B 单测定位 |
| W1 | 轨 B 门禁绿 + 轨 D 快清项 | 改密决策纪要 |
| W2 | 轨 C 排期写路径 hardening | 轨 E 方案设计 |
| W3+ | 轨 E 落地 | advisory 批量确认 |

**WIP 纪律**：工作区未提交 CSS/notice/分页样式 **不得** 与轨 A/C 混 commit。样式 WIP 要么独立 commit「chore(ui): …」，要么先 stash。

---

## 1. 轨 A — 我的关注产品（不属 GLM 报告，但排在最前）

详见：`.cursor/plans/follow-page-gemini-prompt-20260910.md`（及聊天中的加硬版提词）。

**完成定义（摘要）**

- 业需：默认未关闭；删「重点关注 / 未关闭+重点 / 未关闭·正常推进」一级筛；星标仍走禅道 ajax + `appFetch`。
- 周报：默认 `scope=mine`（PM ∪ `zt_team` 项目成员 ∪ follow）；删「状态关注」、周期空下拉、「本周投入」列。
- 验收：`:8090/follow` + `<本地测试账号>`；**不 push 除非产品主确认**。

**与 GLM 交叉注意**

- `servicefollow.go`（503 行）、`zentao/client.go`（553）、`follow.css`（691）已超行数门禁；轨 A 若继续涨行，**同 commit 内必须拆文件或登记基线**，否则轨 B 永远红。

---

## 2. 轨 B — 门禁恢复（对应报告 §2）

### B1. 前端单测变绿（报告标 P0）

**现象**：`make check` → `check-frontend-test` 失败，指向 `tests/unit/frontend/home-list-caption.test.js`。

**注意（复查结论）**：当前 WIP 夹具**已加载** `home-render.js`（约 L64），报告「只加载 home.js」可能过时。执行时：

1. 干净跑：`node tests/unit/frontend/home-list-caption.test.js`（或仓库既有 frontend test 脚本）。
2. 若仍失败：对照断言「共 100 条」/「已筛选」与 `home.js`/`home-render.js` 在默认 `focus=my_action` 后的 caption 行为，**改实现或改断言二选一**，禁止删测试蒙混。
3. 扫描其它 frontend unit：凡拆文件后夹具漏加载兄弟脚本的，一并补齐。

**验收**：`make check` 中 `check-frontend-test` PASS。

### B2. 行数门禁（MUST ≤500）

| 文件 | 约行数 | 动作 |
|------|--------|------|
| `internal/module/po/servicefollow.go` | 503 | 拆：列表查询 / SetDemand / 展示映射 至少两文件 |
| `internal/pkg/zentao/client.go` | 553 | 拆：token+HTTP 基础 / demand review·clarify / follow ajax |
| `web/static/css/po/follow.css` | 691 | 拆：`follow-weekly.css` + `follow-demand.css`，模板分别引用 |
| `web/static/css/metrics.css` | 683 | 拆或登记 debt 基线（若本迭代不碰 metrics，优先 **更新 quality 基线 + 决策记录**） |
| layout/components/schedule/ui.js 超基线增长 | — | 本迭代未改则 ratchet 基线；若本次 WIP 导致增长，拆或回退无关 diff |
| WIP：`demand-detail.css` / `personal-workspace.css` / `notice.js` | 过线 | **提交前拆分**；禁止把 WIP 涨行混进功能 commit |

**验收**：`check-file-length` PASS，或有书面 debt + 精确基线更新（走 quality.md 流程）。

### B3. 架构门禁

| 项 | 动作 |
|----|------|
| `testtask/handler_test.go` 直连 gorm | 夹具迁 repo 测或按门禁登记基线 |
| `dept/service.go` Service 持 DB 开事务 | 下沉 `repo.Transaction`（可 P1 单开，不挡关注页） |

**验收**：`check-architecture` PASS 或基线条目与决策一致。

### B4. 改密 MD5（报告 P1，需决策）

**冲突**：`profile/service.go` 新密码写 `encode.MD5`，与 AGENTS 规则 7 MUST 冲突；共库 `zt_user.password` 又要求与禅道一致。

**决策选项（选一，写入 ADR/决策纪要再改代码）**

| 选项 | 做法 | 风险 |
|------|------|------|
| D1 维持兼容 | 明确「禅道共库豁免」写进 AGENTS/质量债；advisory 可 suppress 并注释指向决策 | 合规债永久 |
| D2 走禅道改密 API | 工作台只调禅道接口，本地不写 hash | 依赖禅道可用性 |
| D3 双写/迁移新哈希 | 需禅道侧同步，**禁止单边改** | 高，本迭代不做 |

**本 Plan 默认建议**：先出 **D1 或 D2 决策纪要（半页）**；代码改动挂 W1 末或 W2，不堵轨 A。

---

## 3. 轨 C — 共库写路径 hardening（报告 §3 DB-R1～R4）

> 执行代理：Codex/Gemini；需熟悉 `internal/module/schedule`。
> 死锁项标注「需 DB 证据」：修完后在隔离 MySQL 补 reverse-order 用例（database.md 规则 8）。

### C1. 排期保存批量上限（DB-R1）— P1

**文件**：`service_scheduling_save.go`、`form.go` Validate、`service_story_tasks.go`

**改动**

1. `Validate`：`len(Stories) ≤ N`、`每 story 的 Tasks ≤ M`（建议先 N=50、M=50，用现网/演示数据量校准后写入注释依据）。
2. 超限 → **422**，**不开事务**。
3. `SaveStoryTasks` 同样上限。

**验收**：单测超限 422；正常规模保存仍绿；注释写明 N/M 依据。

### C2. 锁序统一 + ID 排序（DB-R2 / DB-R3）— P1

**约定（写进 schedule 模块注释）**：凡同时碰 demand + story，**先 `FOR UPDATE` 目标 `zt_demand`，再按 storyID 升序锁/改 `zt_story`**。

**改动**

1. `SaveScheduling` 事务开头：对涉及的 demandID 先锁。
2. 进事务前对 storyID/taskID **排序去重（canonicalize）**，禁止按客户端数组顺序加锁。
3. 识别 MySQL `1213`/`1205`：返回明确错误码（可重试提示），禁止裸 500。
4. 隔离套件：两事务反向顺序重叠写，断言不死锁或可重试成功。

**验收**：单测 canonicalize；隔离套件用例；代码注释锁序。

### C3. UpdateStory 归属谓词 + RowsAffected（DB-R4）— P1

**文件**：`repo_scheduling_write.go`、`service_scheduling_save.go`

**改动**

1. edit/delete story：事务内锁读复核 `fromDemand`（复用/对齐 `ValidateStoryForTaskMutation`）。
2. `UPDATE` 带 `fromDemand = ?`（或等价归属谓词）；`RowsAffected==0` → 冲突错误，禁止假成功。

**验收**：单测「story 已改挂/已删」返回冲突；快乐路径不变。

### C4. 关注写入竞态（DB-R5）— P2，挂轨 D 更合适

见 §4 D2：删 `SaveDemandFollow` 后，只保留 `EnsureDemandUnfollowed`；用 `ON DUPLICATE KEY UPDATE` **仅当确认唯一键存在**；无唯一键则 **不改共库 schema**，改为短事务内 upsert 或接受极低概率重复并文档化。

### C5. 读写池一致性（DB-R6）— P2

1. 查部署：只读池是否真副本。
2. 若是副本：写后自读（角标、本人关注首页）走 `writeDB` 或响应带回计数。
3. 若同实例：文档标注「无延迟风险」，降级观察项。

---

## 4. 轨 D — 正确性小修与死代码（报告 §4–6、§8）

按「低风险、可平行」打包，建议 **1～2 个 chore commit**。

### D1. 假成功 / 绑定错误（DB-R7、L-1～L-3）

| ID | 改动 | 验收 |
|----|------|------|
| DB-R7 | `FollowSetDemand` 等：非法 actor/参数 **return error**，禁止 `nil` 假成功 | handler 测 4xx |
| L-1 | `handlerreview`：`ShouldBindJSON` 错误 → 400 | 非法 JSON 400 |
| L-2 | `DoneDetail`：`ErrCodeInvalidParam` → 422，非 401 | 与 detail 一致 |
| L-3 | `repo_clarify` ParseUint 失败 skip+log 或返回错误 | 不静默 dept=0 |

### D2. 死代码清理（D-1～D-4、C-1～C-3）

| 项 | 动作 |
|----|------|
| `SaveDemandFollow` | 删除生产方法；`repo_rw_test` 改测 `EnsureDemandUnfollowed` 走 writeDB |
| `homecompact.css` | 确认零引用后删除 |
| `/workbench/api/*`、`/watches/*` | 确认无外部调用后下线路由（或留一行兼容注释+issue） |
| `TodoListReq.tab` / `FollowListReq.tab` | 删除死分支与校验 |
| `html-sanitize` 三份实现 | **本波可不做**；单开「richtext 共享模块」债 |

### D3. UI 漏项（U-1、U-2）

| 项 | 动作 |
|----|------|
| U-1 | `scheduleintegrated.js` / `scheduleintegratedrd.js`：`window.open` 补 `noopener,noreferrer` |
| U-2 | `workboard.js`：硬编码色改语义 token；补 `--orange`/`--t3` 或换现有 token，light/dark 各看一眼 |

### D4. advisory 大清单（U-3）

**不做批量盲改**。新建 issue：「NEW_WINDOW / DIRECT_PAGE_FETCH 人工确认」；仅扫 **本迭代新增行**。

---

## 5. 轨 E — 性能（报告 §7）

> 不挡关注页与排期 hardening。先出方案再改 SQL。

| ID | 原标 | 本 Plan 定级 | 动作 |
|----|------|--------------|------|
| PERF-1 看板指标 5N SQL | P0 | **P1 性能债** | `FindBoardTeamMetrics` 合并为 1～2 条 GROUP BY；成员 IN 批量化 |
| PERF-2 已办 ~12 SQL/页 | P0 | **P1 性能债** | 对象上下文批量；facet/审批可缓存或并入 |
| PERF-3 看板树无 Limit | P1 | P1 | children/stories 补 Limit，契约写上界 |
| PERF-4 函数包裹列 | P1 | P2 | 时间改范围比较；ID 精确优先；FIND_IN_SET 长期债 |
| PERF-5 其它 | P2 | P2 | 记债，不排期 |

**验收**：关键接口 trace/日志 SQL 条数前后对比；页面可感延迟下降；无行为回归。

---

## 6. Commit / 分支 / 验收清单

### Commit 切片（示例）

1. `fix(po): follow defaults — open demands + mine weeklies`（轨 A，可含必要拆文件）
2. `test(po): fix home-list-caption fixture/assertions for make check`（B1）
3. `chore: split oversized follow/zentao files for line gate`（B2，可与 1 合并若同改）
4. `fix(schedule): cap scheduling batch size and unify lock order`（C1+C2）
5. `fix(schedule): story update ownership predicate`（C3）
6. `chore(po): remove dead SaveDemandFollow and compat routes`（D2）
7. `fix(po): harden follow/review/done error semantics`（D1）
8. `fix(ui): noopener + workboard theme tokens`（D3）
9. `perf(po): batch board team metrics queries`（E，可后）

### 每波通用验收

- [ ] `make check` 目标 gate 变绿（或债务基线已更新）
- [ ] 相关 `go test` / frontend unit 有执行记录
- [ ] `:8090` 冒烟：登录 `<本地测试账号>`、首页、关注、排期保存（若碰 schedule）
- [ ] 不提交 `docs/PRD/`、不 push 除非主确认
- [ ] PR/说明里引用本 Plan 章节 ID（如 C2、D2）

### 明确不做（本 Plan 范围外）

- 单边改禅道密码算法 / 共库加唯一键未评审就 DDL
- 借修复扫完全库 NEW_WINDOW 92 处
- `user` legacy 模块大改
- Demo/HTML 原型目录

---

## 7. 给执行代理的最短指令（可复制）

**Wave-B1+B2（门禁）**
> 只修 `make check`：跑并修 `home-list-caption` 直至绿；拆分或基线化超限文件（优先 `servicefollow.go` / `zentao/client.go` / `follow.css`）。禁止改产品行为。WIP 样式单独处理。

**Wave-C（排期写）**
> 按 Plan §3：Validate 批量上限；事务内先锁 demand、storyID 排序；UpdateStory 归属谓词+RowsAffected；补单测与隔离死锁用例。

**Wave-D（清理）**
> 删 `SaveDemandFollow`（测改 Ensure）；假成功改真错误；DoneDetail 422；noopener×2；确认后删 homecompact 与死路由。

**Wave-A（关注产品）**
> 只用加硬版 follow 提词；禁止只改 CSS；做完按提词验收清单自测。

---

## 8. 决策待办（需要你拍板的）

1. **改密**：D1 豁免文档 vs D2 禅道 API？（建议本周定）
2. **metrics.css / 超基线 CSS**：本迭代拆 vs 只更新 debt 基线？
3. **兼容路由 `/watches/*`**：有无外部系统依赖？无则删，有则保留并注明消费者。
4. **轨 A 与轨 B**：若关注改造必涨行，是否允许「功能 commit 内强制拆文件」为默认（建议：是）。

---

## 9. 进度跟踪表（可打勾）

| ID | 项 | 状态 |
|----|----|------|
| A | 关注页产品语义 | ☐ |
| B1 | 前端单测绿 | ☐ |
| B2 | 行数门禁 | ☐ |
| B3 | 架构门禁 | ☐ |
| B4 | 改密决策纪要 | ☐ |
| C1 | 排期批量上限 | ☐ |
| C2 | 锁序+canonicalize | ☐ |
| C3 | Story 归属谓词 | ☐ |
| D1 | 错误语义 | ☐ |
| D2 | 死代码/路由 | ☐ |
| D3 | noopener/token | ☐ |
| E1 | 看板指标批量化 | ☐ |
| E2 | 已办上下文批量 | ☐ |
| E3 | 看板树 Limit | ☐ |
