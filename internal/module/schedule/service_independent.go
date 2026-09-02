// =============================================================================
// 文件: internal/module/schedule/service_independent.go
// 模块: 排期工作台
// 类型: action
// 职责: 独立研发需求列表装配。
// 依赖: internal/model
//       internal/module/schedule/repo.go
// =============================================================================

package schedule

import (
	"context"
	"errors"
	"strings"

	"workbench/internal/model"
)

// ListIndependentStories 查询排期工作台独立研发需求 Tab 列表。
func (s *Service) ListIndependentStories(ctx context.Context, actor *model.User, req ListIndependentReq) (*ListIndependentResp, error) {
	account := actorAccount(actor)
	if account == "" {
		return &ListIndependentResp{Total: 0, Items: []IndependentStoryItem{}}, nil
	}

	req.Normalize()

	productIDs, err := s.getVisibleProductIDs(ctx, account)
	if err != nil {
		return nil, err
	}
	if len(productIDs) == 0 {
		return &ListIndependentResp{Total: 0, Items: []IndependentStoryItem{}}, nil
	}

	topStories, total, err := s.repo.ListIndependentStories(ctx, req, productIDs, account)
	if err != nil {
		return nil, err
	}
	if len(topStories) == 0 {
		return &ListIndependentResp{Total: total, Items: []IndependentStoryItem{}}, nil
	}

	topIDs := pluckStoryIDs(topStories)
	childStories, err := s.repo.FindChildStories(ctx, topIDs)
	if err != nil {
		return nil, err
	}
	childByParent := groupChildStoriesByParent(childStories)

	allStories := append([]ZtStory(nil), topStories...)
	allStories = append(allStories, childStories...)
	storyIDs := pluckStoryIDs(allStories)
	productIDsForName := collectStoryProductIDs(allStories)
	accounts := collectStoryAccounts(allStories)

	windowByStory, err := s.repo.FindStoryWindowMappings(ctx, storyIDs)
	if err != nil {
		return nil, err
	}
	teamgroupIDs := make([]uint, 0)
	for _, ref := range windowByStory {
		teamgroupIDs = appendUniqueUint(teamgroupIDs, ref.TeamgroupID)
	}

	taskStatByStory, err := s.repo.CountStoryTasks(ctx, storyIDs)
	if err != nil {
		return nil, err
	}
	productNameByID, err := s.repo.FindProductsByIDs(ctx, productIDsForName)
	if err != nil {
		return nil, err
	}
	teamgroupNameByID, err := s.loadTeamgroupDisplayNamesByIDs(ctx, teamgroupIDs)
	if err != nil {
		return nil, err
	}
	realnameByAccount, err := s.repo.FindUsersByAccounts(ctx, accounts)
	if err != nil {
		return nil, err
	}

	assembleCtx := independentStoryAssembleContext{
		childByParent:     childByParent,
		productNameByID:   productNameByID,
		windowByStory:     windowByStory,
		taskStatByStory:   taskStatByStory,
		teamgroupNameByID: teamgroupNameByID,
		realnameByAccount: realnameByAccount,
	}

	items := make([]IndependentStoryItem, 0, len(topStories))
	for _, top := range topStories {
		items = append(items, assembleCtx.buildIndependentStoryItem(top))
	}

	return &ListIndependentResp{Total: total, Items: items}, nil
}

// GetStoryScheduling 查询独立研发需求排期弹窗加载数据。
func (s *Service) GetStoryScheduling(ctx context.Context, actor *model.User, storyID uint) (*DemandSchedulingResp, error) {
	if storyID == 0 {
		return nil, errors.New("研发需求 ID 无效")
	}
	_ = actorAccount(actor)

	detail, err := s.repo.GetStorySchedulingDetail(ctx, storyID)
	if err != nil {
		return nil, err
	}
	detail.CanEditWindow = true
	windows, err := s.repo.ListUpcomingSchedulingWindows(ctx)
	if err != nil {
		return nil, err
	}
	users, err := s.repo.ListInsideUsersForScheduling(ctx)
	if err != nil {
		return nil, err
	}

	involvedProducts := []ZtProductOption{}
	if detail.MainSystemID > 0 {
		involvedProducts = append(involvedProducts, ZtProductOption{
			ID:   detail.MainSystemID,
			Name: detail.MainSystemName,
		})
	}

	return &DemandSchedulingResp{
		DemandSchedulingDetail: detail,
		InvolvedProducts:       involvedProducts,
		ProductProjects:        map[string][]DemandSchedulingProjectOption{},
		ProjectExecutions:      map[string][]ZtExecutionOption{},
		Stories:                []DemandSchedulingStoryItem{},
		Windows:                windows,
		Users:                  users,
	}, nil
}

func (s *Service) getVisibleProductIDs(ctx context.Context, account string) ([]uint, error) {
	isAdmin, err := s.repo.IsAdmin(ctx, account)
	if err != nil {
		return nil, err
	}
	if isAdmin {
		return s.repo.ListAllProductIDs(ctx)
	}
	products, err := s.repo.GetUserProducts(ctx, account)
	if err != nil {
		return nil, err
	}
	ids := make([]uint, 0, len(products))
	for _, product := range products {
		if product.ID > 0 {
			ids = append(ids, product.ID)
		}
	}
	return ids, nil
}

