# Codex 交接：研发工作台 V2 三个已确认待修缺陷 + 一个待拍板项（2026-10-08）

请完整阅读本文件后开工。**不要向用户提问**，按第 3 节的优先级顺序执行。

---

## 1. 环境与基线

- 仓库：`~/GitHub/Dev-CT`，分支 **`Dev-CT`**，开工前 `git branch --show-current` 核对。
- 基线 HEAD：**`ac889938`**（`fix(schedule): drop the separate change-window button`）。
- 规范：`AGENTS.md`（唯一主规则，动手前完整读一遍）、`.cursor/rules/conventions.mdc`、`.agents/rules/frontend.md`。
- **不要碰禅道源码** `~/GitHub/csrcb20-gitfox`（只读参考可以）。
- 演示服务在 **8099**（`zentaopms_migrated` 库），当前进程 `/private/tmp/devct-demo-20261008`，是 `ac889938` 编译产物。**不要重启它**，你自己另起端口验证（建议 8100）。

### 开工前必须知道的两个前置事实

1. 工作区有一个**与本任务无关的未提交改动**：`web/static/css/auth/login.css` 第 201 行 `clip-path` 被改成 `inset(3% 6% 3% 3% round 28%)`。**这不是你改的，不要提交它，也不要还原它。** 提交时逐文件 `git add`，不要用 `git add -A` / `git add .`。
2. 本轮**只允许改下面点名的文件**（AGENTS.md 前端规则 h）。其余文件即使发现问题也只写进报告，不动手。

---

## 2. 硬性红线

- 不 push、不合并 main（AGENTS.md 协作纪律）。
- 数据库**只允许 SELECT**，任何写库操作先停下报告。
- 不碰 8090 / 8095 / 8080 / 9000 / 8098。
- 不改禅道表结构、不改禅道源码、不新增 `zt_*` 表或字段。
- 禁止 `git reset --hard` / `force push` / `git checkout <sha> --`（不带路径）。
- 禁止新增 `!important`、禁止新增内联 `style=`（含 JS 拼接的 style 字符串）、禁止字面颜色（十六进制 / rgb / white）。
- 禁止新增 `window.*` 全局函数；禁止引入只用一次的抽象层。
- 已超 500 行的文件，本轮**净增行数必须 ≤ 0**。若某处修复确实无法避免变长，**停下来写进报告问用户**，不要自己决定。
- 截图、走查脚本、临时文件放仓库外（如 `/tmp/v2bugfix/`），**不提交进 git**。

---

## 3. 待修缺陷（按优先级，全部已定位根因）

### 【P0】缺陷 1：产品经理显示名称搜索不到

**现象**：角色管理列表显示的角色名是「产品经理」，但搜索框输入「产品经理」搜不到，输入「产品负责人」反而能搜到。

**根因（已确认，不要重新猜）**：`workbenchroles.DisplayLabel()` 只在**查询返回之后**在内存里做名称映射，数据库里的真实值仍是「产品负责人」。

```
internal/module/role/repo.go:44      db = db.Where("name LIKE ?", "%产品经理%")   ← 库里没这个名字，永远 0 命中
internal/module/role/service.go:49   items[i].Name = workbenchroles.DisplayLabel(...)  ← 查完才转换
```

**核对更正**：用户列表和个人资料中的角色名是显示映射，未发现同类角色名搜索入口，本轮不修改这两个模块。

**修法要求**：
- 关键词要同时匹配**原始值和显示值**。「产品经理」「产品负责人」「PO」三种输入都必须能命中 `code = po` 的那条记录。
- 保持分页 `total` 正确（`total` 与 `items` 必须用同一个条件，否则翻页会出现空页）。
- **不要改数据库里的值**，也不要把「产品负责人」直接改名写库——这是禅道侧历史数据，只在查询层做别名匹配。
- 别名清单不要散落在 Repo 里硬编码字符串数组；`internal/pkg/workbenchroles/display.go` 已经是这个映射的唯一权威，让它同时导出「code → 所有可搜索别名」，Repo 用它拼 OR 条件。
- 补单测：输入三个别名各自都能命中；`total` 与 `items` 条数一致；关键字为空时行为不变。

---

### 【P0】缺陷 2：新增角色页控件样式散架

**现象**：`/admin/roles/new` 页面的按钮不换行、间距塌陷、错误提示不显示。`/admin/users/new` 正常。

**根因（已确认）**：`web/templates/role/create.html` 整页在用 **Bootstrap 语义类**，但本项目**没有 Bootstrap**（`web/static/vendor/` 只有 echarts / fontawesome / fonts / jquery / quill）。逐 class 实测结果：

| class | 项目 CSS 中定义文件数 |
|---|---|
| `mb-3` `mb-4` `d-flex` `gap-2` | 0 |
| `btn-outline-secondary` `invalid-feedback` | 0 |
| `border-0` `shadow-sm` `link-secondary` | 0 |
| `form-control` `form-label` `card` `btn-primary` | 有（定义在别的语境里，勉强能用） |

