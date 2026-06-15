// =============================================================================
// 文件: internal/module/schedule/repo.go
// 模块: 排期工作台
// 类型: action
// 职责: 封装禅道敏捷小组、产品/系统、计划及版本窗口数据访问。
// 依赖: internal/model
// =============================================================================

package schedule

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"time"

	"gorm.io/gorm"

	"workbench/internal/model"
)

// WindowProductRow 版本窗口关联产品及计划查询结果。
type WindowProductRow struct {
	ProductID   uint   `gorm:"column:product"`
	ProductName string `gorm:"column:product_name"`
	PlanID      *uint  `gorm:"column:plan"`
	PlanSynced  uint8  `gorm:"column:planSynced"`
	PlanTitle   string `gorm:"column:plan_title"`
	PlanBegin   string `gorm:"column:plan_begin"`
	PlanEnd     string `gorm:"column:plan_end"`
}

// ZtProduct 表示禅道 zt_product 表只读字段。
type ZtProduct struct {
	ID        uint   `gorm:"column:id"`
	Name      string `gorm:"column:name"`
	Code      string `gorm:"column:code"`
	Status    string `gorm:"column:status"`
	PO        string `gorm:"column:PO"`
	QD        string `gorm:"column:QD"`
	RD        string `gorm:"column:RD"`
	CreatedBy string `gorm:"column:createdBy"`
	Whitelist string `gorm:"column:whitelist"`
}

// TableName 指定 zt_product 表。
func (ZtProduct) TableName() string {
	return "zt_product"
}

// ZtProductplan 表示禅道 zt_productplan 表只读字段。
type ZtProductplan struct {
	ID      uint   `gorm:"column:id"`
	Product uint   `gorm:"column:product"`
	Title   string `gorm:"column:title"`
	Begin   string `gorm:"column:begin"`
	End     string `gorm:"column:end"`
	Status  string `gorm:"column:status"`
}

// TableName 指定 zt_productplan 表。
func (ZtProductplan) TableName() string {
	return "zt_productplan"
}

// ZtTeamgroup 表示禅道 zt_teamgroup 表只读字段。
type ZtTeamgroup struct {
	ID     uint   `gorm:"column:id"`
	Name   string `gorm:"column:name"`
	Parent uint   `gorm:"column:parent"`
	Path   string `gorm:"column:path"`
}

// TableName 指定 zt_teamgroup 表。
func (ZtTeamgroup) TableName() string {
	return "zt_teamgroup"
}

// ZtHoliday 表示禅道 zt_holiday 表只读字段。
type ZtHoliday struct {
	ID    uint   `gorm:"column:id"`
	Name  string `gorm:"column:name"`
	Type  string `gorm:"column:type"`
	Begin string `gorm:"column:begin"`
	End   string `gorm:"column:end"`
}

// TableName 指定 zt_holiday 表。
func (ZtHoliday) TableName() string {
	return "zt_holiday"
}

// Repo 封装排期相关只读数据访问。
type Repo struct {
	db *gorm.DB
}

// NewRepo 创建 Repo。
func NewRepo(db *gorm.DB) *Repo {
	return &Repo{db: db}
}

// IsAdmin 判断账号是否为禅道超级管理员（zt_company.admins）。
func (r *Repo) IsAdmin(ctx context.Context, account string) (bool, error) {
	account = strings.TrimSpace(account)
	if account == "" {
		return false, nil
	}

	const query = `
SELECT 1 AS ok
FROM zt_company
WHERE CONCAT(',', admins, ',') LIKE CONCAT('%,', ?, ',%')
LIMIT 1`

	var row struct {
		OK int `gorm:"column:ok"`
	}
	err := r.db.WithContext(ctx).Raw(query, account).Scan(&row).Error
	if err != nil {
		return false, err
	}
	return row.OK == 1, nil
}

