# 工作台优化交付记录

优化基线为 Dev-CT `7d45c28af4c37e839cc46413833b93ab17bef390`，对照 Main `0a892260040306f2da19caf5d4a3ed981135f2c9`。按用户后续要求，三批提交已移植到当前 `Dev-CT` 工作树 `/private/tmp/wb-dev-ct`，保留其最新开发提交 `117c13a9ce0e45d91ba50cec519ca149fa3d0d44`；对应优化提交为 `51a8f4f0`、`da7d2501`、`c05c9c57`。其他工作树未修改。

## 全局文件修改理由

用户批准的第三批要求移除 Bootstrap 和重复图标依赖、统一弹窗及请求处理，因此修改 `layout/base.html`、`layout/auth.html`、`app.js` 和 `ui.js` 的公共入口。公共弹窗统一使用已有 `openShowModals` / `closeShowModals`，迁移删除与导出调用方后删除旧实现，保留触发与关闭方式。组件按原顺序直接加载，删除运行时脚本加载器。`layout/header.html` 仅补充角标查询不可用的提示。唯一主题入口仍为 `layout/theme.js`，没有新增主题实现、前端框架或第三方依赖。

## 验收边界

用户后续授权了隔离环境安装、迁移和测试写入。本机生产连接经只读核实为 OceanBase CE 4.2.5.5；验收另建相同版本的隔离实例，监听 33381，禅道 18080，工作台 18091。生产数据库、禅道源码及受保护服务未修改，没有推送或合入 Main。隔离禅道保留镜像标准 API，叠加公司禅道仓库 master `7ba6632cf7893def5dc2fd6f45f29e66c2288d00` 的原样 API、custom 和路由文件；该版本与 Main 的任务 #90092 对应，没有修改 PHP 源码。未通过隔离验证的直写例外仍不具备上线资格。

## 27 项审查问题验收表

状态“源码修复”仅表示实现及对应静态 / 模拟验证，不代表生产或隔离运行验收。