**正确参照物**：`web/templates/user/create.html` —— 它用的是项目自有体系（`page-header` / `table-panel` / `panel-header` / `panel-title` / `form-group` / `input` / `field-error`），所以显示正常。

**修法要求**：
- 按 `user/create.html` 的既有 class 体系重写 `role/create.html`，**不要引入 Bootstrap，不要新增 CSS**。
- 按已确认计划同步修复 `role/edit.html`；列表模板仅核对，本轮不修改。
- 页面数据 key 必须符合 AGENTS.md 第 10 条：`Form` / `Resource`（编辑页）/ `BaseUrl`。当前 `handler.go:95-106` 的 `NewForm` 没有传 `BaseUrl`，检查是否需要补。
- 颜色只能引用 `variables.css` 的 `--color-*` 或 `tokens.css` 的 `--wb-*`，禁止字面值。
- 提交前跑 AGENTS.md 第 108-138 行那段「未定义颜色变量扫描」脚本，退出码必须为 0。

---

### 【P1】缺陷 3：澄清说明在 textarea 中显示 HTML 源码

**现象**：澄清弹窗「澄清说明」输入框里，如果该需求历史上被写入过 HTML，textarea 中会直接显示 `<p>...</p>` 这类标签源码，而不是纯文本。

**根因（已确认，不要重新猜）**：

1. **禅道侧**：`zentao/extension/custom/sync/control.php:126-139` 有个专门叫「Format clarifyDesc」的清洗脚本，做的是 `strip_tags(htmlspecialchars_decode(...))`——证明该字段**历史上确实被写成过 HTML**。
2. **工作台侧**：`web/static/js/po/demand-clarify.js:114` 是 `$("#poClarifyDesc").val(data.clarifyDesc || "")`，**原样塞进 textarea**，零清洗。textarea 只显示文本，HTML 标签就以源码形态露出。

**修法要求**：
- 项目里**已有** bluemonday 白名单净化能力：`internal/module/po/html_sanitize.go` 的 `SanitizeRichTextHTML()`。但它目前只用在 `service_detail.go:165-170` 的 `SpecHtml` / `VerifyHtml` / `Desc` / `VerifyPlan`，**没覆盖 clarifyDesc**。
- 你要判断的是：clarifyDesc 在澄清弹窗里是**纯文本语义**（用户要编辑、要原样回传禅道），所以应当走「**去标签 + 反转义**」，而不是走 `SanitizeRichTextHTML` 的「保留白名单标签」。
  - 去标签：`strip_tags`
  - 实体反转义：把 `&nbsp; &amp; &lt; &gt; &quot; &#39;` 等还原
  - `<br>` / `</p>` 这类块级标签要**转成换行**，不能直接粘成一个长行
- **按用户最终确认**：读取侧转为可读文本。提交时后端与原内容转换后的文本比较：未修改说明则提交原 HTML；修改说明才提交用户编辑的纯文本。空值、纯文本含义不变，不改禅道源码。
- 后端在 `internal/module/po/service_clarify.go:141` 的 `ClarifyDesc: demand.ClarifyDesc` 出口处清洗，前端 `demand-clarify.js:114` 不动（前端做清洗属于重复实现，且 AGENTS.md 克制条款 3 禁止多余防御）。
- 如果你在排查中发现**其它**弹窗 textarea 也有同类问题（从禅道读出来直接塞 textarea 的富文本字段），写进报告，**本轮不改**。
- 补单测：`<p>abc</p><br>def` → `abc\ndef`；`&amp;&amp;` → `&&`；空串 → 空串；纯文本无标签 → 原样不变。

---

### 【P2】待拍板项：普通页面新窗口行为（**先只报告，不动手**）

**冲突事实**：`AGENTS.md:46` 第 13 条明确写「除弹窗外的所有页面均在当前页打开，禁止打开新窗口」。但实际存在大量违规：

模板中 `target="_blank"` 共 **16 处**：

| 文件 | 处数 |
|---|---|
| `web/templates/schedule/index.html` | 8 |
| `web/templates/kanban/task.html` | 2 |
| `web/templates/po/workboard.html` | 1 |
| `web/templates/po/notice.html` | 1 |
| `web/templates/po/linkstory.html` | 1 |
| `web/templates/po/issue-risk.html` | 1 |
| `web/templates/po/done.html` | 1 |
| `web/templates/kanban/story.html` | 1 |

JS 中另有：

| 文件:行 | 行为 |
|---|---|
| `web/static/js/schedule/scheduleintegratedrd.js:318` | `window.open(detailUrl, "_blank")` |
| `web/static/js/po/demand-detail-render.js:383` | 附件下载链接 `target="_blank"` |
| `web/static/js/po/demand-detail-review.js:77` | 附件下载链接 `target="_blank"` |

**你的任务**：
1. 逐处打开看，确认每一处**实际业务语义**是什么（跳禅道？跳外部系统？下载附件？本页详情？）。**不得预设外部系统或附件已获规则例外**。先分类报告，所有新窗口行为本轮保持不变。
2. 输出一张分类表：**内部页面跳转** / **外部系统跳转** / **附件下载** / **动态目标需确认**（均只报告）。
3. 记录真实页面链接目标；条件分支及没有实际数据的入口须标注来源核对，不能以截图替代点击验证。受影响的修复页面另按双主题截图验收。
4. **一行代码都不要改**，等用户拍板范围后再开工。

