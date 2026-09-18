// =============================================================================
// 文件: internal/module/kanban/repo_issue.go
// 模块: 工作看板
// 类型: action
// 职责: 看板问题栏查询（按提出人 + 状态，排除已删/已关闭）。
// 依赖: internal/model/zentao
// =============================================================================

package kanban

import (
	"context"
	"strings"

	ztmodel "workbench/internal/model/zentao"
)

const kanbanIssueLimit = 200

// issueRow 问题栏查询原始行。
type issueRow struct {
	ID         int64
	Title      string
	Severity   string
	Status     string
	CreatedBy  string
	AssignedTo string
}

// FindKanbanIssues 查询选中账号（可多个）提出的问题。
// 仅返回 status IN statuses，且 deleted=0、排除 closed。
func (r *Repo) FindKanbanIssues(ctx context.Context, accounts, statuses []string) ([]issueRow, error) {
	if r == nil || r.db == nil {
		return []issueRow{}, nil
	}
	cleaned := cleanAccounts(accounts)
	cleanedStatus := cleanStatuses(statuses)
	if len(cleaned) == 0 || len(cleanedStatus) == 0 {
		return []issueRow{}, nil
	}

	type rawRow struct {
		ID         int64  `gorm:"column:id"`
		Title      string `gorm:"column:title"`
		Severity   string `gorm:"column:severity"`
		Status     string `gorm:"column:status"`
		CreatedBy  string `gorm:"column:createdBy"`
		AssignedTo string `gorm:"column:assignedTo"`
	}

	var rows []rawRow
	err := r.db.WithContext(ctx).
		Table((ztmodel.ZtIssue{}).TableName()).
		Select("id, title, severity, status, createdBy, assignedTo").
		Where("deleted = ?", "0").
		Where("status <> ?", "closed").
		Where("createdBy IN ?", cleaned).
		Where("status IN ?", cleanedStatus).
		Order("id DESC").
		Limit(kanbanIssueLimit).
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return []issueRow{}, nil
	}

	out := make([]issueRow, 0, len(rows))
	for _, row := range rows {
		out = append(out, issueRow{
			ID:         row.ID,
			Title:      row.Title,
			Severity:   row.Severity,
			Status:     row.Status,
			CreatedBy:  row.CreatedBy,
			AssignedTo: row.AssignedTo,
		})
	}
	return out, nil
}

// CountKanbanIssuesByStatus 按状态统计选中账号提出的未删、非关闭问题数。
func (r *Repo) CountKanbanIssuesByStatus(ctx context.Context, accounts, statuses []string) (int64, error) {
	if r == nil || r.db == nil {
		return 0, nil
	}
	cleaned := cleanAccounts(accounts)
	cleanedStatus := cleanStatuses(statuses)
	if len(cleaned) == 0 || len(cleanedStatus) == 0 {
		return 0, nil
	}

	var total int64
	err := r.db.WithContext(ctx).
		Table((ztmodel.ZtIssue{}).TableName()).
		Where("deleted = ?", "0").
		Where("status <> ?", "closed").
		Where("createdBy IN ?", cleaned).
		Where("status IN ?", cleanedStatus).
		Count(&total).Error
	if err != nil {
		return 0, err
	}
	return total, nil
}

func cleanAccounts(accounts []string) []string {
	out := make([]string, 0, len(accounts))
	seen := make(map[string]struct{}, len(accounts))
	for _, a := range accounts {
		a = strings.TrimSpace(a)
		if a == "" {
			continue
		}
		if _, ok := seen[a]; ok {
			continue
		}
		seen[a] = struct{}{}
		out = append(out, a)
	}
	return out
}

func cleanStatuses(statuses []string) []string {
	out := make([]string, 0, len(statuses))
	seen := make(map[string]struct{}, len(statuses))
	for _, s := range statuses {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		if _, ok := seen[s]; ok {
			continue
		}
		seen[s] = struct{}{}
		out = append(out, s)
	}
	return out
}
