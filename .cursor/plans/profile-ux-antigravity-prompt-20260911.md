# Antigravity 执行提词：个人资料「PO 化」（不是删死代码）

> **仓库：**`/Users/yuyan9923/GitHub/workbench-claude-po`
> **分支：**`release/po-integrate-main-202609`（HEAD `fada8cc1`，已校验）
> **校验对象：**Antigravity / Cursor / Claude Code 等同源 Agent；阅读本文件即视为已加载工程总规范
> **本机验：**`WORKBENCH_MODE=dev WORKBENCH_APP_ADDR=:8090 WORKBENCH_SESSION_COOKIENAME=wb_session_po_8090`，账号 `003030` / `123456`
> **不动范围：**`internal/module/profile/**`、表 `zt_user` / `zt_wb_profile_prefs`、`/api/profile*`、`/profile/data`、`/profile/password`、`profile_service_test.go`、`AGENTS.md`、`docs/PRD/`、A+B LinkStory/提测弹窗、其它非个人资料的页面
> **不要 push**

---

## 0. 先纠正认知（必读，不要在这点上来回）

**"旧框架"是体验债，不是死代码。** 事实如下，已逐项对照源码验证：

| 项 | 事实 | 证据 |
|---|---|---|
| 入口 | `header.html` 写死 `<a href="/profile">个人资料</a>` | `web/templates/layout/header.html:48` |
| 后端 | 已接线 bootstrap/routes；读写 `zt_user` + `zt_wb_profile_prefs` | `internal/module/profile/handler.go`、`service.go`、`repo.go`、`form.go` |
| API | `GET/PUT /profile`、 `/profile/data`、 `/profile/password`、`GET/PUT /api/profile`、`PUT /api/profile/password` | `handler.go:42-51` |
| 前端 | `web/templates/profile/index.html` + `web/static/{js,css}/po/po-profile.{js,css}`；**`openProfileModal` 已实现**（474 行 JS） | `web/static/js/po/po-profile.js:520-…` |
| 壳 | `/profile` 已进 `po-workbench-shell`（sidebar+底栏） | `web/templates/layout/base.html:26` |
| 表单字段 | 邮箱/性别/自选角色/默认敏捷小组 + 改密子区块 | `internal/module/profile/form.go` |
| 底栏 | `bottomtabbar.js` 检测到 `/profile` 就注入「个人资料」Tab（`bi bi-person-vcard`） | `web/static/js/layout/bottomtabbar.js:98-104` |

`internal/module/profile` **禁止删除**；后端 + 模板 + CSS + JS 全部留。本任务是把"视觉与入口形态"对齐 PO 工作台。

---

## 1. 产品目标（写死）

把「个人资料」收成 **PO 工作台内的设置体验**：

- **主路径**：顶栏头像菜单点击「个人资料」→ 打开 `openProfileModal`，不离开当前页（URL hash 可选 `#profile`）。
- **次路径**：`/profile` 深链保留（书签/刷新），但页面与弹窗共用同一份渲染（已有 `profilePageBody` / `profileModalBody` 双宿主，无需新增组件，复用 `renderBody` 即可）。
- **底栏**：访问 `/profile` 时 **不**把「个人资料」插入工作 Tab。
- **侧栏**：**不要**新增「个人资料」项（设置不属于待办/关注同级）。
- **图标**：统一 Font Awesome `fas`，消除 `bi-*` 在个人资料路径上的使用（其它页面遗留 `bi-*` 不在本任务范围）。
- **字段能力** 全部保留（只读：姓名、用户名、手机、部门、敏捷小组；可编辑：邮箱、性别、自选视图、默认小组；改密）。

---

## 2. 必做改造（每条都给具体文件 + 行号）

### A. 入口改为弹窗优先

**A.1** `web/templates/layout/header.html:48`

```diff
-        <a href="/profile" class="dropdown-item topbar-dropdown-item"><i class="bi bi-person-vcard"></i><span>个人资料</span></a>
+        <button type="button" class="dropdown-item topbar-dropdown-item"
+                data-open-profile-modal
+                onclick="return openProfileModalFromHeader(event)">
+          <i class="fas fa-id-card"></i><span>个人资料</span>
+        </button>
```

实现方式：
- `openProfileModalFromHeader(event)` 来自 `po-profile.js`（新加，内部就是 `openProfileModal()` + `preventDefault`）。
- 若 `po-profile.js` 未加载则降级 `<a href="/profile">`，因此**按钮本身不依赖 JS**——脚本加载失败也能跳整页。

**A.2** `web/templates/layout/base.html:75-86`（脚本区）

`po-profile.js` **只**在 `profile/index.html:30` 的 `page_js` 引入，业务页没有，所以 `openProfileModal` 在 home/todos/follow 等页面是缺失。最小做法：