| 问题 | 状态 | 结果与边界 |
|---|---|---|
| R01 | 隔离转研需通过 | 恢复 Main 计划、指派、转研需和任务接口；对应禅道版本的 /demand/:id/tostory 已通过真实 HTTP 和 OB 验证；独立研需和扩展字段残留直写逐项登记，未具备上线资格。 |
| R02 | 源码修复 | 任务编辑不发送 left / consumed / status，回归验证不覆盖剩余工时。 |
| R03 | 源码修复 | 任何排期动作前批量校验全部子 ID、产品范围与执行归属；越权用例验证零调用、零写入。 |
| R04 | 有界并发验收通过 | 验收、交付原生调用在本地事务外；6 轮双系统排期 / 交付共 12 请求成功，字段与单条关联正确。不重放创建/删除，不承诺跨系统整体回滚。 |
| R05 | 部分完成 | 普通基本字段保存只锁目标行；拓扑变更仍保留既有整树一致性协议，已只读核实索引；禅道现有小组更新没有与工作台相同的事务锁协议，不能擅自改禅道，跨系统最小锁集合未完成。 |
| R06 | 源码修复 | 接入现有验收、交付 gateway；窗口关联失败明确返回原生动作已成功、关联失败。 |
| R07 | 源码修复 | API / SQL / 操作日志入口脱敏；合成凭据测试覆盖 GORM Scan 日志及文件权限。 |
| R08 | 源码修复 | 默认 CSRF 实际挂载；真实注册路由验证缺失、错误及有效 token，支持 HTTP 本地环境。 |
| R09 | 隔离 OB 安装通过 | 补三张敏捷业务表安装、升级脚本；在已安装禅道原生结构的隔离 OB 执行安装与升级通过。生产未执行；现有不兼容表不能由 CREATE IF NOT EXISTS 自动修复。 |
| R10 | 源码修复 | 已锁定存在性后允许无变化保存成功。 |
| R11 | 门禁收紧，例外待验证 | 显式自有表集合，覆盖模型与动态表名，未知表失败；28 个残留函数/表命中逐项登记，静态扫描并非完整 AST 证明。 |
| R12 | 部分完成 | 资料与偏好同一短事务，故障回归验证原子性；隔离验证了资料保存、无变化保存、禅道 MD5 密码登录、新密码工作台登录及原会话兼容；测试密码已恢复。 |
| R13 | 源码修复 | 直接需求、直接研发需求、Main 计划关联统一 UNION 去重；展示与删除共享口径。 |
| R14 | 有界并发验收通过 | 后端展示 / 删除共享关联口径及窗口锁；删除先提交时更新明确失败、无孤立产品关联；原生交付阻塞期间窗口删除 9ms 完成，交付随后返回部分完成错误，无失效窗口关联。 |
| R15 | 配置完成，生产值待定 | 连接池、DB / HTTP 超时纳入现有配置；只读回退复用主池，合计生产容量未测。 |
| R16 | 源码修复 | 删除启动与请求 AutoMigrate；运行期 SELECT 检查五张自有表必需列，不执行 DDL。 |
| R17 | 部分完成 | 工作台成员写入先锁小组锚点并固定账户顺序；已确认原生唯一索引 (root,type,account)，成员改为单次原子 upsert。隔离 OB 12 并发创建保持单行及首次加入日期；已验证工作台与原生 addMember 并发，以及原生先提交再确认工作台调整：最终单条成员、工时 6；并发撞重复时原生返回明确失败。只证明这两种有界场景。 |
| R18 | 源码修复 | 角标独立 250ms 总预算，失败显示不可用，保留已有 DTO。 |
| R19 | 窗口规模实测通过 | 任务 / 项目 / 执行批量读取；窗口从 2 个增至 20 个，整条请求 SQL 固定 10 次；基线为 16 → 124 次。排期从 1 个增至 20 个研发需求，SQL 固定 18 次；基线为 18 → 37 次。其他列表仍需验证。 |
| R20 | 源码修复，待实测 | 单条 SQL 内十二项独立计数改为三个条件聚合，使用日期范围；未做生产 EXPLAIN 或性能推断。 |
| R21 | 源码修复 | 操作日志直接短超时写入，删除无界 goroutine；失败记录原因，不改变业务结果。 |
| R22 | 源码清理 | 删除无调用 gateway / 辅助函数，复用用户模块账户映射和公共 HTML 转义。 |
| R23 | 增量改善，存量未清零 | 固定开发基线 117c13a9 到本轮生产源码净减少 14 行；已有超长文件各批不增长；固定基线不变，存量复杂度和重复不因此视为全部消除。 |
| R24 | 部分完成，待视觉验收 | 删除登录 Bootstrap 与 Bootstrap Icons，统一 Font Awesome；请求、删除和通用弹窗入口收敛；历史业务 jQuery / 私有控件尚未全部迁移。 |
| R25 | 部分完成 | 本次登录 CSS 颜色进入已有两套变量职责，唯一主题入口保留；其他存量内联样式 / !important / 新窗口行为未宣称全部清理。 |
| R26 | 静态与模拟验收通过，运行验收待定 | 修复三个前端测试失配，Go / 前端 / make quality 通过；已经取得部分真实页面、Console、HTTP / 数据库及隔离并发、性能证据；全部页面和全部跨系统竞争尚未完成。 |
| R27 | 源码修复 | 非 GET 原生动作按业务 result / status fail 返回失败，保留查询型提示契约。 |

## 验证和减量

- `GOPROXY=off make check`：通过；包括全部 Go 测试、vet、前端脚本测试及直写门禁回归。真实数据库测试按显式隔离授权条件跳过。
- `GOPROXY=https://goproxy.cn,direct make quality`：通过。先前依赖下载阻塞已解决，未移动检查基线或修改既有存量豁免来获得通过。
- 修改 JS 的 `node --check`：通过；未定义颜色变量为 0；新增第三方依赖为 0。
- 两窗口统计的 SQL mock 验证固定三次统计查询；这只是结构性查询数证据，不是禅道同时运行的性能对比。
- Go AST 重复检测同配置、threshold=50、排除测试且不限输出数量：155 → 144 条命中；包括结构性相似，不等于全仓语义重复率。
- 已完成登录、资料、敏捷成员调整，以及本轮团队检索、排期子需求折叠、问题编号的亮暗主题检查；其他管理页面有截图，全部创建 / 编辑交互尚未验收。HTTP 日志与数据库证据不能替代所有浏览器网络证据。

物理行数包含注释、空行，不压行、不移动目录。生产口径为 cmd / internal 非测试 Go，以及 web 自有 HTML / CSS / JS；排除 vendor / node_modules / dist。测试、SQL 和配置工具单列。

