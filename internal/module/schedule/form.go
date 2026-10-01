// =============================================================================
// 文件: internal/module/schedule/form.go
// 模块: 排期工作台
// 类型: action
// 职责: 定义列表与保存请求及保留的历史校验。
// 保留: 历史复杂度、嵌套及重复校验沿用质量门基线。
// 依赖: 无
// =============================================================================

package schedule

import (
	"strconv"
	"strings"
	"time"
)

// Validate 校验新建版本窗口请求。
func (r *CreateReq) Validate() []FieldError {
	errs := validateWindowSaveFields(
		r.ReleaseDate,
		r.Name,
		r.StartDate,
		r.TeamgroupID,
		r.GroupSize,
		r.Products,
	)
	errs = append(errs, validateWindowMilestoneFields(r.WindowType, r.PlanTestDone, r.TestDone, r.AcceptDone)...)
	return errs
}

// Validate 校验更新版本窗口请求。
func (r *UpdateReq) Validate() []FieldError {
	var errs []FieldError
	if r.ID == 0 {
		errs = append(errs, FieldError{Field: "id", Message: "窗口 ID 无效"})
	}
	errs = append(errs, validateWindowSaveFields(
		r.ReleaseDate,
		r.Name,
		r.StartDate,
		r.TeamgroupID,
		r.GroupSize,
		r.Products,
	)...)
	errs = append(errs, validateWindowMilestoneFields(r.WindowType, r.PlanTestDone, r.TestDone, r.AcceptDone)...)
	return errs
}

// Validate 校验删除版本窗口请求。
func (r *DeleteReq) Validate() []FieldError {
	if r.ID == 0 {
		return []FieldError{{Field: "id", Message: "窗口 ID 无效"}}
	}
	return nil
}

func validateWindowMilestoneFields(windowType, planTestDone, testDone, acceptDone string) []FieldError {
	var errs []FieldError
	if wt := strings.TrimSpace(windowType); wt != "" && wt != "regular" && wt != "fast" && wt != "urgent" {
		errs = append(errs, FieldError{Field: "windowType", Message: "发布类型无效"})
	}
	fields := []struct {
		field string
		value string
		label string
	}{
		{field: "planTestDone", value: planTestDone, label: "预计提测/开发完成日期"},
		{field: "testDone", value: testDone, label: "预计测试完成日期"},
		{field: "acceptDone", value: acceptDone, label: "预计验收完成日期"},
	}
	for _, f := range fields {
		value := strings.TrimSpace(f.value)
		if value == "" {
			errs = append(errs, FieldError{Field: f.field, Message: f.label + "不能为空"})
			continue
		}
		if _, err := time.Parse("2006-01-02", value); err != nil {
			errs = append(errs, FieldError{Field: f.field, Message: f.label + "格式无效"})
		}
	}
	return errs
}

// ListBizDemandsReq 业务需求 Tab 列表查询入参。
type ListBizDemandsReq struct {
	Page        int    `form:"page"`
	PageSize    int    `form:"pageSize"`
	TeamgroupID uint   `form:"teamgroupId"`
	ProductID   uint   `form:"productId"`
	Stage       string `form:"stage"`
	Status      string `form:"status"`
	Keyword     string `form:"keyword"`
	WindowID    uint   `form:"windowId"`
	Scope       string `form:"scope"`
	Filter      string `form:"filter"`    // all_open, unscheduled(待排期=与我相关+clarified/developing+已澄清系统+待排期态), pending_review(待受理=与我相关+draft/wait/refuse), manager_reviewing(主管审批中=与我相关+isManagerReview=reviewing), closed
	Suspended   bool   `form:"suspended"` // true 时叠加 AND hang = '1'
	Groups      string `form:"groups"`    // 逗号分隔的小组 ID
	Products    string `form:"products"`  // 逗号分隔的产品 ID
	Stages      string `form:"stages"`    // 逗号分隔的阶段值
	Windows     string `form:"windows"`   // 逗号分隔的版本窗口 ID
	Pri         string `form:"pri"`       // 单值优先级：0-4
	TestOwner   string `form:"test"`      // 测试负责人账号
	AcceptOwner string `form:"accept"`    // 验收负责人账号
}

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

// ListIndependentReq 独立研发需求 Tab 列表查询入参。
type ListIndependentReq struct {
	Page      int    `form:"page"`
	PageSize  int    `form:"pageSize"`
	Filter    string `form:"filter"`
	Suspended bool   `form:"suspended"` // story 无 hang 字段，查询时忽略
	Groups    string `form:"groups"`    // 逗号分隔的小组 ID
	Products  string `form:"products"`  // 逗号分隔的产品 ID
	Stages    string `form:"stages"`    // 逗号分隔的阶段值
	Stage     string `form:"stage"`     // 兼容单值 stage
	Windows   string `form:"windows"`   // 逗号分隔的版本窗口 ID
	Keyword   string `form:"keyword"`   // 编号/标题/负责人/系统
	Pri       string `form:"pri"`       // 单值优先级：0-4
	TestOwner string `form:"test"`      // 独立研发需求当前按测试任务 assignedTo 过滤
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

// SaveSchedulingReq 排期一体化「确认并同步」保存请求。
type SaveSchedulingReq struct {
	WindowID        uint                  `json:"windowId"`
	QD              string                `json:"qd"`
	Accepter        string                `json:"accepter"`
	DevelopFinish   string                `json:"developFinish"`
	TestFinish      string                `json:"testFinish"`
	AcceptancedDate string                `json:"acceptancedDate"`
	Stories         []SaveSchedulingStory `json:"stories"`
}

// Validate 校验排期保存请求。
func (r *SaveSchedulingReq) Validate() []FieldError {
	var errs []FieldError
	if r.WindowID == 0 {
		errs = append(errs, FieldError{Field: "windowId", Message: "版本窗口不能为空"})
	}
	dateFields := []struct {
		field string
		value string
		label string
	}{
		{field: "developFinish", value: r.DevelopFinish, label: "提测/开发日期"},
		{field: "testFinish", value: r.TestFinish, label: "测试完成日期"},
		{field: "acceptancedDate", value: r.AcceptancedDate, label: "验收完成日期"},
	}
	for _, f := range dateFields {
		value := strings.TrimSpace(f.value)
		if value == "" {
			errs = append(errs, FieldError{Field: f.field, Message: f.label + "不能为空"})
			continue
		}
		if _, err := time.Parse("2006-01-02", value); err != nil {
			errs = append(errs, FieldError{Field: f.field, Message: f.label + "格式无效"})
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
				errs = append(errs, FieldError{Field: prefix + ".assignedTo", Message: "指派给不能为空"})
			}
		case "edit", "delete":
			if story.ID == 0 {
				errs = append(errs, FieldError{Field: prefix + ".id", Message: "研发需求 ID 无效"})
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
				if action != "delete" {
					errs = append(errs, FieldError{Field: taskPrefix + ".action", Message: "任务操作类型不能为空"})
				}
			default:
				errs = append(errs, FieldError{Field: taskPrefix + ".action", Message: "不支持的任务操作类型"})
			}
		}
	}
	return errs
}

// SaveStoryTasksReq 维护任务弹窗保存请求。
// 每条任务独立携带 projectId/executionId，支持弹窗内逐行选择项目与执行。
type SaveStoryTasksReq struct {
	ProjectID   uint                 `json:"projectId"`
	ExecutionID uint                 `json:"executionId"`
	Tasks       []SaveStoryTasksTask `json:"tasks"`
}

// Validate 校验维护任务保存请求。
func (r *SaveStoryTasksReq) Validate() []FieldError {
	var errs []FieldError
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
