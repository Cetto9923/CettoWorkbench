# Engineering Specification Index

这是仓库工程文档的统一目录。唯一工程总规范是根目录的 [AGENTS.md](../../AGENTS.md)，
当前包含 MUST 1–16。本文负责导航和分类，不复制第二套规范。
业务需求仍由当前用户要求、有效 PRD 和已确认决策确定；工程目录不代替业务授权。

## 1. 阅读顺序和权威关系

1. [AGENTS.md](../../AGENTS.md)：工程总规范。
2. [agent-onboarding.md](agent-onboarding.md)：任务、分支/HEAD/WIP、实际加载和交接流程。
3. 本页下表中的相关专题；按任务加载，不要求每次通读所有历史报告。
4. 对应文件的 `.cursor/rules/*.mdc`：局部提示，不能覆盖总规范或专题。
5. 当前任务指定的 PRD、Plan/Card 和证据：确认业务行为及执行范围。

```text
AGENTS.md                         工程总规范 / 通用 Agent 入口
docs/engineering/spec-index.md    统一目录（本文件）
docs/engineering/*.md             专题规范、目录和明确标识的历史文件
CLAUDE.md / GEMINI.md 等           工具要求的发现入口，只引用规范
.cursor/rules/*.mdc               Cursor 自动加载入口和文件类型提示
docs/plan/<task>/                 有范围和版本的任务契约、计划、审查及证据
docs/operations/                  部署拓扑和运维操作说明
scripts/ + tests/ + .github/      实际门禁、测试和 CI 实现
```

工具入口必须保留在相应工具可发现的位置，不能为了“一个目录”全部移动到 docs。
早期报告暂保留原路径以保持链接有效；所在目录不赋予报告新的规范效力。

## 2. 现行工程规范与流程

| 文件 | 职责 / 何时加载 | 相关验证及边界 |
|---|---|---|
| [architecture.md](architecture.md) | 分层、事务职责、内容/注释、目录、授权与错误语义 | 架构扫描 + 实际调用链审查；对应 MUST 1/2/3/8/11/14 |
| [database.md](database.md) | Schema、SQL、性能、共享禅道写库并发 | 模式扫描 + SQL/索引证据 + 隔离并发试验；MUST 5/6/13 |
| [frontend.md](frontend.md) | 交互、共享能力、导航、主题 | 前端测试及真实登录浏览器；MUST 4/9 |
| [testing.md](testing.md) | 测试分类、隔离、回归有效性 | 包测试、任务相关 integration/E2E；MUST 10/16 |
| [quality.md](quality.md) | 门禁、秘密、日志权限与生命周期、审查证据与定级 | `make check`、门禁自测及任务验收；MUST 7/8/12/16 |
| [ai-boundary.md](ai-boundary.md) | 员工 AI 功能的数据、身份、工具、确认及预算边界 | 仅在 AI 功能任务中适用；文档不授权引入模型或实现功能 |
| [agent-onboarding.md](agent-onboarding.md) | 启动、上下文恢复、WIP 完整性和交接 | 真实读取、Git 状态、差异、退出码；MUST 15/16 |
| [agent-compatibility.md](agent-compatibility.md) | 各工具发现入口、冷启动验证、通用启动提词 | 文件存在、实际加载、遵守行为分别验证 |

上述命令只是相关验证入口，不代表所有 MUST 都已自动强制执行。
现有前端 Make target 是显式子集；共享写库的并发、语义授权、N+1、日志 ACL 和注释真实性
不能仅靠文本扫描证明。具体执行范围和跳过条件见 quality.md/testing.md。

## 3. 实现目录与债务记录

| 文件 | 定位 |
|---|---|
| [module-index.md](module-index.md) | 模块/职责/数据来源目录；源码与 schema 状态按证据日期核验 |
| [shared-frontend.md](shared-frontend.md) | 共享能力接口与使用合同；实现和验收状态需复核，不能覆盖 frontend.md |
| [debt.md](debt.md) | 已知债务和历史清单；不是允许复制违规的新规则 |

## 4. 历史文件与任务契约

| 位置 | 定位 |
|---|---|
| [rules-audit.md](rules-audit.md) | Phase 1 规则冲突的历史审查 |
| [phase2-audit.md](phase2-audit.md) | Phase 2 历史代码审查；原严重级别和验收不认证当前版本 |
| [governance-plan.md](governance-plan.md) | 已保存的原始治理计划；不是自动执行授权 |
| [governance-progress.md](governance-progress.md) | 原治理计划的历史执行记录 |
| [performance-baseline.md](remediation-evidence/performance-baseline.md) | 指定版本/隔离数据集的性能证据，不能当作当前线上性能 |
| [Executable Plan 入口](../plan/workbench-executable-plan-20260905/README-先读我.md) | 分卡实施合同；仅执行当前授权卡，前置与 COMPLETE 均核对原版本 |
| [docs/plan/](../plan/) | 后续任务 Plan/决策/证据；任务内约束不可静默推广成全仓规范 |
| [docs/operations/](../operations/) | 运维与环境文档；配置出现不代表有访问或执行授权 |

需要更新单文件汇总的任务包时，遵守该包自己的生成/同步规则，不独立编辑两份冲突合同。
`.workbuddy/memory/` 等工具记忆不是规范副本；其他仓库或本机绝对路径中的 PRD
不会自动随本仓库发布。需要跨工具执行时，在任务中提供已确认且可访问的业务来源。

## 5. 工具入口与发布核验

入口清单见 [agent-compatibility.md](agent-compatibility.md)。新增工具只添加经过核实的薄入口，
不复制整份工程规范。修改总规范时检查受影响专题、索引和入口是否仍一致。

“本地存在”“已暂存”“本地已提交”“目标 GitHub 分支有相同内容”“CI 通过”是不同状态。
发布核验应记录远端 URL、目标分支、远端 SHA，并按文件 blob 比对；仅看 remote 名称、
本地 tracking ref 或同名文件存在都不够。GitHub 默认分支可能是旧快照；远端 Agent 必须
显式选择已核验分支。只读核验不授权 commit、push、切分支或修改远端默认分支。

近期核对记录：
- [2026-09-07 规范、架构和共享库审查](../plan/agent-governance-audit-20260907/REPORT.md)
- [规范体系与 GitHub 发布核对](../plan/agent-governance-audit-20260907/SYSTEM-AND-PUBLICATION.md)
