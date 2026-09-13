# Main 后端能力选择性吸收计划

状态：A 阶段只读能力已落地，B 阶段完成未认证边界收口；C 阶段写入功能仍受授权、幂等和失败恢复门槛约束。

## 1. 比较基准与边界

- 当前分支：release/po-integrate-main-202609，HEAD 2b5ec597f0a0ebbf64f8b7b9a55ec379e5909218。
- Main：main 与 origin/main 均为 b61c5c3cb1e3499b0e01c0f8ddb5f953ce9f8aaf；github/main 是旧快照，不作为本轮来源。
- Main 检出目录：/Users/yuyan9923/GitHub/workbench；当前目录：/Users/yuyan9923/GitHub/workbench-claude-po。
- 两分支共同祖先：e030a3f70c27c7b15a8a7b1b062b197e6e5c314c。两端差异不等于新增功能，更不等于应直接应用的补丁。
- 当前有 15 个已修改文件，涉及 profile、共享 app、PO JS/CSS 和模板；另外有未跟踪计划及原型。均保留，不与本轮吸收混合认定为已验证。
- Main 工作区有未跟踪 AGENTS.md、backups/，没有已跟踪文件修改；来源以固定提交为准。
- 本轮只写本计划；不合并、切换、提交、推送分支，不连接或写入业务数据库。
- 工程约束沿用当前仓库 AGENTS.md，以及已读取的 onboarding、spec-index、architecture、database、frontend、testing、quality、shared-frontend、module-index。

## 2. 真实增量与取舍

| 能力 | Main 证据 | 当前分支 | 处理方式 |
| --- | --- | --- | --- |
| 产品已有版本列表 | testtask.ListProductBuilds，GET /products/:id/builds，builds.go、gateway.go | 无此接口 | 优先适配吸收，支持已有版本选择；补访问控制、分页完整性和错误语义 |
| 版本已关联研发需求列表 | build.ListLinkedStories，GET /builds/:id/linkedstories；合并父/子版本 stories，再批量取标题 | 有可关联列表和关联/取消关联写入，无已关联列表接口 | 优先适配吸收，保留批量读取；明确无标题/不可见项，不泄露对象信息 |
| 非联调测试单创建 | testtask.CreateTesttasks 与 Gateway；实际 POST /projects/:id/testtasks | 无此创建接口 | 功能价值高；完成授权、上下文、批量失败恢复设计后吸收 |
| 创建新版本 | Main testtask.CreateBuilds | 当前已实现 /demands/:id/testtask/builds | 不重复引入；检查当前需求上下文未参与验证的问题 |
| 关联/解除研发需求、搜索 | Main build 模块 | 当前已有对应接口和搜索实现；searchcond/reposearch 的部分差异只是格式 | 不整包复制，补齐已关联回显即可 |
| 提测上下文错误处理 | Main GetContext 对人员、产品、内部用户查询失败返回错误 | 当前降级为空数据或账号后继续返回成功 | 优先吸收：失败显示不可用，不伪装为无产品/无人员 |
| SQL 彩色输出 | sqllog/console.go | 无彩色输出，有 SQL 脱敏和错误分类 | 本轮不吸收；Main 同时移除了 SanitizeSQL/SafeErrorCategory，不能整包替换 |
| 需求评审 | Main 直接调用 Repo 改共享表 | 当前通过禅道原生 API 处理评审、提交、撤回 | 保留当前流转边界，不能以本地三表写入替换外部流程 |
| 排期 | Main 的若干单项查询与较早保存逻辑 | 当前有对象归属校验、删除对象产品检查、批量加载和 US 编号支持 | 保留当前；本轮未发现值得替换这些保护的增量 |

来源提交候选：8cebbcc6（已关联列表）、135ecfce/ea85b803（产品版本）、88a64011/b61c5c3c（测试单及修正）。提交仅用于定位，实施按功能差异适配，不直接整组 cherry-pick。

## 3. 双方需要补齐的地方

1. Main CreateTesttasks 显式忽略 actor 和 demandID；版本与产品一致性检查有价值，但不能证明操作人有权办理当前需求，或该版本属于当前需求的允许上下文。
2. Main 创建版本、关联/解除需求从当前分支的 DoAs(account) 改为 Do。吸收时保留用户身份传递，且仍须由 Service 验证业务对象，不能把 DoAs 等同于全部授权。
3. Main CreateTesttasks 在循环中校验并逐项远程创建：后项失败时前项可能已成功，却整体返回错误；重试可能重复创建。没有看到有限批量、幂等或部分结果恢复合同。
4. Main 产品版本 Gateway 单次读取，没有体现完整的分页消费；必须核实禅道响应与分页参数，不能把第一页当全集。
5. 当前 testtask.CreateBuilds 也忽略 demandID；ListProductExecutions 不使用 actor。当前 RequirePerm 对 PoDemandReview、BuildLinkStory 有登录即通过分支。Main 对这两个 capability 的检查更严格，但其旧实现缺少当前的 OR 权限及 JSON 403 处理，不能整体覆盖中间件。
6. 当前 profile 的组织角色输出属于未验收 WIP，不能作为吸收权限设计的权威，也不能用导航视图替代操作权限。
7. Main testtask JS 超过千行，不能整文件引入当前有文件大小门禁的前端；只迁移新增接口的消费者，按现有职责分拆。

