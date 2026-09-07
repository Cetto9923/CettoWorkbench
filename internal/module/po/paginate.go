// =============================================================================
// 文件: internal/module/po/paginate.go
// 模块: PO 工作台
// 类型: action
// 职责: 工作项列表内存切页（合并列表 / 「全部」并集后使用）。
// 依赖: 无
// =============================================================================

package po

// paginateWorkItems 按 page/pageSize 切一页；page 越界时返回空切片与正确 total。
func paginateWorkItems(items []WorkItemDetail, page, pageSize int) ([]WorkItemDetail, int64) {
	total := int64(len(items))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 10
	}
	if total == 0 {
		return []WorkItemDetail{}, 0
	}
	start := (page - 1) * pageSize
	if start >= len(items) {
		return []WorkItemDetail{}, total
	}
	end := start + pageSize
	if end > len(items) {
		end = len(items)
	}
	out := make([]WorkItemDetail, end-start)
	copy(out, items[start:end])
	return out, total
}
