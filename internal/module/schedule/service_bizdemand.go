// =============================================================================
// 文件: internal/module/schedule/service_bizdemand.go
// 模块: 排期工作台
// 类型: action
// 职责: 业需列表装配与阶段计算。
// 依赖: internal/model
//       internal/module/schedule/repo.go
// =============================================================================

package schedule

import (
	"context"
	"errors"
	"strconv"
	"strings"

	"workbench/internal/model"
)

// ListBizDemands 查询排期工作台业务需求 Tab 列表。
func (s *Service) ListBizDemands(ctx context.Context, actor *model.User, req ListBizDemandsReq) (*ListBizDemandsResp, error) {
	account := actorAccount(actor)
	if account == "" {
		return &ListBizDemandsResp{Total: 0, Items: []BizDemandItem{}}, nil
	}

	req.Normalize()

	poolIDs, err := s.repo.GetUserDemandPools(ctx, account)
	if err != nil {
		return nil, err
	}
	if len(poolIDs) == 0 {
		return &ListBizDemandsResp{Total: 0, Items: []BizDemandItem{}}, nil
	}

	topDemands, total, err := s.repo.ListBizDemands(ctx, req, poolIDs, account)
	if err != nil {
		return nil, err
	}
	if len(topDemands) == 0 {
		return &ListBizDemandsResp{Total: total, Items: []BizDemandItem{}}, nil
	}

	topIDs := pluckDemandIDs(topDemands)

	childDemands, err := s.repo.FindChildDemandsByParents(ctx, topIDs)
	if err != nil {
		return nil, err
	}
	childByParent := groupChildDemandsByParent(childDemands)

	allDemandIDs := mergeDemandIDs(topIDs, pluckDemandIDs(childDemands))

	stories, err := s.repo.FindStoriesByDemands(ctx, allDemandIDs)
	if err != nil {
		return nil, err
	}
	storiesByDemand := groupStoriesByFromDemand(stories)

	productCountByDemand, err := s.repo.CountClarifyProductsByDemands(ctx, allDemandIDs)
	if err != nil {
		return nil, err
	}
	productIDs := collectBizDemandProductIDs(topDemands, childDemands, stories)
	storyIDs := pluckStoryIDs(stories)
	teamgroupIDs := collectBizDemandTeamgroupIDs(topDemands)
	accounts := collectBizDemandAccounts(topDemands, childDemands, stories)

	productNameByID, err := s.repo.FindProductsByIDs(ctx, productIDs)
	if err != nil {
		return nil, err
	}
	windowByStory, err := s.repo.FindStoryWindowMappings(ctx, storyIDs)
	if err != nil {
		return nil, err
	}
	taskStatByStory, err := s.repo.CountStoryTasks(ctx, storyIDs)
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

	assembleCtx := bizDemandAssembleContext{
		childByParent:        childByParent,
		storiesByDemand:      storiesByDemand,
		productCountByDemand: productCountByDemand,
		productNameByID:      productNameByID,
		windowByStory:        windowByStory,
		taskStatByStory:      taskStatByStory,
		teamgroupNameByID:    teamgroupNameByID,
		realnameByAccount:    realnameByAccount,
	}

	items := make([]BizDemandItem, 0, len(topDemands))
	for _, top := range topDemands {
		items = append(items, assembleCtx.buildBizDemandItem(top))
	}

	return &ListBizDemandsResp{Total: total, Items: items}, nil
}

type bizDemandAssembleContext struct {
	childByParent        map[int][]ZtDemand
	storiesByDemand      map[uint][]ZtStory
	productCountByDemand map[uint]int
	productNameByID      map[uint]string
	windowByStory        map[uint]StoryWindowRef
	taskStatByStory      map[uint]StoryTaskStat
	teamgroupNameByID    map[uint]string
	realnameByAccount    map[string]string
}

func (c bizDemandAssembleContext) buildBizDemandItem(top ZtDemand) BizDemandItem {
	children := c.childByParent[int(top.ID)]
	subtreeStories := collectSubtreeStories(top.ID, children, c.storiesByDemand)
	mainSystemStories := filterMainSystemStories(subtreeStories)
	teamgroupName := c.teamgroupName(top.TeamGroup)

	return BizDemandItem{
		ID:               top.ID,
		Name:             strings.TrimSpace(top.Name),
		Pri:              parseDemandPri(top.Pri),
		Status:           strings.TrimSpace(top.Status),
		MainSystemName:   c.productNameByID[parseUintString(top.MainSystem)],
		ExtraSystemCount: extraSystemCount(c.productCountByDemand[top.ID]),
		TeamgroupName:    teamgroupName,
		OwnerName:        resolveDemandOwner(top.BRA, c.realnameByAccount),
		Stage:            calcBizDemandStage(subtreeStories, mainSystemStories, c.windowByStory, c.taskStatByStory),
		WindowName:       pickBizWindowName(subtreeStories, c.windowByStory),
		Children:         c.buildSubDemandItems(top, children),
		Stories:          c.buildStoryItems(top.TeamGroup, teamgroupName, c.storiesByDemand[top.ID]),
	}
}

