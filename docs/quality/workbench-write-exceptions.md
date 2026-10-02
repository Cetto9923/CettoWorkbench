# 禅道直写逐项登记与发布前置条件

源码版本：Dev-CT `7d45c28a`，对照 Main `0a892260`。部署的禅道版本尚未取得；只读索引连接失败。以下登记不证明原生接口缺失，也不构成上线授权。不得以 Client 未实现某方法推断禅道没有接口。

已替换的原生动作：计划创建、业需转研需、研需指派、任务创建/编辑/删除、研需删除、业需验收与交付。共享 Client 对写动作的 `result=fail` / `status=fail` 识别为失败；读取审批提示保留 Main 的响应含义。HTTP 调用均放在本地事务之外，不自动重放创建或删除。

所有下列例外当前均为**阻止上线，待验证**。逐项补齐部署版本、原生接口目录与响应契约、字段支持、权限、禅道索引及共同并发协议后，才能决定迁移至接口或批准保留。若原生接口可满足，则必须删除例外及对应直写。

## 共用的权限与并发依据

- 排期：Service 先校验根对象写权限、研需/任务归属和产品/执行范围。窗口关联本地事务锁窗口；独立研需的记录、描述及历史在同一短事务完成。原生计划创建后窗口可能失效、并发首次建计划可能产生重复计划，仍须隔离验证。
- 小组：普通信息保存只锁目标行，观察到父级变化则失败；拓扑暂保留旧全树一致性协议。成员确认先锁所属小组，再处理调整单，账号固定顺序。该锁只能证明工作台内部排序，尚不能证明禅道也遵守，不能声称跨系统竞争已解决。
- 个人资料：仅当前账号与 ID 的 Service 校验；资料和工作台偏好一次事务提交，失败回滚。密码哈希与会话行为沿用既有实现；双系统密码与会话兼容仍须真实验证。
- 用户管理：既有管理路由权限和 Service actor 校验；本轮未改业务行为。原生账号 CRUD 与并发协议尚未验证。
- PO 关注/催办：沿用对应 Service 的对象权限和既有 SQL；关注补缺行、通知、审计需验证原生能力及重复写并发。
- action/history 通道：仅允许由已通过授权的业务调用方使用；同一事务维护本地直写的历史。不能为已经由禅道接口处理的动作再补写历史。

## 逐项清单

