// =============================================================================
// 文件: internal/module/kanban/service.go
// 模块: 工作看板
// 类型: readonly
// 职责: 需求看板业务编排（所属小组 + 成员排序展示）。
// 依赖: internal/model
//       internal/module/po
//       internal/module/user
// =============================================================================

package kanban

import (
	"context"
	"strings"
	"unicode/utf8"

	"workbench/internal/model"
	"workbench/internal/module/po"
	"workbench/internal/module/user"
)

// Service 看板业务服务。
type Service struct {
	repo    *Repo
	userSvc *user.Service
	poSvc   *po.Service
}

// NewService 创建 Service。
func NewService(repo *Repo, userSvc *user.Service, poSvc *po.Service) *Service {
	return &Service{repo: repo, userSvc: userSvc, poSvc: poSvc}
}

// ListMyTeamgroups 返回当前用户所属敏捷小组（含排序后的成员）。
func (s *Service) ListMyTeamgroups(ctx context.Context, actor *model.User) ([]TeamgroupItem, error) {
	if actor == nil {
		return []TeamgroupItem{}, nil
	}

	rows, err := s.repo.ListUserTeamgroups(ctx, actor.Account)
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return []TeamgroupItem{}, nil
	}

	roots := make([]uint, 0, len(rows))
	for _, row := range rows {
		roots = append(roots, row.ID)
	}
	memberRows, err := s.repo.ListTeamMembersByRoots(ctx, roots)
	if err != nil {
		return nil, err
	}
	membersByRoot := map[uint][]string{}
	for _, m := range memberRows {
		acc := strings.TrimSpace(m.Account)
		if acc == "" {
			continue
		}
		membersByRoot[m.Root] = append(membersByRoot[m.Root], acc)
	}

	displayMap, err := s.loadAccountDisplayMap(ctx, actor)
	if err != nil {
		return nil, err
	}

	out := make([]TeamgroupItem, 0, len(rows))
	for _, row := range rows {
		out = append(out, TeamgroupItem{
			ID:      row.ID,
			Name:    row.Name,
			Members: buildOrderedMembers(row.PO, row.Manager, membersByRoot[row.ID], displayMap),
		})
	}
	return out, nil
}

func (s *Service) loadAccountDisplayMap(ctx context.Context, actor *model.User) (map[string]string, error) {
	if s.userSvc == nil {
		return map[string]string{}, nil
	}
	return s.userSvc.AccountDisplayMap(ctx, actor)
}

// buildOrderedMembers 按 PO → 敏捷教练 → 普通成员排序；同账号优先保留更高角色。
func buildOrderedMembers(poCSV, managerCSV string, teamAccounts []string, displayMap map[string]string) []MemberItem {
	poList := splitAccounts(poCSV)
	coachList := splitAccounts(managerCSV)

	seen := map[string]struct{}{}
	out := make([]MemberItem, 0, len(poList)+len(coachList)+len(teamAccounts))

	appendRole := func(accounts []string, role string) {
		for _, acc := range accounts {
			if _, ok := seen[acc]; ok {
				continue
			}
			seen[acc] = struct{}{}
			display := lookupDisplay(displayMap, acc)
			out = append(out, MemberItem{
				Account: acc,
				Display: display,
				Initial: firstRune(display),
				Role:    role,
			})
		}
	}

	appendRole(poList, MemberRolePO)
	appendRole(coachList, MemberRoleCoach)
	appendRole(teamAccounts, MemberRoleMember)

	if out == nil {
		return []MemberItem{}
	}
	return out
}

func splitAccounts(csv string) []string {
	parts := strings.Split(csv, ",")
	out := make([]string, 0, len(parts))
	seen := map[string]struct{}{}
	for _, p := range parts {
		acc := strings.TrimSpace(p)
		if acc == "" {
			continue
		}
		if _, ok := seen[acc]; ok {
			continue
		}
		seen[acc] = struct{}{}
		out = append(out, acc)
	}
	return out
}

func lookupDisplay(displayMap map[string]string, account string) string {
	if displayMap != nil {
		if d := strings.TrimSpace(displayMap[account]); d != "" {
			return d
		}
	}
	return account
}

func firstRune(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return "?"
	}
	r, _ := utf8.DecodeRuneInString(s)
	if r == utf8.RuneError {
		return "?"
	}
	return string(r)
}
