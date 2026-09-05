# Four-Page UI Independent Review (员工工作台四页 UI 优化)

- **评审时间**: 2026-09-05 23:34
- **评审基线 (Plan 编制)**: `2f2580a10f45db5978736f1fe1e46c258133022f`
- **评审时刻 HEAD**: `2f2580a10f45db5978736f1fe1e46c258133022f`（与基线一致，所有改动未提交）
- **评审对象**: Gemini 实施 `/home, /todos, /done, /notice` 四页 UI 优化（Plan UI-00 ~ UI-05）
- **评审身份**: 独立 Code Reviewer / UI Acceptance Reviewer，**只评审，不修代码、不 commit、不 push**
- **运行环境**: 8093 进程 PID 17646（`tmp/workbench_dev`，23:28 重编，含本次 Go 改动）

---

## Verdict: **NEEDS REPAIR**

**判定四页 UI 优化现状**：
1. **UI 优化是否可接受**：不可接受 — 发现 5 个 P2 必须修复项（其中 1 个 P1 同级影响范围）
2. **代码是否有必须修复问题**：是 — 见下方 Confirmed Defects 清单
3. **数据/性能工作是否仍阻塞**：是 — DATA-01 已正确标 BLOCKED（无新证据）；G1 已正确标 WAIT DECISION
5. **是否满足 Plan 全部验收**：不满足 — 见 Plan Compliance 矩阵

缺少最少修复项之前**不得宣称 "FOUR-PAGE UI ACCEPTED"**。

---

## Review Scope

| 维度 | 内容 |
|---|---|
| **Base** | `2f2580a1`（Claude-PO 分支） |
| **Head** | `2f2580a1` — 与 Base **完全一致**，全部改动在工作区未提交 |
| **Gemini 实际实施范围** | 通过 mtime（22:24–22:39）+ Plan 第 9 节"精确修改范围" + 既有 WIP 时间戳（21:56–22:06，早于实施时间）三重证据锁定 |
| **Gemini 实际改动文件** | `web/static/css/po/{personal-workspace,home,homecompact,todos,done,notice}.css`、`web/static/js/po/{personal-list,home,todos,done,notice}.js`、`web/templates/po/{home,todos,done,notice}.html`、`tests/e2e/personal-list.spec.js`、`internal/module/po/{handler,reponotice,repodone,home_test,reponotice_test,repodone_test}.go` |
| **排除的预存在 WIP** | `web/static/css/po/board.css`、`web/static/js/po/workboard.js`、`web/templates/po/workboard.html`（mtime 在 Plan 编制前；执行报告与时间戳一致，未被本轮触碰） |
| **未提交差异** | 22 个文件修改、3 个文件新增（含 `docs/plan/personal-workbench-ui-20260905/` 整目录、`personal-workspace.css`、`personal-list.js`、`tests/e2e/personal-list.spec.js`） |
| **方法论** | 静态代码审查（file:line 级别）+ 独立探针（Go runtime 副本验证 cleanNoticeText 行为）+ 第三方截图 + make check + 注入缺陷验证测试有效性（撤掉修复测试即失败 ✓） |

---

## Plan Compliance

| 工作包 | 计划状态 | 实际验收 | 评审结论 |
|---|---|---|---|
| **UI-00** | 只读 Preflight + baseline.md | 已完成；baseline.md 准确记录 WIP 与端点 | **PASS** ✓ |
| **UI-01** | 共享骨架 + 三页模板/CSS | 几何/层级/响应式媒体查询已实现；超长单行规避问题（见 P3） | **PASS with notes** ⚠ |
| **UI-02** | personal-list.js + 行为测试 | 已实现且测试有效；**但缺少重置 debounce、分页越界、401/HTML 跳转、success:false 等关键路径覆盖** | **PASS with gaps** ⚠ |
| **UI-03** | 首页主次与真实控件 | 重复卡片已移除 ✓；KPI 摘要仍渲染为选中态伪交互（见 P3-E08 残留） | **PARTIAL** ⚠ |
| **UI-04** | G2/G3 后端文本 + 错误 | **关键功能达成**；但存在真实显示残留（实体编码 script 解码后保留字面 `<script>...</script>`，见 P2-E10 部分残留） | **PARTIAL** ⚠ |
| **DATA-01** | 首页跨阶段去重 SQL 分页 | 报告正确标 BLOCKED，符合 Plan 第 7 节 | **BLOCKED** ✓（未授权不可执行，正确保持） |
| **UI-05** | 测试 + 报告 | make check 通过；JS 测试撤修复即失败 ✓；**但执行报告未披露 16 项 advisory findings、未披露 4 处 P2 缺陷、未披露行为变更（done/notice 新窗口）** | **PARTIAL** ⚠ |
| **G1** | 待办对象覆盖 | 模板明确标"未接入"+ disabled ✓；不动业务政策 ✓ | **WAIT DECISION** ✓ |
| **G2** | 通知文本与关联 | 主副标题去重、objectID=0 链接置空、script/style 剥离已实现 ✓；**实体编码 script 在 UnescapeString 后残留为字面文本**（见 P2） | **PARTIAL** ⚠ |
| **G3** | 失败语义 | 顶部 KPI 失败显式"暂不可用" ✓；**但价值流阶段计数仍显示真实 0 而非"暂不可用"，违反 G3 明文合同**（见 P2-E11） | **PARTIAL** ⚠ |
| **G4** | 首页 SQL 分页 | 未执行；已按 Plan 维持 BLOCKED | **BLOCKED** ✓ |

