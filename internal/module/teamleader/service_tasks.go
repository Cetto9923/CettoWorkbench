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

	parent, err := s.repo.FindParentTeamByID(ctx, req.TeamID)
	if err != nil {
		return nil, errorx.Wrap(errorx.ErrCodeInternal, "查询团队信息失败", err)
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

	actorAccount := strings.TrimSpace(actor.Account)
	isSuperAdmin := actor.IsSuperAdmin
	isTeamLeader := isSuperAdmin || (parent != nil && strings.TrimSpace(parent.Manager) == actorAccount)
	isGroupLeader := strings.TrimSpace(targetGroup.Manager) == actorAccount
	isGroupPO := strings.TrimSpace(targetGroup.PO) == actorAccount
	isGroupMember := containsString(memberAccounts, actorAccount)

	// 1. 小组级访问控制：团队长、本组组长、本组PO、本组正式成员有权访问该小组
	if !isTeamLeader && !isGroupLeader && !isGroupPO && !isGroupMember {
		return nil, errorx.New(errorx.ErrCodeForbidden, "无权访问该小组任务数据")
	}

	// 2. 成员级筛选鉴权：
	targetAccount := strings.TrimSpace(req.Account)
	if targetAccount != "" && targetAccount != "all" {
		if !containsString(memberAccounts, targetAccount) {
			return nil, errorx.New(errorx.ErrCodeForbidden, "无权查看该成员或该成员不属于当前小组")
		}
		// 普通成员只能查看本人任务，不可指定组内其他成员
		if !isTeamLeader && !isGroupLeader && !isGroupPO && targetAccount != actorAccount {
			return nil, errorx.New(errorx.ErrCodeForbidden, "普通成员仅可查看本人任务，无权查询其他成员任务")
		}
	}

	// 3. 待核查任务可见性范围判定：
	// 团队长、小组长可核查本组全部组员的待核查任务；普通成员仅能核查指派给本人的待核查任务
	var pendingAccounts []string
	if isTeamLeader || isGroupLeader {
		if targetAccount != "" && targetAccount != "all" {
			pendingAccounts = []string{targetAccount}
		} else {
			pendingAccounts = memberAccounts
		}
	} else if isGroupMember {
		pendingAccounts = []string{actorAccount}
	}

	// 4. 标准分页与时间窗口定义
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	thirtyDaysAgo := today.AddDate(0, 0, -30)

	page := req.Page
	if page <= 0 {
		page = 1
	}
	pageSize := req.PageSize
	if pageSize <= 0 {
		pageSize = 50
	} else if pageSize > 200 {
		pageSize = 200
	}
	offset := (page - 1) * pageSize

	pendingPage := req.PendingPage
	if pendingPage <= 0 {
		pendingPage = 1
	}
	pendingPageSize := req.PendingPageSize
	if pendingPageSize <= 0 {
		pendingPageSize = 20
	} else if pendingPageSize > 200 {
		pendingPageSize = 200
	}
	pendingOffset := (pendingPage - 1) * pendingPageSize

	// 5. 真实数据库指标统计
	summary, err := s.repo.CountGroupConfirmedTasks(ctx, targetGroup.ID, targetAccount, thirtyDaysAgo, today)
	if err != nil {
		return nil, errorx.Wrap(errorx.ErrCodeInternal, "统计小组任务指标失败", err)
	}

	pendingTotal, err := s.repo.CountPendingTasksByAccounts(ctx, pendingAccounts, actorAccount, isSuperAdmin)
	if err != nil {
		return nil, errorx.Wrap(errorx.ErrCodeInternal, "统计待核查任务失败", err)
	}
	summary.PendingReviewTotal = pendingTotal

	// 6. 分页查询正式研发任务
	confirmedRows, err := s.repo.FindGroupConfirmedTasksPaged(ctx, targetGroup.ID, targetAccount, thirtyDaysAgo, pageSize, offset)
	if err != nil {
		return nil, errorx.Wrap(errorx.ErrCodeInternal, "查询小组研发任务失败", err)
	}

	// 7. 分页查询待核查候选任务
	pendingRows, err := s.repo.FindPendingTasksByAccountsPaged(ctx, pendingAccounts, actorAccount, isSuperAdmin, pendingPageSize, pendingOffset)
	if err != nil {
		return nil, errorx.Wrap(errorx.ErrCodeInternal, "查询待核查任务失败", err)
	}

	accountMap := s.resolveTaskAccounts(ctx, confirmedRows, pendingRows)

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

	cols := assembleKanbanColumns(confirmedItems)
	timeRangeLabel := "已完成任务统计近30天，逾期仅统计未完成任务"

	return &GroupTasksResp{
		GroupID:        targetGroup.ID,
		GroupName:      targetGroup.Name,
		TimeRangeLabel: timeRangeLabel,
		Summary:        summary,
		Columns:        cols,
		ConfirmedPagination: TaskPaginationInfo{
			Page:     page,
			PageSize: pageSize,
			Total:    summary.ConfirmedTotal,
		},
		PendingReviewTasks: pendingItems,
		PendingPagination: TaskPaginationInfo{
			Page:     pendingPage,
			PageSize: pendingPageSize,
			Total:    pendingTotal,
		},
	}, nil
}

func assembleKanbanColumns(items []GroupTaskItem) []GroupTaskColumn {
	waitItems := make([]GroupTaskItem, 0, len(items))
	doingItems := make([]GroupTaskItem, 0, len(items))
	doneItems := make([]GroupTaskItem, 0, len(items))

	for _, it := range items {
		switch it.Status {
		case "wait":
			waitItems = append(waitItems, it)
		case "doing":
			doingItems = append(doingItems, it)
		case "done":
			doneItems = append(doneItems, it)
		}
	}

	return []GroupTaskColumn{
		{Key: "wait", Name: "未开始", Count: len(waitItems), Items: waitItems},
		{Key: "doing", Name: "进行中", Count: len(doingItems), Items: doingItems},
		{Key: "done", Name: "已完成", Count: len(doneItems), Items: doneItems},
	}
}

func containsString(slice []string, val string) bool {
	val = strings.TrimSpace(val)
	for _, it := range slice {
		if strings.TrimSpace(it) == val {
			return true
		}
	}
	return false
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

