// =============================================================================
// 文件: internal/module/kanban/repo.go
// 模块: 工作看板
// 类型: readonly
// 职责: 需求看板只读数据访问（敏捷小组与成员）。
// 依赖: internal/model/zentao
// =============================================================================

package kanban

import (
	"context"
	"strings"

	"gorm.io/gorm"

	ztmodel "workbench/internal/model/zentao"
)

// Repo 封装看板只读查询。
type Repo struct {
	db *gorm.DB
}

// NewRepo 创建 Repo。
func NewRepo(db *gorm.DB) *Repo {
	return &Repo{db: db}
}

// teamgroupRow 用户所属小组原始行（含 PO / 敏捷教练）。
type teamgroupRow struct {
	ID      uint
	Name    string
	PO      string
	Manager string
}

// teamMemberRow 小组成员原始行。
type teamMemberRow struct {
	Root    uint
	Account string
	Order   int8
}

// ListUserTeamgroups 查询账号所属敏捷小组，按加入时间倒序。
func (r *Repo) ListUserTeamgroups(ctx context.Context, account string) ([]teamgroupRow, error) {
	account = strings.TrimSpace(account)
	if account == "" {
		return []teamgroupRow{}, nil
	}

	var rows []teamgroupRow
	err := r.db.WithContext(ctx).
		Table((ztmodel.ZtTeam{}).TableName()+" AS t").
		Select("g.id, g.name, g.PO, g.manager").
		Joins("LEFT JOIN "+(ztmodel.ZtTeamgroup{}).TableName()+" AS g ON t.root = g.id").
		Where("t.account = ? AND t.type = ? AND g.deleted = ?", account, "teamgroup", "0").
		Order("t.`join` DESC").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	if rows == nil {
		return []teamgroupRow{}, nil
	}
	return rows, nil
}

// ListTeamMembersByRoots 批量查询敏捷小组成员账号（按 order 升序）。
func (r *Repo) ListTeamMembersByRoots(ctx context.Context, roots []uint) ([]teamMemberRow, error) {
	if len(roots) == 0 {
		return []teamMemberRow{}, nil
	}

	var rows []teamMemberRow
	err := r.db.WithContext(ctx).
		Table((ztmodel.ZtTeam{}).TableName()).
		Select("root, account, `order`").
		Where("root IN ? AND type = ?", roots, "teamgroup").
		Order("`order` ASC, id ASC").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	if rows == nil {
		return []teamMemberRow{}, nil
	}
	return rows, nil
}
