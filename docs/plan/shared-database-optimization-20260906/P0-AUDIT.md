# P0 只读核验：工作台 × 禅道共享数据库

日期：2026-09-06  
状态：**partial**（只读盘点完成；生产拓扑未验证；未实施）  
审计人：接手 Cursor 会话  
授权：只读核验；可新增本目录文档；不改业务代码/配置/规则/数据库；不轮换凭据；不 DDL/DML/GRANT；不建分支、不提交。

## 1. 当前基线与范围

| 项 | 接手时核验 |
|---|---|
| 工作台仓库 | `/Users/yuyan9923/GitHub/workbench-claude-po` |
| 分支 | `Claude-PO`（与设计基线一致） |
| HEAD | `8f1003b8048031b9187615b22c6eafc3a8830d86`（与设计基线一致） |
| 工作区 | 仅未跟踪 `docs/plan/shared-database-optimization-20260906/`；无已暂存业务改动 |
| PLAN | Codex 仅新增 PLAN，未改业务代码 |
| 禅道源码 | 用户指定 `/Users/yuyan9923/GitHub/csrcb20-gitfox/zentao`。另有完整 max5.6.1 树 `/Users/yuyan9923/GitHub/ZentaoPMS`（含 `config/routes.php`）。`/Users/yuyan9923/GitHub/csrcb20` 是设计/原型仓，没有 PMS PHP |
| 禅道源码 HEAD | `csrcb20-gitfox` `master` / `e1e443afe90f696ccb9a3588939b295f64acbd75` |
| 禅道基础包 VERSION | `ZentaoPMS/VERSION` 与 `禅道部署基础包/zentaopms/VERSION` = `max5.6.1`。`csrcb20-gitfox/zentao/VERSION` 不存在 |
| 本轮未改 | 业务代码、运行配置、工程规则、数据库、凭据、分支 |

范围：全仓生产写入口（`internal/`、`cmd/`、`db/`），不含 `tests/integration` 种子/DDL、`*_test.go` mock。用户原诊断与网页版 GPT 结论只作待核验材料。

## 2. 配置加载链（不能证明生产）

### 2.1 代码链

```text
cmd/server/main.go:18
  → bootstrap.Run()
    → config.Load()                         // loader.go:22
         WORKBENCH_MODE=="dev" → configs/config.dev.yaml   // gitignore
         否则 → configs/config.yaml
         Viper prefix workbench；AutomaticEnv；键名 `.` → `_`
    → database.New(cfg)                     // 主库；DSN 含 sessionVariables
    → db.AutoMigrate(&model.OperationLog{}) // bootstrap.go:77  运行期 DDL
    → 若 databaseReadonly.host 非空：database.Open(cfg.DatabaseReadonly)
         失败只打 Warn，价值流降级
    → po.NewRepo(dbReadonly)                // bootstrap.go:133  PO 读写都走只读连接
    → schedule.NewRepo(db)                  // 主库
    → RecordOperationLog(db, ...)           // 首次写请求再 AutoMigrate  // operationlog.go:56
```

有效运行配置 = 文件 + `WORKBENCH_*` 环境覆盖 + 启动时实际连上的实例。Git 中的 YAML **不能**证明生产账号、隔离级别或主备拓扑。

### 2.2 已验证 / 未验证

