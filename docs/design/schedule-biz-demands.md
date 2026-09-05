# 排期工作台 · 业务需求 Tab 列表 — 数据层设计

> 模块：`internal/module/schedule`
> 范围：本阶段仅方法签名 + SQL 草稿 + Service 装配伪代码，**不写 Go 实现**。
> 模式：禅道「主表分页 + 批量 IN 补字段」，禁止大宽表 JOIN。

---

## 一、整体流程图

### 文字描述

`ListBizDemands` 采用 **7 步装配** 流水线：先确定当前用户可见的需求池（pool），再对**顶层业需**做分页主查；随后按 parent / fromDemand 批量拉子业需与研发需求；用多路 IN 查询补齐产品名、PM 澄清、版本窗口、任务统计、敏捷小组名、用户 realname；在 Service 层聚合三层树并计算业需 5 态排期阶段；最后返回嵌套 JSON。

所有关联表查询均独立、参数化，阶段判定**不在 SQL** 中完成。

### ASCII 框图

```
┌─────────────────────────────────────────────────────────────────────────┐
│                     ListBizDemands(actor, req)                          │
└─────────────────────────────────────────────────────────────────────────┘
                                    │
    ┌───────────────────────────────┼───────────────────────────────┐
    │                               ▼                               │
    │  ① GetUserDemandPools(account)                                │
    │     IsAdmin? → 全部 pool : 按 participant/审批人/部门 ACL 过滤   │
    └───────────────────────────────┬───────────────────────────────┘
                                    │ poolIDs[]
                                    ▼
    ┌─────────────────────────────────────────────────────────────────┐
    │  ② ListBizDemands(req, poolIDs)  — 顶层业需分页主查              │
    │     parent=0, deleted='0', pool IN (?), 筛选, ORDER id DESC     │
    │     + 单独 COUNT(*) 得 total（TODO：筛选拼接细节）               │
    └───────────────────────────────┬─────────────────────────────────┘
                                    │ topDemands[] (pageSize 条)
                                    ▼
    ┌─────────────────────────────────────────────────────────────────┐
    │  ③ FindChildDemandsByParents(parentIDs)                         │
    │     parent IN (?) AND deleted='0'                               │
    └───────────────────────────────┬─────────────────────────────────┘
                                    │ childDemands[]
                                    ▼
    ┌─────────────────────────────────────────────────────────────────┐
    │  ④ FindStoriesByDemands(allDemandIDs)                          │
    │     fromDemand IN (?), type='story', deleted='0'                │
    └───────────────────────────────┬─────────────────────────────────┘
                                    │ stories[]
                                    ▼
    ┌─────────────────────────────────────────────────────────────────┐
    │  ⑤ 批量 IN 补字段（可顺序/并发，互不 JOIN）                      │
    │     · CountClarifyProductsByDemands   → productCount             │
    │     · FindClarifyPMsByDemands         → PM 列表                  │
    │     · FindProductsByIDs               → 主系统/产品名             │
    │     · FindStoryWindowMappings         → story → window           │
    │     · CountStoryTasks                 → taskTotal / unassigned   │
    │     · FindTeamgroupsByIDs（已有）      → teamGroup 显示名         │
    │     · FindUsersByAccounts             → realname                 │
    └───────────────────────────────┬─────────────────────────────────┘
                                    │ maps / stats
                                    ▼
    ┌─────────────────────────────────────────────────────────────────┐
    │  ⑥ calcBizDemandStage(topDemand, children, stories, maps)       │
    │     5 态判定（Service Go 代码，首个命中为准）                      │
    └───────────────────────────────┬─────────────────────────────────┘
                                    │ stage per top demand
                                    ▼
    ┌─────────────────────────────────────────────────────────────────┐
    │  ⑦ 装配三层嵌套 ListBizDemandsResp                               │
    │     BizDemandItem → []SubDemandItem → []StoryItem               │
    └─────────────────────────────────────────────────────────────────┘
```

---

## 二、Form 层结构体

> 落位：`internal/module/schedule/form.go`（下一阶段实现）。
> 读取类 Req 仅用 `form` tag；JSON 响应用 `json` tag。

