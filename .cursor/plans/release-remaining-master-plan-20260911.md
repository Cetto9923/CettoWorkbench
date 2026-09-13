# Master Plan：Release 剩余工作一次做完（给 Cursor / Codex / Antigravity）

> 仓库：`/Users/yuyan9923/GitHub/workbench-claude-po`  
> 分支：`release/po-integrate-main-202609`  
> 参考 Main：`/Users/yuyan9923/GitHub/workbench` @ `88a64011`（只读）  
> 本地：`WORKBENCH_MODE=dev WORKBENCH_APP_ADDR=:8090 WORKBENCH_SESSION_COOKIENAME=wb_session_po_8090`，账号 `003030` / `123456`  
> 规范：`docs/engineering/frontend.md` Theme Contract；`button-system` / `action-btn`；选人 `initUserPicker`；禅道写操作用 `DoAs(当前登录 account)`  
> **默认不要 push**，除非本 Plan 某 Track 明确写「可 push」且验收方口头确认；**禁止 commit `docs/PRD/`**；**禁止动 `.wip/`**。

---

## 0. 当前基线（先读再改）

### 已完成（本地 ahead，可能尚未 push）

| Commit | 内容 |
|--------|------|
| `16230a37` | zentao `DoAs`/`Do` + APILog |
| `d76fe2b4` | render 兄弟模板（linkstory fragment P0） |
| `a940fbdc` | build/LinkStory 模块接线 |
| `1ed0eb2c` | testtask executions + CreateBuilds via DoAs |
| `05774d60` | 提测弹窗接真 builds API，去掉静态演示 |

执行：`git log origin/release/po-integrate-main-202609..HEAD --oneline` 核对上述 5 笔仍在。

### 工作区旁路 WIP（**未提交，必须分轨**）

典型路径（以 `git status` 为准）：

- **看板轨**：`serviceboard.go`、`board_task_transition_test.go`、`service.go`（po）、`task_client.go`（UpdateTask）、`workboard.js`、`board.css`、`deploy/zentao/routes.php`
- **排期 gofmt 轨**：`schedule/form.go`、`schedule/form_batch_cap_test.go`
- **资料轨**：`profile/handler.go`、`po-profile.js/css`、`layout/header.html`、`layout/base.html`、`bottomtabbar.js`、**`web/templates/profile/index.html` 可能已删**、`docs/Demo/profile-modal-redesign.html`

**铁律**：每一轨单独 commit；禁止把看板/gofmt/profile 混进吸收或 dark 提交。

### 已知诚实缺口

1. LinkStory **能力在、产品入口弱**（首页版本卡无「关联需求」；可靠 `openPoLinkstoryModal({buildId})` / 深链）。  
2. 提测弹窗已能 **建 Build**，**尚未**创建禅道 `zt_testtask`（Main 也无）。  
3. 六页仍有大量 `html[data-theme="dark"]` 拷贝（Theme Contract §5）。  
4. `/profile` 体验偏旧；已有弹窗 API `openProfileModal` 与 Demo。

---

## 执行总序（按 Track 顺序，可停）

```
T0 盘点与卫生
 → T1 push 已完成的 5 笔（可选，需确认）
 → T2 旁路 WIP 分轨收口（看板 / gofmt / 先别丢 profile）
 → T3 个人资料弹窗优先 + 视觉对齐
 → T4 LinkStory / 提测产品入口补齐
 → T5 六页 dark → token（含合规吸收首页版本卡深色意图）
 → T6 Phase D：调研并实现「真创建测试单」（独立，可后置）
 → T7 禅道 E2E 联调清单
```

每个 Track 结束：编译相关包、必要单测、`:8090` 冒烟要点、**单独 commit**；未确认不 push。

---

# T0 — 盘点（30 分钟，只读+报告）

1. `git status -sb`、`git log --oneline -8`、`git diff --stat`  
2. 列出三轨文件清单；确认 `ProfileHandler` 仍在 `bootstrap` + `routes`（A+B 曾误删已修，勿再丢）  
3. `rg '静态演示|未提交后端' web/static/js/po/testtask*.js` 应为 0  
4. 输出「本 Plan 将改 / 不改」清单后进入 T1/T2  

