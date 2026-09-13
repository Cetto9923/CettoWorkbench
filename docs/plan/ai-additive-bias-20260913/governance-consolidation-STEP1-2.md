# AI 研发治理规则收口 — STEP 1 只读审计 + STEP 2 目标架构

审计时间：2026-09-13　HEAD：`3507193e`（`release/po-integrate-main-202609`）
方法：全部数字为本次实测。命令附在每节末尾，可复现。
状态：**只读，未修改任何文件。**

---

## STEP 1 — 只读审计

### 1.1 规则面清单与规模

「规则面」= Agent 在开工前**可能被要求读的、带规范效力**的文件。

| 层 | 文件数 | 行数 | 说明 |
|---|---:|---:|---|
| 宪法 | 1 | 178 | `AGENTS.md`（MUST 1–16） |
| 专题规范 | 8 | 812 | `architecture`126 `database`125 `frontend`96 `testing`47 `quality`150 `agent-onboarding`111 `agent-compatibility`74 `spec-index`83 |
| 目录/债务 | 3 | 256 | `module-index`33 `shared-frontend`108 `debt`115 |
| 未来能力规范 | 1 | 65 | `ai-boundary.md` |
| 工具入口 adapter | 11 | 199 | `CLAUDE`16 `GEMINI`10 `.cursorrules`9 `.cursor/rules/*.mdc`109（conventions29 database23 engineering-entry10 frontend29 testing18）`.clinerules`10 `.agents/rules`35 `.github/copilot-instructions`10 |
| **规范文本面小计** | **24** | **1,510** | Agent 被要求读、且带规范效力的部分 |
| 门禁实现 | 8 | 781 | `scripts/check-*.sh` 7 个 642 行 + `test-quality-gates.sh` 139 行 |
| 基线 | 6 | 90 | `scripts/quality-baseline/*` |
| **规则面合计** | **38** | **2,381** | 其中 1,510 行是规范文本，871 行是机器实现 |

**规范文本面之外（Agent 仍会被指向）**：`docs/plan/` 77 文件 / 16,263 行；`docs/review/` 263 行；
`docs/operations/` 276 行；自述为「历史」的 `docs/engineering/*` 5 文件 1,473 行
（`phase2-audit`843 `ui-audit`391 `governance-plan`169 `governance-progress`41 `rules-audit`29）。

复现：`wc -l AGENTS.md docs/engineering/*.md CLAUDE.md GEMINI.md .cursorrules .cursor/rules/*.mdc .clinerules/00-workbench.md .agents/rules/workbench-engineering.md .github/copilot-instructions.md scripts/*.sh`

### 1.2 重复规则（同一约束存在两份以上）

| # | 重复内容 | 位置 A | 位置 B | 性质 |
|---|---|---|---|---|
| D1 | 「读 AGENTS.md + agent-onboarding，跑 git diff --check + make check」 | `.clinerules/00-workbench.md` | `.github/copilot-instructions.md` | **逐字重复**：`diff` 仅差 1 个词（`Verify active rule loading` vs `Verify loading`） |
| D2 | 同一工具两个入口 | `.cursorrules`（9 行） | `.cursor/rules/engineering-entry.mdc`（10 行） | Cursor 重复入口，`.cursorrules` 是旧格式 |
| D3 | MUST 15（WIP 完整性）+ MUST 16（不得放宽门禁） | `AGENTS.md` MUST 15/16 | `.agents/rules/workbench-engineering.md`「开发前/交付前」 | adapter **重述正文规则**，与 `AGENTS.md:5`「adapter MUST NOT redefine」自相矛盾 |
| D4 | 工程/业务双真源链 | `AGENTS.md:7-20` | `agent-onboarding.md §1`（逐句同义） | 概念二次定义 |
| D5 | 门禁命令清单 | `Makefile:8` | `quality.md`「It executes: 1.–8.」 | 手抄清单，已漂移（见 C1） |
| D6 | AI 边界规范 | `ai-boundary.md`（65 行，7 条边界 + 9 个测试场景） | `governance-plan.md §三`（同 8 条边界） | 同一规范两处维护 |
| D7 | MUST 1/4/5/8/9 本地复述 | `.cursor/rules/conventions.mdc`、`database.mdc`、`frontend.mdc` | `AGENTS.md` + 对应专题 | 文件级提示重述正文（可接受，但已出现「baseline files may not grow」等抄写漂移） |