### 排期阶段常量（Service 层使用）

```go
const (
    StageNoWindow       = "未关联窗口"
    StageNoStory        = "未转研发"
    StageNoTask         = "未建任务"
    StageTaskUnassigned = "已建任务未指派"
    StageTaskAssigned   = "已建任务并指派"

    StoryStageNoWindow  = "未关联窗口"
    StoryStageHasWindow = "已关联窗口"
)
```

### 请求 / 响应

```go
// ListBizDemandsReq 业务需求 Tab 列表查询入参。
type ListBizDemandsReq struct {
    Page         int    `form:"page"`         // 页码，从 1 开始
    PageSize     int    `form:"pageSize"`     // 每页顶层业需条数
    TeamgroupID  uint   `form:"teamgroupId"`  // 敏捷小组筛选，0=全部
    ProductID    uint   `form:"productId"`    // 主系统/产品筛选，0=全部
    Stage        string `form:"stage"`        // 排期阶段筛选（Service 层过滤，见 §五）
    Status       string `form:"status"`       // 业需 status 筛选（TODO：SQL 细节）
    Keyword      string `form:"keyword"`      // 编号/名称/责任人/系统模糊搜（TODO）
    WindowID     uint   `form:"windowId"`     // 版本窗口筛选（TODO）
    Scope        string `form:"scope"`        // 快捷 chip：notClosed/unscheduled/...（TODO）
}

// Validate 校验分页与基础参数。
func (r *ListBizDemandsReq) Validate() []FieldError

// ListBizDemandsResp 业务需求 Tab 列表响应。
type ListBizDemandsResp struct {
    Total int64           `json:"total"`
    Items []BizDemandItem `json:"items"`
}
```

### 嵌套列表项

```go
// BizDemandItem 顶层业需（树形一级）。
type BizDemandItem struct {
    ID              uint            `json:"id"`
    Name            string          `json:"name"`
    Pri             int             `json:"pri"`             // 禅道 pri 原值，前端映射 P0/P1
    Status          string          `json:"status"`
    MainSystemName  string          `json:"mainSystemName"`  // zt_demand.mainSystem → zt_product.name
    ProductCount    int             `json:"productCount"`    // zt_demandclarify 多系统计数（不含主系统时可 +1，见装配说明）
    TeamgroupName   string          `json:"teamgroupName"`   // zt_demand.teamGroup → FindTeamgroupsByIDs
    PMs             []string        `json:"pms"`             // clarify PM realname 列表
    Stage           string          `json:"stage"`           // 业需 5 态，Service 计算
    WindowName      string          `json:"windowName"`      // 聚合展示：子树首个有窗口的研发对应窗口名（TODO 展示规则）
    Children        []SubDemandItem `json:"children"`        // 子业需
    Stories         []StoryItem     `json:"stories"`         // 直接挂在顶层业需下的研发（parent 业需自身）
}

// SubDemandItem 子业需（树形二级）。
type SubDemandItem struct {
    ID              uint        `json:"id"`
    Name            string      `json:"name"`
    Pri             int         `json:"pri"`
    Status          string      `json:"status"`
    MainSystemName  string      `json:"mainSystemName"`
    ProductCount    int         `json:"productCount"`
    TeamgroupName   string      `json:"teamgroupName"` // 继承父业需 teamGroup
    PMs             []string    `json:"pms"`
    Stage           string      `json:"stage"`       // 子业需行可留空或继承父级，列表以顶层 stage 为准
    WindowName      string      `json:"windowName"`
    Stories         []StoryItem `json:"stories"`
}

// StoryItem 研发需求（树形三级）。
type StoryItem struct {
    ID                      uint   `json:"id"`
    Title                   string `json:"title"`
    Pri                     int    `json:"pri"`
    ProductName             string `json:"productName"`
    Stage                   string `json:"stage"`           // 研发 2 态：未关联窗口 / 已关联窗口
    WindowName              string `json:"windowName"`
    TeamgroupName           string `json:"teamgroupName"`   // 显示父业需的 teamGroup
    AssignedTo              string `json:"assignedTo"`      // account
    AssignedToName          string `json:"assignedToName"`  // realname
    TaskCount               int    `json:"taskCount"`
    IsMainSystemAssociation int    `json:"isMainSystemAssociation"` // 0/1，阶段聚合用
}
```

