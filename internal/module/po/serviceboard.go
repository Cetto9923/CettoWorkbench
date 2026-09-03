// =============================================================================
// 文件: internal/module/po/serviceboard.go
// 模块: PO 工作台
// 类型: action
// 职责: PO 工作看板服务（V1.3 需求+任务双视图）。
//       复用 todos 的 actor scope 基础集 + 7 维筛选；状态派生不可拖拽。
// 依赖: 无
// =============================================================================

package po

import (
	"context"
	"strings"

	"workbench/internal/model"
)

// BoardDemand 我的需求看板（PO 视角工作对象树）。
func (s *Service) BoardDemand(ctx context.Context, actor *model.User, req BoardDemandReq) (*BoardDemandResp, error) {
	if actor == nil || strings.TrimSpace(actor.Account) == "" {
		return &BoardDemandResp{Tree: nil, Teamgroups: nil}, nil
	}
	if req.POAccount == "" {
		req.POAccount = actor.Account
	}
	displayMap, err := s.loadAccountDisplayMap(ctx, actor)
	if err != nil {
		return nil, err
	}
	tree, summary, err := s.repo.FindBoardDemandTree(ctx, req, displayMap)
	if err != nil {
		return nil, err
	}
	teams, err := s.repo.FindBoardTeamgroups(ctx, actor.Account)
	if err != nil {
		return nil, err
	}
	return &BoardDemandResp{Tree: tree, Summary: summary, Teamgroups: teams}, nil
}

// BoardIssues 右侧问题栏：当前账号可见真实问题。
func (s *Service) BoardIssues(ctx context.Context, actor *model.User) (*BoardIssueResp, error) {
	if actor == nil || strings.TrimSpace(actor.Account) == "" {
		return &BoardIssueResp{Items: []BoardIssueItem{}}, nil
	}
	return s.repo.FindBoardIssues(ctx, actor.Account)
}

// BoardGroupMetrics 小组效能快照：选定具体敏捷小组时返回 8 项真实指标。
func (s *Service) BoardGroupMetrics(ctx context.Context, actor *model.User, req GroupMetricsReq) (*GroupMetricsResp, error) {
	if actor == nil || strings.TrimSpace(actor.Account) == "" {
		return &GroupMetricsResp{HasGroup: false, Metrics: []*BoardMetric{}}, nil
	}
	if req.TeamgroupID == 0 {
		return &GroupMetricsResp{HasGroup: false, GroupID: 0, Metrics: []*BoardMetric{}}, nil
	}
	teams, err := s.repo.FindBoardTeamgroups(ctx, actor.Account)
	if err != nil {
		return nil, err
	}
	name := ""
	allowed := false
	for _, t := range teams {
		if t.ID == req.TeamgroupID {
			allowed = true
			name = t.Name
			break
		}
	}
	if !allowed {
		return &GroupMetricsResp{HasGroup: false, Metrics: []*BoardMetric{}}, nil
	}
	ms, err := s.repo.FindBoardTeamMetrics(ctx, req.TeamgroupID)
	if err != nil {
		return nil, err
	}
	if ms == nil {
		ms = []*BoardMetric{}
	}
	return &GroupMetricsResp{GroupID: req.TeamgroupID, HasGroup: true, GroupName: name, Metrics: ms}, nil
}

// BoardTask 我的任务看板（按任务负责人筛选 + 4 列）。
func (s *Service) BoardTask(ctx context.Context, actor *model.User, req BoardTaskReq) (*BoardTaskResp, error) {
	if actor == nil || strings.TrimSpace(actor.Account) == "" {
		return &BoardTaskResp{Columns: nil}, nil
	}
	teams, err := s.repo.FindBoardTeamgroups(ctx, actor.Account)
	if err != nil {
		return nil, err
	}
	req.TeamgroupID = selectBoardTeamgroup(req.TeamgroupID, teams)
	displayMap, err := s.loadAccountDisplayMap(ctx, actor)
	if err != nil {
		return nil, err
	}
	cols, summary, err := s.repo.FindBoardTaskList(ctx, req, displayMap)
	if err != nil {
		return nil, err
	}
	owners, err := s.repo.FindBoardTaskOwners(ctx, req.TeamgroupID, displayMap)
	if err != nil {
		return nil, err
	}
	return &BoardTaskResp{Columns: cols, Owners: owners, Teamgroups: teams, SelectedTeamgroupID: req.TeamgroupID, Summary: summary}, nil
}

func selectBoardTeamgroup(requested uint, teams []BoardTeamgroupOption) uint {
	for _, team := range teams {
		if team.ID == requested {
			return requested
		}
	}
	if len(teams) > 0 {
		return teams[0].ID
	}
	return 0
}
