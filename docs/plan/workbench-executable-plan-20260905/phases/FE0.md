# Phase FE0 — 共享前端能力只认证不大改

Complexity: LOW
Risk: LOW
Recommended Executor: General Coding Agent
Planning Status: READY AFTER DEPENDENCIES
Dependencies: G0；H0；X1
Evidence Base: f5f97a42b802b2eadabb393f18a88bb8fcb1aeec / 2026-09-05

本卡独立交接时必须同时读取根 AGENTS.md、受影响的 docs/engineering/{architecture,database,frontend,testing,quality}.md 以及本包总计划。以当前用户要求和已批准业务决策为业务真源；历史代码只说明现状。
开工记录 repo root、branch、HEAD、status、脱敏remote和前置报告。main/master禁止写；禁止未授权branch create/switch、merge、rebase、cherry-pick、reset、clean、stash、force-add、bare push。保留全部既有WIP。Commit Boundary不构成commit/push授权。
只做本卡Allowed；除此以外默认Forbidden，尤其AGENTS、其他专题、baseline、CI、无关业务。每卡可更新自己的外部验收报告；不能顺手改全局工程规则。遇到合同/schema冲突或将覆盖WIP，STOP AND REPORT；停止受阻部分，其他卡必须重新获指派才执行。

## 1. Goal

提供可复用且注明边界的fetch/token/error示例，禁止把legacy模块当万能模板。

## 2. Why This Phase Exists

app.js注释声称唯一脚本/表单优先已与当前多脚本和MUST写协议不同；实际appFetch非JSON自动处理器。

## 3. Confirmed Evidence

E33 | web/static/js/app.js头部与appFetch；web/static/js/schedule/schedulefetch.js |能力合同与旧注释不同；多个模块独立fetch存在，不等于全部可合并。

## 4. Current Behavior Contract

shared ui/components/layout/module分工已有；没有认证golden module。

## 5. Target Behavior Contract

WHEN模型查共享目录 THEN看到真实输入输出、失败处理、适用/不适用、引用示例和测试证据。
认证能力限H0/X1已验证的传输/错误/token，不宣称全user模块合格。
不同caller合同不相同则保留局部代码并记录原因，不为了减少重复改变行为。

## 6. Allowed Change Scope

docs/engineering/shared-frontend.md、module-index.md；web/static/js/app.js仅失效头部注释；其余源码只读。

## 7. Explicitly Forbidden Scope

除 Allowed 外全部禁止；尤其 AGENTS.md、无关 docs/engineering 规范、其他模块、CI及baseline。卡内明确列出的精确减债/专题更新为唯一例外。禁止连接或修改未批准环境。

## 8. Implementation Constraints

不做全仓前端迁移、不改样式、不认证未测试toast/modal；超长文件拆分另卡经证据批准。

## 9. Suggested Implementation Direction

**RECOMMENDED, NOT MANDATORY**：优先复用现有局部调用链，以能满足目标合同的最小改动实现；局部命名/SQL形态自由，行为、权限和scope不可自由改变。

## 10. Data / Schema Contract

N/A：本卡不改变数据库结构或数据合同；禁止DDL和真实业务库写入。若实际实现需要DB变更，停止并提出范围修订。

## 11. Authorization Contract

N/A：本卡不新增业务授权策略。保留现有认证/能力/对象检查；不能借技术修复扩大权限。

## 12. Transaction Contract

N/A：本卡不改变业务事务或写入语义。测试仅使用临时文件/合成输入；不得对实际业务数据做写操作。

## 13. Error / HTTP Contract

N/A：无HTTP合同变更；现有成功响应、页面地址和调用方不改。若发现必须修改HTTP行为，需在本卡目标中先冻结而非临场发明。

## 14. Required Regression Tests

CapabilityExampleReview |目录示例|对照真实函数/测试|输入输出和错误一致。
FalseReferenceRejected |把appFetch描述成自动JSON|评审|拒绝。
文档/注释任务无业务红转绿要求，不添加低价值镜像测试。

## 15. Testing Level

文档引用验证+既有H0/X1证据复用，不虚构浏览器结果。

## 16. Validation Commands

从获准仓库根目录运行：
```sh
node --check web/static/js/app.js
make check
git diff --check
```
新增路径/target属于本卡交付，当前未存在不能冒充已可运行。Go -run 表达式必须匹配真实测试：实现时测试总名采用本卡命令中的suite前缀，子测试使用第14项场景名；执行前用go test -list核对，零测试或SKIP不算PASS。所有需要integration的验收用批准的WB_TEST_MYSQL_DSN注入；不在命令参数/报告打印连接值。没有环境标PARTIAL/BLOCKED。

## 17. Manual Verification

人工审阅diff与原始证据，核对无越界、无秘密、测试确实命中旧缺陷。未触及UI无须制造浏览器验收；适用卡要求的真实环境证据不能用静态检查替代。

## 18. Acceptance Criteria

每项认证有版本/测试；未认证明确标；0行为改动。

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

2文档+1注释文件；0新抽象/DDL/CI。

## 23. Commit Boundary

docs(frontend): certify bounded shared capabilities
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
