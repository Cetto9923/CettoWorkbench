# P1a / P1b 可实施切片

日期：2026-09-06
依赖：`P0-AUDIT.md`
状态：P1a / P1b-code **CLOSED**。**P1b-vm Preflight Closeout** 已对齐矩阵↔SQL（`P1B-VM-GRANT-CHECKLIST.md`）。**禁止**未过 EXECUTION GATE 前执行 CREATE USER / GRANT。本 Agent **未**执行 DDL/GRANT。

约束：不拆双连接事务；不改 RC；不接禅道 API；不引入通用框架；不扩大 file-length 基线；不顺手修无关文件。

---

## P1a — 去掉运行期 DDL

### 目标

运行账号不再 `CREATE/ALTER`。`zt_operation_logs` 只通过受控迁移存在。启动时结构不够则失败退出，而不是静默建表。

### 精确改动

| 文件 | 现状 | 拟改 |
|---|---|---|
| `internal/bootstrap/bootstrap.go:77–79` | `db.AutoMigrate(&model.OperationLog{})`，失败则进程退出 | 删除 AutoMigrate。改为调用本包小函数：`HasTable("zt_operation_logs")` 且必需列存在（与 Model 一致：`id,tenantId,userId,account,method,path,query,body,ip,userAgent,statusCode,createdAt`）。缺表/缺列 → `fmt.Errorf` 退出。**不要**只 `HasTable`。 |
| `internal/middleware/operationlog.go:33,54–56` | `sync.Once` + AutoMigrate | 删除 `operationLogTableInit` 与 AutoMigrate。写日志失败仍吞错（保持现语义）。 |
| `db/install.sql` | 无 `zt_operation_logs` | 追加 `CREATE TABLE IF NOT EXISTS zt_operation_logs`，列/类型对齐 VM 已存在结构（见 P0 §3.7），InnoDB utf8mb4。 |
| 新建 `internal/bootstrap/operationlog_schema.go`（或同文件私有函数，保持 <300 行） | 无 | 只读检查。单测用 sqlite/sqlmock **不要**连真库，或纯函数测列名集合。 |

不改 `RecordOperationLog` 的异步写入、敏感字段过滤、goroutine。不改审计失败是否阻断 HTTP——现行为是业务 `c.Next()` 之后写，失败忽略。

### 依赖

- 目标环境由 **迁移账号** 先执行 install 增量（VM 已有表则 IF NOT EXISTS 为 no-op）。
- 确认无其他 AutoMigrate（P0 仅这两处）。
- GORM AutoMigrate 可能加索引；本切片 **不**在启动时补索引。若以后缺索引，另开迁移。

### 测试

- 单元：结构检查在「缺表 / 缺列 / 列齐全」三态。见 `internal/bootstrap/operationlog_schema_test.go`。
- 若有集成库：无 DDL 权限账号启动成功（表已存在）；删列后启动失败。**本切片无隔离集成库，该门禁跳过**（不得连开发/生产实例冒充通过）。
- `git diff --check` + `make check`。不碰 `workboard.js`。

### 发布顺序

1. 迁移账号在目标库执行 `zt_operation_logs` DDL（评估该表 MDL；该表很小，不是「锁住同 schema 所有查询」）。
2. 发应用。
3. 用无 DDL 权限账号验证启动与一次 POST（审计行可插入）。

### 验收

- 无 CREATE/ALTER 权限下进程能起。
- 缺表或缺 `createdAt` 等必需列时启动明确失败。
- 已有 25 行级审计数据仍可读（VM 样例）；新写请求仍插入（权限足够时）。
- 日志/SQL 跟踪一次启动，确认无 `CREATE TABLE`/`ALTER`。

### 回滚

- 切片 ID：**P1a-remove-runtime-ddl**。
- 回退到「仍禁止自动 DDL」的兼容构建：保留 install.sql 表，启动检查可暂时放宽为仅 HasTable——**仍不恢复 AutoMigrate、不恢复 DDL 权限、不 DROP 审计表**。
- 禅道影响：无。只动工作台审计表。

### 风险

- AutoMigrate 若曾隐式加过索引，去掉后不会再补。
- 中间件写失败本来就静默；缺表时审计丢数据但 HTTP 仍 200——启动检查就是为了避免这种长期丢审计。

