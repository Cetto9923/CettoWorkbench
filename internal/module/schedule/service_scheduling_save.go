// =============================================================================
// 文件: internal/module/schedule/service_scheduling_save.go
// 模块: 排期工作台
// 类型: action
// 职责: 排期一体化「确认并同步」保存业务逻辑。
// 依赖: internal/model
//       internal/module/schedule/gateway.go
//       internal/module/schedule/repo_scheduling_write.go
// =============================================================================

package schedule

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"workbench/internal/model"
)

// SaveScheduling 保存业需排期并同步禅道研发需求与任务。
// 写操作正逐步改为禅道接口，本地不做大事务包裹（跨接口本身无法原子回滚）。
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
	if err := s.applyZenTaoDeletes(ctx, req); err != nil {
		return err
	}
	if err := s.applyZenTaoAssigns(ctx, req); err != nil {
		return err
	}
	if err := s.applyZenTaoTaskEdits(ctx, collectSchedulingTaskEdits(req)); err != nil {
		return err
	}

	window, err := s.repo.FindByID(ctx, uint64(req.WindowID))
	if err != nil {
		return err
	}
	if window == nil {
		return errors.New("版本窗口不存在")
	}
	// 与弹窗 #scheduleIntegratedSchedulePlanDate 一致：窗口 releaseDate。
	estimateLaunch := window.ReleaseDate.Format("2006-01-02")

	savedStory := false
	for _, storyReq := range req.Stories {
		action := strings.TrimSpace(storyReq.Action)
		storyID, productID, _, err := s.applySchedulingStory(ctx, s.repo, account, demandID, mainSystemID, req.WindowID, estimateLaunch, req.DevelopFinish, req.TestFinish, storyReq)
		if err != nil {
			return err
		}
		if action == "delete" {
			continue
		}
		savedStory = true
		if err := s.applySchedulingTasks(ctx, s.repo, account, storyID, productID, storyReq.Tasks); err != nil {
			return err
		}
	}

	if err := s.saveEditedDemandScheduling(ctx, s.repo, account, demandID, mainSystemID, req, estimateLaunch, window); err != nil {
		return err
	}

	// 对齐禅道 tostory：保存研发需求后，clarified → developing。
	if savedStory {
		if _, err := s.repo.PromoteDemandToDevelopingIfClarified(ctx, demandID, account); err != nil {
			return fmt.Errorf("promote demand status: %w", err)
		}
	}

	return s.repo.SaveDemandLevelWindow(ctx, demandID, uint64(req.WindowID), account)
}

