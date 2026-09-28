package zentao

import (
	"context"
	"fmt"
	"net/http"
)

// UpdateTaskParams 是对任务的完整更新参数，参考 ZenTao API 文档
type UpdateTaskParams struct {
	TaskID  uint   `json:"taskId"`
	Account string `json:"account"`
	// 可选字段，依据 API 文档填写
	Module       *int     `json:"module,omitempty"`
	Story        *int     `json:"story,omitempty"`
	FromBug      *int     `json:"fromBug,omitempty"`
	Name         *string  `json:"name,omitempty"`
	Type         *string  `json:"type,omitempty"`
	AssignedTo   []string `json:"assignedTo,omitempty"`
	Pri          *int     `json:"pri,omitempty"`
	Estimate     *float64 `json:"estimate,omitempty"`
	Status       *string  `json:"status,omitempty"`
	FinishedBy   *string  `json:"finishedBy,omitempty"`
	FinishedDate *string  `json:"finishedDate,omitempty"`
}

// UpdateTask 调用 ZenTao 官方的 PUT /tasks/:id 接口更新任务信息。
func (c *Client) UpdateTask(ctx context.Context, p UpdateTaskParams) error {
	if c == nil {
		return fmt.Errorf("zentao client is nil")
	}
	if AccountFrom(ctx) == "" && p.Account != "" {
		ctx = WithAccount(ctx, p.Account)
	}
	return c.Do(ctx, http.MethodPut, fmt.Sprintf("/tasks/%d", p.TaskID), p, nil)
}
