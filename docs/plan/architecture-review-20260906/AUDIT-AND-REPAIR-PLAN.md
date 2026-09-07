# 架构与工程规范审查、修复计划

审查开始：2026-09-06；报告形成：2026-09-07（跨午夜）。审查状态：**PARTIAL（源码、机器门禁和隔离探针已核验；真实数据库、认证浏览器和当前远端 CI 未核验）**。

## 1. 判断与范围

**总体架构方向正确，工程规范执行不一致。当前不能认定全面合规，也没有证据支持推倒重写。**

Go / Gin / GORM / MySQL / SSR 单体及 Handler → Service → Repo 的主体结构仍然存在；近期权限、CSRF、SQL 分页、错误处理和读写连接拆分确有进展。问题集中在新增详情与周报能力：业务真值、对象访问、错误语义、查询规模和前端复用没有同步受到约束。文件拆小和 `make check` 通过，并不能消除这些问题。

审查基于 `Claude-PO`，HEAD `a430311a6d46d99a12d821180efbb3b1854d9d39`，包括当前未提交及未跟踪代码。开始时有 38 项已跟踪修改、24 项未跟踪状态项（目录状态项可能包含多个文件）。追溯了 2026-09-04 至本次审查的近期提交；重点阅读 PO、schedule、登录/会话、路由、中间件、共享前端、质量脚本及相关计划。其它模块只做边界扫描和关键点抽查，不是逐函数认证。

本次只新增本报告和证据说明；未修改业务代码、质量基线、数据库或运行服务，未切分支、提交或推送。临时探针通过 Go overlay 注入，未落入生产源码树。

适用规则为当前 [AGENTS.md](/Users/yuyan9923/GitHub/workbench-claude-po/AGENTS.md) 和五份工程规范。旧审查与施工计划用于定位背景，不作为当前通过证明；个人列表范围也不能自动替代完整的对象访问规则。

## 2. 已确认的问题

P1 表示应先修复再把相关能力视为可交付；P2 表示需排入后续修复批次。以下编号是本报告的问题组，不等同于扫描器命中数量。

### F01 · P1 · 两条详情路径未执行对象访问决策

- [需求详情 Service](/Users/yuyan9923/GitHub/workbench-claude-po/internal/module/po/service_detail.go:29) 接收 `actor` 却没有使用，直接按 ID 读取正文、父子关系、历史及执行信息；[Repo](/Users/yuyan9923/GitHub/workbench-claude-po/internal/module/po/repo_detail.go:114) 仅限制 ID 和未删除。
- [已办详情 Service](/Users/yuyan9923/GitHub/workbench-claude-po/internal/module/po/servicedone.go:139) 同样忽略 `actor`；[详情查询](/Users/yuyan9923/GitHub/workbench-claude-po/internal/module/po/repodone_enrich.go:310) 按动作 ID 读取，并扩展对象及其他操作者时间线。对照已办列表的 `a.actor = req.Account`，详情没有继承列表范围，也没有另一项授权判断。
- 路由具有 `RequirePerm`，因此不能称为“匿名 HTTP 接口”。缺口在能力权限之后的对象访问，两个旧路径别名也需要覆盖。
- **隔离探针结果**：需求详情接受 nil actor；已办详情中，合成账号 `audit_requester` 可以读取 `audit_other` 的动作，期间没有授权查询。未进行真实跨用户访问。
- **修复**：先冻结需求/动作/关联父子对象/邻近时间线的可见矩阵，再由 Service 决策、Repo 执行受约束查询。需求详情不能简单等同于首页当前待办范围，否则会误伤已办、历史及合法跟进。若某资源确实是能力范围内全可见，必须有有效业务依据，不能凭参数未使用认定豁免。
- **验收**：无身份、合法账号、无关系账号、已撤销关系、父可见子不可见、猜测 ID、旧别名路径；拒绝时不返回正文或时间线。明确区分 403/404 与存储故障。

### F02 · P1 · 需求富文本存在未净化的 HTML 注入链路

