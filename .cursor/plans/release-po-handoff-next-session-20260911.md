# Release/po-integrate-main-202609 — 下一会话开局说明

> 上一会话收尾时间：2026-09-11 18:30 CST
> 当前 goal 已 budget_limited，本文档作为跨会话接力。
> 上一会话原文：`.cursor/plans/release-remaining-master-plan-20260911.md`
> 总计划：`/Users/yuyan9923/GitHub/workbench` 仓 `.cursor/plans/absorb-main-into-release-20260911.md`（context）

---

## 0. 当前基线（一行命令自检）

```bash
cd /Users/yuyan9923/GitHub/workbench-claude-po
git branch --show-current   # → release/po-integrate-main-202609
git rev-parse --short HEAD  # → 4e54777a
git status --short         # 应只看到 7 个 看板 轨 M + plan/demo ??（见 §3）
```

HEAD = `4e54777a`。ahead origin 7 commits，未 push。

## 1. 已 commit 的全部 7 笔

| Commit | 标题 | Track |
|---|---|---|
| `16230a37` | zentao `DoAs`/`Do` + APILog | A 吸收 |
| `d76fe2b4` | render 兄弟模板 | A 吸收 |
| `a940fbdc` | build/LinkStory 模块接线 | A 吸收 |
| `1ed0eb2c` | testtask executions + createBuilds via DoAs | B 吸收 |
| `05774d60` | 提测弹窗接真 builds API，去静态演示 | C 吸收 |
| `874543af` | chore(schedule): gofmt form batch cap files | T2-a |
| `f2e0daa` | fix(ui): open profile from header as modal + deep-link page | **T3-A** |
| `4e54777` | fix(ui): restyle profile to personal-workspace | **T3-B** |

**注意：上一会话做了 8 笔，不是 7 笔**——T3-A 和 T3-B 是本次新增。

## 2. T3 已完成（不要再碰）

`/profile` 深链页恢复（31 行）+ 顶栏入口挂 `openProfileModal` + `bottomtabbar.js` 排除 `/profile` + `constants.TEMPLATE_PROFILE_INDEX` 已加 + `bootstrap.go` 注释更新 + `po-profile.js`/`po-profile.css` 视觉重做。

**不要再改的：**
- `internal/module/profile/handler.go`（已与 HEAD 一致，含 Index handler）
- `web/templates/profile/index.html`（已恢复，等于 HEAD）
- `internal/constants/templates.go`（已加 TEMPLATE_PROFILE_INDEX，gofmt 过）
- `web/templates/layout/header.html` / `base.html` / `bottomtabbar.js`（WIP 已收口 commit）
- `web/static/js/po/po-profile.js` / `web/static/css/po/po-profile.css`（WIP 已收口 commit）

**唯一仍存的债务：** `po-profile.js` 670 行 + `po-profile.css` 597 行超 500 上限。HEAD 已 593/430，**既有 legacy**。`check-file-length.sh` 拒绝新增 baseline 条目。会话内验收标 **BLOCKED BY EXISTING BASELINE**，不要试图绕过 gate（删测试、关 gate、改阈值都不行）。

## 3. WIP（旁路三轨之看板轨，**未 commit，禁止与 T4/T5 混**）

```
 M deploy/zentao/routes.php
 M internal/module/po/board_task_transition_test.go
 M internal/module/po/service.go
 M internal/module/po/serviceboard.go
 M internal/pkg/zentao/task_client.go
 M web/static/css/po/board.css
 M web/static/js/po/workboard.js
```

**这一轨**：`feat(board): …` 独立 commit 收口；用 `gofmt -w` 同步 baseline；跑 `internal/module/po/...` + `internal/pkg/zentao/...` 单测。如不稳，保留 WIP，不要塞别的 track。

## 4. T4 linkstory 入口（**最小可行设计已就绪**，可直接落地）

**目标文件：** `web/static/js/po/testtask-builds.js`

**改动位置：** `handleSubmit()` 成功分支，当前是：
```js
toast(wrap.body.message || summary, "success");
sessionStorage.removeItem(draftKey());
if (typeof window.closeShowModals === "function") {
  window.closeShowModals(["poTesttaskModal", "poTesttaskOverlay"]);
}
```

