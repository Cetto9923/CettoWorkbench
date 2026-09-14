# 代码目录地图（Code Directory Map）

> 本文件只是**目录地图**，不是工程规范。权威真源：
> [`AGENTS.md`](../AGENTS.md) → [`docs/engineering/`](./engineering/)。
> 分层与契约细则见 [`architecture.md`](./engineering/architecture.md)；
> 模块职责表见 [`module-index.md`](./engineering/module-index.md)。
> 遇冲突以真源为准，勿把本文件当成第二规则源。

仓库定位：**Go / Gin / GORM / MySQL 服务端渲染单体（monolith）**。  
主链路：`HTTP → Handler → Service → Repo → MySQL`；模板与浏览器 JS 仅作表现/交互，**不是业务真源**。

---

## 1. 顶层目录一览

| 路径 | 一句话职责 |
| --- | --- |
| `AGENTS.md` | 工程宪法（权威源） |
| `CLAUDE.md` / `GEMINI.md` | 工具适配器：只指向 `AGENTS.md`，不另立规则 |
| `README.md` | 仓库简介与导航 |
| `Makefile` | `make check` 等门禁入口 |
| `go.mod` / `go.sum` | Go 模块与依赖锁定 |
| `cmd/` | 可执行入口（当前仅 `cmd/server`） |
| `internal/` | 应用代码（bootstrap / server / middleware / model / module / config / constants / pkg） |
| `web/` | 服务端模板 + 静态资源 |
| `tests/` | `unit` / `integration` / `e2e` |
| `configs/` | 运行时配置占位（`config.yaml`、`config.dev.yaml`） |
| `db/` | 本地库初始化 SQL（`install.sql`、`01_main_base.sql`、`02_po_incremental.sql`） |
| `docs/` | 文档分层（见 §7） |
| `scripts/` | 质量门禁脚本与基线 |
| `deploy/` | 部署相关（如禅道扩展） |
| `design-evidence/` | 设计评审证据 |
| `design-qa.md` | 设计 QA 记录 |
| `logs/` / `tmp/` / `.wip/` / `test-results/` / `.run/` | 本地运行/WIP/结果产物（可再生） |
| `.cursor/` / `.clinerules/` / `.github/` / `.air.toml` | 编辑器、CI、热重载 |

---

## 2. 应用代码 `internal/`

### 2.1 `internal/module/<name>/` 典型文件角色

| 文件模式 | 职责（边界见 `AGENTS.md` MUST 1） |
| --- | --- |
| `handler*.go` | 绑定协议输入，调 Service，渲染/返回 HTTP；**不访问 DB** |
| `service*.go` | 业务规则、对象级授权、事务编排 |
| `repo*.go` | DB 访问与查询塑形；**不决定权限或业务状态** |
| `form*.go` | 入参结构与校验 |

复杂能力模块会按流程再拆文件（如 `handlerboard.go`、`servicedone.go`、`*_enrich.go`），**不强制对称 CRUD**（MUST 2）。

当前模块（`ls internal/module/`）：

| 模块 | 备注 |
| --- | --- |
| `agileteam` | 敏捷团队 |
| `build` | 版本关联；以 Handler/API 为主，无独立 `web/templates/build` |
| `debug` | 调试面板（超管） |
| `dept` | 部门 |
| `login` | 登录/会话；模板在 `web/templates/auth/` |
| `loginlog` | 登录日志 |
| `menu` | 菜单 |
| `metrics` | 指标；部分 JS 在 `web/static/js/metrics-*.js` 根级文件 |
| `operationlog` | 操作日志 |
| `po` | PO 工作台（价值流/看板/已办/通知等） |
| `profile` | 个人资料 |
| `query` | 查询/检索 |
| `role` | 角色与权限 |
| `schedule` | 排期 |
| `testtask` | 提测办理；以 Handler/API 为主，无独立 `web/templates/testtask` |
| `user` | **legacy / reference-not-authoritative**（见 AGENTS Golden Reference） |

模块级表/职责细节以 [`module-index.md`](./engineering/module-index.md) 为准（该索引可能滞后于磁盘；以 `ls` 为准）。

### 2.2 其他 `internal/` 包

| 包 | 职责 |
| --- | --- |
| `bootstrap` | 启动装配 |
| `server` | 路由、中间件链、模板渲染器 |
| `middleware` | Gin 中间件 |
| `model`（含 `model/zentao`） | 模型定义 |
| `config` | 配置读取 |
| `constants` | 常量 |
| `pkg/` | 小型共享能力（session、perm、pagination、zentao 适配等）；新通用层需有重复/边界证据 |

---

## 3. 入口与配置

