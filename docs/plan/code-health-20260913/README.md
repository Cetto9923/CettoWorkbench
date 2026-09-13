# 代码健康治理执行记录

状态：PARTIAL。本文记录实际执行，不代表三批计划全部完成。

## 范围与快照

- 用户依据：2026-09-13 确认的行为等价治理计划；随后明确允许最小修复 picker 缺失依赖并补回归。
- 分支：`release/po-integrate-main-202609`。
- HEAD：`3507193e9970dfefd430b8dbf22b079e1c354193`。
- 未提交、未推送、未部署；不修改 API、权限、数据库及正式业务数据。
- 原有 WIP：`variables.css`、`wb-priority.css`；本任务未修改这两份文件。
- 原有本地计划、`.hermes-cache/`、根目录审计 HTML、Demo 与其他任务文档保留。
- 已读规范：agent-onboarding、architecture、database、frontend、testing、quality；上下文恢复后再次读取 onboarding。

初始主题文件 SHA256：

```text
a6384b3502692e622bcb44f199394984134155943c10320d3d965fde6bd1a785  web/static/css/layout/variables.css
183b7660cceb7869692fcdebfdcf02b15efb9556c12a53be5913731a213b53c8  web/static/css/po/wb-priority.css
```

## 第一批已落地差异

- 修正 debug、encode 文件头、BatchPage 注释、主操作路由/权限来源说明。
- 移除 BaseModel 已完成 TODO/注释字段、导出旧调用、认证旧菜单过滤注释；当前菜单行为不变。
- 修正 done.css 绝对“无消费者”描述：共享 helper 为主，done.js 仍有 fallback。
- 删除 PO 零引用声明：TimeRangeThisWeek、TimeRangeThisMonth、两个旧错误变量、KeyEdit、KeyPublish、KeyViewEvaluate、ObjectSubDemand、inputShape。
- 删除 `po_work_scope.go`：POWorkScope、POWorkScopeDemandIDs、poworkScopeSQL、DemandScopeRole 全仓生产/测试查询仅指向该文件内部；无接口/构建注册入口。删除后相关包与全库 Go 测试通过。
- 历史语义来源保留：该文件描述 2026-08-30 冻结的“六个 owner 字段或关注、叶需求去重、不含历史参与”独立 scope。它不再是当前页面查询入口，本轮未重新接入。
- 可复核删除证据：`git show HEAD:internal/module/po/po_work_scope.go`，对其中声明执行 `rg -n 'POWorkScope|POWorkScopeDemandIDs|poworkScopeSQL|DemandScopeRole' internal tests`，并检查 `git diff`。历史版本仍可由 Git 恢复。

第一批未声明验收通过：完整候选逐簇核查与交付审查尚未收口。

## Picker 最小修复和回归

发现：生产 wb-picker.js、wb-person-picker.js 调用 WBUtils，但仓库没有实现/加载入口。
用户明确授权后，改用 layout/base.html 已加载的 `window.escapeHtml`，数组处理使用 `Array.isArray`；不引入新的工具层。

已迁移并接入 `make check-frontend-test`：

| 原静态目录测试 | 新目录测试 | 保留/新增证据 |
|---|---|---|
| picker/wb-picker-core.regression.test.js | tests/unit/frontend/wb-picker-core.regression.test.js | 加载真实 autocomplete-options → ui → picker 链；搜索、选择、清空、portal、头像、副行、隐藏 account、change 次数、setValue、同名/未知值不猜测 |
| picker/wb-picker-search.regression.test.js | tests/unit/frontend/wb-picker-search.regression.test.js | 真实拼音 vendor 与 WbPickerSearch；中文、工号、全拼、首字母、多 token AND、服务端拼音 |

测试纠偏：

- 搜索测试原先复制算法，现直接执行生产搜索模块。
- 原 uniqueResolve 的“同名/未知返回空”是测试自身算法，非生产契约；生产保留原始值而不猜账号。已用真实 adapter 覆盖同名、未知值及精确 account，未改生产兼容行为。
- 原头像断言检查 host 而非 portal，存在空集合假通过；现要求实际选项非空再校验。
- 原 DOM stub 对 `.wb-picker-option` 重复收集同一节点，导致测试重复注册点击回调；修正 stub 后验证一次选择仅一次 change。

