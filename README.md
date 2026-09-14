# Workbench 员工工作台

当前以 PO 工作台为主要业务场景，采用 Go / Gin / GORM / MySQL 与服务端模板渲染。
架构为模块化单体：HTTP → Handler → Service → Repo → 数据库。

开发入口为 [AGENTS.md](AGENTS.md)；按任务加载其指定的
`docs/engineering/` 专题（architecture / database / frontend / testing / quality）。
历史报告、任务计划与运维说明在 `docs/archive/`、`docs/plan/`、`docs/operations/`。
已有模块不是自动合格的参考实现。

治理执行范围由当前任务指定。分卡合同见
[Executable Plan](docs/plan/workbench-executable-plan-20260905/README-先读我.md)，
后续任务见 [docs/plan/](docs/plan/)；不自动重跑 COMPLETE 卡。
[原治理计划](docs/archive/governance-plan.md)和
[原执行记录](docs/archive/governance-progress.md)保留用于历史追溯。
计划约束任务范围，工程规范仍以 `AGENTS.md` 为准。

从仓库根目录运行检查：

```sh
git diff --check
make check
```

检查通过表示回归门禁通过；必须同时检查输出中的存量债及任务专属验收。
服务入口为 `go run ./cmd/server`。启动前确认配置和数据库属于获准使用的环境；
当前加载器在 `WORKBENCH_MODE=dev` 时读取 `configs/config.dev.yaml`，
其他情况读取 `configs/config.yaml`。环境覆盖、敏感配置及启动迁移行为仍在治理范围内，
不能把当前启动方式视为已验收的安全部署方案。

## Agent 接入与导航

各类 AI 编程智能体接入前请查阅对应入口；文件存在不等于已实际加载：

- **Claude Code**: 读取 [CLAUDE.md](CLAUDE.md)，通过 `@AGENTS.md` 自动加载工程真源。
- **Gemini**: 读取 [GEMINI.md](GEMINI.md)。
- **Cursor**: 自动应用 [.cursor/rules/engineering-entry.mdc](.cursor/rules/engineering-entry.mdc)。
- **Codex / Jules / Windsurf**: 根目录 [AGENTS.md](AGENTS.md)。
- **GitHub Copilot**: [.github/copilot-instructions.md](.github/copilot-instructions.md)。
- **Cline**: [.clinerules/00-workbench.md](.clinerules/00-workbench.md)。
- **工具覆盖与实际加载验证**: [agent-compatibility.md](docs/engineering/agent-compatibility.md)。
- **模块目录**: 见 [docs/engineering/module-index.md](docs/engineering/module-index.md)。
- **代码目录地图**: 见 [docs/CODE_DIRECTORY.md](docs/CODE_DIRECTORY.md)。
- **前端共享能力**: 见 [docs/engineering/shared-frontend.md](docs/engineering/shared-frontend.md)。

远端 Agent 使用前核对目标仓库、分支和规范版本；不要默认 GitHub 默认分支就是
当前开发分支。本地未提交内容不属于远端交付，具体核验见
[规范体系与 GitHub 发布核对](docs/plan/agent-governance-audit-20260907/SYSTEM-AND-PUBLICATION.md)。
