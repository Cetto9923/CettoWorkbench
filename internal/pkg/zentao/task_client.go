package zentao

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// TaskStatusParams 是任务状态流转参数。
type TaskStatusParams struct {
	TaskID       uint   `json:"taskId"`
	Account      string `json:"account"`
	Status       string `json:"status"`
	FinishedBy   string `json:"finishedBy,omitempty"`
	FinishedDate string `json:"finishedDate,omitempty"`
}

// UpdateTaskStatus 调用禅道自定义 REST entry 变更任务状态。
func (c *Client) UpdateTaskStatus(ctx context.Context, p TaskStatusParams) error {
	token, err := c.GetUserToken(ctx, p.Account)
	if err != nil {
		return err
	}

	reqURL := c.apiURL(fmt.Sprintf("/task/%d/status", p.TaskID))
	postData := map[string]string{
		"status": strings.TrimSpace(p.Status),
	}
	if strings.TrimSpace(p.FinishedBy) != "" {
		postData["finishedBy"] = strings.TrimSpace(p.FinishedBy)
	}
	if strings.TrimSpace(p.FinishedDate) != "" {
		postData["finishedDate"] = strings.TrimSpace(p.FinishedDate)
	}
	dataBytes, err := json.Marshal(postData)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, reqURL, bytes.NewReader(dataBytes))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Token", token)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrZentaoUnreachable, err)
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	return validateDemandReviewResponse(respBytes, resp.StatusCode)
}

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
	token, err := c.GetUserToken(ctx, p.Account)
	if err != nil {
		return err
	}
	// 构造请求 URL
	reqURL := c.apiURL(fmt.Sprintf("/tasks/%d", p.TaskID))
	// 将结构体序列化为 JSON（omitempty 会去掉空值）
	dataBytes, err := json.Marshal(p)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, reqURL, bytes.NewReader(dataBytes))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Token", token)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrZentaoUnreachable, err)
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	// 与其它接口统一使用 validateDemandReviewResponse 检查返回
	return validateDemandReviewResponse(respBytes, resp.StatusCode)
}