其余两份 phase-a / agileteam 测试尚未迁移；失效断言映射仍待完成。

## 测试执行

| 命令 | 结果 |
|---|---|
| go test ./internal/module/po/... ./internal/middleware ./internal/pkg/fileexport ./internal/module/user ./internal/module/debug ./internal/pkg/encode | exit 0 |
| make check | exit 2；Go 全库测试、前端原入口、vet、whitespace 通过；长度扫描 awk 打开工作树已删除但 index 仍记录的 po_work_scope.go 失败 |
| node tests/unit/frontend/wb-picker-core.regression.test.js | exit 0（含新增交互/隐藏值/歧义值断言） |
| node tests/unit/frontend/wb-picker-search.regression.test.js | exit 0 |
| make check-frontend-test | exit 0（含两份迁移测试） |
| bash scripts/test-quality-gates.sh | exit 0；36 项通过 |
| make check-patterns check-secrets check-architecture | exit 0；hard pattern 0 既有项，秘密指纹 0（不输出值），architecture 1 既有债务；advisory 未升级阻断 |
| git diff --check | exit 0 |
| shasum -a 256 两份主题 WIP | 与初始哈希一致 |

未削弱扫描器、未扩展基线，也没有为了扫描通过而暂存删除。
长度扫描当前提前失败不能解释成“仅 Demo 阻断”：生产超限仍未清零。
原有 Demo/审计原件导致的门禁债务仍为 BLOCKED BY EXISTING BASELINE。

## 未完成与后续专项

- 第二批其余两份测试迁移、旧断言逐项映射、剩余 upload/URL/排期/敏捷团队死代码逐簇核查。
- 第三批生产文件按职责拆分尚未实施；没有拆分映射或视觉等价通过结论。
- 浅深色、四档宽度、加载/空态/错误/弹窗、Console/network/资产版本及隔离环境写入回归尚未完成。
- 保持独立专项：MD5 兼容迁移、删除保护与并发、菜单/路由/限流、指标下沉/错误语义/a11y、颜色 token/innerHTML/Observer/跨模块公共层。

最终完成条件仍是生产违规清零且所需回归通过；当前不可称为全计划完成或全库门禁恢复。

## `docs/Demo` 迁出仓库（决策 2 执行，2026-09-13）

用户决策：迁至 `/Users/yuyan9923/GitHub/CRCBWorkbench/docs`；若 Main 有先例则对齐 Main 结构。

Main 先例（`/Users/yuyan9923/GitHub/workbench`，`main` @ `b61c5c3c`）：主干**不跟踪** `docs/Demo`
（`git ls-files docs/Demo` = 0，本地仅 1 个未跟踪的 `metrics-redesign-preview.html`），
`docs/` 下被跟踪的只有 `demo-analysis.md` 与 `design/schedule-biz-demands.md`。
主干 `.gitignore` 另有 `/demo`、`/demo2`、`/docs/demo-analysis.md`、`/docs/design`
（大文件本地源不入库的既有口径）。本次迁出与该结构对齐。

执行与证据：

| 项 | 结果 |
|---|---|
| 目标 | `/Users/yuyan9923/GitHub/CRCBWorkbench/docs/Demo` |
| 内容校验 | **175 / 175 文件 SHA-256 全等**；字节总量两端均为 **8,040,314** |
| 源侧移除 | 工作树 `docs/Demo` 已删除，物理文件 0；`git status` 显示 169 条 ` D` |
| 可回滚性 | `HEAD` 仍保留 169 个 `docs/Demo/*` 条目（`git checkout HEAD -- docs/Demo` 可恢复） |
| 安全备份 | `/tmp/docs-Demo-backup-20260913.tar.gz`（2.6M，含 175 文件） |
| 原入库状态 | 169 跟踪 + 6 未跟踪（6 个 `*-redesign*.html` 本地设计稿**无 git 历史**，先复制后删除） |

