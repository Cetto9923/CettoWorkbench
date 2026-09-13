# 执行记录

日期：2026-09-11
分支：release/po-integrate-main-202609
基线：2b5ec597f0a0ebbf64f8b7b9a55ec379e5909218

## 已执行

- A 阶段：新增 `GET /products/:id/builds`，返回产品已有版本；新增 `GET /builds/:id/linkedstories`，批量合并版本及子版本已关联研发需求并返回标题与禅道地址。
- A 阶段：提测上下文的涉及产品、内部用户和用户展示名查询失败改为返回错误，避免错误降级为空数据。
- 保留当前分支的 `DoAs(actor.Account, ...)` 远程写入方式、现有权限中间件和禅道评审边界。
- 为新增查询及错误传播补充/更新 SQL mock 预期。
- 前端接入：提测向导第二步按产品加载已有版本，下拉区分加载中、空列表与失败状态。
- B 阶段边界收敛：产品执行、产品已有版本、版本已关联需求三个 Service 查询均显式拒绝未认证或空账号 actor；不把登录态推断交给前端隐藏逻辑。
- 为上述认证边界补充 Service 单元测试，使用业务错误码 `forbidden` 验证不会触发 Repo 或远程客户端调用。
- 角色查询按仓库现有表定义补齐 `ur.deleted`、`r.deleted` 过滤，并按 `sortOrder、id` 稳定排序，避免失效关联重新出现在组织角色中。
- 创建新版本入口先校验 `demandID` 和需求存在性，再选择禅道客户端；无效或不存在需求不会发起远程写入。
- 为创建新版本补充无效需求与不存在需求测试，并验证数据库预检失败时不会进入远程创建分支。

### C 阶段（非联调测试单批量远程写入 + 完整归属授权 + 幂等/超时/部分成功）

- form：`internal/module/testtask/form.go` 新增 `CreateTesttaskItem`、`CreateTesttasksReq`、`CreateTesttaskResult`、`CreateTesttasksResp`、`IdempotencyKey()` 同批 `(buildID, name)` 去重、`Validate()` 拒绝 `joint=1`、空批、字段必填、日期合法、`end≥begin`、`pri∈[1,4]`。
- repo：`internal/module/testtask/repo.go` 新增 `FindBuildsByIDs`（按版本 ID 批量读取产品/项目/执行，`deleted='0'`，空输入返回空 map）和 `VerifyExecutionBelongsToProject`（execution 类型 `sprint/stage/kanban` 且 `project` 匹配）。
- gateway：`internal/module/testtask/gateway.go` 新增 `createTesttaskReq`、`createdTesttask`、`testtaskCallTimeout = 10s`、`withTesttaskTimeout`、`createTesttask` 走 `DoAs(req.Owner, POST /projects/:id/testtasks, …)`。
- service：`internal/module/testtask/service.go` 新增 `PartialTesttaskError{Cause, Succeeded}` 与 `CreateTesttasks` 主流程：业务校验 → `FindDemandContext` → 同批去重 → 一次性 `FindBuildsByIDs` → 逐项 `VerifyExecutionBelongsToProject` → 逐项 `createTesttask`（10s 超时）→ 部分失败包装 `PartialTesttaskError` 透传已成功 ID；`context.DeadlineExceeded` 标记「结果未知，请勿重复提交」。
- handler：`internal/module/testtask/handler.go` 注册 `POST /demands/:id/testtask/tasks`；`CreateTesttasks` handler 用 `errors.As(&partial)` 区分部分成功与业务错误：207 + `success:false`/`cause`/`succeeded`/`data.tasks` JSON 外壳；业务错误走 `contextHTTPError`；422 携带 `errors` 字段级错误。
- zentao client 修复：`internal/pkg/zentao/client_do.go:182` 把 `%w: %v` 改为 `%w: %w`，让 `errors.Is(returnedErr, context.DeadlineExceeded)` 在超时路径命中；仅这一行，其他 zentao 文件不动。
- 测试：`internal/module/testtask/service_testtask_test.go` 新增 17 个 `CreateTesttasks` 用例，覆盖业务校验（未登录 / 空账号 / demandID=0 / joint=1 / 空批）、`DemandNotFoundStopsBeforeRemoteWrite`、归属校验（build 缺失 / 产品不一致 / execution=0 / execution 不属于项目）、`DedupesSameBuildAndName`、`PartialSuccessWrapsPartialError`（3 条第 2 条失败 → 1 条已成功 + 明确 cause）、`TimeoutMarksResultUnknown`（父 ctx 50ms）、`ValidateFieldErrors`、`ValidateRejectsEndBeforeBegin`、Handler `Returns207OnPartial` 与 `ReturnsValidationErrors`。
- sqlmock 适配：`WithArgs` 严格匹配 `n+1` 占位符（`FindBuildsByIDs` 的 n 个 ID + `deleted='0'`，`VerifyExecutionBelongsToProject` 的 id + project + `deleted='0'`）；通过 `expectBuildsByIDs` / `expectProjectBelongs` / `anyBuildArgs` 辅助函数与 `failOnCall atomic.Int32` 表达「仅在指定序号失败」。
- gofmt：本会话新增/修改的 `service.go`、`service_testtask_test.go` 已 `gofmt -w` 收敛，不修改既有基线指纹文件。

