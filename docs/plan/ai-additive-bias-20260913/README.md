# AI 增肥模式审查报告

审计对象：`/Users/yuyan9923/GitHub/workbench-claude-po`（分支 `release/po-integrate-main-202609`，HEAD `3507193e`）
审计日期：2026-09-13 · **只读审计**（本轮未修改任何代码）
驱动问题：*为什么 AI 无论「加/改/删」功能，结果都是代码行数增加？项目里还有多少同类问题？*

> 本报告自身遵守它审计的规则：无「明确不做」章节、无反向注释、无样板头块。数据全部可复现。

---

## 0. 一句话结论

**项目不存在「AI 只会加」的例外：近 200 次提交中 87.5% 是净增行数，纯删除提交为 0；一个文件在 3 天内从 683 行长到 2326 行。** 这不是模型能力问题，是**规则与门禁的结构问题**——删除路径没有被任何机制奖励，而新增路径被默认放行。

| 指标 | 实测值 | 出处 |
|---|---|---|
| 近 200 次提交 added / deleted | **134,799 / 28,293 = 4.76 : 1** | `git log -200 --numstat` |
| 净增 / 净减 / 持平 提交数 | **165 / 21 / 14**（净增占 87.5%） | 同上 |
| 纯删除提交（added=0） | **0** | 同上 |
| `web/static/css/metrics.css` 3 天增长 | 683 → **2,326** 行（**+1,643**） | `docs/engineering/debt.md:31` vs `git show HEAD:...` |

---

## 1. 为什么会这样：五条机制

1. **「删除」没有对应的授权动词。** 用户能说「加个 X」，但很少说「删掉 Y」。模型在无明确指令时，**删除不在候选动作空间里**——它只会新增。你举的「东坡肉」例子里，去掉东坡肉是你补的指令，不是它自己做的裁剪。
2. **注释被当成交付物。** 主干规范 `.cursor/rules/conventions.mdc` §4 强制每个 Go/HTML 文件写 5–7 行头块，§16 要求「安全边界/架构决策点」写清「为什么」。规范本身合理，但**只规定了「要写」，没规定「写多少算多」**，于是「为什么不需要东坡肉」这类反向解释被当作合规产物保留下来——实测 **165 条**。
3. **门禁只对「新增」设卡，对「该删没删」零约束。** `check-patterns` / `check-file-length` / `statik` 系棘轮全是「不许变差」的单向棘轮。**没有任何一条规则会因为「文件比三个月前更胖」而失败。**
4. **长期红着的门禁 = 没有门禁。** `check-file-length` 已红到被记为 `BLOCKED BY EXISTING BASELINE`（当前 21 项）。红着之后，新增超标文件不再有心理成本——这正是 `metrics.css` 能从 683 涨到 2,326 行的直接原因。
5. **「改功能」被实现成「加一条路径」。** 模型倾向在原实现旁边**并行加一个分支**（`if newMode { ... } else { ...旧... }`），而不是原地替换。于是每次「优化」都留下旧分支，等下一轮再写注释解释「旧分支为什么不用了」。这是历史污染（清单第 3 项）的生成机制。

---

## 2. 审查清单逐项结果

| # | 检查项 | 判定 | 证据量 | 最高等级 |
|---|---|---|---|---|
| 1 | 死代码残留 | **不通过** | 25 个 CSS 文件 / 558 行已证死；8 个 Go 零引用符号 | P1 |
| 2 | 无意义注释 | **不通过** | 反向注释 165 条；强制样板头 1,709 行 | P1 |
| 3 | 开发历史污染 | **不通过** | 点名「旧壳/旧整页/曾经的」注释 8 处 | P1 |
| 4 | 项目/PR 卫生 | **不通过** | 3 个根目录 HTML 报告污染门禁；23 项未跟踪产物；13 个计划目录 | P1 |
| 5 | 重复领域对象 / 第二套流程 | **不通过** | 指标模块前后端两套不相交字典 | **P0** |
| 6 | 假完成 / 未接线 | **不通过** | 死端点 1 个；`localStorage` 假落库；债务文档自述 PARTIAL | **P0** |
| 7 | 危险默认值 / 数据风险 | **不通过** | 删除守卫缺失致有关联数据可被删 | **P0** |
| 8 | 测试覆盖盲区 | **部分不通过** | 6 个夹具曾依赖 fail-open；门禁 21 项长期红 | P1 |
| 9 | 依赖 / 版本风险 | 未发现 | 无 `http://localhost`、无硬编码密钥、无裸 token | — |
| 10 | 未完成 / 假完成 | **不通过** | 3 条 TODO，其中 1 条为真实业务规则缺口 | P1 |
| 11 | 范围外修改 | 无法归因 | 工作区 76 个 `M` 文件分属 3 个并行 Agent | P2 |

---

## 3. 最典型的 5 个「东坡肉」案例