func (c bizDemandAssembleContext) buildSubDemandItems(parent ZtDemand, children []ZtDemand) []SubDemandItem {
	if len(children) == 0 {
		return []SubDemandItem{}
	}
	parentTeamgroupName := c.teamgroupName(parent.TeamGroup)
	items := make([]SubDemandItem, 0, len(children))
	for _, child := range children {
		childStories := c.storiesByDemand[child.ID]
		subtreeStories := append([]ZtStory(nil), childStories...)
		items = append(items, SubDemandItem{
			ID:               child.ID,
			Name:             strings.TrimSpace(child.Name),
			Pri:              parseDemandPri(child.Pri),
			Status:           strings.TrimSpace(child.Status),
			MainSystemName:   c.productNameByID[parseUintString(child.MainSystem)],
			ExtraSystemCount: extraSystemCount(c.productCountByDemand[child.ID]),
			TeamgroupName:    parentTeamgroupName,
			OwnerName:        resolveDemandOwner(child.BRA, c.realnameByAccount),
			WindowName:       pickBizWindowName(subtreeStories, c.windowByStory),
			Stories:          c.buildStoryItems(parent.TeamGroup, parentTeamgroupName, childStories),
		})
	}
	return items
}

func (c bizDemandAssembleContext) buildStoryItems(teamGroup, teamgroupName string, stories []ZtStory) []StoryItem {
	if len(stories) == 0 {
		return []StoryItem{}
	}
	_ = teamGroup
	items := make([]StoryItem, 0, len(stories))
	for _, story := range stories {
		windowRef := c.windowByStory[story.ID]
		taskStat := c.taskStatByStory[story.ID]
		assignedTo := strings.TrimSpace(story.AssignedTo)
		items = append(items, StoryItem{
			ID:                      story.ID,
			Title:                   strings.TrimSpace(story.Title),
			Pri:                     story.Pri,
			ProductName:             c.productNameByID[story.Product],
			Stage:                   calcStoryStage(story.ID, c.windowByStory),
			WindowName:              windowRef.WindowName,
			TeamgroupName:           teamgroupName,
			AssignedTo:              assignedTo,
			AssignedToName:          resolveRealname(assignedTo, c.realnameByAccount),
			TaskCount:               taskStat.Total,
			IsMainSystemAssociation: story.IsMainSystemAssociation,
		})
	}
	return items
}

func (c bizDemandAssembleContext) teamgroupName(teamGroup string) string {
	return c.teamgroupNameByID[parseUintString(teamGroup)]
}
func pluckDemandIDs(demands []ZtDemand) []uint {
	ids := make([]uint, 0, len(demands))
	for _, demand := range demands {
		if demand.ID == 0 {
			continue
		}
		ids = append(ids, demand.ID)
	}
	return ids
}

