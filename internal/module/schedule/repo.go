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
	"time"

	"gorm.io/gorm"

	"workbench/internal/model"
)

// WindowProductRow 版本窗口关联产品及计划查询结果。
type WindowProductRow struct {
	ProductID   uint   `gorm:"column:product_id"`
	ProductName string `gorm:"column:product_name"`
	PlanID      *uint  `gorm:"column:plan_id"`
	PlanSynced  uint8  `gorm:"column:plan_synced"`
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
	PMT       string `gorm:"column:PMT"`
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

// Repo 封装排期相关只读数据访问。
type Repo struct {
	db *gorm.DB
}

// NewRepo 创建 Repo。
func NewRepo(db *gorm.DB) *Repo {
	return &Repo{db: db}
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
SELECT id, name, code, status, PO, QD, RD, createdBy, whitelist, PMT
FROM zt_product
WHERE deleted = '0' AND status != 'closed'
  AND (
    PO = ?
    OR QD = ?
    OR RD = ?
    OR createdBy = ?
    OR CONCAT(',', whitelist, ',') LIKE CONCAT('%,', ?, ',%')
    OR CONCAT(',', PMT, ',') LIKE CONCAT('%,', ?, ',%')
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

// GetVersionWindowByID 按 ID 查询未删除的版本窗口。
func (r *Repo) GetVersionWindowByID(ctx context.Context, id uint64) (*model.VersionWindow, error) {
	var window model.VersionWindow
	err := r.db.WithContext(ctx).
		Where("id = ? AND deleted = ?", id, 0).
		First(&window).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &window, nil
}

// GetWindowProducts 查询窗口关联产品及计划信息。
func (r *Repo) GetWindowProducts(ctx context.Context, windowID uint64) ([]WindowProductRow, error) {
	const query = `
SELECT
  vwp.product_id,
  p.name AS product_name,
  vwp.plan_id,
  vwp.plan_synced,
  pp.title AS plan_title,
  pp.begin AS plan_begin,
  pp.end AS plan_end
FROM version_window_product vwp
INNER JOIN zt_product p ON p.id = vwp.product_id
LEFT JOIN zt_productplan pp ON pp.id = vwp.plan_id AND pp.deleted = '0'
WHERE vwp.window_id = ?
ORDER BY vwp.id ASC`

	var rows []WindowProductRow
	if err := r.db.WithContext(ctx).Raw(query, windowID).Scan(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

// UpdateVersionWindow 更新版本窗口基本信息。
func (r *Repo) UpdateVersionWindow(ctx context.Context, window *model.VersionWindow) error {
	if window == nil || window.ID == 0 {
		return errors.New("version window is invalid")
	}
	return r.db.WithContext(ctx).
		Model(window).
		Select("Name", "ReleaseDate", "StartDate", "TeamgroupID", "GroupSize").
		Updates(window).Error
}

// DeleteWindowProducts 删除窗口关联的产品记录。
func (r *Repo) DeleteWindowProducts(ctx context.Context, windowID uint64) error {
	return r.db.WithContext(ctx).
		Where("window_id = ?", windowID).
		Delete(&model.VersionWindowProduct{}).Error
}

// SoftDeleteVersionWindow 软删除版本窗口。
func (r *Repo) SoftDeleteVersionWindow(ctx context.Context, id uint64) error {
	return r.db.WithContext(ctx).
		Model(&model.VersionWindow{}).
		Where("id = ? AND deleted = ?", id, 0).
		Update("deleted", 1).Error
}

// ListVersionWindows 查询未删除的版本窗口，按预计上线日期升序。
func (r *Repo) ListVersionWindows(ctx context.Context) ([]model.VersionWindow, error) {
	var rows []model.VersionWindow
	if err := r.db.WithContext(ctx).
		Where("deleted = ?", 0).
		Order("release_date ASC").
		Find(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

// CreateVersionWindow 写入 version_window 并回填自增 ID。
func (r *Repo) CreateVersionWindow(ctx context.Context, window *model.VersionWindow) error {
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

// CreateWindowProduct 写入 version_window_product 关联记录。
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
