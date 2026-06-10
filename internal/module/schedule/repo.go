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
	"time"

	"gorm.io/gorm"

	"workbench/internal/model"
)

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
