// =============================================================================
// 文件: internal/module/kanban/service_issue.go
// 模块: 工作看板
// 类型: action
// 职责: 看板问题栏列表（按选中提出人 + 未解决/已解决 tab）。
// 依赖: internal/model
//       internal/pkg/errorx
//       internal/pkg/zentao
// =============================================================================

package kanban

import (
	"context"
	"fmt"
	"strings"

	"workbench/internal/model"
	"workbench/internal/pkg/errorx"
	"workbench/internal/pkg/zentao"
)

// 禅道问题状态：未解决 = 未确认+已确认；已解决 = resolved。
var (
	issueStatusesUnresolved = []string{"unconfirmed", "confirmed"}
	issueStatusesResolved   = []string{"resolved"}
)

// ListIssues 按选中负责人（提出人 createdBy）与 tab 拉取问题列表。
// 始终排除 deleted / closed；返回双侧 tab 计数。
func (s *Service) ListIssues(ctx context.Context, actor *model.User, req ListIssuesReq) (ListIssuesResp, error) {
	empty := ListIssuesResp{Items: []IssueItem{}, Counts: IssueTabCounts{}}
	if errs := req.Validate(); len(errs) > 0 {
		return empty, errorx.New(errorx.ErrCodeInvalidParam, errs[0].Message)
	}

	actorAccount := ""
	if actor != nil {
		actorAccount = actor.Account
	}

	groups, err := s.ListMyTeamgroups(ctx, actor)
	if err != nil {
		return ListIssuesResp{}, err
	}
	targets, err := resolveKanbanAccounts(actorAccount, req.Account, req.TeamgroupID, groups)
	if err != nil {
		return ListIssuesResp{}, err
	}
	if len(targets) == 0 {
		return empty, nil
	}

	unresolvedN, err := s.repo.CountKanbanIssuesByStatus(ctx, targets, issueStatusesUnresolved)
	if err != nil {
		return ListIssuesResp{}, err
	}
	resolvedN, err := s.repo.CountKanbanIssuesByStatus(ctx, targets, issueStatusesResolved)
	if err != nil {
		return ListIssuesResp{}, err
	}

	statuses := issueStatusesUnresolved
	if req.Tab == IssueTabResolved {
		statuses = issueStatusesResolved
	}
	rows, err := s.repo.FindKanbanIssues(ctx, targets, statuses)
	if err != nil {
		return ListIssuesResp{}, err
	}

	displayMap, err := s.loadAccountDisplayMap(ctx, actor)
	if err != nil {
		return ListIssuesResp{}, err
	}

	items := make([]IssueItem, 0, len(rows))
	for _, row := range rows {
		ownerAcc := strings.TrimSpace(row.AssignedTo)
		if ownerAcc == "" {
			ownerAcc = strings.TrimSpace(row.CreatedBy)
		}
		sev := strings.TrimSpace(row.Severity)
		items = append(items, IssueItem{
			ID:            row.ID,
			Title:         row.Title,
			Severity:      sev,
			SeverityLabel: issueSeverityLabel(sev),
			Status:        row.Status,
			StatusLabel:   issueStatusLabel(row.Status),
			CreatedBy:     lookupDisplay(displayMap, row.CreatedBy),
			AssignedTo:    lookupDisplay(displayMap, row.AssignedTo),
			Owner:         lookupDisplay(displayMap, ownerAcc),
			OwnerAccount:  ownerAcc,
			URL:           zentao.URL("issue", "view", fmt.Sprintf("issueID=%d", row.ID)),
		})
	}

	return ListIssuesResp{
		Items:  items,
		Counts: IssueTabCounts{Unresolved: unresolvedN, Resolved: resolvedN},
	}, nil
}

func issueSeverityLabel(severity string) string {
	switch strings.TrimSpace(severity) {
	case "1":
		return "严重"
	case "2":
		return "较严重"
	case "3":
		return "较小"
	case "4":
		return "建议"
	default:
		return ""
	}
}

func issueStatusLabel(status string) string {
	switch strings.TrimSpace(status) {
	case "unconfirmed":
		return "未确认"
	case "confirmed":
		return "已确认"
	case "resolved":
		return "已解决"
	case "closed":
		return "已关闭"
	case "canceled":
		return "已取消"
	default:
		return status
	}
}