### Repo 辅助结构体（同 package，供 map 返回值）

```go
// ClarifyPM 业需澄清 PM 行。
type ClarifyPM struct {
    Demand  uint
    Product uint
    PM      string // account
}

// StoryWindowRef 研发需求关联的版本窗口。
type StoryWindowRef struct {
    StoryID    uint
    WindowID   uint
    WindowName string
}

// StoryTaskStat 研发任务统计。
type StoryTaskStat struct {
    StoryID    uint
    Total      int
    Unassigned int
}

// ZtDemand 禅道 zt_demand 只读投影（List / FindChild 共用）。
type ZtDemand struct {
    ID              uint
    Name            string
    Pri             int
    Status          string
    MainSystem      uint
    TeamGroup       uint   // 列名 teamGroup
    BRA             string
    QD              string
    RD              string
    CreatedBy       string
    Pool            uint
    Parent          uint
    Hang            string
    Category        string
    EstimateLaunch  string
}

// ZtStory 禅道 zt_story 只读投影。
type ZtStory struct {
    ID                      uint
    Title                   string
    Pri                     int
    Product                 uint
    Plan                    uint
    Stage                   string
    Status                  string
    FromDemand              uint   // 列名 fromDemand
    IsMainSystemAssociation int    // 列名 isMainSystemAssociation
    AssignedTo              string
}
```

---

## 三、Repo 层新增方法清单

> 落位：`internal/module/schedule/repo.go`（下一阶段实现）。
> **复用已有**：`IsAdmin`、`FindTeamgroupsByIDs`（见现有 `internal/module/schedule/repo.go`）。

---

### 3.1 GetUserDemandPools

```go
// GetUserDemandPools 返回用户可见的需求池 ID 列表。
// 超管（IsAdmin=true）返回全部未删除 pool；普通用户按 ACL 过滤。
func (r *Repo) GetUserDemandPools(ctx context.Context, account string) ([]uint, error)
```

**前置：用户部门路径**

```sql
-- 入参：? = 当前用户 dept（zt_user.dept）
SELECT path FROM zt_dept WHERE id = ? AND deleted = '0'
```

`path` 为逗号分隔祖先 id 串（如 `,1,5,12,`）。Service/Repo 将其拆为 `[]uint`，与自身 `dept` 一并用于 `IN (?)`。

**普通用户 SQL**

```sql
SELECT id
FROM zt_demandpool
WHERE deleted = '0'
  AND (
    acl = 'open'
    OR FIND_IN_SET(?, participant) > 0
    OR FIND_IN_SET(?, businessReviewer) > 0
    OR dept IN (?)
  )
ORDER BY id ASC
```

| 参数 | 含义 |
|------|------|
| `?` (×2) | 当前用户 account |
| `?` (IN) | 用户 dept + path 解析出的祖先 dept id 列表 |

**超管 SQL**

```sql
SELECT id FROM zt_demandpool WHERE deleted = '0' ORDER BY id ASC
```

**字段映射**

| SELECT | 用途 |
|--------|------|
| `id` | 后续 `ListBizDemands` 的 `pool IN (?)` 条件 |

**装配伪代码**

```go
isAdmin, _ := r.IsAdmin(ctx, account)
if isAdmin {
    // 超管 SQL
} else {
    // 查 user.dept → dept.path → 解析 deptIDs
    // 普通用户 SQL
}
```

---

### 3.2 ListBizDemands

```go
// ListBizDemands 顶层业需分页主查询（parent=0）。
// 返回当前页顶层业需切片与匹配筛选条件的 total。
// poolIDs 来自 GetUserDemandPools；poolIDs 为空时直接返回空列表。
func (r *Repo) ListBizDemands(ctx context.Context, req ListBizDemandsReq, poolIDs []uint) ([]ZtDemand, int64, error)
```

**COUNT SQL（与 LIST 共用 WHERE，禁止用 LIMIT 那条查 total）**