---

## 4. 执行纪律

1. **先读后改**：AGENTS.md 完整读一遍再动手。
2. **先验证根因再改**：上面给的行号可能因未提交改动有偏移，以你自己读到的为准。**如果你读到的代码和本文件描述不符，以代码为准，并在报告里明确指出差异**——不要为了对上提词而改错东西。
3. 逐项改，每项改完立即在 8100 自测，**不要攒到最后一起测**。
4. 验证用真数据：缺陷 2 用 `/admin/roles` 真实列表点进新建页；缺陷 3 需要一条 `clarifyDesc` 含 HTML 的需求（**只允许 SELECT 查询**，不要为了造数据去写库）。
5. 每个改动页面浅色、深色各截一张全页图，内部滚动区截到底。截图放 `/tmp/v2bugfix/`，不提交。
6. 跑 `make check` 和 `make quality`，基线必须保持通过。

## 5. 交付报告要求

逐项列出，必须包含：

1. **FIX-LIST**：每个文件改了什么、**底层根因**、改动落在哪一层（Repo / Service / Handler / 模板 / 前端）。
2. **净增行数**：每个文件的净增减（AGENTS.md 克制条款 5 硬要求）。
3. **新增函数与 CSS 选择器清单**（局部表单 JS、文本转换及测试函数属于已批准范围；本轮不新增 CSS 选择器）。
4. **VERIFY 证据**：`make check` / `make quality` 输出 + 浅色深色截图路径 + 手工验证步骤的实际结果。
5. **第 3 节 P2 的分类表与截图**，标注「未改动」。
6. **发现但本轮未改的问题清单**，每条写清为什么没改。
7. **风险与未完成项**：写操作（保存 / 删除 / 澄清 / 交付 / 催办 / 标读）在当前只读约束下**未真实提交验证**，这一点必须明写，不能用「接口 200」「测试通过」当闭环。

**不要生成 handback.md 之类的额外交付物，本文件就是唯一交接文档。**

## 6. 本轮实施结果（2026-10-08）

用户确认计划是本轮范围依据。在 Dev-CT 上完成，未推送。8099 始终保持原 ac889938 演示版本；8100 为临时验证版本，结束后关闭。login.css 原有未提交修改保留且未提交。无新增依赖、框架、全局函数、CSS 选择器，未修改路由权限。

| 问题 | 结果 / 根因与修复层 | 独立提交 |
|---|---|---|
| 角色搜索 | 原查询只匹配存储名称。现有别名工具统一兼容显示名与存储名；Repo 同一条件用于 Count/Find，支持别名及部分关键词 | a1c8ca19 |
| 新增／编辑角色 | 原模板使用缺失样式、传统 form 提交。复用 user/create.css 现有表单体系；补 BaseUrl；Handler JSON 绑定、422 字段错误与成功 redirectUrl；页面局部 JS 防重复并保留失败输入；内置角色前后端均禁止修改 | 45f2eeab |
| 澄清说明 | 原表单出口返回 HTML。Service 出口转换文本；保存时比较说明，未改保留原 HTML、改后提交纯文本；文本转换使用已有 x/net/html tokenizer | dd3cb100 |
| 新窗口 | 源码全部命中逐处分类，页面实际 href 抽查；未改动，不认定规则例外 | 无代码修改 |

### 验证

- `make check`：退出 0（全 Go 测试、vet、直写检查与 JS 回归）；日志 `/private/tmp/v2bugfix/check.log`。
- `make quality`：退出 0（golangci 0 issues、直写豁免检查、stylelint、文件大小）；日志 `/private/tmp/v2bugfix/quality.log`。首次新增测试/转换函数复杂度检查未过，已简化后复跑通过，未降低规则。
- AGENTS.md 颜色扫描：未定义变量 0、引用次数 0，退出 0；`/private/tmp/v2bugfix/color-scan.log`。
- 搜索：三个别名、经理/负责人/产品/po、空关键词、管理员与分页测试通过。模拟 Count 和 Find 同条件、页 2 每页 1 条总数 2。真实列表别名与经理均 1 条，管理员 1 条，空关键词 2 条；库只有两条内置角色。
- 表单：模拟数据库实际执行 POST 创建、PUT 更新；两者均 422 字段错误；内置角色拒绝更新。Node 模拟 DOM/响应验证 JSON、成功跳转、失败保留输入、422 原地提示、挂起请求重复点击只有一次请求和按钮恢复。真实新建页、内置产品经理编辑页加载成功，编辑控件只读、保存禁用；未创建或修改真实角色。
- 澄清：HTML、实体、编码 HTML、段落换行、空串、保留纯文本空白与比较符号、脚本隐藏测试通过。模拟禅道 HTTP 响应检查未改说明实际发送原 HTML、改说明发送纯文本、清空发送空串。
- SELECT `SELECT id, clarifyDesc FROM zt_demand WHERE id=63360`：真实原文含 p/span/style 标签。003030 在 8100 打开 US63360 澄清表单，textarea 为可读文本，无 HTML 标签。没有按保存；SELECT 证据 `/private/tmp/v2bugfix/clarify-select.txt`。
- 角色与澄清浅色／深色截图已逐张看过，无新增白块、文字不可读或布局溢出。角色页内容不产生内部滚动；需求 main 滚到底（scrollTop 111 + clientHeight 987 = scrollHeight 1098），澄清底部按钮完整可见。角色页控制台 error/warn 为空。

