# Demo 原型功能清单分析

> 分析来源：`demo/personal-workbench.html`、`demo/personal-workbench.app.js`（约 15781 行，780 个 function）、`demo/personal-workbench.css`（3499 行）、`demo/business-demand-detail.js`（51 个 function）、`demo/_top5_snippet.js`（废弃片段，仅 1 行 motion 标签，可忽略）。
> 对照基准：当前 Go 单体项目 `web/templates/po/home.html`、`web/templates/schedule/index.html` 及对应 handler。
> Demo 中 `navTo()` MVP 限制：**仅 `home` 与 `schedule` 可导航**，其余页面 toast「本期不提供」。

## 一、页面清单

| 页面 ID | 中文名称 | 导航 data-page | UI 区块 | JS 渲染入口 |
|---------|----------|----------------|---------|-------------|
| `page-home` | 工作台首页 | `home` | `po-home-head / homeKpiGrid`：问候语 + KPI 数字卡片网格<br>`home-focal-section`：今日推进焦点（demo 标注为待推出占位）<br>`homeValueStreamSection / homeVsCompact`：业务需求价值流紧凑卡片条，可点击筛选<br>`homeBlockerSection / homeBlockerStrip`：卡点快速响应横向卡片条<br>`top5Section / top5List`：当前应推进事项 Top 列表<br>`home-risk-panel / homeRiskAnalysisList`：卡点与风险分析侧栏<br>`homeVersionCollapsible`：版本窗口 Dock：当前+后续 3 窗口概览与指标条<br>`homeDataInfo / lastUpdateTime`：页脚数据更新时间/来源 | `renderPOHomeSections()` |
| `page-todos` | 我的待办 | `todos` | `todoSubTabs`：子 Tab：今日/超期/阻塞等动作筛选<br>`todoFullBody`：待办全量表格（ID/标题/类型/状态/优先级/截止/来源/下一步/操作） | `renderFullTodos()` |
| `page-riskRadar` | 指标雷达 | `riskRadar` | `riskRadarFullContent`：风险维度卡片 + 钻取入口 | `renderRiskRadarFull()` |
| `page-qualityWarning` | 质效预警 | `qualityWarning` | `qualityWarningContent`：预警分类摘要区<br>`qualityWarningList`：预警明细列表 | `renderQualityWarning()` |
| `page-demandWorkspace` | 需求综合查询 | `demandWorkspace` | `demandRolePresets`：列预设：默认/业务/研发/负责人<br>`dw-common-filters`：快捷查询 chip：未关闭/我相关/有卡点等<br>`demandPrimaryAdvancedBar`：主筛选条 + 高级筛选 + 视图切换 + 导出/统计<br>`demandAdvancedFilters`：高级筛选折叠区（类别/重要度/日期等）<br>`demandSavedQueries / demandDimensionSummary`：已保存查询与维度摘要<br>`demandAnalyticsPanel`：统计分析面板<br>`dwSplitShell`：列表/看板/分组主区 + 右侧业务需求详情 aside<br>`demandListView / demandKanbanView / demandGroupView`：三种视图容器<br>`bdDetailDrawerOverlay / dwBizDetailAside`：业务需求聚合详情侧栏 | `renderDemandListView() / renderDemandKanbanView() / renderDemandGroupView()` |
| `page-query` | 需求综合查询(旧) | `query` | `savedViews`：保存视图 pill<br>`queryAgileMulti 等`：多选筛选器组<br>`queryTable / queryBody`：需求表格 + 分页<br>`schedule-batch`：批量进入排期 | `renderQuery()` |
| `page-kanban` | 需求看板(旧) | `kanban` | `kanbanFilterAgile / kanbanViewSelect`：小组与视图维度选择<br>`kanbanBoard`：看板列与卡片 | `renderKanban()` |
| `page-schedule` | 排期工作台 | `schedule` | `schedule-page-head`：标题 + 窗口维护/新建/刷新<br>`schedulePoSummaryBar / scheduleCapacitySummaryLine`：版本窗口概览与容量摘要<br>`schedule-quick-scope`：快捷筛选 chip（未关闭/待排期/风险等）<br>`schedule-filter--primary / scheduleMoreFilters`：主筛选 + 更多筛选<br>`scheduleAlert`：风险告警条<br>`schedule-data-tabs`：业务需求 / 独立研发需求 Tab<br>`schedule-table / scheduleBody`：树形排期列表<br>`schedulePagination`：分页<br>`schedule-batch`：批量改窗口/校验<br>`scheduleKanbanViewWrap / schedulePlannerKanban`：排期看板（demo 列表 Tab 标注待推出，容器已实现）<br>`scheduleGroupViewWrap / scheduleGroupView`：按人员/窗口/优先级分组视图 | `renderSchedule()` |
| `page-mr` | MR/门禁与例外 | `mr` | `mrSubTabs`：MR待评审/门禁失败/例外申请/例外审批 Tab<br>`mrContent`：Tab 内容区 | `renderMr('review')` |
| `page-productWorkspace` | 系统工作区 | `productWorkspace` | `filter-bar`：系统范围/角色/状态/关键词筛选<br>`productListContainer`：系统卡片列表<br>`productPagination`：分页 | `renderProductList()` |
| `page-quickLinks` | 快捷入口 | `quickLinks` | `quickLinksContainer`：分组快捷链接卡片 | `renderQuickLinks()` |
| `page-watch` | 我关注的事项 | `watch` | `watchPageBody`：关注项列表 | `renderWatchPage()` |
| `page-versionFollow` | 版本跟进 | `versionFollow` | `vfVersionSwitch`：版本窗口切换条 + 历史窗口<br>`vfHealthBanner`：健康度 Banner<br>`vfProgressStrip / vfUnifiedBar`：阶段进度 + 统一工具栏<br>`vfFilterMore`：更多筛选面板<br>`vfTabBody`：概览/明细 Tab 内容 | `renderVersionFollow()` |
| `page-issueRisk` | 问题风险 | `issueRisk` | `issueRiskBody`：状态卡片 + Tab + 筛选 + 表格 | `renderIssueRisk()` |
| `page-demandDetail` | 需求详情(全页) | `demandDetail` | `demandFullDetailBody`：复用 business-demand-detail 渲染 | `renderDemandFullDetailPage()` |
| `page-notifications` | 通知中心 | `notifications` | `noticePageBody`：通知列表与筛选 | `renderNotifications()` |
| `page-admin` | 后台管理 | `admin` | `adminPageBody`：用户视图权限管理 | `renderAdminPage()` |

**说明：** `page-query` 与 `page-kanban` 为旧版需求查询/看板，已被 `page-demandWorkspace` 的三视图取代，但 HTML 仍保留。

## 二、全局组件/弹窗清单

| 组件 ID | 用途 | 触发方式 | JS 函数 |
|---------|------|----------|---------|
| `loginModal` | PO 禅道登录 | 未登录点击用户信息 / authLogin 流程 | `showLoginModal / hideLoginModal / handleLoginSubmit (HTML inline)` |
| `drawer + drawerOverlay` | PO 需求详情 Drawer | viewDetail(id) / 表格行详情 | `openDrawer / closeDrawer / renderPODrawerDetail` |
| `batchModal` | 批量操作弹窗 | 排期页「批量改窗口」等 | `openBatchModal / closeBatchModal` |
| `scheduleIntegratedModal` | 排期一体化办理 | 排期行「一体化」按钮 | `openScheduleIntegratedModal / saveScheduleIntegratedModal / closeScheduleIntegratedModal` |
| `scheduleRowModal` | 逐条排期 | 行内排期/详情页跳转 | `openScheduleRowModal / saveScheduleRowModal / closeScheduleRowModal` |
| `globalDemandDetailOverlay + globalDemandDetailAside` | 跨页业务需求聚合详情 | 任意页 openGlobalBusinessDemandDetail | `openGlobalBusinessDemandDetail / closeGlobalBusinessDemandDetail / renderBusinessDemandDetailPage` |
| `scheduleVersionWindowModal` | 版本窗口属性编辑 | 窗口卡片菜单/维护 | `openScheduleVersionWindowEditModal / saveScheduleVersionWindowModal` |
| `taskBuildModal` | 拆任务 | 研发需求行拆任务 | `openTaskBuildModal / saveTaskBuildModal / closeTaskBuildModal` |
| `transferDevModal` | 转研发需求 | 业务需求转研发 | `openTransferDevModal / saveTransferDevModal / closeTransferDevModal` |
| `demandDimModal` | 需求维度多选 Picker | 需求工作区阶段/状态/系统/团队/小组按钮 | `openDemandDimPicker / confirmDemandDimPicker / closeDemandDimPicker` |
| `columnModal` | 自定义列 | 需求/排期列表齿轮按钮 | `openColumnConfigModal / saveColumnConfig / closeColumnConfigModal` |
| `exceptionModal + exceptionForm` | 门禁例外申请 | MR 页例外 Tab | `openExceptionModal / closeExceptionModal` |
| `deliveryModal + deliveryForm` | 发起交付 | Drawer 内交付动作 | `openDeliveryModal / confirmDelivery / closeDeliveryModal` |
| `manageVersionWindowsModal` | 版本窗口管理列表 | 排期页「窗口维护」 | `openManageVersionWindowsModal / renderManageVersionWindowsModal` |
| `configDrawer + configOverlay` | 工作台配置 | 侧栏「工作台配置」 | `openConfig / closeConfig / switchConfigTab / saveWorkbenchConfig` |
| `toast (#toast)` | 全局 Toast | showToast(msg) 任意触发 | `showToast` |
| `demandDetailDrawer` | 需求详情 Drawer(旧) | 部分列表行 | `closeDemandDetail / showDemandDetail` |
| `actionDrawer` | 处理需求 Drawer | 待办/列表处理按钮 | `openActionDrawer / closeActionDrawer / submitAction` |
| `bdDetailDrawerOverlay` | 需求工作区内嵌详情遮罩 | 需求工作区行点击 | `closeBusinessDemandDetail / openBusinessDemandDetail` |