### 1.3 冲突与失效

| # | 问题 | 证据 | 严重度 |
|---|---|---|---|
| C1 | `quality.md` 称「frontend Make target **enumerates four files**」，实为 **23 个** | `Makefile:19-41`（23 行 `node ...test.js`）vs `quality.md:76` | P1 文档漂移 |
| C2 | `debt.md` 记「New over-limit files **(7)**」，实际 **13 个**，且计数全错（`metrics.css` 记 683，实际 **2154**） | `debt.md:28-36` vs `make check-file-length` 实测 | P1 数字失效 |
| C3 | 同一文件的超限值有两个互相冲突的真源 | `debt.md:44` 记 `ui.js 775→819`；`file-length.tsv` 记 738；实测 HEAD 751 | P1 真源分裂 |
| C4 | `debt.md:10` 称「`make check` currently stops when the length scanner encounters an unstaged deleted tracked file」——**已不成立** | 实测 `check-file-length` 跑完并输出 21 条；`check-gofmt` 仅 stderr 噪音、exit 0 | P1 失效陈述 |
| C5 | `governance-progress.md` 记阶段 1–4「**未开始**」、门禁「均通过」——与当前实际（阶段已实施、门禁全红）矛盾 | `governance-progress.md` 全表 vs 本次门禁实测 | P1 失效看板 |
| C6 | `agent-onboarding.md §4` 要求「规则/门禁变更须跑 `bash scripts/test-quality-gates.sh`」——该脚本**不在 Makefile、不在 CI**，且本地跑 >2 min 未跑完 | `Makefile:8`（`check` 不含它）；`test-quality-gates` 全仓仅被文档引用 | P1 无法执行的规则 |
| C7 | MUST 8「MUST not exceed 500 lines」**100% 未执行** | 21 个文件超限，其中 13 个从未入基线、8 个已超其记录值 | P0 规则失效 |
| C8 | `module-index.md` 把 `dept` 的 `gorm.DB` 依赖列为「正常模块形状」，而 MUST 1 禁止 Service 操作 DB | `module-index.md` dept 行 vs `architecture.tsv` 已把它记为债务 | P2 认知冲突 |
| C9 | `ai-boundary.md` 为**零运行时**的功能规定 65 行 MUST 与 9 个「NOT IMPLEMENTED」测试场景 | `ai-boundary.md:6`「zero AI runtime implementations」 | P2 空转规范 |

### 1.4 当前门禁实测（HEAD `3507193e`）

| 门禁 | 结果 | 关键输出 |
|---|---|---|
| check-local-artifacts | ✅ | `passed` |
| check-gofmt | ✅（有噪音） | `lstat internal/module/po/po_work_scope.go: no such file or directory` |
| check-test | ✅ | `go test ./...` 全绿 |
| check-frontend-test | ✅ | 23 个 frontend 测试全绿 |
| check-vet | ✅ | 0 条诊断 |
| check-whitespace | ✅ | `passed` |
| **check-file-length** | ❌ **21 条** | 见 1.5 |
| check-patterns | ✅（噪音） | `181 finding(s), 154 new`，**耗时 8m02s** |
| check-secrets | ✅ | 0 指纹 |
| check-architecture | ✅ | 1 条存量债 |
| **`make check`** | ❌ | **HEAD 即失败** |
| **CI `quality.yml`** | ❌ | `on: push / pull_request` → `make check` → 每次必红 |
| `test-quality-gates.sh` | 未运行 | **未接入 Makefile / CI**，且 120s 内跑不完 |

