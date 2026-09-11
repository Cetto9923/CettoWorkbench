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
