// =============================================================================
// 文件: internal/module/schedule/gateway.go
// 模块: 排期工作台
// 类型: action
// 职责: 出站适配层：调用禅道检查业需转研发提醒、指派研需、编辑任务，以及批量删除研发需求/任务。
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

// storyAssignItem 待同步到禅道的研需指派。
type storyAssignItem struct {
	StoryID    uint
	AssignedTo string
}

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

// assignStory 调用禅道 POST /stories/{id}/assign，body 为 {"assignedTo":"..."}。
func assignStory(ctx context.Context, client *zentao.Client, storyID uint, assignedTo string) error {
	if storyID == 0 {
		return fmt.Errorf("研发需求 ID 无效")
	}
	if client == nil {
		return fmt.Errorf("禅道 API 未配置")
	}
	path := fmt.Sprintf("/stories/%d/assign", storyID)
	body := map[string]any{"assignedTo": strings.TrimSpace(assignedTo)}
	if err := client.Do(ctx, http.MethodPost, path, body, nil); err != nil {
		return fmt.Errorf("指派研发需求失败: %w", err)
	}
	return nil
}

// resolveChangedStoryAssigns 从排期保存请求中筛出指派人相对旧值有变化的编辑研需。
func resolveChangedStoryAssigns(req *SaveSchedulingReq, oldByID map[uint]string) []storyAssignItem {
	if req == nil {
		return nil
	}
	out := make([]storyAssignItem, 0)
	for _, story := range req.Stories {
		if strings.TrimSpace(story.Action) != "edit" || story.ID == 0 {
			continue
		}
		assignedTo := strings.TrimSpace(story.AssignedTo)
		if assignedTo == strings.TrimSpace(oldByID[story.ID]) {
			continue
		}
		out = append(out, storyAssignItem{StoryID: story.ID, AssignedTo: assignedTo})
	}
	return out
}

// buildZentaoTaskEditBody 组装禅道 PUT /tasks/:id 的编辑字段；不传 left（不覆盖剩余工时）。
func buildZentaoTaskEditBody(taskReq SaveSchedulingTask) map[string]any {
	return map[string]any{
		"name":       strings.TrimSpace(taskReq.Name),
		"type":       strings.TrimSpace(taskReq.Type),
		"pri":        normalizeTaskPriority(taskReq.Pri),
		"assignedTo": strings.TrimSpace(taskReq.AssignedTo),
		"estimate":   taskReq.Estimate,
		"estStarted": strings.TrimSpace(taskReq.EstStarted),
		"deadline":   strings.TrimSpace(taskReq.Deadline),
		"execution":  taskReq.ExecutionID,
	}
}

// updateTask 调用禅道 PUT /tasks/{id} 更新任务字段（含历史由禅道写入）。
func updateTask(ctx context.Context, client *zentao.Client, taskID uint, body map[string]any) error {
	if taskID == 0 {
		return fmt.Errorf("任务 ID 无效")
	}
	if client == nil {
		return fmt.Errorf("禅道 API 未配置")
	}
	if len(body) == 0 {
		return fmt.Errorf("更新内容为空")
	}
	path := fmt.Sprintf("/tasks/%d", taskID)
	if err := client.Do(ctx, http.MethodPut, path, body, nil); err != nil {
		return fmt.Errorf("编辑任务失败: %w", err)
	}
	return nil
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

// collectSchedulingTaskEdits 收集待同步到禅道的编辑任务
func collectSchedulingTaskEdits(req *SaveSchedulingReq) []SaveSchedulingTask {
	if req == nil {
		return nil
	}
	var out []SaveSchedulingTask
	for _, story := range req.Stories {
		if strings.TrimSpace(story.Action) == "delete" {
			continue
		}
		for _, task := range story.Tasks {
			if strings.TrimSpace(task.Action) == "edit" && task.ID > 0 {
				out = append(out, task)
			}
		}
	}
	return out
}