| 类别 | 固定基线 117c13a9 | 当前 | 净变化 |
|---|---:|---:|---:|
| 生产源码 | 125541 | 125527 | -14 |
| 测试 | 18072 | 19177 | +1105 |
| 安装 / 升级 SQL | 465 | 673 | +208 |
| 配置和工具 | 1227 | 1242 | +15 |

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
- 使用用户提供的授权文件，仅安装到隔离禅道。与现有生产 API 一致的 token 和 user 查询通过；原生任务创建、编辑、删除、验收和交付已做隔离 HTTP 与数据库验证；业务需求转研需已在对应 Main 的禅道版本验证通过。
- 工作台 Client 拒绝 HTML 200、原生 Ret 非 000000 和写动作 result/status fail；401 后只重试 GET，避免写动作自动重放。
- 登录页保留与 Main 相同的 Logo 文件，仅裁掉源图片右侧白色边缘；亮、暗主题截图已经人工检查，没有替换或生成品牌图。

本次后续修复按当前 Dev-CT `117c13a9` 重新统计：生产源码 125541 → 125303，净变化 -238 行。此前表格是原始三批口径。

后续文件净变化：repo_basic_atomic_concurrency_test.go +7；repo_confirm.go -10；service_adjustment_test.go -3；client.go +18；safety_test.go +24；新增 repo_member_concurrency_test.go 77 行。新增函数为 TestMemberConcurrentUpsertPreservesIdentity、TestBasicSaveDoesNotWaitForUnrelatedRow；新增 CSS 选择器为 0。文档不计入生产源码。

## 本轮修复与验收补充

2026-10-03 再次 fetch 远端小写 main，确认仍为 `0a892260040306f2da19caf5d4a3ed981135f2c9`。继续使用当前 Dev-CT，不切换、合并或推送 Main。仅重建隔离 18091 进程，用户 8098 运行环境未更新。

- 原生任务 DELETE 使用部署版本真实路由，失败后停止剩余动作，不重放已提交删除；真实任务增改删验证工时 consumed=3 / left=5 / status=doing 未被编辑覆盖。
- 修复邮箱正则；小组对象、成员验证及列表从主连接读取，消除弱读造成的新建后不存在和确认后旧工时 / 挂靠显示。工时输入即时记录；成员弹窗层级修复。
- 挂靠团队使用现有 initAutocomplete，983 个组织选项缓存一次读取，中文检索匹配 5 个“产品五部”相关结果。只有选择有效值才保存；输入搜索文本不写入。隔离小组 184 选择后数据库与界面均显示产品五部。旧原生 select 及其选项渲染已删除。
- 子需求 US63379 的三条研发需求 72321 / 72322 / 72323 可独立收起；父级 US63378 收起再展开后保持子需求折叠；再次点击恢复三条，亮暗主题已人工检查。
- 问题编号 #1093 的 chip 实测 clientWidth=scrollWidth=86，列宽 125，取消编号单元格省略；两种主题完整显示。可见性只在隔离副本临时调整，原字段已恢复。
- 公共 ui.js 仅修正搜索下拉底部空间不足时向上定位；原因是本轮真实截图发现团队选项被屏幕下缘遮住，复用同一搜索组件，未新增组件或全局函数。
- 最终 make check、make quality（0 issues）、相关模块 focused Go 测试与新 JS 语法检查通过，固定门禁基线未调整。日志位于仓库外 /private/tmp/wb-opt-devct-{check7,quality7}.log。
- 同一隔离数据、每组 5 次窗口请求：2 窗口基线 / 当前中位数 12.35 / 9.16 ms；20 窗口 205.82 / 31.66 ms。查询数 16 / 10 与 124 / 10。临时 18 个空窗口已删除。这不是生产延迟或禅道无退化的证明。

本轮补跑 9 项真实 OB / 主读验证通过（包括 6 项拓扑、12 并发 upsert、无关行锁和主读测试），无跳过；日志 ob-live-tests-final.log。复查隔离 CDB_OB_DEADLOCK_EVENT_HISTORY 为 0，这不证明所有跨系统场景均安全。

本轮截图位于 `/private/tmp/wb-opt-isolated-20261003/screenshots`，团队搜索、子需求折叠和问题编号的亮暗主题图均已逐张检查；当前窗口下无内部表格纵向滚动，截图包含全部筛选结果。截图和验收脚本不提交 Git。

当前生产物理行数统一按 117c13a9 的 cmd / internal 非测试 Go 与 web HTML / CSS / JS，排除 vendor / node_modules / dist，计入新增文件，统计注释与空行：125541 → 125527，净减少 14 行。第三方 Bootstrap 资源删除与历史批次口径仍见上表，不混用基线。

