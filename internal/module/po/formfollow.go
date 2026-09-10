// =============================================================================
// 文件: internal/module/po/formfollow.go
// 模块: PO 工作台
// 类型: action
// 职责: 定义我的关注页面请求与响应契约。
// 依赖: 无
// =============================================================================

package po

import (
	"strings"

	"workbench/internal/module/po/primaryaction"
)

// FollowTab 我的关注对象视图。
type FollowTab string

const (
	FollowTabDemand        FollowTab = "demand"
	FollowTabProjectReport FollowTab = "project_report"
)

// FollowScope 业务需求关注视图的二级关注维度筛选（对齐 CRCBWorkbench）。
type FollowScope string

const (
	FollowScopeOpen      FollowScope = "open"       // 全部未关闭（默认）
	FollowScopeAll       FollowScope = "all"        // 全量（含关闭）
	FollowScopeKey       FollowScope = "key"        // 重点关注（保留兼容）
	FollowScopeKeyOpen   FollowScope = "key_open"   // 未关闭 + 重点关注（保留兼容）
	FollowScopeClosed    FollowScope = "closed"     // 已关闭（仅 closed）
	FollowScopeOpenClean FollowScope = "open_clean" // 未关闭 · 正常推进（保留兼容）
)

// FollowLifecycle 业务需求生命周期桶（统计卡口径）。
type FollowLifecycle string

const (
	FollowLifecycleAll          FollowLifecycle = "all"
	FollowLifecycleClarifying   FollowLifecycle = "clarifying"   // 梳理中
	FollowLifecycleImplementing FollowLifecycle = "implementing" // 实施中
	FollowLifecycleReleased     FollowLifecycle = "released"     // 已发布
	FollowLifecycleClosed       FollowLifecycle = "closed"       // 已关闭
)

// FollowListReq 我的关注列表请求。
type FollowListReq struct {
	Tab       FollowTab       `form:"tab"`
	Scope     FollowScope     `form:"scope"`
	Lifecycle FollowLifecycle `form:"lifecycle"`
	Keyword   string          `form:"keyword"`
	Page      int             `form:"page"`
	PageSize  int             `form:"pageSize"`
}

// Validate 校验 FollowListReq。
func (r *FollowListReq) Validate() []FieldError {
	r.Tab = FollowTab(strings.TrimSpace(string(r.Tab)))
	if r.Tab == "" {
		r.Tab = FollowTabDemand
	}
	switch r.Tab {
	case FollowTabDemand, FollowTabProjectReport:
	default:
		return []FieldError{{Field: "tab", Message: "无效的对象视图"}}
	}
	r.Scope = FollowScope(strings.TrimSpace(string(r.Scope)))
	if r.Scope == "" {
		r.Scope = FollowScopeOpen
	}
	switch r.Scope {
	case FollowScopeOpen, FollowScopeAll, FollowScopeKey, FollowScopeKeyOpen, FollowScopeClosed, FollowScopeOpenClean:
	default:
		return []FieldError{{Field: "scope", Message: "无效的二级筛选"}}
	}
	r.Lifecycle = FollowLifecycle(strings.TrimSpace(string(r.Lifecycle)))
	if r.Lifecycle == "" {
		r.Lifecycle = FollowLifecycleAll
	}
	switch r.Lifecycle {
	case FollowLifecycleAll, FollowLifecycleClarifying, FollowLifecycleImplementing, FollowLifecycleReleased, FollowLifecycleClosed:
	default:
		return []FieldError{{Field: "lifecycle", Message: "无效的生命周期筛选"}}
	}
	r.Keyword = strings.TrimSpace(r.Keyword)
	if r.Page < 1 {
		r.Page = 1
	}
	if r.PageSize < 1 || r.PageSize > 100 {
		r.PageSize = 20
	}
	return nil
}

// FollowItem 我的关注单条。
type FollowItem struct {
	ID              int64                        `json:"id"`
	Title           string                       `json:"title"`
	Status          string                       `json:"status"`
	Stage           string                       `json:"stage"`
	Role            string                       `json:"role"`
	SystemName      string                       `json:"systemName"`
	SupportSystems  string                       `json:"supportSystems"`
	Risk            string                       `json:"risk"`
	Reason          string                       `json:"reason"`
	Priority        string                       `json:"priority"`
	Owner           string                       `json:"owner"`
	LatestNote      string                       `json:"latestNote"`
	Date            string                       `json:"date"`
	IsKey           bool                         `json:"isKey"`
	IsClosed        bool                         `json:"isClosed"`
	URL             string                       `json:"url"`
	LifecycleBucket string                       `json:"lifecycleBucket,omitempty"`
	DevelopFinish   string                       `json:"developFinish,omitempty"`
	TestFinish      string                       `json:"testFinish,omitempty"`
	Deadline        string                       `json:"deadline,omitempty"`
	ProgressStatus  string                       `json:"progressStatus,omitempty"` // normal | delayed | done | unknown
	ProgressLabel   string                       `json:"progressLabel,omitempty"`
	ScheduleSummary string                       `json:"scheduleSummary,omitempty"`
	PrimaryAction   *primaryaction.PrimaryAction `json:"primaryAction,omitempty"`
}

// FollowDemandStats 业务需求关注统计（统计卡 + 工具栏计数同源）。
type FollowDemandStats struct {
	Open         int64 `json:"open"`
	All          int64 `json:"all"`
	Clarifying   int64 `json:"clarifying"`
	Implementing int64 `json:"implementing"`
	Released     int64 `json:"released"`
	Closed       int64 `json:"closed"`
	Key          int64 `json:"key"`
	KeyOpen      int64 `json:"keyOpen"`
	OpenClean    int64 `json:"openClean"`
}

// FollowListResp 我的关注响应。
type FollowListResp struct {
	Items    []FollowItem       `json:"items"`
	Total    int64              `json:"total"`
	Page     int                `json:"page"`
	PageSize int                `json:"pageSize"`
	Stats    *FollowDemandStats `json:"stats,omitempty"`
}

// FollowSetReq 更新业务需求关注关系。
type FollowSetReq struct {
	ID       int64 `json:"-"`
	Followed *bool `json:"followed"`
}

// Validate 校验关注关系更新参数。
func (r *FollowSetReq) Validate() []FieldError {
	if r.ID <= 0 {
		return []FieldError{{Field: "id", Message: "无效的业务需求 ID"}}
	}
	if r.Followed == nil {
		return []FieldError{{Field: "followed", Message: "关注状态不能为空"}}
	}
	return nil
}
