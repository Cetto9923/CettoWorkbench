# Cursor 提词：Phase A+B 吸收后回归（不改功能，只验）

> 仓库：`/Users/yuyan9923/GitHub/workbench-claude-po`  
> 分支：`release/po-integrate-main-202609`  
> 背景：已落地未提交的 A+B（`DoAs`/`APILog` + `internal/module/build` + LinkStory）。工作区可能还混有**旁路 WIP**（看板拖拽 / `task_client.UpdateTask` / schedule gofmt / `deploy/zentao/routes.php`）——回归时注明哪些失败与 A+B 无关。  
> 本地：`WORKBENCH_MODE=dev WORKBENCH_APP_ADDR=:8090 WORKBENCH_SESSION_COOKIENAME=wb_session_po_8090`，账号 `003030` / `123456`  
> **本提词只测试与报告，默认不改代码、不 commit、不 push。** 若发现 P0 阻断（编译失败/核心页 500），可做最小修复并单列说明。

---

## 0. 先隔离工作区（必做）

1. `git status -sb` + `git diff --stat`  
2. 标出两类文件：  
   - **A+B 核心**：`internal/pkg/zentao/{client_do,apilog,client_test}.go`、`internal/module/build/**`、`internal/model/zentao/{build,module,product,productplan,projectstory}.go`、bootstrap/routes/perm/templates/render、`web/**/linkstory.*`、`home.html` 中 linkstory 挂载行  
   - **旁路 WIP**：`serviceboard.go`、`board_task_transition_test.go`、`task_client.go` 的 `UpdateTask`、`workboard.js`、`board.css`、`schedule/form*.go`、`deploy/zentao/routes.php` 等  
3. 报告里分开写；若旁路导致测试红，用 `git stash push -k -u -- <旁路路径>` **临时**挪开再跑一轮对照（测完 pop 回来），并写明对比结果。

---

## 1. 自动门禁

```bash
go test ./internal/pkg/zentao/... ./internal/module/build/... ./internal/bootstrap/... ./internal/server/...
make check-frontend-test
go build -o /tmp/wb-ab-regress ./cmd/workbench   # 或仓库实际 main 包路径
```

可选（注明环境依赖）：

```bash
go test ./internal/module/po/...
# make check 可能被既有 gofmt 基线挡住——记录文件名，勿为过门禁大改格式 unless 用户允许
```

**通过标准**：zentao / build / bootstrap / server / frontend 全绿；`go build` 成功。

---

## 2. 现有功能冒烟（确认 A+B 无倒退）

启动或复用 `:8090`，登录 `003030`：

| # | 页面/动作 | 期望 |
|---|-----------|------|
| 1 | `/home` | 200，价值流/列表可刷，无 JS 控制台红错 |
| 2 | 打开需求澄清弹窗 | 选人 UserPicker 可用；不因 home 挂了 linkstory 脚本而挂 |
| 3 | 催办弹窗 | 可打开、预览区正常 |
| 4 | 提测弹窗 | 仍可打开；提交仍可为「静态演示」toast（Phase C 未做，**不算失败**） |
| 5 | `/follow` 业需星标取关/再关注 | 成功（走原 `FollowDemandObject`，不经 `DoAs`） |
| 6 | `/todos` 芯片切换 | 列表刷新正常 |
| 7 | `/done` 或通知 | 可打开详情抽屉 |
| 8 | 排期页（若有） | 选人仍为 UserPicker；无 500 |

每项记：通过 / 失败 + 一句话现象 + 是否怀疑旁路 WIP。

---

## 3. 吸收能力验收（A+B 实际效果）

### 3.1 基础设施

- 代码存在：`DoAs`、`Do`、`InitAPILog`（bootstrap 已调）  
- 用现有单测即可；若有禅道 API：随便触发一次将来走 `DoAs` 的路径（见下），检查 log.dir 下 API 日志有记录且 **无 password 明文**

### 3.2 LinkStory 能力

1. **深链**：登录后浏览器打开或 curl 带 cookie：  
   `GET /builds/{真实buildId}/linkstory`  
   - 有权限/已登录：返回 HTML 片段（`po/linkstory_body`），含检索与列表  
   - `buildId=0` 或非法：4xx JSON message  
2. **首页挂载**：`/home` 源码含 `linkstory.css`、`po/linkstory` 模板、`linkstory.js`；控制台执行：  
   `typeof openPoLinkstoryModal === 'function'` → `true`  
3. **打开弹窗**（控制台）：  
   `openPoLinkstoryModal({ buildId: '<真实ID>' })`  
   - 弹窗出现并加载列表；无 CSRF 导致 POST 失败时应用 `appFetch`  
4. **关联/解绑**（若禅道可写）：勾选 → 关联成功 toast；禅道侧可核对。失败时错误文案可读。  
5. **入口缺口（预期）**：首页版本卡**可能仍无**「关联需求」按钮；只保证 API + `openPoLinkstoryModal` 可用。报告写清「产品入口未挂 = 能力已进、入口待接（Phase C/排期）」，**不要当成回归失败去改一堆 UI**（除非用户另加范围）。

### 3.3 权限

- `BuildLinkStory` 在 perm 常量中；未登录访问 `/builds/...` 应跳登录或 401/403  
- 已登录默认策略以代码为准（当前常量注释写「已登录即可」）——测实际行为并记下来

---

## 4. 明确非本轮失败

- 提测提交仍是演示 toast  
- `make check` 因旁路/旧 gofmt 基线红  
- 六页 dark token 债  
- 无 buildId 时无法演示关联（需从库里找一条 `zt_build` id）

找 buildId 示例（只读 SQL 或现有 admin/禅道 UI）：任取一个未删除 build 的数字 id。

---

## 5. 回报格式（交给验收方）

```markdown
## 工作区隔离
- A+B 文件：…
- 旁路 WIP：…
- 是否 stash 旁路复测：是/否 → 差异

## 自动门禁
- zentao/build/bootstrap/server：PASS/FAIL
- frontend：PASS/FAIL
- go build：PASS/FAIL
- po 包（可选）：…

## 现有功能冒烟
| 项 | 结果 | 备注 |

## 吸收效果
- DoAs/APILog：…
- GET /builds/{id}/linkstory：…
- openPoLinkstoryModal：…
- 关联/解绑：…（或环境不可写）
- 产品入口：未挂/已挂

## 风险与建议
- P0/P1 列表（若有）
- 是否可安全 commit A+B（旁路是否剥离）
```

停。不要开始 Phase C，不要 push。