截图均在 `/private/tmp/v2bugfix/`，未入库：

| 页面 | 浅色 | 深色 |
|---|---|---|
| 搜索真实结果 | role-search-light.png | role-search-dark.png |
| 新增角色（原有示例值，未提交） | role-create-light.png | role-create-dark.png |
| 编辑内置角色（真实数据） | role-edit-light.png | role-edit-dark.png |
| US63360 澄清（真实数据） | clarify-light.png、clarify-bottom-light.png | clarify-dark.png、clarify-bottom-dark.png |

### 每个代码文件净增减（相对 ac889938，不含无关 login.css）

| 文件 | 新增 | 删除 | 净变化 |
|---|---:|---:|---:|
| `internal/module/po/clarify_text.go` | 58 | 0 | +58 |
| `internal/module/po/clarify_text_test.go` | 43 | 0 | +43 |
| `internal/module/po/service_clarify.go` | 2 | 2 | +0 |
| `internal/module/po/service_clarify_text_test.go` | 70 | 0 | +70 |
| `internal/module/role/form.go` | 7 | 7 | +0 |
| `internal/module/role/handler.go` | 14 | 47 | -33 |
| `internal/module/role/handler_form_test.go` | 102 | 0 | +102 |
| `internal/module/role/repo.go` | 7 | 1 | +6 |
| `internal/module/role/repo_search_test.go` | 64 | 0 | +64 |
| `internal/pkg/workbenchroles/display.go` | 24 | 1 | +23 |
| `internal/pkg/workbenchroles/display_test.go` | 15 | 0 | +15 |
| `web/static/js/role/form.js` | 55 | 0 | +55 |
| `web/static/js/role/form.test.js` | 61 | 0 | +61 |
| `web/templates/role/create.html` | 5 | 56 | -51 |
| `web/templates/role/edit.html` | 5 | 79 | -74 |
| `web/templates/role/form.html` | 44 | 0 | +44 |

测试代码 +355/-0，净 +355；非测试代码 +221/-193，净 +28。非测试为正的原因：必须增加局部 JSON 提交与失败恢复 JS（55 行）及说明文本转换/原文保留（58 行）；通过删除旧表单重复模板和回显 Handler 抵消大部分增量。所有新增文件小于 500 行，本轮没有使既有超长文件变长。交接文档本身原为未跟踪文件，本轮修正并纳入版本；不计入上述业务/测试代码统计。

新增生产函数：`SearchAliases`（别名入口）、`clarifyDescriptionText`、`clarifyDescriptionForSubmit`（文本转换与提交比较）、局部 `renderErrors` 和表单 submit 回调。已有 `DisplayLabel` 调整为复用同一别名表。新增测试函数：`TestSearchAliases`、`newRoleTestDB`、`TestRoleSearchAliasesAndPagination`、`TestRoleFormJSON`、`testRoleFormJSON`、`TestRoleFormFieldErrors`、`TestRoleBuiltinCannotUpdate`、JS `setup`/`run`、`TestClarifyDescriptionText`、`TestClarifyDescriptionForSubmit`、`TestClarifyDemandPreservesOriginalDescription`、`testClarifyDemandDescription`。CSS 选择器新增 0；仅增加页面 DOM id/错误定位属性。重复的新增／编辑模板抽到角色局部共用模板，无新架构层。

### 新窗口：完整源码分类（未改动）

排除注释和 html-sanitize 中“保留 target 并补 rel”的判断后，53 处实际创建新窗口入口：模板 16、JS 37（其中 window.open 2）。下面逐处行号覆盖这 53 处；行号以本轮代码为准。多数目标由禅道 URL 工具生成，但不因此认定已获得规则例外。两个通用绝对 URL 分支须按具体 href 区分站内或外部。

