// =============================================================================
// 文件: internal/module/build/form.go
// 模块: 版本管理
// 类型: action
// 职责: 关联研发需求列表 Req/Resp。
// 依赖: 无
// =============================================================================

package build

// LinkStoryListReq 关联需求默认列表查询（GET query）。
type LinkStoryListReq struct {
	Page     int `form:"page"`
	PageSize int `form:"pageSize"`
}

// Normalize 默认 page=1、pageSize=100（对齐禅道 linkStory），上限 100。
func (r *LinkStoryListReq) Normalize() {
	if r.Page < 1 {
		r.Page = 1
	}
	if r.PageSize < 1 {
		r.PageSize = 100
	}
	if r.PageSize > 100 {
		r.PageSize = 100
	}
}

// LinkStoryItem 列表行（模板渲染用）。
type LinkStoryItem struct {
	ID             uint
	Pri            uint8
	PriClass       string
	Title          string
	OpenedBy       string
	AssignedTo     string
	EstimateText   string
	StatusLabel    string
	StageLabel     string
	Stage          string
	DefaultChecked bool
}

// LinkStoryListResp 关联需求列表响应（供模板）。
type LinkStoryListResp struct {
	BuildID  uint
	BaseUrl  string
	Stories  []LinkStoryItem
	Total    int64
	Page     int
	PageSize int
}
