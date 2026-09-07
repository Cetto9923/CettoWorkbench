# Dev / Acceptance Topology (local MySQL)

> 状态：**描述当前事实**，不包含任何业务变更。
> 适用：本机验收（Mac + 本机 MySQL 9.6）。
> **本文件不适用于生产；生产拓扑未连、未验证。**
>
> 历史：PD VM `vm-zentao`（MySQL 8.0.46 + SSH `:13306` tunnel）已退役；旧 VM 证据见
> `docs/plan/shared-database-optimization-20260906/P1B-VM-EXECUTION-RESULT.md`（BLOCKED）与
> `P1B-VM-PREFLIGHT-LOCAL.md`（本机 rebase）。

## 1. 服务与连接

```text
Mac 验收 Workbench
  WORKBENCH_MODE=dev
  → configs/config.dev.yaml
  → 127.0.0.1:3306 (本机 MySQL 9.6.0)
  → zentaopms
  监听 :8093
```

| 应用 | 位置 | DB 主机 | DB 库 | DB 账号（当前） |
|---|---|---|---|---|
| Workbench 验收 | Mac 进程（例：`./tmp/workbench_dev` 或 `dist/workbench/workbench`） | `127.0.0.1:3306` | `zentaopms` | `zentao`（**SHARED / DO NOT REVOKE**）|
| ZenTao Web | 本机当前未观察到运行中的实例 | — | — | — |
| MySQL | Homebrew / 本机服务 | `127.0.0.1:3306` | `zentaopms` | `root@localhost`（DBA）、`zentao@127.0.0.1` / `zentao@localhost` |

## 2. 现存 DB 账号（与本治理相关）

| Account | Host | 当前 GRANT | 角色 | 治理动作 |
|---|---|---|---|---|
| `zentao` | `127.0.0.1` | schema-level ALL on `zentaopms.*` | 过渡期 Workbench runtime | **不动**（无 REVOKE/ALTER/DROP/密码轮换）|
| `zentao` | `localhost` | 同上 | 同 identity，本机 socket | **不动** |
| `workbench_rw` | `127.0.0.1` | **未创建** | 计划：Workbench 主库 | **P1b-vm 暂停** |
| `workbench_ro` | `127.0.0.1` | **未创建** | 计划：PO readonly pool | **P1b-vm 暂停** |
| `root` | `localhost` | full DBA | 仅人工 DBA 路径 | Agent 不执行 GRANT |

## 3. Option A（当前）

用户选择 **Option A**：先让本机 Workbench 用现有 `zentao` 账号跑通验收；**暂停 P1b-vm**（不 CREATE USER / GRANT）。

验收冒烟（2026-09-06，本机）：

| Check | Result |
|---|---|
| `WORKBENCH_MODE=dev` | YES |
| DB target | `zentao@127.0.0.1:3306/zentaopms` |
| MySQL version | `9.6.0` |
| TCP to `:3306` from Workbench | ESTABLISHED |
| `GET /` | 302 → login |
| `GET /login` | 200 |
| `GET /home` / `/demands` / `/schedule`（未登录） | 302/303 重定向登录 |
| `zt_user` / `zt_operation_logs` 可读 | YES |

**不宣称** P1b-vm 关闭；**不宣称** 已登录业务页验收完成（需人工账号登录后确认）。

## 4. 重要规则

- 本机 MySQL **不是生产**。`@@read_only=0` 不构成生产证据。
- 对 `zentao` 的 `REVOKE` / `ALTER USER` / `DROP USER` / 密码轮换一律不在本窗口处理。
- 用 `zentao` 全库 ALL 跑 Workbench 是**过渡期**；最终目标仍是 `workbench_rw` + `workbench_ro`。
- 恢复 P1b-vm 前：人工确认本机为验收环境 → 重审 `grants-workbench-runtime.sql`（MySQL 9.6）→ 人工执行 DBA bundle。

## 5. 相关文档

- `docs/plan/shared-database-optimization-20260906/P1B-VM-PREFLIGHT-LOCAL.md`
- `docs/plan/shared-database-optimization-20260906/P1B-VM-PERMISSIONS.md`
- `docs/plan/shared-database-optimization-20260906/P1B-VM-GRANT-CHECKLIST.md`
- `docs/plan/shared-database-optimization-20260906/P1B-VM-RUNBOOK.md`
- `docs/plan/shared-database-optimization-20260906/P1B-VM-EXECUTION-RESULT.md`
- `docs/plan/shared-database-optimization-20260906/P1B-VM-DBA-BUNDLE.md`
- `docs/plan/shared-database-optimization-20260906/grants-workbench-runtime.sql`