| 文件 / 函数 | 原生表 | 字段 / 动作 | 权限和并发依据 | 版本与验收 |
|---|---|---|---|---|
| `internal/module/agileteam/repo_confirm.go` / `ApplyConfirmedAdjustment` | `zt_team` | root/type/teamgroup/account/role/join/days/hours/estimate/consumed/left/order；成员删除 | 见上述小组依据 | 部署版本未知；阻止上线 |
| `internal/module/agileteam/repo_confirm.go` / `upsertTeamMemberTx` | `zt_team` | root/type/teamgroup/account/role/join/days/hours/estimate/consumed/left/order；成员删除 | 见上述小组依据 | 部署版本未知；阻止上线 |
| `internal/module/po/repo_home_actions.go` / `insertHomeDemandAction` | `zt_action` | objectType/objectID/product/project/execution/actor/action/date/comment/extra | 见上述PO 关注/催办依据 | 部署版本未知；阻止上线 |
| `internal/module/po/repo_home_actions.go` / `insertAcceptanceUrge` | `zt_action` | objectType/objectID/product/project/execution/actor/action/date/comment/extra | 见上述PO 关注/催办依据 | 部署版本未知；阻止上线 |
| `internal/module/po/repofollow.go` / `EnsureDemandUnfollowed` | `zt_starinfo` | objectType/objectID/account/followed | 见上述PO 关注/催办依据 | 部署版本未知；阻止上线 |
| `internal/module/po/repofollow.go` / `RemoveProjectReportFollow` | `zt_project` | follow（项目周报关注用户 ID CSV） | 见上述PO 关注/催办依据 | 部署版本未知；阻止上线 |
| `internal/module/po/repo_home_actions.go` / `insertAcceptanceUrge` | `zt_notify` | 催办消息关联对象、收件人与消息内容 | 见上述PO 关注/催办依据 | 部署版本未知；阻止上线 |
| `internal/module/schedule/repo_scheduling_write.go` / `CreateAction` | `zt_action` | objectType/objectID/product/project/execution/actor/action/date/comment/extra | 见上述排期依据 | 部署版本未知；阻止上线 |
| `internal/module/schedule/repo_history.go` / `LogEditedRecord` | `zt_action` | objectType/objectID/product/project/execution/actor/action/date/comment/extra | 见上述排期依据 | 部署版本未知；阻止上线 |
| `internal/module/schedule/repo_history.go` / `LogEditedRecord` | `zt_history` | action/field/old/new | 见上述排期依据 | 部署版本未知；阻止上线 |
| `internal/module/schedule/repo_scheduling_write.go` / `CreateStory` | `zt_story` | title/product/assignedTo/estimate/排期日期/来源字段；独立研需创建 | 见上述排期依据 | 部署版本未知；阻止上线 |
| `internal/module/schedule/repo_scheduling_write.go` / `UpdateStory` | `zt_story` | title/product/assignedTo/estimate/排期日期/来源字段；独立研需创建 | 见上述排期依据 | 部署版本未知；阻止上线 |
| `internal/module/schedule/repo_scheduling_write.go` / `CreateStorySpec` | `zt_storyspec` | story/version/title/spec | 见上述排期依据 | 部署版本未知；阻止上线 |
| `internal/module/schedule/repo_scheduling_write.go` / `UpdateDemandScheduling` | `zt_demand` | QD/RD/estimateLaunch/developFinish/testFinish/verifyFinish/assignedTo/assignedDate | 见上述排期依据 | 部署版本未知；阻止上线 |
| `internal/module/schedule/repo_story_link.go` / `EnsurePlanStoryRelation` | `zt_planstory` | plan/story | 见上述排期依据 | 部署版本未知；阻止上线 |
| `internal/module/schedule/repo_story_link.go` / `UnlinkStoryFromPlan` | `zt_planstory` | plan/story | 见上述排期依据 | 部署版本未知；阻止上线 |
| `internal/module/schedule/repo_story_link.go` / `ReplaceProjectStory` | `zt_projectstory` | project/story/product/branch/version/order | 见上述排期依据 | 部署版本未知；阻止上线 |
| `internal/pkg/ztaction/action.go` / `Create` | `zt_action` | objectType/objectID/product/project/execution/actor/action/date/comment/extra | 见上述action/history 通道依据 | 部署版本未知；阻止上线 |
| `internal/pkg/ztaction/action.go` / `LogHistory` | `zt_history` | action/field/old/new | 见上述action/history 通道依据 | 部署版本未知；阻止上线 |
| `internal/module/agileteam/repo_basic_atomic.go` / `applyTopologyChange` | `zt_teamgroup` | name/slogan/declaration/logo；parent/type/grade/path | 见上述小组依据 | 部署版本未知；阻止上线 |
| `internal/module/agileteam/repo_basic_atomic.go` / `refreshDescendantsIterative` | `zt_teamgroup` | name/slogan/declaration/logo；parent/type/grade/path | 见上述小组依据 | 部署版本未知；阻止上线 |
| `internal/module/agileteam/repo_basic_atomic.go` / `updateTeamgroupFields` | `zt_teamgroup` | name/slogan/declaration/logo；parent/type/grade/path | 见上述小组依据 | 部署版本未知；阻止上线 |
| `internal/module/profile/repo.go` / `SaveSelfProfile` | `zt_user` | 资料、mainTeam、password、locked、deleted；具体字段受对应 Req 限制 | 见上述个人资料依据 | 部署版本未知；阻止上线 |
| `internal/module/profile/repo.go` / `UpdatePassword` | `zt_user` | 资料、mainTeam、password、locked、deleted；具体字段受对应 Req 限制 | 见上述个人资料依据 | 部署版本未知；阻止上线 |
| `internal/module/user/repo.go` / `Delete` | `zt_user` | 资料、mainTeam、password、locked、deleted；具体字段受对应 Req 限制 | 见上述用户管理依据 | 部署版本未知；阻止上线 |
| `internal/module/user/repo.go` / `Update` | `zt_user` | 资料、mainTeam、password、locked、deleted；具体字段受对应 Req 限制 | 见上述用户管理依据 | 部署版本未知；阻止上线 |
| `internal/module/user/repo.go` / `UpdatePassword` | `zt_user` | 资料、mainTeam、password、locked、deleted；具体字段受对应 Req 限制 | 见上述用户管理依据 | 部署版本未知；阻止上线 |
| `internal/module/user/repo.go` / `UpdateStatus` | `zt_user` | 资料、mainTeam、password、locked、deleted；具体字段受对应 Req 限制 | 见上述用户管理依据 | 部署版本未知；阻止上线 |
| `internal/module/user/repo.go` / `createWithTx` | `zt_user` | 资料、mainTeam、password、locked、deleted；具体字段受对应 Req 限制 | 见上述用户管理依据 | 部署版本未知；阻止上线 |

## 质量门边界

门禁采用明确自有表清单，不再依据 model 目录或 `zt_wb_*` 前缀自动豁免。原生直写按文件、函数和表逐项登记，失效条目必须删除；原先三项“有接口仍直写”待迁移已移除。增加了模型目录、动态小组表和未登记自有前缀的反例测试。

这是静态质量检查与审查登记，不等同于部署版本或数据库并发验收。源码门禁通过不能解除上面任何发布阻止项。