## 三、数据模型清单

| 变量名 | 类型 | 主要字段 | 使用页面/函数 |
|--------|------|----------|---------------|
| `scheduleData` | 排期核心数组 | id, title, agile, iterationPlan, system, role, pri, status, dev/test/accept, targetDate, reviewDate, windowType, blocked, overdue, suspended, anomaly, scheduleStage, devRequirements[], subBizRequirements[], systemPersonnel[], createdTasks[] 等 | page-schedule, page-home(版本窗口), versionFollow, business-demand-detail |
| `DEMAND_TREE_DATA` | 需求工作区树形数据 | id, reqCode, nodeType(biz/subBiz/rd), parentId, title, agile, team, phase, priority, deadline, status, actionType, riskType, systemRole, systems[] | page-demandWorkspace 列表/看板/分组 |
| `vfPrimaryVersions / vfHistoryVersions` | 版本跟进窗口元数据 | id, name, range, status, capacity 等 | page-versionFollow |
| `poReqData` | PO 首页需求池 | 禅道业务需求归一化字段 + lane/actionType/riskTags | page-home Top5/价值流/卡点 |
| `drawerData` | Drawer 详情字典 | 按 ID 索引：summary/lifecycle/blockers/collaboration/schedule/fields/tl | openDrawer, PO Flow |
| `demandRelationData` | 需求关联关系 | 主配系统、研发需求分组 | drawerData 关联区 |
| `BUSINESS_DEMAND_DETAILS (bd.js)` | 业务需求 BR 聚合 mock | summary, deliveryUnits[], risks[], milestones[], people[] | business-demand-detail.js |
| `RISK_RADAR_DATA` | 指标雷达 mock | 各风险维度 count/level/items | page-riskRadar |
| `QUALITY_WARNING_DATA` | 质效预警 mock | sectionKey → 预警项 | page-qualityWarning |
| `productData` | 系统工作区 | id, name, scope, status, stats{}, owner, team, myRole, followed | page-productWorkspace |
| `quickLinksData` | 快捷入口分组 | group → items[]{label,url,type} | page-quickLinks |
| `queryData / kanbanData` | 旧查询/看板 mock | 需求行字段 | page-query, page-kanban |
| `poTodos / devTodos / smTodos / allTodos` | 待办 mock | id, title, type, filter tags, sort | page-todos |
| `noticeData` | 通知 mock | type, title, time, read, link | page-notifications |
| `watchItems` | 关注项 mock | type, title, id, updated | page-watch |
| `issueRiskItems / issueRiskAgileMap` | 问题风险 mock | kind, status, agile, team, priority, due | page-issueRisk |
| `testAcceptanceItems` | 版本跟进测试验收 | cases, bugs, passRate 等 | versionFollow 测试 Tab |
| `searchData` | 全局搜索索引 | id, title, type, url | handleGlobalSearch |
| `schedulePoolData / collabData / releaseData / riskData` | PO Flow 子模块 mock | 排期池/协作/发布/风险卡片 | renderPOScheduleArea 等（首页旧模块） |
| `scheduleIterationDefinitions / scheduleFilters / schedulePaginationState` | 排期状态 | 筛选条件、分页、窗口定义 | renderSchedule 全系 |
| `columnConfigDefaults` | 列配置默认值 | demand/schedule 可见列集合 | openColumnConfigModal |
| `VALUE_STREAM_STAGES / STAGE_GUIDANCE / HOME_VS_STAGE_ORDER` | 价值流阶段常量 | 阶段 key/label/icon/guidance | page-home 价值流 |
| `VC / WORKBENCH_ACCESS_DEFAULT` | 视图权限配置 | allowedViews per user | renderAdminPage, navTo MVP 限制 |
| `currentUser / poCache / poDetailCache` | 运行时状态 | 登录用户、API 缓存 | authLogin, openDrawer API 拉取 |

## 四、核心 JS 函数清单

`personal-workbench.app.js` 共 **780** 个 `function` 声明；`business-demand-detail.js` 共 **51** 个。以下按页面/模块分组（同一模块内按字母序）。

### 首页 (page-home)（41）

| 函数名 | 功能说明 |
|--------|----------|
| `clearHomeAllFilters` | clearHomeAllFilters |
| `clearHomeValueStreamStage` | clearHomeValueStreamStage |
| `getCurrentPoUserLabel` | 获取current po user label |
| `getHomeValueStageRows` | 获取home value stage rows |
| `getHomeValueStageStats` | 获取home value stage stats |
| `getHomeValueStageStayDays` | 获取home value stage stay days |
| `getTop3Recommendations` | 获取top3 recommendations |
| `normalizeHomeReason` | normalizeHomeReason |
| `poIsDateReached` | poIsDateReached |
| `poParseIsoDate` | poParseIsoDate |
| `poPersonMatchesMe` | poPersonMatchesMe |
| `poReqPassesHomeFocal` | poReqPassesHomeFocal |
| `poTodayIso` | poTodayIso |
| `renderBlockerSection` | 渲染blocker section区块/列表 |
| `renderBlockerTagHtml` | 渲染blocker tag html区块/列表 |
| `renderDemandOverview` | 渲染demand overview区块/列表 |
| `renderHomeBlockerStrip` | 渲染home blocker strip区块/列表 |
| `renderHomeCurrentVersionSummary` | 渲染home current version summary区块/列表 |
| `renderHomeFocalActiveBar` | 渲染home focal active bar区块/列表 |
| `renderHomeKpis` | 渲染home kpis区块/列表 |
| `renderHomeMetricsStrip` | 渲染home metrics strip区块/列表 |
| `renderHomeRiskAnalysis` | 渲染home risk analysis区块/列表 |
| `renderHomeVsCompact` | 渲染home vs compact区块/列表 |
| `renderHomeVsCompactSummary` | 渲染home vs compact summary区块/列表 |
| `renderPOActionHub` | 渲染p o action hub区块/列表 |
| `renderPOHomeSections` | 首页主入口：KPI/价值流/Top5/版本窗口等区块 |
| `renderPORisk` | 渲染p o risk区块/列表 |
| `renderProductOverview` | 渲染product overview区块/列表 |
| `renderTodayRecommendations` | 渲染today recommendations区块/列表 |
| `renderTop5` | 渲染top5区块/列表 |
| `renderValueStreamOverview` | 渲染value stream overview区块/列表 |
| `renderValueStreamProgress` | 渲染value stream progress区块/列表 |
| `rowToTop5Item` | rowToTop5Item |
| `selectValueStage` | selectValueStage |
| `setHomeFocalFilter` | 设置home focal filter |
| `toggleHomeValueStreamExpanded` | 切换home value stream expanded |
| `toggleHomeVersionPanel` | 切换home version panel |
| `toggleHomeVsStagePills` | 切换home vs stage pills |
| `updateHomeFocalChipCounts` | 更新home focal chip counts |
| `updateHomeFocalChipsActive` | 更新home focal chips active |
| `updateLastUpdateTime` | 更新last update time |

### 指标雷达 (page-riskRadar)（6）

| 函数名 | 功能说明 |
|--------|----------|
| `getRiskStatus` | 获取risk status |
| `openRiskDetail` | 打开risk detail |
| `renderRiskCard` | 渲染risk card区块/列表 |
| `renderRiskCardFull` | 渲染risk card full区块/列表 |
| `renderRiskRadar` | 渲染risk radar区块/列表 |
| `renderRiskRadarFull` | 渲染risk radar full区块/列表 |

### 质效预警 (page-qualityWarning)（3）

| 函数名 | 功能说明 |
|--------|----------|
| `buildQualityWarningRows` | 构建quality warning rows |
| `openQualityWarningList` | 打开quality warning list |
| `renderQualityWarning` | 渲染quality warning区块/列表 |

### 我的待办 (page-todos)（22）

| 函数名 | 功能说明 |
|--------|----------|
| `applyTodoFilter` | 应用todo filter |
| `buildPoTodoRows` | 构建po todo rows |
| `buildPoTodosFromPoReqData` | 构建po todos from po req data |
| `countPoTodoRecords` | countPoTodoRecords |
| `ensurePoTodoParentsExpandedForFilter` | ensurePoTodoParentsExpandedForFilter |
| `handlePOAction` | handlePOAction |
| `openPOActionDrawer` | 打开p o action drawer |
| `passesPoTodoCore` | passesPoTodoCore |
| `passesPoTodoItemFilters` | passesPoTodoItemFilters |
| `passesPoTodoItemOrChildVisible` | passesPoTodoItemOrChildVisible |
| `poTodoNextPage` | poTodoNextPage |
| `poTodoPrevPage` | poTodoPrevPage |
| `renderFullTodos` | 渲染full todos区块/列表 |
| `renderPoTodoList` | 渲染po todo list区块/列表 |
| `resetPoTodoFilters` | resetPoTodoFilters |
| `setPoTodoActionFilter` | 设置po todo action filter |
| `setPoTodoTypeFilter` | 设置po todo type filter |
| `setPoTodoTypeSelect` | 设置po todo type select |
| `setPoTodoValueStreamFilter` | 设置po todo value stream filter |
| `setPoTodoValueStreamSelect` | 设置po todo value stream select |
| `todoActionHint` | todoActionHint |
| `togglePoTodoTreeExpand` | 切换po todo tree expand |

### 需求综合查询 (page-demandWorkspace)（76）

