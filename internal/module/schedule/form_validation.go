// =============================================================================
// File: internal/module/schedule/form_validation.go
// Module: schedule workbench
// Purpose: Validation methods for schedule module request structs.
// =============================================================================

package schedule

import (
	"strconv"
	"strings"
)

// Validate 校验分页与基础参数。
func (r *ListBizDemandsReq) Validate() []FieldError {
	var errs []FieldError
	if r.Page < 1 {
		errs = append(errs, FieldError{Field: "page", Message: "页码必须大于等于 1"})
	}
	if r.PageSize < 1 || r.PageSize > 100 {
		errs = append(errs, FieldError{Field: "pageSize", Message: "每页条数须在 1 到 100 之间"})
	}
	return errs
}

// Validate 校验分页参数。
func (r *ListIndependentReq) Validate() []FieldError {
	var errs []FieldError
	if r.Page < 1 {
		errs = append(errs, FieldError{Field: "page", Message: "页码必须大于等于 1"})
	}
	if r.PageSize < 1 || r.PageSize > 100 {
		errs = append(errs, FieldError{Field: "pageSize", Message: "每页条数须在 1 到 100 之间"})
	}
	return errs
}

// Validate 校验排期保存请求。
func (r *SaveSchedulingReq) Validate() []FieldError {
	var errs []FieldError
	if r.WindowID == 0 {
		errs = append(errs, FieldError{Field: "windowId", Message: "版本窗口不能为空"})
	}
	// DB-R1 批量上限：超过即 422 拒绝，不开事务。
	if len(r.Stories) > MaxSchedulingStories {
		errs = append(errs, FieldError{Field: "stories", Message: "单次排期最多支持 " + strconv.Itoa(MaxSchedulingStories) + " 条研发需求"})
	}
	for i, story := range r.Stories {
		if len(story.Tasks) > MaxSchedulingTasksPerStory {
			prefix := "stories[" + strconv.Itoa(i) + "].tasks"
			errs = append(errs, FieldError{Field: prefix, Message: "单研发需求任务最多 " + strconv.Itoa(MaxSchedulingTasksPerStory) + " 条"})
		}
	}
	for i, story := range r.Stories {
		prefix := "stories[" + strconv.Itoa(i) + "]"
		action := strings.TrimSpace(story.Action)
		switch action {
		case "new":
			if story.ProductID == 0 {
				errs = append(errs, FieldError{Field: prefix + ".productId", Message: "系统不能为空"})
			}
			if strings.TrimSpace(story.Title) == "" {
				errs = append(errs, FieldError{Field: prefix + ".title", Message: "研发需求标题不能为空"})
			}
			if strings.TrimSpace(story.Spec) == "" {
				errs = append(errs, FieldError{Field: prefix + ".spec", Message: "研发需求描述不能为空"})
			}
			if strings.TrimSpace(story.AssignedTo) == "" {
				errs = append(errs, FieldError{Field: prefix + ".assignedTo", Message: "研发需求指派人不能为空"})
			}
		case "edit", "delete":
			if story.ID == 0 {
				errs = append(errs, FieldError{Field: prefix + ".id", Message: "研发需求 ID 无效"})
			}
			if action == "edit" {
				if story.ProductID == 0 {
					errs = append(errs, FieldError{Field: prefix + ".productId", Message: "系统不能为空"})
				}
				if strings.TrimSpace(story.Title) == "" {
					errs = append(errs, FieldError{Field: prefix + ".title", Message: "研发需求标题不能为空"})
				}
				if strings.TrimSpace(story.Spec) == "" {
					errs = append(errs, FieldError{Field: prefix + ".spec", Message: "研发需求描述不能为空"})
				}
				if strings.TrimSpace(story.AssignedTo) == "" {
					errs = append(errs, FieldError{Field: prefix + ".assignedTo", Message: "研发需求指派人不能为空"})
				}
			}
		case "":
			errs = append(errs, FieldError{Field: prefix + ".action", Message: "操作类型不能为空"})
		default:
			errs = append(errs, FieldError{Field: prefix + ".action", Message: "不支持的操作类型"})
		}
		for j, task := range story.Tasks {
			taskPrefix := prefix + ".tasks[" + strconv.Itoa(j) + "]"
			taskAction := strings.TrimSpace(task.Action)
			switch taskAction {
			case "new":
				if task.ExecutionID == 0 {
					errs = append(errs, FieldError{Field: taskPrefix + ".executionId", Message: "执行不能为空"})
				}
				if strings.TrimSpace(task.Name) == "" {
					errs = append(errs, FieldError{Field: taskPrefix + ".name", Message: "任务名称不能为空"})
				}
				if task.Estimate <= 0 {
					errs = append(errs, FieldError{Field: taskPrefix + ".estimate", Message: "预估不能为空"})
				}
				if strings.TrimSpace(task.Deadline) == "" {
					errs = append(errs, FieldError{Field: taskPrefix + ".deadline", Message: "截止时间不能为空"})
				}
			case "edit", "delete":
				if task.ID == 0 {
					errs = append(errs, FieldError{Field: taskPrefix + ".id", Message: "任务 ID 无效"})
				}
				if taskAction == "edit" {
					if strings.TrimSpace(task.Name) == "" {
						errs = append(errs, FieldError{Field: taskPrefix + ".name", Message: "任务名称不能为空"})
					}
					if task.Estimate <= 0 {
						errs = append(errs, FieldError{Field: taskPrefix + ".estimate", Message: "预估不能为空"})
					}
					if strings.TrimSpace(task.Deadline) == "" {
						errs = append(errs, FieldError{Field: taskPrefix + ".deadline", Message: "截止时间不能为空"})
					}
				}
			case "":
				errs = append(errs, FieldError{Field: taskPrefix + ".action", Message: "任务操作类型不能为空"})
			default:
				errs = append(errs, FieldError{Field: taskPrefix + ".action", Message: "不支持的任务操作类型"})
			}
		}
	}
	return errs
}

