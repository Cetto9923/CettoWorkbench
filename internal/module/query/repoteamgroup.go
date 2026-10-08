// =============================================================================
// 文件: internal/module/query/repoteamgroup.go
// 模块: 需求查询
// 类型: readonly
// 职责: 敏捷小组筛选下拉的选项查询（全部未删除小组，供全域筛选使用）。
// 依赖: internal/model/zentao
// =============================================================================

package query

import (
	"context"

	ztmodel "workbench/internal/model/zentao"
)

// GroupOption 敏捷小组筛选项；ID 同时用于回显与 SQL 比较。
type GroupOption struct {
	ID   uint
	Name string
}

// 研发需求没有 teamGroup 列，其敏捷小组口径继承排期页现有做法：
// 沿 fromDemand 上溯到需求树的根需求取 teamGroup，顶层需求取自身，
// 子需求取其父需求；两者皆空则该研需不参与小组筛选。
const (
	groupCaliberJoins = `LEFT JOIN zt_demand gd ON gd.id = s.fromDemand AND gd.deleted = '0'
		LEFT JOIN zt_demand gg ON gg.id = gd.parent AND gg.deleted = '0'`

	groupCaliberExpr = `CASE WHEN gd.parent IN (0, -1)
		THEN COALESCE(gd.teamGroup, '')
		ELSE COALESCE(gg.teamGroup, '')
		END`
)

// ListGroupOptions 返回全部未删除的敏捷小组。
// 按 name、id 双键排序：name 含中文时排序结果仍确定，且 id 作为并列时的兜底键，
// 避免每次刷新下拉顺序变化。
func (r *Repo) ListGroupOptions(ctx context.Context) ([]GroupOption, error) {
	var rows []ztmodel.ZtTeamgroup
	if err := r.db.WithContext(ctx).
		Table(ztmodel.ZtTeamgroup{}.TableName()).
		Select("id", "name").
		Where("deleted = ?", "0").
		Order("name ASC, id ASC").
		Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]GroupOption, 0, len(rows))
	for _, row := range rows {
		out = append(out, GroupOption{ID: row.ID, Name: row.Name})
	}
	return out, nil
}
