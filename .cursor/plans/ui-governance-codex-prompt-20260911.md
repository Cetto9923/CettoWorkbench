# Codex 提词：PO 全局设计系统治理补漏（排除交付弹窗）

> 仓库：`workbench-claude-po`
> 分支：`release/po-integrate-main-202609`
> 规范源：`docs/engineering/frontend.md`（Theme Contract）+ `docs/engineering/ui-audit-20260910.md`
> 全局按钮：`web/static/css/components/button-system.css`（已在 `layout/base.html` 引入）
> 全局选人：`window.initUserPicker`（`web/static/js/ui.js`，mode=user 的 autocomplete）
> 本地验：`WORKBENCH_MODE=dev WORKBENCH_APP_ADDR=:8090 WORKBENCH_SESSION_COOKIENAME=wb_session_po_8090`，账号 `<本地测试账号>` / `<本地测试密码>`
> **不要 push**；**不要改 `docs/PRD/`**；**不要动交付弹窗**（`po_deliver_modal.html` / `po-deliver.js` / `po-deliver.css` 由另一条 Codex 任务独占）。
> 本轮目标：补「未用全局设计」的遗漏 + 可安全删的 UI 死代码；**零产品行为变更**（筛选/接口/文案逻辑保持原样，只换视觉实现与选人 API）。

---

## 0. 明确排除（勿改）

- 发起交付弹窗全链路：`web/templates/components/po_deliver_modal.html`、`web/static/js/po/po-deliver.js`、`web/static/css/po/po-deliver.css`
- 催办业务逻辑（`urge.js` 收件人解析、预览文案）——仅允许 CSS 去掉多余 dark 拷贝时触碰 `po-urge.css`
- 禅道 PATH_INFO / 外链新标签 / 星标 CSRF `appFetch` / 提测四步产品流程
- Admin 部门/菜单页的 Bootstrap `ui/action-group`（不是 PO 死代码）
- `.wip/demand-edit/`（WIP 停放，勿删勿提交）

---

## 1. 必做 Track A — 选人统一 UserPicker（小改、优先）

### 现状
澄清页人员已用 `initUserPicker`；排期集成弹窗仍用 `initAutocomplete`：

- `web/static/js/schedule/scheduleintegrated.js`：`scheduleIntRDInput` / `scheduleIntQDInput` / `scheduleIntAccepterInput`
- `web/static/js/schedule/scheduleintegratedshared.js`：`bindUserAutocomplete`（或同等）内部 `initAutocomplete`

### 必做
1. 凡「选人」调用改为 `initUserPicker(...)`；产品/版本/窗口等非人员搜索继续 `initAutocomplete`。
2. `toAutocompleteItems` 产出的 item 形状保持与 UserPicker 兼容（`id`/`name`/`account` 等现有字段）。
3. 若有前端单测覆盖 schedule autocomplete，同步断言改为 UserPicker；没有则不必硬造大测，但需本地打开排期弹窗冒烟。

### 验收 A
- [ ] 代码中排期 RD/QD/验收人三处不再直接 `initAutocomplete` 选人。
- [ ] 排期弹窗可搜索姓名/工号选人；深浅色下拉可用（依赖已有 `autocomplete.css` user mode）。
- [ ] 交付弹窗文件 diff 为空。

---

## 2. 必做 Track B — 澄清弹窗样式并入全局设计

### 现状
`web/static/css/po/demand-clarify.css` 约 **95 处纯硬编码色**（`#fff` / `#2563eb` / `#0f172a`…），深色主题必挂。
Footer 使用私有 `clarify-btn-primary` / `clarify-btn-default` / `clarify-action-btn`，未走 `action-btn` / `button-system`。

### 必做
1. 颜色全部改为 `variables.css` 已有 semantic token（`--color-page-bg`、`--color-surface`、`--color-text-*`、`--color-border*`、`--color-primary*`、`--color-danger` 等）。禁止新增 `--dark-*` 外观命名 token。
2. Footer / 行内操作按钮：模板 `po_clarify_modal.html` 改为与催办一致的 `action-btn` + `action-btn primary`（或 `batch-modal-footer` 模式）；删除或降级为 alias 的 `clarify-btn-*`（若怕大面积 HTML 改，可让 `.clarify-btn-primary` 仅 `@extend`/复刻 `button-system` 的 token 写法，但**优先改 class 名对齐全局**）。
3. **禁止**新增 `html[data-theme="dark"] .clarify-…` 拷贝选择器；深浅差只靠 root token。
4. JS 行为、接口、校验文案不动；UserPicker 绑定保留。

### 验收 B
- [ ] `demand-clarify.css` 中「非 var() 的纯 `#hex`」接近 0（允许 SVG/品牌色极少数例外并注释）。
- [ ] Light / Dark 切换澄清弹窗：背景、输入框、分区标题、主次按钮对比度可读。
- [ ] 保存澄清流程冒烟通过。

---

## 3. 必做 Track C — 提测弹窗样式并入全局设计

### 现状
`web/static/css/po/testtask.css` 私有色板 `--tt-primary/#165dff`、`--tt-page`… 与全局 token 脱节；按钮 `po-testtask-btn` / `is-primary` 自成体系。

### 必做
1. 将 `--tt-*` **映射到**全局语义 token（例如 `--tt-primary: var(--color-primary)`），删除重复 hex；页面内残留 `#fff`/`#f2f3f5` 改为 token。
2. Footer 按钮优先改为 `action-btn` / `action-btn primary`（模板 `po_testtask_modal.html`）；若步进条视觉依赖 `po-testtask-btn`，保留类名但样式体必须 token 化且视觉对齐 button-system（高度/圆角/主色）。
3. 不改四步流程、草稿/提交 API、需求关联逻辑。
4. 禁止新增页面级 dark 拷贝块。

