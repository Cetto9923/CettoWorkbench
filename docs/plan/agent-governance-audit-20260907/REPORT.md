# Agent 规范、架构与共享数据库审查

日期：2026-09-07，Asia/Shanghai。范围：近期提交及当前 WIP 的检查、工程规范更新。
应用状态：**BLOCKED BY EXISTING BASELINE**。本轮只写规范/报告，没有修业务代码、修改门禁脚本或数据库、启动/重启服务、提交或推送。

## 1. 结论与证据边界

Go/Gin/GORM/MySQL SSR 单体和 Handler → Service → Repo 的总体方向合理，当前没有拆微服务或重写前端的证据。主要问题是现有规范执行不完整、部分代码与注释相矛盾、门禁覆盖不足，以及共享写库的并发合同缺失。增加更多口号不能替代可执行测试、独立审查与部署权限控制。

- 仓库：`workbench-claude-po`；分支：`Claude-PO`；观察 HEAD：`a430311a6d46d99a12d821180efbb3b1854d9d39`。
- 检查了最近约 35 条提交索引，重点追踪 PO 首页/列表/详情/主操作、schedule 写入、DB/bootstrap、质量脚本和各 Agent 入口；这是聚焦风险审查，不是所有模块逐行认证。
- 开始时存在大量 tracked/untracked WIP。审查期间至少 51 个已捕获文件的内容发生外部变化，大量 tracked diff 消失但新增文件保留；随后还出现新的文件。HEAD 没变不能代表源码没变。
- 本 Agent 没有在项目工作区执行 reset、restore、checkout、clean、stash 或覆盖业务文件；门禁自测的临时测试仓库操作不涉及用户工作区。不能从源码变化归因某个具体 AI 工具，更不能据此断言其故意绕过规则。
- 以下区分“稳定源码事实”“观察期 WIP 问题”“条件性运行风险”。业务行为并未在真实登录浏览器重验；没有读取真实业务库，没有宣称出现了实际死锁。

证据：初始文件指纹 `/tmp/wb-governance-20260907-input-hashes.json`；后续只读源码副本 `/tmp/wb-governance-20260907-source-snapshot/manifest.json`（395 个文件，排除运行配置/凭据）。副本仅供复核，不是新开发 checkout。临时证据可能被系统清理。

## 2. 已有规则被越过／未执行到位

### G01 · P1 · 混合 WIP 已破坏编译与验收一致性

首次 `make check`：`sidebar_badges.go:41/45` 调用不存在的 `CountRecentDone`、`CountUnreadNotices`，PO/server 等包编译失败。外部变化后，`form.go` 与未跟踪 `form_done.go` 同时声明 DoneTab/TimeRange，vet 报重复声明；后续 Make 又先被新增测试格式拦截。

这证明观察到的工作树不能交付，不能用上一轮 CI 或运行中的旧二进制说明“当前代码没问题”。不把开发中的暂态错误描述为已经发布的回归。最小处理：由当前实现任务核对其文件集合与重复声明、完成迁移，再对一致的最终源码重跑门禁。本审查不删除任何 WIP。

### G02 · P1 · 权限 helper 忽略参数，以登录替代能力授权

`internal/module/po/service_primaryaction.go:211` 的 `hasCapability(actor, _ ...perm.Permission)` 不消费权限参数；普通非空账号即 true。`:187–194` 用它生成审批、澄清、排期、提测、催办、交付、评价的能力标记。注释声称“最小保守放行”，与代码相反。

确认的是 helper/操作可用性合同错误。没有证明全部写接口可越权，因为目标接口可能仍有独立权限检查，且当前 WIP 不可编译。最小处理：接入现有真实 capability 来源，区分页面读权和动作权，Service 再验对象/状态；补已登录但未授予、撤销、跨对象、nil actor 用例。禁止新造权限码或仅修改注释。

### G03 · P1 · 新测试没有纳入统一入口，失败被漏掉

`Makefile:24` 的 frontend target 仅列 4 个测试。实际目录有 10 个；另外 6 个不会被该 target 执行。隔离源码副本中直接运行全部 10 个，**5 通过、5 失败**：