| 文件及行号 | 用途 / 分类 | 依据与限制 |
|---|---|---|
| `web/static/js/po/demand-detail-render-execution.js`：130 | 执行任务禅道详情 | t.ztUrl |
| `web/static/js/po/primary-action.js`：32、44 | 32 外部主操作；44 动态绝对 URL | pa.kind=external / https 判定，绝对 URL 可能同源，不能概括为全部外部 |
| `web/static/js/po/follow-demand.js`：161 | 关注业需禅道详情 | ztUrl |
| `web/static/js/schedule/scheduleintegratedrd.js`：318 | 确认后打开禅道业务需求详情 | currentDemandDetailURL 来自排期 DetailURL；未触发确认写流程 |
| `web/static/js/po/workboard-demand.js`：126 | 独立研需禅道详情 | root.url |
| `web/static/js/po/demand-detail-review.js`：77、122、127 | 77 附件；122/127 禅道编辑业需 | f.download / zentaoEditUrl |
| `web/static/js/layout/globalsearch.js`：140 | 编号搜索跳禅道 | 全局对象 URL 构造 |
| `web/static/js/po/issue-risk.js`：174 | 问题/风险禅道详情 | 对象 URL |
| `web/static/js/po/demand-detail-flow.js`：36 | 流程相关对象禅道详情 | 禅道流程对象 URL |
| `web/static/js/po/demand-detail-render.js`：90、191、383、402 | 90 禅道业需编号；191 动态绝对动作 URL；383 附件；402 禅道澄清兜底 | 191 仅检测 http(s)，可能同源；383 /file/download/:id |
| `web/static/js/po/follow-drawer.js`：68 | 禅道项目周报 | item.url |
| `web/static/js/po/workboard.js`：201、408 | 任务编号/完整详情跳禅道 | task.url / t.url |
| `web/static/js/po/home-render.js`：121、152 | 首页对象编号/研需标题跳禅道 | 对象 URL |
| `web/static/js/po/notice.js`：139、165、199、215 | 通知关联对象、编号、标题及去处理跳禅道 | 对象 URL，动态通知条目分支 |
| `web/static/js/query/query.js`：78、93、111 | 研需编号/标题/详情跳禅道 | zentaoUrl；业需链接当前页 |
| `web/static/js/kanban/story.js`：167 | 看板业需标题跳禅道 | item.zentaoUrl |
| `web/static/js/kanban/issue.js`：121 | 看板问题详情跳禅道 | 问题 URL |
| `web/static/js/po/done.js`：242、259 | 已办对象标题/编号跳禅道 | it.url |
| `web/static/js/kanban/task.js`：97、109 | 任务编号/标题跳禅道 | 任务 URL |
| `web/static/js/po/todos.js`：188、189、207 | 待办编号/标题/查看跳禅道 | todo_query_repo 的对象 URL |
| `web/templates/schedule/index.html`：292、305、319、332、346、376、439、476 | 业需/研需标题与动作：禅道详情，bizdemand_view.go 构造 DetailURL | 模板及 URL 生产端核对 |
| `web/templates/kanban/task.html`：41、42 | 禅道创建任务/问题，Handler 构造 URL | 模板及 URL 生产端核对 |
| `web/templates/kanban/story.html`：42 | 禅道创建问题 | 模板及 URL 生产端核对 |
| `web/templates/po/workboard.html`：153 | 研需抽屉：禅道研需详情 | 模板及 URL 生产端核对 |
| `web/templates/po/linkstory.html`：111 | 关联研需：禅道标题详情 | 模板及 URL 生产端核对 |
| `web/templates/po/notice.html`：80 | 通知抽屉：禅道对象详情 | 模板及 URL 生产端核对 |
| `web/templates/po/done.html`：110 | 已办抽屉：禅道对象详情 | 模板及 URL 生产端核对 |
| `web/templates/po/issue-risk.html`：49 | 问题风险抽屉：禅道对象详情 | 模板及 URL 生产端核对 |

实际 DOM href 证据 `/private/tmp/v2bugfix/new-window-dom.json`：排期、工作看板和问题风险显示的链接均指向 localhost:8080 禅道；US63360 编号亦为 demand-view-63360。未点击这些链接或创建按钮，不操作受保护 8080 服务。首页/待办/已办/通知等异步加载分支以源码与 URL 生产端核对为依据，不以导航瞬间的空 DOM 认定“无链接”。缺少真实附件样本及条件动作，未逐处真实打开；这是链接用途分类，不是每个新窗口行为的端到端通过报告。

附件两处的显示用途是下载，但 Service 输出 `/file/download/:id`；全仓未找到相应工作台下载路由。故目前不能声称附件下载成功或属于外部禅道链接。保留待确认/待另轮修复，不在本轮擅改。

### 未真实执行及范围外发现

- 没有真实创建／保存／删除角色，也没有真实保存澄清、交付、催办、标读；写流程只走模拟数据库/禅道响应。不能据此宣称共享库写入端到端安全。
- 验收库无自定义角色，非内置编辑成功路径用模拟数据库验证；真实编辑页只核对内置禁改。
- 角色列表既有 mb-3、d-flex、gap-2 等缺失样式类仍存在；列表不属于批准修改模板范围。新增/编辑页已清除这些类。
- 角色删除和分配权限仍是旧提交/跳转流程；本轮只修新增/编辑，未扩大范围。
- systemClarifyDesc 等其他说明字段仍原样进入输入框，未确认含 HTML 的真实异常样本；按范围限制未改。
- 澄清界面真实历史数据有用户故事字段空值/部分窄列截断，原有页面行为；本轮修改范围仅说明转换，未修改弹窗布局或其他数据。
- 所有新窗口入口未修改；外部系统、附件的规则例外仍未拍板。

质量自查：范围核对完成；无新增单次接口/工厂抽象；函数与文件门禁通过；新增函数及净变化已列；无新增 CSS/依赖；截图与临时脚本未提交。


