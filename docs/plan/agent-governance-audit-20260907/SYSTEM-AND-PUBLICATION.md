# 规范体系与 GitHub 发布核对

核对日期：2026-09-07（Asia/Shanghai）。这是指定快照的发布证据，不是持续更新的远端状态。

## 1. 本轮范围与结论

本轮只更新规范、入口索引和本报告；没有修改业务代码、测试、门禁脚本、数据库、服务或全局 Agent 配置。没有 add/commit/push、切分支、合并或改变 GitHub 默认分支。保留所有已有 WIP。

规范体系已统一为：根 AGENTS.md → docs/engineering 专题 → 工具局部提示。业务真源独立，任务 Plan 只约束其授权范围。工具入口因自动发现机制保留在根目录或工具专用目录，不能全部搬到 docs。旧工程目录中的审查/计划已在总索引单独分类，相关文件补历史说明；未移动文件、破坏旧链接或把旧任务重新开启。

统一导航：[spec-index.md](../../engineering/spec-index.md)。工具入口与冷启动：[agent-compatibility.md](../../engineering/agent-compatibility.md)。

本次修正：

- quality.md：日志最小访问权限、目录/既有文件/轮转权限、保留与错误处理；仅对新建或实质修改路径强制落地，旧实现仍是待处理债务。
- quality.md：区分源码事实、条件性风险、已复现缺陷；严重级别要有触发和影响依据；核对生效配置、旧问题和建议的前置条件。
- Cursor database.mdc：禁止动态值插入 SQL，允许封闭、经验证的服务端结构片段，与 database.md 保持一致。
- README/spec-index/AGENTS/onboarding：统一入口、补 MUST 13–16 和 AI/工具规范目录，区分规范/能力目录/历史记录/任务契约/运维；澄清门禁自测适用范围。
- rules-audit/phase2-audit/governance-plan/governance-progress：历史状态限定原版本和范围，不再由目录或旧措辞暗示当前授权/验收。

管理员操作范围、是否允许修改超管/分配高权限角色等仍是业务权限矩阵问题，本轮没有发明限制或修改角色授权。

## 2. GitHub 实际状态

