// =============================================================================
// 文件: internal/module/demandauthz/authz.go
// 模块: 需求对象级授权
// 类型: service
// 职责: 业务需求（zt_demand）对象级读写授权的唯一决策口径。
//       PO 工作台详情（读）与排期工作台保存（写）共用本文件的关系判断，
//       避免两处各写一套导致口径漂移。
// 依赖: internal/model
//       internal/module/demandauthz/repo.go
// =============================================================================

package demandauthz

import (
	"context"
	"strings"

	"workbench/internal/model"
)

// Access 是 actor 对某个业务需求的对象级访问级别。
type Access int

const (
	// AccessNone 无任何对象级关系。
	AccessNone Access = iota
	// AccessRelated 命中干系人 / 团队长管辖 / PMO 关系，可读可写。
	AccessRelated
	// AccessSuperAdmin 超级管理员，可读可写。
	AccessSuperAdmin
)

// CanWrite 返回该级别是否允许执行写库或同步禅道的操作。
func (a Access) CanWrite() bool { return a == AccessRelated || a == AccessSuperAdmin }

// WriteDenialMessage 是无权写该需求时统一对外的错误提示。
const WriteDenialMessage = "无权修改该业务需求"

// Evaluate 判定 actor 对 demandID 的对象级访问级别（只含关系判断，不含任何
// 功能权限位放行）。这是读写的唯一关系口径，判断顺序固定为：
//
//	超级管理员（actor.IsSuperAdmin）→ AccessSuperAdmin
//	组织角色 PMO（zt_roles.code = 'pmo'）→ AccessRelated
//	个人干系人关系 → AccessRelated
//	组织角色团队长：管辖部门（或其下级部门）内干系人需求 → AccessRelated
//
// 命中任一即返回，不再继续下探。actor 为 nil / 账号为空 / demandID 为 0 时
// 返回 AccessNone（调用方据此决定 403 或 400）。
//
// 读授权 = 本函数命中 OR 持有 perm.ScheduleList（只读放行，由调用方叠加）；
// 写授权 = 本函数命中（ScheduleList 明确不含写权限）。
func (r *Repo) Evaluate(ctx context.Context, actor *model.User, demandID uint) (Access, error) {
	if r == nil || r.db == nil || demandID == 0 || actor == nil {
		return AccessNone, nil
	}
	if actor.IsSuperAdmin {
		return AccessSuperAdmin, nil
	}
	account := strings.TrimSpace(actor.Account)
	if account == "" {
		return AccessNone, nil
	}

	// 1. 组织角色 PMO：全量放行
	if actor.ID > 0 {
		isPMO, err := r.IsPMORole(ctx, actor.ID)
		if err != nil {
			return AccessNone, err
		}
		if isPMO {
			return AccessRelated, nil
		}
	}

	// 2. 个人干系人关系
	related, err := r.CheckDemandVisibility(ctx, demandID, account)
	if err != nil {
		return AccessNone, err
	}
	if related {
		return AccessRelated, nil
	}

	// 3. 组织角色团队长：管辖部门（或其下级部门）内干系人需求
	deptIDs, err := r.ListLeaderDeptTreeIDs(ctx, account)
	if err != nil {
		return AccessNone, err
	}
	if len(deptIDs) > 0 {
		leaderVisible, err := r.CheckLeaderDemandVisibility(ctx, demandID, deptIDs)
		if err != nil {
			return AccessNone, err
		}
		if leaderVisible {
			return AccessRelated, nil
		}
	}

	return AccessNone, nil
}

// CanWriteDemand 便捷判定：actor 是否可对 demandID 执行写操作。
func (r *Repo) CanWriteDemand(ctx context.Context, actor *model.User, demandID uint) (bool, error) {
	access, err := r.Evaluate(ctx, actor, demandID)
	if err != nil {
		return false, err
	}
	return access.CanWrite(), nil
}
