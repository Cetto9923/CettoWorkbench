// =============================================================================
// 文件: internal/module/kanban/service_createissue.go
// 模块: 工作看板
// 类型: action
// 职责: 提问题弹窗下拉（所属项目 / 指派给 / 提出人），对齐禅道 simpleCreate。
// 依赖: internal/model
//       internal/pkg/errorx
// =============================================================================

package kanban

import (
	"context"
	"strconv"
	"strings"

	"workbench/internal/model"
	"workbench/internal/pkg/errorx"
)

// CreateIssue 返回提问题弹窗选项与默认值。
// 项目：看板选中人（或小组全员）的任务参与项目 ∪ 问题提出项目，排除 closed。
// 指派给：小组成员 ∪ 项目团队；默认 Scrum Master。
// 提出人：当前登录人 ∪ 小组成员 ∪ 项目团队；默认看板选中人（否则当前用户）。
func (s *Service) CreateIssue(ctx context.Context, actor *model.User, req CreateIssueReq) (CreateIssueResp, error) {
	empty := CreateIssueResp{
		Project:    CreateIssueFieldOptions{Options: []CreateIssueOption{}},
		AssignedTo: CreateIssueFieldOptions{Options: []CreateIssueOption{}},
		Owner:      CreateIssueFieldOptions{Options: []CreateIssueOption{}},
	}
	if actor == nil {
		return empty, errorx.New(errorx.ErrCodeForbidden, "未登录")
	}

	memberRows, err := s.repo.ListTeamMembersByRoots(ctx, []uint{req.TeamgroupID})
	if err != nil {
		return empty, err
	}
	teamAccounts := memberRowsToAccounts(memberRows)

	scopeAccounts := teamAccounts
	sel := strings.TrimSpace(req.Account)
	if sel != "" && sel != AccountFilterAll {
		scopeAccounts = []string{sel}
	}
	if len(scopeAccounts) == 0 {
		return empty, nil
	}

	taskProjectIDs, err := s.repo.ListParticipateTaskProjectIDs(ctx, scopeAccounts)
	if err != nil {
		return empty, err
	}
	issueProjectIDs, err := s.repo.ListIssueOwnerProjectIDs(ctx, scopeAccounts)
	if err != nil {
		return empty, err
	}
	projectIDs := mergeUintIDs(taskProjectIDs, issueProjectIDs)

	projectRows, err := s.repo.FindOpenProjectsByIDs(ctx, projectIDs)
	if err != nil {
		return empty, err
	}
	projectOpts := make([]CreateIssueOption, 0, len(projectRows))
	for _, row := range projectRows {
		name := strings.TrimSpace(row.Name)
		if name == "" {
			name = strconv.FormatUint(uint64(row.ID), 10)
		}
		projectOpts = append(projectOpts, CreateIssueOption{
			Value: strconv.FormatUint(uint64(row.ID), 10),
			Label: name,
		})
	}

	// 默认取 id 倒序后的第一项。
	var defaultProjectID uint
	if len(projectRows) > 0 {
		defaultProjectID = projectRows[0].ID
	}

	projectTeam := []string{}
	if defaultProjectID > 0 {
		projectTeam, err = s.repo.ListProjectTeamAccounts(ctx, defaultProjectID)
		if err != nil {
			return empty, err
		}
	}

	managerCSV, err := s.repo.FindTeamgroupManager(ctx, req.TeamgroupID)
	if err != nil {
		return empty, err
	}
	smAccounts := splitAccounts(managerCSV)
	defaultAssigned := ""
	if len(smAccounts) > 0 {
		defaultAssigned = smAccounts[0]
	}

	defaultOwner := sel
	if defaultOwner == "" || defaultOwner == AccountFilterAll {
		defaultOwner = strings.TrimSpace(actor.Account)
	}

	displayMap, err := s.loadAccountDisplayMap(ctx, actor)
	if err != nil {
		return empty, err
	}

	assignedAccounts := append(append([]string{}, teamAccounts...), projectTeam...)
	assignedOpts := optionsFromAccounts(assignedAccounts, displayMap)
	assignedOpts = ensureOption(assignedOpts, defaultAssigned, displayMap)

	actorAcc := strings.TrimSpace(actor.Account)
	ownerAccounts := make([]string, 0, len(assignedAccounts)+1)
	if actorAcc != "" {
		ownerAccounts = append(ownerAccounts, actorAcc)
	}
	for _, acc := range assignedAccounts {
		if acc == actorAcc {
			continue
		}
		ownerAccounts = append(ownerAccounts, acc)
	}
	ownerOpts := optionsFromAccounts(ownerAccounts, displayMap)
	ownerOpts = ensureOption(ownerOpts, defaultOwner, displayMap)

	defaultProjectValue := ""
	if defaultProjectID > 0 {
		defaultProjectValue = strconv.FormatUint(uint64(defaultProjectID), 10)
	}

	return CreateIssueResp{
		Project:    CreateIssueFieldOptions{Options: projectOpts, Value: defaultProjectValue},
		AssignedTo: CreateIssueFieldOptions{Options: assignedOpts, Value: defaultAssigned},
		Owner:      CreateIssueFieldOptions{Options: ownerOpts, Value: defaultOwner},
	}, nil
}

func memberRowsToAccounts(rows []teamMemberRow) []string {
	out := make([]string, 0, len(rows))
	for _, m := range rows {
		out = append(out, m.Account)
	}
	return out
}

func mergeUintIDs(parts ...[]uint) []uint {
	seen := map[uint]struct{}{}
	out := make([]uint, 0)
	for _, part := range parts {
		for _, id := range part {
			if id == 0 {
				continue
			}
			if _, ok := seen[id]; ok {
				continue
			}
			seen[id] = struct{}{}
			out = append(out, id)
		}
	}
	return out
}

func optionsFromAccounts(accounts []string, displayMap map[string]string) []CreateIssueOption {
	out := make([]CreateIssueOption, 0, len(accounts))
	seen := map[string]struct{}{}
	for _, acc := range accounts {
		acc = strings.TrimSpace(acc)
		if acc == "" {
			continue
		}
		if _, ok := seen[acc]; ok {
			continue
		}
		seen[acc] = struct{}{}
		out = append(out, CreateIssueOption{
			Value: acc,
			Label: lookupDisplay(displayMap, acc),
		})
	}
	if out == nil {
		return []CreateIssueOption{}
	}
	return out
}

func ensureOption(opts []CreateIssueOption, account string, displayMap map[string]string) []CreateIssueOption {
	account = strings.TrimSpace(account)
	if account == "" {
		return opts
	}
	for _, o := range opts {
		if o.Value == account {
			return opts
		}
	}
	return append([]CreateIssueOption{{
		Value: account,
		Label: lookupDisplay(displayMap, account),
	}}, opts...)
}
