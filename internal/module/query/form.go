// 需求查询模块的页面查询模型。
package query

import (
	"strings"
)

const (
	tabBiz = "biz"
	tabRD  = "rd"
)

type ListReq struct {
	Tab      string
	Keyword  string
	Status   string
	Priority string
	Owner    string
	System   string
	Stage    string
	Page     int
	PageSize int
}

func (r *ListReq) Normalize() {
	if r.Tab != tabRD {
		r.Tab = tabBiz
	}
	r.Keyword = strings.TrimSpace(r.Keyword)
	r.Status = strings.TrimSpace(r.Status)
	r.Priority = strings.TrimSpace(r.Priority)
	r.Owner = strings.TrimSpace(r.Owner)
	r.System = strings.TrimSpace(r.System)
	r.Stage = strings.TrimSpace(r.Stage)
	if r.Page < 1 {
		r.Page = 1
	}
	if r.PageSize < 1 || r.PageSize > 100 {
		r.PageSize = 15
	}
}

// Validate 在 Normalize 之后调用，做边界收紧；当前主要限制 pageSize 上限，
// 返回的 error 由 Handler 决定是否响应 400 / 渲染空数据。
func (r *ListReq) Validate() error {
	if r.Tab != tabBiz && r.Tab != tabRD {
		r.Tab = tabBiz
	}
	if r.PageSize > 100 {
		r.PageSize = 100
	}
	return nil
}

type Row struct {
	ID       uint   `json:"id"`
	Title    string `json:"title"`
	Kind     string `json:"kind"`
	Priority string `json:"priority"`
	Status   string `json:"status"`
	Stage    string `json:"stage"`
	Owner    string `json:"owner"`
	System   string `json:"system"`
	Deadline string `json:"deadline"`
	Source   string `json:"source"`
	URL      string `json:"url"`
}

type ListResp struct {
	Kind     string
	Rows     []Row
	Total    int64
	Page     int
	PageSize int
}
