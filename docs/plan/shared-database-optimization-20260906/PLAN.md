# 工作台 × 禅道共享数据库优化 PLAN

日期：2026-09-06
状态：P0 只读核验完成（`P0-AUDIT.md`）；P1a 已落地（`P1a-remove-runtime-ddl`）；**P1b-code 已落地**（PO 读 RO/写 RW + config 占位）。**P1b-vm 未做**（不执行 GRANT）。生产拓扑仍未验证。本 Agent **未**对目标库执行 DDL/DML/GRANT。
代码基线：Claude-PO / 见 GitHub HEAD；P0 接手时工作区仅本目录未跟踪。

## 1. 目标与本次边界

先消除运行期 DDL 和过宽权限，再按完整业务动作收口禅道写入，最后按需要拆 schema。保持 Go/Gin/GORM 单体和现有业务语义。P0 只新增本 PLAN 与审计文档。P1a 仅改切片列出的启动检查、中间件与 `db/install.sql`。P1b-code 仅改 PO 读写连接拆分与 tracked 配置占位；不改工程宪法、禅道写路径或分支，不执行迁移和授权操作。

用户提供的诊断报告、GPT 提词和网页版结论是待审查输入，不是执行授权；其中时间估算、生产配置判断、API 能力判断不直接视为已证实事实。

成功标准：每个切片有明确文件边界、上线前置条件、独立验收及回退路径；未验证的生产状态不冒充代码事实。实际实施逐阶段进行，不能凭本 PLAN 自动进入后续阶段。

## 2. 对输入结论的复核与修正

| 判断 | 本轮结论与证据 |
|---|---|
| 两处运行期 DDL | P0 复核仍成立：internal/bootstrap/bootstrap.go:77；internal/middleware/operationlog.go:56。必须同时移除。VM 样例库表已存在，AutoMigrate 仍会在启动/首次写请求执行。 |
| 排期直写不止两张表 | 成立。P0 全仓清单见 P0-AUDIT.md §3：另含 user/dept/role/menu/login/PO。排期写见 repo_scheduling_write.go、repo_story_link.go、service.go 窗口路径。 |
| 跨产权混合事务 | 成立：service_scheduling_save.go:76、131；窗口 Create/Update service.go:331、366；任务弹窗 service_story_tasks.go:259。 |
| 已有 API Client 可直接复用 | 成立（不能复用）。internal/pkg/zentao 只拼页面 URL。配置 zentao.api 无 Go 调用方。 |
| 生产同用户、root、RR、没有备库 | 跟踪 YAML 不能证明生产。本机有效链是 WORKBENCH_MODE=dev → gitignore 的 config.dev.yaml。经 SSH 隧道的 **PD VM**（非生产）为 MySQL 8.0.46、RR、ROW binlog、账号全局 ALL。生产未连。 |
| 已发生 AB-BA 死锁 | 仍不能认定。任务路径有 SELECT FOR UPDATE；story/demand 更新无 version CAS。 |
| AutoMigrate 阻塞同 schema 所有查询 | 仍过度。P1a 只评估审计表 MDL。 |
| 切 RC 可立即解决并发 | 仍不成立。VM 当前 RR+ROW，P1d 另判。 |
| 同 schema 无法单独恢复 | 仍过度。 |
| API 后禅道只有一个 writer | 仍应说「统一业务写边界」。定制树无 stories/tasks/plans POST；基础包 max5.6.1 有。部署装配未验证，不能声称现有 API 可承接排期。 |
| PO 只读连接 | **P0 事实 / P1b-code 纠正**：原 `po.NewRepo(dbReadonly)` 使关注/已读写入只读池。现行口径是 **读 RO / 写 RW**（`NewRepo(dbReadonly, db)`），**不是**整模块改回主库。 |
| 禅道源码位置 | 不是 csrcb20 设计仓。用户指定 `/Users/yuyan9923/GitHub/csrcb20-gitfox/zentao`。本机另有完整 `/Users/yuyan9923/GitHub/ZentaoPMS`（max5.6.1 + `config/routes.php`）。标准 REST 无 projectstory 写路由。 |