**不要**在 T0 commit。

---

# T1 — Push 吸收主线（可选）

**仅当用户确认 push 时执行。**

```bash
git push -u origin HEAD
```

范围仅已存在的 5 个 commit。旁路 WIP 必须仍在 working tree 未进 commit。

若用户说不 push：跳过，在报告写「ahead N」。

---

# T2 — 旁路 WIP 分轨（先清干净工作区）

## T2-a 排期 gofmt（最小）

- 仅对 `internal/module/schedule/form.go`、`form_batch_cap_test.go` 跑 `gofmt -w`  
- 单 commit：`chore(schedule): gofmt form batch cap files`  
- 不改逻辑  

## T2-b 看板拖拽轨（独立验收）

- 审 `UpdateTask` / `serviceboard` / `workboard.js` / `deploy/zentao/routes.php`  
- 补齐 `gofmt`；保证 `go test` 相关包过  
- 行为：任务看板拖拽改状态仍通；标题/ID 外链不回归  
- commit：`feat(board): …`（按实际 diff 写准）  
- **若未完成或不稳：stash 或保留 WIP，不要强行塞进别的 Track**

## T2-c profile 文件

- **不要**在本轨删除资料能力；若 `profile/index.html` 已删，T3 必须恢复深链页或改为仅弹窗+占位页（见 T3）  
- 看板/gofmt 提交时 **勿** `git add` profile/layout 文件  

---

# T3 — 个人资料：弹窗优先 + PO 工作台风（必做）

详细依据：`.cursor/plans/profile-ux-antigravity-prompt-20260911.md`  
视觉参考：`docs/Demo/profile-modal-redesign.html`（若存在）

### 产品

- **主路径**：顶栏「个人资料」→ `openProfileModal()`，**不离开当前页**  
- **次路径**：`/profile` 深链仍可用；与弹窗 **同一套渲染**（`profilePageBody` / `profileModalBody`）  
- 底栏：**不要**把个人资料当常驻工作 Tab  
- **禁止**删除 `internal/module/profile` 与 `/api/profile*`  

### 必做

1. `header.html`：入口改调 `openProfileModal`；无脚本时 `href=/profile` 降级；`bi-*` → `fas`  
2. PO shell 页能加载 `po-profile.js`（layout 统一引入或等价懒加载）  
3. 视觉对齐 `todos`/`follow`：`workspace-header`、token 色、`action-btn`；密码区默认折叠  
4. Theme Contract：无新增页面 dark 拷贝  
5. 若 `index.html` 被删：恢复最小深链页（可极简，只挂 host + 脚本）或 handler 改为渲染 modal-only 壳——**必须**让 `/profile` 200  

### 验收

- [ ] `/home` 点资料 → 弹窗，URL 仍 home  
- [ ] `/profile` 不 404；保存邮箱/角色/小组、改密可用  
- [ ] 底栏无多余「个人资料」页签（弹窗路径）  
- [ ] Light/Dark 可读  

### Commit

1. `fix(ui): open profile from header as modal`  
2. `fix(ui): restyle profile to personal-workspace`  

---

# T4 — 产品入口补齐（LinkStory + 提测后挂钩）

### LinkStory

1. 提测弹窗建版成功后：若返回 `buildId`，提供「关联研发需求」按钮 → `openPoLinkstoryModal({ buildId })`  
2. 首页/排期版本卡：有稳定 `buildId` 处加次要按钮「关联需求」（token/`action-btn`）  
3. 无 buildId 时 toast 提示先建版本  

### 提测文案

- 主按钮成功文案保持诚实：「已同步 N 个版本」；副文案可写「测试单创建后续迭代」  
- 勿宣称已创建禅道测试单（除非 T6 完成）  

### 验收

- [ ] 建版成功 → 一键打开 LinkStory  
- [ ] 深链 `/builds/{id}/linkstory` 仍可用  
- [ ] 禅道未启时错误可读  

### Commit

`feat(ui): hook linkstory entry after createBuilds / version cards`

---

