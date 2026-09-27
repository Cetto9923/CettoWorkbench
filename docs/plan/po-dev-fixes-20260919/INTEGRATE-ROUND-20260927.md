# Integrate 轮次记录 · 2026-09-27

> 分支 `integrate/main-plus-explore-20260922`（worktree `~/GitHub/workbench-claude-po`）｜未 push｜Main worktree 未动
> 起点 `2b390319` → 终点 `71cddc5c`（功能轮止于 f5017b70，视觉轮 4 个提交）｜合入 Main `84ab76df`（= company/main tip，ls-remote 核实）
> 探索基线 `baseline/po-explore-a0cdcfe9`（只读对照）

## 1. 本轮提交

| SHA | 说明 |
|---|---|
| `fb049683` | merge: 合入 main (84ab76df)，含冲突裁决（见 §2） |
| `3414fdee` | feat: `GET /schedule/window-options`（复用 Main GetCreateWindowFormData）+ 搬入 picker 静态资源（修 /agileteam 404）+ 修 workboard-render 前端单测 |
| `61ebb1e7` | fix: ui.js 导出 `window.escapeHtml`（首页/待办/已办/敏捷小组/风险/指标等页面脚本初始化即报 `esc is not a function`，列表空白；上一轮 curl 冒烟未发现） |
| `e9ab82bf` | fix: PO 侧栏分组以当前路由为准，记忆分组仅兜底（原来直达 /issues/risk、/board/task 展开“需求规划”面板） |
| `a2e71968` | fix: 首页超期计算排除 0001-01-01 零日期（原“超期 106751 天”）+ 单测 |
| `f5017b70` | fix: /demands/:id 独立详情页显式覆盖 page_js（原继承兄弟模板 workboard 脚本报错）+ 文件头 |

每批后：`go build ./...`、`go vet ./...`、`go test ./...` 全绿；`tests/unit/frontend/*.test.js` 8/8 PASS。

## 2. Merge 冲突与裁决

| 文件 | 冲突 | 裁决 |
|---|---|---|
| internal/bootstrap/bootstrap.go | integrate 侧栏角标 provider + testtask 读写分库 vs Main `testtask.NewRepo(db)` | 保留角标 provider；testtask 跟 Main 只走主库（936caa88，提测不容忍备库延迟） |
| po/repodetail.go、servicedetail.go | integrate 已删除（DetailService 取代）vs Main 修改（bug #39365 BusinessReviewers） | 保持删除；#39365 语义改在 `review-candidates` 落地：候选人 = 需求池 businessReviewer |
| po/form.go、po/repo.go | integrate 拆分重构 vs Main 小改 | 保留 integrate 结构；Main 排期/交付独立研需口径（排除 closed、isParent=0、product!=0、指派人或产品 ReqM）同步到 repovaluestream.go，servicefollow_test 参数同步 |
| po/servicereview.go | 提交评审权限 | 回到 Main：仅创建人；文案“仅暂存或已驳回的需求可发起评审”（review-candidates 同口径） |
| layout/base.html、sidebar.html、po/home.html | PO 壳层 vs Main 侧栏收起首屏恢复脚本 / 去 AppName | 保留 PO 壳层 + 并入 Main 脚本；模板头去 `.AppName` |
| po/home_test.go | Main 删除 `config.App.Name` | 测试字面量去掉 Name |

## 3. 探索能力搬迁清单（a0 vs integrate）