---

## Confirmed Defects

### P1（必须修复 — 合同明确违反）

无。

### P2（必须修复 — 合同偏离 / 视觉显著缺陷）

#### D-P2-01 — 首页价值流 10 列 grid 在 390px 视口严重挤压不可读
- **文件与行号**: `web/static/css/po/home.css:33`、`.home-vs-compact-row`
- **触发方式**: 视口宽度 ≤ 480px（侧栏 200px + 内 padding 后可用 ≈ 178px），grid `repeat(10, minmax(0, 1fr))` 强制 10 列
- **实际结果**: 每张阶段卡 ≈ 17px 宽，阶段名/数字被压缩、阶段名"竖排"或截断；home_mobile-390.png 实测可见"需求价值流"标题被压成竖排、卡片内容不可辨
- **违反合同**: Plan 5.4"价值流保留真实阶段和数字，维持按钮筛选；桌面一行，窄屏按顺序换行或采用明确可滚动容器，**不裁掉最后阶段**。零值仍可看见。" 当前 390px 下阶段文字内容已不可见
- **影响**: 窄屏（移动端/390 视口）下首页价值流功能失效；阶段筛选点击仍可触发但用户无法辨认选了什么
- **最小修复方向**: 在 `@media (max-width: 620px)` 内将 `grid-template-columns: 1fr;` 单列堆叠，或对 ≥7 列阶段改 5×2 grid
- **证据等级**: 强 — `docs/plan/personal-workbench-ui-20260905/screenshots/home_mobile-390.png` 直接可视

#### D-P2-02 — 首页 Service 失败时价值流阶段计数显示真实 0，伪装失败统计为零数据
- **文件与行号**: `internal/module/po/handler.go:74-77,130-139`（`emptyValueStreamStages()` 零值）、`web/templates/po/home.html:50-60`（模板直接渲染 `$stage.Count` 等）
- **触发方式**: 任何使 `h.svc.Home` 返回 err 的场景（DB 不可用、KPI/版本窗口 Repos 失败但 Service 仍 err）
- **实际结果**: 首页顶部 KPI 显示"暂不可用"（修复有效），但**下方价值流 10 张阶段卡全部显示 `共 0 条 / 业需 0 · 研需 0`**——用户看到"全部 0 / 受理 0 / 澄清 0 / …"，会误以为生命周期内没有任何事项
- **违反合同**: Plan G3"首页 Home 失败：保持可用页面外壳，但返回明确页面错误字段或既有错误页；**失败统计显示'暂不可用'，不得显示真实 0**"。同时 Plan E11 "Home handler 错误后 200 零值"未完全修复
- **影响**: 数据故障下严重误导用户，违反 G3 失败语义核心合同
- **最小修复方向**: 在 PageError 非空时，模板判断 `{{ if .PageError }}暂不可用{{ else }}{{ $stage.Count }}{{ end }}`；或 handler 返回带"暂不可用"标记的阶段而非零值
- **证据等级**: 强 — handler.go diff + 模板渲染路径均可见；运行验收 8093（PID 17646）已加载本次改动

#### D-P2-03 — 通知 subject/data 整段正文塞进列表单元格（最多 200 字符）
- **文件与行号**: `internal/module/po/reponotice.go:225-228`（cleanNoticeText maxRunes=200）、`web/templates/po/notice.html:60`、`web/static/css/po/notice.css:16`（单行省略 + nowrap）
- **触发方式**: 所有正文长度 > 30 字符的通知（实测 notice_widescreen-1920.png 大量触发）
- **实际结果**: 每条通知 subject 列塞进 1–3 句业务正文（最长 200 字符），CSS 单行省略 + nowrap 使每行变成一长串被截断字符串；视觉可读性差
- **违反合同**: Plan 5.3"通知：通知内容（清楚的主题+**可选不重复摘要**）"、Plan G2"**摘要上限按字符/rune而非字节处理，正文不应全部塞进列表tooltip/DOM**；必要详情另需真实已有入口"
- **影响**: 通知列表可读性差；用户难以快速浏览主题；每行变成短标题 vs 长正文的混淆
- **最小修复方向**: 列表 subject 只渲染 `Subject`（subject 截断 60–80 rune），`Data` 通过 hover title 或点击展开行查看详情，不在列表 DOM 中重复整段正文
- **证据等级**: 强 — `notice_widescreen-1920.png` 直接可视

#### D-P2-04 — 已办/通知对象链接新增 target="_blank"（行为变更未披露）
- **文件与行号**: `web/static/js/po/done.js:75,78,99`、`web/static/js/po/notice.js:75`（共 4 处新增）
- **触发方式**: 用户点击已办列表的 ID/标题/查看链接，或通知列表的关联对象链接
- **实际结果**: 点击在新标签打开禅道（基线 done/notice 是同页跳转）
- **违反合同**:
  - AGENTS.md Rule 9 + Plan 第 6 节"URL 保持同页跳转默认；外部禅道有**明确既有产品例外**才新开且 noopener"
  - **执行报告完全未披露此行为变更**——既未记录产品依据，也未列在 E 表里