## 7. 用户追加：公共导航计数长期横线（2026-10-08）

根因：SidebarBadges 三次串行 SELECT 共用 250ms 超时，任一超时把所有计数标成 unavailable。模板显示横线，没有 JS 补取，等待多久也不会自动恢复。真实补取耗时 1.465s，证明原 250ms 预算无法覆盖该样本。

修复：SSR 保留 250ms 上限，页面快速打开；新增权限保护 GET /navigation/badges，独立 10s 上限异步补取。请求期间显示省略号，完成后显示真实数字（包括 0）；侧栏通知与顶部铃铛同时更新。失败显示“重试”，点击仅补取计数；防重复请求。不增加轮询、缓存、第三方依赖或全局函数，不写库。

公共文件变更理由：用户截图明确点名侧栏和全局顶部铃铛；sidebar/header 添加稳定计数定位和加载标记，base 仅加载局部职责的 badges.js。bootstrap 将 250ms 页面预算留在 SSR provider，避免异步接口沿用相同超时。未修改主题逻辑、配色、页面业务和列表查询口径。

验证：003030 在 8100 问题风险页首次出现加载状态，随后我的待办 73、侧栏未读 0、铃铛 0，三处均为真实查询结果。接口日志 200/1.465024292s。浅色、深色截图逐张检查通过，控制台 error/warn 为空；内容未产生纵向滚动。新 Node 测试覆盖挂起请求加载状态、失败、点击重试与真实零值；make check、make quality、AGENTS 颜色扫描退出 0。日志 badges-check.log、badges-quality.log、badges-colors.log 在 /private/tmp/v2bugfix。截图 badges-light.png、badges-dark.png 同目录，未提交。未真实执行任何业务写操作。

| 文件 | 净变化 |
|---|---:|
| internal/bootstrap/bootstrap.go | +4 |
| internal/module/po/handler.go | +1 |
| internal/module/po/handler_badges.go | +22 |
| internal/module/po/sidebar_badges.go | 0 |
| web/static/js/layout/badges.js | +43 |
| web/static/js/layout/badges.test.js | +32 |
| web/templates/layout/base.html | +1 |
| web/templates/layout/header.html | 0 |
| web/templates/layout/sidebar.html | 0 |

测试净 +32；非测试净 +71，为新增异步计数接口和恢复机制所需。新增函数 NavigationBadges、局部 load、测试 run 及事件回调；新增 CSS 选择器 0，无超长文件净增长。login.css 原有改动保留未提交。8100 结束后关闭，8099 原演示服务不重启且未更新本轮代码。质量自查各项通过。


## 8. 用户追加：登录 logo 白边与深色配色

logo 源图 csrcb-icon.png 自带白色边缘，修正已有 login.css 裁切为 inset(4% 8% 4% 4% round 28%)，图案和源图片不改。本次用户明确点名登录页，原未提交 logo 裁切修改在此次同范围调整并一并提交。深色仅修改登录专用 --wb-auth-* token：背景由近黑改蓝灰，Hero/按钮降低蓝色饱和度；链接和焦点保留可识别对比度。新增 --wb-auth-button-end 在浅色、深色两块均定义，区分链接颜色与按钮深色终点，确保白字可读。其他页面 token 不改，浅色配色不变。

8100 登录页浅色／深色全页截图均已逐张检查，logo 无白边、文字清晰、无纵向内部滚动；登录页不需要业务数据。截图 /private/tmp/v2bugfix/login-new-light.png、login-new-dark.png，未提交。make check、make quality 和颜色扫描均退出 0（未定义变量 0）。无新增测试、函数、CSS 选择器或依赖。login.css +2/-2 净 0；tokens.css +14/-12 净 +2（浅深两处按钮终点定义）；本节文档净 +7 行。仅样式调整，未提交登录或写库。8100 用完关闭；8099 不重启，仍为原演示样式。质量自查各项完成。


## 9. 首页说明浮层被裁切

13eeda9：home.css 中 .po-home .top5-section 的 overflow:hidden 裁掉向上展开的说明，改为 visible。仅 +1/-1 净 0 行，不新增函数、CSS 选择器、颜色或行为。真实账号 003030 的浅深全页及内部底部截图逐张核对完成，提示完整，列表与分页布局正常；截图 /private/tmp/v2bugfix/tooltip-{light,dark}.png、tooltip-bottom-{light,dark}.png 不提交。make check、make quality、颜色扫描通过，无真实业务写操作。更新 8099 样式，用完关闭 8100；保留上一版部署用于回滚。


## 10. 用户确认：登录页固定浅色

本节取代第 8 节的登录深色方案。删除 tokens.css 中仅供登录页使用的深色 --wb-auth-* 覆盖，沿用浅色表单与蓝色强调；login.css 的 body 指定 color-scheme:light，保证原生复选框也不跟随深色，并将 SSO hover 改为登录专用背景变量。logo 白边裁切保留。公共 token 文件修改理由：登录配色唯一来源在该文件，仅删除登录专用覆盖，不改内部页面 token、主题脚本或存储偏好。