# T5 — 六页 dark → semantic token（Theme Contract §5）

### 范围文件

`follow.css`、`wb-priority.css`、`po-urge.css`、`home.css`、`notice.css`、`done.css`（以 `rg 'html\[data-theme=.dark.\]' web/static/css/po` 为准）

### 方法

1. 缺的语义补进 `variables.css` 的 light/dark **token 层**（禁止 `--dark-bg` 外观命名）  
2. 删除页面级 `html[data-theme="dark"] .component` 拷贝  
3. **吸收**此前 Antigravity 首页版本卡深色**视觉意图**（柔和对比、链接色），但用 token 实现，禁止再堆 dark 选择器  
4. `wb-priority` 已有 `--po-pri-*` 则删冗余 dark  
5. 禁止 `filter: invert`  

### 验收

- [ ] 六文件 dark 选择器清零（极少数例外须注释+报告）  
- [ ] `/home` `/follow` `/done` `/notice` + 催办 Dark 无大块白底  
- [ ] 优先级色语义不变  

### Commit

`fix(ui): migrate PO page dark overrides into theme tokens`

可拆 follow/home 与其余两笔。

---

# T6 — Phase D：真创建禅道测试单（可后置，独立）

Main **无**现成 CreateTestTask。本 Track 是产品+研发调研落地：

1. 查禅道 REST：创建 testtask 的 path/字段（项目/执行/版本/名称/负责人/起止等）  
2. 在 `internal/module/testtask/gateway.go` 增加 `createTestTask` + `DoAs`  
3. `POST /demands/:id/testtask`（或 `/submit`）校验并创建；弹窗「提交提测」改为调此 API（版本已建则带 buildId）  
4. 失败映射可读错误  
5. 单测 + 禅道联调  

若 API 不可用/权限不够：**停止并报告**，不要假成功。

### Commit

`feat(testtask): create zentao testtask via DoAs`（仅 API 确认后）

---

# T7 — 联调清单（每个相关 Track 后跑）

禅道 `127.0.0.1:8080` 可达时：

1. 登录工作台 → 打开提测弹窗 → 选执行 → 新建版本 → 禅道可见 Build  
2. 「关联需求」→ LinkStory 勾选关联/解绑  
3. （T6 后）提交提测 → 禅道可见测试单  
4. 关注星标、澄清、催办冒烟不回归  
5. 资料弹窗保存  

禅道不可达：只验 HTTP 错误路径 + 单测，报告注明。

---

## 全局禁止

- 整仓 merge Main；用目录移植 + 接线  
- Main 整页 `testtask.html` 覆盖弹窗  
- 覆盖 `zentao` 领域客户端（demand/task/follow）  
- 页面 dark 选择器拷贝；第二套主题  
- 删除 metrics/profile/query 模块  
- 把三轨 WIP 捆成一个巨型 commit  
- 未经确认 `git push --force`  

---

## 建议 commit 总览（按轨）

1. （已有）A+B+C 五笔 — T1 push  
2. `chore(schedule): gofmt …`  
3. `feat(board): …`（若收口）  
4. `fix(ui): profile modal …` ×2  
5. `feat(ui): linkstory entry hooks …`  
6. `fix(ui): dark tokens migration …`（可拆）  
7. `feat(testtask): create zentao testtask …`（T6）  

---

## 最终回报模板

```markdown
## 基线
- ahead / pushed:
- WIP 三轨处理结果:

## Track 完成情况
| Track | 状态 | commits | 冒烟 |

## 未做与原因
## 风险
## 是否建议 push 剩余 commits
```

---

## 开工指令（给 Agent）

1. 先 T0 盘点，输出文件清单与风险。  
2. 严格按 T2→T3→T4→T5 顺序清 WIP 并交付体验（T1 仅在用户确认 push 时做）。  
3. T6 仅在用户要「真测试单」或 T0 发现禅道 API 已清晰时启动，否则标后置。  
4. 每 Track 结束自检验收勾选；旁路与主线 commit 分离。  
5. 配额紧时优先：**T2-a gofmt → T3 profile → T4 入口 → T5 dark**；T6 可砍。