| 路径 | 说明 |
| --- | --- |
| `cmd/server/main.go` | 唯一可执行入口 → `bootstrap` → `server` |
| `configs/config.yaml` | 非 dev 默认配置（Git 中仅占位） |
| `configs/config.dev.yaml` | `WORKBENCH_MODE=dev` 时加载 |

敏感值经环境变量或受控 secret 注入；勿把当前启动方式当作已验收的安全部署方案。

---

## 4. 门禁（`scripts/` + `Makefile`）

| 路径 | 用途 |
| --- | --- |
| `Makefile` | `make check` 汇总入口 |
| `scripts/check-patterns.sh` | 通用模式门禁 |
| `scripts/check-architecture.sh` | 分层/架构检查 |
| `scripts/check-file-length.sh` | 行数门禁 |
| `scripts/check-go-vet.sh` / `check-gofmt.sh` | Go vet / 格式 |
| `scripts/check-secrets.sh` | 防密钥入库 |
| `scripts/check-local-artifacts.sh` / `check-root-artifacts.sh` | 防本地/根产物污染 |
| `scripts/test-quality-gates.sh` | 测试质量门禁 |
| `scripts/quality-baseline/` | 既有基线快照 |
| `scripts/testdata/` | 脚本自测数据 |

`make check` 只覆盖机器可判项。对象级授权、锁序、N+1、真实浏览器验收等仍须人工核验。

---

## 5. 前端 `web/`

### 5.1 模板 `web/templates/`

按页面能力分包（**不必与每个 `internal/module` 一一对应**）：

`agileteam` / `auth` / `components` / `debug` / `dept` / `loginlog` / `menu` / `metrics` / `operationlog` / `po` / `profile` / `query` / `role` / `schedule` / `user` / `layout/`

- `error.html`：全局错误页  
- `layout/`：共享布局  
- `auth/`：登录相关页面（对应 `login` 模块）

### 5.2 静态资源 `web/static/`

| 路径 | 说明 |
| --- | --- |
| `js/` | 浏览器 JS：模块子目录 + 共享（`app.js`、`ui.js`、`layout/`、`components/`、`permission/`、`picker/`） |
| `js/picker/` | 通用选择器 |
| `css/` | 模块样式 + 共享（`layout/`、`components/` 等） |
| `css/layout/variables.css` | 设计 Token 源 |
| `img/` / `uploads/` / `vendor/` | 图片 / 上传 / 第三方 |

说明：部分模块仅有模板或仅有后端接口，没有对应 `js/<name>/` 或 `css/<name>/`；以磁盘为准。

前端写协议与共享能力约定见 `AGENTS.md` MUST 4/9 与 [`frontend.md`](./engineering/frontend.md)、[`shared-frontend.md`](./engineering/shared-frontend.md)。

---

## 6. 测试 `tests/`

| 路径 | 用途 |
| --- | --- |
| `tests/unit/` | 前端等单元测（`frontend/`、`zentao/`）；Go 单元测多在包旁 `*_test.go` |
| `tests/integration/` | 跨模块 / DB / HTTP；夹具在 `integration/testdata/` |
| `tests/e2e/` | 浏览器相关规格 |

---

## 7. 文档分层 `docs/`

| 路径 | 性质 |
| --- | --- |
| `AGENTS.md` + `docs/engineering/` | **工程真源**（architecture / database / frontend / testing / quality 等） |
| `docs/engineering/module-index.md` | 模块索引（现状清单，非 Golden Reference） |
| `docs/engineering/decisions/` | ADR |
| `docs/plan/` | **规划草案 / 任务工作记录，非规范** |
| `docs/operations/` | 运维拓扑与本地部署说明 |
| `docs/design/` | 设计/业务侧文档 |
| `docs/archive/` / `docs/review/` / 根下健康检查报告 | 历史与评审证据 |

工程链：`AGENTS.md` → `docs/engineering/` → `.cursor/rules/*.mdc`。  
业务链：当前任务要求 → 有效 PRD/原型/已确认决策 → 实现证据。  
冲突时上报，勿自行择一。

---

## 8. 主链路（摘要）

```text
HTTP → Handler → Service → Repo → MySQL
              ↓
     template / JSON 响应（Handler 输出）
```

- Handler：协议与响应；Service：业务与授权与事务；Repo：SQL/持久化。  
- 浏览器 JS / 模板：交互与展示，不是权限或状态真源。  
- 无权威模块 Golden Reference；勿把 `user` 模块违规写法当模板复用。

---

## 9. 本文件边界

- 只描述目录与职责地图；不复制/改写 `AGENTS.md` 或 `architecture.md` 全文。  
- 不把未提交 WIP 写成已交付能力。  
- 目录清单以仓库磁盘为准；本文件滞后时先 `ls`/`find` 再改文档。
