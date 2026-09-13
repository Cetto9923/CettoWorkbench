# 前端 HTML 转义命名约定（决策 3 结论）

- 日期：2026-09-13
- 分支：`release/po-integrate-main-202609`
- 状态：**约定已定**；§5 的 ①②③④ **已于 2026-09-13 全部执行完毕**（含最初因位于他人未提交
  hunk 内而暂缓、后经授权清零的 2 处 fail-open，见 §5.3）。§6 口径已定为 A。本文件尚未提升为
  `docs/engineering/` 的绑定规则
- 前置：`docs/plan/code-health-20260913/README.md`（收口执行记录）、`slim-plan-2026-09-13.html`（决策 3）

## 0. 一句话规则

> **HTML 转义只有一个真源：`window.escapeHtml`——capability-level canonical source，
> 当前实现由 `layout/base.html` 全站加载的 `ui.js` 提供。
> 普通模块禁止重新定义 `escapeHtml`；已经存在且存在真实外部消费者的兼容 API，
> 可保留一行薄委托到 `window.escapeHtml`。不新增工具层，不移动真源。**

"一行薄委托"的充要条件：函数体只有 `return window.escapeHtml(x);`——
**不含算法、不含 `A || B || …` 探测链、不含任何返回未转义原串的兜底**：

```js
// 模块内已经存在的兼容 API（有真实外部消费者）——允许
function escapeHtml(value) { return window.escapeHtml(value); }
function esc(value) { return window.escapeHtml(value); }
```

判定依据是"**是否重新定义算法/真源**"，不是"名字叫什么"：`escapeHtml` 与 `esc` 同为
兼容 API 名，两者都允许作薄委托，两者也都禁止携带算法或兜底链。

`ui.js` 是真源的**当前存放处**，不是判断依据（`ui.js` 整体并非 global golden reference，见 §6）。

## 1. 为什么是这个名字——三条独立证据指向同一结论

### 证据 A：主干（Main）的既有事实

`/Users/yuyan9923/GitHub/workbench`，`main` @ `b61c5c3c`：

| 项 | 事实 |
|---|---|
| 全局转义入口 | `web/static/js/ui.js:10` 定义 `escapeHtml`，`ui.js:730` 导出 `window.escapeHtml` |
| `base.html` 加载 | `ui.js`（L72）→ `app.js`（L73）→ `components/components.js`（L74）→ … |
| 同名导出覆盖 | `ui.js:731/733` 与 `app.js:404/406` 都导出 `getCsrfToken` / `closeModal`；`app.js` 后加载，**实际生效的是 `app.js` 版本**。`escapeHtml`、`showToast` 无此冲突，`ui.js` 是唯一提供方 |
| 本地拷贝 | 3 处，**名字全部是 `escapeHtml`**：`po/home.js:33`、`schedule/scheduetaskmodal.js:30`、`schedule/scheduetasklistmodal.js:12` |
| `function esc(` | **不存在**——`esc` 短名是 release 分支引入的，不是主干遗产 |
| 前端布局 | 扁平 `web/static/{css,img,js,uploads,vendor}`，**没有** `workbench/shared/` 分层 |

结论：主干对"转义叫什么"没有书面规定，但**代码事实只有 `escapeHtml` 一个名字**。

### 证据 B：你项目自己的门禁已经把"本地重复定义"登记为待收敛项

`scripts/check-patterns.sh:52`：

```bash
scan advisory LOCAL_ESCAPE_HTML 'function[[:space:]]+escapeHtml[[:space:]]*\(' "${sources[@]}"
```

即"本地出现 `function escapeHtml(`"本身就是一条 advisory。这条规则隐含了"真源不在本地"的前提，
与本约定一致。**但它只匹配 `escapeHtml`，不匹配 `esc`**——所以 9 处 `function esc(` 与
24 处 `esc = ` 兜底链当时完全在门禁盲区里（见 §4）。该盲区已于 §5.4 用两条**名字无关**的
fail-open 形态 advisory 收口；`function esc(` 本身**不设为**违规（合规薄委托会误报）。

### 证据 C：本仓 2026-09-13 已经就同一问题做过决定

`docs/plan/code-health-20260913/README.md:36-37`（原文）：

> 发现：生产 wb-picker.js、wb-person-picker.js 调用 WBUtils，但仓库没有实现/加载入口。
> 用户明确授权后，改用 layout/base.html 已加载的 `window.escapeHtml`，数组处理使用
> `Array.isArray`；**不引入新的工具层**。

这就是本约定的先例：真源 = `window.escapeHtml`，且**明确否决**引入新工具层。
本约定只是把这条已生效的口径从 picker 两文件推广到全前端。