8100 两种主题偏好的全页截图逐张检查通过，无白边、截断和内部滚动：/private/tmp/v2bugfix/login-fixed-light-{dark,light}-preference.png。深色偏好下登录背景 rgb(248,250,252)，按钮恢复蓝色；真实账号 003030 登录后主题及 color-scheme 仍为 dark，控制台无 error/warn。登录页无业务数据；未执行业务写操作。make check、make quality、颜色扫描均通过，未定义变量 0；日志 login-light-{check,quality}.log 同目录。

文件净变化：login.css +2/-1 净 +1；tokens.css +1/-69 净 -68；本交接文档净 +9。测试代码净 0，非测试代码净 -67。新增函数、CSS 选择器、变量、依赖均为 0。质量自查各项完成。更新 8099，旧版二进制及运行目录保留用于回滚，8100 用完关闭；不推送。


## 11. 用户确认：只统一业务需求标题

业务需求标题统一打开既有 DemandDetail 抽屉：待办按 item.kind=demand 构造 /demands/:id 并删除失效的 ID DOM 拦截；排期的业务及子业务标题改同一路径；主看板父需求标题可查看，点击不折叠，箭头仍负责折叠；父详情内子需求标题复用既有 open；版本跟进标题补入口；通知仅按真实 objectType=demand、有效 objectId 打开需求详情，原标读行为保留。研发及其他对象链接原样保留，编号链接和操作按钮不改。三个原未加载详情的页面复用组件模板中的现有 CSS/JS 资源，不改全局模板、主题或路由权限。旧 /kanban/story 模板已 location.replace('/board/demand')，上一轮静态盘点忽略该跳转；无需修改其休眠脚本。

8100 真实账号 003030 验证：待办 US2511、排期 US63425、主看板父需求 US63420 和子需求 US63422 均打开同一抽屉，列表 URL 不变；看板标题不切折叠状态，箭头仍可展开；排期研需 72353 的禅道链接及 target 保留。浅深全页及详情内部底部截图逐张检查，无新增白块或溢出，控制台 error/warn 为空。截图 /private/tmp/v2bugfix/titles-{todos,schedule,board,notice,version}-{light,dark}.png，带内部滚动的底部截图同目录，未提交。版本跟进当前窗口无关联需求，不能声称真实标题点击通过；已读业务通知唯一样本 US10 详情返回“需求不存在”，不擅改历史数据。四个 Node 回归覆盖待办多对象区分、业务/研发通知及无对象消息、版本标题与展开按钮、父详情子标题；全部通过。未执行真实业务写操作，通知仅点已读记录。

工作区 make check 通过；make quality 被既有未提交 repo_story_deliver.go 与 repo_deliver.go 的 dupl 拦住，该交付补丁不属本轮，未修改。使用当前 HEAD c6a13389 的独立 Git 副本，只加入本轮文件，保留原质量基线，make check 与 make quality 均通过；日志 titles-{check,quality}.log、titles-clean-{check,quality}.log 同目录。AGENTS 颜色扫描未定义变量 0，git diff --check 通过。新增生产函数、CSS 选择器、颜色变量、依赖均 0；新增模板定义 po/detail_css、po/detail_js、po/detail_clarify，均在本轮页面使用；测试辅助 render、escapeHtml 及测试回调。超长 schedule 模板净减 1 行。质量自查完成。

| 文件 | 本轮净变化 |
|---|---:|
| web/static/js/po/todos.js | -21 |
| web/static/js/po/notice.js | +1 |
| web/static/js/po/version-follow.js | +0 |
| web/static/js/po/workboard-demand.js | +0 |
| web/static/js/po/demand-detail-parent.js | +0 |
| web/templates/po/notice.html | +0 |
| web/templates/po/version_follow.html | +0 |
| web/templates/schedule/index.html | -1 |
| web/templates/components/po_detail_assets.html | +28 |
| web/static/js/po/demand-title.test.js | +57 |
| 本交接文档（第 11 节） | +25 |

测试净 +57；非测试代码净 +7，增长来自三个页面当轮复用的既有详情资源加载模板（28 行），没有新增业务层。当前工作树已收口为 Dev-CT，用户已授权提交；本次仅提交已验证的标题修复及本节记录，独立研需交付 WIP 原样保留，不推送。8099 保持原服务运行，不覆盖其部署；8100 用完关闭。


## 12. 用户授权：独立研需交付补丁收口与重复代码治理

本节收口此前保留的 13 个文件，不推送、不合并、不改禅道源码，不新增直写或真实业务写操作。复用既有交付弹窗，按对象类型分别请求 /demands/:id/deliver 与 /stories/:id/deliver；业需带 US 展示号，研需用纯编号，缺失研需不回落到业需。

治理结果：缺陷计数与窗口结果读取抽到两个共用 Repo 方法（各被业需和研需调用），交付错误映射复用一个 Handler 函数；未移动质量基线或添加 lint 豁免。研需查询补齐 fromDemand=0、type=story、非父、非需求池，权限保留指派人/产品 ReqM/超管；已交付、未绑定窗口、严重缺陷阻断提交，已交付判断沿用 storyDeliveredSQL，不看 deliverDate。取消/编辑 mode 不得绕过新建必填校验，成功 JSON 带 redirectUrl；窗口和人员取数错误向上返回。

