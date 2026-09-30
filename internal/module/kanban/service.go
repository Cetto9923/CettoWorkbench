// =============================================================================
// 文件: internal/module/kanban/service.go
// 模块: 工作看板
// 类型: readonly
// 职责: 看板业务编排（所属小组 + 成员排序展示；按页面填充任务/需求角标）。
// 依赖: internal/model
//       internal/module/po
//       internal/module/user
//       internal/pkg/zentao
// =============================================================================

package kanban

import (
	"context"
	"strings"
	"unicode/utf8"

	"workbench/internal/model"
	"workbench/internal/module/po"
	"workbench/internal/module/user"
	"workbench/internal/pkg/zentao"
)

// Service 看板业务服务。
type Service struct {
	repo    *Repo
	userSvc *user.Service
	poSvc   *po.Service
	ztAPI   *zentao.Client
}

// NewService 创建 Service。
func NewService(repo *Repo, userSvc *user.Service, poSvc *po.Service, ztAPI *zentao.Client) *Service {
	return &Service{repo: repo, userSvc: userSvc, poSvc: poSvc, ztAPI: ztAPI}
}

// 成员角标统计范围：按页面只算需要的一侧，避免任务页被需求 N+1 拖慢。
const (
	WorkloadNone   = ""       // JSON 接口：只要成员名单，不算量
	WorkloadTask   = "task"   // 任务看板：只填 TaskCount
	WorkloadDemand = "demand" // 需求看板：只填 DemandCount
)

// ListMyTeamgroups 返回当前用户所属敏捷小组（含排序后的成员）。
// workload 决定是否填充负责人角标：任务页只算任务，需求页只算需求，其余不算。
func (s *Service) ListMyTeamgroups(ctx context.Context, actor *model.User, workload string) ([]TeamgroupItem, error) {
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
	if err := s.fillMemberWorkload(ctx, out, workload); err != nil {
		return nil, err
	}
	return out, nil
}

// fillMemberWorkload 按 workload 填充负责人角标（任务页 / 需求页互不串算）。
func (s *Service) fillMemberWorkload(ctx context.Context, groups []TeamgroupItem, workload string) error {
	if workload != WorkloadTask && workload != WorkloadDemand {
		return nil
	}
	accounts := make([]string, 0)
	for acc := range collectMemberAccounts(groups) {
		accounts = append(accounts, acc)
	}
	if len(accounts) == 0 {
		return nil
	}

	if workload == WorkloadTask {
		tasks, err := s.repo.CountMemberTasks(ctx, accounts)
		if err != nil {
			return err
		}
		for i := range groups {
			for j := range groups[i].Members {
				groups[i].Members[j].TaskCount = tasks[groups[i].Members[j].Account]
			}
		}
		return nil
	}

	// WorkloadDemand：按人统计价值流「全部」（与首页一致）；暂无批量接口。
	if s.poSvc == nil {
		return nil
	}
	for i := range groups {
		for j := range groups[i].Members {
			n, demErr := s.poSvc.CountValueStreamAll(ctx, groups[i].Members[j].Account)
			if demErr != nil {
				return demErr
			}
			groups[i].Members[j].DemandCount = n
		}
	}
	return nil
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