| 能力 | 状态 | 备注 |
|---|---|---|
| /home 九阶段/焦点/KPI/抽屉/澄清/催办/交付/验收/主操作 | 已搬 | 本轮修 escapeHtml、超期零日期 |
| /demands/:id 独立详情页 + /detail JSON | 已搬 | 本轮修 page_js 继承 |
| /done 已办全链路 + 侧栏角标 | 已搬 | |
| /todos /notice /follow（含周报、导出） | 已搬 | |
| /board/demand /board/task /board/issues /board/group/metrics | 已搬 | 与 Main /kanban/* 并行（见问题 9） |
| /query /agileteam /metrics/* /profile /issues/risk(/issue-risk) | 已搬 | 本轮补 picker 静态资源 |
| 壳层（双栏侧栏、主题、常用固定页、底栏） | 已搬 | 本轮修分组记忆优先级 |
| /schedule/window-options（首页内联排期新建窗口） | 本轮已搬 | 数据源用 Main |
| /schedule 探索版 UI + 后端（defaults/precheck/batch/teamgroup/task_authorization/window_milestone） | 与 Main 冲突，未搬 | 排期写入跟 Main（问题 8） |
| 提测“联调总单”（testtask service_joint/gateway_joint） | 与 Main 冲突，未搬 | 问题 7 |
| button-system.css 全局按钮契约 | 未搬 | 会改 Main 后台页面观感，需确认 |
| permission/* 权限抽屉 JS/CSS、autocomplete-options.js | 未搬 | a0 模板无引用，属死代码 |
| middleware/superadmin.go、operationlog_schema.go、zentao client_do/client_site 拆分 | 未搬 | integrate 已有等价实现或非用户可见 |
| /debug/sqlperf | 未搬 | 调试工具 |

## 4. 8098 部署

- 镜像 `workbench:main-plus-explore-f5017b70`（linux/arm64，host 交叉编译 + alpine，与历史 8098 镜像同方式，含 configs/config.dev.yaml）
- 容器 `workbench-main-plus-explore-8098`，`-p 8098:8080`，`--restart unless-stopped`，环境变量复制自 8096（`WORKBENCH_MODE=dev`）
- 构建脚本：`/tmp/wb_build_deploy.sh`；冒烟：`/tmp/wb_smoke.sh`、`/tmp/wb_browser_smoke.py`
- 基础设施恢复（Docker Desktop 重启后）：OceanBase `obstandalone` 退出(255) 且 ob-ip-holder 抢占了 OB 配置的 172.17.0.3 → `docker stop obstandalone` → `docker restart ob-ip-holder`（回到 .2）→ `docker start obstandalone`（回到 .3），observer 恢复。未改数据、未改库表。8090/8096 随后按自身 restart 策略自动恢复（未重建、未改动）；8097 仍为 4 天前的 Exited 状态（未动）。

## 5. 冒烟结果（账号 003030，Chromium 无头，逐页新标签）

| 页面 | HTTP | Console 错误 | 4xx/5xx 请求 | 说明 |
|---|---|---|---|---|
| /home | 200 | 0 | 0 | 九阶段、待我处理列表、聚焦专区有真实数据 |
| /todos | 200 | 0 | 0 | |
| /done | 200 | 0 | 0 | |
| /notice | 200 | 0 | 0 | |
| /follow | 200 | 0 | 0 | 关注 1 条 |
| /schedule | 200 | 0 | 0 | Main 排期页 |
| /query | 200 | 0 | 0 | |
| /board/demand | 200 | 0 | 0 | |
| /board/task | 200 | 0 | 0 | 当前小组无任务（空态正常） |
| /agileteam | 200 | 0 | 0 | |
| /issues/risk | 200 | 0 | 0 | |
| /metrics/radar、/metrics/manage | 200 | 0 | 0 | |
| /profile | 200 | 0 | 0 | |
| /admin/roles、/admin/users | 200 | 0 | 0 | 后台旧壳层 |
| /kanban/story、/kanban/task | 200 | 0 | 0 | Main 看板仍可达 |
| /demands/63425 | 200 | 0 | 0 | |

JSON 接口抽检均 200：todos/done/notice/follow/board/query/issues-risk items、/demands、/demands/:id/detail|clarify|deliver|primary-action|review-candidates|testtask、/done/meta、/follow/project-weeklies、/schedule/windows|window-options|filter-counts|demands/:id/scheduling 等。写操作未做（避免推进真实业务）。

截图：Mac `~/GitHub/workbench-claude-po/tmp/wb_shots_20260927/`（含 browser_smoke.jsonl）；box `/workspace/workbench_round/`。

## 6. 过程事故（已修复）

执行中误用 `git checkout 2b390319 --`（本意取单个文件）导致 HEAD 游离，随后一个提交 `9b2129b3` 落在游离 HEAD 上（树里不含 merge）。已切回分支并 cherry-pick 为 `3414fdee`；`9b2129b3` 为悬空提交无引用，对应镜像 `workbench:main-plus-explore-9b2129b3` 可删除。分支历史未改写、未 reset。

## 7. 业务口径问题清单（待产品确认）

格式：页面/功能｜Main｜探索｜可疑漏洞｜当前运行｜建议

1. **首页业需可见范围（roleDemandBase，影响九阶段计数/待我处理）**｜Main：澄清 PM 精确等于、QD/RD/BRA，排除 parent=-1｜探索：PM 用 FIND_IN_SET（多人）、另加业需评审人/主管审批人/验收人，父需求按“存在有效子需求”排除｜Main 的 `PM = ?` 漏掉多 PM 记录；探索把只挂评审/验收身份的需求也算入各阶段，003030 首页“待我处理”3004 条明显偏大｜**已决策：PM 用 FIND_IN_SET，评审人/审批人仅进受理，验收人仅进验收｜已改：c82308a**
2. **提交评审权限**｜Main：仅创建人｜探索：创建人/指派人/超管｜Main 创建人离职/转岗后无人可提交；探索让指派人代提交，可能违背创建人意图｜**已决策：Main（仅创建人）+ 超管兜底｜已改：dd579d5**
3. **评审人候选来源**｜Main（bug #39365）：需求池 businessReviewer｜探索：全体内部用户｜需求池未配置 businessReviewer 时下拉为空、无法提交，两边都无提示｜**已决策：空态提示“请先在禅道需求池配置业务评审人”并置灰提交按钮｜已改：a0f3ef9**
4. **排期/交付阶段独立研发需求口径**｜Main：非需求池、非父、非关闭、（排期）product!=0、指派人或产品 ReqM｜探索：fromDemand 为空、仅指派人｜Main 未排除由业需转化的研需（fromDemand>0），可能与业需在首页重复计数；交付口径没有 product!=0，与排期不一致；探索漏掉产品 ReqM 视角｜**已决策：Main 口径 + 加 fromDemand=0（交付不补 product!=0）｜已改：c20d251**
5. **受理阶段排序**｜Main：待评审→已驳回→暂存，同状态 id 倒序｜integrate（探索）：当前账号待评的排前，其余 id 倒序｜两种“优先”含义不同｜**已决策：保持现状**
6. **首页“待我处理”与侧栏“我的待办”角标口径不一致**｜同一账号首页 3004、侧栏待办 73｜两套筛选规则并存，用户会困惑｜**已决策：改文案区分口径（首页：我参与阶段的需求，侧栏：需要我操作的事项）｜已改：9951b0a**
7. **提测单创建**｜Main：按产品分别建测试单｜探索：建一张联调总单｜影响禅道测试单数量与后续统计｜**已决策：保持现状**
8. **排期探索能力（批次默认值、保存前预检、任务授权、小组窗口里程碑、探索版排期 UI）**｜integrate 排期整体为 Main｜会改排期写入口径，属业务规则｜未搬｜**已决策：保持现状**
9. **/kanban 与 /board 并行**｜PO 壳层导航到 /board/*；后台旧壳层 DB 菜单仍指向 /kanban/*，两套取数口径不同｜同一“工作看板”两种结果｜两者都可访问｜**已决策：PO 入口统一 /board，编写 hide-kanban-menu.sql 待用户按需执行｜已改：9ae3aef**
10. **Main 漏洞：排期路由未绑定 RequirePerm**（违反核心底线 1）｜本轮新增 window-options 与同组保持一致，也未绑｜**已决策：核对发现 PO 角色（roleId=2）仅分配了 schedule:list，缺失 schedule:create/update/delete 权限码，按指示保持现状不改动路由，待分配权限后再补**
11. **零日期字面量**：todo_query_repo 用 `= '0000-00-00'`，仓库自己的注释说明 NO_ZERO_DATE 下会触发 Error 1525（目前 OB 未报错）｜**已决策：统一改用 dateUnsetExpr / dateSetExpr｜已改：013e11f**
12. **大体量下拉数据**：GET /demands/:id/clarify 约 856KB、deliver 约 468KB、testtask 约 300KB（全量人员/候选）｜**已决策：保持现状**
13. **侧栏分组记忆规则（UI）**：本轮改为路由优先，只有从“常用”进入的页面才保持“我的工作台”面板｜**已决策：保持现状**


## 8. 视觉验收与优化（19:08 追加范围）

基线：冻结探索 8096（只读）为主；docs/Demo 原型仅作宽松参考；不一致处按页面一致性/层级判断。
截图（仓库外）：Mac `/tmp/wb-visual/`：`before-8098/`（修前）、`ref-8096/`（基线）、`after-8098/`（首轮修后）、`final-8098/`（最终 71cddc5c）；`main-8090/schedule.png`（Main 自身排期页对照）。box 副本 `/workspace/wb-visual/`（cmp*/ 为左 8098、右 8096 并排图）。
最终版 19 页：全部 200，console 错误 0，4xx/5xx 静态资源 0。

