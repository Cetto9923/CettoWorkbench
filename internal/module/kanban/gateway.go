// =============================================================================
// 文件: internal/module/kanban/gateway_task.go
// 模块: 工作看板
// 类型: action
// 职责: 出站适配：任务状态更新、简易创建问题等禅道 REST 调用。
// 依赖: internal/pkg/zentao
// =============================================================================

package kanban

import (
	"context"
	"fmt"
	"net/http"

	"workbench/internal/pkg/zentao"
)

func updateZentaoTask(ctx context.Context, client *zentao.Client, taskID int64, body map[string]any) error {
	if client == nil {
		return fmt.Errorf("禅道 API 未配置")
	}
	if taskID <= 0 {
		return fmt.Errorf("任务 ID 无效")
	}
	if len(body) == 0 {
		return fmt.Errorf("更新内容为空")
	}
	path := fmt.Sprintf("/tasks/%d", taskID)
	return client.Do(ctx, http.MethodPut, path, body, nil)
}

// createIssueSimpleReq 禅道 POST /issuesimplecreate。
type createIssueSimpleReq struct {
	Title      string
	Project    uint
	AssignedTo string
	Owner      string
}

// createIssueSimple 调用禅道简易创建问题接口，返回新建问题 ID。
func createIssueSimple(ctx context.Context, client *zentao.Client, req createIssueSimpleReq) (int64, error) {
	if client == nil {
		return 0, fmt.Errorf("禅道 API 未配置")
	}
	body := map[string]any{
		"title":      req.Title,
		"project":    req.Project,
		"assignedTo": req.AssignedTo,
		"owner":      req.Owner,
	}
	var out struct {
		ID int64 `json:"id"`
	}
	if err := client.Do(ctx, http.MethodPost, "/issuesimplecreate", body, &out); err != nil {
		return 0, err
	}
	if out.ID <= 0 {
		return 0, fmt.Errorf("禅道未返回问题 ID")
	}
	return out.ID, nil
}
