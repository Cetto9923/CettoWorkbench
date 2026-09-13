# 指标模块 P0「诚实化」最小修复方案

状态：**PROPOSED** —— 仅方案，本轮未改任何代码、未提交、未部署。
产出日期：2026-09-13。分支：`release/po-integrate-main-202609`，HEAD：`3507193e9970dfefd430b8dbf22b079e1c354193`。

---

## 0. 任务定位

**一句话**：把 `/metrics/radar` 与 `/metrics/manage` 上「不是真数据、却被渲染成真实指标」的内容清掉，使页面符合业务真源，并保持改动面最小（仅前端 4 个文件、零后端、零 DB）。

**驱动真源**（`docs/PRD/17-PO角色开发需求说明书/科技研发智能工作台_PO角色开发需求说明书_V11.1_两批交付收口版.html`）：

| 行 | 原文要点 |
|---|---|
| 262-263 | 已有真实口径的"交付周期"可以展示；其它指标如果数据真源/口径尚未落地，**可以保留位置但显示"-- / 暂无数据"，不得用假数据或前端拼算** |
| 756 | **没有分母、数据未同步或样本为空时显示"--/无数据"，不能用 0% 代替**；分类得分必须可解释 |
| 763 | 优先核对 ③ **指标雷达中的 0% 要区分真实 0 和无数据** |
| 774 / 805 | 指标管理/雷达/预警属**第二批**，须用新仓 `zt_wb_*` 自有表，状态「未实现」 |

**问题定性（重要，勿误判）**：本任务**不是**"配置存不下"（那只是功能缺失），而是**假数据被当作真实指标呈现给 PO**。后者触碰 PRD :263 明文红线，优先级更高。`/metrics/manage` 性质不同——它**不展示指标数值**（见 P0-5），问题仅为"配置不落库 + 假生效"。

---

## 1. 强制保护（forbidden-actions）

- **不动后端**：`internal/**` 0 改动；不新增路由、不新增 perm 常量、不新建 `zt_wb_*` 表（该表不存在）。
- **不动数据库**：0 迁移、0 SQL、0 schema。
- **不动既有接口契约**：`/metrics/api`、`/metrics/radar/data`、`/board/group/metrics` 的请求/响应结构不得变更。
- **不动分支**：不新建、不切换、不 merge/rebase/cherry-pick；禁 `reset`/`restore`/`stash`/`clean`。
- **不动他人 WIP**：当前工作树 76 modified / 22 untracked 一律保留；提交时**显式列路径**，禁 `git add .` / `-A` / `-am`。
- **No drive-by refactor**（AGENTS §11）：不替换 toast/escape helper、不拆文件、不改 CSS 主题体系、不顺手清理其它 mock。
- **不为了过测试而改测试或造 mock**（AGENTS §10）。
- **本轮只出方案**；实施与提交须用户另行明确授权。

---

## 2. 必读治理真源（实施前）

工程真源（按 `docs/engineering/agent-onboarding.md` 做 pre-flight）：

- `AGENTS.md` —— §8 文件长度（≤500 MUST，例外只可缩不可涨）、§9 前端复用、§11 最小改动、§12 交付真值（`可交付` 仅在全部门禁通过时可用）。
- `docs/engineering/frontend.md` —— 尤其：模板渲染服务端数据、JS 不得是 totals 的唯一实现；**加载/空/错误三态必须区分，错误不得渲染成空**。
- `docs/engineering/shared-frontend.md` —— §1.1 `window.appFetch` 返回**原始 Response**，调用方必须自查 `resp.ok`；§3 UI 组件边界。
- `.cursor/rules/frontend.mdc` —— 复用共享能力、长度门禁、light/dark 双主题验证。
- `docs/engineering/quality.md` —— 访问与评审证据要求。

---

## 3. 事实认定（均已实测复核，含 `文件:行`）

后端是**真代码**：`internal/module/metrics/handler.go:42-49` 注册 4 条 GET（无任何写端点）；`repo.go:55-66` 单条 SQL 11 个 COUNT 子查询读 `zt_story/zt_bug/zt_task`；`service.go:241-408 buildItems()` 硬编码 12 条定义；`internal/bootstrap/bootstrap.go:148` 注入只读 DB；`internal/server/routes.go:114` 注册路由。

**但真值从未到达页面。** 逐条：

