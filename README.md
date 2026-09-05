# Workbench 员工工作台

当前以 PO 工作台为主要业务场景，采用 Go / Gin / GORM / MySQL 与服务端模板渲染。
架构为模块化单体：HTTP → Handler → Service → Repo → 数据库。

开发前先阅读 [AGENTS.md](AGENTS.md)，再阅读受影响的
[架构](docs/engineering/architecture.md)、[数据库](docs/engineering/database.md)、
[前端](docs/engineering/frontend.md)、[测试](docs/engineering/testing.md)及
[质量门禁](docs/engineering/quality.md)。已有模块不是自动合格的参考实现。

当前治理工作按 [Astra 原计划](docs/engineering/governance-plan.md)执行，
进度及未完成验收见 [执行记录](docs/engineering/governance-progress.md)。
计划是执行顺序，工程规范仍以 `AGENTS.md` 为准。

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

各类 AI 编程智能体接入前请查阅对应入口：

- **Claude Code**: 读取 [CLAUDE.md](CLAUDE.md)，通过 `@AGENTS.md` 自动加载工程真源。
- **Gemini**: 读取 [GEMINI.md](GEMINI.md)。
- **Cursor**: 自动应用 [.cursor/rules/engineering-entry.mdc](.cursor/rules/engineering-entry.mdc)。
- **通用接入指南**: 见 [docs/engineering/agent-onboarding.md](docs/engineering/agent-onboarding.md)。
- **模块目录**: 见 [docs/engineering/module-index.md](docs/engineering/module-index.md)。
- **前端共享能力**: 见 [docs/engineering/shared-frontend.md](docs/engineering/shared-frontend.md)。