### 视觉提交
| SHA | 内容 |
|---|---|
| `37f9761a` | 外壳/主题：variables.css 三方合入探索 PO 语义 token（类型色/阶段色/badge/info 描边）；layout/app/base/components/pager/form/autocomplete 三方合并；PO 页标题式顶栏；底栏 tabs+actions；button-system.css 仅 PO 页加载；pinned-pages.js 先于 sidebar.js；移除 body 末尾误渲染的主题选择器；base.html 静态资源统一 asset() 版本号；ui.js 导出 initUserPicker/destroyUserPicker/getCsrfToken；app.js 新增 appJson |
| `13ab5be7` | auth.go 挂 `perm.WithGranted`（Main 缺失导致非超管所有 PO 主操作置灰；同一份 userPerms 快照，未新增规则） |
| `07dfbee0` | 排期页（PO 外壳）列宽/重复标题；/demands/:id 卡片化（po/demand-page.css） |
| `71cddc5c` | 首页抽屉关闭态右缘投影；已办“查看记录”按钮溢出 |
| `63f9f7c` | 排期页独立研发需求表取消 min-width 与内联宽度限制（贴合 PO 外壳） |
| `9f28099` | 后台侧栏移除非 PO 外壳多余品牌条与重复收起按钮（修 profile/admin 错位） |
| `6e28096` | 人员选择器增量扩展 mode:"user"（头像/部门/拼音/分组标题） |

