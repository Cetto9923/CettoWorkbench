// =============================================================================
// 文件: internal/module/schedule/service_scheduling_save.go
// 模块: 排期工作台
// 类型: action
// 职责: 排期一体化「确认并同步」保存业务逻辑。
// 依赖: internal/model
//       internal/module/schedule/gateway.go
//       internal/module/schedule/repo_scheduling_write.go
//       internal/module/schedule/service_scheduling_tostory.go
// =============================================================================

package schedule

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"workbench/internal/model"
)

func (s *Service) SaveScheduling(ctx context.Context, actor *model.User, demandID uint, req *SaveSchedulingReq) error {
	return s.saveScheduling(ctx, actor, demandID, req, false)
}

func (s *Service) SaveStoryScheduling(ctx context.Context, actor *model.User, storyID uint, req *SaveSchedulingReq) error {
	return s.saveScheduling(ctx, actor, storyID, req, true)
}

func (s *Service) saveScheduling(ctx context.Context, actor *model.User, id uint, req *SaveSchedulingReq, independent bool) error {
	if id == 0 || req == nil {
		return errors.New("排期对象或请求参数无效")
	}
	if actor.TrimmedAccount() == "" {
		return errors.New("未登录或无法识别当前用户")
	}
	access := s.RequireDemandWriteAccess
	if independent {
		access = s.RequireStoryWriteAccess
	}
	if err := access(ctx, actor, id); err != nil {
		return err
	}
	demandID, storyID := id, uint(0)
	if independent {
		demandID, storyID = 0, id
	}
	if demandID > 0 && len(collectNewSchedulingStories(req)) > 0 && strings.TrimSpace(req.QD) == "" {
		return errors.New("测试负责人不能为空")
	}
	if err := s.validateSchedulingObjects(ctx, actor, demandID, storyID, req); err != nil {
		return err
	}
	window, err := s.repo.FindByID(ctx, uint64(req.WindowID))
	if err != nil {
		return err
	}
	if window == nil {
		return errors.New("版本窗口不存在")
	}
	mainSystem := s.repo.GetDemandMainSystem
	if independent {
		mainSystem = s.repo.GetStoryProductID
	}
	productID, err := mainSystem(ctx, id)
	if err != nil {
		return err
	}
	input := schedulingSaveReq{Account: actor.TrimmedAccount(), DemandID: demandID, StoryID: storyID, ProductID: productID, Window: window, Form: req}
	if err := s.syncScheduling(ctx, input); err != nil {
		return fmt.Errorf("排期同步失败，已有步骤可能提交，请刷新核对：%w", err)
	}
	return s.saveSchedulingMetadata(ctx, input)
}

type schedulingSaveReq struct {
	Account                      string
	DemandID, StoryID, ProductID uint
	Window                       *model.VersionWindow
	Form                         *SaveSchedulingReq
}

func (s *Service) syncScheduling(ctx context.Context, req schedulingSaveReq) error {
	if err := s.applyZenTaoDeletes(ctx, req.Form); err != nil {
		return err
	}
	if err := s.applyZenTaoAssigns(ctx, req.Form); err != nil {
		return err
	}
	launch := req.Window.ReleaseDate.Format("2006-01-02")
	if req.DemandID > 0 {
		form := req.Form
		if _, err := s.applyNewStoriesViaToStory(ctx, req.Account, req.DemandID, form.WindowID, launch, form.DevelopFinish, form.TestFinish, form.AcceptancedDate, form.QD, collectNewSchedulingStories(form)); err != nil {
			return err
		}
	}
	return s.saveSchedulingStories(ctx, req.Account, req.DemandID, req.ProductID, launch, req.Form)
}

func (s *Service) saveSchedulingMetadata(ctx context.Context, req schedulingSaveReq) error {
	launch := req.Window.ReleaseDate.Format("2006-01-02")
	if req.StoryID > 0 {
		return s.saveIndependentScheduling(ctx, req.Account, req.StoryID, req.ProductID, launch, req.Form)
	}
	err := s.repo.Transaction(ctx, func(tx *Repo) error {
		if err := tx.LockWindow(ctx, req.Window.ID); err != nil {
			return err
		}
		if err := s.saveEditedDemandScheduling(ctx, tx, req.Account, req.DemandID, req.ProductID, req.Form, launch, req.Window); err != nil {
			return err
		}
		return tx.SaveDemandLevelWindow(ctx, req.DemandID, req.Window.ID, req.Account)
	})
	if err != nil {
		return fmt.Errorf("排期同步后窗口保存失败，请刷新核对：%w", err)
	}
	return nil
}