行号只定位当前基线，不是未来精确改动承诺。新文件均在实施时结合真实模块结构确认。

## 3. 目标边界与过渡约束

目标：Workbench 自有数据经 wbDB 读写；禅道原生数据经 ztReadDB 只读；禅道业务修改经业务 API。迁移期间保留受限 legacy DML 清单。

不能先把当前混合事务中的 Workbench 写和禅道写分配给不同 DB 连接：两个连接即便在同一实例，也不是同一事务。第一阶段保留现有单连接事务，用一个专用运行账号获得必要的逐表权限；每个业务动作完成 API 化后，才移除相应旧写路径。最终才能整体撤销禅道 DML。

不按 schedule/PO 模块机械拆池，不新建事务框架，不引入 XA、消息队列或双向 CDC。连接池总预算必须同时计算工作台、禅道及后台作业。

## 4. 分阶段切片

估时为人日范围，不含生产窗口和禅道维护方排期；P0 后重估，不构成交付日期承诺。

| 切片 | 文件/操作边界 | 前置与主要动作 | 验收 | 回退及禅道影响 | 估时 |
|---|---|---|---|---|---|
| P0 事实及产权盘点 | 全仓 Repo/Model/SQL；configs 加载链；禅道实际部署版；本目录后续 evidence 文档 | 逐动作盘点所有 DML、表产权、事务、写权限、API 覆盖；只读核验运行拓扑及数据库参数 | 完整动作→Service→Repo→表→owner→权限矩阵；未知项列出，禁止仅凭 zt_ 前缀分类 | 只读，无业务回退；诊断采样限时限量 | 2–4 |
| P1a 去运行期 DDL | bootstrap.go:77；operationlog.go:56；db/install.sql（补 zt_operation_logs） | 见 P1a-P1b-SLICES.md。VM 表已存在且列匹配 Model；install.sql 仍缺表。启动改为只读列检查。 | 无 DDL 账号启动+写审计；缺列启动失败 | 切片 ID P1a-remove-runtime-ddl；不恢复 AutoMigrate | 1–2 |
| P1b-code PO 读写 + 凭据占位 | `po.NewRepo(read, write)`；`configs/config.yaml` 占位 | 见切片。**读 RO / 写 RW**；不整模块切主库。不执行 GRANT。 | 写路径单测；config 无真实秘密；`WORKBENCH_*` 仍可覆盖 | 切片 ID P1b-code-po-rw-config | 0.5–1 |
| P1b-vm 逐表授权 | DBA GRANT 脚本 | 见切片。禁止 REVOKE 禅道共用账号。依赖 P1b-code 已上线。 | 逐表授权+越权拒绝；禅道原账号不受影响 | 切片 ID P1b-runtime-grants；补 GRANT 不恢复 ALL | 1–2 |
| P1c 写边界防新增 | docs/engineering/database.md；scripts 现有质量入口及精确基线 | P0 后另行治理切片，冻结现有直写位置及用途；扫描 GORM Model 映射和原生 SQL，未知候选人工审查 | 新直写与迁移 DDL 回归样例被阻止；删除旧项同步收缩清单 | 可回退检测器缺陷修复，不扩大 legacy 目录豁免；无禅道数据变更 | 1–2 |
| P1d RC 对照实验 | internal/pkg/database/database.go:95 配置入口；配置模板；tests/integration/ 对照用例 | 核验引擎版本、binlog_format、会话参数；审查依赖事务稳定快照的路径 | 多个新建物理连接读回隔离级别；RR/RC 并发业务断言通过；等待和延迟无不可接受回归 | 单独恢复原隔离配置并更换连接池；不改禅道全局隔离级别；测试失败则保留原值 | 1–3 |
| P2a API 能力与契约 | 禅道部署源码/API 路由；schedule 请求/Service；internal/pkg/zentao | 对照完整业务动作检验现有 API、权限及事务；仅缺失时新增业务端点；架构确认后实现 | 字段、权限、对象范围、原子边界、冲突、幂等、查询结果协议全部可测试 | 契约阶段不导流；不能仅根据一个 story.php 缺 POST 推断全站不支持创建 | 2–4 |
| P2b 可恢复业务动作 | service_scheduling_save.go:34、101；repo_scheduling_write.go；repo_story_link.go；新增局部命令记录及客户端 | 先实现持久化命令、远端幂等及恢复，再接单个完整动作；不得按关联表逐张迁移 | 第 6 节故障矩阵全部通过；本地与远端对账无丢失/重复 | 停止新 API 动作并处理在途记录；未知结果不切 SQL；不自动撤销已成功业务 | 5–10/首动作 |
| P2c 扩展及撤权 | P0 全部写入口，包括非 schedule 模块；连接注入和权限 | 按完整动作扩大灰度；全量后观察一个约定业务周期 | 无剩余禅道直接写；撤权后全业务通过；积压清零或明确人工处置 | 保留修复/查询能力；不能只因排期迁完就撤全库 DML | 依 P0 重估 |
| P3 自有 schema 独立 | 自有 Model 表映射、Repo 跨表查询、迁移/部署/备份脚本 | API 化稳定、跨产权事务消除后；先同实例独立 schema | 全量/增量校验、切换演练、独立恢复演练、跨表读性能通过 | 停写→逆向同步切换后增量→校验→切回；禁止简单切连接丢掉新数据 | 5–10 |

