// =============================================================================
// 文件: internal/module/po/repostar.go
// 模块: PO 工作台
// 类型: action
// 职责: 「我关注的」业需候选集（角色价值流 ∩ zt_starinfo）。
// 依赖: 无
// =============================================================================

package po

import (
	"context"

	"gorm.io/gorm"
)

// starredDemandScope 口径 A：roleDemandScope（角色+阶段）再叠加 zt_starinfo 关注。
func (r *Repo) starredDemandScope(ctx context.Context, account string, filter mysqlStageFilter) *gorm.DB {
	return r.roleDemandScope(ctx, account, filter).Where(`id IN (
			SELECT objectID FROM zt_starinfo
			WHERE objectType = ? AND account = ? AND followed = ?
		)`, "demand", account, "1")
}

// FindStarredDemandIDs 按价值流阶段查询当前账号关注的业需 ID（供角标去重）。
func (r *Repo) FindStarredDemandIDs(ctx context.Context, account string, filter mysqlStageFilter) ([]int, error) {
	if r == nil || r.db == nil || !filterReady(account, filter) {
		return nil, nil
	}
	var ids []int
	err := r.starredDemandScope(ctx, account, filter).Pluck("zt_demand.id", &ids).Error
	if err != nil {
		return nil, err
	}
	return ids, nil
}

// CountStarredDemands 按阶段统计当前账号关注的业需数量。
func (r *Repo) CountStarredDemands(ctx context.Context, account string, filter mysqlStageFilter) (int64, error) {
	if r == nil || r.db == nil || !filterReady(account, filter) {
		return 0, nil
	}
	var total int64
	err := r.starredDemandScope(ctx, account, filter).Count(&total).Error
	return total, err
}

// FindStarredDemands 按阶段查询当前账号关注的业需列表（字段对齐 FindRoleDemands）。
// limit>0 时分页；limit<=0 表示不分页。
func (r *Repo) FindStarredDemands(ctx context.Context, account string, filter mysqlStageFilter, limit, offset int) ([]DemandRow, error) {
	if r == nil || r.db == nil || !filterReady(account, filter) {
		return nil, nil
	}
	q := applyDemandListOrder(
		r.starredDemandScope(ctx, account, filter).
			Select(`zt_demand.id, zt_demand.name, zt_demand.pri, zt_demand.status, zt_demand.createdBy,
			zt_demand.assignedTo, zt_demand.QD, zt_demand.RD, zt_demand.BRA, zt_demand.mainSystem,
			zt_demand.hang, clarify_pm.PM AS pm`).
			Joins(`LEFT JOIN (
			SELECT demand, GROUP_CONCAT(PM) AS PM
			FROM zt_demandclarify
			WHERE PM IS NOT NULL AND PM <> ''
			GROUP BY demand
		) AS clarify_pm ON clarify_pm.demand = zt_demand.id`),
		filter,
	)
	if limit > 0 {
		if offset < 0 {
			offset = 0
		}
		q = q.Limit(limit).Offset(offset)
	}
	var rows []DemandRow
	err := q.Find(&rows).Error
	if err != nil {
		return nil, err
	}
	return rows, nil
}