func (s *Service) saveSchedulingStories(ctx context.Context, account string, demandID, mainSystemID uint, launch string, req *SaveSchedulingReq) error {
	for _, story := range req.Stories {
		if demandID > 0 && story.Action == "new" {
			continue
		}
		id, product, _, err := s.applySchedulingStory(ctx, s.repo, account, demandID, mainSystemID, req.WindowID, launch, req.DevelopFinish, req.TestFinish, story)
		if err != nil {
			return err
		}
		if story.Action == "delete" {
			continue
		}
		if err := s.applySchedulingTasks(ctx, s.repo, account, id, product, story.Tasks); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) saveIndependentScheduling(ctx context.Context, account string, storyID, productID uint, launch string, req *SaveSchedulingReq) error {
	planID, err := s.resolvePlanForProduct(ctx, s.repo, account, req.WindowID, productID)
	if err != nil {
		return err
	}
	if err := s.removeStoryFromOtherPlans(ctx, storyID, planID); err != nil {
		return err
	}
	return s.repo.Transaction(ctx, func(tx *Repo) error {
		if err := tx.LockWindow(ctx, uint64(req.WindowID)); err != nil {
			return err
		}
		if err := tx.LinkStoryToPlan(ctx, storyID, productID, planID, account); err != nil {
			return err
		}
		return s.saveEditedStoryDates(ctx, tx, account, storyID, productID, launch, req.DevelopFinish, req.TestFinish, req.AcceptancedDate)
	})
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
		err = txRepo.Transaction(ctx, func(tx *Repo) error {
			if err := tx.LockWindow(ctx, uint64(windowID)); err != nil {
				return err
			}
			isMain := "0"
			if productID > 0 && productID == mainSystemID {
				isMain = "1"
			}
			storyID, err = tx.CreateStory(ctx, &ZtStoryInsert{
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
				return fmt.Errorf("create story: %w", err)
			}
			if err := tx.CreateStorySpec(ctx, &ZtStorySpec{
				Story:   storyID,
				Version: 1,
				Title:   storyReq.Title,
				Spec:    storyReq.Spec,
			}); err != nil {
				return fmt.Errorf("create story spec: %w", err)
			}
			if err := tx.CreateAction(ctx, "story", storyID, "Opened", account, productID, 0, 0, ""); err != nil {
				return fmt.Errorf("create story action: %w", err)
			}
			// 关联到计划：排在 Opened 之后，保证 story 详情页 action 顺序 Opened → linked2plan → linked2project → linked2execution。
			if err := tx.LinkStoryToPlan(ctx, storyID, productID, planID, account); err != nil {
				return fmt.Errorf("link story to plan: %w", err)
			}
			return nil
		})
		return storyID, productID, planID, err

	case "edit":
		planID, err := s.resolvePlanForProduct(ctx, txRepo, account, windowID, storyReq.ProductID)
		if err != nil {
			return 0, 0, 0, err
		}
		if err := s.removeStoryFromOtherPlans(ctx, storyReq.ID, planID); err != nil {
			return 0, 0, 0, err
		}
		err = txRepo.Transaction(ctx, func(tx *Repo) error {
			if err := tx.LockWindow(ctx, uint64(windowID)); err != nil {
				return err
			}
			if err := s.saveEditedStory(ctx, tx, account, storyReq.ID, storyReq.ProductID, estimateLaunch, developFinish, testFinish, storyReq); err != nil {
				return err
			}
			return tx.LinkStoryToPlan(ctx, storyReq.ID, storyReq.ProductID, planID, account)
		})
		return storyReq.ID, storyReq.ProductID, planID, err

	case "delete":
		// 软删与 action 已由禅道 POST /deletestories 完成（见 applyZenTaoDeletes）。
		return storyReq.ID, 0, 0, nil

	default:
		return 0, 0, 0, fmt.Errorf("unsupported story action: %s", storyReq.Action)
	}
}

// applyZenTaoDeletes 交由禅道接口删除研发需求并记录历史。
func (s *Service) applyZenTaoDeletes(ctx context.Context, req *SaveSchedulingReq) error {
	storyIDs, _ := collectSchedulingDeletes(req)
	return deleteSchedulingObjects(ctx, s.ztAPI, "stories", storyIDs)
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
		if err := s.saveEditedTask(ctx, taskReq); err != nil {
			return err
		}
		projectID, err := txRepo.GetProjectIDByExecution(ctx, taskReq.ExecutionID)
		if err != nil {
			return err
		}
		if err := txRepo.LinkStoryToProjectAndExecution(ctx, storyID, productID, projectID, taskReq.ExecutionID, account); err != nil {
			return fmt.Errorf("link story to project/execution: %w", err)
		}
	case "delete":
		return deleteSchedulingObjects(ctx, s.ztAPI, "tasks", []uint{taskReq.ID})
	}
	return nil
}

func (s *Service) resolvePlanForProduct(ctx context.Context, repo *Repo, account string, windowID, productID uint) (uint, error) {
	link, err := repo.FindWindowProductPlan(ctx, windowID, productID)
	if err != nil {
		return 0, err
	}
	if link != nil && link.PlanID != nil && *link.PlanID > 0 {
		return *link.PlanID, nil
	}
	window, err := repo.FindByID(ctx, uint64(windowID))
	if err != nil {
		return 0, err
	}
	if window == nil {
		return 0, errors.New("版本窗口不存在")
	}
	end := window.ReleaseDate.Format("2006-01-02")
	begin := end
	if window.StartDate != nil {
		begin = window.StartDate.Format("2006-01-02")
	}
	title := strings.TrimSpace(window.Name)
	if title == "" {
		title = end
	}
	plans, err := repo.GetMatchingPlans(ctx, productID, end)
	if err != nil {
		return 0, err
	}
	var planID uint
	if len(plans) > 0 {
		planID = plans[0].ID
	} else {
		planID, err = createProductPlan(ctx, s.ztAPI, productID, title, begin, end)
		if err != nil {
			return 0, err
		}
	}
	err = repo.Transaction(ctx, func(tx *Repo) error {
		if err := tx.LockWindow(ctx, uint64(windowID)); err != nil {
			return err
		}
		if link != nil {
			return tx.UpdateWindowProductPlanID(ctx, link.ID, planID, account)
		}
		return tx.CreateWindowProduct(ctx, &model.VersionWindowProduct{WindowID: uint64(windowID), ProductID: productID, PlanID: &planID, PlanSynced: 1, CreatedBy: account, UpdatedBy: account})
	})
	return planID, err
}