## 验证

- `go test ./internal/module/build ./internal/module/testtask ./internal/module/profile -count=1`：通过。
- `go test ./... -count=1`：通过。
- `node tests/unit/frontend/personal-list.test.js`：通过。
- `node tests/unit/frontend/auth-errors.test.js`：通过。
- `node tests/unit/frontend/ui-consistency.test.js`：通过。
- `go build ./cmd/server`：通过。
- `git diff --check`：通过。
- `make check`：退出码 1；local-artifact gate 通过后，check-gofmt 仅被既有指纹阻断，涉及 `internal/module/po/board_task_transition_test.go`、`internal/module/po/serviceboard.go`、`internal/pkg/zentao/task_client.go`。本会话修改的 `internal/module/testtask/service.go` 与 `internal/module/testtask/service_testtask_test.go` gofmt 已收敛，未触动上述基线文件，**BLOCKED BY EXISTING BASELINE**。

## 未执行（partial / blocked）

- 真实禅道请求、共享数据库读写、认证浏览器、弱网与性能验收：等待用户提供可写禅道测试环境 + 共享 MySQL 凭据 + 浏览器登录态，**blocked**。
- capability / 对象级 `RequirePerm` 改造：`CreateTesttasks` 与现有 `CreateBuilds` 保持同口径（不内置 capability 校验），后续若需统一收口由产品 / 治理授权另行指示，**partial**。
- 跨模块 capability 与对象可见性矩阵（产品 / 项目 / 执行 / 版本）：仍按 `docs/engineering/` 既定边界，B 阶段后续债务，**partial**。
- 前端回显与提测向导接入：新增查询接口已注册，前端尚未接入回显，需先与产品确认禅道版本列表分页响应格式，**partial**。

## 风险与下一步

- `PartialTesttaskError` 把已成功条目 ID 透传到客户端，UI 需明确区分「已成功」与「未知（超时）」两类；超时条目禁止前端自动重试，避免禅道重复创建。
- `IdempotencyKey` 当前是同批 `(buildID, name)` 去重的提示信号；跨请求幂等需要在禅道侧建立稳定的 request-key 头或业务字段，本会话未实现。
- 新增写权限审查（产品 / 项目 / 执行 / 版本的完整归属授权校验）当前依赖 Service 内置的 `FindDemandContext` + `VerifyExecutionBelongsToProject`，与现有 capability 边界一致；如需在中间件层强制，需要新增独立任务并经授权。