## 2. 为什么不照抄 CRCBWorkbench 的 `WBUtils`

`CRCBWorkbench`（`po-v2`）是**另一套血缘**，它把转义收敛进共享层：

- `web/static/workbench/shared/workbench-utils.js`（72 行）→ `WBUtils.escapeHtml`
- 文件头自述：「算法真源 (escapeHtml) 自 `po-core.js` `poEscapeHtml` 迁移至此」
- 依据：`docs/audit/code-shrink-r1b1-pure-helper-20260828.md`——把 30 处实现收敛为单源，
  原函数名保留为**薄委托 compat alias**；并如实标注"净 LOC ≈ −1，收益在一致性而非行数"
- 状态：`docs/audit/code-shrink-r3b-shared-ui-components-20260828.md` 记为 `CANONICAL_ALREADY_SHARED`

**可以借鉴的是结论（单源 + 薄委托），不能照抄的是结构**：CRCBWorkbench 的前端是
`web/static/workbench/shared/` 分层，本仓与主干都是扁平 `web/static/js/`。
在本仓新造一个 `workbench-utils.js` 就是"引入新的工具层"，与证据 C 的既定否决项冲突。

**正确姿势**：让 `ui.js` 承担 `workbench-utils.js` 的角色。它已经在承担了。

## 3. 改造前实测快照（release 分支，2026-09-13）

> 本节是**改造前**的形态清点，保留作为改造前后对比与"按名字枚举为何不够"的证据。
> 收敛后的状态见 §5。

| 形态 | 处数 | 说明 |
|---|---:|---|
| `function escapeHtml(` 定义 | **5** | `ui.js:10`（真源）、`po/personal-list.js:15`、`permission/members.js:56`、`schedule/scheduetaskmodal.js:30`、`schedule/scheduetasklistmodal.js:12` |
| `function esc(` 定义 | **9** | `agileteam/agileteam.js`、`agileteam/agileteam-members.js`、`agileteam/agileteam-detail.js`、`picker/wb-picker.js`、`picker/adapters/wb-person-picker.js`、`po/follow-drawer.js`、`po/workboard-issue.js`、`po/schedule-link.js`、`po/demand-detail-richtext.js` |
| `var esc = <兜底链>` | **24** | 见下 |
| `esc(` 真实调用点 | ≈ **661** | 改写量在此，不在定义处 |

### 3.1 兜底链的真实形态（这才是"防御性编程"的准确描述）

24 处 `var esc = …` 的典型写法：

```js
// po/notice.js:4
var esc = (window.PersonalList && window.PersonalList.escapeHtml) || function (v) { return String(v == null ? "" : v); };

// po/urge.js:25
var esc = (window.PersonalList && window.PersonalList.escapeHtml) || window.escapeHtml || function (s) { return String(s == null ? "" : s); };
```

两个必须记录的事实：

1. **只有 3 / 24 条链把 `window.escapeHtml` 作为兜底级别**。另外 **21 条不含它**：
   18 条只依赖 `window.PersonalList.escapeHtml`，其余依赖 `RichText.esc` / `WB.esc` / `PL.escapeHtml`。
   即真源被**绕过**而不是被复用。
2. **兜底链的末级是 fail-open**：`function (v) { return String(v == null ? "" : v); }`
   在真源缺失时返回**未转义的原串**。这不是健壮性，是安全缺陷——真源一旦缺位，
   该模块变成 HTML 注入面。

> 修正 `slim-plan-2026-09-13.html` 的口径：原文"27 份 `esc()`"应读作
> "**9 份算法拷贝 + 24 条兜底链**"。数量级接近，但性质不同：9 份是真重复，
> 24 条是"绕开真源的防御性包装"。

### 3.2 `PersonalList.escapeHtml`：改造前是"方向正确但兜底 fail-open"

```js
// po/personal-list.js:15（改造前的工作区状态）
function escapeHtml(value) {
  if (typeof window !== "undefined" && typeof window.escapeHtml === "function") {
    return window.escapeHtml(value);
  }
  return String(value == null ? "" : value);
}
```

它委托到真源，方向正确。问题只在它的**兜底同样 fail-open**，以及 19 个模块把它当成
唯一入口从而制造出单点依赖。`permission/members.js:56` 同理
（`window.escapeHtml ? window.escapeHtml(s) : String(s == null ? '' : s)`）。

> **后续（2026-09-13 Last Mile）**：这两处已获授权清零，现为
> `function escapeHtml(value) { return window.escapeHtml(value); }`（见 §5.3）。
> 该收敛使 3 个夹具暴露缺真源桩的旧债，已补齐（见 §5.2）。

## 4. 判定表