```sql
SELECT COUNT(*) AS total
FROM zt_demand
WHERE deleted = '0'
  AND parent = 0
  AND pool IN (?)
  -- AND status = ?           -- TODO：status 筛选
  -- AND teamGroup = ?        -- TODO：teamgroupId 筛选
  -- AND mainSystem = ?       -- TODO：productId 筛选（主系统）
  -- AND (id = ? OR name LIKE ?)  -- TODO：keyword
```

**LIST SQL**

```sql
SELECT
  id,
  name,
  pri,
  status,
  mainSystem,
  teamGroup,
  BRA,
  QD,
  RD,
  createdBy,
  pool,
  parent,
  hang,
  category,
  estimateLaunch
FROM zt_demand
WHERE deleted = '0'
  AND parent = 0
  AND pool IN (?)
  -- 同上可选筛选占位
ORDER BY id DESC
LIMIT ? OFFSET ?
```

| 参数 | 含义 |
|------|------|
| `pool IN (?)` | GetUserDemandPools 结果 |
| `LIMIT ?` | req.PageSize |
| `OFFSET ?` | (req.Page - 1) * req.PageSize |

**字段映射**

| SELECT 字段 | 用途 |
|-------------|------|
| `id` | 顶层业需主键；后续 parent / fromDemand 关联 |
| `name` | BizDemandItem.Name |
| `pri` | BizDemandItem.Pri |
| `status` | BizDemandItem.Status；快捷 chip 筛选 |
| `mainSystem` | 查 zt_product 得主系统名 |
| `teamGroup` | FindTeamgroupsByIDs 得 TeamgroupName |
| `BRA/QD/RD/createdBy` | 预留：keyword 搜责任人、权限扩展 |
| `pool` | 数据范围校验 |
| `parent` | 固定 0（顶层） |
| `hang/category/estimateLaunch` | 预留：风险/挂起/超期（TODO §七） |

> **注意**：`stage` 筛选不在 SQL；Repo 返回原始业需，Service 装配后按 `req.Stage` 过滤（或下一阶段改为二次过滤策略）。

---

### 3.3 FindChildDemandsByParents

```go
// FindChildDemandsByParents 批量查询子业需。
func (r *Repo) FindChildDemandsByParents(ctx context.Context, parentIDs []uint) ([]ZtDemand, error)
```

```sql
SELECT
  id, name, pri, status, mainSystem, teamGroup,
  BRA, QD, RD, createdBy, pool, parent, hang, category, estimateLaunch
FROM zt_demand
WHERE deleted = '0'
  AND parent IN (?)
ORDER BY parent ASC, id ASC
```

| SELECT 字段 | 用途 |
|-------------|------|
| `parent` | 挂到对应 BizDemandItem.Children |
| 其余 | 同 3.2，装配 SubDemandItem |

---

### 3.4 FindStoriesByDemands

```go
// FindStoriesByDemands 按业需 ID（顶层+子业需）批量查研发需求。
func (r *Repo) FindStoriesByDemands(ctx context.Context, demandIDs []uint) ([]ZtStory, error)
```

```sql
SELECT
  id,
  title,
  pri,
  product,
  plan,
  stage,
  status,
  fromDemand,
  isMainSystemAssociation,
  assignedTo
FROM zt_story
WHERE fromDemand IN (?)
  AND type = 'story'
  AND deleted = '0'
ORDER BY fromDemand ASC, isMainSystemAssociation DESC, id ASC
```

| SELECT 字段 | 用途 |
|-------------|------|
| `fromDemand` | 挂到顶层/子业需的 Stories |
| `isMainSystemAssociation` | 业需 5 态判定：仅统计 =1 的主系统研发任务 |
| `product` | FindProductsByIDs → ProductName |
| `assignedTo` | StoryItem.AssignedTo；Collect 进 FindUsersByAccounts |
| `plan` | FindStoryWindowMappings 关联用 |

---

### 3.5 CountClarifyProductsByDemands

```go
// CountClarifyProductsByDemands 统计业需关联的多系统（clarify 去重 product 数）。
func (r *Repo) CountClarifyProductsByDemands(ctx context.Context, demandIDs []uint) (map[uint]int, error)
```