| 函数名 | 功能说明 |
|--------|----------|
| `applyDemandCommonFilter` | 应用demand common filter |
| `applyDemandFilterByType` | 应用demand filter by type |
| `applyDemandFilters` | 应用demand filters |
| `applyDemandRolePreset` | 应用demand role preset |
| `clearDemandDimPickerSelection` | clearDemandDimPickerSelection |
| `clearDemandFilters` | clearDemandFilters |
| `closeDemandDimPicker` | 关闭demand dim picker |
| `collectNodeWithDescendants` | collectNodeWithDescendants |
| `confirmDemandDimPicker` | confirmDemandDimPicker |
| `countDemandBy` | countDemandBy |
| `demandPageNext` | demandPageNext |
| `demandPagePrev` | demandPagePrev |
| `demandTreeItemMatchesBarFilters` | demandTreeItemMatchesBarFilters |
| `ensureDefaultDemandExpansion` | ensureDefaultDemandExpansion |
| `exportDemandWorkspace` | exportDemandWorkspace |
| `getDemandAgileOptions` | 获取demand agile options |
| `getDemandCurrentUser` | 获取demand current user |
| `getDemandDimPickerConfig` | 获取demand dim picker config |
| `getDemandFilteredFlatRows` | 获取demand filtered flat rows |
| `getDemandItemAcceptOwner` | 获取demand item accept owner |
| `getDemandItemAssignee` | 获取demand item assignee |
| `getDemandItemBlockerInfo` | 获取demand item blocker info |
| `getDemandItemCategoryLabel` | 获取demand item category label |
| `getDemandItemChangeLabel` | 获取demand item change label |
| `getDemandItemCreatedDate` | 获取demand item created date |
| `getDemandItemDeadline` | 获取demand item deadline |
| `getDemandItemDeliveryDate` | 获取demand item delivery date |
| `getDemandItemDepartment` | 获取demand item department |
| `getDemandItemDevTeam` | 获取demand item dev team |
| `getDemandItemEstimateLaunch` | 获取demand item estimate launch |
| `getDemandItemFocusLabel` | 获取demand item focus label |
| `getDemandItemHangLabel` | 获取demand item hang label |
| `getDemandItemIteration` | 获取demand item iteration |
| `getDemandItemMainSystem` | 获取demand item main system |
| `getDemandItemManagerReviewLabel` | 获取demand item manager review label |
| `getDemandItemOwner` | 获取demand item owner |
| `getDemandItemProposer` | 获取demand item proposer |
| `getDemandItemRelation` | 获取demand item relation |
| `getDemandItemReviewer` | 获取demand item reviewer |
| `getDemandItemSchedulePlanDate` | 获取demand item schedule plan date |
| `getDemandItemSeverityLabel` | 获取demand item severity label |
| `getDemandItemSourceLabel` | 获取demand item source label |
| `getDemandItemTester` | 获取demand item tester |
| `getDemandItemValueStage` | 获取demand item value stage |
| `getDemandItemWindowType` | 获取demand item window type |
| `getDemandItemZentaoStatus` | 获取demand item zentao status |
| `getDemandNodeMeta` | 获取demand node meta |
| `getDemandSystemOptions` | 获取demand system options |
| `getDemandTeamOptions` | 获取demand team options |
| `goDemandWorkspaceWithContext` | goDemandWorkspaceWithContext |
| `initDemandDefaultFilters` | initDemandDefaultFilters |
| `openDemandDimPicker` | 打开demand dim picker |
| `openDemandFullPage` | 打开demand full page |
| `poReqToKanbanCol` | poReqToKanbanCol |
| `renderDemandAnalyticsPanel` | 渲染demand analytics panel区块/列表 |
| `renderDemandDimPickerOptions` | 渲染demand dim picker options区块/列表 |
| `renderDemandDimensionSummary` | 渲染demand dimension summary区块/列表 |
| `renderDemandFilterSummary` | 渲染demand filter summary区块/列表 |
| `renderDemandGroupView` | 渲染demand group view区块/列表 |
| `renderDemandHeaderCounts` | 渲染demand header counts区块/列表 |
| `renderDemandKanbanView` | 渲染demand kanban view区块/列表 |
| `renderDemandListView` | 需求工作区列表视图（树形表格） |
| `renderDemandSavedQueries` | 渲染demand saved queries区块/列表 |
| `renderDevDemandGroupsSection` | 渲染dev demand groups section区块/列表 |
| `resetDemandFiltersToDefault` | resetDemandFiltersToDefault |
| `setDemandActionFilter` | 设置demand action filter |
| `switchDemandView` | switchDemandView |
| `syncDemandActionChips` | 同步demand action chips |
| `syncDemandTreeDevTeamsFromSchedule` | 同步demand tree dev teams from schedule |
| `toggleDemandAdvancedFilters` | 切换demand advanced filters |
| `toggleDemandAnalyticsPanel` | 切换demand analytics panel |
| `toggleDemandDimPickerOption` | 切换demand dim picker option |
| `toggleDemandExpand` | 切换demand expand |
| `toggleGroupSection` | 切换group section |
| `treeItemToKanbanCol` | treeItemToKanbanCol |
| `updateDemandPageInfo` | 更新demand page info |

### 需求查询/看板旧页 (page-query / page-kanban)（13）

| 函数名 | 功能说明 |
|--------|----------|
| `applySavedView` | 应用saved view |
| `onQueryAgileMultiChange` | 处理query agile multi change事件 |
| `onQueryDeptMultiChange` | 处理query dept multi change事件 |
| `onQueryImportanceMultiChange` | 处理query importance multi change事件 |
| `onQueryPoolMultiChange` | 处理query pool multi change事件 |
| `onQueryPriMultiChange` | 处理query pri multi change事件 |
| `onQueryStatusMultiChange` | 处理query status multi change事件 |
| `onQuerySystemMultiChange` | 处理query system multi change事件 |
| `renderKanban` | 渲染kanban区块/列表 |
| `renderLifecycleKanban` | 渲染lifecycle kanban区块/列表 |
| `renderQuery` | 渲染query区块/列表 |
| `switchKanbanView` | switchKanbanView |
| `toggleKanbanCard` | 切换kanban card |

### 排期工作台 (page-schedule)（309）