// SaveStoryScheduling 保存独立研发需求（zt_story fromDemand=0）排期并同步计划/窗口/历史。
// 与 SaveScheduling（业需）的区别：主条目通过 zt_story.product 反查主系统，
// 不写 zt_demand 窗口表；窗口归属经 plan 关联反推（zt_planstory → zt_versionwindowproduct → zt_versionwindow）。
func (s *Service) SaveStoryScheduling(ctx context.Context, actor *model.User, storyID uint, req *SaveSchedulingReq) error {
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

	// 主系统取自 zt_story.product（独立研发需求无 zt_demand.mainSystem）。
	mainSystemID, err := s.repo.GetStoryProductID(ctx, storyID)
	if err != nil {
		return err
	}
	if err := s.applyZenTaoDeletes(ctx, req); err != nil {
		return err
	}
	if err := s.applyZenTaoAssigns(ctx, req); err != nil {
		return err
	}
	if err := s.applyZenTaoTaskEdits(ctx, collectSchedulingTaskEdits(req)); err != nil {
		return err
	}

	window, err := s.repo.FindByID(ctx, uint64(req.WindowID))
	if err != nil {
		return err
	}
	if window == nil {
		return errors.New("版本窗口不存在")
	}
	estimateLaunch := window.ReleaseDate.Format("2006-01-02")

	// (a) 子节点循环：独立研发需求常态为空，兼容未来子节点；demandID 传 0。
	for _, storyReq := range req.Stories {
		storyID2, productID, _, err := s.applySchedulingStory(ctx, s.repo, account, 0, mainSystemID, req.WindowID, estimateLaunch, req.DevelopFinish, req.TestFinish, storyReq)
		if err != nil {
			return err
		}
		if strings.TrimSpace(storyReq.Action) == "delete" {
			continue
		}
		if err := s.applySchedulingTasks(ctx, s.repo, account, storyID2, productID, storyReq.Tasks); err != nil {
			return err
		}
	}

	// (b) 主条目关联三连：自动勾选系统/建计划 → 从老计划移除 → 关联到新计划 → 写主条目历史。
	planID, err := s.resolvePlanForProduct(ctx, s.repo, account, req.WindowID, mainSystemID)
	if err != nil {
		return err
	}
	if err := s.repo.RemoveStoryFromOtherPlans(ctx, storyID, planID, mainSystemID, account); err != nil {
		return err
	}
	if err := s.repo.LinkStoryToPlan(ctx, storyID, mainSystemID, planID, account); err != nil {
		return err
	}
	// (c) 日期存 zt_story，并按禅道 Edited + zt_history 记录实际变更。
	return s.saveEditedStoryDates(ctx, s.repo, account, storyID, mainSystemID, estimateLaunch, req.DevelopFinish, req.TestFinish, req.AcceptancedDate)
}

func (s *Service) applySchedulingStory(
	ctx context.Context,
	txRepo *Repo,
	account string,
	demandID uint,
	mainSystemID uint,
	windowID uint,
	estimateLaunch string,
	developFinish string,
	testFinish string,
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
			EstimateLaunch:          estimateLaunch,
			DevelopFinish:           developFinish,
			TestFinish:              testFinish,
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
		if err := txRepo.CreateAction(ctx, "story", storyID, "Opened", account, productID, 0, 0, ""); err != nil {
			return 0, 0, 0, fmt.Errorf("create story action: %w", err)
		}
		// 关联到计划：排在 Opened 之后，保证 story 详情页 action 顺序 Opened → linked2plan → linked2project → linked2execution。
		if err := txRepo.LinkStoryToPlan(ctx, storyID, productID, planID, account); err != nil {
			return 0, 0, 0, fmt.Errorf("link story to plan: %w", err)
		}
		return storyID, productID, planID, nil

	case "edit":
		storyID = storyReq.ID
		productID = storyReq.ProductID
		if err := s.saveEditedStory(ctx, txRepo, account, storyID, productID, estimateLaunch, developFinish, testFinish, storyReq); err != nil {
			return 0, 0, 0, err
		}
		// 同步计划关联:解析目标计划 → 从该 story 的其他计划移除 → 幂等关联到目标计划。
		// 对 productID 未变化的情况也幂等(Ensure INSERT IGNORE + Remove 保留当前 plan)。
		newPlanID, err := s.resolvePlanForProduct(ctx, txRepo, account, windowID, storyReq.ProductID)
		if err != nil {
			return 0, 0, 0, fmt.Errorf("resolve plan for edited story %d: %w", storyID, err)
		}
		if err := txRepo.RemoveStoryFromOtherPlans(ctx, storyID, newPlanID, productID, account); err != nil {
			return 0, 0, 0, fmt.Errorf("remove story %d from other plans: %w", storyID, err)
		}
		if err := txRepo.LinkStoryToPlan(ctx, storyID, productID, newPlanID, account); err != nil {
			return 0, 0, 0, fmt.Errorf("link story to plan: %w", err)
		}
		return storyID, productID, 0, nil

	case "delete":
		// 软删与 action 已由禅道 POST /deletestories 完成（见 applyZenTaoDeletes）。
		return storyReq.ID, 0, 0, nil

	default:
		return 0, 0, 0, fmt.Errorf("unsupported story action: %s", storyReq.Action)
	}
}