| 对象 | 结论 | 证据 |
|---|---|---|
| 跟踪的 `configs/config.yaml` | 含非占位凭据（债务，P1b 处理）。本文件不复制密码/DSN。`app.env=prod`；主库与只读库键均指向本机 `127.0.0.1:3306`；注释掉的 OceanBase 主机不能当生产拓扑。`zentao.api` 指向公网禅道云，不是行内实例。 | 文件存在；本轮用其凭据连本机 3306 → **Access denied** |
| `configs/config.dev.yaml` | 存在且 gitignore。`WORKBENCH_MODE=dev` 时这才是本机有效文件。主库与只读库 **同一 endpoint**（本机 `127.0.0.1:13306`），`sessionVariables` 空。 | 文件存在；探测成功 |
| 本机 `mysqld :3306` | 在监听。跟踪配置的账号连不上。 | lsof；Access denied |
| SSH 隧道 `127.0.0.1:13306` | 在监听，对应手册中的 PD VM `vm-zentao`。 | lsof；`ps` 可见 ssh 隧道（不记录密钥） |
| 经隧道连上的实例 | **开发/验收 VM，不是生产。** MySQL `8.0.46` Ubuntu；库名 `zentaopms`；会话/全局隔离 **REPEATABLE-READ**；`binlog_format=ROW`、`binlog_row_image=FULL`、`log_bin=1`；`read_only=0`；当前用户 `zentao@127.0.0.1`；`ALL PRIVILEGES` 全局、无 `GRANT OPTION`。`SHOW MASTER/REPLICA STATUS` 无权限或无行，**不能**据此断言有/无备库。 | 只读 `SELECT @@...` / `SHOW GRANTS` / `information_schema` |
| 生产引擎、账号、主备、binlog、备份 | **未验证** | 无生产连接授权 |

`internal/pkg/zentao` 只拼页面 URL（`zentao.go`），不是业务写 Client。配置里的 `zentao.api` / account / password **没有任何 Go 调用方**发出 HTTP 写请求。

## 3. 表产权与写入口矩阵

产权规则：不凭 `zt_` 前缀猜测。依据 = `db/install.sql`、禅道源码/`zcsrcb.sql`、GORM `TableName`、本轮 VM 表是否存在。

| 符号 | 含义 |
|---|---|
| WB | Workbench 自有（install.sql 或明确工作台表） |
| ZT | 禅道原生或行内禅道定制（禅道源码/二次开发 SQL 为真源） |
| MIX | 两边都会写 |
| UNVERIFIED | 证据不足 |

### 3.1 运行期 DDL

| 动作 | 入口 | 文件:行 | 表 | DML | 产权 | 授权 | 事务 | 其他调用者 | 迁移依赖 |
|---|---|---|---|---|---|---|---|---|---|
| 启动保证审计表 | `bootstrap.Run` | `internal/bootstrap/bootstrap.go:77` | `zt_operation_logs` | DDL（GORM AutoMigrate） | WB | 无业务授权；启动即执行 | 无 | 中间件兜底 | **不在** `db/install.sql`。VM 上表已存在且列与 Model 一致 |
| 首次写请求兜底建表 | `RecordOperationLog` | `internal/middleware/operationlog.go:54–56` | `zt_operation_logs` | DDL | WB | 无 | 无 | bootstrap 已迁仍会再调一次（sync.Once） | 同上 |
| 登录列探测 | `login.Repo.hasColumn` | `internal/module/login/repo.go:102` | `zt_login_failures` / `zt_login_logs` | 非 DML（`Migrator().HasColumn`） | WB | 无 | 无 | RecordFailure / InsertLoginLog | install.sql 有这两表 |

### 3.2 中间件 / 登录（绕过部分 Handler→Service 形态）

| 业务动作 | 路由 | Handler | Service | 写函数 | 表 | DML | 产权 | 授权 | 事务 |
|---|---|---|---|---|---|---|---|---|---|
| 操作审计 | 所有 POST/PUT/DELETE（admin 与 PO group） | `RecordOperationLog` `operationlog.go:36`；路由 `routes.go:60,85` | 无 | `db.Create` `operationlog.go:88–90`（**响应后 goroutine**，失败吞掉） | `zt_operation_logs` | INSERT | WB | 无对象授权；登录后写 | 无；且在业务提交之后 |
| 登录失败 | `POST /login` | `login/handler.go:58` | `Login` `login/service.go:68` → `failLogin:128` | `RecordFailure` `login/repo.go:49` | `zt_login_failures` | INSERT | WB | 公开路由 | 无 |
| 登录日志 | 同上 | 同上 | `recordLoginLog:136` | `InsertLoginLog` `login/repo.go:68` | `zt_login_logs` | INSERT | WB | 公开 | 无 |
| 清理旧失败记录 | **无路由** | — | **无调用方** | `CleanOldFailures` `login/repo.go:60` | `zt_login_failures` | DELETE | WB | 死代码 | 无 |
| 登出 | `POST /logout` | `login/handler.go:66` | `Logout:117` | 仅销毁 session | — | — | — | 需登录 | 无 |