```sql
SELECT demand, COUNT(DISTINCT product) AS productCount
FROM zt_demandclarify
WHERE demand IN (?)
GROUP BY demand
```

| 字段 | 用途 |
|------|------|
| `demand` | map key |
| `productCount` | BizDemandItem/ProductCount（多系统「+N」展示） |

---

### 3.6 FindClarifyPMsByDemands

```go
// FindClarifyPMsByDemands 查询业需澄清 PM，按 demand 分组。
func (r *Repo) FindClarifyPMsByDemands(ctx context.Context, demandIDs []uint) (map[uint][]ClarifyPM, error)
```

```sql
SELECT demand, product, PM
FROM zt_demandclarify
WHERE demand IN (?)
ORDER BY demand ASC, product ASC
```

| 字段 | 用途 |
|------|------|
| `PM` | account → FindUsersByAccounts → PMs[] realname |
| `product` | 预留：按系统展示负责人 |

---

### 3.7 FindProductsByIDs

```go
// FindProductsByIDs 批量查产品/系统名称。
func (r *Repo) FindProductsByIDs(ctx context.Context, productIDs []uint) (map[uint]string, error)
```

```sql
SELECT id, name
FROM zt_product
WHERE id IN (?)
  AND deleted = '0'
```

| 字段 | 用途 |
|------|------|
| `id` | map key |
| `name` | MainSystemName / StoryItem.ProductName |

---

### 3.8 FindStoryWindowMappings

```go
// FindStoryWindowMappings 查研发需求关联的版本窗口（每 story 取 vw.id 最小的一条）。
func (r *Repo) FindStoryWindowMappings(ctx context.Context, storyIDs []uint) (map[uint]StoryWindowRef, error)
```

```sql
SELECT ps.story, vw.id AS windowID, vw.name AS windowName
FROM zt_planstory ps
INNER JOIN zt_versionwindowproduct vwp
  ON vwp.plan = ps.plan AND vwp.deletedAt IS NULL
INNER JOIN zt_versionwindow vw
  ON vw.id = vwp.versionWindow AND vw.deletedAt IS NULL
WHERE ps.story IN (?)
ORDER BY ps.story ASC, vw.id ASC
```

| 字段 | 用途 |
|------|------|
| `story` | map key |
| `windowID` | 阶段判定：>0 表示已关联窗口 |
| `windowName` | StoryItem.WindowName；业需 WindowName 聚合 |

**说明**：同一 story 多 plan/window 时，Go 层取 **ORDER BY vw.id ASC 的首条**；其余窗口下一阶段再处理。

---

### 3.9 CountStoryTasks

```go
// CountStoryTasks 统计研发需求下任务总数与未指派数。
func (r *Repo) CountStoryTasks(ctx context.Context, storyIDs []uint) (map[uint]StoryTaskStat, error)
```

```sql
SELECT
  story,
  COUNT(*) AS total,
  SUM(CASE WHEN assignedTo IS NULL OR assignedTo = '' THEN 1 ELSE 0 END) AS unassigned
FROM zt_task
WHERE story IN (?)
  AND deleted = '0'
GROUP BY story
```

| 字段 | 用途 |
|------|------|
| `total` | StoryItem.TaskCount；业需阶段：主系统研发 task 汇总 |
| `unassigned` | 业需 5 态：「已建任务未指派」判定 |

---

### 3.10 FindUsersByAccounts

```go
// FindUsersByAccounts 批量查用户 realname。
func (r *Repo) FindUsersByAccounts(ctx context.Context, accounts []string) (map[string]string, error)
```

```sql
SELECT account, realname
FROM zt_user
WHERE account IN (?)
  AND deleted = '0'
```

| 字段 | 用途 |
|------|------|
| `account` | map key |
| `realname` | PMs[]、AssignedToName |

---

## 四、Service 层装配伪代码

