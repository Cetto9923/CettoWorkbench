// =============================================================================
// 文件: internal/module/po/form.go
// 模块: PO 工作台
// 类型: action
// 职责: PO 工作台页面 Req/Resp 结构体。
// 依赖: 无
// =============================================================================

package po

import (
	"strings"

	"workbench/internal/module/schedule"
)

// FieldError 字段级验证错误。
type FieldError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

// ValueStreamStage 价值流阶段卡片数据（对应 homeVsCompact 单个阶段）。
type ValueStreamStage struct {
	Label       string
	Status      string
	Count       int64
	DemandCount int64
	StoryCount  int64
}

// HomeResp PO 工作台首页数据。
type HomeResp struct {
	Stages         []ValueStreamStage
	VersionWindows []schedule.HomeVersionWindowCard
}

// DemandsReq 按价值流状态查询需求/故事详情。
type DemandsReq struct {
	Status string `form:"status"`
}

// Validate 校验查询参数。
func (r *DemandsReq) Validate() []FieldError {
	status := strings.TrimSpace(r.Status)
	if status == "" {
		return []FieldError{{Field: "status", Message: "状态不能为空"}}
	}
	if !isValidValueStreamStatus(status) {
		return []FieldError{{Field: "status", Message: "无效的价值流状态"}}
	}
	r.Status = status
	return nil
}

// ReviewDemandReq 业需评审提交（JSON Body，对应禅道 demand-review 表单）。
//
// 字段对照禅道 POST：
//
//	result      → 评审结果 pass=确认通过 / refuse=拒绝
//	isNeedFocus → 是否重点关注 0=否 / 1=是
//	mailto      → 通知人，逗号分隔账号
//	comment     → 备注（纯文本）
//
// ID 不从 JSON 读，由 Handler 从 URL :id 填进来（和 DeleteReq 同一套路）。
type ReviewDemandReq struct {
	ID          int64  `json:"-"`
	Result      string `json:"result"`
	IsNeedFocus string `json:"isNeedFocus"`
	Mailto      string `json:"mailto"`
	Comment     string `json:"comment"`
}

// Validate 校验评审表单。返回空切片表示通过。
func (r *ReviewDemandReq) Validate() []FieldError {
	var errs []FieldError
	r.Result = strings.TrimSpace(r.Result)
	r.IsNeedFocus = strings.TrimSpace(r.IsNeedFocus)
	r.Mailto = strings.TrimSpace(r.Mailto)
	r.Comment = strings.TrimSpace(r.Comment)

	if r.ID <= 0 {
		errs = append(errs, FieldError{Field: "id", Message: "需求 ID 无效"})
	}
	if r.Result != "pass" && r.Result != "refuse" {
		errs = append(errs, FieldError{Field: "result", Message: "请选择评审结果"})
	}
	if r.IsNeedFocus != "0" && r.IsNeedFocus != "1" {
		errs = append(errs, FieldError{Field: "isNeedFocus", Message: "请选择是否需要重点关注"})
	}
	return errs
}

// ReviewDemandResp 评审成功响应（目前只回 ID，方便以后加字段）。
type ReviewDemandResp struct {
	ID int64 `json:"id"`
}

// WorkItemDetail 单条需求或故事详情。
type WorkItemDetail struct {
	Kind         string `json:"kind"`
	ID           string `json:"id"`
	Pri          string `json:"pri"`
	Title        string `json:"title"`
	Stage        string `json:"stage"`
	Blocker      string `json:"blocker"`
	Next         string `json:"next"`
	Owner        string `json:"owner"`     // 与 NextOwner 同值，兼容旧字段
	NextOwner    string `json:"nextOwner"` // 下一责任人展示名（DeriveCurrentHandler）
	ZentaoUrl    string `json:"zentaoUrl"`
	ValueStream  string `json:"valueStream"`
	ZentaoStatus string `json:"zentaoStatus"` // 禅道 status 原文，前端按业需/研需分别映射中文
	CanReview    bool   `json:"canReview"`    // 当前登录人是待评业务评审人（与指派给无关）
}

// DemandsResp 价值流状态下的需求详情列表。
type DemandsResp struct {
	Items []WorkItemDetail `json:"items"`
}
