# Cursor 提词：吸收 Main → Release（Phase A + B）

> 目标仓库（改这里）：`/Users/yuyan9923/GitHub/workbench-claude-po`  
> 分支：`release/po-integrate-main-202609`（先 `git status` / `git pull` 确认干净或可续）  
> 参考仓库（只读对照，禁止在参考仓提交）：`/Users/yuyan9923/GitHub/workbench` @ `88a64011`（`main`）  
> 总计划：`.cursor/plans/absorb-main-into-release-20260911.md`  
> 本地验：`WORKBENCH_MODE=dev WORKBENCH_APP_ADDR=:8090 WORKBENCH_SESSION_COOKIENAME=wb_session_po_8090`，账号 `003030` / `123456`  
> **不要 push**；**不要改 `docs/PRD/`**；**不要动 `.wip/`**；遵守 AGENTS.md / `docs/engineering/frontend.md` Theme Contract。

---

## 本轮范围（只做 A+B）

| Phase | 内容 |
|-------|------|
| **A** | 为 Release 的 `internal/pkg/zentao` **增量**合并通用 `Do` + `APILog`，不破坏现有 demand/task/follow |
| **B** | 从 Main **整模块引入** `build` + LinkStory 前端，挂路由/权限/bootstrap，UI 对齐 Release 设计系统 |

## 明确不做（Phase C / 禁区）

- **不要改**提测弹窗闭环去接 executions/CreateBuilds（那是 Phase C）。
- **不要**引入或切换到 Main 的整页 `web/templates/po/testtask.html` / 用 Main `testtask.js` 覆盖 Release。
- **不要**用 Main 的 `internal/pkg/zentao/client.go` **整文件覆盖** Release（Release 已有 `GetUserToken` + `demand_client.go` / `task_client.go` / follow / issue gateway）。
- **不要**删除 Release 独有模块：`metrics` / `profile` / `query`。
- **不要**新增页面级 `html[data-theme="dark"] .xxx` 拷贝；LinkStory 颜色只用 semantic token。
- **不要**动交付弹窗三文件（若仍存在并行任务）。

读完先输出：将改文件清单 + Token 设计选择（见 A）+ 风险，再动手。

---

# Phase A — `Do` + `APILog`（基础设施）

## 现状（必须先读）

**Release**（`workbench-claude-po`）：

- `internal/pkg/zentao/client.go`：`NewClient(baseURL string)`、`apiURL`、`GetUserToken(ctx, account)`  
- 领域调用在 `demand_client.go` / `task_client.go` 等：**各自** `GetUserToken` + `http.NewRequest` + `c.httpClient.Do`，**没有**统一 `Client.Do`  
- **没有** `apilog.go`

**Main**（`workbench`）：

- `client.go`：`NewClient(cfg config.ZentaoConfig)`，服务账号 `GetToken` + 通用 `Do` / `doRaw`，带 401 清 token 重试  
- `apilog.go`：`InitAPILog` / 写耗时与请求摘要  
- `bootstrap`：`zentaopkg.InitAPILog(cfg)`；build 使用 `zentaopkg.API()` 服务端 Client

## 必做设计（写死）

1. **保留** Release 的 `GetUserToken` 与所有现有领域方法签名/行为。  
2. **新增**统一出站能力，供 build gateway 使用，推荐二选一（选 A1，除非你有证据必须服务账号）：

### 推荐 A1 — 用户态 `DoAs`（与 Release 一致）

```text
DoAs(ctx, account, method, path, body, out any) error
```

- 内部：`GetUserToken(ctx, account)` → 发 REST → 解析 JSON 到 `out`  
- 401/鉴权失败：清该用户缓存（若有）并重试一次  
- path 规则复用现有 `apiURL`  
- 成功/失败都走 APILog（若已 Init）

可选再提供：

```text
Do(ctx, method, path, body, out) error  // 使用配置里的服务账号 Token（对齐 Main GetToken）
```

仅当禅道 Build API **必须**服务账号时才在 build 里用 `Do`；默认 LinkStory/建版本优先 `DoAs(当前登录 account)`。

### 禁止

- 删掉或改挂 `ClarifyDemand` / `FollowDemandObject` / `ReviewDemand` 等现有方法  
- 把 Main `NewClient(cfg)` 直接替换导致 Release `NewClient(baseURL)` 与 `client_test` 全挂——若需从 config 构造，用 **新函数**如 `NewClientFromConfig(cfg)`，旧 `NewClient` 保留

3. **拷贝并适配** Main `apilog.go` → Release `internal/pkg/zentao/apilog.go`  
   - 在 `internal/bootstrap/bootstrap.go` 启动处调用 `InitAPILog(cfg)`（对照 Main 位置）  
   - `log.dir` 为空则不写文件（与 Main 行为一致）  
   - `Do`/`DoAs` 内记录 method/path/耗时/状态；**不要**把 password 打进日志

4. 单测：扩展 `client_test.go`（apiURL 回归 + DoAs 对 httptest mock 的 200/401 重试）；现有 zentao 测试全绿。

## Phase A 验收