登录**不写** `zt_user.last` / `ip`。密码校验用 `encode.MD5`（`login/service.go:87`），与禅道 `zt_user.password` 兼容；这是既有债务，本切片不改。

### 3.3 用户 / 角色 / 菜单 / 部门（admin）

路由组：`/admin` + `RequireLogin` + `RecordOperationLog`（`routes.go:58–79`）。对象级：能力权限；user/role/menu/dept Service **不**做对象所有权检查（`_ = actor` 常见）。`user` 模块为 `legacy/reference-not-authoritative`。

| 业务动作 | 路由 | Handler | Service | Repo | 表 | DML | 产权 | 事务 |
|---|---|---|---|---|---|---|---|---|
| 创建用户 | `POST /admin/users` | `user/handler.go:60,183` | `Create` `user/service.go:78`（MD5 写 password） | `Create` `user/repo.go:152–170` | `zt_user` | INSERT | **ZT**（禅道用户表） | 无；随后 `ReplaceUserRoles` 另开事务 |
| 更新用户 | `POST /admin/users/:id` | `:63,363` | `Update:99` | `Update:190` | `zt_user` | UPDATE | ZT | 无 |
| 启停 | `POST .../toggle-status` | `:64` | `UpdateStatus` | `UpdateStatus:219` | `zt_user` | UPDATE | ZT | 无 |
| 重置密码 | `POST .../reset-password` | `:65` | `UpdatePassword` | `UpdatePassword:236` | `zt_user` | UPDATE | ZT | 无 |
| 删除用户 | `DELETE /admin/users/:id` | `:67,382` | `Delete:111` | `Delete:208`（`deleted='1'`） | `zt_user` | UPDATE | ZT | 无 |
| 分配角色 | 创建/更新后 | — | `ReplaceUserRoles` | `ReplaceUserRoles:284` | `zt_gf_user_roles` | DELETE+INSERT | WB | 单事务 |
| 批量创建 | `POST /admin/users/batch/create` | `:62` | `BatchCreate` | `BatchCreate:308` | `zt_user` + `zt_gf_user_roles` | INSERT | ZT+WB | 单事务 |
| 角色 CRUD / 赋权 | `/admin/roles` | `role/handler.go:56–62` | `role/service.go:81,114,134` | `Create:74` `Update:78` `Delete:92` `ReplacePermissions:162` | `zt_roles`, `zt_role_permissions` | INSERT/UPDATE/DELETE | WB | Delete 与 ReplacePermissions 各一事务 |
| 菜单 CRUD | `/admin/menus` | `menu/handler.go:74–78` | `menu/service.go:80,104,120` | `Create:66` `Update:70` `Delete:87` | `zt_menus` | INSERT/UPDATE/软删 | WB | 无 |
| 部门 CRUD / 状态 | `/admin/depts` | `dept/handler.go:87–92` | `Create/Update/Delete` `dept/service.go` | `repo.go:72–148` | `zt_depts` | INSERT/UPDATE | **UNVERIFIED**（Workbench 风格 Model，**不在** `install.sql`；禅道读路径用 `zt_dept`。VM 上两表都在） | 祖先启用时 Service 直接 `s.repo.db.Transaction`（`dept/service.go:80`，层次越界，既有事实） |

`zt_permissions`：仅有 Model（`permission.go:21`），**无生产写路径**。VM 上表存在。产权 UNVERIFIED（工作台 Model + 无 install.sql）。

### 3.4 PO 工作台写（走 databaseReadonly 连接）

`bootstrap.go:133`：`poRepo := po.NewRepo(dbReadonly)`。本机 dev 主/只读同一实例，写入能成功。若生产只读库是副本，关注/已读会写到副本或失败。**生产只读拓扑未验证。**