新增 Go 测试：TestNativeDeletesStopAfterFailureWithoutReplay、TestProfileEmailValidation、TestObjectAuthorizationReadsPrimary。新增 JS 为 agileteam-org.js 的页面事件监听及内部回调，无新增全局函数；既有 toggleBizChildren 增加子级状态，未新增业务抽象。新增 CSS 选择器：.at-org-team-picker、.ui-autocomplete-dropdown[data-autocomplete-for^="atOrgTeam"]、.schedule-child-row:is(.is-hidden, .is-sub-hidden)、.schedule-row-expand[hidden]。新增第三方依赖为 0。

仍不能宣称“所有功能验收通过”：窗口关联 / 删除的已测竞争场景见后文，拓扑跨系统共同锁协议、生产合计容量与全部直写例外发布验证仍未完成；28 项直写例外仍受上线门禁限制。对应 Main 的禅道接口版本已经找到；之前的 404 来自较旧的本机副本，不是最新 Main 缺失接口。

### 本轮文件净行数（相对 ff89c3a）

- internal/module/agileteam/repo.go: +0
- internal/module/agileteam/repo_member_concurrency_test.go: +37
- internal/module/agileteam/service_adjustment.go: -4
- internal/module/profile/form.go: +0
- internal/module/profile/service_test.go: +15
- internal/module/schedule/gateway.go: +1
- internal/module/schedule/safety_test.go: +31
- web/static/css/agileteam/agileteam.css: +3
- web/static/css/auth/login.css: +0
- web/static/css/po/issue-risk.css: +2
- web/static/css/schedule/schedulelist.css: +0
- web/static/css/tokens.css: -2
- web/static/js/agileteam/agileteam-members.js: +0
- web/static/js/agileteam/agileteam.js: -6
- web/static/js/schedule/schedule.js: -4
- web/static/js/ui.js: +0
- web/templates/agileteam/index.html: +1
- web/templates/schedule/index.html: -1
- web/static/js/agileteam/agileteam-org.js: +32
- docs/quality/workbench-optimization.md: 文档更新，单列不计生产源码。

### 搜索结果竞争补充

真实页面检查发现初始列表请求晚返回可覆盖新搜索结果，已增加请求序号，过期成功与失败都丢弃；加载占位前销毁旧检索下拉，避免筛选掉的小组残留浮层和监听。折叠按钮重复渲染合并后，agileteam.js 本批净变化 0，保持 499 行。实际页面立即搜索 WB隔离验收后稳定为 4 行、4 个检索输入、4 个浮层。

新增测试 verifyLatestListWins 运行完整生产 JS，以逆序成功及旧请求失败验证最新结果不被覆盖，并验证移除输入前调用 destroyAutocomplete。tests/unit/frontend/agileteam-url-filters.test.js 净 +38 行；新增 CSS 选择器与第三方依赖均为 0。报告文字单列，不计生产代码。最终门禁日志更新为仓库外 check8 / quality8，基线未变。

## 最新 Main 接口复核（2026-10-03）

再次 fetch 公司 main 确认为 `0a892260`；对应公司禅道 master `7ba6632c` 同为任务 #90092。仅更新远端引用，未合并、推送或改生产安装。隔离安装保留镜像标准接口，叠加原样定制文件。工作台共享删除函数恢复 Main 的 POST `/deletetasks`、`/deletestories` 契约；批量失败明确提示可能部分提交，不自动重放。

隔离真实接口验证：需求 1063429 转出研发需求 1072380，归属产品 1；任务创建、编辑、批量删除通过，编辑后 consumed=3、left=5、status=doing，删除后 deleted=1。转研需请求数组、响应 ID、失败无替代写入，以及两种批量删除请求和 401 不重放均有契约回归测试。证据留在 `/private/tmp/wb-opt-isolated-20261003`，不提交测试凭据、截图和脚本。

本轮文件净行数：gateway.go +2（368 行，未超过 500）；safety_test.go +64；本文 +10 单独计为文档。生产源码本轮 +2，但相对开发优化基线仍净减少 214 行。没有新增生产函数、CSS 选择器、依赖或前端改动。新增测试函数为 TestToStoryPreservesMainContract、TestToStoryFailureNeverUsesAlternateWriteRoute、TestNativeDeletesPreserveMainContractWithoutReplay；替代旧顺序删除测试。已有共享函数服务两种删除操作，无新增包装层。

