// =============================================================================
// 文件: internal/module/build/form.go
// 模块: 版本管理
// 类型: action
// 职责: 关联研发需求列表 Req/Resp。
// 依赖: 无
// =============================================================================

package build

import "strings"

// LinkStoryListReq 关联需求列表查询（GET query；含 bySearch）。
type LinkStoryListReq struct {
	Page       int    `form:"page"`
	PageSize   int    `form:"pageSize"`
	BrowseType string `form:"browseType"`

	Field1    string `form:"field1"`
	Operator1 string `form:"operator1"`
	Value1    string `form:"value1"`
	AndOr     string `form:"andOr"`
	Field2    string `form:"field2"`
	Operator2 string `form:"operator2"`
	Value2    string `form:"value2"`
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
	r.BrowseType = strings.TrimSpace(r.BrowseType)
	r.Field1 = strings.TrimSpace(r.Field1)
	r.Operator1 = NormalizeOperator(r.Operator1)
	r.Value1 = strings.TrimSpace(r.Value1)
	r.AndOr = NormalizeAndOr(r.AndOr)
	r.Field2 = strings.TrimSpace(r.Field2)
	r.Operator2 = NormalizeOperator(r.Operator2)
	r.Value2 = strings.TrimSpace(r.Value2)
	if r.Field1 == "" {
		r.Field1 = "title"
		if r.Operator1 == "=" && r.Value1 == "" {
			r.Operator1 = "include"
		}
	}
	if r.Field2 == "" {
		r.Field2 = "status"
	}
}

// IsBySearch 是否走禅道 bySearch 数据源。
func (r *LinkStoryListReq) IsBySearch() bool {
	return strings.EqualFold(strings.TrimSpace(r.BrowseType), "bySearch")
}

// Cond1 / Cond2 用户条件。
func (r *LinkStoryListReq) Cond1() SearchCond {
	return SearchCond{Field: r.Field1, Operator: r.Operator1, Value: r.Value1}
}
func (r *LinkStoryListReq) Cond2() SearchCond {
	return SearchCond{Field: r.Field2, Operator: r.Operator2, Value: r.Value2}
}

// LinkStoryItem 列表行（模板渲染用）。
type LinkStoryItem struct {
	ID             uint
	Pri            uint8
	PriClass       string
	Title          string
	ZentaoUrl      string
	OpenedBy       string
	AssignedTo     string
	EstimateText   string
	StatusLabel    string
	StageLabel     string
	Stage          string
	DefaultChecked bool
}

// SearchOption 下拉选项。
type SearchOption struct {
	Value string `json:"value"`
	Label string `json:"label"`
}

// SearchFieldDef 搜索字段元数据（前端动态取值控件）。
type SearchFieldDef struct {
	Key        string `json:"key"`
	Label      string `json:"label"`
	Control    string `json:"control"` // input|select|date
	Operator   string `json:"operator"`
	OptionsKey string `json:"optionsKey"` // status|stage|pri|users|modules|plans|...
}

// LinkStorySearchForm 回填与元数据。
type LinkStorySearchForm struct {
	BrowseType string
	Field1     string
	Operator1  string
	Value1     string
	AndOr      string
	Field2     string
	Operator2  string
	Value2     string
	Fields     []SearchFieldDef
	Options    map[string][]SearchOption
}

// LinkStoryListResp 关联需求列表响应（供模板）。
type LinkStoryListResp struct {
	BuildID    uint
	BaseUrl    string
	Stories    []LinkStoryItem
	Total      int64
	Page       int
	PageSize   int
	SearchForm LinkStorySearchForm
	QuerySuffix string // 分页链接附加查询串（含前置 &）
}