| 函数名 | 功能说明 |
|--------|----------|
| `addDaysIso` | addDaysIso |
| `addIntegratedTransferDevDraftRow` | addIntegratedTransferDevDraftRow |
| `addScheduleClarifiedSystem` | addScheduleClarifiedSystem |
| `addTransferDevDraftRow` | addTransferDevDraftRow |
| `applyScheduleCreateOrgTemplateDates` | 应用schedule create org template dates |
| `applyScheduleNavigationContext` | 应用schedule navigation context |
| `applyScheduleWindowMilestonesToItem` | 应用schedule window milestones to item |
| `buildDefaultScheduleDevRequirements` | 构建default schedule dev requirements |
| `buildDefaultScheduleWindowNameFromOnline` | 构建default schedule window name from online |
| `buildDemoTasksForRd` | 构建demo tasks for rd |
| `buildScheduleUrgeMessage` | 构建schedule urge message |
| `buildScheduleVersionWindowOptionMarkup` | 构建schedule version window option markup |
| `buildTransferDevDraftRows` | 构建transfer dev draft rows |
| `buildTransferStoryRows` | 构建transfer story rows |
| `calculateReviewDate` | calculateReviewDate |
| `calculateWindowType` | calculateWindowType |
| `changeScheduleModalAgile` | changeScheduleModalAgile |
| `clearScheduleFilters` | clearScheduleFilters |
| `closeBatchModal` | 关闭batch modal |
| `closeManageVersionWindowsModal` | 关闭manage version windows modal |
| `closeScheduleIntegratedModal` | 关闭schedule integrated modal |
| `closeScheduleRowModal` | 关闭schedule row modal |
| `closeScheduleVersionWindowModal` | 关闭schedule version window modal |
| `closeTaskBuildModal` | 关闭task build modal |
| `closeTransferDevModal` | 关闭transfer dev modal |
| `createIntegratedTaskDrafts` | createIntegratedTaskDrafts |
| `createScheduleVersionCreateDraft` | createScheduleVersionCreateDraft |
| `createTaskBuildExecutionQuick` | createTaskBuildExecutionQuick |
| `createTaskDraftFromSchedule` | createTaskDraftFromSchedule |
| `defaultScheduleSystemAnalyst` | defaultScheduleSystemAnalyst |
| `deleteManageVersionWindow` | deleteManageVersionWindow |
| `deriveBizStatusFromText` | deriveBizStatusFromText |
| `draftRowToDevRequirement` | draftRowToDevRequirement |
| `editManageVersionWindow` | editManageVersionWindow |
| `ensureIntegratedRdDemoTasks` | ensureIntegratedRdDemoTasks |
| `ensureIntegratedRdDrafts` | ensureIntegratedRdDrafts |
| `ensureIntegratedTaskDrafts` | ensureIntegratedTaskDrafts |
| `ensureRdTaskBuildBrief` | ensureRdTaskBuildBrief |
| `ensureScheduleCreateSystemPlanDraft` | ensureScheduleCreateSystemPlanDraft |
| `ensureScheduleIterationEditable` | ensureScheduleIterationEditable |
| `ensureScheduleModalSystemPlanDraft` | ensureScheduleModalSystemPlanDraft |
| `ensureScheduleSystemPersonnel` | ensureScheduleSystemPersonnel |
| `ensureTransferDevDraft` | ensureTransferDevDraft |
| `escapeAttr` | escapeAttr |
| `filterScheduleMainSub` | 筛选schedule main sub |
| `filterSchedulePool` | 筛选schedule pool |
| `filterScheduleTable` | 筛选schedule table |
| `findScheduleSystemPlanForWindow` | findScheduleSystemPlanForWindow |
| `findScheduleTaskTarget` | findScheduleTaskTarget |
| `formatBizDevTeamsLabel` | formatBizDevTeamsLabel |
| `formatTaskOwnerNames` | formatTaskOwnerNames |
| `getBizScheduleFields` | 获取biz schedule fields |
| `getFilteredScheduleData` | 获取filtered schedule data |
| `getIterationDeliverySummary` | 获取iteration delivery summary |
| `getIterationLoad` | 获取iteration load |
| `getRdAnalystAssignee` | 获取rd analyst assignee |
| `getRdParticipantsFromTasks` | 获取rd participants from tasks |
| `getRdRoleMeta` | 获取rd role meta |
| `getRelatedDevRequirementsForSchedule` | 获取related dev requirements for schedule |
| `getReviewDateStatus` | 获取review date status |
| `getScheduleAgileLabel` | 获取schedule agile label |
| `getScheduleAgileMembers` | 获取schedule agile members |
| `getScheduleBasisLabel` | 获取schedule basis label |
| `getScheduleBizTaskCount` | 获取schedule biz task count |
| `getScheduleCustomWindowDef` | 获取schedule custom window def |
| `getScheduleIntegratedPermission` | 获取schedule integrated permission |
| `getScheduleItemIterationKey` | 获取schedule item iteration key |
| `getScheduleItemLoad` | 获取schedule item load |
| `getScheduleIterationMeta` | 获取schedule iteration meta |
| `getScheduleListRisk` | 获取schedule list risk |
| `getScheduleListStageClass` | 获取schedule list stage class |
| `getScheduleListStageLabel` | 获取schedule list stage label |
| `getScheduleModalPeopleOptions` | 获取schedule modal people options |
| `getScheduleModalPermission` | 获取schedule modal permission |
| `getScheduleNextActionText` | 获取schedule next action text |
| `getSchedulePeopleForRows` | 获取schedule people for rows |
| `getSchedulePersonCapacity` | 获取schedule person capacity |
| `getSchedulePersonRoleLoad` | 获取schedule person role load |
| `getScheduleReqOwner` | 获取schedule req owner |
| `getScheduleRowByDemandItem` | 获取schedule row by demand item |
| `getScheduleRowForReq` | 获取schedule row for req |
| `getScheduleRowOwnerText` | 获取schedule row owner text |
| `getScheduleSourceTypeMeta` | 获取schedule source type meta |
| `getScheduleSourceTypeMetaForRd` | 获取schedule source type meta for rd |
| `getScheduleStageLabel` | 获取schedule stage label |
| `getScheduleStatusBucket` | 获取schedule status bucket |
| `getScheduleSystemCount` | 获取schedule system count |
| `getScheduleSystemHourDefaults` | 获取schedule system hour defaults |
| `getScheduleTaskProgressForBusiness` | 获取schedule task progress for business |
| `getScheduleTaskProgressForRd` | 获取schedule task progress for rd |
| `getScheduleVersionCreateDraft` | 获取schedule version create draft |
| `getScheduleVersionModalDraft` | 获取schedule version modal draft |
| `getScheduleVersionModalWindowMeta` | 获取schedule version modal window meta |
| `getScheduleWindowConfig` | 获取schedule window config |
| `getScheduleWindowStatusForRow` | 获取schedule window status for row |
| `getSuggestedReviewDate` | 获取suggested review date |
| `getTaskBuildExecutions` | 获取task build executions |
| `getTaskBuildProject` | 获取task build project |
| `getTaskOwnersFromDrafts` | 获取task owners from drafts |
| `getTransferPriorityValue` | 获取transfer priority value |
| `getVersionWindowFullName` | 获取version window full name |
| `getVersionWindowRange` | 获取version window range |
| `getVersionWindowShortName` | 获取version window short name |
| `getVersionWindowStatusLabel` | 获取version window status label |
| `getVersionWindowTitle` | 获取version window title |
| `getWindowStatus` | 获取window status |
| `getZentaoProjectsBySystem` | 获取zentao projects by system |
| `getZentaoTransferURL` | 获取zentao transfer u r l |
| `goToScheduleAdjustFromVersionFollow` | goToScheduleAdjustFromVersionFollow |
| `goToScheduleByIteration` | goToScheduleByIteration |
| `goToScheduleFromHome` | goToScheduleFromHome |
| `goToSchedulePersonGroupFromVersionFollow` | goToSchedulePersonGroupFromVersionFollow |
| `goToScheduleWithContext` | goToScheduleWithContext |
| `goToVersionFollowByIteration` | goToVersionFollowByIteration |
| `goToZentaoTransfer` | goToZentaoTransfer |
| `handleDateChange` | handleDateChange |
| `handleModalDateChange` | handleModalDateChange |
| `hasBizTaskBuildIncomplete` | hasBizTaskBuildIncomplete |
| `inferTaskStartDate` | inferTaskStartDate |
| `initScheduleVersionEditSystems` | initScheduleVersionEditSystems |
| `initTaskBuildContext` | initTaskBuildContext |
| `isBizScheduleIncomplete` | 判断是否biz schedule incomplete |
| `isProjectDirectScheduleItem` | 判断是否project direct schedule item |
| `isRdScheduleIncomplete` | 判断是否rd schedule incomplete |
| `isRdTaskBuildComplete` | 判断是否rd task build complete |
| `isReviewDateOverdue` | 判断是否review date overdue |
| `isScheduleDraft` | 判断是否schedule draft |
| `isScheduleTaskOwnerComplete` | 判断是否schedule task owner complete |
| `isScheduleUnscheduledCandidate` | 判断是否schedule unscheduled candidate |
| `iterationKeyFromWindowLabel` | iterationKeyFromWindowLabel |
| `iterationKeyToScheduleScope` | iterationKeyToScheduleScope |
| `mapIterationPlanToExecutionName` | mapIterationPlanToExecutionName |
| `matchesPoValueScheduleBiz` | matchesPoValueScheduleBiz |
| `nextIntegratedRdId` | nextIntegratedRdId |
| `normalizeScheduleIteration` | normalizeScheduleIteration |
| `normalizeScheduleIterationPlanStorage` | normalizeScheduleIterationPlanStorage |
| `normalizeVersionWindowKey` | normalizeVersionWindowKey |
| `onIterationDragOver` | 处理iteration drag over事件 |
| `onIterationDrop` | 处理iteration drop事件 |
| `onScheduleAgileMultiChange` | 处理schedule agile multi change事件 |
| `onScheduleDragStart` | 处理schedule drag start事件 |
| `onScheduleRowWindowChange` | 处理schedule row window change事件 |
| `openBatchModal` | 打开batch modal |
| `openManageVersionWindowsModal` | 打开manage version windows modal |
| `openScheduleCreateVersionWindowModal` | 打开schedule create version window modal |
| `openScheduleCustomWindowCreator` | 打开schedule custom window creator |
| `openScheduleIntegratedAddWindowModal` | 打开schedule integrated add window modal |
| `openScheduleIntegratedModal` | 打开schedule integrated modal |
| `openScheduleRowModal` | 打开schedule row modal |
| `openScheduleRowModalFromDemandDetail` | 打开schedule row modal from demand detail |
| `openScheduleRowModalFromKanban` | 打开schedule row modal from kanban |
| `openScheduleTaskListModal` | 打开schedule task list modal |
| `openScheduleVersionWindowEditFromScope` | 打开schedule version window edit from scope |
| `openScheduleVersionWindowEditModal` | 打开schedule version window edit modal |
| `openTaskBuildModal` | 打开task build modal |
| `openTaskBuildModalFromKanban` | 打开task build modal from kanban |
| `openTransferDevInZentao` | 打开transfer dev in zentao |
| `openTransferDevInZentaoFromIntegrated` | 打开transfer dev in zentao from integrated |
| `openTransferDevModal` | 打开transfer dev modal |
| `persistScheduleCustomWindows` | persistScheduleCustomWindows |
| `persistScheduleSystemPlans` | persistScheduleSystemPlans |
| `recomputeScheduleItemHoursFromSystems` | recomputeScheduleItemHoursFromSystems |
| `removeIntegratedRdDraftRow` | removeIntegratedRdDraftRow |
| `removeTransferDevDraftRow` | removeTransferDevDraftRow |
| `renderBizReqLevel1Row` | 渲染biz req level1 row区块/列表 |
| `renderBizReqLevel2Row` | 渲染biz req level2 row区块/列表 |
| `renderBizReqLevel3Row` | 渲染biz req level3 row区块/列表 |
| `renderBizReqRows` | 渲染biz req rows区块/列表 |
| `renderIndependentRDRow` | 渲染independent r d row区块/列表 |
| `renderIndependentRDRows` | 渲染independent r d rows区块/列表 |
| `renderIntegratedRdDraftTableRow` | 渲染integrated rd draft table row区块/列表 |
| `renderIntegratedRdSavedNode` | 渲染integrated rd saved node区块/列表 |
| `renderManageVersionWindowsModal` | 渲染manage version windows modal区块/列表 |
| `renderPOScheduleArea` | 渲染p o schedule area区块/列表 |
| `renderPoolRow` | 渲染pool row区块/列表 |
| `renderRdRoleSystemInline` | 渲染rd role system inline区块/列表 |
| `renderSchedule` | 排期页主入口：概览条 + 列表/看板/分组 + 快捷计数 |
| `renderScheduleBusinessRow` | 渲染schedule business row区块/列表 |
| `renderScheduleCapacitySummaryLine` | 渲染schedule capacity summary line区块/列表 |
| `renderScheduleCreateVersionWindowModalBody` | 渲染schedule create version window modal body区块/列表 |
| `renderScheduleDevRequirementRow` | 渲染schedule dev requirement row区块/列表 |
| `renderScheduleGroupModeTabs` | 渲染schedule group mode tabs区块/列表 |
| `renderScheduleGroupView` | 渲染schedule group view区块/列表 |
| `renderScheduleIntegratedModal` | 渲染schedule integrated modal区块/列表 |
| `renderScheduleIntegratedPersonSelect` | 渲染schedule integrated person select区块/列表 |
| `renderScheduleIntegratedSection1` | 渲染schedule integrated section1区块/列表 |
| `renderScheduleIntegratedSection2` | 渲染schedule integrated section2区块/列表 |
| `renderScheduleIterationGroupView` | 渲染schedule iteration group view区块/列表 |
| `renderScheduleKanbanStatusChips` | 渲染schedule kanban status chips区块/列表 |
| `renderScheduleModalPersonSelect` | 渲染schedule modal person select区块/列表 |
| `renderScheduleOwnerAvatar` | 渲染schedule owner avatar区块/列表 |
| `renderScheduleOwnerStack` | 渲染schedule owner stack区块/列表 |
| `renderSchedulePersonGroupView` | 渲染schedule person group view区块/列表 |
| `renderSchedulePersonSelect` | 渲染schedule person select区块/列表 |
| `renderSchedulePlannerKanban` | 渲染schedule planner kanban区块/列表 |
| `renderSchedulePoSummaryBar` | 渲染schedule po summary bar区块/列表 |
| `renderSchedulePriorityGroupView` | 渲染schedule priority group view区块/列表 |
| `renderScheduleQuickCounts` | 渲染schedule quick counts区块/列表 |
| `renderScheduleRdMetrics` | 渲染schedule rd metrics区块/列表 |
| `renderScheduleRdTaskRow` | 渲染schedule rd task row区块/列表 |
| `renderScheduleRdTitlePrefix` | 渲染schedule rd title prefix区块/列表 |
| `renderScheduleRiskCell` | 渲染schedule risk cell区块/列表 |
| `renderScheduleRowModalContent` | 渲染schedule row modal content区块/列表 |
| `renderScheduleRowWindowPicker` | 渲染schedule row window picker区块/列表 |
| `renderScheduleSection` | 渲染schedule section区块/列表 |
| `renderScheduleSourceTypeBadge` | 渲染schedule source type badge区块/列表 |
| `renderScheduleStageTag` | 渲染schedule stage tag区块/列表 |
| `renderScheduleSystemHourInput` | 渲染schedule system hour input区块/列表 |
| `renderScheduleSystemPersonSelect` | 渲染schedule system person select区块/列表 |
| `renderScheduleSystemPersonnelTable` | 渲染schedule system personnel table区块/列表 |
| `renderScheduleTableHead` | 渲染schedule table head区块/列表 |
| `renderScheduleTaskProgress` | 渲染schedule task progress区块/列表 |
| `renderScheduleVersionSystemsAndPlansSection` | 渲染schedule version systems and plans section区块/列表 |
| `renderScheduleVersionWindowModalBody` | 渲染schedule version window modal body区块/列表 |
| `renderTaskBuildModalContent` | 渲染task build modal content区块/列表 |
| `renderTaskBuildProjectPanel` | 渲染task build project panel区块/列表 |
| `renderTaskBuildRdEssentials` | 渲染task build rd essentials区块/列表 |
| `renderTransferDevModalContent` | 渲染transfer dev modal content区块/列表 |
| `renderTransferStoryTableBody` | 渲染transfer story table body区块/列表 |
| `renderTransferStoryTableHead` | 渲染transfer story table head区块/列表 |
| `rerenderScheduleIntegratedIfOpen` | rerenderScheduleIntegratedIfOpen |
| `resetScheduleRiskFilter` | resetScheduleRiskFilter |
| `resetScheduleSuspendedFilter` | resetScheduleSuspendedFilter |
| `resolveRdDevTeamByAnalyst` | resolveRdDevTeamByAnalyst |
| `resolveRdDevTeamForRd` | resolveRdDevTeamForRd |
| `resolveRdDevTeamForSystem` | resolveRdDevTeamForSystem |
| `resolveTransferStoryDevTeam` | resolveTransferStoryDevTeam |
| `saveIntegratedTasksForRd` | 保存integrated tasks for rd |
| `saveScheduleCreateVersionWindow` | 保存schedule create version window |
| `saveScheduleIntegratedCollabTasks` | 保存schedule integrated collab tasks |
| `saveScheduleIntegratedModal` | 保存schedule integrated modal |
| `saveScheduleIntegratedTransferRd` | 保存schedule integrated transfer rd |
| `saveScheduleRowModal` | 保存schedule row modal |
| `saveScheduleVersionModalSystemPlans` | 保存schedule version modal system plans |
| `saveScheduleVersionWindowModal` | 保存schedule version window modal |
| `saveTaskBuildModal` | 保存task build modal |
| `saveTransferDevModal` | 保存transfer dev modal |
| `scheduleAddDaysIso` | scheduleAddDaysIso |
| `scheduleItemMatchesScope` | scheduleItemMatchesScope |
| `scheduleIterationLabel` | scheduleIterationLabel |
| `scheduleKanbanRowColumn` | scheduleKanbanRowColumn |
| `scheduleSelectValueForItemPlan` | scheduleSelectValueForItemPlan |
| `seedRdTaskBuildZentaoMeta` | seedRdTaskBuildZentaoMeta |
| `selectVfIteration` | selectVfIteration |
| `setIterationCapacityMode` | 设置iteration capacity mode |
| `setScheduleAnomalyFilter` | 设置schedule anomaly filter |
| `setScheduleCustomWindowScope` | 设置schedule custom window scope |
| `setScheduleDataType` | 设置schedule data type |
| `setScheduleGroupMode` | 设置schedule group mode |
| `setScheduleScope` | 设置schedule scope |
| `setTaskBuildIteration` | 设置task build iteration |
| `simplifyAnomalyText` | simplifyAnomalyText |
| `splitDirectEstimateHours` | splitDirectEstimateHours |
| `sumRdTaskHours` | sumRdTaskHours |
| `switchScheduleView` | switchScheduleView |
| `syncScheduleDevRequirementsFromSystems` | 同步schedule dev requirements from systems |
| `syncScheduleMainSystemPersonnel` | 同步schedule main system personnel |
| `syncScheduleOwnersFromTaskDrafts` | 同步schedule owners from task drafts |
| `syncSchedulePeopleFilters` | 同步schedule people filters |
| `syncScheduleScopeChips` | 同步schedule scope chips |
| `syncScheduleScopeControls` | 同步schedule scope controls |
| `syncScheduleVersionWindowFilterDropdown` | 同步schedule version window filter dropdown |
| `todayIsoDate` | todayIsoDate |
| `toggleAllPoolSchedule` | 切换all pool schedule |
| `toggleAllSchedule` | 切换all schedule |
| `toggleAllScheduleRows` | 切换all schedule rows |
| `toggleIntegratedRdNode` | 切换integrated rd node |
| `toggleIntegratedTaskDraft` | 切换integrated task draft |
| `toggleScheduleBusinessChildren` | 切换schedule business children |
| `toggleScheduleCreateSystem` | 切换schedule create system |
| `toggleScheduleCreateSystemPlanSync` | 切换schedule create system plan sync |
| `toggleScheduleCreateUseOrgTemplate` | 切换schedule create use org template |
| `toggleScheduleModalExternal` | 切换schedule modal external |
| `toggleScheduleModalSystem` | 切换schedule modal system |
| `toggleScheduleModalSystemPlanSync` | 切换schedule modal system plan sync |
| `toggleScheduleMoreFilters` | 切换schedule more filters |
| `toggleScheduleRdTasks` | 切换schedule rd tasks |
| `toggleScheduleRiskFilter` | 切换schedule risk filter |
| `toggleScheduleSuspendedFilter` | 切换schedule suspended filter |
| `toggleScheduleUnscheduledPreset` | 切换schedule unscheduled preset |
| `toggleScheduleVersionUseOrgTemplate` | 切换schedule version use org template |
| `toggleTaskBuildCreateIteration` | 切换task build create iteration |
| `toggleTransferDevDraftRow` | 切换transfer dev draft row |
| `toggleVersionCardMenu` | 切换version card menu |
| `updateIntegratedTaskDraftField` | 更新integrated task draft field |
| `updatePoolSelection` | 更新pool selection |
| `updateScheduleBatch` | 更新schedule batch |
| `updateScheduleCreateDraftField` | 更新schedule create draft field |
| `updateScheduleCreateSystemPlanName` | 更新schedule create system plan name |
| `updateScheduleField` | 更新schedule field |
| `updateScheduleListShell` | 更新schedule list shell |
| `updateScheduleModalSystemPlanName` | 更新schedule modal system plan name |
| `updateScheduleRushJumpQueue` | 更新schedule rush jump queue |
| `updateScheduleSelection` | 更新schedule selection |
| `updateScheduleSystemPersonnel` | 更新schedule system personnel |
| `updateScheduleVersionDraftAgile` | 更新schedule version draft agile |
| `updateScheduleVersionDraftDate` | 更新schedule version draft date |
| `updateScheduleVersionDraftWindowType` | 更新schedule version draft window type |
| `updateScheduleVersionWindow` | 更新schedule version window |
| `updateTaskBuildExecution` | 更新task build execution |
| `updateTaskBuildNewIterationName` | 更新task build new iteration name |
| `updateTaskBuildProject` | 更新task build project |
| `updateTaskBuildQuickEnd` | 更新task build quick end |
| `updateTaskBuildQuickStart` | 更新task build quick start |
| `updateTaskBuildQuickWeeks` | 更新task build quick weeks |
| `updateTaskDraftField` | 更新task draft field |
| `updateTransferDevDraftField` | 更新transfer dev draft field |
| `validateScheduleSave` | validateScheduleSave |
| `vfRowMatchesIteration` | vfRowMatchesIteration |