本轮验收命令：`make check`、`make quality` 和最终排期模块回归均通过；离线 quality 首次因 lint 工具解析阻塞，联网后发现的新测试复杂度问题已通过缩短测试结构修复，未改变检查基线。生成的研发需求也通过工作台删除接口验证 deleted=1。

### 2026-10-03 全局固定入口补验

用户要求所有业务页面遵循同一个“固定到工作台”设计，因此修改共享 `web/static/js/layout/pinned-pages.js`：沿用现有账号隔离存储与顶栏按钮，从已渲染、受权限控制的导航补齐目录；任务看板保留独立目标；编辑子路径和旧别名归属对应功能入口。没有修改 header/base、增加组件体系或全局函数。

本项文件变化（相对本轮 HEAD）：`pinned-pages.js` 净增 19 行，无新增函数或 CSS 选择器；`tests/unit/frontend/pinned-pages.test.js` 新增 35 行，新增测试辅助函数 `openPage`，不计生产源码。

隔离 18091 已验证版本跟进固定按钮、任务看板固定按钮以及页面切换后状态保留，检查版本跟进亮暗截图与 Console；截图在私有验收目录，不提交 Git。`GOPROXY=off make check`、`make quality` 与 `git diff --check` 通过。该结果仅覆盖此项，不代表全部功能验收完成；8098 未重建，Main 与禅道源码未修改。

### 2026-10-03 分页弹层与底部脚注

分页条数选择在共享 `PersonalList.renderPager` 中复用现有 `initAutocomplete`，独立控件容器避免清除按钮拉宽分页栏；只接受原有允许条数。隔离 18091 我的待办真实数据验证 20→10 条切换、URL 与分页更新，以及亮暗菜单在输入框正上方展开，Console 无错误。移除首页列表获取时间、数据来源脚注及其时间生成和样式，仓库检索不存在剩余相同脚注。

文件净行数（相对本轮 HEAD）：`personal-list.js` +1；`personal-workspace.css` -1；`home-render.js` -10；`home.js` -8；`home.html` -4；`contrast-light.css` -10。合计生产源码 -32。无新增具名函数；新增 CSS 选择器 `.pager-size-control`，删除两处 `.data-info` 规则。模板删除的浏览器验收仍需服务重新加载模板后补验；8098 保持原服务。本项不代表全部业务验收通过。


## 本会话页面反馈集中验收（2026-10-03）

重新获取 company/main，仍为 `0a892260040306f2da19caf5d4a3ed981135f2c9`；当前 Dev-CT 修改，未合并、推送 Main。8098 未部署；以下真实验收来自隔离 OB 33381、禅道 18080、工作台 18091，不能代替生产发布验收。

| 用户反馈 | 实现与真实验收 |
|---|---|
| 深色登录及 Logo | 沿用前批次 Main 素材和主题修复；没有重新生成 Logo。已有亮暗截图与登录流程证据。 |
| 问题 / 编号右侧出现省略号 | 前批次问题 #1093 徽章已通过双主题；本轮实际发现待办七位编号仍超宽，编号列改为 160px。 |
| 挂靠团队支持搜索 | 复用公共 autocomplete；开发小组输入“核心”只出现匹配组织，已看亮暗全页图。父级小组同样支持检索。 |
| 零任务的人仍显示 | 在现有成员读取结果中补零，无额外 SQL；程统(003030) 0 显示且可筛选，三列为空，已看亮暗图。展开更多成员不重复。 |
| 子需求折叠研发需求 | #US63379 独立折叠三条研发需求，#US63380 仍显示；保留父级折叠与持久化，已看亮暗图。 |
| 窗口卡片风格 | 统一主题变量、卡片边框和状态顶部色，统计固定三列排列，避免不同数字造成卡片换行不一致。 |
| 快速创建迭代 | 两周 / 四周从北京时间当天起，分别含 14 / 28 日；计划模式使用所选产品计划的完整日期。Main 原生 POST /projects/:id/executions，products 数组与 plans 按产品 ID 索引的数组契约，未改 PHP。 |
| 迭代自动命名 | 后端按“开始日期 - 结束日期”生成名称，前端只读展示且随周期 / 计划更新，无手工重复输入。实际创建 21096 四周、21097 两周、21098 计划周期；计划 23609 的关联真实存在。 |
| 创建与选用交互 | 已验证先创建后拆任务时三行自动选用；已有任务行选择不被覆盖。切换项目 / 需求的过期响应不应用；创建失败不重试。禅道同项目同名错误明确返回失败，未重复创建。 |
| 人员调整与团队信息混淆 | 列表分别提供“团队信息”“成员管理”；人员入口直达成员页且隐藏保存基本信息；切换基本信息才显示保存。真实两主题及显示状态验证通过。 |
| 查看关闭 / 过期版本 | 全部窗口可检索；快捷条仅显示未关闭窗口及选中历史窗口。在 16 个窗口中找到首 12 条之外的关闭窗口，选中 ID=62、已过期 256 天。修复日期距离读取首窗口而非所选窗口。隔离历史样本已软删除清理。 |
| 全局固定到工作台 | 已有公共固定组件从授权导航收集目录，保留同一持久化入口；版本跟进、任务看板、待办等均有按钮，没有第二套组件。 |
| 每页条数下拉错位 | 个人列表 / 看板 / 抽屉的共享分页及服务端组件复用同一 autocomplete，固定控件宽度；待办 20 → 10 实际请求和菜单位置通过；服务端排期页下拉也在输入框附近。 |
| 底部无关小字 | 删除首页“列表获取时间 / 数据来源”生成、模板和样式；全仓未再找到该页脚文案，首页真实数据亮暗全页 / 滚动到底已看。业务指标表本身的“数据来源”字段保留。 |

