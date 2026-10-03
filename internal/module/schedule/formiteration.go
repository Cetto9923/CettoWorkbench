// =============================================================================
// 文件: internal/module/schedule/formiteration.go
// 模块: 排期工作台
// 类型: action
// 职责: 快速创建迭代请求。
// 依赖: 无
// =============================================================================
package schedule

type CreateIterationReq struct {
	StoryID   uint   `json:"-"`
	ProjectID uint   `json:"projectId"`
	Period    string `json:"period"`
	PlanID    uint   `json:"planId"`
}

func (r *CreateIterationReq) Validate() []FieldError {
	var errors []FieldError
	if r.ProjectID == 0 {
		errors = append(errors, FieldError{Field: "projectId", Message: "请选择项目"})
	}
	if r.Period != "2w" && r.Period != "4w" && r.Period != "plan" {
		errors = append(errors, FieldError{Field: "period", Message: "请选择迭代周期"})
	}
	if r.Period == "plan" && r.PlanID == 0 {
		errors = append(errors, FieldError{Field: "planId", Message: "请选择产品计划"})
	}
	return errors
}