**改为：** 当 `data.builds.length > 0` 时**不关弹窗**，把 `.po-testtask-actions` 区替换成 inline 摘要 + 两个 secondary 按钮（点击时关 testtask 弹窗再开 linkstory 弹窗）；否则维持原关闭行为。

**新增函数（注意用单引号外层避免转义坑）：**
```js
function showPostBuildActions(builds) {
  var $modal = $("#poTesttaskModal");
  var $actions = $modal.find(".po-testtask-actions");
  if (!$actions.length) {
    if (window.closeShowModals) window.closeShowModals(["poTesttaskModal","poTesttaskOverlay"]);
    return;
  }
  var primary = builds[0];
  var $summary = $('<div class="po-testtask-post-build-summary"></div>');
  $summary.append(
    '<div class="po-testtask-post-build-msg"><i class="fas fa-check-circle"></i> ' +
    builds.map(function (b) { return '#' + b.buildId + ' ' + b.name; }).join('，') +
    ' 已写入禅道</div>'
  );
  var $btnRow = $('<div class="po-testtask-post-build-actions"></div>');
  if (primary && primary.buildId && typeof window.openPoLinkstoryModal === 'function') {
    $btnRow.append('<button type="button" class="action-btn primary" id="poTesttaskLinkstoryBtn"><i class="fas fa-link"></i> 关联研发需求</button>');
  }
  $btnRow.append('<button type="button" class="action-btn" id="poTesttaskPostBuildCloseBtn">关闭</button>');
  $summary.append($btnRow);
  $actions.empty().append($summary);
  $modal.off('click.postbuild').on('click.postbuild', '#poTesttaskLinkstoryBtn', function () {
    if (window.closeShowModals) window.closeShowModals(["poTesttaskModal","poTesttaskOverlay"]);
    if (window.openPoLinkstoryModal && primary) window.openPoLinkstoryModal({ buildId: primary.buildId });
  });
  $modal.off('click.postbuildclose').on('click.postbuildclose', '#poTesttaskPostBuildCloseBtn', function () {
    if (window.closeShowModals) window.closeShowModals(["poTesttaskModal","poTesttaskOverlay"]);
  });
}
```

**改动点（handleSubmit 内）：**
```diff
- if (typeof window.closeShowModals === "function") {
-   window.closeShowModals(["poTesttaskModal", "poTesttaskOverlay"]);
- }
+ if (builds.length > 0) {
+   showPostBuildActions(builds);
+ } else if (typeof window.closeShowModals === "function") {
+   window.closeShowModals(["poTesttaskModal", "poTesttaskOverlay"]);
+ }
```

**Commit message 模板：**
```
feat(ui): hook linkstory entry after createBuilds

Plan release-remaining-master-plan-20260911.md §T4.

When CreateBuilds returns data.builds, keep the testtask modal open
and replace its .po-testtask-actions with an inline summary plus
two secondary buttons:
- "关联研发需求" closes the modal and opens openPoLinkstoryModal
  for the first buildId
- "关闭" closes the modal

No data.builds path keeps the existing close behaviour. No buildId
or no openPoLinkstoryModal suppresses the secondary button.
Toast, draft cleanup, and error path are unchanged.
```

**验收：**
- [ ] 创建版本成功 → 弹窗不立即关，按钮出现
- [ ] 点「关联研发需求」→ linkstory 弹窗打开，buildId 正确
- [ ] 点「关闭」→ 仅关闭 testtask 弹窗
- [ ] 失败路径行为不变

## 5. T5 dark → token（**大改造，需先读 Theme Contract §5**）

**先读：** `docs/engineering/frontend.md` §5 Theme Contract。

**禁止：**
- 新增页面级 `html[data-theme="dark"] .component` 拷贝
- `--dark-bg` 外观命名（必须语义 token）
- `filter: invert`
- 第二套主题