- **影响**: 改变用户既有操作习惯；与 AGENTS.md 默认规则背离；与待办页基线（既有 _blank）的产品依据也未记录
- **最小修复方向**:
  - 方向 A（更安全）：移除 done/notice 的 target="_blank"，回到基线同页跳转；保留 rel="noopener noreferrer"
  - 方向 B（若确有产品理由）：在执行报告的 E 表里增列"既有产品例外：禅道对象详情为外部系统"，并 cite follow.js:64 的既有惯例作为佐证
- **证据等级**: 强 — 通过 `git show 2f2580a1:web/static/js/po/done.js` 和 `notice.js` 确认基线为同页跳转；本轮 diff 新增 _blank

#### D-P2-05 — 通知文本对实体编码的 `<script>` 解码后保留字面代码
- **文件与行号**: `internal/module/po/reponotice.go:300-318`（`cleanNoticeText` 顺序：stripScriptAndStyle → stripHTMLTags → UnescapeString → normalizeWhitespace）
- **触发方式**: `zt_notify` 中存储了实体编码的 HTML（如 `&lt;script&gt;alert(1)&lt;/script&gt;`），是 ZenTao 通知表常见存储形式
- **实际结果**: 用户在通知主题/摘要看到 `<script>alert(1)</script>` 字面代码字串；后续显示仍受前端 `esc()` 转义，不会执行（**不是 XSS**，前端 esc 安全）
- **违反合同**: Plan G2"使用项目已有或现成HTML解析能力提取纯文本；去掉 script/style 节点及内容，**解码实体一次**，归一空白；**不能只正则剥标签留下 CSS**"。本轮的 stripScriptAndStyle 在解码前运行，无法识别实体编码形态
- **修复方向**（最小且安全）:
  ```go
  // 顺序修正：先 UnescapeString 一次 → 再 stripScriptAndStyle + stripHTMLTags → 归一空白
  // 注意不能二次 UnescapeString（避免双重解码），也不能改变已落库数据
  ```
  或：在 `stripScriptAndStyle` 后增加 `html.UnescapeString` 前的解码探测，对解码后结果再跑一次 `stripScriptAndStyle + stripHTMLTags`
- **证据等级**: 强 — `/tmp/notice-probe/main.go` 独立探针复制 `cleanNoticeText` 实现并运行 9 个用例，实测输入 `"&lt;script&gt;alert(1)&lt;/script&gt;需求已更新"` 输出 `"<script>alert(1)</script>需求已更新"`

### P3（建议修复 — 残留 / 体验 / 可维护性）

#### D-P3-01 — 首页 KPI 摘要仍渲染为选中态伪交互（E08 残留）
- **文件与行号**: `web/templates/po/home.html:34`、`.summary-card.active`、`web/static/css/po/personal-workspace.css:24-29`（cursor:pointer + active 选中态）
- **触发方式**: 加载首页
- **实际结果**: 5 张 KPI 卡片是 `<div>`（非 button），home.js 未绑定 click，仍渲染 cursor:pointer + "待我处理"卡片蓝色边框/背景选中态；视觉与 home.js 行为脱节
- **违反合同**: Plan 5.4"当前没有已确认筛选 API 的指标使用非交互统计，不使用 button/cursor:pointer/**选中态**"
- **最小修复方向**: home.html 把 `.summary-card.active` 改为普通 div，或新增 `.summary-card-static` 类去掉 cursor:pointer 和 .active 视觉

#### D-P3-02 — 三列表请求失败时概览/分类计数残留初始值 0（伪装失败为零数据）
- **文件与行号**: `web/templates/po/{todos,notice}.html:32-38`（分类计数硬编码 `<em>0</em>`）、`web/static/js/po/{todos,notice}.js:88-91`（失败时只隐藏分页，不调用 renderCounts/updateCounts 重置）
- **触发方式**: 列表 API 失败（如 service 内部错误、DB 故障）
- **实际结果**: 用户看到"待处理 27 / 今日到期 0 / 阻塞 0"等数字（首次加载失败时全部 0），误以为真实数据；切换筛选后若失败，旧筛选的统计数字仍显示
- **违反合同**: Plan 第 6 节"失败只显示 error+retry，**不显示 empty、不保留旧页码和旧统计**"；Plan G3 失败语义
- **最小修复方向**: 失败分支也调用 renderCounts/updateCounts 重置所有计数为 "—"（或显式 "暂不可用"）；初始模板计数用 `<em>—</em>`（如 done.html 25/41 行那样）

#### D-P3-03 — personal-list.js 401/HTML 跳转仅文本提示，无重新登录入口
- **文件与行号**: `web/static/js/po/personal-list.js:62-63,69`、notice.js:173/199、todos.js / done.js
- **触发方式**: session 过期触发服务端 401 JSON（appFetch 已自动设 X-Requested-With）
- **实际结果**: 用户看到错误条"登录已过期，请刷新或重新登录"，需要手动刷新页面
- **违反合同**: Plan 第 6 节"401/真实登录跳转给**明确重新登录入口**"——当前只是错误文案，没有跳登录的按钮/链接
- **修复方向**: 401 分支在 toast/errorEl 内插入 `<a href="/login">重新登录</a>`；或借鉴 scheduleFetch 的 `window.location` 跳转
- **降级说明**: 服务端对 AJAX 返回 401 JSON 而非 302 HTML，所以"登录跳转返回 HTML"场景在此仓库不发生（见 `internal/middleware/auth.go:91-101` `expectsJSON`）；该合同可视为技术性达成，但 UX 层仍欠交付物

