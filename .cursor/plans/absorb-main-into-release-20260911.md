# 吸收计划：Main (`workbench@88a64011`) → Release (`workbench-claude-po` / `release/po-integrate-main-202609`)

> 核对日：2026-09-11  
> 原则：**Release 体验与规范为准**（弹窗提测、Theme Token、UserPicker、button-system）；**Main 补能力**（Build/LinkStory、执行下拉、禅道建版本、APILog）。禁止整页覆盖、禁止倒退已上线的 PO 闭环。

---

## 0. 双方现状（纠偏 Gemini）

| 能力 | Release | Main | 结论 |
|------|---------|------|------|
| Build + LinkStory 整模块 | **无** | `internal/module/build/` + `linkstory.*` | **完整吸收** |
| 提测 UI | **四步弹窗** `po_testtask_modal` + token 化 CSS | 整页 `testtask.html` + 大 JS | **保留 Release UI** |
| 提测后端 | 仅 `GET …/testtask` 上下文；提交/草稿是 **toast 演示** | `GET executions` + `POST …/testtask/builds`（禅道建 Build） | **移植 Main 后端进弹窗** |
| 创建 `zt_testtask` | 无 | **也无**（gateway 只建 Build） | Gemini「真实创建测试单」**夸大**；本轮目标先对齐 Main 已有闭环 |
| `zentao.Client` | 已有 + demand/task/follow/issue 领域方法；用户 Token | 有通用 `Do` + 服务 Token + **APILog** | **增量合并**，勿用 Main 整文件覆盖 |
| `sqllog` | 有基础，缺 `console.go` | 有 console | 可选吸收 |
| metrics / profile / query | **有** | 无 | **勿从 Main 回退删除** |

---

## 1. 吸收原则

1. **Cherry-pick / 目录拷贝 + 接线**，不要把 Main 整仓 merge 进 Release。
2. 前端冲突时：**Release 壳（modal / tokens / action-btn / UserPicker）优先**；Main 只贡献数据契约与交互逻辑。
3. 禅道出站：统一走 `internal/pkg/zentao.Client`；新建 gateway 文件，禁止散落 `http.Client`。
4. 每阶段可独立编译、`:8090` 冒烟、**不 push 除非验收方点头**。
5. `docs/PRD/`、`.wip/` 不进 git。

---

## 2. 分阶段路径

### Phase A — 基础设施（0.5–1 天）⭐

**目标**：Main 的 API 调用能力能在 Release Client 上跑，且不破坏现有 Clarify/Follow/Review。

1. 对照 Main `client.go` 的 `Do` / `doRaw` / token 刷新，与 Release `GetUserToken` 模型做**兼容设计**：
   - Build/Testtask 网关需要的 `Do(ctx, method, path, body, out)`；
   - 优先「当前登录用户」Token（与 Release 一致）；若某接口必须服务账号，显式参数区分。
2. 引入 Main `apilog.go`（`InitAPILog` / 写文件），在 Release 启动路径接线；配置项对齐 `config`。
3. 单测：`client_test` 不回归；新增 Do + 401 重试冒烟。

**验收**：现有关注星标 / 澄清 / 评审仍通；新 `Do` 能对禅道打通一次只读或建 Build 沙箱调用。

**风险**：两套 Token 语义混用 → 必须在设计笔记写清「谁用 user token / 谁用 app token」。

---

### Phase B — Build & LinkStory（1–2 天）⭐⭐⭐⭐⭐

**目标**：Release 具备「版本关联/解绑研发需求」。

1. 拷贝 `internal/module/build/`（form/handler/service/repo/reposearch/searchcond/gateway/labels…）。
2. 模型：按需补 `internal/model/zentao/build.go` 等 Main 有而 Release 缺的。
3. 前端：`linkstory.js/css/html`；挂路由 `RegisterRoutes` + perm `BuildLinkStory`。
4. **UI 对齐 Release**：颜色改 token；按钮 `action-btn`；选人若有则 `initUserPicker`；Light/Dark 不写页面级 dark 拷贝。
5. 入口：从排期/提测/版本卡找到打开 LinkStory 的点（对齐 Main 入口，接到 Release 已有页面）。

**验收**：打开关联弹窗 → 检索 Story → 关联/解绑成功（禅道侧可查）；Dark 不花屏。

---

### Phase C — 提测弹窗接真后端（1–2 天）⭐⭐⭐⭐⭐

**目标**：保留 `po_testtask_modal`，去掉「静态演示」toast，对齐 Main 已实现能力。

1. 移植到 Release `internal/module/testtask/`：
   - `execution.go` / `repo_execution.go`
   - `gateway.go`（`createProjectBuild`）
   - handler：`GET /products/:id/executions`、`POST /demands/:id/testtask/builds`
   - form/service/labels 增量合并（勿整文件覆盖冲掉 Release 测试）
2. 改 `web/static/js/po/testtask.js`：
   - 第 2 步版本：拉执行列表、提交 CreateBuilds；
   - 「保存草稿 / 提交提测」：至少建 Build 成功有真实反馈；**若 Main 仍无 zt_testtask 创建，本阶段不要假装已建测试单**，产品文案写清「先落版本，测试单创建列入 Phase D」。
3. **不要**引入 Main 整页 `testtask.html` 作为主路径；整页仅可作对照参考。

**验收**：弹窗选执行 → 新建版本 → 禅道出现 Build；失败有明确错误；CSS 仍用 Release token。

---

### Phase D — 提测真正落 `zt_testtask`（独立产品迭代）

Main 当前也没有 → 需禅道 API 调研 + 产品字段（负责人、轮次、关联需求）。单独立项，不与 A–C 绑死。

---

### Phase E — 可选

- `sqllog/console.go` + debug 路由（开发态）。
- Main 其它小修（如 #89666 testtask 默认值）在 C 阶段对照 cherry-pick。

---

## 3. 明确不做

- 用 Main 覆盖 Release 的 follow / urge / clarify / theme / UserPicker。
- Merge 删除 Release 独有的 `metrics` / `profile` / `query`。
- 把 Antigravity 式 `html[data-theme=dark] .xxx` 大段拷进新页（LinkStory 必须一次做对 token）。

---

## 4. 建议执行顺序（本周）

```
A Client.Do + APILog
    ↓
B build/linkstory（低耦合、业务可见）
    ↓
C 弹窗接 executions + CreateBuilds
    ↓
（并行）六页 dark→token 治理续任务
    ↓
D zt_testtask 真创建（另开）
```

每阶段 1 个 commit 族；B/C 完成后可 push；A 可与 B 同 PR 若回归稳。

---

## 5. 给 Codex/Antigravity 的一句话 brief

> 从 `/Users/yuyan9923/GitHub/workbench@88a64011` **按目录移植** build 模块与 testtask 的 execution/builds 能力到 `workbench-claude-po` 的 release 分支；提测 UI **必须继续用** `po_testtask_modal`，禁止换回 Main 整页；禅道 Client **合并 Do+APILog**，禁止覆盖已有 demand/task/follow 方法；新 UI 遵守 Theme Contract，零页面 dark 拷贝。
