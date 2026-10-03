# 工作台优化交付记录

优化基线为 Dev-CT `7d45c28af4c37e839cc46413833b93ab17bef390`，对照 Main `0a892260040306f2da19caf5d4a3ed981135f2c9`。按用户后续要求，三批提交已移植到当前 `Dev-CT` 工作树 `/private/tmp/wb-dev-ct`，保留其最新开发提交 `117c13a9ce0e45d91ba50cec519ca149fa3d0d44`；对应优化提交为 `51a8f4f0`、`da7d2501`、`c05c9c57`。其他工作树未修改。

## 全局文件修改理由

用户批准的第三批要求移除 Bootstrap 和重复图标依赖、统一弹窗及请求处理，因此修改 `layout/base.html`、`layout/auth.html`、`app.js` 和 `ui.js` 的公共入口。公共弹窗统一使用已有 `openShowModals` / `closeShowModals`，迁移删除与导出调用方后删除旧实现，保留触发与关闭方式。组件按原顺序直接加载，删除运行时脚本加载器。`layout/header.html` 仅补充角标查询不可用的提示。唯一主题入口仍为 `layout/theme.js`，没有新增主题实现、前端框架或第三方依赖。

## 验收边界

用户后续授权了隔离环境安装、迁移和测试写入。本机生产连接经只读核实为 OceanBase CE 4.2.5.5；验收另建相同版本的隔离实例，监听 33381，禅道 18080，工作台 18091。生产数据库、禅道源码及受保护服务未修改，没有推送或合入 Main。禅道验收代码只读复制自现有生产安装，不修改源码来适配工作台；生产基线以 Main 对应版本为准。未通过隔离验证的直写例外仍不具备上线资格。

## 27 项审查问题验收表

状态“源码修复”仅表示实现及对应静态 / 模拟验证，不代表生产或隔离运行验收。