// ListAllTeamgroups 查询全部未删除的敏捷小组。
func (r *Repo) ListAllTeamgroups(ctx context.Context) ([]ZtTeamgroup, error) {
	var rows []ZtTeamgroup
	if err := r.db.WithContext(ctx).
		Table((ZtTeamgroup{}).TableName()).
		Select("id", "name", "parent", "path").
		Where("deleted = '0'").
		Order("id ASC").
		Find(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

// GetUserTeamgroups 查询当前用户所属的敏捷小组。
func (r *Repo) GetUserTeamgroups(ctx context.Context, account string) ([]ZtTeamgroup, error) {
	const query = `
SELECT tg.id, tg.name, tg.parent, tg.path
FROM zt_teamgroup tg
INNER JOIN zt_team t ON t.root = tg.id AND t.type = 'teamgroup'
WHERE t.account = ?
  AND tg.deleted = '0'
ORDER BY tg.id`

	var rows []ZtTeamgroup
	if err := r.db.WithContext(ctx).Raw(query, account).Scan(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

// FindTeamgroupsByIDs 按 ID 批量查询敏捷小组名称。
func (r *Repo) FindTeamgroupsByIDs(ctx context.Context, ids []uint) ([]ZtTeamgroup, error) {
	if len(ids) == 0 {
		return []ZtTeamgroup{}, nil
	}
	var rows []ZtTeamgroup
	if err := r.db.WithContext(ctx).
		Table((ZtTeamgroup{}).TableName()).
		Select("id", "name", "parent", "path").
		Where("id IN ? AND deleted = '0'", ids).
		Find(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

// GetUserProducts 查询当前用户参与的产品/系统列表。
func (r *Repo) GetUserProducts(ctx context.Context, account string) ([]ZtProduct, error) {
	const query = `
SELECT id, name, code, status, PO, QD, RD, createdBy, whitelist
FROM zt_product
WHERE deleted = '0' AND status != 'closed'
  AND (
    PO = ?
    OR QD = ?
    OR RD = ?
    OR createdBy = ?
    OR CONCAT(',', whitelist, ',') LIKE CONCAT('%,', ?, ',%')
    OR id IN (SELECT DISTINCT CAST(dc.product AS UNSIGNED) FROM zt_demandclarify dc WHERE dc.PM = ?)
  )
ORDER BY ` + "`order`" + ` ASC, id ASC`

	var rows []ZtProduct
	if err := r.db.WithContext(ctx).Raw(query, account, account, account, account, account, account).Scan(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

// ztProductplanCreate 用于向禅道 zt_productplan 插入新计划。
type ztProductplanCreate struct {
	ID           uint      `gorm:"column:id;primaryKey;autoIncrement"`
	Product      uint      `gorm:"column:product"`
	Branch       string    `gorm:"column:branch"`
	Parent       uint      `gorm:"column:parent"`
	Title        string    `gorm:"column:title"`
	Status       string    `gorm:"column:status"`
	Begin        string    `gorm:"column:begin"`
	End          string    `gorm:"column:end"`
	Order        string    `gorm:"column:order"`
	ClosedReason string    `gorm:"column:closedReason"`
	CreatedBy    string    `gorm:"column:createdBy"`
	CreatedDate  time.Time `gorm:"column:createdDate"`
	Deleted      string    `gorm:"column:deleted"`
}

// TableName 指定 zt_productplan 表。
func (ztProductplanCreate) TableName() string {
	return "zt_productplan"
}

// Transaction 在事务中执行 fn，失败时自动回滚。
func (r *Repo) Transaction(ctx context.Context, fn func(txRepo *Repo) error) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return fn(&Repo{db: tx})
	})
}

// FindByID 按 ID 查询未删除的版本窗口。
func (r *Repo) FindByID(ctx context.Context, id uint64) (*model.VersionWindow, error) {
	var window model.VersionWindow
	err := r.db.WithContext(ctx).
		Where("id = ?", id).
		First(&window).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &window, nil
}

// GetWindowConsumedHours 查询窗口关联任务的已消耗工时总和。
// 链路: zt_versionwindowproduct.plan → zt_planstory.story → zt_task.consumed
func (r *Repo) GetWindowConsumedHours(ctx context.Context, windowID uint64) (float64, error) {
	const query = `
SELECT COALESCE(SUM(t.consumed), 0) AS total
FROM zt_versionwindowproduct vwp
JOIN zt_planstory ps ON ps.plan = vwp.plan
JOIN zt_task t ON t.story = ps.story AND t.deleted = '0'
WHERE vwp.versionWindow = ? AND vwp.deletedAt IS NULL AND vwp.plan IS NOT NULL`

	var row struct {
		Total float64 `gorm:"column:total"`
	}
	if err := r.db.WithContext(ctx).Raw(query, windowID).Scan(&row).Error; err != nil {
		return 0, err
	}
	return row.Total, nil
}

// WindowStageStats 窗口需求阶段统计。
type WindowStageStats struct {
	DemandCount  int // 业需数 + 非需求池软需数
	DevCount     int // 开发中
	TestCount    int // 测试中
	DeliverCount int // 待交付
}

// GetWindowStageStats 查询窗口关联 story 的需求/开发/测试/待交付统计。
// 链路: zt_versionwindowproduct.plan → zt_planstory.story → zt_story
func (r *Repo) GetWindowStageStats(ctx context.Context, windowID uint64) (*WindowStageStats, error) {
	const query = `
SELECT
  COUNT(DISTINCT CASE WHEN s.fromDemand > 0 THEN s.fromDemand ELSE NULL END)
  + COUNT(CASE WHEN s.fromDemand = 0 THEN 1 ELSE NULL END) AS demandCount,
  SUM(CASE WHEN s.stage = 'developing' THEN 1 ELSE 0 END) AS devCount,
  SUM(CASE WHEN s.stage = 'testing' THEN 1 ELSE 0 END) AS testCount,
  SUM(CASE WHEN s.stage IN ('verified','tested','delivering','delivered') THEN 1 ELSE 0 END) AS deliverCount
FROM zt_versionwindowproduct vwp
JOIN zt_planstory ps ON ps.plan = vwp.plan
JOIN zt_story s ON s.id = ps.story AND s.deleted = '0'
WHERE vwp.versionWindow = ? AND vwp.deletedAt IS NULL AND vwp.plan IS NOT NULL`

	var row struct {
		DemandCount  int64 `gorm:"column:demandCount"`
		DevCount     int64 `gorm:"column:devCount"`
		TestCount    int64 `gorm:"column:testCount"`
		DeliverCount int64 `gorm:"column:deliverCount"`
	}
	if err := r.db.WithContext(ctx).Raw(query, windowID).Scan(&row).Error; err != nil {
		return nil, err
	}
	return &WindowStageStats{
		DemandCount:  int(row.DemandCount),
		DevCount:     int(row.DevCount),
		TestCount:    int(row.TestCount),
		DeliverCount: int(row.DeliverCount),
	}, nil
}

// GetWindowDemandCount 查询窗口关联的需求数量（业需去重 + 独立软需）。
func (r *Repo) GetWindowDemandCount(ctx context.Context, windowID uint64) (int, error) {
	const query = `
SELECT
  COUNT(DISTINCT CASE WHEN s.fromDemand > 0 THEN s.fromDemand ELSE NULL END)
  + COUNT(CASE WHEN s.fromDemand = 0 THEN 1 ELSE NULL END) AS demandCount
FROM zt_versionwindowproduct vwp
JOIN zt_planstory ps ON ps.plan = vwp.plan
JOIN zt_story s ON s.id = ps.story AND s.deleted = '0'
WHERE vwp.versionWindow = ? AND vwp.deletedAt IS NULL AND vwp.plan IS NOT NULL`

	var row struct {
		DemandCount int64 `gorm:"column:demandCount"`
	}
	if err := r.db.WithContext(ctx).Raw(query, windowID).Scan(&row).Error; err != nil {
		return 0, err
	}
	return int(row.DemandCount), nil
}

// GetWindowProducts 查询窗口关联产品及计划信息。
func (r *Repo) GetWindowProducts(ctx context.Context, windowID uint64) ([]WindowProductRow, error) {
	const query = `
SELECT
  vwp.product,
  p.name AS product_name,
  vwp.plan,
  vwp.planSynced,
  pp.title AS plan_title,
  DATE_FORMAT(pp.` + "`begin`" + `, '%Y-%m-%d') AS plan_begin,
  DATE_FORMAT(pp.` + "`end`" + `, '%Y-%m-%d') AS plan_end
FROM zt_versionwindowproduct vwp
INNER JOIN zt_product p ON p.id = vwp.product
LEFT JOIN zt_productplan pp ON pp.id = vwp.plan AND pp.deleted = '0'
WHERE vwp.versionWindow = ? AND vwp.deletedAt IS NULL
ORDER BY vwp.id ASC`

	var rows []WindowProductRow
	if err := r.db.WithContext(ctx).Raw(query, windowID).Scan(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

// Update 更新版本窗口基本信息。
func (r *Repo) Update(ctx context.Context, window *model.VersionWindow) error {
	if window == nil || window.ID == 0 {
		return errors.New("version window is invalid")
	}
	return r.db.WithContext(ctx).
		Model(window).
		Select("Name", "ReleaseDate", "StartDate", "TeamgroupID", "GroupSize", "UpdatedBy").
		Updates(window).Error
}

// DeleteWindowProducts 物理删除窗口关联的产品记录（更新时重建关联，须绕过软删以免唯一索引冲突）。
func (r *Repo) DeleteWindowProducts(ctx context.Context, windowID uint64) error {
	return r.db.WithContext(ctx).
		Unscoped().
		Where("versionWindow = ?", windowID).
		Delete(&model.VersionWindowProduct{}).Error
}

// Delete 软删除版本窗口。
func (r *Repo) Delete(ctx context.Context, id uint64) error {
	return r.db.WithContext(ctx).
		Where("id = ?", id).
		Delete(&model.VersionWindow{}).Error
}

// FindAll 查询未删除的版本窗口（count + find），按预计上线日期升序。
func (r *Repo) FindAll(ctx context.Context) ([]model.VersionWindow, int64, error) {
	query := r.db.WithContext(ctx).Model(&model.VersionWindow{})

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var rows []model.VersionWindow
	if err := query.
		Order("releaseDate ASC").
		Find(&rows).Error; err != nil {
		return nil, 0, err
	}
	return rows, total, nil
}

// ListUpcomingVersionWindowsForTeamgroups 查询指定敏捷小组未过期版本窗口（最多 limit 条）。
func (r *Repo) ListUpcomingVersionWindowsForTeamgroups(ctx context.Context, teamgroupIDs []uint, limit int) ([]model.VersionWindow, error) {
	if len(teamgroupIDs) == 0 {
		return []model.VersionWindow{}, nil
	}
	if limit <= 0 {
		limit = 4
	}

	var rows []model.VersionWindow
	if err := r.db.WithContext(ctx).
		Where("releaseDate >= CURDATE() AND teamgroup IN ?", teamgroupIDs).
		Order("releaseDate ASC").
		Limit(limit).
		Find(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

// Create 写入 zt_versionwindow 并回填自增 ID。
func (r *Repo) Create(ctx context.Context, window *model.VersionWindow) error {
	return r.db.WithContext(ctx).Create(window).Error
}

// CreateProductPlan 在禅道创建产品计划，返回新计划 ID。
func (r *Repo) CreateProductPlan(ctx context.Context, productID uint, title, begin, end, account string) (uint, error) {
	row := ztProductplanCreate{
		Product:      productID,
		Branch:       "0",
		Parent:       0,
		Title:        title,
		Status:       "wait",
		Begin:        begin,
		End:          end,
		Order:        "0",
		ClosedReason: "",
		CreatedBy:    account,
		CreatedDate:  time.Now(),
		Deleted:      "0",
	}
	if err := r.db.WithContext(ctx).Create(&row).Error; err != nil {
		return 0, err
	}
	return row.ID, nil
}

// CreateWindowProduct 写入 zt_versionwindowproduct 关联记录。
func (r *Repo) CreateWindowProduct(ctx context.Context, wp *model.VersionWindowProduct) error {
	return r.db.WithContext(ctx).Create(wp).Error
}

// GetMatchingPlans 根据产品 ID 和结束日期查询匹配的计划。
func (r *Repo) GetMatchingPlans(ctx context.Context, productID uint, endDate string) ([]ZtProductplan, error) {
	const query = `
SELECT id, product, title, begin, end, status
FROM zt_productplan
WHERE product = ? AND end = ? AND deleted = '0'
ORDER BY id DESC`

	var rows []ZtProductplan
	if err := r.db.WithContext(ctx).Raw(query, productID, endDate).Scan(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

// GetHolidays 获取与指定日期范围重叠的法定节假日。
func (r *Repo) GetHolidays(ctx context.Context, begin, end string) ([]ZtHoliday, error) {
	const query = `
SELECT id, name, type, ` + "`begin`" + `, ` + "`end`" + `
FROM zt_holiday
WHERE type = 'holiday' AND ` + "`begin`" + ` <= ? AND ` + "`end`" + ` >= ?`

	var rows []ZtHoliday
	if err := r.db.WithContext(ctx).Raw(query, end, begin).Scan(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

// GetWorkingDays 获取与指定日期范围重叠的补班日。
func (r *Repo) GetWorkingDays(ctx context.Context, begin, end string) ([]ZtHoliday, error) {
	const query = `
SELECT id, name, type, ` + "`begin`" + `, ` + "`end`" + `
FROM zt_holiday
WHERE type = 'working' AND ` + "`begin`" + ` <= ? AND ` + "`end`" + ` >= ?`

	var rows []ZtHoliday
	if err := r.db.WithContext(ctx).Raw(query, end, begin).Scan(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

type workhoursConfigRow struct {
	Key   string `gorm:"column:key"`
	Value string `gorm:"column:value"`
}

// GetWorkhoursConfig 读取 execution 模块的每日工时与周末规则配置。
func (r *Repo) GetWorkhoursConfig(ctx context.Context) (int, int, error) {
	const query = `
SELECT ` + "`key`" + `, value
FROM zt_config
WHERE module = 'execution' AND ` + "`key`" + ` IN ('defaultWorkhours', 'weekend')`

	var rows []workhoursConfigRow
	if err := r.db.WithContext(ctx).Raw(query).Scan(&rows).Error; err != nil {
		return 7, 2, err
	}

	defaultWorkhours := 7
	weekend := 2
	for _, row := range rows {
		value, err := strconv.Atoi(strings.TrimSpace(row.Value))
		if err != nil {
			continue
		}
		switch row.Key {
		case "defaultWorkhours":
			defaultWorkhours = value
		case "weekend":
			weekend = value
		}
	}
	return defaultWorkhours, weekend, nil
}

type clarifyPMRow struct {
	Demand  uint   `gorm:"column:demand"`
	Product string `gorm:"column:product"`
	PM      string `gorm:"column:PM"`
}

type clarifyProductCountRow struct {
	Demand       uint `gorm:"column:demand"`
	ProductCount int  `gorm:"column:productCount"`
}

type storyWindowRow struct {
	StoryID    uint   `gorm:"column:story"`
	WindowID   uint   `gorm:"column:windowID"`
	WindowName string `gorm:"column:windowName"`
}

type storyTaskStatRow struct {
	StoryID    uint `gorm:"column:story"`
	Total      int  `gorm:"column:total"`
	Unassigned int  `gorm:"column:unassigned"`
}

type userRealnameRow struct {
	Account  string `gorm:"column:account"`
	Realname string `gorm:"column:realname"`
}

// GetUserDemandPools 返回用户可见的需求池 ID 列表。
func (r *Repo) GetUserDemandPools(ctx context.Context, account string) ([]uint, error) {
	account = strings.TrimSpace(account)
	if account == "" {
		return []uint{}, nil
	}

	isAdmin, err := r.IsAdmin(ctx, account)
	if err != nil {
		return nil, err
	}
	if isAdmin {
		return r.listAllDemandPoolIDs(ctx)
	}
	return r.listDemandPoolIDsForUser(ctx, account)
}

func (r *Repo) listAllDemandPoolIDs(ctx context.Context) ([]uint, error) {
	const query = `
SELECT id
FROM zt_demandpool
WHERE deleted = '0'
ORDER BY id ASC`

	var ids []uint
	if err := r.db.WithContext(ctx).Raw(query).Scan(&ids).Error; err != nil {
		return nil, err
	}
	return ids, nil
}

func (r *Repo) listDemandPoolIDsForUser(ctx context.Context, account string) ([]uint, error) {
	deptIDs, err := r.loadUserDeptIDs(ctx, account)
	if err != nil {
		return nil, err
	}

	const query = `
SELECT id
FROM zt_demandpool
WHERE deleted = '0'
  AND (
    acl = 'open'
    OR FIND_IN_SET(?, participant) > 0
    OR FIND_IN_SET(?, businessReviewer) > 0
    OR dept IN ?
  )
ORDER BY id ASC`

	var ids []uint
	if err := r.db.WithContext(ctx).Raw(query, account, account, deptIDs).Scan(&ids).Error; err != nil {
		return nil, err
	}
	return ids, nil
}

func (r *Repo) loadUserDeptIDs(ctx context.Context, account string) ([]uint, error) {
	const userQuery = `
SELECT dept
FROM zt_user
WHERE account = ? AND deleted = '0'
LIMIT 1`

	var dept uint
	if err := r.db.WithContext(ctx).Raw(userQuery, account).Scan(&dept).Error; err != nil {
		return nil, err
	}
	if dept == 0 {
		return []uint{0}, nil
	}

	const pathQuery = `
SELECT path
FROM zt_dept
WHERE id = ? AND deleted = '0'
LIMIT 1`

	var path string
	if err := r.db.WithContext(ctx).Raw(pathQuery, dept).Scan(&path).Error; err != nil {
		return nil, err
	}

	deptIDs := parseDeptPathIDs(path)
	seen := make(map[uint]struct{}, len(deptIDs)+1)
	unique := make([]uint, 0, len(deptIDs)+1)
	for _, id := range append(deptIDs, dept) {
		if id == 0 {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		unique = append(unique, id)
	}
	if len(unique) == 0 {
		return []uint{dept}, nil
	}
	return unique, nil
}

func parseDeptPathIDs(path string) []uint {
	path = strings.Trim(path, ",")
	if path == "" {
		return []uint{}
	}
	parts := strings.Split(path, ",")
	ids := make([]uint, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		var id uint
		for i := 0; i < len(part); i++ {
			if part[i] < '0' || part[i] > '9' {
				id = 0
				break
			}
			id = id*10 + uint(part[i]-'0')
		}
		if id > 0 {
			ids = append(ids, id)
		}
	}
	return ids
}

// ListBizDemands 顶层业需分页主查询（parent=0）。
func (r *Repo) ListBizDemands(ctx context.Context, req ListBizDemandsReq, poolIDs []uint) ([]ZtDemand, int64, error) {
	if len(poolIDs) == 0 {
		return []ZtDemand{}, 0, nil
	}

	const countQuery = `
SELECT COUNT(*) AS total
FROM zt_demand
WHERE deleted = '0'
  AND parent = 0
  AND pool IN ?
  -- AND status = ?           -- TODO：status 筛选
  -- AND teamGroup = ?        -- TODO：teamgroupId 筛选
  -- AND mainSystem = ?       -- TODO：productId 筛选（主系统）
  -- AND (id = ? OR name LIKE ?)  -- TODO：keyword`

	var total int64
	if err := r.db.WithContext(ctx).Raw(countQuery, poolIDs).Scan(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (req.Page - 1) * req.PageSize
	const listQuery = `
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
  AND pool IN ?
  -- AND status = ?           -- TODO：status 筛选
  -- AND teamGroup = ?        -- TODO：teamgroupId 筛选
  -- AND mainSystem = ?       -- TODO：productId 筛选（主系统）
  -- AND (id = ? OR name LIKE ?)  -- TODO：keyword
ORDER BY id DESC
LIMIT ? OFFSET ?`

	var rows []ZtDemand
	if err := r.db.WithContext(ctx).Raw(listQuery, poolIDs, req.PageSize, offset).Scan(&rows).Error; err != nil {
		return nil, 0, err
	}
	return rows, total, nil
}

// FindChildDemandsByParents 批量查询子业需。
func (r *Repo) FindChildDemandsByParents(ctx context.Context, parentIDs []uint) ([]ZtDemand, error) {
	if len(parentIDs) == 0 {
		return []ZtDemand{}, nil
	}

	const query = `
SELECT
  id, name, pri, status, mainSystem, teamGroup,
  BRA, QD, RD, createdBy, pool, parent, hang, category, estimateLaunch
FROM zt_demand
WHERE deleted = '0'
  AND parent IN ?
ORDER BY parent ASC, id ASC`

	var rows []ZtDemand
	if err := r.db.WithContext(ctx).Raw(query, parentIDs).Scan(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

// FindStoriesByDemands 按业需 ID 批量查研发需求。
func (r *Repo) FindStoriesByDemands(ctx context.Context, demandIDs []uint) ([]ZtStory, error) {
	if len(demandIDs) == 0 {
		return []ZtStory{}, nil
	}

	const query = `
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
WHERE fromDemand IN ?
  AND type = 'story'
  AND deleted = '0'
ORDER BY fromDemand ASC, isMainSystemAssociation DESC, id ASC`

	var rows []ZtStory
	if err := r.db.WithContext(ctx).Raw(query, demandIDs).Scan(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

// CountClarifyProductsByDemands 统计业需关联的多系统数。
func (r *Repo) CountClarifyProductsByDemands(ctx context.Context, demandIDs []uint) (map[uint]int, error) {
	if len(demandIDs) == 0 {
		return map[uint]int{}, nil
	}

	const query = `
SELECT demand, COUNT(DISTINCT product) AS productCount
FROM zt_demandclarify
WHERE demand IN ?
GROUP BY demand`

	var rows []clarifyProductCountRow
	if err := r.db.WithContext(ctx).Raw(query, demandIDs).Scan(&rows).Error; err != nil {
		return nil, err
	}
	out := make(map[uint]int, len(rows))
	for _, row := range rows {
		out[row.Demand] = row.ProductCount
	}
	return out, nil
}

// FindClarifyPMsByDemands 查询业需澄清 PM，按 demand 分组。
func (r *Repo) FindClarifyPMsByDemands(ctx context.Context, demandIDs []uint) (map[uint][]ClarifyPM, error) {
	if len(demandIDs) == 0 {
		return map[uint][]ClarifyPM{}, nil
	}

	const query = `
SELECT demand, product, PM
FROM zt_demandclarify
WHERE demand IN ?
ORDER BY demand ASC, product ASC`

	var rows []clarifyPMRow
	if err := r.db.WithContext(ctx).Raw(query, demandIDs).Scan(&rows).Error; err != nil {
		return nil, err
	}
	out := make(map[uint][]ClarifyPM, len(rows))
	for _, row := range rows {
		out[row.Demand] = append(out[row.Demand], ClarifyPM{
			Demand:  row.Demand,
			Product: row.Product,
			PM:      strings.TrimSpace(row.PM),
		})
	}
	return out, nil
}

// FindProductsByIDs 批量查产品/系统名称。
func (r *Repo) FindProductsByIDs(ctx context.Context, productIDs []uint) (map[uint]string, error) {
	if len(productIDs) == 0 {
		return map[uint]string{}, nil
	}

	const query = `
SELECT id, name
FROM zt_product
WHERE id IN ?
  AND deleted = '0'`

	var rows []ZtProduct
	if err := r.db.WithContext(ctx).Raw(query, productIDs).Scan(&rows).Error; err != nil {
		return nil, err
	}
	out := make(map[uint]string, len(rows))
	for _, row := range rows {
		out[row.ID] = strings.TrimSpace(row.Name)
	}
	return out, nil
}

// FindStoryWindowMappings 查研发需求关联的版本窗口（每 story 取 vw.id 最小的一条）。
func (r *Repo) FindStoryWindowMappings(ctx context.Context, storyIDs []uint) (map[uint]StoryWindowRef, error) {
	if len(storyIDs) == 0 {
		return map[uint]StoryWindowRef{}, nil
	}

	const query = `
SELECT ps.story, vw.id AS windowID, vw.name AS windowName
FROM zt_planstory ps
INNER JOIN zt_versionwindowproduct vwp
  ON vwp.plan = ps.plan AND vwp.deletedAt IS NULL
INNER JOIN zt_versionwindow vw
  ON vw.id = vwp.versionWindow AND vw.deletedAt IS NULL
WHERE ps.story IN ?
ORDER BY ps.story ASC, vw.id ASC`

	var rows []storyWindowRow
	if err := r.db.WithContext(ctx).Raw(query, storyIDs).Scan(&rows).Error; err != nil {
		return nil, err
	}
	out := make(map[uint]StoryWindowRef, len(rows))
	for _, row := range rows {
		if _, exists := out[row.StoryID]; exists {
			continue
		}
		out[row.StoryID] = StoryWindowRef{
			StoryID:    row.StoryID,
			WindowID:   row.WindowID,
			WindowName: strings.TrimSpace(row.WindowName),
		}
	}
	return out, nil
}

// CountStoryTasks 统计研发需求下任务总数与未指派数。
func (r *Repo) CountStoryTasks(ctx context.Context, storyIDs []uint) (map[uint]StoryTaskStat, error) {
	if len(storyIDs) == 0 {
		return map[uint]StoryTaskStat{}, nil
	}

	const query = `
SELECT
  story,
  COUNT(*) AS total,
  SUM(CASE WHEN assignedTo IS NULL OR assignedTo = '' THEN 1 ELSE 0 END) AS unassigned
FROM zt_task
WHERE story IN ?
  AND deleted = '0'
GROUP BY story`

	var rows []storyTaskStatRow
	if err := r.db.WithContext(ctx).Raw(query, storyIDs).Scan(&rows).Error; err != nil {
		return nil, err
	}
	out := make(map[uint]StoryTaskStat, len(rows))
	for _, row := range rows {
		out[row.StoryID] = StoryTaskStat{
			StoryID:    row.StoryID,
			Total:      row.Total,
			Unassigned: row.Unassigned,
		}
	}
	return out, nil
}

// FindUsersByAccounts 批量查用户 realname。
func (r *Repo) FindUsersByAccounts(ctx context.Context, accounts []string) (map[string]string, error) {
	if len(accounts) == 0 {
		return map[string]string{}, nil
	}

	const query = `
SELECT account, realname
FROM zt_user
WHERE account IN ?
  AND deleted = '0'`

	var rows []userRealnameRow
	if err := r.db.WithContext(ctx).Raw(query, accounts).Scan(&rows).Error; err != nil {
		return nil, err
	}
	out := make(map[string]string, len(rows))
	for _, row := range rows {
		account := strings.TrimSpace(row.Account)
		if account == "" {
			continue
		}
		out[account] = strings.TrimSpace(row.Realname)
	}
	return out, nil
}