| 业务动作 | 路由 | Handler | Service | Repo | 表 | DML | 产权 | 授权 | 事务 |
|---|---|---|---|---|---|---|---|---|---|
| 关注业需 | `PUT /follow/demand/:id` | `po/handler.go:63,368` | `FollowSetDemand` `servicefollow.go:48` | `SaveDemandFollow` `repofollow.go:213–229` | `zt_starinfo` | UPDATE 或 INSERT | **MIX**：禅道 `zcsrcb.sql` + `changshu.php` 也写；工作台并行 upsert | `RequirePerm(PoFollowUpdate)`；无对象级 | 无 |
| 通知已读 | `PUT /notice/:id/read` | `handler.go:61` | `NoticeMarkRead` `servicenotice.go:43`（先 `CheckNoticeAccess`） | `SaveNoticeRead` `reponotice.go:319` | `zt_workbench_notify_reads` | INSERT SELECT | WB（**不在** `install.sql`；VM 表存在） | 能力 + toList 成员 | 无 |
| 全部已读 | `PUT /notice/read-all` | `handler.go:62` | `NoticeMarkAllRead:61` | `SaveAllNoticeReads:328` | 同上 | INSERT SELECT | WB | 登录 | 无 |

### 3.5 排期写（主库，多表混合事务）

路由前缀 `/schedule`（`handler.go:102–120`），组中间件登录 + 操作日志。

| 业务动作 | 路由 | Handler | Service | 主要写 | 表 | DML | 产权 |
|---|---|---|---|---|---|---|---|
| 业需排期确认并同步 | `POST /schedule/demands/:id/save-scheduling` | `handler_demand.go:463` | `SaveScheduling` `service_scheduling_save.go:34`；事务 **:76** | 见 §4 | `zt_story` `zt_storyspec` `zt_action` `zt_planstory` `zt_projectstory` `zt_task` `zt_taskspec` `zt_productplan` `zt_demand` `zt_versionwindowproduct` `zt_demandwindow` | INSERT/UPDATE/DELETE/UPSERT | ZT + WB 同一事务 |
| 独立研需排期 | `POST /schedule/stories/:id/save-scheduling` | `handler_demand.go:530` | `SaveStoryScheduling` **:101**；事务 **:131** | 见 §4 | 上表除 `zt_demand`/`zt_demandwindow` | 同上 | ZT + WB |
| 维护任务弹窗保存 | `POST /schedule/stories/:id/save-tasks` | `handler_story_tasks.go:110` | `SaveStoryTasks` `service_story_tasks.go:230`；事务 **:259** | `applySingleSchedulingTask` | `zt_task` `zt_taskspec` `zt_action` `zt_projectstory` | INSERT/UPDATE | ZT |
| 创建版本窗口 | `POST /schedule/windows` | `handler.go:241` | `Create` `service.go:313`；事务 **:331** | `Create` + `saveWindowProducts` | `zt_versionwindow` `zt_versionwindowproduct` `zt_productplan` | INSERT | WB + 可能 ZT |
| 更新版本窗口 | `PUT /schedule/windows/:id` | `handler.go:306` | `Update` `service.go:340`；事务 **:366** | Update + 物理删产品行 + 重建 | `zt_versionwindow` `zt_versionwindowproduct` `zt_productplan` | UPDATE/DELETE/INSERT | WB + 可能 ZT |
| 删除版本窗口 | `DELETE /schedule/windows/:id` | `handler.go:347` | `Delete` `service.go:378` | `repo.Delete:358` GORM 软删 | `zt_versionwindow` | UPDATE `deletedAt` | WB；**无 Transaction**；**不**级联删 `zt_versionwindowproduct`；TODO：已关联需求仍可删（`:390`） |

排期授权：