新增依赖为 0，继续 SSR / 原生 JavaScript；jQuery 仅使用已有任务插件。没有新的 window 全局函数。`schedulemodal.css` 的浅色字面底色改为已有双主题变量，深色迭代区域白块已消除。共享分页模板 / CSS、公共固定页目录及公共用户映射属于用户要求的全局一致性范围。

Go / 前端 / vet / 直写回归 `make check` 通过，在线 `make quality` 为 0 issues，未更改固定检查基线；离线 GOPROXY=off 的质量命令曾因模块元数据检查失败，不作为通过证据。隔离 OB 显式授权的拓扑 / 成员竞争测试无跳过并通过，未定义颜色变量为 0。Console 警告 / 错误为 0（已验收页面）；业务同名失败响应另行保留，不作为浏览器成功请求。

当前固定口径包括注释和空行；生产 125541 → 125527，净减少 14 行。测试为 *_test.go 与 tests 下 Go / JS / Python；SQL 为 db/**/*.sql；配置工具为 configs、tools 与 Makefile / .golangci.yml / .stylelintignore。没有压行、生成代码或改变生产统计目录。已有超过 500 行的修改文件均净减少或持平。

### 本轮文件行数和符号（相对 f4756a0d）

| 文件 | 净行数 | 当前行数 |
|---|---:|---:|
| `docs/quality/workbench-optimization.md` | +169 | 335 |
| `docs/quality/workbench-write-exceptions.md` | +8 | 64 |
| `internal/module/po/boardownermembers_test.go` | +42 | 42 |
| `internal/module/po/form_version_follow.go` | +1 | 156 |
| `internal/module/po/repoboardtask.go` | +14 | 307 |
| `internal/module/po/service_version_follow.go` | +2 | 338 |
| `internal/module/po/stage_unify_test.go` | +28 | 191 |
| `internal/module/profile/atomic_test.go` | +34 | 75 |
| `internal/module/profile/repo.go` | +4 | 220 |
| `internal/module/schedule/form_task.go` | +1 | 165 |
| `internal/module/schedule/formiteration.go` | +29 | 29 |
| `internal/module/schedule/gateway.go` | +22 | 390 |
| `internal/module/schedule/handler_story_tasks.go` | +1 | 126 |
| `internal/module/schedule/handleriteration.go` | +34 | 34 |
| `internal/module/schedule/iteration_test.go` | +49 | 49 |
| `internal/module/schedule/nativeplanunlink_test.go` | +71 | 71 |
| `internal/module/schedule/repo_story_link.go` | -75 | 207 |
| `internal/module/schedule/repoiteration.go` | +17 | 17 |
| `internal/module/schedule/routes.go` | +1 | 45 |
| `internal/module/schedule/service.go` | +3 | 436 |
| `internal/module/schedule/service_scheduling_save.go` | +0 | 383 |
| `internal/module/schedule/service_story_tasks.go` | +5 | 268 |
| `internal/module/schedule/service_window_authz_test.go` | +17 | 233 |
| `internal/module/schedule/serviceiteration.go` | +103 | 103 |
| `tests/unit/frontend/agileteamdetail.test.js` | +54 | 54 |
| `tests/unit/frontend/pinnedpages.test.js` | +39 | 39 |
| `tests/unit/frontend/workboardownerchips.test.js` | +31 | 31 |
| `tools/zentao-write-allowlist.txt` | -1 | 51 |
| `web/static/css/auth/login.css` | +0 | 334 |
| `web/static/css/agileteam/agileteam.css` | +16 | 319 |
| `web/static/css/components/pager.css` | +2 | 146 |
| `web/static/css/po/contrast-light.css` | -10 | 217 |
| `web/static/css/po/personal-workspace.css` | -1 | 461 |
| `web/static/css/po/todos.css` | +0 | 158 |
| `web/static/css/po/version-follow.css` | +5 | 464 |
| `web/static/css/schedule/schedulemodal.css` | +0 | 700 |
| `web/static/css/schedule/schedulewindow.css` | -3 | 428 |
| `web/static/js/agileteam/agileteam-detail.js` | +4 | 306 |
| `web/static/js/agileteam/agileteam.js` | +0 | 499 |
| `web/static/js/layout/pinned-pages.js` | +19 | 296 |
| `web/static/js/po/home-render.js` | -10 | 287 |
| `web/static/js/po/home.js` | -8 | 431 |
| `web/static/js/po/personal-list.js` | +1 | 465 |
| `web/static/js/po/version-follow.js` | +8 | 459 |
| `web/static/js/po/workboard-core.js` | +0 | 220 |
| `web/static/js/schedule/scheduleintegratedshared.js` | +6 | 277 |
| `web/static/js/schedule/scheduleintegratedtasks.js` | -16 | 973 |
| `web/static/js/schedule/scheduletasklistmodal.js` | +74 | 226 |
| `web/static/js/schedule/scheduletaskmodal.js` | -54 | 750 |
| `web/templates/components/pager.html` | +7 | 127 |
| `web/templates/po/home.html` | -4 | 402 |
| `web/templates/po/version_follow.html` | +2 | 104 |
| `web/templates/schedule/index.html` | +0 | 853 |