| # | 事实 | 位置 |
|---|---|---|
| F1 | 雷达页 fetch 了 `/metrics/api`，返回值赋给 `baseResp` 后**全文再无引用** → 12 条真值取到即丢 | `web/static/js/metrics-radar.js:105,108,112` |
| F2 | `/metrics/radar/data`（后端 5 分类真聚合）**前端 0 处引用 = 死端点** | `handler.go:48` → `Service.Radar` |
| F3 | `/metrics/manage` 只加载 `personal-list.js` + `metrics-manage.js`；后者**无 fetch/XHR**；渲染源是本地 14 项常量，与后端 12 条是**两套不相交字典** | `manage.html:290-293`、`metrics-manage.js:30` |
| F4 | 雷达页 13 项中 4 项为纯 seed 假算（`doneRate`/`active`/`bug.open`/`sp.deviationRate`，由月份取模 + 团队名 hash 生成），3 项有常量兜底（gate/bugClose/bugResponse），1 项硬编码 `"96%"` | `metrics-radar.js:161,205,249,260,216,227,238,273` |
| F5 | 有真值的 7 项来自**另一个模块** `/board/group/metrics`（po 模块真 SQL）；且汇总视图 `teamId=0` 时**该请求根本不发** | `metrics-radar.js:104,106`、`internal/module/po/repoboardmetrics.go` |
| F6 | **模板首屏即假**（服务端渲染就有数字）：`kpiScore=96`、"质效优秀 ↑"、"2026年09月 运行快照"、月份 option 写死；manage 页 14/14/13/13/5 与"共 14 条指标定义" | `radar.html:27,79,106,107,168`、`manage.html:20,25,30,35,40,109` |
| F7 | 无数据被渲染成"最差"：`score` 初始 100，`total>0` 时按 `(normal+warn*0.6)/total` 计算 → 全 unknown 的分类得 **0 分 / severity="danger"**；`:471` 的 `unknown` 默认分支永远走不到 | `metrics-radar.js:291-326,471` |
| F8 | fetch 失败被吞成空数组 → **错误渲染成空态**（违反 frontend.md「Errors must not render as empty」） | `metrics-radar.js:108,109` |
| F9 | localStorage 伪持久化 + toast「已成功保存并生效」 | `metrics-manage.js:257,277,448,508` |

**行数门禁现状（P0 必须知道的约束）**：`scripts/check-file-length.sh:72` 扫描 `web/static/js/*`；`>500` 行且不在 `scripts/quality-baseline/file-length.tsv` 中即判 `new over-limit file` 并 exit 1。实测 `metrics-radar.js` **708 行**、`metrics-manage.js` **618 行**，且两者**均不在基线**（基线仅 16 条），脚本第 52 行明确拒绝把新超限文件加入基线。→ **该门禁当前即为红**，属既有基线失败，按 AGENTS §12 记 `BLOCKED BY EXISTING BASELINE`，**本任务不承担把 708 降到 500 的责任**（那需要拆文件，属 R3），但本任务改动**必须使两文件行数净减少**。

---

## 4. 改动清单

改动面：`web/static/js/metrics-radar.js`、`web/static/js/metrics-manage.js`、`web/templates/metrics/radar.html`、`web/templates/metrics/manage.html`。**零 Go 改动。**

### P0-1 删除全部 seed 假算与常量兜底（F4）

- 现状：`metrics-radar.js:121-133` 构造 `monthSeed`/`orgSeed`；`:161,205,249,260` 用其生成假值；`:216,227,238` 用常量兜底；`:273` 硬编码 `"96%"`。
- 改法：删除 `monthSeed`/`orgSeed` 及其全部下游表达式。规则统一为——**数据源无值 → `value:"—"`、`valueNum:null`、`status:"unknown"`**；保留 `gmMap.*`（真 SQL）通路不动。
- 依据：PRD :263「不得用假数据或前端拼算」。
- 预估：净减约 45–60 行（实施时实测）。

### P0-2 无数据不得算作 0 分/高危（F7）

- 现状：`computeCategoryScores`（`:291-326`）中 `d.total>0` 但 `normal=warn=danger=0` → `score=0` → `severity="danger"`；`:328` 的 `overallScore` 分母 `totalAll` 含 unknown，被无数据压低。
- 改法：引入 `measurable = normal + warn + danger`；`measurable === 0` → 分类 `score:"—"`、`severity:"unknown"`；总评分分母改用可测量项数，全无数据时总数显示 "—"。
- 依据：PRD :756「不能用 0% 代替」、:763「0% 要区分真实 0 和无数据」。
- **关键理由**：若只做 P0-1 不做 P0-2，摘掉假数后雷达页会立刻大面积显示"0 分 / 高危风险"——那是**另一种假警报**，比原问题更糟。
- 预估：净增约 8–12 行。

