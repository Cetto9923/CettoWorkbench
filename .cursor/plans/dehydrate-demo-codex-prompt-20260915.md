# 脱水示范提词（CreateBuilds / 提测构建链路）

把下面整段复制给 **Codex / Cursor Agent / Antigravity**。目标：在**零行为变更**前提下瘦身一条已吸收的 Main 能力，验证「抗膨胀」规则可执行。规则源：仓库根 `AGENTS.md` MUST 17 + `docs/engineering/maintainability.md`。

---

## 角色与约束

你是 Workbench（`workbench-claude-po`）工程 Agent。先完整阅读根目录 `AGENTS.md`，再读 `docs/engineering/maintainability.md`、`architecture.md`。本任务是 **dehydration（脱水）**，不是新功能。

生产环境不能用 AI 运维：改完的代码必须让未参与本会话的人能读懂、能排障。

## Scope Gate（只改这些）

**允许修改（按实际存在路径调整，改前先 `git status` / `rg` 确认）：**

- `internal/module/build/**`
- 与 CreateBuilds / LinkStory 构建列表直接相关的 handler/service 调用点（若在 `internal/module/testtask/` 仅有薄封装，可纳入；否则不要扩到整个 testtask）
- 对应前端：提测弹窗里「构建 / CreateBuilds」相关 JS/模板片段（只动该能力；禁止重写整页）

**禁止：**

- 改 board / profile / schedule / follow / urge / 澄清等并行 WIP
- 改 `docs/PRD/`、加新抽象层、改权限模型、改 ZenTao 写协议语义
- push / 切分支 / merge / rebase（除非用户另说）
- 行为「优化」、补产品入口、顺手修 OUT_OF_SCOPE 问题

**当前分支：** `release/po-integrate-main-202609`（以实际 `git branch --show-current` 为准）

## 要删什么（优先）

1. 只转发一次的无意义 wrapper
2. 注释掉的死代码、unreachable 分支、未使用 export
3. 无调用方的「兼容/预留」参数、config、feature 开关
4. 重复的错误处理/日志样板（合并为一条清晰失败路径，Fail Fast）
5. 为不可能状态堆的防御性嵌套

## 验收

1. `git diff --check`
2. `make check`（若仅被既有 baseline / 无关 WIP 挡住：标 `BLOCKED BY EXISTING BASELINE`，并证明本 diff 未引入新失败）
3. 相关 Go 单测（build / testtask 中与 CreateBuilds 相关的）
4. 行为对照：同一登录用户、同一需求，打开提测弹窗 → 拉构建列表 → CreateBuilds 成功条件与改前一致（含错误时仍返回明确错误，禁止变「空成功」）
5. Handoff 必须包含：净行数变化、删除的 symbol 列表、行为风险、实际跑过的命令

## 完成标准

- 允许文件 **净行数下降**（或持平且删除了明确死路径）；净增必须有书面理由
- 无新共享「Manager/Engine/Provider」
- 交付结论只能是 `partial` / `failed` / `blocked` / 在全部门禁通过时才可 `done`
- 不要 commit，除非用户明确要求；保留无关 dirty

## 开场动作

1. 打印 Scope Gate 表
2. `git status` + 允许路径的 `wc -l` / 粗读
3. 列出拟删清单，再动手