### 版本跟进 (page-versionFollow)（49）

| 函数名 | 功能说明 |
|--------|----------|
| `applyVersionFollowColumnVisibility` | 应用version follow column visibility |
| `applyVersionFollowFilters` | 应用version follow filters |
| `clearVfFilters` | clearVfFilters |
| `collectVfOwnerOptions` | collectVfOwnerOptions |
| `enrichRowData` | enrichRowData |
| `formatVfShortRange` | formatVfShortRange |
| `getCurrentStage` | 获取current stage |
| `getCurrentVfRows` | 获取current vf rows |
| `getCurrentVfVersion` | 获取current vf version |
| `getVersionFollowRows` | 获取version follow rows |
| `getVfBaseRows` | 获取vf base rows |
| `getVfExceptionFilter` | 获取vf exception filter |
| `getVfHealthLevel` | 获取vf health level |
| `getVfKpiRows` | 获取vf kpi rows |
| `getVfMemberDisplayName` | 获取vf member display name |
| `getVfProgressStages` | 获取vf progress stages |
| `getVfVersionStats` | 获取vf version stats |
| `goToVersionFollow` | goToVersionFollow |
| `isVfBlocked` | 判断是否vf blocked |
| `isVfOverdue` | 判断是否vf overdue |
| `jumpVersionFollowFromOverview` | jumpVersionFollowFromOverview |
| `mergeVfFilters` | mergeVfFilters |
| `openVersionFollowInNewTab` | 打开version follow in new tab |
| `renderVersionFollow` | 版本跟进页主入口 |
| `renderVersionFollowTestAcceptanceTable` | 渲染version follow test acceptance table区块/列表 |
| `renderVfFilterMore` | 渲染vf filter more区块/列表 |
| `renderVfHealthBanner` | 渲染vf health banner区块/列表 |
| `renderVfOverview` | 渲染vf overview区块/列表 |
| `renderVfProgressStrip` | 渲染vf progress strip区块/列表 |
| `renderVfTabBody` | 渲染vf tab body区块/列表 |
| `renderVfUnifiedBar` | 渲染vf unified bar区块/列表 |
| `renderVfVersionSwitch` | 渲染vf version switch区块/列表 |
| `rowMatchesVfException` | rowMatchesVfException |
| `rowMatchesVfMyWork` | rowMatchesVfMyWork |
| `selectVfHistoryVersion` | selectVfHistoryVersion |
| `selectVfStage` | selectVfStage |
| `setVfExceptionFilter` | 设置vf exception filter |
| `setVfHistorySearchQuery` | 设置vf history search query |
| `setVfInlineFilter` | 设置vf inline filter |
| `setVfMoreFilter` | 设置vf more filter |
| `setVfMyScope` | 设置vf my scope |
| `toggleVfFilterPanel` | 切换vf filter panel |
| `toggleVfHistoryMenu` | 切换vf history menu |
| `toggleVfKpiPopover` | 切换vf kpi popover |
| `vfCapacityMetaKey` | vfCapacityMetaKey |
| `vfH` | vfH |
| `vfHistoryQuickSearchInput` | vfHistoryQuickSearchInput |
| `vfNameRoughMatch` | vfNameRoughMatch |
| `vfRowActions` | vfRowActions |