```go
// ListBizDemands 查询排期工作台业务需求 Tab 列表。
func (s *Service) ListBizDemands(ctx context.Context, actor *model.User, req ListBizDemandsReq) (*ListBizDemandsResp, error) {
    account := actorAccount(actor)
    if account == "" {
        return &ListBizDemandsResp{Total: 0, Items: []BizDemandItem{}}, nil
    }

    // 0. 校验 & 规范化分页
    if errs := req.Validate(); len(errs) > 0 {
        return nil, validationError(errs) // 由 Handler 转 422
    }
    normalizePage(&req)

    // 1. 拿可见 pool（含超管判定，复用 repo.IsAdmin）
    poolIDs, err := s.repo.GetUserDemandPools(ctx, account)
    if err != nil {
        return nil, err
    }
    if len(poolIDs) == 0 {
        return &ListBizDemandsResp{Total: 0, Items: []BizDemandItem{}}, nil
    }

    // 2. 顶层业需分页主查询（total 来自独立 COUNT）
    topDemands, total, err := s.repo.ListBizDemands(ctx, req, poolIDs)
    if err != nil {
        return nil, err
    }
    if len(topDemands) == 0 {
        return &ListBizDemandsResp{Total: total, Items: []BizDemandItem{}}, nil
    }

    topIDs := pluckIDs(topDemands)

    // 3. 批量拿子业需
    childDemands, err := s.repo.FindChildDemandsByParents(ctx, topIDs)
    if err != nil {
        return nil, err
    }
    childByParent := groupByParent(childDemands)

    // 4. 业需 id 集合 = 顶层 + 子业需
    allDemandIDs := mergeIDs(topIDs, pluckIDs(childDemands))

    // 5. 拿研发需求
    stories, err := s.repo.FindStoriesByDemands(ctx, allDemandIDs)
    if err != nil {
        return nil, err
    }
    storiesByDemand := groupStoriesByFromDemand(stories)

    // 6. 收集批量 ID / account，顺序或 errgroup 并发查询
    productIDs := collectProductIDs(topDemands, childDemands, stories)
    storyIDs := pluckStoryIDs(stories)
    teamgroupIDs := collectTeamgroupIDs(topDemands) // 子业需继承父 teamGroup
    accounts := collectAccounts(stories, /* clarify PMs 需先查 PM */)

    productCountByDemand, _ := s.repo.CountClarifyProductsByDemands(ctx, allDemandIDs)
    clarifyPMsByDemand, _ := s.repo.FindClarifyPMsByDemands(ctx, allDemandIDs)
    accounts = mergeAccounts(accounts, pluckPMAccounts(clarifyPMsByDemand))

    productNameByID, _ := s.repo.FindProductsByIDs(ctx, productIDs)
    windowByStory, _ := s.repo.FindStoryWindowMappings(ctx, storyIDs)
    taskStatByStory, _ := s.repo.CountStoryTasks(ctx, storyIDs)
    teamgroupNameByID, _ := s.loadTeamgroupDisplayNames(ctx, teamgroupIDs) // 复用现有 parent/child 拼接逻辑
    realnameByAccount, _ := s.repo.FindUsersByAccounts(ctx, accounts)

    // 7. 逐条顶层业需：计算 stage、装配 SubDemandItem / StoryItem
    items := make([]BizDemandItem, 0, len(topDemands))
    for _, top := range topDemands {
        children := childByParent[top.ID]
        subtreeStories := collectSubtreeStories(top.ID, children, storiesByDemand)
        mainSystemStories := filterMainSystemStories(subtreeStories)

        stage := calcBizDemandStage(subtreeStories, mainSystemStories, windowByStory, taskStatByStory)

        item := BizDemandItem{
            ID:             top.ID,
            Name:           top.Name,
            Pri:            top.Pri,
            Status:         top.Status,
            MainSystemName: productNameByID[top.MainSystem],
            ProductCount:   productCountByDemand[top.ID],
            TeamgroupName:  teamgroupNameByID[top.TeamGroup],
            PMs:            resolvePMNames(clarifyPMsByDemand[top.ID], realnameByAccount),
            Stage:          stage,
            WindowName:     pickBizWindowName(subtreeStories, windowByStory),
            Children:       buildSubItems(children, ...maps),
            Stories:        buildStoryItems(storiesByDemand[top.ID], top.TeamGroup, ...maps),
        }
        items = append(items, item)
    }

    // 8. req.Stage / scope 等 Service 层过滤（TODO：与 total 一致性策略见 §七）
    if req.Stage != "" {
        items = filterByStage(items, req.Stage)
    }

    return &ListBizDemandsResp{Total: total, Items: items}, nil
}
```