前端修复：切换需求清空旧上下文，重试只请求当前对象，关闭或换对象后忽略旧 GET/POST 响应；422 保留输入、不提示成功，提交中禁止重复点击。已绑定的历史窗口加入当前选项，避免模型中已选窗口却显示“请选择”。真实验收还发现 form 不是弹性容器造成底部被裁，现用 hidden 控制表单、正文滚动、底部按钮固定；交付 CSS 字面颜色替换为既有语义变量，深色检查标签不再泛白。未新增全局 JS 函数、颜色变量或依赖。

只读 SELECT：库 zentaopms_migrated，独立研需 7,930 条；veriFier 或有效 verifyDate 已填 2 条，2 条均有 zt_action(objectType=story, action=deliver) 记录。68485 为 active/wait，未发起交付，产品计划 23606 对应有效窗口 2（26-0918窗口，2026-09-18）；数据中另有失效窗口 1，读取联表排除它。SQL 结果 /private/tmp/v2bugfix/delivery-select.txt，未写库。

验证：make check、make quality 均通过，dupl 0、新增 Go 质量问题 0、未豁免禅道直写 0；AGENTS 未定义颜色变量扫描 0，git diff --check 通过。新增回归覆盖匿名/他人、已交付、未排期、严重缺陷、原生拒绝及请求字段、422、取消/编辑校验、错误状态码、对象路由、重试、旧加载/提交响应、失败保留输入和重复点击。写流程全部使用 sqlmock/httptest/VM 模拟，没有发起真实交付。

8100 账号 003030：真实研需 68485 和业需 US63275 的表单正确加载；两主题及内部底部 8 张全页图逐张检查，责任人与保存按钮可见，控制台 error/warn 为空；滚动底部 scrollTop=149，clientHeight=524，scrollHeight=673。截图 /private/tmp/v2bugfix/delivery-{story,demand}-{light,dark}.png 及对应 -bottom.png，日志 delivery-govern-{check,quality}.log，不提交临时文件。8100 用完关闭；8099 原服务不覆盖。

验证边界：现有验收配置的 zentao.api 为空；本地 ~/GitHub/ZentaoPMS 的标准 stories API 源码没有交付扩展，不能据此确认实际定制禅道的 /stories/:id/deliver 是否已部署。保留原补丁接口路径，只验证本项目组装与模拟响应，不声称真实原生提交或窗口变更已通过；真实写入及定制接口契约仍需后续联调。另选窗口时，此研需补丁只校验窗口存在，提交 payload 不含 windowId，也未写回产品计划关联；不能声称窗口已变更，关联应如何更新须先确认原生契约，不能擅加直写。

| 文件 | 本轮净变化 |
|---|---:|
| internal/module/po/gateway.go | +8 |
| internal/module/po/handler.go | +2 |
| internal/module/po/handler_deliver.go | +38 |
| internal/module/po/handler_story_deliver_test.go | +49 |
| internal/module/po/primaryaction/primaryaction.go | +1 |
| internal/module/po/primaryaction/primaryaction_test.go | +6 |
| internal/module/po/primaryaction/primaryaction_url.go | +3 |
| internal/module/po/repo_deliver.go | -25 |
| internal/module/po/repo_deliver_scan.go | +40 |
| internal/module/po/repo_story_deliver.go | +82 |
| internal/module/po/service_home_actions.go | +1 |
| internal/module/po/service_story_deliver.go | +131 |
| internal/module/po/service_story_deliver_test.go | +140 |
| tests/unit/frontend/story-deliver.test.js | +107 |
| web/static/css/po/po-deliver.css | +3 |
| web/static/js/po/homereview.js | +2 |
| web/static/js/po/po-deliver.js | +26 |
| web/static/js/po/primary-action.js | +1 |
| web/templates/components/po_deliver_modal.html | +0 |

测试代码净 +302，非测试代码净 +313：增长主要来自原未提交的独立研需交付读接口、服务和原生适配接入；去重部分替换原两处读取与错误处理，不新增架构层。所有改动代码文件均小于 500 行。

新增生产符号：postZentaoDeliver、deliverStoryViaZentao、GetStoryDeliver、DeliverStory（Handler/Service）、writeDeliverActionError、readDeliverBugCounts、readDeliverWindow、FindDeliverStory、CheckStoryDeliverBlockers、FindStoryLinkedWindow、loadDeliverStory、GetStoryDeliverMeta、storyDeliverMeta；交付 JS storyKind、deliverDisplay、deliverEndpoint。新增结构 deliverStoryRow、错误 errStoryNotFound。新增 CSS 选择器 #poDeliverForm:not([hidden])，其余复用已有选择器；新增测试辅助 expectDeliverStory、node、jquery、flush、respond、meta、verifyRetryAndStaleResponse、verifySubmit、verifyStaleSubmit 及测试函数/回调。无第三方依赖。质量自查逐项完成。
本交接文档第 12 节净 +44 行。
