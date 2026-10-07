// =============================================================================
// 文件: internal/module/schedule/service_scheduling_scope.go
// 模块: 排期工作台
// 类型: service
// 职责: 在原生调用前检查排期对象归属。
// 依赖: internal/pkg/errorx
// =============================================================================

package schedule

import (
	"context"
	"workbench/internal/model"
	"workbench/internal/pkg/errorx"
)

func (s *Service) validateSchedulingObjects(ctx context.Context, actor *model.User, demandID, storyID uint, req *SaveSchedulingReq) error {
	rows, err := s.repo.SchedulingStories(ctx, demandID, storyID)
	if err != nil {
		return err
	}
	stories := make(map[uint]uint, len(rows))
	for _, row := range rows {
		stories[row.ID] = row.Product
	}
	var taskIDs []uint
	for _, story := range req.Stories {
		if story.Action != "new" && stories[story.ID] == 0 {
			return errorx.New(errorx.ErrCodeForbidden, "研发需求不属于当前排期对象")
		}
		for _, task := range story.Tasks {
			if task.Action != "new" {
				taskIDs = append(taskIDs, task.ID)
			}
		}
	}
	owners, err := s.repo.SchedulingTaskOwners(ctx, taskIDs)
	if err != nil {
		return err
	}
	products, err := s.schedulingProductAccess(ctx, actor)
	if err != nil {
		return err
	}
	for _, story := range req.Stories {
		if err := s.validateSchedulingStory(ctx, products, owners, story, stories[story.ID]); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) validateSchedulingStory(ctx context.Context, products map[uint]bool, owners map[uint]uint, story SaveSchedulingStory, originalProduct uint) error {
	if products != nil && originalProduct > 0 && !products[originalProduct] {
		return errorx.New(errorx.ErrCodeForbidden, "无权操作原产品")
	}
	if story.Action != "delete" && products != nil && !products[story.ProductID] {
		return errorx.New(errorx.ErrCodeForbidden, "无权操作该产品")
	}
	for _, task := range story.Tasks {
		if err := s.validateSchedulingTask(ctx, story, owners, task); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) validateSchedulingTask(ctx context.Context, story SaveSchedulingStory, owners map[uint]uint, task SaveSchedulingTask) error {
	if task.Action != "new" && (owners[task.ID] == 0 || owners[task.ID] != story.ID) {
		return errorx.New(errorx.ErrCodeForbidden, "任务不属于当前研发需求")
	}
	if task.Action == "delete" {
		return nil
	}
	allowed, err := s.repo.ExecutionMatchesProduct(ctx, task.ExecutionID, story.ProductID)
	if err != nil {
		return err
	}
	if !allowed {
		return errorx.New(errorx.ErrCodeForbidden, "执行不属于该产品")
	}
	return nil
}

func (s *Service) schedulingProductAccess(ctx context.Context, actor *model.User) (map[uint]bool, error) {
	if actor == nil || actor.TrimmedAccount() == "" {
		return nil, errorx.New(errorx.ErrCodeForbidden, "未登录")
	}
	if actor.IsSuperAdmin {
		return nil, nil
	}
	if actor.ID > 0 {
		pmo, err := s.demandWriteAuthz().IsPMORole(ctx, actor.ID)
		if err != nil {
			return nil, err
		}
		if pmo {
			return nil, nil
		}
	}
	ids, err := s.getVisibleProductIDs(ctx, actor.TrimmedAccount())
	if err != nil {
		return nil, err
	}
	products := make(map[uint]bool, len(ids))
	for _, id := range ids {
		products[id] = true
	}
	return products, nil
}

func (s *Service) validateWindowProducts(ctx context.Context, actor *model.User, products []WindowProductInput) error {
	access, err := s.schedulingProductAccess(ctx, actor)
	if err != nil {
		return err
	}
	for _, product := range products {
		if access != nil && !access[product.ProductID] {
			return errorx.New(errorx.ErrCodeForbidden, "无权操作该产品")
		}
	}
	return nil
}
