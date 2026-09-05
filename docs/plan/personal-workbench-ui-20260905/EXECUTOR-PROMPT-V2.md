# Gemini / Coding Agent 执行提词：员工工作台四页 UI V2 对齐治理

你现在作为本仓库的执行工程师，必须严格按以下合同工作。不要把它当成自由发挥的 UI 美化任务。你的目标是把 /home、/todos、/done、/notice 对齐到当前 PRD 和 CRCBWorkbench 已确认的交互语义，并把“未接入”与真实数据源状态分开。

## 先读什么

在任何编辑前，按此顺序读取：

1. /Users/yuyan9923/GitHub/workbench-claude-po/AGENTS.md
2. docs/engineering/architecture.md
3. docs/engineering/database.md
4. docs/engineering/frontend.md
5. docs/engineering/testing.md
6. docs/engineering/quality.md
7. docs/plan/personal-workbench-ui-20260905/PLAN-V2-COORDINATED-ALIGNMENT.md
8. 旧 PLAN.md、EXECUTION-REPORT.md、review/INDEPENDENT-REVIEW.md，只用于识别已完成能力和已知缺陷。
9. CRCBWorkbench 的 PO PRD、Done Action Inventory 和首页/待办/已办/通知原型。

如果本提词与旧计划的视觉决策冲突，以 V2 为准；旧计划中已经通过的请求竞态、分页、错误态和安全转义能力不要重写一遍。

## 必须先做 Preflight

运行并保存输出：

~~~sh
pwd
git rev-parse --show-toplevel
git branch --show-current
git status --short
git remote -v
git rev-parse HEAD
~~~

必须在 /Users/yuyan9923/GitHub/workbench-claude-po、Claude-PO 分支上工作。若 HEAD 不是当前任务记录的基线，先报告差异再继续。保留全部用户 WIP；不要 reset、clean、restore、stash、切分支、merge、rebase 或删除文件。不要触碰 workboard 相关 CSS/JS/template。

8093 是用户运行实例。不要 kill 或替换它。只有在确认进程 cwd、构建时间、CSS/JS asset revision 和 API 响应后，才能把浏览器看到的版本归因于你的代码。

## 数据库和权限安全

仓库里存在配置不代表你获得了数据库访问授权。没有已批准的只读 Schema evidence 时，不要连接未知数据库，不要猜 DSN。禁止任何写 SQL、migration、AutoMigrate、INSERT、UPDATE、DELETE、ALTER、CREATE、DROP、TRUNCATE、REPLACE。若必须读取 Schema，只能使用明确的测试/审计只读来源，并且日志和报告不得泄露密码、token、DSN 或个人业务数据。

## 不得自行发明的业务规则

不要自行决定以下内容：

- actor/owner/成员/租户范围；
- 审批、问题、风险、关注和个人事项的责任政策；
- 没有 Schema 证据的字段（deletedAt、tenantId 等）；
- 通知正文里的数字是否是对象 ID；
- 没有服务端规则的推荐优先级；
- 没有可信 URL 的处理入口；
- 已办是否等同于待办消失；
- 参考项目中不适用于本仓库的 DB、前端框架或跨模块架构。

遇到这些问题要标 WAIT DECISION 或 BLOCKED，说明缺少哪条证据。

## 你必须实现的终态

### A. 共享外壳

四页都采用：标题和说明 → 右上紧凑快捷筛选 → 分类/关系 → 工具栏 → 结果条 → 列表 → 分页。沿用现有侧栏和顶栏。标题 20px；面板白底、1px 边框、6px 圆角；按钮/输入 32px 左右；行操作使用统一轻量按钮样式。

快捷筛选必须是真实控件：有真实查询参数、刷新列表、激活态和 aria 状态。没有后端筛选的数字只能是静态信息，不能有 active、cursor 或按钮行为。加载、空、错误分开，错误不能显示 0 条或保留旧计数。

### B. 首页

删除五张大摘要卡和无依据 top-5 推荐。标题行右侧放：

~~~text
全部 | 今日必推 | 待我处理 | 阻塞 | 超期 | 挂起
~~~

焦点只过滤行动列表，不改变九阶段计数。阶段条继续显示真实九阶段；挂起是叠加状态，不得变成一个阶段。行动表显示事项、阶段、原始状态、下一步、下一责任人和真实操作。操作是业务办理入口；无可执行动作才显示 —。加载占位要紧凑，不能留大块固定空白。