### P0-3 修正 fetch 错误被吞成空态（F8）

- 现状：`metrics-radar.js:108-109` 的 `.catch()` 返回 `{items:[]}` / `{metrics:[]}`。
- 改法：区分三态——`error`（请求失败，显示错误态 + 可重试）/ `empty`（成功但无数据，显示"暂无数据"）/ `data`。错误态不得与空态共用渲染。
- 依据：`docs/engineering/frontend.md:24-27`。
- 建议：请求改用 base 布局已全局加载的 `window.appFetch`（`layout/base.html:78`），并按 `shared-frontend.md:50` 的 CAUTION **显式检查 `resp.ok`**，不得假定其自动解析或抛错。
- 预估：净增约 10–15 行。

### P0-4 清理模板首屏硬编码数字（F6）

- 现状：见 F6 所列行。
- 改法：初始渲染一律置为 `—` / 空占位（如 `kpiScore` → `—`、摘要卡 → `—`、时间徽标 → 由 JS 按实际快照时间填充；月份 option 保留但不得暗示已有当月数据）。**JS 未跑完或抛错时，页面不得出现任何具体数字。**
- 注意：`radar.html:240-253` 下钻舱默认写死"交付周期"及其公式 → 改为空态提示（如"请选择指标"），不得预置某条指标。
- 依据：PRD :263；`frontend.md:24`（初始不得展示未验证数据）。
- 预估：模板 4 文件合计净持平（占位替换，行数不变）。

### P0-5 manage 页伪成功措辞（F9）—— 与雷达页性质不同

- **复核结论**：`metrics-manage.js:317-358 renderRow()` 的列为 名称 / 分类 / 数据源 / 目标 / 阈值 / 责任角色 / 方向 / 状态（启用中·已停用）/ 操作，**不含"当前值"列**，`defaultMetrics`（`:30`）也无 `value` 字段 → **该页不展示指标数值，不存在"假数值"问题**。其问题仅为：配置只写 localStorage、toast 谎称"已生效"。
- 改法（方案 a，本 P0 采用）：
  - `:508` toast 文案改为「已保存到本机草稿（仅本机可见，未影响雷达/列表口径）」；
  - 页内新增常驻提示条，明示"指标元数据配置尚未接入后端，本页为原型；保存不影响任何计算口径"；
  - 文案不得再出现"生效"。
- 依据：AGENTS §12（不得声称未达成的能力）；PRD :263 精神。
- 方案 b（接通后端持久化）**明确不在 P0**，属第二批（`zt_wb_*` 字典表 + 写端点）。
- 预估：净增约 5–10 行（提示条）。

### P0-6 口径标注（避免新增误导）

- 现状：即使接通真值，`/metrics/api` 是**全仓口径**，`/board/group/metrics` 是**小组口径**，两者在同一屏混排且无标注。
- 改法：为每个数据格子标注口径来源（`全仓` / `选定小组` / `暂无数据`）。若某指标在某视图下无对应口径数据，一律 "—"，**不得跨口径借用**。
- 依据：PRD :234「切换小组后…禁止跨组串数据」。
- 预估：净增约 5–8 行。

### P0-7（需裁决，见 §5-c）浏览器端评分归属

- 现状：雷达页 5 分类得分与总评**全部由浏览器计算**（`metrics-radar.js:291-334`），后端 `Service.Radar` 有等价实现却未被调用。
- 冲突点：`docs/engineering/frontend.md:3-5`「Browser JavaScript… MUST NOT be the only implementation of… **totals**」；当前**页面展示的 totals 只来自 JS**。
- 选项：
  - **(a) 仅记录**：P0 保留 JS 聚合（因其输入已改为真实 status），把"改由 `/metrics/radar/data` 渲染"列入 R3。代价：暂时保留一处规范偏离。
  - **(b) P0 内接线**：改调已存在的 `/metrics/radar/data`（零后端改动）。代价：该接口产出的是"5 分类 KPI + 异常项"，与页面"13 项明细 + 前端聚合"结构不匹配，需重写渲染逻辑，**超出"最小修复"**，且牵动 §5-a 的字典收敛问题。
- **建议 (a)**，并在文档中显式记录该偏离及其 R3 归属。

---

## 5. 需要你裁决的 3 个边界问题

**a. 雷达页的指标字典要不要收敛到后端 12 条？**
三份字典并存：后端 `service.go` 12 条、雷达页 13 条、manage 页 14 条，两两不相交或仅 4 条同名。
建议：**P0 不收敛**。收敛=产品可见的内容变更（大屏会换一批指标），属 R3，须先定"哪个是唯一真源"。
但 P0-2/P0-6 必须让页面在该字典下**诚实**（无口径就 "—"）。

