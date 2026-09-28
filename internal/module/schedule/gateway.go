// =============================================================================
// 文件: internal/module/schedule/gateway.go
// 模块: 排期工作台
// 类型: action
// 职责: 出站适配层：调用禅道检查业需转研发提醒，以及批量删除研发需求/任务。
// 依赖: internal/pkg/zentao
// =============================================================================

package schedule

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"workbench/internal/pkg/zentao"
)

const reviewToStoryNoticeFallback = "请完成主管部门审批后，再进行转研发需求操作。"

// checkReviewToStoryNotice 调用禅道 GET /demand/{id}/checkReviewToStoryNotice。
// 需拦截时返回提示文案；允许转研发时返回空串。
func checkReviewToStoryNotice(ctx context.Context, client *zentao.Client, demandID uint) (string, error) {
	if client == nil {
		return "", fmt.Errorf("禅道 API 未配置")
	}
	if demandID == 0 {
		return "", fmt.Errorf("业需 ID 无效")
	}

	var out struct {
		Result  string `json:"result"`
		Message string `json:"message"`
	}
	path := fmt.Sprintf("/demand/%d/checkReviewToStoryNotice", demandID)
	err := client.Do(ctx, http.MethodGet, path, nil, &out)
	if err != nil {
		if zentao.StatusCode(err) == http.StatusBadRequest {
			return reviewNoticeMessage(err.Error()), nil
		}
		return "", err
	}
	if strings.EqualFold(strings.TrimSpace(out.Result), "fail") {
		return reviewNoticeMessage(out.Message), nil
	}
	return "", nil
}

func reviewNoticeMessage(message string) string {
	message = strings.TrimSpace(message)
	if message == "" {
		return reviewToStoryNoticeFallback
	}
	return message
}

// deleteTasks 调用禅道 POST /deletetasks，body 为 {"taskIdList":[...]}。
func deleteTasks(ctx context.Context, client *zentao.Client, taskIDs []uint) error {
	if len(taskIDs) == 0 {
		return nil
	}
	if client == nil {
		return fmt.Errorf("禅道 API 未配置")
	}
	body := map[string]any{"taskIdList": taskIDs}
	if err := client.Do(ctx, http.MethodPost, "/deletetasks", body, nil); err != nil {
		return fmt.Errorf("删除任务失败: %w", err)
	}
	return nil
}

// deleteStories 调用禅道 POST /deletestories，body 为 {"storyIdList":[...]}。
func deleteStories(ctx context.Context, client *zentao.Client, storyIDs []uint) error {
	if len(storyIDs) == 0 {
		return nil
	}
	if client == nil {
		return fmt.Errorf("禅道 API 未配置")
	}
	body := map[string]any{"storyIdList": storyIDs}
	if err := client.Do(ctx, http.MethodPost, "/deletestories", body, nil); err != nil {
		return fmt.Errorf("删除研发需求失败: %w", err)
	}
	return nil
}

// collectSchedulingDeletes 从排期保存请求中收集待删研发需求/任务 ID。
func collectSchedulingDeletes(req *SaveSchedulingReq) (storyIDs []uint, taskIDs []uint) {
	if req == nil {
		return nil, nil
	}
	for _, story := range req.Stories {
		if strings.TrimSpace(story.Action) == "delete" {
			if story.ID > 0 {
				storyIDs = append(storyIDs, story.ID)
			}
			continue
		}
		for _, task := range story.Tasks {
			if strings.TrimSpace(task.Action) == "delete" && task.ID > 0 {
				taskIDs = append(taskIDs, task.ID)
			}
		}
	}
	return storyIDs, taskIDs
}
