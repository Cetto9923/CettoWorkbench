# 员工工作台四页 UI 对齐与接入治理 Plan V2

状态：**PLAN READY**。本文件是下一轮执行合同，不是实现完成证明。

适用仓库：/Users/yuyan9923/GitHub/workbench-claude-po
执行分支：Claude-PO
编制基线：2f2580a10f45db5978736f1fe1e46c258133022f
运行观察地址：http://127.0.0.1:8093
目标页面：/home、/todos、/done、/notice

本文件补充并**覆盖旧 PLAN.md 中关于首页大摘要条、已办时间下拉、通知操作列和“全部未接入占位”的视觉/交互决策**。旧计划中已经完成且仍然适用的请求竞态、分页、错误态、转义和文件边界要求继续有效；不要重新执行已经 COMPLETE 的工作包，也不要把旧 EXECUTION-REPORT.md 的 UI READY 当作本轮四页验收结论。

本轮不修改 workboard 相关 WIP，不修改分支，不提交，不推送。执行者必须先完成本文件的证据和契约门，再按卡片执行。任何无法由当前 PRD、参考项目或当前代码/接口证据确认的业务规则，必须标为 WAIT DECISION，不能由执行模型自行补全。

## 1. 这轮要解决的实际问题

四页目前有一个“统一了颜色和边框、没有统一任务语言”的问题：页面外壳相似，但焦点入口、筛选密度、行操作、状态表达和数据接入状态不一致。首页又把有限的可视高度消耗在五张大卡片上，真正的行动列表在截图中出现大片空白。已办使用宽大的时间选择条，用户必须多一步才能切换。通知把“标记已读”当成唯一行操作，无法判断消息是否需要业务处理。待办的多个分类直接写成“未接入”，但仓库已经存在一批未被调用的对象查询函数，尚未区分“代码存在但没有接线”和“确实没有可信数据源”。

本轮交付结果必须同时满足：

1. 四页共享同一套标题、右上快捷筛选、分类、工具栏、结果条、表格操作和响应式几何。
2. 首页优先展示“下一步行动”，使用紧凑快捷筛选，不再使用五张大 KPI 卡片占据主区域。
3. 我的待办、我的已办、通知中心的行操作表达业务语义：可办理时办理，可查看时查看，已读是独立状态动作。
4. 每一个“未接入”都能说明是已接线、有适配器但未接线、无可信来源、被权限/Schema 阻塞或等待业务决策，不能用 0 伪装成空数据。
5. 页面只显示服务端确认的对象、动作、计数和 URL；浏览器不得从标题、正文或展示编号猜业务关联。
6. UI 改造不能破坏现有路由、权限、CSRF、筛选参数和 ZenTao 对象链接协议。

## 2. 证据等级和参考材料

执行顺序按以下优先级处理：