- `home-focus.test.js`：找不到焦点 click handler；同期模板重新含 `/todos?focus=...`，JS 不再传递 focus。源码与本会话明确“首页内筛选”的要求不符。
- `html-sanitize.test.js`：`R.sanitizeRichText` 不是函数；拆文件后测试导出合同失配。不能仅凭此异常断言净化成功或漏洞已被利用。
- `navigation-primary-action.test.js`：首页标题新窗口断言失败。
- `notice-filters.test.js`：两次处理函数调用产生两个 mutation 请求。还需修复后继续执行其后续断言；不要把未执行断言视为通过。
- `priority-helpers.test.js`：预期 helper 不存在，得到 undefined。

最小处理：修复真实行为或已确认变更后的测试加载合同，将新增测试注册到统一入口。禁止删除断言、移出 runner 或仅更改名字得到绿灯。本轮未修改 Makefile/测试。

### P01 · P1 · 首页依然全量加载 ID，再在 Go 中去重/分页

`servicefollow.go:65` 的 `countAllStageUniq`、`:112` 的 `listAllStageDemands` 按阶段调用 `FindRoleDemandIDs` 等，加载所有命中 ID，最后 `allRefs[offset:end]`。Repo 的 ID 查询没有分页上限。“只取 ID”减少列宽，不能消除随全部数据量增长的内存和扫描成本。

`service.go:85–158` 先分阶段统计，再查各阶段 ID 求并集，再做 4 个 KPI：按当前 9 个具体阶段和 2 个独立故事分支，成功路径仅这些部分就约 26 次 SQL，未包含版本窗口和页面布局开销。这是静态调用计数，不是实测延迟。

违反原有查询规则，不需要先发明新架构。最小处理：冻结同一个候选集/阶段去重语义，将 count/page 下推 SQL，页内批量补充。审核 SQL 扫描量和执行计划，不能只把循环改成并行。

### P02 · P1 · “批量／避免 N+1”注释与逐 ID 查询相反

`service_primaryaction.go:143–149` 在 storyIDs 循环调用 `findStoryMetaForAction`，后者 `:271` 发 SQL；一次测试单聚合加 N 次对象查询。它还把任何单行错误折叠成 None，无法区分断库与不可见对象。文件头却明确声称避免 N+1。

最小处理：当前页 ID 批量投影；区分业务不存在/不可见和基础设施错误。缺少实际消费者的 helper 不能称为已接入全站。SQL 仍由 Repo 实现，Service 只派生操作规则。

### S01 · P2 · 文件职责、声明和目录说明存在漂移

- `service_primaryaction.go:266` 实际声明 Repo receiver 并查询 DB，却放在 Service 文件，触发架构门禁。应移到合适 Repo 文件；这与“Service receiver 直接查询”是两种不同问题。
- `service.go:53–59`（早期 WIP）通过 `repo.db` 构造 DetailService，后者持有 parent Service：依赖方向难以独立测试，应在既有 composition boundary 组装，避免为此发明通用框架。
- `po_work_scope.go` 声称“唯一真源、三个页面统一消费”，搜索仅有自身调用；首页实际上另写 `roleDemandBase`。这是未完成接入/重复规则风险，不是已完成统一。
- `repokpi.go` 尾部残留 FindTodoItems 注释，`repo.go` 的“依赖：无”与 GORM import、只读失败注释与当前调用行为不一致。不要继续批量复制长模板注释。
- 早期 WIP `repo_detail.go` 达 620 行；外部变化后为 315 行，不可当成当前 620 行。后续独立门禁报告 `repodone.go` 504 行、`internal/pkg/render/render.go` 512 行的新超限。已存在的大文件基线不是新超限授权。

最小处理：按真实 capability/layer 拆分，连同调用方/导出/脚本顺序和测试一起迁移；不按行数机械分段，不通过压缩代码或扩大基线过关。

### S02 · P2 · 历史文档和工具记忆易误导后续模型

`debt.md` 仍描述“无集成测试、排期无 capability、存在 tracked 明文”等历史状态；源码及扫描已发生变化。`module-index.md` 的 schema 标记未说明 V0 已有指定实例证据。`.workbuddy/memory/` 中的已实现行为优先说明不能自行扩大为工程安全规则让位于代码。

本轮给历史清单加时效说明，链接当前审查和 V0 实例证据。保留历史原文，避免未经验证就批量关闭债务。规范明确：已确认业务决策按具体范围生效，代码注释、记忆和旧报告本身不能新增授权。

## 3. 禅道共享库：确认的是哪些风险

普通 InnoDB 一致性 SELECT 在常见 RC/RR 下通常不持行锁，但并不意味着读路径无影响。重查询会争用 CPU/I/O/连接；长快照影响 purge；locking reads、DML 和 metadata locks 要分别分析。

