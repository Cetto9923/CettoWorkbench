// =============================================================================
// 文件: internal/module/schedule/service_story_tasks.go
// 模块: 排期工作台
// 类型: action
// 职责: 维护任务弹窗加载与保存业务逻辑。
// 依赖: internal/model
//       internal/module/schedule/repo_story_tasks.go
// =============================================================================

package schedule

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strings"

	"workbench/internal/model"
)

// GetStoryTasks 加载维护任务弹窗数据。
func (s *Service) GetStoryTasks(ctx context.Context, actor *model.User, storyID uint) (*StoryTasksResp, error) {
	_ = actorAccount(actor)
	if storyID == 0 {
		return nil, errors.New("研发需求 ID 无效")
	}

	detail, err := s.repo.GetStoryTaskDetail(ctx, storyID)
	if err != nil {
		return nil, err
	}

	taskRows, err := s.repo.GetStoryTasks(ctx, storyID)
	if err != nil {
		return nil, err
	}

	accounts := []string{detail.AssignedTo}
	projectIDs := make([]uint, 0)
	for _, task := range taskRows {
		accounts = append(accounts, strings.TrimSpace(task.AssignedTo))
		accounts = append(accounts, strings.TrimSpace(task.FinishedBy))
		if task.Project > 0 {
			projectIDs = append(projectIDs, task.Project)
		}
		if task.Execution > 0 {
			projectIDs = append(projectIDs, task.Execution)
		}
	}

	productNameByID, err := s.repo.FindProductsByIDs(ctx, []uint{detail.ProductID})
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

	projectRows, err := s.repo.GetProductProjects(ctx, detail.ProductID)
	if err != nil {
		return nil, err
	}
	projects := make([]DemandSchedulingProjectOption, 0, len(projectRows))
	for _, project := range projectRows {
		projects = append(projects, DemandSchedulingProjectOption{
			ID:   project.ID,
			Name: strings.TrimSpace(project.Name),
		})
	}

	users, err := s.repo.ListInsideUsersForScheduling(ctx)
	if err != nil {
		return nil, err
	}

	demandID := detail.FromDemand
	demandName := detail.DemandName
	if demandName == "" && demandID > 0 {
		demandName = fmt.Sprintf("REQ-%d", demandID)
	}

	return &StoryTasksResp{
		Story: StoryTaskStoryItem{
			ID:             detail.StoryID,
			Title:          detail.Title,
			ProductID:      detail.ProductID,
			ProductName:    productNameByID[detail.ProductID],
			AssignedToName: resolveRealname(detail.AssignedTo, realnameByAccount),
			Spec:           detail.Spec,
			Verify:         detail.Verify,
			DemandID:       demandID,
			DemandName:     demandName,
			WindowName:     detail.WindowName,
			ReleaseDate:    detail.ReleaseDate,
			Attachments:    detail.Attachments,
		},
		Tasks:              buildStoryTaskItems(taskRows, realnameByAccount, projectNameByID),
		Summary:            buildStoryTaskSummary(taskRows),
		Projects:           projects,
		Users:              users,
		DefaultProjectID:   detail.DefaultProjectID,
		DefaultExecutionID: detail.DefaultExecutionID,
	}, nil
}

func buildStoryTaskItems(
	tasks []ZtTaskItem,
	realnameByAccount map[string]string,
	projectNameByID map[uint]string,
) []StoryTaskItem {
	if len(tasks) == 0 {
		return []StoryTaskItem{}
	}
	out := make([]StoryTaskItem, 0, len(tasks))
	for _, task := range tasks {
		assignedTo := strings.TrimSpace(task.AssignedTo)
		finishedBy := strings.TrimSpace(task.FinishedBy)
		out = append(out, StoryTaskItem{
			ID:             task.ID,
			Type:           strings.TrimSpace(task.Type),
			TypeLabel:      taskTypeLabel(task.Type),
			Pri:            normalizeTaskPriority(task.Pri),
			Name:           strings.TrimSpace(task.Name),
			PriLabel:       formatTaskPriority(task.Pri),
			Status:         strings.TrimSpace(task.Status),
			StatusLabel:    taskStatusLabel(task.Status),
			AssignedTo:     assignedTo,
			AssignedToName: resolveRealname(assignedTo, realnameByAccount),
			FinishedBy:     finishedBy,
			FinishedByName: resolveRealname(finishedBy, realnameByAccount),
			FinishedDate:   formatZenTaoDate(task.FinishedDate),
			Estimate:       task.Estimate,
			Consumed:       task.Consumed,
			Left:           task.Left,
			Progress:       calculateTaskProgress(task.Estimate, task.Consumed, task.Left, task.Status),
			EstStarted:     formatZenTaoDate(task.EstStarted),
			Deadline:       formatZenTaoDate(task.Deadline),
			ProjectID:      task.Project,
			ProjectName:    projectNameByID[task.Project],
			ExecutionID:    task.Execution,
			ExecutionName:  projectNameByID[task.Execution],
		})
	}
	return out
}

