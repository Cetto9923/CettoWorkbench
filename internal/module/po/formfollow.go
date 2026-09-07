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

// FollowScope 业务需求关注视图的二级筛选。
type FollowScope string

const (
	FollowScopeAll    FollowScope = "all"
	FollowScopeKey    FollowScope = "key"
	FollowScopeClosed FollowScope = "closed"
)

// FollowListReq 我的关注列表请求。
type FollowListReq struct {
	Tab      FollowTab   `form:"tab"`
	Scope    FollowScope `form:"scope"`
	Keyword  string      `form:"keyword"`
	Page     int         `form:"page"`
	PageSize int         `form:"pageSize"`
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
		r.Scope = FollowScopeAll
	}
	switch r.Scope {
	case FollowScopeAll, FollowScopeKey, FollowScopeClosed:
	default:
		return []FieldError{{Field: "scope", Message: "无效的二级筛选"}}
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
	ID            int64                        `json:"id"`
	Title         string                       `json:"title"`
	Status        string                       `json:"status"`
	Priority      string                       `json:"priority"`
	Owner         string                       `json:"owner"`
	LatestNote    string                       `json:"latestNote"`
	Date          string                       `json:"date"`
	IsKey         bool                         `json:"isKey"`
	IsClosed      bool                         `json:"isClosed"`
	URL           string                       `json:"url"`
	PrimaryAction *primaryaction.PrimaryAction `json:"primaryAction,omitempty"`
}

// FollowListResp 我的关注响应。
type FollowListResp struct {
	Items    []FollowItem `json:"items"`
	Total    int64        `json:"total"`
	Page     int          `json:"page"`
	PageSize int          `json:"pageSize"`
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