## 4. 分阶段执行 Plan

### A：先补提测公共读取与错误语义

- 在现有 testtask/build 模块新增两个 GET 接口，不新增平行模块或第二套页面。
- 保留 Main 的版本选项装配、父子版本关联 ID 合并和批量标题读取；Repo 返回事实，由 Service 做当前用户可见性与上下文检查。
- 提测 GetContext 查询失败原样传播成明确错误，前端清除旧数据并允许重试；不展示伪空态。
- 版本列表采用有界分页/检索，前后端同步；需确认禅道接口的实际分页行为后固定参数。
- 已关联列表用于选择版本后的回显及关联/解除后的刷新；异步切换版本时，旧请求不得覆盖新版本内容。
- 接口保持当前 JSON 外壳 success/data 与明确 HTTP 错误码；保留现有写入接口的请求合同。
- 验收：无版本、父子版本重复需求、失效或不可见对象、分页第二页、读取失败、快速切换版本、关联后回显。

### B：收口操作身份和授权

- 盘点现有已分配 capability 与业务办理身份，明确每个读写接口的能力门禁及需求/产品/版本访问范围；不得猜测权限码或扩大授权。
- 对比 Main 的严格 capability 检查，适配当前 OR 权限和 JSON 错误响应；取消特殊放行前先形成现有用户兼容性矩阵，避免正常办理人无声失去权限。
- 修正当前 CreateBuilds 的需求上下文缺失；Main 新接口也使用同一检查。校验产品、项目、执行、版本的真实归属，不信任客户端字段。
- 远程写入继续走用户态 DoAs；只在必要且已确认的范围内使用现有能力，不引入新的通用权限框架。
- 验收：匿名、无 capability、有 capability 但对象不属于本人范围、伪造 demand/product/build/execution、正常办理人、超级管理员。
- 前置门槛：既有权限/办理规则未确认时仅完成审查，不启用新写入口。

### C：吸收真实非联调测试单写入

- 使用 Main 的请求字段、日期/优先级校验、批量读取版本元信息和返回真实测试单 ID 的能力；实际禅道路径以 b61c5c3c 中的 /projects/:id/testtasks 为源代码证据，联调前核实部署版本。
- 保留产品与版本一致性检查，增加 B 阶段授权与所有条目预校验，任何预校验失败均不得开始远程写入。
- 不支持联调测试单；只接受明确的非联调值，避免其他非法 joint 值被当作非联调。
- 有限批量、超时、同批重复项、部分成功、超时后结果未知和重试去重必须先形成接口合同；不可使用本地 DB 事务假装回滚远程创建。
- 默认失败即停止后续创建并保留已成功 ID；结果未知时禁止自动重发。持久化恢复/幂等机制如需要表结构或外部接口支持，单独提交设计与授权，不隐式迁移。
- UI 仅在收到真实 ID 后提示成功；保留可复核的部分结果，关闭/重开不能诱导用户重复提交。
- 验收使用模拟禅道 HTTP 服务覆盖第二项失败、超时、重复点击、身份传递、无 ID 响应；真实写入验收必须有指定测试环境和测试数据授权。
- 此阶段是条件计划：没有确定业务授权映射、失败恢复合同和测试环境，不进入生产实施。

### D：接回当前交互并回归

- 使用当前 primaryAction 和已存在的提测入口，保留主页、详情、列表的授权动作合同；不复制 Main 的首页整体结构。
- UI 只接 A/C 新能力，复用当前共享请求、人员选择、主题和弹窗能力；避免重复入口和“已有关联列表但前端仍用模拟数据”。
- 每批重新核对 HEAD/WIP，检查消费者及失去引用的代码，仅删除该批造成的孤儿代码。
- 运行新增/受影响 Go 和前端测试、git diff --check、make check；不靠删除测试、放宽基线消除差异。
- 登录浏览器检查完整提测流程、权限失败、版本切换、浅/深色、Console/network 和加载资源版本；再冒烟首页、待办、详情、排期。

## 5. 本轮验证及限制

- 当前分支：`go test ./...` 退出码 0；受影响模块及 profile 单测均通过。
- Main 检出：相同命令退出码 0；build 测试通过，testtask 显示 [no test files]，不能当作新测试单写入已验证。
- 当前 git diff --check，退出码 0。
- 未运行真实数据库、禅道写请求、浏览器或性能测试；上述结论为源码与包测试证据，不是线上功能完成证明。
- 本轮 make check 退出码 2：local-artifact gate 通过后，在 check-gofmt 阻断。涉及 board_task_transition_test.go、serviceboard.go、task_client.go 的格式指纹；后续门禁未执行，状态为 BLOCKED BY EXISTING BASELINE。本轮没有修改上述文件或基线。
- 推荐顺序：A → B → C → D。A 可先交付只读价值，C 的真实写入不得绕过 B 或结果恢复门槛。
