# 第二轮债务修复执行记录

Scope Contract
目标 / Goal: 按用户指定提词落地 P0 和 P1-1 至 P1-4，本地提交，不 push。
必须改变 / Must change: 敏捷小组允许零工时；两条交付路径共用缺陷守卫；单一调整说明；reviewID 错误上抛；真实用户选项映射回归。
允许影响 / Allowed to affect: agileteam 工时校验及测试、po 交付/focus/stage 与看板弹窗及测试、testtask 标签测试、Makefile 注册看板测试、本记录。
明确不处理 / OUT_OF_SCOPE: 禅道扩展、角色矩阵、项目团队规则、全局样式、数据库结构、无关 WIP。
预计修改 / Expected to modify (file:symbol): service_validation.go:validateAdjustmentWriteBoundary; service_adjustment.go:ConfirmAdjustment; po service/handler deliver; home_focus_repo.go; home_stage_repo.go 及必要调用方; workboard-modal.js; labels_test.go。
预计不修改 / Expected NOT to modify: deploy/zentao、权限配置、门禁阈值、已有未跟踪文档。
验收条件 / Acceptance: 定向 Go/前端测试、git diff --check、make check、适用浏览器/集成验收；缺失门禁明确报告。
发现的额外问题 / Extra findings: DeliverDemand 先读 JSON 再调用 homeAction 二次读 body，极简请求会被 EOF 拒绝；同交付入口修复。

## Pre-flight
- Repository: /Users/yuyan9923/GitHub/workbench-claude-po
- Branch: release/po-integrate-main-202609
- HEAD: fdcf130248915daef0a95f2cd5a074f46629860e
- WIP: 4 个 .cursor/plans 未跟踪提词，docs/audit/、docs/release-requirements/ 未跟踪；保留且不提交。
- 已读规则: AGENTS.md; docs/engineering/{architecture,database,frontend,testing,quality,maintainability,agent-compatibility}.md。

## P0 排查结论（修改前）
1. workboard-modal.js:108-144 从 PoWB 选中小组取 ID，读 agile-teams/:id；agileteam/repo.go:125-139 查 zt_teamgroup。parent/type 是小组层级属性，不是 zt_team 项目类型。
2. SubmitAdjustment 写 zt_wb_agileteam_adjustment / adjustment_item / history；availableHours 保存在 item，ConfirmAdjustment 通过 repo_confirm.go:106-126 写 zt_team.hours，严格限定 type=teamgroup、root=小组 ID。项目团队使用同表但不同 type/root；同字段不足以证明相同业务约束。
3. PMO agileteam-members.js:210 默认 hours=7，:323-344 提交同 API。看板候选人无 hours，提交 Number(m.hours)||0。34f7ff56 引入的 workboard-team-modal.test.js 明确禁止工时列；历史提交未提供是否必填的产品结论。
4. service_validation.go:54 和 service_adjustment.go:179 均拒绝 add/roleChange 的 0；form.go:255 的 Validate 不校验工时。本仓搜索未发现独立项目团队可用工时校验实现，不能推断禅道外部规则。
5. 临时 Go overlay 调用实际 SubmitAdjustment Handler，add / roleChange 的 availableHours=0 均 HTTP 400，错误为“可用工时必须大于 0 且不超过 24”，无 DB 访问。TestRound2ZeroHoursReproduction PASS（复现成立）。
6. 用户在本会话明确确认“A：敏捷小组不要求工时”。选择路径 A：允许 [0,24]，保留非零历史值，不给看板加列，不动项目团队规则。

DB 边界：仅改变既有校验可接受的值，不调整 schema、SQL 写入顺序/字段、事务与权限；本轮未授权共享业务库实写。SQL mock 不证明真实并发安全。