### 1.5 永久红灯：check-file-length 21 条

**21 条全部是 HEAD 提交态成立，0 条由 WIP 造成。**即任何 Agent、任何时候跑 `make check` 都红。

| 类别 | 条数 | 文件（HEAD 行数） |
|---|---:|---|
| 已入基线但 HEAD 已超其记录值 | 8 | `schedule/form.go`913→**935**、`components.css`595→**616**、`layout.css`531→**643**、`schedule.css`539→**545**、`scheduleintegrated.css`879→**929**、`scheduleintegrated.js`837→**840**、`ui.js`738→**751**、`schedule/index.html`742→**751** |
| 超 500 但从未入基线 | 13 | `metrics.css`**2326**、`permission/members.js`580、`metrics-radar.js`708、`metrics-manage.js`600、`po/po-profile.css`733、`po/follow.css`711、`po/shell.css`571、`po/board.css`519、`permission/list.css`514、`po/wb-priority.css`504、`po/po-profile.js`868、`po/linkstory.js`611、`po/notice.js`508 |

**为什么它不能靠改基线变绿**：`check-file-length.sh:46-62` 的防放宽守卫把「worktree 基线里出现 trusted(HEAD/upstream) 基线没有的条目」直接判为 `baseline expansion rejected`，把「数值变大」判为 `baseline loosening rejected`。即**新增条目在机制上被拒绝**。`debt.md:21-24` 自己也记录了这一点。
→ 结论：**该门禁自 2026-09-11 起结构性常红**（基线最后修改 `1745f918`），约 2 天、数十次提交里所有 Agent 都在「红灯下交付」，MUST 8 实际是死规则。

### 1.6 其它门禁缺陷（不改则 CI 无意义）

| # | 缺陷 | 证据 | 影响 |
|---|---|---|---|
| G1 | `check-patterns.sh` 单次 8m02s | 实测；CI `timeout-minutes: 15` | 根因：`scan()` 每条规则一次 `grep -EnHi` 扫 ~520 文件（14 条规则）；`check-patterns.sh:110` 每条命中再 spawn 一次 `git hash-object`（181 次进程） |
| G2 | advisory 噪音 154/181 = **85% 未入清单** | `patterns.tsv` 实际 68 条 vs 本轮 181 命中 | 基线名存实亡，「新增 154 条」长期为常态 → Agent 学会忽略门禁输出 |
| G3 | `NEW_WINDOW` 一条规则占 **63/154 = 41%** 噪音 | 本轮分布；`quality.md` 自认「the token alone is not a violation」 | 文本无法判定的规则在扫，等于纯噪音 |
| G4 | advisory 扫到**测试夹具** | 27 条命中在 `tests/unit/frontend/*.test.js`、`*_test.go` | 生产行为断言被当生产问题 |
| G5 | 未跟踪的根目录报告污染扫描集 | `slim-plan-2026-09-13.html`(LOCAL_ESCAPE_HTML×2)、`code-audit-2026-09-13.html`(WEAK_PASSWORD_HASH×6)、`code-review-2026-09-10.html`(×2) | `check-patterns.sh:19` 用 `--others`，根目录 `.html` 进入扫描集 |
| G6 | advisory 命中规范真源本体 | `LOCAL_ESCAPE_HTML web/static/js/ui.js:10`（即 canonical `escapeHtml`） | 永久假阳性 |
| G7 | `.css` 无扩展名级扫描；无死 CSS 检测 | `check-patterns.sh:19` 只含 `'*.go' '*.js' '*.html'`；`.css` 仅被 ZENTAO 主机规则覆盖 | Request A/S1 已证实有死 CSS，无门禁 |
| G8 | 架构门禁按**文件名 glob** 定界 | `check-architecture.sh` 只扫 `*handler*/*repo*/*service*.go`，209 个非测试 .go 中覆盖 199 | 当前真盲区 **0**（实测 50 个未覆盖文件均无 DB 访问）——**这是结构性脆弱而非现存缺陷**，需如实标注 |
| G9 | 幽灵索引条目 174 个 | `git ls-files --cached` 中磁盘不存在：`docs/Demo/*` 169 + `internal/module/po/po_work_scope.go` + 4 个已迁移测试 | `check-secrets` 已加 `-f` 守卫；`check-gofmt:22` **未加** → stderr 噪音 |
| G10 | `.gitignore` 未提交改动含**本机绝对路径** | `.gitignore` 新增注释 `/Users/yuyan9923/GitHub/CRCBWorkbench/docs/Demo` | 环境信息写进受版本控制的配置 |