- 路由：`RequirePerm(ScheduleUpdate/Create/Delete)`  
- 窗口 Update/Delete：Service 校验 `CreatedBy == account`（`service.go:349,386`）  
- 排期保存：事务前 `loadDemandStoryScopeAndWindow` + `validateSaveSchedulingStoryScope`（伪造 story ID → 403）+ `precheckDemandSchedulingProducts` / `precheckSchedulingProducts`（产品访问）  
- 产品预校验走 `validateProductsAccess` → **仅** `GetUserProducts`（`service_scheduling_precheck.go:132–138`），**不**走 `UserCanAccessProduct` 的 `IsAdmin` 短路（`repo_scheduling_write.go:211–224`）。窗口列表的小组查询才用 `IsAdmin`（`service.go:43`）。  
- 任务：事务内 `SELECT ... FOR UPDATE` + `UpdateTaskForStory` 带 `id AND story AND deleted='0'`（`task_authorization_repo.go:16–88`）  
- **不是** CAS：`zt_story.version` 只用于 spec/项目关联版本，更新 story **不**比较 version。  

生产写函数中无调用方（死代码，P1 不删）：`CleanOldFailures`；`CreatePlanStory`（已被 `EnsurePlanStoryRelation` 替代）；`UpdateTask` / `CloseTask`（生产用 `*ForStory`）。

### 3.6 只读引用（本轮无生产写）

`zt_product` `zt_project` `zt_holiday` `zt_teamgroup` `zt_notify` `zt_dept`（禅道部门，区别于 `zt_depts`）`zt_demandclarify` 等：排期/PO 查询用。`loginlog`/`operationlog` 模块 Handler 只读列表。

### 3.7 VM 表存在性 vs install.sql 漂移

经隧道的 `zentaopms`：盘点表均存在。与仓库 DDL 不一致处：

| 表 | 漂移 |
|---|---|
| `zt_operation_logs` | **install.sql 无此表**；靠 AutoMigrate。VM 列与 `OperationLog` 一致（含 `tenantId`、`createdAt` datetime(3)） |
| `zt_demandwindow` | install.sql / Go Model **无** `plan`、`product`；VM **有** 这两列。工作台 `SaveDemandLevelWindow` 不写它们 |
| `zt_depts` | **install.sql 无此表**。VM 同时有 `deletedAt` 与 `deleted`；Go Model 只有 `deletedAt` |
| `zt_workbench_notify_reads` | **install.sql 无此表**；VM 存在 |
| `zt_dept` vs `zt_depts` | 两张都在。禅道用前者；工作台部门 CRUD 用后者 |

## 4. 排期完整事务

公共事务入口：`Repo.Transaction` `repo.go:234` → `db.Transaction`。单连接、单事务。禅道表与工作台表 **不得**先拆到两个连接池。

### 4.1 SaveScheduling（业需）

```text
Handler SaveScheduling (handler_demand.go:463)
  绑定 JSON、Validate
  Service.SaveScheduling (service_scheduling_save.go:34)
    [事务外]
      loadDemandStoryScopeAndWindow          // 读 demand/story/window 映射
      终排窗口不可改
      validateSaveSchedulingStoryScope       // edit/delete 必须属于该业需
      GetDemandMainSystem                    // zt_demand.mainSystem
      precheckDemandSchedulingProducts       // 产品访问；可能返回提示、零写入
    [TOCTOU：校验与 Begin 之间无锁]
    repo.Transaction (:76)
      for each story:
        applySchedulingStory                 // 见下；new 依赖 CreateStory 自增 ID
        unless delete: applySchedulingTasks  // 依赖 storyID
      UpdateDemandScheduling                 // zt_demand RD/QD/accepter/日期
      SaveDemandLevelWindow                  // Unscoped DELETE + INSERT zt_demandwindow
    提交
  中间件事后异步写 zt_operation_logs
```

`applySchedulingStory`（`:172`）：

| action | 顺序 | 表 | 依赖上一 ID |
|---|---|---|---|
| new | resolvePlanForProduct → CreateStory → CreateStorySpec → CreateAction(Opened) → LinkStoryToPlan | plan / `zt_story` / `zt_storyspec` / `zt_action` / `zt_planstory` | story 自增 ID → spec/action/plan |
| edit | UpdateStory → CreateAction(Edited) → resolvePlan → RemoveStoryFromOtherPlans → LinkStoryToPlan | `zt_story` `zt_action` `zt_planstory` | 否 |
| delete | CloseStory → CreateAction(Closed) | `zt_story` `zt_action` | 否 |