// applyZenTaoDeletes 先删任务再删研发需求，交由禅道接口完成软删与历史。
func (s *Service) applyZenTaoDeletes(ctx context.Context, req *SaveSchedulingReq) error {
	storyIDs, taskIDs := collectSchedulingDeletes(req)
	if err := deleteTasks(ctx, s.ztAPI, taskIDs); err != nil {
		return err
	}
	return deleteStories(ctx, s.ztAPI, storyIDs)
}

// applyZenTaoTaskEdits 调用禅道编辑任务
func (s *Service) applyZenTaoTaskEdits(ctx context.Context, tasks []SaveSchedulingTask) error {
	for _, taskReq := range tasks {
		if err := s.saveEditedTask(ctx, taskReq); err != nil {
			return err
		}
	}
	return nil
}

// applyZenTaoAssigns 对编辑研需中指派人有变化的条目调用禅道 assign（写 Assigned 历史）。
func (s *Service) applyZenTaoAssigns(ctx context.Context, req *SaveSchedulingReq) error {
	if req == nil {
		return nil
	}
	oldByID := make(map[uint]string)
	for _, story := range req.Stories {
		if strings.TrimSpace(story.Action) != "edit" || story.ID == 0 {
			continue
		}
		if _, ok := oldByID[story.ID]; ok {
			continue
		}
		snapshot, err := s.repo.FindStoryHistorySnapshot(ctx, story.ID)
		if err != nil {
			return fmt.Errorf("load story %d for assign: %w", story.ID, err)
		}
		oldByID[story.ID] = strings.TrimSpace(snapshot.AssignedTo)
	}
	for _, item := range resolveChangedStoryAssigns(req, oldByID) {
		if err := assignStory(ctx, s.ztAPI, item.StoryID, item.AssignedTo); err != nil {
			return err
		}
	}
	return nil
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
		// 任务本体与 Opened 历史由禅道 POST /executions/{id}/tasks 完成。
		if err := createTask(ctx, s.ztAPI, storyID, taskReq); err != nil {
			return err
		}
		projectID, err := txRepo.GetProjectIDByExecution(ctx, taskReq.ExecutionID)
		if err != nil {
			return err
		}
		if err := txRepo.LinkStoryToProjectAndExecution(ctx, storyID, productID, projectID, taskReq.ExecutionID, account); err != nil {
			return fmt.Errorf("link story to project/execution: %w", err)
		}

	case "edit":
		// 禅道 PUT 已在 applyZenTaoTaskEdits 完成；此处只补本地项目/执行关联。
		projectID, err := txRepo.GetProjectIDByExecution(ctx, taskReq.ExecutionID)
		if err != nil {
			return err
		}
		if err := txRepo.LinkStoryToProjectAndExecution(ctx, storyID, productID, projectID, taskReq.ExecutionID, account); err != nil {
			return fmt.Errorf("link story to project/execution: %w", err)
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

	// vwp == nil:系统不在窗口,自动勾选到窗口并复用已有计划或新建计划。
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
	plans, err := txRepo.GetMatchingPlans(ctx, productID, endDate)
	if err != nil {
		return 0, fmt.Errorf("get matching plans for product %d: %w", productID, err)
	}
	var planID uint
	if len(plans) > 0 {
		planID = plans[0].ID
	} else {
		planID, err = txRepo.CreateProductPlan(ctx, productID, title, beginDate, endDate, account)
		if err != nil {
			return 0, fmt.Errorf("create product plan: %w", err)
		}
	}
	wp := &model.VersionWindowProduct{
		WindowID:   uint64(windowID),
		ProductID:  productID,
		PlanID:     &planID,
		PlanSynced: 1,
		CreatedBy:  account,
		UpdatedBy:  account,
	}
	if err := txRepo.CreateWindowProduct(ctx, wp); err != nil {
		return 0, fmt.Errorf("create window product for product %d: %w", productID, err)
	}
	return planID, nil
}
