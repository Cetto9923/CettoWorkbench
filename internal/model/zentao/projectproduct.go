// =============================================================================
// 文件: internal/model/zentao/projectproduct.go
// 模块: 数据模型
// 类型: model
// 职责: 定义项目/执行与产品关联表字段与表映射。
// 依赖: 无
// =============================================================================

package model

// ZtProjectproduct 项目/执行-产品关联表（zt_projectproduct）。
// 复合主键：project + product + branch；project 列可为项目或执行 ID。
type ZtProjectproduct struct {
	Project uint   `gorm:"column:project;primaryKey;type:mediumint unsigned;not null;default:0" json:"project"`
	Product uint   `gorm:"column:product;primaryKey;type:mediumint unsigned;not null;default:0" json:"product"`
	Branch  uint   `gorm:"column:branch;primaryKey;type:mediumint unsigned;not null;default:0" json:"branch"`
	Plan    string `gorm:"column:plan;type:varchar(255);not null;default:''" json:"plan"`
	Roadmap string `gorm:"column:roadmap;type:varchar(255);not null;default:''" json:"roadmap"`
}

// TableName 指定表名。
func (ZtProjectproduct) TableName() string {
	return "zt_projectproduct"
}
