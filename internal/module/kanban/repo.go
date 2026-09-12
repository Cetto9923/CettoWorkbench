// =============================================================================
// 文件: internal/module/kanban/repo.go
// 模块: 工作看板
// 类型: readonly
// 职责: 需求看板只读数据访问（敏捷小组等）。
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

// ListUserTeamgroups 查询账号所属敏捷小组，按加入时间倒序。
func (r *Repo) ListUserTeamgroups(ctx context.Context, account string) ([]TeamgroupItem, error) {
	account = strings.TrimSpace(account)
	if account == "" {
		return []TeamgroupItem{}, nil
	}

	var rows []TeamgroupItem
	err := r.db.WithContext(ctx).
		Table((ztmodel.ZtTeam{}).TableName()+" AS t").
		Select("g.id, g.name").
		Joins("LEFT JOIN "+(ztmodel.ZtTeamgroup{}).TableName()+" AS g ON t.root = g.id").
		Where("t.account = ? AND t.type = ? AND g.deleted = ?", account, "teamgroup", "0").
		Order("t.`join` DESC").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	if rows == nil {
		return []TeamgroupItem{}, nil
	}
	return rows, nil
}