---

## 五、排期阶段判定（重点）

> **全部在 Service 层 Go 代码计算**，不写入 SQL。
> 判定输入：某**顶层业需**及其**全部子业需**下的**全部研发 story**，以及 `windowByStory`、`taskStatByStory`。

### 5.1 业需层面 5 态（按顺序判定，首个命中为准）

| 顺序 | 阶段 | 判定条件 | 伪代码 |
|------|------|----------|--------|
| 1 | 未关联窗口 | 业需子树内**存在**研发 story，且**全部** story 的 windowID == 0 | `len(stories)>0 && all(windowID==0)` |
| 2 | 未转研发 | 业需 + 子业需**没有任何** type='story' 的研发 | `len(stories)==0` |
| 3 | 未建任务 | 已转研发；**主系统**研发（`isMainSystemAssociation=1`）的任务总数 == 0 | `sum(mainStories.taskTotal)==0` |
| 4 | 已建任务未指派 | 已转研发；主系统任务总数 > 0；存在 unassigned > 0 | `taskTotal>0 && sum(unassigned)>0` |
| 5 | 已建任务并指派 | 已转研发；主系统任务总数 > 0；主系统任务全部已指派 | `taskTotal>0 && sum(unassigned)==0` |

**顺序说明（与需求文档一致）**

需求文档表格顺序为：**未关联窗口 → 未转研发 → 未建任务 → …**

- 「未转研发」(`len(stories)==0`) 隐含没有窗口、没有任务，若放在「未关联窗口」之后则**永远不会命中**。
- 实现时仍按需求文档顺序编写 `if` 链，但 **第 1 条必须先排除 `len(stories)==0`**，否则逻辑矛盾。

**推荐实现（等价于文档顺序 + 排除空集）**

```go
func calcBizDemandStage(
    allStories []ZtStory,
    mainStories []ZtStory,
    windowByStory map[uint]StoryWindowRef,
    taskStatByStory map[uint]StoryTaskStat,
) string {
    // 2. 未转研发（无研发，优先于窗口判定）
    if len(allStories) == 0 {
        return StageNoStory
    }

    // 1. 未关联窗口（有研发但全无窗口）
    if allStoriesEvery(windowByStory, func(w StoryWindowRef) bool { return w.WindowID == 0 }) {
        return StageNoWindow
    }

    taskTotal, unassignedTotal := sumMainSystemTasks(mainStories, taskStatByStory)

    // 3. 未建任务
    if taskTotal == 0 {
        return StageNoTask
    }

    // 4. 已建任务未指派
    if unassignedTotal > 0 {
        return StageTaskUnassigned
    }

    // 5. 已建任务并指派
    return StageTaskAssigned
}
```

> 上式将「未转研发」提前到第一步，与文档 5 态语义一致；若严格按表格行号写 if-else，须在「未关联窗口」分支加 `len(allStories)>0`  guard。

**主系统任务聚合 SQL 表达（仅供理解，实际用 3.9 map 汇总）**

```sql
-- 概念：对 isMainSystemAssociation=1 的 story 汇总 zt_task
-- 不在 ListBizDemands 主流程执行，已由 CountStoryTasks + Go 过滤 mainStories 代替
SELECT
  SUM(ts.total) AS taskTotal,
  SUM(ts.unassigned) AS unassignedTotal
FROM (
  -- mainStories 的 story id 集合由 Go 传入
) s
LEFT JOIN (
  SELECT story,
         COUNT(*) AS total,
         SUM(CASE WHEN assignedTo IS NULL OR assignedTo = '' THEN 1 ELSE 0 END) AS unassigned
  FROM zt_task
  WHERE deleted = '0'
  GROUP BY story
) ts ON ts.story = s.id
```

### 5.2 研发需求行阶段（独立 2 态，列表展示用）

