// =============================================================================
// 文件: internal/module/schedule/repo_zentao.go
// 模块: 排期工作台
// 类型: action
// 职责: 禅道基础:共用 row struct + 需求池权限 + 业需(zt_demand)查询。
// 依赖: (无项目内部包)
// =============================================================================

package schedule

import (
	"context"
	"strings"
)

type clarifyProductCountRow struct {
	Demand       uint `gorm:"column:demand"`
	ProductCount int  `gorm:"column:productCount"`
}

type storyWindowRow struct {
	StoryID     uint   `gorm:"column:story"`
	WindowID    uint   `gorm:"column:windowID"`
	WindowName  string `gorm:"column:windowName"`
	TeamgroupID uint   `gorm:"column:teamgroupID"`
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
WHERE id = ?
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

// ListBizDemands 顶层业需分页主查询（parent IN (0,-1)：含被拆分的父需求）。
func (r *Repo) ListBizDemands(ctx context.Context, req ListBizDemandsReq, poolIDs []uint, account string) ([]ZtDemand, int64, error) {
	if len(poolIDs) == 0 {
		return []ZtDemand{}, 0, nil
	}

	clause := mergeFilterClauses(
		applyBizDemandSuspended(buildBizDemandFilterClause(req.Filter, account), req.Suspended),
		buildBizDemandAdvancedClause(advancedFilterParamsFromBizReq(req)),
	)
	countArgs := append([]interface{}{poolIDs}, clause.args...)
	const countQuery = `
SELECT COUNT(*) AS total
FROM zt_demand d
WHERE d.deleted = '0'
  AND d.parent IN (0, -1)
  AND d.pool IN ?`

	var total int64
	if err := r.db.WithContext(ctx).Raw(countQuery+clause.sql, countArgs...).Scan(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (req.Page - 1) * req.PageSize
	listArgs := append([]interface{}{poolIDs}, clause.args...)
	listArgs = append(listArgs, req.PageSize, offset)
	const listQuery = `
SELECT
  d.id,
  d.name,
  d.pri,
  d.status,
  d.assignedTo,
  d.mainSystem,
  d.teamGroup,
  d.BRA,
  d.QD,
  d.RD,
  d.createdBy,
  d.pool,
  d.parent,
  d.hang,
  d.category,
  d.estimateLaunch
FROM zt_demand d
WHERE d.deleted = '0'
  AND d.parent IN (0, -1)
  AND d.pool IN ?`

	var rows []ZtDemand
	if err := r.db.WithContext(ctx).Raw(listQuery+clause.sql+`
ORDER BY d.id DESC
LIMIT ? OFFSET ?`, listArgs...).Scan(&rows).Error; err != nil {
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
  id, name, pri, status, assignedTo, mainSystem, teamGroup,
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
