# 首页 Design QA

source visual truth path: `design-evidence/home-reference-full-1366x768.png`

implementation screenshot path: `design-evidence/home-rebuild-full-1366x768.png`

viewport: 1366 x 768 CSS px；补充检查 1920 x 1080 与 1024 x 768。

source and implementation pixel dimensions, CSS size, and density normalization used: 两张主对比图均为 1366 x 768 px，CSS 视口均为 1366 x 768，deviceScaleFactor=1，无缩放归一化。1920 补充图同为 1920 x 1080 px、deviceScaleFactor=1。

state: 两端均为已登录 PO 首页、浅色主题、价值流“全部”状态；数据量与用户身份来自各自运行环境，因此不作为视觉一致性缺陷。

full-view comparison evidence: `design-evidence/home-comparison-full-1366x768.png`；`design-evidence/home-comparison-1920x1080.png`。

focused region comparison evidence: `design-evidence/home-focus-comparison-full-1366x540.png`，覆盖品牌区、角色切换、左侧导航、标题、价值流、高优摘要和行动列表密集区域。

**Findings**

- 无剩余 P0/P1/P2。共享外壳的 56px 顶栏、200px 左栏、20px 内容边距、40px 标题行、56px 阶段卡片、列表与底部页签位置均已与参照页几何对齐。
- 字体与排版：标题 18px/600，导航 13px，辅助文字 10-11px，数字使用等宽字体；未发现影响层级、截断或可读性的偏差。
- 间距与布局：1366 与 1920 无横向页面溢出；1024 下顶部控件、侧栏和主内容未互相覆盖。密集列表使用内部横向滚动保护窄桌面。
- 颜色与 token：主色、浅蓝选中态、页面底色、边框、风险橙与按钮蓝已映射到 Main 的 PO token。
- 图片与图标：页面不含照片或插画；品牌标记与界面图标均使用项目现有 Font Awesome/文本品牌资源，无自绘 SVG 或占位图。
- 文案与内容：固定界面文案已对齐；需求数量、责任人、用户名及底部已打开页签数量属于两个运行环境的真实状态差异，保留各自数据。
- 状态与交互：价值流“排期”点击后 `aria-pressed=true`，标题切换为“统一行动列表 · 排期（3）”，渲染 3 行；Console error 为 0。

**Open Questions**

- 首页摘要中的“今日必推 / 阻塞 / 超期 / 挂起”当前显示破折号，明确表达 Main 尚未提供这些口径；本轮按用户要求不补后端。
- “我的关注”和“需求查询”仅在左栏作为后续交付入口展示，本轮未创建对应路由。

**Comparison History**

- Pass 1：P2，高优摘要三张卡片被平均拉满整行，视觉密度明显低于参照；P2，价值流卡片间距与参照不一致。
- Fix：高优卡固定为 177px、6px 间距、5px 7px 内距、4px 圆角；价值流改为 6px 间距，并校准标题区与卡片的 4px 垂直节奏。
- Pass 2：同尺寸全图与密集区域同屏复核通过；关键几何误差在亚像素到 1px 范围，未发现新的 P0/P1/P2。

**Implementation Checklist**

- [x] 参照与实现使用同一视口、主题、登录态和首页状态。
- [x] 顶栏、侧栏、主内容、价值流、摘要卡、表格与底部页签完成视觉对齐。
- [x] 1366、1920、1024 三档无页面级横向溢出或控件遮挡。
- [x] 首页阶段筛选交互与控制台检查通过。
- [x] 后端未覆盖的指标以破折号展示，不伪造数字。

**Follow-up Polish**

- P3：待“我的关注”页面交付时再让底部工作页签累积多个真实页面状态，避免本轮用静态假页签模拟历史访问。

final result: passed
