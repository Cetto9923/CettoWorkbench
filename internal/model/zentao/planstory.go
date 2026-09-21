// =============================================================================
// 文件: internal/model/zentao/planstory.go
// 模块: 数据模型
// 类型: model
// 职责: 定义计划与研发需求关联表 zt_planstory。
// 依赖: 无
// =============================================================================

package model

// ZtPlanstory 计划-研发需求关联表。
type ZtPlanstory struct {
	Plan  uint `gorm:"column:plan;primaryKey"`
	Story uint `gorm:"column:story;primaryKey"`
	Order int  `gorm:"column:order"`
}

// TableName 指定表名。
func (ZtPlanstory) TableName() string { return "zt_planstory" }
