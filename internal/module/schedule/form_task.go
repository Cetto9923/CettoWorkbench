// =============================================================================
// 文件: internal/module/schedule/form_task.go
// 模块: 排期工作台
// 类型: action
// 职责: 定义任务展示类型、故事点标签及基础参数解析校验。
// 依赖: 无
// =============================================================================

package schedule

import (
	"strconv"
	"strings"
)

// Validate 校验计划匹配查询参数。
func (r *MatchingPlansReq) Validate() []FieldError {
	var errs []FieldError
	if r.ProductID == 0 {
		errs = append(errs, FieldError{Field: "product_id", Message: "产品 ID 不能为空"})
	}
	endDate := strings.TrimSpace(r.EndDate)
	if endDate == "" {
		errs = append(errs, FieldError{Field: "end_date", Message: "结束日期不能为空"})
	} else if len(endDate) != 10 || endDate[4] != '-' || endDate[7] != '-' {
		errs = append(errs, FieldError{Field: "end_date", Message: "结束日期格式无效"})
	}
	return errs
}

// StoryAttachmentItem 研发需求附件条目。
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

// ParseCommaSeparatedUints 解析逗号分隔的无符号整型列表。
func ParseCommaSeparatedUints(raw string) []uint {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	out := make([]uint, 0, len(parts))
	seen := make(map[uint]struct{}, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		value, err := strconv.ParseUint(part, 10, 64)
		if err != nil || value == 0 {
			continue
		}
		id := uint(value)
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}
