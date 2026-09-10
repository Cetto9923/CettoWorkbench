# 整合分支稳妥收口记录（2026-09-10）

状态：PARTIAL / BLOCKED BY EXISTING BASELINE。未提交、未推送。
分支 `release/po-integrate-main-202609`，HEAD `cc5640365bc2cfc5961a7188b7f902d0b1d48867`。
工具：Codex；具体运行模型未核实。本记录不认证其他 WIP 或线上写链已完成验收。

## 本轮约定

允许：忽略本地产物、阻止误暂存、确定死代码删除、未接线按钮降级、最小回归门禁。
保留：评审/提交/撤回、澄清、交付、提测、搜索和现有验收/催办；不改变流程、权限或数据库。
暂缓：指标真数、看板 mock 写、假角色 Tab、版本跟进全套催办矩阵；不新增通知/燕讯。
不吸收 Main，不重置/切换分支，不搬 PO 包，不提高文件长度基线。
Demo 为原型仓非运行时；保留 vendor 双份供离线打开，另案核验版本/许可证再决定统一。
本地 PRD HTML（包括 4,936,766 字节 V11.1）及截图保留磁盘，只忽略、不入库。
新增 PRD 正文评审使用 Markdown；确需发布 HTML 时单独明确范围，不强制 add 绕过门禁。

## 已核实与改动

- 全仓引用扫描确认 `isPathInfo`、`pathInfoURL`、`pathInfoValues` 无活调用，删除；`URLWithBase` 强制 GET 不变。
- `filterNoticeRows` 仅测试调用，删除；随之删除仅供该函数使用的 `noticeMatchesKeyword` / `noticeMatchesTimeRange`。
- 替换旧内存筛选测试，实际调用 `queryPagedNoticeRows`，断言审批 CASE、收件人过滤、稳定排序、LIMIT/OFFSET 和参数。
- `noticeQuickCounts` 等其他旧助手未扩大清理；不删仍被测试使用的 `sameNoticeDay`。
- 主机规则原有 WIP 双重转义导致漏报；修正并覆盖 CSS、中文路径、白名单，已知主机不能通过 baseline 放行。
- 文件长度扫描改 NUL 分隔，中文路径不会被 Git 引号编码破坏；baseline 未改。
- .gitignore 新增 .run/.verify/.visual、verify-output/visual-output、test-results/playwright-report、screenshots 和 docs/PRD HTML。
- 新增暂存产物门禁，检查暂存新增/修改/复制/重命名；历史已提交证据不删除，不安装 hook。

## 操作降级表（仅待办页渲染）

| key | 原状 | 现状 |
|---|---|---|
| approve / withdraw_review / submit_review | enabled 描述会生成按钮，但待办页未加载 homereview | 本地渲染副本 enabled=false，保留 key/label/url |
| schedule | 会生成按钮，但待办页未加载 schedule-inline | 同上 |
| submit_test / accept / remind_accept / deliver | 会生成按钮，但待办页未加载对应事件 | 同上 |

reason：当前待办页尚未接入此操作，请前往工作台办理。
服务端权限描述不被修改，首页调用不传页面限制，所有原有入口保持原样。
待办页已加载的 clarify 以及 external 测试单/评价链接保留；服务端 disabled 不会被提升。
首页验收已追踪到 acceptance POST → Service → 状态写入；催办已追踪到 urge POST → Service → Repo 通知写入。
这里只证明代码链存在，未重发真实催办、未执行验收写入，不作数据库并发安全背书。

## 验证执行

- `GOCACHE="$PWD/tmp/gocache" go test ./internal/pkg/zentao ./internal/module/po ./internal/module/po/primaryaction`：exit 0。
- 新 `TestPagedNoticeRowsTreatsApprovalAsAnObjectFilter`：随 PO 包运行通过。
- `node tests/unit/frontend/primary-action-availability.test.js`：exit 0；覆盖八个 key、首页保留、澄清/外链保留和服务端拒绝。
- `bash scripts/test-quality-gates.sh`：exit 0，36 项；包含五个主机、CSS、中文模板、测试/Demo/vendor 白名单及强制 add 生成物。
- `make check`：exit 2；产物、gofmt、全量 Go、Makefile 枚举前端、vet、whitespace 通过，file-length 阻塞。
- `make check-patterns check-secrets check-architecture`：主机/硬模式、secret 通过；architecture exit 2（既有 testtask/handler_test.go 被扫描为 Handler DB 访问）。
- `git diff --check`：exit 0；暂存区为空。
- 浏览器：已登录 localhost:8090/todos 页面和布局可见，Console error/warn 查询为空；页面脚本 URL `v=1789010507` 的两个响应内容均与磁盘一致。
- 浏览器限制：当前首屏操作列都是 —，未取得真实 enabled 样本的降级交互证据；未完整采集网络错误。因此 UI 门禁为 partial，不以单测替代真实交互。

## 原有门禁阻塞与刻意未动

文件长度包括 Demo schedule.html 36944 行、mock-data.js 15299 行，以及正式 home.js 577、workboard.js 715、components.css 616、ui.js 819、metrics.css 683 等。
部分 schedule baseline 还存在已缩短未下调的历史条目。本轮没有修改这些正式文件或 baseline。
上述文件相对开工哈希未改变，属于本轮前已有问题；未把 Demo/vendor 加入豁免以掩盖失败。
保留开工时 config、zentao client、催办、搜索/评审修复的全部 WIP；此前核对未发现漂移，但最终核对出现并行修改，见下方补记。
下一刀建议：先按职责拆 home.js / workboard.js，再单独处理 Demo 巨石及剩余正式超限；架构扫描误报单独补例证修复，均不扩功能。

## 实际加载与 Git 证据

