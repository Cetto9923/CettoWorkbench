// =============================================================================
// 文件: internal/module/schedule/service_scheduling_save.go
// 模块: 排期工作台
// 类型: action
// 职责: 排期一体化「确认并同步」保存业务逻辑。
// 依赖: internal/model
//       internal/module/schedule/repo_scheduling_write.go
// =============================================================================

package schedule

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"workbench/internal/model"
)

// SaveScheduling 保存业需排期并同步禅道研发需求与任务。
func (s *Service) SaveScheduling(ctx context.Context, actor *model.User, demandID uint, req *SaveSchedulingReq) error {
	if demandID == 0 {
		return errors.New("业需 ID 无效")
	}
	if req == nil {
		return errors.New("请求参数无效")
	}
	account := actorAccount(actor)
	if account == "" {
		return errors.New("未登录或无法识别当前用户")
	}

	mainSystemID, err := s.repo.GetDemandMainSystem(ctx, demandID)
	if err != nil {
		return err
	}

	// 进事务前做整体校验:任一目标系统不在当前用户有权限的产品集合内则零写入、返回提示。
	notice, err := s.precheckSchedulingProducts(ctx, req.WindowID, req.Stories, account)
	if err != nil {
		return err
	}
	if notice != nil {
		return notice // *ProductAccessNoticeError,由 handler 用 errors.As 识别
	}

	return s.repo.Transaction(ctx, func(txRepo *Repo) error {
		for _, storyReq := range req.Stories {
			storyID, productID, _, err := s.applySchedulingStory(ctx, txRepo, account, demandID, mainSystemID, req.WindowID, storyReq)
			if err != nil {
				return err
			}
			if strings.TrimSpace(storyReq.Action) == "delete" {
				continue
			}
			if err := s.applySchedulingTasks(ctx, txRepo, account, storyID, productID, storyReq.Tasks); err != nil {
				return err
			}
		}

		if err := txRepo.UpdateDemandScheduling(ctx, demandID, buildDemandSchedulingUpdates(req, account)); err != nil {
			return fmt.Errorf("update demand scheduling: %w", err)
		}

		return txRepo.SaveDemandLevelWindow(ctx, demandID, uint64(req.WindowID), account)
	})
}

func (s *Service) applySchedulingStory(
	ctx context.Context,
	txRepo *Repo,
	account string,
	demandID uint,
	mainSystemID uint,
	windowID uint,
	storyReq SaveSchedulingStory,
) (storyID uint, productID uint, planID uint, err error) {
	switch strings.TrimSpace(storyReq.Action) {
	case "new":
		productID = storyReq.ProductID
		planID, err = s.resolvePlanForProduct(ctx, txRepo, account, windowID, productID)
		if err != nil {
			return 0, 0, 0, err
		}
		isMain := "0"
		if productID > 0 && productID == mainSystemID {
			isMain = "1"
		}
		storyID, err = txRepo.CreateStory(ctx, &ZtStoryInsert{
			Product:                 productID,
			Title:                   storyReq.Title,
			AssignedTo:              storyReq.AssignedTo,
			Estimate:                storyReq.Estimate,
			FromDemand:              demandID,
			IsMainSystemAssociation: isMain,
			OpenedBy:                account,
		})
		if err != nil {
			return 0, 0, 0, fmt.Errorf("create story: %w", err)
		}
		if err := txRepo.CreateStorySpec(ctx, &ZtStorySpec{
			Story:   storyID,
			Version: 1,
			Title:   storyReq.Title,
			Spec:    storyReq.Spec,
		}); err != nil {
			return 0, 0, 0, fmt.Errorf("create story spec: %w", err)
		}
		if err := txRepo.EnsurePlanStoryRelation(ctx, planID, storyID); err != nil {
			return 0, 0, 0, fmt.Errorf("ensure plan story relation: %w", err)
		}
		if err := txRepo.CreateAction(ctx, "story", storyID, "Opened", account, productID, 0, 0); err != nil {
			return 0, 0, 0, fmt.Errorf("create story action: %w", err)
		}
		return storyID, productID, planID, nil

	case "edit":
		storyID = storyReq.ID
		productID = storyReq.ProductID
		if err := txRepo.UpdateStory(ctx, storyID, map[string]interface{}{
			"title":          strings.TrimSpace(storyReq.Title),
			"assignedTo":     strings.TrimSpace(storyReq.AssignedTo),
			"product":        storyReq.ProductID,
			"lastEditedBy":   account,
			"lastEditedDate": time.Now(),
		}); err != nil {
			return 0, 0, 0, fmt.Errorf("update story %d: %w", storyID, err)
		}
		if err := txRepo.CreateAction(ctx, "story", storyID, "Edited", account, storyReq.ProductID, 0, 0); err != nil {
			return 0, 0, 0, fmt.Errorf("create story action: %w", err)
		}
		// 同步计划关联:解析目标计划 → 从该 story 的其他计划移除 → 幂等关联到目标计划。
		// 对 productID 未变化的情况也幂等(Ensure INSERT IGNORE + Remove 保留当前 plan)。
		newPlanID, err := s.resolvePlanForProduct(ctx, txRepo, account, windowID, storyReq.ProductID)
		if err != nil {
			return 0, 0, 0, fmt.Errorf("resolve plan for edited story %d: %w", storyID, err)
		}
		if err := txRepo.RemoveStoryFromOtherPlans(ctx, storyID, newPlanID); err != nil {
			return 0, 0, 0, fmt.Errorf("remove story %d from other plans: %w", storyID, err)
		}
		if err := txRepo.EnsurePlanStoryRelation(ctx, newPlanID, storyID); err != nil {
			return 0, 0, 0, fmt.Errorf("ensure plan-story relation: %w", err)
		}
		return storyID, productID, 0, nil

	case "delete":
		storyID = storyReq.ID
		if err := txRepo.CloseStory(ctx, storyID, account); err != nil {
			return 0, 0, 0, fmt.Errorf("close story %d: %w", storyID, err)
		}
		if err := txRepo.CreateAction(ctx, "story", storyID, "Closed", account, 0, 0, 0); err != nil {
			return 0, 0, 0, fmt.Errorf("create story action: %w", err)
		}
		return storyID, 0, 0, nil

	default:
		return 0, 0, 0, fmt.Errorf("unsupported story action: %s", storyReq.Action)
	}
}

