# 给执行 Agent 的提示词

你负责执行 CRCB Workbench 的 Agent 约束闭环与性能治理。先读取本目录 AUDIT.md、PLAN.md 和仓库 AGENTS.md、docs/engineering/agent-onboarding.md、architecture.md、database.md、frontend.md、testing.md、quality.md。

仓库：`/Users/yuyan9923/GitHub/workbench-claude-po`。目标分支 Claude-PO。
计划审计基线：`8adbe0f7cc918dd1ca61aa481e71c32e23e0cc92`。
不要假定仍是当前HEAD，不切回旧版本。先执行pwd、git rev-parse --show-toplevel、git branch --show-current、git status --short、git rev-parse HEAD、git remote -v。

本次默认只执行 PLAN 的第一批 AE-01、AE-02；完成后停止。用户指定其他卡时只执行指定卡及其已获授权前置。不要自动执行旧Executable Plan中的COMPLETE卡，不要自动推进第二批，不要因为发现性能问题就顺手改首页SQL。

在修改前给出：当前SHA与WIP、实际读取的规范及适用MUST、卡范围、允许修改的文件、业务来源、测试环境、验收命令。保留全部用户WIP，尤其未跟踪test-results/。本任务没有授权push、切分支、发布、改公司origin或访问未知数据库。

必须解决的规则缺口：

1. 工程规范只保留一个真相入口，各模型适配器引用它。实际冷启动加载证明与“文件存在”分开记录。
2. 当前门禁允许同时扩大文件baseline后通过，必须用可信旧基线检查扩大例外；自测模拟660→728场景，不能通过改当前baseline使其绿灯。
3. 将已有3个Node行为测试和门禁自测纳入实际执行入口/CI，准确分类，不能当真实浏览器E2E。
4. 现有集成测试只黑名单排除zentaopms，但会删建表。先实现拨号前的明确隔离目标验证；未确认实例绝不执行seed或DDL。
5. 核验story任务抽屉的Service对象授权，使用直接请求的无权actor负例。只有确认缺陷后做最小修复；没有业务scope证据则记录具体WAIT DECISION，不能猜成所有人可见或仅本人可见。

每卡遵循PLAN的固定决策、允许diff和验收，不发明新框架、通用引擎、Redis缓存或全站重构。不要简单把所有advisory升级hard；那些报告里包含共享组件和测试模拟的误报。

已有make check通过不等于语义/性能/安全全面通过；旧远端CI不是当前代码的CI。运行并报告git diff --check、make check及本卡新增测试。CI/权限保护不可用时说明实际限制，不得修改仓库可见性或套餐。

未来用户明确授权性能卡后，重点遵循：

- 首页所有候选ID仍在Go去重分页，详情分页不代表查询已优化；改成SQL候选去重/计数/排序/分页，保持阶段首次命中与kind:id语义。
- 保留待办/通知已有SQL分页，不重做旧卡。先用当前二进制、认证页面、新数据规模采集基线。
- 先按本页账号查询展示名，再考虑公共成本；不得先缓存权限或全局actor结果。
- 看板新增分层加载必须携带真实分页与逐请求对象授权；不把Limit后的数量当完整总数。
- 8093在审计时离线，未采集当前延迟。端口恢复也必须核对运行产物版本；不得编造优化百分比。

输出本批报告：Status、代码基线、实际改动、规范加载证据、防放宽测试、测试DB安全测试、对象授权证据、验证命令及退出码、未完成项、Commit/CI。任何必需门禁未过不得写COMPLETE；然后STOP。