### C01 · P1 风险 · 多对象锁顺序取决于客户端顺序

`schedule/service_scheduling_save.go:76–94` 按 `req.Stories` 顺序变更 story，再处理 tasks、最后改 demand/window。`task_authorization_repo.go:25/43` 真实使用 `FOR UPDATE`；两个 SaveScheduling 请求按 A→B 与 B→A 更新重叠 story，存在锁循环的条件。其他计划/窗口锁可能提前串行化某些请求，因此这不是一次已复现死锁。

`SaveStoryTasks` 先锁同一 parent story，能串行化这一条同 parent 路径；不能由此推断跨 parent、跨入口或禅道原生写入也安全。必须核查所有写方锁顺序和索引，不能仅把 tasks 排序就声称闭环。

### C02 · P1 风险 · 长事务、锁等待与连接池缺少应用预算

`schedule/form.go:860–938` 验证 Stories/Tasks 的字段但没有批量数量上限；事务中逐条写 spec/action、解析 project 层级和维护关系。`GetProjectIDByExecution` 每次最多逐层查 20 次，在多个 task 中重复。

`internal/pkg/database/database.go` 每池固定 50 open/10 idle，DSN 构造未提供驱动连接/读写超时；`internal/server/server.go:87` 的 http.Server 未配置超时；检索到的 WithTimeout 主要为 shutdown，而非请求业务期限。`WithContext` 可传递取消，但传入的请求 context 本身不保证 deadline。两个独立池均打开时理论上可占 100 个连接，真实是否用满未测。

最小处理：按环境确定并执行 body/batch/transaction/query/pool 预算；批量前置验证、合并重复查询、限制锁持有时间。不要直接扩大池、添加并发查询，或拆坏业务原子性。无运行证据，不能给出“当前 p95 已合格”或随意设一个全局秒数。

### C03 · P1 风险 · 事务外检查与无版本条件更新可能覆盖其他写方

`SaveScheduling:46–68` 在事务外读 story 所属需求/产品并检查；`repo_scheduling_write.go` 的 UpdateStory/UpdateDemandScheduling 更新条件主要是 id/deleted，没有 expected ownership/version。期间禅道改变归属或排期字段时，该预检查可能陈旧。

`resolvePlanForProduct` 先查关联 plan，再建 plan/回填；`repo_story_link.go` 的 SELECT exists、MAX(order)+1、写关系/动作不等于并发幂等。真实唯一键、隔离级别、other writer 的行为决定是否重复计划/顺序冲突/重复动作；本轮没有 live DDL/并发证据，不能断言已发生。

最小处理：primary 事务内复核可变关系，明确每个系统拥有的字段，使用锁或条件写检测冲突，验证唯一键及重复提交语义。冲突/不确定提交不能静默成功或盲目重试。

### C04 · P1 条件性风险 · “全部已读”不是普通只读 SELECT

早期 WIP `po/reponotice.go:328` 使用无批量上限的 `INSERT ... SELECT` 从 zt_notify 读取并写 Workbench 已读表，含 FIND_IN_SET/REPLACE。即便目标是自有表，也要评估 ZenTao 源表扫描及隔离级别下的源记录锁、目标唯一键竞争。不能用“只写自己表”排除共享库影响。后续代码变化需在实施前重查该路径。

### C05 · P1 条件性风险 · 两套连接不证明最小权限或读写一致性

bootstrap 创建 primary/read 两个句柄并传给 PO。历史 P1b 记录新账号/GRANT 未完成；同时 V0 为 MySQL 8.4 本地实例、VM 文档为 8.0.46、后续本地迁移说明为 9.6。这些是不同日期/目标的证据，不能混成当前生产事实。

若读库未来是 replica，写后读和授权读取需明确一致性；若仍同实例，只是两个池，不代表资源隔离。只有当前部署目标的 grants/topology 才能证明权限收敛。本轮未连接任何库，未改变账号或环境。

### 运行证据仍缺失

尚未取得：当前实际引擎/隔离/索引、禅道原生写路径的锁序、真实锁等待/事务时长、pool wait、峰值负载、执行计划及事务冲突统计。现有隔离 integration 文件没有发现控制两个写事务竞争的测试；sqlmock 的 FOR UPDATE 断言不能填补它。

后续只在获准环境做：元数据核实 → 脱敏 digest/lock/transaction 采样 → 隔离合成数据的反序写/重复提交/重分配/回滚/取消试验 → 再决定实施方案。不得用真实业务库做压测、迁移或制造死锁；不得原样保存可能含业务内容的 PROCESSLIST/InnoDB 输出。