新增命名函数 / 方法（测试单列于文件名中）：

- `internal/module/po/boardownermembers_test.go: TestBoardOwnersIncludeZeroTaskMembersWithoutExtraQueries`
- `internal/module/po/boardownermembers_test.go: TestBoardStoryOwnersDoNotAddUnrelatedMembers`
- `internal/module/po/stage_unify_test.go: TestVersionFollowDistanceUsesSelectedWindow、TestVFStayMissingDate`
- `internal/module/profile/atomic_test.go: TestProfileSaveAtomicCases`
- `internal/module/profile/atomic_test.go: testProfileSave`
- `internal/module/schedule/formiteration.go: Validate`
- `internal/module/schedule/gateway.go: removeStoryFromOtherPlans`
- `internal/module/schedule/handleriteration.go: CreateIteration`
- `internal/module/schedule/iteration_test.go: TestCreateIterationDenialPrecedesNativeCall`
- `internal/module/schedule/iteration_test.go: TestIterationDatesAndNativePlanArray`
- `internal/module/schedule/nativeplanunlink_test.go: TestNativePlanUnlinkContractAndFailure`
- `internal/module/schedule/nativeplanunlink_test.go: testNativePlanUnlink`
- `internal/module/schedule/repoiteration.go: IterationPlans`
- `internal/module/schedule/service_window_authz_test.go: TestUpdateWindowDeletedAfterReadDoesNotRecreateLinks`
- `internal/module/schedule/serviceiteration.go: CreateIteration`
- `internal/module/schedule/serviceiteration.go: iterationBody`
- `tests/unit/frontend/agileteamdetail.test.js: latestDetailWins`
- `tests/unit/frontend/pinnedpages.test.js: openPage`
- `web/static/js/po/version-follow.js: selectWindow`
- `web/static/js/schedule/scheduletasklistmodal.js: createIteration`
- `web/static/js/schedule/scheduletasklistmodal.js: previewIteration`
- `web/static/js/schedule/scheduleintegratedshared.js: resolveUserLabel`

新增 CSS 选择器：

