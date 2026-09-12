// =============================================================================
// 文件: internal/module/kanban/form.go
// 模块: 工作看板
// 类型: readonly
// 职责: 需求看板页展示用结构体。
// 依赖: 无
// =============================================================================

package kanban

// TeamgroupItem 当前用户所属敏捷小组（页头 chips）。
type TeamgroupItem struct {
	ID   uint
	Name string
}
