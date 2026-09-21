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