首轮实施建议只排 P0、P1a、P1b；P1d 单独判断，P2 不在信息不完整时启动。

## 5. 三项关键改造结构

### A. 禅道写入按业务动作收口

现状：Service → 一个 Repo 事务 → 禅道多表 + Workbench 关联。

目标伪代码（不是可直接复制实现）：

```text
Service 校验当前操作者与对象范围
本地短事务：保存 operationID、请求摘要、可恢复业务载荷、目标版本、固定 writer
提交本地事务
调用禅道业务 API(operationID, intent)          // 不持有本地 DB 事务
禅道事务：验证身份/状态/冲突 → 完整业务动作 → 持久化幂等结果
本地短事务：依据结果保存窗口/计划映射，并标记 applied
```

Service 保留业务编排；Repo 只做持久化。不能把 HTTP 偷塞进 Repo.Transaction，也不能把禅道业务规则搬成通用表 CRUD。resolvePlanForProduct 等依赖本地窗口与远端计划的路径要明确输入快照、远端 plan ID 回传和本地映射归属。

### B. DDL 移出

```text
部署迁移步骤：专用账号 + 经核对的增量 SQL + 备份/锁等待限制
启动：检查审计表及必需列/索引 → 不满足则明确报错退出
中间件：写审计 → 按已确认的审计失败策略处理；绝不自动建表
```

先迁移后发应用。仅 HasTable 不足以验证结构兼容。业务已提交后的日志失败是否阻止响应不能随意改：必须核验现有行为，另行明确审计持久性要求；本切片不承诺事务外日志与业务原子一致。

### C. RC 条件切换

复用现有 SessionVariables 入口，候选值为 transaction_isolation=READ-COMMITTED；不改 MySQL 全局默认。实际键名、驱动编码及数据库兼容性用集成测试验证。启用前核验 ROW binlog 要求与部署版本；只读/OceanBase 连接单独验证。

RC 下事务内多次读取可见不同已提交数据，必须检查排期预校验与写入间的状态变化。需要时通过禅道已有并发协议、条件更新或事务内重检解决；不能凭空认定某个 version 字段就是并发版本。

## 6. API 一致性与灰度必备契约

网页版方案缺少“远端调用前本地持久化”步骤；若仅靠内存保存 key，进程退出后就无法可靠补偿。

