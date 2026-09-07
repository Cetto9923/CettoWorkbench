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
	"workbench/internal/module/po/primaryaction"
	"workbench/internal/pkg/errorx"
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
	if err := s.attachPrimaryActions(ctx, actor, tree); err != nil {
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
	// 对象级鉴权：通过故事抽屉查看特定故事任务时，必须校验当前账号对该故事的对象级访问权限
	if req.StoryID > 0 && !actor.IsSuperAdmin {
		allowed, err := s.repo.CheckStoryAccess(ctx, actor.Account, uint(req.StoryID))
		if err != nil {
			return nil, err
		}
		if !allowed {
			return nil, errorx.New(errorx.ErrCodeForbidden, "无权访问该研发需求任务")
		}
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

// attachPrimaryActions 为需求树每个节点附加服务端 primaryAction（Stage 5）。
//
// 先遍历一次树收集需求 / 故事 ID（去重），再走 DeriveDemandPrimaryActions /
// DeriveStoryPrimaryActions 的 IN (?) 批量派生，最后回填到每个节点。
//
// 子节点 demand/sub_demand 共用 demandIDs；独立研发需求走独立 storyIDs 标记。
func (s *Service) attachPrimaryActions(
	ctx context.Context,
	actor *model.User,
	tree []*BoardDemandItem,
) error {
	if len(tree) == 0 {
		return nil
	}

	demandIDs := make([]uint, 0)
	storyIDs := make([]uint, 0)
	var walk func(node *BoardDemandItem)
	walk = func(node *BoardDemandItem) {
		if node == nil {
			return
		}
		switch node.Kind {
		case "demand", "sub_demand":
			if node.ID > 0 {
				demandIDs = append(demandIDs, uint(node.ID))
			}
		case "story":
			if node.ID > 0 {
				if node.Independent {
					storyIDs = append(storyIDs, uint(node.ID))
				} else {
					// 树内研需独立派生（Independent=false），按非独立走。
					storyIDs = append(storyIDs, uint(node.ID))
				}
			}
		}
		for _, child := range node.Children {
			walk(child)
		}
	}
	for _, root := range tree {
		walk(root)
	}

	demandActions, err := s.DeriveDemandPrimaryActions(ctx, actor, demandIDs)
	if err != nil {
		return err
	}
	storyActions, err := s.DeriveStoryPrimaryActions(ctx, actor, storyIDs, false)
	if err != nil {
		return err
	}

	var fill func(node *BoardDemandItem)
	fill = func(node *BoardDemandItem) {
		if node == nil {
			return
		}
		switch node.Kind {
		case "demand", "sub_demand":
			if pa, ok := demandActions[uint(node.ID)]; ok {
				paCopy := pa
				node.PrimaryAction = &paCopy
			} else {
				none := primaryaction.None()
				node.PrimaryAction = &none
			}
		case "story":
			if pa, ok := storyActions[uint(node.ID)]; ok {
				paCopy := pa
				node.PrimaryAction = &paCopy
			} else {
				none := primaryaction.None()
				node.PrimaryAction = &none
			}
		}
		for _, child := range node.Children {
			fill(child)
		}
	}
	for _, root := range tree {
		fill(root)
	}
	return nil
}