---

## P1b-code — PO 读写拆分与 tracked 凭据占位

### 目标

1. PO **读走只读池、写走主库**（最小补丁，不是整模块切回主库）。
2. Git 跟踪的 `configs/config.yaml` 无真实秘密；运行时仍可用 `WORKBENCH_*` 覆盖。
3. **不**在本切片执行 GRANT / 建号 / REVOKE（那是 **P1b-vm**）。

### 过期口径更正（相对 P0 草案）

| 过期说法 | 现行口径 |
|---|---|
| 把整个 `po.NewRepo` 从 `dbReadonly` **改回主库** | `NewRepo(readDB, writeDB)`：**查询**用只读池，`SaveDemandFollow` / `SaveNoticeRead` / `SaveAllNoticeReads` **只用** `writeDB`（主库）。其它 PO 查询不得误走写连接。 |
| P1b = 代码 + VM GRANT 一次做完 | 拆成 **P1b-code**（本切片）与 **P1b-vm**（DBA 脚本与授权演练，另开；本 Agent 不执行）。 |

### 精确改动

| 文件 | 拟改 |
|---|---|
| `internal/module/po/repo.go` | `Repo` 增加 `writeDB`；`NewRepo(readDB, writeDB *gorm.DB)`。 |
| `internal/module/po/repofollow.go` / `reponotice.go` | 写路径改用 `writeDB`；读路径仍用 `db`。 |
| `internal/bootstrap/bootstrap.go` | `po.NewRepo(dbReadonly, db)`。 |
| `configs/config.yaml` | `user` / `zentao.account` 用 `CHANGE_ME`；`password` 用扫描器已认可的占位 `changeme`。不提交真实值。 |
| 单测 | `repo_rw_test.go`：写只触 write mock；读只触 read mock。 |

不改 `database.go` 连接池实现。不改隔离级别。不引入 DB manager / adapter / 事务框架。

### 环境变量（运行时，不写密码进 Git）

有效配置 = 文件 + `WORKBENCH_*`（`.` → `_`）。至少：

- `WORKBENCH_DATABASE_USER` / `WORKBENCH_DATABASE_PASSWORD` / `WORKBENCH_DATABASE_HOST` / …
- `WORKBENCH_DATABASEREADONLY_*`
- `WORKBENCH_ZENTAO_ACCOUNT` / `WORKBENCH_ZENTAO_PASSWORD`（若使用）

`WORKBENCH_MODE=dev` 仍加载 gitignore 的 `configs/config.dev.yaml`。

### 测试 / 验收（P1b-code）

- 单元：读写连接隔离断言通过（含 `SaveAllNoticeReads` 写连接契约）。
- `git diff --check` + `make check`。当前 P1b-code GitHub `regression-gates` 已通过。
- **不做** VM 上 GRANT 后的登录/排期写验收（属 P1b-vm）。

### 回滚

- 切片 ID：**P1b-code-po-rw-config**。
- 可回退为单连接构造，但 **不要**把真实密码写回 Git。

---

## P1b-vm — 凭据与逐表授权（本轮不做）

### 目标

工作台运行账号不再全局 ALL。不撤销禅道进程仍在用的账号。

### 前置

1. 生产有效配置从哪加载（本机 `config.dev.yaml` ≠ 生产）。
2. 禅道 PHP 用的 DB 用户名是否就是隧道里看到的 `zentao@127.0.0.1`。若是，**禁止**对该账号 REVOKE。
3. 跟踪的 `configs/config.yaml` 中曾入库的非占位密码是否需轮换（另开变更窗口；本 Agent 不执行轮换）。
4. P1b-code 已上线（PO 写已走主库）。

### 拟改（工作台仓，供 DBA；Agent 不执行）

| 文件 | 拟改 |
|---|---|
| 本目录 `grants-workbench-runtime.sql`（**新文件，供 DBA，本机不执行**） | 见下方 GRANT 清单。 |

不改 `database.go` 连接池实现。不改隔离级别。

### 运行账号授权清单（P0 证实的写表）

专用用户例如 `wb_runtime@<host>`（名字待 DBA）。只授需要的表。MySQL 无单独 TRUNCATE 权限（依赖 DROP）——不授 DROP。

