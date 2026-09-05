# 员工工作台四页优化基线记录 (UI-00 Baseline)

- **记录日期**: 2026-09-05
- **当前执行分支**: `Claude-PO`
- **当前 HEAD Commit**: `2f2580a10f45db5978736f1fe1e46c258133022f` (与计划编制基线完全一致)
- **本地服务地址**: `http://127.0.0.1:8093`
- **运行进程**: PID 91974 (`./tmp/workbench_dev`, cwd: `/Users/yuyan9923/GitHub/workbench-claude-po`)

---

## 1. Preflight Git 状态与资产保护

```text
=== PWD ===
/Users/yuyan9923/GitHub/workbench-claude-po

=== TOPLEVEL ===
/Users/yuyan9923/GitHub/workbench-claude-po

=== BRANCH ===
Claude-PO

=== HEAD ===
2f2580a10f45db5978736f1fe1e46c258133022f

=== STATUS ===
 M web/static/css/po/board.css
 M web/static/js/po/workboard.js
 M web/templates/po/workboard.html
?? docs/plan/personal-workbench-ui-20260905/
```

> [!IMPORTANT]
> **保护范围**: `board.css`、`workboard.js`、`workboard.html` 属于工作看板已有 WIP，不在本次四页优化修改范围之内。本次优化必须严格保护上述文件，严禁修改、回退、删除或引入不相关差异。

---

## 2. 目标页面接口与加载资产现状

| 页面路由 | 页面标题 | 主模板 | 专属 CSS | 专属 JS | 数据 API 端点 |
|---|---|---|---|---|---|
| `/home` | 工作台首页 - workbench | `web/templates/po/home.html` | `home.css`, `homecompact.css` | `home.js` | SSR 注入 + `/home` 模板数据 |
| `/todos` | 我的待办 - workbench | `web/templates/po/todos.html` | `todos.css` | `todos.js` | `GET /todos/items` |
| `/done` | 我的已办 - workbench | `web/templates/po/done.html` | `done.css` | `done.js` | `GET /done/items` |
| `/notice` | 通知中心 - workbench | `web/templates/po/notice.html` | `notice.css` | `notice.js` | `GET /notice/items`, `PUT /notice/:id/read`, `PUT /notice/read-all` |

### 实测端点响应基线（已验证）

- `GET /home`: HTTP 200, 响应大小 ~14.0 KB
- `GET /todos`: HTTP 200, 响应大小 ~10.9 KB
- `GET /todos/items`: HTTP 200, 返回 `{success: true, total: 27, items: 20, groups: {...}, summary: {...}}`
- `GET /done`: HTTP 200, 响应大小 ~10.6 KB
- `GET /done/items`: HTTP 200, 返回 `{success: true, total: 4600, items: 20, summary: {...}}`
- `GET /notice`: HTTP 200, 响应大小 ~10.7 KB
- `GET /notice/items`: HTTP 200, 返回 `{success: true, total: 212, items: 20, categories: {...}, abnormal: 0, action: 0}`

---

## 3. 问题台账定位与基线核对（E01–E12）

1. **E01 (几何与层级不统一)**:
   - `todos.css`: 标题字号 22px，概览卡片高 76px；分类用下划线高 40px；分页用 `pager-btn`。
   - `done.css`: 标题字号 20px，概览卡片高 68px（平铺 6 个时间区间按钮）；分类用自制下划线高 42px；分页用独立 `.po-done .pager-btn`。
   - `notice.css`: 标题字号 22px，概览卡片高 76px；分类用圆角胶囊药丸按钮（`border-radius: 18px`）；分页用自制 `notice-pager-btn`。
   - 结论：三页缺少统一的 `.po-personal-workspace` 共享容器与几何规范。

2. **E02 (窄视口溢出与多行换行)**:
   - `done-stat-strip` 在窄视口强行横向排布导致横滚；
   - `todos` 和 `notice` 概览卡片固定 5 列或 `repeat(3, 1fr)` 导致换行不平整；
   - `todos-toolbar` 和 `notice-toolbar` 控件宽度固定，窄视口容易错位。

3. **E03 (notice 页文本残留与缺失关联)**:
   - 类别标签中存在无用正文或 `mail / —`、`story / —` 的虚假展示；缺少对象 ID 时未显示“关联信息缺失”。

4. **E04 (首页宽屏留白与重复卡片)**:
   - `top5-section` 顶部固定渲染前三项的高优大卡片（`homePriorityStrip`），占用了纵向空间并且在右侧留白严重，同时行动列表标题宽度分配不合理。

5. **E05 (加载与错误空态混淆)**:
   - `todos.js` 和 `notice.js` 请求失败时直接显示空状态或吞错，没有独立可重试的错误状态展示。

6. **E06 (缺乏请求时序保护与竞态防御)**:
   - 三列表及首页异步刷新没有携带请求序号（sequence ID）或 AbortController，快速切换分类或筛选时旧响应会覆盖新响应。

7. **E07 (重置行为不一致)**:
   - `done` 重置全部筛选条件，而 `todos` 和 `notice` 仅重置了部分选择器，遗留了隐藏有效过滤导致假空态。

8. **E08 (首页摘要按钮无响应)**:
   - `home.html` 顶部 `home-hl-kpi` 包含 `data-summary` 按钮，但实际上没有对应后端筛选 API，属于非交互统计，却渲染成带 hover 和 cursor 的按钮。

9. **E09 (待办未接入对象缺少标识)**:
   - `QueryTodoUnified` 只实现了 demand、task、bug 三类对象，但前端分类 tab 列出了审批决策、测试质量、问题风险、个人事项，没有明确标注“未接入”。

10. **E10 (首页全量查询与客户端截断)**:
    - `FindRoleDemands` 无 LIMIT，首页行动列表在客户端进行切片分页。此项属于 DATA-01 有条件独立包，不混入 UI 改造。

11. **E11 (服务端失败返回 200 零值)**:
    - 首页 Home handler 在出错后返回 200 零值响应；已办标题查询吞错。

12. **E12 (首页更新时间虚假)**:
    - 首页展示的更新时间直接取自浏览器当前系统时间，推荐卡片直接取数组前三项无实际排序算法。

---

## 4. UI-00 结论

基线证据收集完毕，当前工作树中的已有 WIP（`board.css`、`workboard.js`、`workboard.html`）已严格识别并隔离，四页优化施工环境（UI-01 ~ UI-05）就绪。
