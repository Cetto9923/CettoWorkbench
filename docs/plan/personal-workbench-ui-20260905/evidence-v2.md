# R0 证据与契约冻结报告 (Evidence & Contract Freeze)

- **执行基线**: `2f2580a10f45db5978736f1fe1e46c258133022f`
- **工作分支**: `Claude-PO`
- **工作区路径**: `/Users/yuyan9923/GitHub/workbench-claude-po`
- **执行时间**: 2026-09-06

---

## 1. Preflight 执行证据

```sh
$ pwd
/Users/yuyan9923/GitHub/workbench-claude-po
$ git rev-parse --show-toplevel
/Users/yuyan9923/GitHub/workbench-claude-po
$ git branch --show-current
Claude-PO
$ git rev-parse HEAD
2f2580a10f45db5978736f1fe1e46c258133022f
```

### 工作区保护文件确认
- `web/static/css/po/board.css`: 预存在 WIP，严格保持原样，不修改。
- `web/static/js/po/workboard.js`: 预存在 WIP，严格保持原样，不修改。
- `web/templates/po/workboard.html`: 预存在 WIP，严格保持原样，不修改。

---

## 2. 我的待办数据源矩阵 (Todo Source Matrix)

| 对象域 | 对应对象 | 数据表 | 责任人 (Actor) 条件 | 状态过滤条件 | 截止时间字段 | 操作动作 | 分页/查询方式 | sourceStatus | 接入状态说明与原因 |
|---|---|---|---|---|---|---|---|---|---|
| **需求治理** | `demand` (业务需求) | `zt_demand` | `owner = :account OR assignedTo = :account` | `deleted = '0' AND status NOT IN ('closed', 'cancel')` | `deliverDate` / `endDate` | 办理 | `QueryTodoUnified` SQL UNION 分页 | **supported** | 已接入统一 SQL 分页与计数；已做父子需求去重 |
| **研发执行** | `task` (任务) | `zt_task` | `assignedTo = :account` | `deleted = '0' AND status NOT IN ('done', 'closed', 'cancel')` | `deadline` | 办理 | `QueryTodoUnified` SQL UNION 分页 | **supported** | 已接入统一 SQL 分页与计数 |
| **测试质量** | `bug` (Bug) | `zt_bug` | `assignedTo = :account` | `deleted = '0' AND status = 'active'` | `deadline` | 办理 | `QueryTodoUnified` SQL UNION 分页 | **supported** | 已接入统一 SQL 分页与计数 |
| **研发执行** | `story` (研发需求) | `zt_story` | `assignedTo = :account` | `deleted = '0' AND status NOT IN ('closed', 'released') AND IFNULL(sourceType, '') <> 'demandpool'` | `deliverDate` | 办理 | `repotodoextra.go` 独立内存查询 | **available_not_wired** | 存在候选适配函数，但目前未合入 `QueryTodoUnified` 联合 SQL 排序与分页；若直接接入需统一 UNION 并评审索引 |
| **测试质量** | `testtask` (测试单) | `zt_testtask` | `owner = :account` | `deleted = '0' AND status <> 'done'` | `end` | 办理 | `repotodoextra.go` 独立内存查询 | **available_not_wired** | 存在候选适配函数，未合入统一分页 |
| **审批决策** | `approval` (审批节点) | `zt_approvalnode`, `zt_approvalobject` | `assignedTo = :account` | `status = 'doing' AND type = 'review'` | — | 审批/办理 | `repotodoapproval.go` 独立查询 | **available_not_wired** | 存在候选适配函数，只取当前 doing review 节点，未合入统一分页与排序 |
| **问题风险** | `issue` (问题) | `zt_issue` | `assignedTo = :account OR createdBy = :account` | `status NOT IN ('closed', 'cancel')` | `deadline` | 办理 | `repotodoextra.go` 独立内存查询 | **available_not_wired** | 存在候选函数，但责任人判定与生命周期规则尚待业务裁决 |
| **问题风险** | `risk` (风险) | `zt_risk` | `assignedTo = :account OR createdBy = :account` | `status NOT IN ('closed', 'hangup')` | — | 办理 | `repotodoextra.go` 独立内存查询 | **available_not_wired** | 存在候选函数，但风险不直接等同于阻塞待办，责任规则需裁决 |
| **个人事项** | `todo` (个人待办) | `zt_todo` | `account = :account OR assignedTo = :account` | `deleted = '0' AND status NOT IN ('done', 'closed')` | `date` | 办理 | `repotodoextra.go` 独立内存查询 | **available_not_wired** | 存在候选函数，个人日程事项与工作台业务待办的权责划分需裁决 |
| **我的关注** | `follow` (关注) | — | — | — | — | — | — | **wait_decision** | 缺乏统一跨对象关注关系表与权威数据源，保持等待业务决策 |

