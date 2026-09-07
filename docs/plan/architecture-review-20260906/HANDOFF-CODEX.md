# Codex 续作交接（Claude-PO）

生成：2026-09-07 14:05（UTC+8）。以工作树证据为准，不依赖聊天记录。

## 环境快照

| 项 | 值 |
|---|---|
| 仓库 | `/Users/yuyan9923/GitHub/workbench-claude-po` |
| 分支 | `Claude-PO`（ahead origin 3；**勿写 main/master**） |
| HEAD | `10c79360` |
| 8093 | `curl http://127.0.0.1:8093/login` → **200**；进程 `./dist/workbench/workbench`（约 13:49 起） |
| Session | **内存会话**；重启二进制会丢登录，需浏览器重登 |

---

## 1. 当前结论（完成度）

### F01–F12

| 编号 | 状态 | 证据要点 |
|---|---|---|
| F01 | **partial** | `service_detail_authz*` + Handler 401/403/404/500；**缺**真实跨账号 DB/浏览器 |
| F02 | **partial** | `SanitizeRichTextHTML` + 前端净化；单测过；**缺**真实 XSS 浏览器复验 |
| F03 | **code-fixed / partial** | 假 MR/92.5/「实际 2 天」已去；UI「代码质量未接入」；浏览器未复验 |
| F04 | **code-fixed / partial** | 详情错误上抛 + 404/500 区分；隔离探针未重跑 |
| F05 | **code-fixed / partial** | `mapValueStage` 与阶段条对齐；终态不回澄清 |
| F06 | **partial** | `FindHomeFocus` SQL 去重分页；独立研需并入「全部」**仍待补** |
| F07 | **open** | `Find*ProjectWeeklyProjects` 仍 `LIMIT`；Service 再 `filterProjectWeeklyItems`；详情 `Limit: 500` 后找 ID |
| F08 | **open** | `repo_weekly_enrich.go` 逐项目 `FIND_IN_SET`；详情父子/排期热点未批量化 |
| F09 | **open** | Service 拿 db / Repo 展示决策 |
| F10 | **open** | follow 等旁路 fetch |
| F11 | **open** | 详情硬编码色 |
| F12 | **partial** | Makefile 已含 home-focus / notice-filters / priority-helpers / navigation-primary-action / home-list-caption；**真实 E2E 仍缺**；`make check` **未全绿** |

**并行 agent `484a570b`（F07+F08）**：**未完成**——transcript 仅 user prompt，工作树周报路径无对应修复。

### 会话 UI / 体验项

| 项 | 状态 |
|---|---|
| `wb-priority` + `PersonalList.priorityBadge` / `objectTypeBadge` 真源 | **done / partial**（单元+部分截图；全站肉眼未齐） |
| 类型列截断禁止 ellipsis、「业务需求」全文 | **partial→接近 done**（`type-badge-verify.png`；建议再刷 `/home`） |
| 清 Trae `*.wip` 双轨 | **done**（工作树无 `internal/module/po/*.wip`） |
| 看板 `ownerHint` | **done**（源码无匹配） |
| P 标签「标题前」 | **纠正**：看板标题行 P 在标题前；首页等为独立 `c-pri` 列，不是标题内前缀 |
| 通知默认未读 | **done**（`notice.js` / `formnotice` 默认 `unread`） |
| 详情 401 语义 | **code-fixed**；重启后 session 丢 → **需重登再验** |
| pageSize 记忆 | **partial**：home/todos/done/notice 有 `PersonalList.load/savePageSize`；follow 仍自管分页、未同等落地 |
| 侧栏角标 | **partial**：待办/通知有数字；**我的已办故意无数字**（模板无 `SidebarBadges.Done`） |
| UI Stage 4–7（`docs/plan/ui-actions-unification-20260907/`） | Stage 1–4 **partial**；Stage 5+ 与业务契约阻塞项仍在 PROGRESS.md |

### `make check`

**BLOCKED BY EXISTING BASELINE**（与近期 UI 清理无直接关系）：

1. `TestDeriveDemandPrimaryActions_BatchIN` / `TestDeriveStoryPrimaryActions_BatchIN` — sqlmock 参数数与当前 `repo_detail` 查询不一致  
2. `repo_detail.go` ≈ **655** 行且 **未入** `scripts/quality-baseline/file-length.tsv`

---

## 2. 工作区注意

- **脏树极大**：大量 `M`/`MM`（staged+unstaged 混杂）+ 未跟踪 `STATUS.md` / `home-list-caption.test.js` / `.workbuddy/`。**保留无关 WIP**，禁止 `reset`/`clean`/`stash` 整树回滚。
- **不要回滚已修项**：F01–F06 代码路径、富文本净化、质量 `available=false`、wb-priority 真源、通知默认未读、详情 AuthZ/401、已删 `.wip` / `ownerHint`。
- **8093**：静态资源多从 `web/static` 直出；Go 变更需重建/重启二进制；**重启丢内存 session**。
- **勿 commit/push/切分支**，除非用户明确授权。
- 计划真源：`AUDIT-AND-REPAIR-PLAN.md` + 本目录 `STATUS.md`；UI 并行计划见 `docs/plan/ui-actions-unification-20260907/PROGRESS.md`。