`applySingleSchedulingTask`（`:291`）：

| action | 顺序 | 表 | 依赖 |
|---|---|---|---|
| new | GetProjectIDByExecution（读 `zt_project` 父链）→ CreateTask → CreateTaskSpec → CreateAction(Opened) → LinkStoryToProjectAndExecution | `zt_task` `zt_taskspec` `zt_action` `zt_projectstory` | task 自增 ID；projectID 来自 execution |
| edit | 推导 project → UpdateTaskForStory → CreateAction(Edited) → LinkStoryToProjectAndExecution | `zt_task` `zt_action` `zt_projectstory` | 否 |
| delete | CloseTaskForStory → CreateAction(Closed) | `zt_task` `zt_action` | 否 |

`resolvePlanForProduct`（`:376`）：读 `zt_versionwindowproduct`；无 plan 则读窗口 → `CreateProductPlan`（`zt_productplan`，自增 plan ID）→ `UpdateWindowProductPlanID` 或 `CreateWindowProduct`。计划创建 **不**写禅道 `zt_action`。

关联：

- `LinkStoryToPlan`：`INSERT IGNORE zt_planstory`；仅首次写 `linked2plan` / `linkstory` action（`repo_story_link.go:199`）  
- `UnlinkStoryFromPlan`：`DELETE zt_planstory` + unlinked actions（`:247`）  
- `ReplaceProjectStory`：`INSERT ... ON DUPLICATE KEY UPDATE`（`:78`）。注释：行内为 `zt_projectstory` 加了自增 PK，不能用 REPLACE  

**并发：** 预校验在事务外。任务路径有 `FOR UPDATE`（story/task 行锁）。Story/demand/窗口更新只有 `deleted='0'` / `id=?`，**没有** version CAS。幂等关联 ≠ 防两人互覆盖。InnoDB 锁仍在；绕过的是禅道业务层。无死锁图，**不宣称已发生死锁。**

### 4.2 SaveStoryScheduling（独立研需）

事务 `:131`：子节点循环（常空，`demandID=0`）→ `resolvePlanForProduct` → `RemoveStoryFromOtherPlans` → `LinkStoryToPlan` → `CreateAction(Edited)` → `UpdateStory` 日期字段（`developFinish`/`testFinish`/`verifyFinish`）。**不写** `zt_demand` / `zt_demandwindow`。

### 4.3 SaveStoryTasks

事务外：读 story 详情 + 产品 precheck（`windowID=0`）。事务内：`ValidateStoryForTaskMutation` + 逐任务 ownership + `applySingleSchedulingTask`。不写窗口/demand。

### 4.4 窗口创建 / 更新

Create：事务外产品访问校验 → 事务内 `Create(window)` 拿自增 window ID → `saveWindowProducts`（匹配已有 plan 或 `SyncPlan` 时 `CreateProductPlan`）→ `CreateWindowProduct`。

Update：所有权校验 → 事务内 Update 窗口 → `Unscoped` 物理删该窗口全部 `zt_versionwindowproduct` → 再 insert。会丢掉旧 plan 映射后重建。

Delete：无事务；只软删 `zt_versionwindow`。

## 5. 禅道 API 能力

### 5.1 源码分层（必须区分）

