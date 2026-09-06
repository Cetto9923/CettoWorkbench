# Frontend engineering

Templates render server-owned data. Browser JavaScript owns events, requests,
transient view state, and rendering. It MUST NOT be the only implementation of
permission checks, business filtering, totals, or state transitions.

Before page-local code, inspect shared support for HTML escaping; fetch/JSON/error
and authentication handling; toast/modal; pagination/dropdowns; form errors;
CSRF; and distinct loading/empty/error states. New code MUST reuse a matching
stable capability. If similar implementations have different contracts, keep
them local and record why. Extract only after behavior and API are stable.

`app.js`/shared UI files provide transport and primitives; `components/` owns
reusable widgets; `layout/` owns shell navigation; module directories own page
behavior. Page scripts SHOULD be capability-focused and MUST meet the size gate.

CSS tokens define design values, shared component CSS defines reusable widget
geometry/states, and page CSS composes/specializes a page. Do not copy tokens or
a full shared component into page CSS.

Internal/object navigation opens in the current page by default. A new window
requires an explicit product reason and safe `noopener` handling.

Loading, empty, and error are separate states. Errors must not render as empty,
stale rows must not remain after failed refresh, and retry should be visible when
possible. Fetch handling covers non-2xx, invalid payload, auth expiry, and user
feedback consistently.

## Theme Contract

1. **Theme state**
   整个应用只能存在一个主题状态。
   The effective document theme MUST only be: `light`, `dark`.
   `html[data-theme]` MUST therefore only resolve to:
   `html[data-theme="light"]`
   `html[data-theme="dark"]`
   `data-theme="system"` MUST NOT be introduced.
   页面模块不得创建 `.dark`, `.night`, `.black-theme`, `page-local theme state` 等独立主题系统。

2. **Preference**
   The persisted/user preference MAY be: `system`, `light`, `dark`.
   When preference is `system`, `prefers-color-scheme` resolves the effective theme.
   *(注：本任务只写合同，不实现 runtime behavior)*

3. **Semantic tokens**
   所有可主题化视觉颜色必须通过 Semantic CSS Variables 表达。
   优先语义：
   `--color-page-bg`, `--color-surface`, `--color-surface-hover`, `--color-surface-muted`,
   `--color-text-primary`, `--color-text-secondary`, `--color-text-disabled`,
   `--color-border`, `--color-border-strong`,
   `--color-primary`, `--color-success`, `--color-warning`, `--color-danger`
   以及经过评审新增的真实语义状态 token。
   禁止因 Theme 新增 `--dark-bg`, `--dark-text`, `--black`, `--white-card` 这类“外观命名”的共享 token。

4. **Page CSS**
   新页面或被实质修改的页面不得使用硬编码颜色替代已有 semantic token。
   例如禁止直接写：`background: #fff`, `color: #1f2937` (如果已有 semantic token 能表达)。
   - New declarations and visual states touched by the current task must comply with semantic theme tokens.
   - Existing untouched legacy hardcoded colors are migration debt.
   - Theme compliance does NOT authorize unrelated full-page/full-module CSS cleanup.
   - Large legacy theme migration requires its own explicit task.
   例外：
   - SVG / logo 品牌固定色
   - 数据可视化必须保持的业务色
   - 经明确说明的第三方组件
   - 极少数无法语义化的局部值
   例外必须局部，不得形成第二套主题。

5. **Theme override**
   Light / Dark 差异只能主要通过 root semantic token override 实现。
   禁止创建 `dark.css` 或者复制整个 selector 集合 (`.component { ... }`, `[data-theme=dark] .component { ... }`) 只为了更换普通背景/文字/边框颜色。
   组件结构、尺寸、布局不得因为 Theme 重复实现。

6. **Business semantic colors**
   风险、错误、成功、警告等业务语义在 light / dark 下必须保持语义一致。
   红色不能在 dark theme 中变成普通装饰色。warning 不能改成 brand blue。

7. **Accessibility**
   两种主题必须保持以下清晰可辨：
   - 可读文字对比度
   - hover, active, selected, disabled, focus-visible, loading, empty, error
   禁止使用 `filter: invert(...)`, 全页 `brightness`, `opacity hack` 实现 dark mode。
   图片、截图、Logo 不得全局反色。

8. **Shared components**
   新增或实质修改的共享 UI 组件必须确认 Light / Dark 都可用。
   至少覆盖：page, panel/card, table, input/select, dropdown, modal, tabs, button, toast, empty/error/loading, scrollbar（如果项目自定义）。

9. **No business coupling**
   Theme 只属于 presentation。
   主题切换不得改变：permission, route, query, API, 业务状态, 对象可见性, workflow, 数据内容。

10. **Verification**
    前端涉及主题的任务完成前必须至少验证 `light` 和 `dark`。
    如支持 system，则验证 system resolve。
    页面视觉工作仍遵循现有 authenticated browser acceptance 规则。
