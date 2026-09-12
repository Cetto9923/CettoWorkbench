// =============================================================================
// 文件: internal/module/kanban/service_task.go
// 模块: 工作看板
// 类型: readonly
// 职责: 按选中负责人组装任务看板三列。
// 依赖: internal/model
//       internal/pkg/errorx
// =============================================================================

package kanban

import (
	"context"
	"time"

	"workbench/internal/model"
)

// ListTasks 按选中负责人拉取任务三列（wait/doing=指派；done=完成者）。
func (s *Service) ListTasks(ctx context.Context, actor *model.User, req ListTasksReq) (ListTasksResp, error) {
	empty := ListTasksResp{Columns: newEmptyTaskColumns(), Summary: TaskSummary{}}
	actorAccount := ""
	if actor != nil {
		actorAccount = actor.Account
	}

	groups, err := s.ListMyTeamgroups(ctx, actor)
	if err != nil {
		return ListTasksResp{}, err
	}
	target, err := resolveDemandAccount(actorAccount, req.Account, collectMemberAccounts(groups))
	if err != nil {
		return ListTasksResp{}, err
	}

	rows, err := s.repo.FindKanbanTasks(ctx, target)
	if err != nil {
		return ListTasksResp{}, err
	}

	displayMap, err := s.loadAccountDisplayMap(ctx, actor)
	if err != nil {
		return ListTasksResp{}, err
	}

	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	cols, summary := buildTaskColumns(rows, displayMap, today)
	if cols == nil {
		return empty, nil
	}
	return ListTasksResp{Columns: cols, Summary: summary}, nil
}