### 1.7 AI 增肥机制（本项目实测成因）

| # | 机制 | 证据 |
|---|---|---|
| M1 | **没有任何门禁度量净行数** | `make check` 11 个目标中 0 个统计 added/deleted → 删除无回报 |
| M2 | **尺寸规则死掉 → 鼓励长文件** | 1.5；`metrics.css` HEAD 683→**2154**（+215%）而无人被拦 |
| M3 | **advisory 是噪音不是信号** | G2/G3/G4；154 条新增长期正常化 |
| M4 | **规则面只增不减** | `docs/plan` 77 文件/16,263 行；每任务新建目录；治理结束后 `governance-plan/progress` 仍留在规范目录 |
| M5 | **规范目录混入历史快照** | `docs/engineering/` 内 5 个自述「历史」文件共 1,473 行，Agent 无法区分「规则」与「历史」，过期数字被反复引用（C2/C3/C4/C5） |
| M6 | **门禁自身无回归保护** | `test-quality-gates.sh` 不运行 → 新规则可以只写文本、不配扫描器 |
| M7 | **入口按工具增殖，各抄子集** | 7 个 adapter，D1/D3/D7；每次新增工具就多一份可能漂移的规则副本 |
| M8 | **注释被当交付物** | Request A 已证：反向注释 165 条、强制文件头样板 1,709 行；MUST 里无一句约束注释 |

---

## STEP 2 — 目标规则架构（设计，未实施）

### 2.1 设计原则

1. **一个规范真源**：`AGENTS.md` 是唯一带规范效力的文件。其余全部降级为「提示」「契约」「历史」。
2. **一个执行真源**：`make check` 是唯一门禁入口；`scripts/` 是唯一强制实现。
3. **能机器判定的进 hard gate，不能的明确写成 review-only**——不允许「写了规则但没门禁、也没声明是软的」。
4. **文件数只降不升**；新增一个文件必须删掉至少一个。
5. **不新增 MUST 文本条目**（保持 1–16），只把已有 MUST 接上/修好门禁。

### 2.2 目标结构（1 + 1 + 5 + 入口 + 门禁）

```
AGENTS.md                      ← 唯一规范真源（含 pre-flight / handoff 记录模板 / review-only 清单）
   │
   ├── docs/engineering/       ← 5 个专题契约（非独立真源，AGENTS.md 指定才读）
   │     architecture.md  database.md  frontend.md  testing.md  quality.md
   │
   ├── docs/evidence/          ← 非规范：module-index / shared-frontend / decisions / performance
   │
   ├── scripts/ + Makefile     ← 唯一执行真源：make check
   │
   └── 工具入口（薄，3–4 行，只指向 AGENTS.md）
         CLAUDE.md  GEMINI.md  .cursor/rules/engineering-entry.mdc
         .clinerules/00-workbench.md  .github/copilot-instructions.md
```

### 2.3 STEP 3 删除 / 合并 / 归档清单

