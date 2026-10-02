// =============================================================================
// 文件: internal/module/schedule/gateway.go
// 模块: 排期工作台
// 类型: action
// 职责: 出站适配层：调用禅道检查业需转研发提醒、业需转研发、指派研需、创建产品计划/任务，以及批量删除研发需求/任务。
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

// createdProductPlan 禅道创建产品计划响应的必要字段。
type createdProductPlan struct {
	ID uint `json:"id"`
}

// createProductPlan 调用禅道 POST /products/{id}/plans 创建产品计划（含 opened 历史由禅道写入）。
func createProductPlan(ctx context.Context, client *zentao.Client, productID uint, title, begin, end string) (uint, error) {
	if productID == 0 {
		return 0, fmt.Errorf("产品 ID 无效")
	}
	if client == nil {
		return 0, fmt.Errorf("禅道 API 未配置")
	}
	title = strings.TrimSpace(title)
	if title == "" {
		return 0, fmt.Errorf("计划名称不能为空")
	}
	var out createdProductPlan
	body := map[string]any{"title": title, "begin": strings.TrimSpace(begin), "end": strings.TrimSpace(end), "branch": 0}
	if err := client.Do(ctx, http.MethodPost, fmt.Sprintf("/products/%d/plans", productID), body, &out); err != nil {
		return 0, fmt.Errorf("创建产品计划失败: %w", err)
	}
	if out.ID == 0 {
		return 0, fmt.Errorf("禅道未返回计划 ID")
	}
	return out.ID, nil
}

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

// buildZentaoTaskCreateBody 组装禅道 POST /executions/:id/tasks 的创建字段。
// execution 由 path 提供；project 由禅道按执行反填。
func buildZentaoTaskCreateBody(storyID uint, taskReq SaveSchedulingTask) map[string]any {
	return map[string]any{
		"name":       strings.TrimSpace(taskReq.Name),
		"type":       strings.TrimSpace(taskReq.Type),
		"pri":        normalizeTaskPriority(taskReq.Pri),
		"assignedTo": strings.TrimSpace(taskReq.AssignedTo),
		"estimate":   taskReq.Estimate,
		"estStarted": strings.TrimSpace(taskReq.EstStarted),
		"deadline":   strings.TrimSpace(taskReq.Deadline),
		"story":      storyID,
	}
}

func zentaoTaskCreatePath(executionID uint) string {
	return fmt.Sprintf("/executions/%d/tasks", executionID)
}

