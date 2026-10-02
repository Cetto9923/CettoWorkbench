// =============================================================================
// 文件: internal/module/schedule/service_scheduling_tostory.go
// 模块: 排期工作台
// 类型: action
// 职责: 业需排期保存时通过禅道 tostory 批量创建研发需求，并挂计划、拆任务。
// 依赖: internal/module/schedule/gateway.go
//       internal/module/schedule/repo_scheduling_write.go
// =============================================================================

package schedule

import (
	"context"
	"fmt"
	"strings"
)

// applyNewStoriesViaToStory 调用禅道转研发需求，返回新建研需 ID（与 newStories 顺序一致）。
// 成功后为每条研需补 linked2plan，并继续拆任务。
func (s *Service) applyNewStoriesViaToStory(
	ctx context.Context,
	account string,
	demandID uint,
	windowID uint,
	estimateLaunch string,
	developFinish string,
	testFinish string,
	verifyFinish string,
	qd string,
	newStories []SaveSchedulingStory,
) ([]uint, error) {
	if demandID == 0 {
		return nil, fmt.Errorf("业需 ID 无效")
	}
	if len(newStories) == 0 {
		return nil, nil
	}
	if strings.TrimSpace(qd) == "" {
		return nil, fmt.Errorf("测试负责人不能为空")
	}

	inputs := make([]toStoryStoryInput, 0, len(newStories))
	for _, storyReq := range newStories {
		planID, err := s.resolvePlanForProduct(ctx, s.repo, account, windowID, storyReq.ProductID)
		if err != nil {
			return nil, err
		}
		inputs = append(inputs, toStoryStoryInput{
			ProductID:  storyReq.ProductID,
			PlanID:     planID,
			Title:      storyReq.Title,
			Spec:       storyReq.Spec,
			AssignedTo: storyReq.AssignedTo,
			Estimate:   storyReq.Estimate,
		})
	}

	storyIDs, err := toStory(ctx, s.ztAPI, demandID, toStoryBodyInput{
		EstimateLaunch: estimateLaunch,
		DevelopFinish:  developFinish,
		TestFinish:     testFinish,
		VerifyFinish:   verifyFinish,
		QD:             qd,
		Stories:        inputs,
	})
	if err != nil {
		return nil, err
	}

	for i, storyReq := range newStories {
		storyID := storyIDs[i]
		if err := s.repo.LinkStoryToPlan(ctx, storyID, storyReq.ProductID, inputs[i].PlanID, account); err != nil {
			return nil, err
		}
		if err := s.applySchedulingTasks(ctx, s.repo, account, storyID, storyReq.ProductID, storyReq.Tasks); err != nil {
			return nil, err
		}
	}
	return storyIDs, nil
}
