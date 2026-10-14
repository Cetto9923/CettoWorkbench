// =============================================================================
// 文件: internal/module/schedule/service_execution.go
// 模块: 排期工作台
// 类型: action
// 职责: 快速创建迭代业务编排（查团队成员并代理禅道创建执行，再关联当前研发需求）。
// 依赖: internal/model
//       internal/module/schedule/gateway.go
//       internal/pkg/zentao
// =============================================================================

package schedule

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"workbench/internal/model"
	"workbench/internal/pkg/zentao"
)

// CreateExecution 以当前用户身份创建项目下的执行（迭代）；
// 仅当 StoryID > 0 时关联研发需求（未保存研需跳过关联）。
func (s *Service) CreateExecution(ctx context.Context, actor *model.User, req CreateExecutionReq) (CreateExecutionResp, error) {
	var empty CreateExecutionResp
	if actorAccount(actor) == "" {
		return empty, errors.New("未登录或无法识别当前用户")
	}
	client := s.ztAPI
	if client == nil {
		client = zentao.API()
	}
	if client == nil {
		return empty, errors.New("禅道 API 未配置")
	}

	var teamMembers []string
	if req.ScrumTeam > 0 {
		accounts, err := s.repo.ListTeamgroupMemberAccounts(ctx, req.ScrumTeam)
		if err != nil {
			return empty, err
		}
		teamMembers = accounts
	}

	resp, err := createExecution(ctx, client, req, teamMembers)
	if err != nil {
		return empty, err
	}
	if req.StoryID > 0 {
		if err := linkExecutionStories(ctx, client, resp.ID, []uint{req.StoryID}); err != nil {
			return empty, err
		}
	}
	return resp, nil
}

// linkNewStoryToTaskExecutions 将新建研需关联到任务所选执行。
// 用于补齐「未保存研需时创建迭代」跳过的执行-需求关联，须在拆任务前调用。
func (s *Service) linkNewStoryToTaskExecutions(ctx context.Context, storyID uint, tasks []SaveSchedulingTask) error {
	if storyID == 0 || len(tasks) == 0 {
		return nil
	}
	client := s.ztAPI
	if client == nil {
		client = zentao.API()
	}
	if client == nil {
		return errors.New("禅道 API 未配置")
	}

	seen := make(map[uint]struct{})
	for _, task := range tasks {
		if strings.TrimSpace(task.Action) == "delete" {
			continue
		}
		executionID := task.ExecutionID
		if executionID == 0 {
			continue
		}
		if _, ok := seen[executionID]; ok {
			continue
		}
		seen[executionID] = struct{}{}
		if err := linkExecutionStories(ctx, client, executionID, []uint{storyID}); err != nil {
			return fmt.Errorf("关联研发需求 %d 到执行 %d: %w", storyID, executionID, err)
		}
	}
	return nil
}