// createTask 调用禅道 POST /executions/{id}/tasks 创建任务（含 Opened 历史由禅道写入）。
func createTask(ctx context.Context, client *zentao.Client, storyID uint, taskReq SaveSchedulingTask) error {
	if taskReq.ExecutionID == 0 {
		return fmt.Errorf("执行 ID 无效")
	}
	if client == nil {
		return fmt.Errorf("禅道 API 未配置")
	}
	body := buildZentaoTaskCreateBody(storyID, taskReq)
	if err := client.Do(ctx, http.MethodPost, zentaoTaskCreatePath(taskReq.ExecutionID), body, nil); err != nil {
		return fmt.Errorf("创建任务失败: %w", err)
	}
	return nil
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

func deleteSchedulingObjects(ctx context.Context, client *zentao.Client, kind string, ids []uint) error {
	if len(ids) == 0 {
		return nil
	}
	if client == nil {
		return fmt.Errorf("禅道 API 未配置")
	}
	key := map[string]string{"tasks": "taskIdList", "stories": "storyIdList"}[kind]
	if key == "" {
		return fmt.Errorf("删除对象类型无效")
	}
	if err := client.Do(ctx, http.MethodPost, "/delete"+kind, map[string]any{key: ids}, nil); err != nil {
		return fmt.Errorf("删除 %s 失败: %w", kind, err)
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

// toStoryStoryInput 业需转研发需求的单条研需入参。
type toStoryStoryInput struct {
	ProductID  uint
	PlanID     uint
	Title      string
	Spec       string
	AssignedTo string
	Estimate   float64
}

// toStoryBodyInput 组装禅道 POST /demand/:id/tostory 请求体。
type toStoryBodyInput struct {
	EstimateLaunch string
	DevelopFinish  string
	TestFinish     string
	VerifyFinish   string
	QD             string
	Stories        []toStoryStoryInput
}

// toStoryResp 禅道转研发需求成功响应。
type toStoryResp struct {
	Message  string `json:"message"`
	StoryIDs []uint `json:"storyIds"`
}

func collectNewSchedulingStories(req *SaveSchedulingReq) []SaveSchedulingStory {
	if req == nil {
		return nil
	}
	out := make([]SaveSchedulingStory, 0)
	for _, story := range req.Stories {
		if strings.TrimSpace(story.Action) == "new" {
			out = append(out, story)
		}
	}
	return out
}

func formatToStoryEstimate(v float64) string {
	return strings.TrimRight(strings.TrimRight(fmt.Sprintf("%.4f", v), "0"), ".")
}

func formatToStoryPlan(planID uint) string {
	if planID == 0 {
		return ""
	}
	return fmt.Sprintf("%d", planID)
}

// buildZentaoToStoryBody 组装禅道 POST /demand/:id/tostory 的创建字段（数组按研需条目对齐）。
func buildZentaoToStoryBody(in toStoryBodyInput) map[string]any {
	n := len(in.Stories)
	product := make([]any, n)
	module := make([]any, n)
	plan := make([]any, n)
	title := make([]any, n)
	spec := make([]any, n)
	source := make([]any, n)
	sourceNote := make([]any, n)
	verify := make([]any, n)
	assignedTo := make([]any, n)
	category := make([]any, n)
	pri := make([]any, n)
	estimate := make([]any, n)
	keywords := make([]any, n)
	color := make([]any, n)
	for i, story := range in.Stories {
		product[i] = story.ProductID
		module[i] = 0
		plan[i] = formatToStoryPlan(story.PlanID)
		title[i] = strings.TrimSpace(story.Title)
		spec[i] = story.Spec
		source[i] = ""
		sourceNote[i] = ""
		verify[i] = ""
		assignedTo[i] = strings.TrimSpace(story.AssignedTo)
		category[i] = "feature"
		pri[i] = "3"
		estimate[i] = formatToStoryEstimate(story.Estimate)
		keywords[i] = ""
		color[i] = ""
	}
	return map[string]any{
		"estimateLaunch": strings.TrimSpace(in.EstimateLaunch),
		"developFinish":  strings.TrimSpace(in.DevelopFinish),
		"testFinish":     strings.TrimSpace(in.TestFinish),
		"verifyFinish":   strings.TrimSpace(in.VerifyFinish),
		"QD":             strings.TrimSpace(in.QD),
		"product":        product,
		"module":         module,
		"plan":           plan,
		"title":          title,
		"spec":           spec,
		"source":         source,
		"sourceNote":     sourceNote,
		"verify":         verify,
		"assignedTo":     assignedTo,
		"category":       category,
		"pri":            pri,
		"estimate":       estimate,
		"keywords":       keywords,
		"color":          color,
	}
}

func zentaoDemandToStoryPath(demandID uint) string {
	return fmt.Sprintf("/demand/%d/tostory", demandID)
}

// toStory 调用禅道 POST /demand/{id}/tostory，返回新建研需 ID（与请求条目顺序一致）。
func toStory(ctx context.Context, client *zentao.Client, demandID uint, in toStoryBodyInput) ([]uint, error) {
	if demandID == 0 {
		return nil, fmt.Errorf("业需 ID 无效")
	}
	if client == nil {
		return nil, fmt.Errorf("禅道 API 未配置")
	}
	if len(in.Stories) == 0 {
		return nil, fmt.Errorf("转研发需求条目为空")
	}
	var out toStoryResp
	if err := client.Do(ctx, http.MethodPost, zentaoDemandToStoryPath(demandID), buildZentaoToStoryBody(in), &out); err != nil {
		return nil, fmt.Errorf("转研发需求失败: %w", err)
	}
	if len(out.StoryIDs) != len(in.Stories) {
		return nil, fmt.Errorf("禅道返回的研需数量与请求不一致：期望 %d，实际 %d", len(in.Stories), len(out.StoryIDs))
	}
	for i, id := range out.StoryIDs {
		if id == 0 {
			return nil, fmt.Errorf("禅道返回的第 %d 条研需 ID 无效", i+1)
		}
	}
	return out.StoryIDs, nil
}
