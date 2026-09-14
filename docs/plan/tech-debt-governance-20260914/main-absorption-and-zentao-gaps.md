# Main 吸收盘点与禅道接口缺口

审计日期：2026-09-14
审计仓库：`/Users/yuyan9923/GitHub/workbench-claude-po`
审计分支：`release/po-integrate-main-202609`
禅道原始数据库：只读红线，本次未执行 DDL/DML、迁移或结构修改。

## 1. 本次同步事实

- 已执行 `git fetch origin main`，最新公司仓库 Main 远端引用为
  `21263dc5`（`Fix follow menu.`）。
- Release 当前提交为 `34f7ff56`。
- 最新 `origin/main` 尚未成为 Release 的祖先，因此不能把 Main 宣称为已全部吸收。
- `/Users/yuyan9923/GitHub/workbench` 的本地 `main` 工作区存在未提交 WIP，不能直接
  `pull`、重置或覆盖。当前只更新了远端引用，保留本地 Main WIP 不动。
- 当前 Release 工作区也存在未提交 WIP；本记录不包含这些 WIP，也不替它们做决策。

## 2. 最新 Main 的提交级差异

### 2.1 已等价吸收，不直接 cherry-pick

| Main 提交 | 主题 | 结论 | 证据 |
|---|---|---|---|
| `13536648` | PO 日期条件兼容 `NO_ZERO_DATE` / SQL mode | 已有等价实现，但代码已拆到 Release 的 `repovaluestream.go`；不直接移植 `repo.go` 补丁 | `internal/module/po/repovaluestream.go:111-193,310-318` |
| `1b1fb1d2` | 提测非联调测试单创建 | Release 已有 `CreateTesttasks`、`createTesttask`、用户态 `DoAs`、批量预校验和部分成功错误处理 | `internal/module/testtask/service_testtasks.go`, `internal/module/testtask/gateway.go` |

说明：上述“等价吸收”只表示代码路径存在，不等于已完成真实禅道环境验收。仍需使用授权测试账号验证版本、执行、测试单和权限边界。

### 2.2 已有替代实现，不应直接吸收

| Main 提交 | 主题 | Release 当前情况 | 处理决定 |
|---|---|---|---|
| `3aedcb9d` | 新增静态 `follow` 页面、首页/路由合并结果 | Release 已有功能更完整的 `internal/module/po` 我的关注：业务需求、项目周报、筛选、分页和取消关注 | 不引入 Main 的静态快照页；如发现视觉差异，只做局部对齐 |
| `21263dc5` | 注释掉 `db/install.sql` 中“我的关注”菜单种子 | Release 仍保留 `/follow` 菜单与权限种子 | 不吸收该删除行为；否则会回归用户已要求的菜单功能 |

### 2.3 尚未确认、需要独立评审的 Main WIP

Main 工作区未提交修改集中在以下能力，不能按普通 commit 直接搬运：

- 提测向导：涉及产品/研发需求、内部人员候选、已有版本、非联调与联调测试单字段。
- 提测禅道出站：`/projects/:id/builds`、`/products/:id/builds`、
  `/projects/:id/testtasks` 的参数和返回契约调整。
- LinkStory 弹窗与关联研发需求交互。
- 排期窗口和排期集成页样式/交互重写。
- 侧栏“我的关注”菜单和首页模板改动。

这些 WIP 与 Release 已有同名文件但实现形态不同，必须逐项做行为对照、测试和真实
浏览器验收后再决定是否吸收；本次不覆盖、不回滚、不合并。

## 3. 已确认的禅道接口依赖与未接通项

以下项目在 Release 中有明确的未实现或失败关闭路径，记录为后续任务：

| 优先级 | 能力 | 当前证据 | 说明 |
|---|---|---|---|
| P1 | 看板“提问题” | `web/static/js/po/workboard-modal.js:48-51` | 仅弹窗，提交时提示“问题登记尚未接入服务端”；需要禅道问题创建接口、字段映射、权限和失败幂等设计 |
| P1 | 看板“建任务” | `web/static/js/po/workboard-modal.js:82-85` | 仅弹窗，提交时提示“任务创建尚未接入服务端”；需要禅道任务创建/负责人/执行归属契约 |
| P1 | 问题状态动作 | `internal/pkg/zentao/issue_action_gateway.go:36-51` | 当前使用 `UnavailableIssueActionGateway` 显式失败；`resolve/close/activate` 的原生接口尚未配置或验证 |
| P1 | 提测“添加已有需求” | `web/static/js/po/testtask.js:328-330` | 当前点击后提示“功能尚未开放”；需要禅道研发需求与版本的关联查询/写入契约 |
| P2 | 待办页跨对象主操作 | `web/static/js/po/primary-action.js:56` | 未接线的动作会降级为“请前往工作台办理”；需要逐对象补齐禅道动作 API 和权限验证，不能由前端放开 |
| P2 | 项目参与关系取消 | `web/static/js/po/follow.js:181` | 当前明确禁用，参与关系要求在禅道团队管理维护；不是本地可直接实现的关注按钮 |

## 4. 已接通但必须做真实禅道验收的能力

这些能力已有代码，不列为“未实现”，但不能仅凭单测视为完成：

- 创建版本：`internal/module/testtask/gateway.go` 的 `createProjectBuild`。
- 创建非联调/联调测试单：`internal/module/testtask/service_testtasks.go`、
  `gateway.go`、`gateway_joint.go`。
- 关联研发需求：`internal/module/build` 与 `web/static/js/po/linkstory.js`。
- 看板任务状态更新：`internal/pkg/zentao/task_client.go`。
- 业务需求关注/取消关注：`internal/module/po/servicefollow.go` 和禅道关注接口。
- 项目周报取消关注：直接维护禅道项目的 `follow` 字段，必须确认字段所有者、并发和回滚边界。

验收至少要覆盖：真实登录态、权限拒绝、禅道接口 4xx/5xx、超时、响应缺少 ID、
重复提交和部分成功；不得把空响应当成成功，也不得修改禅道原始数据库结构。

## 5. 后续实施顺序

1. 保留两边 WIP，单独建立“Main→Release 差异矩阵”；先处理 P1 接口契约。
2. 对提测/LinkStory 做真实禅道 API 只读探测与最小写入验收，确认部署版本和权限后再改代码。
3. 实现看板问题、任务创建前，先补 Handler → Service → ZenTao gateway 的对象权限和幂等设计。
4. 每项完成后分别跑 Go/前端测试和认证浏览器验收，再决定是否提交和推送。

## 6. 本次未做事项

- 未执行本地 Main 分支快进：因存在未提交 WIP，避免覆盖用户修改。
- 未 cherry-pick 或 merge 最新 Main。
- 未修改禅道原始数据库、安装 SQL 或运行时结构。
- 未修改任何上述功能代码；本文件仅记录盘点结果。