### 验收 C
- [ ] 提测弹窗 Light/Dark 可用；主按钮色与全局 primary 一致。
- [ ] 步进、保存草稿、提交提测冒烟不回归。
- [ ] 交付相关文件无 diff。

---

## 4. 必做 Track D — 去掉六页 `html[data-theme="dark"]` 拷贝（Theme Contract §5）

### 违规文件（按拷贝量）
| 文件 | 约 dark 块数 | 说明 |
|------|-------------|------|
| `web/static/css/po/follow.css` | 46 | 含 `pw-action-btn` dark |
| `web/static/css/po/wb-priority.css` | 25 | 应已有 `--po-pri-*`，覆盖多为冗余 |
| `web/static/css/po/po-urge.css` | 18 | 催办；只删冗余 dark，勿改 JS |
| `web/static/css/po/home.css` | 15 | |
| `web/static/css/po/notice.css` | 12 | |
| `web/static/css/po/done.css` | 5 | |

### 方法（写死）
1. 先读 `web/static/css/layout/variables.css` 现有 light/dark token。
2. 对每个 dark 拷贝块：若只是换背景/文字/边框，**删掉该选择器**，把缺的语义补进 `variables.css` 的 `:root` / `html[data-theme="dark"]` **token 层**（只加语义名，不加 `--dark-bg` 这类）。
3. `wb-priority.css`：确认已用 `--po-pri-*` 后，直接删除其 dark 覆盖块。
4. `follow.css` 的 `pw-action-btn`：样式改为 token；能并入 `action-btn`/`table-action-btn` 则并，否则保留类名但删 dark 拷贝。
5. **不要**为了省事写 `filter: invert` 或全页 brightness。

### 验收 D
- [ ] 上述六文件 `rg 'html\[data-theme=.dark.\]' web/static/css/po/{follow,wb-priority,po-urge,home,notice,done}.css` 结果为 **0**（或仅剩无法语义化且带注释的极少数，需在 PR 说明列出）。
- [ ] `/follow` `/home` `/done` `/notice` 及催办弹窗在 Dark 下无「浅色块/白底残留」。
- [ ] 优先级色语义不变（P1 仍红系、不可变成 brand blue）。

---

## 5. 必做 Track E — 个人资料页按钮泄漏

### 现状
`web/static/css/po/po-profile.css` 定义了**无作用域**的全局 `.action-btn` / `.action-btn.primary`（靛蓝 `#6366f1`），与 `button-system.css` 冲突风险；该 CSS 仅 profile 页引入，但仍是错误模式。

### 必做
1. 删除本地 `.action-btn` 重定义；模板改用全局 `action-btn`（button-system）。
2. 若必须局部微调，选择器必须挂在页面根上，例如 `.po-profile .action-btn` 或 `body…`，且只用 token，**禁止**再写一套 indigo 硬编码主色。
3. `var(--token, #fallback)` 可保留作兜底，但主色 fallback 应对齐全局 primary，不要 `#6366f1` 另起炉灶。

### 验收 E
- [ ] `po-profile.css` 无顶层裸 `.action-btn {`。
- [ ] `/profile` Light/Dark 按钮与全站一致。

---

## 6. 建议 Track F — 小脏点（有余力再做）

1. `web/static/js/po/demand-detail-render-execution.js` 内联 `style="color:#8a99ad…"` → 改为 class + token。
2. 排期 `schedule.css` 里对 `.batch-modal .action-btn` 的重复定义：能删则删，依赖 button-system + `po-workbench-shell .batch-modal-footer .action-btn`；删前 diff 视觉，不确定则本轮跳过并在报告注明。
3. **不要**删除 `web/templates/components/ui/*`（Admin 仍在用）。

---

## 7. 提交与验证

### 建议 commit 切分（便于回滚）
1. `fix(ui): schedule person fields use initUserPicker`
2. `fix(ui): clarify modal tokens + action-btn footer`
3. `fix(ui): testtask map private palette to semantic tokens`
4. `fix(ui): drop page-level dark selector copies (follow/home/…)`
5. `fix(ui): profile stop redefining global action-btn`

### 门禁
- 相关前端单测（若有 user-picker / urge-modal / schedule）通过
- `go test` 被本轮改动波及的包通过（通常无 Go 改动）
- 编译重启 `:8090`，手动切 Dark 冒烟：澄清、提测、排期选人、关注、首页、完成、通知、资料

### 交付回报告知验收方
- 改了哪些文件
- 六页 dark 选择器清零前后计数（`rg -c`）
- 澄清/提测 pure `#hex` 清零前后计数
- **确认交付弹窗三文件无 diff**
- 未做项与原因（若有）

---

## 8. 禁止事项（再强调）

- 禁止借机做产品功能、接口、文案大改
- 禁止新建第二套主题或 `--tt-dark-*` / `--clarify-dark-*`
- 禁止改交付弹窗抢另一条任务
- 禁止 commit `docs/PRD/`；禁止 push
- 禁止删除 `.wip/` 或 Admin `ui/*` 组件

读完本提词后先输出「将改文件清单 + 风险」，再动手；每 Track 做完自检对应验收勾选。


> **已合并超集**：请改用 [`ui-and-deadcode-codex-prompt-20260911.md`](./ui-and-deadcode-codex-prompt-20260911.md)。本文件仅保留设计系统 Part II，避免两份提词分叉。
