// =============================================================================
// 文件: internal/module/teamleader/service_tasks.go
// 模块: 团队长工作台
// 类型: service
// 职责: 小组研发工作看板查询、需求链归属判定、待核查候选任务隔离与对象级鉴权。
// 依赖: internal/model
//       internal/pkg/errorx
//       internal/pkg/zentao
// =============================================================================

package teamleader

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"workbench/internal/model"
	"workbench/internal/pkg/errorx"
	"workbench/internal/pkg/zentao"
)

// ListGroupTasks 获取指定小组的研发工作任务，按需求链判定真实归属并隔离待核查任务。
func (s *Service) ListGroupTasks(ctx context.Context, actor *model.User, req ListGroupTasksReq) (*GroupTasksResp, error) {
	if actor == nil || strings.TrimSpace(actor.Account) == "" {
		return nil, errorx.New(errorx.ErrCodeForbidden, "请先登录")
	}
	if errs := req.Validate(); len(errs) > 0 {
		return nil, errorx.New(errorx.ErrCodeInvalidParam, errs[0].Message)
	}

	authorizedTeams, err := s.resolveAuthorizedTeams(ctx, actor)
	if err != nil {
		return nil, err
	}
	if !isTeamAuthorized(req.TeamID, authorizedTeams) {
		return nil, errorx.New(errorx.ErrCodeForbidden, "无权访问该团队数据")
	}

	subGroups, err := s.repo.FindSubGroupsByParentID(ctx, req.TeamID)
	if err != nil {
		return nil, errorx.Wrap(errorx.ErrCodeInternal, "查询子小组失败", err)
	}
	targetGroup := findSubGroup(req.GroupID, subGroups)
	if targetGroup == nil {
		return nil, errorx.New(errorx.ErrCodeInvalidParam, "指定小组不属于当前团队")
	}

	memberRows, err := s.repo.FindMembersByGroupIDs(ctx, []uint{req.GroupID})
	if err != nil {
		return nil, errorx.Wrap(errorx.ErrCodeInternal, "查询小组成员失败", err)
	}
	memberAccounts := extractMemberAccounts(memberRows)

	account := strings.TrimSpace(req.Account)
	if account != "" && account != "all" {
		return s.fetchMemberLevelTasks(ctx, targetGroup, account, memberAccounts)
	}
	return s.fetchGroupLevelTasks(ctx, targetGroup, memberAccounts)
}

func (s *Service) fetchGroupLevelTasks(ctx context.Context, group *TeamgroupRaw, memberAccounts []string) (*GroupTasksResp, error) {
	confirmedRows, err := s.repo.FindGroupTasksByDemand(ctx, group.ID, 200)
	if err != nil {
		return nil, errorx.Wrap(errorx.ErrCodeInternal, "查询正式小组任务失败", err)
	}

	pendingRows, err := s.repo.FindPendingTasksByAccounts(ctx, memberAccounts, 200)
	if err != nil {
		return nil, errorx.Wrap(errorx.ErrCodeInternal, "查询待核查任务失败", err)
	}

	accountMap := s.resolveTaskAccounts(ctx, confirmedRows, pendingRows)
	today := time.Now().Truncate(24 * time.Hour)

	confirmedItems := make([]GroupTaskItem, 0, len(confirmedRows))
	for _, row := range confirmedRows {
		it := buildGroupTaskItem(row, accountMap, today, "confirmed", "")
		confirmedItems = append(confirmedItems, it)
	}

	pendingItems := make([]GroupTaskItem, 0, len(pendingRows))
	for _, row := range pendingRows {
		it := buildGroupTaskItem(row, accountMap, today, "pending", "")
		pendingItems = append(pendingItems, it)
	}

	columns, summary := assembleKanbanColumns(confirmedItems, len(pendingItems))
	return &GroupTasksResp{
		GroupID:            group.ID,
		GroupName:          group.Name,
		Summary:            summary,
		Columns:            columns,
		PendingReviewTasks: pendingItems,
	}, nil
}

func (s *Service) fetchMemberLevelTasks(ctx context.Context, group *TeamgroupRaw, targetAccount string, memberAccounts []string) (*GroupTasksResp, error) {
	rows, err := s.repo.FindAllTasksByAccount(ctx, targetAccount, 200)
	if err != nil {
		return nil, errorx.Wrap(errorx.ErrCodeInternal, "查询成员任务失败", err)
	}

	accountMap := s.resolveTaskAccounts(ctx, rows, nil)
	groupNameMap := s.resolveOtherGroupNames(ctx, rows, group.ID)
	today := time.Now().Truncate(24 * time.Hour)
	currentGroupIDStr := strconv.FormatUint(uint64(group.ID), 10)

	var confirmedItems []GroupTaskItem
	var pendingItems []GroupTaskItem

	for _, row := range rows {
		tg := strings.TrimSpace(row.DemandTeamGroup)
		if tg == currentGroupIDStr {
			confirmedItems = append(confirmedItems, buildGroupTaskItem(row, accountMap, today, "confirmed", ""))
		} else if tg != "" {
			otherName := groupNameMap[parseUintOrZero(tg)]
			confirmedItems = append(confirmedItems, buildGroupTaskItem(row, accountMap, today, "confirmed", otherName))
		} else {
			pendingItems = append(pendingItems, buildGroupTaskItem(row, accountMap, today, "pending", ""))
		}
	}

	columns, summary := assembleKanbanColumns(confirmedItems, len(pendingItems))
	return &GroupTasksResp{
		GroupID:            group.ID,
		GroupName:          group.Name,
		Summary:            summary,
		Columns:            columns,
		PendingReviewTasks: pendingItems,
	}, nil
}

