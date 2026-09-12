// =============================================================================
// 文件: internal/module/kanban/gateway_task.go
// 模块: 工作看板
// 类型: action
// 职责: 出站适配：调用禅道 PUT /tasks/:id 更新任务字段。
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
