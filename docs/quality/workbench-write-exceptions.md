# 禅道直写逐项登记与发布前置条件

源码版本：Dev-CT `7d45c28a`，对照 Main `0a892260`。隔离验收版本为禅道 max 5.6.1、公司 Main 对应原生扩展 `7ba6632c`、OceanBase CE 4.2.5.5；原生索引已核实。生产运行版本未据此认定。以下登记不证明原生接口缺失，也不构成上线授权。不得以 Client 未实现某方法推断禅道没有接口。

已替换的原生动作：计划创建、业需转研需、研需指派、任务创建/编辑/删除、研需删除、业需验收与交付。共享 Client 对写动作的 `result=fail` / `status=fail` 识别为失败；读取审批提示保留 Main 的响应含义。HTTP 调用均放在本地事务之外，不自动重放创建或删除。

所有下列例外当前均为**阻止上线，待验证**。逐项补齐部署版本、原生接口目录与响应契约、字段支持、权限、禅道索引及共同并发协议后，才能决定迁移至接口或批准保留。若原生接口可满足，则必须删除例外及对应直写。

## 共用的权限与并发依据

- 排期：Service 先校验根对象写权限、研需/任务归属和产品/执行范围。窗口关联本地事务锁窗口；独立研需的记录、描述及历史在同一短事务完成。已验证原生交付提交期间删除窗口不被 HTTP 阻塞，随后关联失败明确提示原生已提交且不新增关联；窗口更新与删除竞争已验证。原生并发首次建计划仍未验证。
- 小组：普通信息保存只锁目标行，观察到父级变化则失败；拓扑暂保留旧全树一致性协议。成员确认先锁所属小组，再处理调整单，账号固定顺序。该锁只能证明工作台内部排序，尚不能证明禅道也遵守，不能声称跨系统竞争已解决。
- 个人资料：仅当前账号与 ID 的 Service 校验；资料和工作台偏好一次事务提交，失败回滚。保存前按 ID / account / deleted 锁当前用户行，防止校验后删除导致偏好单独成功。隔离验证了双系统 MD5 密码登录、原会话可用并恢复测试密码；账号并发覆盖仍需扩展。
- 用户管理：既有管理路由权限和 Service actor 校验；本轮未改业务行为。原生账号 CRUD 与并发协议尚未验证。
- PO 关注/催办：沿用对应 Service 的对象权限和既有 SQL；关注补缺行、通知、审计需验证原生能力及重复写并发。
- action/history 通道：仅允许由已通过授权的业务调用方使用；同一事务维护本地直写的历史。不能为已经由禅道接口处理的动作再补写历史。

## 逐项清单