原理依据：[MySQL 锁说明](https://dev.mysql.com/doc/refman/8.0/en/innodb-locks-set.html)、[死锁处理](https://dev.mysql.com/doc/refman/8.0/en/innodb-deadlocks-handling.html)。运行行为以实际版本为准。

## 4. 本轮规范补齐

- `AGENTS.md`：增加共享数据库、禁止伪造权限/错误语义、WIP 完整性和测试可执行性；强制加载 onboarding。
- `architecture.md`：注释真实性、Repo/Service 文件职责、按能力拆分、单一来源需查真实消费者、文档时效；禁止非空账号充当授权。
- `database.md`：全量 ID 同属无界加载；请求总成本；共享写方、写时复核、锁序、有限事务、幂等和 retry、读写池一致性及环境证据。
- `quality.md` / `testing.md`：明确 runner 覆盖盲区、失败后未执行门禁、行为测试与 SQL mock 证据边界；未扩大任何 baseline。
- `agent-onboarding.md`：换工具/上下文恢复的加载记录、外部变更核对和最小交接模板；未知模型/会话 ID 不得编造。
- `agent-compatibility.md`：跨工具入口与冷启动步骤；新增 Copilot/Cline 薄入口，修正 Claude 对目录/通配符的伪导入；Cursor DB 规则消除跨 Repo 事务表述歧义。
- `debt.md` / `module-index.md` / `spec-index.md`：明确历史与当前证据，提供新规范入口。

Codex/Claude/Cursor/Gemini/Jules/Windsurf/Copilot/Cline 有对应入口策略。WorkBuddy/Antigravity 的本机版本自动加载未经验证，提供统一启动提词，不伪造兼容性。详见 [工具覆盖与验证](../../engineering/agent-compatibility.md)。本轮没有逐一启动这些 Agent 验收，也没有提交/push；远端 Agent 尚无法读取这些本地修改。

## 5. 自动验证

- `git diff --check`：通过。
- 修改文档的本地相对链接检查、Agent 薄入口的规范指针检查：通过；不等于各工具运行时已加载。
- 首轮 `make check`：失败于 PO 编译（缺失 Repo 方法），日志 `/tmp/wb-governance-20260907-check.log`。
- 规范修改后 `make check`：失败于外部新增 `service_primaryaction_test.go` 的 gofmt，日志 `/tmp/wb-governance-20260907-final-check.log`；后续 Make 项未因此执行。
- 独立 architecture gate：失败，新增 Service 文件内 Repo SQL；既有 dept 债务仍在。
- 独立 file-length：失败，观察期间超限文件集合变化；最近该次扫描为 repodone 504、render 512 行。
- 独立 vet：失败，重复类型声明/编译错误。
- patterns 硬规则与 secrets：通过；advisory 不等于逐条已接受，零 secret fingerprint 不等于完成轮换。
- `bash scripts/test-quality-gates.sh`：23 项通过，日志 `/tmp/wb-governance-20260907-gate-selftest.log`。
- 固定源码副本全部 10 个前端 unit：5 通过、5 失败，明细 `/tmp/wb-governance-20260907-frozen-frontend.json`。
- DB integration/live、真实登录浏览器、线上性能、当前远端 CI/分支保护：本轮未执行/未验证。

上述结果属于各次采样和源码副本，不把正在被其他任务改变的工作区称为最终稳定基线。没有因检查失败而修改业务代码、删除测试、改门禁或放宽规则。

## 6. 建议的后续处理顺序（本轮不执行）

1. 当前开发任务先核对完整 WIP 文件集合，处理重复声明/缺失方法、尺寸与职责门禁；固定可编译源码后重跑全部受影响测试。
2. 优先修复真实 capability 检查和首页筛选/回归入口；把“有测试文件”变成“默认会运行且有拒绝路径”。
3. 首页 SQL count/page/批量操作投影形成一个明确页面查询计划；用代表性规模验证，不能继续只改 UI。
4. 给排期共享写库补并发合同与隔离测试，再对获准环境收集锁、事务、连接池证据，决定最小补丁。明确实际 grants 后才称最小权限已落地。

文档可以降低歧义，不能保证所有模型永不越界。可靠交付需要实际加载证据、不可被该功能修改随意放宽的门禁、明确负责人和权限控制共同成立。
