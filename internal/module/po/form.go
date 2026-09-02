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

// WorkItemDetail 单条需求或故事详情。
type WorkItemDetail struct {
	Kind        string `json:"kind"`
	ID          string `json:"id"`
	Pri         string `json:"pri"`         // 呈现给前端用 "P1"/"P2"/"P3"/"P4"，由 Service 层从真实列格式化
	PriRank     int    `json:"priRank"`     // 排序权重（1 最高，未识别填 99）
	Title       string `json:"title"`
	Stage       string `json:"stage"`
	Blocker     string `json:"blocker"`
	Next        string `json:"next"`
	Owner       string `json:"owner"`
	DueAt       string `json:"dueAt"`       // YYYY-MM-DD 或空
	DueLabel    string `json:"dueLabel"`    // 人类可读 "今日已超 3 天" / "距今 2 天"
	ZentaoUrl   string `json:"zentaoUrl"`
	ValueStream string `json:"valueStream"`
}

// DemandsResp 价值流状态下的需求详情列表。
type DemandsResp struct {
	Items []WorkItemDetail `json:"items"`
}