- [渲染入口](/Users/yuyan9923/GitHub/workbench-claude-po/web/static/js/po/demand-detail-render.js:230) 将 `specHtml` 和 `verifyHtml` 原样拼接；[详情控制器](/Users/yuyan9923/GitHub/workbench-claude-po/web/static/js/po/demand-detail.js:91) 通过 `innerHTML` 写入 DOM。值来自数据库的 `desc`、`verifyPlan`，中间未见可信净化边界。
- [CSP](/Users/yuyan9923/GitHub/workbench-claude-po/internal/middleware/secureheaders.go:17) 允许 inline script，不能依赖它阻止事件属性。现有 `esc` 测试仅证明转义函数自身可用，没有覆盖这两个旁路字段。
- **隔离探针结果**：传入带 `onerror` 的合成 HTML 后，渲染输出完整保留事件属性。确认了不安全输出链路；没有在用户浏览器执行载荷，没有证明现存数据含恶意内容。
- **修复**：使用经过审查的成熟允许列表净化组件；定义可保留标签、属性、URL 协议。不要复制局部正则黑名单或以“数据来自禅道”为可信理由。保留合法富文本展示需求。
- **验收**：事件属性、危险 URL、SVG/畸形 HTML 等不能执行；正常列表、表格、换行和合规链接仍可展示；同时覆盖抽屉和独立页。

### F03 · P1 · 演示性质的数据被作为实际质量和耗时返回

- [代码质量树](/Users/yuyan9923/GitHub/workbench-claude-po/internal/module/po/service_detail_exec.go:200) 根据 ID 拼出分支和 MR，固定 `pass`、92.5 分、82% 覆盖率、扫描时间；应用门禁固定通过。页面实际消费并展示这些字段。
- [价值流耗时](/Users/yuyan9923/GitHub/workbench-claude-po/internal/module/po/service_detail_tabs.go:57) 对已完成阶段统一返回“实际 2 天”，当前阶段使用需求创建时间，而非阶段进入时间。
- **隔离探针结果**：仅输入一个合成故事，无扫描结果，也会得到通过门禁和 92.5 分。
- **修复**：未接入数据源的质量字段明确返回“未接入/未知”，不能用 0 或 pass 冒充真实值；真实周期取阶段事件，缺失时明确未知。预测值如需保留，单独标记预测及依据。接入扫描系统另行立项，不为消除假值而新增平台框架。
- **验收**：没有扫描、扫描失败、扫描通过、时间缺失均有不同结果；API 和 UI 不再生成虚构 MR、分支、扫描时间或实际耗时。

### F04 · P1 · 查询失败被转为成功空数据，甚至无风险

- [需求详情组装](/Users/yuyan9923/GitHub/workbench-claude-po/internal/module/po/service_detail.go:41)、[执行信息](/Users/yuyan9923/GitHub/workbench-claude-po/internal/module/po/service_detail_exec.go:20)、[历史](/Users/yuyan9923/GitHub/workbench-claude-po/internal/module/po/service_detail_tabs.go:248) 忽略多项 Repo 错误。父子查询失败还会改变详情模式。
- 缺陷读取失败后可得到零阻塞，再被 [交付判断](/Users/yuyan9923/GitHub/workbench-claude-po/internal/module/po/service_detail_tabs.go:222) 解释为缺陷已解决。已办辅助查询与周报历史也存在吞错。
- 主详情查询失败则被 [Handler](/Users/yuyan9923/GitHub/workbench-claude-po/internal/module/po/handler_detail.go:51) 一律映射为 404，数据库异常无法与不存在区分。
- **隔离探针结果**：主记录成功、子需求及后续分区 SQL 返回错误时，最终仍返回 `Success=true`。
- **修复**：核心查询错误向上传递；允许局部降级的分区返回明确 unavailable/error 状态。风险、质量、完成率不以缺省零代表正常；复用现有错误码及统一返回协议。
- **验收**：逐项注入 DB 错误，确认不会出现“暂无记录/无风险/全部通过”；保留可诊断日志并避免暴露 SQL 参数。

### F05 · P2 · 详情阶段映射与展示阶段集合不一致

- [映射函数](/Users/yuyan9923/GitHub/workbench-claude-po/internal/module/po/service_detail_tabs.go:299) 为 `waitdeliver/released/closed` 返回 `release/feedback/closed`，但 [阶段集合](/Users/yuyan9923/GitHub/workbench-claude-po/internal/module/po/service_detail_tabs.go:22) 没有这些键；匹配失败默认选澄清。
- **隔离探针结果**：上述三个状态均被突出显示为 `clarify`。这不是未确认的产品取舍，而是两个内部集合不能匹配。
- **修复**：使用一个经业务确认的阶段定义与映射；终态单独处理；未知状态展示未知，不能退回澄清。
- **验收**：覆盖全部已确认状态、结束状态及未知值，同时检查标题、阶段条和主操作入口一致。

### F06 · P2 · 首页筛选和候选分页仍违反完整集合查询要求