---

## 3. 给 Codex 的完整提词（可整段复制）

```
你在 /Users/yuyan9923/GitHub/workbench-claude-po（分支 Claude-PO）继续架构审计修复。用中文回报。

## 目标
1) 先让 `make check` 在当前 WIP 下可解释地变绿或明确剩余 BLOCKED BY EXISTING BASELINE；
2) 再落地 F07+F08（周报 LIMIT/Go 过滤 → SQL 筛选+正确 total；周报/详情/排期 N+1 → 批量查询）；
3) 再推进 F09–F11 与 F12 收口，并补真实浏览器验收证据。

## 必读（开工前）
- AGENTS.md + docs/engineering/agent-onboarding.md
- docs/plan/architecture-review-20260906/{AUDIT-AND-REPAIR-PLAN.md,STATUS.md,HANDOFF-CODEX.md,EVIDENCE.md}
- docs/plan/ui-actions-unification-20260907/PROGRESS.md（UI 并行 WIP，勿回滚）
- git status -sb；git branch --show-current（必须 Claude-PO，非 main/master）
- curl -s -o /dev/null -w '%{http_code}' http://127.0.0.1:8093/login

## 硬约束
- 不切分支 / 不 commit / 不 push（除非用户另下明确指令）。
- 保留无关 dirty/untracked WIP；禁止 reset/clean/stash 整树；禁止回滚 F01–F06 与 UI 真源收口。
- Handler→Service→Repo；对象授权在 Service；Repo 不做权限/UI 决策。
- 列表：SQL filter→sort→count→paginate；禁止先 LIMIT 再 Go 过滤当 total。
- 禅道表按真实 schema；参数化 SQL；最小改动；文件 ≤500 行（精确基线可缩小不可抬高无关项）。
- 完成声称前必须：git diff --check + make check；失败如实报 BLOCKED BY EXISTING BASELINE。
- 不要改 notice「默认未读」合同（已落地）。不要为统一外观新造框架。

## 已完成 / 勿重做（核实后跳过）
- F01/F02 代码：详情 AuthZ、富文本净化（仍缺真实跨账号/XSS 浏览器复验）。
- F03–F05：假质量数据去掉、吞错上抛、阶段映射对齐。
- F06 部分：FindHomeFocus SQL 分页（独立研需进「全部」仍 open）。
- UI：wb-priority 真源、类型列不截断、清 *.wip、ownerHint 删除、通知默认 unread、侧栏待办/通知角标（已办无数字是故意的）、详情 401 语义。
- 并行 agent 484a570b 声称做 F07/F08：**未完成**，须按源码重做。

## 建议优先顺序
A. make check 阻塞
   - 修 primaryAction 相关 sqlmock（TestDerive*PrimaryActions_BatchIN）与当前 repo_detail 查询参数对齐；
   - 处理 repo_detail.go ~655 行：按责任拆分，或经明确决策写入 file-length baseline（禁止无说明抬高）。
B. F07：周报候选筛选/Count/分页进 SQL；详情按项目 ID 直读+可见性；禁止 Limit:500 整表再找。
C. F08：周报 release 风险批量查（保留禅道 CSV/FIND_IN_SET 语义但禁止逐项目循环）；详情父子缺陷/任务批量；排期故事任务批量（可分 PR/分提交意图，但仍不擅自 commit）。
D. F06 收尾 → F09 → F10 → F11 → F12（Makefile/契约已部分纳入；补真实 E2E 与隔离探针重跑）。
E. 浏览器验收清单（登录后）：/home 类型全文与 P 列；详情质量「未接入」；通知默认未读；侧栏角标；401 后重登再开详情。

## 验收命令（最低）
git diff --check
make check
go test -count=1 ./internal/module/po/ -run 'Weekly|Detail|Authz|MapValueStage|PrimaryAction'
# 触及前端时：
node tests/unit/frontend/priority-helpers.test.js
node tests/unit/frontend/notice-filters.test.js
node tests/unit/frontend/home-focus.test.js
# 8093 若刚重启：浏览器重新登录后再验 UI

## 交付格式
1. 关闭/推进了哪些 F（done / partial / open）及文件列表
2. 验证：每条命令 PASS/FAIL/BLOCKED 原文摘要
3. 仍 partial 的原因与下一切片
4. 更新 docs/plan/architecture-review-20260906/STATUS.md（只改状态，不改 EVIDENCE 原文）
5. 明确写出：不要回滚的已修项 + 用户是否需要重登 8093
```

---

## 4. 用户本地需手动做的一步

1. 浏览器打开 `http://127.0.0.1:8093/login`，用业务账号（如 `003030`）**重新登录**（8093 约 13:49 重启过，内存 session 已失效）。  
2. 登录后快速目视：`/home` 类型列是否全文、`/notice` 是否默认「未读」、侧栏待办/通知是否有角标且**已办无数字**、打开一条需求详情是否不再因旧 session 误报 401。  
3. 不要在未授权时让 Agent commit/push；续作把上面「给 Codex 的完整提词」整段粘贴即可。