首页焦点必须落到服务端查询：DemandsReq 增加 focus=all/today/pending/blocked/overdue/suspended，Repo 在个人行动基础集上 SQL 过滤、稳定排序、count 和分页，返回 items/total/page/pageSize。阶段计数不随 focus 改变。缺少真实字段或审计事实的焦点保持 WAIT DECISION 并不可点击；严禁拿回全量列表后在浏览器 filter。

### C. 我的待办

标题右侧放：

~~~text
待处理 | 今日到期 | 超期 | 阻塞 | P1
~~~

对象域固定为：全部待办、审批决策、需求治理、研发执行、测试质量、问题风险、个人事项。逐域检查当前真实数据源：QueryTodoUnified 目前已有 demand/task/bug；repotodoextra.go 的 approval/story/testtask/issue/risk/personal 等函数只是候选，不能因函数存在就说已接入。检查每个函数的 Schema、actor、状态、关键词括号、分页和错误传播。

对每个域返回 source status：supported、wired、available_not_wired、no_authoritative_source、blocked_schema、wait_decision 或 error。只有支持或已接线才允许正常点击；其它状态要显示原因，不用 0 欺骗用户。关系（我负责/我配合/我关注）和办理责任（待我处理/待我跟进）必须独立过滤。分类切换要清理不兼容的对象、阶段、动作条件。

R0 的 source matrix 至少核对：QueryTodoUnified 的 demand/task/bug；FindCurrentApprovalTodos 的当前 doing review 节点；FindOwnedStories、FindOwnedTesttasks、FindIssueTodos、FindRiskTodos、FindPersonalTodos 等候选函数；以及真实的 follow/star 来源。函数存在不等于接入。每个源都必须验证 actor、状态排除、截止字段、动作入口、关键词括号、错误传播、去重和 SQL 分页。任一条件缺失就保持未接入原因。

列表操作由服务端规则提供，不能在 JS 中根据状态猜。标题、ID、操作的点击语义与首页一致。

### D. 我的已办

移除宽大的累计卡和时间下拉。标题右侧直接放：

~~~text
全部 | 今天 | 近7天 | 本周 | 近30天 | 本月 | 本季度 | 更多时间
~~~

保留现有 TimeRange、对象、动作、结果筛选和正式动作白名单。表格至少清楚表达处理时间、对象、事项标题、我做了什么、处理结果和操作。最右操作统一为“查看记录”，不是普通“查看”。必须确认它指向 actor 绑定的只读动作记录详情；若没有安全详情接口，不能伪装链接，需报告阻塞并保留明确降级。

默认详情路由固定为 GET /done/records/:id，受 PoDoneList 保护。Service 必须先限制 zt_action.actor=当前账号，再执行对象查看 ACL；不存在、actor 不符或无权时返回 404/403。不能把普通 ZenTao URL 改名为“查看记录”，也不能返回无关 comment/extra 或个人数据。

不要用 target=_blank 改变既有同页行为，除非有明确外部 ZenTao 产品依据并在报告中记录。不要把待办消失自动算成已办。

### E. 通知中心

标题右侧放：

~~~text
全部通知 | 未读 | 需处理 | 异常提醒 | 今日新增
~~~

通知主题只显示短标题，摘要与主题不同才显示并限制长度；长正文进入真实详情。操作规则固定：

- 有 NeedAction 且有可信办理入口：显示“去处理”；
- 无办理入口但有可信详情：显示“查看详情”；
- 没有可信入口：显示 — 或明确“关联信息缺失”；
- “标为已读”是独立状态动作，已读不能把查看/处理入口变成 —。

桌面保留统一表格容器，通知内容单元使用短主题加一行摘要/元数据；窄屏改为堆叠行，但主题、关联对象、状态和操作必须可见。不要让通知页变成与其它页面完全不同的无边框卡片。

先审计 NoticeItem。单一 URL 不足以区分 detail/handle，必要时增加可选 DTO 字段。只能使用 notify/action 真源的 objectType/objectID，不能从正文猜 ID。mail 显示“邮件通知”。修复实体编码的 script/style、CSS 残留、标签和重复摘要；只清理显示副本，不写回数据库。PUT 2xx 非 JSON 不能假定成功。