| 优先级 | 真源 | 用途 |
|---|---|---|
| 1 | 当前用户要求和本文件冻结决策 | 决定本轮要做什么、页面终态交互是什么 |
| 2 | 当前仓库 AGENTS.md 与 docs/engineering/*.md | 决定架构、权限、查询、文件和验证边界 |
| 3 | CRCBWorkbench 的 PO PRD、原型和已验证业务模式 | 提供业务语义、布局方向和操作命名；不直接复制代码 |
| 4 | 当前仓库实际 Handler/Service/Repo/Template/JS/CSS/API | 判断哪些能力已存在、哪些只是占位、哪些会回归 |
| 5 | 旧截图和旧执行报告 | 只作为问题证据和历史记录，不作为终态批准 |

必须阅读当前仓库：

- AGENTS.md
- docs/engineering/architecture.md
- docs/engineering/database.md
- docs/engineering/frontend.md
- docs/engineering/testing.md
- docs/engineering/quality.md
- docs/plan/personal-workbench-ui-20260905/PLAN.md
- docs/plan/personal-workbench-ui-20260905/EXECUTION-REPORT.md
- docs/plan/personal-workbench-ui-20260905/review/INDEPENDENT-REVIEW.md

必须阅读参考项目材料：

- /Users/yuyan9923/GitHub/CRCBWorkbench/docs/PRD/17-PO角色开发需求说明书/PO开发需求说明书_MD_V1/科技研发智能工作台_PO角色开发需求说明书_V1.md 的首页、我的待办、我的已办、通知中心章节。
- /Users/yuyan9923/GitHub/CRCBWorkbench/docs/PRD/15-我的待办和已办/Done Action Inventory.md
- /Users/yuyan9923/GitHub/CRCBWorkbench/docs/PRD/15-我的待办和已办/我的已办_详细需求设计_V1.0.md
- /Users/yuyan9923/GitHub/CRCBWorkbench/docs/PRD/17-PO角色开发需求说明书/PO开发需求说明书_MD_V1/images/01-home.png
- /Users/yuyan9923/GitHub/CRCBWorkbench/docs/PRD/17-PO角色开发需求说明书/PO开发需求说明书_MD_V1/images/02-todo*.png
- /Users/yuyan9923/GitHub/CRCBWorkbench/docs/PRD/17-PO角色开发需求说明书/PO开发需求说明书_MD_V1/images/04-notify.png
- /Users/yuyan9923/GitHub/CRCBWorkbench/docs/PRD/17-PO角色开发需求说明书/screenshots/04-my-done.png
- /Users/yuyan9923/GitHub/CRCBWorkbench/docs/PRD/17-PO角色开发需求说明书/screenshots/05-notice-center.png

参考项目的工程实现不是本仓库 Golden Reference。可以借鉴“快捷筛选位于标题行右侧”“通知有去处理/查看详情”“已办显示查看记录”等已确认产品语义，不能直接复制其数据库、Service、前端框架或跨模块假设。

## 3. 执行前 Preflight 和保护边界

执行模型必须先运行并记录：

~~~sh
pwd
git rev-parse --show-toplevel
git branch --show-current
git status --short
git remote -v
git rev-parse HEAD
~~~

必须确认仓库根目录、分支和基线。若 HEAD 已变化，先报告差异并重新建立基线；不能把旧截图或旧报告直接套在新代码上。必须保留所有已有修改和未跟踪文件，尤其是：

- web/static/css/po/board.css
- web/static/js/po/workboard.js
- web/templates/po/workboard.html
- 当前 Gemini 对四页的工作树修改

禁止：

- reset、clean、restore、stash、回滚或删除用户 WIP；
- 切换、合并、变基或创建分支；
- 访问未知生产库、猜测 DSN、运行 migration/AutoMigrate 或任何写 SQL；
- 把 install.sql 当作真实线上 Schema；
- 扩展为 React/Vue、设计系统框架、全站重构或新建通用数据总线；
- 重新执行旧计划已 COMPLETE 的通用 transport/pagination 工作，只修本文件列出的确认缺陷或为新契约补测试。

8093 是用户正在使用的实例。不要 kill、替换或声称它已经加载了最新 Go 源码，除非通过构建时间、进程 cwd、资产版本和接口响应四项证据确认。浏览器验收必须使用已认证会话，并记录 DOM、网络状态、Console 和实际加载的 CSS/JS 版本。

## 4. 当前问题台账

| 编号 | 当前证据 | 影响 | 本轮要求 |
|---|---|---|---|
| V2-HOME-01 | home_desktop-1440.png 和用户截图显示五张大摘要卡高度大、内容稀疏；焦点没有右上快捷入口 | 首屏被指标占用，切换焦点不高效 | 删除首页大卡片结构，改为标题行右侧紧凑快捷筛选 |
| V2-HOME-02 | 首页行动列表在截图中出现“正在加载行动列表…”和大片空白 | 行动入口不可用或误以为无数据 | 加载占位必须紧凑；成功后列表自然增长；失败与空态分离 |
| V2-HOME-03 | 当前首页快捷指标没有确认筛选绑定，却有 active/cursor 视觉 | 用户误以为可点击 | 只有有真实查询参数的控件才交互；静态统计不得有选中态 |
| V2-DONE-01 | done-summary-strip 由累计卡和宽时间下拉组成 | 时间范围多一步，页面层级浪费 | 改为标题行右侧直接时间 chips；自定义范围收进更多时间 |
| V2-DONE-02 | done.js 行操作显示普通文本“查看” | 与待办操作不一致，也没有表达查看动作记录 | 使用统一操作样式，文案固定为“查看记录”；目标必须是真实记录或明确详情 |
| V2-NOTICE-01 | notice.js 已读行显示 —，未读行只显示“标为已读” | 无法进入业务处理或详情，已读被误当成无操作 | 去处理、查看详情、标为已读分离；read 不能决定是否可查看 |
| V2-NOTICE-02 | notice_desktop-1440.png 出现 span.pass(color:#60D978 span.fail(color:red) 等残留 | 暴露内部标记，清洗不完整 | 修复实体编码、HTML/CSS 残留；前端继续安全转义 |
| V2-NOTICE-03 | NoticeItem 只有一个 URL；action 的对象类型和 notify 的对象 ID 组合未验证 | 可能链接到错误对象，无法区分处理和查看 | 先固定关联证据和 DTO，再实现操作；缺失关联不得造链接 |
| V2-TODO-01 | QueryTodoUnified 实际只 UNION demand/task/bug；模板出现多个“未接入”分类 | 混淆未接线和无可信来源 | 建立 source status；已有可靠 adapter 优先接线，真正缺失才保留说明 |
| V2-TODO-02 | repotodoextra.go 有候选查询但无调用；关键词 OR 有优先级风险，重复加载字典并吞错 | 盲目接入会产生错误结果、N+1 和假分页 | 逐适配器审查，修括号、错误传播和查询计划后再接入 |
| V2-CROSS-01 | 颜色接近，但概览、分类、工具栏、操作列和长文本表达仍各自为政 | 页面之间需要重新学习 | 统一 shell 几何和状态语义，保留各页业务字段 |
| V2-QUALITY-01 | 旧报告把静态门禁通过等同 UI READY，未披露截图残留 | 评审依据失真 | 新报告必须区分静态、接口、浏览器视觉和业务接入状态 |

## 5. 冻结的产品与交互决策

### 5.1 共享页面骨架

四页统一为：

~~~text
标题与说明                         右上紧凑快捷筛选
分类/关系（按页面需要）
工具栏（搜索 → 常用筛选 → 更多筛选 → 重置）
结果状态条
列表/表格
分页
~~~

共享几何：

- 侧栏和顶栏沿用现有外壳，不改全站 layout；内容区左右 20px、顶部 16px，窄屏 12px。
- 标题 20px/28px，说明 12px/18px；面板白底、1px 边框、6px 圆角；模块间距 12px。
- 快捷筛选为 30–36px 高的轻量按钮/chip，标签和数字同一行；不能使用 64–80px 大卡片。
- 激活态使用蓝色边框/底线和 aria-pressed=true 或 aria-selected=true；不能只改变背景色。
- 操作入口统一使用 .table-action-btn 或已确认的共享操作类；同类动作不要一个页面用文字、另一个页面用实心大按钮。
- — 只表示没有可执行动作且没有可用详情入口。
- 加载、空、错误是三种独立状态；错误时不得保留旧计数和旧分页。
- 结果条使用 aria-live=polite；列表请求期间设置 aria-busy=true；所有按钮显式 type=button。
- 内部链接同页打开。只有已有产品理由明确要求外部 ZenTao 新窗口时才使用 target=_blank rel=noopener noreferrer，并在报告中列明理由。

### 5.2 首页：从统计仪表盘改为行动中枢

首页固定结构：

1. 标题“工作台首页”和一行说明；标题右侧放焦点 chips。
2. 需求价值流阶段条：保留九阶段及真实计数，阶段点击只过滤下方行动列表，不重算阶段总数。
3. 统一行动列表：真实 API 返回多少就展示多少，采用自然高度和分页，不用固定大片空白。
4. 版本窗口：紧凑卡片，仅显示真实窗口名称、范围、团队和统计。

焦点 chips 固定顺序和语义：

| key | 文案 | 作用 |
|---|---|---|
| all | 全部 | 清除首页焦点条件 |
| today | 今日必推 | 今日有效截止或已逾期且未完成 |
| pending | 待我处理 | 办理责任明确属于当前用户 |
| blocked | 阻塞 | 存在明确阻止下一步的事实 |
| overdue | 超期 | 有有效截止日期、已过期且未完成 |
| suspended | 挂起 | 最近一次挂起尚未被恢复闭合 |

焦点只过滤行动列表；阶段 chips 的计数不随焦点变化。挂起是叠加维度，不能变成第十个价值流阶段。没有可确认的后端焦点参数时，必须保留真实可用的阶段筛选，并将该焦点标为 WAIT DECISION，不能做前端猜测。

首页不再渲染 summary-strip 的五张大卡片。若需要保留总数，使用标题右侧或阶段条旁的非占位数字文本，不增加第二套统计条。不得恢复高优前三卡片，不得新增没有业务规则依据的推荐/优先级算法。

行动列表列固定为：事项（标题+ID）、当前阶段、原始状态、下一步、下一责任人、操作。操作显示当前业务动作，如受理、澄清、排期、建任务、验收、催验收、发起交付、跟进上线、挂起、恢复；没有真实动作时显示 — 或只读状态。标题打开工作台详情，ID 打开 ZenTao 原生详情，操作直接进入业务办理。

首页快捷筛选的服务端契约固定为：DemandsReq 增加可选 focus，允许 all、today、pending、blocked、overdue、suspended；Repo 在 SQL 基础集合上过滤，不能由浏览器拿回全量列表后过滤。该过滤只影响 /demands 返回的行动列表，不影响 Home 阶段计数。若当前阶段没有经过 Schema/业务证据确认的 focus 条件，页面必须把该 chip 标为 WAIT DECISION 并保持不可点击，不得用前端猜测代替查询。

### 5.3 我的待办：统一筛选层级，接入状态可解释

顶部右侧使用紧凑焦点 chips，固定映射当前 TodoSummary：

~~~text
待处理 | 今日到期 | 超期 | 阻塞 | P1
~~~

焦点 chips 必须通过真实 focus 查询参数工作，点击后刷新列表；当前默认焦点保持现有合同，不得因视觉改造默默改默认值。统计数字与下方同一响应来源，不在前端重新计算。

对象域顺序固定为：

~~~text
全部待办 | 审批决策 | 需求治理 | 研发执行 | 测试质量 | 问题风险 | 个人事项
~~~

每个域的呈现状态：

| sourceStatus | 页面表达 | 是否允许点击 |
|---|---|---|
| supported | 正常显示数量和列表 | 是 |
| wired | 正常显示；报告记录 adapter 来源 | 是 |
| available_not_wired | 显示待接线及原因 | 否，直到接线验收 |
| no_authoritative_source | 显示暂无可信来源及原因 | 否 |
| blocked_schema | 显示 Schema/权限阻塞 | 否 |
| wait_decision | 显示待业务决策 | 否 |
| error | 显示加载失败，提供重试 | 可重试，不显示 0 |

必须先核对 repotodoextra.go。若某个查询能够依据当前 Schema、actor 规则、状态和排序接入，就将其接到统一结果；若只能证明函数存在而无法证明契约，则保留明确原因，不把它标成支持。story/testtask/issue/risk/personal/approval/follow 的接入不得靠复制 reference project 的前端数组完成。

筛选层级固定为：对象域 → 我的关系（我负责/我配合/我关注）→ 办理责任（待我处理/待我跟进）→ 办理场景 → 价值流阶段 → 具体对象 → 关键词。切换对象域时清除不兼容的对象、阶段、办理责任和办理场景；URL 只保存当前有效条件。

待办表格至少保留：ID/事项、对象、优先级、关系、形成原因/阶段、截止、责任人、操作。操作由后端动作规则产生，不在 JS 里通过状态字符串臆测。标题和 ID 的跳转规则与首页一致。

R0 输出的待办 source matrix 必须至少按下面的现状线索核验，不能把“参考项目有实现”直接写成当前仓库支持：

| 对象域 | 当前仓库线索 | 只有满足以下条件才可标 supported/wired |
|---|---|---|
| 需求治理 | QueryTodoUnified 的 demand 分支 | 沿用真实 deleted/status、父子去重、当前账号责任和既有动作规则；count 与分页同源 |
| 研发执行 | QueryTodoUnified 的 task 分支及可能的 story adapter | task 的 assigned/team 责任、状态排除、截止和操作 URL 均有当前 Schema 证据；story 不能仅因函数存在接入 |
| 测试质量 | 当前 bug 分支及 FindOwnedTesttasks 候选 | bug/testtask 的真实字段、责任、关闭状态和动作均验证；不能把 bug 数当测试单数 |
| 审批决策 | FindCurrentApprovalTodos 候选，基于 zt_approvalnode/approvalobject 等表 | 只取当前用户 status=doing、type=review 的节点；未来 wait/抄送不计入；每种审批类型有可信办理入口 |
| 问题风险 | FindIssueTodos/FindRiskTodos 候选 | assignedTo/createdBy 规则、closed/cancel 排除、标题字段和对象级 ACL 由证据确认；风险不自动等于阻塞 |
| 个人事项 | FindPersonalTodos 候选 | account/assignedTo 语义、状态和动作入口由产品与 Schema 同时确认；不能把任意 todo 记录塞进工作台 |
| 我的关注 | 只有确认的 star/follow 真源 | followed=1、取消关注覆盖和对象范围均有现有契约；没有来源则保留 wait_decision |

候选函数的关键词条件必须把所有 OR 子句括在同一组内；字典加载错误必须返回错误；接入后必须由一个统一查询计划负责排序、去重、count 和分页。若某一行缺少任意条件，sourceStatus 不得升级为 supported。

### 5.4 我的已办：直接时间 chips + 动作记录

标题行右侧使用直接可点时间 chips，固定顺序：

~~~text
全部 | 今天 | 近7天 | 本周 | 近30天 | 本月 | 本季度 | 更多时间
~~~

前七项直接映射现有 TimeRange；“更多时间”才打开自定义范围，不允许把常用范围藏在原生下拉中。累计处理和当前范围计数可作为 chips 中的数字或一条紧凑辅助文本，不能再使用一块横向填满的 done-range-bar。

分类仍按业务场景：全部已办、审批决策、需求治理、研发执行、测试质量、问题风险。对象类型和动作/结果筛选保留；不要为了好看删除正式动作过滤。

表格优先采用参考原型语义列：

~~~text
处理时间 | 对象 | 事项标题 | 我做了什么 | 处理结果 | 状态变化 | 所属项目/执行 | 当前状态 | 操作
~~~

当前 DTO 尚未提供的字段不得造值。可以先保留当前已确认字段，但列名和层级必须表达“我做了什么”，不能将动作记录写成笼统的“查看”。最右操作文案固定为 **“查看记录”**，统一按钮样式。

“查看记录”必须有真实目标：优先新增 actor 绑定的只读已办记录详情接口/抽屉，返回该 zt_action.id 的动作、对象、时间、结果、状态变化和可查看历史；如果当前阶段不能安全提供记录详情，则不得伪装成普通对象链接，需将该卡标为 WAIT DECISION 并暂时使用明确对象详情入口或 —。详情查询必须校验当前用户只能查看自己的动作记录和其有权查看的对象，防止 IDOR。

为避免执行模型自行选择路由，默认详情契约为 GET /done/records/:id，受 PoDoneList 保护。Service 先按当前账号读取 zt_action，再按 objectType/objectID 调用已有对象查看 ACL；找不到动作、actor 不匹配或对象无权查看均返回 404/403，不泄露动作是否存在。响应只返回动作记录所需字段，不能把整个 zt_action.comment/extra 或无关个人数据塞进列表。若在 R0 证明该路由会改变既有权限/对象协议，R2 必须停在 WAIT DECISION，而不是将普通对象 URL 改名为查看记录。

### 5.5 通知中心：事件卡片语义，已读独立于业务处理

顶部右侧快捷 chips 固定为：

~~~text
全部通知 | 未读 | 需处理 | 异常提醒 | 今日新增
~~~

分类固定为：全部分类、业务动态、审批流程、时效提醒、协作消息、风险异常、系统消息；实际分类必须由服务端事件映射产生，前端不根据正文猜分类。

通知行/卡片分成四层：

1. 主题：短标题，最多 60–80 个 rune；不把 200 字正文全部塞入一行。
2. 摘要：有真实摘要且与主题不同才显示，默认 1–2 行；点击详情可看完整正文。
3. 元数据：分类、对象、触发人、通知时间、已读状态。
4. 操作：
   - NeedAction=true 且有可信办理入口：去处理；
   - 没有办理入口但有可信通知/对象详情：查看详情；
   - 以上都没有：显示 — 或关联信息缺失，不造链接；
   - 标为已读是独立次要按钮，已读后也不能移除查看/处理入口。

桌面（可用宽度大于等于 960px）保留统一表格容器和固定操作列；通知内容单元使用“短主题 + 一行摘要/元数据”的两行结构，不把长正文扩成多列。窄屏将每行堆叠成同一条通知卡，主题、关联对象、状态和操作仍然直接可达。不要在四页之间混用一页无边框卡片、一页裸表格的容器层级。

当前 NoticeItem.URL 不足以表达这些能力。执行者必须先完成 DTO/关联审计，再选择兼容扩展字段，例如 detailUrl、handleUrl、canView、canHandle、read；旧 url 字段过渡期保持兼容，但不能把同一 URL 同时当办理入口和详情入口而不记录语义。

通知关联只使用 zt_notify 和对应 zt_action/事件真源中的权威 objectType/objectID。若 action 与 notify 对象不一致，按真实关联规则处理并记录；不能从主题、正文、邮件 HTML 或展示编号提取 ID。mail 显示“邮件通知”，无对象时明确显示“关联信息缺失”。

通知能力判定固定按以下顺序执行：先验证当前用户确实在 toList/事件接收范围内，再验证关联对象和对象级查看权限，最后从服务端白名单查找 handle route。只有三项都通过才返回 canHandle=true/handleUrl；只有前两项通过且存在通知详情接口才返回 canView=true/detailUrl。对象详情 URL 不能未经动作规则验证就冒充办理 URL。建议新增的只读详情路由为 GET /notice/items/:id，必须绑定 PoNoticeList、按当前账号查通知并限制正文长度；路由不存在或 ACL 未完成时，显示明确降级，不渲染“查看详情”假按钮。

## 6. 后端与前端契约冻结

### 6.1 响应 DTO 最小扩展

列表 API 继续使用现有路由和权限，不破坏已有字段。允许新增以下可选字段：

| 字段 | 页面 | 含义 | 生成位置 |
|---|---|---|---|
| sourceStatus | 待办域 | supported/wired/blocked 等 | Service/Repo 来源审计 |
| sourceReason | 待办域 | 未接入或阻塞原因 | 服务端固定文案 |
| canView | 通知/已办 | 当前 actor 是否有只读详情入口 | Service ACL |
| canHandle | 通知/待办 | 当前 actor 是否有真实业务办理入口 | Service 规则 |
| detailUrl | 通知/已办 | 只读详情入口 | 服务端可信路由 |
| handleUrl | 通知/待办 | 业务处理入口 | 服务端可信路由 |
| actionLabel | 首页/待办 | 当前业务动作中文名 | Service action rule |
| statusChange | 已办 | 已确认的前后状态变化 | zt_action/history 真源 |

字段为空时前端必须有明确降级，不得凭 objectType、状态、标题或 action code 拼出不存在的业务能力。API 错误用统一 JSON envelope；401/403/500 分别给登录入口、无权限提示和重试，不把错误渲染成空列表。

### 6.2 查询与性能边界

- 任何列表均遵循 SQL filter → SQL sort → SQL count → SQL pagination。
- 跨多表聚合先给出查询计划、索引、去重键、稳定排序和预期 SQL 数；不能分别取全量再在 Go 中拼页。
- repotodoextra.go 每个查询必须修复用户关键词 OR 的括号优先级；不得吞掉字典查询错误；同一请求只加载一次用户展示名映射。
- 不使用 SELECT *，不把大正文列全部返回列表；通知正文只在详情接口/抽屉按需读取。
- 现有 ZenTao 表按真实 Schema 使用 deleted 等字段；不要引入 deletedAt、tenantId 或 Workbench BaseModel 假设。
- 写操作仍由 Service 做权限和业务状态判定，浏览器隐藏按钮不能代替授权。

### 6.3 URL、权限和安全

- 当前 /home、/todos、/done、/notice 和 JSON 路由及权限码保持不变。
- 新增详情 GET 必须挂在对应列表权限下，并执行 actor/object ACL；不新增只要知道 ID 就能看的查询。
- PUT /notice/:id/read 与 PUT /notice/read-all 继续使用 CSRF 和现有 envelope；读取失败、空体或非 JSON 不能假定成功。
- HTML/邮件正文只做显示副本清理：解码一次、去除 script/style 内容、剥离标签、归一空白、按 rune 截断；不得回写 zt_notify。
- 所有用户可见文本经过 textContent 或统一 escape；不要把清洗后的正文当模板 HTML 插入。

## 7. 执行卡片与依赖

每张卡必须在报告中标 COMPLETE、PARTIAL、WAIT DECISION 或 BLOCKED；没有通过卡片验收不能进入下一张依赖卡。

### R0 — Evidence / Contract Freeze

目标：锁定当前代码、参考 PRD、真实 API 和可接入对象清单。

允许修改：本目录新增证据/设计文档；不改业务代码。

必须完成：

- 记录 preflight 和工作树 WIP；
- 对照当前页面 DOM/API，确认每个计数、tab、操作 URL 的真实来源；
- 建立待办 source matrix：对象、表、actor 规则、状态、截止、动作、分页、当前是否接线；
- 建立通知 association matrix：notify/action 对象来源、read 来源、handle/detail 能力；
- 明确哪些“未接入”是 available_not_wired，哪些是 no_authoritative_source。

验收：没有架构或业务假设留给后续模型；未知项有具体证据缺口和决策输入。需要数据库时只能使用批准的只读 Schema evidence，禁止写 SQL。

### R1 — Shared Shell and Home Redesign

依赖：R0。目标：只改变四页共享几何和首页信息层级，不扩大业务范围。

允许文件：home 模板、personal-workspace.css、home.css、homecompact.css、home.js 及相邻测试。

必须完成：

- 移除首页五张大摘要卡及其伪激活样式；
- 将首页焦点 chips 放在标题行右侧；
- 保留阶段条、真实计数和阶段筛选；
- 消除固定大空白和无依据 top-5 推荐；
- 确保 390/960/1280/1440/1920 视口阶段和操作入口可发现；
- 修复成功/加载/空/错误布局跳动。

禁止：本卡接入新对象、改变 KPI 统计口径、增加图表或业务动作。首页 focus 查询契约由 R1-Q 单独完成；在 R1-Q 未通过时，相关 chip 必须保持不可用并显示待决策原因。

验收：截图中首屏标题、快捷筛选、阶段条和行动列表层级符合本文件；快捷筛选点击可追踪到真实 query；无绑定的指标没有 cursor:pointer 或 active 假象。

### R1-Q — Home Focus Query and Bounded Action List

依赖：R0；若没有真实 Schema/排序/隔离测试证据则 BLOCKED，不得用前端过滤补偿。目标：让首页焦点可用且不把全量行动列表加载到浏览器。

允许文件：DemandsReq、Home/Service、价值流 Repo、/demands Handler、首页 JS/测试，以及本卡专属证据文档。

冻结契约：/demands 接受 status 与 focus；返回 items、total、page、pageSize。Repo 先按个人行动基础集、父子去重、status/stage/focus 过滤，再按固定稳定顺序排序、count、LIMIT/OFFSET。阶段计数继续由 Home 阶段查询产生，不受 focus 影响。focus 的 blocked/overdue/suspended 条件必须逐项指向真实字段或审计事实；缺少事实的项保持 WAIT DECISION。

验收：同一焦点相邻页无重复/漏项；不同 focus 不扩大 actor 范围；空、错误、越界页有明确响应；查询计划没有 Go 全量分页、N+1 或未经审查的 per-stage fan-out。未取得隔离数据库和业务排序证据时，只完成 R1 视觉，不把 R1-Q 标为完成。

### R2 — Done Direct Range and Record Action

依赖：R0、R1 的 shared shell。目标：已办采用紧凑时间切换和“查看记录”语义。

允许文件：done 模板/CSS/JS、form.go、servicedone.go、repodone.go、handler.go 及测试；若新增详情接口，必须单独列出路由和 ACL。

必须完成：

- 直接时间 chips 映射现有 TimeRange；自定义范围单独处理；
- 统一操作按钮为“查看记录”；
- 明确动作记录详情的数据来源和 actor 边界；
- 修复 tab/动作/结果/对象之间的不兼容条件残留；
- 保留正式动作白名单、计数和分页，不把待办消失当已办。

禁止：把“查看记录”伪装成普通 ZenTao 链接；删掉正式动作筛选；复制 reference project 的大块 Service。

验收：每个时间 chip 改变 timeRange 并刷新；当前范围计数与响应字段一致；行按钮样式和待办一致；详情/对象链接同页规则、权限和失败态可验证。

### R3 — Notice Actions, Association and Text

依赖：R0、R1 的 shared shell。目标：恢复通知处理/查看能力，修复文本和对象关联风险。

允许文件：notice 模板/CSS/JS、formnotice.go、servicenotice.go、reponotice.go、notice_query_repo.go、handler.go 及测试。

必须完成：

- DTO 分离 canHandle/canView/handleUrl/detailUrl/read；
- NeedAction + 可信 handle URL 显示“去处理”；否则有 detail 显示“查看详情”；已读动作单独存在；
- 验证 notify/action objectType/objectID 组合，禁止拼错关联；
- 主题/摘要短文本显示；详情按需读取；
- 清洗实体编码的 script/style、标签、注释和 CSS 残留，前端继续转义；
- PUT 2xx 非 JSON 或失败必须提示失败/不确定，不静默成功。

禁止：从正文猜 ID；把所有已读通知操作改为 —；回写通知表；引入重量级 HTML 解析框架。

验收：至少有一条需处理、一条仅查看、一条无关联、一条已读通知的可验证 fixture/Mock；四种行操作符合契约；恶意/实体编码文本不执行且不露 CSS/JS 标记。

### R4 — Todo Source Reconciliation and Wiring

依赖：R0；R1 shell 可并行，但业务接线不得跳过 R0。

目标：把“未接入”变成有证据的 source capability，并在安全时接入现有 adapter。

允许文件：todo_query_repo.go、repotodoextra.go、repotodoapproval.go、对应 Service/Form/测试，以及待办模板/JS/CSS。

必须完成：

- 逐源确认 demand/task/bug/approval/story/testtask/issue/risk/personal/follow 的 actor、状态、关系、截止、动作和对象链接；
- 将已存在且契约完整的 adapter 接入统一结果；
- 修复关键词 OR 括号、重复 display map、错误吞掉和无界加载；
- 统一排序、去重、count、LIMIT/OFFSET；
- 返回 source status/reason，让 UI 区分 0 条和未接入；
- 关系和办理责任分别过滤；
- 保持审批只读当前 doing 节点，未来 wait/cc 不进入待办；风险/问题按确认的 assigned/created 规则处理。

禁止：把所有 extra 函数一次性 UNION 而不验证 Schema；用空数组覆盖错误；用前端状态启发式生成操作；为了显示全接入发明 tenant/owner/阶段政策。

验收：每个标 supported/wired 的域都有 API/测试证据；每个未接入域有明确 reason；分页相邻页无重复/漏项；无 N+1、无 Go 全量分页；权限和对象范围不扩大。

### R5 — Integrated Browser Acceptance and Independent Review

依赖：R1、R1-Q、R2、R3、R4 中实际标 COMPLETE 的卡；R1-Q 若 BLOCKED，首页焦点验收必须明确标 BLOCKED，不能把 R1 的静态 chips 当可用筛选。

必须完成：

- 视口 390×844、960×900、1280×800、1440×900、1920×1080；
- 每页首次加载、筛选、重置、分页、错误/空态、返回恢复；
- 首页每个快捷焦点、阶段筛选、列表操作；
- 待办七域 source status、关系/责任/办理筛选；
- 已办全部时间 chips、自定义范围、查看记录；
- 通知去处理/查看详情/标已读/全部已读，read 不改变业务待办；
- 记录 Console、Network、实际 asset revision、HTTP status 和响应 envelope；
- 运行 go test -count=1 ./internal/module/po/...、go test -count=1 ./internal/server/...、go test -count=1 ./...、make check、git diff --check 及现有前端行为测试。

验收：新报告逐卡列出静态、API、浏览器和业务证据；任何阻塞或未接入保持原状态，不得写成 READY/COMPLETE。

### 7.1 推荐执行批次

为避免 Gemini 一次同时改视觉、通知和跨表聚合，第一次执行只安排 **R0 + R1**。R0 必须先产出 source/association matrix；R1 只做共享外壳和首页视觉层级。R1-Q 需要独立的 Schema、排序和隔离测试证据，不能因为 R1 完成就自动开始。第二批最多安排 R2 + R3；R4 单独执行并单独评审；最后才执行 R5。

## 8. 测试要求

测试目标是保护契约，不是为 CSS 颜色写镜像测试。

### Go / Repo / Service

- 通知文本清洗覆盖普通 HTML、实体编码 script/style、CSS 残留、中文 rune 截断、重复 subject/data、空正文和恶意标签。
- 通知关联覆盖 action object 与 notify object 一致、不一致、0 ID、mail、无 URL；检查不会从正文补 ID。
- Done 覆盖 actor 过滤、正式动作白名单、时间 chips 映射、动作/结果筛选、详情 actor ACL、对象查询错误向上传播。
- Todo adapter 覆盖每一源的 actor/状态条件、关键词括号、去重键、稳定排序、分页边界和 source status；没有真实 Schema 的源必须明确 skip/block reason。
- Handler 覆盖 401/403/500/空响应和 JSON envelope；写操作失败不得返回假成功。

### Browser / JS

- 共享控制器继续保护旧响应不覆盖新筛选；重置清理 debounce；错误清理旧计数和页码。
- 快捷 chips 更新 URL/query 和 aria-pressed；浏览器返回可恢复；不兼容筛选不能隐藏残留。
- 操作按钮键盘可达、文案明确、focus-visible；外链仅在有记录理由时新窗口。
- 通知主题不能显示内部 CSS/HTML；完整摘要通过真实详情入口可读。

## 9. 报告、Git 和停止条件

执行者必须新增或更新本目录内的 EXECUTION-REPORT-V2.md，至少包含：

~~~text
# Four-Page UI V2 Execution Report
执行基线与工作树
证据来源与冲突处理
R0-R5 与 R1-Q 状态
首页/待办/已办/通知行为变化
Todo source matrix
Notice association matrix
静态测试、API、浏览器和 Console/Network 证据
确认缺陷与未修复项
WAIT DECISION / BLOCKED
commit / push（未做则明确写未做）
~~~

本轮未授权提交或推送。若后续用户单独授权提交，只能精确暂存本轮文件，不能 git add .；push 只允许用户明确指定的 github Claude-PO，禁止 bare push、origin、audit 或 force push。未完成 R5 或仍有业务门阻塞时，报告必须写 PARTIAL/BLOCKED，不能用 UI READY 概括。

以下任一情况必须 STOP AND REPORT：

- 分支、仓库或用户 WIP 不符合 Preflight；
- 发现需要更改权限/租户/actor 规则但没有业务决策；
- 需要未知或生产数据库、写 SQL、migration 或真实通知写入；
- 发现参考 PRD 与当前实现冲突且无法由当前用户要求裁决；
- 无法提供真实 handle/detail URL 却试图显示可执行按钮；
- 关键列表只能靠内存全量加载或猜排序才能实现。

## 10. 完成判定

只有当 R0–R5 与 R1-Q 中本轮被授权的卡片全部有对应证据，且 make check、相关 Go/JS 测试、真实认证浏览器验收、Console/Network/asset 检查均通过时，才可以称“四页 UI V2 已完成”。

视觉好看、静态截图无横滚、接口返回 200 或单元测试通过，均不足以单独形成完成结论。业务源仍未接入、通知关联未确认、查看记录没有 actor ACL，或任何关键动作只是前端占位时，必须保留 WAIT DECISION/BLOCKED。
