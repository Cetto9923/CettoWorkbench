// =============================================================================
// 文件: internal/model/zentao/storyspec.go
// 模块: 数据模型
// 类型: model
// 职责: 定义研发需求描述表 zt_storyspec 字段与表映射。
// 依赖: 无
// =============================================================================

package model

// ZtStoryspec 研发需求描述表。
type ZtStoryspec struct {
	Story   uint   `gorm:"column:story;primaryKey"`
	Version int    `gorm:"column:version;primaryKey"`
	Title   string `gorm:"column:title"`
	Spec    string `gorm:"column:spec"`
}

// TableName 指定表名。
func (ZtStoryspec) TableName() string { return "zt_storyspec" }