| 文件 / 函数 | 原生表 | 字段 / 动作 | 权限和并发依据 | 版本与验收 |
|---|---|---|---|---|
| `internal/module/agileteam/repo_confirm.go` / `ApplyConfirmedAdjustment` | `zt_team` | root/type/teamgroup/account/role/join/days/hours/estimate/consumed/left/order；成员删除 | 见上述小组依据 | Main 原生扩展 7ba6632c；本项发布验证未完成 |
| `internal/module/agileteam/repo_confirm.go` / `upsertTeamMemberTx` | `zt_team` | root/type/teamgroup/account/role/join/days/hours/estimate/consumed/left/order；成员删除 | 见上述小组依据 | Main 原生扩展 7ba6632c；本项发布验证未完成 |
| `internal/module/po/repo_home_actions.go` / `insertHomeDemandAction` | `zt_action` | objectType/objectID/product/project/execution/actor/action/date/comment/extra | 见上述PO 关注/催办依据 | Main 原生扩展 7ba6632c；本项发布验证未完成 |
| `internal/module/po/repo_home_actions.go` / `insertAcceptanceUrge` | `zt_action` | objectType/objectID/product/project/execution/actor/action/date/comment/extra | 见上述PO 关注/催办依据 | Main 原生扩展 7ba6632c；本项发布验证未完成 |
| `internal/module/po/repofollow.go` / `EnsureDemandUnfollowed` | `zt_starinfo` | objectType/objectID/account/followed | 见上述PO 关注/催办依据 | Main 原生扩展 7ba6632c；本项发布验证未完成 |
| `internal/module/po/repofollow.go` / `RemoveProjectReportFollow` | `zt_project` | follow（项目周报关注用户 ID CSV） | 见上述PO 关注/催办依据 | Main 原生扩展 7ba6632c；本项发布验证未完成 |
| `internal/module/po/repo_home_actions.go` / `insertAcceptanceUrge` | `zt_notify` | 催办消息关联对象、收件人与消息内容 | 见上述PO 关注/催办依据 | Main 原生扩展 7ba6632c；本项发布验证未完成 |
| `internal/module/schedule/repo_scheduling_write.go` / `CreateAction` | `zt_action` | objectType/objectID/product/project/execution/actor/action/date/comment/extra | 见上述排期依据 | Main 原生扩展 7ba6632c；本项发布验证未完成 |
| `internal/module/schedule/repo_history.go` / `LogEditedRecord` | `zt_action` | objectType/objectID/product/project/execution/actor/action/date/comment/extra | 见上述排期依据 | Main 原生扩展 7ba6632c；本项发布验证未完成 |
| `internal/module/schedule/repo_history.go` / `LogEditedRecord` | `zt_history` | action/field/old/new | 见上述排期依据 | Main 原生扩展 7ba6632c；本项发布验证未完成 |
| `internal/module/schedule/repo_scheduling_write.go` / `CreateStory` | `zt_story` | title/product/assignedTo/estimate/排期日期/来源字段；独立研需创建 | 见上述排期依据 | Main 原生扩展 7ba6632c；本项发布验证未完成 |
| `internal/module/schedule/repo_scheduling_write.go` / `UpdateStory` | `zt_story` | title/product/assignedTo/estimate/排期日期/来源字段；独立研需创建 | 见上述排期依据 | Main 原生扩展 7ba6632c；本项发布验证未完成 |
| `internal/module/schedule/repo_scheduling_write.go` / `CreateStorySpec` | `zt_storyspec` | story/version/title/spec | 见上述排期依据 | Main 原生扩展 7ba6632c；本项发布验证未完成 |
| `internal/module/schedule/repo_scheduling_write.go` / `UpdateDemandScheduling` | `zt_demand` | QD/RD/estimateLaunch/developFinish/testFinish/verifyFinish/assignedTo/assignedDate | 见上述排期依据 | Main 原生扩展 7ba6632c；本项发布验证未完成 |
| `internal/module/schedule/repo_story_link.go` / `EnsurePlanStoryRelation` | `zt_planstory` | plan/story | 见上述排期依据 | Main 原生扩展 7ba6632c；本项发布验证未完成 |
| `internal/module/schedule/repo_story_link.go` / `ReplaceProjectStory` | `zt_projectstory` | project/story/product/branch/version/order | 见上述排期依据 | Main 原生扩展 7ba6632c；本项发布验证未完成 |
| `internal/pkg/ztaction/action.go` / `Create` | `zt_action` | objectType/objectID/product/project/execution/actor/action/date/comment/extra | 见上述action/history 通道依据 | Main 原生扩展 7ba6632c；本项发布验证未完成 |
| `internal/pkg/ztaction/action.go` / `LogHistory` | `zt_history` | action/field/old/new | 见上述action/history 通道依据 | Main 原生扩展 7ba6632c；本项发布验证未完成 |
| `internal/module/agileteam/repo_basic_atomic.go` / `applyTopologyChange` | `zt_teamgroup` | name/slogan/declaration/logo；parent/type/grade/path | 见上述小组依据 | Main 原生扩展 7ba6632c；本项发布验证未完成 |
| `internal/module/agileteam/repo_basic_atomic.go` / `refreshDescendantsIterative` | `zt_teamgroup` | name/slogan/declaration/logo；parent/type/grade/path | 见上述小组依据 | Main 原生扩展 7ba6632c；本项发布验证未完成 |
| `internal/module/agileteam/repo_basic_atomic.go` / `updateTeamgroupFields` | `zt_teamgroup` | name/slogan/declaration/logo；parent/type/grade/path | 见上述小组依据 | Main 原生扩展 7ba6632c；本项发布验证未完成 |
| `internal/module/profile/repo.go` / `SaveSelfProfile` | `zt_user` | 资料、mainTeam、password、locked、deleted；具体字段受对应 Req 限制 | 见上述个人资料依据 | Main 原生扩展 7ba6632c；本项发布验证未完成 |
| `internal/module/profile/repo.go` / `UpdatePassword` | `zt_user` | 资料、mainTeam、password、locked、deleted；具体字段受对应 Req 限制 | 见上述个人资料依据 | Main 原生扩展 7ba6632c；本项发布验证未完成 |
| `internal/module/user/repo.go` / `Delete` | `zt_user` | 资料、mainTeam、password、locked、deleted；具体字段受对应 Req 限制 | 见上述用户管理依据 | Main 原生扩展 7ba6632c；本项发布验证未完成 |
| `internal/module/user/repo.go` / `Update` | `zt_user` | 资料、mainTeam、password、locked、deleted；具体字段受对应 Req 限制 | 见上述用户管理依据 | Main 原生扩展 7ba6632c；本项发布验证未完成 |
| `internal/module/user/repo.go` / `UpdatePassword` | `zt_user` | 资料、mainTeam、password、locked、deleted；具体字段受对应 Req 限制 | 见上述用户管理依据 | Main 原生扩展 7ba6632c；本项发布验证未完成 |
| `internal/module/user/repo.go` / `UpdateStatus` | `zt_user` | 资料、mainTeam、password、locked、deleted；具体字段受对应 Req 限制 | 见上述用户管理依据 | Main 原生扩展 7ba6632c；本项发布验证未完成 |
| `internal/module/user/repo.go` / `createWithTx` | `zt_user` | 资料、mainTeam、password、locked、deleted；具体字段受对应 Req 限制 | 见上述用户管理依据 | Main 原生扩展 7ba6632c；本项发布验证未完成 |

