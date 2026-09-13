# Cursor 提词：吸收 Main → Release（Phase C：提测弹窗接真后端）

> 目标仓：`/Users/yuyan9923/GitHub/workbench-claude-po`  
> 分支：`release/po-integrate-main-202609`（A+B 已提交：`16230a37` / `d76fe2b4` / `a940fbdc`，可先 `git pull` 确认）  
> 参考仓（只读）：`/Users/yuyan9923/GitHub/workbench` @ `88a64011`  
> 总计划：`.cursor/plans/absorb-main-into-release-20260911.md`  
> 本地：`WORKBENCH_MODE=dev WORKBENCH_APP_ADDR=:8090 WORKBENCH_SESSION_COOKIENAME=wb_session_po_8090`，`003030` / `123456`  
> **不要 push**；**不要改 `docs/PRD/`**；**不要动旁路 WIP**（看板 `serviceboard`/`UpdateTask`/`workboard`、`schedule` gofmt、`deploy/zentao/routes.php`、未提交的 profile 改造）。

---

## 目标

**保留 Release 四步弹窗 UI**（`po_testtask_modal.html` + `testtask.css` token/`action-btn`），把 Main 已有能力接进来：

1. `GET /products/:id/executions` — 执行下拉  
2. `POST /demands/:id/testtask/builds` — 禅道创建 Build（`createProjectBuild`）  
3. 去掉「静态演示」toast，第 2 步/保存版本走真 API  

**诚实边界（写死）**：Main **也没有**创建 `zt_testtask` 的 API。本轮 **不要假装**「提交提测」已落测试单；提交按钮语义改为：

- 已选/已建版本校验通过 → 调 CreateBuilds（若有「新建版本」项）→ toast「版本已同步禅道」或展示 buildId  
- 若产品文案仍叫「提交提测」，副文案注明「本阶段同步版本；测试单创建待 Phase D」  
- 「保存草稿」：可先本地/sessionStorage，或同样只保存版本信息；**不要**空 toast 伪装成功  

---

## 禁止

- 引入 Main 整页 `web/templates/po/testtask.html` 当主路径，或用 Main `testtask.js`（1000+ 行）整文件覆盖 Release `testtask.js`  
- 破坏澄清/催办/交付弹窗；交付三文件若仍有并行任务勿碰  
- 用 Main `zentao.Client.Do`（服务账号）替换；**写操作走 `DoAs(当前登录 account)`**（与 A+B LinkStory 一致）  
- 新增页面级 `html[data-theme="dark"]` 拷贝  
- 删除 `/profile` 接线或旁路看板改动混进本 PR  

读完先输出：将改文件清单 + CreateBuilds 的 DoAs 方案 + 提交按钮文案策略，再动手。

---

## 从 Main 移植（对照拷贝后改）

### 后端（`internal/module/testtask/`）

| Main 文件 | 动作 |
|-----------|------|
| `execution.go` | 移植 |
| `repo_execution.go` | 移植 |
| `gateway.go` | 移植；`createProjectBuild` 改为 `client.DoAs(ctx, account, POST, /projects/{id}/builds, …)`，account 来自 actor |
| `form.go` | **增量合并** `CreateBuildsReq` / `CreateBuildItem` / `ExecutionOption` 等，勿丢掉 Release 已有字段 |
| `service.go` | 合并 `ListProductExecutions`、`CreateBuilds`；注入 `*zentao.Client`（`DefaultClient()`） |
| `handler.go` | 增加两条路由；保留现有 `GET /demands/:id/testtask` |
| `labels.go` | 按需合并，跑通现有 `handler_test` |

### bootstrap

- `testtask.NewService(...)` 若需 client，传入 `zentaopkg.DefaultClient()`（对照 build 模块）。

### 前端（只改 Release 弹窗脚本/模板）

- `web/static/js/po/testtask.js`  
  - 打开弹窗/进入步骤 2：按产品 ID 拉 `/products/:id/executions` 填下拉  
  - 「新建版本」保存或主按钮：`POST /demands/:id/testtask/builds`，body 对齐 Main `CreateBuildsReq`  
  - 使用 `appFetch` + CSRF  
  - 成功后可把返回的 `buildId` 写入表单隐藏域，并可选调用已有 `openPoLinkstoryModal({ buildId })`（**可选**，作为入口挂钩；不做大改首页版本卡）  
- `po_testtask_modal.html`：若缺 execution/build 字段名，按 JS 需要最小补齐；保持四步结构与 `action-btn`  
- CSS：沿用现有 token，不引入 Main 整页样式  

### 单测

- 扩展/移植 Main 相关单测（labels/CreateBuilds validate）  
- `go test ./internal/module/testtask/... ./internal/pkg/zentao/...`  
- 前端若有 testtask 单测则更新；没有不硬造  

---

## 验收

- [ ] Release 弹窗仍是主入口（详情/首页打开 `openPoSubmitTestModal`）  
- [ ] 有产品时执行下拉有数据（或空列表友好提示）  
- [ ] 新建版本 → 禅道可达时出现 Build；不可达时错误可读（非演示 toast）  
- [ ] 代码中无「静态演示，未提交后端」字符串  
- [ ] **无** Main 整页 testtask 成为默认路由  
- [ ] `DoAs` 用于建版本；领域 demand/follow 未回归  
- [ ] 旁路 WIP / profile 未提交改动未混入  
- [ ] 不 push  

建议 commit：

1. `feat(testtask): port executions + createBuilds via DoAs`  
2. `feat(testtask): wire po modal to real builds API`  

---

## 环境提示

- 禅道 dev：`127.0.0.1:8080`（回归报告曾未启）；联调前先确认可访问，否则只验 API 错误路径 + 单测。  
- LinkStory：`openPoLinkstoryModal` 已在 home；建版成功后可选用。  

## 回报

改动文件、按钮语义、是否挂 LinkStory、禅道联调结果（或跳过原因）。