| 情形 | 判定 | 依据 |
|---|---|---|
| 模块需要转义 | 调用 `window.escapeHtml` | 证据 A / C |
| 普通模块**新**定义 `escapeHtml`（含算法） | **禁止** | `check-patterns.sh` `LOCAL_ESCAPE_HTML` 已登记 |
| 模块内**已存在**的兼容 API（`escapeHtml` 或 `esc`）保留**一行薄委托** | **允许** | 真实外部消费者已存在（`PersonalList.escapeHtml` 有 19 处消费点） |
| 在模块内定义 `function esc(` / `function escapeHtml(` 且内含 `.replace(...)` 算法体 | **禁止** | 第二份算法真源 |
| `A \|\| B \|\| fallback` 式多级兜底链 | **禁止** | 真源唯一，无需探测；且末级 fail-open |
| 兜底返回未转义原串 | **禁止** | 安全缺陷，非健壮性 |
| 新建 `workbench-utils.js` / `WBUtils` 之类的工具层 | **禁止** | 证据 C 明确否决；主干无此结构 |
| 把 `escapeHtml` 从 `ui.js` 搬到 `app.js` | **禁止** | 2026-09-13 裁决采用口径 A（见 §6） |
| 把 `escapeHtml` 与 `escapeJsString` 合并 | **禁止** | 字符集不同（JS 词法 vs HTML 元字符），r1b1 §3.1 已判定 |

## 5. 落地执行记录（2026-09-13 已执行）

### 5.1 已执行：①②③ 合并为一次改造，33 处供给点全部收敛

| 步骤 | 内容 | 结果 |
|---|---|---|
| ① | 消除 fail-open 末级 | **根除**（链式 34/34 清零，见 5.4 实测） |
| ② | 9 处 `function esc(` 算法体改薄委托 | 完成 |
| ③ | 消掉 24 条 `A \|\| B \|\| …` 兜底链 | 完成，统一为一行委托 |
| ③′ | 补漏：**按名字枚举漏掉的第 34 处** | `po/linkstory.js:287`（原名 `var escapeHtml =`，不是 `esc`） |

收敛后的两种统一形态：

```js
// 以 PersonalList 为第一顺位
var esc = (window.PersonalList && window.PersonalList.escapeHtml) || function (s) { return window.escapeHtml(s); };

// 以 PL（= window.PersonalList）为第一顺位
var esc = PL.escapeHtml || function (v) { return window.escapeHtml(v); };
```

要点：末级从「返回未转义原串」改为「委托真源」，真源缺失时**抛 `TypeError`（fail-closed）**，
错误信息直指真源；不再存在静默降级路径。

**验收证据（实测，非估算）**

| 门禁 | 结果 |
|---|---|
| `node --check` | 27 + 7 个改动文件通过 |
| `tests/unit/frontend/*.test.js` | 27 通过 / 1 既有失败（`workboard-render.test.js`，与本改造无关，且不在 `make check-frontend-test` 清单内） |
| `make check-frontend-test` | **passed** |
| `go vet` / `go test ./...` | **passed** / **exit 0** |
| `check-architecture.sh` | **passed**（existing debt 1 file） |
| `check-gofmt` / `check-local-artifacts` / `check-whitespace` | passed |
| `check-patterns.sh` | hard gate **passed**；advisory 183 条 / 156 条新增，**全部为存量基线漂移**（新增行经全模式正则核验零命中） |
| `check-file-length.sh` | **失败 21 项 = `BLOCKED BY EXISTING BASELINE`**，改造未使其恶化（改动文件行数相对 HEAD 均为 ≤0，唯一 +11 的 `metrics-manage.js` 来自并行 Agent 的 +17） |
| 净行数 | 约 −58 行（第一遍）+ 第二遍小幅回填 |

### 5.2 改造暴露的真问题：测试夹具一直在"靠 fail-open 通过"

`done-status-labels.test.js`、`po-deliver.test.js`（原为完全空的 `mockWindow = {}`）、
`urge-modal.test.js` 的合成 `window` **从不提供 `window.escapeHtml` / `PersonalList.escapeHtml`**。
它们过去通过，正是因为 fail-open 静默返回了原串。已按 production 加载顺序补真源桩。

`personal-list.js` 收敛后又有 3 个夹具同时暴露同一旧债（它们加载 `personal-list.js` 的
`objectTypeBadge` / `idChipHtml`，而这两个导出内部调用 `escapeHtml`）：
`priority-helpers.test.js`、`render-row-isolated.test.js`、`unified-object-id-ui.test.js`
——同样补真源桩（共 6 个夹具）。

> 教训：夹具不提供真源 + 实现 fail-open = 测试与生产行为同时失真，且互相掩盖。
> 只要实现里还留着"真源缺失就返回原串"的分支，夹具缺真源这个错误就永远暴露不出来。