## 质量门边界

门禁采用明确自有表清单，不再依据 model 目录或 `zt_wb_*` 前缀自动豁免。原生直写按文件、函数和表逐项登记，失效条目必须删除；原先三项“有接口仍直写”待迁移已移除。增加了模型目录、动态小组表和未登记自有前缀的反例测试。

这是静态质量检查与审查登记，不等同于部署版本或数据库并发验收。源码门禁通过不能解除上面任何发布阻止项。

## 已取得的接口与并发证据（2026-10-03）

- 原生 `POST /productplans/:id/linkstories` 在上述未修改 PHP 的隔离版本返回 HTTP 400，未创建关联；`unlinkstories` 返回 HTTP 200。存在接口不等于完整动作可用，摘除已迁移到既有 Client 的原生调用，位于本地事务外并保留部分成功语义，不可按整个关联模型豁免。证据位于隔离目录 `native-plan-link-contract.json`。
- 原生小组成员 `(root,type,account)` 唯一索引与工作台 upsert 已有有界双系统竞争证据；原生小组拓扑更新为逐条 autocommit，没有共同的树锁协议。因此现有普通保存最小行锁已验收，拓扑整树锁不能据此缩小，也不能把本项标为跨系统安全。
- 6 轮工作台排期与原生交付并发，全部 12 个请求成功，交付与排期字段均保留，每个需求仅一条窗口关联。窗口删除先提交的更新请求返回 success=false，产品关联为 0。
- 以上是明确场景证据，28 项仍逐项受发布阻止条件约束；用户管理、通知与关注等未做完整原生能力及并发验证，不得因整体门禁通过而解除。

摘除计划已迁移至原生接口，两个调用方均在本地事务之前执行；失败停止后续动作，不重试、不补写原生 action，删除对应直写豁免。