**b. manage 页"保存"要不要 P0 就落库？**
建议：**不**。落库需要 `zt_wb_*` 表 + 写端点，属 PRD 第二批（:805 标注"未实现"），且会引入 CSRF/JSON/Capability perm 一整套（AGENTS §4）。P0 只改措辞。

**c. P0-7 选 (a) 还是 (b)？** 见上，建议 (a)。

---

## 6. 验收（缺一不可）

**功能**
1. `/metrics/radar` 在「全部敏捷小组（汇总）」下：凡无真实来源的格子显示 `—` + 「暂无数据」，**不出现任何具体数字**。
2. 无数据的分类卡显示 `—` + 「暂无数据」，**不出现 0 分 / 高危风险**。
3. 选定具体敏捷小组后，有小组口径真值的指标（`delivery`/`implement`/`overIteration`/`unscheduled`/`onlineDelay` 等）显示真实数值；无小组口径的显示 `—`（不跨口径借用）。
4. 断开 `/metrics/api` 或伪造 500/403 时，页面显示**错误态**而非空态；不得残留 stale 行。
5. `/metrics/manage`：清空浏览器本地存储后首次加载显示 14 项元数据；点击保存后 toast **不再声称"生效"**；页内可见"未接后端"提示。
6. JS 未执行（禁用 JS）时，首屏**不出现** 96 分、14 条 等任何硬编码数字。

**门禁**
7. `git diff --stat` 仅含上述 4 文件；`internal/**` 0 改动；无新增依赖。
8. 两 JS 文件改动后行数**净减少**（实测值须写入提交说明）。
9. `git diff --check` exit 0；`make check-patterns check-secrets` 无新增项。
10. 长度门禁：本任务**不接受**把超限文件加入基线；门禁若仍红，须在报告中写明是既有基线失败（`761`/`618` 等）而非本任务引入。

**浏览器（AGENTS §9 / `.cursor/rules/frontend.mdc`）**
11. 真实登录态浏览器交互验证，**light 与 dark 双主题**；记录 Console 报错与 network 请求结果、加载资源清单。
12. 触及的空态/错误态样式必须使用语义 token；**不得新增硬编码颜色**（含 `metrics-manage.js:321` 的 `var(--po-blue, #2563eb)` 这类 fallback——若该处被改动，须改为语义 token）。

---

## 7. 明确不在本任务范围

- 建 `zt_wb_*` 指标字典表、写端点（POST/PUT/DELETE）、CSRF/JSON/Capability 接入。
- 收敛三份指标字典、统一 code 命名。
- 让 `/metrics/radar/data` 成为雷达页唯一数据源。
- 拆分 `metrics-radar.js`（708 行）/ `metrics-manage.js`（618 行）至 500 行以下。
- 打通"规范执行"类指标（`norm.*` 依赖门禁数据，后端现返回 `—`+unknown，属真无数据，无需改）。
- 任何 Go / DB / 路由 / 权限改动。

---

## 8. 风险

| 风险 | 说明 | 缓解 |
|---|---|---|
| **页面"变空"引发误判** | 摘掉 seed 后，汇总视图下雷达页会大面积显示 `—`，外观上像"改坏了" | 这是 PRD :263 要求的诚实状态；**实施前先与 PMO 对齐预期**，并在页面用醒目空态文案说明"数据源未落地"，而非默默留白 |
| 规范冲突（P0-7） | JS 仍承担 totals 计算，偏离 `frontend.md:3-5` | 采用选项 (a) 并在文档显式记录偏离 + R3 归属；不得静默忽略 |
| 空态样式引入硬编码色 | 新增空态易顺手写死颜色 | 强制用语义 token；验收第 12 条把关 |
| 门禁既红被误归因 | 长度门禁当前即红，易被算在本任务头上 | 验收第 10 条要求明确区分既有失败与本任务引入 |
| 行数反弹 | 两文件已超限，改动若净增会加重债务 | P0-1 净减、P0-2/3/5/6 净增，须保证**合计净减**；实施后实测并记录 |

---

## 9. 建议实施顺序

P0-1（删假算）→ P0-2（修 0 分高危）→ P0-4（清模板首屏）→ P0-3（错误态）→ P0-6（口径标注）→ P0-5（manage 措辞）。
理由：先切断假数据来源，再修正由此暴露的显示逻辑缺陷，最后处理措辞与标注类文案；P0-5 独立且风险最低，可并行。