### 5.3 最后 2 处 fail-open：已获授权并清零（Last Mile）

| 位置 | 收敛前 | 收敛后 |
|---|---|---|
| `po/personal-list.js` | `if` 块守护式 + `return String(value == null ? "" : value);` | `function escapeHtml(value) { return window.escapeHtml(value); }` |
| `permission/members.js` | 三元守护式 + `String(s == null ? '' : s)` | `function escapeHtml(s) { return window.escapeHtml(s); }` |

两处都是 **19 条链的第一顺位提供方**（`PersonalList.escapeHtml`），是 `ui.js` 之外承重最大的转义入口。

落地纪律：**只改 `escapeHtml` 对应的最小 hunk**。`permission/members.js` 同文件另有
`apiHeaders()` 的 CSRF 改动（他人 hunk）——已核对 `git diff` 确认**原样未动**；
`personal-list.js` 全文件只有这一个 hunk。

> 首次发现时暂缓的原因是多 Agent 纪律（不覆盖他人在飞 hunk）；本轮取得显式授权后清零。
> 副作用：`personal-list.js` 的收敛使 3 个夹具的旧债暴露（它们从不提供真源桩），已一并补齐（见 §5.2）。

### 5.4 第 ④ 步（门禁 advisory）：已加两条棘轮，当前命中 0

原拟两条正则**都会出错**，实测结论：

1. `LOCAL_ESCAPE_SHORT 'function[[:space:]]+esc[[:space:]]*\('` —— **误报**。收敛后留下的
   `function esc(s) { return window.escapeHtml(s); }` 正是约定允许的"一行薄委托"，
   用它当告警会把**合规形态**当成债务。
2. `ESCAPE_FAIL_OPEN 'esc[[:space:]]*=.*\|\|…'` —— **漏报**。被漏掉的 3 处
   （`po/linkstory.js`、`po/personal-list.js`、`permission/members.js`）**名字全都不叫 `esc`**。
   按名字枚举是漏检的根因。

**最终按"缺陷形态"匹配（名字无关），已加入 `scripts/check-patterns.sh`**：

```bash
# 链式 fail-open 末级
scan advisory ESCAPE_FAIL_OPEN_CHAIN '\|\| *function *\([a-zA-Z_$]*\) *\{ *return String\(' "${sources[@]}"

# 守护式 fail-open 回退（三元写法）
scan advisory ESCAPE_FAIL_OPEN_GUARD '\? *(window\.)?escapeHtml\([^)]*\) *: *String\(' "${sources[@]}"
```

**上棘轮的前置条件已满足**：两条规则当前命中数均为 **0**（先清零、再从 0 起步防新增）。

已知局限（**本轮明确不修**）：曾存在于 `personal-list.js` 的 `if (...) { … }` + 裸
`return String(…)` 跨行写法，单一 POSIX ERE 抓不全。不为此引入复杂 multiline shell regex；
该形态经全量扫描确认已不再存在于代码中。

**明确不加**：`function esc(` 的告警（本节第 1 条已论证会误报合规薄委托）。

**不做**：不加 `function esc(` 的告警（§5.4 第 1 条已论证会误报合规的薄委托）。


## 6. 文档口径：已采用口径 A（2026-09-13 裁决）

`docs/engineering/shared-frontend.md` 声明 `ui.js` 是 legacy、**未**被认证为 global golden
reference；而本约定的真源恰好由 `ui.js` 提供。两处口径已按**口径 A** 对齐：

> **`window.escapeHtml` 是 capability-level canonical source，当前实现由 `ui.js` 提供。
> 这不代表 `ui.js` 整个文件成为 global golden reference。**

即把"真源"限定为**能力级**而非**文件级**：`shared-frontend.md` 对 `ui.js` 的整体判断继续成立
（它不对 `ui.js` 的其它能力作背书）。

**已否决口径 B**：禁止把 `escapeHtml` 从 `ui.js` 搬到 `app.js`。理由：纯结构洁净收益，
却要改 base 加载链上的两份文件、需要 UI 回归，且与本裁决的能力级定位冲突。

无论哪种口径，**都不应**新建工具层（见 §2）。

## 7. 明确不做

- 不为转义新建共享层/工具层/打包入口。
- 不批量重写 661 个调用点（成本远大于收益；定义处收敛即可让调用点行为一致）。
- 不触碰 `escapeJsString`（PO 词法专用，见 r1b1 §3.1）。
- 不改 `ui.js` 的 `escapeHtml` 算法（5 元字符 `& < > " '`，与主干、与 `WBUtils` 一致）。
- 不在本轮提升为 `docs/engineering/frontend.md` 绑定规则——需你确认后再提升。
