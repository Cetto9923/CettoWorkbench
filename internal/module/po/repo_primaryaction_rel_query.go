// =============================================================================
// 文件: internal/module/po/repo_primaryaction_rel_query.go
// 模块: PO 工作台
// 类型: repo
// 职责: 团队长管辖口径的批量查询。语义与 demandauthz.CheckLeaderDemandVisibility
//       逐条判定严格一致（同样 11 个关系列 + 澄清 PM），但一次 IN 查询覆盖整批，
//       避免按需求逐条判定造成 N+1。
// =============================================================================

package po

import (
	"context"
	"strings"

	"gorm.io/gorm"

	"workbench/internal/module/demandauthz"
)

// newDemandAuthzRepo 基于同一 db 构造 demandauthz 仓储，
// 保证关系口径与详情页 / 写权限校验完全同源。
func newDemandAuthzRepo(db *gorm.DB) *demandauthz.Repo {
	if db == nil {
		return nil
	}
	return demandauthz.New(db)
}

// DemandIDsRelatedToAccount 返回「account 命中任一干系人关系」的需求 ID 集合。
// 关系列与 demandauthz.CheckDemandVisibility 保持逐条一致，但一次 IN 查询覆盖整批，
// 避免逐需求判定造成 N+1。
func (r *Repo) DemandIDsRelatedToAccount(ctx context.Context, demandIDs []uint, account string) ([]uint, error) {
	if r == nil || r.db == nil || len(demandIDs) == 0 || strings.TrimSpace(account) == "" {
		return nil, nil
	}
	var rows []uint
	err := r.db.WithContext(ctx).Raw(`
SELECT d.id FROM zt_demand d
WHERE d.deleted = '0' AND d.id IN ?
  AND (
    d.originator = ?
    OR d.assignedTo = ?
    OR d.QD = ?
    OR d.RD = ?
    OR d.BRA = ?
    OR d.accepter = ?
    OR d.reviewer = ?
    OR d.createdBy = ?
    OR d.closedBy = ?
    OR d.editedBy = ?
    OR d.feedbackedBy = ?
    OR EXISTS (
      SELECT 1 FROM zt_demandclarify c
      WHERE c.demand = d.id AND c.PM = ?
    )
  )`, demandIDs, account, account, account, account, account, account,
		account, account, account, account, account, account).Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	return rows, nil
}

// DemandIDsManagedByDepts 返回「干系人所属部门落在 deptIDs 内」的需求 ID 集合。
// 关系列与 demandauthz.CheckLeaderDemandVisibility 保持逐条一致。
func (r *Repo) DemandIDsManagedByDepts(ctx context.Context, demandIDs []uint, deptIDs []uint) ([]uint, error) {
	if r == nil || r.db == nil || len(demandIDs) == 0 || len(deptIDs) == 0 {
		return nil, nil
	}
	var rows []uint
	err := r.db.WithContext(ctx).Raw(`
SELECT DISTINCT d.id FROM zt_demand d
WHERE d.deleted = '0' AND d.id IN ?
  AND EXISTS (
    SELECT 1 FROM zt_user u
    WHERE u.deleted = '0' AND u.dept IN (?)
      AND (
        u.account = d.originator
        OR u.account = d.assignedTo
        OR u.account = d.QD
        OR u.account = d.RD
        OR u.account = d.BRA
        OR u.account = d.accepter
        OR u.account = d.createdBy
        OR u.account = d.closedBy
        OR u.account = d.editedBy
        OR u.account = d.feedbackedBy
        OR FIND_IN_SET(u.account, REPLACE(d.reviewer, ' ', '')) > 0
        OR EXISTS (
          SELECT 1 FROM zt_demandclarify c
          WHERE c.demand = d.id AND c.PM = u.account
        )
      )
  )`, demandIDs, deptIDs).Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	return rows, nil
}
