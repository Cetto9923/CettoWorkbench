// =============================================================================
// 文件: internal/module/schedule/service_scheduling.go
// 模块: 排期工作台
// 类型: action
// 职责: 排期一体化弹窗故事、任务与项目执行装配。
// 依赖: internal/model
//       internal/module/schedule/repo_scheduling.go
// =============================================================================

package schedule

import (
	"context"
	"strconv"
	"strings"

	"workbench/internal/model"
)

func (s *Service) buildDemandSchedulingStories(
	ctx context.Context,
	demandID uint,
	projectsByProduct map[string][]DemandSchedulingProjectOption,
) ([]DemandSchedulingStoryItem, error) {
	rows, err := s.repo.GetDemandStories(ctx, demandID)
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return []DemandSchedulingStoryItem{}, nil
	}

	productIDs := make([]uint, 0, len(rows))
	accounts := make([]string, 0, len(rows))
	projectIDs := make([]uint, 0)
	for _, row := range rows {
		if row.Product > 0 {
			productIDs = append(productIDs, row.Product)
		}
		accounts = append(accounts, strings.TrimSpace(row.AssignedTo))
	}

	storyIDs := make([]uint, 0, len(rows))
	for _, row := range rows {
		storyIDs = append(storyIDs, row.ID)
	}
	storyTasks, err := s.repo.GetTasksByStories(ctx, storyIDs)
	if err != nil {
		return nil, err
	}
	for _, row := range rows {
		for _, task := range storyTasks[row.ID] {
			accounts = append(accounts, strings.TrimSpace(task.AssignedTo))
			projectIDs = append(projectIDs, task.Project, task.Execution)
		}
	}

	productNameByID, err := s.repo.FindProductsByIDs(ctx, productIDs)
	if err != nil {
		return nil, err
	}
	realnameByAccount, err := s.repo.FindUsersByAccounts(ctx, collectNonEmptyAccounts(accounts...))
	if err != nil {
		return nil, err
	}
	projectNameByID, err := s.repo.FindProjectsByIDs(ctx, uniqueUints(projectIDs))
	if err != nil {
		return nil, err
	}

	projectsByProductUint := productProjectsMapKeyToUint(projectsByProduct)

	items := make([]DemandSchedulingStoryItem, 0, len(rows))
	for _, row := range rows {
		tasks := buildDemandSchedulingTasks(storyTasks[row.ID], realnameByAccount, projectNameByID)
		assignedTo := strings.TrimSpace(row.AssignedTo)
		projects := projectsByProductUint[row.Product]
		if len(projects) == 0 {
			projects = []DemandSchedulingProjectOption{}
		}
		items = append(items, DemandSchedulingStoryItem{
			ID:             row.ID,
			Title:          strings.TrimSpace(row.Title),
			ProductID:      row.Product,
			ProductName:    productNameByID[row.Product],
			IsMain:         row.IsMainSystemAssociation > 0,
			Estimate:       row.Estimate,
			AssignedTo:     assignedTo,
			AssignedToName: resolveRealname(assignedTo, realnameByAccount),
			Tasks:          tasks,
			Projects:       projects,
		})
	}
	return items, nil
}

// buildDemandUserStories 装配业需级用户故事条目（来自 zt_demanduserstory）。
// EffectivePoint = revpoint > 0 ? revpoint : point（未校准 fallback 到建议值）。
func (s *Service) buildDemandUserStories(
	ctx context.Context,
	demandID uint,
) ([]UserStoryItem, error) {
	rows, err := s.repo.GetDemandUserStories(ctx, demandID)
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return []UserStoryItem{}, nil
	}

	productIDs := make([]uint, 0, len(rows))
	for _, row := range rows {
		if row.Product > 0 {
			productIDs = append(productIDs, row.Product)
		}
	}
	productNameByID, err := s.repo.FindProductsByIDs(ctx, productIDs)
	if err != nil {
		return nil, err
	}

	items := make([]UserStoryItem, 0, len(rows))
	for _, row := range rows {
		effectivePoint := row.Point
		if row.Revpoint > 0 {
			effectivePoint = row.Revpoint
		}
		items = append(items, UserStoryItem{
			ID:             row.ID,
			Role:           row.Role,
			GV:             row.GV,
			ProductID:      row.Product,
			ProductName:    productNameByID[row.Product],
			Revpoint:       row.Revpoint,
			PointLabel:     storyPointLabel(effectivePoint),
			EffectivePoint: effectivePoint,
		})
	}
	return items, nil
}

func (s *Service) buildProductProjectsMap(ctx context.Context, productIDs []uint) (map[string][]DemandSchedulingProjectOption, error) {
	out := make(map[string][]DemandSchedulingProjectOption)
	byProduct, err := s.repo.GetProjectsByProducts(ctx, productIDs)
	if err != nil {
		return nil, err
	}
	for _, productID := range uniqueUints(productIDs) {
		projectRows := byProduct[productID]
		projects := make([]DemandSchedulingProjectOption, 0, len(projectRows))
		for _, project := range projectRows {
			projects = append(projects, DemandSchedulingProjectOption{
				ID:   project.ID,
				Name: strings.TrimSpace(project.Name),
			})
		}
		out[formatUintKey(productID)] = projects
	}
	return out, nil
}