- 操作记录：稳定 operationID、业务作用域、操作者、请求摘要、最小必要恢复载荷、writer、远端结果 ID、状态、重试次数/时间。敏感载荷按现有安全规范处理，日志不输出凭据。
- 状态：prepared → remote_confirmed → applied；确定拒绝记 rejected；超时记 unknown，保留查询/同 key 重试能力。
- 幂等：远端按调用方+key 唯一约束；相同 key 不同摘要拒绝；幂等记录与业务结果同事务提交。保留期覆盖恢复窗口，未解决记录不可清理。
- 并发：限制同一排期对象的在途操作，或采用真实可用的冲突令牌；幂等只防同一操作重放，不防两个不同操作相互覆盖。禅道 Web 并发也必须测试。
- 授权：Workbench 与禅道分别执行适用授权；不能信任请求体中的任意 actor 或以服务账号绕过真实用户权限。身份映射方案是 P2 前置决定。
- 恢复：启动后/现有后台机制有界扫描待处理记录；超时先查远端结果，按契约同 key 重试；退避、次数上限、人工处置入口。不用新同步框架。
- 展示：远端成功但本地待补时，不显示整体失败诱导重复提交，也不显示全部成功；返回可查询的处理中状态。相关 UI 改动必须做真实浏览器验收。
- 灰度：先无副作用校验，再按产品/小组对“新动作”固定 writer；在途操作不随开关改变。dry-run 之后真实执行仍需再次校验。
- 回退：关闭新 API 导流，保留结果查询及本地补齐；unknown 操作不能 SQL fallback，也不能直接用新 key 重做。

必须故障注入：调用前崩溃；远端提交后响应丢失；远端成功本地失败；本地提交后响应丢失；重复并发请求；相同 key 不同载荷；两个操作者冲突；处理中切换开关；恢复期间权限变化；删除/关闭对象后重试。每项检查最终表数据、关联、审计数量与用户可见状态。

## 7. 数据库方案对比与授权设计

| 方案 | 建议 | 能解决 / 不能解决 |
|---|---|---|
| 专用账号逐表授权 | 优先 | 限制误写/DDL；不能消除仍获准的业务直写冲突，也不隔离实例资源 |
| 同实例独立 schema | P3 推荐 | 明确归属和迁移范围；共享故障域、binlog、连接/IO；跨 schema 同连接事务技术上可行，但不作为新的跨系统耦合 |
| 视图代理层 | 默认不选 | 受限授权可控制读取，但可更新视图和 DEFINER 权限须审查；不隔离资源、恢复和数据所有权 |
| 独立实例 + 同步 | 有容量/容灾证据再选 | 隔离资源和恢复；新增同步延迟、运维及对账成本 |

过渡账号：仅逐表 SELECT + P0 证实必需的 INSERT/UPDATE/DELETE；Workbench-owned 表同样按需要授权；不授予 CREATE/ALTER/DROP/GRANT OPTION 或全局管理权限。MySQL TRUNCATE 依赖 DROP 权限，不存在单独 TRUNCATE grant。

最终账号：wb_runtime 仅自有 schema 所需 DML；zt_reader 仅原生表 SELECT；迁移账号离线使用。旧共用账号是否仍被禅道使用必须查明，不能直接撤销其权限。只读连接名称不能替代权限验证。

报表允许的延迟由产品确认；写前授权、对象状态和写后确认走权威数据，不依赖延迟副本。小规模低频报表可定时同步；只有证实吞吐/时效需求后才考虑单向 ZenTao→Workbench CDC。DDL 漂移、删除语义、断点和重放均需验收。

拆 schema 切换：先备份→建目标→复制自有数据→受控停写→补增量/对账→切连接→验收→保留原表只读观察。回退需要回灌切换后的增量；同实例拆 schema 不保证无损独立 PITR，必须隔离恢复演练后再承诺 RPO/RTO。

## 8. 九条风险再评估（条件性目标，不代表已降低）