### 视觉问题表
| # | 页面 | 问题 | 严重度 | 原因 | 状态 | 依据 |
|---|---|---|---|---|---|---|
| V1 | 全部 PO 页 | 类型标签（业需/Bug）白底无色、主操作按钮变成文字链、Tab/分段控件无选中色 | 高 | integrate 用 Main 的 variables.css，缺探索 PO token；button-system.css 未加载 | 已修 37f9761a | 8096 |
| V2 | 全部 PO 页 | 顶栏显示“研 研发工作台”面包屑，无页面标题/说明/通知铃/固定按钮 | 高 | header.html 用 Main 版本 | 已修 37f9761a（搜索框文案/行为保持 Main “编号搜索”） | 8096 + Main |
| V3 | 全部 PO 页 | 侧栏“常用”列表为空 | 中 | sidebar.js 早于 pinned-pages.js 加载 | 已修 | 8096 |
| V4 | 全部 PO 页 | 底部页签缺“关闭其他页签”操作区 | 低 | base.html 底栏结构为 Main 版 | 已修 | 8096 |
| V5 | 首页 | “澄清/排期”等主操作全部灰色虚线 | 高 | auth 未挂权限快照（见 13ab5be7） | 已修 | 8096 |
| V6 | 首页 | 页面右缘一条灰色阴影 | 低 | 屏外抽屉 box-shadow 露出（8096 同样存在） | 已修 | 自判 |
| V7 | 已办 | “查看记录”按钮超出操作列 | 低 | 按钮契约覆盖 small 按钮尺寸（8096 同样） | 已修 | 自判 |
| V8 | 排期 | 操作列被挤出可视区；敏捷小组折 3-4 行；页内标题与顶栏重复 | 中 | Main 模板内联 500px 标题列 + table min-width 1180 | 已修 07dfbee0（业务表）、63f9f7c（独立研需表） | Main 8090 同病，自判 |
| V9 | /demands/:id | 裸文本无样式、嵌套 main | 中 | 模板引用未加载的 home-page-card | 已修（卡片化） | 自判（8096 同样裸） |
| V10 | 看板弹窗/澄清/提交评审/提测 | 人员选择器不初始化、看板弹窗“页面请求能力未加载” | 高（功能性） | ui.js/app.js 缺 initUserPicker/appJson | 已修（人员选择器暂为通用 autocomplete，无拼音/分组标题） | 8096 |
| V11 | /profile、/admin/*、/kanban/* | 仍是 Main 后台外壳（品牌区 + “个人入口/工作区”菜单），与 PO 外壳两套观感 | 中 | 这些路由不在 PO 外壳路由表 | 错位已修 9f28099；归属待决定 | 8096 同样 |
| V12 | 排期 | 探索版排期 UI（窗口卡片、快捷筛选样式等）与 Main 差异大 | 中 | 排期保留 Main（业务冲突） | 未修 | 问题 8 |
| V13 | 人员选择器 | 缺探索版 user 模式（头像/部门/拼音检索） | 低 | ui.js 保留 Main autocomplete | 已修 6e28096 | 8096 |
| V14 | 深色主题 | variables.css 已带 dark token，但头像菜单只提供浅色（Main 决策） | 低 | — | 保持 Main | Main |
| V15 | 全局 | `/kanban/story` 标题含 `&quot;` 实体未解码（数据侧转义） | 低 | 禅道数据双重转义 | 已修 09c97b3 | 自判 |

### 需用户决定
- V11：/profile、/admin/*、/kanban/* 是否纳入 PO 外壳。
- 13ab5be7 让非超管按角色权限看到可点的主操作，属“恢复设计行为”；若希望暂保持全部禁用可单独回退该提交。