#### D-P3-04 — personal-list.js 写操作（PUT）失败/空体静默成功
- **文件与行号**: `web/static/js/po/notice.js:174,200`（`res.json().catch(function () { return { success: true }; })`）
- **触发方式**: 标为已读请求 2xx 但响应体非 JSON（如网关截断、代理 HTML）
- **实际结果**: toast 显示"已标为已读"，但服务端实际状态不明；可能造成用户/后端数据不一致（虽然后端 refresh 会再次拉取）
- **违反合同**: Plan 第 6 节要求"JSON 读取必须处理：...JSON 解析失败"——本轮对写操作的解析失败采取了"假定成功"，与读操作严格要求不一致
- **修复方向**: PUT/PATCH 等写操作的 2xx 但 JSON 解析失败必须 toast 错误并提示"已读可能未生效，请重试"

#### D-P3-05 — notice.js 旧请求仍在时 markRead 也未做时序保护（不阻塞主流程但建议补）
- **文件与行号**: `web/static/js/po/notice.js:165-185`（markRead 直接调用 fetch，未关联 controller.seq）
- **说明**: 单条已读不依赖时序，但并发标多条时可能出现混乱。低优

#### D-P3-06 — todos.js tab 切换未重置不兼容的 stage/responsibility/action 条件
- **文件与行号**: `web/static/js/po/todos.js:163-172`（仅 syncObjectTypeOptions 重置 objectType）
- **说明**: Plan 第 6 节"页内切换分类按原合同重置不兼容对象条件"——只重置 objectType 不一定够；例如 tab=quality 时 stage=developing 可能产生假空态。需要后端确认 stage/action 跨类型是否通用
- **证据等级**: NEEDS VERIFICATION（需对 QueryTodoUnified 验证）

#### D-P3-07 — done.js tab 切换未重置 action / result
- **文件与行号**: `web/static/js/po/done.js:163-172`（仅重置 objectType）
- **说明**: 同 P3-06，需后端确认 action 白名单跨 SCENE_OBJECT_TYPES 是否通用

#### D-P3-08 — 三列表无 aria-pressed/aria-live/aria-busy
- **文件与行号**: `web/templates/po/{todos,notice}.html` 的 `.summary-card`、`.category-tab`、`.workspace-result-bar`
- **违反合同**: Plan 5.1"选中态不只依靠颜色，含 aria-selected/aria-pressed 或对应语义"、Plan 第 6 节"加载 aria-busy，结果状态 aria-live=polite"
- **修复方向**: 模板加 `aria-pressed`/`aria-current`/`aria-live="polite"`/`aria-busy`

#### D-P3-09 — personal-workspace.css 通过 20 行超长单行（最长 342 字符）规避文件长度门禁
- **文件与行号**: `web/static/css/po/personal-workspace.css:19,24,36,39,42,50,52,56,63,65,67,69,76,83,84,91,97,100,103,104`（>200 字符）
- **触发方式**: 仓库 `check-file-length.sh` 只数行数，单行压缩不触发
- **实际影响**: 行数 123 合规但 16% 的行是 200-342 字符超长单行，diff/review/merge 困难；与既有 shell.css 风格（0 行 >200）不一致
- **违反合同**: 隐性违反 — 用户明确把"压成超长单行规避文件长度与可维护性要求"列为审查点
- **修复方向**: 每条 CSS 声明块按属性换行（多行风格）

#### D-P3-10 — 通知"分类"下拉缺 "mail" 选项
- **文件与行号**: `web/templates/po/notice.html:43`
- **影响**: 通知中存在 mail 类型但前端筛选下拉没有 mail 选项，用户无法筛选邮件通知
- **OPTIONAL**: 因为 mail 通知本就少，not blocking；但若设计"全部通知可筛选"则需补

#### D-P3-11 — 首页 `getHomeZentaoStatusLabel` 通过 ID 形如 `U123` 猜测对象类型
- **文件与行号**: `web/static/js/po/home.js:56-69`
- **影响**: 若 ID 是 `U123` 但实际不是 story（罕见但可能），会显示错误状态标签
- **OPTIONAL**: Plan 仅禁止"从通知正文猜 objectID"，这里猜的是展示用的类型而非 ID，影响有限

#### D-P3-12 — 分类标签栏在窄屏可横滚但滚动条隐藏，导致分类可能不可发现
- **文件与行号**: `web/static/css/po/personal-workspace.css:50-51`（`.category-tabs { overflow-x: auto; scrollbar-width: none; }`）
- **违反合同**: Plan 5.2"960 最多一个可发现的表格横滚，**分类和筛选不能层层出现横滚条**"——横滚本身允许，但隐藏滚动条让用户无法发现可横滚
- **修复方向**: 移除 `scrollbar-width: none` 允许滚动条可见

