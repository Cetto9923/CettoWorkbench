// =============================================================================
// 文件: internal/module/po/form.go
// 模块: PO 工作台
// 类型: action
// 职责: PO 工作台页面 Req/Resp 结构体。
// 依赖: 无
// =============================================================================

package po

import (
	"strconv"
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
	Status   string `form:"status"`
	Page     int    `form:"page"`
	PageSize int    `form:"pageSize"`
}

// Normalize 规范化分页参数（默认 page=1、pageSize=10，上限 100）。
func (r *DemandsReq) Normalize() {
	if r.Page < 1 {
		r.Page = 1
	}
	if r.PageSize < 1 {
		r.PageSize = 10
	}
	if r.PageSize > 100 {
		r.PageSize = 100
	}
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

// ReviewDemandReq 业需评审提交（JSON Body，转发禅道 POST /demand/:id/review）。
//
// 字段对照禅道 OpenAPI：
//
//	result  → 评审结果 pass=确认通过 / refuse=拒绝（必填）
//	comment → 备注（纯文本，可选；驳回时前端会填）
//	mailto  → 通知人（可选，当前抽屉不传）
//
// ID 不从 JSON 读，由 Handler 从 URL :id 填进来（和 DeleteReq 同一套路）。
type ReviewDemandReq struct {
	ID      int64  `json:"-"`
	Result  string `json:"result"`
	Mailto  string `json:"mailto"`
	Comment string `json:"comment"`
}

// Validate 校验评审表单。返回空切片表示通过。
func (r *ReviewDemandReq) Validate() []FieldError {
	var errs []FieldError
	r.Result = strings.TrimSpace(r.Result)
	r.Mailto = strings.TrimSpace(r.Mailto)
	r.Comment = strings.TrimSpace(r.Comment)

	if r.ID <= 0 {
		errs = append(errs, FieldError{Field: "id", Message: "需求 ID 无效"})
	}
	if r.Result != "pass" && r.Result != "refuse" {
		errs = append(errs, FieldError{Field: "result", Message: "请选择评审结果"})
	}
	return errs
}

// ReviewDemandResp 评审成功响应（目前只回 ID，方便以后加字段）。
type ReviewDemandResp struct {
	ID int64 `json:"id"`
}

// WorkItemDetail 单条需求或故事详情。
type WorkItemDetail struct {
	Kind            string `json:"kind"`
	ID              string `json:"id"` // 展示编号：业需 US{id}，研需 U{id}
	Pri             string `json:"pri"`
	Title           string `json:"title"`
	Stage           string `json:"stage"`
	Blocker         string `json:"blocker"`
	Next            string `json:"next"`
	Owner           string `json:"owner"`     // 与 NextOwner 同值，兼容旧字段
	NextOwner       string `json:"nextOwner"` // 下一责任人展示名（DeriveCurrentHandler）
	ZentaoUrl       string `json:"zentaoUrl"`
	ClarifyUrl      string `json:"clarifyUrl"`  // 禅道业需澄清页（demand-clarify）
	AppraiseUrl     string `json:"appraiseUrl"` // 禅道业需评价页（demand-appraise）
	ValueStream     string `json:"valueStream"`
	ZentaoStatus    string `json:"zentaoStatus"`    // 禅道 status 原文，前端按业需/研需分别映射中文
	CanReview       bool   `json:"canReview"`       // 待评审且当前账号是未出结果的业务评审人
	CanCancelReview bool   `json:"canCancelReview"` // 待评审且当前账号是提交人（创建人）
	CanSubmitReview bool   `json:"canSubmitReview"` // 草稿/已驳回且当前账号是创建人
	CanEdit         bool   `json:"canEdit"`         // 草稿/已驳回且当前账号是创建人
}

// DemandsResp 价值流状态下的需求详情列表。
type DemandsResp struct {
	Items    []WorkItemDetail `json:"items"`
	Total    int64            `json:"total"`
	Page     int              `json:"page"`
	PageSize int              `json:"pageSize"`
}

// DemandDetailReq 业需详情查询（path :id，支持 US123 / 123）。
type DemandDetailReq struct {
	ID string
}

// ExtractDemandID 从 US{id} 或纯数字解析主键。
func (r *DemandDetailReq) ExtractDemandID() int64 {
	raw := strings.TrimSpace(r.ID)
	raw = strings.TrimPrefix(raw, "US")
	raw = strings.TrimPrefix(raw, "us")
	id, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || id <= 0 {
		return 0
	}
	return id
}

// Validate 校验需求 ID。
func (r *DemandDetailReq) Validate() []FieldError {
	if r.ExtractDemandID() <= 0 {
		return []FieldError{{Field: "id", Message: "需求 ID 无效"}}
	}
	return nil
}

// DemandAttachment 需求附件。
type DemandAttachment struct {
	ID       uint   `json:"id"`
	Title    string `json:"title"`
	Size     string `json:"size"`
	Download string `json:"download"`
}

// DemandDetailResp 业需评审抽屉详情（字段对齐禅道 demand-view）。
type DemandDetailResp struct {
	ID                string             `json:"id"`
	DemandID          int64              `json:"demandId"`
	Title             string             `json:"title"`
	Pri               string             `json:"pri"`
	Category          string             `json:"category"`
	Source            string             `json:"source"`
	PoolName          string             `json:"poolName"`
	Deadline          string             `json:"deadline"` // 期望上线，zt_demand.deadline
	ProposerName      string             `json:"proposerName"`
	ProposerDept      string             `json:"proposerDept"`
	OwnerName         string             `json:"ownerName"`
	Reviewer          string             `json:"reviewer"`
	CreatedName       string             `json:"createdName"`
	CurrentOwner      string             `json:"currentOwner"` // UI 展示为「指派给」，取 assignedTo
	ZentaoStatus      string             `json:"zentaoStatus"`
	ZentaoStatusLabel string             `json:"zentaoStatusLabel"`
	ValueStageLabel   string             `json:"valueStageLabel"`
	SpecHtml          string             `json:"specHtml"`
	VerifyHtml        string             `json:"verifyHtml"`
	ZentaoURL         string             `json:"zentaoUrl"`
	ZentaoEditURL     string             `json:"zentaoEditUrl"`
	Attachments       []DemandAttachment `json:"attachments"`
}