| 动作 | 对象 | 行数 | 理由 |
|---|---|---:|---|
| **删** | `.agents/rules/workbench-engineering.md` | 35 | D3：重述 MUST 15/16 并追加一条无法执行的命令 = 竞争真源 |
| **删** | `.cursorrules` | 9 | D2：与 `.cursor/rules/engineering-entry.mdc` 同工具重复，旧格式 |
| **删** | `docs/engineering/spec-index.md` | 83 | 规则面降到 ~14 文件后，目录页无存在必要；导航并入 README 6 行 |
| **删** | `docs/engineering/agent-onboarding.md` | 111 | D4 重复真源链、C6 不可执行命令；真正增量的 pre-flight 清单与 handoff 模板（~25 行）并入 AGENTS.md |
| **删** | `docs/engineering/ai-boundary.md` | 65 | C9：零运行时功能的 65 行 MUST + 9 个未实现场景 |
| **删** | `docs/engineering/governance-plan.md §三` | ~30 | D6：与 `ai-boundary.md` 重复的 AI 边界 |
| **归档 → `docs/archive/`** | `governance-plan.md` `governance-progress.md` `rules-audit.md` `phase2-audit.md` `ui-audit-20260910.md` `debt.md` | 1,588 | M5：自述历史/已被门禁取代；移出规范目录 |
| **合并** | `agent-compatibility.md` 74 → ~20 | −54 | 只留「工具 → 发现文件名」表；删除「加载证据」等不可验证流程 |
| **合并** | adapter 6 份（`CLAUDE`/`GEMINI`/`.cursorrules`/`.clinerules`/`.agents`/`.github`）90 → 13 | −77 | D1/D7 |
| **合并** | `Makefile:19-41` 23 行硬编码测试 → `for f in tests/unit/frontend/*.test.js` | −20 | 消除 C1 那类漂移 |
| **移出规范面 → `docs/evidence/`** | `module-index.md` `shared-frontend.md` | 141 | 是目录/现状清单，不是规则 |
| **净效果（规范文本面）** | **24 文件 / 1,510 行 → 16 文件 / ~846 行** | **−664 行（−44%）** | 规范真源 **2 → 1**；`.cursor/rules/*.mdc`（109 行文件级提示）保留 |
| 规则面合计（含门禁实现与基线，不变） | 38 文件 / 2,381 行 → 30 文件 / ~1,717 行 | −664 | |

**保留并只做小修**：`AGENTS.md`（178→~160，吸收 pre-flight + handoff 模板）、`architecture/database/frontend/testing/quality`（544 行，不动）、
`.cursor/rules/*.mdc`（109 行，文件级提示，不动）。
**MUST 条目数保持 1–16，不新增文本规则。**

### 2.4 hard gate / soft guidance 边界（明确声明）

| 类别 | 内容 | 判据 |
|---|---|---|
| **Hard（红即阻断）** | gofmt、go test、frontend test、go vet、whitespace、secret、architecture 边界、`SELECT *`、ZenTao 主机字面量、local-artifact 索引、**file-length**、**root-artifact** | 机器可判定，无上下文 |
| **Review-only（写进 AGENTS.md，不假称已强制）** | 对象级授权语义、事务/锁序/死锁、N+1 与页查询预算、注释真实性、最小改动判断、UI 真机验收、是否重复已有能力 | 需业务/运行时上下文 |
| **删除（判定不了的规则不扫）** | `NEW_WINDOW`（63 条噪音，quality.md 自认 token 不构成违规） | 文本无法判定 |

### 2.5 STEP 4 门禁修正（均不放松安全要求）