### C1 · P0 · 指标模块：后端有一套真字典，前端另存一套到浏览器
- **证据**：后端 `internal/module/metrics/service.go:244-395` 定义 12 条 `Code:`（`story.total` … `task.open`）；前端 `web/static/js/metrics-manage.js:255,270` 把配置写进 `localStorage`（key `crcb_metrics_meta_config_v2`），全文 **0 处 `fetch`**。
- **Why**：两套字典不相交且无同步机制；页面上「保存成功」是浏览器本地生效，**不回写数据库**。
- **Expected**：单一真源。配置落 `zt_wb_*` 自有表，前端只读服务端。
- **Recommendation**：短期把「指标管理」标为只读（避免用户以为配置已生效）；中期落库。（`docs/plan/metrics-honesty-20260913/README.md` 已记录同一结论，但**尚未实施**。）

### C2 · P0 · 后端实现了 5 分类真聚合，前端 0 引用
- **证据**：`internal/module/metrics/handler.go:48` 注册 `GET /metrics/radar/data`；`web/static/js/metrics-radar.js` 对它 **0 引用**。
- **Why**：真值取到即丢，页面用不到；端点是纯负担（测试、权限、维护成本）。
- **Expected**：端点要么接线，要么删除。
- **Recommendation**：接线或下线，二选一，别留着。

### C3 · P0 · 版本窗口「已关联需求」仍可删除
- **证据**：`internal/module/schedule/service.go:382`
  ```go
  // TODO: 如果窗口已关联需求，不允许删除
  return s.repo.Delete(ctx, req.ID)
  ```
  上一行只校验了 `window.CreatedBy != account`，**关联校验不存在**。
- **Why**：删除有关联需求的窗口会产生孤儿 `zt_demandwindow` 行——正是你那条「工作台与禅道共用同一数据库」评审线要防的数据不一致。
- **Expected**：删除前查关联，非空则拒绝。
- **Recommendation**：实现该守卫 + 一条「有关联时拒绝删除」的单测。**这是本报告里唯一的高危数据风险项。**

### C4 · P1 · 一整套从未落地的后台 shell 设计（`.zen-*`）
- **证据**：`web/static/css/layout/app.css` 原 L42–188，125 行选择器，**全仓 0 引用**（`zen` token 命中 0；模板早已切到 `po/shell.css`）。
- **Why**：属于「先设计了，后来没用」的推测性产物；在仓库里躺到被我删掉。
- **Status**：**本轮已删除（−147 行）**，同一批还删掉 `metrics.css` 的 `.metric-radar-*`（20 类）与 `.metric-detail*`（17 类）死段，共 **−477 行 / +1 行**。

### C5 · P1 · 反向注释：把「不做的事」写成交付物
- **证据**：165 条，典型如
  - `internal/module/po/service_detail.go:189` —「故意不写 ActionURL，避免详情误链到 submit_test.html 旧壳」
  - `internal/module/po/primaryaction/primaryaction.go:10` —「不再兜底为"查看详情"」、`:259` —「不再挂旧整页 /submit-test」
  - `web/static/js/po/done.js:254` —「不再走本地 .done-obj-chip-* 别名」
  - `internal/module/metrics/radar_service.go:45` —「防御：buildItems 不应输出未在白名单的分类，但万一出现就跳过」（**为不可能状态写的防御分支**）
- **Why**：注释在描述一个**已经不存在**的形态。三个月后读代码的人不知道 `submit_test.html` 是什么，只学到一堆噪音；被删掉的模块名反被注释永久保存。
- **Expected**：注释只说「现值 + 不变式」。理由属于 commit message / PR 描述，不属于代码。
- **Recommendation**：按「删掉后是否能理解代码」判定——能，就删。

---

## 4. 其余逐项证据

**P1 · 死代码（清单 1）** 已证死但未删：`po/follow.css` 137 · `po/demand-detail.css` 55 · `permission/list.css` 52 · `schedule/schedulelist.css` 50 · `po/todos.css` 42 · `permission/drawer.css` 40 · 其余 19 文件 182。8 个 Go 零引用符号（`inputShape`、`KeyEdit`、`KeyPublish`、`KeyViewEvaluate`、`ObjectSubDemand`、`errDeliverNotAcceptanced`(拼写错误)、`errDeliverAlreadyWait`、`TimeRangeThisWeek/ThisMonth`）。

**P1 · 样板注释（清单 2）** 强制文件头 **1,709 行**（Go 213 文件 / 1,411 行 + HTML 40 文件 / 298 行）。既无运行时价值，又会腐烂——已抓到 3 处 `// 文件:` 路径与实际路径不符。`web/templates/schedule/index.html:17` 自己也承认「内联 style 过多（约 135 处）」；实测该文件 `style="` **70 处**，全部模板 **201 处**——**连自述数字都是错的**。

**P1 · 绕过共享层（清单 5）** `web/static/js/metrics-manage.js` 用 `alert()` **5 处**（`:447,452,457,468,474`），而共享层 `web/static/js/ui.js:33` 已有 `showToast`、`:729` 已导出 `window.showToast`。这是「第二套流程」的前端版本。