- [ ] `rg 'func \(c \*Client\) Do' internal/pkg/zentao` 有 `Do` 或 `DoAs`  
- [ ] 存在 `apilog.go`；bootstrap 有 `InitAPILog`  
- [ ] `go test ./internal/pkg/zentao/...` 通过  
- [ ] 手动：登录后走一遍关注取关或打开澄清（不回归）

---

# Phase B — Build + LinkStory 模块

## 从 Main 移植的清单（对照拷贝，再改接线）

### 后端

- 整个目录：`internal/module/build/`（`form.go` `handler.go` `service.go` `repo.go` `reposearch.go` `searchcond.go` `searchcond_test.go` `searchfields.go` `gateway.go` `labels.go`）  
- 权限：`internal/pkg/perm/constants.go` 增加 `BuildLinkStory = "build:linkstory"` 及注册表项（照 Main）  
- 模板常量：`internal/constants/templates.go` 增加 `TEMPLATE_PO_LINKSTORY = "po/linkstory"`（若 Release 有该文件）  
- 模型：Main 有而 Release 缺的 `internal/model/zentao/build.go`、`projectstory.go`、`productplan.go`、`module.go` 等——**只补 build 编译所需**，不要无差别刷模型  
- `gateway.go`：改为调用 Phase A 的 `Do`/`DoAs`（注入 `*zentao.Client`），路径保持 Main：  
  - `POST /build/:id/linkstories`  
  - `POST /build/:id/unlinkstories`

### 接线（对照 Main `bootstrap.go` + `routes.go`）

Release 已有 testtask 接线范式（`bootstrap.go` 约 168–170 行、`routes.go` 的 `TesttaskHandler`）。按同样方式：

1. `bootstrap`：`build.NewRepo` / `NewService(..., zentaoClient, log)` / `NewHandler`  
2. `RouteDeps` 增加 `BuildHandler *build.Handler`  
3. `registerRoutes`：`deps.BuildHandler.RegisterRoutes(po)`（与 Main 一样挂在已登录 po group）  
4. handlers 装配处把 BuildHandler 塞进 deps（搜 Release 里 TesttaskHandler 赋值点一并加）

### 前端

- `web/templates/po/linkstory.html`（含 `po/linkstory_body` fragment）  
- `web/static/js/po/linkstory.js`  
- `web/static/css/po/linkstory.css`  
- 在实际打开入口的父页面按需 `<link>` / `<script>`（对照 Main 谁加载了这些资源；Release 用 `asset` 助手）

### UI 对齐 Release（强制）

1. 颜色：硬编码 `#hex` → `var(--color-*)`；禁止新暗色选择器拷贝  
2. 按钮：主/次操作优先 `action-btn` / `action-btn primary`（`button-system.css` 已全局引入）  
3. 若有选人：`initUserPicker`，不要新造 autocomplete  
4. 弹层结构能靠 `batch-modal` / 现有 modal 规范则靠拢，但**不要**为对齐而重写检索业务逻辑

### 入口

- 实现 Main 已有路由即可：`GET /builds/:id/linkstory`、`POST …/linkstories`、`POST …/unlinkstories`  
- 若 Main 在排期/版本卡有「关联需求」按钮，把**同等入口**接到 Release 对应页（排期或首页版本卡）；找不到稳定挂点时：先保证深链 `/builds/{id}/linkstory` 在登录+权限下可打开，并在 PR 说明「入口挂点待产品定」——**不要**瞎改提测弹窗当入口（留给 Phase C）

## Phase B 验收

- [ ] `go test ./internal/module/build/...` 及 `./internal/pkg/perm/...` 相关通过  
- [ ] 编译启动 `:8090`；持权账号打开 LinkStory 片段/页；检索 + 关联/解绑（禅道可配时）或至少网关错误信息清晰  
- [ ] Light/Dark 下 LinkStory 无大块未 token 化白底  
- [ ] `go test ./internal/pkg/zentao/...` 仍绿；澄清/关注冒烟不挂  
- [ ] **无** Phase C 文件被误改：`web/static/js/po/testtask.js` 的「静态演示」toast 可仍在；不要在本轮接 builds API

---

# 提交切分

1. `feat(zentao): add DoAs/Do + APILog without breaking domain clients`  
2. `feat(build): port linkstory module from main and wire routes`  
3. （若有）`fix(ui): align linkstory with PO design tokens`

每 commit 可编译；信息写清来源 `workbench@88a64011`。

---

# 门禁与回报

```text
go test ./internal/pkg/zentao/... ./internal/module/build/...
# 再跑一圈被接线碰到的 bootstrap/server 测试（若有）
```

回报验收方：

1. Token 方案选了 A1 `DoAs` 还是服务账号 `Do`（及原因）  
2. 新增/修改文件列表  
3. LinkStory 入口如何打开  
4. 未做项（Phase C、zt_testtask 真创建、六页 dark token）  
5. 确认 testtask 弹窗/Main 整页未被引入  

---

# 给 Agent 的开工顺序

1. 只读打开 Main 的 `zentao/client.go`、`apilog.go`、`module/build/*`、`bootstrap` 中 build/APILog 段、`perm`、`templates`、`linkstory.*`  
2. 只读打开 Release 对应文件，确认差异  
3. 实施 Phase A → 测试  
4. 实施 Phase B → 测试  
5. 停：不要开始 Phase C（executions / CreateBuilds / 改 testtask.js 提交逻辑）