### 问题风险 (page-issueRisk)（17）

| 函数名 | 功能说明 |
|--------|----------|
| `applyIssueRiskFilters` | 应用issue risk filters |
| `issueRiskApplyStatusFilter` | issueRiskApplyStatusFilter |
| `issueRiskCountByStatus` | issueRiskCountByStatus |
| `issueRiskDetectKind` | issueRiskDetectKind |
| `issueRiskEsc` | issueRiskEsc |
| `issueRiskFormatDate` | issueRiskFormatDate |
| `issueRiskGetAgileGroup` | issueRiskGetAgileGroup |
| `issueRiskIsOpen` | issueRiskIsOpen |
| `issueRiskNormText` | issueRiskNormText |
| `onIssueRiskSearchInput` | 处理issue risk search input事件 |
| `renderIssueRisk` | 渲染issue risk区块/列表 |
| `renderIssueRiskTable` | 渲染issue risk table区块/列表 |
| `resetIssueRiskFilters` | resetIssueRiskFilters |
| `setIssueRiskAgileFilter` | 设置issue risk agile filter |
| `setIssueRiskKindFilter` | 设置issue risk kind filter |
| `setIssueRiskStatusFilter` | 设置issue risk status filter |
| `setIssueRiskTeamFilter` | 设置issue risk team filter |

### 系统工作区 (page-productWorkspace)（29）

| 函数名 | 功能说明 |
|--------|----------|
| `applyProductFilters` | 应用product filters |
| `checkProductAnomaly` | checkProductAnomaly |
| `clearProductFilters` | clearProductFilters |
| `closeProductDetail` | 关闭product detail |
| `getAnomalySeverity` | 获取anomaly severity |
| `getManagementStrategyDisplay` | 获取management strategy display |
| `getModeDescription` | 获取mode description |
| `getMyRoleInProduct` | 获取my role in product |
| `getRndMode` | 获取rnd mode |
| `getRndModeDisplayName` | 获取rnd mode display name |
| `getSeverityLabel` | 获取severity label |
| `getStrategyLabel` | 获取strategy label |
| `goToProductDemands` | goToProductDemands |
| `goToProductRiskDemands` | goToProductRiskDemands |
| `initRndModeConfigList` | initRndModeConfigList |
| `navigateToProduct` | navigateToProduct |
| `openRndModeDrawer` | 打开rnd mode drawer |
| `renderEmptyState` | 渲染empty state区块/列表 |
| `renderModeConfig` | 渲染mode config区块/列表 |
| `renderPagination` | 渲染pagination区块/列表 |
| `renderProductList` | 渲染product list区块/列表 |
| `renderRiskRulePreview` | 渲染risk rule preview区块/列表 |
| `saveModeConfig` | 保存mode config |
| `saveRndModeConfigFromDrawer` | 保存rnd mode config from drawer |
| `selectRndModeTab` | selectRndModeTab |
| `showProductDetail` | showProductDetail |
| `showProductDetailEnhanced` | showProductDetailEnhanced |
| `updateProductRndMode` | 更新product rnd mode |
| `updateRndConfig` | 更新rnd config |

### 快捷入口 (page-quickLinks)（2）

| 函数名 | 功能说明 |
|--------|----------|
| `openQuickLink` | 打开quick link |
| `renderQuickLinks` | 渲染quick links区块/列表 |

### 我关注 (page-watch)（3）

| 函数名 | 功能说明 |
|--------|----------|
| `renderWatchPage` | 渲染watch page区块/列表 |
| `setWatchTypeFilter` | 设置watch type filter |
| `showWatchPage` | showWatchPage |

### 通知中心 (page-notifications)（5）

| 函数名 | 功能说明 |
|--------|----------|
| `getNoticeTypeClass` | 获取notice type class |
| `markAllNoticeRead` | markAllNoticeRead |
| `openNoticeLink` | 打开notice link |
| `renderNoticeSummary` | 渲染notice summary区块/列表 |
| `renderNotifications` | 渲染notifications区块/列表 |

### MR/门禁 (page-mr)（4）

| 函数名 | 功能说明 |
|--------|----------|
| `closeExceptionModal` | 关闭exception modal |
| `openExceptionModal` | 打开exception modal |
| `renderMr` | 渲染mr区块/列表 |
| `switchMrTab` | switchMrTab |

### Drawer/详情/交付弹窗（28）

| 函数名 | 功能说明 |
|--------|----------|
| `buildFallbackDrawerData` | 构建fallback drawer data |
| `closeDeliveryModal` | 关闭delivery modal |
| `closeDrawer` | 关闭drawer |
| `confirmDelivery` | confirmDelivery |
| `executeStageAction` | executeStageAction |
| `extractDemandId` | extractDemandId |
| `openDeliveryModal` | 打开delivery modal |
| `openDrawer` | 打开 PO 详情 Drawer |
| `renderActionSection` | 渲染action section区块/列表 |
| `renderApprovalSection` | 渲染approval section区块/列表 |
| `renderCollaborationSection` | 渲染collaboration section区块/列表 |
| `renderCurrentStageGuidance` | 渲染current stage guidance区块/列表 |
| `renderCycleAnalysisSection` | 渲染cycle analysis section区块/列表 |
| `renderDeliveryStatusSection` | 渲染delivery status section区块/列表 |
| `renderLifecycleBar` | 渲染lifecycle bar区块/列表 |
| `renderLifecycleSection` | 渲染lifecycle section区块/列表 |
| `renderLifecycleSection_OLD` | 渲染lifecycle section_ o l d区块/列表 |
| `renderPODrawerDetail` | 渲染p o drawer detail区块/列表 |
| `renderRelationSummarySection` | 渲染relation summary section区块/列表 |
| `renderSummarySection` | 渲染summary section区块/列表 |
| `renderSuspendSection` | 渲染suspend section区块/列表 |
| `renderTimelineSection` | 渲染timeline section区块/列表 |
| `showPOFullDetail` | showPOFullDetail |
| `toggleDevGroup` | 切换dev group |
| `transformToDrawerFormat` | transformToDrawerFormat |
| `updateDeliveryWindowType` | 更新delivery window type |
| `viewDetail` | viewDetail |
| `viewInZentao` | viewInZentao |

### 全局/导航/认证/配置（173）

