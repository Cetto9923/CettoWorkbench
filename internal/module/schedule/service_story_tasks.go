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
		Tasks:             buildStoryTaskItems(taskRows, realnameByAccount, projectNameByID),
		Projects:          projects,
		Users:             users,
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
		out = append(out, StoryTaskItem{
			ID:             task.ID,
			Type:           strings.TrimSpace(task.Type),
			TypeLabel:      taskTypeLabel(task.Type),
			Name:           strings.TrimSpace(task.Name),
			AssignedTo:     assignedTo,
			AssignedToName: resolveRealname(assignedTo, realnameByAccount),
			Estimate:       task.Estimate,
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