| 树 | 是什么 | 与排期相关的 API 文件 |
|---|---|---|
| `/Users/yuyan9923/GitHub/ZentaoPMS` | 完整 max5.6.1；`VERSION` 已核 | `config/routes.php` 有 `POST /products/:id/stories`、`/executions/:id/tasks`、`/products/:id/plans`、`/productplans/:id/linkstories`。**无** project/execution 关联写路由（`/projects/:id/stories` 仅为 GET）。无 `/demands` 写路由 |
| `csrcb20-gitfox/zentao` | 行内定制树（用户指定） | `api/v1/entries/` **没有** `stories.php` / `tasks.php` / `productplan.php`。有 `story.php` GET/PUT/DELETE、`storyclose.php` POST、大量 demand/OA。`config/ext/routes.php` 只加 demand 等。`db/add_id_column.sql` 给 `zt_projectstory`/`zt_planstory` 加 `PKID`（与工作台不用 REPLACE 的注释一致） |
| `禅道部署基础包/zentaopms` | max5.6.1 基础包 | 与 ZentaoPMS 同类 REST entry 文件 |
| 实际部署 | **未验证** | 若按「基础包 + overlay」发布，基础 REST 可能仍在；若只发布 `zentao/` 树，则没有标准 stories/tasks/plans POST |

`internal/pkg/zentao` 不能当 API Client。

### 5.2 不能直接承接排期的原因（已证明缺口，不是「没有 POST」误判）

1. **完整业务动作是「业需转研需 + 计划/项目关联 + 任务 + 业需字段 + 窗口」**，不是单表 CRUD。禅道真源动作是 `demand/tostory` → `batchCreateStory`（`extension/custom/demand/ext/control/tostory.php:35`；`changshu.php` 约 1844 行写 `fromDemand`、`sourceType=demandpool`、`developFinish`/`testFinish`/`verifyFinish`、`isMainSystemAssociation`，并 `action->create('demand', ..., 'tostory')`）。工作台直写 **绕过** 该动作。  
2. 基础包 `stories.php` POST 字段（`title,spec,verify,...`）**不含** `fromDemand` / `isMainSystemAssociation` / `sourceType=demandpool` / 三个 Finish 日期；默认 status 还是 `draft`，工作台写的是 `active`+`stage=planned`。  
3. 定制 `story.php` PUT 字段列表同样不含上述定制列。  
4. 定制 `demand.php` **只有 GET**，没有 PUT 可写 RD/QD/accepter/日期。`demandcreate.php` 是 OA 创建/删除，不是排期。  
5. 基础包 task/plan API 调 `task/create`、`productplan/create`、`linkStory`，会走禅道通知/日志/校验；工作台只写部分列 + 少量 `zt_action`，**副作用覆盖未证明等价**。  
6. **`zt_projectstory` 项目/执行关联没有 REST 写路由**（`ZentaoPMS/config/routes.php` 仅 GET list）。工作台 `LinkStoryToProjectAndExecution` 目前无现成 API 对位。  
7. 身份：API 用禅道 token/账号；工作台用 session。映射方案未定（PLAN §6）。  
8. HTTP 不能放进 GORM 事务；现有混合事务一旦改 API 必须先有本地 operation 记录。

**结论：尚未证明可复用现有 API 直接承接排期。P2 前必须先确认部署树是否含基础包 REST，再按完整动作对照 tostory/create/edit，而不是逐表替换。**

### 5.3 工作台直写相对禅道 tostory 的已知缺口（代码对比，非生产事故）

- 不写 demand 的 `tostory` action  
- 不跑澄清/评审拦截（`tostory.php` 前半）  
- `CreateProductPlan` 无计划 Opened action  
- 无通知、无文件、无 workflow 扩展字段  
- Story close 用 `closedReason=done`，未证明等于禅道 close 流程  

## 6. 已验证 / 未验证 / 待决策

### 已验证（代码或本机 VM）

- 两处 AutoMigrate 行号仍成立。  
- 排期写入远不止 `projectstory`/`planstory`。  
- 禅道表与 WB 表可在同一 GORM 事务。  
- `internal/pkg/zentao` 不是写 Client。  
- 全仓生产写入口见 §3（含 user/dept/role/menu/login/PO）。  
- 任务路径有 `FOR UPDATE` + 归属谓词；排期主对象无 version CAS。  
- 本机 dev 有效库：隧道 VM MySQL 8.0.46、RR、ROW binlog、账号全局 ALL。  
- 定制源码树缺标准 stories/tasks/plans POST；基础包有。  

### 未验证

