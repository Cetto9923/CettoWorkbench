# Phase AI0 — 冻结未来员工AI能力边界

Complexity: MEDIUM
Risk: HIGH
Recommended Executor: Strong Reasoning + Coding Agent
Planning Status: READY DOCUMENTATION ONLY
Dependencies: G0；不依赖实现AI
Evidence Base: f5f97a42b802b2eadabb393f18a88bb8fcb1aeec / 2026-09-05

本卡独立交接时必须同时读取根 AGENTS.md、受影响的 docs/engineering/{architecture,database,frontend,testing,quality}.md 以及本包总计划。以当前用户要求和已批准业务决策为业务真源；历史代码只说明现状。
开工记录 repo root、branch、HEAD、status、脱敏remote和前置报告。main/master禁止写；禁止未授权branch create/switch、merge、rebase、cherry-pick、reset、clean、stash、force-add、bare push。保留全部既有WIP。Commit Boundary不构成commit/push授权。
只做本卡Allowed；除此以外默认Forbidden，尤其AGENTS、其他专题、baseline、CI、无关业务。每卡可更新自己的外部验收报告；不能顺手改全局工程规则。遇到合同/schema冲突或将覆盖WIP，STOP AND REPORT；停止受阻部分，其他卡必须重新获指派才执行。

## 1. Goal

后续AI功能有身份、数据、工具、确认、故障与验收约束，不提前建设框架。

## 2. Why This Phase Exists

员工工作台未来接模型，复用现有Service边界可防止模型成为权限决策者；本轮只定义合同。

## 3. Confirmed Evidence

E36 | AGENTS分层/授权规则；当前源码未建设本轮AI能力；用户已确认内网默认、首功能只读、当前不选供应商。

## 4. Current Behavior Contract

模型API、RAG、向量库、tool gateway未实现，本卡不能据未来需要新增空接口。

## 5. Target Behavior Contract

WHEN检索前 THEN服务器session决定actor并过滤数据；不把无权记录先送模型后隐藏。
WHENtool执行 THEN调用现有业务Service重新鉴权，不能任意SQL或信任模型actor。
WHEN附件/检索文本包含指令 THEN视为数据，不覆盖系统/工具权限。
WHEN输出进入HTML/SQL/命令 THEN按目标上下文验证编码，不直接执行。
首功能只读；未来写需展示具体对象/字段差异、员工确认绑定本次意图、执行时再鉴权和防重放，过期确认失效。
模型timeout/预算/取消/并发上限由首功能任务给数值后才准上线，普通工作台不依赖模型成功。
只允许获准内网环境；供应商变更独立批准；回答带用户可访问来源和时间，无证据明确未知。

## 6. Allowed Change Scope

docs/engineering/ai-boundary.md（新增）及module-index.md一行导航；0实现。

## 7. Explicitly Forbidden Scope

除 Allowed 外全部禁止；尤其 AGENTS.md、无关 docs/engineering 规范、其他模块、CI及baseline。卡内明确列出的精确减债/专题更新为唯一例外。禁止连接或修改未批准环境。

## 8. Implementation Constraints

禁止模型router/provider/manager/空interface/vector库/依赖SDK；不选择供应商；不写虚构token预算数值。

## 9. Suggested Implementation Direction

**RECOMMENDED, NOT MANDATORY**：优先复用现有局部调用链，以能满足目标合同的最小改动实现；局部命名/SQL形态自由，行为、权限和scope不可自由改变。

## 10. Data / Schema Contract

无DDL；索引/embedding/缓存未来也携带文档ACL及失效删除，未有实际AI需求不建存储。

## 11. Authorization Contract

认证来自session；工具cap/object在每次执行重验；不同员工conversation/cache严格隔离；确认不能替代权限。

## 12. Transaction Contract

本轮无写；未来业务tool复用Service事务且确认ID一次性，同key同payload重试不重复作用，不同payload拒绝；详细协议在首写功能卡冻结。