```diff
   <script src="{{ asset "/static/js/components/autocomplete-options.js" }}"></script>
   <script src="{{ asset "/static/js/ui.js" }}"></script>
   <script src="/static/js/app.js"></script>
   <script src="/static/js/components/components.js"></script>
   <script src="{{ asset "/static/js/layout/sidebar.js" }}"></script>
   <script src="{{ asset "/static/js/layout/globalsearch.js" }}"></script>
+  <script src="{{ asset "/static/js/po/po-profile.js" }}" defer></script>
   {{ if not .HideChrome }}<script src="{{ asset "/static/js/layout/bottomtabbar.js" }}"></script>{{ end }}
```

理由：让所有 PO 壳页拿到 `openProfileModal`；`/profile` 的 `page_js` 重复引入同一文件是无害的（IIFE 闭包、`defer` 加载幂等）。

**A.3** `header.html:53,79,93` 三处遗留的 `bi bi-box-arrow-right` / `bi bi-person`

| 位置 | 现在 | 改 |
|---|---|---|
| `:53` 登出按钮 | `bi bi-box-arrow-right` | `fas fa-right-from-bracket` |
| `:79` 顶栏按钮图标 | `bi bi-person` | `fas fa-user` |
| `:93` 顶栏登出 | `bi bi-box-arrow-right` | `fas fa-right-from-bracket` |

**A.4** 图标 fallback：保留 `<link rel="stylesheet" href="{{ asset "/static/vendor/bootstrap-icons/bootstrap-icons.css" }}">` 在 `base.html:19`，其它页面继续使用；只改个人资料路径上的 `bi-*` 实例，避免扩大改动范围。

### B. 视觉对齐 PO 工作台（整页 + 弹窗同一套）

**B.1** 模板 `web/templates/profile/index.html`

- 顶栏副标题缩短成一行（"维护当前登录用户资料、自选视图、默认小组"）。其它不动。
- 卡片骨架不动（已用 `workspace-panel` + `workspace-header`）。

**B.2** CSS `web/static/css/po/po-profile.css`

- 删除文件 318–352 行（`@media (prefers-color-scheme: dark)` 等页面级暗色选择器拷贝——违反 frontend.md §5 theme override 合同）。
- 暗色统一走 `web/static/css/layout/variables.css` 的 `:root[data-theme="dark"]` token 覆盖。
- 按钮 `.po-profile .action-btn` 改为 `action-btn / action-btn primary` **不重写基础样式**，只覆盖 `height:36px`、`gap` 等微调，并确保与 `web/static/css/components/button-system.css` 主按钮样式不冲突（`button-system` 已全局引入，`action-btn primary` 就是主按钮）。
- `width:36px` 在 `.po-profile .action-btn` 类的限制如 `gap` 与按钮系统相冲，按钮系统优先，本地只保留布局（`display: inline-flex` 等）。
- 卡片用现有 `card-base` / `workspace-panel` 圆角与边框；删掉本地 `.profile-card` 的私有 `--shadow-sm, 0 1px 2px 0 rgba(0,0,0,0.05)` 阴影（不属于 token），如有阴影需求改为 token。

**B.3** 密码区折叠

`web/static/js/po/po-profile.js` 中「修改密码」卡片加 `<details>` 或 `<button data-toggle="profilePwdBlock">` 控制显隐（默认收起）。CSS 一行 `:not([open]) .profile-pwd-grid { display: none; }`。`renderBody` 模板（当前 `web/static/js/po/po-profile.js:241-262`）改造：

```html
<div class="profile-card">
  <div class="profile-card-h">
    <i class="fas fa-key"></i>
    <button type="button" class="profile-card-toggle" data-toggle="profilePwdBlock" aria-expanded="false">
      修改密码 <i class="fas fa-chevron-down"></i>
    </button>
  </div>
  <div id="profilePwdBlock" class="profile-pwd-grid" hidden>
    <!-- 现有三个 pwd-input-wrap + 更新密码按钮 -->
  </div>
</div>
```

### C. 底栏与导航

**C.1** `web/static/js/layout/bottomtabbar.js:98-104`

```diff
-    if (path === "/profile") {
-      return {
-        path: "/profile",
-        title: "个人资料",
-        icon: "bi bi-person-vcard",
-      };
-    }
+    if (path === "/profile") {
+      // 不把「个人资料」当工作 Tab；保持单 Tab 列表。
+      return null;
+    }
```

注意：`bottomtabbar.js` 中 `tabs.push`/`upsertTab` 等逻辑，若返回 `null`，外层要安全跳过——已在 `currentPath()` 流程中允许 `null`。**不要新增 Tab 注入**。

**C.2** 侧栏 `web/templates/layout/sidebar.html`