func buildStoryTaskSummary(tasks []ZtTaskItem) StoryTaskSummary {
	summary := StoryTaskSummary{}
	for _, task := range tasks {
		summary.Total++
		switch strings.TrimSpace(task.Status) {
		case "wait":
			summary.WaitCount++
		case "doing":
			summary.DoingCount++
		}
		summary.EstimateTotal += task.Estimate
		summary.ConsumedTotal += task.Consumed
		summary.LeftTotal += task.Left
	}
	return summary
}

func formatTaskPriority(pri int) string {
	if pri <= 0 {
		return ""
	}
	return "P" + fmt.Sprintf("%d", normalizeTaskPriority(pri))
}

func normalizeTaskPriority(pri int) int {
	if pri < 0 || pri > 4 {
		return 3
	}
	return pri
}

func taskStatusLabel(status string) string {
	switch strings.TrimSpace(status) {
	case "wait":
		return "未开始"
	case "doing":
		return "进行中"
	case "done":
		return "已完成"
	case "pause":
		return "已暂停"
	case "cancel":
		return "已取消"
	case "closed":
		return "已关闭"
	default:
		return strings.TrimSpace(status)
	}
}

func calculateTaskProgress(estimate, consumed, left float64, status string) int {
	switch strings.TrimSpace(status) {
	case "done", "closed":
		return 100
	case "wait":
		if consumed <= 0 {
			return 0
		}
	}
	total := estimate
	if total <= 0 {
		total = consumed + left
	}
	if total <= 0 {
		return 0
	}
	progress := int(math.Round(consumed / total * 100))
	if progress < 0 {
		return 0
	}
	if progress > 100 {
		return 100
	}
	return progress
}

// SaveStoryTasks 保存维护任务弹窗中的任务变更。
func (s *Service) SaveStoryTasks(ctx context.Context, actor *model.User, storyID uint, req *SaveStoryTasksReq) error {
	if storyID == 0 {
		return errors.New("研发需求 ID 无效")
	}
	if req == nil {
		return errors.New("请求参数无效")
	}
	account := actorAccount(actor)
	if account == "" {
		return errors.New("未登录或无法识别当前用户")
	}

	detail, err := s.repo.GetStoryTaskDetail(ctx, storyID)
	if err != nil {
		return err
	}

	return s.repo.Transaction(ctx, func(txRepo *Repo) error {
		for _, taskReq := range req.Tasks {
			if err := s.applyStoryTaskSave(ctx, txRepo, account, storyID, detail.ProductID, req, taskReq); err != nil {
				return err
			}
		}
		return nil
	})
}

func (s *Service) applyStoryTaskSave(
	ctx context.Context,
	txRepo *Repo,
	account string,
	storyID, productID uint,
	req *SaveStoryTasksReq,
	taskReq SaveStoryTasksTask,
) error {
	switch strings.TrimSpace(taskReq.Action) {
	case "new":
		if !taskReq.Create {
			return nil
		}
		return s.applySingleSchedulingTask(ctx, txRepo, account, storyID, productID, SaveSchedulingTask{
			Action:      "new",
			ExecutionID: taskReq.ExecutionID,
			Type:        taskReq.Type,
			Pri:         taskReq.Pri,
			Name:        taskReq.Name,
			AssignedTo:  taskReq.AssignedTo,
			Estimate:    taskReq.Estimate,
			EstStarted:  taskReq.EstStarted,
			Deadline:    taskReq.Deadline,
		})

	case "edit":
		return s.applySingleSchedulingTask(ctx, txRepo, account, storyID, productID, SaveSchedulingTask{
			Action:      "edit",
			ID:          taskReq.ID,
			ExecutionID: taskReq.ExecutionID,
			Type:        taskReq.Type,
			Pri:         taskReq.Pri,
			Name:        taskReq.Name,
			AssignedTo:  taskReq.AssignedTo,
			Estimate:    taskReq.Estimate,
			EstStarted:  taskReq.EstStarted,
			Deadline:    taskReq.Deadline,
		})

	case "delete":
		return s.applySingleSchedulingTask(ctx, txRepo, account, storyID, productID, SaveSchedulingTask{
			Action: "delete",
			ID:     taskReq.ID,
		})

	default:
		return fmt.Errorf("unsupported task action: %s", taskReq.Action)
	}
}