func buildGroupTaskItem(row TaskDataRow, accountMap map[string]string, today time.Time, attr, srcGroup string) GroupTaskItem {
	ownerAcc := strings.TrimSpace(row.AssignedTo)
	if row.Status == "done" && strings.TrimSpace(row.FinishedBy) != "" {
		ownerAcc = strings.TrimSpace(row.FinishedBy)
	}
	ownerName := accountMap[ownerAcc]
	if ownerName == "" {
		ownerName = ownerAcc
	}

	deadlineStr := ""
	overdue := false
	if row.Deadline != nil && !row.Deadline.IsZero() {
		deadlineStr = row.Deadline.Format("2006-01-02")
		if row.Deadline.Before(today) && row.Status != "done" {
			overdue = true
		}
	}

	return GroupTaskItem{
		ID:              row.ID,
		DisplayID:       fmt.Sprintf("#%d", row.ID),
		Title:           row.Name,
		Status:          row.Status,
		Type:            row.Type,
		StoryID:         row.StoryID,
		StoryTitle:      row.StoryTitle,
		Owner:           ownerName,
		OwnerAccount:    ownerAcc,
		Deadline:        deadlineStr,
		Overdue:         overdue,
		URL:             zentao.URL("task", "view", fmt.Sprintf("taskID=%d", row.ID)),
		Attribution:     attr,
		SourceGroupName: srcGroup,
		IsPendingReview: attr == "pending",
	}
}

func assembleKanbanColumns(items []GroupTaskItem, pendingTotal int) ([]GroupTaskColumn, GroupTaskSummary) {
	waitItems := make([]GroupTaskItem, 0, len(items))
	doingItems := make([]GroupTaskItem, 0, len(items))
	doneItems := make([]GroupTaskItem, 0, len(items))

	summary := GroupTaskSummary{
		ConfirmedTotal:     len(items),
		PendingReviewTotal: pendingTotal,
	}

	for _, it := range items {
		if it.Overdue {
			summary.ConfirmedOverdue++
		}
		switch it.Status {
		case "wait":
			summary.ConfirmedWait++
			waitItems = append(waitItems, it)
		case "doing":
			summary.ConfirmedDoing++
			doingItems = append(doingItems, it)
		case "done":
			summary.ConfirmedDone++
			doneItems = append(doneItems, it)
		}
	}

	cols := []GroupTaskColumn{
		{Key: "wait", Name: "未开始", Count: len(waitItems), Items: waitItems},
		{Key: "doing", Name: "进行中", Count: len(doingItems), Items: doingItems},
		{Key: "done", Name: "已完成", Count: len(doneItems), Items: doneItems},
	}
	return cols, summary
}

func isTeamAuthorized(teamID uint, authorized []TeamOptionDTO) bool {
	for _, t := range authorized {
		if t.ID == teamID {
			return true
		}
	}
	return false
}

func findSubGroup(groupID uint, groups []TeamgroupRaw) *TeamgroupRaw {
	for i := range groups {
		if groups[i].ID == groupID {
			return &groups[i]
		}
	}
	return nil
}

func extractMemberAccounts(rows []TeamMemberRaw) []string {
	out := make([]string, 0, len(rows))
	seen := make(map[string]bool)
	for _, r := range rows {
		a := strings.TrimSpace(r.Account)
		if a != "" && !seen[a] {
			seen[a] = true
			out = append(out, a)
		}
	}
	return out
}

func (s *Service) resolveTaskAccounts(ctx context.Context, list1, list2 []TaskDataRow) map[string]string {
	var accounts []string
	for _, r := range list1 {
		accounts = append(accounts, r.AssignedTo, r.FinishedBy)
	}
	for _, r := range list2 {
		accounts = append(accounts, r.AssignedTo, r.FinishedBy)
	}
	res, err := s.repo.ResolveRealnames(ctx, accounts)
	if err != nil {
		return make(map[string]string)
	}
	return res
}

func (s *Service) resolveOtherGroupNames(ctx context.Context, rows []TaskDataRow, currentGroupID uint) map[uint]string {
	var otherGIDs []uint
	for _, r := range rows {
		tg := strings.TrimSpace(r.DemandTeamGroup)
		if tg != "" {
			gid := parseUintOrZero(tg)
			if gid > 0 && gid != currentGroupID {
				otherGIDs = append(otherGIDs, gid)
			}
		}
	}
	if len(otherGIDs) == 0 {
		return make(map[uint]string)
	}
	res, err := s.repo.FindSubGroupNamesMap(ctx, otherGIDs)
	if err != nil {
		return make(map[uint]string)
	}
	return res
}

func parseUintOrZero(s string) uint {
	v, err := strconv.ParseUint(strings.TrimSpace(s), 10, 32)
	if err != nil {
		return 0
	}
	return uint(v)
}