func mergeDemandIDs(a, b []uint) []uint {
	if len(b) == 0 {
		return append([]uint(nil), a...)
	}
	out := make([]uint, 0, len(a)+len(b))
	seen := make(map[uint]struct{}, len(a)+len(b))
	for _, id := range append(a, b...) {
		if id == 0 {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	return out
}

func groupChildDemandsByParent(children []ZtDemand) map[int][]ZtDemand {
	out := make(map[int][]ZtDemand, len(children))
	for _, child := range children {
		out[child.Parent] = append(out[child.Parent], child)
	}
	return out
}

func groupStoriesByFromDemand(stories []ZtStory) map[uint][]ZtStory {
	out := make(map[uint][]ZtStory, len(stories))
	for _, story := range stories {
		out[story.FromDemand] = append(out[story.FromDemand], story)
	}
	return out
}

func collectSubtreeStories(topID uint, children []ZtDemand, storiesByDemand map[uint][]ZtStory) []ZtStory {
	out := append([]ZtStory(nil), storiesByDemand[topID]...)
	for _, child := range children {
		out = append(out, storiesByDemand[child.ID]...)
	}
	return out
}

func filterMainSystemStories(stories []ZtStory) []ZtStory {
	out := make([]ZtStory, 0, len(stories))
	for _, story := range stories {
		if story.IsMainSystemAssociation == 1 {
			out = append(out, story)
		}
	}
	return out
}

func calcBizDemandStage(
	allStories []ZtStory,
	mainStories []ZtStory,
	windowByStory map[uint]StoryWindowRef,
	taskStatByStory map[uint]StoryTaskStat,
) string {
	// 业需窗口仅经 story → planstory → versionwindowproduct 关联，无 story 时不可能有窗口。
	// 故须先判「未转研发」，再判「未关联窗口」，否则 len(stories)==0 会误落未关联窗口。
	if len(allStories) == 0 {
		return StageNoStory
	}
	if allStoriesHaveNoWindow(allStories, windowByStory) {
		return StageNoWindow
	}

	taskTotal, unassignedTotal := sumMainSystemTasks(mainStories, taskStatByStory)
	if taskTotal == 0 {
		return StageNoTask
	}
	if unassignedTotal > 0 {
		return StageTaskUnassigned
	}
	return StageTaskAssigned
}
func calcStoryStage(storyID uint, windowByStory map[uint]StoryWindowRef) string {
	if ref, ok := windowByStory[storyID]; ok && ref.WindowID > 0 {
		return StoryStageHasWindow
	}
	return StoryStageNoWindow
}
func extraSystemCount(distinctProductCount int) int {
	if distinctProductCount <= 1 {
		return 0
	}
	return distinctProductCount - 1
}
func collectBizDemandProductIDs(topDemands, childDemands []ZtDemand, stories []ZtStory) []uint {
	seen := make(map[uint]struct{})
	ids := make([]uint, 0)
	add := func(id uint) {
		if id == 0 {
			return
		}
		if _, ok := seen[id]; ok {
			return
		}
		seen[id] = struct{}{}
		ids = append(ids, id)
	}
	for _, demand := range append(topDemands, childDemands...) {
		add(parseUintString(demand.MainSystem))
	}
	for _, story := range stories {
		add(story.Product)
	}
	return ids
}

func collectBizDemandTeamgroupIDs(topDemands []ZtDemand) []uint {
	seen := make(map[uint]struct{})
	ids := make([]uint, 0, len(topDemands))
	for _, demand := range topDemands {
		id := parseUintString(demand.TeamGroup)
		if id == 0 {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		ids = append(ids, id)
	}
	return ids
}

func collectBizDemandAccounts(topDemands, childDemands []ZtDemand, stories []ZtStory) []string {
	seen := make(map[string]struct{})
	accounts := make([]string, 0)
	add := func(account string) {
		account = strings.TrimSpace(account)
		if account == "" {
			return
		}
		if _, ok := seen[account]; ok {
			return
		}
		seen[account] = struct{}{}
		accounts = append(accounts, account)
	}
	for _, demand := range append(topDemands, childDemands...) {
		add(demand.BRA)
	}
	for _, story := range stories {
		add(story.AssignedTo)
	}
	return accounts
}
func parseDemandPri(pri string) int {
	value, err := strconv.Atoi(strings.TrimSpace(pri))
	if err != nil {
		return 0
	}
	return value
}

func parseUintString(raw string) uint {
	value, err := strconv.ParseUint(strings.TrimSpace(raw), 10, 64)
	if err != nil {
		return 0
	}
	return uint(value)
}

// GetDemandScheduling 查询排期一体化弹窗加载数据（业需详情、窗口与用户下拉）。
func (s *Service) GetDemandScheduling(ctx context.Context, actor *model.User, demandID uint) (*DemandSchedulingResp, error) {
	if demandID == 0 {
		return nil, errors.New("业需 ID 无效")
	}
	_ = actorAccount(actor)

	detail, err := s.repo.GetDemandSchedulingDetail(ctx, demandID)
	if err != nil {
		return nil, err
	}
	windows, err := s.repo.ListUpcomingSchedulingWindows(ctx)
	if err != nil {
		return nil, err
	}
	users, err := s.repo.ListInsideUsersForScheduling(ctx)
	if err != nil {
		return nil, err
	}
	involvedProducts, err := s.repo.GetDemandInvolvedProducts(ctx, demandID)
	if err != nil {
		return nil, err
	}
	productIDs := make([]uint, 0, len(involvedProducts))
	for _, product := range involvedProducts {
		if product.ID > 0 {
			productIDs = append(productIDs, product.ID)
		}
	}
	productProjects, err := s.buildProductProjectsMap(ctx, productIDs)
	if err != nil {
		return nil, err
	}
	stories, err := s.buildDemandSchedulingStories(ctx, demandID, productProjects)
	if err != nil {
		return nil, err
	}
	userStories, err := s.buildDemandUserStories(ctx, demandID)
	if err != nil {
		return nil, err
	}
	projectExecutions, err := s.buildProjectExecutionsMap(ctx, productProjects)
	if err != nil {
		return nil, err
	}
	return &DemandSchedulingResp{
		DemandSchedulingDetail: detail,
		InvolvedProducts:       involvedProducts,
		ProductProjects:        productProjects,
		ProjectExecutions:      projectExecutions,
		Stories:                stories,
		UserStories:            userStories,
		Windows:                windows,
		Users:                  users,
	}, nil
}
