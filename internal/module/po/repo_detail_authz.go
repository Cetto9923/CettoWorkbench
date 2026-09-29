// =============================================================================
// 文件: internal/module/po/repo_detail_authz.go
// 模块: PO 工作台
// 类型: repo
// 职责: 业务需求详情的组织角色与团队长可见范围持久层查询（F01）。
//       关系判断的 SQL 与口径统一由 internal/module/demandauthz 承载，
//       本文件只做薄转发，保证详情读授权与排期写授权共用同一套判断。
// =============================================================================

package po

import (
	"context"
	"strings"

	"workbench/internal/module/demandauthz"
)

// authzRepo 返回共用授权仓储。
func (r *DemandDetailRepo) authzRepo() *demandauthz.Repo {
	if r == nil {
		return nil
	}
	return demandauthz.New(r.db)
}

// IsPMORole 查询用户是否具备激活状态的 PMO 组织角色。
func (r *DemandDetailRepo) IsPMORole(ctx context.Context, userID int64) (bool, error) {
	if r == nil || r.db == nil {
		return false, nil
	}
	return r.authzRepo().IsPMORole(ctx, userID)
}

// IsPORole 查询用户是否具备激活状态的 PO / 产品经理 组织角色。
func (r *DemandDetailRepo) IsPORole(ctx context.Context, userID int64) (bool, error) {
	if r == nil || r.db == nil {
		return false, nil
	}
	return r.authzRepo().IsPORole(ctx, userID)
}

// IsDeptManager 判断账号是否为某部门/部室负责人（zt_dept.manager 或 补缺表覆盖）。
func (r *DemandDetailRepo) IsDeptManager(ctx context.Context, account string) (bool, error) {
	if r == nil || r.db == nil {
		return false, nil
	}
	ids, err := r.authzRepo().ListLeaderDeptTreeIDs(ctx, account)
	if err != nil {
		return false, err
	}
	return len(ids) > 0, nil
}

// ListLeaderDeptTreeIDs 返回账号负责的部门及其下级部门 ID。
// 优先查补缺表 zt_wb_dept_manager_override；未命中 fallback 到 zt_dept.manager。
func (r *DemandDetailRepo) ListLeaderDeptTreeIDs(ctx context.Context, account string) ([]uint, error) {
	if r == nil || r.db == nil {
		return nil, nil
	}
	return r.authzRepo().ListLeaderDeptTreeIDs(ctx, strings.TrimSpace(account))
}

// CheckLeaderDemandVisibility 判断需求干系人是否属于该团队长管辖部门或其下级部门。
func (r *DemandDetailRepo) CheckLeaderDemandVisibility(ctx context.Context, demandID uint, deptIDs []uint) (bool, error) {
	if r == nil || r.db == nil {
		return false, nil
	}
	return r.authzRepo().CheckLeaderDemandVisibility(ctx, demandID, deptIDs)
}