type independentStoryAssembleContext struct {
	childByParent     map[uint][]ZtStory
	productNameByID   map[uint]string
	windowByStory     map[uint]StoryWindowRef
	taskStatByStory   map[uint]StoryTaskStat
	teamgroupNameByID map[uint]string
	realnameByAccount map[string]string
}

func (c independentStoryAssembleContext) buildIndependentStoryItem(top ZtStory) IndependentStoryItem {
	children := c.childByParent[top.ID]
	childItems := c.buildChildItems(children)
	aggregateStories := storiesForIndependentAggregate(top, children)

	taskCount := sumStoryTaskCounts(children, c.taskStatByStory)
	if len(children) == 0 {
		taskCount = c.taskStatByStory[top.ID].Total
	}

	return IndependentStoryItem{
		ID:             top.ID,
		Title:          strings.TrimSpace(top.Title),
		Pri:            top.Pri,
		ProductName:    c.productNameByID[top.Product],
		AssignedToName: resolveRealname(strings.TrimSpace(top.AssignedTo), c.realnameByAccount),
		Stage:          calcIndependentStoryStage(aggregateStories, c.windowByStory, c.taskStatByStory),
		WindowName:     pickBizWindowName(aggregateStories, c.windowByStory),
		TaskCount:      taskCount,
		TeamgroupName:  c.pickStoryTeamgroupName(aggregateStories),
		Children:       childItems,
	}
}

func (c independentStoryAssembleContext) buildChildItems(children []ZtStory) []IndependentStoryItem {
	if len(children) == 0 {
		return []IndependentStoryItem{}
	}
	items := make([]IndependentStoryItem, 0, len(children))
	for _, child := range children {
		subtree := []ZtStory{child}
		items = append(items, IndependentStoryItem{
			ID:             child.ID,
			Title:          strings.TrimSpace(child.Title),
			Pri:            child.Pri,
			ProductName:    c.productNameByID[child.Product],
			AssignedToName: resolveRealname(strings.TrimSpace(child.AssignedTo), c.realnameByAccount),
			Stage:          calcIndependentStoryStage(subtree, c.windowByStory, c.taskStatByStory),
			WindowName:     pickBizWindowName(subtree, c.windowByStory),
			TaskCount:      c.taskStatByStory[child.ID].Total,
			TeamgroupName:  c.pickStoryTeamgroupName(subtree),
		})
	}
	return items
}

func (c independentStoryAssembleContext) pickStoryTeamgroupName(stories []ZtStory) string {
	for _, story := range stories {
		ref, ok := c.windowByStory[story.ID]
		if !ok || ref.TeamgroupID == 0 {
			continue
		}
		if name := strings.TrimSpace(c.teamgroupNameByID[ref.TeamgroupID]); name != "" {
			return name
		}
	}
	return ""
}

func calcIndependentStoryStage(
	stories []ZtStory,
	windowByStory map[uint]StoryWindowRef,
	taskStatByStory map[uint]StoryTaskStat,
) string {
	if len(stories) == 0 {
		return StageNoWindow
	}
	if allStoriesHaveNoWindow(stories, windowByStory) {
		return StageNoWindow
	}
	taskTotal, unassignedTotal := sumMainSystemTasks(stories, taskStatByStory)
	if taskTotal == 0 {
		return StageNoTask
	}
	if unassignedTotal > 0 {
		return StageTaskUnassigned
	}
	return IndependentStageTaskAssigned
}

// storiesForIndependentAggregate 父行聚合范围：有子需求时只看子需求，否则看自身。
func storiesForIndependentAggregate(top ZtStory, children []ZtStory) []ZtStory {
	if len(children) > 0 {
		return children
	}
	return []ZtStory{top}
}

func groupChildStoriesByParent(children []ZtStory) map[uint][]ZtStory {
	out := make(map[uint][]ZtStory, len(children))
	for _, child := range children {
		out[child.Parent] = append(out[child.Parent], child)
	}
	return out
}

func collectStoryProductIDs(stories []ZtStory) []uint {
	seen := make(map[uint]struct{})
	ids := make([]uint, 0, len(stories))
	for _, story := range stories {
		if story.Product == 0 {
			continue
		}
		if _, ok := seen[story.Product]; ok {
			continue
		}
		seen[story.Product] = struct{}{}
		ids = append(ids, story.Product)
	}
	return ids
}

func collectStoryAccounts(stories []ZtStory) []string {
	seen := make(map[string]struct{})
	accounts := make([]string, 0, len(stories))
	for _, story := range stories {
		account := strings.TrimSpace(story.AssignedTo)
		if account == "" {
			continue
		}
		if _, ok := seen[account]; ok {
			continue
		}
		seen[account] = struct{}{}
		accounts = append(accounts, account)
	}
	return accounts
}

func sumStoryTaskCounts(children []ZtStory, taskStatByStory map[uint]StoryTaskStat) int {
	total := 0
	for _, child := range children {
		total += taskStatByStory[child.ID].Total
	}
	return total
}
