// =============================================================================
// 文件: internal/module/schedule/formtask.go
// 模块: 排期工作台
// 类型: action
// 职责: 研发需求拆任务 / 任务列表的 Request / Response 结构体。
//       拆分自 form.go:任务类表单集中,含 StoryTaskItem 等嵌套数据。
// 依赖: internal/module/schedule/formshared.go (FieldError)
// =============================================================================

package schedule

import (
	"strconv"
	"strings"
)

type StoryAttachmentItem struct {
	ID    uint   `json:"id"`
	Title string `json:"title"`
}

// StoryTaskStoryItem 维护任务弹窗研发需求详情。
type StoryTaskStoryItem struct {
	ID             uint                  `json:"id"`
	Title          string                `json:"title"`
	ProductID      uint                  `json:"productId"`
	ProductName    string                `json:"productName"`
	AssignedToName string                `json:"assignedToName"`
	Spec           string                `json:"spec"`
	Verify         string                `json:"verify"`
	DemandID       uint                  `json:"demandId"`
	DemandName     string                `json:"demandName"`
	WindowName     string                `json:"windowName"`
	ReleaseDate    string                `json:"releaseDate"`
	Attachments    []StoryAttachmentItem `json:"attachments"`
}

// StoryTaskItem 维护任务弹窗任务条目。
type StoryTaskItem struct {
	ID             uint    `json:"id"`
	Type           string  `json:"type"`
	TypeLabel      string  `json:"typeLabel"`
	Name           string  `json:"name"`
	Pri            int     `json:"pri"`
	PriLabel       string  `json:"priLabel"`
	Status         string  `json:"status"`
	StatusLabel    string  `json:"statusLabel"`
	AssignedTo     string  `json:"assignedTo"`
	AssignedToName string  `json:"assignedToName"`
	FinishedBy     string  `json:"finishedBy"`
	FinishedByName string  `json:"finishedByName"`
	FinishedDate   string  `json:"finishedDate"`
	Estimate       float64 `json:"estimate"`
	Consumed       float64 `json:"consumed"`
	Left           float64 `json:"left"`
	Progress       int     `json:"progress"`
	EstStarted     string  `json:"estStarted"`
	Deadline       string  `json:"deadline"`
	ProjectID      uint    `json:"projectId"`
	ProjectName    string  `json:"projectName"`
	ExecutionID    uint    `json:"executionId"`
	ExecutionName  string  `json:"executionName"`
}

// StoryTaskSummary 只读任务列表汇总。
type StoryTaskSummary struct {
	Total         int     `json:"total"`
	WaitCount     int     `json:"waitCount"`
	DoingCount    int     `json:"doingCount"`
	EstimateTotal float64 `json:"estimateTotal"`
	ConsumedTotal float64 `json:"consumedTotal"`
	LeftTotal     float64 `json:"leftTotal"`
}

// StoryTasksResp 维护任务弹窗加载响应。
type StoryTasksResp struct {
	Story              StoryTaskStoryItem              `json:"story"`
	Tasks              []StoryTaskItem                 `json:"tasks"`
	Summary            StoryTaskSummary                `json:"summary"`
	Projects           []DemandSchedulingProjectOption `json:"projects"`
	Users              []SchedulingUserOption          `json:"users"`
	DefaultProjectID   uint                            `json:"defaultProjectId"`
	DefaultExecutionID uint                            `json:"defaultExecutionId"`
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

// SaveStoryTasksTask 维护任务弹窗保存任务条目。
// ProjectID/ExecutionID 由每条任务独立携带，projectId 仅用于前端联动加载执行，
// 后端通过 executionId 反查所属 project。
type SaveStoryTasksTask struct {
	Action      string  `json:"action"`
	ID          uint    `json:"id"`
	ProjectID   uint    `json:"projectId"`
	ExecutionID uint    `json:"executionId"`
	Type        string  `json:"type"`
	Pri         int     `json:"pri"`
	Name        string  `json:"name"`
	AssignedTo  string  `json:"assignedTo"`
	Estimate    float64 `json:"estimate"`
	EstStarted  string  `json:"estStarted"`
	Deadline    string  `json:"deadline"`
	Create      bool    `json:"create"`
}

// storyPointLabel 将故事点数字映射为关键词，与禅道 config/changshu.php:152-156 一致。
func storyPointLabel(point int) string {
	switch point {
	case 2:
		return "微型"
	case 3:
		return "小型"
	case 5:
		return "中型"
	case 8:
		return "大型"
	default:
		return ""
	}
}