func productProjectsMapKeyToUint(src map[string][]DemandSchedulingProjectOption) map[uint][]DemandSchedulingProjectOption {
	if len(src) == 0 {
		return map[uint][]DemandSchedulingProjectOption{}
	}
	out := make(map[uint][]DemandSchedulingProjectOption, len(src))
	for key, projects := range src {
		id := parseUintString(key)
		if id == 0 {
			continue
		}
		out[id] = projects
	}
	return out
}

func formatUintKey(id uint) string {
	return strconv.FormatUint(uint64(id), 10)
}

func (s *Service) buildProjectExecutionsMap(
	ctx context.Context,
	productProjects map[string][]DemandSchedulingProjectOption,
) (map[string][]ZtExecutionOption, error) {
	out := make(map[string][]ZtExecutionOption)
	ids := make([]uint, 0)
	for _, projects := range productProjects {
		for _, project := range projects {
			ids = append(ids, project.ID)
		}
	}
	ids = uniqueUints(ids)
	byProject, err := s.repo.GetExecutionsByProjects(ctx, ids)
	if err != nil {
		return nil, err
	}
	for _, id := range ids {
		executions := make([]ZtExecutionOption, 0, len(byProject[id]))
		for _, row := range byProject[id] {
			executions = append(executions, ZtExecutionOption{ID: row.ID, Name: strings.TrimSpace(row.Name), Type: strings.TrimSpace(row.Type), Status: strings.TrimSpace(row.Status)})
		}
		out[formatUintKey(id)] = executions
	}

	return out, nil
}

// GetProductProjects 查询产品关联的项目列表（排期弹窗任务项目下拉）。
func (s *Service) GetProductProjects(ctx context.Context, actor *model.User, productID uint) ([]DemandSchedulingProjectOption, error) {
	_ = actorAccount(actor)
	if productID == 0 {
		return []DemandSchedulingProjectOption{}, nil
	}
	projectMap, err := s.buildProductProjectsMap(ctx, []uint{productID})
	if err != nil {
		return nil, err
	}
	projects := projectMap[formatUintKey(productID)]
	if projects == nil {
		return []DemandSchedulingProjectOption{}, nil
	}
	return projects, nil
}

func buildDemandSchedulingTasks(
	tasks []ZtTaskItem,
	realnameByAccount map[string]string,
	projectNameByID map[uint]string,
) []DemandSchedulingTaskItem {
	if len(tasks) == 0 {
		return []DemandSchedulingTaskItem{}
	}
	out := make([]DemandSchedulingTaskItem, 0, len(tasks))
	for _, task := range tasks {
		assignedTo := strings.TrimSpace(task.AssignedTo)
		out = append(out, DemandSchedulingTaskItem{
			ID:             task.ID,
			Name:           strings.TrimSpace(task.Name),
			Type:           strings.TrimSpace(task.Type),
			TypeLabel:      taskTypeLabel(task.Type),
			Pri:            normalizeTaskPriority(task.Pri),
			AssignedTo:     assignedTo,
			AssignedToName: resolveRealname(assignedTo, realnameByAccount),
			Estimate:       task.Estimate,
			EstStarted:     formatZenTaoDate(task.EstStarted),
			Deadline:       formatZenTaoDate(task.Deadline),
			Project:        task.Project,
			ProjectName:    projectNameByID[task.Project],
			Execution:      task.Execution,
			ExecutionName:  projectNameByID[task.Execution],
		})
	}
	return out
}

var taskTypeLabels = map[string]string{
	"devel":      "开发",
	"test":       "测试",
	"OLtest":     "上线测试",
	"review":     "评审",
	"affair":     "事务",
	"request":    "需求",
	"misc":       "其他",
	"codereview": "代码评审",
	"design":     "设计",
	"study":      "研究",
	"OLDebug":    "上线调试",
	"Online":     "上线",
	"Train":      "培训",
	"meeting":    "会议",
	"discuss":    "讨论",
	"ui":         "界面",
}

func taskTypeLabel(typ string) string {
	typ = strings.TrimSpace(typ)
	if label, ok := taskTypeLabels[typ]; ok {
		return label
	}
	return typ
}

func uniqueUints(ids []uint) []uint {
	if len(ids) == 0 {
		return []uint{}
	}
	seen := make(map[uint]struct{}, len(ids))
	out := make([]uint, 0, len(ids))
	for _, id := range ids {
		if id == 0 {
			continue
		}
		if _, exists := seen[id]; exists {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	return out
}

// GetProjectExecutions 查询项目下的执行列表（排期弹窗级联下拉）。
func (s *Service) GetProjectExecutions(ctx context.Context, actor *model.User, projectID uint) ([]ZtExecutionOption, error) {
	_ = actorAccount(actor)
	if projectID == 0 {
		return []ZtExecutionOption{}, nil
	}
	rows, err := s.repo.GetProjectExecutions(ctx, projectID)
	if err != nil {
		return nil, err
	}
	out := make([]ZtExecutionOption, 0, len(rows))
	for _, row := range rows {
		out = append(out, ZtExecutionOption{
			ID:     row.ID,
			Name:   strings.TrimSpace(row.Name),
			Type:   strings.TrimSpace(row.Type),
			Status: strings.TrimSpace(row.Status),
		})
	}
	return out, nil
}
