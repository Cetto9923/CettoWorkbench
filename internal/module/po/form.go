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
	KPI            KPICounts
}

// KPICounts 首页 5 个焦点摘要的真实计数。
// 4 个 KPI 由 repo CountKPI{...} 真实统计;MyPending 由 Service 计算。
// 字段为零时前端仍展示数字 0,不显示破折号。
type KPICounts struct {
	Today     int64 // 今日必推：今日到期 OR 已逾期 且未完成
	MyPending int64 // 待我处理：handlingResponsibility=currentUser（=价值流 all 计数）
	Blocked   int64 // 阻塞：主管部门审批存在拒绝 ∪ 验收阶段超期
	Overdue   int64 // 超期：today > deadline 且未完成（缺日期不算）
	Suspended int64 // 挂起：hang='1' 且未关闭
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

// WorkItemDetail 单条需求或故事详情。
type WorkItemDetail struct {
	Kind         string `json:"kind"`
	ID           string `json:"id"` // 展示编号：业需 US{id}，研需 U{id}
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
}

// DemandsResp 价值流状态下的需求详情列表。
type DemandsResp struct {
	Items []WorkItemDetail `json:"items"`
}