| 问题 | 状态 | 结果与边界 |
|---|---|---|
| R01 | 源码修复 | 恢复 Main 计划、指派、转研需和任务接口；独立研需和扩展字段残留直写逐项登记，未具备上线资格。 |
| R02 | 源码修复 | 任务编辑不发送 left / consumed / status，回归验证不覆盖剩余工时。 |
| R03 | 源码修复 | 任何排期动作前批量校验全部子 ID、产品范围与执行归属；越权用例验证零调用、零写入。 |
| R04 | 原冲突链已移除，待并发验收 | 验收、交付原生调用在本地事务外；不重放创建/删除，不承诺跨系统整体回滚。 |
| R05 | 部分完成 | 普通基本字段保存只锁目标行；拓扑变更仍保留既有整树一致性协议，已只读核实索引；禅道现有小组更新没有与工作台相同的事务锁协议，不能擅自改禅道，跨系统最小锁集合未完成。 |
| R06 | 源码修复 | 接入现有验收、交付 gateway；窗口关联失败明确返回原生动作已成功、关联失败。 |
| R07 | 源码修复 | API / SQL / 操作日志入口脱敏；合成凭据测试覆盖 GORM Scan 日志及文件权限。 |
| R08 | 源码修复 | 默认 CSRF 实际挂载；真实注册路由验证缺失、错误及有效 token，支持 HTTP 本地环境。 |
| R09 | 隔离 OB 安装通过 | 补三张敏捷业务表安装、升级脚本；在已安装禅道原生结构的隔离 OB 执行安装与升级通过。生产未执行；现有不兼容表不能由 CREATE IF NOT EXISTS 自动修复。 |
| R10 | 源码修复 | 已锁定存在性后允许无变化保存成功。 |
| R11 | 门禁收紧，例外待验证 | 显式自有表集合，覆盖模型与动态表名，未知表失败；29 个残留函数/表命中逐项登记，静态扫描并非完整 AST 证明。 |
| R12 | 部分完成 | 资料与偏好同一短事务，故障回归验证原子性；原生用户 / 密码例外及会话兼容性仍需部署版本验证。 |
| R13 | 源码修复 | 直接需求、直接研发需求、Main 计划关联统一 UNION 去重；展示与删除共享口径。 |
| R14 | 源码修复，待并发验收 | 后端锁窗口并检查关联后删除；本地关联写入遵守同一窗口锁，禅道侧竞争未实测。 |
| R15 | 配置完成，生产值待定 | 连接池、DB / HTTP 超时纳入现有配置；只读回退复用主池，合计生产容量未测。 |
| R16 | 源码修复 | 删除启动与请求 AutoMigrate；运行期 SELECT 检查五张自有表必需列，不执行 DDL。 |
| R17 | 部分完成 | 工作台成员写入先锁小组锚点并固定账户顺序；已确认原生唯一索引 (root,type,account)，成员改为单次原子 upsert。隔离 OB 12 并发创建保持单行及首次加入日期；禅道与工作台共同竞争仍未验收。 |
| R18 | 源码修复 | 角标独立 250ms 总预算，失败显示不可用，保留已有 DTO。 |
| R19 | 源码修复，待实测 | 任务 / 项目 / 执行批量读取；两窗口统计测试固定三次 SQL；真实数据端到端计数与延迟尚未记录。 |
| R20 | 源码修复，待实测 | 十二次统计降为三次条件聚合，使用日期范围；未做生产 EXPLAIN 或性能推断。 |
| R21 | 源码修复 | 操作日志直接短超时写入，删除无界 goroutine；失败记录原因，不改变业务结果。 |
| R22 | 源码清理 | 删除无调用 gateway / 辅助函数，复用用户模块账户映射和公共 HTML 转义。 |
| R23 | 增量改善，存量未清零 | 全生产源码净减少 246 行；已有超长文件各批不增长；固定基线不变，存量复杂度和重复不因此视为全部消除。 |
| R24 | 部分完成，待视觉验收 | 删除登录 Bootstrap 与 Bootstrap Icons，统一 Font Awesome；请求、删除和通用弹窗入口收敛；历史业务 jQuery / 私有控件尚未全部迁移。 |
| R25 | 部分完成 | 本次登录 CSS 颜色进入已有两套变量职责，唯一主题入口保留；其他存量内联样式 / !important / 新窗口行为未宣称全部清理。 |
| R26 | 静态与模拟验收通过，运行验收待定 | 修复三个前端测试失配，Go / 前端 / make quality 通过；真实页面截图、网络与 Console、隔离并发和性能尚未完成。 |
| R27 | 源码修复 | 非 GET 原生动作按业务 result / status fail 返回失败，保留查询型提示契约。 |

## 验证和减量

- `GOPROXY=off make check`：通过；包括全部 Go 测试、vet、前端脚本测试及直写门禁回归。真实数据库测试按显式隔离授权条件跳过。
- `GOPROXY=https://goproxy.cn,direct make quality`：通过。先前依赖下载阻塞已解决，未移动检查基线或修改既有存量豁免来获得通过。
- 修改 JS 的 `node --check`：通过；未定义颜色变量为 0；新增第三方依赖为 0。
- 两窗口统计的 SQL mock 验证固定三次统计查询；这只是结构性查询数证据，不是禅道同时运行的性能对比。
- Go AST 重复检测同配置、threshold=50、排除测试且不限输出数量：155 → 144 条命中；包括结构性相似，不等于全仓语义重复率。
- 改动页面的真实数据亮暗主题全页、内部滚动截图及 Console / 网络证据：未完成。布局保持是实现目标，尚未取得视觉验收结论。

物理行数包含注释、空行，不压行、不移动目录。生产口径为 cmd / internal 非测试 Go，以及 web 自有 HTML / CSS / JS；排除 vendor / node_modules / dist。测试、SQL 和配置工具单列。

| 类别 | Dev-CT 基线 | 优化后 | 净变化 |
|---|---:|---:|---:|
| 生产源码 | 125620 | 125374 | -246 |
| 测试 | 17894 | 18344 | +450 |
| 安装 / 升级 SQL | 465 | 673 | +208 |
| 配置和工具 | 1145 | 1161 | +16 |

