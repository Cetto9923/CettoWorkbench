# 研发工作台 UI 设计体系指南 (Design System Guide)

研发工作台面向银行产品经理与研发团队，采用专业、克制、信息密度适中的企业级工具设计语言（对标 Linear、飞书项目）。

---

## 1. 核心设计变量（`tokens.css`）

所有业务样式严禁硬编码 Hex 颜色、像素间距或圆角，统一引用以下 CSS 变量：

- **品牌主色 (Brand Blue)**:
  - `--wb-primary`: `#1d4ed8`（主操作、激活项、主链接）
  - `--wb-primary-hover`: `#1e40af`
  - `--wb-primary-light`: `#eff6ff`（高亮轻背景）
  - `--wb-primary-border`: `#bfdbfe`（轻量边框）
- **中性灰阶 (Slate Palette)**:
  - `--wb-bg-page`: `#f8fafc`（页面底层背景）
  - `--wb-bg-surface`: `#ffffff`（卡片与表格背景）
  - `--wb-bg-muted`: `#f1f5f9`（次级与表头背景）
  - `--wb-border-subtle`: `#e2e8f0`（标准分割线）
  - `--wb-border-default`: `#cbd5e1`（表单边框）
  - `--wb-text-primary`: `#0f172a`（主标题与正文）
  - `--wb-text-secondary`: `#475569`（辅助标签与元信息）
  - `--wb-text-tertiary`: `#94a3b8`（占位说明与次级图标）
- **功能语义色 (Semantic)**:
  - `--wb-success`: `#15803d` / `--wb-success-bg`: `#f0fdf4`
  - `--wb-warning`: `#b45309` / `--wb-warning-bg`: `#fffbeb`
  - `--wb-danger`: `#b91c1c` / `--wb-danger-bg`: `#fef2f2`
  - `--wb-info`: `#0369a1` / `--wb-info-bg`: `#f0f9ff`

---

## 2. 通用组件类（`components.css`）

- **标准卡片**: `.wb-card`, `.wb-stat-card`
- **通用表格**: `.wb-table`（高度 40px，轻量底线）
- **筛选栏**: `.wb-filter-bar`（包含 `.wb-filter-item`, `.wb-filter-select`, `.wb-search-input`）
- **三态按钮**:
  - 主要操作: `.wb-btn-primary`
  - 次要线框: `.wb-btn-secondary`
  - 幽灵按钮: `.wb-btn-ghost`
- **状态徽标**: `.wb-tag`, `.wb-tag-success`, `.wb-tag-warning`, `.wb-tag-danger`, `.wb-tag-info`
- **规范空状态**: `.wb-empty`（含 `.wb-empty-icon`, `.wb-empty-title`, `.wb-empty-desc`）
- **解释说明 Tooltip**: `.wb-tooltip-wrap` > `.wb-tooltip-icon` (ⓘ) + `.wb-tooltip-content`

---

## 3. 全局辅助函数（`app.js`）

- `window.formatDateTime(val)`: 统一时间格式为 `YYYY-MM-DD HH:mm`（消除尾随秒与零散格式）。