已读：AGENTS.md；docs/engineering 的 agent-onboarding、spec-index、architecture、database、frontend、testing、quality、module-index。
适用 MUST：限定 diff / 保护 WIP、分层与 schema 所有权、真实按钮合同、非增长门禁、实际执行测试与诚实报告。
不新增 SQL、DB 写入、schema 或锁行为；生产分页路径未改。
开工 status：15 个已跟踪文件修改，另有 .agents/、.cursor/plans/、.run/、.workbuddy/、docs/PRD/、service_home_urge_test.go、urge.js 未跟踪；暂存为空。
开工对照 Claude-PO：181 files changed, 16862 insertions(+), 2829 deletions(-)。
结束工作区对照（不含未跟踪新文件）：189 files changed, 16961 insertions(+), 2947 deletions(-)

本轮修改的已跟踪文件（相对开工哈希，不混入其他 WIP）：
- `web/static/js/po/urge.js`
- `.gitignore`
- `Makefile`
- `docs/Demo/README.md`
- `docs/engineering/quality.md`
- `internal/module/po/reponotice.go`
- `internal/module/po/reponotice_test.go`
- `internal/pkg/zentao/zentao.go`
- `scripts/check-file-length.sh`
- `scripts/check-patterns.sh`
- `scripts/test-quality-gates.sh`
- `web/static/js/po/primary-action.js`
- `web/static/js/po/todos.js`
- `web/static/js/schedule/scheduleintegrated.js`
- `web/static/js/schedule/scheduleintegratedrd.js`
- `web/templates/po/home.html`

本轮新文件：scripts/check-local-artifacts.sh、tests/unit/frontend/primary-action-availability.test.js、本记录。

结束 `git status -sb`（本记录写入前）：
```text
## release/po-integrate-main-202609...origin/release/po-integrate-main-202609
 M .gitignore
 M Makefile
 M docs/Demo/README.md
 M docs/engineering/quality.md
 M internal/config/loader.go
 M internal/config/loader_test.go
 M internal/module/po/form_home_actions.go
 M internal/module/po/handler.go
 M internal/module/po/handler_home_actions.go
 M internal/module/po/repo_home_actions.go
 M internal/module/po/reponotice.go
 M internal/module/po/reponotice_test.go
 M internal/module/po/service_home_actions.go
 M internal/pkg/zentao/client.go
 M internal/pkg/zentao/client_test.go
 M internal/pkg/zentao/zentao.go
 M scripts/check-file-length.sh
 M scripts/check-patterns.sh
 M scripts/test-quality-gates.sh
 M web/static/js/layout/globalsearch.js
 M web/static/js/po/demand-detail-review.js
 M web/static/js/po/homereview.js
 M web/static/js/po/primary-action.js
 M web/static/js/po/todos.js
 M web/static/js/schedule/scheduleintegrated.js
 M web/static/js/schedule/scheduleintegratedrd.js
 M web/templates/po/home.html
?? .agents/
?? .cursor/plans/
?? .workbuddy/
?? docs/PRD/
?? internal/module/po/service_home_urge_test.go
?? scripts/check-local-artifacts.sh
?? tests/unit/frontend/primary-action-availability.test.js
?? web/static/css/po/po-urge.css
?? web/static/js/po/urge.js
?? web/templates/components/po_urge_modal.html
```

## 最终并行修改补记

用户追问后的最终核对发现外部修改；上述完整门禁和浏览器证据只适用于此前快照，不认证最新整树。
本轮未修改、未覆盖这些并行路径：
- `web/static/js/po/urge.js`
- `web/static/js/schedule/scheduleintegrated.js`
- `web/static/js/schedule/scheduleintegratedrd.js`
- `web/templates/po/home.html`

新增的并行文件还包括 `web/static/css/po/po-urge.css` 和 `web/templates/components/po_urge_modal.html`。
停止这些路径的编辑和验收；需并行任务稳定后重新验收。先前“正式 schedule 文件未变”只适用于此前快照。
最终工作区 shortstat（包含并行 WIP，不含未跟踪文件）：189 files changed, 16961 insertions(+), 2947 deletions(-)

最终 status：
```text
## release/po-integrate-main-202609...origin/release/po-integrate-main-202609
 M .gitignore
 M Makefile
 M docs/Demo/README.md
 M docs/engineering/quality.md
 M internal/config/loader.go
 M internal/config/loader_test.go
 M internal/module/po/form_home_actions.go
 M internal/module/po/handler.go
 M internal/module/po/handler_home_actions.go
 M internal/module/po/repo_home_actions.go
 M internal/module/po/reponotice.go
 M internal/module/po/reponotice_test.go
 M internal/module/po/service_home_actions.go
 M internal/pkg/zentao/client.go
 M internal/pkg/zentao/client_test.go
 M internal/pkg/zentao/zentao.go
 M scripts/check-file-length.sh
 M scripts/check-patterns.sh
 M scripts/test-quality-gates.sh
 M web/static/js/layout/globalsearch.js
 M web/static/js/po/demand-detail-review.js
 M web/static/js/po/homereview.js
 M web/static/js/po/primary-action.js
 M web/static/js/po/todos.js
 M web/static/js/schedule/scheduleintegrated.js
 M web/static/js/schedule/scheduleintegratedrd.js
 M web/templates/po/home.html
?? .agents/
?? .cursor/plans/
?? .workbuddy/
?? docs/PRD/
?? docs/plan/po-consolidate-20260910/
?? internal/module/po/service_home_urge_test.go
?? scripts/check-local-artifacts.sh
?? tests/unit/frontend/primary-action-availability.test.js
?? web/static/css/po/po-urge.css
?? web/static/js/po/urge.js
?? web/templates/components/po_urge_modal.html
```