门禁影响：**零**。已逐一核实——

- `scripts/check-file-length.sh` 的 `case` 白名单仅含 `cmd/ internal/ web/templates/
  web/static/js/ web/static/css/ tests/`，`docs/` 整体不在扫描范围；
- `scripts/check-patterns.sh:63` 的 `docs/Demo/*` 跳过项本身已是冗余（该处 `sources[]`
  只从 `internal/ web/static/ web/templates/` 取值）；
- `scripts/quality-baseline/{file-length,patterns}.tsv` 中 `docs/Demo` 条目数均为 **0**；
- `scripts/test-quality-gates.sh:110,122` 的 `docs/Demo/host.html` 是自检夹具，
  运行在 `mktemp -d` 隔离仓库内并由 `trap` 清理，与仓库内容无关。

配套变更：

- `.gitignore`：新增 `/docs/Demo/`（沿用同文件 `/docs/PRD/`、`/docs/design` 的既有口径）。
- `docs/engineering/debt.md`：原第 4 条 "`docs/Demo/` artifacts (≈22)" 记载的是
  "脚本未过滤地扫描 Demo 树"，与当前脚本的实际范围不符，已改写为 resolved 并注明两点原因。

本轮**未** commit、未 push、未 `git add`。

## 决策 3 结论（前端转义命名）

见本目录 `frontend-escape-html-convention.md`。结论：名字统一为 `escapeHtml`，
真源为 `ui.js` 的 `window.escapeHtml`；`esc` 仅允许作一行薄委托；禁止兜底链与 fail-open 末级。

## 行为等价清理（本轮执行，2026-09-13）

状态：PARTIAL（`scripts/check-file-length.sh` 预期失败，不得称为可交付）。

范围（仅本轮）：

1. 删除 `internal/module/po/po_work_scope.go`；`rg` 确认 `POWorkScope|POWorkScopeDemandIDs|DemandScopeRole|poworkScopeSQL` 在 `internal`/`tests` 零引用；`go test ./internal/module/po/...` 通过。
2. 移除 `go.mod` 的 `replace workbench => /home/wds/repo/workbench`；`go build ./...` 通过。
3. 新增 `internal/pkg/personlabel.Format`；`po.FormatAccountName` / `testtask.FormatPersonName` / `user.formatAccountDisplay` 薄委托；补 suffix-dedupe 包测。
4. 新增 `internal/pkg/demandstage`：`Map`（PO，developing→「提测」）与 `Label`（query，developing→「研发中」）**有意保持文案分歧**；query↔po 无互相 import。
5. 前端窄清理：`members.js`、`personal-list.js` 去掉仅转发的本地 `escapeHtml`，调用点改用 `window.escapeHtml`；未动 schedule 全量拷贝或 fail-open `esc` 链。
6. `docs/engineering/module-index.md` 补录 `agileteam` / `metrics` / `profile` / `query`（SOURCE VERIFIED / SCHEMA UNVERIFIED）。
7. **未**修改 `variables.css` / `wb-priority.css`；**未** commit。

验收命令与退出码（本轮）：

| 命令 | exit |
|---|---|
| `git diff --check` | 0 |
| `go build ./...` | 0 |
| `go test ./internal/module/po/... ./internal/module/testtask/... ./internal/module/user/... ./internal/module/query/... ./internal/pkg/personlabel/... ./internal/pkg/demandstage/...` | 0（user/query 无 `*_test.go`） |
| `make check-frontend-test` | 0 |
| `make check-architecture check-patterns check-secrets` | 0（patterns advisory 154 new，hard-pattern 0；secrets debt 0） |
| `bash scripts/check-file-length.sh` | 1（21 条 over-limit/grew/new；BLOCKED BY EXISTING BASELINE） |

结论：PARTIAL。行为等价清理范围内实现与相关测试通过；文件长度门禁失败，不得称为 done/可交付。

未改计划正文文件本身。