## 13. Error / HTTP Contract

本轮无AI API；未来错误不得泄露prompt、凭证或无权来源。

## 14. Required Regression Tests

文档规定未来强制场景：CrossUserRetrieval、PromptInjectionInAttachment、RevokedPermissionBeforeTool、ForgedActor、ToolArgumentValidation、DuplicateConfirmation、ProviderTimeout、BudgetExhaustion、SourceVisibility。
每场景写明Setup/Action/Expected；本轮只检查规约完整，不能把未实现测试标PASS。

## 15. Testing Level

文档架构评审；实际AI测试NOT IMPLEMENTED且不算本轮实现。

## 16. Validation Commands

从获准仓库根目录运行：
```sh
git diff --check
make check
```
新增路径/target属于本卡交付，当前未存在不能冒充已可运行。Go -run 表达式必须匹配真实测试：实现时测试总名采用本卡命令中的suite前缀，子测试使用第14项场景名；执行前用go test -list核对，零测试或SKIP不算PASS。所有需要integration的验收用批准的WB_TEST_MYSQL_DSN注入；不在命令参数/报告打印连接值。没有环境标PARTIAL/BLOCKED。

## 17. Manual Verification

架构评审确认普通员工流程不依赖AI；任一未来上线任务必须填写真实模型环境、字段白名单、限制数值和上述测试证据。

## 18. Acceptance Criteria

身份/数据/内容/工具/行动/可靠性/来源七边界齐；无AI实现；未来必测表完整。

- [ ] 第5项每个WHEN结果满足，证据可定位。
- [ ] 第14项适用测试有OLD FAIL/NEW PASS；纯文档/测量任务明确N/A原因。
- [ ] make check 与 git diff --check 通过；存量债单列。
- [ ] 所有该卡必需人工/集成证据齐全，无SKIP冒充PASS。
- [ ] diff在白名单内；独立review结果记录。缺任何项不得COMPLETE。

## 19. Stop Conditions

若实际证据与目标合同冲突、测试不能击中旧缺陷、需要越界编辑或覆盖已有WIP，STOP AND REPORT。
共同STOP：main/master、未知schema被当真源、工具失败被吞、为绿灯扩大baseline。基线已有失败记BLOCKED BY EXISTING BASELINE；可继续本卡独立准备，不能发布为完成。

## 20. Decision Gates

无新的用户业务决策。出现未列政策需求时写Question/Why/Options/Recommended/Risks/Blocked/Unblocked，提交计划owner；不得自行批准推荐值。

## 21. Out-of-Scope Findings

只写入本卡完成报告 Out-of-scope findings（路径、符号、现象、建议归属卡、证据等级），不顺手修。已有计划卡优先引用；未确认问题标 NEEDS VERIFICATION。

## 22. Expected Diff Shape

1–2Markdown，0代码/config/schema/依赖。

## 23. Commit Boundary

docs(ai): define employee AI service and data boundaries
独立review和回退评估不等于自动发布。安全卡回退必须保留访问封锁或前向修复，不恢复已知越权/CSRF缺口。DB迁移不以DROP撤销审计记录。

## 24. Completion Report Template

```text
Phase:
Status: COMPLETE / PARTIAL / BLOCKED
Input HEAD / rules hash / schema evidence:
Files changed:
Behavior changed:
Regression tests (test name, old failure, new pass):
Commands run (exit code and evidence path):
make check (baseline debt listed):
git diff --check:
Commit SHA (NOT COMMITTED if absent):
Remote (NOT PUSHED if absent):
CI Run / tested SHA:
CI Result (NOT RUN if absent):
Manual verification:
Remaining risks:
Out-of-scope findings:
Next authorized task:
```
COMPLETE仅指本卡所有required验收通过；NOT COMMITTED/NOT PUSHED如实填写，不能伪造SHA或CI。最终交付另受总计划I0和用户验收约束。
