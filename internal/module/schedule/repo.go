// =============================================================================
// 文件: internal/module/schedule/repo.go
// 模块: 排期工作台
// 类型: action
// 职责: Repo 主入口 + 基础类型 + 用户/权限查询 + 窗口主表 CRUD + 用户辅助 FindUsersByAccounts。
// 依赖: internal/model
// =============================================================================

package schedule

import (
	"context"
	"errors"
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

// ListAllProductIDs 返回全部未关闭产品 ID（超管可见范围）。
func (r *Repo) ListAllProductIDs(ctx context.Context) ([]uint, error) {
	const query = `
SELECT id
FROM zt_product
WHERE deleted = '0' AND status != 'closed'
ORDER BY id ASC`

	var ids []uint
	if err := r.db.WithContext(ctx).Raw(query).Scan(&ids).Error; err != nil {
		return nil, err
	}
	return ids, nil
}

// ListAllProducts 返回全部未删除产品（筛选下拉用）。
func (r *Repo) ListAllProducts(ctx context.Context) ([]ZtProduct, error) {
	const query = `
SELECT id, name
FROM zt_product
WHERE deleted = '0'
ORDER BY id DESC`

	var rows []ZtProduct
	if err := r.db.WithContext(ctx).Raw(query).Scan(&rows).Error; err != nil {
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
    OR CONCAT(',', whitelist, ',') LIKE CONCAT('%,', ?, ',%')
    OR id IN (SELECT DISTINCT CAST(dc.product AS UNSIGNED) FROM zt_demandclarify dc WHERE dc.PM = ?)
  )
ORDER BY ` + "`order`" + ` ASC, id ASC`

	var rows []ZtProduct
	if err := r.db.WithContext(ctx).Raw(query, account, account, account, account, account).Scan(&rows).Error; err != nil {
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