## 实现与回归
- P0：两处校验改为 [0,24]；TestAdjustmentHoursBoundary 覆盖 add/roleChange 的 -1/0/7/24/25；TestSubmitZeroHoursHandlerCreatesPendingAdjustment 实际 Handler → Service → Repo 的 sqlmock 待确认单提交返回 200，并断言 availableHours=0 写入参数；TestConfirmAdjustmentWritesTeamAtomically 覆盖 0/7 的确认落库参数。
- P1-1：service_deliver.go:checkDemandDeliverBlockers 被完整表单和 homeAction deliver 共用；Handler 只解析一次 body。严重缺陷返回 409、数据库故障保留 500；极简成功保留 /home redirectUrl。新增两条请求分支的严重缺陷/DB 故障与极简成功回归。
- P1-2：删除重绘 body 中未提交的 kbTeamReason，保留 footer kbTeamAdjustReason。前端测试加载生产脚本、点击候选回调、调用真实 submit，断言唯一 POST 的 reason / account / actionType / availableHours；Makefile 注册该测试。
- P1-3：6 处吞错改为显式 error 返回，查询构造器及所有调用方同步返回 error；不改变 SQL 条件和角色矩阵。TestReviewLookupErrorsReachCallers 覆盖 10 个入口，不允许查询错误后继续查列表或计数。
- P1-4：TestUserOptionLabelDeduplication 调用真实 Repo.ListInsideUsers，断言裸姓名、已有账号后缀、空姓名的标签及 account/pinyin。
- P2：仅随交付守卫修复业务错误 HTTP 状态，并补看板提交行为测试；其他可选项未扩大。

## 验证记录
- 修改前 Go overlay TestRound2ZeroHoursReproduction：exit 0，add/roleChange 均复现 HTTP 400。
- go test ./internal/module/agileteam/... ./internal/module/po/... ./internal/module/testtask/... ./internal/module/schedule/...：exit 0。
- 新增测试均已实际执行；编写期间出现未使用 import、sqlmock 字段顺序不匹配，修正测试后重跑通过，未放宽生产断言。
- node tests/unit/frontend/workboard-team-modal.test.js：exit 0（静态 + 真实生产提交函数行为）。
- make check：exit 0，含 go test ./...、frontend、vet、gofmt、长度、密钥、架构；30 个既有超长文件，87 条 advisory，其中两条未记录 inventory（未改 handler.go / done.js）。
- make check-gates：exit 0，38 passed。
- git diff --check：exit 0；withdrawreview.php 不存在。
- 真实浏览器：控制工具首次超时，重连成功；http://localhost:8090 重定向 /login，无可用登录态。已展示登录页供用户恢复；当前未完成登录后的点击/视觉/Console/network/加载资源版本验收。
- 共享 MySQL 写入/并发：未执行本轮业务实写，不以 sqlmock 声称死锁、并发安全或线上 schema 已验证；未更改原事务/锁顺序。
- CI：未 push，未触发远端 CI。

## 交接
- 新增行用于显式错误返回、两入口共用守卫与生产路径回归，不引入框架或新抽象层。查询数量未新增，错误时提前停止；SQL 排序/过滤/分页语义保持。
- 未调整门禁阈值/基线，未触碰禅道扩展、项目团队约束、角色矩阵或已有未跟踪文档。
- 应用状态：PARTIAL，自动化通过，真实登录后的 UI 与目标环境验收仍缺失。文档与本地提交不等于应用可交付。

## 本地 commits（未 push）

- 2009dfbfd7bdd0d46165b101699d9e326a8992d6 fix(agileteam): allow zero hours for agile members (path A)
- 504d2f23025d85f9aa7808bb55dd09369a280e71 fix(po): enforce deliver blockers on both submission paths
- 20ba8055928d13acc22aaa77168b4f2ba246609d fix(board): retain one submitted adjustment reason
- 753acf6967becdc3b14703d182d5f4fed127b032 fix(po): propagate pending review lookup errors
- 2c5090ff2d3adccef30aa493585801ae2ab66964 test(testtask): verify user labels through actual repository mapping

最终暂存树 make check exit 0；随后仅整理两处测试 import 分组，相关包重跑 exit 0。git diff --cached --check exit 0。