| # | 修正 | 说明 |
|---|---|---|
| F1 | **修好 `check-file-length`** | 见 2.6 决策。目标是「基线如实 + 后续只许降」，把常红死门禁变成真门禁 |
| F2 | `check-gofmt.sh:22` 加 `[[ -f "$file" ]] || continue` | 同 `check-secrets.sh` 已有做法，纯正确性 |
| F3 | `check-patterns.sh` 单遍扫描 + 去 `git hash-object` 逐条 spawn | 8m02s → 目标 <60s；否则 CI 15min 超时必炸（G1） |
| F4 | advisory 扫描范围收窄到 `internal/` `web/static/js/` `web/templates/`，排除 `tests/**` | 消除 G4 的 27 条夹具噪音 |
| F5 | 删除 `NEW_WINDOW` 规则，要求留在 MUST 9 | 消除 G3 的 63 条噪音 |
| F6 | 把 `LOCAL_ESCAPE_HTML` 从 advisory 改成**精确 hard 规则**：`web/static/js` 下 `function escapeHtml(` 定义数必须 == 1 且位于 `ui.js` | escapeHtml 已收敛，规则从「提醒」升级为「封死」（G6） |
| F7 | 新增 `check-root-artifacts`（hard）：仓库根目录不允许未跟踪文件 | 一处同时封死 G5 污染面 + AGENTS.md 已禁止的 drive-by 产物 |
| F8 | 把 `scripts/test-quality-gates.sh` 接入 `make check`（或独立 `make check-gates` + CI job） | 修 C6：门禁自己终于有回归保护（M6） |
| F9 | `.gitignore` 去掉本机绝对路径 | G10 |
| F10 | 修 `quality.md` 的 C1/C2/C3/C4/C5 失效陈述 | 文档漂移不修，收口就是假的 |

### 2.6 唯一需要决策的分叉：F1 怎么修

`check-file-length` 的 21 条全在 HEAD 成立，且防放宽守卫机械上禁止补录。两条非伪造路线：

- **方案 A（推荐）**：把 ratchet 的比较基准从「HEAD 的基线文件」改为「分支 merge-base 的基线文件」，然后做**一次显式的、写进提交信息的基线对账**——把 13 个从未入基线的文件按实测值登记，把 8 个已过期的数字改成实测值。此后机制恢复正常：不许增长、缩小必须同步下调。
  - 代价：一次「承认 500 行规则历史上从未执行」的对账。
  - 收益：门禁当天变真门禁；MUST 8 从死规则变成有牙齿的棘轮。
  - 注意：这**不是「放宽基线换绿灯」**——现状是基线双向失真（8 条错小、13 条漏登），对账是把记录改成事实，且之后**比现在更严**（今天是常红=0 执行）。
- **方案 B**：不动基线，实拆 21 个文件（含 2326 行 `metrics.css`）到 500 行以内。
  - 代价：一次跨前端全站 + Go 模块的大重构，与本轮「收口」任务正交，且会与并行 Agent 大面积冲突。

### 2.7 STEP 6 反向自检（本轮设计是否过度）

| 风险 | 自查 |
|---|---|
| 是否新增了文档增肥？ | 净减 1,290 行；新增文件 0（本文件是任务记录，不在规则面） |
| 是否新增了 MUST 文本？ | 否，MUST 保持 1–16；只吸收 pre-flight/handoff 模板 |
| 是否新增了永久 baseline？ | 否。F1 之后基线必须「只许降」；新增 `check-root-artifacts` 无 baseline |
| 是否产生 Agent 无法执行的规则？ | C6 已消除（自测接入）；`ai-boundary.md` 的空转规则已删 |
| 是否让 Agent 选择空间更小？ | 是：规范真源 2→1、入口重述 7→5 且各 3–4 行、判定不了的规则不扫 |
| 新增 gate 是否过度？ | 只加 1 个（root-artifacts，封死 G5）+ 接入 1 个已存在未运行的自测；其余全是修既有 |

---

## STEP 1–2 结论

**规则更多不是问题，规则不生效才是问题。**
本项目 1,510 行现行规则里，最核心的 MUST 8（500 行上限）**已被证伪为死规则**（21 个文件常红）；
门禁自测**从未运行**；advisory 有 **85%** 长期漂移且 41% 来自一条自认无法判定的规则；
`make check` 与 CI **在 HEAD 即红**。收口的目标不是重写规则文本，而是：
**把唯一真源做薄、把死门禁修活、把判定不了的规则删掉、把软规则声明为软规则。**

待用户确认 2.6 的分叉后进入 STEP 3。