**P1 · 静默失败（清单 7）** 9 处 `catch (e) {}` / `.catch(function(){})`：`layout/theme.js:94`、`po/po-profile.js:117`、`po/demand-detail-review.js:51`、`po/done.js:99` 等。错误被吞掉 = 危险默认值（清单第 7 项）。

**P1 · 假完成（清单 6/10）** `docs/engineering/debt.md:1-14` 自述 `Status is PARTIAL`、「Two remaining test migrations and production file splits are not complete」、「browser acceptance has not been certified」。**文档是诚实的**，但因此暴露：上一个「清理 wave」本身就没有收口，而这轮又在它上面叠了新目录。

**P1 · 产物累积（清单 4）** 23 项未跟踪：3 个根目录 HTML 报告（直接给门禁贡献 advisory——`code-audit-2026-09-13.html` 7 处 `WEAK_PASSWORD_HASH`、`code-review-2026-09-10.html` 2 处）、7 个 `.cursor/plans/*.md`、`.hermes-cache/`、2 个未跟踪 Go 文件、4 个未跟踪测试。

**P1 · 文档体量失衡** `docs/plan` **104 文件 / 30,450 行**（13 个计划目录 + 1 个 md），`.cursor/plans` **12 文件 / 2,215 行**。对照生产代码：`web/static/js` 26,090 行、`web/static/css` 18,856 行、`web/templates` 8,033 行。**计划/过程文档比整个前端生产代码还多**；其中 5 个文档带「明确不做 / forbidden-actions」章节——「不做」被写进文档，但没人真正执行它。

**P2 · 提交信息命名（清单 3/4）** `7461b5e2 fix(po): clean legacy UI remnants and review actions`、`c8f648f1 chore(po): remove unreachable follow and todo code`、`3507193e … remove redundant body titles v1`、`5dcc7df9 … architecture v1`。**`v1` 后缀就是这个项目里的「（无东坡肉）」**。

**健康项（如实记录，不硬凑问题）** 遗留标记仅 3 条 TODO、0 条 FIXME/XXX/HACK；`console.log` 仅 1 处（`agileteam.js:79`，在 `if (window.console)` 调试分支内）；无 `http://localhost`、无硬编码密钥/token；裸 IP 41 处、密码字面量 4 处，除 `tests/integration/db_safety.go`（回环地址白名单 + 破坏性 DDL 显式 opt-in 的安全守卫，刻意为之）外**全在 `_test.go` 夹具内**，属正常。

---

## 5. 可落地的规则与门禁改动

| # | 改动 | 对应机制 | 成本 |
|---|---|---|---|
| R1 | **新增「净删除」提交约定**：每 N 次功能提交必须跟随 1 次纯删除提交（`chore: remove dead X`）。可先用人工约定，成熟后写进 PR 模板 | 机制 1 | 低 |
| R2 | **门禁加「不许变大」棘轮**：对每个文件同时记录「当前行数」与「30 天前行数」，只允许 `current <= baseline`；新超标文件**不许入基线**（现有 `file-length.tsv` 已有此语义，只是被 `BLOCKED BY EXISTING BASELINE` 架空了） | 机制 3/4 | 中 |
| R3 | **注释规范补上限**：§4 头块从 5–7 行压到 1–2 行；§16 增加「禁止反向注释（描述已不存在的形态）」并给出正反例 | 机制 2 | 低 |
| R4 | **把「为什么」赶出代码**：架构/历史的理由只允许写在 commit message 或 PR 描述里。代码注释的验收标准改为「删掉注释后是否还能理解代码」 | 机制 2/5 | 低 |
| R5 | **`make check` 必须全绿**：21 项 file-length debt 要么真删到家、要么走正式基线登记。红着的门禁是所有其他措施的失效前提 | 机制 4 | 高（需专项） |
| R6 | **把根目录当禁写区**：审计报告产物一律写到 `docs/audit/`（`*.md`，不进 `check-patterns` 扫描扩展名），别放仓库根 | 清单 4 | 低 |
| R7 | **`|| {}` / `catch (e) {}` 加 advisory 计数**（只报不挡），逼出「兜底分支为哪个真实失败态服务」的说明 | 清单 7 | 低 |

---

## 6. 本次未核验 / 无法归因

- **清单 11（范围外修改）**：工作区 76 个 `M` 文件分属 3 个并行 Agent（Codex / Cursor / HY4），**无法按提交归因**。需要你按 Agent 分线做一次 `git diff` 归因审查，或改为每 Agent 一个 worktree。
- **清单 9（依赖/版本风险）**：仅做了字符串级扫描（localhost / 裸 IP / 密钥字面量），**未做 `go.mod` / `vendor/` 的 CVE 比对**。
- **`docs/Demo` 的 169 个删除**当前是未暂存状态（−116,906 行），归属上一个决策，未计入本轮。
- 本轮 **未修改任何代码**；上一轮（Slimming Batch S1）已执行 −477 行净删除，未 commit / 未 stage。