- [前端筛选](/Users/yuyan9923/GitHub/workbench-claude-po/web/static/js/po/home.js:227) 仅对当前页 `rawItems` 做关键词/对象/优先级过滤。若命中项在下一页，当前页显示空，还会隐藏分页，但 `total` 仍是未筛选总数。
- [全部列表](/Users/yuyan9923/GitHub/workbench-claude-po/internal/module/po/servicefollow.go:112) 逐阶段读取全部 ID，Go 去重后切片；[ID 查询](/Users/yuyan9923/GitHub/workbench-claude-po/internal/module/po/repovaluestream.go:211) 无界 Pluck。首页阶段 Count 后又读取同批候选 ID 做全量去重。
- 新增焦点查询确实已在 SQL 分页，不能把它误报为旧路径；但 `focus=all`、全部阶段和部分混合列表仍保留旧实现。
- **修复**：在 PO Repo 内形成参数化候选查询，按 `kind:id` 去重，明确阶段优先顺序；筛选、Count、排序、分页使用相同集合，再批量补当前页详情。先复核有效业务口径，不盲目把不同页面的范围合并。
- **验收**：匹配项只在后续页、跨对象、阶段交叉去重、稳定排序、无结果；1 千/1 万合成数据验证返回、扫描和内存规模，记录 EXPLAIN。不能用“只取 ID”或提高缓存掩盖无界候选。

### F07 · P2 · 周报先截取再筛选，详情复用大列表

- [Repo](/Users/yuyan9923/GitHub/workbench-claude-po/internal/module/po/repo_weekly.go:111) 先取最新 200/500 个项目；[Service](/Users/yuyan9923/GitHub/workbench-claude-po/internal/module/po/service_weekly.go:145) 随后才按关键词/状态过滤，并将结果长度作为 total。
- [详情](/Users/yuyan9923/GitHub/workbench-claude-po/internal/module/po/service_weekly.go:161) 加载最多 500 个项目的整套周报后再找 ID。截取范围以外的合法项目会被当作不存在。
- **修复**：筛选与总数移到 SQL，明确统计卡片是否跟随筛选；详情按项目 ID 直接读取并校验可见性，历史同理。500 的截断不是已证明的数据集上界。
- **验收**：至少 501 个项目，末尾项目可被关键词命中并打开详情；页内、筛选总数和统计口径有明确测试。

### F08 · P2 · 三处存在随对象数量增长的逐项查询

- [周报上线风险](/Users/yuyan9923/GitHub/workbench-claude-po/internal/module/po/repo_weekly_enrich.go:211)：每个项目执行一次 release 查询，且使用 `FIND_IN_SET`。返回 500 个项目时，这一段本身就会执行 500 次额外查询，尚未计其它统计。
- [父需求聚合](/Users/yuyan9923/GitHub/workbench-claude-po/internal/module/po/service_detail.go:175)：逐子需求读 stories，再分别读缺陷/任务计数；有故事时每个子需求最多增加三次查询。
- [排期故事组装](/Users/yuyan9923/GitHub/workbench-claude-po/internal/module/schedule/service_scheduling.go:43)：逐故事取任务；同文件另有逐产品取项目、逐项目取执行。
- **修复**：按本页/本对象的授权 ID 批量查询和分组聚合。保留禅道 CSV 关系的真实语义；先验证服务器版本和执行计划，不擅自迁移禅道 schema。批量化后仍须评估扫描量，不能只减少 SQL 次数。
- **验收**：分别用 1、10、100 个父子/项目/故事记录查询次数，确认不会逐对象线性增长；核对聚合结果、去重及缺陷阻塞含义。

### F09 · P2 · 新分层违规逃过了文本门禁

- [PO Service 构造器](/Users/yuyan9923/GitHub/workbench-claude-po/internal/module/po/service.go:54) 直接拿 `repo.db` 创建详情 Repo，将 GORM 连接暴露到 Service 组装过程。
- [已办详情 Repo](/Users/yuyan9923/GitHub/workbench-claude-po/internal/module/po/repodone_enrich.go:317) 还承担动作中文、结果文案、对象 URL、`CanOpenObject` 和页面 DTO 组装，违反 Repo 不做业务/UI 决策的边界。
- 当前扫描器只命中部分 `.db.Method` 和显式 `*gorm.DB`，所以仍只报告 dept 一项既有债务。
- **修复**：在已有依赖装配位置创建 Repo/Service 并注入；Repo 返回数据库投影，Service 解释动作/可见性/展示数据。复用当前能力，不新增全局 manager、provider 或事务框架。
- **验收**：Service 不接触连接句柄；Repo 不决定可见性或 UI 动作；新增覆盖实际漏检形态的窄检查，避免把所有 SQL 状态过滤都误判为业务分层违规。

### F10 · P2 · 页面绕开共享能力，错误处理再次分叉

