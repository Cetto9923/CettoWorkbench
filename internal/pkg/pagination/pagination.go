// =============================================================================
// 文件: internal/pkg/pagination/pagination.go
// 模块: 基础设施
// 类型: infra
// 职责: 定义分页结构与分页计算辅助方法。
// 依赖: 无
// =============================================================================

package pagination

import "strings"

// 分页默认配置。
const (
	DefaultPageSize = 20
	MaxPageSize     = 2000
)

// Pager 描述分页状态及用于渲染页码的数据。
type Pager struct {
	TotalItems    int64
	TotalPages    int
	CurrentPage   int
	PageSize      int
	HasPrev       bool
	HasNext       bool
	Pages         []int
	PageParam       string            // query 参数名，默认 page
	PreserveParams  map[string]string // 分页链接需保留的其他 query 参数
}

// New 按 totalItems/currentPage/pageSize 创建分页对象，并自动规范化参数。
func New(totalItems int64, currentPage, pageSize int) *Pager {
	if currentPage < 1 {
		currentPage = 1
	}
	if pageSize < 1 {
		pageSize = DefaultPageSize
	}
	if pageSize > MaxPageSize {
		pageSize = MaxPageSize
	}
	if totalItems < 0 {
		totalItems = 0
	}

	totalPages := 0
	if totalItems > 0 {
		totalPages = int((totalItems + int64(pageSize) - 1) / int64(pageSize))
	}
	if totalPages > 0 && currentPage > totalPages {
		currentPage = totalPages
	}

	p := &Pager{
		TotalItems:  totalItems,
		TotalPages:  totalPages,
		CurrentPage: currentPage,
		PageSize:    pageSize,
	}
	p.HasPrev = p.CurrentPage > 1
	p.HasNext = p.TotalPages > 0 && p.CurrentPage < p.TotalPages
	p.Pages = buildPages(p.CurrentPage, p.TotalPages, 2)
	return p
}

// Offset 返回分页查询偏移量。
func (p *Pager) Offset() int {
	if p == nil {
		return 0
	}
	if p.CurrentPage < 1 {
		return 0
	}
	return (p.CurrentPage - 1) * p.Limit()
}

// Limit 返回分页查询每页条数。
func (p *Pager) Limit() int {
	if p == nil {
		return DefaultPageSize
	}
	if p.PageSize < 1 {
		return DefaultPageSize
	}
	if p.PageSize > MaxPageSize {
		return MaxPageSize
	}
	return p.PageSize
}

// QueryPageParam 返回页码 query 参数名。
func (p *Pager) QueryPageParam() string {
	if p == nil || strings.TrimSpace(p.PageParam) == "" {
		return "page"
	}
	return strings.TrimSpace(p.PageParam)
}

func buildPages(currentPage, totalPages, around int) []int {
	if totalPages <= 0 {
		return []int{}
	}
	start := currentPage - around
	if start < 1 {
		start = 1
	}
	end := currentPage + around
	if end > totalPages {
		end = totalPages
	}
	pages := make([]int, 0, end-start+1)
	for i := start; i <= end; i++ {
		pages = append(pages, i)
	}
	return pages
}
