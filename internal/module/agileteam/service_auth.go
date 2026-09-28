// =============================================================================
// 文件: internal/module/agileteam/service_auth.go
// 模块: 敏捷小组治理
// 类型: security
// 职责: 敏捷小组写操作的对象级授权。粗粒度权限只负责进入 Service；
//       Service 仍必须校验具体 teamgroup 是否属于当前 PO/敏捷教练可写范围。
// =============================================================================

package agileteam

import (
	"context"
	"errors"
	"strings"
	"unicode"

	"gorm.io/gorm"

	"workbench/internal/model"
	"workbench/internal/pkg/errorx"
)

func requireActorAccount(actor *model.User) (string, error) {
	account := actorAccount(actor)
	if actor == nil || account == "" {
		return "", errorx.New("unauthorized", "请先登录")
	}
	return account, nil
}

func splitTeamgroupAccounts(raw string) []string {
	parts := strings.FieldsFunc(raw, func(r rune) bool {
		switch r {
		case ',', '，', ';', '；', '\n', '\r', '\t':
			return true
		default:
			return unicode.IsSpace(r)
		}
	})
	out := make([]string, 0, len(parts))
	seen := map[string]bool{}
	for _, part := range parts {
		account := strings.TrimSpace(part)
		if account == "" || seen[account] {
			continue
		}
		seen[account] = true
		out = append(out, account)
	}
	return out
}

// canEditTeamgroupObject 判断当前账号是否可修改指定敏捷小组。
// allowGlobal 只能由已通过 middleware 权限解析的 PMO AgileTeamConfirm 能力传入；
// 普通工作台角色即使拥有 AgileTeamUpdate 粗权限，也不能跨小组写入。
func canEditTeamgroupObject(actor *model.User, row *TeamgroupRow, allowGlobal bool) bool {
	if actor == nil || row == nil {
		return false
	}
	account := actorAccount(actor)
	if account == "" {
		return false
	}
	if actor.IsSuperAdmin || allowGlobal {
		return true
	}
	if strings.TrimSpace(row.PO) == account {
		return true
	}
	for _, manager := range splitTeamgroupAccounts(row.Manager) {
		if manager == account {
			return true
		}
	}
	return false
}

func requireTeamgroupObjectEdit(actor *model.User, row *TeamgroupRow, allowGlobal bool) error {
	if _, err := requireActorAccount(actor); err != nil {
		return err
	}
	if canEditTeamgroupObject(actor, row, allowGlobal) {
		return nil
	}
	return errorx.New("forbidden", "仅该敏捷小组的 PO、敏捷教练或 PMO 可执行此操作")
}

// CanViewTeamgroupLeadScope 校验团队管理视图是否有权查看指定敏捷团队。
// 敏捷教练按 manager 关系授权；部门负责人按本人负责部门及其挂靠团队授权。
func (s *Service) CanViewTeamgroupLeadScope(ctx context.Context, actor *model.User, id uint) (bool, error) {
	account, err := requireActorAccount(actor)
	if err != nil {
		return false, err
	}
	if actor.IsSuperAdmin {
		return true, nil
	}
	managedIDs, err := s.repo.ListManagedTeamgroupIDs(ctx, account)
	if err != nil {
		return false, err
	}
	if containsUint(managedIDs, id) {
		return true, nil
	}
	isDeptManager, err := s.repo.IsDeptManager(ctx, account)
	if err != nil {
		return false, err
	}
	if !isDeptManager {
		return false, nil
	}
	deptIDs, err := s.repo.ListDeptTreeIDs(ctx, account)
	if err != nil {
		return false, err
	}
	teamgroupIDs, err := s.repo.ListMappedTeamgroupIDsByDepts(ctx, deptIDs)
	if err != nil {
		return false, err
	}
	return containsUint(teamgroupIDs, id), nil
}

// CanViewTeamgroupMemberDetails applies the still-unconfirmed department-manager
// personal-detail policy conservatively. An explicit coach assignment grants
// detail only for that coach's team scope; department management alone does not.
func (s *Service) CanViewTeamgroupMemberDetails(ctx context.Context, actor *model.User, id uint) (bool, error) {
	account, err := requireActorAccount(actor)
	if err != nil {
		return false, err
	}
	if actor.IsSuperAdmin {
		return true, nil
	}
	isDeptManager, err := s.repo.IsDeptManager(ctx, account)
	if err != nil {
		return false, err
	}
	if isDeptManager {
		return false, nil
	}
	managedIDs, err := s.repo.ListManagedTeamgroupIDs(ctx, account)
	if err != nil {
		return false, err
	}
	return containsUint(managedIDs, id), nil
}

// CanEnterLeadView 判断账号是否登记为敏捷教练或禅道部门负责人。
func (s *Service) CanEnterLeadView(ctx context.Context, actor *model.User) (bool, error) {
	account, err := requireActorAccount(actor)
	if err != nil {
		return false, err
	}
	if actor.IsSuperAdmin {
		return true, nil
	}
	managedIDs, err := s.repo.ListManagedTeamgroupIDs(ctx, account)
	if err != nil {
		return false, err
	}
	if len(managedIDs) > 0 {
		return true, nil
	}
	return s.repo.IsDeptManager(ctx, account)
}

// CanEditTeamgroupObject 为详情页返回对象级可编辑能力。
// coarseAllowed 由 Handler 的 AgileTeamUpdate 粗权限决定；allowGlobal 仅来自
// AgileTeamConfirm（PMO）能力。这样前端 CanEdit 与真正写接口使用同一授权口径。
func (s *Service) CanEditTeamgroupObject(ctx context.Context, actor *model.User, id uint, coarseAllowed, allowGlobal bool) (bool, error) {
	if !coarseAllowed {
		return false, nil
	}
	row, err := s.repo.FindTeamgroupByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return false, errorx.New("not_found", "敏捷小组不存在")
		}
		return false, err
	}
	return canEditTeamgroupObject(actor, row, allowGlobal), nil
}