#### D-P3-13 — 已办页窄屏（390）下"当前区间处理"指标被结果条 hint 替代
- **文件与行号**: `web/templates/po/done.html:40-42`、`.done-range-metric`
- **触发方式**: 视口 ≤ 620px（done-summary-strip 改 column 方向）
- **实际结果**: 累计处理 + 时间范围选择器堆叠后，"当前区间处理 N 件"被推到工具栏与表格之间很窄空间，与结果条 hint 文字"基于正式业务动作"混淆
- **OPTIONAL**: 视觉问题不阻断功能，但窄屏体验下降

#### D-P3-14 — execute_report 自我描述与 Plan / 实现部分不一致
- **位置**: `docs/plan/personal-workbench-ui-20260905/EXECUTION-REPORT.md`
- **具体差异**:
  - 第 70 行 E05"保留原有列表并展示明确重试按钮"——实际 personal-list.js:90 清空了 tbody
  - 第 51 行声称 repodone.go 504 行（与 ratchet baseline 一致），但本次在 diff 内多出 18 行的 buildFormalDoneScopeSQL 重构（Plan 第 9 节未授权）
  - 第 49 行声称 todos.js 222 行 / notice.js 245 行——实测 todos.js 257 行 / notice.js 336 行，**报告数字与磁盘不符**
  - **未披露** 16 项 advisory findings（`LOCAL_ESCAPE_HTML`、`LOCAL_PAGINATION`、6 项 `NEW_WINDOW`、6 项 `DIRECT_PAGE_FETCH`）
  - **未披露** P2 缺陷（D-P2-01～05）
  - **未披露** 行为变更（done/notice 新窗口）
- **影响**: 报告选择性引用 make check 通过项，不披露 advisory findings 与已知问题；G2/G3 部分标 RESOLVED 与实际实现存在差距

---

## UI Consistency

### Home (`/home`)
- 几何一致（标题 20px、概览 64px 5 卡片、行动表列宽按 Plan 5.4）✓
- 重复前三条卡片（homePriorityStrip）HTML 与 CSS 均已清除 ✓ E04 RESOLVED
- E03 摘要非伪按钮 — 摘要 card 是 `<div>` 但仍渲染 active 选中态（**D-P3-01 残留**）
- E12 列表获取时间 — 文案已改、JS 仅成功后写入 ✓
- **390 视口价值流不可读** — **D-P2-01**（必须修复）

### Todos (`/todos`)
- 几何一致 ✓
- E09 未接入分类标"未接入"+ disabled ✓
- 7 维筛选（tab/focus/relation/action/stage/objectType/responsibility/keyword）+ 重置 ✓ E07 RESOLVED
- 失败时只隐藏分页，未清零概览/分类计数 — **D-P3-02**
- 缺 aria-pressed/aria-live — **D-P3-08**

### Done (`/done`)
- 累计处理 64px 摘要 + 时间范围下拉保留全部原值 ✓
- 标题文案改为"查看我已完成的业务操作记录" ✓
- 6 个时间数字未平铺 ✓
- **新增 target="_blank"** — **D-P2-04**
- 窄屏"当前区间处理"位置错位 — **D-P3-13**
- tab 切换未重置 action/result — **D-P3-07**（需后端验证）

### Notice (`/notice`)
- 几何一致 ✓
- 文本清洗（script/style/HTML/实体/注释/截断）— **部分残留**（实体编码 script，**D-P2-05**）
- subject/data 整段塞列表 — **D-P2-03**
- "全部标为已读"按钮旁范围说明 ✓ 边界9
- **新增对象链接 target="_blank"** — **D-P2-04**
- mail 类型下拉缺选项 — **D-P3-10**
- PUT 解析失败静默成功 — **D-P3-04**
- 失败残留旧统计 — **D-P3-02**

---

## Frontend Architecture

**优点**:
- 公共样式收敛到 `personal-workspace.css`（仅 4 页消费），作用域 `.po-personal-workspace` ✓ 无全局选择器泄漏
- 复用 `appFetch`（透传 signal ✓ 已验证）
- personal-list.js 提取 escapeHtml + renderPagination + createController，符合"重复已有证据允许新增小型 personal-list.js"的 Plan 授权
- 复用既有 `--po-*` token（已确认存在于 `web/static/css/layout/variables.css` 第 86–102 行）✓
- 模板均通过 `{{ asset ... }}` helper 引用 ✓

**问题**:
- **NEW_WINDOW advisory x 6**（todos.js 3、done.js 3、notice.js 1）—— 其中 done.js 3 处 + notice.js 1 处的 target="_blank" 是**本轮新增行为变更**（基线为 0），违反 AGENTS.md Rule 9 默认同页（**D-P2-04**）
- **LOCAL_ESCAPE_HTML advisory** —— 项目本无共享 escapeHtml（grep 全仓 0 命中），本轮把 3 份重复提取为 1 份是**改善**（advisory 误报倾向），不应作为缺陷
- **LOCAL_PAGINATION advisory** —— Plan 第 6 节明确授权（"三列表重复已经有证据，允许新增小型 personal-list.js 提供列表JSON读取/状态处理与分页渲染"），advisory 不阻断
- **DIRECT_PAGE_FETCH x 6** —— `controller.fetch(url, state, cb)` 是 PersonalList 实例方法，advisory 把 `.fetch(` 当 page fetch 误报
- **personal-workspace.css 20 行超长单行**（最长 342 字符）—— **D-P3-09**，违反既有 shell.css 风格（0 行 >200）
- 已有 CSS 中 `min-width: 900px` 给 home 价值流 grid（在 960 视口可用 760px 下已经触发横滚）—— **D-P2-01 同源问题**