- 生产主机、账号是否即 VM 的 `zentao@127.0.0.1`、是否 root、是否有备库。  
- 生产是否 OceanBase（跟踪配置有注释主机）。  
- 禅道 Web 进程使用的数据库账号是否与工作台相同。  
- 部署实例的 API 文件集合、是否启用 `api.php/v1`、版本是否等于 max5.6.1。  
- `zt_demandwindow.plan/product` 列的写入者。  
- AutoMigrate 在已存在表上是否仍发 ALTER（本轮未开 SQL 日志跟一次启动）。  

### 待你决策（进入 P1 前）

1. P1a/P1b 是否针对 **本 VM** 演练，还是必须等生产只读核验？  
2. 生产有效配置来源（环境变量 / 未入库文件 / 密钥系统）？  
3. PO 关注/已读是否允许继续走「只读」连接？若只读将变成真副本，这两条写必须先改回主库。  
4. `zt_user` 管理工作台直写是否纳入后续 API 化（P2c），还是长期 legacy。  
5. 排期冲突产品策略：拒绝 / 覆盖 / 合并（影响 P2，不影响 P1a）。  

## 7. P1a / P1b 最小实施切片

详细文件/验收/回滚见同目录 `P1a-P1b-SLICES.md`。摘要：

**P1a 去运行期 DDL（先于 P1b 收权）**

- 改：`bootstrap.go:77` 改为只读结构检查（表+必需列，缺则启动失败）；`operationlog.go:54–56` 删除 AutoMigrate。  
- 补：`db/install.sql` 增加与 VM 已存在列一致的 `zt_operation_logs`（GORM 不会补 install.sql）。  
- 迁移账号在目标环境执行一次 IF NOT EXISTS（VM 已有表则 no-op）。  
- 不恢复 DDL 权限；不删审计表。  
- 审计失败策略保持「业务已返回后 goroutine 吞错」——本切片不改语义。  

**P1b 凭据与权限（P1a 之后）**

- 跟踪配置含非占位秘密：协调是否轮换；Git 改为占位。  
- 为工作台建专用账号；**逐表**授权见切片清单。  
- **不**直接撤销可能仍被禅道使用的 `zentao` 账号。  
- VM 上该账号现为全局 ALL，收权前必须证明不影响禅道自身。  
- 隔离库用新账号测：无 DDL、无未授权表 DML。  

**明确不做（本两切片）**

- 不拆连接池、不改隔离级别、不接 API、不引入框架、不改 workboard.js 基线。  

## 8. 检查结果

本轮只新增本目录文档，无业务 diff。

| 门禁 | 结果 |
|---|---|
| `git diff --check` | 未作为独立步骤跑工作区（无已跟踪 diff）。新增 md 后应再跑 |
| `scripts/check-file-length.sh` | **失败**：`baseline loosening rejected: web/static/js/po/workboard.js (cannot increase baseline from 660 to 728 lines)`。机制：工作区 baseline 文件相对 HEAD 干净时，脚本用 `merge-base HEAD origin/Claude-PO`（`17bd8e7d`）作 trusted 基线，该提交中该文件上限为 660；HEAD 已把基线写成 728。**与本 PLAN 无关。** 状态：**BLOCKED BY EXISTING BASELINE** |
| `make check` 其余步骤 | **未执行**（文件长度门禁已失败；按质量文档不把后续当成通过） |
| DB/API/浏览器验收 | P0 只读；API 未对部署实例发请求 |

Codex 当时的 file-length 失败在本 HEAD 上 **复现**，原因比「工作区把文件改到 728 行」更准确：是 **已提交的 baseline 相对 origin merge-base 放宽**。当前 `workboard.js` 为 728 行，等于 HEAD baseline，不是本轮引入。

## 9. 对 PLAN 的事实修正（依据见上）

已写入 `PLAN.md` §2 / §11。要点：PO 写走只读池；`zt_demandwindow` 生产样例列多于 install.sql；禅道 API 分「定制树 / 基础包 / 未验证部署」三层；本机 RR+ROW 只描述 VM，不是生产。
