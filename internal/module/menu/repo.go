// =============================================================================
// 文件: internal/module/menu/repo.go
// 模块: 菜单管理
// 类型: crud
// 职责: 封装菜单与用户固定菜单数据访问。
// 依赖: internal/model
// =============================================================================

package menu

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"

	"workbench/internal/model"
)

// Repo 封装菜单数据访问。
type Repo struct {
	db *gorm.DB
}

// NewRepo 创建 Repo。
func NewRepo(db *gorm.DB) *Repo {
	return &Repo{db: db}
}

// FindAll 查询全部未删除菜单（分页三元组返回）。
func (r *Repo) FindAll(ctx context.Context, req RepoFindAllReq) ([]model.Menu, int64, error) {
	query := r.db.WithContext(ctx).
		Model(&model.Menu{}).
		Where("deletedAt IS NULL")

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var rows []model.Menu
	if err := query.
		Order("sort ASC, id ASC").
		Offset((req.Page - 1) * req.PageSize).
		Limit(req.PageSize).
		Find(&rows).Error; err != nil {
		return nil, 0, err
	}
	return rows, total, nil
}

// FindByID 按 ID 查询菜单。
func (r *Repo) FindByID(ctx context.Context, id uint64) (*model.Menu, error) {
	var row model.Menu
	if err := r.db.WithContext(ctx).
		Model(&model.Menu{}).
		Where("id = ? AND deletedAt IS NULL", id).
		First(&row).Error; err != nil {
		return nil, err
	}
	return &row, nil
}

// Create 创建菜单。
func (r *Repo) Create(ctx context.Context, m *model.Menu) error {
	return r.db.WithContext(ctx).Create(m).Error
}

// Update 更新菜单。
func (r *Repo) Update(ctx context.Context, m *model.Menu) error {
	return r.db.WithContext(ctx).
		Model(&model.Menu{}).
		Where("id = ? AND deletedAt IS NULL", m.ID).
		Updates(map[string]any{
			"parentId":  m.ParentID,
			"type":      m.Type,
			"title":     m.Title,
			"icon":      m.Icon,
			"path":      m.Path,
			"perm":      m.Perm,
			"sort":      m.Sort,
			"updatedAt": m.UpdatedAt,
		}).Error
}

// Delete 软删除菜单。
func (r *Repo) Delete(ctx context.Context, id uint64) error {
	now := time.Now()
	return r.db.WithContext(ctx).
		Model(&model.Menu{}).
		Where("id = ? AND deletedAt IS NULL", id).
		Updates(map[string]any{
			"deletedAt": now,
			"updatedAt": now,
		}).Error
}

// FindChildren 查询某父菜单下的直接子菜单。
func (r *Repo) FindChildren(ctx context.Context, parentID uint64) ([]*model.Menu, error) {
	var rows []*model.Menu
	err := r.db.WithContext(ctx).
		Model(&model.Menu{}).
		Where("parentId = ? AND deletedAt IS NULL", parentID).
		Order("sort ASC, id ASC").
		Find(&rows).Error
	return rows, err
}

// UserMenuJoined 固定记录 JOIN 菜单后的投影。
type UserMenuJoined struct {
	ID     uint64 `gorm:"column:id"`
	MenuID uint64 `gorm:"column:menuId"`
	Sort   int    `gorm:"column:sort"`
	Title  string `gorm:"column:title"`
	Path   string `gorm:"column:path"`
	Icon   string `gorm:"column:icon"`
	Perm   string `gorm:"column:perm"`
}

// FindUserMenusByAccount 查询账号下固定菜单（JOIN zt_menus，按 sort/id）。
func (r *Repo) FindUserMenusByAccount(ctx context.Context, account string) ([]UserMenuJoined, error) {
	var rows []UserMenuJoined
	err := r.db.WithContext(ctx).
		Table("zt_usermenu AS um").
		Select("um.id AS id, um.menuId AS menuId, um.sort AS sort, m.title AS title, m.path AS path, m.icon AS icon, m.perm AS perm").
		Joins("INNER JOIN zt_menus AS m ON m.id = um.menuId AND m.deletedAt IS NULL").
		Where("um.account = ?", account).
		Where("m.path <> ''").
		Order("um.sort DESC, um.id DESC").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	return rows, nil
}

// FindPinnableMenus 查询可固定的叶子菜单（有 path、未删除）。
func (r *Repo) FindPinnableMenus(ctx context.Context) ([]model.Menu, error) {
	var rows []model.Menu
	err := r.db.WithContext(ctx).
		Model(&model.Menu{}).
		Where("deletedAt IS NULL AND path <> '' AND path <> ?", "/home").
		Order("LENGTH(path) DESC, id ASC").
		Find(&rows).Error
	return rows, err
}

// FindUserMenuByAccountMenuID 查询账号+菜单是否已固定。
func (r *Repo) FindUserMenuByAccountMenuID(ctx context.Context, account string, menuID uint64) (*model.UserMenu, error) {
	var row model.UserMenu
	err := r.db.WithContext(ctx).
		Where("account = ? AND menuId = ?", account, menuID).
		First(&row).Error
	if err != nil {
		return nil, err
	}
	return &row, nil
}

// MaxUserMenuSort 当前账号固定项的最大 sort。
func (r *Repo) MaxUserMenuSort(ctx context.Context, account string) (int, error) {
	var maxSort *int
	err := r.db.WithContext(ctx).
		Model(&model.UserMenu{}).
		Select("MAX(sort)").
		Where("account = ?", account).
		Scan(&maxSort).Error
	if err != nil {
		return 0, err
	}
	if maxSort == nil {
		return 0, nil
	}
	return *maxSort, nil
}

// CreateUserMenu 新增固定记录。
func (r *Repo) CreateUserMenu(ctx context.Context, m *model.UserMenu) (int64, error) {
	if err := r.db.WithContext(ctx).Create(m).Error; err != nil {
		return 0, err
	}
	return int64(m.ID), nil
}

// DeleteUserMenu 按账号+菜单真删除。
func (r *Repo) DeleteUserMenu(ctx context.Context, account string, menuID uint64) error {
	res := r.db.WithContext(ctx).
		Where("account = ? AND menuId = ?", account, menuID).
		Delete(&model.UserMenu{})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

// isNotFound 判断是否为记录不存在。
func isNotFound(err error) bool {
	return errors.Is(err, gorm.ErrRecordNotFound)
}
