// =============================================================================
// 文件: internal/module/kanban/service_task.go
// 模块: 工作看板
// 类型: action
// 职责: 任务看板三列查询，及 wait↔doing 拖拽更新（禅道 PUT /tasks/:id）。
// 依赖: internal/model
//       internal/pkg/errorx
// =============================================================================

package kanban

import (
	"context"
	"errors"
	"strings"
	"time"

	"gorm.io/gorm"

	"workbench/internal/model"
	"workbench/internal/pkg/errorx"
)

const taskTimeLayout = "2006-01-02 15:04:05"

// ListTasks 按选中负责人拉取任务三列（wait/doing=指派；done=完成者）。
// account=all 时按当前敏捷小组全员聚合。
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
	targets, err := resolveKanbanAccounts(actorAccount, req.Account, req.TeamgroupID, groups)
	if err != nil {
		return ListTasksResp{}, err
	}
	if len(targets) == 0 {
		return empty, nil
	}

	rows, err := s.repo.FindKanbanTasks(ctx, targets)
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

// UpdateTaskStatus 将任务在未开始与进行中之间切换并调用禅道接口。
func (s *Service) UpdateTaskStatus(ctx context.Context, actor *model.User, req UpdateTaskStatusReq) error {
	if actor == nil || strings.TrimSpace(actor.Account) == "" {
		return errorx.New(errorx.ErrCodeForbidden, "请先登录")
	}
	if req.ID <= 0 {
		return errorx.New(errorx.ErrCodeInvalidParam, "任务 ID 无效")
	}
	target := strings.TrimSpace(req.Status)
	if target != "wait" && target != "doing" {
		return errorx.New(errorx.ErrCodeInvalidParam, "仅支持未开始与进行中互转")
	}

	row, err := s.repo.FindTaskStatusByID(ctx, req.ID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errorx.New(errorx.ErrCodeNotFound, "任务不存在或已删除")
		}
		return err
	}
	from := strings.TrimSpace(row.Status)
	if from == target {
		return nil
	}
	body := buildKanbanTaskUpdateBody(from, target, row.AssignedTo, actor.Account, time.Now())
	if body == nil {
		return errorx.New(errorx.ErrCodeInvalidParam, "仅支持未开始与进行中互转")
	}
	if err := updateZentaoTask(ctx, s.ztAPI, req.ID, body); err != nil {
		return errorx.Wrap(errorx.ErrCodeInternal, "更新任务状态失败", err)
	}
	return nil
}

// buildKanbanTaskUpdateBody 按拖拽方向组装禅道任务更新字段；非法流转返回 nil。
func buildKanbanTaskUpdateBody(fromStatus, toStatus, assignedTo, actorAccount string, now time.Time) map[string]any {
	fromStatus = strings.TrimSpace(fromStatus)
	toStatus = strings.TrimSpace(toStatus)
	assignedTo = strings.TrimSpace(assignedTo)
	actorAccount = strings.TrimSpace(actorAccount)
	nowStr := now.Format(taskTimeLayout)

	switch {
	case fromStatus == "wait" && toStatus == "doing":
		editor := assignedTo
		if editor == "" {
			editor = actorAccount
		}
		return map[string]any{
			"status":         "doing",
			"lastEditedBy":   editor,
			"lastEditedDate": nowStr,
			"realStarted":    nowStr,
		}
	case fromStatus == "doing" && toStatus == "wait":
		return map[string]any{
			"status":         "wait",
			"lastEditedBy":   actorAccount,
			"lastEditedDate": nowStr,
			"realStarted":    "",
		}
	default:
		return nil
	}
}