| 阶段 | 判定 |
|------|------|
| 未关联窗口 | `windowByStory[storyID].WindowID == 0` |
| 已关联窗口 | `windowByStory[storyID].WindowID > 0` |

```go
func calcStoryStage(storyID uint, windowByStory map[uint]StoryWindowRef) string {
    if ref, ok := windowByStory[storyID]; ok && ref.WindowID > 0 {
        return StoryStageHasWindow
    }
    return StoryStageNoWindow
}
```

研发行**不**计算「建任务」维度；TaskCount 仅展示数字。

---

## 六、字段映射对照

| 前端列 | Go 字段 | DB 来源 | 备注 |
|--------|---------|---------|------|
| 标题 | `Name` / `Title` | `zt_demand.name` / `zt_story.title` | 业需 Name，研发 Title |
| 优先级 | `Pri` | `zt_demand.pri` / `zt_story.pri` | 前端格式化为 P0/P1 |
| 关联系统 | `MainSystemName` + `ProductCount` | `zt_demand.mainSystem` → `zt_product.name`；`zt_demandclarify` COUNT | 业需/子业需 |
| 关联系统 | `ProductName` | `zt_story.product` → `zt_product.name` | 研发 |
| 排期阶段 | `Stage` | **计算** | 业需 5 态 / 研发 2 态 |
| 排期窗口 | `WindowName` | `zt_versionwindow.name` | 经 `zt_planstory` → `zt_versionwindowproduct` |
| 敏捷小组 | `TeamgroupName` | `zt_demand.teamGroup` → `zt_teamgroup.name` | 研发行显示**父业需** teamGroup |
| 负责人 | `PMs` / `AssignedToName` | 业需：`zt_demandclarify.PM` 列表；研发：`zt_story.assignedTo` | 均转 realname |
| 任务 | `TaskCount` | `zt_task` COUNT（3.9） | 业需行留空/0；研发行显示 |
| 操作 | — | — | 前端按层级显示详情/排期/改任务（TODO §七） |

---

## 七、暂不实现部分（TODO）

| 项 | 说明 |
|----|------|
| 筛选条件 SQL 拼接 | `status` / `teamgroupId` / `productId` / `keyword` / `windowId` / 快捷 `scope` chip 的 WHERE 细节；本设计仅列占位注释 |
| 分页 total 一致性 | `stage` / `scope` 在 Service 层过滤时，total 是否重算或改为「近似 total + 页内过滤」——需产品确认 |
| COUNT 与 LIST WHERE 同步 | 强制同一套 `buildBizDemandListWhere(req, poolIDs)` 生成条件，避免 total 与列表不一致 |
| 风险状态判定 | 「风险需求」chip：hang、estimateLaunch 超期、blocked 等规则 |
| 操作列按钮权限 | 详情/排期/改任务与 `perm.ScheduleXxx` 的映射 |
| 操作动作 | 跳禅道详情 URL、排期弹窗、改任务弹窗 — Handler/前端阶段 |
| 业需 WindowName 展示规则 | 子树多个窗口时的聚合文案（首个有窗口研发 / 最多出现窗口） |
| story 多窗口 | 3.8 仅取 `vw.id` 最小；多窗口编辑/展示下一阶段 |
| ProductCount 语义 | 是否含 mainSystem 自身 + clarify 多系统「+N」展示格式 |
| `ListBizDemands` API 路由 | `GET /schedule/biz-demands` + `RequirePerm(perm.ScheduleList)` — Handler 阶段 |
| 子业需 Stage 列 | 列表是否展示子业需独立 stage，或仅顶层 |

---

## 附录：已有 Repo 方法（直接复用）

| 方法 | 位置 | 用途 |
|------|------|------|
| `IsAdmin(ctx, account) (bool, error)` | `schedule/repo.go` | GetUserDemandPools 超管分支 |
| `FindTeamgroupsByIDs(ctx, ids) ([]ZtTeamgroup, error)` | `schedule/repo.go` | teamGroup 显示名 |
| `loadTeamgroupDisplayNames` | `schedule/service.go` | 父/子小组名拼接 `父 / 子` |