- [关注操作](/Users/yuyan9923/GitHub/workbench-claude-po/web/static/js/po/follow.js:356) 自写 fetch/CSRF，只在 `res.ok` 分支处理；403/500 响应不会抛异常，也没有 else，因此用户可能看不到失败提示。
- 关注列表/周报加载重新维护请求、空态和分页；源码未见这两条加载路径的旧响应隔离，快速切换条件时存在旧响应覆盖风险，需浏览器复验。已办列表已保留序号保护，不能一概而论。
- 已有 [PersonalList](/Users/yuyan9923/GitHub/workbench-claude-po/web/static/js/po/personal-list.js:34)、appFetch、scheduleFetch 及测试，但复用不一致。扫描器把 `controller.fetch`、测试桩也记为 advisory，不能把 64 条提示全算违规。
- **修复**：复用相符的已有传输和状态能力，统一非 2xx、非法 JSON、会话过期、重复点击及旧响应处理；保留页面特有布局，不为统一外观创建通用页面框架。
- **验收**：关注写入 403/409/500 可见失败并可重试；快速切换筛选与慢响应不显示旧结果；401 处理与 403 区分。

### F11 · P2 · 详情页面颜色没有遵守已建立的主题合同

- [详情 CSS](/Users/yuyan9923/GitHub/workbench-claude-po/web/static/css/po/demand-detail.css:7) 定义独立固定色板，面板和头部硬编码白底；[渲染器](/Users/yuyan9923/GitHub/workbench-claude-po/web/static/js/po/demand-detail-render.js:230) 仍大量输出内联固定色。
- 现有语义变量已经覆盖表面、文字、边框等含义；这些普通视觉颜色不是 Logo 或必须固定的图表业务色例外。
- **修复**：仅将本次新增/实质修改的详情视觉状态映射到现有语义 token；删除对应的重复固定色，不扩展到无关历史页面。无需新增 dark.css 或复制整套选择器。
- **验收**：认证浏览器检查 light/dark、抽屉/独立页、loading/empty/error、focus/disabled 和实际加载资源。当前静态 token 测试通过不代表这些页面已经验收。

### F12 · P2 · 测试入口未包含新增行为，目录名称夸大了覆盖类型

- [Makefile](/Users/yuyan9923/GitHub/workbench-claude-po/Makefile:18) 仅列四个前端测试，未包含当前存在的 home-focus 和 notice-filters；这两个本次单独运行均通过，但今后 `make check` 不会自动覆盖它们。
- `theme-tokens.spec.js` 在 E2E 目录内，实际只用 Node 读 CSS 并断言字符串；当前目录未见驱动浏览器的用户旅程测试。历史截图不能证明当前 WIP。
- 需求详情测试主要覆盖解析、映射和 HTML 片段，未保护权限拒绝、查询失败、阶段终态或无质量来源；隔离探针已证明这些问题能与现有绿灯共存。
- **修复**：在明确测试目录内建立稳定的测试发现/清单校验，把源码契约测试归到正确类别；补本报告的行为回归，并将真实 E2E 与隔离 MySQL 作为各相关批次的交付门禁。不要用弱断言或覆盖率数字替代关键行为。
- **验收**：故意破坏新增筛选或错误处理会使统一门禁失败；真实 E2E 必须有用户交互及 Console/network/资源版本证据。

## 3. 维护性、历史债务与待确认项

本次统计仓库登记及未忽略未跟踪的 Go/JS/CSS/HTML，共 347 个文件（含测试，排除 vendor）；87 个超过 300 行，18 个超过 500 行。300 是 SHOULD，不把 87 个都算硬违规；18 个超长文件处于精确非增长基线内，本次基线变更仅降低两项，没有发现抬高额度。

最大的热点：排期模板 1176 行、排期 form 1126 行、排期集成 CSS 879 行、排期集成 JS 842 行、ui.js 775 行、PO workboard.js 700 行。PO 后端已有 45 个生产 Go 文件、14 个测试文件，文件多本身不等于乱。更实质的问题是跨层组装、重复业务映射、错误吞没和共享能力旁路。

保留为后续债务：dept Service 直接操作数据库事务；user 新密码仍采用 MD5；超长前端/模板。密码兼容不能直接换算法，否则可能破坏禅道登录，应单独设计 Workbench-owned 写入和 legacy verification 边界。它们不是本次 WIP 新引入的问题，也不能因处于基线而称为符合规范。

尚需确认，不能据此直接编码：