删除 13 个 Bootstrap / Bootstrap Icons 资源文件（第三方资源不计入上述生产减量）。保留 SSR、原生 JavaScript、Font Awesome 及现有插件依赖，没有新增前端框架。

## 发布前仍需完成

1. 隔离 OB 和已授权禅道已经准备完成；继续验证原生业务响应契约及 29 项残留直写例外。
2. 完成拓扑最小锁协议与跨系统成员唯一性方案，再做排期 / 交付、窗口关联 / 删除、小组调整 / 成员竞争并发测试，采集死锁、等待、重复数据和连接峰值。
3. 用相同真实数据规模测优化前后 SQL 数、延迟和禅道响应；完成所有修改页面双主题及主要交互证据。
4. 对存量业务 jQuery、私有控件和样式补丁继续逐调用方迁移并删除旧实现。不能在缺少页面验收时批量替换未知交互。
5. 生产迁移与部署单独审批，禁止直接合入 Main；图标升级脚本仅在隔离库执行。

## 自查

任务范围和全局修改理由已核对；无第三方新增依赖；新增 Go 由固定质量门禁验证，未扩大存量豁免。已有超过 500 行的生产文件逐批核对未净增长。每批文件净行数及新增 Go 声明见批次报告；下方补充 JS 声明与 CSS 选择器变化。运行验收缺项如实保留，不勾选完成。

### 本批新增 / 调整的 JS 函数声明

- `web/static/js/schedule/schedule-inline.js`：`return window.appFetch(url, { headers: { Accept: "application/json" } }).then(function (response) {`
- `web/static/js/schedule/scheduletasklistmodal.js`：`return window.appJson("/schedule/stories/" + storyId + "/tasks").then(function (resp) {`
- `web/static/js/schedule/scheduletaskmodal.js`：`return window.appJson("/schedule/stories/" + storyId + "/tasks").then(function (resp) {`
- `web/static/js/ui.js`：`.then(function (data) {`
- `web/static/js/ui.js`：`else window.setTimeout(function () { window.location.reload(); }, 250);`
- `web/static/js/ui.js`：`.catch(function (err) { showToast(err.message || "删除失败，请稍后重试", "error"); });`

### CSS 选择器变化

- `web/static/css/auth/login.css`：新增 []；删除 [':root ']。
- `web/static/css/components/modal.css`：新增 ['.modal-backdrop.show ']；删除 ['.modal-backdrop.open ']。
- `web/static/css/tokens.css`：新增 []；删除 []。

## 隔离验收进展（2026-10-03）

- 使用 OceanBase CE 4.2.5.5，复制 355 张禅道表结构及业务数据；隔离数据规模为需求 41980、研发需求 49808、任务 151438。历史、附件、搜索索引等大表数据排除，因此该样本不能代表完整生产负载。
- 8 项真实 OB 测试通过，包括既有拓扑测试、12 并发成员 upsert 和无关小组锁不阻塞普通保存。没有据此声明跨禅道共同写入安全。
- 使用用户提供的授权文件，仅安装到隔离禅道。与现有生产 API 一致的 token 和 user 查询通过；原生业务写入验收尚未完成。
- 工作台 Client 拒绝 HTML 200、原生 Ret 非 000000 和写动作 result/status fail；401 后只重试 GET，避免写动作自动重放。
- 登录深色页用户指出 Logo 与样式问题，尚未视觉验收通过。Logo 资产与 Main 相同，原图带白色边角；等待正确品牌素材确认，不擅自绘制或替换品牌图。

本次后续修复按当前 Dev-CT `117c13a9` 重新统计：生产源码 125541 → 125303，净变化 -238 行。此前表格是原始三批口径。

后续文件净变化：repo_basic_atomic_concurrency_test.go +7；repo_confirm.go -10；service_adjustment_test.go -3；client.go +18；safety_test.go +24；新增 repo_member_concurrency_test.go 77 行。新增函数为 TestMemberConcurrentUpsertPreservesIdentity、TestBasicSaveDoesNotWaitForUnrelatedRow；新增 CSS 选择器为 0。文档不计入生产源码。