**目标文件 + 当前 dark 块数：**
| 文件 | dark blocks | 行数 | 备注 |
|---|---|---|---|
| `web/static/css/po/follow.css` | 46 | 706 | 超 500，**需先拆分或留 legacy** |
| `web/static/css/po/wb-priority.css` | 25 | 489 | |
| `web/static/css/po/po-urge.css` | 18 | 417 | |
| `web/static/css/po/home.css` | 23 | 364 | |
| `web/static/css/po/notice.css` | 12 | 300 | |
| `web/static/css/po/done.css` | 5 | 345 | |

**方法：**
1. 找 light/dark 都缺的语义，补到 `web/static/css/layout/variables.css` 两层（命名如 `--surface-card`, `--text-muted` 等）
2. 把六个文件里的 `html[data-theme="dark"]` 块**整段删掉**
3. 复用已有 token；`wb-priority.css` 已有 `--po-pri-*` 则删冗余 dark
4. **吸收 Antigravity 首页版本卡深色视觉意图**——柔和对比、链接色用 token 实现，不堆选择器

**建议拆 commit：** follow/home 一笔（视觉最重），其余 4 个一笔。

**关键风险：** `follow.css` 706 行超 500 上限 + 还要删 dark 块。两种解法：
- (a) 拆分 follow.css 为 follow-base.css + follow-extensions.css（侵入大）
- (b) 直接进 baseline（脚本拒绝）
- (c) 留 legacy（标 BLOCKED BY EXISTING BASELINE）

建议 (c) 最稳。

## 6. 不做清单（按 plan 锁死）

- ❌ **不 push**（除非用户明确口头确认）
- ❌ **不 commit `docs/PRD/`**（Plan 红线）
- ❌ **不引入 Main 整页 `testtask.html`**（已是默认路由禁）
- ❌ **不假成功建 zt_testtask**（T6，Main 无 API，按 plan 后置）
- ❌ **不混三轨 WIP**（看板 / gofmt / profile / linkstory / dark 各 commit）

## 7. T7 禅道联调（建议跳过或文档化）

- `127.0.0.1:8080` 历史不可达
- 仅验证 HTTP 错误路径 + 单测
- 在 commit message 或 PR description 注明「zentao dev 不可达，仅 API 错误路径验证」

## 8. Gate 现状速查（基于上一会话末态）

| Gate | 状态 |
|---|---|
| `check-test` | PASS |
| `check-go-vet` | PASS |
| `check-architecture` | PASS |
| `check-secrets` | PASS |
| `check-whitespace` | PASS |
| `check-patterns` | PASS（advisory） |
| `check-gofmt` | **BLOCKED BY EXISTING BASELINE**（看板 WIP 未 `gofmt -w`） |
| `check-file-length` | **BLOCKED BY EXISTING BASELINE**（docs/Demo/* 多份 + po-profile 既有 + 看板 schedule 增长） |

## 9. 推荐下一步顺序

1. **T4 linkstory 入口**（小，1 commit，估计 <300 行 diff）
2. **看板 WIP 收口**（独立 commit，先 `gofmt -w` 同步 baseline，再写 commit）
3. **T5 dark tokens**（大，2 commit）
4. **T1 push**：等用户口头确认 `git push -u origin HEAD`

## 10. 关键参考路径

- 总计划：`.cursor/plans/release-remaining-master-plan-20260911.md`
- Phase C 已落地：`.cursor/plans/absorb-main-phase-c-cursor-prompt-20260911.md`
- profile 设计原档：`.cursor/plans/profile-ux-antigravity-prompt-20260911.md`
- Frontend 规范：`docs/engineering/frontend.md` §5 Theme Contract
- Renderer：`internal/pkg/render/render.go`（page = sibling dir 模式）
- Theme token 文件：`web/static/css/layout/variables.css`
- linkstory 入口签名：`web/static/js/po/linkstory.js:391` `openPoLinkstoryModal({ buildId })`
- 提测弹窗 DOM：`web/templates/components/po_testtask_modal.html`

## 11. 一行启动口令

下一会话开头直接说：

> 「继续 `.cursor/plans/release-po-handoff-next-session-20260911.md`：先做 T4，再收看板 WIP，最后做 T5」

即可按本文 §4 / §3 / §5 顺序推进。