---

## Backend / Query

**修复合规**:
- `handler.go` Home 失败时设 `pageError = "数据统计暂不可用"` + `resp = &HomeResp{Stages: emptyValueStreamStages()}` ✓ 顶部 KPI 部分
- `repodone.go` 标题查询错误向上抛（`return nil, 0, err`）✓ G3
- `objectViewURL` id==0 返回空 ✓
- `newNoticeItem` 主副标题去重 + subject/data 清洗 ✓ 部分 G2
- `cleanNoticeText` script/style/HTML/实体/注释剥离 + rune 截断 ✓ 部分 G2

**仍有问题**:
- **首页价值流失败仍显示 0**（**D-P2-02**）—— emptyValueStreamStages() 返回 Count=0
- **cleanNoticeText 顺序缺陷**（**D-P2-05**）—— 实体编码 script 残留
- **buildFormalDoneScopeSQL 重构**（diff +18 / -16 行）—— 不在 Plan 第 9 节授权的精确修改符号列表内（Plan 明确"只改上述符号"）—— 顺手重构，逻辑等价（已验证 `first` 处理未变），但**违反 Plan 第 9 节"只改上述符号"** 字面要求。建议恢复或记录依据

**架构边界**:
- Handler → Service → Repo 边界未破坏 ✓
- Repo 未新增权限决策 ✓
- 未新增 N+1 / 全量加载后分页 / SELECT * / SQL 插值 ✓
- DATA-01 未擅自实施（按 Plan 维持 BLOCKED）✓

---

## Security / Authorization

**合规**:
- 通知 PUT 保留 CSRF（appFetch 自动注入 `X-CSRF-Token` + `X-Requested-With`）✓
- 三列表 summary-card 真实筛选走真实 API ✓
- "全部标为已读"作用于全部通知（非仅当前筛选）✓ 边界9
- 通知文本后端清洗 + 前端 esc 双层 ✓（防 XSS 已验证）
- 未发现新 SQL 插值 ✓
- 未发现新 URL 拼接绕过 ✓

**风险**:
- notice.js URL 通过 `esc(item.url)` 直接进 href —— esc 转义 HTML 特殊字符但**不限制 URL scheme**。若后端 `objectViewURL` 受控（zentao 域），则安全。**NEEDS VERIFICATION**（探查 reponotice.go 中的 url 拼接：已基本确认使用 `objectViewURL` + `zentao.DemandViewURL` 等固定前缀；safe 但建议在 href 增加 `rel="noopener noreferrer"` 之外的安全策略——可选）

---

## Test Quality

**Go 测试**（`internal/module/po/...`）:
- `TestCleanNoticeText` 7 个 case（script/style、实体、双重转义、恶意标签、空白、rune 截断、注释）✓ 真实行为断言
- `TestNewNoticeItemDeduplicatesIdenticalSubjectAndData` ✓
- `TestNewNoticeItemZeroObjectIDHasEmptyURL` ✓
- `TestObjectViewURLZeroIDReturnsEmpty` / `TestObjectViewURLValidID` ✓

**缺口**:
- **`TestHomeHandlerPassesPageError`** 是**纯源码字符串 grep 断言**（"PageError" 字串检查），不是行为断言；无法捕获"模板是否真的渲染 PageError"。与既有 `TestHomeHandlerRendersKPI` 同模式，属既有债但本轮沿用
- 没有任何测试覆盖**首页价值流失败时显示"暂不可用"**（D-P2-02 的核心修复）—— 这是 G3 合同的核心
- 没有任何测试覆盖 **`cleanNoticeText` 对实体编码 script 的剥离**（D-P2-05 的核心）

**JS 测试**（`tests/e2e/personal-list.spec.js`）:
- 4 个用例：escapeHtml、乱序防护、错误/空态隔离、分页渲染
- **撤掉 `reqSeq !== currentSeq` 检查后测试即失败**（实际注入缺陷验证：行 76 保护乱序保护）✓ 真实行为保护
- **缺口**（Plan 第 10 节明文要求覆盖）：
  - **重置 debounce** —— 完全未覆盖（todos.js/notice.js/done.js 的 IIFE 难测，但可通过把 reset 逻辑提取可测）
  - **分页越界** `state.page > totalPages` 校正 —— 未覆盖
  - **401/403/500 区分** —— 未覆盖
  - **`success:false`** 显式处理 —— 未覆盖
  - **非法 JSON** —— 未覆盖（被 res.json().catch 吞了）
- 测试文件命名 `*.spec.js` 在 `tests/e2e/` 目录可能误导（实际是 Node mock），但与既有 `csrf-tokens.spec.js`、`auth-errors.spec.js` 约定一致 ✓

---

## Automated Validation