默认通知详情路由固定为 GET /notice/items/:id，受 PoNoticeList 保护并按当前账号验证接收范围。只有 action 白名单和对象级权限都通过时才返回 handleUrl/canHandle；只有详情能力通过时才返回 detailUrl/canView。对象详情 URL 不得未经动作规则验证就充当办理入口。

## 实施顺序

按以下卡片执行，不得跳过：

1. R0 Evidence/Contract Freeze：先写 source matrix、notice association matrix 和当前证据；没有业务规则就标等待。
2. R1 Shared Shell/Home：先改首页信息层级和共享几何，不接新对象。
3. R1-Q Home Focus Query：在有 Schema、排序和隔离测试证据时接入首页 focus；否则保持 WAIT DECISION，不用前端全量过滤。
4. R2 Done：做直接时间 chips、统一“查看记录”和 actor 绑定详情/降级。
5. R3 Notice：做操作分离、关联审计、文本清理和写入失败语义。
6. R4 Todo：逐源接入有证据的 adapter；修 SQL 括号、分页、去重和错误传播。
7. R5 Acceptance：真实认证浏览器、API、Console、Network、asset revision 和独立 review。

每张卡都记录允许文件、实际修改文件、测试、视觉证据、业务门状态。不要把 R4 的数据接入混到纯 CSS 提交里。

## 代码质量和架构硬规则

- Handler 只绑定请求/返回页面或 JSON；Service 做业务规则、权限和对象级 ACL；Repo 做 SQL。
- 列表必须数据库过滤、排序、count、分页；不能全量加载后 Go 过滤/分页；不能 N+1 或每行查字典。
- 复用已有 appFetch、toast、分页、转义和错误处理；不要新建通用 ListEngine 或设计系统框架。
- 不使用 SELECT *，不拼接用户输入；动态 SQL 片段只来自服务端白名单。
- 不引入 tenantId 或 Workbench soft-delete 字段到 ZenTao 表；按真实 Schema 使用 deleted 等历史字段。
- 第一方 Go/JS/CSS/HTML 新文件保持 300 行内、绝不超过 500 行；不要压缩成超长单行逃避审查。
- 所有输入、链接、按钮、表格头和状态具备键盘/ARIA 语义；操作入口不能只靠 hover。

## 必须运行的验证

至少运行并记录：

~~~sh
go test -count=1 ./internal/module/po/...
go test -count=1 ./internal/server/...
go test -count=1 ./...
make check
git diff --check
node tests/e2e/personal-list.spec.js
~~~

若新增 Go/JS 测试，测试必须保护真实契约：

- 通知实体编码 script/style 清洗、重复主题、无关联对象、处理/详情/已读独立；
- 已办时间 chips、正式动作、actor 详情 ACL；
- 待办各源 actor/状态/关键词括号/分页/去重/source status；
- 快速筛选 URL 状态、错误清空旧计数、重置 debounce 和请求乱序；
- 非 2xx、401/403、HTML 登录跳转、2xx 非 JSON 写响应。

真实浏览器验收必须覆盖 390、960、1280、1440、1920 宽度，并在每页操作筛选、重置、分页和行操作。记录 Console 错误、Network status、请求参数、响应 envelope、实际加载资产版本和截图路径。截图好看但 API/Console/权限不对时，结论仍是失败或部分完成。

## 报告和停止条件

新增 docs/plan/personal-workbench-ui-20260905/EXECUTION-REPORT-V2.md。报告必须分开写：

- 静态代码结果；
- Go/JS/make check；
- API 结果；
- 真实浏览器视觉和交互结果；
- Todo source matrix；
- Notice association matrix；
- 已修复缺陷；
- WAIT DECISION、BLOCKED、未接入和风险；
- commit/push（本轮未授权则写未做）。

以下情况立即停止并报告，不要猜：分支/HEAD 不符；需要未知数据库或写 SQL；必须改变 actor/tenant/权限政策；只有伪造 URL 才能显示操作；必须用全量内存分页；参考 PRD 与当前业务冲突无人裁决。

完成用词只能依据证据：所有卡和门通过才写 COMPLETE；视觉或数据只完成一部分写 PARTIAL；缺真实数据/业务决策写 BLOCKED 或 WAIT DECISION。不要再用一个 UI READY 覆盖未接入、详情入口缺失或浏览器验收缺失。
