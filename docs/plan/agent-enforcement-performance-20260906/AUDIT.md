# Agent 约束与页面性能现状核查

核查日期：2026-09-06。本文是静态核查与本地门禁结果，不是全项目安全认证或当前运行时性能验收。

## 1. 证据基线

- 仓库：`/Users/yuyan9923/GitHub/workbench-claude-po`。
- 分支：`Claude-PO`；本地 HEAD：`8adbe0f7cc918dd1ca61aa481e71c32e23e0cc92`。
- 开始时已有未跟踪目录：`test-results/`，保持原样。
- 本轮只新增本目录的计划文档，不修改生产代码、规范、门禁、旧卡状态或数据库。
- `127.0.0.1:8093` 无监听，GET `/home` 连接失败。本轮无法取得认证浏览器、请求瀑布、当前延迟和真实 EXPLAIN 证据；没有启动可能连接业务库的服务。
- GitHub 的 `Claude-PO` 远端查询结果为 `cb3cc2b8ccd19d9d2b5ffdc67f24c7a3092471e3`，与本地不同。不能拿远端 CI 为本地背书，也不据此自动同步分支。
- [远端运行 34006008092](https://github.com/Cetto9923/workbench-claude-po/actions/runs/34006008092)：`regression-gates`、`integration-tests` 及对应执行步骤 success，绑定远端 SHA。按本地 HEAD 查询没有返回运行。
- GitHub branch protection API 返回 403，提示当前仓库需要升级套餐或公开仓库。无法确认强制合并门禁已启用；本轮未改仓库可见性、套餐、权限或远端配置。

## 2. 总判断

规范已经形成了一部分有效约束，但没有形成端到端闭环。能证明的是文件存在、检查能拦截部分错误、当前实现是否符合规则；不能证明每个 Agent 已阅读并理解全部规则。

| 环节 | 当前证据 | 判断 |
|---|---|---|
| 规范入口 | AGENTS、CLAUDE、GEMINI、Cursor 适配器及 onboarding 已存在 | 入口有基础，避免继续复制出多套宪法 |
| 实际加载 | `agent-onboarding.md:59` 明确各工具独立冷启动 NOT VERIFIED | 不得声称跨模型约束已验证 |
| 机械检查 | `make check` 实际通过；门禁自测 20 项通过 | 有用，但不能覆盖语义、体验和全部安全性 |
| 基线防篡改 | 187d6107 同时把 `workboard.js` 上限从 660 提到 728 | 当前门禁比较工作树内基线，不能独立防止放宽自身上限 |
| 性能规则 | 首页仍把全部匹配 ID 拉到 Go 后分页，门禁通过 | MUST 与机器覆盖之间有确定缺口 |
| 前端验证 | 3 个 `tests/e2e/*.spec.js` 实为 Node assert/mock 测试 | 有行为保护，但不等于真实浏览器 E2E，且没有接入当前 CI |
| CI 与发布 | 远端另一个 SHA 有真实绿色 Job；本地 SHA 未取得 CI | 交付报告必须绑定代码、运行产物和 CI 的实际版本 |
| 历史债务 | 门禁报告 18 个超长文件、3 个 secret 指纹、1 个架构例外 | 绿色表示没有触发当前增量检查，不代表零债务 |

没有查到本轮所读规范/计划中对 660→728 的明确规则豁免；提交说明写了提高上限，但这本身不是授权证据。不推断是哪一种模型所为，也不推断开发者主观意图。AGENTS 的遗留例外不得增长要求仍适用。

## 3. 当前代码中的性能与规则缺口

### F01：首页分页只做到“详情有界”，没有做到“候选集有界”——确认

- `internal/module/po/servicefollow.go:112` 的 `listAllStageDemands` 循环阶段收集所有需求/故事 ID，Go 中按 `kind:id` 去重，再在 175 行切片。
- 同文件 245 行对排期/交付的混合列表执行相同内存分页。
- `repovaluestream.go:213` 的 ID 查询使用无 Limit 的 Pluck。只取 ID 比全详情好，但数据量与用户全部匹配记录相关，违反默认 SQL 排序/计数/分页要求。
- `Home` 先逐阶段 Count，再通过 `countAllStageUniq` 重复取各阶段全量 ID 求并集，另有四项 KPI 和版本窗口调用。存在确定的串行往返和重复扫描机会；具体耗时占比尚未测量。
- 当前 `home.js` 已传 Page/PageSize，并用序号防止旧响应覆盖；不能再将“前端一次下载全部详情”当作当前缺陷。
- `check-patterns.sh:54` 只匹配 `[start:end]`，实际 `[offset:end]` 没被识别。扩展正则也只能产生候选，不足以证明数据规模有界。

### F02：看板限量不等于完整分页，子树仍可放大——确认

- `repoboard.go:79` 根需求 Limit(200)，147 行独立故事 Limit(100)，没有完整根集合的分页契约。
- 子需求、关联故事批量读取，任务进度聚合再读取关联任务；根条数限制不能约束子树和任务数量。
- `repoboardtask.go` 对任务先 Limit，再在 Go 处理部分 Focus 条件，Summary 由截断后的 rows 计算。应补超过上限的样本，区分“本页数量”与“全部匹配数量”。
- 最新 8adbe0f7 已修正遮罩初始可见并去掉 blur；不能再将它列为待实施旧修复。

### F03：展示名查询与数据规模耦合——确认

- `repotodo.go:137`、`user/repo.go:102` 都读取全部未删除用户的账号/姓名映射。
- 首页、看板、已办、通知等链路调用该能力。已办只显示当前操作者，也会查询整张用户字典。
- 不能据此声称每个请求都重复查询多次；必须通过请求计数确认。方案应按本页引用账号批量查，而不是先引入全局用户缓存。

### F04：已办、通知仍有扫描成本候选——结构确认，耗时待测

- `repodone.go` 已有 SQL Count、分页和按对象类型批量标题补齐，不应重做旧分页治理。
- 今日/本周/本月/季度筛选在时间列上使用 DATE/YEARWEEK/DATE_FORMAT/QUARTER；它们限制普通时间索引用于范围查找的机会。真实索引与执行计划未核验，不能断言必定全表扫描。
- `notice_query_repo.go:69` 用 `FIND_IN_SET(?, REPLACE(n.toList,...))` 判断收件人，分页和多口径计数仍需要评估底层扫描。
- 通知分页目前还选取 `n.data`；它被内容渲染使用。不能直接删除而改变通知详情能力，应测字节数后再决定摘要与详情分离。
- 通知、待办的现有 SQL 全局分页和结果等价测试是保留资产，不把旧实现的 21 次/2 次查询数字当作当前数字。

### F05：公共链路需要分段测量——候选

- `middleware/auth.go` 每个受保护请求读取当前用户、非超级管理员权限及菜单；JSON 列表请求也会走这条链路。
- `render.go` 开发模式每次解析模板，生产模式已有预热缓存；资源 URL 通过文件时间生成版本。
- 已有 `SQLRequestContext`、`sqllog` 和 integration QueryCounter，应复用。不能在没有测量时将鉴权缓存或新增监控平台当作优先方案。

### F06：测试环境防误操作边界不足——确认

- `tests/integration/performance_baseline_test.go:33` 仅通过 DSN 字符串排除 `/zentaopms` 等名称，不能证明其他名字的数据库为隔离库。
- `InitMinimalSchema`、`SeedSyntheticDatabase` 会删表/建表/删除并写入数据。DB 名称黑名单不能充当这种操作的授权机制。
- 当前 GitHub job 使用一次性 MySQL service，和未知本地 DSN 是不同安全条件。本轮没有执行任何真实 DB integration，也没有读取配置凭据。

### F07：近期 story 抽屉需要对象权限回归——高优先核验项

- `repoboardtask.go` 的 StoryID 分支跳过小组成员过滤；`serviceboard.go:85` 的 BoardTask 验证/选择小组，但所读方法未见对指定故事的对象访问决策。
- 这是具体的授权缺口线索；完整 Handler/路由路径、有效业务授权范围与跨用户请求尚未完成验证，不宣称已复现越权。
- 后续必须用另一个无权用户和猜测 StoryID 做 Service/HTTP 回归；不能用“按钮只能从可见树打开”替代权限。

## 4. 已运行验证及边界

| 检查 | 本轮结果 |
|---|---|
| make check | PASS；Go 部分结果使用缓存，未宣称全部 uncached |
| bash scripts/test-quality-gates.sh | PASS，20 项临时仓库自测 |
| node tests/e2e/personal-list.spec.js | PASS，Node/mock 行为测试 |
| node tests/e2e/auth-errors.spec.js | PASS，Node/mock 行为测试 |
| node tests/e2e/csrf-tokens.spec.js | PASS，Node/mock 行为测试 |
| git diff --check | PASS |
| 当前认证浏览器、性能录制、真实 DB EXPLAIN | NOT RUN：8093 未监听，无获准运行/DB证据 |
| 本地 HEAD 的远端 CI | NOT VERIFIED：查询未返回对应运行 |

make check 输出 47 个 advisory，其中 12 个未在清单中。这里包含 `controller.fetch`、共享组件和测试模拟代码的匹配，不能把 12 个都算作违规。需要精确分诊而非全部变为 hard。

已有 `performance-baseline.md` 绑定 2b4ed960 的旧实现、小规模隔离样本，不等于最新首页/看板、真实网络和多用户并发的性能验收。`home_demands_baseline_test.go` 对页长/总数有断言，但耗时和 SQL 数主要为 Log，无法单独阻止成本回退。