| 原风险 | 覆盖项 | 完成后的残余风险 / 新风险 |
|---|---|---|
| R1 双系统写冲突 | P0、P2 | 中：禅道内部并发仍需控制；新增远端结果未知和恢复积压 |
| R2 软删除语义 | P0、schema 契约测试 | 低至中：原报告尚未证明现存缺陷；版本升级仍可能改变语义 |
| R3 运行期 DDL | P1a、P1b | 低：受控迁移仍有 MDL；新增结构不符拒绝启动的部署风险 |
| R4 长事务 | P0 测量、P2 短本地事务 | 中：远端业务事务仍可长；连接池不能代替 SQL/锁优化 |
| R5 弱读误导 | P0 拓扑/权限、读一致性契约 | 中：副本延迟和主实例故障仍存在 |
| R6 恢复耦合 | P3、恢复演练 | 中：跨系统业务恢复点仍需对账 |
| R7 表产权混淆 | P0 清单、P3 schema | 低：旧命名与历史脚本仍需兼容；当前不批量改表名 |
| R8 字符集/时区/隔离 | P0 参数采样、P1d | 低至中：RC 新读语义、时区转换与排序规则仍需专项验证 |
| R9 审计边界 | P1a、P2 操作链路 | 中：日志失败策略、保留期和合规要求需业务/运维确认 |

## 9. 后续 PR 检查清单草案

仅列草案，未修改 docs/engineering/database.md 或 AGENTS.md。

- [ ] 每张表有实际 schema 与 ownership 证据，不凭前缀推断。
- [ ] 原生表新写走已确认业务 API；保留直写有精确 legacy 项及退出条件。
- [ ] Handler/Service/Repo 层次正确，真实用户及对象授权未弱化。
- [ ] HTTP 不在本地数据库事务内；幂等、未知结果、恢复和并发冲突均有测试。
- [ ] 运行期不执行 DDL，迁移只作用于批准的自有对象；受限账号验收通过。
- [ ] 查询计划、SQL 数量和数据规模有依据，无新增 N+1/内存分页。
- [ ] 无凭据进入 Git、日志或证据；环境及授权变更可追踪。
- [ ] 单写灰度与在途动作回退可演练；无超时 SQL fallback。
- [ ] git diff --check、make check 及该切片集成/E2E 门禁通过，失败如实记录。
- [ ] 涉及 UI 时有登录态交互、视觉、Console/network 和加载资源版本证据。

## 10. 实施前待解决的问题

1. 实际部署引擎、有效账号/权限、主只读连接拓扑、binlog/备份策略；不从静态配置猜生产。
2. 全仓原生表写入口，以及哪些自定义字段和业务规则实际归禅道维护。
3. 部署版禅道的完整 API 路由及 model 副作用：是否自行提交事务、发通知、写文件等，能否满足一个业务动作原子性。
4. 排期冲突时产品希望拒绝、合并还是覆盖；处理中状态的展示与恢复时限。
5. 身份映射、审计失败策略、允许维护窗口及 RPO/RTO。
6. 跨 Repo/跨系统事务方案按 architecture.md 在 P2 实施前确认；本 PLAN 提供待评审设计，不声称已获批准。

## 11. 验证依据与本次检查

技术依据：
- MySQL 隔离与锁：[Transaction Isolation Levels](https://dev.mysql.com/doc/refman/8.0/en/innodb-transaction-isolation-levels.html)、[InnoDB Locking](https://dev.mysql.com/doc/refman/8.0/en/innodb-locking.html)。支持 RC 风险评估，不支持无条件切换。
- MDL：[Metadata Locking](https://dev.mysql.com/doc/refman/8.4/en/metadata-locking.html)。仅引用对象级锁原理，部署版仍需核验。
- 恢复：[Point-in-Time Recovery](https://dev.mysql.com/doc/refman/8.0/en/point-in-time-recovery.html)。需基线备份与后续 binlog，不能用 schema 拆分替代恢复演练。

P0 已完成只读核验（P0-AUDIT.md）：含全仓写入口、排期事务、配置链、VM 样例库参数、禅道定制树与基础包 API 分层。未对部署实例发写请求，未改业务代码。

本次检查结果（P0 接手复核）：`scripts/check-file-length.sh` 失败：`baseline loosening rejected: web/static/js/po/workboard.js (cannot increase baseline from 660 to 728 lines)`。trusted 基线来自 `merge-base HEAD origin/Claude-PO`（`17bd8e7d`）中的 660，HEAD 已提交 728。与本目录文档无关。状态 **BLOCKED BY EXISTING BASELINE**。`make check` 后续步骤未执行。不把上述称为全部检查通过。