`zt_depts`、`zt_workbench_notify_reads` 当前仍未纳入 `db/install.sql`（VM 上表已存在）。`zt_operation_logs` 已由 P1a 纳入 `db/install.sql`，由受控迁移创建，运行账号不得执行 DDL。P1b-vm 仍按表授权；不顺手补部门/已读表 DDL。

**Workbench 自有 DML**

| 表 | INSERT | UPDATE | DELETE | 理由 |
|---|---|---|---|---|
| `zt_operation_logs` | ✓ | | | 中间件审计 |
| `zt_login_failures` | ✓ | | ✓ | 失败记录 + 清理 |
| `zt_login_logs` | ✓ | | | 登录日志 |
| `zt_roles` | ✓ | ✓ | | 软删走 UPDATE |
| `zt_role_permissions` | ✓ | | ✓ | 替换绑定 |
| `zt_gf_user_roles` | ✓ | | ✓ | 替换绑定 |
| `zt_menus` | ✓ | ✓ | | 软删 UPDATE |
| `zt_depts` | ✓ | ✓ | | 软删 UPDATE |
| `zt_versionwindow` | ✓ | ✓ | | 软删 |
| `zt_versionwindowproduct` | ✓ | ✓ | ✓ | 更新时 Unscoped 物理删 |
| `zt_demandwindow` | ✓ | | ✓ | Unscoped 物理删 + INSERT |
| `zt_workbench_notify_reads` | ✓ | | | 已读 |

**禅道 / 混写（过渡期仍要 DML）**

| 表 | INSERT | UPDATE | DELETE | 理由 |
|---|---|---|---|---|
| `zt_user` | ✓ | ✓ | | admin 用户模块；软删 UPDATE |
| `zt_starinfo` | ✓ | ✓ | | PO 关注 |
| `zt_story` | ✓ | ✓ | | 排期 |
| `zt_storyspec` | ✓ | | | 排期 |
| `zt_task` | ✓ | ✓ | | 排期 |
| `zt_taskspec` | ✓ | | | 排期 |
| `zt_action` | ✓ | | | 排期 |
| `zt_planstory` | ✓ | | ✓ | 关联/解除 |
| `zt_projectstory` | ✓ | ✓ | | UPSERT |
| `zt_productplan` | ✓ | | | 自动建计划 |
| `zt_demand` | | ✓ | | 排期字段 |

**只读 SELECT**：上表 + 排期/PO 查询用到的 `zt_product` `zt_project` `zt_dept` `zt_holiday` `zt_teamgroup` `zt_notify` `zt_demandclarify` 等。GRANT 时对查询表授 SELECT。不要 `*.*`。

**明确不授：** CREATE/ALTER/DROP/GRANT OPTION/SUPER/RELOAD 等。

### 测试 / 验收

- 新账号：启动 + 登录 + 一次排期只读页 + 一次已读。
- 故意对 `zt_product` 做 INSERT → 拒绝。
- 故意 CREATE TABLE → 拒绝。
- 回归：`make check`；排期保存/窗口创建在 **授权后的验收环境** 做一次。

### 发布顺序

1. P1a + P1b-code 已上线。
2. DBA 建用户、按表 GRANT；工作台改连接用户。
3. 观察禅道 Web 仍用原账号。
4. 轮换若需要，另开变更窗口。

### 回滚

- 切片 ID：**P1b-runtime-grants**（= P1b-vm）。
- 连接改回仍可用的旧工作台账号（若未撤销）。
- **不**把 root / 已泄露密码写回 Git。
- 补漏 GRANT，而不是重新 ALL PRIVILEGES。
- 禅道：只要没动禅道自己的账号，无影响。

### 禅道影响

无代码影响。风险只来自误 REVOKE 共用账号——本切片禁止。

---

## 进入实施确认

- [x] 先做 P1a，再做 P1b-code
- [x] PO 采用 **读 RO / 写 RW**（不是整模块改回主库）
- [x] tracked `configs/config.yaml` 占位化（P1b-code）
- [ ] P1b-vm：实施环境仅 VM / 含生产
- [ ] 是否授权轮换已入库的非占位密码（本 Agent 不执行轮换）
- [ ] 本轮 **不做** P1b-vm / P1c