---

## 3. 通知关联与操作矩阵 (Notice Association Matrix)

| 场景 / 对象类型 | 数据来源表 | 关联主键与对象 ID 依据 | `canHandle` 判定条件 | `canView` 判定条件 | 操作列呈现 | 说明 |
|---|---|---|---|---|---|---|
| **业务需求通知** (`demand`) | `zt_notify` JOIN `zt_action` | `action.objectType = 'demand' AND action.objectID > 0` | `NeedAction = 1` 且存在合法 `DemandViewURL` | 存在有效 `DemandViewURL` | 优先“去处理”；无处理则“查看详情” | 点击同页打开业务需求详情 |
| **研发需求通知** (`story`) | `zt_notify` JOIN `zt_action` | `action.objectType = 'story' AND action.objectID > 0` | `NeedAction = 1` 且存在合法 `StoryViewURL` | 存在有效 `StoryViewURL` | 优先“去处理”；无处理则“查看详情” | 点击同页打开研发需求详情 |
| **任务/Bug/测试单** (`task`/`bug`/`testtask`) | `zt_notify` JOIN `zt_action` | `action.objectType IN ('task','bug','testtask') AND objectID > 0` | `NeedAction = 1` 且有动作入口 | 存在有效禅道对象详情 URL | 优先“去处理”；否则“查看详情” | 对象 ID 真实可靠时才生成链接 |
| **邮件通知** (`mail`) | `zt_notify` | 无对应关联业务对象（`objectID <= 0`） | 否 (`canHandle = false`) | 否 (`canView = false`) | 显示 `—` 或“邮件通知” | 不伪造 URL，不猜测 ID |
| **关联缺失通知** | `zt_notify` | `objectID <= 0` 或 `objectType` 未知 | 否 | 否 | 显示 `—` 或“关联信息缺失” | 严禁从主题/正文中正则提取数字冒充 ID |
| **标为已读操作** | `zt_workbench_notify_reads` | `notify_id` + `account` | 独立次要按钮 | 独立次要按钮 | 行内独立按钮“标为已读” | 已读状态不改变“去处理”或“查看详情”的能力与可见性 |

---

## 4. 首页焦点矩阵 (Home Focus Matrix)

| focus 键值 | 中文文案 | 预期业务过滤逻辑 | 当前 SQL / 字段支持现状 | 实施判定 |
|---|---|---|---|---|
| `all` | 全部 | 显示全部行动列表 | 原有 `/demands` 查询直接支持 | **PASS** |
| `today` | 今日必推 | 今日截止或已逾期且未完成 | 需结合 `deliverDate` 等截止字段 SQL 过滤 | **WAIT DECISION** (待冻结排序与环境) |
| `pending` | 待我处理 | 当前责任人属于当前账号 | 基础集已有当前用户过滤 | **PASS** |
| `blocked` | 阻塞 | 存在明确阻塞事实 | 缺少权威阻塞事实表或审计状态证据 | **BLOCKED** |
| `overdue` | 超期 | 有效截止时间已逾期 | 需统一截止日期判定标准 | **WAIT DECISION** |
| `suspended` | 挂起 | 最近一次处于挂起状态未恢复 | 需结合挂起日志或标记字段 | **BLOCKED** |

**规范结论**：
按照宪章与指令要求，缺乏真实 Schema 字段与业务排序规则的 focus 选项标为 WAIT DECISION，UI 上显式禁用或标明状态，严禁前端全量加载后在内存中过滤！