- `web/static/css/agileteam/agileteam.css: .at-change-counts, .at-review-summary`
- `web/static/css/agileteam/agileteam.css: .at-child-indent .at-sub-id`
- `web/static/css/agileteam/agileteam.css: .at-pending-actions`
- `web/static/css/agileteam/agileteam.css: .at-review-reason`
- `web/static/css/agileteam/agileteam.css: .at-review-summary`
- `web/static/css/agileteam/agileteam.css: .po-agileteam .page:not(.active)`
- `web/static/css/components/pager.css: .pagination-size-form .pager-size-control`
- `web/static/css/po/personal-workspace.css: .pager-size-control`
- `web/static/css/po/version-follow.css: .vf-window-bar .ui-autocomplete-input-wrap`
- `web/static/css/schedule/schedulewindow.css: .task-iteration-dates`
- `web/static/css/schedule/schedulewindow.css: .task-iteration-fields`
- `web/static/css/schedule/schedulewindow.css: .task-iteration-fields label`
- `web/static/css/schedule/schedulewindow.css: .task-iteration-form`
- `web/static/css/schedule/schedulewindow.css: .task-iteration-form summary`
- `web/static/css/schedule/schedulewindow.css: .task-iteration-periods .active`
- `web/static/css/schedule/schedulewindow.css: .task-iteration-periods, .task-iteration-dates`


自查：任务范围、函数规模 / 复杂度 / 嵌套门禁、已有超长文件不增长、实际重复逻辑共用、无新增依赖、逐文件行数 / 符号登记及两套主题变量检查均完成。仍未解除 R05 共同拓扑锁协议与 28 项逐项直写发布阻止条件；这部分不能通过修改工作台单方代码宣称禅道已遵守协议。


### 扩大共享库压力样本与发布阻止项

同一隔离数据集分别 120 次禅道读取；共享阶段各 240 次工作台读取、2 个原生并发读者和 4 个工作台读者。三阶段结束后停止比较基线运行，恢复验收工作台；各阶段均无非 2xx。OB 系统视图 CDB_OB_DEADLOCK_EVENT_HISTORY 最终为 0，仅代表已测场景。

| 阶段 | 禅道中位 / P95 ms | 工作台中位 / P95 ms | 合计连接峰值 |
|---|---:|---:|---:|
| 禅道单独运行 | 1751.10 / 3112.31 | — | 4 |
| 开发基线共同运行 | 2262.98 / 5157.33 | 366.01 / 989.40 | 11 |
| 优化版共同运行 | 1781.68 / 4223.67 | 278.91 / 587.73 | 11 |
| 优化版主池上限 2 的隔离试验 | 2375.39 / 6849.38 | 366.76 / 758.78 | 9 |

优化版比开发基线延迟降低，但禅道 P95 仍比单独运行高，不能把“禅道无可测退化”标为通过。单方缩主池试验也未改善结果，已恢复隔离默认配置，未据此修改生产值。只读池成功连接也会贡献连接占用，生产须根据两套服务及只读池合计预算验证；回退复用主池不能代替正常多池容量评估。不同阶段存在系统负载波动，结果不能外推生产容量。

发布仍阻止：R05 原生拓扑 autocommit 缺少共同锁协议；28 项原生直写例外仍非全部能力 / 并发验收；生产合计容量与禅道无退化未达到通过条件。保留 Main 与禅道源码基线，不能在工作台单方修改后假称这些条件成立。本会话页面反馈的修复与截图通过不等于原 27 项全部通过。

### 2026-10-04 登录图标补验

登录 Logo 原图含白色边缘；移除单边裁切，使用四周对称圆角裁切，原始品牌图像不变。login.css 净增 0 行、无新增函数或选择器。真实浅色、深色全页截图 login-logo-edge-light.png / login-logo-edge-dark.png 已逐张检查，白边与不对称消失。版本跟进同时修复数据库零日期造成虚假 106751 天超期，service_version_follow.go 净增不变；stage_unify_test.go 新增 TestVFStayMissingDate 覆盖零日期与截止日回退。截图存于隔离验收目录，不入 Git。

版本跟进搜索组件实测外层为 header 自身，最终宽度约束改到已有 .ui-autocomplete-input-wrap；默认仅渲染需求行，点击展开 / 收起分别产生 1 / 0 条明细行。version-compact-final-light.png / dark.png 已逐张检查 7 条真实需求；schedule-child-collapsed-light.png / dark.png 已检查子需求收起后的研发行隐藏。原生迭代 21098 只读确认名称 2026-10-03 - 2026-10-29、起止日期一致，产品 397 的计划关联为 ,23609,。