- 详情和周报 `scope=all` 的能力范围、关系可见性、是否允许读取其他人的已办上下文。个人页面计划仍有待决业务项；当前审查没有替产品冻结答案。
- 周报历史接口把当前风险/问题/发布偏差填入所有历史周。若页面要表达当周快照则数据不准确；需明确是“当前上下文”还是“历史快照”，缺少快照数据时不得合成历史。
- 读写连接在代码中已拆分，通知/关注写库的 mock 测试存在；实际数据库账号最小权限未由本次验证。VM 执行文档仍记载 GRANT 未完成及过渡方案，不得把代码拆分当作部署账号隔离完成。
- 真实禅道 DDL/索引、通知幂等唯一约束、新周报表/CSV 关系与线上基数需要在获准环境补证。

## 4. 分批修复计划

执行顺序：**先数据安全和真实性，再集合正确性及性能，然后边界与复用，最后逐步削减历史债务。** 每批保持独立 diff，不混入其它工作区改动；是否提交/推送继续以明确授权为准。

| 批次 | 覆盖 | 最小实施范围 | 进入条件与验收 |
|---|---|---|---|
| A：详情安全 | F01、F02 | 两个详情 Service/Repo/Handler、富文本输出边界、对应测试 | 先确认对象可见矩阵；跨账号与父子/时间线用例、净化攻击用例、认证浏览器验证。两个历史别名同测。 |
| B：详情真值 | F03、F04、F05 | 质量/阶段/耗时/交付 DTO 与渲染，错误映射 | 无来源显示未知；DB 故障不出成功零值；终态不回澄清；不顺带建设扫描平台。 |
| C：首页集合 | F06 | 请求模型、候选 SQL、阶段/KPI 口径、筛选与分页 | 冻结范围与期限真值；隔离 MySQL 跨页等价、去重排序、查询计划；所有筛选作用完整集合。 |
| D：周报与聚合性能 | F07、F08 | 周报 SQL/独立详情、父子批量统计、排期批量读取 | 至少 501 项验证；查询次数与扫描量证据；对真实 schema 无侵入。三个查询热点可分别交付。 |
| E：工程收口 | F09、F10、F11 | 依赖装配、Repo 投影、既有前端组件复用、详情主题 | 删除本批替代后的孤儿代码；无跨层句柄/虚构 CanOpen；失败提示与慢请求、两主题真实浏览器验收。 |
| 持续门禁 | F12，贯穿 A—E | Makefile、分类准确的测试及 CI 入口 | 每批先补有效回归再修复；新测试被统一入口执行；不抬高基线换取绿灯。 |
| 后续历史债务 | dept/user/超长文件 | 按真实能力逐个缩小或合并重复实现 | 单独限定行为边界及兼容条件；改前后回归；禁止按行数机械拆分或全项目重命名。 |

回退策略：每批保留原有确认过的业务契约与独立差异；安全修复的替代措施应收敛入口而非恢复无授权/不净化行为。审查阶段不执行任何数据库迁移或部署操作。

每批通用门禁：`git diff --check`、`make check`、相关 Go/JS 行为测试；涉及 SQL 必须补隔离 MySQL 数据等价和查询计划；涉及 UI 必须补真实登录交互、Console/network、实际资源版本和必要的 light/dark 视觉检查。未执行或失败必须明确标记，不能凭 unit 通过称可交付。

## 5. 本次验证结果

| 验证 | 结果与边界 |
|---|---|
| `git diff --check` | PASS |
| `make check` | PASS；18 项超长文件、1 项既有架构债务；64 项 advisory，其中 34 项不在库存中。advisory 不是确认违规数。 |
| `go test -count=1 ./...` | PASS；无缓存重跑。含 tests/integration 包里的普通安全单测，不代表真实 MySQL 集成执行。 |
| home-focus、notice-filters | 单独执行 PASS；尚未进入 Makefile 统一清单。 |
| PO/schedule JS 语法 | 24 个文件 PASS。 |
| theme token 静态契约 | PASS；未启动浏览器。 |
| Go overlay 三个探针 | 成功复现权限缺口、吞错、固定质量值和终态映射错误；探针 PASS 表示坏行为被复现，不是功能正确。 |
| Node 富文本探针 | 确认事件属性原样输出；未向真实页面插入载荷。 |
| 真实 MySQL | NOT RUN；本次环境未配置隔离测试 DSN，不使用业务库替代。 |
| 认证浏览器 / E2E / 当前远端 CI | NOT RUN；本次不作运行环境验收结论。 |

复现命令与日志说明见 [EVIDENCE.md](/Users/yuyan9923/GitHub/workbench-claude-po/docs/plan/architecture-review-20260906/EVIDENCE.md)。本报告为当前工作树的定点审查，不替代后续每批修复后的重新验收。