| 函数名 | 功能说明 |
|--------|----------|
| `adminClearUserViews` | adminClearUserViews |
| `adminEditUser` | adminEditUser |
| `adminRemoveUser` | adminRemoveUser |
| `adminRenderUserEditor` | adminRenderUserEditor |
| `adminResetConfig` | adminResetConfig |
| `adminSaveUserViews` | adminSaveUserViews |
| `adminStartNewUser` | adminStartNewUser |
| `adminToggleDefaultView` | adminToggleDefaultView |
| `applyCrossPageContext` | 应用cross page context |
| `applyHomeLayoutForView` | 应用home layout for view |
| `applySavedDemandQuery` | 应用saved demand query |
| `applySuspendedFilter` | 应用suspended filter |
| `applyTableColumnVisibility` | 应用table column visibility |
| `applyWorkbenchAccessControl` | 应用workbench access control |
| `applyWorkbenchUrlParams` | 应用workbench url params |
| `bottleneckReasonFromItem` | bottleneckReasonFromItem |
| `buildFallbackRelation` | 构建fallback relation |
| `buildLifecycleDataFromPoReqData` | 构建lifecycle data from po req data |
| `buildPOFlowFlags` | 构建p o flow flags |
| `buildPoUrgeMessage` | 构建po urge message |
| `buildRiskDataFromPoReqData` | 构建risk data from po req data |
| `calcN` | calcN |
| `calcOverdueDays` | calcOverdueDays |
| `calcSortScore` | calcSortScore |
| `changeScope` | changeScope |
| `cleanupAllModals` | cleanupAllModals |
| `clearWorkbenchCaches` | clearWorkbenchCaches |
| `closeAllPanels` | 关闭all panels |
| `closeColumnConfigModal` | 关闭column config modal |
| `closeConfig` | 关闭config |
| `closeModalPage` | 关闭modal page |
| `computeBizPoValueStage` | computeBizPoValueStage |
| `computePoValueStage` | computePoValueStage |
| `computeRdPoValueStage` | computeRdPoValueStage |
| `copyTextToClipboard` | copyTextToClipboard |
| `detectActionType` | detectActionType |
| `detectLane` | detectLane |
| `detectObjectType` | detectObjectType |
| `detectRiskTags` | detectRiskTags |
| `detectTodayAction` | detectTodayAction |
| `displayZenTaoId` | displayZenTaoId |
| `drilldownPoHub` | drilldownPoHub |
| `enterProcessing` | enterProcessing |
| `executeAction` | executeAction |
| `fallbackCopy` | fallbackCopy |
| `fetchWithTimeout` | fetchWithTimeout |
| `filterBySummary` | 筛选by summary |
| `filterPOByCard` | 筛选p o by card |
| `getActionButtonConfig` | 获取action button config |
| `getActionLabel` | 获取action label |
| `getAllowedViewsForUser` | 获取allowed views for user |
| `getCurrentAccount` | 获取current account |
| `getFilteredPOFlowItems` | 获取filtered p o flow items |
| `getLaneLabel` | 获取lane label |
| `getLifecycleIndex` | 获取lifecycle index |
| `getPOFlowItems` | 获取p o flow items |
| `getPOFlowTabCount` | 获取p o flow tab count |
| `getPoReqBlockerInfo` | 获取po req blocker info |
| `getPoValueStageLabel` | 获取po value stage label |
| `getRdTimeFields` | 获取rd time fields |
| `getRiskTagStyle` | 获取risk tag style |
| `getScopedPOFlowItems` | 获取scoped p o flow items |
| `getSearchIcon` | 获取search icon |
| `getWorkbenchApiBase` | 获取workbench api base |
| `getWorkbenchCurrentUserName` | 获取workbench current user name |
| `getZentaoURL` | 获取zentao u r l |
| `goToZentao` | goToZentao |
| `handleAction` | handleAction |
| `handleGlobalSearch` | handleGlobalSearch |
| `handleTop5Item` | handleTop5Item |
| `hasBizTransferredToDev` | hasBizTransferredToDev |
| `hideLoginModal` | hideLoginModal |
| `inferDeliverDeadline` | inferDeliverDeadline |
| `inferDevelopFinish` | inferDevelopFinish |
| `inferTestFinish` | inferTestFinish |
| `isAdminUser` | 判断是否admin user |
| `isHomeScopeBizReq` | 判断是否home scope biz req |
| `isPersonEmpty` | 判断是否person empty |
| `jumpRoleStep` | jumpRoleStep |
| `loadAccessConfig` | loadAccessConfig |
| `loadColumnConfig` | loadColumnConfig |
| `mapPoPhaseToValueStage` | mapPoPhaseToValueStage |
| `mapWorkbenchStatusToZentao` | mapWorkbenchStatusToZentao |
| `matchesPoValueAccept` | matchesPoValueAccept |
| `matchesPoValueAcceptance` | matchesPoValueAcceptance |
| `matchesPoValueClarify` | matchesPoValueClarify |
| `matchesPoValueDeliveryBiz` | matchesPoValueDeliveryBiz |
| `matchesPoValueDeliveryRd` | matchesPoValueDeliveryRd |
| `matchesPoValueRelease` | matchesPoValueRelease |
| `matchesPoValueSubmitTest` | matchesPoValueSubmitTest |
| `matchesPoValueTest` | matchesPoValueTest |
| `mergeTodos` | mergeTodos |
| `navTo` | SPA 页面切换，按 page 调用对应 render* |
| `navigateToDemand` | navigateToDemand |
| `navigateToTodo` | navigateToTodo |
| `normalizeApproval` | normalizeApproval |
| `normalizeBizStatusKey` | normalizeBizStatusKey |
| `normalizeDemandAgile` | normalizeDemandAgile |
| `normalizeDrawerDate` | normalizeDrawerDate |
| `normalizePriority` | normalizePriority |
| `normalizeSuspend` | normalizeSuspend |
| `normalizeToPoWorkItem` | normalizeToPoWorkItem |
| `nudgeSuspendOwner` | nudgeSuspendOwner |
| `openColumnConfigModal` | 打开column config modal |
| `openConfig` | 打开config |
| `openMoreFilters` | 打开more filters |
| `openZentaoPlaceholder` | 打开zentao placeholder |
| `poNextFromDemandItem` | poNextFromDemandItem |
| `poReqIsBlockerCandidate` | poReqIsBlockerCandidate |
| `processPOAction` | processPOAction |
| `renderAdminMenu` | 渲染admin menu区块/列表 |
| `renderAdminPage` | 渲染admin page区块/列表 |
| `renderBottom` | 渲染bottom区块/列表 |
| `renderFilters` | 渲染filters区块/列表 |
| `renderPECollab` | 渲染p e collab区块/列表 |
| `renderPOCards` | 渲染p o cards区块/列表 |
| `renderPOFlow` | 渲染p o flow区块/列表 |
| `renderPOFlowRow` | 渲染p o flow row区块/列表 |
| `renderPOFlowRows` | 渲染p o flow rows区块/列表 |
| `renderPOFlowSection` | 渲染p o flow section区块/列表 |
| `renderPORelease` | 渲染p o release区块/列表 |
| `renderPOScope` | 渲染p o scope区块/列表 |
| `renderPOSummary` | 渲染p o summary区块/列表 |
| `renderPlanBreakpoint` | 渲染plan breakpoint区块/列表 |
| `renderRoleGuide` | 渲染role guide区块/列表 |
| `renderSimpleLifecycle` | 渲染simple lifecycle区块/列表 |
| `renderSummary` | 渲染summary区块/列表 |
| `renderTodos` | 渲染todos区块/列表 |
| `renderViewDropdown` | 渲染view dropdown区块/列表 |
| `resetColumnConfig` | resetColumnConfig |
| `resetFilters` | resetFilters |
| `resetWorkbenchConfig` | resetWorkbenchConfig |
| `resumeDemand` | resumeDemand |
| `saveAccessConfig` | 保存access config |
| `saveColumnConfig` | 保存column config |
| `saveColumnConfigStore` | 保存column config store |
| `saveCurrentDemandQuery` | 保存current demand query |
| `saveWorkbenchConfig` | 保存workbench config |
| `setFilter` | 设置filter |
| `setHubFilter` | 设置hub filter |
| `setPOFlowFilter` | 设置p o flow filter |
| `setPOLane` | 设置p o lane |
| `setPOScope` | 设置p o scope |
| `setTypeFilter` | 设置type filter |
| `showLoginModal` | showLoginModal |
| `showModalPage` | showModalPage |
| `showShortcutsPage` | showShortcutsPage |
| `showStageActionPanel` | showStageActionPanel |
| `showTeamRiskPage` | showTeamRiskPage |
| `showToast` | 全局 Toast 提示 |
| `submitDepartmentApproval` | submitDepartmentApproval |
| `switchConfigTab` | switchConfigTab |
| `switchDemandPerspective` | switchDemandPerspective |
| `switchToPOSideNav` | switchToPOSideNav |
| `switchToPage` | switchToPage |
| `syncPOBadges` | 同步p o badges |
| `toggleColumnConfigDraft` | 切换column config draft |
| `toggleMultiSelect` | 切换multi select |
| `toggleRoleGuide` | 切换role guide |
| `toggleSidebar` | 切换sidebar |
| `toggleSubMenu` | 切换sub menu |
| `toggleTodoExpand` | 切换todo expand |
| `toggleViewDD` | 切换view d d |
| `transformToPoReq` | transformToPoReq |
| `updateFocalChipsActive` | 更新focal chips active |
| `updateUserInfo` | 更新user info |
| `validateNoResidualOverlay` | validateNoResidualOverlay |
| `wbDefaultConfig` | wbDefaultConfig |
| `wbGetEnabledViewNames` | wbGetEnabledViewNames |
| `wbLoadConfig` | wbLoadConfig |
| `wbRebuildAllTodos` | wbRebuildAllTodos |
| `wbSaveConfig` | wbSaveConfig |
| `withdrawDepartmentApproval` | withdrawDepartmentApproval |

### business-demand-detail.js（51）

| 函数名 | 功能说明 |
|--------|----------|
| `attachZentaoClarificationMock` | attachZentaoClarificationMock |
| `bdEscape` | bdEscape |
| `bdExpandAll` | bdExpandAll |
| `bdFocusDev` | bdFocusDev |
| `bdFocusUnit` | bdFocusUnit |
| `bdFormatDateYmd` | bdFormatDateYmd |
| `bdGateClass` | bdGateClass |
| `bdGetPoActionMeta` | bdGetPoActionMeta |
| `bdResolveTimelineStageIndex` | bdResolveTimelineStageIndex |
| `bdScrollToStructure` | bdScrollToStructure |
| `bdSetTab` | 切换详情 Tab（structure/valuestream/schedule/quality…） |
| `bdSummaryCardClick` | bdSummaryCardClick |
| `bdToastLink` | bdToastLink |
| `bdToggleDev` | 展开/折叠研发需求 |
| `bdToggleMoreMenu` | bdToggleMoreMenu |
| `bdToggleMoreTab` | bdToggleMoreTab |
| `bdToggleMoreTab` | bdToggleMoreTab |
| `bdToggleUnit` | 展开/折叠交付单元 |
| `bdValueStreamStripClick` | bdValueStreamStripClick |
| `bdVersionWindowLinkHtml` | bdVersionWindowLinkHtml |
| `buildBdTimelineFromCanonicalPhase` | 构建bd timeline from canonical phase |
| `buildSyntheticBrFromReq` | 无 BR 时由 REQ 合成聚合视图 |
| `closeActionDrawer` | 关闭action drawer |
| `closeBusinessDemandDetail` | 关闭business demand detail |
| `closeBusinessDemandDetailFromUi` | 关闭business demand detail from ui |
| `closeDemandDetail` | 关闭demand detail |
| `closeGlobalBusinessDemandDetail` | 关闭全屏聚合详情 |
| `executeGuidanceAction` | executeGuidanceAction |
| `getActiveBusinessDemandDetail` | 获取active business demand detail |
| `getBdTimelineStageNames` | 获取bd timeline stage names |
| `getBusinessDemandDetailById` | 获取business demand detail by id |
| `getDemandFullDetailData` | 由 scheduleData 推导需求全貌字段 |
| `initAll` | initAll |
| `mergeFullDetailIntoBr` | 合并 getDemandFullDetailData 到 BR 模型 |
| `openActionDrawer` | 打开action drawer |
| `openBusinessDemandDetail` | 打开business demand detail |
| `openGlobalBusinessDemandDetail` | 全屏 aside 打开聚合详情 |
| `prepareBusinessDemandDetailModel` | REQ/BR → 统一详情视图模型 |
| `renderBdHierarchyRail` | 渲染bd hierarchy rail区块/列表 |
| `renderBdNextActionBar` | 渲染bd next action bar区块/列表 |
| `renderBdQualityPanel` | 渲染bd quality panel区块/列表 |
| `renderBdValueStreamStrip` | 渲染bd value stream strip区块/列表 |
| `renderBusinessDemandDetailPage` | 渲染 BR 聚合详情主 UI（Tab/侧栏/KPI） |
| `renderDemandFullDetailPage` | 渲染demand full detail page区块/列表 |
| `resolveBusinessDemandIdForOverview` | resolveBusinessDemandIdForOverview |
| `resolveRepresentativeReqIdForBr` | resolveRepresentativeReqIdForBr |
| `showDemandDetail` | showDemandDetail |
| `showStageDetail` | showStageDetail |
| `showStageGuidance` | showStageGuidance |
| `submitAction` | submitAction |
| `toggleBdKpiRow` | 切换bd kpi row |