func (s *Service) applySchedulingTasks(
	ctx context.Context,
	txRepo *Repo,
	account string,
	storyID uint,
	productID uint,
	tasks []SaveSchedulingTask,
) error {
	for _, taskReq := range tasks {
		if err := s.applySingleSchedulingTask(ctx, txRepo, account, storyID, productID, taskReq); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) applySingleSchedulingTask(
	ctx context.Context,
	txRepo *Repo,
	account string,
	storyID uint,
	productID uint,
	taskReq SaveSchedulingTask,
) error {
	switch strings.TrimSpace(taskReq.Action) {
	case "new":
		projectID, err := txRepo.GetProjectIDByExecution(ctx, taskReq.ExecutionID)
		if err != nil {
			return err
		}
		taskID, err := txRepo.CreateTask(ctx, &ZtTaskInsert{
			Name:       taskReq.Name,
			Type:       taskReq.Type,
			Story:      storyID,
			Project:    projectID,
			Execution:  taskReq.ExecutionID,
			AssignedTo: taskReq.AssignedTo,
			Estimate:   taskReq.Estimate,
			EstStarted: taskReq.EstStarted,
			Deadline:   taskReq.Deadline,
			OpenedBy:   account,
		})
		if err != nil {
			return fmt.Errorf("create task: %w", err)
		}
		if err := txRepo.CreateTaskSpec(ctx, &ZtTaskSpec{
			Task:       taskID,
			Version:    1,
			Name:       taskReq.Name,
			EstStarted: taskReq.EstStarted,
			Deadline:   taskReq.Deadline,
		}); err != nil {
			return fmt.Errorf("create task spec: %w", err)
		}
		if err := txRepo.CreateAction(ctx, "task", taskID, "Opened", account, productID, projectID, taskReq.ExecutionID); err != nil {
			return fmt.Errorf("create task action: %w", err)
		}
		if err := txRepo.LinkStoryToProjectAndExecution(ctx, storyID, productID, projectID, taskReq.ExecutionID, account); err != nil {
			return fmt.Errorf("link story to project/execution: %w", err)
		}

	case "edit":
		projectID, err := txRepo.GetProjectIDByExecution(ctx, taskReq.ExecutionID)
		if err != nil {
			return err
		}
		if err := txRepo.UpdateTask(ctx, taskReq.ID, map[string]interface{}{
			"name":           strings.TrimSpace(taskReq.Name),
			"type":           strings.TrimSpace(taskReq.Type),
			"assignedTo":     strings.TrimSpace(taskReq.AssignedTo),
			"estimate":       taskReq.Estimate,
			"left":           taskReq.Estimate,
			"estStarted":     nullableDateValue(taskReq.EstStarted),
			"deadline":       nullableDateValue(taskReq.Deadline),
			"execution":      taskReq.ExecutionID,
			"project":        projectID,
			"lastEditedBy":   account,
			"lastEditedDate": time.Now(),
		}); err != nil {
			return fmt.Errorf("update task %d: %w", taskReq.ID, err)
		}
		if err := txRepo.CreateAction(ctx, "task", taskReq.ID, "Edited", account, productID, projectID, taskReq.ExecutionID); err != nil {
			return fmt.Errorf("create task action: %w", err)
		}
		if err := txRepo.LinkStoryToProjectAndExecution(ctx, storyID, productID, projectID, taskReq.ExecutionID, account); err != nil {
			return fmt.Errorf("link story to project/execution: %w", err)
		}

	case "delete":
		if err := txRepo.CloseTask(ctx, taskReq.ID, account); err != nil {
			return fmt.Errorf("close task %d: %w", taskReq.ID, err)
		}
		if err := txRepo.CreateAction(ctx, "task", taskReq.ID, "Closed", account, productID, 0, 0); err != nil {
			return fmt.Errorf("create task action: %w", err)
		}
	}
	return nil
}

func (s *Service) resolvePlanForProduct(
	ctx context.Context,
	txRepo *Repo,
	account string,
	windowID uint,
	productID uint,
) (uint, error) {
	vwp, err := txRepo.FindWindowProductPlan(ctx, windowID, productID)
	if err != nil {
		return 0, err
	}
	if vwp != nil && vwp.PlanID != nil && *vwp.PlanID > 0 {
		return *vwp.PlanID, nil
	}

	// vwp != nil:系统已在窗口中但无计划,自动创建计划并回填窗口产品。
	if vwp != nil {
		window, err := txRepo.FindByID(ctx, uint64(windowID))
		if err != nil {
			return 0, err
		}
		if window == nil {
			return 0, errors.New("版本窗口不存在")
		}

		endDate := window.ReleaseDate.Format("2006-01-02")
		beginDate := endDate
		if window.StartDate != nil {
			beginDate = window.StartDate.Format("2006-01-02")
		}
		title := strings.TrimSpace(window.Name)
		if title == "" {
			title = endDate
		}

		planID, err := txRepo.CreateProductPlan(ctx, productID, title, beginDate, endDate, account)
		if err != nil {
			return 0, fmt.Errorf("create product plan: %w", err)
		}
		if err := txRepo.UpdateWindowProductPlanID(ctx, vwp.ID, planID, account); err != nil {
			return 0, err
		}
		return planID, nil
	}

	// vwp == nil:系统不在窗口,使用禅道已有匹配计划(前置校验保证非空,这里是兜底)。
	window, err := txRepo.FindByID(ctx, uint64(windowID))
	if err != nil {
		return 0, err
	}
	if window == nil {
		return 0, errors.New("版本窗口不存在")
	}
	releaseDate := window.ReleaseDate.Format("2006-01-02")
	plans, err := txRepo.GetMatchingPlans(ctx, productID, releaseDate)
	if err != nil {
		return 0, fmt.Errorf("get matching plans for product %d: %w", productID, err)
	}
	if len(plans) == 0 {
		return 0, fmt.Errorf("系统 %d 不在窗口且无匹配计划", productID)
	}
	return plans[0].ID, nil
}

func buildDemandSchedulingUpdates(req *SaveSchedulingReq, account string) map[string]interface{} {
	return map[string]interface{}{
		"RD":              strings.TrimSpace(req.RD),
		"QD":              strings.TrimSpace(req.QD),
		"accepter":        strings.TrimSpace(req.Accepter),
		"developFinish":   nullableSchedulingDate(req.DevelopFinish),
		"testFinish":      nullableSchedulingDate(req.TestFinish),
		"acceptancedDate": nullableSchedulingDate(req.AcceptancedDate),
		"lastEditedBy":    account,
		"lastEditedDate":  time.Now(),
	}
}
