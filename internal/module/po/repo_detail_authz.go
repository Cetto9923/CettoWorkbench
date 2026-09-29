// =============================================================================
// 文件: internal/module/po/repo_detail_authz.go
// 模块: PO 工作台
// 类型: repo
// 职责: 业务需求详情的组织角色与团队长可见范围持久层查询（F01）。
// =============================================================================

package po

import (
	"context"
	"fmt"
	"regexp"
	"strings"
)

// IsPMORole 查询用户是否具备激活状态的 PMO 组织角色。
func (r *DemandDetailRepo) IsPMORole(ctx context.Context, userID int64) (bool, error) {
	if r == nil || r.db == nil || userID <= 0 {
		return false, nil
	}
	var count int64
	err := r.db.WithContext(ctx).Raw(`
SELECT COUNT(*) FROM zt_gf_user_roles ur
JOIN zt_roles r ON r.id = ur.roleId
WHERE ur.userId = ? AND ur.deleted = '0' AND r.deleted = '0' AND r.isActive = 1 AND r.code = 'pmo'`,
		userID).Scan(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// IsPORole 查询用户是否具备激活状态的 PO / 产品经理 组织角色。
func (r *DemandDetailRepo) IsPORole(ctx context.Context, userID int64) (bool, error) {
	if r == nil || r.db == nil || userID <= 0 {
		return false, nil
	}
	var count int64
	err := r.db.WithContext(ctx).Raw(`
SELECT COUNT(*) FROM zt_gf_user_roles ur
JOIN zt_roles r ON r.id = ur.roleId
WHERE ur.userId = ? AND ur.deleted = '0' AND r.deleted = '0' AND r.isActive = 1 AND r.code = 'po'`,
		userID).Scan(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// IsDeptManager 判断账号是否为某部门/部室负责人（zt_dept.manager 或 补缺表覆盖）。
func (r *DemandDetailRepo) IsDeptManager(ctx context.Context, account string) (bool, error) {
	account = strings.TrimSpace(account)
	if account == "" {
		return false, nil
	}
	ids, err := r.ListLeaderDeptTreeIDs(ctx, account)
	if err != nil {
		return false, err
	}
	return len(ids) > 0, nil
}

// ListLeaderDeptTreeIDs 返回账号负责的部门及其下级部门 ID。
// 针对 52 科技部本部及其子孙部门，优先查补缺表 zt_wb_dept_manager_override；未命中 fallback 到 zt_dept.manager。
func (r *DemandDetailRepo) ListLeaderDeptTreeIDs(ctx context.Context, account string) ([]uint, error) {
	if r == nil || r.db == nil {
		return nil, fmt.Errorf("demand detail repo is not configured")
	}
	account = strings.TrimSpace(account)
	if account == "" {
		return []uint{}, nil
	}

	type managedDept struct {
		ID   uint   `gorm:"column:id"`
		Path string `gorm:"column:path"`
	}
	var managed []managedDept
	err := r.db.WithContext(ctx).Raw(`
SELECT d.id, COALESCE(d.path, '') AS path FROM zt_dept d
LEFT JOIN zt_wb_dept_manager_override o
  ON o.dept = d.id AND o.deleted = '0' AND (d.id = 52 OR d.path LIKE '%,52,%')
WHERE COALESCE(
  CASE
    WHEN (d.id = 52 OR d.path LIKE '%,52,%') AND o.id IS NOT NULL THEN o.account
    ELSE d.manager
  END, ''
) REGEXP ? ORDER BY d.id ASC`, accountTokenRegexp(account)).Scan(&managed).Error
	if err != nil {
		return nil, err
	}
	if len(managed) == 0 {
		return []uint{}, nil
	}

	deptIDs := make([]uint, 0, len(managed))
	query := `SELECT DISTINCT id FROM zt_dept WHERE id IN ?`
	args := []interface{}{make([]uint, 0, len(managed))}
	for _, dept := range managed {
		deptIDs = append(deptIDs, dept.ID)
		if path := strings.TrimSpace(dept.Path); path != "" {
			query += ` OR path LIKE ?`
			args = append(args, path+"%")
		}
	}
	args[0] = deptIDs
	query += ` ORDER BY id ASC`

	var ids []uint
	if err := r.db.WithContext(ctx).Raw(query, args...).Scan(&ids).Error; err != nil {
		return nil, err
	}
	if ids == nil {
		ids = []uint{}
	}
	return ids, nil
}

// CheckLeaderDemandVisibility 判断需求干系人（BRA/RD/assignedTo/originator/QD/accepter/reviewer/createdBy/closedBy/editedBy/feedbackedBy/PM）
// 是否属于该团队长管辖部门（zt_dept.manager，补缺表 zt_wb_dept_manager_override 优先）或其下级部门。
func (r *DemandDetailRepo) CheckLeaderDemandVisibility(ctx context.Context, demandID uint, deptIDs []uint) (bool, error) {
	if r == nil || r.db == nil {
		return false, fmt.Errorf("demand detail repo is not configured")
	}
	if demandID == 0 || len(deptIDs) == 0 {
		return false, nil
	}

	var hits int64
	err := r.db.WithContext(ctx).Raw(`
SELECT COUNT(*) FROM zt_demand d
WHERE d.id = ? AND d.deleted = '0'
  AND (
    EXISTS (
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
    )
  )`, demandID, deptIDs).Scan(&hits).Error
	if err != nil {
		return false, err
	}
	return hits > 0, nil
}

func accountTokenRegexp(account string) string {
	return `(^|[[:space:],;])` + regexp.QuoteMeta(strings.TrimSpace(account)) + `([[:space:],;]|$)`
}