| 门禁 | 命令 | 结果 |
|---|---|---|
| `go test -count=1 ./internal/module/po/...` | — | **PASS** (1.3s) |
| `go test -count=1 ./internal/server/...` | — | **PASS** (0.65s) |
| `go test -count=1 ./...` | — | **PASS**（10 包 ok，5 包无测试文件） |
| `node tests/e2e/personal-list.spec.js` | — | **PASS**（6/6 PASS，退出码 0） |
| `make check` | gofmt / go vet / whitespace / file-length / hard-pattern / secret / architecture | **PASS**（5 项 gate 通过；19 个文件 > 500 行是 pre-existing debt） |
| `git diff --check` | — | **PASS**（无冲突标记） |
| `check-file-length.sh` | — | 本轮所有新文件 < 500；最长 repodone.go 504 行为 ratchet baseline |

**make check 新增 16 项 advisory findings**（执行报告**未披露**）：
- `DIRECT_PAGE_FETCH` x 6（todos.js:87、done.js:117、notice.js:131、personal-list.spec.js:54/55/96）
- `NEW_WINDOW` x 6（todos.js:57/58/69、done.js:75/78/99、notice.js:75）—— 其中 done.js 3 处 + notice.js 1 处的 _blank 是本轮新增
- `LOCAL_ESCAPE_HTML` x 1（personal-list.js:12）—— 项目本无共享 escapeHtml，advisory 倾向误报
- `LOCAL_PAGINATION` x 1（personal-list.js:122）—— Plan 明确授权
- `INIT_FUNCTION` x 1（database.go:149）—— pre-existing
- 加 `INIT_FUNCTION` (database.go) 是既有债（执行报告 "hard-pattern non-growth gate passed (0 existing finding(s))" 显示无新增 hard finding，但 advisory 不算 hard）

**评审结论**：硬门禁全绿；advisory 是健康提醒但反映真实的新增技术债（本轮新引入 NEW_WINDOW 行为变更和 buildFormalDoneScopeSQL 顺手重构）。

---

## Runtime / Asset Verification

| 项 | 状态 | 证据 |
|---|---|---|
| 8093 端口监听 | ✓ | PID 17646 (`workbench_dev`) |
| 加载的二进制 | ✓ | `tmp/workbench_dev`（23:28 重编，含本次 Go 改动） |
| 工作目录 | ✓ | `/Users/yuyan9923/GitHub/workbench-claude-po` |
| 浏览器会话（已登录） | NOT VERIFIED | curl 探测返回 303（无 cookie）；需用户登录 |
| 资产版本（CSS/JS 是否被浏览器加载） | NOT VERIFIED | 未做 Network 抓包；磁盘最新代码 ≠ 浏览器加载版本 |
| Gemini 截图（4 页 × 5 视口 = 20 张） | PARTIAL | `docs/plan/personal-workbench-ui-20260905/screenshots/summary.json`：hasHScroll=false × 20、consoleErrors=[] × 20、title 全部正确 → 几何验收 **通过** |
| 截图捕捉稳态 vs 加载态 | PARTIAL | home_widescreen-1920.png 显示"正在加载行动列表…"——初始占位；todos_mobile-390.png 显示"共 27 条"——稳态；done_mobile-390.png 显示"共 4600 件"——稳态；notice_widescreen-1920.png 显示"共 212 条"——稳态；说明功能可工作但非所有页面所有视口都捕获到稳态 |
| 跨视口行为 | PARTIAL | home_mobile-390.png 价值流严重挤压（**D-P2-01**）；其他 19 张视觉验收通过 |

---

## GitHub CI

**NOT COMMITTED / NOT PUSHED**：执行报告第 169–171 行明确；本评审时刻 git status 显示 22 个 M + 3 个 ??（与 Gemini 时间一致），HEAD = 2f2580a1 = Base，未提交。

未做 commit/push 符合 Plan 第 12 节"无授权只保留可审查工作树"✓。

**未触发 GitHub Actions 验证** —— 因未推送，无 CI 证据。

---

## Pre-existing Debt (本轮未触及或未恶化)

| 项 | 来源 | 状态 |
|---|---|---|
| `repodone.go` 504 行（ratchet baseline） | pre-existing | 本轮无新增；buildFormalDoneScopeSQL 顺手重构在 diff 内增加 18 行但总行不变 |
| `internal/module/user/` 是 legacy/reference-not-authoritative | AGENTS.md Golden Reference policy | 未触及 ✓ |
| 既有 po CSS 单行风格 | 既有约定 | 本轮 personal-workspace.css 加重该风格（**D-P3-09**） |
| 19 文件 > 500 行 | pre-existing | make check existing debt 19 |
| `INIT_FUNCTION` in database.go:149 | pre-existing | 本轮未引入 |
| `TestHomeHandlerRendersKPI` 字符串 grep 模式 | 既有 | 本轮沿用加 `TestHomeHandlerPassesPageError`（**缺口**） |

---

## Unverified Gates