- 仓库根：`/Users/yuyan9923/GitHub/workbench-claude-po`。
- 本地分支：`Claude-PO`；本地 HEAD：`a430311a6d46d99a12d821180efbb3b1854d9d39`。
- 核对目标：`github` → [Cetto9923/workbench-claude-po](https://github.com/Cetto9923/workbench-claude-po)。没有连接公司 origin 或另一 audit remote。
- `git ls-remote --symref github HEAD refs/heads/Claude-PO refs/heads/main refs/heads/master` 返回：
  - 远端 `Claude-PO`：`6c6341234a22dc157c73304d35f4c9ba3b9df833`。
  - 默认分支 `main`：`858de7c7bbdf67006dc7147690ee62426d056b24`。
- GitHub compare API：远端 Claude-PO 相对本地 HEAD `ahead_by=2, behind_by=0`；本地仓库未持有该远端对象，使用固定 SHA 的 GitHub tree API 比对，没有 fetch/checkout/merge。
- 固定 SHA 的两个 recursive tree 均未截断；逐文件对比本地 Git blob、HEAD/index 和远端 blob。当前暂存区为空。

**并非所有最新版规范都已提交到 GitHub。** 核对下列 29 个规范、入口、索引及配套历史文件：

- 10 个：本地已提交，当前内容与远端 Claude-PO 相同。
- 16 个：本地修改未提交，远端有旧版，当前内容不同。
- 3 个：本地未跟踪，远端缺失——Copilot/Cline 入口、agent-compatibility.md。
- 默认 main：24 个缺失，5 个内容不同，不能作为这套新规范的执行入口。
- 29 个路径均未被 .gitignore 忽略；这不等于已经提交。

本报告和同目录 REPORT.md 另属新审查证据，不计入上面的 29 个规范/配套文件；它们当前也未提交。不能用本地文件存在或历史 CI success 声称已发布。

## 3. 逐文件清单

“不同”是当前本地内容与所列远端固定 SHA 的差异；不意味着远端文件完全没有规范。

| 文件 | 本地状态 | GitHub Claude-PO | GitHub 默认 main |
|---|---|---|---|
| `.clinerules/00-workbench.md` | 未跟踪 | 缺失 | 缺失 |
| `.cursor/rules/conventions.mdc` | 本地已提交 | 相同 | 不同 |
| `.cursor/rules/database.mdc` | 本地已修改、未提交 | 不同 | 缺失 |
| `.cursor/rules/engineering-entry.mdc` | 本地已提交 | 相同 | 缺失 |
| `.cursor/rules/frontend.mdc` | 本地已提交 | 相同 | 缺失 |
| `.cursor/rules/testing.mdc` | 本地已提交 | 相同 | 缺失 |
| `.cursorrules` | 本地已提交 | 相同 | 不同 |
| `.github/copilot-instructions.md` | 未跟踪 | 缺失 | 缺失 |
| `AGENTS.md` | 本地已修改、未提交 | 不同 | 不同 |
| `CLAUDE.md` | 本地已修改、未提交 | 不同 | 不同 |
| `GEMINI.md` | 本地已提交 | 相同 | 缺失 |
| `README.md` | 本地已修改、未提交 | 不同 | 不同 |
| `docs/engineering/agent-compatibility.md` | 未跟踪 | 缺失 | 缺失 |
| `docs/engineering/agent-onboarding.md` | 本地已修改、未提交 | 不同 | 缺失 |
| `docs/engineering/ai-boundary.md` | 本地已提交 | 相同 | 缺失 |
| `docs/engineering/architecture.md` | 本地已修改、未提交 | 不同 | 缺失 |
| `docs/engineering/database.md` | 本地已修改、未提交 | 不同 | 缺失 |
| `docs/engineering/debt.md` | 本地已修改、未提交 | 不同 | 缺失 |
| `docs/engineering/frontend.md` | 本地已提交 | 相同 | 缺失 |
| `docs/engineering/governance-plan.md` | 本地已修改、未提交 | 不同 | 缺失 |
| `docs/engineering/governance-progress.md` | 本地已修改、未提交 | 不同 | 缺失 |
| `docs/engineering/module-index.md` | 本地已修改、未提交 | 不同 | 缺失 |
| `docs/engineering/phase2-audit.md` | 本地已修改、未提交 | 不同 | 缺失 |
| `docs/engineering/quality.md` | 本地已修改、未提交 | 不同 | 缺失 |
| `docs/engineering/remediation-evidence/performance-baseline.md` | 本地已提交 | 相同 | 缺失 |
| `docs/engineering/rules-audit.md` | 本地已修改、未提交 | 不同 | 缺失 |
| `docs/engineering/shared-frontend.md` | 本地已提交 | 相同 | 缺失 |
| `docs/engineering/spec-index.md` | 本地已修改、未提交 | 不同 | 缺失 |
| `docs/engineering/testing.md` | 本地已修改、未提交 | 不同 | 缺失 |

## 4. 目录与可发现性边界

核对范围包括根 README/AGENTS/CLAUDE/GEMINI/.cursorrules、全部 .cursor/rules、Copilot/Cline 入口、全部 docs/engineering Markdown。总索引包含工程目录的现行专题、AI 边界、工具指引、能力目录、债务、历史报告和性能证据；未发现第二个模块级 AGENTS.md。

`docs/plan/<task>/` 包含大量有范围的执行契约，不是另一套全仓工程规范；在任务内有效的业务决策不能被历史标签抹除。`docs/operations/` 是环境和操作资料，不能自行授权连库。`.workbuddy/memory/` 是历史工具上下文，不作为新规范入口。本机其他仓库 PRD、被忽略的 docs/design 或本地附件不会自动进入此 GitHub 仓库；需要远端执行时必须显式提供当前任务的业务来源。

适配器统一指向根规范，但本轮没有逐一启动 Codex/Claude/Cursor/Gemini/Jules/Windsurf/Copilot/Cline 验证冷启动。WorkBuddy/Antigravity 的本机版本自动加载机制仍未认证。文件与指针检查只证明结构一致，不保证所有模型未来都遵守。

## 5. 验证与待发布步骤

- 本轮直接修改 10 个既有规范/导航文件，另新增本报告；其余规范脏文件为之前的治理 WIP，保留未覆盖。
- Markdown 本地链接检查：通过；工程目录全部 Markdown 均可在 spec-index 找到；入口均指向 AGENTS.md。
- `git diff --check`：通过。
- `bash scripts/test-quality-gates.sh`：退出 0，23 项通过；日志 `/tmp/wb-rules-system-20260907-selftest.log`。
- `make check`：退出 2，gofmt 通过后在 Go 编译失败；该次源码的 `internal/pkg/render/render.go:18` 有未使用的 `errors` import。后续 Make 门禁未执行；日志 `/tmp/wb-rules-system-20260907-make-check.log`。应用状态仍为 **BLOCKED BY EXISTING BASELINE**，不把规范更新等同于代码修复。
- 工作区在检查期间继续被外部任务修改：包括 repodone/render、主操作测试和首页/待办/已办/通知 JS，另新增拆分文件。未对这些文件做覆盖或修复，测试结果仅认证采样时源码，不认证外部后续改动。
- 未验证真实数据库、部署/浏览器、各工具冷启动、当前远端 CI 或分支保护；本轮没有发布，所以也不存在“本轮修改的 GitHub CI 已通过”。

文件指纹、逐文件远端 blob 清单和结构验证结果分别保存在 `/tmp/wb-rules-system-20260907-start.json`、`/tmp/wb-rules-system-20260907-file-comparison.json`、`/tmp/wb-rules-system-20260907-validation.json`；临时文件可能被系统清理。上面的远端 SHA 和逐文件状态已在本报告固化。未修改扫描器或 baseline 来消除失败。

后续如安排发布，只提交核实过的规范/入口/本报告文件，明确排除其他用户业务 WIP。先复查最新远端与文件差异，处理本地落后状态时不得 reset/stash/restore 用户工作树。发布必须有单独明确请求；本次“是否已提交”只作核验。获准后使用已核验的 github remote 和 Claude-PO，发布后核对远端 blob 与实际 CI，不能 bare push 或推向 origin/audit。是否更改默认分支属于另一项仓库设置决策。
