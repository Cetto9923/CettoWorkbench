// =============================================================================
// 文件: internal/module/build/linked.go
// 模块: 版本管理
// 类型: action
// 职责: 将版本已关联 story ID 装配为前端 link-list 项。
// 依赖: 无
// =============================================================================

package build

// storyTitleRow 已关联需求的 id/title（Repo 查询行）。
type storyTitleRow struct {
	ID    uint   `gorm:"column:id"`
	Title string `gorm:"column:title"`
}

// BuildLinkedStoryItems 按 linkedIDs 顺序装配；无标题行仍保留 id。
func BuildLinkedStoryItems(linkedIDs []uint, byID map[uint]storyTitleRow) []LinkedStoryItem {
	out := make([]LinkedStoryItem, 0, len(linkedIDs))
	for _, id := range linkedIDs {
		if id == 0 {
			continue
		}
		title := ""
		if row, ok := byID[id]; ok {
			title = row.Title
		}
		out = append(out, LinkedStoryItem{ID: id, Title: title})
	}
	return out
}