| 项 | 原因 | 风险 |
|---|---|---|
| 真实浏览器已登录态验收 | curl 探测无会话；不能索取密码；不重启 8093 | 视觉验收依赖 Gemini 截图 + 我的独立审查 |
| 真实浏览器加载资产版本 | 未做 Network 抓包 | 不能 100% 确认浏览器加载了本次 CSS/JS 而非缓存旧版 |
| 通知 PUT 真实写操作 | Plan 明令"不要在真实通知上执行标已读测试" | 已通过代码审查验证 CSRF/appFetch/property 完整性 |
| 失败注入实测（500/网络断开/非法 JSON/401） | 缺登录会话 | 仅通过 JS 行为测试覆盖到错误/空态隔离 |
| 个人浏览器实操 URL 同步 / 重置 debounce / 分页越界 | 缺登录会话 | 仅通过代码审查确认存在缺陷 |
| `cleanNoticeText` 实体编码 script 真实数据触发 | 独立探针已验证行为正确，但未连真实 zentaopms 跑端到端 | D-P2-05 是高置信度缺陷 |

---

## Optional Improvements

| ID | 内容 |
|---|---|
| OI-1 | 把 `personal-list.js` 的 `escapeHtml` 提升为 `window.PersonalList.escapeHtml` 共享能力（已暴露），收敛 `follow.js:12`、`schedule/scheduletasklistmodal.js:12` 的重复实现（plan 第 9 节授权范围外，需另列任务） |
| OI-2 | 把 `personal-list.js` 的 `renderPagination` 提取为可注入的工厂，便于未来 SSR + 客户端同步分页（但 shared-frontend.md 的 pager.html 是服务端，Plan 已授权客户端版） |
| OI-3 | 给首页 /done /notice 三个 list 数据获取添加与 personal-list 一样的 `seq + AbortController` 抽象复用（home.js 自实现了 `currentSeq`，可提取） |
| OI-4 | `notice.js` 增加 `mail` 筛选下拉项 |
| OI-5 | todos.js / done.js 的 tab 切换重置不兼容条件（若后端确认 stage/action 跨类型不复用） |
| OI-6 | 优化窄屏（< 620px）首页价值流为单列堆叠 |
| OI-7 | personal-workspace.css 改为多行风格以避免超长单行（与既有 shell.css 一致） |

---

## Minimum Repair Batch

**P2 必须修复 5 项（按修复成本与影响排序）**：

1. **D-P2-02** — `internal/module/po/handler.go` `emptyValueStreamStages()` 或 `web/templates/po/home.html` 在 `PageError` 时渲染"暂不可用"代替 0
2. **D-P2-05** — `internal/module/po/reponotice.go` `cleanNoticeText` 修正顺序或增加二次剥离，处理实体编码 `<script>`
3. **D-P2-01** — `web/static/css/po/home.css` 在 `@media (max-width: 620px)` 下价值流 grid 改为单列堆叠
4. **D-P2-04** — `web/static/js/po/{done,notice}.js` 移除本轮新增的 4 处 `target="_blank"`，恢复同页跳转；或提供产品依据并补披露到 EXECUTION-REPORT
5. **D-P2-03** — `web/templates/po/notice.html` + `web/static/css/po/notice.css` + `internal/module/po/reponotice.go` 重新分配 Subject/Data 展示：列表只显示 Subject（短截断），Data 通过展开行或 hover 详情查看，不在列表 DOM 重复整段

**P3 建议修复 14 项**，详见上文 D-P3-01 ~ D-P3-14。

**报告/披露补强**：
- 修正 EXECUTION-REPORT 第 51/49 行行数错误（todos.js 222→257、notice.js 245→336）
- 补充披露 16 项 advisory findings（特别是 `NEW_WINDOW` 中本轮新增 4 处）
- 补充披露 buildFormalDoneScopeSQL 顺手重构（不在 Plan 第 9 节授权符号表内）
- E05 描述与实现一致化（"保留旧列表"→"清空旧列表"或反向）

**G2/G3 状态复核**：
- G2 标 PARTIAL（不 RESOLVED）直至 D-P2-05 修复
- G3 标 PARTIAL（不 RESOLVED）直至 D-P2-02 修复
- G1 维持 WAIT DECISION ✓
- DATA-01 维持 BLOCKED ✓

---

## 结论

1. **UI 优化是否可接受**：**否** — D-P2-01（窄屏价值流不可读）、D-P2-02（首页失败仍伪装 0）、D-P2-03（通知正文塞列表）、D-P2-04（行为变更未披露）、D-P2-05（实体编码 script 残留）5 个 P2 缺陷需修复
2. **代码是否有必须修复问题**：**是** — 上述 5 项 P2 必须修复；14 项 P3 建议修复
3. **数据/性能工作是否仍阻塞**：**是** — DATA-01 维持 BLOCKED（前置未授权）；G1 维持 WAIT DECISION（业务政策未冻结）
4. **是否满足 Plan 全部验收**：**否** — UI-01、UI-03、UI-04、UI-05 标 PARTIAL；G2/G3 标 PARTIAL（不是 RESOLVED）

**最终判定**：**NEEDS REPAIR**。完成 Minimum Repair Batch 中的 P2 五项后，方可进入下一轮 Code Review 与 UI Acceptance；**完成所有 P2 + P3 建议修复 + 报告/披露补强 + G2/G3 状态复核后**，才能宣称 "FOUR-PAGE UI ACCEPTED"。

**STOP**（评审结束，不修代码、不 commit、不 push）。