## 五、CSS 结构

| 模块/类名前缀 | 行号范围 | 对应页面 | 说明 |
|---------------|----------|----------|------|
| `Layout / CSS 变量 / .app .sidebar` | 1–47 | 全局布局 |  |
| `.nav-item--disabled 等` | 48–59 | 导航置灰（MVP） |  |
| `.main .header .content` | 60–89 | 主区域/顶栏 |  |
| `.bd-* / .dw-biz-detail-aside` | 90–224 | 业务需求聚合详情 | demandWorkspace + 跨页详情 |
| `.po-home-* .home-vs-* .home-blocker-*` | 225–306 | PO 首页区块 | page-home |
| `.summary-*` | 307–318 | 摘要卡片 | 首页/待办 |
| `.ir-* issue-risk` | 319–473 | 问题风险页 | page-issueRisk |
| `.vf-*` | 474–603 | 版本跟进 | page-versionFollow |
| `.global-entry-*` | 604–609 | 全局入口 |  |
| `.todo-section` | 610–622 | 待办表格区 | page-todos |
| `.tbl` | 623–632 | 通用表格 |  |
| `.badge .pri-tag .status-tag` | 633–689 | 徽章/标签 | 全局 |
| `.bottom-*` | 690–709 | 页脚 |  |
| `.page-hdr` | 710–714 | 页面标题 | 多页 |
| `.kanban-*` | 715–728 | 看板 | page-kanban / demand kanban |
| `.schedule-* (基础)` | 729–774 | 排期基础 | page-schedule |
| `.form-*` | 775–784 | 表单控件 | Modal/Drawer |
| `.modal-* .batch-modal-*` | 785–794 | 弹窗 | 全局 Modal |
| `.drawer-*` | 795–853 | Drawer | 详情/配置 |
| `.audit-note` | 854–861 | 审计提示 | 例外/交付 Modal |
| `#toast` | 858–861 | Toast | 全局 |
| `.search-box` | 865–886 | 全局搜索 | header |
| `.sub-tabs .view-tabs` | 887–909 | 子 Tab | 多页 |
| `.sv-pill .saved-views` | 892–897 | 保存视图 pill | query/demand |
| `.po-hub-* .po-flow-*` | 910–1069 | PO 行动中枢 | 首页旧模块 |
| `.top5-*` | 1070–1240 | Top5 列表 | page-home |
| `.overview-grid-*` | 1241–1304 | 概览网格 | 首页 |
| `.risk-radar-*` | 1317–1453 | 指标雷达 | page-riskRadar |
| `.quick-link-*` | 1454–1481 | 快捷入口 | page-quickLinks |
| `.dw-* .demand-*` | 1522–1612 | 需求工作区 | page-demandWorkspace |
| `.schedule-table-mini` | 1613–1629 | 排期迷你表格 |  |
| `.po-dashboard-* .value-stream-*` | 1630–1760 | PO 首页价值流 | page-home |
| `.dvs-* stage-guidance` | 1761–1932 | 需求价值流进度/阶段引导 |  |
| `.schedule-area .schedule-filter*` | 2182–2336 | 排期列表区 | page-schedule |
| `z-index 层叠 / schedule modal` | 2337–2658 | 排期弹窗层叠 |  |
| `.product-*` | 2728–2769 | 系统工作区 | page-productWorkspace |
| `.detail-drawer` | 2793–2815 | overlay drawer | demandDetail/action |
| `.po-home Round 2-7 优化` | 2899–3319 | PO 首页迭代样式 | page-home |
| `schedule 列表 redesign / 树形层级` | 3320–3499 | 排期列表增强 | page-schedule |

CSS 采用 `:root` 设计令牌（`--blue`、`--border`、`--t1` 等）+ BEM 风格组件类；排期与需求工作区样式在文件后半段多次迭代（Round 2–7、Schedule redesign）。

## 六、已完成 vs 待开发对照表

| 功能域 | 状态 | 说明 |
|--------|------|------|
| 工作台首页 page-home | 🔨 | 已有 `/po/home` 模板+价值流阶段 SSR+Top5 AJAX；缺 KPI 网格、卡点条、风险分析、版本窗口 Dock、今日焦点交互 |
| 价值流阶段筛选 | 🔨 | 阶段卡片已渲染，点击筛选与 API 未接通（占位文案） |
| Top5 当前应推进事项 | 🔨 | 表格+`/po/demands` 已对接；缺价值流筛选联动、处理 Drawer、查看更多 |
| 今日推进焦点 | ❌ | demo 标注待推出；项目仅静态占位 |
| 卡点快速响应 | ❌ | 模板中 HTML 注释掉，未实现 |
| 版本窗口 Dock（首页） | ❌ | 未实现 |
| 排期工作台 page-schedule | 🔨 | SSR 页面+静态树形列表+基础 JS 筛选 UI；缺 API、Modal、批量、看板/分组、窗口 CRUD |
| 版本窗口概览卡片 | 🔨 | 服务端渲染 Windows 卡片；缺点击跳转、编辑、容量模式切换 |
| 快捷筛选/主筛选/更多筛选 | 🔨 | HTML+部分前端切换；计数与后端筛选未接 |
| 排期列表树形层级 | 🔨 | SSR 三级行+展开折叠 JS；缺动态数据与行内操作 |
| 逐条排期/一体化/转研发/拆任务 Modal | ❌ | demo 完整；项目未开始 |
| 排期看板/分组视图 | ❌ | demo JS 已实现；项目未开始 |
| 需求综合查询 page-demandWorkspace | ❌ | demo 完整；项目无对应路由/模块 |
| 需求查询/看板旧页 page-query/page-kanban | ❌ | demo 有；项目未迁移（被 demandWorkspace 取代） |
| 版本跟进 page-versionFollow | ❌ | demo 完整；项目未开始 |
| 问题风险 page-issueRisk | ❌ | demo 完整；项目未开始 |
| 系统工作区 page-productWorkspace | ❌ | demo 完整；项目未开始 |
| 我的待办 page-todos | ❌ | demo 完整；项目未开始 |
| 通知中心 page-notifications | ❌ | demo 完整；项目未开始 |
| 我关注 page-watch | ❌ | demo 完整；项目未开始 |
| 指标雷达 page-riskRadar | ❌ | demo 完整；项目未开始 |
| 质效预警 page-qualityWarning | ❌ | demo 完整；项目未开始 |
| 快捷入口 page-quickLinks | ❌ | demo 完整；项目未开始 |
| MR/门禁 page-mr | ❌ | demo 完整；项目未开始 |
| 业务需求聚合详情 (bd.js) | ❌ | demo 完整独立模块；项目未集成 |
| 全局 Drawer/Toast/登录 Modal | 🔨 | demo 完整；项目有登录页但无 SPA Modal；Toast 在 ui.js 部分存在 |
| 工作台配置 Drawer | ❌ | demo 有；nav 标注待推出 |
| 后台管理 page-admin | ❌ | demo 有本地权限 mock；项目用 RBAC 模块但未做此页 |
| 全局搜索 | ❌ | demo header 搜索；项目未实现 |
| 侧栏导航全量页面 | 🔨 | demo MVP 仅 home+schedule 可进；项目菜单体系独立，仅 po/home+schedule 有部分页面 |

**图例：** ✅ 已完成 · 🔨 部分完成 · ❌ 未开始

---

## 附录 A：Demo 文件职责

| 文件 | 职责 |
|------|------|
| `personal-workbench.html` | 单页应用壳：侧栏、17 个 page div、全局 Modal/Drawer 挂载点 |
| `personal-workbench.app.js` | 全部页面逻辑、mock 数据、API 对接（PO 登录/列表）、路由 `navTo` |
| `personal-workbench.css` | 全站样式（布局/首页/排期/需求工作区/版本跟进等） |
| `business-demand-detail.js` | 业务需求（BR）聚合详情侧栏/全屏层，与 app.js 中 scheduleData 联动 |
| `_top5_snippet.js` | 历史片段，未接入 HTML |

## 附录 B：当前项目已有入口

| 路由 | 模板 | 后端 | 前端 JS |
|------|------|------|---------|
| `GET /po/home` | `web/templates/po/home.html` | `internal/module/po` Home + Demands API | `web/static/js/po/home.js` |
| `GET /schedule` | `web/templates/schedule/index.html` | `internal/server/schedulehandler.go` SSR | `web/static/js/schedule/schedule.js` |