// Validate 校验维护任务保存请求。
func (r *SaveStoryTasksReq) Validate() []FieldError {
	var errs []FieldError
	// DB-R1 批量上限：超过则 422 拒绝，不开事务。
	if len(r.Tasks) > MaxStoryTasks {
		errs = append(errs, FieldError{Field: "tasks", Message: "单次保存最多 " + strconv.Itoa(MaxStoryTasks) + " 条任务"})
	}
	for i, task := range r.Tasks {
		prefix := "tasks[" + strconv.Itoa(i) + "]"
		action := strings.TrimSpace(task.Action)
		switch action {
		case "new":
			if task.Create {
				if strings.TrimSpace(task.Name) == "" {
					errs = append(errs, FieldError{Field: prefix + ".name", Message: "任务名称不能为空"})
				}
				if task.ExecutionID == 0 {
					errs = append(errs, FieldError{Field: prefix + ".executionId", Message: "执行不能为空"})
				}
				if task.Estimate <= 0 {
					errs = append(errs, FieldError{Field: prefix + ".estimate", Message: "预估不能为空"})
				}
				if strings.TrimSpace(task.Deadline) == "" {
					errs = append(errs, FieldError{Field: prefix + ".deadline", Message: "截止时间不能为空"})
				}
			}
		case "edit", "delete":
			if task.ID == 0 {
				errs = append(errs, FieldError{Field: prefix + ".id", Message: "任务 ID 无效"})
			}
			if action == "edit" && task.ExecutionID == 0 {
				errs = append(errs, FieldError{Field: prefix + ".executionId", Message: "执行不能为空"})
			}
			if action == "edit" {
				if strings.TrimSpace(task.Name) == "" {
					errs = append(errs, FieldError{Field: prefix + ".name", Message: "任务名称不能为空"})
				}
				if task.Estimate <= 0 {
					errs = append(errs, FieldError{Field: prefix + ".estimate", Message: "预估不能为空"})
				}
				if strings.TrimSpace(task.Deadline) == "" {
					errs = append(errs, FieldError{Field: prefix + ".deadline", Message: "截止时间不能为空"})
				}
			}
		case "":
			errs = append(errs, FieldError{Field: prefix + ".action", Message: "操作类型不能为空"})
		default:
			errs = append(errs, FieldError{Field: prefix + ".action", Message: "不支持的操作类型"})
		}
	}
	return errs
}