**不要**新增「个人资料」菜单项。个人资料入口只在顶栏头像菜单。

### D. 行为与无障碍

- 保存资料成功：`showToast('个人资料已保存', 'success')`（已存在）。
- 改密成功：`showToast('密码已更新', 'success')`。
- 失败：弹窗内显示 `message`，不要用 alert。
- `appFetch` + CSRF 已有（po-profile.js:65-100）。
- 弹窗：Esc 关闭 + 遮罩关闭由 `openProfileModal` 内部处理；保留。
- Light / Dark 各看一遍。

---

## 3. 明确不做（写死）

- ❌ 不删 `internal/module/profile/`、表结构、API 契约。
- ❌ 不改 `profile_service_test.go`（现有测试覆盖 role 校验/密码校验）。
- ❌ 不借机做 A+B LinkStory 收尾、提测弹窗 Phase C、六页 dark 大迁移。
- ❌ 不引入 Bootstrap 组件库。
- ❌ 不动 `button-system.css` / `variables.css` 既有 token。
- ❌ 不改其它页面遗留 `bi-*`（范围控制；只处理个人资料路径上的 `bi-*`）。
- ❌ 不做头像上传 / 修改姓名（接口已能改姓名但默认不开放 UI）。
- ❌ 不重做角色体系。

---

## 4. 验证脚本（Antigravity 必须自验；不通过即不算 done）

```bash
cd /Users/yuyan9923/GitHub/workbench-claude-po

# 1. 后端契约不动（仓库未编译，但全仓 build 必过）
go build ./...

# 2. profile 单测必绿
go test ./internal/module/profile/... -count=1

# 3. 现有前端单测必绿
node tests/unit/frontend/po-profile.test.js 2>/dev/null || true   # 若文件不存在则跳过
make check-frontend-test

# 4. 真实浏览器冒烟（手动）
#    登录 003030 / 123456
#    - /home 点顶栏头像 → 个人资料 → 期望：弹窗打开，URL 不跳 /profile
#    - /home 直访 /profile → 期望：整页渲染，整页 + 弹窗视觉一致
#    - Light / Dark 各看一遍：表单 / 卡片 / 输入框 / 提示文案
#    - 改密区块默认收起；点开后输入合法 → 200 → toast「密码已更新」
#    - 底栏无「个人资料」Tab（即使在 /profile）
#    - 网络面板：无 404 / 500；CSRF 头存在；fetch 走 /api/profile 与 /api/profile/password
```

人工冒烟每条记录"通过 / 失败 + 一句话"。

---

## 5. 建议 commit（可拆三段，便于 Antigravity 复核）

1. `fix(ui): open profile from header as modal on PO shell`
   - 改 `web/templates/layout/header.html`、`web/templates/layout/base.html`、`web/static/js/po/po-profile.js`（加 `openProfileModalFromHeader` + 密码区折叠）
2. `fix(ui): restyle profile form to match personal-workspace`
   - 改 `web/static/css/po/po-profile.css`（删 dark 副本，收紧到 token）
3. `fix(ui): stop pinning profile in bottom tab bar`
   - 改 `web/static/js/layout/bottomtabbar.js`

不要混合提交非本任务范围改动。

---

## 6. 风险与门禁（Antigravity 收尾自检表）

| 风险 | 缓解 |
|---|---|
| `po-profile.js` 已在 `base.html` 引入，业务页可能与 `/profile` 重复加载 | IIFE 闭包，重复定义幂等；不影响 |
| 顶栏 button 改造后，`<a href>` 降级需保留以应对 JS 加载失败 | 已在 A.1 写明降级方案 |
| 删 `.po-profile .action-btn` 私有样式可能与 `button-system` 冲突 | 先量 `:has(.po-profile)` 内 `action-btn primary` 视觉，必要时保留高度/间距微调，但边框、背景、圆角交回 `button-system` |
| dark 模式失去覆盖 | 由 `variables.css` 的 `:root[data-theme="dark"]` 接管；如有微调差值，新增语义 token |
| 底栏返回 `null` 后调用方应跳过 | `bottomtabbar.js:130 render(tabs)` 已循环空数组安全；若无空安全检查，加 `if (!entry) return;` |

---

## 7. 回报（提交前必须给验收方）

- 修改文件清单（精确到行号段）
- 入口行为前后对比（一句话）
- `openProfileModalFromHeader` 与 `openProfileModal` 的关系（一句话）
- Light / Dark 截图各一张（弹窗 + 可选整页）
- `go test ./internal/module/profile/...` 截图/输出
- 验收方：`docs/engineering/frontend.md` Theme Contract 自检：未新增 dark 副本 / 用了 semantic token / 按钮走 `button-system`

不 push。